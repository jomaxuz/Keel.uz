/**
 * The shopping list, driven the way a cashier drives it: pick, count, send.
 *
 * ⚠️ **These assertions are about the three steps, not the markup.** The screen
 * this replaced put everything in one column, and the failures it produced were
 * all reachable with a finger: a list sent with a line nobody wanted because
 * the picker only added, and a quantity typed into a field whose unit had
 * scrolled off the top. So the tests here hold the two things that turn a tap
 * into somebody else's morning — that a step cannot be left before it has been
 * answered, and that the pack flag travels with the number it belongs to.
 */

import { screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { adminUz as t } from "@/lib/i18n/admin";
import { bindDevice, renderTill } from "@/test/render";
import { setSignedInStaff, tillStaff } from "@/test/staffMock";
import { installTillServer, type TillServer } from "@/test/tillServer";

import ZakupScreen from "./ZakupScreen";

let server: TillServer;

beforeEach(() => {
  server = installTillServer();
  bindDevice();
  setSignedInStaff(tillStaff());
});

function render() {
  return renderTill(<ZakupScreen onError={() => {}} />);
}

/** The picker card for one ingredient — a button, so it toggles. */
async function card(name: string) {
  return (await screen.findByText(name)).closest("button") as HTMLElement;
}

describe("Bozorlik", () => {
  it("offers the whole catalogue, not only what is short", async () => {
    render();
    // Two the arithmetic flagged...
    expect(await card("Kartoshka")).toBeInTheDocument();
    // ...and one it did not, which is every ingredient in a restaurant that
    // never set a minimum.
    expect(await card("Limon")).toBeInTheDocument();
  });

  it("will not move on until something is chosen", async () => {
    const { user } = render();
    await card("Kartoshka");

    const next = screen.getByRole("button", { name: t.zakup.next });
    expect(next).toBeDisabled();

    await user.click(await card("Kartoshka"));
    expect(next).toBeEnabled();
  });

  it("takes a line back on a second tap", async () => {
    const { user } = render();
    await user.click(await card("Kartoshka"));
    expect(await card("Kartoshka")).toHaveAttribute("aria-pressed", "true");

    await user.click(await card("Kartoshka"));
    expect(await card("Kartoshka")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("button", { name: t.zakup.next })).toBeDisabled();
  });

  it("sends what was picked, counted and read back", async () => {
    const { user } = render();

    // 1 — what do we need.
    await user.click(await card("Kartoshka"));
    await user.click(screen.getByRole("button", { name: t.zakup.next }));

    // 2 — how much of it. The shortage pre-filled the figure; a manager may
    // overwrite it, and here somebody does.
    const qty = screen.getByDisplayValue("12");
    await user.clear(qty);
    await user.type(qty, "20");
    await user.click(screen.getByRole("button", { name: t.zakup.next }));

    // 3 — is this right. Read back in the unit it will arrive in.
    expect(screen.getByText("20 kg")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: t.zakup.send }));
    await waitFor(() => expect(server.calls.buyOrders).toHaveLength(1));
    expect(server.calls.buyOrders[0].lines).toEqual([
      { ingredientId: "ing-1", name: "Kartoshka", qty: 20, pack: undefined },
    ]);
  });

  it("sends the pack flag rather than the converted figure", async () => {
    const { user } = render();

    await user.click(await card("Un"));
    await user.click(screen.getByRole("button", { name: t.zakup.next }));

    // The unit is a switch only where a market packaging is written down.
    const row = screen.getByDisplayValue("25").closest("li") as HTMLElement;
    await user.click(within(row).getByRole("button", { name: "kg" }));

    const qty = within(row).getByDisplayValue("25");
    await user.clear(qty);
    await user.type(qty, "2");

    await user.click(screen.getByRole("button", { name: t.zakup.next }));
    // ⚠️ Both, because "2" leaves the building on its own otherwise.
    expect(screen.getByText("2 qop = 50 kg")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: t.zakup.send }));
    await waitFor(() => expect(server.calls.buyOrders).toHaveLength(1));
    // ⚠️ The number typed and the flag — never 50. The server converts, and a
    // screen that converted too would buy two tonnes of flour.
    expect(server.calls.buyOrders[0].lines[0]).toMatchObject({
      name: "Un",
      qty: 2,
      pack: true,
    });
  });
});
