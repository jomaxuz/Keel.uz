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
import { drainSales, isNetworkError, pendingSales } from "@/lib/offline/sales";
import { bindDevice, renderTill } from "@/test/render";
import { dishTile, openShift, tableTile, unlock, waitForFloor } from "@/test/tillFlow";
import { installTillServer, PLAIN_DISH, type TillServer } from "@/test/tillServer";

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
    const closed = vi.spyOn(server.api, "tillClose");
    closed.mockRejectedValueOnce(new TypeError("Failed to fetch"));

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
    closed.mockRejectedValueOnce(new TypeError("Failed to fetch"));
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
