"use client";

// One model, many things on the shelf.
//
// ⚠️ **This screen removes the typing, not the rows.** A clothes shop does not
// sell "a shirt", it sells M/black: each size and colour has its own barcode,
// its own count and its own delivery. A hundred models in five sizes and three
// colours is fifteen hundred products either way — the question is whether the
// shop enters them by hand over a fortnight or presses a button. Its absence is
// the one thing that sends a clothes shop to a competitor on the first demo.
//
// ⚠️ **Only on a saved product.** The variants are written against an id, and a
// model that has not been saved has none. Offering the button first would open
// a dialog that cannot finish and lose the form behind it — the same rule the
// technical card link follows.

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import type { MenuItem } from "@/lib/types";
import { useAdminT } from "@/lib/i18n/admin";

type Axis = { name: string; values: string };

export default function VariantsEditor({
  item,
  variants,
  onDone,
}: {
  /** The saved model these variants belong to. */
  item: MenuItem;
  /** The variants it already has, so the screen can say what exists.
   *
   *  ⚠️ Named `variants` and not `children`: React gives that prop a meaning of
   *  its own, and a component that takes both is a component whose two lists
   *  are one typo apart. */
  variants: MenuItem[];
  /** Reload the list — new products have appeared in it. */
  onDone: () => void;
}) {
  const t = useAdminT();
  const [axes, setAxes] = useState<Axis[]>(() =>
    (item.variantAxes ?? []).length > 0
      ? // ⚠️ The existing axes come back **without their values**: the values
        // live on the variants themselves, and re-deriving them from the rows
        // would offer to regenerate exactly what is already there. The shop
        // types what it is adding.
        (item.variantAxes ?? []).map((name) => ({ name, values: "" }))
      : [{ name: "", values: "" }],
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [made, setMade] = useState<number | null>(null);

  async function generate() {
    setBusy(true);
    setError("");
    setMade(null);
    try {
      const res = await api.generateVariants(
        item.id,
        axes.map((a) => ({
          name: a.name.trim(),
          // Commas, because that is how a person writes a list of sizes.
          values: a.values
            .split(",")
            .map((v) => v.trim())
            .filter(Boolean),
        })),
      );
      setMade(res.created);
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-xl bg-ink/[0.03] p-3">
      {axes.map((a, i) => (
        <div key={i} className="mb-2 grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
          <input
            className="rounded-lg border border-line bg-surface px-3 py-2 text-sm"
            placeholder={t.menu.variantAxisPh}
            value={a.name}
            onChange={(e) =>
              setAxes(axes.map((x, j) => (i === j ? { ...x, name: e.target.value } : x)))
            }
          />
          <input
            className="rounded-lg border border-line bg-surface px-3 py-2 text-sm"
            placeholder={t.menu.variantValuesPh}
            value={a.values}
            onChange={(e) =>
              setAxes(axes.map((x, j) => (i === j ? { ...x, values: e.target.value } : x)))
            }
          />
        </div>
      ))}

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="rounded-lg border border-line px-3 py-1.5 text-xs font-semibold"
          onClick={() => setAxes([...axes, { name: "", values: "" }])}
        >
          {t.menu.variantAddAxis}
        </button>
        <button
          type="button"
          className="rounded-lg bg-brand px-3 py-1.5 text-xs font-semibold text-white disabled:opacity-50"
          disabled={busy}
          onClick={() => void generate()}
        >
          {t.menu.variantGenerate}
        </button>
        {variants.length > 0 && (
          <span className="text-xs text-ink-muted">
            {t.menu.variantHave(variants.length)}
          </span>
        )}
      </div>

      {/* ⚠️ **Says how many were created, not "saved".** Pressing this twice is
          ordinary — a shop adds a colour in March — and "0 ta yaratildi" is the
          honest answer that stops somebody pressing it a third time. */}
      {made !== null && !error && (
        <p className="mt-2 text-xs text-ink-muted">{t.menu.variantMade(made)}</p>
      )}
      {error && <p className="mt-2 text-xs text-danger">{error}</p>}

      <p className="mt-2 text-xs text-ink-muted">{t.menu.variantHint}</p>
    </div>
  );
}
