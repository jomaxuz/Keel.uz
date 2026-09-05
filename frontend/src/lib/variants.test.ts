import { describe, expect, it } from "vitest";

import { oneCardPerModel } from "@/lib/types";
import type { MenuItem } from "@/lib/types";

/**
 * ⚠️ **A guest browsing a clothes shop should see shirts, not sizes.** Twelve
 * cards of the same photograph is a catalogue nobody scrolls, and the choice
 * between them cannot be made from a grid anyway — it is made on the page,
 * where every size carries its own price and its own stock.
 */
const item = (id: string, variantOf?: string, variant?: string[]): MenuItem =>
  ({ id, name: "Ko'ylak", price: 100000, variantOf, variant }) as MenuItem;

describe("one card per model", () => {
  it("keeps the first variant and drops its siblings", () => {
    const list = [
      item("a", "m1", ["S"]),
      item("b", "m1", ["M"]),
      item("c", "m1", ["L"]),
    ];
    const cards = oneCardPerModel(list);
    expect(cards).toHaveLength(1);
    expect(cards[0].id).toBe("a");
  });

  it("leaves ordinary products entirely alone", () => {
    // ⚠️ The case that matters most: every menu ever written is this one, and a
    // grouping rule that swallowed a dish would empty a restaurant's menu page.
    const list = [item("a"), item("b"), item("c")];
    expect(oneCardPerModel(list)).toHaveLength(3);
  });

  it("keeps one card per model when there are several", () => {
    const list = [
      item("a", "m1", ["S"]),
      item("b", "m2", ["S"]),
      item("c", "m1", ["M"]),
      item("d"),
    ];
    const cards = oneCardPerModel(list).map((m) => m.id);
    expect(cards).toEqual(["a", "b", "d"]);
  });
});
