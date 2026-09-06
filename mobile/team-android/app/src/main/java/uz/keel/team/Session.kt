package uz.keel.team

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore
import uz.keel.team.data.ApiError
import uz.keel.team.data.KeelApi
import uz.keel.team.data.Staff

// This sitting on this phone: which restaurant, and who is signed in.
//
// ⚠️ **The address is remembered, the person is remembered, nothing else is.**
// This is somebody's own phone — signing in every morning would be the fastest
// way to have the password written on the back of the case. The restaurant's
// address is set once, when a manager hands the app over.

@Immutable
sealed interface Session {
    data object Loading : Session
    data object NoServer : Session

    /** ⚠️ **Not the same as being signed out, and telling them apart is the
     *  whole of this state.** A phone opened in a basement or out of data
     *  reached the login screen — where the password is typed correctly, the
     *  request fails, and the message is "could not sign in". The cook then
     *  types it again, blaming themselves for a network they cannot see. */
    data class Offline(val address: String) : Session
    data class SignedOut(val address: String) : Session

    /** @param openShift the shift already running, if there is one. ⚠️ Held in
     *  the session rather than only inside the attendance screen: it decides
     *  what the one button on this app says, and a screen that had to fetch it
     *  before it could draw would flash the wrong word first. */
    data class Ready(
        val address: String,
        val staff: Staff,
        val onShift: Boolean,
    ) : Session
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
                val me = api.me()
                _state.value = Session.Ready(address, me.staff, me.openShift != null)
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
        tokens.write(TokenStore.STAFF_TOKEN, res.token)
        // ⚠️ Asked rather than assumed closed: somebody who reinstalled mid-shift
        // would otherwise be shown "start the shift" over a shift that is open,
        // and pressing it is a second punch the panel has to unpick.
        val open = runCatching { api.me().openShift != null }.getOrDefault(false)
        _state.value = Session.Ready(address, res.staff, open)
    }

    fun signOut(address: String) {
        tokens.drop(TokenStore.STAFF_TOKEN)
        _state.value = Session.SignedOut(address)
    }

    /** ⚠️ Kept apart from signing out. Forgetting the restaurant is what happens
     *  when a phone moves to another job; signing out is the end of a day, and
     *  answering both with one button would make the common one cost the rare
     *  one's setup — and the rare one is destructive in a way the daily one is
     *  not. */
    fun forgetServer() {
        tokens.drop(TokenStore.STAFF_TOKEN)
        tokens.drop(TokenStore.SERVER_ADDRESS)
        api.useServer("")
        _state.value = Session.NoServer
    }

    /** The shift opened or closed. ⚠️ Told rather than re-fetched: the punch's
     *  own answer is the newest fact there is, and a second request would put a
     *  network round trip between a press and the word on the button. */
    fun setOnShift(open: Boolean) {
        val s = _state.value
        if (s is Session.Ready) _state.value = s.copy(onShift = open)
    }
}
