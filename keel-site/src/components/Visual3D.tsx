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

/** Rectangles on a screen face — the face that spans `a` and height, at a
 *  fixed depth `b`.
 *
 *  ⚠️ **Shared, because every device on this page draws a user interface, and
 *  a UI is nothing but axis-aligned rectangles.** Written per component it was
 *  the same six lines four times, and the fourth copy is where somebody swaps
 *  two arguments and a screen quietly renders inside out.
 */
function screenRect(o: P, b = 0) {
  return (aFrom: number, aTo: number, zFrom: number, zTo: number) =>
    poly(
      pt(o, aFrom, b, zFrom),
      pt(o, aTo, b, zFrom),
      pt(o, aTo, b, zTo),
      pt(o, aFrom, b, zTo),
    );
}

/** A screen slab standing on the ground plane: its dark body, plus the two
 *  thin edges that give it depth. The caller draws the display over it.
 *
 *  Returned as a fragment so the edges are always painted before the glass and
 *  nobody can put the receding side over the face.
 */
function Slab({ o, a, h, t = 0.09 }: { o: P; a: number; h: number; t?: number }) {
  return (
    <>
      <path
        d={poly(pt(o, 0, 0, h), pt(o, a, 0, h), pt(o, a, -t, h), pt(o, 0, -t, h))}
        fill={FACE.rightB}
      />
      <path
        d={poly(pt(o, a, 0, 0), pt(o, a, 0, h), pt(o, a, -t, h), pt(o, a, -t, 0))}
        fill={FACE.leftB}
      />
    </>
  );
}

/** The monoblock running Keel — the page's lead image.
 *
 *  ⚠️ **The screen shows the actual program, not a grey wireframe.** This is
 *  the object we are now mainly selling, and the single most useful thing a
 *  picture of it can do is answer "what does it look like to use". A dish grid
 *  on the left, an open check with its lines and a total on the right, and the
 *  two buttons a cashier reaches for — drawn small, but drawn as what they are.
 *  A screen full of anonymous rectangles is a picture of any tablet.
 */
export function MonoblockVisual({ className = "", decorative }: VisualProps) {
  const o: P = [122, 216];
  const bodyA = 1.5;
  const bodyB = 0.78;
  const bodyH = 18;

  // The screen stands on the body's far edge and runs along `a`, so the display
  // is the one face pointing at the viewer and the panel's thickness recedes.
  const so: P = pt(o, 0, bodyB, bodyH);
  const H = 104;
  const f = screenRect(so);

  // The two columns of the till screen, in screen units along `a`.
  const gridL = 0.07;
  const gridR = 0.83;
  const chkL = 0.88;
  const chkR = bodyA - 0.07;

  return (
    <svg
      // ⚠️ Measured from the geometry, not guessed. The first box started at
      // y=40 and cut the top off the screen — including the app's own header
      // bar, which is the strip that says this is a program and not a poster.
      viewBox="34 2 232 248"
      className={className}
      {...a11y("Keel dasturi ishlab turgan monoblok kassa", decorative)}
    >
      <defs>
        <linearGradient id="v3d-mono-glass" x1="0" y1="0" x2="0.45" y2="1">
          <stop offset="0%" stopColor="var(--iso-screen-a)" />
          <stop offset="100%" stopColor="var(--iso-screen-b)" />
        </linearGradient>
      </defs>

      <ellipse cx="150" cy="230" rx="112" ry="17" fill="var(--iso-shadow)" />

      {/* ---- The body ---- */}
      <Box
        o={o}
        a={bodyA}
        b={bodyB}
        h={bodyH}
        top={FACE.topB}
        left={FACE.leftB}
        right={FACE.rightB}
      />

      {/* ---- The screen ---- */}
      <Slab o={so} a={bodyA} h={H} />
      <path d={f(0, bodyA, 0, H)} fill="url(#v3d-mono-glass)" />

      {/* The app's top bar, with our mark as a square at the left and the
          shift's state as a dot at the right. */}
      <path d={f(0.03, bodyA - 0.03, H - 13, H - 4)} fill="#FFFFFF" opacity="0.06" />
      <path d={f(0.06, 0.12, H - 11.5, H - 5.5)} fill="#F5A524" />
      <path d={f(0.15, 0.36, H - 10.5, H - 6.5)} fill="#FFFFFF" opacity="0.3" />
      <path d={f(bodyA - 0.13, bodyA - 0.07, H - 11.5, H - 5.5)} fill="#3FB27F" opacity="0.85" />

      {/* ---- Left: the dish grid ---- */}
      {[0, 1, 2].map((row) =>
        [0, 1, 2].map((col) => {
          const w = (gridR - gridL - 0.04) / 3;
          const x = gridL + col * (w + 0.02);
          const y = 20 + row * 22;
          return (
            <g key={`${row}-${col}`}>
              <path d={f(x, x + w, y, y + 18)} fill="#FFFFFF" opacity="0.07" />
              {/* A dish tile is a picture with a name under it. Two shapes are
                  enough to say that at this size; three would be mud. */}
              <path
                d={f(x + 0.015, x + w - 0.015, y + 7, y + 16)}
                fill="#F5A524"
                opacity={0.26 + ((row * 3 + col) % 4) * 0.13}
              />
              <path d={f(x + 0.015, x + w - 0.06, y + 3, y + 5)} fill="#FFFFFF" opacity="0.22" />
            </g>
          );
        }),
      )}

      {/* ---- Right: the open check ---- */}
      <path d={f(chkL, chkR, 6, H - 18)} fill="#FFFFFF" opacity="0.05" />
      {/* Its lines: a name and a price on each row. */}
      {[0, 1, 2, 3].map((i) => {
        const y = H - 32 - i * 11;
        return (
          <g key={i}>
            <path d={f(chkL + 0.03, chkR - 0.18, y, y + 4)} fill="#FFFFFF" opacity="0.26" />
            <path d={f(chkR - 0.14, chkR - 0.03, y, y + 4)} fill="#FFFFFF" opacity="0.16" />
          </g>
        );
      })}
      {/* The total: the one bright thing on the screen, because it is the one
          number everybody at the counter is looking at. */}
      <path d={f(chkL + 0.03, chkR - 0.03, 30, 40)} fill="#F5A524" />
      {/* And the two buttons under it. */}
      <path d={f(chkL + 0.03, (chkL + chkR) / 2 - 0.01, 12, 24)} fill="#FFFFFF" opacity="0.14" />
      <path d={f((chkL + chkR) / 2 + 0.01, chkR - 0.03, 12, 24)} fill="#3FB27F" opacity="0.75" />

      {/* ---- The receipt printer ---- */}
      <Box
        o={pt(o, bodyA + 0.2, 0.06)}
        a={0.38}
        b={0.38}
        h={15}
        top={FACE.topC}
        left={FACE.leftC}
        right={FACE.rightC}
      />
      <path
        d={poly(
          pt(o, bodyA + 0.3, 0.2, 15),
          pt(o, bodyA + 0.44, 0.2, 15),
          pt(o, bodyA + 0.44, 0.2, 38),
          pt(o, bodyA + 0.3, 0.2, 38),
        )}
        fill="#FFFFFF"
      />
      <path
        d={poly(
          pt(o, bodyA + 0.32, 0.2, 23),
          pt(o, bodyA + 0.42, 0.2, 23),
          pt(o, bodyA + 0.42, 0.2, 25),
          pt(o, bodyA + 0.32, 0.2, 25),
        )}
        fill="#AFBECB"
      />
      <path
        d={poly(
          pt(o, bodyA + 0.32, 0.2, 28),
          pt(o, bodyA + 0.4, 0.2, 28),
          pt(o, bodyA + 0.4, 0.2, 30),
          pt(o, bodyA + 0.32, 0.2, 30),
        )}
        fill="#AFBECB"
      />
    </svg>
  );
}

/** The floor tablet: a table map, lying flat on its stand.
 *
 *  ⚠️ Drawn **flat rather than upright**, unlike every other screen here. A
 *  waiter's tablet is held or propped, and a floor plan seen edge-on is a floor
 *  plan nobody can read — the whole point of the picture is that the shapes on
 *  it are recognisably tables in a room.
 */
export function FloorVisual({ className = "", decorative }: VisualProps) {
  const o: P = [130, 168];
  const a = 1.15;
  const b = 0.86;

  /** A rectangle on the tablet's flat glass, in (a, b). */
  const g = (a0: number, b0: number, a1: number, b1: number) =>
    poly(pt(o, a0, b0, 7), pt(o, a1, b0, 7), pt(o, a1, b1, 7), pt(o, a0, b1, 7));

  /** A round table. ⚠️ Drawn as a squashed ellipse at the right centre, not as
   *  a circle: on this grid a circle is a table standing on its edge. */
  const round = (ca: number, cb: number, r: number) => {
    const c = pt(o, ca, cb, 7);
    return { cx: c[0], cy: c[1], rx: r * 64, ry: r * 37 };
  };

  return (
    <svg
      viewBox="62 96 158 92"
      className={className}
      {...a11y("Zal xaritasi planshetda", decorative)}
    >
      <ellipse cx="150" cy="176" rx="72" ry="10" fill="var(--iso-shadow)" />
      <Box o={o} a={a} b={b} h={7} top="var(--iso-screen-a)" left={FACE.leftB} right={FACE.rightB} />

      {/* The room. ⚠️ **Occupancy is the only colour**, because it is the only
          thing this screen exists to answer — a plan where every table looks
          the same is a plan nobody would open. Two shapes, because a dining
          room has both and a map of identical squares reads as a spreadsheet. */}
      <path d={g(0.08, 0.1, 0.34, 0.36)} fill="#FFFFFF" opacity="0.15" />
      <ellipse {...round(0.56, 0.24, 0.16)} fill="#F5A524" opacity="0.9" />
      <path d={g(0.78, 0.1, 1.04, 0.36)} fill="#FFFFFF" opacity="0.15" />
      <path d={g(0.08, 0.52, 0.34, 0.78)} fill="#F5A524" opacity="0.45" />
      <ellipse {...round(0.56, 0.65, 0.16)} fill="#FFFFFF" opacity="0.15" />
      <path d={g(0.78, 0.52, 1.04, 0.78)} fill="#FFFFFF" opacity="0.15" />
    </svg>
  );
}

/** The kitchen screen: tickets waiting, oldest first.
 *
 *  One upright panel with three cards on it — the pass screen has no money, no
 *  customer and no filters, and a drawing of it should not invent any.
 */
export function KitchenVisual({ className = "", decorative }: VisualProps) {
  const o: P = [110, 188];
  const a = 1.6;
  const H = 72;
  const so: P = pt(o, 0, 0.34, 10);
  const f = screenRect(so);

  return (
    <svg
      viewBox="76 30 152 176"
      className={className}
      {...a11y("Oshxona ekrani", decorative)}
    >
      <ellipse cx="150" cy="194" rx="70" ry="10" fill="var(--iso-shadow)" />
      <Box o={o} a={a} b={0.34} h={10} top={FACE.topB} left={FACE.leftB} right={FACE.rightB} />
      <Slab o={so} a={a} h={H} />
      <path d={f(0, a, 0, H)} fill="var(--iso-screen-b)" />

      {/* Three tickets, side by side and wide — a pass screen is landscape, and
          a row of narrow strips reads as a bar chart.

          ⚠️ **The leftmost is the accent**: the oldest ticket is the only thing
          this screen ranks by, and a drawing where all three look alike says
          the opposite of what the screen does. */}
      {[0, 1, 2].map((i) => {
        const w = (a - 0.2) / 3;
        const x = 0.08 + i * (w + 0.02);
        return (
          <g key={i}>
            <path
              d={f(x, x + w, 8, H - 10)}
              fill={i === 0 ? "#F5A524" : "#FFFFFF"}
              opacity={i === 0 ? 0.9 : 0.08}
            />
            {[0, 1, 2].map((r) => (
              <path
                key={r}
                d={f(
                  x + 0.04,
                  x + w - 0.06 - r * 0.05,
                  H - 26 - r * 10,
                  H - 22 - r * 10,
                )}
                fill={i === 0 ? "#0A1A28" : "#FFFFFF"}
                opacity={i === 0 ? 0.5 : 0.3}
              />
            ))}
          </g>
        );
      })}
    </svg>
  );
}

/** A monoblock till: the body on the counter, the screen rising from its far
 *  edge, and the receipt printer beside it.
 *
 *  The object being sold, drawn rather than described — "what is a Keel till"
 *  is a question a picture answers in a moment and a paragraph does not.
 */
export function TillVisual({ className = "", decorative }: VisualProps) {
  // The near-bottom corner of the body. Everything else is measured from here,
  // so the whole object moves by editing one pair of numbers.
  const o: P = [128, 212];
  const bodyA = 1.35;
  const bodyB = 0.7;
  const bodyH = 20;

  // ⚠️ The screen stands on the body's far-**left** edge and runs along `a`,
  // so the display is the single face pointing down-left at the viewer and the
  // panel's thickness recedes away from it. That is the whole fix.
  const so: P = pt(o, 0, bodyB, bodyH);
  const scrH = 88;
  const scrT = 0.1; // panel thickness, in grid units along b

  /** A rectangle on the display face, in (along the screen, up) coordinates. */
  const face = (aFrom: number, aTo: number, zFrom: number, zTo: number) =>
    poly(
      pt(so, aFrom, 0, zFrom),
      pt(so, aTo, 0, zFrom),
      pt(so, aTo, 0, zTo),
      pt(so, aFrom, 0, zTo),
    );

  return (
    <svg
      viewBox="0 0 320 260"
      className={className}
      {...a11y("Keel kassa monoblok", decorative)}
    >
      <defs>
        <linearGradient id="v3d-till-screen" x1="0" y1="0" x2="0.5" y2="1">
          <stop offset="0%" stopColor="var(--iso-screen-a)" />
          <stop offset="100%" stopColor="var(--iso-screen-b)" />
        </linearGradient>
      </defs>

      {/* The ground shadow. A flattened ellipse rather than a blur filter:
          filters are the one part of SVG that costs real paint time on a cheap
          Android phone, and this page is mostly read on one. */}
      <ellipse cx="162" cy="226" rx="96" ry="16" fill="var(--iso-shadow)" />

      {/* ---- The body ---- */}
      <Box
        o={o}
        a={bodyA}
        b={bodyB}
        h={bodyH}
        top={FACE.topB}
        left={FACE.leftB}
        right={FACE.rightB}
      />

      {/* ---- The screen ----
          Back to front: the thin top and far edges give the panel its depth,
          then the display goes over them. */}
      <path
        d={poly(
          pt(so, 0, 0, scrH),
          pt(so, bodyA, 0, scrH),
          pt(so, bodyA, -scrT, scrH),
          pt(so, 0, -scrT, scrH),
        )}
        fill={FACE.rightB}
      />
      <path
        d={poly(
          pt(so, bodyA, 0, 0),
          pt(so, bodyA, 0, scrH),
          pt(so, bodyA, -scrT, scrH),
          pt(so, bodyA, -scrT, 0),
        )}
        fill={FACE.leftB}
      />
      <path d={face(0, bodyA, 0, scrH)} fill="url(#v3d-till-screen)" />

      {/* What is on it: a dish grid on the left, the running total on the
          right. Legible as a till rather than as abstract rectangles, which
          would make this a picture of any tablet. */}
      <g>
        {[0, 1, 2].map((row) =>
          [0, 1, 2].map((col) => (
            <path
              key={`${row}-${col}`}
              d={face(
                0.09 + col * 0.185,
                0.09 + col * 0.185 + 0.145,
                16 + row * 22,
                16 + row * 22 + 16,
              )}
              fill="#F5A524"
              opacity={0.22 + ((row * 2 + col) % 3) * 0.15}
            />
          )),
        )}
        <path d={face(0.7, 1.26, 62, 76)} fill="#F5A524" opacity="0.92" />
        <path d={face(0.7, 1.16, 46, 54)} fill="#FFC46B" opacity="0.38" />
        <path d={face(0.7, 1.08, 32, 40)} fill="#FFC46B" opacity="0.26" />
        <path d={face(0.7, 1.12, 18, 26)} fill="#FFC46B" opacity="0.26" />
      </g>

      {/* ---- The receipt printer ---- */}
      <Box
        // ⚠️ Clear of the body, not touching it: abutted, the printer's left
        // face disappears behind the till and the two read as one moulded
        // object rather than as a machine standing beside another.
        o={pt(o, bodyA + 0.22, 0.04)}
        a={0.4}
        b={0.4}
        h={16}
        top={FACE.topC}
        left={FACE.leftC}
        right={FACE.rightC}
      />
      {/* The paper coming out of the top — the one detail that names the
          object. Without it this is a white cube. */}
      <path
        d={poly(
          pt(o, bodyA + 0.32, 0.18, 16),
          pt(o, bodyA + 0.47, 0.18, 16),
          pt(o, bodyA + 0.47, 0.18, 40),
          pt(o, bodyA + 0.32, 0.18, 40),
        )}
        fill="#FFFFFF"
      />
      <path
        d={poly(
          pt(o, bodyA + 0.34, 0.18, 24),
          pt(o, bodyA + 0.45, 0.18, 24),
          pt(o, bodyA + 0.45, 0.18, 26),
          pt(o, bodyA + 0.34, 0.18, 26),
        )}
        fill="#AFBECB"
      />
      <path
        d={poly(
          pt(o, bodyA + 0.34, 0.18, 30),
          pt(o, bodyA + 0.43, 0.18, 30),
          pt(o, bodyA + 0.43, 0.18, 32),
          pt(o, bodyA + 0.34, 0.18, 32),
        )}
        fill="#AFBECB"
      />
    </svg>
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
