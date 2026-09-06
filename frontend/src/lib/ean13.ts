// The bars an EAN-13 actually has.
//
// ⚠️ **Computed, not sketched.** The label chooser is a screen whose entire job
// is "what will come out of the printer", and a barcode drawn as a repeating
// stripe answers that question with a decoration. The encoding is a table and a
// parity rule — the same ones the printer's firmware applies to the same digits
// — so the pattern here is the pattern on the paper, and a shop looking at the
// card can see that the code is on it and how much room it takes.
//
// ⚠️ **It is still not a promise that a scan will work.** Whether a thermal head
// resolves the bars is a question about the printer, the roll and the label
// width, and the only honest answer to it is the test print. What this gets
// right is the layout.

/** Left-hand digits, odd parity. */
const L = [
  "0001101", "0011001", "0010011", "0111101", "0100011",
  "0110001", "0101111", "0111011", "0110111", "0001011",
];

/** Left-hand digits, even parity. */
const G = [
  "0100111", "0110011", "0011011", "0100001", "0011101",
  "0111001", "0000101", "0010001", "0001001", "0010111",
];

/** Right-hand digits — the complement of L, which is what makes a scan
 *  readable in either direction. */
const R = L.map((s) =>
  [...s].map((c) => (c === "0" ? "1" : "0")).join(""),
);

/** Which of the first six digits are even-parity. ⚠️ This is where the
 *  thirteenth digit lives: an EAN-13 encodes twelve digits and hides the first
 *  one in the parity pattern of the other six. */
const PARITY = [
  "LLLLLL", "LLGLGG", "LLGGLG", "LLGGGL", "LGLLGG",
  "LGGLLG", "LGGGLL", "LGLGLG", "LGLGGL", "LGGLGL",
];

/** The check digit for the first twelve. */
export function ean13Check(first12: string): number | null {
  if (!/^\d{12}$/.test(first12)) return null;
  let sum = 0;
  for (let i = 0; i < 12; i++) {
    sum += Number(first12[i]) * (i % 2 === 0 ? 1 : 3);
  }
  return (10 - (sum % 10)) % 10;
}

/** True when this is a whole, self-consistent EAN-13. */
export function isEan13(code: string): boolean {
  if (!/^\d{13}$/.test(code)) return false;
  return ean13Check(code.slice(0, 12)) === Number(code[12]);
}

/** The 95 modules of an EAN-13, as "1" for a bar and "0" for a space.
 *
 *  ⚠️ Null for anything that is not one. A shop's own codes are EAN-13 by
 *  construction (internal/barcode), but a product can carry whatever the
 *  manufacturer printed on it, and drawing 95 modules of nonsense would be a
 *  picture of a code that does not exist. */
export function ean13Modules(code: string): string | null {
  if (!isEan13(code)) return null;
  const lead = Number(code[0]);
  const left = code.slice(1, 7);
  const right = code.slice(7);
  let out = "101"; // start guard
  for (let i = 0; i < 6; i++) {
    out += PARITY[lead][i] === "L" ? L[Number(left[i])] : G[Number(left[i])];
  }
  out += "01010"; // centre guard
  for (const d of right) out += R[Number(d)];
  return out + "101"; // end guard
}

/** Where the guard bars are, in module positions. ⚠️ They are drawn longer than
 *  the rest — that is not decoration, it is what the standard says, and it is
 *  what makes a printed barcode look like one. */
export const EAN13_GUARDS: [number, number][] = [
  [0, 3],
  [45, 50],
  [92, 95],
];
