// The visitor's answer to the cookie notice.
//
// ⚠️ **"Decline" has to switch something off, or the button is a visible lie.**
//
// This site sets four things: the chosen language, the chosen brand and branch, the cart and
// the session. None of them can be declined — without them the cart empties on the next page
// and signing in does not stick — so a banner offering to refuse them would be offering
// something it cannot do.
//
// What *is* optional is the visit counter (see TrackVisit): an anonymous, daily-hashed page
// view that exists so a restaurant can tell "nobody wants this" from "nobody can find us". So
// declining stops that beacon, and the notice says exactly that rather than implying the site
// will run on no storage at all.
//
// ⚠️ Kept in `localStorage`, not in a cookie. A cookie recording that somebody refused cookies
// is the joke every consent library makes without noticing, and this one has a real
// alternative.

const KEY = "cookie_choice_v1";

export type CookieChoice = "accepted" | "declined" | null;

export function readCookieChoice(): CookieChoice {
  if (typeof window === "undefined") return null;
  try {
    const v = window.localStorage.getItem(KEY);
    return v === "accepted" || v === "declined" ? v : null;
  } catch {
    // Storage refused: treated as "not answered yet", which shows the notice again. The
    // alternative — remembering nothing and counting anyway — is the one that breaks a
    // promise.
    return null;
  }
}

export function writeCookieChoice(choice: Exclude<CookieChoice, null>) {
  try {
    window.localStorage.setItem(KEY, choice);
  } catch {
    /* nothing to remember it with; the notice shows again next visit */
  }
  // So anything already mounted — the beacon in particular — reacts without a reload.
  window.dispatchEvent(new CustomEvent("keel-cookie-choice", { detail: choice }));
}

/** Whether the optional counting may run. ⚠️ Unanswered means **no**: counting somebody before
 *  they have been asked is the thing the notice exists to avoid. */
export function countingAllowed(): boolean {
  return readCookieChoice() === "accepted";
}
