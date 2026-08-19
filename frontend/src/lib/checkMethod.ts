import type { AdminDict } from "@/lib/i18n/admin";

/** How the money came in, in the reader's language.
 *
 *  ⚠️ One helper for the list and the opened check. Two copies drift, and the
 *  drift shows up as the same sale reading "Karta" in one place and "card" in
 *  the other — which is exactly the kind of difference somebody investigating a
 *  short till stops to worry about. */
export function paymentLabel(m: string | undefined, t: AdminDict): string {
  if (m === "cash") return t.sales.cash;
  if (m === "card") return t.sales.card;
  if (m === "transfer") return t.sales.transfer;
  return m || "—";
}
