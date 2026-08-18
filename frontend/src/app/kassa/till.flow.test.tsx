/**
 * The till, driven the way it is used: PIN, drawer, room, check, dish, total.
 *
 * ⚠️ **These tests exist because of what the live trial found.** Every defect
 * that day was on the screen and invisible to the Go tests: the floor was
 * imported and never rendered, the option dialog did not exist so a dish with a
 * required group could not be sold at all, and there was no way back to the
 * room on a monoblock with no browser chrome. A server test cannot fail for any
 * of those.
 */

import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { adminUz as t } from "@/lib/i18n/admin";
import { bindDevice, renderTill } from "@/test/render";
import { setSignedInStaff, tillStaff } from "@/test/staffMock";
import {
  dishTile,
  openShift,
  price,
  tableTile,
  unlock,
  waitForFloor,
} from "@/test/tillFlow";
import {
  installTillServer,
  OPTION_DISH,
  PLAIN_DISH,
  type TillServer,
} from "@/test/tillServer";

import TillPage from "./page";

let server: TillServer;

beforeEach(() => {
  server = installTillServer();
  bindDevice();
});

describe("the lock screen", () => {
  it("shows nothing but the pad until somebody names themselves", async () => {
    renderTill(<TillPage />);

    expect(await screen.findByText(t.till.pinTitle)).toBeInTheDocument();
    // ⚠️ The whole point of the gate: a till that kept working while locked
    // would be the old behaviour with a pad in front of it.
    expect(screen.queryByText(t.till.openChecks)).not.toBeInTheDocument();
    expect(screen.queryByText(PLAIN_DISH)).not.toBeInTheDocument();
  });

  it("keeps the wrong code out and says so in the server's words", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);

    await unlock(user, "9999");

    expect(await screen.findByText("Kod noto'g'ri")).toBeInTheDocument();
    expect(screen.getByText(t.till.pinTitle)).toBeInTheDocument();
  });
});

describe("the shift gate", () => {
  it("draws the drawer instead of the room, not above it", async () => {
    server = installTillServer({ shiftOpen: false });
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);

    expect(await screen.findByText(t.till.shiftClosed)).toBeInTheDocument();
    // ⚠️ A banner on a working till is a banner that gets worked past: the
    // first guest is already standing there. So the floor must not be reachable.
    expect(screen.queryByText(t.till.counter)).not.toBeInTheDocument();
  });

  it("opens the room once the float is entered", async () => {
    server = installTillServer({ shiftOpen: false });
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await screen.findByText(t.till.shiftClosed);

    await openShift(user, "200000");

    await waitForFloor();
    expect(server.calls.openShift).toEqual([200000]);
  });
});

describe("the floor", () => {
  it("is what the till opens on — the menu comes second", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);

    // ⚠️ The regression that shipped: TablesScreen was imported, the `view`
    // state existed, and it was missing from the JSX — so "the room first" was
    // believed done while the till still opened on dishes.
    await waitForFloor();
    expect(tableTile("8")).toBeInTheDocument();
    expect(screen.queryByText(PLAIN_DISH)).not.toBeInTheDocument();
  });

  it("offers the room three ways, and opens on the plan when there is one", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    // ⚠️ The plan is only offered when the owner has drawn one: coordinates
    // default to zero, so a branch that filled in table numbers and never
    // opened the editor would get every table stacked in the corner — a room
    // that reads as broken.
    expect(
      screen.getByRole("button", { name: t.till.planView }),
    ).toBeInTheDocument();

    // The cards view answers the question the room cannot: what is *on* table
    // 7, without walking there and opening its check.
    await user.click(screen.getByRole("button", { name: t.till.waiterView }));
    expect(
      await screen.findByRole("button", {
        name: new RegExp(`^${t.till.allWaiters}`),
      }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: t.till.gridView }));
    await waitForFloor();
  });

  it("puts the takeaway counter behind its own tab", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    // Two zones means a strip; the hall is showing, so 101 is not.
    expect(screen.queryByText("101")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Saboy" }));
    expect(await screen.findByText("101")).toBeInTheDocument();
  });
});

describe("the header", () => {
  it("keeps the lock reachable by name now that it is only an icon", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    // ⚠️ An icon-only button is the ordinary way a label gets lost: nothing on
    // screen changes when the accessible name goes, and the control simply
    // stops existing for a screen reader — and for this test, which is how the
    // loss gets noticed at all.
    expect(
      screen.getByRole("button", { name: t.till.lock }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: t.till.lock }));

    // And it really locks: the pad is back and the room is gone.
    expect(await screen.findByText(t.till.pinTitle)).toBeInTheDocument();
    expect(screen.queryByText(t.till.openChecks)).not.toBeInTheDocument();
  });
});

describe("selling", () => {
  async function reachTheMenu(user: Awaited<ReturnType<typeof renderTill>>["user"]) {
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();
    await user.click(tableTile("7"));
    // The tap on a free table answered "which table"; the dialog only asks how
    // many.
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    await screen.findByText(PLAIN_DISH);
  }

  it("opens a check on the tapped table and swaps the room for the menu", async () => {
    const { user } = renderTill(<TillPage />);
    await reachTheMenu(user);

    expect(server.calls.openCheck).toEqual([{ tableId: "t1", guests: 2 }]);
    // Named on the check panel and on the rail: the cashier must be able to
    // see whose bill this is from either side of the screen.
    expect(
      screen.getAllByText(`7-${t.till.table.toLowerCase()}`).length,
    ).toBeGreaterThan(0);
  });

  it("adds a plain dish in one tap", async () => {
    const { user } = renderTill(<TillPage />);
    await reachTheMenu(user);

    await user.click(dishTile(PLAIN_DISH));

    await waitFor(() =>
      expect(server.calls.addLines).toEqual([
        { checkId: "chk-1", menuItemId: "m1", qty: 1 },
      ]),
    );
    // And it lands on the check, priced.
    expect(
      (await screen.findAllByText(price(32000), { exact: false })).length,
    ).toBeGreaterThan(0);
  });

  it("counts a second tap instead of stacking a second line", async () => {
    const { user } = renderTill(<TillPage />);
    await reachTheMenu(user);

    await user.click(dishTile(PLAIN_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));
    await user.click(dishTile(PLAIN_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(2));

    // ⚠️ A till is used by tapping — four coffees is the tile pressed four
    // times — and four identical rows is a check nobody can read back to a
    // guest, with no way to correct a miscount except removing rows one by one.
    const check = [...server.checks.values()][0]!;
    expect(check.lines).toHaveLength(1);
    expect(check.lines[0].qty).toBe(2);
    expect(
      (await screen.findAllByText(price(64000), { exact: false })).length,
    ).toBeGreaterThan(0);
  });

  it("keeps a different portion on its own line", async () => {
    const { user } = renderTill(<TillPage />);
    await reachTheMenu(user);

    // Two sizes of the same dish are different food, not two of one thing.
    await user.click(dishTile(OPTION_DISH));
    await user.click(await screen.findByRole("button", { name: /Katta/ }));
    await user.click(screen.getByRole("button", { name: t.till.add }));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));

    await user.click(dishTile(OPTION_DISH));
    await user.click(await screen.findByRole("button", { name: /Kichik/ }));
    await user.click(screen.getByRole("button", { name: t.till.add }));

    await waitFor(() => {
      const check = [...server.checks.values()][0]!;
      expect(check.lines).toHaveLength(2);
    });
  });

  it("changes a line's quantity from the check, before the kitchen has it", async () => {
    const { user } = renderTill(<TillPage />);
    await reachTheMenu(user);
    await user.click(dishTile(PLAIN_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));

    await user.click(
      await screen.findByRole("button", { name: `${PLAIN_DISH} +` }),
    );

    await waitFor(() => {
      const check = [...server.checks.values()][0]!;
      expect(check.lines[0].qty).toBe(2);
    });

    // ⚠️ Down to one, and no further: taking a line off is a different act with
    // a different record, and a stepper that voids at zero is how a till stops
    // being able to say where the food went.
    await user.click(
      screen.getByRole("button", { name: `${PLAIN_DISH} −` }),
    );
    await waitFor(() => {
      const check = [...server.checks.values()][0]!;
      expect(check.lines[0].qty).toBe(1);
    });
    expect(
      screen.getByRole("button", { name: `${PLAIN_DISH} −` }),
    ).toBeDisabled();
  });

  it("never offers a dish that is off the menu", async () => {
    const { user } = renderTill(<TillPage />);
    await reachTheMenu(user);

    expect(screen.queryByText("Norin")).not.toBeInTheDocument();
  });

  it("goes back to the room from the rail, not with the browser", async () => {
    const { user } = renderTill(<TillPage />);
    await reachTheMenu(user);

    // ⚠️ A monoblock runs fullscreen with no chrome: a waiter who cannot get
    // back to the floor opens a second check for the same table. The way back
    // is the navigation rail, which is on screen the whole time.
    await user.click(screen.getByRole("button", { name: t.till.tables }));

    await waitForFloor();
  });
});

describe("a dish that asks a question", () => {
  async function openTheDialog(
    user: Awaited<ReturnType<typeof renderTill>>["user"],
  ) {
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();
    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    await screen.findByText(PLAIN_DISH);
    await user.click(dishTile(OPTION_DISH));
    await screen.findByText(t.till.qty);
  }

  it("asks before adding, and refuses to add until it is answered", async () => {
    const { user } = renderTill(<TillPage />);
    await openTheDialog(user);

    // ⚠️ Without this dialog the tablet sent {menuItemId, qty}, the server
    // refused the unanswered required group, and no screen could answer it —
    // every dish with a portion size was unsellable from the till.
    const add = screen.getByRole("button", { name: t.till.add });
    expect(add).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /Katta/ }));
    expect(add).toBeEnabled();
    await user.click(add);

    await waitFor(() =>
      expect(server.calls.addLines).toEqual([
        {
          checkId: "chk-1",
          menuItemId: "m2",
          qty: 1,
          // ⚠️ The base (uz) names, never the translated labels: that is what
          // goes on the wire and what the server matches against.
          options: [{ name: "Hajm", choice: "Katta", priceDelta: 5000 }],
        },
      ]),
    );
  });

  it("prices the choice before the guest is told the total", async () => {
    const { user } = renderTill(<TillPage />);
    await openTheDialog(user);

    await user.click(screen.getByRole("button", { name: /Katta/ }));
    // 25 000 + 5 000 — a number that is on no tile, which is the reason the
    // dialog shows it at all.
    expect(await screen.findByText(price(30000))).toBeInTheDocument();
  });

  it("does not add twice when the second tap lands before the first answer", async () => {
    const { user } = renderTill(<TillPage />);
    await openTheDialog(user);

    await user.click(screen.getByRole("button", { name: /Katta/ }));
    const add = screen.getByRole("button", { name: t.till.add });
    await user.click(add);
    await user.click(add);

    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));
  });

  it("keeps a required group answered once it has been", async () => {
    const { user } = renderTill(<TillPage />);
    await openTheDialog(user);

    const katta = screen.getByRole("button", { name: /Katta/ });
    await user.click(katta);
    // ⚠️ Tapping the current choice again would clear an optional group; on a
    // required one it must not, or the cashier meets a button that just died.
    await user.click(katta);

    expect(screen.getByRole("button", { name: t.till.add })).toBeEnabled();
  });
});

describe("permissions", () => {
  it("says so plainly to somebody who may not use the till", async () => {
    server = installTillServer({ canWaiter: false, canCashier: false });
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);

    expect(await screen.findByText(t.till.noAccess)).toBeInTheDocument();
  });
});

describe("a till signed in with a staff login", () => {
  it("still sells when the branch has set no PINs", async () => {
    // The installation from before device binding: no device token, a staff
    // account, and nobody has been given a code.
    window.localStorage.clear();
    server = installTillServer({ pinsUsed: false });
    setSignedInStaff(tillStaff());

    renderTill(<TillPage />);

    // ⚠️ No pad — a restaurant that has set no codes must not be locked out by
    // an upgrade — and the room is drawn, which means the menu and the checks
    // were actually fetched.
    await waitForFloor();
    expect(screen.queryByText(t.till.pinTitle)).not.toBeInTheDocument();
  });
});
