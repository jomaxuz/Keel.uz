// The two drawings the dead-end pages use.
//
// ⚠️ **Drawn here rather than uploaded**, for the same reason the alert chime is
// synthesised: an illustration served as a file is a 404 inside a 404 on the one
// page that must never fail, and it would be missing on exactly the customer
// whose upload volume is the thing that broke.
//
// They are line art in `currentColor` with one accent, so they follow the
// restaurant's own theme and read correctly in dark mode without a second copy —
// a flat picture would keep its own background and sit on the page as a pale
// rectangle the moment somebody switches the site to dark.

/** An empty plate with the cutlery set aside: the page asked for is not on the
 *  menu. Chosen over a broken-robot or a big "404" because the visitor is
 *  standing in a restaurant, and the joke should be the restaurant's. */
export function EmptyPlateArt({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 200 140"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      {/* the plate */}
      <ellipse cx="100" cy="78" rx="52" ry="34" className="text-ink/25" />
      <ellipse cx="100" cy="78" rx="36" ry="23" className="text-ink/15" />
      {/* fork, left: three tines meeting a neck, then the handle. Drawn as one
          continuous outline rather than as separate strokes — tines that stop in
          mid-air read as a broken drawing, which on a page about something being
          missing is the wrong joke twice. */}
      <path d="M34 42v18a6 6 0 0 0 6 6h0a6 6 0 0 0 6-6V42" />
      <path d="M40 42v18" />
      <path d="M40 66v46" />
      {/* knife, right */}
      <path d="M160 42c7 9 7 22 0 30v40" />
      {/* the steam that is not there any more */}
      <path
        d="M86 40c4-5-4-9 0-14M100 34c4-5-4-9 0-14M114 40c4-5-4-9 0-14"
        className="text-brand/60"
      />
    </svg>
  );
}

/** A pot that has boiled over. Used for the error page rather than the 404,
 *  because the two failures are genuinely different: one is "you asked for
 *  something that is not here", the other is "we broke". */
export function BoiledOverArt({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 200 140"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      {/* pot */}
      <path d="M56 66h88v34a14 14 0 0 1-14 14H70a14 14 0 0 1-14-14z" />
      <path d="M48 66h104" />
      <path d="M56 78H44M144 78h12" />
      {/* lid, knocked askew — the one line that says something went wrong */}
      <path d="M74 60l58-10" className="text-brand" />
      <path d="M100 47l3-6" className="text-brand" />
      {/* what came out */}
      <path
        d="M76 66c2-8 8-8 10-2M104 66c2-10 10-9 12-1"
        className="text-brand/70"
      />
      {/* heat */}
      <path d="M70 126h60" className="text-brand/50" />
    </svg>
  );
}
