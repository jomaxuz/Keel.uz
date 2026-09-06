import { describe, expect, it } from "vitest";

import { ean13Check, ean13Modules, isEan13 } from "@/lib/ean13";

describe("EAN-13", () => {
  // ⚠️ GS1's own published example, not one invented here. A made-up code that
  // fails its own check digit is how the backend's first test was wrong.
  it("agrees with the standard's example", () => {
    expect(ean13Check("400638133393")).toBe(1);
    expect(isEan13("4006381333931")).toBe(true);
  });

  it("rejects a code whose check digit does not hold", () => {
    expect(isEan13("4006381333932")).toBe(false);
    expect(isEan13("21000000000")).toBe(false);
    expect(isEan13("abcdefghijklm")).toBe(false);
  });

  // The shop's own internal codes, which is what most of these labels carry.
  //
  // ⚠️ **This is the check that caught a made-up sample**: the label preview
  // shipped with "2100000000017", whose check digit is 2 — so the printer would
  // have sent it as CODE128 while the chooser drew an EAN. Second time an EAN
  // has been invented in this repository.
  it("accepts an in-store code", () => {
    expect(isEan13("2100000000012")).toBe(true);
    expect(isEan13("2100000000017")).toBe(false);
  });

  it("draws 95 modules, with the three guards", () => {
    const m = ean13Modules("2100000000012");
    expect(m).not.toBeNull();
    expect(m).toHaveLength(95);
    expect(m!.slice(0, 3)).toBe("101");
    expect(m!.slice(45, 50)).toBe("01010");
    expect(m!.slice(92)).toBe("101");
  });

  it("draws nothing for a code that is not an EAN-13", () => {
    // ⚠️ A picture of 95 modules of nonsense is a picture of a code that does
    // not exist — worse than saying there is nothing to draw.
    expect(ean13Modules("HELLO")).toBeNull();
  });
});
