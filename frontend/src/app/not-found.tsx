"use client";

// The page a guest lands on after a stale QR card, a mistyped address or a link
// that outlived the dish it pointed at.
//
// ⚠️ **At the app root, not inside `(site)`.** Next only reaches a route group's
// boundary for URLs that matched a route in it; an address that matches nothing
// at all is answered here, which is also why this page draws its own way back
// rather than relying on the site header — the group's layout is not applied.
//
// ⚠️ **A client component so it is translated.** The language is already in the
// provider the root layout mounts; reading it again from headers here would make
// `/_not-found` a dynamic route for every visitor and buy nothing. A 404 in the
// wrong language is how a Russian guest concludes the site is not merely missing
// a page but broken.

import DeadEnd from "@/components/site/DeadEnd";
import { EmptyPlateArt } from "@/components/site/ErrorArt";
import { useI18n } from "@/lib/i18n/client";

export default function NotFound() {
  const { t } = useI18n();
  return (
    <DeadEnd
      art={<EmptyPlateArt className="w-full" />}
      title={t.errors.notFoundTitle}
      text={t.errors.notFoundText}
    />
  );
}
