import { describe, expect, it } from "vitest";
import { normalizeQty, qtyNumber, qtyText } from "./qty";

// ⚠️ **This fails by saving the wrong number, not by refusing one.** A quantity
// that comes out as 94 instead of 9.4 writes a shelf that is ten times too
// full, and the difference surfaces weeks later at a count, as an accusation
// against whoever counted.

describe("what a person is allowed to type", () => {
  it("reads a comma as the decimal point", () => {
    // The separator on every keyboard these are typed on.
    expect(normalizeQty("9,4")).toBe("9.4");
    expect(qtyNumber("9,4")).toBe(9.4);
  });

  it("keeps a half-typed number half-typed", () => {
    // ⚠️ The whole reason the field holds text: parsing here would turn "9."
    // into "9" as the point is pressed, and the fraction could never be typed.
    expect(normalizeQty("9.")).toBe("9.");
    expect(normalizeQty("9,")).toBe("9.");
  });

  it("takes only the first separator", () => {
    // A second one has no reading, and dropping the rest is what a person
    // pressing the key twice by accident expects.
    expect(normalizeQty("1.2.3")).toBe("1.23");
    expect(normalizeQty("1,2,3")).toBe("1.23");
  });

  it("writes a leading separator as a zero and a point", () => {
    expect(normalizeQty(",5")).toBe("0.5");
    expect(normalizeQty(".5")).toBe("0.5");
  });

  it("refuses everything a quantity cannot contain", () => {
    // Signs, spaces, letters and a paste from a spreadsheet.
    expect(normalizeQty("-5")).toBe("5");
    expect(normalizeQty("9 kg")).toBe("9");
    expect(normalizeQty("1 234,5")).toBe("1234.5");
    expect(normalizeQty("abc")).toBe("");
  });
});

describe("the number that reaches the server", () => {
  it("treats anything unfinished as zero", () => {
    // The answer `Number("") || 0` already gave, kept so nothing downstream
    // has to learn a new one.
    expect(qtyNumber("")).toBe(0);
    expect(qtyNumber(".")).toBe(0);
    expect(qtyNumber("abc")).toBe(0);
  });

  it("carries fractions through untouched", () => {
    expect(qtyNumber("0.25")).toBe(0.25);
    expect(qtyNumber("1234.5")).toBe(1234.5);
  });
});

describe("putting a stored quantity back on screen", () => {
  it("shows nothing rather than a zero somebody has to delete", () => {
    expect(qtyText(0)).toBe("");
    expect(qtyText(NaN)).toBe("");
  });

  it("keeps a real number readable", () => {
    expect(qtyText(9.4)).toBe("9.4");
    expect(qtyText(12)).toBe("12");
  });

  it("cleans a stored string the same way typing is cleaned", () => {
    // Values that were saved as text before this existed.
    expect(qtyText("9,4")).toBe("9.4");
  });
});
