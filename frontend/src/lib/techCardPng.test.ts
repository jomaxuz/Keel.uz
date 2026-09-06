import { describe, expect, it } from "vitest";

import {
  CARD_COLUMN_ORDER,
  CARD_WORDS,
  cardFileName,
  DEFAULT_DESIGN,
  nettoOf,
} from "@/lib/techCardPng";

/**
 * ⚠️ **The sheet and its file are one thing.** A Russian card that lands as
 * "Texnologik-karta-…" is a file somebody renames before sending it on — which
 * is the whole reason the language is a choice on that screen rather than the
 * language of whoever pressed the button.
 */
describe("what the file is called", () => {
  it("is named in the language the sheet is written in", () => {
    expect(cardFileName("Lag'mon", "uz")).toBe("Texnologik-karta-Lag'mon.png");
    expect(cardFileName("Лагман", "ru")).toBe("Тех-карта-Лагман.png");
    expect(cardFileName("Lagman", "en")).toBe("Tech-card-Lagman.png");
  });

  it("drops what a filesystem refuses", () => {
    // ⚠️ A slash in a dish name is a directory on one operating system and an
    // error on another — and "Choy 1/2" is an ordinary thing to call a dish.
    expect(cardFileName('Choy 1/2 "katta"', "uz")).toBe(
      "Texnologik-karta-Choy-1-2-katta.png",
    );
  });

  it("numbers the pages only when there is more than one", () => {
    expect(cardFileName("Osh", "uz", 1, 1)).toBe("Texnologik-karta-Osh.png");
    expect(cardFileName("Osh", "uz", 2, 3)).toBe("Texnologik-karta-Osh-2.png");
  });
});

/**
 * ⚠️ **A half-translated sheet is a document in two languages**, and nothing on
 * the screen would show it: the preview is drawn in whichever language is
 * selected, so a missing word is only ever seen by the person who receives the
 * card. The words live beside the drawing for that reason, and this checks
 * every language carries the same set.
 */
describe("the words on the sheet", () => {
  it("exist in all three languages", () => {
    const keys = Object.keys(CARD_WORDS.uz).sort();
    for (const lang of ["ru", "en"] as const) {
      expect(Object.keys(CARD_WORDS[lang]).sort()).toEqual(keys);
      for (const k of keys) {
        expect(
          CARD_WORDS[lang][k as keyof typeof CARD_WORDS.uz],
          `${lang}.${k}`,
        ).toBeTruthy();
      }
    }
  });

  it("never leaves the ingredient column out of the default", () => {
    // ⚠️ A card with no ingredient column is not a card. The constructor does
    // not offer to remove it; this pins the default it starts from.
    expect(DEFAULT_DESIGN.columns).toContain("name");
  });
});

/**
 * ⚠️ **The card is drawn for a cook, and a cook weighs netto.** The recipe
 * figure is brutto — what leaves the store, which is what the dish costs
 * whether or not a third of it is peel — so netto is derived and never used for
 * money. Getting this backwards would understate every dish on every report,
 * quietly and everywhere.
 */
describe("what actually goes in the pot", () => {
  const line = (qty: number, waste?: number) => ({
    name: "Kartoshka",
    qty,
    unit: "g",
    rate: 4,
    waste,
  });

  it("takes the peel off the brutto figure", () => {
    expect(nettoOf(line(1000, 25))).toBe(750);
  });

  it("is the same weight when nothing is thrown away", () => {
    // ⚠️ Zero waste is netto equal to brutto, not a blank: flour, salt and oil
    // are exactly that, and a column with holes in it beside the lines that do
    // have peel reads as a card somebody forgot to finish.
    expect(nettoOf(line(120))).toBe(120);
    expect(nettoOf(line(120, 0))).toBe(120);
  });

  it("refuses a waste that would empty the line", () => {
    // ⚠️ A hundred per cent is an ingredient of which nothing reaches the pot,
    // and "0 g" beside a line somebody weighs is worse than no column at all.
    // The server clamps it too; this is the half a stale tab cannot get past.
    // ⚠️ Compared loosely: 100 × (1 − 0.99) is not exactly 1 in binary
    // floating point. The sheet never shows the difference — `qty` rounds to
    // two decimals — and a test that demanded exactness here would be pinning
    // the arithmetic of the language rather than the rule.
    expect(nettoOf(line(100, 100))).toBeCloseTo(1, 6);
    expect(nettoOf(line(100, -5))).toBe(100);
  });

  it("orders the columns the same way the table draws them", () => {
    // ⚠️ Netto sits beside the quantity it is derived from. Anywhere else on
    // the sheet and a cook reads two numbers with a price between them.
    expect(CARD_COLUMN_ORDER).toEqual([
      "no",
      "name",
      "qty",
      "netto",
      "rate",
      "cost",
    ]);
  });
});
