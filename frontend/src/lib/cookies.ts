// Whether the visitor has seen the cookie notice.
//
// ⚠️ **A flag, not a choice.** The notice on this site tells people what is stored; it does not
// ask permission, because everything stored is needed for the site to work: the chosen
// language, the chosen brand and branch, the cart and the session. There is no advertising
// pixel, no third-party analytics and no profile — the only thing beyond that is an anonymous
// page count whose id is hashed with the day, so it cannot follow anybody from one day to the
// next even for us.
//
// So there is one button, and it means "read". A second button labelled "decline" would run
// exactly the same code, and a control that changes nothing teaches people that consent
// controls are decoration.
//
// ⚠️ Kept in `localStorage`, not in a cookie: a cookie recording that somebody was told about
// cookies is the joke every consent library makes without noticing.

const KEY = "cookie_notice_v1";

/** Whether the notice has already been acknowledged on this device. */
export function noticeSeen(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(KEY) === "seen";
  } catch {
    // Storage refused (private mode). Shown again next visit — the honest failure, and it
    // costs a guest one dismissal rather than hiding what the site stores.
    return false;
  }
}

export function markNoticeSeen() {
  try {
    window.localStorage.setItem(KEY, "seen");
  } catch {
    /* nothing to remember it with */
  }
}
