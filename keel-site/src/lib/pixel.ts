// The Meta (Facebook) pixel on keel.uz.
//
// ⚠️ **keel.uz only.** Tenant restaurant sites are a different build and carry
// no advertising script at all. A guest ordering lag'mon from a restaurant did
// not come here to be measured by us, and the restaurant did not agree to hand
// its customers to our ad account. This file exists because *we* advertise
// *ourselves*, which is a different relationship entirely.
//
// ⚠️ **It loads with the page, not behind a button.** The cookie notice is a
// notice: it tells the visitor cookies are in use and links to the policy that
// says which. It is not a gate, and nothing here waits on it — a decision the
// owner took deliberately, because a gated pixel measures only the half of the
// audience that clicks, and half a measurement is what makes an ad budget get
// spent on the wrong thing. The disclosure lives in the notice and in the
// privacy policy; both have to keep saying so.

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
  //
  // ⚠️ Their copy-paste block also carries a <noscript> tracking image. It is
  // deliberately not here: it would fire from the raw HTML, before this file
  // has any say, which makes every rule above unenforceable the day somebody
  // decides the pixel should wait for something after all.
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
