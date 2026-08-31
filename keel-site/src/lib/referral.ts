/** The referral code a visitor arrived with.
 *
 *  ⚠️ **A cookie for us, a visible code for them.** Signing up here is a
 *  Telegram conversation rather than a form, so nothing carries the code from
 *  this browser into the tenant record on its own. The cookie is what tells us
 *  the link was used at all; the code printed on the page is what the visitor
 *  quotes when the operator asks who sent them. An automatic hand-off would
 *  need a bot on the other end of the Telegram link — until there is one, an
 *  honest manual step beats a broken automatic one.
 */
export const REF_COOKIE = "keel_ref";

/** A year. The gap between "saw the leaflet" and "wrote to us" is measured in
 *  weeks: it gets left on a counter and picked up when the current way of doing
 *  things annoys somebody enough. */
export const REF_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

/** The parameter the `/h/<kod>` route redirects with.
 *
 *  ⚠️ A query string here and a path segment there, on purpose: this one only
 *  has to survive one hop between two pages of ours. The address a person
 *  types, reads aloud or pastes is the path. */
export const REF_PARAM = "h";

/** A code as it may arrive from a URL — trimmed to the alphabet the control
 *  plane will accept, so nothing further down has to guard against a code that
 *  was never valid. */
export function cleanRefCode(raw: string | undefined): string {
  return (raw ?? "").toLowerCase().replace(/[^a-z0-9-]/g, "").slice(0, 24);
}
