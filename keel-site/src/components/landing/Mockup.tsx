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
  side = "left",
  className = "",
}: {
  src: string;
  alt: string;
  w: number;
  h: number;
  kind?: "browser" | "phone" | "tablet" | "screen";
  badge?: React.ReactNode;
  /** Which corner the badge hangs off. It goes on the outside of the row, so
   *  on a mirrored layout it must move — pinned to one side it lands on the
   *  screenshot's own heading. */
  side?: "left" | "right";
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
          loading="lazy"
          decoding="async"
          className="block w-full"
        />
      </Frame>
    </div>
  );
}
