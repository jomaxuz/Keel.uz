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

import { api, ApiError, imageUrl } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import { useAdminScope } from "@/lib/adminScope";
import { ListScroll } from "@/components/admin/PagedList";
import Modal from "@/components/admin/Modal";
import RecipeEditor, { ratePerUnit } from "@/components/admin/RecipeEditor";
import TechCardPrint from "@/components/admin/TechCardPrint";
import type { CardData } from "@/lib/techCardPng";
import type {
  Ingredient,
  MenuItem,
  RecipeLine,
  StockCoverage,
  StockCoverageRow,
} from "@/lib/types";
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
  const [coverage, setCoverage] = useState<StockCoverage | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const [tab, setTab] = useState<"preps" | "dishes">("preps");
  const [prep, setPrep] = useState<PrepDraft | null>(null);
  const [dish, setDish] = useState<MenuItem | null>(null);
  const [dishLines, setDishLines] = useState<RecipeLine[]>([]);
  /** The card being turned into a sheet of paper, if any. */
  const [printing, setPrinting] = useState<CardData | null>(null);
  /** The name and logo the sheet is headed with. ⚠️ The brand's, not the
   *  branch's: a technical card belongs to the catalogue, and the catalogue is
   *  the brand's — the same rule the cards themselves already follow. */
  const [brand, setBrand] = useState<{ name: string; logo?: string }>({
    name: "",
  });
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
    // ⚠️ Its own request, and its failure is not this screen's failure. The
    // cards can be written without the coverage figure; the figure without the
    // cards is nothing. A shared `Promise.all` would have let a slow month of
    // orders leave somebody unable to type a recipe.
    api
      .adminStockCoverage()
      .then(setCoverage)
      .catch(() => setCoverage(null));
    // The name and logo the printed sheet is headed with. ⚠️ Its own request
    // and its own failure, for the reason above: a card can be written and
    // printed without a logo, and a heading nobody can fetch must not be the
    // reason a recipe cannot be typed. The brand's, not the branch's — a
    // technical card belongs to the catalogue.
    api
      .getRestaurant({ raw: true })
      .then((r) =>
        setBrand({
          // ⚠️ The brand's face first, the company's underneath — the same
          // fallback the settings screen edits through. A chain whose second
          // brand has never been given a logo still gets a headed sheet.
          name: scope.brand?.name || r.restaurant.name || "",
          logo: scope.brand?.logoUrl || r.restaurant.logoUrl || "",
        }),
      )
      .catch(() => setBrand({ name: scope.brand?.name ?? "" }));
    // ⚠️ **The brand is a dependency now.** This closure reads which brand is
    // selected, and an empty list would leave a chain's second brand printing
    // the first one's name on its sheets — the exact confusion the lens exists
    // to prevent, on the one artefact that leaves the building.
  }, [scope.brand]);

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
  /** What each dish sold while it had no working card. ⚠️ Keyed by id, not by
   *  name: the coverage report and the menu agree on ids and nothing else — a
   *  renamed dish would otherwise silently drop out of the ordering. */
  const gaps = useMemo(() => {
    const m = new Map<string, StockCoverageRow>();
    coverage?.rows.forEach((row) => m.set(row.id, row));
    return m;
  }, [coverage]);
  const shownDishes = useMemo(() => {
    const q = search.trim().toLowerCase();
    let list = noCardOnly ? uncosted : cardable;
    if (q) list = list.filter((d) => d.name.toLowerCase().includes(q));
    // ⚠️ **By what it sold, and only inside the filter.** Alphabetically this
    // is two hundred names and a job with no end; by money the first ten lines
    // are most of the answer. Left alone outside the filter, where the reader
    // came looking for a particular dish and expects to find it where it was.
    if (!noCardOnly) return list;
    return [...list].sort((a, b) => {
      const d = (gaps.get(b.id)?.revenue ?? 0) - (gaps.get(a.id)?.revenue ?? 0);
      return d !== 0 ? d : a.name.localeCompare(b.name);
    });
  }, [cardable, uncosted, noCardOnly, search, gaps]);

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

  /** How much one portion weighs, adding only the lines measured by weight or
   *  volume.
   *
   *  ⚠️ **Pieces are left out rather than added in.** Two eggs and 300 g of
   *  flour is not 302 of anything, and a smell test that silently mixed the
   *  units would fire on every dish with an egg in it. */
  function portionWeight(lines: RecipeLine[]): number {
    let total = 0;
    for (const l of lines) {
      const ing = ingredients.find((i) => i.id === l.ingredientId);
      if (ing && (ing.unit === "kg" || ing.unit === "l")) total += l.qty;
    }
    return total;
  }

  /** Turn a card into the shape the sheet is drawn from.
   *
   *  ⚠️ **Rates come from `ratePerUnit`, the rule the editor already prices
   *  with.** A second implementation here is how a printed card and a report
   *  end up disagreeing about one dish, on paper, in front of a supplier. */
  function sheetOf(
    name: string,
    lines: RecipeLine[],
    price?: number,
    yieldText?: string,
  ): CardData {
    const rows = lines
      .map((l) => {
        const ing = ingredients.find((i) => i.id === l.ingredientId);
        if (!ing) return null;
        return {
          name: ing.name,
          qty: l.qty,
          unit: t.ingredients.recipeUnits[ing.unit as "kg"] ?? "",
          rate: ratePerUnit(ing),
          // ⚠️ Carried, never computed here: the netto a cook weighs is the
          // ingredient's own waste applied to a brutto figure, and a second
          // opinion about peel on the printing screen is how one card says
          // 120 g and the next says 130.
          waste: ing.waste ?? 0,
        };
      })
      .filter(Boolean) as CardData["lines"];
    return {
      brand: brand.name,
      // ⚠️ 600 wide: the sheet draws it at 16 mm, and at 300 dpi that is
      // nearly 200 pixels — a thumbnail scaled up prints as a blur on the
      // one document somebody hands to a supplier.
      logoUrl: imageUrl(brand.logo, 600) ?? undefined,
      dish: name,
      yield: yieldText,
      lines: rows,
      // ⚠️ Rounded once over the whole card, exactly as the server does it —
      // rounding each line would print a total no report agrees with.
      cost: Math.round(rows.reduce((sum, r) => sum + r.rate * r.qty, 0)),
      price,
    };
  }

  async function saveDish() {
    if (!dish) return;
    // ⚠️ **The one card mistake nothing downstream can see.** A kilo of beef
    // written as `1.5` instead of `1500` makes the card say a gram and a half:
    // the dish costs three hundred so'm, the store takes nothing off the shelf,
    // and every screen reports it happily — a low cost reads as a good margin
    // and a shelf that never moves reads as a shelf nobody is stealing from.
    // The opposite slip is louder but no more visible.
    //
    // ⚠️ **Asked, never refused.** A tasting portion really is two grams and a
    // catering tray really is four kilos, and a form that blocked either would
    // be a form somebody works around by leaving the card empty — which is the
    // state this whole screen exists to end.
    const weight = portionWeight(dishLines);
    if (weight > 0 && (weight < 20 || weight > 3000)) {
      const ok = await ask({
        title: t.techCards.weightOdd(Math.round(weight)),
        body: t.techCards.weightOddHint,
        confirmLabel: t.common.save,
      });
      if (!ok) return;
    }
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
          {/* ⚠️ **A measurement with its period, not a banner.** The rule this
              screen already followed — "a filter, never a warning" — was right
              about the wrong question: "some dishes have no card" is true in
              every restaurant forever. What share of the *money* left the shelf
              untraceably is a number that shrinks as the cards that matter get
              written, and a number that shrinks is one somebody comes back to.
              Shown beside the days it measures, the same way the stock balance
              is shown beside the date it is counted from. */}
          {coverage && coverage.soldTotal > 0 && (
            <div className="max-w-2xl rounded-xl bg-ink/[0.04] p-3 text-sm">
              <p className="font-medium">
                {t.techCards.coverage(coverage.share)}
              </p>
              {coverage.rows.length > 0 ? (
                <p className="mt-1 text-ink-soft">
                  {t.techCards.coverageGap(
                    100 - coverage.share,
                    coverage.rows.length,
                  )}{" "}
                  {!noCardOnly && (
                    <button
                      className="underline underline-offset-2"
                      onClick={() => setNoCardOnly(true)}
                    >
                      {t.techCards.coverageOpen}
                    </button>
                  )}
                </p>
              ) : (
                <p className="mt-1 text-ink-soft">
                  {t.techCards.coverageFull}
                </p>
              )}
              {/* ⚠️ Preps named separately, because one of them is usually the
                  cause of a dozen of those dishes: a sushi rice with no yield
                  makes every roll uncovered, and a reader given only the dish
                  list would card twelve rolls and fix nothing. */}
              {coverage.preps.length > 0 && (
                <p className="mt-1 text-ink-muted">
                  {t.techCards.coveragePreps(coverage.preps.length)}{" "}
                  {coverage.preps
                    .map(
                      (p) =>
                        `${p.name} (${
                          p.issue === "no_output"
                            ? t.techCards.coveragePrepNoOutput
                            : t.techCards.coveragePrepIncomplete
                        })`,
                    )
                    .join(", ")}
                </p>
              )}
              {/* ⚠️ **The off-switch sits beside the thing it silences**, which
                  is the rule the alert bell already follows: every noise the
                  panel makes has exactly one stop button, and it is where the
                  noise is. A reminder whose setting lives three screens away in
                  "Sozlamalar" is silenced by people avoiding this screen — and
                  then the measurement is gone along with the reminder. */}
              <label className="mt-2 flex items-center gap-2 text-xs text-ink-muted">
                <input
                  type="checkbox"
                  checked={coverage.warnOff}
                  onChange={(e) => {
                    const off = e.target.checked;
                    setCoverage({ ...coverage, warnOff: off });
                    api.setStockCardWarn({ off }).catch(() => load());
                  }}
                />
                {t.techCards.coverageMute(coverage.warnFrom)}
              </label>
            </div>
          )}
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
                    {/* ⚠️ Only where it sorts. A revenue-ordered list with no
                        revenue column looks arbitrary, and a reader who cannot
                        see why a row is first does not trust that it should
                        be. */}
                    {noCardOnly && (
                      <th className="px-3 py-2 text-right">
                        {t.techCards.coverageSold}
                      </th>
                    )}
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
                    const gap = gaps.get(row.id);
                    return (
                      <tr key={row.id} className="border-t border-line">
                        <td className="px-3 py-2 font-medium">
                          {row.name}
                          {/* A broken card is not a missing one, and the fix is
                              different: somebody wrote this and a deleted
                              ingredient took it apart. */}
                          {gap?.issue === "partial" && (
                            <span className="ml-1.5 text-xs text-danger">
                              {t.techCards.coverageBroken}
                            </span>
                          )}
                          {gap?.via && (
                            <span className="ml-1.5 text-xs text-ink-muted">
                              {t.techCards.coverageVia(gap.via)}
                            </span>
                          )}
                        </td>
                        {noCardOnly && (
                          <td className="px-3 py-2 text-right tabular-nums">
                            {gap ? (
                              <span title={`${gap.qty}`}>
                                {formatPrice(gap.revenue)}
                              </span>
                            ) : (
                              <span className="text-ink-muted">—</span>
                            )}
                          </td>
                        )}
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
            {prep.recipe.length > 0 && (
              <button
                className="btn-ghost mr-auto px-4 py-2 text-sm"
                onClick={() =>
                  setPrinting(
                    sheetOf(
                      prep.name,
                      prep.recipe,
                      undefined,
                      prep.output > 0
                        ? `${prep.output} ${t.ingredients.recipeUnits[prep.unit as "kg"] ?? ""}`
                        : undefined,
                    ),
                  )
                }
              >
                {t.techCards.printCard}
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
            {/* ⚠️ Only once there is something to print. An empty sheet with a
                logo on it is a document that says a dish has no recipe. */}
            {dishLines.length > 0 && (
              <button
                className="btn-ghost mr-auto px-4 py-2 text-sm"
                onClick={() =>
                  setPrinting(sheetOf(dish.name, dishLines, dish.price))
                }
              >
                {t.techCards.printCard}
              </button>
            )}
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
      {printing && (
        <TechCardPrint data={printing} onClose={() => setPrinting(null)} />
      )}
    </div>
  );
}
