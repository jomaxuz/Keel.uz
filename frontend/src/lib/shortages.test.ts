import { describe, expect, it } from "vitest";

import { verdictsFor } from "@/lib/shortages";
import { adminEn, adminRu, adminUz } from "@/lib/i18n/admin";

// ⚠️ **A chemist cannot answer "the card takes more than the kitchen does".**
// A grocery, a pharmacy and a clothes shop sell the packet they bought, and the
// one-line card behind it is written by the server — it cannot be padded, so
// that answer is a reason that cannot be true. Offering it is how a screen
// teaches a shop that the panel was built for somebody else.
describe("which answers a business is offered", () => {
  it("keeps the card answer where something is composed", () => {
    for (const type of ["", "fastfood", "flowers"]) {
      expect(verdictsFor({ businessType: type })).toContain("card");
    }
  });

  it("drops it where the packet on the shelf is the packet that was sold", () => {
    for (const type of ["grocery", "pharmacy", "clothing"]) {
      expect(verdictsFor({ businessType: type })).not.toContain("card");
    }
  });

  // ⚠️ **A brand that has not loaded yet reads as a restaurant**, exactly as
  // every other business-type rule does — and every brand written before the
  // field existed is one.
  it("reads an unknown or missing brand as a restaurant", () => {
    expect(verdictsFor(null)).toContain("card");
    expect(verdictsFor({ businessType: "bakery-with-a-cinema" })).toContain(
      "card",
    );
  });

  // ⚠️ **The mis-scan answer is offered to everybody, and a shop cannot do
  // without it.** Two similar packets and one barcode leave this row short and
  // its twin over; a grocery without this answer files every one of them under
  // "lost", and a month of those reads as theft in a shop that has a barcode
  // problem. A waiter taps the wrong tile for the same reason.
  it("offers the mis-scan answer to every business", () => {
    for (const type of ["", "fastfood", "grocery", "pharmacy", "flowers"]) {
      expect(verdictsFor({ businessType: type })).toContain("swap");
    }
  });

  // ⚠️ **"Lost" is last wherever it appears.** It is the only entry a reader
  // can hear as an accusation, and a list that opens with it invites the whole
  // queue to be filed under it.
  it("puts the answer that admits defeat at the end", () => {
    for (const type of ["", "grocery"]) {
      const list = verdictsFor({ businessType: type });
      expect(list[list.length - 1]).toBe("lost");
    }
  });

  // Every offered answer has a word in all three languages — a select with an
  // empty option is a select nobody can use, and it would only be empty for
  // whichever language nobody on the team reads.
  it("has a label in all three languages", () => {
    for (const dict of [adminUz, adminRu, adminEn]) {
      for (const v of verdictsFor(null)) {
        expect(dict.shortages.verdicts[v]).toBeTruthy();
      }
    }
  });
});
