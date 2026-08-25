// What a scanned Asl Belgisi code is allowed to be, on the till's side.
//
// ⚠️ **The same rules as `internal/marking`, and they are duplicated on
// purpose.** The server is what the receipt is filed from and is the authority;
// this copy exists so the refusal happens where somebody is holding the bottle
// and can scan it again. A cashier told "wrong code" *after* the payment has
// been attempted has already had the argument with the guest.
//
// ⚠️ **A scanner is a keyboard.** Pistol scanners run in HID mode: no driver,
// no permission, no device API — the code arrives as typed text ending in
// Enter. That is why nothing here talks to hardware, and why what arrives has
// to be normalised: the separator inside a GS1 code is emitted differently
// depending on the firmware, and a code refused because of a scanner's
// configuration is one the cashier cannot fix.

/** ASCII 29, the GS1 element separator a DataMatrix carries between fields. */
const GS = "\x1d";

/** The shortest real code is longer than a barcode and shorter than a URL,
 *  which are the two things scanned by mistake. */
const MIN = 20;
const MAX = 200;

export function normalizeMark(raw: string): string {
  // ⚠️ Three spellings of one boundary: the character itself, the printable
  // "\x1d" some firmwares type, and the ASCII 232 substitute others are
  // configured with. All three mean the same thing to the bottle.
  let s = raw.split("\\x1d").join(GS);
  s = s.split("\u00e8").join(GS);
  s = s.trim();
  // A trailing separator carries no field after it; kept, it makes two scans
  // of one bottle look like two different codes.
  while (s.endsWith(GS)) s = s.slice(0, -1);
  return s;
}

export type MarkProblem = "empty" | "shape" | "duplicate";

/** Why this code cannot be filed, or null. */
export function checkMark(code: string): MarkProblem | null {
  if (code === "") return "empty";
  if (code.length < MIN || code.length > MAX) return "shape";
  // ⚠️ "01" is the GS1 application identifier for a GTIN, and it is what
  // separates a DataMatrix from the EAN-13 printed beside it on the same
  // bottle. A scanner reads whichever it is pointed at.
  if (!code.startsWith("01")) return "shape";
  for (const ch of code) {
    const c = ch.codePointAt(0)!;
    if (c === 0x1d) continue;
    if (c < 0x20 || c > 0x7e) return "shape";
  }
  return null;
}

/** Whether this code is already on the check.
 *
 *  ⚠️ **The failure a per-line check cannot see.** The pistol beeps, nobody is
 *  sure it took, the bottle is scanned again — and two lines then withdraw one
 *  bottle from circulation while the second stays in it. The register accepts
 *  that receipt. */
export function isDuplicateMark(code: string, existing: string[]): boolean {
  return existing.some((c) => c !== "" && c === code);
}
