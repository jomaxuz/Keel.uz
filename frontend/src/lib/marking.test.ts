import { describe, expect, it } from "vitest";

import { checkMark, isDuplicateMark, normalizeMark } from "./marking";

const GS = "\x1d";
const good = "0104607034170203215Fw2R" + GS + "93dGVz";

describe("what the scanner types", () => {
  it("means the same code whichever separator the firmware sends", () => {
    for (const raw of [
      good,
      "0104607034170203215Fw2R\\x1d93dGVz",
      "0104607034170203215Fw2R\u00e893dGVz",
      "  " + good + "\n",
      good + GS,
    ]) {
      expect(normalizeMark(raw)).toBe(good);
    }
  });
});

describe("what cannot be a marking code", () => {
  it("refuses the barcode printed next to it", () => {
    // The mistake that actually happens: the pistol is pointed at the EAN-13.
    expect(checkMark(normalizeMark("4607034170203"))).toBe("shape");
  });

  it("refuses a half-read code and an empty field", () => {
    expect(checkMark(normalizeMark("010460703417020"))).toBe("shape");
    expect(checkMark("")).toBe("empty");
  });

  it("accepts a real one", () => {
    expect(checkMark(good)).toBeNull();
  });
});

describe("the same bottle scanned twice", () => {
  it("is caught against the codes already on the check", () => {
    expect(isDuplicateMark(good, ["", "0104607034170203215Zz9Q"])).toBe(false);
    expect(isDuplicateMark(good, [good])).toBe(true);
  });
});
