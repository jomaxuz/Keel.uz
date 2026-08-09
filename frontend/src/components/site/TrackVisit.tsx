"use client";

// One beacon per page view.
//
// The restaurant's dashboard could say how many orders arrived and nothing at
// all about how many people looked — which is the difference between "nobody
// wants this" and "nobody can find it", and those need opposite responses.
//
// Deliberately small:
//
//   • **No third-party analytics.** Loading somebody else's script onto every
//     customer's site means their visitors are tracked by a company neither of
//     us chose, and it is the single heaviest thing that could be added to a
//     page that has to open on a phone in a bazaar.
//   • **The id never leaves the browser in the clear** — the server hashes it
//     with the day before storing, so it cannot be used to follow anybody
//     across days. Counting visitors and tracking people are different jobs.
//   • **Fire and forget.** `keepalive` so a click away does not cancel it, and
//     every failure swallowed: a counter must never be something a guest sees.
//   • **Not gated on the cookie notice.** ⚠️ Nothing here identifies a person: the id is the
//     browser's own, it is hashed with the day before it is stored, and the rows expire. What
//     is left is a daily count of page views, which is the restaurant's own operating figure —
//     the same fact its door counter or its waiter would give it. The notice says the site
//     counts visits anonymously; it does not offer to stop, because there is nothing here to
//     stop that would protect anybody.

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { API_URL } from "@/lib/api";
import { isLocalizedPath, splitLangPath } from "@/lib/i18n";

const KEY = "visitor_id";

/** A random id, made in the browser and kept there. */
function visitorId(): string {
  try {
    let id = localStorage.getItem(KEY);
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem(KEY, id);
    }
    return id;
  } catch {
    // Private mode, or storage disabled. The server falls back to something
    // that lasts a day, which is all this needs.
    return "";
  }
}

export default function TrackVisit() {
  const pathname = usePathname();

  useEffect(() => {
    // The path without its language prefix, for both reasons it appears in.
    //
    // The counter answers "how many people looked at the menu" — one page, so
    // one row. Counting `/menu`, `/ru/menu` and `/en/menu` separately would
    // split every page three ways and make the busiest pages look like three
    // quiet ones.
    //
    // And the exclusion below is a path test: `/ru/admin` does not start with
    // `/admin`, so a hand-typed prefix would have started counting the owner's
    // own panel as customer traffic — the exact thing the exclusion exists to
    // prevent, back again through the new prefix.
    const path = splitLangPath(pathname).path;

    // Admin, courier and staff screens are the business using its own tools,
    // not a customer visiting. Counting them would make a quiet week look
    // busy to the one person who must not be misled about that.
    if (!isLocalizedPath(path)) return;

    const body = JSON.stringify({ vid: visitorId(), path });
    try {
      fetch(`${API_URL}/visit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body,
        keepalive: true,
      }).catch(() => {});
    } catch {
      /* nothing here is worth interrupting a page for */
    }
  }, [pathname]);

  return null;
}
