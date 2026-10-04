package uz.keel.courier

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore
import uz.keel.courier.data.ApiError
import uz.keel.courier.data.Courier
import uz.keel.courier.data.KeelApi

// This sitting on this phone: which restaurant, and who is signed in.
//
// ⚠️ **The address is remembered, the person is remembered, nothing else is.** A
// courier's phone is theirs — signing in every shift would be the fastest way to
// have the password written on the back of the case. The restaurant's address is
// set once, when the phone is handed over.
//
// ⚠️ **The shift status is not kept here.** It lives on the server, because it
// is not a fact about this phone: a dispatcher can take a courier off shift from
// the panel, and a phone that remembered its own answer would keep reporting
// "available" to a screen that had already said otherwise.

@Immutable
sealed interface Session {
    data object Loading : Session
    data object NoServer : Session

    /** ⚠️ **Not the same as being signed out.** A launch asks the server who
     *  this is; when the request never arrives the answer used to be "signed
     *  out", so a courier in a basement or out of data met a password field —
     *  types the right password, watches it fail, and blames themselves for a
     *  network they cannot see. Nothing on that screen could have helped. */
    data class Offline(val address: String) : Session
    data class SignedOut(val address: String) : Session
    data class Ready(val address: String, val courier: Courier) : Session
}

class SessionViewModel(
    private val api: KeelApi,
    private val tokens: TokenStore,
) : ViewModel() {

    private val _state = MutableStateFlow<Session>(Session.Loading)
    val state: StateFlow<Session> = _state.asStateFlow()

    init { probe() }

    /** Ask the server who this is — the one question a launch has to answer.
     *
     *  ⚠️ **Three outcomes, not two.** "The server says no" and "the server did
     *  not answer" look identical from a `catch` and mean opposite things: one
     *  is a sign-in, the other is a network. `ApiError` is the server having
     *  spoken — any status, 401 included. */
    fun probe() {
        viewModelScope.launch {
            val address = tokens.read(TokenStore.SERVER_ADDRESS)
            if (address.isNullOrBlank()) {
                _state.value = Session.NoServer
                return@launch
            }
            api.useServer(address)
            try {
                _state.value = Session.Ready(address, api.me())
            } catch (e: ApiError) {
                // A refused token is a sign-in, not an error screen: it expired,
                // or the account was switched off, and both have the same answer
                // for the person holding the phone.
                _state.value = Session.SignedOut(address)
            } catch (e: Throwable) {
                _state.value = Session.Offline(address)
            }
        }
    }

    fun useServer(address: String): Boolean {
        if (ServerAddress.apiBase(address).isEmpty()) return false
        tokens.write(TokenStore.SERVER_ADDRESS, address)
        api.useServer(address)
        _state.value = Session.SignedOut(address)
        return true
    }

    suspend fun signIn(address: String, username: String, password: String) {
        val res = api.login(username, password)
        tokens.write(TokenStore.COURIER_TOKEN, res.token)
        _state.value = Session.Ready(address, res.courier)
    }

    fun signOut(address: String) {
        tokens.drop(TokenStore.COURIER_TOKEN)
        _state.value = Session.SignedOut(address)
    }

    /** ⚠️ Kept apart from signing out. Forgetting the restaurant is what happens
     *  when a courier changes employer; signing out is the end of a shift, and
     *  answering both with one button would make the common one cost the rare
     *  one's setup — and the rare one is destructive in a way the daily one is
     *  not. */
    fun forgetServer() {
        tokens.drop(TokenStore.COURIER_TOKEN)
        tokens.drop(TokenStore.SERVER_ADDRESS)
        api.useServer("")
        _state.value = Session.NoServer
    }

    /** The shift switch.
     *
     *  ⚠️ **The server is told first and the screen follows it.** The reverse
     *  order shows a courier as available on a phone the dispatcher's list still
     *  has as off — and the courier waits for orders that are being given to
     *  somebody else. */
    suspend fun setStatus(status: String) {
        api.setStatus(status)
        val s = _state.value
        if (s is Session.Ready) {
            _state.value = s.copy(courier = s.courier.copy(status = status))
        }
    }

    /** Re-read the profile.
     *
     *  ⚠️ Called after every list refresh, because the status can change without
     *  this phone doing anything: finishing the last order flips a courier back
     *  to free on the server (`syncCourierBusy`), and a switch that disagrees
     *  with the server is the one control here somebody would press twice. */
    fun refresh() {
        viewModelScope.launch {
            runCatching { api.me() }.onSuccess { me ->
                val s = _state.value
                if (s is Session.Ready) _state.value = s.copy(courier = me)
            }
            // A dropped request is not a sign-out: the courier is riding through
            // a basement, and the next poll answers.
        }
    }
}
