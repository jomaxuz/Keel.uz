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
/** One drain at a time, per queue.
 *
 * ⚠️ **Two drains running together send the same sale twice.** Both read the
 * queue before either has removed anything from it, so both post it — and the
 * two callers are ordinary: the reconnect flushes as soon as a request
 * succeeds, and the thirty-second fallback is still ticking. The server treats
 * a repeat as success (it is keyed by the id the till minted), so nothing is
 * charged twice — but it is a second sale on the wire and a second line in
 * somebody's log, and the till has no reason to make it.
 *
 * A caller that arrives mid-drain joins the one already running rather than
 * starting a second: what it wants is "everything waiting is sent", and that is
 * what the in-flight one is doing.
 */
function once(
  slot: { p: Promise<number> | null },
  run: () => Promise<number>,
): Promise<number> {
  if (slot.p) return slot.p;
  const p = run().finally(() => {
    slot.p = null;
  });
  slot.p = p;
  return p;
}

const salesDrain: { p: Promise<number> | null } = { p: null };
const checksDrain: { p: Promise<number> | null } = { p: null };

export function drainSales(): Promise<number> {
  return once(salesDrain, drainSalesOnce);
}

async function drainSalesOnce(): Promise<number> {
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

/**
 * Hand over the sales this till rang up entirely on its own.
 *
 * ⚠️ **Whole sales, not the steps that made them.** A check opened offline was
 * never on the server, so there is nothing to replay against — it goes as one
 * finished sale with the times it actually happened at. Replaying the taps
 * instead would need the server to accept an open check from a device, which is
 * a second way to own a table and the first thing to go wrong when two tills
 * think they own the same one.
 *
 * Returns how many are still waiting.
 */
export function drainLocalChecks(): Promise<number> {
  return once(checksDrain, drainLocalChecksOnce);
}

async function drainLocalChecksOnce(): Promise<number> {
  const { openLocalChecks, localChecks, forgetLocalCheck, syncPayload } =
    await import("./checks");
  const paid = (await localChecks()).filter((c) => c.paidAt);
  if (paid.length === 0) return (await openLocalChecks()).length;

  try {
    // ⚠️ In batches: a till that has been offline for an evening must not send
    // its whole night in one request that times out halfway and is retried
    // whole.
    const batch = paid.slice(0, 20);
    const res = await api.tillSyncChecks(batch.map(syncPayload));
    for (const r of res.results) {
      // ⚠️ A duplicate is a success: it means the previous attempt did arrive
      // and only the answer was lost. Keeping it would resend the same dinner
      // every thirty seconds for the rest of the evening.
      if (!r.error) await forgetLocalCheck(r.clientId);
      // A sale the server refuses outright stays here, with its reason, where
      // somebody can see it — an evening's takings must not vanish because one
      // check was malformed.
    }
  } catch {
    // Still no server, or it answered badly. Nothing is lost.
  }
  return (await localChecks()).filter((c) => c.paidAt).length;
}
