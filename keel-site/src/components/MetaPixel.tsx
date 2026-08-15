"use client";

// Loads the Meta pixel and reports the two events this site sends.
// See `lib/pixel.ts` for why it does not wait for the cookie notice.

import { useEffect, useRef } from "react";
import { usePathname } from "next/navigation";
import { TELEGRAM } from "@/lib/links";
import { loadPixel, pageView, trackLead } from "@/lib/pixel";

export default function MetaPixel({ id }: { id?: string }) {
  const pathname = usePathname();
  // The pixel's own `init` already sent one PageView. Without this the first
  // navigation effect would send a second for the same view.
  const started = useRef(false);

  useEffect(() => {
    if (!id) return;
    loadPixel(id);
    started.current = true;
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
