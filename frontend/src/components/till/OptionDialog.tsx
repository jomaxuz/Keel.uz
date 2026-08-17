"use client";

// Choosing a dish's options at the till.
//
// ⚠️ **Without this, every dish with a required option was unsellable from the
// till.** The tablet sent `{menuItemId, qty}` and nothing else, the server did
// the right thing — `resolveOptions` refuses a required group that was not
// answered — and the cashier got «Osh (palov): "hajm" tanlanmagan» with no way
// on any screen to answer it. Not a rare corner: portion size is the most
// ordinary option a restaurant has, and the failure looked like a broken till
// rather than a missing screen.
//
// ⚠️ **It opens only for dishes that have groups.** One tap adds a dish, and
// that is the whole speed argument for a till — a confirm step on every dish
// doubles every tap on the screen that is used most. So the dialog is not a
// general "add dish" flow that happens to show options; it is the answer to a
// question the dish actually asks.
//
// ⚠️ **Courtesy only, exactly like the disabled buttons elsewhere.** The server
// validates the same rules on the way in. If a group is edited while the dialog
// is open, the refusal is still correct and still arrives.

import { useMemo, useState } from "react";

import { formatPrice } from "@/lib/format";
import { contentName } from "@/lib/i18n/content";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { MenuItem, OrderItemOption } from "@/lib/types";

export default function OptionDialog({
  item,
  currency,
  busy,
  onCancel,
  onAdd,
}: {
  item: MenuItem;
  currency: string;
  busy: boolean;
  onCancel: () => void;
  /** The choices in wire shape, plus how many. The server re-prices both. */
  onAdd: (options: OrderItemOption[], qty: number) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const groups = item.options ?? [];

  // ⚠️ Keyed by the **base (uz) name**, which is what goes on the wire and what
  // the server matches against. Storing the translated label would send the
  // cashier's interface language into the order and fail on a Russian till.
  const [picked, setPicked] = useState<OrderItemOption[]>([]);
  const [qty, setQty] = useState(1);

  const has = (group: string, choice: string) =>
    picked.some((p) => p.name === group && p.choice === choice);

  function toggle(groupName: string, multiple: boolean, choiceName: string) {
    const group = groups.find((g) => g.name === groupName);
    const choice = group?.choices.find((c) => c.name === choiceName);
    if (!group || !choice) return;
    const entry: OrderItemOption = {
      name: group.name,
      choice: choice.name,
      // Sent for the running total below only. The server reads the delta off
      // the menu — the tablet says which choice, never what it costs.
      priceDelta: choice.priceDelta,
    };
    if (multiple) {
      setPicked((prev) =>
        has(groupName, choiceName)
          ? prev.filter(
              (p) => !(p.name === groupName && p.choice === choiceName),
            )
          : [...prev, entry],
      );
      return;
    }
    // Single choice replaces whatever this group held. ⚠️ Tapping the current
    // choice clears it only when the group is optional: a required group with
    // nothing selected is a dialog whose button just went dead, and the cashier
    // reads that as the screen freezing.
    setPicked((prev) => {
      const rest = prev.filter((p) => p.name !== groupName);
      if (has(groupName, choiceName) && !group.required) return rest;
      return [...rest, entry];
    });
  }

  /** Required groups nobody has answered — the same rule the server applies. */
  const missing = useMemo(
    () =>
      groups
        .filter(
          (g) =>
            g.required &&
            g.choices.length > 0 &&
            !picked.some((p) => p.name === g.name),
        )
        .map((g) => g.name),
    [groups, picked],
  );

  // ⚠️ Shown because it is not the price on the tile. A large portion at
  // +15 000 lands on the check at a number the cashier never saw, and reading
  // the total back to a guest is the one thing this screen exists to make
  // possible.
  const unit =
    item.price + picked.reduce((sum, p) => sum + (p.priceDelta || 0), 0);

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/50 p-4 sm:items-center">
      <div className="till flex max-h-[85vh] w-full max-w-md flex-col rounded-[14px] border border-line bg-surface">
        <header className="shrink-0 border-b border-line px-4 py-3">
          <h2 className="font-display text-lg font-bold">
            {contentName(item, lang)}
          </h2>
        </header>

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
          {groups.map((group) => (
            <div key={group.name}>
              <div className="flex items-baseline gap-2">
                <span className="till-label">{contentName(group, lang)}</span>
                {/* ⚠️ Required is said, not implied by a disabled button. A
                    cashier who cannot see why "Qo'shish" is dead starts tapping
                    it. */}
                {group.required && (
                  <span className="text-[11px] font-semibold text-brand">
                    {t.till.optionRequired}
                  </span>
                )}
              </div>
              <div className="mt-1.5 grid grid-cols-2 gap-1.5">
                {group.choices.map((choice) => {
                  const on = has(group.name, choice.name);
                  return (
                    <button
                      key={choice.name}
                      type="button"
                      onClick={() =>
                        toggle(group.name, group.multiple, choice.name)
                      }
                      className={`flex min-h-12 flex-col justify-center rounded-[10px] border px-3 py-2 text-left transition active:scale-[0.98] ${
                        on
                          ? "border-brand bg-brand/10"
                          : "border-line bg-surface hover:bg-ink/[0.03]"
                      }`}
                    >
                      <span className="text-sm font-semibold leading-tight">
                        {contentName(choice, lang)}
                      </span>
                      {/* Zero is left blank rather than printed as "+0 so'm":
                          a price that changes nothing is not information. */}
                      {choice.priceDelta !== 0 && (
                        <span className="text-xs tabular-nums text-ink-muted">
                          {choice.priceDelta > 0 ? "+" : "−"}
                          {formatPrice(
                            Math.abs(choice.priceDelta),
                            currency,
                            lang,
                          )}
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>

        <footer className="shrink-0 space-y-2 border-t border-line p-3">
          {/* ⚠️ Quantity lives here rather than as four taps on the tile: the
              dialog has already interrupted, so this is the one moment where
              asking costs nothing. A dish without options keeps its one tap. */}
          <div className="flex items-center justify-between gap-2">
            <span className="till-label">{t.till.qty}</span>
            <div className="flex items-center gap-1.5">
              <button
                type="button"
                className="till-btn w-12 px-0 text-lg"
                onClick={() => setQty((n) => Math.max(1, n - 1))}
                disabled={qty <= 1}
              >
                −
              </button>
              <span className="w-10 text-center font-display text-xl font-bold tabular-nums">
                {qty}
              </span>
              <button
                type="button"
                className="till-btn w-12 px-0 text-lg"
                onClick={() => setQty((n) => Math.min(99, n + 1))}
              >
                +
              </button>
            </div>
          </div>

          <div className="flex items-baseline justify-between gap-2">
            <span className="till-label">{t.till.total}</span>
            <span className="font-display text-xl font-bold tabular-nums">
              {formatPrice(unit * qty, currency, lang)}
            </span>
          </div>

          <div className="flex gap-2">
            <button type="button" className="till-btn flex-1" onClick={onCancel}>
              {t.till.back}
            </button>
            <button
              type="button"
              className="till-btn-primary flex-1"
              disabled={busy || missing.length > 0}
              onClick={() => onAdd(picked, qty)}
            >
              {t.till.add}
            </button>
          </div>
        </footer>
      </div>
    </div>
  );
}
