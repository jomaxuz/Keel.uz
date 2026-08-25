// A quantity somebody types on a shelf: kilograms, litres, portions.
//
// ⚠️ **`type="number"` cannot take a comma, and a comma is what people type.**
// A browser hands `<input type="number">` an empty string for anything it does
// not consider a number, and "9,4" is one of those — so the digits vanish as
// the fourth key is pressed, on the screens most likely to need fractions at
// all: a store counted on a phone, oil measured in litres. The field being
// numeric is also what made the on-screen pad refuse a comma key, because a
// key that types a character the field drops teaches the cashier the pad is
// lying.
//
// So these fields are text, and the meaning is decided here instead of by the
// browser. That moves three rules out of the platform and into one file that
// can be read and tested:
//
//  - a comma is a decimal point (it is the separator on every Uzbek and
//    Russian keyboard, and half of these are typed on a phone);
//  - only one separator survives, because "1.2.3" has no reading;
//  - what is not a digit is not typed — a quantity has no sign, no spaces and
//    no letters, and a field that silently keeps them saves a zero.

/** What the field shows while it is being typed in.
 *
 *  ⚠️ **Kept as text, not parsed and re-printed.** Parsing on every keystroke
 *  turns "9." into "9" the moment the point is pressed, so the fraction can
 *  never be typed at all — the bug this whole file exists to avoid, in the
 *  other direction. A trailing point is a legitimate half-typed number. */
export function normalizeQty(raw: string): string {
  let out = "";
  let dotted = false;
  for (const ch of raw) {
    if (ch >= "0" && ch <= "9") {
      out += ch;
      continue;
    }
    if ((ch === "." || ch === ",") && !dotted) {
      // A leading separator is written as "0." so the field never shows a
      // number that starts with a point — it reads as a typo and copies badly.
      out += out === "" ? "0." : ".";
      dotted = true;
    }
  }
  return out;
}

/** The number behind what is shown. Anything unfinished is zero: an empty
 *  field, a lone "0." — the same answer `Number("")` used to give, so nothing
 *  downstream has to learn a new one. */
export function qtyNumber(raw: string): number {
  const n = Number(normalizeQty(raw));
  return Number.isFinite(n) ? n : 0;
}

/** How a stored quantity is put back into a field.
 *
 *  ⚠️ Zero is shown as an empty field rather than "0": these forms open with
 *  nothing entered, and a pre-filled 0 is a number somebody has to delete
 *  before typing — with a thumb, on a phone, in a cold store. */
export function qtyText(value: number | string): string {
  if (typeof value === "string") return normalizeQty(value);
  if (!Number.isFinite(value) || value === 0) return "";
  return String(value);
}
