"use client";

// Loads the Meta pixel once the visitor has said yes, and reports the two
// events this site sends. See `lib/pixel.ts` for why none of it runs before
// that answer exists.

import { useEffect, useRef } from "react";
import { usePathname } from "next/navigation";
import { TELEGRAM } from "@/lib/links";
import { loadPixel, pageView, readConsent, trackLead } from "@/lib/pixel";

/** Broadcast by the cookie notice when the visitor answers, so the pixel can
 *  start in the same visit rather than on the next page load. Without it,
 *  everybody who accepts is invisible until they navigate — and on a
 *  single-page landing, most of them never do. */
export const CONSENT_EVENT = "keel-consent-change";

export default function MetaPixel({ id }: { id?: string }) {
  const pathname = usePathname();
  // The pixel's own `init` already sent one PageView. Without this the first
  // navigation effect would send a second for the same view.
  const started = useRef(false);

  useEffect(() => {
    if (!id) return;
    const start = () => {
      if (readConsent() !== "granted") return;
      loadPixel(id);
      started.current = true;
    };
    start();
    window.addEventListener(CONSENT_EVENT, start);
    return () => window.removeEventListener(CONSENT_EVENT, start);
  }, [id]);

  // Client-side navigations. Next does not reload the document, so Meta would
  // otherwise record the whole site as a single view of the entry page — and
  // "time on page" would then describe a session rather than a page.
  useEffect(() => {
    if (!id || !started.current) return;
    pageView();
  }, [id, pathname]);

  // ⚠️ One delegated listener rather than an onClick on every Telegram button.
  // There are five of them across the page, the header and the footer, and the
  // next one added would silently not be counted — a conversion that stops
  // being reported does not look broken, it looks like a worse ad.
  useEffect(() => {
    if (!id) return;
    const onClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement | null;
      const link = target?.closest?.("a");
      if (link && link.getAttribute("href")?.startsWith(TELEGRAM)) trackLead();
    };
    document.addEventListener("click", onClick, true);
    return () => document.removeEventListener("click", onClick, true);
  }, [id]);

  return null;
}
