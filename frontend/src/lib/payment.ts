import type { PaymentMethod } from "./types";

export const PAYMENT_METHODS: PaymentMethod[] = [
  "cash",
  "payme",
  "click",
  "uzum",
];

export const PAYMENT_LABEL: Record<PaymentMethod, string> = {
  cash: "Naqd",
  payme: "Payme",
  click: "Click",
  uzum: "Uzum",
};
