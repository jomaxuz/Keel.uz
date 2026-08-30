"use client";

// Something threw while rendering a page: the backend was unreachable, a
// response came back in a shape nothing expected, a component crashed.
//
// ⚠️ **The difference from the 404 is not decorative.** "This is not here" and
// "we broke" ask different things of the guest: the first one is answered by
// going somewhere else, the second by waiting a moment and trying again. A
// single "something went wrong" page for both teaches people that the retry
// button never works, and then they do not press it on the day it would have.
//
// Next requires this to be a client component and hands it a `reset()` that
// re-renders the segment — a real retry, not a page reload, so the cart and the
// session survive it.

import { useEffect } from "react";
import DeadEnd from "@/components/site/DeadEnd";
import { BoiledOverArt } from "@/components/site/ErrorArt";
import { useI18n } from "@/lib/i18n/client";
import { report } from "@/lib/report";

export default function SiteError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const { t } = useI18n();

  useEffect(() => {
    console.error("[site]", error);
    // ⚠️ **This note used to say the opposite, and the reason it changed is
    // worth keeping.** It argued that posting errors anywhere central would
    // make every guest's browser a client of ours, and that we ran no place to
    // put them. The second half is no longer true — see lib/report.ts and the
    // console's Reports screen — and the first was answered rather than
    // ignored: the guest's browser posts to the restaurant's own server, which
    // forwards with a credential the browser never sees. The digest below is
    // still what the guest reads out; it is now also on our screen.
    report(error, { where: "site", context: error.digest });
  }, [error]);

  return (
    <DeadEnd
      art={<BoiledOverArt className="w-full" />}
      title={t.errors.serverTitle}
      text={t.errors.serverText}
      code={error.digest}
      onRetry={reset}
    />
  );
}
