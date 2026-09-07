import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import {
  navLabel,
  needsMet,
  orderedFor,
  printKindsFor,
  receiptKindsFor,
  settingsTabLabel,
  settingsTabs,
  SHOP_STOCK_ORDER,
  type Needs,
} from "@/lib/adminNav";
import { adminEn, adminRu, adminUz } from "@/lib/i18n/admin";

// What each kind of business sees in the store section.
//
// ⚠️ **The failure this pins was invisible from a restaurant.** Every screen
// worked; the sidebar simply described somebody else's work — a chemist read
// "Masalliqlar" over its shelf of paracetamol, and found the three screens it
// actually opens weekly (price labels, marked goods, expiry dates) at the bottom
// of the list under four documents it touches once a month. Nothing errors, so
// the only way this is ever checked is here or by logging in as a pharmacy.

/** The store rows as the sidebar actually declares them.
 *
 *  ⚠️ **Read out of the layout rather than duplicated here.** A copy would keep
 *  passing after somebody added a row to the real one — and a row nobody
 *  ordered or gated is exactly the bug this file is about. */
function stockRows(): { key: string; needs?: Needs }[] {
  const src = readFileSync("src/app/admin/layout.tsx", "utf8");
  const start = src.indexOf('key: "stock",\n    items: [');
  if (start < 0) throw new Error("the store group was renamed");
  const end = src.indexOf("\n  },", start);
  return [...src.slice(start, end).matchAll(/\{ href: "[^"]+", key: "(\w+)"(?:, needs: "(\w+)")? \}/g)]
    .map((m) => ({ key: m[1], needs: m[2] as Needs | undefined }));
}

const shown = (brand: { businessType?: string } | null) =>
  orderedFor({ key: "stock", items: stockRows() }, brand)
    .filter((i) => needsMet(i.needs, brand))
    .map((i) => i.key);

const RESTAURANT = null;
const GROCERY = { businessType: "grocery" };
const PHARMACY = { businessType: "pharmacy" };
// ⚠️ **A shop that composes**, and the case that broke the first version of
// this order: its cards and its batches fell to the bottom of the section
// because the order named neither.
const FLOWERS = { businessType: "flowers" };
// ⚠️ **A counter that cooks and sells nothing it bought.** It is neither of the
// two readings this file was written for, which is exactly why it is here.
const BAKERY = { businessType: "bakery" };

describe("the store section, by business", () => {
  // ⚠️ **A florist composes and has no kitchen**, which is the pair of facts
  // the `composes` rule exists for. Its cards belong beside its catalogue, not
  // after the stocktake.
  it("gives a florist its cards, in the middle rather than at the end", () => {
    const rows = shown(FLOWERS);
    expect(rows).toContain("techCards");
    expect(rows.indexOf("techCards")).toBeLessThan(rows.indexOf("writeoffs"));
    expect(rows.indexOf("production")).toBeLessThan(rows.indexOf("stocktake"));
  });

  it("keeps the kitchen's own screens out of a shop", () => {
    for (const shop of [GROCERY, PHARMACY]) {
      expect(shown(shop)).not.toContain("techCards");
      expect(shown(shop)).not.toContain("production");
    }
    expect(shown(RESTAURANT)).toContain("techCards");
    expect(shown(RESTAURANT)).toContain("production");
  });

  it("gives a shop the three screens it opens weekly", () => {
    for (const shop of [GROCERY, PHARMACY]) {
      expect(shown(shop)).toEqual(
        expect.arrayContaining(["labels", "marking", "expiring"]),
      );
    }
    // ⚠️ And keeps them off a kitchen's sidebar, where they are three rows read
    // past forever.
    expect(shown(RESTAURANT)).not.toContain("labels");
  });

  // ⚠️ **The order is the complaint.** Nothing was missing for a grocery; its
  // own screens were simply last, under the monthly paperwork.
  it("puts a shop's weekly screens above its monthly ones", () => {
    const rows = shown(GROCERY);
    const at = (k: string) => rows.indexOf(k);
    expect(at("stock")).toBe(0);
    for (const weekly of ["labels", "marking", "expiring"]) {
      for (const monthly of ["writeoffs", "transfers", "stocktake"]) {
        expect(at(weekly)).toBeLessThan(at(monthly));
      }
    }
  });

  it("leaves a restaurant's order exactly as it was", () => {
    // ⚠️ The kitchen order is the one written in the layout, and re-sorting it
    // would be a change nobody asked for on every restaurant on the platform.
    expect(shown(RESTAURANT)).toEqual(
      stockRows()
        .filter((i) => needsMet(i.needs, RESTAURANT))
        .map((i) => i.key),
    );
  });

  // ⚠️ **The two lists must not drift.** A row added to the layout and not named
  // in the order lands at the end of a shop's sidebar without anybody deciding
  // that — which is how the three screens above got there in the first place.
  it("orders every store row a shop can see, and no row it cannot", () => {
    const rows = stockRows().map((i) => i.key);
    for (const key of SHOP_STOCK_ORDER) expect(rows).toContain(key);
    // ⚠️ Every shop, not just the commonest one: a florist sees two rows a
    // grocery does not, and they were the two nothing ordered.
    for (const shop of [GROCERY, PHARMACY, FLOWERS]) {
      for (const key of shown(shop)) expect(SHOP_STOCK_ORDER).toContain(key);
    }
  });
});

describe("what a row is called", () => {
  it("calls a shop's stock list goods and a kitchen's ingredients", () => {
    expect(navLabel("ingredients", adminUz, PHARMACY)).toBe("Tovarlar");
    expect(navLabel("ingredients", adminUz, RESTAURANT)).toBe("Masalliqlar");
    expect(navLabel("ingredients", adminRu, GROCERY)).toBe("Товары");
    expect(navLabel("ingredients", adminEn, GROCERY)).toBe("Goods");
  });

  it("leaves the warehouse words alone in both", () => {
    // "Kirim" and "Inventarizatsiya" are warehouse words, not kitchen ones, and
    // a second translation of them would be two dictionaries drifting for free.
    for (const brand of [RESTAURANT, PHARMACY]) {
      expect(navLabel("purchases", adminUz, brand)).toBe("Kirim");
      expect(navLabel("stocktake", adminUz, brand)).toBe("Inventarizatsiya");
    }
  });
});

// ⚠️ **An unrecognised business is a restaurant, and it was not.** Every
// predicate in lib/types fell back on its own — so a brand written by a newer
// console answered *no* to all of them at once: no tables, no kitchen, no tech
// cards, and none of the shop screens either. The emptiest sidebar the panel can
// draw, on the newest customer we have. The Go side carries the same note,
// having been fixed there and not here.
it("reads a business it has never heard of as a restaurant", () => {
  const future = { businessType: "bowling-alley" };
  expect(shown(future)).toEqual(shown(RESTAURANT));
  expect(navLabel("ingredients", adminUz, future)).toBe("Masalliqlar");
});

describe("the settings tabs, by business", () => {
  // ⚠️ **The same failure as the store section, one screen along.** A chemist
  // was asked about its "Restoran profili" under a tab called "Zal va
  // buyurtma", which for a shop holds the scale, the marked goods and the
  // till's screen — and no room at all.
  it("calls the two restaurant tabs what a shop calls them", () => {
    expect(settingsTabLabel("restaurant", adminUz, PHARMACY)).toBe("Do'kon");
    expect(settingsTabLabel("hall", adminUz, GROCERY)).toBe("Kassa va javon");
    expect(settingsTabLabel("restaurant", adminUz, RESTAURANT)).toBe("Restoran");
    expect(settingsTabLabel("hall", adminUz, RESTAURANT)).toBe(
      "Zal va buyurtma",
    );
    expect(settingsTabLabel("restaurant", adminRu, GROCERY)).toBe("Магазин");
    expect(settingsTabLabel("restaurant", adminEn, GROCERY)).toBe("Shop");
  });

  it("leaves the tabs that mean the same thing in both alone", () => {
    for (const brand of [RESTAURANT, PHARMACY]) {
      expect(settingsTabLabel("money", adminUz, brand)).toBe(
        adminUz.settings.groups.money,
      );
      expect(settingsTabLabel("site", adminUz, brand)).toBe(
        adminUz.settings.groups.site,
      );
    }
  });

  // ⚠️ **The counter second, delivery near the end.** What a shop sets up on
  // opening day is the scale and the marked goods; delivery is off by default
  // for every shop but a florist, and its tab sat third in front of every
  // chemist on the platform.
  it("puts a shop's counter before its delivery", () => {
    const shop = settingsTabs(GROCERY);
    expect(shop.indexOf("hall")).toBeLessThan(shop.indexOf("delivery"));
    expect(shop.indexOf("hall")).toBe(1);
  });

  it("leaves a restaurant's tabs in the order they always were", () => {
    expect(settingsTabs(RESTAURANT)).toEqual([
      "restaurant",
      "site",
      "hall",
      "delivery",
      "money",
      "integrations",
    ]);
  });

  // ⚠️ Both orders hold every tab: one dropped from a list is a page nobody can
  // reach, and there is no address to type — the tabs are state on one screen.
  it("keeps every tab in both orders", () => {
    expect([...settingsTabs(GROCERY)].sort()).toEqual(
      [...settingsTabs(RESTAURANT)].sort(),
    );
  });

  // The first tab is the same in both, so a page that renders before the brand
  // has loaded opens where it always did rather than jumping when it arrives.
  it("opens on the same tab whatever the business", () => {
    expect(settingsTabs(GROCERY)[0]).toBe(settingsTabs(RESTAURANT)[0]);
  });
});

describe("what a printer prints, and which receipts exist", () => {
  // ⚠️ **A grocery was being asked to design a kitchen ticket** and offered a
  // printer to send it to; a restaurant was offered a price label for a shelf
  // it does not have. Both are ticks that can only ever produce paper nobody
  // reads.
  it("offers a kitchen ticket to a kitchen and a label to a shop", () => {
    expect(printKindsFor(RESTAURANT, [])).toContain("kitchen");
    expect(printKindsFor(RESTAURANT, [])).not.toContain("label");
    expect(printKindsFor(GROCERY, [])).toContain("label");
    expect(printKindsFor(GROCERY, [])).not.toContain("kitchen");
    // The bill and the two receipts are everybody's.
    for (const brand of [RESTAURANT, GROCERY, PHARMACY]) {
      expect(printKindsFor(brand, [])).toEqual(
        expect.arrayContaining(["precheck", "till", "customer"]),
      );
    }
  });

  // ⚠️ **Data wins over the template.** A kind already ticked stays offered
  // whatever the business is — otherwise a setting is switched on and there is
  // no box left to switch it off with. A bakery counter inside a shop is real.
  it("keeps offering a kind this branch already uses", () => {
    const withKitchen = [{ kinds: ["kitchen"] }];
    expect(printKindsFor(GROCERY, withKitchen)).toContain("kitchen");
    const withLabel = [{ kinds: ["label"] }];
    expect(printKindsFor(RESTAURANT, withLabel)).toContain("label");
  });

  it("designs two receipts for a shop and three for a kitchen", () => {
    expect(receiptKindsFor(GROCERY, false)).toEqual(["customer", "till"]);
    expect(receiptKindsFor(RESTAURANT, false)).toEqual([
      "customer",
      "till",
      "kitchen",
    ]);
    // ⚠️ Already switched on: the same rule, so a shop that set one up can
    // still find it.
    expect(receiptKindsFor(PHARMACY, true)).toContain("kitchen");
  });

  // ⚠️ A fast food cooks and has no dining room — the pair of facts that keeps
  // `hasKitchen` and `hasTables` separate questions.
  it("gives a fast food its kitchen ticket", () => {
    expect(receiptKindsFor({ businessType: "fastfood" }, false)).toContain(
      "kitchen",
    );
  });
});

// ⚠️ **A bakery is neither a shop nor a restaurant, and the sidebar has to read
// as one thing to it.** What it sells was flour an hour ago, so it keeps the
// technical cards and the batch document a grocery never opens — and its stock
// list is still "Masalliqlar", because flour is exactly that. The mistake to
// avoid is the florist's in reverse: reading "not a shop" as "a restaurant"
// would hand it a floor plan and a booking list for a counter nobody sits at.
describe("a counter that cooks", () => {
  it("keeps its cards and its batches", () => {
    const rows = shown(BAKERY);
    expect(rows).toContain("techCards");
    expect(rows).toContain("production");
  });

  it("calls its shelf by the kitchen's word, not the shop's", () => {
    expect(navLabel("ingredients", adminUz, BAKERY)).toBe(adminUz.nav.ingredients);
    expect(navLabel("ingredients", adminUz, GROCERY)).toBe(adminUz.nav.goodsList);
  });

  it("is not given a floor plan", () => {
    expect(needsMet("tables", BAKERY)).toBe(false);
    expect(needsMet("composes", BAKERY)).toBe(true);
    expect(needsMet("goods", BAKERY)).toBe(false);
  });
});
