"use client";

// Tech cards: what is made from what.
//
// ⚠️ **The card used to be edited on the dish, and half the cards had nowhere
// to live.** A prep item — a sauce, a dough, sushi rice — is not on the menu,
// so its card was hidden inside the ingredient form, under a collapsed section,
// on a screen called "the shopping list". Restaurants did not find it and wrote
// the rice into forty dishes by hand instead; those forty copies stop agreeing
// within a month, and the month they stop agreeing is the month the costing
// becomes fiction.
//
// ⚠️ **Two kinds of card on one screen, and the screen has to keep them
// apart.** A prep card describes a pot — what one batch takes, and how much
// comes out of it. A dish card describes a plate — what one portion takes. The
// yield is the difference and it only exists on the first: written on a dish it
// would mean nothing, forgotten on a sauce it means the sauce costs nothing.
//
// ⚠️ **Nothing here computes a cost the server has not already computed.** Both
// lists arrive priced — dishes from `/admin/menu`, preps from
// `/admin/ingredients` — because a prep's rate depends on every other rate, and
// a browser resolving that chain would be the second implementation of costing
// that the cards exist to end.

import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import Modal from "@/components/admin/Modal";
import RecipeEditor from "@/components/admin/RecipeEditor";
import type { Ingredient, MenuItem, RecipeLine } from "@/lib/types";
import { qtyNumber } from "@/lib/qty";
import { QtyInput } from "@/components/QtyInput";

const UNITS = ["kg", "l", "pcs"] as const;

/** What a card's quantities are measured in — grams for a kilo, millilitres for
 *  a litre. Exactly as the server has it, and as RecipeEditor draws it. */
function recipeUnit(unit: string): string {
  if (unit === "kg") return "g";
  if (unit === "l") return "ml";
  return "pcs";
}

type PrepDraft = {
  /** Empty on a new one. */
  id: string;
  name: string;
  unit: string;
  recipe: RecipeLine[];
  output: number;
  batched: boolean;
};

const EMPTY_PREP: PrepDraft = {
  id: "",
  name: "",
  unit: "kg",
  recipe: [],
  output: 0,
  batched: false,
};

export default function TechCardsPage() {
  const t = useAdminT();
  const { ask } = useAsk();
  const scope = useAdminScope();
  const params = useSearchParams();

  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [dishes, setDishes] = useState<MenuItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const [tab, setTab] = useState<"preps" | "dishes">("preps");
  const [prep, setPrep] = useState<PrepDraft | null>(null);
  const [dish, setDish] = useState<MenuItem | null>(null);
  const [dishLines, setDishLines] = useState<RecipeLine[]>([]);
  const [search, setSearch] = useState("");
  const [noCardOnly, setNoCardOnly] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([api.adminIngredients(), api.adminMenu()])
      .then(([ing, menu]) => {
        setIngredients(ing.ingredients);
        setDishes(menu);
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : String(e)))
      .finally(() => setLoading(false));
  }, []);

  // ⚠️ Re-read when the lens moves: the catalogue belongs to the brand, and a
  // chain switching brands with a stale list would offer one brand's sauces
  // inside the other brand's dish.
  useEffect(load, [load, scope.scopeKey]);

  // Arriving from the menu screen's "edit the card" link. ⚠️ Opened once the
  // dishes are in: the link carries an id, not a dish.
  const wanted = params.get("dish");
  useEffect(() => {
    if (!wanted || dishes.length === 0) return;
    const found = dishes.find((d) => d.id === wanted);
    if (found) {
      setTab("dishes");
      setDish(found);
      setDishLines(found.recipe ?? []);
    }
  }, [wanted, dishes]);

  const preps = useMemo(() => ingredients.filter((i) => i.made), [ingredients]);

  // ⚠️ A combo never appears: it has no card of its own, and one written here
  // would be counted **beside** its members rather than instead of them.
  const cardable = useMemo(
    () => dishes.filter((d) => !d.comboContents?.length),
    [dishes],
  );
  const uncosted = useMemo(
    () => cardable.filter((d) => !d.recipe?.length),
    [cardable],
  );
  const shownDishes = useMemo(() => {
    const q = search.trim().toLowerCase();
    let list = noCardOnly ? uncosted : cardable;
    if (q) list = list.filter((d) => d.name.toLowerCase().includes(q));
    return list;
  }, [cardable, uncosted, noCardOnly, search]);

  async function savePrep() {
    if (!prep || !prep.name.trim()) return;
    setBusy(true);
    setError("");
    try {
      // ⚠️ The whole ingredient goes back, because saving one replaces the
      // document: the row is merged into rather than posted from the draft, or
      // an edit made here would clear the minimum, the note and the store
      // somebody filed it in on the ingredients screen.
      const existing = ingredients.find((i) => i.id === prep.id);
      await api.adminSaveIngredient({
        ...(existing ?? {}),
        id: prep.id || undefined,
        name: prep.name.trim(),
        unit: prep.unit,
        recipe: prep.recipe,
        output: prep.output,
        batched: prep.batched,
      });
      setPrep(null);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function removePrep(row: Ingredient) {
    if (!(await ask({ title: `${row.name}?`, danger: true }))) return;
    setBusy(true);
    setError("");
    try {
      await api.adminDeleteIngredient(row.id);
      setPrep(null);
      load();
    } catch (e) {
      // ⚠️ The refusal names the dishes that still hold it: a card pointing at
      // a deleted ingredient stops costing its dish, and the dish looks
      // cheaper rather than broken.
      setError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveDish() {
    if (!dish) return;
    setBusy(true);
    setError("");
    try {
      const saved = await api.saveDishRecipe(dish.id, dishLines);
      setDishes((list) =>
        list.map((d) => (d.id === saved.id ? { ...d, ...saved } : d)),
      );
      setDish(null);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const tabCls = (on: boolean) =>
    `rounded-full px-4 py-1.5 text-sm ${
      on ? "bg-ink text-surface" : "bg-ink/[0.05] text-ink-soft"
    }`;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.techCards.title}</h1>
        <p className="mt-1 max-w-3xl text-sm text-ink-muted">
          {t.techCards.intro}
        </p>
      </div>

      {error && (
        <p className="rounded-xl bg-danger/10 px-4 py-2 text-sm text-danger">
          {error}
        </p>
      )}

      <div className="flex gap-2">
        <button
          data-help="tabs"
          className={tabCls(tab === "preps")}
          onClick={() => setTab("preps")}
        >
          {t.techCards.preps} · {preps.length}
        </button>
        <button
          data-help="dishes"
          className={tabCls(tab === "dishes")}
          onClick={() => setTab("dishes")}
        >
          {t.techCards.dishes} · {cardable.length}
        </button>
      </div>

      {tab === "preps" ? (
        <div className="space-y-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <p className="max-w-2xl text-sm text-ink-muted">
              {t.techCards.prepsHint}
            </p>
            <button
              data-help="add"
              className="btn-primary px-4 py-2 text-sm"
              onClick={() => setPrep({ ...EMPTY_PREP })}
            >
              {t.techCards.newPrep}
            </button>
          </div>

          <div className="card p-0">
            <ListScroll>
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
                  <tr>
                    <th className="px-3 py-2">{t.techCards.prepName}</th>
                    <th className="px-3 py-2">{t.recipe.title}</th>
                    <th data-help="batch" className="px-3 py-2 text-right">
                      {t.techCards.batchCost}
                    </th>
                    <th data-help="rate" className="px-3 py-2 text-right">
                      {t.ingredients.price}
                    </th>
                    <th className="px-3 py-2" />
                  </tr>
                </thead>
                <tbody>
                  {preps.map((row) => (
                    <tr key={row.id} className="border-t border-line">
                      <td className="px-3 py-2 font-medium">
                        {row.name}
                        {row.batched && (
                          <span className="ml-2 rounded-full bg-ink/[0.06] px-2 py-0.5 text-[11px] text-ink-soft">
                            {t.ingredients.batched}
                          </span>
                        )}
                      </td>
                      {/* How many lines, and what the batch yields — the two
                          facts that say whether the card is finished. */}
                      <td className="px-3 py-2 text-ink-soft">
                        {row.recipe?.length ?? 0} · {row.output ?? 0}
                        {t.ingredients.recipeUnits[row.unit as "kg"]}
                      </td>
                      <td className="px-3 py-2 text-right tabular-nums">
                        {/* ⚠️ Named rather than shown as zero: a prep whose own
                            inputs are unpriced has no rate, and a zero here
                            would make every dish containing it look cheap. */}
                        {row.unpriced
                          ? t.ingredients.unpriced
                          : formatPrice(row.batchCost ?? 0)}
                      </td>
                      <td className="px-3 py-2 text-right text-xs text-ink-muted tabular-nums">
                        {row.unpriced
                          ? "—"
                          : `${formatPrice(Math.round(row.rate ?? 0))} / ${
                              t.ingredients.recipeUnits[row.unit as "kg"]
                            }`}
                      </td>
                      <td className="px-3 py-2 text-right whitespace-nowrap">
                        <button
                          className="btn-ghost px-2 py-1 text-xs"
                          onClick={() =>
                            setPrep({
                              id: row.id,
                              name: row.name,
                              unit: row.unit,
                              recipe: row.recipe ?? [],
                              output: row.output ?? 0,
                              batched: !!row.batched,
                            })
                          }
                        >
                          {t.common.edit}
                        </button>
                        <button
                          className="btn-ghost px-2 py-1 text-xs text-danger"
                          disabled={busy}
                          onClick={() => removePrep(row)}
                        >
                          {t.common.delete}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {!loading && preps.length === 0 && (
                <div className="p-8 text-center">
                  <p className="text-sm text-ink-muted">
                    {t.techCards.noPreps}
                  </p>
                  <p className="mx-auto mt-1 max-w-md text-xs text-ink-muted">
                    {t.techCards.example}
                  </p>
                </div>
              )}
            </ListScroll>
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          <p className="max-w-2xl text-sm text-ink-muted">
            {t.techCards.dishesHint}
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <input
              className="input w-64"
              placeholder={t.techCards.searchDish}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
            {/* ⚠️ A filter, never a warning. Most of a menu is legitimately
                uncosted — a restaurant cards the ten dishes that matter first
                — and a banner that is always on is one nobody reads. */}
            {uncosted.length > 0 && uncosted.length < cardable.length && (
              <button
                className={`rounded-full px-3 py-1.5 text-xs ${
                  noCardOnly
                    ? "bg-ink text-surface"
                    : "bg-ink/[0.05] text-ink-soft"
                }`}
                onClick={() => setNoCardOnly(!noCardOnly)}
              >
                {noCardOnly
                  ? t.techCards.showAll
                  : t.techCards.noCardOnly(uncosted.length)}
              </button>
            )}
          </div>

          <div className="card p-0">
            <ListScroll>
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-surface text-left text-xs text-ink-muted">
                  <tr>
                    <th className="px-3 py-2">{t.menu.name}</th>
                    <th className="px-3 py-2 text-right">{t.menu.price}</th>
                    <th className="px-3 py-2 text-right">
                      {t.techCards.cardCost}
                    </th>
                    <th className="px-3 py-2 text-right">{t.recipe.margin}</th>
                    <th className="px-3 py-2" />
                  </tr>
                </thead>
                <tbody>
                  {shownDishes.map((row) => {
                    const cost = row.recipeCost ?? 0;
                    return (
                      <tr key={row.id} className="border-t border-line">
                        <td className="px-3 py-2 font-medium">{row.name}</td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {formatPrice(row.price)}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {cost > 0 ? (
                            formatPrice(cost)
                          ) : row.cost ? (
                            // ⚠️ The typed figure is shown as what it is, and
                            // named: it is the one number here with two
                            // possible sources, and the card wins the moment
                            // there is one.
                            <span
                              className="text-ink-muted"
                              title={t.techCards.manualCostHint}
                            >
                              {formatPrice(row.cost)} · {t.techCards.manualCost}
                            </span>
                          ) : (
                            <span className="text-ink-muted">
                              {t.techCards.cardMissing}
                            </span>
                          )}
                        </td>
                        <td className="px-3 py-2 text-right tabular-nums">
                          {cost > 0 && row.price > 0 ? (
                            <span
                              className={row.price <= cost ? "text-danger" : ""}
                            >
                              {row.price > cost
                                ? `${Math.round(((row.price - cost) / row.price) * 100)}%`
                                : t.menu.belowCost}
                            </span>
                          ) : (
                            <span className="text-ink-muted">—</span>
                          )}
                        </td>
                        <td className="px-3 py-2 text-right whitespace-nowrap">
                          <button
                            className="btn-ghost px-2 py-1 text-xs"
                            onClick={() => {
                              setDish(row);
                              setDishLines(row.recipe ?? []);
                            }}
                          >
                            {row.recipe?.length
                              ? t.common.edit
                              : t.recipe.addLine}
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              {!loading && shownDishes.length === 0 && (
                <p className="p-6 text-center text-sm text-ink-muted">
                  {t.techCards.noDishes}
                </p>
              )}
            </ListScroll>
          </div>
        </div>
      )}

      {/* ---- One prep ---- */}
      {prep && (
        <Modal wide onClose={() => setPrep(null)}>
          <h2 className="text-lg font-semibold">
            {prep.id ? t.techCards.editPrep : t.techCards.newPrep}
          </h2>
          <p className="mt-1 text-xs text-ink-muted">{t.techCards.example}</p>

          <div className="mt-4 flex flex-wrap items-end gap-3">
            <label className="block flex-1 text-sm">
              <span className="text-xs text-ink-muted">
                {t.techCards.prepName}
              </span>
              <input
                className="input mt-1 w-full"
                placeholder={t.techCards.prepNamePh}
                value={prep.name}
                onChange={(e) => setPrep({ ...prep, name: e.target.value })}
              />
            </label>
            <label className="block text-sm">
              <span className="text-xs text-ink-muted">
                {t.techCards.prepUnit}
              </span>
              <select
                className="input mt-1 w-28"
                value={prep.unit}
                onChange={(e) => setPrep({ ...prep, unit: e.target.value })}
              >
                {UNITS.map((u) => (
                  <option key={u} value={u}>
                    {t.ingredients.units[u]}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <p className="mt-1 text-xs text-ink-muted">
            {t.techCards.prepUnitHint}
          </p>

          <div className="mt-4">
            <span className="text-sm font-medium">{t.recipe.title}</span>
            <div className="mt-2">
              {/* ⚠️ A prep cannot contain itself, and the list it picks from
                  says so. Two cards naming each other are not refused — the
                  server resolves in passes and leaves an unresolvable pair
                  unpriced — but the one loop a form can prevent, it should. */}
              <RecipeEditor
                lines={prep.recipe}
                ingredients={ingredients.filter((i) => i.id !== prep.id)}
                price={0}
                onChange={(recipe) => setPrep({ ...prep, recipe })}
              />
            </div>
          </div>

          <label className="mt-4 block text-sm">
            <span className="font-medium">
              {t.techCards.output(
                t.ingredients.recipeUnits[prep.unit as "kg"] ?? "",
              )}
            </span>
            <QtyInput
              className="input mt-1 w-40"
              value={prep.output}
              onValue={(v) => setPrep({ ...prep, output: qtyNumber(v) })}
            />
            <span className="mt-1 block text-xs text-ink-muted">
              {t.techCards.outputHint}
            </span>
          </label>

          {/* ⚠️ Said out loud rather than left to the cost column, because a
              card with no yield is not "nearly finished": it prices at nothing,
              and so does every dish that uses it. */}
          {prep.recipe.length > 0 && prep.output <= 0 && (
            <p className="mt-2 text-xs text-danger">
              {t.techCards.outputNeeded}
            </p>
          )}

          <label className="mt-4 flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={prep.batched}
              onChange={(e) => setPrep({ ...prep, batched: e.target.checked })}
            />
            <span>
              <span className="font-medium">{t.ingredients.batched}</span>
              <span className="mt-1 block text-xs text-ink-muted">
                {t.ingredients.batchedHint}
              </span>
            </span>
          </label>

          <div className="mt-5 flex justify-end gap-2">
            {prep.id && (
              <button
                className="btn-ghost mr-auto px-3 py-2 text-sm text-danger"
                disabled={busy}
                onClick={() => {
                  const row = ingredients.find((i) => i.id === prep.id);
                  if (row) removePrep(row);
                }}
              >
                {t.common.delete}
              </button>
            )}
            <button
              className="btn-ghost px-4 py-2 text-sm"
              onClick={() => setPrep(null)}
            >
              {t.common.cancel}
            </button>
            <button
              className="btn-primary px-4 py-2 text-sm"
              disabled={busy || !prep.name.trim()}
              onClick={savePrep}
            >
              {t.common.save}
            </button>
          </div>
        </Modal>
      )}

      {/* ---- One dish ---- */}
      {dish && (
        <Modal wide onClose={() => setDish(null)}>
          <h2 className="text-lg font-semibold">{dish.name}</h2>
          <p className="mt-1 text-xs text-ink-muted">{t.recipe.hint}</p>
          <div className="mt-4">
            <RecipeEditor
              lines={dishLines}
              ingredients={ingredients}
              price={dish.price}
              onChange={setDishLines}
            />
          </div>
          <div className="mt-5 flex justify-end gap-2">
            <button
              className="btn-ghost px-4 py-2 text-sm"
              onClick={() => setDish(null)}
            >
              {t.common.cancel}
            </button>
            <button
              className="btn-primary px-4 py-2 text-sm"
              disabled={busy}
              onClick={saveDish}
            >
              {t.common.save}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}
