import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import {
  composes,
  hasKitchen,
  hasTables,
  hasVariants,
  sellsGoods,
  sellsOnlineOnly,
  shipsByPost,
} from "@/lib/types";

// The business-type predicates, and the one rule that binds them to the server.
//
// ⚠️ **These five functions are a copy of `models/businesstype.go`, joined to it
// by nothing the compiler can see.** The two repositories do not import each
// other, so the only thing that can notice a type added on one side and
// forgotten on the other is a test — and the way that failure shows up is not an
// error but the emptiest sidebar the panel can draw, on the newest customer we
// have (see the note on `known()` in lib/types.ts, which is that bug's
// headstone).

/** Every type the server knows, read out of its own source. */
function serverTypes(): string[] {
  const src = readFileSync(
    "../backend/internal/models/businesstype.go",
    "utf8",
  );
  const start = src.indexOf("var BusinessTypes = []BusinessType{");
  if (start < 0) throw new Error("BusinessTypes was renamed on the server");
  const body = src.slice(start, src.indexOf("\n}", start));
  return [...body.matchAll(/\bBiz(\w+)\b/g)].map((m) =>
    m[1] === "Restaurant" ? "" : m[1].toLowerCase(),
  );
}

/** And every type this panel knows, read out of its own. */
function panelTypes(): string[] {
  const src = readFileSync("src/lib/types.ts", "utf8");
  const start = src.indexOf("const BUSINESS_TYPES: readonly string[] = [");
  if (start < 0) throw new Error("BUSINESS_TYPES was renamed");
  const body = src.slice(start, src.indexOf("\n];", start));
  return [...body.matchAll(/"([a-z]*)"/g)].map((m) => m[1]);
}

describe("the panel and the server agree on what a business can be", () => {
  it("knows every type the server offers", () => {
    expect(panelTypes().sort()).toEqual(serverTypes().sort());
  });
});

describe("an online store", () => {
  const shop = { businessType: "ecommerce" };

  // It sells the packet it bought, like every other shop.
  it("sells goods and composes nothing", () => {
    expect(sellsGoods(shop)).toBe(true);
    expect(composes(shop)).toBe(false);
    expect(hasKitchen(shop)).toBe(false);
    expect(hasTables(shop)).toBe(false);
  });

  // ⚠️ And it is the only one with no room, which is what decides whether a
  // navigation bar is worth drawing rather than assuming.
  it("is the only business with no room to walk into", () => {
    expect(sellsOnlineOnly(shop)).toBe(true);
    for (const t of panelTypes().filter((t) => t !== "ecommerce")) {
      expect(sellsOnlineOnly({ businessType: t })).toBe(false);
    }
  });

  it("sells one model in sizes and colours", () => {
    expect(hasVariants(shop)).toBe(true);
    expect(hasVariants({ businessType: "clothing" })).toBe(true);
    // ⚠️ A box of paracetamol has no colour, and a grocery's pack sizes are
    // separate products with separate barcodes — a matrix generator there
    // would double its catalogue by accident.
    expect(hasVariants({ businessType: "pharmacy" })).toBe(false);
    expect(hasVariants({ businessType: "grocery" })).toBe(false);
    expect(hasVariants(null)).toBe(false);
  });
});

// ⚠️ **A fact about the goods, not about the shop**, which is why this is not
// `sellsGoods`. A butcher and a florist sell what they bought exactly as a
// boutique does, and a parcel of mince or of tulips is a parcel nobody
// collects.
describe("which businesses are offered a postal carrier", () => {
  it("offers one to the three whose goods travel in a parcel", () => {
    for (const t of ["ecommerce", "clothing", "cosmetics"]) {
      expect(shipsByPost({ businessType: t })).toBe(true);
    }
  });

  it("offers one to nobody else", () => {
    for (const t of ["butcher", "flowers", "grocery", "pharmacy", "hardware"]) {
      expect(shipsByPost({ businessType: t })).toBe(false);
    }
    // A restaurant, and a brand from a newer console we cannot read.
    expect(shipsByPost(null)).toBe(false);
    expect(shipsByPost({ businessType: "" })).toBe(false);
    expect(shipsByPost({ businessType: "spaceport" })).toBe(false);
  });
});
