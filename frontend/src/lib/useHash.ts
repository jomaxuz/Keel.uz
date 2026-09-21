"use client";

import { useEffect, useState } from "react";

/** The fragment the browser is showing, kept current however it changed.
 *
 *  ⚠️ **`usePathname` cannot see a fragment, and `hashchange` does not fire for
 *  the way a Next.js site actually moves between them.** The router navigates
 *  with `history.pushState`, which changes the address bar synchronously and
 *  emits no event at all. So a bar that reads the fragment once keeps whatever
 *  it read: click «Erkaklar», then click «Katalog», and the marker stays on
 *  «Erkaklar» while the URL says otherwise — which is the bug this hook exists
 *  to close, found by clicking through the shop rather than by reasoning.
 *
 *  The two history methods are therefore wrapped **once for the document**, and
 *  they call through untouched — the wrapper only announces what already
 *  happened. `hashchange` and `popstate` stay because they cover what the
 *  wrapper cannot: the back button, and a fragment typed into the address bar.
 *
 *  ⚠️ Starts empty and fills in after mount: the server rendered no fragment
 *  either, and any other first value is a hydration mismatch on every page that
 *  has a section link. */
export function useHash(): string {
  const [hash, setHash] = useState("");
  useEffect(() => {
    const read = () => setHash(window.location.hash);
    read();
    patchHistory();
    window.addEventListener("hashchange", read);
    window.addEventListener("popstate", read);
    window.addEventListener(LOCATION_CHANGE, read);
    return () => {
      window.removeEventListener("hashchange", read);
      window.removeEventListener("popstate", read);
      window.removeEventListener(LOCATION_CHANGE, read);
    };
  }, []);
  return hash;
}

const LOCATION_CHANGE = "keel:locationchange";

/** Makes `pushState` and `replaceState` announce themselves.
 *
 *  ⚠️ Once per document, guarded on the window: patching twice would stack
 *  wrappers on every mount of every component that wants this, and the stack
 *  would grow for as long as the tab is open. */
function patchHistory() {
  const w = window as Window & { __keelHistoryPatched?: boolean };
  if (w.__keelHistoryPatched) return;
  w.__keelHistoryPatched = true;
  for (const name of ["pushState", "replaceState"] as const) {
    const original = history[name];
    history[name] = function patched(
      this: History,
      ...args: Parameters<History["pushState"]>
    ) {
      const out = original.apply(this, args);
      window.dispatchEvent(new Event(LOCATION_CHANGE));
      return out;
    };
  }
}
