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

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { API_URL } from "@/lib/api";

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
    // Admin, courier and staff screens are the business using its own tools,
    // not a customer visiting. Counting them would make a quiet week look
    // busy to the one person who must not be misled about that.
    if (/^\/(admin|kuryer|staff|kiosk)/.test(pathname)) return;

    const body = JSON.stringify({ vid: visitorId(), path: pathname });
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
