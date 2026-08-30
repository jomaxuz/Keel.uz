/** Device frames and the slot a product screenshot goes into.
 *
 *  ⚠️ **The frame is the theme-aware part and the picture is not.** A screenshot
 *  is a fixed image; the page it sits on inverts. A bare light screenshot on the
 *  dark page is a hole punched in it. Wrapping every shot in a frame drawn from
 *  the page's own tokens gives the image a border that always belongs, which is
 *  cheaper than shipping and maintaining two screenshots of every screen.
 *
 *  The shots themselves do not exist yet (`docs/LANDING_REDESIGN.md` §4 — they
 *  come from the b5somsa test tenant, never a live customer's panel), so for now
 *  the frames hold the drawings the page already had. The frame is what makes a
 *  drawing read as a screen rather than as an ornament, and it is the part that
 *  stays when the picture inside it is replaced. */

type Kind = "browser" | "phone" | "tablet" | "screen";

/** ⚠️ **The four kinds have to be told apart at a glance or they are one kind.**
 *  The first pass gave them different corner radii and nothing else, and a
 *  tablet beside a monitor read as two identical white boxes — which is worse
 *  than no frame, because the reader spends a moment looking for the difference
 *  before concluding there is none. Each kind now carries the one detail that
 *  identifies it: chrome and an address bar, a notch, a bezel, a stand. */
const BODY: Record<Kind, string> = {
  browser: "rounded-2xl",
  screen: "rounded-2xl",
  // A phone is recognisable by its corner radius before anything else on it.
  phone: "rounded-[2rem] mx-auto max-w-[15rem] p-2",
  // A tablet is a screen with a visible bezel all the way round.
  tablet: "rounded-3xl p-3",
};

export function Frame({
  kind = "browser",
  label,
  children,
  className = "",
}: {
  kind?: Kind;
  /** Address-bar text for a browser frame. Ignored by the other kinds. */
  label?: string;
  children?: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`relative overflow-hidden border border-line-strong bg-raised shadow-xl shadow-hull-950/5 dark:shadow-hull-950/40 ${BODY[kind]} ${className}`}
    >
      {kind === "browser" && (
        <div className="flex items-center gap-2 border-b border-line px-3 py-2">
          <span className="flex gap-1.5" aria-hidden>
            <i className="block h-2 w-2 rounded-full bg-line-strong" />
            <i className="block h-2 w-2 rounded-full bg-line-strong" />
            <i className="block h-2 w-2 rounded-full bg-line-strong" />
          </span>
          {label && (
            <span className="ml-1 truncate rounded-md bg-surface px-2 py-0.5 text-[11px] text-ink-muted">
              {label}
            </span>
          )}
        </div>
      )}
      {kind === "phone" && (
        // The notch, and it is the whole reason the shape reads as a phone.
        <div className="flex justify-center pt-2" aria-hidden>
          <span className="h-1.5 w-16 rounded-full bg-line-strong" />
        </div>
      )}
      {/* The ground the picture stands on. `page` rather than `surface` because
          these frames sit inside surface cards, and a white screen on a white
          card is a frame with nothing in it. */}
      <div
        className={`overflow-hidden bg-page ${
          kind === "phone" ? "rounded-[1.5rem]" : kind === "tablet" ? "rounded-xl" : ""
        }`}
      >
        {children}
      </div>
      {kind === "screen" && (
        // The stand. One shape, and it is the whole difference between a
        // monitor on a pass and a picture in a box.
        <div className="flex flex-col items-center pb-2 pt-2.5" aria-hidden>
          <span className="h-2.5 w-10 bg-line-strong" />
          <span className="h-1.5 w-24 rounded-full bg-line-strong" />
        </div>
      )}
    </div>
  );
}

/** A rounded glyph tile hanging off a mockup's corner. Position is a caller's
 *  business — it depends on which corner of which card is free. */
export function FloatBadge({
  children,
  className = "",
  tone = "signal",
}: {
  children: React.ReactNode;
  className?: string;
  /** Green means working, connected, saved — never decoration
   *  (`docs/LANDING_REDESIGN.md` §3). */
  tone?: "signal" | "good";
}) {
  return (
    <span
      className={`float-badge ${
        tone === "good" ? "bg-emerald-500" : "bg-signal-500 text-hull-950"
      } ${className}`}
      aria-hidden
    >
      {children}
    </span>
  );
}
