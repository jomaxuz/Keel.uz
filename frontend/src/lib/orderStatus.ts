import type { OrderStatus } from "./types";

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

// Tailwind badge classes per status.
export const STATUS_BADGE: Record<OrderStatus, string> = {
  pending: "bg-amber-100 text-amber-700",
  confirmed: "bg-blue-100 text-blue-700",
  preparing: "bg-indigo-100 text-indigo-700",
  on_the_way: "bg-purple-100 text-purple-700",
  delivered: "bg-emerald-100 text-emerald-700",
  cancelled: "bg-rose-100 text-rose-700",
};
