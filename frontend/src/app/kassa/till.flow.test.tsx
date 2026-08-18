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

import { screen, waitFor, within } from "@testing-library/react";
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
  TEA_DISH,
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

  it("draws the takeaway numbers on the counter, not in the room", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    // ⚠️ **A zone marked "list" in the settings is the counter.** Its numbers
    // are orders waiting to be called, not tables anybody sits at: seats and
    // coordinates were never filled in, so on the floor plan they would all
    // pile up in the corner at 0,0. They belong beside the counter — and there
    // whichever view of the room is showing, because "one coffee to take away"
    // arrives while you are looking at something else.
    expect(
      screen.getByRole("button", { name: /^101\s*·/ }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: t.till.gridView }));
    expect(
      screen.getByRole("button", { name: /^101\s*·/ }),
    ).toBeInTheDocument();
    // ...and the hall still shows its own tables.
    expect(tableTile("7")).toBeTruthy();
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
  async function reachTheMenu(
    user: Awaited<ReturnType<typeof renderTill>>["user"],
  ) {
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
    await user.click(screen.getByRole("button", { name: `${PLAIN_DISH} −` }));
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

describe("splitting a bill and sending it in courses", () => {
  async function openTable(
    user: Awaited<ReturnType<typeof renderTill>>["user"],
  ) {
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();
    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    await screen.findByText(PLAIN_DISH);
  }

  it("puts the next dish on the guest whose tab is open", async () => {
    const { user } = renderTill(<TillPage />);
    await openTable(user);

    // The table was opened for two, so the tabs are already there.
    await user.click(
      screen.getByRole("button", { name: `${t.till.guestTab} 2` }),
    );
    await user.click(dishTile(PLAIN_DISH));

    // ⚠️ The tab is where the dish goes — that is the whole mechanism, and it
    // is why splitting happens while the order is taken rather than as a
    // sorting exercise once the guests are asking for their bills.
    await waitFor(() =>
      expect(server.calls.addLines).toEqual([
        { checkId: "chk-1", menuItemId: "m1", qty: 1, guest: 2 },
      ]),
    );
  });

  it("keeps two guests' identical dishes on separate lines", async () => {
    const { user } = renderTill(<TillPage />);
    await openTable(user);

    await user.click(
      screen.getByRole("button", { name: `${t.till.guestTab} 1` }),
    );
    await user.click(dishTile(PLAIN_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));
    await user.click(
      screen.getByRole("button", { name: `${t.till.guestTab} 2` }),
    );
    await user.click(dishTile(PLAIN_DISH));

    // ⚠️ Merging them would hand one guest a bill for both — the case the line
    // model was always worried about.
    await waitFor(() => {
      const check = [...server.checks.values()][0]!;
      expect(check.lines).toHaveLength(2);
    });
  });

  it("sends one course without sending the rest", async () => {
    const { user } = renderTill(<TillPage />);
    await openTable(user);

    // Starters on course one, dessert on course two.
    await user.click(screen.getByRole("button", { name: t.till.courseOf(1) }));
    await user.click(dishTile(PLAIN_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));
    await user.click(screen.getByRole("button", { name: t.till.courseOf(2) }));
    // A different category, so the two courses are two different dishes.
    await user.click(screen.getByRole("button", { name: /Ichimliklar/ }));
    await user.click(dishTile("Choy"));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(2));

    await user.click(
      await screen.findByRole("button", { name: t.till.fireCourse(1) }),
    );

    // ⚠️ Firing everything at once is what a kitchen cannot undo, because the
    // food is already being made.
    await waitFor(() => {
      const check = [...server.checks.values()][0]!;
      expect(check.lines.filter((l) => l.fired)).toHaveLength(1);
      expect(check.lines.find((l) => l.fired)!.course).toBe(1);
    });
  });
});

describe("the bill", () => {
  it("prints, and marks the table as having asked to pay", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();
    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    await screen.findByText(PLAIN_DISH);
    await user.click(dishTile(PLAIN_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));

    await user.click(screen.getByRole("button", { name: t.till.precheck }));

    // ⚠️ **Printing the bill is an event on the check, not just paper.** A
    // table that has asked to pay is waiting for a person with a card machine,
    // and until this was recorded the only way to know was to have been the
    // one who printed it. The floor draws it as its own state.
    await waitFor(() => {
      const check = [...server.checks.values()][0]!;
      expect(check.precheckAt).toBeTruthy();
    });
    await user.click(screen.getByRole("button", { name: t.till.tables }));
    expect(
      await screen.findByRole("button", {
        name: new RegExp(`^7\\s*·\\s*${t.till.billed}`),
      }),
    ).toBeInTheDocument();
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

describe("dividing a bill at the table", () => {
  it("splits the ticked dishes onto a new check and keeps the waiter where they were", async () => {
    // ⚠️ The request every dining room gets several times an evening, and the
    // one the till could not answer: before this, a table paying separately had
    // to be opened as two checks *before anybody ordered*.
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    // Two different dishes: splitting **everything** off is refused by the
    // server (it leaves an empty check nobody can pay), so a bill worth
    // dividing has at least two lines — which is also the only shape a real
    // table ever asks about.
    await user.click(dishTile(PLAIN_DISH));
    // The tea is in another category, so the tab has to be opened first —
    // exactly what the waiter does.
    await user.click(screen.getByRole("button", { name: "Ichimliklar" }));
    await user.click(dishTile(TEA_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(2));

    await user.click(screen.getByRole("button", { name: t.till.moveLines }));
    // Inside the dialog: the menu behind it lists the same dish, and a test
    // that ticks the wrong one would pass while the screen did nothing.
    const dialog = (
      await screen.findByRole("heading", { name: t.till.moveLines })
    ).closest("div") as HTMLElement;
    const sheet = dialog.parentElement as HTMLElement;
    await user.click(within(sheet).getByText(TEA_DISH));
    await user.click(
      within(sheet).getByRole("button", { name: t.till.splitNew }),
    );
    // ⚠️ The button says what it will do. "Move" on a screen that is about to
    // create a second bill is the label answering the wrong question.
    await user.click(within(sheet).getByRole("button", { name: t.till.split }));

    await waitFor(() => expect(server.calls.split).toHaveLength(1));
    expect(server.calls.split[0].checkId).toBe("chk-1");

    // ⚠️ The screen stays on the check the waiter was standing at: the tea
    // moved onto the new bill, the lag'mon is still on this one. Following the
    // new half would lose their place in the meal.
    // ⚠️ Counted rather than queried: the tea is still on the menu behind the
    // check — it is a dish, not only a line — so "gone" means gone from the
    // order column, which is one occurrence fewer.
    await waitFor(() => expect(screen.getAllByText(TEA_DISH)).toHaveLength(1));
  });

  it("offers a new check even when it is the only one open", async () => {
    // ⚠️ The old rule was "two checks or the button is dead", which is right
    // for moving food and exactly wrong for splitting: the first table of the
    // evening is the one most likely to ask.
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    await user.click(dishTile(PLAIN_DISH));

    expect(
      screen.getByRole("button", { name: t.till.moveLines }),
    ).not.toBeDisabled();
  });
});
