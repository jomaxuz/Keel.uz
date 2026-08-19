"use client";

// Checks opened while the server was not there.
//
// ⚠️ **The restaurant does not stop because we do.** During a wifi outage the
// kitchen still cooks, the drawer still opens and the fiscal register — which
// is on the restaurant's own network — still registers a sale. The only thing
// that stops is our end of a wire, and a till that refuses to open a table over
// it is a till the restaurant keeps a paper pad next to.
//
// ⚠️ **A local check is not a server check, and the screen says so.** Two facts
// follow from where it lives, and both reach people:
//
//   - **The kitchen screen cannot see it.** It is on this device. Firing a
//     course marks it here and tells nobody; the waiter has to walk. That is
//     worse than the online path and better than not selling.
//   - **The stop list cannot be checked.** Two tills offline can sell the last
//     portion twice. The plan accepts this on purpose (docs/pos-reja.md §6): a
//     till that will not sell is an unsellable dish, and a restaurant
//     apologising for the last portion is an ordinary evening.

import type {
  Check,
  CheckLine,
  MenuItem,
  OrderItemOption,
  TillPaymentMethod,
} from "@/lib/types";

import { all, LOCAL_CHECKS, put, remove } from "./store";
import { newClientId } from "./sales";

/** The store checks live in until the server has them. */
const CHECKS = LOCAL_CHECKS;

export interface LocalCheck extends Check {
  /** The id the server will know this sale by. ⚠️ Minted here, before anybody
   *  has seen it, which is what makes the sync idempotent — a resend is the
   *  same dinner rather than a second one. */
  clientId: string;
  /** Always true. The screens read it to say "this one is only here yet". */
  local: true;
  /** Set when the money was taken; the sale is then owed to the server. */
  paidAt?: string;
  paymentMethod?: TillPaymentMethod;
  discount?: number;
  discountReason?: string;
}

/** The room's percentage, rounded half-up to the som — the browser's copy of
 *  the server's `serviceOn`.
 *
 *  ⚠️ Duplicated rather than imported because there is nothing to import from:
 *  the rule lives in Go. The test that seals it compares both against the same
 *  numbers, which is the only thing that keeps two implementations honest. */
export function serviceOn(payable: number, percent: number): number {
  if (payable <= 0 || percent <= 0) return 0;
  return Math.floor((payable * percent + 50) / 100);
}

/** Marks a locally-minted number so nobody mistakes it for a printed one.
 *
 *  ⚠️ Two tills offline at once would otherwise mint the same number, and the
 *  server keeps what it is sent. The prefix plus randomness makes a collision
 *  a curiosity rather than a bill that overwrites another. */
function localNumber(): string {
  const rnd = Math.random().toString(36).slice(2, 6).toUpperCase();
  return `OFF-${rnd}`;
}

export async function localChecks(): Promise<LocalCheck[]> {
  const rows = await all<LocalCheck>(CHECKS);
  return rows.sort((a, b) => a.openedAt.localeCompare(b.openedAt));
}

/** Only the ones still on the floor — a paid one belongs to the sync queue. */
export async function openLocalChecks(): Promise<LocalCheck[]> {
  return (await localChecks()).filter((c) => !c.paidAt);
}

export async function openLocalCheck(
  tableId: string,
  tableNumber: string,
  guests: number,
  serverName: string,
  /** The room's service rate, as the branch last told this device. ⚠️ Copied
   *  onto the check when the table sits down, exactly as the server does it —
   *  a rate read at payment would charge a table that has been eating for an
   *  hour at whatever the setting says now. */
  servicePercent = 0,
): Promise<LocalCheck | null> {
  const now = new Date().toISOString();
  const check: LocalCheck = {
    clientId: newClientId(),
    local: true,
    // ⚠️ A local id that cannot collide with a server ObjectId, so a screen
    // holding both never sends one to the wrong place.
    id: `local:${newClientId()}`,
    number: localNumber(),
    status: "pending",
    tableId: tableId || undefined,
    tableNumber: tableNumber || undefined,
    guests,
    serverName,
    servicePercent: tableId ? servicePercent : 0,
    openedAt: now,
    openMin: 0,
    lines: [],
    subtotal: 0,
    unfired: 0,
    total: 0,
  };
  return (await put(CHECKS, check)) ? check : null;
}

export function isLocal(check: Check | null | undefined): boolean {
  return !!check && check.id.startsWith("local:");
}

async function save(check: LocalCheck): Promise<LocalCheck> {
  retotal(check);
  await put(CHECKS, check);
  return { ...check };
}

/** The money, recomputed from the lines every time.
 *
 *  ⚠️ Never nudged by each edit: a running total that drifts is invisible until
 *  a guest adds it up, and the same rule the server follows (applyCheckTotals)
 *  has to hold here or the two disagree at sync. */
function retotal(check: LocalCheck): void {
  check.subtotal = check.lines
    .filter((l) => !l.void)
    .reduce((sum, l) => sum + l.sum, 0);
  // ⚠️ **The same arithmetic as the server's, on purpose and in the same
  // shape** (`serviceOn`, half-up, after the discount). A till that charged a
  // different amount during an outage than it would have a minute earlier is a
  // guest paying two prices for one dinner depending on the wifi — and the one
  // who paid less never finds out.
  check.service = serviceOn(check.subtotal, check.servicePercent ?? 0);
  check.total = check.subtotal + check.service;
  check.unfired = check.lines.filter((l) => !l.void && !l.fired).length;
}

export async function addLocalLine(
  check: LocalCheck,
  item: MenuItem,
  qty: number,
  options: OrderItemOption[] | undefined,
  guest: number,
  course: number,
): Promise<LocalCheck> {
  // ⚠️ Priced from the menu this device already loaded. It is the same menu the
  // server priced from a minute ago, and it is the price the guest is being
  // told — which is what the receipt in their pocket will say.
  const unit =
    item.price + (options ?? []).reduce((s, o) => s + (o.priceDelta || 0), 0);

  // The same merge rule the server applies, or a check built offline would read
  // differently from one built online — four taps, four rows.
  const same = check.lines.find(
    (l) =>
      !l.void &&
      !l.fired &&
      l.menuItemId === item.id &&
      (l.guest ?? 0) === guest &&
      (l.course ?? 0) === course &&
      sameOptions(l.options, options),
  );
  if (same) {
    same.qty += qty;
    same.sum = same.price * same.qty;
  } else {
    const line: CheckLine & { firedAt?: string } = {
      lineId: newClientId().slice(0, 12),
      menuItemId: item.id,
      name: item.name,
      price: unit,
      qty,
      sum: unit * qty,
      fired: false,
      ...(options?.length ? { options } : {}),
      ...(guest ? { guest } : {}),
      ...(course ? { course } : {}),
    };
    check.lines.push(line);
  }
  return save(check);
}

export async function setLocalQty(
  check: LocalCheck,
  lineId: string,
  qty: number,
): Promise<LocalCheck> {
  const line = check.lines.find((l) => l.lineId === lineId);
  if (line && !line.fired && qty >= 1 && qty <= 99) {
    line.qty = qty;
    line.sum = line.price * qty;
  }
  return save(check);
}

/** Take a line off. ⚠️ Only before it is fired: a line the kitchen has been
 *  told about needs a reason and a cashier, and neither of those can be
 *  recorded properly here — so offline, that is a job for when the server is
 *  back. */
export async function removeLocalLine(
  check: LocalCheck,
  lineId: string,
): Promise<LocalCheck> {
  check.lines = check.lines.filter((l) => l.lineId !== lineId || l.fired);
  return save(check);
}

/** Mark what has been told to the kitchen.
 *
 *  ⚠️ **It tells the kitchen nothing.** There is no ticket and no kitchen
 *  screen: this records that the waiter has walked. Saying so on the screen is
 *  the whole honesty of the offline mode. */
export async function fireLocal(check: LocalCheck): Promise<LocalCheck> {
  const now = new Date().toISOString();
  for (const line of check.lines) {
    if (!line.void && !line.fired) {
      line.fired = true;
      (line as CheckLine & { firedAt?: string }).firedAt = now;
    }
  }
  return save(check);
}

export async function payLocal(
  check: LocalCheck,
  method: TillPaymentMethod,
  discount: number,
  discountReason: string,
): Promise<void> {
  check.paidAt = new Date().toISOString();
  check.paymentMethod = method;
  check.discount = discount;
  check.discountReason = discountReason;
  await put(CHECKS, check);
}

export async function forgetLocalCheck(clientId: string): Promise<void> {
  await remove(CHECKS, clientId);
}

/** What the server is sent for one finished sale. */
export function syncPayload(check: LocalCheck) {
  return {
    clientId: check.clientId,
    number: check.number,
    openedAt: check.openedAt,
    closedAt: check.paidAt,
    tableId: check.tableId,
    tableNumber: check.tableNumber,
    guests: check.guests,
    serverName: check.serverName,
    paymentMethod: check.paymentMethod,
    discount: check.discount,
    discountReason: check.discountReason,
    // ⚠️ The rate, not the amount. The server recomputes the som from it with
    // the same helper the online path uses — the till is trusted about what it
    // charged, the server about the arithmetic, so an outage cannot produce a
    // sale whose parts do not add up.
    servicePercent: check.servicePercent,
    lines: check.lines
      .filter((l) => !l.void)
      .map((l) => ({
        menuItemId: l.menuItemId ?? "",
        name: l.name,
        price: l.price,
        qty: l.qty,
        options: l.options,
        comment: l.comment,
        guest: l.guest,
        course: l.course,
        firedAt: (l as CheckLine & { firedAt?: string }).firedAt,
      })),
  };
}

function sameOptions(a?: OrderItemOption[], b?: OrderItemOption[]): boolean {
  const x = a ?? [];
  const y = b ?? [];
  if (x.length !== y.length) return false;
  const left = [...y];
  return x.every((want) => {
    const i = left.findIndex(
      (got) => got.name === want.name && got.choice === want.choice,
    );
    if (i < 0) return false;
    left.splice(i, 1);
    return true;
  });
}
