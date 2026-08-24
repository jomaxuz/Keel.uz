"use client";

// What this restaurant bought, for the panel to draw with.
//
// ⚠️ **Courtesy, not the boundary.** The rule lives on the server, in one table
// matched on the request path (`handlers/modulegate.go`), and everything here
// exists only so an owner meets a sentence and a price instead of a screen
// whose every button answers 402. A panel that hid a section it could not
// reach *without saying why* teaches somebody their tools are broken; a panel
// that let them press on regardless teaches them the same thing more slowly.
//
// ⚠️ The path table below is a **second copy** of the server's, and it is
// allowed to drift in exactly one harmless direction: a path this file forgets
// still gets refused by the server, and a path it names that the server does
// not gate simply shows an upgrade card to somebody who could have used the
// screen. The first is a support call; the second is a bug we would hear about
// immediately. Neither can grant anything, because nothing here decides access.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { api } from "@/lib/api";

export interface SubscriptionPlan {
  id: string;
  monthly: number;
  registers: number;
  modules: string[];
  individual: boolean;
}

export interface SubscriptionNotice {
  days: number;
  level: "warn" | "urgent" | "expired";
  until: string;
}

export interface SubscriptionState {
  enabled: boolean;
  plan?: string;
  modules: string[];
  /** Bought on top of the rung. Already inside `modules`; this is for naming
   *  it in a heading, never for deciding access. */
  addons?: string[];
  registers?: number;
  branches?: number;
  /** So'm per month, resolved by the console — discounts, add-ons and any
   *  negotiated price already applied. ⚠️ 0 means "agreed separately", not
   *  free: an Enterprise price is per customer and the console mirrors nothing
   *  rather than invent one. */
  monthly?: number;
  /** Paid through, "YYYY-MM-DD", already local. ⚠️ A string from the server,
   *  never a timestamp sliced in the browser — that returns the previous day
   *  in Tashkent. */
  paidUntil?: string;
  notice?: SubscriptionNotice | null;
  plans: SubscriptionPlan[];
}

const Ctx = createContext<{
  sub: SubscriptionState | null;
  /** Whether a module is available.
   *
   *  ⚠️ **True while the answer is still in flight**, and only then. An upgrade
   *  card that flashes over a working screen on every page load is worse than
   *  one that appears a beat late — and the server is the boundary either way,
   *  so guessing generously here costs nothing but a moment of courtesy.
   *
   *  ⚠️ It is deliberately **not** true for an install with no subscription any
   *  more. That was the old rule on both sides, and it handed the stock module
   *  to every restaurant on the per-order website plan. The server now refuses
   *  those paths, so a panel that still drew them would offer a section whose
   *  every request answers 402. */
  has: (mod: string) => boolean;
  /** The cheapest rung that includes a module, for the button's label. */
  planFor: (mod: string) => SubscriptionPlan | undefined;
}>({ sub: null, has: () => true, planFor: () => undefined });

export function SubscriptionProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [sub, setSub] = useState<SubscriptionState | null>(null);

  useEffect(() => {
    let alive = true;
    api
      .subscription()
      .then((s) => {
        if (alive) setSub(s);
      })
      // ⚠️ Swallowed on purpose. This request failing must never take a screen
      // with it: the panel's job does not depend on knowing the plan, and an
      // error here would close sections the customer has paid for.
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, []);

  const has = useCallback(
    (mod: string) => {
      // ⚠️ `null` is "not answered yet", not "nothing bought" — including when
      // the request failed, which the provider swallows on purpose. A network
      // blip must never close a section the customer pays for; the server
      // decides, and it is still deciding correctly while this is null.
      if (!sub) return true;
      if (!sub.enabled) return false;
      return sub.modules.includes(mod);
    },
    [sub],
  );

  const planFor = useCallback(
    (mod: string) => sub?.plans.find((p) => p.modules.includes(mod)),
    [sub],
  );

  const value = useMemo(() => ({ sub, has, planFor }), [sub, has, planFor]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useSubscription() {
  return useContext(Ctx);
}

/** Module ids, mirroring models/subscription.go. Stored strings — renaming one
 *  here only breaks the panel's courtesy, but breaking it silently is still
 *  worse than not having it. */
export const MOD = {
  stock: "stock",
  multibranch: "multibranch",
  posint: "posint",
  franchise: "franchise",
} as const;

/** Panel paths to modules. Longest prefix wins, as on the server — the
 *  supplier report is stock, not reports, and it sits inside the reports path. */
const PANEL_ROUTES: Array<[string, string]> = ([
  ["/admin/ingredients", MOD.stock],
  ["/admin/warehouses", MOD.stock],
  ["/admin/purchases", MOD.stock],
  ["/admin/suppliers", MOD.stock],
  ["/admin/writeoffs", MOD.stock],
  ["/admin/transfers", MOD.stock],
  ["/admin/stocktake", MOD.stock],
  ["/admin/stock", MOD.stock],
  ["/admin/shopping", MOD.stock],
  ["/admin/pos", MOD.posint],
  // ⚠️ Reports, campaigns and the call centre are **not** here, and must not be
  // added: they are in the price for everybody. Gating them means a restaurant
  // that buys a till loses screens it already had — see modulegate.go.
] as Array<[string, string]>).sort((a, b) => b[0].length - a[0].length);

/** Which module a panel path needs, or "" when it is not sold separately. */
export function moduleForPath(path: string): string {
  for (const [prefix, mod] of PANEL_ROUTES) {
    if (path.startsWith(prefix)) return mod;
  }
  return "";
}
