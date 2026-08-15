// The Meta (Facebook) pixel on keel.uz — consent, loading, and the two events
// this site actually sends.
//
// ⚠️ **keel.uz only.** Tenant restaurant sites are a different build and carry
// no advertising script at all. A guest ordering lag'mon from a restaurant did
// not come here to be measured by us, and the restaurant did not agree to hand
// its customers to our ad account. This file exists because *we* advertise
// *ourselves*, which is a different relationship entirely.
//
// ⚠️ **Nothing loads before consent.** The script is injected from an effect
// rather than rendered into the document, precisely so that "no answer yet"
// and "declined" produce the same result: no request to Meta, no cookie, no
// identifier. A consent banner that runs the tracker while it asks is not a
// consent banner, and the previous version of this site was honest about
// having no tracker at all — that honesty is the thing being preserved here,
// not a checkbox.

/** Where the answer lives. ⚠️ A new key rather than the old notice's
 *  `cookie_notice_v1`: that banner said, truthfully at the time, that there
 *  was nothing to switch off. An answer to that sentence is not an answer to
 *  this one, so reusing the key would turn "I read your notice" into "I agreed
 *  to be tracked" — retroactively, and for the people who trusted the earlier
 *  wording most. */
const CONSENT_KEY = "keel_ads_consent_v1";

export type Consent = "granted" | "denied" | "unanswered";

export function readConsent(): Consent {
  if (typeof window === "undefined") return "unanswered";
  try {
    const v = window.localStorage.getItem(CONSENT_KEY);
    return v === "granted" || v === "denied" ? v : "unanswered";
  } catch {
    // Storage refused (private mode, blocked cookies). Treated as unanswered,
    // which means the pixel stays off — the safe direction of the two.
    return "unanswered";
  }
}

export function writeConsent(value: "granted" | "denied") {
  try {
    window.localStorage.setItem(CONSENT_KEY, value);
  } catch {
    /* Nothing to remember it with; the banner returns next visit. */
  }
}

type Fbq = ((...args: unknown[]) => void) & {
  callMethod?: (...args: unknown[]) => void;
  queue?: unknown[];
  push?: unknown;
  loaded?: boolean;
  version?: string;
};

declare global {
  interface Window {
    fbq?: Fbq;
    _fbq?: Fbq;
  }
}

let loadedFor: string | null = null;

/** Injects Meta's base code and sends the first PageView.
 *
 *  Idempotent: React runs effects twice in development, and a second `init`
 *  would double every number this campaign is judged by — the kind of error
 *  that looks like good news. */
export function loadPixel(id: string) {
  if (typeof window === "undefined" || !id || loadedFor === id) return;
  loadedFor = id;

  // Meta's official snippet, transcribed rather than string-injected so that
  // it type-checks and so nobody has to read minified code to see what runs.
  const fbq: Fbq = function (...args: unknown[]) {
    if (fbq.callMethod) fbq.callMethod.apply(fbq, args);
    else fbq.queue?.push(args);
  } as Fbq;
  if (!window._fbq) window._fbq = fbq;
  fbq.push = fbq;
  fbq.loaded = true;
  fbq.version = "2.0";
  fbq.queue = [];
  window.fbq = fbq;

  const s = document.createElement("script");
  s.async = true;
  s.src = "https://connect.facebook.net/en_US/fbevents.js";
  document.head.appendChild(s);

  window.fbq("init", id);
  window.fbq("track", "PageView");
}

export function pageView() {
  window.fbq?.("track", "PageView");
}

/** The one conversion that matters here.
 *
 *  ⚠️ Fired on the Telegram button, not on the pricing section or a scroll
 *  depth. Meta optimises towards whatever you report as a Lead, so reporting
 *  something cheap teaches it to find people who do the cheap thing: report
 *  "scrolled to pricing" and the ad account will faithfully buy scrollers.
 *  Opening the conversation is the only action on this page that has ever
 *  turned into a customer. */
export function trackLead() {
  window.fbq?.("track", "Lead");
}
