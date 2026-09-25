package uz.keel.design

import java.io.IOException
import java.util.concurrent.ExecutionException
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay

// Asking Firebase for this phone's token, the way it has to be asked.
//
// ⚠️ **`SERVICE_NOT_AVAILABLE` is "not yet", not "no".** It is Play services
// saying it could not reach Google this minute — the phone just woke, the wifi
// is still associating, Play services is updating itself. Asked once, the first
// launch after an install routinely lands on it, and the settings screen then
// shows `java.util.concurrent.ExecutionException: java.io.IOException:
// SERVICE_NOT_AVAILABLE` to somebody who did nothing wrong and whose
// notifications would have worked ten seconds later. So the transient codes are
// retried here, quietly, before anything is reported.
//
// ⚠️ **One copy, for all five applications.** Firebase is not a dependency of
// this module — the caller passes the fetch in — so the retry rule lives here
// and the SDK stays in the apps that ship it.

/** The codes Play services uses for "try again shortly". Everything else —
 *  `MISSING_INSTANCEID_SERVICE`, `AUTHENTICATION_FAILED`, a bad sender id — is
 *  the same answer on every attempt, and retrying it only delays the report. */
private val transientCodes = setOf("SERVICE_NOT_AVAILABLE", "TIMEOUT", "INTERNAL_SERVER_ERROR")

/** Seconds between attempts: about a minute in total, which covers a phone
 *  waking up on a weak network without keeping the settings screen on
 *  "checking…" long enough to look stuck. */
private val backoffMs = longArrayOf(2_000, 5_000, 15_000, 30_000)

/** Fetch the token, retrying the transient failures. Throws the last error,
 *  unwrapped, if every attempt fails. */
suspend fun fetchPushToken(fetch: suspend () -> String): String {
    var attempt = 0
    while (true) {
        try {
            return fetch()
        } catch (e: CancellationException) {
            throw e
        } catch (e: Throwable) {
            val root = unwrapPushError(e)
            if (attempt >= backoffMs.size || !isTransientPushError(root)) throw root
            delay(backoffMs[attempt++])
        }
    }
}

/** The error Play services actually raised. Tasks wrap it in an
 *  `ExecutionException`, which is the part nobody fixing the install needs. */
fun unwrapPushError(e: Throwable): Throwable {
    var cur = e
    while (cur is ExecutionException) cur = cur.cause ?: break
    return cur
}

fun isTransientPushError(e: Throwable): Boolean =
    e is IOException && e.message?.trim() in transientCodes

/** The line the settings screen shows under "not registered": the code alone
 *  (`SERVICE_NOT_AVAILABLE`), still untranslated so it can be searched for, but
 *  without two Java class names in front of it. */
fun pushErrorDetail(e: Throwable): String {
    val root = unwrapPushError(e)
    return root.message?.takeIf { it.isNotBlank() } ?: root::class.java.simpleName
}
