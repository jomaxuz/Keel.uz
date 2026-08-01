"use client";

// Client-side cart persisted in localStorage. Checkout sends these lines to
// POST /orders, where the backend re-prices every item (and every selected
// option) from the DB — client totals are display-only and never trusted.

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useState,
  type ReactNode,
} from "react";
import { api } from "./api";
import { cartKeyFor } from "./siteBrand";
import type { MenuItem, MenuOption, OptionChoice } from "./types";

// A choice the customer picked, kept with its translations so the cart can be
// displayed in any language. The order itself is always sent with the base (uz)
// names — see checkout.
export interface SelectedOption {
  group: { name: string; nameRu?: string; nameEn?: string };
  choice: { name: string; nameRu?: string; nameEn?: string };
  priceDelta: number;
}

export interface CartLine {
  // Identity of the line: the same dish with different options = two lines.
  lineId: string;
  menuItemId: string;
  name: string; // uz (base) — this is what the order is created with
  nameRu?: string;
  nameEn?: string;
  basePrice: number; // dish price without options
  price: number; // unit price incl. option deltas
  imageUrl: string;
  qty: number;
  options: SelectedOption[];
  /** Set on a combo: what the set contains, for the basket and the receipt. */
  comboContents?: { name: string; qty: number }[];
  // Note the customer typed for this line ("piyozsiz") — sent with the order
  // and shown to the kitchen. Not part of the line identity: the same dish with
  // the same options stays one line, and the note applies to all of it.
  comment?: string;
}

interface CartState {
  lines: CartLine[];
}

type CartAction =
  | { type: "add"; line: CartLine }
  | { type: "sync"; lines: CartLine[] }
  | { type: "setQty"; lineId: string; qty: number }
  | { type: "setComment"; lineId: string; comment: string }
  | { type: "remove"; lineId: string }
  | { type: "clear" }
  | { type: "hydrate"; state: CartState };

// v2: lines are keyed by dish + options (v1 carts are simply dropped). A
// company with several brands gets one cart per brand — see cartKeyFor.

// Stable identity for a dish + option selection. Sorted so the order in which
// the choices were ticked cannot create a second line for the same selection.
export function lineIdFor(
  menuItemId: string,
  options: SelectedOption[],
): string {
  if (!options.length) return menuItemId;
  const sig = options
    .map((o) => `${o.group.name}=${o.choice.name}`)
    .sort()
    .join("|");
  return `${menuItemId}#${sig}`;
}

/**
 * A stored cart turned into state we can reduce over, dropping anything that
 * does not look like a line. Storage is not ours to trust: a cart written by an
 * older build, a half-finished write or a hand-edited value used to reach the
 * reducer as-is, and one missing `lines` array then threw on every render —
 * taking down every page on the site, not just the cart.
 */
function parseStored(raw: string): CartState {
  const data: unknown = JSON.parse(raw);
  const lines = (data as { lines?: unknown } | null)?.lines;
  if (!Array.isArray(lines)) return { lines: [] };
  return {
    lines: lines.filter((l): l is CartLine => {
      const line = l as Partial<CartLine> | null;
      return (
        !!line &&
        typeof line.lineId === "string" &&
        typeof line.menuItemId === "string" &&
        typeof line.price === "number" &&
        typeof line.qty === "number" &&
        Array.isArray(line.options)
      );
    }),
  };
}

function unitPrice(base: number, options: SelectedOption[]): number {
  return Math.max(
    0,
    options.reduce((sum, o) => sum + o.priceDelta, base),
  );
}

function reducer(state: CartState, action: CartAction): CartState {
  switch (action.type) {
    case "add": {
      const existing = state.lines.find((l) => l.lineId === action.line.lineId);
      if (existing) {
        return {
          lines: state.lines.map((l) =>
            l.lineId === action.line.lineId
              ? { ...l, qty: l.qty + action.line.qty }
              : l,
          ),
        };
      }
      return { lines: [...state.lines, action.line] };
    }
    case "setQty": {
      if (action.qty <= 0) {
        return { lines: state.lines.filter((l) => l.lineId !== action.lineId) };
      }
      return {
        lines: state.lines.map((l) =>
          l.lineId === action.lineId ? { ...l, qty: action.qty } : l,
        ),
      };
    }
    case "setComment":
      return {
        lines: state.lines.map((l) =>
          l.lineId === action.lineId ? { ...l, comment: action.comment } : l,
        ),
      };
    case "remove":
      return { lines: state.lines.filter((l) => l.lineId !== action.lineId) };
    case "clear":
      return { lines: [] };
    case "sync":
      return { lines: action.lines };
    case "hydrate":
      return action.state;
    default:
      return state;
  }
}

interface CartContextValue {
  lines: CartLine[];
  count: number;
  subtotal: number;
  add: (item: MenuItem, qty?: number, options?: SelectedOption[]) => void;
  setQty: (lineId: string, qty: number) => void;
  setComment: (lineId: string, comment: string) => void;
  remove: (lineId: string) => void;
  clear: () => void;
  // How many lines were dropped because the dish disappeared from the menu.
  removedCount: number;
  dismissRemoved: () => void;
}

const CartContext = createContext<CartContextValue | null>(null);

// Re-resolve a stored selection against the live menu: an option group or
// choice may have been renamed, repriced or deleted in the admin panel.
// Returns null when the line is no longer orderable as stored.
function resyncOptions(
  item: MenuItem,
  stored: SelectedOption[],
): SelectedOption[] | null {
  const groups: MenuOption[] = item.options ?? [];
  const next: SelectedOption[] = [];
  for (const sel of stored) {
    const group = groups.find((g) => g.name === sel.group.name);
    const choice: OptionChoice | undefined = group?.choices?.find(
      (c) => c.name === sel.choice.name,
    );
    if (!group || !choice) return null;
    next.push({
      group: { name: group.name, nameRu: group.nameRu, nameEn: group.nameEn },
      choice: {
        name: choice.name,
        nameRu: choice.nameRu,
        nameEn: choice.nameEn,
      },
      priceDelta: choice.priceDelta,
    });
  }
  // A group that became required after the dish was added must be answered.
  for (const g of groups) {
    if (
      g.required &&
      g.choices?.length &&
      !next.some((s) => s.group.name === g.name)
    ) {
      return null;
    }
  }
  return next;
}

export function CartProvider({
  children,
  brandId,
}: {
  children: ReactNode;
  /** Which brand's basket this is. Empty on a single-brand install. */
  brandId?: string;
}) {
  const [state, dispatch] = useReducer(reducer, { lines: [] });
  const [removedCount, setRemovedCount] = useState(0);
  const storageKey = cartKeyFor(brandId);

  // Hydrate on mount, and again whenever the guest moves to another brand:
  // switching brand swaps the whole basket, it does not merge it.
  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(storageKey);
      dispatch({
        type: "hydrate",
        state: raw ? parseStored(raw) : { lines: [] },
      });
    } catch {
      // corrupt storage — start empty
      dispatch({ type: "hydrate", state: { lines: [] } });
    }
  }, [storageKey]);

  // Reconcile the stored cart with the live menu: a dish may have been renamed,
  // repriced, made unavailable or deleted since the customer added it. Without
  // this the order endpoint rejects the whole cart ("… menyuda topilmadi").
  useEffect(() => {
    let cancelled = false;
    async function sync() {
      const raw = window.localStorage.getItem(storageKey);
      if (!raw) return;
      let stored: CartState;
      try {
        stored = parseStored(raw);
      } catch {
        return;
      }
      if (!stored.lines?.length) return;
      try {
        // Scoped to this brand: the menu the cart's dishes actually come from.
        const groups = await api.getMenu(brandId ? { brand: brandId } : undefined);
        if (cancelled) return;
        const live = new Map(
          groups.flatMap((g) => g.items).map((i) => [i.id, i]),
        );
        const next: CartLine[] = [];
        for (const line of stored.lines) {
          const item = live.get(line.menuItemId);
          if (!item || !item.isAvailable) continue;
          const options = resyncOptions(item, line.options ?? []);
          if (!options) continue;
          next.push({
            ...line,
            lineId: lineIdFor(item.id, options),
            name: item.name,
            nameRu: item.nameRu,
            nameEn: item.nameEn,
            basePrice: item.price,
            price: unitPrice(item.price, options),
            imageUrl: item.imageUrl,
            options,
            comboContents: item.comboContents,
          });
        }
        if (next.length !== stored.lines.length) {
          setRemovedCount(stored.lines.length - next.length);
        }
        dispatch({ type: "sync", lines: next });
      } catch {
        // Menu unreachable — keep the cart as-is.
      }
    }
    sync();
    return () => {
      cancelled = true;
    };
  }, [storageKey, brandId]);

  // Persist on every change.
  useEffect(() => {
    try {
      window.localStorage.setItem(storageKey, JSON.stringify(state));
    } catch {
      // storage full / unavailable — ignore
    }
  }, [state, storageKey]);

  const value = useMemo<CartContextValue>(() => {
    const count = state.lines.reduce((n, l) => n + l.qty, 0);
    const subtotal = state.lines.reduce((s, l) => s + l.price * l.qty, 0);
    return {
      lines: state.lines,
      count,
      subtotal,
      add: (item, qty = 1, options = []) =>
        dispatch({
          type: "add",
          line: {
            lineId: lineIdFor(item.id, options),
            menuItemId: item.id,
            name: item.name,
            nameRu: item.nameRu,
            nameEn: item.nameEn,
            basePrice: item.price,
            price: unitPrice(item.price, options),
            imageUrl: item.imageUrl,
            qty,
            options,
            comboContents: item.comboContents,
          },
        }),
      setQty: (lineId, qty) => dispatch({ type: "setQty", lineId, qty }),
      setComment: (lineId, comment) =>
        dispatch({ type: "setComment", lineId, comment }),
      remove: (lineId) => dispatch({ type: "remove", lineId }),
      clear: () => dispatch({ type: "clear" }),
      removedCount,
      dismissRemoved: () => setRemovedCount(0),
    };
  }, [state, removedCount]);

  return <CartContext.Provider value={value}>{children}</CartContext.Provider>;
}

export function useCart(): CartContextValue {
  const ctx = useContext(CartContext);
  if (!ctx) throw new Error("useCart must be used within a CartProvider");
  return ctx;
}
