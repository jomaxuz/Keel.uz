import type { PaymentMethod, OrderPaymentMethod } from "./types";

export const PAYMENT_METHODS: PaymentMethod[] = [
  "cash",
  "payme",
  "click",
  "uzum",
  "atmos",
];

export const PAYMENT_LABEL: Record<OrderPaymentMethod, string> = {
  cash: "Naqd",
  payme: "Payme",
  click: "Click",
  uzum: "Uzum",
  // The gateway's own name, as the guest sees it on checkout.atmos.uz — a
  // label they do not recognise on the button they are about to press is a
  // reason to close the tab.
  atmos: "ATMOS",
  // Paid to the marketplace, which settles later — see the payouts screen.
  uzum_tezkor: "Uzum Tezkor",
};
