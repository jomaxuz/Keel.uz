"use client";

// The shopping list: what the kitchen buys and what it pays.
//
// ⚠️ **One price, in the unit it is bought in.** An invoice says "one kilo,
// 90 000" — nobody has a price per gram written anywhere, and a form asking for
// one is a form that gets the wrong number. The conversion to grams happens in
// the tech card, where the recipe is written.
//
// ⚠️ **This is costing, not stock.** There is no quantity on hand here and
// there deliberately is not: a restaurant that believes a stock figure and
// finds it wrong stops believing the panel entirely. What this list answers is
// "what does a portion cost", and the answer updates itself the day a price
// changes — which is the whole reason it exists.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";

import { api, ApiError } from "@/lib/api";
import { formatDate, formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import Modal from "@/components/admin/Modal";
import PosImport from "@/components/admin/PosImport";
import WarehousesEditor, {
  WarehousePicker,
} from "@/components/admin/WarehousesEditor";
import type { Ingredient, RecipeLine, Warehouse } from "@/lib/types";
import { qtyNumber } from "@/lib/qty";
import { QtyInput } from "@/components/QtyInput";

const UNITS = ["kg", "l", "pcs"] as const;

type Draft = Omit<Partial<Ingredient>, "recipe" | "output" | "minQty"> & {
  name: string;
  unit: string;
  price: number;
  /** Always an array in the draft: "no card" is an empty one, and a field that
   *  might be undefined would have every use of it guarded for a state this
   *  form cannot be in. */
  recipe: RecipeLine[];
  output: number;
  /** Warn below this. Zero is "do not warn me". */
  minQty: number;
  /** Which store it is kept in. Empty is the undivided one. */
  warehouseId: string;
};

const EMPTY: Draft = {
  name: "",
  unit: "kg",
  price: 0,
  recipe: [],
  output: 0,
  minQty: 0,
  warehouseId: "",
};

export default function IngredientsPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [rows, setRows] = useState<Ingredient[]>([]);
  // When the expected figures were last anchored to a count. ⚠️ Shown, and
  // "never" is the most important thing it can say: without a count the
  // estimate is every delivery ever, less everything the cards account for.
  const [countedAt, setCountedAt] = useState<string | null>(null);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [draft, setDraft] = useState<Draft>(EMPTY);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [importing, setImporting] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminIngredients()
      .then((d) => {
        setRows(d.ingredients);
        setCountedAt(d.countedAt);
      })
      .catch(() => setError(t.common.loadFailed))
      .finally(() => setLoading(false));
  }, [t.common.loadFailed]);

  useEffect(load, [load, scope.scopeKey]);

  const loadWarehouses = useCallback(() => {
    api
      .adminWarehouses()
      .then((d) => setWarehouses(d.warehouses))
      .catch(() => setWarehouses([]));
  }, []);
  useEffect(loadWarehouses, [loadWarehouses, scope.scopeKey]);

  async function save() {
    if (!draft.name.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api.adminSaveIngredient({
        id: draft.id,
        name: draft.name.trim(),
        unit: draft.unit,
        price: Math.max(0, Math.round(draft.price) || 0),
        note: draft.note ?? "",
        minQty: draft.minQty,
        // ⚠️ Both halves or neither — the server drops a half-filled pair for
        // the same reason: a size with no name multiplies a buyer's quantity by
        // a factor nobody can see on screen.
        packName: (draft.packName ?? "").trim(),
        packQty: draft.packQty ?? 0,
        warehouseId: draft.warehouseId,
        // ⚠️ **Still sent, even though this form no longer edits it.** Saving
        // an ingredient replaces the whole document, so a prep item whose name
        // or minimum is corrected here would lose its card — and the loss
        // would show up as a sauce that suddenly costs nothing, weeks later,
        // in a margin nobody could explain. Carried through untouched; the
        // card is edited in Ombor → Texkartalar.
        //
        // Empty card and zero yield = an ordinary bought ingredient.
        recipe: draft.recipe,
        output: draft.output,
        batched: !!draft.batched,
      });
      setDraft(EMPTY);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function remove(row: Ingredient) {
    setBusy(true);
    setError("");
    try {
      await api.adminDeleteIngredient(row.id);
      load();
    } catch (e) {
      // ⚠️ The refusal names the dishes that still hold it. A recipe pointing
      // at a deleted ingredient would cost nothing at all, which does not look
      // like an error — the dish simply becomes cheaper to make, on every
      // report, until somebody notices.
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">{t.ingredients.title}</h1>
          <p className="mt-1 text-sm text-ink-soft">
            {t.ingredients.intro}{" "}
            {/* ⚠️ How much of an estimate the expected column is. Without this
              sentence it reads as a stock balance the system has been keeping,
              and somebody orders against it. */}
            {countedAt
              ? t.ingredients.expectedSince(formatDate(countedAt))
              : t.ingredients.expectedNeverCounted}
          </p>
        </div>
        {/* ⚠️ **On this page rather than in settings**, because this is the
            screen somebody is looking at when they realise how much typing is
            in front of them. A migration tool filed under settings is one an
            owner finds after they have entered forty ingredients by hand. */}
        <button
          type="button"
          className="btn-ghost shrink-0 px-4 py-2"
          onClick={() => setImporting(true)}
        >
          {t.posImport.button}
        </button>
      </div>

      {importing && (
        <Modal wide onClose={() => setImporting(false)}>
          <h2 className="mb-4 text-lg font-bold">{t.posImport.title}</h2>
          <PosImport
            onDone={() => {
              load();
              loadWarehouses();
            }}
            onClose={() => setImporting(false)}
          />
        </Modal>
      )}

      {error && <p className="text-sm text-danger">{error}</p>}

      {/* ⚠️ Above the form rather than below the list: a store has to exist
          before an ingredient can be filed in one, and a control that only
          appears after scrolling past two hundred rows is a control nobody
          finds. */}
      <WarehousesEditor onChange={loadWarehouses} />

      <div className="card space-y-2 p-3">
        <div className="flex flex-wrap items-end gap-2">
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.ingredients.name}</span>
            <input
              className="input mt-1 w-56"
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">{t.ingredients.unit}</span>
            <select
              data-help="unit"
              className="input mt-1 w-28"
              value={draft.unit}
              onChange={(e) => setDraft({ ...draft, unit: e.target.value })}
            >
              {UNITS.map((u) => (
                <option key={u} value={u}>
                  {t.ingredients.units[u]}
                </option>
              ))}
            </select>
          </label>
          {/* Where it is kept. ⚠️ Not drawn at all until the restaurant has
              split its store, which is most of them — see WarehousePicker. */}
          <WarehousePicker
            warehouses={warehouses}
            value={draft.warehouseId}
            onChange={(warehouseId) => setDraft({ ...draft, warehouseId })}
            className="w-40"
          />
          {/* ⚠️ A prep item has no price to type: it is what its batch costs.
              Leaving the field on screen would invite a second answer, and the
              stale one always looks the more authoritative. */}
          {draft.recipe.length === 0 && (
            <label className="block text-sm">
              <span className="text-xs text-ink-muted">
                {t.ingredients.pricePer(
                  t.ingredients.units[draft.unit as "kg"],
                )}
              </span>
              <input
                type="number"
                min={0}
                className="input mt-1 w-40"
                value={draft.price || ""}
                onChange={(e) =>
                  setDraft({ ...draft, price: Number(e.target.value) || 0 })
                }
              />
            </label>
          )}
          {/* ⚠️ Opt-in, one ingredient at a time: most never need it, and a
              list where every line eventually turns red is a list nobody
              reads. */}
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.ingredients.minQty}
            </span>
            <QtyInput
              className="input mt-1 w-28"
              value={draft.minQty ?? 0}
              onValue={(v) => setDraft({ ...draft, minQty: qtyNumber(v) })}
            />
          </label>
          {/* ---- How the market sells it ----
              ⚠️ **The gap between how a thing is bought and how it is kept**,
              and it is silent in the worst direction. A market sells mint in
              bunches and flour in sacks; the store counts kilos. A buyer at a
              stall with no scales writes "5" for five bunches into a field
              measured in kilos, five kilos go on the shelf instead of a quarter
              of one, the stop list never fires, and the difference surfaces a
              month later at a count.

              ⚠️ Optional, and empty is the ordinary case: most things are
              bought in the unit they are kept in. */}
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.ingredients.packName}
            </span>
            <input
              className="input mt-1 w-28"
              placeholder={t.ingredients.packNamePh}
              value={draft.packName ?? ""}
              onChange={(e) => setDraft({ ...draft, packName: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            <span className="text-xs text-ink-muted">
              {t.ingredients.packQty(t.ingredients.units[draft.unit as "kg"])}
            </span>
            <QtyInput
              className="input mt-1 w-28"
              value={draft.packQty ?? 0}
              onValue={(v) => setDraft({ ...draft, packQty: qtyNumber(v) })}
            />
          </label>
          <label className="block flex-1 text-sm">
            <span className="text-xs text-ink-muted">{t.ingredients.note}</span>
            <input
              className="input mt-1 w-full"
              placeholder={t.ingredients.notePlaceholder}
              value={draft.note ?? ""}
              onChange={(e) => setDraft({ ...draft, note: e.target.value })}
            />
          </label>
          <button
            className="btn-primary px-4 py-2"
            disabled={busy || !draft.name.trim()}
            onClick={save}
          >
            {draft.id ? t.common.save : t.common.add}
          </button>
          {draft.id && (
            <button
              className="btn-ghost px-3 py-2"
              onClick={() => setDraft(EMPTY)}
            >
              {t.common.cancel}
            </button>
          )}
        </div>
        {/* ---- Made in-house ----

            ⚠️ **This moved out, and finding it was the whole problem.** A prep
            item's card lived here: inside the ingredient form, under a
            collapsed section, on a screen called "the shopping list". It is the
            piece that stops tech cards being abandoned — a kitchen with six
            sauces and forty dishes otherwise lists the same tomatoes in seven
            places — and restaurants did not know it existed. It is now a
            section of its own: Ombor → Texkartalar.

            ⚠️ The row stays, rather than the section disappearing silently:
            somebody who has used this form before will come back looking for
            it here. */}
        <p className="border-t border-line pt-2 text-sm text-ink-muted">
          {t.ingredients.madeMovedHint}{" "}
          <Link href="/admin/tech-cards" className="underline">
            {t.ingredients.madeMovedLink}
          </Link>
        </p>
      </div>

      <div className="card p-0">
        <ListScroll>
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
              <tr>
                <th className="px-3 py-2">{t.ingredients.name}</th>
                <th className="px-3 py-2">{t.ingredients.unit}</th>
                <th className="px-3 py-2 text-right">{t.ingredients.price}</th>
                <th data-help="expected" className="px-3 py-2 text-right">
                  {t.ingredients.expected}
                </th>
                <th className="px-3 py-2">{t.ingredients.note}</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id} className="border-t border-line">
                  <td className="px-3 py-2 font-medium">
                    {row.name}
                    {/* ⚠️ **A half-record, said out loud.** A buyer at a market
                        can add something the catalogue has never heard of —
                        refusing would mean half a delivery never gets recorded
                        at all — but what arrives has a name and a price and
                        nothing else: no unit anybody chose, no minimum, no
                        store, no card. Unmarked it would look finished, and the
                        first dish costed against it would be wrong by whatever
                        "pieces" happens to mean for something sold by the kilo.
                        Saving this row is what clears it. */}
                    {row.needsCare && (
                      <span
                        className="ml-2 rounded-full bg-amber-500/15 px-1.5 text-xs font-semibold text-amber-700 dark:text-amber-300"
                        title={t.ingredients.needsCareHint}
                      >
                        {t.ingredients.needsCare}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-ink-soft">
                    {t.ingredients.units[row.unit as "kg"] ?? row.unit}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {row.made ? (
                      <>
                        <span className="block">
                          {row.unpriced
                            ? t.ingredients.unpriced
                            : formatPrice(row.batchCost ?? 0)}
                        </span>
                        {/* What one batch costs, and what that makes a gram of
                            it — the first is checkable against a pot, the
                            second is what every dish is charged. */}
                        {!row.unpriced && (
                          <span className="block text-xs text-ink-muted">
                            {t.ingredients.madeBadge} · {row.output ?? 0}
                            {t.ingredients.recipeUnits[row.unit as "kg"]}
                          </span>
                        )}
                      </>
                    ) : (
                      formatPrice(row.price)
                    )}
                  </td>
                  <td className="px-3 py-2 text-right tabular-nums">
                    {row.expected !== undefined ? row.expected : "—"}
                    {/* ⚠️ Named, not coloured alone: "tugayapti" tells
                        somebody what to do, a red number tells them something
                        is wrong and leaves them to work out what. */}
                    {row.low && (
                      <span className="ml-1 text-xs text-danger">
                        {t.ingredients.low}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-ink-muted">
                    {row.note}
                    {/* ⚠️ When the price last moved, and what it was before.
                        Without it an owner reading a margin has no way to ask
                        "since when" — and that is the first question after
                        "why is this dish worse than last month". */}
                    {row.history && row.history.length > 1 && (
                      <span className="block text-xs">
                        {t.ingredients.since(
                          formatDate(row.history[row.history.length - 1].at),
                          formatPrice(
                            row.history[row.history.length - 2].price,
                          ),
                        )}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right whitespace-nowrap">
                    <button
                      className="btn-ghost px-2 py-1 text-xs"
                      onClick={() =>
                        setDraft({
                          ...row,
                          recipe: row.recipe ?? [],
                          output: row.output ?? 0,
                          minQty: row.minQty ?? 0,
                          // ⚠️ Defaulted here as well as in EMPTY: an
                          // ingredient saved before warehouses existed has no
                          // field at all, and `undefined` in a controlled
                          // select is React switching it to uncontrolled
                          // halfway through an edit.
                          warehouseId: row.warehouseId ?? "",
                        })
                      }
                    >
                      {t.common.edit}
                    </button>
                    <button
                      className="btn-ghost px-2 py-1 text-xs text-danger"
                      disabled={busy}
                      onClick={() => remove(row)}
                    >
                      {t.common.delete}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!loading && rows.length === 0 && (
            <p className="p-6 text-center text-sm text-ink-muted">
              {t.ingredients.empty}
            </p>
          )}
        </ListScroll>
      </div>
    </div>
  );
}
