import { describe, expect, it } from "vitest";

import { optionLabel, optionShortLabel, payOptionsFrom, railsOf } from "./tillPayOptions";

const t = {
  methodCash: "Naqd",
  methodCard: "Karta (terminal)",
  methodCardShort: "Karta",
  methodTransfer: "O'tkazma",
};

describe("till payment buttons", () => {
  it("uses the owner's buttons when the server sends them", () => {
    const got = payOptionsFrom(["cash", "card", "debt"], [
      { id: "cash", name: "", kind: "cash" },
      { id: "m_humo", name: "Humo", kind: "card" },
    ]);
    expect(got.map((o) => o.id)).toEqual(["cash", "m_humo"]);
  });

  // ⚠️ An older server sends only the kinds; the till must still have buttons.
  it("rebuilds the defaults from the kinds an older server lists", () => {
    const got = payOptionsFrom(["cash", "card", "payme", "debt"]);
    expect(got).toEqual([
      { id: "cash", name: "", kind: "cash" },
      { id: "card", name: "", kind: "card" },
    ]);
  });

  it("keeps the rails apart from the buttons and the slate", () => {
    expect(railsOf(["cash", "card", "transfer", "payme", "click_pass", "yandex_eats", "debt"])).toEqual([
      "payme",
      "click_pass",
      "yandex_eats",
    ]);
  });

  it("names a default in the till's language and a custom one as the owner typed it", () => {
    expect(optionLabel({ id: "card", name: "", kind: "card" }, t)).toBe("Karta (terminal)");
    expect(optionShortLabel({ id: "card", name: "", kind: "card" }, t)).toBe("Karta");
    expect(optionLabel({ id: "m_x", name: "Humo terminal", kind: "card" }, t)).toBe("Humo terminal");
    expect(optionShortLabel({ id: "m_x", name: "Humo terminal", kind: "card" }, t)).toBe("Humo terminal");
  });
});
