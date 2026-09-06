import { describe, expect, it } from "vitest";

import { CARD_WORDS, cardFileName, DEFAULT_DESIGN } from "@/lib/techCardPng";

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
