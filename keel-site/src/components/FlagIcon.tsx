// Small flags for the language switch.
//
// ⚠️ **Not emoji.** `🇺🇿` looks right on a Mac and on Android, and on Windows it
// renders as the letters "UZ" — Windows ships no flag glyphs at all, and that is
// still true today. So the one place a flag is meant to be recognised without
// reading would show a second copy of the text beside it, in a different font,
// for a large share of desktop visitors.
//
// Inline SVG instead: no request, no font dependency, and the same drawing in
// light and dark. Deliberately simplified — at 20×14 the crescent and twelve stars
// of the Uzbek flag are three grey smudges, so what is drawn is what survives at
// that size: the bands, the thin red fimbriations, and one crescent.
//
// ⚠️ English is the **UK** flag rather than the US one. Arbitrary either way, but
// a consistent arbitrary choice beats a switch whose third row changes flag
// depending on who last edited it.

const R = 3;

export default function FlagIcon({
  lang,
  className = "h-3.5 w-5",
}: {
  lang: string;
  className?: string;
}) {
  const common = {
    viewBox: "0 0 20 14",
    className,
    "aria-hidden": true as const,
  };
  // A rounded clip so every flag has the same silhouette; without it the three
  // rows in the popup line up on their corners rather than their edges.
  const clipId = `flag-clip-${lang}`;

  if (lang === "ru") {
    return (
      <svg {...common}>
        <defs>
          <clipPath id={clipId}>
            <rect width="20" height="14" rx={R} />
          </clipPath>
        </defs>
        <g clipPath={`url(#${clipId})`}>
          <rect width="20" height="14" fill="#fff" />
          <rect y="4.67" width="20" height="4.66" fill="#0039a6" />
          <rect y="9.33" width="20" height="4.67" fill="#d52b1e" />
        </g>
        <rect
          width="20"
          height="14"
          rx={R}
          fill="none"
          stroke="currentColor"
          strokeOpacity="0.15"
        />
      </svg>
    );
  }

  if (lang === "en") {
    return (
      <svg {...common}>
        <defs>
          <clipPath id={clipId}>
            <rect width="20" height="14" rx={R} />
          </clipPath>
        </defs>
        <g clipPath={`url(#${clipId})`}>
          <rect width="20" height="14" fill="#012169" />
          {/* Diagonals first, then the cross over them — the order is what makes
              a Union Jack read as one rather than as a saltire with a stripe. */}
          <path d="M0 0 20 14M20 0 0 14" stroke="#fff" strokeWidth="2.8" />
          <path d="M0 0 20 14M20 0 0 14" stroke="#c8102e" strokeWidth="1.4" />
          <path d="M10 0v14M0 7h20" stroke="#fff" strokeWidth="4.6" />
          <path d="M10 0v14M0 7h20" stroke="#c8102e" strokeWidth="2.6" />
        </g>
        <rect
          width="20"
          height="14"
          rx={R}
          fill="none"
          stroke="currentColor"
          strokeOpacity="0.15"
        />
      </svg>
    );
  }

  // Uzbek — the default, as everywhere else in this app.
  return (
    <svg {...common}>
      <defs>
        <clipPath id={clipId}>
          <rect width="20" height="14" rx={R} />
        </clipPath>
      </defs>
      <g clipPath={`url(#${clipId})`}>
        <rect width="20" height="14" fill="#0099b5" />
        <rect y="4.4" width="20" height="5.2" fill="#fff" />
        <rect y="9.6" width="20" height="4.4" fill="#1eb53a" />
        {/* The red fimbriations. Thin, but they are what stops the flag reading
            as Sierra Leone's at this size. */}
        <rect y="4.2" width="20" height="0.4" fill="#ce1126" />
        <rect y="9.4" width="20" height="0.4" fill="#ce1126" />
        {/* One crescent, no stars: twelve stars at 14px tall are noise. */}
        <path
          d="M4.6 2.2a1.9 1.9 0 1 0 0 3.6 2.3 2.3 0 1 1 0-3.6Z"
          fill="#fff"
        />
      </g>
      <rect
        width="20"
        height="14"
        rx={R}
        fill="none"
        stroke="currentColor"
        strokeOpacity="0.15"
      />
    </svg>
  );
}
