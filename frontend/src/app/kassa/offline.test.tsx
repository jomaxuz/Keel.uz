/**
 * The minutes the server is not there.
 *
 * ⚠️ **The moment a till must never lose is the one where money changes
 * hands.** Everything else can wait for the network: a dish added, a table
 * moved, a bill printed. A payment cannot — the cash is in the drawer, the
 * guest has gone, and a sale that failed to close leaves a check open at an
 * empty table, worth its whole total in a shift count that will not balance.
 */

import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { adminUz as t } from "@/lib/i18n/admin";
import { ApiError } from "@/lib/api";
import { localChecks } from "@/lib/offline/checks";
import {
  drainLocalChecks,
  drainSales,
  isNetworkError,
  pendingSales,
} from "@/lib/offline/sales";
import { bindDevice, renderTill } from "@/test/render";
import {
  dishTile,
  openShift,
  tableTile,
  unlock,
  waitForFloor,
} from "@/test/tillFlow";
import {
  installTillServer,
  PLAIN_DISH,
  type TillServer,
} from "@/test/tillServer";

import TillPage from "./page";

let server: TillServer;

beforeEach(() => {
  server = installTillServer();
  bindDevice();
});

describe("telling a lost connection from a refusal", () => {
  it("knows which is which", () => {
    // ⚠️ The distinction the whole queue rests on. "The kitchen already has
    // this line" is an answer, and replaying it forever would never make it
    // true. "fetch failed" is not an answer at all, and the sale is still owed.
    expect(isNetworkError(new TypeError("Failed to fetch"))).toBe(true);
    expect(isNetworkError(new ApiError(409, "chek yopilgan"))).toBe(false);
    expect(isNetworkError(new ApiError(500, "server"))).toBe(false);
  });
});

describe("a table opened while the server is unreachable", () => {
  it("is opened on the device, sold from, and handed over when the server is back", async () => {
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    // The wifi drops before anybody sits down.
    const opened = vi.spyOn(server.api, "tillOpenCheck");
    opened.mockRejectedValue(new TypeError("Failed to fetch"));

    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));

    // ⚠️ **The table opens anyway.** The guests are sitting down; a till that
    // refuses over a wifi drop is a till the restaurant keeps a paper pad
    // beside — and the paper never reaches the reports.
    await screen.findByText(PLAIN_DISH);
    // ⚠️ Said **once**, on the check itself. The kitchen screen cannot see this
    // order, so somebody has to walk in and say it — and the warning lives with
    // the check rather than in a dismissible bar at the top, because it stays
    // true until the connection comes back.
    expect(screen.getAllByText(t.till.offlineKitchen).length).toBe(1);

    // It sells like any other check.
    await user.click(dishTile(PLAIN_DISH));
    await waitFor(async () => {
      const checks = await localChecks();
      expect(checks).toHaveLength(1);
      expect(checks[0].lines[0].qty).toBe(1);
    });
    // ⚠️ Nothing was asked of the server: it has never heard of this table.
    expect(server.calls.addLines).toHaveLength(0);

    // Paid, and now owed to the server.
    await user.click(screen.getByRole("button", { name: t.till.pay }));
    await user.click(
      await screen.findByRole("button", { name: t.till.confirmPay }),
    );
    await waitFor(async () => {
      const paid = (await localChecks()).filter((c) => c.paidAt);
      expect(paid).toHaveLength(1);
    });

    // The connection returns.
    opened.mockRestore();
    await drainLocalChecks();

    // ⚠️ Handed over as one finished sale, with the id the till minted — which
    // is what makes a resend the same dinner rather than a second one.
    expect(server.calls.sync).toHaveLength(1);
    expect(server.calls.sync[0].clientId).toBeTruthy();
    expect(server.calls.sync[0].lines).toHaveLength(1);
    expect(await localChecks()).toHaveLength(0);
  });
});

describe("a payment taken while the server is unreachable", () => {
  async function sell(user: Awaited<ReturnType<typeof renderTill>>["user"]) {
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();
    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    await screen.findByText(PLAIN_DISH);
    await user.click(dishTile(PLAIN_DISH));
    await waitFor(() => expect(server.calls.addLines).toHaveLength(1));
  }

  it("is kept on the device and sent when the server comes back", async () => {
    const { user } = renderTill(<TillPage />);
    await sell(user);

    // The wifi drops between the guest handing over the money and the till
    // telling anybody about it.
    // ⚠️ Kept failing rather than failing once: the till drains the queue the
    // moment a request succeeds, so a single failure would be swept up by the
    // next poll and the test would be asserting a race.
    const closed = vi.spyOn(server.api, "tillClose");
    closed.mockRejectedValue(new TypeError("Failed to fetch"));

    await user.click(screen.getByRole("button", { name: t.till.pay }));
    await user.click(
      await screen.findByRole("button", { name: t.till.confirmPay }),
    );

    // ⚠️ The screen says the sale is done, because it is: the drawer is shut
    // and the guest has gone. What is owed is an answer to the server, and that
    // is the queue's problem, not the cashier's.
    await waitFor(async () => {
      expect((await pendingSales()).length).toBe(1);
    });
    expect(await screen.findByText(t.till.offlineSaved)).toBeInTheDocument();

    // The connection returns.
    closed.mockRestore();
    await drainSales();

    await waitFor(async () => {
      expect((await pendingSales()).length).toBe(0);
    });
  });

  it("stops owing a sale the server has already taken", async () => {
    // ⚠️ The ordinary way this queue fills is a reply that never arrived: the
    // server took the money and the answer was lost on the way back. Replaying
    // it must clear the queue rather than leave a paid sale in it forever,
    // frightening whoever reads the badge at the end of the shift.
    const { user } = renderTill(<TillPage />);
    await sell(user);

    const closed = vi.spyOn(server.api, "tillClose");
    closed.mockRejectedValue(new TypeError("Failed to fetch"));
    await user.click(screen.getByRole("button", { name: t.till.pay }));
    await user.click(
      await screen.findByRole("button", { name: t.till.confirmPay }),
    );
    await waitFor(async () => {
      expect((await pendingSales()).length).toBe(1);
    });

    closed.mockRejectedValue(new ApiError(409, "chek yopilgan"));
    await drainSales();
    expect((await pendingSales()).length).toBe(0);
    closed.mockRestore();
  });
});

describe("what a table is charged does not depend on the wifi", () => {
  it("charges the room's service on a check opened offline, and hands the rate over", async () => {
    // ⚠️ The gap this closes was mine: the offline path had no rate, so the
    // same table paid two different totals depending on whether the connection
    // happened to be up — and the guest who paid less never finds out.
    server = installTillServer({ servicePercent: 10 });
    const { user } = renderTill(<TillPage />);
    await screen.findByText(t.till.pinTitle);
    await unlock(user);
    await waitForFloor();

    const opened = vi.spyOn(server.api, "tillOpenCheck");
    opened.mockRejectedValue(new TypeError("Failed to fetch"));

    await user.click(tableTile("7"));
    await user.click(await screen.findByRole("button", { name: t.till.open }));
    await screen.findByText(PLAIN_DISH);
    await user.click(dishTile(PLAIN_DISH));

    await waitFor(async () => {
      const [check] = await localChecks();
      expect(check?.lines).toHaveLength(1);
      // 32 000 + 10% = 35 200, computed the same way the server does it.
      expect(check.service).toBe(3200);
      expect(check.total).toBe(35200);
    });

    await user.click(screen.getByRole("button", { name: t.till.pay }));
    await user.click(
      await screen.findByRole("button", { name: t.till.confirmPay }),
    );
    await waitFor(async () => {
      const paid = (await localChecks()).filter((c) => c.paidAt);
      expect(paid).toHaveLength(1);
    });

    opened.mockRestore();
    await drainLocalChecks();

    // ⚠️ The **rate** goes over the wire, not the amount: the server recomputes
    // it, so a sale taken offline cannot arrive with parts that do not add up.
    await waitFor(() => expect(server.calls.sync).toHaveLength(1));
    expect(server.calls.sync[0].servicePercent).toBe(10);
  });
});
