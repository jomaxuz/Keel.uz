// The three-dimensional illustrations.
//
// ⚠️ **Drawn as inline SVG rather than shipped as rendered images**, and the
// reason is the same one that keeps a chart library off this page. A set of 3D
// renders good enough to sit on a marketing page is two or three megabytes of
// PNG, and this page is read on a phone on Uzbek mobile data by somebody
// deciding whether we are worth a message. It would also need a second set for
// the dark theme, because a render carries its own background and lighting —
// and the second set is the one that quietly stops matching the first.
//
// Isometric SVG solves all of that at once: it inherits the theme's tokens, it
// is a few kilobytes, it scales to any screen without a second asset, and there
// is exactly one copy to keep true.
//
// ⚠️ **The depth is real geometry, not a drop shadow.** Every solid here is
// built from three faces on the same 2:1 isometric grid — top, left, right —
// each a flat colour a step apart in lightness. That is what makes them read as
// objects rather than as stickers, and it is also why they survive being drawn
// in a light and a dark theme: only the tokens under them change.
//
// ⚠️ **Gradients are declared with unique ids per component.** Two SVGs on one
// page sharing a gradient id is a real bug and a confusing one — the second
// element silently takes the first one's fill, and it looks like a colour
// mistake rather than a name collision.

/** How one drawing serves two jobs.
 *
 *  ⚠️ **The same illustration is content in one place and decoration in
 *  another, and a screen reader must be told which.** The till in its own
 *  section is the object being sold and carries a label; the shop-and-phone
 *  behind the hero card is atmosphere at 25% opacity, and announcing it there
 *  reads out a picture nobody can see the point of. So the caller says which,
 *  and the component sets `role`/`aria-label` or `aria-hidden` accordingly —
 *  rather than the caller passing `aria-hidden` in, which a component that does
 *  not spread its props silently drops. */
type VisualProps = {
  className?: string;
  /** Atmosphere rather than content: hidden from assistive technology. */
  decorative?: boolean;
};

/** The accessibility attributes for one of these, from that single decision. */
function a11y(label: string, decorative?: boolean) {
  return decorative
    ? ({ "aria-hidden": true } as const)
    : ({ role: "img", "aria-label": label } as const);
}

/** The isometric palette: one hue, three faces.
 *
 *  Lightness rather than hue does the work. Shifting hue between faces is what
 *  makes an isometric drawing look like plastic; keeping the hue and moving the
 *  light is what makes it look like a solid object under one lamp. */
const FACE = {
  // Warm accent — the signal colour, which is the product's own.
  //
  // ⚠️ Literal on purpose, unlike the two below: the accent is the same colour
  // in both themes (it is the brand), and it is legible on cream and on hull
  // alike. Only the materials that would collide with the page background are
  // tokenised.
  topA: "#FFC46B",
  leftA: "#D2870F",
  rightA: "#F5A524",
  // Deep marine — the hull. ⚠️ **Tokens, because these collide with the page.**
  // Written as literals first, the till's left face (#0A1A28) sat on a dark
  // page of #05101A and the object lost a whole side to the background — which
  // reads as a rendering fault, not a colour choice. See globals.css.
  topB: "var(--iso-top-b)",
  leftB: "var(--iso-left-b)",
  rightB: "var(--iso-right-b)",
  // Neutral, for the surfaces that must not compete.
  topC: "var(--iso-top-c)",
  leftC: "var(--iso-left-c)",
  rightC: "var(--iso-right-c)",
};

/** The isometric grid, written once.
 *
 *  ⚠️ **A solid shows one face toward the viewer and one *receding* side, never
 *  two faces both angled forward.** The first draft of the till got that wrong
 *  and the result was unmistakable: two screen panels both facing out, which
 *  reads as an open book rather than a monoblock. It is the kind of mistake
 *  that is invisible in path data and obvious the moment anybody looks at the
 *  page — which is exactly why the projection lives in a function here instead
 *  of in forty hand-typed coordinates.
 *
 *  `a` runs right-and-away, `b` runs left-and-away, `z` is height. One unit of
 *  either is 64 across and 37 up: the 2:1-ish ratio that reads as isometric
 *  without the arithmetic of a true 30° projection.
 */
const A: [number, number] = [64, -37];
const B: [number, number] = [-64, -37];

type P = [number, number];

/** A point on the grid, measured from an origin on the ground plane. */
function pt(o: P, a: number, b: number, z = 0): P {
  return [o[0] + a * A[0] + b * B[0], o[1] + a * A[1] + b * B[1] - z];
}

/** A closed polygon from grid points. */
function poly(...points: P[]) {
  return (
    points
      .map(([x, y], i) => `${i ? "L" : "M"}${x.toFixed(1)} ${y.toFixed(1)}`)
      .join(" ") + " Z"
  );
}

/** A box standing on the ground plane: its three visible faces.
 *
 *  A component rather than three calls at each site, so a solid is drawn with
 *  its faces always in the same order and nobody can paint the far side over
 *  the near one.
 */
function Box({
  o,
  a,
  b,
  h,
  top,
  left,
  right,
}: {
  o: P;
  a: number;
  b: number;
  h: number;
  top: string;
  left: string;
  right: string;
}) {
  return (
    <>
      <path
        d={poly(pt(o, 0, 0, h), pt(o, a, 0, h), pt(o, a, b, h), pt(o, 0, b, h))}
        fill={top}
      />
      <path
        d={poly(pt(o, 0, 0), pt(o, 0, b), pt(o, 0, b, h), pt(o, 0, 0, h))}
        fill={left}
      />
      <path
        d={poly(pt(o, 0, 0), pt(o, a, 0), pt(o, a, 0, h), pt(o, 0, 0, h))}
        fill={right}
      />
    </>
  );
}







/** A stack of receipts with a coin — the money illustration for pricing. */
export function PriceVisual({ className = "", decorative }: VisualProps) {
  return (
    <svg
      viewBox="0 0 240 200"
      className={className}
      {...a11y("Oylik hisob", decorative)}
    >
      <defs>
        <linearGradient id="v3d-coin" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#FFC46B" />
          <stop offset="100%" stopColor="#D2870F" />
        </linearGradient>
      </defs>

      <ellipse cx="120" cy="172" rx="82" ry="16" fill="var(--iso-shadow)" />

      {/* Three sheets, offset so the stack reads as a stack rather than as one
          thick slab. */}
      {[
        { dy: 0, top: FACE.topC, left: FACE.leftC, right: FACE.rightC },
        { dy: -14, top: "#F2F6F9", left: "#BCC9D5", right: "#D8E1E8" },
        { dy: -28, top: "#FFFFFF", left: "#C8D4DE", right: "#E4EBF1" },
      ].map((s, i) => (
        <g key={i} transform={`translate(0 ${s.dy})`}>
          <path d="M120 160 L 44 138 L 44 130 L 120 152 Z" fill={s.left} />
          <path d="M120 160 L 196 138 L 196 130 L 120 152 Z" fill={s.right} />
          <path d="M120 152 L 44 130 L 120 108 L 196 130 Z" fill={s.top} />
        </g>
      ))}

      {/* The lines on the top sheet: a bill has rows, and without them this is
          a stack of blank cards. */}
      <g opacity="0.5" stroke="#8C9CAB" strokeWidth="1.6" strokeLinecap="round">
        <path d="M84 104 L 136 89" />
        <path d="M96 112 L 148 97" />
        <path d="M108 120 L 160 105" />
      </g>

      {/* The coin, standing on edge above the stack — the one warm object. */}
      <g transform="translate(150 44)">
        <ellipse cx="0" cy="0" rx="30" ry="30" fill="url(#v3d-coin)" />
        <ellipse cx="0" cy="0" rx="22" ry="22" fill="none" stroke="#8A5A08" strokeWidth="1.6" opacity="0.45" />
        <text
          x="0"
          y="7"
          textAnchor="middle"
          fontSize="19"
          fontWeight="700"
          fill="#5A3C05"
          fontFamily="var(--font-display), system-ui, sans-serif"
        >
          so&apos;m
        </text>
      </g>
    </svg>
  );
}

/** Three cubes on a plinth — the plan ladder, as an object.
 *
 *  ⚠️ The heights are the ladder's shape and not decoration: they rise and the
 *  page's own copy says the per-register price falls as they do. A picture that
 *  contradicted the sentence beside it would be worse than no picture. */
export function LadderVisual({ className = "", decorative }: VisualProps) {
  const bars = [
    { x: 70, h: 34, top: FACE.topC, left: FACE.leftC, right: FACE.rightC },
    { x: 130, h: 62, top: "#FFD79B", left: "#B8760C", right: "#E5951A" },
    { x: 190, h: 92, top: FACE.topA, left: FACE.leftA, right: FACE.rightA },
  ];
  return (
    <svg
      viewBox="0 0 280 190"
      className={className}
      {...a11y("Tarif narvoni", decorative)}
    >
      <ellipse cx="140" cy="163" rx="106" ry="17" fill="var(--iso-shadow)" />
      {bars.map((b) => {
        const baseY = 150 - (b.x - 70) * 0.18;
        const topY = baseY - b.h;
        const w = 26;
        const d = 13;
        return (
          <g key={b.x}>
            {/* left face */}
            <path
              d={`M${b.x} ${baseY} L${b.x} ${topY} L${b.x + w} ${topY + d} L${b.x + w} ${baseY + d} Z`}
              fill={b.left}
            />
            {/* right face */}
            <path
              d={`M${b.x + w} ${baseY + d} L${b.x + w} ${topY + d} L${b.x + w * 2} ${topY} L${b.x + w * 2} ${baseY} Z`}
              fill={b.right}
            />
            {/* top face */}
            <path
              d={`M${b.x} ${topY} L${b.x + w} ${topY - d} L${b.x + w * 2} ${topY} L${b.x + w} ${topY + d} Z`}
              fill={b.top}
            />
          </g>
        );
      })}
    </svg>
  );
}

/** A shop and a customer's phone, with a line between them and nothing in the
 *  middle — the picture of the argument the comparison table makes.
 *
 *  ⚠️ **Redrawn once it stopped being decoration.** The first version was a
 *  plain box beside a plain slab, which was survivable while it sat at 25%
 *  opacity behind the hero card and said nothing to anybody. Moved next to the
 *  paragraph it illustrates, "a box and a slab" reads as clip art — so the shop
 *  got the two details that make a shop (an awning and a door) and the phone
 *  got the one that makes a phone (a lit screen inset into its face).
 */
export function ChannelVisual({ className = "", decorative }: VisualProps) {
  const shop: P = [92, 150];
  const sA = 0.62;
  const sB = 0.62;
  const sH = 46;

  const phone: P = [196, 156];
  const pA = 0.34;
  const pB = 0.1;
  const pH = 74;

  /** A rectangle on the shop's right-hand face, in (along a, up). */
  const front = (aFrom: number, aTo: number, zFrom: number, zTo: number) =>
    poly(
      pt(shop, aFrom, 0, zFrom),
      pt(shop, aTo, 0, zFrom),
      pt(shop, aTo, 0, zTo),
      pt(shop, aFrom, 0, zTo),
    );

  /** A rectangle on the phone's right-hand face. */
  const glass = (aFrom: number, aTo: number, zFrom: number, zTo: number) =>
    poly(
      pt(phone, aFrom, 0, zFrom),
      pt(phone, aTo, 0, zFrom),
      pt(phone, aTo, 0, zTo),
      pt(phone, aFrom, 0, zTo),
    );

  return (
    <svg
      viewBox="20 44 226 138"
      className={className}
      {...a11y("O'z kanalingiz", decorative)}
    >
      <ellipse cx="88" cy="164" rx="62" ry="11" fill="var(--iso-shadow)" />
      <ellipse cx="206" cy="166" rx="30" ry="7" fill="var(--iso-shadow)" />

      {/* ---- The shop ---- */}
      <Box
        o={shop}
        a={sA}
        b={sB}
        h={sH}
        top={FACE.topB}
        left={FACE.leftB}
        right={FACE.rightB}
      />

      {/* The lit shopfront, then the door cut out of it.
          ⚠️ **Two big shapes, not four small ones.** This is drawn at about
          ninety pixels across on the page; a door, a window and a sign at that
          size are three smudges. One warm glow under a canopy reads as an open
          shop at any size, which is the only thing it has to say. */}
      <path d={front(0.06, sA - 0.06, 4, 30)} fill={FACE.rightA} opacity="0.9" />
      <path d={front(0.1, 0.26, 4, 26)} fill="var(--iso-screen-b)" opacity="0.85" />

      {/* The canopy: a slab across the whole front, at the top, hanging out
          over it. The one shape that turns a box into a shop — without it this
          is a crate, which is exactly how the first version read. */}
      <path
        d={poly(
          pt(shop, 0, 0, sH - 2),
          pt(shop, sA, 0, sH - 2),
          pt(shop, sA, -0.2, sH - 12),
          pt(shop, 0, -0.2, sH - 12),
        )}
        fill={FACE.topA}
      />
      <path
        d={poly(
          pt(shop, 0, -0.2, sH - 12),
          pt(shop, sA, -0.2, sH - 12),
          pt(shop, sA, -0.2, sH - 17),
          pt(shop, 0, -0.2, sH - 17),
        )}
        fill={FACE.leftA}
      />

      {/* ---- The phone ---- */}
      <Box
        o={phone}
        a={pA}
        b={pB}
        h={pH}
        top={FACE.topB}
        left={FACE.leftB}
        right="var(--iso-screen-b)"
      />
      {/* The screen, inset from the body so the bezel reads. */}
      <path d={glass(0.04, pA - 0.04, 6, pH - 6)} fill="var(--iso-screen-a)" />
      {/* An order on it: the same bar-and-lines vocabulary as the till screen,
          so the two drawings are recognisably the same product. */}
      <path d={glass(0.07, pA - 0.07, pH - 22, pH - 13)} fill="#F5A524" opacity="0.9" />
      <path d={glass(0.07, pA - 0.12, pH - 34, pH - 29)} fill="#FFC46B" opacity="0.4" />
      <path d={glass(0.07, pA - 0.16, pH - 44, pH - 39)} fill="#FFC46B" opacity="0.3" />
      <path d={glass(0.07, pA - 0.07, 12, 21)} fill="#F5A524" opacity="0.75" />

      {/* ---- The line between them ----
          Dashed, because it is a route rather than a wire, and it goes shop to
          phone with nothing in between. That gap is the whole argument. */}
      <path
        d="M126 116 C 150 104, 164 100, 184 98"
        fill="none"
        stroke="#F5A524"
        strokeWidth="2.4"
        strokeDasharray="5 6"
        strokeLinecap="round"
      />
    </svg>
  );
}
