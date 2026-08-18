"use client";

// Payments the server has not confirmed.
//
// ⚠️ **The moment a till must never lose is the one where money changes
// hands.** Everything else on this screen can wait for the network: a dish
// added, a table moved, a bill printed. A payment cannot — the cash is in the
// drawer, the guest has gone, and a sale that failed to close is a check still
// open at a table nobody is sitting at, worth its whole total in a shift count
// that will not balance.
//
// ⚠️ **The retry is the close, not a new sale.** The check exists on the server
// already; it was created when the table was opened, while the network was
// fine. Sending it as a fresh offline sale would be the same dinner twice. So
// the queue holds the *intent* — this check, this method, this discount — and
// replays it until the server says it is closed.

import { api, ApiError } from "@/lib/api";
import type { TillPaymentMethod } from "@/lib/types";

import { all, available, PENDING, put, remove } from "./store";

export interface PendingSale {
  /** The till's own id for this payment, and the key it is stored under. */
  clientId: string;
  checkId: string;
  /** For the screen: which table, and how much, without opening the record. */
  label: string;
  total: number;
  method: TillPaymentMethod;
  discount?: number;
  discountReason?: string;
  at: number;
  /** How many times this has been tried, so a sale that can never be accepted
   *  stops being retried in a loop and starts being visible instead. */
  tries: number;
  /** The server's own words, when it refused. */
  error?: string;
}

/** How many times a payment is replayed before it needs a person.
 *
 *  ⚠️ Generous: the ordinary case is a wifi drop that lasts a minute, and a
 *  queue that gives up early turns a network blip into a shift that does not
 *  balance. What it is guarding against is a sale the server will never accept,
 *  which would otherwise be retried until the browser is closed. */
const MAX_TRIES = 20;

export function newClientId(): string {
  // crypto.randomUUID is not in every browser a monoblock might run.
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

export function canQueue(): boolean {
  return available();
}

export async function queueSale(sale: PendingSale): Promise<boolean> {
  return put(PENDING, sale);
}

export async function pendingSales(): Promise<PendingSale[]> {
  const rows = await all<PendingSale>(PENDING);
  return rows.sort((a, b) => a.at - b.at);
}

/** Whether a failure was the network rather than the server's judgement.
 *
 *  ⚠️ **The distinction the whole queue rests on.** "The kitchen already has
 *  this line" is an answer — replaying it forever would never make it true.
 *  "fetch failed" is not an answer at all, and the sale is still owed. */
export function isNetworkError(err: unknown): boolean {
  return !(err instanceof ApiError);
}

/**
 * Send everything waiting. Returns how many are still queued afterwards.
 *
 * ⚠️ **Oldest first, and one at a time.** They are payments: a shift count reads
 * them in order, and firing twenty requests at a server that has just come back
 * is how the reconnect itself becomes the outage.
 */
export async function drainSales(): Promise<number> {
  const queue = await pendingSales();
  for (const sale of queue) {
    try {
      await api.tillClose(sale.checkId, {
        paymentMethod: sale.method,
        discount: sale.discount,
        discountReason: sale.discountReason,
      });
      await remove(PENDING, sale.clientId);
    } catch (err) {
      if (isNetworkError(err)) {
        // Still no server. Nothing is lost and nothing is counted twice; the
        // next attempt is the next time the browser thinks it is online.
        break;
      }
      const e = err as ApiError;
      // ⚠️ **"Already closed" is success, not failure.** The ordinary way this
      // queue fills is a reply that never arrived — the server took the money
      // and the answer was lost on the way back. Replaying it must not leave a
      // paid sale sitting in the queue forever, frightening whoever reads the
      // badge.
      if (e.status === 409 || e.status === 404) {
        await remove(PENDING, sale.clientId);
        continue;
      }
      const tries = sale.tries + 1;
      await put(PENDING, {
        ...sale,
        tries,
        error: e.message,
      });
      if (tries >= MAX_TRIES) {
        // Kept, not dropped: a payment nobody can explain is exactly the thing
        // that must still be on screen at the end of the shift.
        continue;
      }
    }
  }
  return (await pendingSales()).length;
}
