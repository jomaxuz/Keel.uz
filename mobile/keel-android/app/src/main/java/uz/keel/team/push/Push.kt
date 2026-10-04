package uz.keel.team.push

import uz.keel.app.push.KeelPush

// Being told something — the team's half.
//
// ⚠️ **Registration is Keel's now, not this role's.** In the single-role app this
// file asked for the permission, fetched the token, registered it and drew what
// arrived. In Keel one phone is up to three accounts on one Firebase token, and
// that changes two things this file used to get right on its own:
//  - **one receiver.** Firebase calls exactly one `MESSAGING_EVENT` service per
//    app — see `uz.keel.app.push.KeelMessagingService`;
//  - **signing out never deletes the token.** It belongs to every account still
//    signed in on the phone; deleting it on one sign-out would silence the
//    others with nothing anywhere saying so. See `uz.keel.app.push.PushHub`.
//
// What stays is the shape this role's settings screen reads.

/** ⚠️ **Four, where the Expo build had six.** "Not on a simulator" and "no
 *  project id" were facts about Expo's relay, and neither can happen here: this
 *  asks Firebase directly, and either it answers with a token or it does not. A
 *  state nothing can produce is a state somebody reads and tries to fix. */
enum class PushState(val key: String) {
    Working("working"), Asking("asking"), Denied("denied"), Failed("failed"),
}

class PushRegistration(
    val state: PushState,
    /** The raw reason it failed, for the settings screen.
     *
     *  ⚠️ **"Failed" was not enough, and the product already learned that.** The
     *  first person to run the Expo build on a real phone read "not registered ·
     *  could not reach the internet or the server", pressed retry, and got the
     *  same line — which is true of at least three completely different faults:
     *  this build has no push credentials, the server is older than the
     *  endpoint, or the phone is offline. None is fixed from the phone, and
     *  whoever *can* fix it needs to know which one it is.
     *
     *  ⚠️ **Deliberately not translated.** It is a diagnostic, read by whoever
     *  is fixing the install, and a translated HTTP status is a status nobody
     *  can search for. */
    val detail: String?,
    val retry: () -> Unit,
    val forget: suspend () -> Unit,
)

/** Keel's registration, in this role's words. A state this role never had a
 *  word for reads as a failure, with the detail that says which. */
fun KeelPush.forRole(): PushRegistration = PushRegistration(
    state = PushState.entries.firstOrNull { it.key == key } ?: PushState.Failed,
    detail = detail,
    retry = retry,
    forget = {},
)
