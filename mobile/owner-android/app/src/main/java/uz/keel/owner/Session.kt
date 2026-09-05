package uz.keel.owner

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import uz.keel.design.ServerAddress
import uz.keel.design.TokenStore
import uz.keel.owner.data.AdminUser
import uz.keel.owner.data.ApiError
import uz.keel.owner.data.Branch
import uz.keel.owner.data.KeelApi

// This sitting on this phone: which restaurant, and who is signed in.
//
// ⚠️ **Four states, not a boolean.** "Which restaurant" and "who is signed in"
// are separate questions with separate answers: a phone that moves to another
// job forgets the restaurant, and signing out does not.

@Immutable
sealed interface Session {
    data object Loading : Session
    data object NoServer : Session

    /** ⚠️ **Not the same as being signed out, and telling them apart is the
     *  whole of this state.** A launch with no network used to land on the
     *  password field, where the right password fails and the app blames the
     *  person for a network they cannot see. Nothing here is fixable by signing
     *  in, so nothing here offers to. */
    data class Offline(val address: String) : Session
    data class SignedOut(val address: String) : Session

    /** @param branches what this account may look at.
     *
     *  ⚠️ **Fetched with the session, not per screen.** The lens sits above four
     *  screens and every one of them would otherwise ask for the list — and a
     *  manager, whom the server clamps to one branch, must never be offered a
     *  picker that implies otherwise. */
    data class Ready(
        val address: String,
        val user: AdminUser,
        val branches: List<Branch>,
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
                _state.value = Session.Ready(address, me, branchesFor(me))
            } catch (e: ApiError) {
                _state.value = Session.SignedOut(address)
            } catch (e: Throwable) {
                _state.value = Session.Offline(address)
            }
        }
    }

    /** ⚠️ **A failure here is not a failure to sign in.** The branch list is a
     *  convenience above the numbers; an owner whose list did not load still has
     *  their takings, and refusing the session over it would lock somebody out
     *  of the app over a lens they may not even have. */
    private suspend fun branchesFor(me: AdminUser): List<Branch> =
        runCatching { api.branches().filter { it.isActive } }.getOrDefault(emptyList())

    fun useServer(address: String): Boolean {
        if (ServerAddress.apiBase(address).isEmpty()) return false
        tokens.write(TokenStore.SERVER_ADDRESS, address)
        api.useServer(address)
        _state.value = Session.SignedOut(address)
        return true
    }

    suspend fun signIn(address: String, username: String, password: String) {
        val res = api.login(username, password)
        tokens.write(TokenStore.ADMIN_TOKEN, res.token)
        _state.value = Session.Ready(address, res.user, branchesFor(res.user))
    }

    fun signOut(address: String) {
        tokens.drop(TokenStore.ADMIN_TOKEN)
        _state.value = Session.SignedOut(address)
    }

    /** ⚠️ Kept apart from signing out. Forgetting the restaurant is what happens
     *  when a phone moves on; signing out is the end of a day, and answering
     *  both with one button would make the common one cost the rare one's setup
     *  — and the rare one is destructive in a way the daily one is not. */
    fun forgetServer() {
        tokens.drop(TokenStore.ADMIN_TOKEN)
        tokens.drop(TokenStore.SERVER_ADDRESS)
        api.useServer("")
        _state.value = Session.NoServer
    }
}
