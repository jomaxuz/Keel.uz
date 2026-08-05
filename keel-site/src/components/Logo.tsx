/** The mark: a hull in cross-section with the keel fin below it.
 *
 *  One stroke, no fill, drawn in currentColor — which is what lets the same
 *  file be the header logo, the favicon, and the tiny badge that sits in a
 *  customer's own footer next to whatever colour they chose. */
export function KeelMark({ className = "h-7 w-7" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      stroke="currentColor"
      strokeWidth={2.4}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      {/* The fin is the whole point of the mark — it is the keel. Drawn long
          enough to survive at favicon size, where a short stub reads as a
          smudge and the logo becomes an anonymous curve. */}
      <path d="M5 6c0 9.5 4.4 14.5 11 14.5S27 15.5 27 6" />
      <path d="M16 20.5V29" />
    </svg>
  );
}

export function Logo({ className = "" }: { className?: string }) {
  return (
    <span className={`inline-flex items-center gap-2 ${className}`}>
      <KeelMark className="h-7 w-7 text-signal-500" />
      <span className="font-display text-xl font-semibold tracking-tight text-ink">
        keel
      </span>
    </span>
  );
}
