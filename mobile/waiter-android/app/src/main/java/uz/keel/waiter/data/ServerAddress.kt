package uz.keel.waiter.data

// Turning what somebody typed into the address of a Keel server.
//
// ⚠️ **A third implementation, and it is written down rather than hidden.** The
// rule already lives in TypeScript (`lib/serverAddress.ts`) and in Go
// (`backend/desktop/config_windows.go` → `apiBase`), because the Windows till
// resolves it before any JavaScript runs. There is nothing to import across
// these boundaries, so the copies are kept honest the only way they can be: one
// set of examples, tested on every side.
//
// ⚠️ **The whole point is that a restaurant types the short thing.** Whoever
// sets up a phone knows their restaurant as "osh", not as
// "https://osh.keel.uz/api/v1" — a form demanding the long form is a form
// answered with a guess.

object ServerAddress {
    /** The API base for what was typed, or "" when nothing usable was. */
    fun apiBase(address: String): String =
        host(address).let { if (it.isEmpty()) "" else "https://$it/api/v1" }

    /** Where this restaurant's pictures come from. Same host, and derived
     *  rather than asked: two fields are two chances to mistype one answer. */
    fun uploadsBase(address: String): String =
        host(address).let { if (it.isEmpty()) "" else "https://$it/uploads" }

    private fun host(address: String): String {
        // ⚠️ Pasted addresses arrive with all of these. Refusing them would be
        // technically defensible and would fail the one person most likely to
        // paste: somebody who copied the link out of the panel.
        var a = address.trim().lowercase()
        a = a.removePrefix("https://").removePrefix("http://")
        a = a.trimEnd('/')
        a = a.removeSuffix("/api/v1").trimEnd('/')
        if (a.isEmpty()) return ""
        // ⚠️ Whitespace or a slash inside means this is not a host at all — a
        // sentence, or a deep link somebody copied. Better to refuse than to
        // build an address that resolves to nothing and fails as "no internet".
        if (a.any { it.isWhitespace() || it == '/' }) return ""
        // A bare word is a Keel subdomain; anything with a dot is its own host.
        return if (a.contains('.')) a else "$a.keel.uz"
    }
}

/** Resolve a stored image path to a URL, at the size the screen actually shows.
 *
 *  ⚠️ **Always pass a width for anything in a list.** The backend resizes on
 *  request and caches the result (`?w=300|600|1200`); a restaurant home page was
 *  once 2.36 MB, and 1.83 MB of it was sixteen photographs served full size into
 *  cards 350 px wide.
 *
 *  ⚠️ **An absolute URL can still be ours, and usually is.** The upload handler
 *  stores `PUBLIC_BASE_URL + /uploads/<name>`, so every photograph an owner
 *  uploads is absolute — only seeded demo images are relative. Ours is decided
 *  by the path (`/uploads/…`), never by the host: a tenant reads its own images
 *  over `<slug>.keel.uz` *and* over its own domain. A pasted third-party link
 *  has no `/uploads/` path and is left alone, which also protects signed CDN
 *  URLs where an extra query parameter breaks the signature. */
fun imageUrl(path: String?, uploadsBase: String, width: Int? = null): String? {
    if (path.isNullOrBlank()) return null
    val q = if (width != null) "?w=$width" else ""
    if (path.startsWith("http://") || path.startsWith("https://")) {
        if (width == null) return path
        val cut = path.indexOf("/uploads/")
        if (cut < 0) return path
        val sep = if (path.contains('?')) "&" else "?"
        return "$path${sep}w=$width"
    }
    if (path.startsWith("/uploads/")) return uploadsBase + path.removePrefix("/uploads") + q
    return "$uploadsBase/${path.trimStart('/')}$q"
}

/** Whole so'm, grouped every three digits.
 *
 *  ⚠️ **Not a locale formatter.** Android's ICU data varies by version and
 *  vendor, so the same total is grouped on one phone and not on the next — and
 *  two spellings of one number on two waiters' phones read as two numbers. This
 *  is how every other Keel screen prints money, which is the point: a guest
 *  compares the phone against the paper. */
fun money(n: Double): String {
    val s = Math.round(n).toString()
    val neg = s.startsWith("-")
    val digits = if (neg) s.substring(1) else s
    val out = StringBuilder()
    for ((i, ch) in digits.withIndex()) {
        if (i > 0 && (digits.length - i) % 3 == 0) out.append(' ')
        out.append(ch)
    }
    return if (neg) "-$out" else out.toString()
}
