package uz.keel.waiter.push

import uz.keel.app.push.KeelPush

// Being told something — the waiter's half.
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

/** What happened when this phone tried to subscribe.
 *
 *  ⚠️ **Every failure here used to be silent, and that was right for the person
 *  and wrong for everybody else.** An app that shows an error about notifications
 *  on the screen somebody is taking an order on is worse than the missing
 *  feature — but with nothing recorded anywhere, "the kitchen pressed ready and
 *  nothing arrived" has five possible causes and no way to tell them apart. So it
 *  is not shown; it is *available*, on the settings screen, where somebody goes
 *  when they are already asking the question. */
enum class PushState(val key: String) {
    Working("working"),
    Asking("asking"),
    Denied("denied"),
    NoDevice("noDevice"),
    NoProject("noProject"),
    Failed("failed"),
}

class PushRegistration(
    val state: PushState,
    val retry: () -> Unit,
    /** ⚠️ Awaited by the sign-out path **before** the token is cleared, or the
     *  request goes out unauthenticated and the row stays — sending the next
     *  evening's tables to whoever went home. */
    val forget: suspend () -> Unit,
)

/** Keel's registration, in this role's words. A state this role never had a
 *  word for reads as a failure, with the detail that says which. */
fun KeelPush.forRole(): PushRegistration = PushRegistration(
    state = PushState.entries.firstOrNull { it.key == key } ?: PushState.Failed,
    retry = retry,
    forget = {},
)
