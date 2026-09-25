package uz.keel.design

import android.content.Context

// What this build calls itself.
//
// ⚠️ **The platform's number, not the app's own.** Every part of Keel — the
// landing, the console, a restaurant's dashboard, the Windows till — says the
// one version in the repo-root `VERSION` file, and a support call starts with
// "which version?". The apps used to carry two numbers each (`versionName` in
// Gradle and an `APP_VERSION` constant beside it), which is how the waiter said
// 1.0.0, the courier 2.0.0 and the dashboard v0.2.1 on the same day. Gradle now
// reads `VERSION` at build time and this reads what Gradle wrote into the APK,
// so the number shown is the number of the code that is running.
//
// ⚠️ Shown with the leading "v", like the dashboard, so the two strings a
// person compares on a support call are the same string.
fun appVersion(ctx: Context): String {
    val name = runCatching {
        ctx.packageManager.getPackageInfo(ctx.packageName, 0).versionName
    }.getOrNull()?.trim().orEmpty()
    if (name.isEmpty()) return "—"
    return if (name.startsWith("v")) name else "v$name"
}
