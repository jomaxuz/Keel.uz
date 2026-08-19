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

/** A table tile on the floor screen, found the way a waiter finds it: by the
 *  number written on it. */
export function tableTile(number: string) {
  return screen.getByRole("button", {
    name: new RegExp(`^${number}\\s*·`),
  });
}

/** A dish tile in the menu grid.
 *
 *  ⚠️ By accessible name, which the tile sets to the dish alone. Matching the
 *  rendered text instead matched "Oshxonaga yuborish" for a dish called "Osh" —
 *  a test that added a dish by pressing *send to the kitchen* and passed. */
export function dishTile(name: string) {
  return screen.getByRole("button", { name });
}

/** The room is drawn: the tables are on screen and tappable. */
export async function waitForFloor() {
  await waitFor(() => expect(tableTile("7")).toBeTruthy());
}
