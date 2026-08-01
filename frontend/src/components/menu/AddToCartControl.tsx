"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useCart, type SelectedOption } from "@/lib/cart";
import { formatPrice } from "@/lib/format";
import { useI18n } from "@/lib/i18n/client";
import OptionPicker from "./OptionPicker";
import type { MenuItem } from "@/lib/types";

export default function AddToCartControl({
  item,
  currency = "UZS",
}: {
  item: MenuItem;
  currency?: string;
}) {
  const { add } = useCart();
  const { lang, t } = useI18n();
  const [qty, setQty] = useState(1);
  const [added, setAdded] = useState(false);
  const [selected, setSelected] = useState<SelectedOption[]>([]);
  const [showErrors, setShowErrors] = useState(false);

  const groups = useMemo(
    () => (item.options ?? []).filter((g) => g.choices?.length),
    [item.options],
  );

  // Required groups the customer has not answered yet.
  const missing = useMemo(
    () =>
      groups
        .filter(
          (g) => g.required && !selected.some((s) => s.group.name === g.name),
        )
        .map((g) => g.name),
    [groups, selected],
  );

  const unitPrice = Math.max(
    0,
    selected.reduce((sum, s) => sum + s.priceDelta, item.price),
  );

  function handleAdd() {
    if (missing.length) {
      setShowErrors(true);
      return;
    }
    add(item, qty, selected);
    setAdded(true);
    setQty(1);
    setShowErrors(false);
    setTimeout(() => setAdded(false), 2000);
  }

  if (!item.isAvailable || item.soldOut) {
    return (
      <p className="rounded-xl bg-ink/5 px-4 py-3 text-sm text-ink-muted">
        {item.soldOut && item.isAvailable ? t.item.soldOut : t.item.unavailable}
      </p>
    );
  }

  return (
    <div className="space-y-5">
      {groups.length > 0 && (
        <OptionPicker
          options={groups}
          selected={selected}
          onChange={(next) => {
            setSelected(next);
            setShowErrors(false);
          }}
          currency={currency}
          missing={showErrors ? missing : []}
        />
      )}

      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center rounded-xl border border-line-strong">
          <button
            type="button"
            onClick={() => setQty((q) => Math.max(1, q - 1))}
            className="px-4 py-2 text-lg text-ink-muted hover:text-brand"
            aria-label={t.item.decrease}
          >
            −
          </button>
          <span className="w-10 text-center font-medium">{qty}</span>
          <button
            type="button"
            onClick={() => setQty((q) => q + 1)}
            className="px-4 py-2 text-lg text-ink-muted hover:text-brand"
            aria-label={t.item.increase}
          >
            +
          </button>
        </div>

        <button
          type="button"
          onClick={handleAdd}
          className="btn-primary px-6 py-2.5"
        >
          {t.item.addToCart}
          {unitPrice !== item.price && (
            <span className="ml-2 font-normal">
              · {formatPrice(unitPrice * qty, currency, lang)}
            </span>
          )}
        </button>

        {added && (
          <Link
            href="/cart"
            className="text-sm font-medium text-brand hover:underline"
          >
            {t.item.added}
          </Link>
        )}
      </div>
    </div>
  );
}
