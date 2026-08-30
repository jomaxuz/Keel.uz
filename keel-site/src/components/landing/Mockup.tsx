import { Frame, FloatBadge } from "./Frame";

/** A product screenshot in a device frame, with a glyph hanging off one corner.
 *
 *  ⚠️ **One component because there are nine of these.** Written out at each
 *  call site, the `img` tag drifted: two of them shipped without `width`/
 *  `height` and the row jumped as they loaded, and the first two published
 *  files carried an artefact nobody spotted because each shot was pasted in by
 *  hand. Frame, badge corner and the loading attributes are decided once here.
 *
 *  The shots come from the b5somsa test tenant filled by `backend/cmd/demodata`
 *  — never a live customer's panel (`docs/LANDING_REDESIGN.md` §4). */
export default function Mockup({
  src,
  alt,
  w,
  h,
  kind = "screen",
  badge,
  chip,
  chipSide = "right",
  side = "left",
  priority = false,
  className = "",
}: {
  src: string;
  alt: string;
  w: number;
  h: number;
  kind?: "browser" | "phone" | "tablet" | "screen";
  badge?: React.ReactNode;
  /** A `FloatChip` hung off the opposite bottom corner. Two floating pieces per
   *  mockup is the density the design calls for; a third starts hiding the
   *  screen they are pointing at. */
  chip?: React.ReactNode;
  chipSide?: "left" | "right";
  /** Which corner the badge hangs off. It goes on the outside of the row, so
   *  on a mirrored layout it must move — pinned to one side it lands on the
   *  screenshot's own heading. */
  side?: "left" | "right";
  /** ⚠️ Set on the shot above the fold, and only there. That image is the page's
   *  largest paint, and `loading="lazy"` on it asks the browser to delay the one
   *  thing the visitor is waiting for. Everywhere else lazy is right: eight more
   *  screenshots eagerly fetched is most of a megabyte nobody has scrolled to. */
  priority?: boolean;
  className?: string;
}) {
  return (
    <div className={`relative ${className}`}>
      {badge && (
        <FloatBadge
          className={side === "right" ? "-right-3 -top-3" : "-left-3 -top-3"}
        >
          {badge}
        </FloatBadge>
      )}
      <Frame kind={kind}>
        <img
          src={src}
          alt={alt}
          width={w}
          height={h}
          loading={priority ? "eager" : "lazy"}
          fetchPriority={priority ? "high" : undefined}
          decoding="async"
          className="block w-full"
        />
      </Frame>
      {chip && (
        <div
          className={`pointer-events-none absolute -bottom-4 hidden sm:block ${
            chipSide === "left" ? "-left-4" : "-right-4"
          }`}
        >
          {chip}
        </div>
      )}
    </div>
  );
}
