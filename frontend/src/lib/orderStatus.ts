import type { CourierStatus, OrderStatus, ReservationStatus } from "./types";

export const ORDER_STATUSES: OrderStatus[] = [
  "pending",
  "confirmed",
  "preparing",
  "on_the_way",
  "delivered",
  "cancelled",
];

export const STATUS_LABEL: Record<OrderStatus, string> = {
  pending: "Yangi",
  confirmed: "Tasdiqlandi",
  preparing: "Tayyorlanmoqda",
  on_the_way: "Yo'lda",
  delivered: "Yetkazildi",
  cancelled: "Bekor qilindi",
};

// Status colour, in one place, used by both the badge and the row it sits in.
//
// The hues carry meaning rather than marking positions in a pipeline:
// **green is settled, amber is in hand, red is a problem, brand is new.** A
// manager scanning the board is asking "is anything wrong, and is anything
// waiting for me" — not "which of six stages is this".
//
// `delivered` is deliberately colourless. A finished order has nothing left to
// do, and a screen where every row shouts is a screen where none of them do.

// Badge classes. Alpha rather than the flat `-100` shades so they hold up on a
// dark surface as well as a light one.
export const STATUS_BADGE: Record<OrderStatus, string> = {
  pending: "bg-brand/15 text-brand-dark dark:text-brand",
  confirmed: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  preparing: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  on_the_way: "bg-violet-500/15 text-violet-700 dark:text-violet-300",
  delivered: "bg-ink/10 text-ink-muted",
  cancelled: "bg-rose-500/15 text-rose-700 dark:text-rose-300",
};

// The tint of the whole row, and its border.
//
// Around 12%, with a border at roughly half strength again. Light enough that
// the text reads through it and the badge still carries the detail, strong
// enough to tell the rows apart across a room — which is where a kitchen
// screen is usually read from.
//
// It has to stay a wash rather than a block: at full strength every row shouts,
// the badges stop meaning anything, and a manager scanning for the one problem
// finds six colours competing instead.
export const STATUS_ROW: Record<OrderStatus, string> = {
  pending: "border-brand/55 bg-brand/[0.10]",
  confirmed: "border-emerald-500/45 bg-emerald-500/[0.12]",
  preparing: "border-amber-500/50 bg-amber-500/[0.13]",
  on_the_way: "border-violet-500/45 bg-violet-500/[0.12]",
  // Done. Nothing to act on, so it recedes to the ordinary card surface.
  delivered: "border-line bg-surface",
  cancelled: "border-rose-500/45 bg-rose-500/[0.12]",
};

// ---- Table bookings ----
//
// Same vocabulary, so an operator who has learned the orders board already
// knows this one: brand is new and wants an answer, green is settled, amber is
// happening right now, red is off, and a finished booking recedes.
//
// Kept here rather than in the bookings screen so the two palettes cannot
// drift apart — the whole point is that a colour means one thing everywhere.
export const RESERVATION_BADGE: Record<ReservationStatus, string> = {
  pending: "bg-brand/15 text-brand-dark dark:text-brand",
  confirmed: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  seated: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  done: "bg-ink/10 text-ink-soft",
  cancelled: "bg-rose-500/15 text-rose-700 dark:text-rose-300",
};

export const RESERVATION_ROW: Record<ReservationStatus, string> = {
  pending: "border-brand/55 bg-brand/[0.10]",
  confirmed: "border-emerald-500/45 bg-emerald-500/[0.12]",
  // The guest is at the table. In progress, like a dish in the kitchen.
  seated: "border-amber-500/50 bg-amber-500/[0.13]",
  done: "border-line bg-surface",
  cancelled: "border-rose-500/45 bg-rose-500/[0.12]",
};

// ---- Couriers ----
//
// The same three ideas one more time: green is ready, amber is out on a job,
// and a rider who is off shift has nothing wanting doing about them, so their
// row stays plain.
//
// Note what is *not* red. Being off shift is not a fault — a courier who
// finished at six should not glow like a cancelled order, or the colour stops
// meaning "look at this".
export const COURIER_BADGE: Record<CourierStatus, string> = {
  off: "bg-ink/10 text-ink-muted",
  free: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  busy: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
};

export const COURIER_ROW: Record<CourierStatus, string> = {
  off: "",
  free: "bg-emerald-500/[0.12]",
  busy: "bg-amber-500/[0.13]",
};
