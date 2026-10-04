package uz.keel.waiter

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import uz.keel.waiter.data.ApiError
import uz.keel.waiter.data.KeelApi
import uz.keel.design.ServerAddress
import uz.keel.waiter.data.Staff
import uz.keel.design.TokenStore

// This sitting on this phone: which restaurant, and who is signed in.
//
// ⚠️ **The address is remembered, the person is remembered, the unlock is not.**
// A waiter's phone is theirs — signing in every shift is the fastest way to have
// the password written on the back of the case. The restaurant's address is set
// once, when the phone is handed over.
//
// ⚠️ **Four states, not a boolean.** "Which restaurant" and "who is signed in"
// are separate questions with separate answers: a phone that moves to another
// job forgets the restaurant, and the end of a shift does not. Collapsing them
// would make the common action cost the rare one's setup.

@Immutable
sealed interface Session {
    data object Loading : Session
    data object NoServer : Session

    /** ⚠️ **Not the same as being signed out, and telling them apart is the
     *  whole of this state.** A phone opened in a basement, on a dead wifi or
     *  out of data used to reach the login screen — where the password is typed
     *  correctly, the request fails, and the message is "could not sign in". The
     *  waiter types it again, and again, blaming themselves for a network they
     *  cannot see. Nothing here can be fixed by signing in, so nothing here
     *  offers to. */
    data class Offline(val address: String) : Session
    data class SignedOut(val address: String) : Session
    /** @param shiftOpen whether this employee is clocked in right now.
     *
     *  ⚠️ **Part of the session rather than fetched by the floor screen.** It
     *  gates opening a table, so it has to be known before the room is drawn —
     *  and asking for it per tap would make the refusal arrive after the wait. */
    data class Ready(
        val address: String,
        val staff: Staff,
        val shiftOpen: Boolean,
    ) : Session
}

class SessionViewModel(
    private val api: KeelApi,
    private val tokens: TokenStore,
) : ViewModel() {

    private val _state = MutableStateFlow<Session>(Session.Loading)
    val state: StateFlow<Session> = _state.asStateFlow()

    init { probe() }

    /** Ask the server who this is. The one question a launch has to answer.
     *
     *  ⚠️ **Three outcomes, not two.** "The server says no" and "the server did
     *  not answer" look identical from a `catch` and mean opposite things: one
     *  is a sign-in, the other is a network. `ApiError` is the server having
     *  spoken — any status, including 401 — and anything else is the request
     *  never having arrived. */
    fun probe() {
        viewModelScope.launch {
            val address = tokens.serverAddress
            if (address.isNullOrBlank()) {
                _state.value = Session.NoServer
                return@launch
            }
            api.useServer(address)
            try {
                val me = api.staffMe()
                _state.value = Session.Ready(address, me.staff, me.openShift != null)
            } catch (e: ApiError) {
                // ⚠️ A refused token is a sign-in, not an error screen. It
                // expires, or the account was switched off — and both have the
                // same answer for the person holding the phone.
                _state.value = Session.SignedOut(address)
            } catch (e: Throwable) {
                _state.value = Session.Offline(address)
            }
        }
    }

    fun useServer(address: String): Boolean {
        if (ServerAddress.apiBase(address).isEmpty()) return false
        tokens.serverAddress = address
        api.useServer(address)
        _state.value = Session.SignedOut(address)
        return true
    }

    /** ⚠️ Throws the server's own words. "Wrong password" and "this account is
     *  switched off" send somebody to two different people, and a network
     *  failure is neither — the screen tells all three apart. */
    suspend fun signIn(address: String, username: String, password: String) {
        val res = api.staffLogin(username, password)
        tokens.staffToken = res.token
        // ⚠️ Asked again rather than assumed false: somebody signing in mid-shift
        // — a phone that died, a reinstall — is already clocked in, and telling
        // them to start a shift they are standing in the middle of is worse than
        // not gating at all.
        val open = runCatching { api.staffMe().openShift != null }.getOrDefault(false)
        _state.value = Session.Ready(address, res.staff, open)
    }

    /** Re-read after clocking in or out, so the floor stops refusing.
     *
     *  ⚠️ Cheap and immediate: the alternative is a waiter who has just started
     *  their shift being told to start their shift. */
    fun refreshShift() {
        viewModelScope.launch {
            val cur = _state.value as? Session.Ready ?: return@launch
            runCatching { api.staffMe() }.onSuccess {
                _state.value = cur.copy(staff = it.staff, shiftOpen = it.openShift != null)
            }
        }
    }

    fun signOut(address: String) {
        tokens.staffToken = null
        _state.value = Session.SignedOut(address)
    }

    /** ⚠️ Kept apart from signing out. Forgetting the restaurant is what happens
     *  when a phone moves to another job; signing out is the end of a shift, and
     *  answering both with one button makes the common one cost the rare one's
     *  setup — and the rare one is destructive in a way the daily one is not. */
    fun forgetServer() {
        tokens.staffToken = null
        tokens.serverAddress = null
        api.useServer("")
        _state.value = Session.NoServer
    }
}
