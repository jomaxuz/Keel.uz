import { screen, waitFor } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";
import { expect } from "vitest";

import { formatPrice } from "@/lib/format";
import { adminUz as t } from "@/lib/i18n/admin";

import { PIN } from "./tillServer";

/** Tap a PIN in, one key at a time — the pad submits itself on the fourth
 *  digit, which is a rule worth exercising rather than bypassing. */
export async function unlock(user: UserEvent, code = PIN) {
  for (const d of code) {
    await user.click(screen.getByRole("button", { name: d }));
  }
}

/** Open the drawer through the gate, the way a cashier arriving at eleven does:
 *  one number, one button. */
export async function openShift(user: UserEvent, float = "200000") {
  const input = await screen.findByPlaceholderText("0");
  await user.type(input, float);
  await user.click(screen.getByRole("button", { name: t.till.openShiftTitle }));
}

/** A sum as the screen writes it, ready to be looked for.
 *
 *  ⚠️ Two traps, both invisible in the failure message. `formatPrice` groups
 *  thousands with a **non-breaking** space, so "30 000" typed into a test never
 *  matches; and Testing Library normalises whitespace in the DOM before
 *  comparing, turning that same character into an ordinary space — so the
 *  expectation has to be normalised too. Left alone, the two strings print
 *  identically and the test fails on a character nobody can see. */
export function price(sum: number): string {
  return formatPrice(sum, "UZS", "uz").replace(/\u00a0/g, " ");
}

function tiles(startsWith: string, marker: RegExp) {
  return screen.getAllByRole("button").filter((b) => {
    const text = (b.textContent ?? "").replace(/\s+/g, " ").trim();
    return text.startsWith(startsWith) && marker.test(text);
  });
}

/** A table tile on the floor screen, found the way a waiter finds it: by the
 *  number written on it. Free tiles say so; occupied ones carry their age. */
export function tableTile(number: string) {
  const found = tiles(number, new RegExp(`${t.till.free}|${t.till.minShort}`));
  if (found.length === 0) throw new Error(`no table tile "${number}" on screen`);
  return found[0];
}

/** A dish tile in the menu grid.
 *
 *  ⚠️ Matched on "name, then a price" rather than on the accessible name alone:
 *  "Osh" is also the first word of the check panel's "Oshxonaga yuborish", and
 *  a test that added a dish by pressing *send to the kitchen* would be green
 *  for the wrong reason. */
export function dishTile(name: string) {
  const found = tiles(name, /so'm|сум|UZS/);
  if (found.length === 0) throw new Error(`no dish tile "${name}" on screen`);
  return found[0];
}

/** The room is drawn: the tables are on screen and tappable. */
export async function waitForFloor() {
  await waitFor(() => expect(tableTile("7")).toBeTruthy());
}
