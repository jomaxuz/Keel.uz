/** The Keel mark: a hull in cross-section with the keel fin below it.
 *
 *  ⚠️ **Copied from keel-site rather than imported**, exactly as the chart
 *  palette is: these are two separate builds, and one restaurant's bundle has
 *  no business reaching into the platform's. Twelve lines of SVG is a cheaper
 *  price than a shared package for it.
 *
 *  Drawn in `currentColor` with no fill, so the same file works black on the
 *  till's lock screen and anywhere else it is later needed. */
export default function KeelMark({
  className = "h-8 w-8",
}: {
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 32 32"
      fill="none"
      stroke="currentColor"
      // Matches logos/keel-mark.svg, which is the source of truth for the mark.
      // At 2.4 the stroke reads thin beside the semibold wordmark.
      strokeWidth={2.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
    >
      {/* The fin is the whole point of the mark — it is the keel. Long enough
          to survive at small sizes, where a short stub reads as a smudge and
          the logo becomes an anonymous curve. */}
      <path d="M5 6c0 9.5 4.4 14.5 11 14.5S27 15.5 27 6" />
      <path d="M16 20.5V29" />
    </svg>
  );
}
