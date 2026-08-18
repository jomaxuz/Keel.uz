/**
 * The floor screen, driven the way a waiter uses it: PIN, room, table, dishes.
 *
 * ⚠️ **Not the till with the payment hidden**, and the differences are what
 * these tests hold in place: opening a table goes straight to the menu (the
 * waiter is standing beside it about to be told what they want), the room shows
 * their own tables first, and the same option dialog is here — the question
 * "which size" is asked *at the table*, by the person holding this screen.
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

import FloorPage from "./page";

let server: TillServer;

beforeEach(() => {
  server = installTillServer();
  bindDevice();
});

async function reachTheFloor(
  user: Awaited<ReturnType<typeof renderTill>>["user"],
) {
  await screen.findByText(t.till.pinTitle);
  await unlock(user);
  await waitForFloor();
}

describe("getting in", () => {
  it("shows nothing but the pad until somebody names themselves", async () => {
    renderTill(<FloorPage />);

    expect(await screen.findByText(t.till.pinTitle)).toBeInTheDocument();
    expect(screen.queryByText(t.till.myTables)).not.toBeInTheDocument();
  });

  it("meets the same shift gate as the till", async () => {
    // ⚠️ The floor screen has no payment button, but it opens the check that
    // will be paid: an order fired before the shift is open is food cooked
    // against no count.
    server = installTillServer({ shiftOpen: false });
    const { user } = renderTill(<FloorPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);

    expect(await screen.findByText(t.till.shiftClosed)).toBeInTheDocument();

    await openShift(user, "100000");

    await waitForFloor();
  });
});

describe("the room", () => {
  it("asks for this waiter's tables first", async () => {
    const { user } = renderTill(<FloorPage />);
    await reachTheFloor(user);

    // ⚠️ Mine by default: a screen showing everybody's tables is a list to read
    // past. The flag is the server's, so the test reads what was actually asked
    // rather than which button looks pressed.
    expect(server.calls.checksMine.every(Boolean)).toBe(true);

    // ...and the whole floor is one tap away, for the evening somebody goes
    // home early and their tables would otherwise be stranded. ⚠️ Both states
    // are on screen at once (a segmented control, not a button that relabels
    // itself): a control whose label is the state you are *not* in is read
    // wrong by half the people who press it.
    await user.click(screen.getByRole("button", { name: t.till.allTables }));

    await waitFor(() => expect(server.calls.checksMine).toContain(false));
  });
});

describe("the header", () => {
  it("keeps the lock reachable by name now that it is only an icon", async () => {
    const { user } = renderTill(<FloorPage />);
    await reachTheFloor(user);

    await user.click(screen.getByRole("button", { name: t.till.lock }));

    expect(await screen.findByText(t.till.pinTitle)).toBeInTheDocument();
  });
});

describe("taking an order", () => {
  async function openTable(
    user: Awaited<ReturnType<typeof renderTill>>["user"],
  ) {
    await reachTheFloor(user);
    await user.click(tableTile("7"));
    await screen.findByText(PLAIN_DISH);
  }

  it("goes straight to the menu, not to an empty check", async () => {
    const { user } = renderTill(<FloorPage />);
    await openTable(user);

    // ⚠️ No dialog and no empty list on the way: the waiter is standing beside
    // the table about to be told what they want, and a screen that stops to
    // show an empty check first is a tap that buys nothing.
    expect(server.calls.openCheck).toEqual([{ tableId: "t1", guests: 0 }]);
    expect(screen.getByText(PLAIN_DISH)).toBeInTheDocument();
  });

  it("adds a dish and shows it on the order", async () => {
    const { user } = renderTill(<FloorPage />);
    await openTable(user);

    await user.click(dishTile(PLAIN_DISH));

    await waitFor(() =>
      expect(server.calls.addLines).toEqual([
        { checkId: "chk-1", menuItemId: "m1", qty: 1 },
      ]),
    );
    // ⚠️ No navigation in between: the order lives in a column that never
    // leaves, so the dish and its price are on screen the moment they are
    // added. That is the whole point of the two-pane layout — a waiter used to
    // have to leave the menu to find out what the table now owed.
    expect(
      (await screen.findAllByText(price(32000), { exact: false })).length,
    ).toBeGreaterThan(0);
  });

  it("asks which size at the table, exactly as the till does", async () => {
    const { user } = renderTill(<FloorPage />);
    await openTable(user);

    await user.click(dishTile(OPTION_DISH));

    const add = await screen.findByRole("button", { name: t.till.add });
    expect(add).toBeDisabled();
    await user.click(screen.getByRole("button", { name: /Katta/ }));
    await user.click(add);

    await waitFor(() =>
      expect(server.calls.addLines).toEqual([
        {
          checkId: "chk-1",
          menuItemId: "m2",
          qty: 1,
          options: [{ name: "Hajm", choice: "Katta", priceDelta: 5000 }],
        },
      ]),
    );
  });
});

describe("a tablet signed in with a staff login", () => {
  it("still works when the branch has set no PINs", async () => {
    // ⚠️ Same trap as the till's, and it was here too: the room and the menu
    // were fetched only once a `person` existed, so on a branch that has set no
    // codes the waiter met an empty floor and a settings page that was already
    // correct.
    window.localStorage.clear();
    server = installTillServer({ pinsUsed: false });
    setSignedInStaff(tillStaff());

    renderTill(<FloorPage />);

    await waitForFloor();
    expect(screen.queryByText(t.till.pinTitle)).not.toBeInTheDocument();
  });
});

describe("permissions", () => {
  it("keeps out somebody who is neither waiter nor cashier", async () => {
    server = installTillServer({ canWaiter: false, canCashier: false });
    const { user } = renderTill(<FloorPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);

    expect(await screen.findByText(t.till.noAccess)).toBeInTheDocument();
  });
});
