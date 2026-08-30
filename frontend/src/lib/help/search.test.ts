import { describe, expect, it } from "vitest";

import { HELP } from "./articles";
import { searchHelp } from "./search";

const uz = HELP.uz;
const ids = (q: string) => searchHelp(uz, q).map((h) => h.article.id);

describe("the help search", () => {
  // ⚠️ The whole reason this is not a whole-word match. Uzbek is agglutinative
  // and a person types the stem.
  it("finds a word by its stem", () => {
    expect(ids("chek")).toContain("receipt-language");
    expect(ids("printer")).toContain("receipt-garbage");
    expect(ids("kassa")).toContain("pin-locked");
  });

  // ⚠️ And the other direction: the article says "chekda", the person typed
  // "chek".
  it("finds a suffixed word in an article", () => {
    expect(ids("savol belgi")).toContain("receipt-garbage");
  });

  // The way people actually ask: a sentence, not a keyword.
  it("answers a whole question", () => {
    expect(ids("chek rus tilida chiqmayapti")[0]).toBe("receipt-language");
    expect(ids("kassa ochilmayapti")[0]).toBe("pin-locked");
  });

  // ⚠️ Every word must hit. Otherwise one common word drags in every article
  // that contains it and buries the one that answers the question.
  it("does not answer on one word out of five", () => {
    expect(ids("chek bilan hech qanday aloqasi yo'q savol")).toHaveLength(0);
  });

  it("says nothing rather than guessing", () => {
    expect(ids("zzzzz")).toHaveLength(0);
    expect(ids("")).toHaveLength(0);
    // Two letters carry no meaning and would match half the base.
    expect(ids("ka")).toHaveLength(0);
  });

  // A title hit is worth more than a passing mention in a body.
  it("puts the article that is about it first", () => {
    const found = searchHelp(uz, "tannarx");
    expect(found[0].article.id).toBe("stock-cost");
  });

  // ⚠️ The apostrophe is a letter in Uzbek. Splitting on it would make
  // "o'zgartirish" two words and stop the stem matching.
  it("treats an apostrophe as part of the word", () => {
    expect(ids("o'zgartir")).toContain("menu-price");
  });

  // The same question twice gives the same order.
  it("is stable", () => {
    expect(ids("kassa")).toEqual(ids("kassa"));
  });
});

describe("the article base", () => {
  it("has no duplicate ids in any language", () => {
    for (const [lang, list] of Object.entries(HELP)) {
      const seen = new Set<string>();
      for (const a of list) {
        expect(seen.has(a.id), `${lang}: ${a.id} twice`).toBe(false);
        seen.add(a.id);
      }
    }
  });

  // ⚠️ A translated article must answer the same question. Matched by id, so a
  // Russian article under an id that means something else in Uzbek would show
  // the wrong answer under the right title.
  it("keeps a translated article on the same subject", () => {
    const uzByID = new Map(HELP.uz.map((a) => [a.id, a]));
    for (const other of [HELP.ru, HELP.en]) {
      for (const a of other) {
        const base = uzByID.get(a.id);
        expect(base, `${a.id} has no Uzbek original`).toBeTruthy();
        expect(base?.cat, `${a.id} changed category`).toBe(a.cat);
      }
    }
  });
});
