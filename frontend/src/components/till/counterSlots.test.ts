import { describe, expect, it } from "vitest";

import type { Check } from "@/lib/types";
import { counterSlots } from "./TablesScreen";

// ⚠️ **Closing #2 must not renumber #3.** The strip used to number counter
// checks by their position, so every close shifted the guests still waiting.
const check = (id: string, counterNo?: number, openedAt = "2026-10-04T10:00:00Z") =>
  ({ id, counterNo, openedAt }) as unknown as Check;

describe("counter slots", () => {
  it("keep the number the server gave, whatever closes around them", () => {
    const slots = counterSlots([check("a", 1), check("c", 3), check("d", 4)]);
    expect(slots.get("c")).toBe(3);
    expect(slots.get("d")).toBe(4);
  });

  it("give a check with no slot a number no slotted check holds", () => {
    const slots = counterSlots([
      check("a", 2),
      check("old", undefined, "2026-10-04T09:00:00Z"),
      check("new", undefined, "2026-10-04T11:00:00Z"),
    ]);
    expect(slots.get("a")).toBe(2);
    expect(slots.get("old")).toBe(3);
    expect(slots.get("new")).toBe(4);
  });
});
