/** Colour as a way to find a dish, not as decoration.
 *
 * ⚠️ **On a till, colour is faster than reading.** A cashier taking an order
 * while talking to a guest does not read two hundred dish names — they reach for
 * the corner of the screen where the drinks always are. Category colour is what
 * turns a grid of identical rectangles into a place with regions in it, and it
 * is the reason a colour-only till can be quicker than one with photographs.
 *
 * ⚠️ **Derived from the category id, never stored.** Nobody has to pick colours
 * for a menu they already spent an evening entering, the assignment is stable
 * across devices and reloads, and a restaurant that renames a category keeps the
 * colour its staff have learnt. A stored palette would be one more thing to fill
 * in before the till is usable, and one more thing to migrate.
 */

/** Eight hues, chosen to stay apart for the common forms of colour blindness.
 *
 * ⚠️ **Red and green are never the only difference between two of them** —
 * deuteranopia is roughly one man in twelve, which in a restaurant is one
 * cashier in twelve. The set leans on blue/orange/purple/brown, where the
 * separation survives.
 */
const HUES: [number, number, number][] = [
  [37, 99, 235], // blue
  [217, 119, 6], // amber
  [13, 148, 136], // teal
  [147, 51, 234], // purple
  [190, 24, 93], // pink
  [120, 113, 108], // stone
  [2, 132, 199], // sky
  [180, 83, 9], // brown
];

/** The tile's background and its accent bar.
 *
 * ⚠️ **The background is a low-alpha tint, never a solid fill.** A solid colour
 * has to be legible against black text in the light theme and white text in the
 * dark one, which no single value manages; a tint sits over `--surface` and
 * therefore darkens or lightens with it, and the text keeps using `text-ink`.
 * The bar carries the full colour, where nothing has to be readable on top of
 * it. Same reason the design tokens use `--line` rather than `ink/[0.07]`.
 */
export function categoryTint(categoryId: string): {
  background: string;
  bar: string;
} {
  const [r, g, b] = HUES[hashOf(categoryId) % HUES.length];
  return {
    background: `rgba(${r}, ${g}, ${b}, 0.12)`,
    bar: `rgb(${r}, ${g}, ${b})`,
  };
}

/** A small stable hash of the id.
 *
 * FNV-1a, ~10 lines and no dependency. It only has to be stable and spread
 * evenly across eight buckets — this is not a security decision, and treating
 * it as one would be the wrong kind of caution.
 */
function hashOf(s: string): number {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return Math.abs(h);
}
