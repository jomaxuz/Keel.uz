package uz.keel.app

import androidx.compose.runtime.Composable
import androidx.compose.runtime.Stable

/** The ways out of a role, as Keel answers them.
 *
 *  ⚠️ **Handed in, not reached for.** A role's code cannot know whether the
 *  person holding the phone has one account or three, so it cannot decide
 *  where "sign out" leads. Before Keel, every one of these ended on that app's
 *  own login screen; here the answer depends on what else is signed in, and
 *  only the shell can see that. */
@Stable
class RoleExit(
    /** This account off this phone. ⚠️ The push row is dropped before the
     *  token — the other order sends the request unauthenticated and the row
     *  stays, delivering tomorrow's alerts to whoever holds the phone next. */
    val signOut: () -> Unit,
    /** Every account out and the restaurant forgotten — a phone moving on. */
    val leaveRestaurant: () -> Unit,
    /** The server refused this account's token: it expired, or somebody in the
     *  office switched the account off. Answered as a sign-in, never an error
     *  screen — both have the same answer for the person holding the phone. */
    val expired: () -> Unit,
    /** Who this is, and the way to the other workspaces — first on the role's
     *  settings screen. */
    val card: @Composable () -> Unit,
)
