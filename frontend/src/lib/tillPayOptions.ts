import { useEffect, useState } from "react";

import { api } from "./api";
import type { TillPayOption, TillPaymentMethod } from "./types";

// The till's payment buttons: the owner's named ones, then the rails.
//
// ⚠️ **A button is a name over a kind.** "Humo terminal" is booked as `card`,
// "Naqd (kassa 2)" as `cash` — the drawer, the shift report and the financial
// report only ever read the kind, so a name can be anything and nothing
// downstream has to learn it.

export const TILL_KINDS = ["cash", "card", "transfer"] as const;

/** The owner's buttons from the server's answer. ⚠️ An older server sends only
 *  `methods`; the three defaults are rebuilt from the kinds it lists, so a till
 *  talking to it still has buttons to press. */
export function payOptionsFrom(
  methods: TillPaymentMethod[],
  options?: TillPayOption[] | null,
): TillPayOption[] {
  if (options && options.length > 0) return options;
  return TILL_KINDS.filter((k) => methods.includes(k)).map((k) => ({
    id: k,
    name: "",
    kind: k,
  }));
}

export interface TillPay {
  methods: TillPaymentMethod[];
  options: TillPayOption[];
}

/** ⚠️ **What a till offers before it has asked**, and what it keeps when it
 *  cannot: a till whose network is down still has to take cash. */
export const DEFAULT_TILL_PAY: TillPay = {
  methods: ["cash", "card", "transfer", "debt"],
  options: payOptionsFrom(["cash", "card", "transfer"]),
};

/** The rails and the slate — everything that is not one of the owner's
 *  buttons. */
export function railsOf(methods: TillPaymentMethod[]): TillPaymentMethod[] {
  return methods.filter(
    (m) => !(TILL_KINDS as readonly string[]).includes(m) && m !== "debt",
  );
}

/** Asks the server once per mount. ⚠️ Not cached across mounts: the owner may
 *  have just renamed a button, and a till that kept the old list until a reload
 *  is a till showing a button that no longer exists. */
export function useTillPay(enabled = true): TillPay {
  const [pay, setPay] = useState<TillPay>(DEFAULT_TILL_PAY);
  useEffect(() => {
    if (!enabled) return;
    let live = true;
    api
      .tillPaymentMethods()
      .then((d) => {
        if (live && d.methods.length > 0) {
          setPay({ methods: d.methods, options: payOptionsFrom(d.methods, d.options) });
        }
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, [enabled]);
  return pay;
}

interface KindLabels {
  methodCash: string;
  methodCard: string;
  methodCardShort: string;
  methodTransfer: string;
}

/** The name a cashier reads. The owner's own name when there is one. */
export function optionLabel(o: TillPayOption, t: KindLabels): string {
  if (o.name) return o.name;
  return o.kind === "cash" ? t.methodCash : o.kind === "card" ? t.methodCard : t.methodTransfer;
}

/** The same, short enough for a chip in a 300px column. */
export function optionShortLabel(o: TillPayOption, t: KindLabels): string {
  if (o.name) return o.name;
  return o.kind === "card" ? t.methodCardShort : optionLabel(o, t);
}
