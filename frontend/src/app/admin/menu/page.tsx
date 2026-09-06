"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { hasId } from "@/lib/id";
import ImageUpload from "@/components/admin/ImageUpload";
import Modal from "@/components/admin/Modal";
import { ListScroll } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import MenuImport from "@/components/admin/MenuImport";
import OptionsEditor, {
  optionProblems,
  fromOptionDrafts,
  toOptionDrafts,
  type OptionGroupDraft,
} from "@/components/admin/OptionsEditor";
import ComboEditor from "@/components/admin/ComboEditor";
import RecommendEditor from "@/components/admin/RecommendEditor";
import type { Category, ComboLine, Ingredient, MenuItem } from "@/lib/types";
import { composes, hasKitchen, hasVariants, sellsGoods } from "@/lib/types";
import VariantsEditor from "@/components/admin/VariantsEditor";
import { useAsk } from "@/components/ui/Ask";

// Editable form shape: prices/oldPrice kept as strings for controlled inputs.
interface Draft {
  id: string;
  categoryId: string;
  name: string;
  description: string;
  nameRu: string;
  nameEn: string;
  descriptionRu: string;
  descriptionEn: string;
  price: string;
  oldPrice: string;
  cost: string;
  /** The card, read-only on this form: how many lines it has and what it works
   *  out to. ⚠️ Not the lines themselves — this form must not be able to post
   *  a card back, or an old tab would revert an edit made on the card screen. */
  recipeLines: number;
  recipeCost: number;
  imageUrl: string;
  isAvailable: boolean;
  isPopular: boolean;
  sortOrder: number;
  tags: string;
  ikpu: string;
  marked: boolean;
  packageCode: string;
  /** Empty means "use the branch's rate", which is what almost every dish
   *  wants. ⚠️ Kept as a string so that "0" and "" stay distinguishable —
   *  a number field would collapse an explicit zero-rating into "unset". */
  vatPercent: string;
  unitCode: number;
  barcode: string;
  sellsItself: boolean;
  /** Which parts of a portion this dish may be sold in, as percents. Empty is
   *  "whole portions only" — every dish, until somebody says otherwise. */
  portions: number[];
  options: OptionGroupDraft[];
  comboItems: ComboLine[];
  /** Dishes to suggest alongside this one, in the owner's own order. Empty is
   *  the normal state and means "work it out from the order history". */
  recommendedIds: string[];
  // Which editor this form is showing.
  //
  // ⚠️ Kept as its own field rather than read off `comboItems.length`, which is
  // what it used to be: a set being built has no dishes in it yet, so the
  // length said "dish", the form kept showing the options editor, and pressing
  // "To'plam" did nothing at all. A mode is a question about the form, and it
  // cannot be answered by data that only exists once the form has been filled.
  kind: "dish" | "combo";
}

function toDraft(m: MenuItem): Draft {
  return {
    id: m.id,
    categoryId: m.categoryId,
    name: m.name,
    description: m.description,
    nameRu: m.nameRu ?? "",
    nameEn: m.nameEn ?? "",
    descriptionRu: m.descriptionRu ?? "",
    descriptionEn: m.descriptionEn ?? "",
    price: String(m.price),
    oldPrice: m.oldPrice != null ? String(m.oldPrice) : "",
    cost: m.cost ? String(m.cost) : "",
    recipeLines: m.recipe?.length ?? 0,
    recipeCost: m.recipeCost ?? 0,
    imageUrl: m.imageUrl,
    isAvailable: m.isAvailable,
    isPopular: m.isPopular,
    sortOrder: m.sortOrder,
    tags: (m.tags ?? []).join(", "),
    ikpu: m.ikpu ?? "",
    marked: !!m.marked,
    packageCode: m.packageCode ?? "",
    // ⚠️ `== null`, not truthiness: an explicit 0 is a zero-rated dish and
    // must come back into the form as "0", not as an empty field.
    vatPercent: m.vatPercent == null ? "" : String(m.vatPercent),
    unitCode: m.unitCode ?? 0,
    barcode: m.barcode ?? "",
    sellsItself: m.sellsItself ?? false,
    portions: m.portions ?? [],
    options: toOptionDrafts(m.options),
    comboItems: m.comboItems ?? [],
    recommendedIds: m.recommendedIds ?? [],
    kind: (m.comboItems ?? []).length > 0 ? "combo" : "dish",
  };
}

function emptyDraft(categoryId: string): Draft {
  return {
    id: "",
    categoryId,
    name: "",
    description: "",
    nameRu: "",
    nameEn: "",
    descriptionRu: "",
    descriptionEn: "",
    price: "",
    oldPrice: "",
    cost: "",
    recipeLines: 0,
    recipeCost: 0,
    imageUrl: "",
    isAvailable: true,
    isPopular: false,
    sortOrder: 0,
    tags: "",
    ikpu: "",
    marked: false,
    packageCode: "",
    vatPercent: "",
    unitCode: 0,
    barcode: "",
    sellsItself: false,
    portions: [],
    options: [],
    comboItems: [],
    recommendedIds: [],
    kind: "dish",
  };
}

export default function AdminMenuPage() {
  const [cats, setCats] = useState<Category[]>([]);
  const { ask, tell } = useAsk();
  // The shopping list. ⚠️ Still read here after the card moved out: an option
  // group can pour from the store too (a double shot is a tech card hanging off
  // a choice), and that editor needs the same list.
  const [ingredients, setIngredients] = useState<Ingredient[]>([]);
  const [items, setItems] = useState<MenuItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [saving, setSaving] = useState(false);
  const [importing, setImporting] = useState(false);
  const t = useAdminT();
  const scope = useAdminScope();
  // What has run out **at this branch** today. The menu itself belongs to the
  // brand, so this is the one thing on this page that is not shared.
  const branch = scope.branch;
  // ⚠️ **Whether this brand's catalogue is goods or dishes.** It decides which
  // fields the form offers, not what may be saved — the server has no such
  // rule, and a restaurant that starts stocking bottled water only has to be
  // told what its type is.
  const brandSellsGoods = sellsGoods(scope.brand);
  // ⚠️ Whether anything here is cooked to order. Separate from the question
  // above, because a fast food is neither: it has a kitchen and sells no goods.
  const brandHasKitchen = hasKitchen(scope.brand);
  // ⚠️ The tech card follows composition, not cooking — see types.ts.
  const brandComposes = composes(scope.brand);
  // ⚠️ **The data wins over the type.** A dish that already carries a barcode
  // keeps its fields visible whatever kind of business this is — otherwise the
  // value is saved on every edit and can never be seen or removed, which is the
  // one outcome worse than an irrelevant field.
  const showGoods =
    brandSellsGoods || !!draft?.barcode || draft?.sellsItself === true;
  const [soldOut, setSoldOut] = useState<Set<string>>(new Set());
  // ⚠️ Which dishes still have no cost. The reports say "cost is set on 1 of 19
  // dishes"; the owner then has to find the other eighteen, and until this
  // existed the only way was to open every dish in turn — which is how a
  // half-costed menu stays half-costed.
  const [uncostedOnly, setUncostedOnly] = useState(false);
  useEffect(() => {
    setSoldOut(new Set(branch?.soldOut ?? []));
  }, [branch?.id, branch?.soldOut]);
  // Dishes the branch's till has stopped. Held apart from the list above
  // because they are somebody else's to lift: offering a toggle here would put
  // the dish back for the three minutes until the next sync, which reads as a
  // broken button rather than as a fact about the till.
  const posSoldOut = useMemo(
    () => new Set(branch?.posSoldOut ?? []),
    [branch?.posSoldOut],
  );

  async function toggleSoldOut(item: MenuItem) {
    if (!branch || posSoldOut.has(item.id)) return;
    const next = !soldOut.has(item.id);
    // Optimistic: this is pressed mid-service, and waiting for a round trip
    // before the label changes makes the counter press it twice.
    setSoldOut((cur) => {
      const copy = new Set(cur);
      if (next) copy.add(item.id);
      else copy.delete(item.id);
      return copy;
    });
    try {
      await api.setSoldOut(branch.id, item.id, next);
    } catch {
      setSoldOut((cur) => {
        const copy = new Set(cur);
        if (next) copy.delete(item.id);
        else copy.add(item.id);
        return copy;
      });
      void tell({ title: t.common.saveFailed });
    }
  }

  function load() {
    setLoading(true);
    Promise.all([
      api.adminCategories(),
      api.adminMenu(),
      // ⚠️ Failing softly on its own: a restaurant that has never opened the
      // ingredients screen must still be able to edit its menu, and an empty
      // list is exactly what the option editor is built to say something
      // useful about.
      api
        .adminIngredients()
        .then((d) => d.ingredients)
        .catch(() => [] as Ingredient[]),
    ])
      .then(([c, m, ing]) => {
        setCats(c);
        setItems(m);
        setIngredients(ing);
      })
      .catch(() => {
        setCats([]);
        setItems([]);
      })
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  async function save() {
    if (!draft || !draft.name.trim() || !draft.categoryId) {
      void tell({ title: t.menu.nameRequired });
      return;
    }
    const price = Number(draft.price);
    if (!Number.isFinite(price) || price < 0) {
      void tell({ title: t.menu.priceInvalid });
      return;
    }
    // ⚠️ A set with nothing in it is a dish, and the server would file it as
    // one — silently, because empty `comboItems` is exactly how a dish is
    // stored. Saying so here is the only place it can be said: afterwards the
    // form has closed and the item looks saved, which it is, as the wrong kind
    // of thing.
    if (draft.kind === "combo" && draft.comboItems.length === 0) {
      void tell({ title: t.menu.comboEmpty });
      return;
    }
    // ⚠️ **Refused rather than quietly dropped.** `fromOptionDrafts` throws away
    // a question with no name or no named answer — correctly; that is not a
    // question. But it did it silently, so filling in the sizes and pressing
    // save produced a dish that saved with the variants gone, which from the
    // owner's side is indistinguishable from a save that did not work. It was
    // reported as exactly that.
    if (optionProblems(draft.options).length > 0) {
      void tell({ title: t.options.incomplete });
      return;
    }
    setSaving(true);
    const payload: Partial<MenuItem> = {
      categoryId: draft.categoryId,
      name: draft.name.trim(),
      description: draft.description,
      nameRu: draft.nameRu.trim(),
      nameEn: draft.nameEn.trim(),
      descriptionRu: draft.descriptionRu,
      descriptionEn: draft.descriptionEn,
      price,
      oldPrice: draft.oldPrice ? Number(draft.oldPrice) : null,
      // ⚠️ Always sent, including as 0 — an empty box means "I do not know",
      // and the server keeps the stored value only when the field is absent
      // entirely. A form that omitted it could never clear a wrong cost.
      cost: draft.cost ? Number(draft.cost) : 0,
      // ⚠️ **The card is deliberately absent.** It is written on its own
      // screen, and the server keeps what is stored for any field this form
      // does not send (`keepRecipe`). Sending `draft.recipe` back would let a
      // dish form opened before an edit on the card screen quietly revert it.
      imageUrl: draft.imageUrl,
      images: [],
      isAvailable: draft.isAvailable,
      isPopular: draft.isPopular,
      sortOrder: draft.sortOrder,
      options: fromOptionDrafts(draft.options),
      comboItems: draft.comboItems,
      recommendedIds: draft.recommendedIds,
      tags: draft.tags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean),
      // Sent as typed. ⚠️ The server is the one that decides what a code is —
      // it keeps only 17 digits and clears anything else — so cleaning it here
      // too would be a second rule to keep in step, and the browser's copy is
      // the one that would drift.
      ikpu: draft.ikpu.trim(),
      marked: draft.marked,
      packageCode: draft.packageCode.trim(),
      // ⚠️ Empty stays null rather than becoming 0. Number("") is 0, which
      // would silently mark every dish in the menu as VAT-exempt the first
      // time somebody saved it without touching this field.
      vatPercent:
        draft.vatPercent.trim() === "" ? null : Number(draft.vatPercent),
      unitCode: draft.unitCode,
      barcode: draft.barcode.trim(),
      sellsItself: draft.sellsItself,
      // ⚠️ Sorted, because the till draws them in this order and a list that
      // reads 3/4, 1/4, 1/2 is a row of buttons somebody has to search.
      portions: [...draft.portions].sort((a, b) => a - b),
    };
    try {
      if (draft.id) {
        await api.updateMenuItem(draft.id, payload as MenuItem);
      } else {
        await api.createMenuItem(payload);
      }
      setDraft(null);
      load();
    } catch {
      void tell({ title: t.common.saveFailed });
    } finally {
      setSaving(false);
    }
  }

  async function remove(m: MenuItem) {
    if (!(await ask({ title: t.menu.confirmDelete(m.name), danger: true })))
      return;
    try {
      await api.deleteMenuItem(m.id);
      load();
    } catch {
      void tell({ title: t.common.deleteFailed });
    }
  }

  // ⚠️ A combo is never counted as uncosted: it has no card of its own — its
  // price is its members' — and listing it here would be a warning nobody can
  // clear, which is a warning people learn to skip past.
  const uncosted = items.filter(
    (i) => !i.recipeCost && !i.cost && !i.comboContents?.length,
  );
  const shown = uncostedOnly ? uncosted : items;
  /** One list in model order: each model followed by its own variants.
   *
   *  ⚠️ **Ordered rather than hidden.** Every variant needs a barcode typed
   *  into it, so they have to stay reachable — but left to the sort order they
   *  are twelve rows of the same word in twelve different places, and the one
   *  with no barcode is the one nobody finds. */
  function grouped(list: MenuItem[]): MenuItem[] {
    // ⚠️ **`hasId`, never a bare truthiness check.** A row with no model
    // arrives with `variantOf: "000000000000000000000000"`, which is truthy —
    // so read naively every dish was a variant of one zero model and no dish
    // was ever a model, and the whole list fell through to the tail loop.
    const kids = new Map<string, MenuItem[]>();
    for (const m of list) {
      if (!hasId(m.variantOf)) continue;
      kids.set(m.variantOf, [...(kids.get(m.variantOf) ?? []), m]);
    }
    const out: MenuItem[] = [];
    const placed = new Set<string>();
    for (const m of list) {
      if (hasId(m.variantOf)) continue;
      out.push(m);
      placed.add(m.id);
      for (const k of kids.get(m.id) ?? []) {
        out.push(k);
        placed.add(k.id);
      }
    }
    // ⚠️ A variant whose model sits in another category — or has been deleted —
    // still has stock behind it, so it must appear somewhere on this screen.
    for (const m of list) {
      if (!placed.has(m.id)) out.push(m);
    }
    return out;
  }

  const byCat = cats.map((c) => ({
    category: c,
    items: shown.filter((i) => i.categoryId === c.id),
  }));
  const orphans = shown.filter((i) => !cats.some((c) => c.id === i.categoryId));

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand";

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t.menu.title}</h1>
        <div className="flex gap-2">
          {/* ⚠️ **No category needed, unlike "add a dish".** Import creates the
              sections it finds — a restaurant that has just signed up has an
              empty menu and no categories, which is precisely the moment this
              button is worth the most. Disabling it beside the other one would
              hide it exactly when it is needed. */}
          <button
            type="button"
            onClick={() => setImporting(true)}
            className="btn-ghost px-4 py-2"
          >
            {t.menuImport.button}
          </button>
          <button
            type="button"
            onClick={() => setDraft(emptyDraft(cats[0]?.id ?? ""))}
            disabled={cats.length === 0}
            data-help="add"
            className="btn-primary px-4 py-2 disabled:opacity-60"
            title={cats.length === 0 ? t.menu.needCategory : ""}
          >
            {t.menu.addNew}
          </button>
        </div>
      </div>

      {importing && (
        <Modal wide onClose={() => setImporting(false)}>
          <h2 className="mb-4 text-lg font-bold">{t.menuImport.title}</h2>
          <MenuImport onDone={load} onClose={() => setImporting(false)} />
        </Modal>
      )}

      {/* ⚠️ Only when there is something to say. On a menu nobody has costed
          this would be a permanent banner counting every dish — and a warning
          that is always on is one nobody reads. It appears once the first cost
          is entered, which is exactly when the rest become findable. */}
      {!loading && uncosted.length > 0 && uncosted.length < items.length && (
        <button
          type="button"
          onClick={() => setUncostedOnly(!uncostedOnly)}
          data-help="uncosted"
          className={`mt-4 block w-full rounded-xl px-4 py-2 text-left text-sm ${
            uncostedOnly
              ? "bg-brand/10 text-brand"
              : "bg-ink/[0.04] text-ink-soft"
          }`}
        >
          {uncostedOnly
            ? t.menu.uncostedShowAll
            : t.menu.uncostedCount(uncosted.length)}
        </button>
      )}

      {cats.length === 0 && !loading && (
        <p className="mt-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-700">
          {t.menu.needCategoryNotice}
        </p>
      )}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : (
        <div className="mt-6 space-y-8">
          {byCat.map(({ category, items: list }) => (
            <section key={category.id}>
              <h2 className="mb-3 text-lg font-bold">
                {category.name}{" "}
                <span className="text-sm font-normal text-ink-muted/70">
                  ({list.length})
                </span>
              </h2>
              {list.length === 0 ? (
                <p className="text-sm text-ink-muted/70">{t.menu.noItems}</p>
              ) : (
                <ListScroll
                  className="divide-y divide-line rounded-3xl border border-line bg-surface shadow-card"
                  max="max-h-[26rem]"
                >
                  {grouped(list).map((m) => (
                    <MenuRow
                      key={m.id}
                      item={m}
                      soldOut={soldOut.has(m.id) || posSoldOut.has(m.id)}
                      posLocked={posSoldOut.has(m.id)}
                      onToggleSoldOut={
                        branch ? () => toggleSoldOut(m) : undefined
                      }
                      onEdit={() => setDraft(toDraft(m))}
                      onDelete={() => remove(m)}
                      t={t}
                    />
                  ))}
                </ListScroll>
              )}
            </section>
          ))}

          {orphans.length > 0 && (
            <section>
              <h2 className="mb-3 text-lg font-bold text-ink-muted">
                {t.menu.uncategorised}
              </h2>
              <ListScroll
                className="divide-y divide-line rounded-3xl border border-line bg-surface shadow-card"
                max="max-h-[26rem]"
              >
                {grouped(orphans).map((m) => (
                  <MenuRow
                    key={m.id}
                    item={m}
                    soldOut={soldOut.has(m.id) || posSoldOut.has(m.id)}
                    posLocked={posSoldOut.has(m.id)}
                    onToggleSoldOut={
                      branch ? () => toggleSoldOut(m) : undefined
                    }
                    onEdit={() => setDraft(toDraft(m))}
                    onDelete={() => remove(m)}
                    t={t}
                  />
                ))}
              </ListScroll>
            </section>
          )}
        </div>
      )}

      {draft && (
        <Modal wide onClose={() => setDraft(null)}>
          <h2 className="text-lg font-bold">
            {draft.id ? t.menu.editTitle : t.menu.newTitle}
          </h2>

          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <label className="block text-sm sm:col-span-2">
              <span className="font-medium">{t.menu.name}</span>
              <input
                className={inputCls}
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                autoFocus
              />
            </label>

            <label className="block text-sm sm:col-span-2">
              <span className="font-medium">{t.menu.description}</span>
              <textarea
                className={inputCls}
                rows={2}
                value={draft.description}
                onChange={(e) =>
                  setDraft({ ...draft, description: e.target.value })
                }
              />
            </label>

            {/* Translations. Bo'sh qoldirilsa — saytda o'zbekchasi ko'rinadi. */}
            <div className="sm:col-span-2 rounded-2xl border border-line bg-ink/[0.02] p-4">
              <p className="text-sm font-semibold">
                {t.menu.translations}{" "}
                <span className="font-normal text-ink-muted">
                  {t.menu.translationsHint}
                </span>
              </p>
              <div className="mt-3 grid gap-4 sm:grid-cols-2">
                <label className="block text-sm">
                  <span className="font-medium">{t.menu.nameRuLabel}</span>
                  <input
                    className={inputCls}
                    value={draft.nameRu}
                    placeholder={draft.name}
                    onChange={(e) =>
                      setDraft({ ...draft, nameRu: e.target.value })
                    }
                  />
                </label>
                <label className="block text-sm">
                  <span className="font-medium">{t.menu.nameEnLabel}</span>
                  <input
                    className={inputCls}
                    value={draft.nameEn}
                    placeholder={draft.name}
                    onChange={(e) =>
                      setDraft({ ...draft, nameEn: e.target.value })
                    }
                  />
                </label>
                <label className="block text-sm">
                  <span className="font-medium">{t.menu.descRuLabel}</span>
                  <textarea
                    className={inputCls}
                    rows={2}
                    value={draft.descriptionRu}
                    onChange={(e) =>
                      setDraft({ ...draft, descriptionRu: e.target.value })
                    }
                  />
                </label>
                <label className="block text-sm">
                  <span className="font-medium">{t.menu.descEnLabel}</span>
                  <textarea
                    className={inputCls}
                    rows={2}
                    value={draft.descriptionEn}
                    onChange={(e) =>
                      setDraft({ ...draft, descriptionEn: e.target.value })
                    }
                  />
                </label>
              </div>
            </div>

            {/* A dish and a set are the same document, so the form offers both
                — but never at once: a fixed set has nowhere to ask a question,
                and a dish with courses inside is not a dish. */}
            {/* ⚠️ **Full width, and its absence was the whole bug.** The form
                is a two-column grid; this card had no `col-span-2`, so it sat
                in one half and everything inside it was laid out in ~250px.
                The options editor's own `sm:col-span-2` could do nothing about
                that — it is not a child of this grid — and the result was three
                name fields twenty pixels wide with their labels wrapped over
                two lines. */}
            <div className="rounded-2xl border border-line p-4 sm:col-span-2">
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() =>
                    setDraft({ ...draft, kind: "dish", comboItems: [] })
                  }
                  className={`rounded-full border px-4 py-1.5 text-sm font-semibold transition-colors ${
                    draft.kind === "dish"
                      ? "border-brand bg-brand-tint text-brand-dark"
                      : "border-line-strong text-ink-soft hover:border-brand"
                  }`}
                >
                  {t.menu.kindDish}
                </button>
                <button
                  type="button"
                  onClick={() =>
                    setDraft({ ...draft, kind: "combo", options: [] })
                  }
                  className={`rounded-full border px-4 py-1.5 text-sm font-semibold transition-colors ${
                    draft.kind === "combo"
                      ? "border-brand bg-brand-tint text-brand-dark"
                      : "border-line-strong text-ink-soft hover:border-brand"
                  }`}
                >
                  {t.menu.kindCombo}
                </button>
              </div>

              <div className="mt-4">
                {draft.kind === "combo" ? (
                  <ComboEditor
                    value={draft.comboItems}
                    price={Number(draft.price) || 0}
                    menu={items.filter((m) => m.id !== draft.id)}
                    onChange={(comboItems) =>
                      setDraft({ ...draft, comboItems })
                    }
                  />
                ) : (
                  <OptionsEditor
                    groups={draft.options}
                    // So a choice can show what the guest will actually pay
                    // rather than a signed number they have to add up.
                    price={Number(draft.price) || 0}
                    onChange={(options) => setDraft({ ...draft, options })}
                    // The same list the dish's own card uses — a pour is a
                    // tech card that happens to hang off a choice.
                    ingredients={ingredients}
                  />
                )}
              </div>

              {/* Applies to a set as much as to a dish — a family combo with a
                  drink suggested beside it is the same sale. */}
              <div className="mt-4">
                <RecommendEditor
                  value={draft.recommendedIds}
                  menu={items.filter((m) => m.id !== draft.id)}
                  onChange={(recommendedIds) =>
                    setDraft({ ...draft, recommendedIds })
                  }
                />
              </div>
            </div>

            <label className="block text-sm">
              <span className="font-medium">{t.menu.category}</span>
              <select
                className={inputCls}
                value={draft.categoryId}
                onChange={(e) =>
                  setDraft({ ...draft, categoryId: e.target.value })
                }
              >
                {cats.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.menu.sortOrder}</span>
              <input
                type="number"
                className={inputCls}
                value={draft.sortOrder}
                onChange={(e) =>
                  setDraft({ ...draft, sortOrder: Number(e.target.value) || 0 })
                }
              />
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.menu.price}</span>
              <input
                type="number"
                className={inputCls}
                value={draft.price}
                onChange={(e) => setDraft({ ...draft, price: e.target.value })}
              />
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.menu.oldPrice}</span>
              <input
                type="number"
                className={inputCls}
                value={draft.oldPrice}
                onChange={(e) =>
                  setDraft({ ...draft, oldPrice: e.target.value })
                }
              />
            </label>

            {/* ⚠️ **Typed by hand, and nothing here can check it.** There are
                no recipes and no stock in this system, so this is the owner's
                own figure — and it never leaves the panel: what a plate costs
                the kitchen is the one number on a dish a competitor would pay
                for, and the menu is public. */}
            <label className="block text-sm">
              <span className="font-medium">{t.menu.cost}</span>
              <input
                type="number"
                min={0}
                className={inputCls}
                value={draft.cost}
                onChange={(e) => setDraft({ ...draft, cost: e.target.value })}
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.menu.costHint}
              </span>
            </label>

            {/* ⚠️ **The card is not edited here any more, but it is still
                shown here** — directly under the cost, because it **replaces**
                it: with lines on the card the typed number stops being used,
                and an owner typing a cost has to be able to see that the
                number will be ignored. What moved is the editing, to
                "Ombor → Texkartalar", where the preps live too; what stays is
                the sentence saying which figure this dish is actually costed
                by.

                ⚠️ A link, not a second editor. Two forms writing one card is
                the drift the cards exist to end — and this one is a whole
                document replace, so the losing side would win silently. */}
            {/* ⚠️ **A kitchen's document, so a shop is not shown it.** A
                product that sells itself already has a one-line card the server
                writes and keeps in step; a link inviting somebody to edit it by
                hand can only break the tie between the packet and the shelf it
                comes off. Shown anyway when a card already exists — data wins
                over the type, here as everywhere on this page. */}
            {(brandComposes || draft.recipeLines > 0) && (
            <div className="block text-sm sm:col-span-2">
              <span className="font-medium">{t.recipe.title}</span>
              <div className="mt-1 rounded-xl bg-ink/[0.03] px-3 py-2">
                {draft.id && draft.recipeCost ? (
                  <p className="text-sm">
                    {t.menu.cardCost(
                      draft.recipeLines,
                      formatPrice(draft.recipeCost),
                    )}
                  </p>
                ) : (
                  <p className="text-sm text-ink-muted">{t.menu.cardNone}</p>
                )}
                {/* ⚠️ Only on a saved dish: the card is written against an id,
                    and a new dish has none until it is saved. Offering the
                    link before then would open an empty screen and lose the
                    form. */}
                {draft.id ? (
                  <Link
                    href={`/admin/tech-cards?dish=${draft.id}`}
                    className="mt-1 inline-block text-xs underline"
                  >
                    {t.menu.cardEdit}
                  </Link>
                ) : (
                  <p className="mt-1 text-xs text-ink-muted">
                    {t.menu.cardAfterSave}
                  </p>
                )}
              </div>
            </div>
            )}

            <label className="block text-sm sm:col-span-2">
              <span className="font-medium">{t.menu.tags}</span>
              <input
                className={inputCls}
                placeholder={t.menu.tagsPh}
                value={draft.tags}
                onChange={(e) => setDraft({ ...draft, tags: e.target.value })}
              />
            </label>

            {/* ⚠️ Shown to everybody, empty by default, and that is deliberate.
                The code only matters to a restaurant issuing fiscal receipts,
                and hiding the field until a payment provider is configured
                would mean the accountant cannot fill the menu in *before* the
                provider is connected — which is the order these actually
                happen in. The help line is what stops it looking mandatory. */}
            <label className="block text-sm sm:col-span-2">
              <span className="font-medium">{t.menu.ikpu}</span>
              <input
                className={inputCls}
                placeholder={t.menu.ikpuPh}
                inputMode="numeric"
                value={draft.ikpu}
                onChange={(e) => setDraft({ ...draft, ikpu: e.target.value })}
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.menu.ikpuHint}
              </span>
            </label>

            {/* The packaging code belongs to the ИКПУ, so it is only asked for
                once there is one. Not disabled but hidden: a greyed-out box
                invites the question "why can't I type here", and the answer is
                one field up. */}
            {draft.ikpu.trim() !== "" && (
              <label className="block text-sm sm:col-span-2">
                <span className="font-medium">{t.menu.packageCode}</span>
                <input
                  className={inputCls}
                  placeholder={t.menu.packageCodePh}
                  inputMode="numeric"
                  value={draft.packageCode}
                  onChange={(e) =>
                    setDraft({ ...draft, packageCode: e.target.value })
                  }
                />
                <span className="mt-1 block text-xs text-ink-muted">
                  {t.menu.packageCodeHint}
                </span>
              </label>
            )}

            {/* ⚠️ **Beside the fiscal fields because that is where it acts.**
                The code is scanned at the till and travels inside the fiscal
                receipt — there is no separate marking system to configure. What
                this box decides is whether the till refuses to sell this dish
                without a scan, which is why the help line says so rather than
                describing the law. */}
            {/* ⚠️ **Which parts of a portion this dish sells in.** Half a loaf
                and a quarter of an opened bottle are ordinary; half a sealed
                bottle of water is not something a bar can hand over, and a till
                that offered it would be offering a sale the shelf cannot
                fulfil. Which dishes divide is knowledge the restaurant has and
                we do not, so it is asked here, per dish, and left empty for
                almost all of them.

                The price follows the part (half costs half, rounded to a
                so'm) and so does the store: a half takes half the card off the
                shelf. */}
            {/* ⚠️ **Half a portion is a kitchen's idea.** A shop measures a
                part of something by weighing it, and the packet's own measure
                code already says so — offering "50%" of a bag of rice beside a
                field that asks for kilograms is two answers to one question. */}
            {(brandHasKitchen || draft.portions.length > 0) && (
            <div className="text-sm sm:col-span-2">
              <span className="font-medium">{t.menu.portions}</span>
              <div className="mt-1.5 flex flex-wrap gap-2">
                {[25, 33, 50, 75].map((p) => {
                  const on = draft.portions.includes(p);
                  return (
                    <button
                      key={p}
                      type="button"
                      onClick={() =>
                        setDraft({
                          ...draft,
                          portions: on
                            ? draft.portions.filter((x) => x !== p)
                            : [...draft.portions, p],
                        })
                      }
                      className={`rounded-full border px-3.5 py-1.5 text-sm font-semibold transition-colors ${
                        on
                          ? "border-brand bg-brand text-white"
                          : "border-line-strong text-ink-soft hover:border-brand"
                      }`}
                    >
                      {t.menu.portionLabel(p)}
                    </button>
                  );
                })}
              </div>
              <span className="mt-1 block text-xs text-ink-muted">
                {t.menu.portionsHint}
              </span>
            </div>
            )}

            <label className="flex items-start gap-2 text-sm sm:col-span-2">
              <input
                type="checkbox"
                className="mt-1"
                checked={draft.marked}
                onChange={(e) =>
                  setDraft({ ...draft, marked: e.target.checked })
                }
              />
              <span>
                <span className="font-medium">{t.menu.marked}</span>
                <span className="mt-1 block text-xs text-ink-muted">
                  {t.menu.markedHint}
                </span>
              </span>
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.menu.vatPercent}</span>
              <input
                className={inputCls}
                placeholder={t.menu.vatPercentPh}
                inputMode="numeric"
                value={draft.vatPercent}
                onChange={(e) =>
                  setDraft({ ...draft, vatPercent: e.target.value })
                }
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.menu.vatPercentHint}
              </span>
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.menu.unitCode}</span>
              <select
                className={inputCls}
                value={draft.unitCode}
                onChange={(e) =>
                  setDraft({ ...draft, unitCode: Number(e.target.value) })
                }
              >
                <option value={0}>{t.menu.units.piece}</option>
                <option value={11}>{t.menu.units.kilogram}</option>
                <option value={10}>{t.menu.units.gram}</option>
                <option value={41}>{t.menu.units.litre}</option>
                <option value={22}>{t.menu.units.metre}</option>
              </select>
              <span className="mt-1 block text-xs text-ink-muted">
                {t.menu.unitCodeHint}
              </span>
            </label>

            {/* ---- Selling goods rather than dishes ----

                ⚠️ **Shown to a shop, and to any dish that already carries
                one.** A restaurant has no use for a barcode — a portion of osh
                will never have one — and two fields nobody can fill in are two
                fields every owner reads past on the way to the price. But
                hiding a field that holds a value is worse than showing an
                irrelevant one: it makes data invisible and un-editable while
                still saving it. So the type decides the default and the data
                overrides it.

                ⚠️ **Two fields, and the second is the one that matters.** A
                barcode is how a shop's counter finds this at all — it is
                scanned, never tapped. "Sells itself" is the whole difference
                between a shop and a kitchen: a kitchen turns inputs into
                outputs, so what is sold and what is stocked are two documents
                with a tech card between them; a shop sells the object it
                bought, so the server keeps the stock row behind this product in
                step and nobody maintains two names by hand. */}
            {showGoods && (
              <>
            <label className="block text-sm">
              <span className="font-medium">{t.menu.barcode}</span>
              <input
                className={inputCls}
                value={draft.barcode}
                inputMode="numeric"
                // ⚠️ Off, all of it: a barcode is not a word, and autocorrect on
                // a tablet turns a digit string into something else entirely.
                autoComplete="off"
                autoCorrect="off"
                spellCheck={false}
                onChange={(e) =>
                  setDraft({ ...draft, barcode: e.target.value.trim() })
                }
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.menu.barcodeHint}
              </span>
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.menu.sellsItself}</span>
              <div className="mt-1 flex items-center gap-2">
                <input
                  type="checkbox"
                  className="h-5 w-5"
                  checked={draft.sellsItself}
                  onChange={(e) =>
                    setDraft({ ...draft, sellsItself: e.target.checked })
                  }
                />
                <span className="text-xs text-ink-muted">
                  {t.menu.sellsItselfHint}
                </span>
              </div>
            </label>

            {/* ⚠️ **Only on a saved product**, for the reason the technical
                card link is: the variants are written against an id, and a
                model that has not been saved has none. */}
            {/* ⚠️ **A clothes shop, and anything that already has variants.**
                Sizes are the whole shape of a boutique's catalogue and mean
                nothing in a pharmacy; a grocery's pack sizes are separate
                products with separate barcodes, which is what it already enters
                them as — a matrix generator there would double its catalogue by
                accident. */}
            {draft.id &&
              (hasVariants(scope.brand) ||
                items.some((m) => m.variantOf === draft.id)) && (
              <div className="block text-sm sm:col-span-2">
                <span className="font-medium">{t.menu.variantTitle}</span>
                <div className="mt-1">
                  <VariantsEditor
                    item={items.find((m) => m.id === draft.id) ?? ({ id: draft.id } as MenuItem)}
                    variants={items.filter((m) => m.variantOf === draft.id)}
                    onDone={load}
                  />
                </div>
              </div>
            )}
              </>
            )}

            <div className="sm:col-span-2">
              <span className="text-sm font-medium">{t.menu.image}</span>
              <div className="mt-1">
                <ImageUpload
                  value={draft.imageUrl}
                  onChange={(url) => setDraft({ ...draft, imageUrl: url })}
                />
              </div>
            </div>

            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={draft.isAvailable}
                onChange={(e) =>
                  setDraft({ ...draft, isAvailable: e.target.checked })
                }
              />
              <span className="font-medium">{t.menu.availableLabel}</span>
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={draft.isPopular}
                onChange={(e) =>
                  setDraft({ ...draft, isPopular: e.target.checked })
                }
              />
              <span className="font-medium">{t.menu.isPopular}</span>
            </label>
          </div>

          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setDraft(null)}
              className="btn-ghost px-4 py-2"
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              onClick={save}
              disabled={saving}
              className="btn-primary px-4 py-2 disabled:opacity-60"
            >
              {saving ? t.common.saving : t.common.save}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

function MenuRow({
  item,
  soldOut,
  posLocked,
  onToggleSoldOut,
  onEdit,
  onDelete,
  t,
}: {
  item: MenuItem;
  soldOut: boolean;
  /** Stopped in the till, not here — the toggle is replaced by a label. */
  posLocked: boolean;
  /** Absent when no single branch is selected — "sold out where?" has no
   *  answer while the panel is looking at a whole brand. */
  onToggleSoldOut?: () => void;
  onEdit: () => void;
  onDelete: () => void;
  t: ReturnType<typeof useAdminT>;
}) {
  return (
    <div className="flex items-center gap-3 p-3">
      <span className="w-8 text-center text-sm text-ink-muted/70">
        {item.sortOrder}
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate font-medium">{item.name}</span>
          {item.isPopular && (
            <span className="rounded-full bg-brand/10 px-1.5 text-xs text-brand">
              {t.menu.popularShort}
            </span>
          )}
          {!item.isAvailable && (
            <span className="rounded-full bg-ink/5 px-1.5 text-xs text-ink-muted">
              {t.menu.hiddenShort}
            </span>
          )}
          {item.comboItems?.length ? (
            <span className="rounded-full bg-emerald-500/15 px-1.5 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
              {t.menu.comboShort}
            </span>
          ) : null}
          {soldOut && item.isAvailable && (
            <span
              className={`rounded-full px-1.5 text-xs font-semibold ${
                posLocked
                  ? "bg-rose-500/15 text-rose-700 dark:text-rose-300"
                  : "bg-amber-500/15 text-amber-700 dark:text-amber-300"
              }`}
            >
              {posLocked ? t.stopList.posBadge : t.menu.soldOutShort}
            </span>
          )}
        </div>
        {item.comboContents?.length ? (
          <p className="truncate text-xs text-ink-muted/70">
            {item.comboContents
              .map((c) => (c.qty > 1 ? `${c.name} × ${c.qty}` : c.name))
              .join(" · ")}
          </p>
        ) : null}
      </div>
      <div className="text-right">
        <span className="font-semibold">{formatPrice(item.price)}</span>
        {/* ⚠️ **The margin belongs where the price is set, not only in a
            report.** The report answers "what sold last month"; this answers
            "what am I charging for this", which is the question being asked at
            the moment somebody opens this row. Shown only when a cost was
            typed — a dash on every dish would be a column of nothing.

            ⚠️ And a dish priced at or below its cost is called out rather than
            rendered as a small number: it is either a typo or a plate the
            restaurant loses money on, and both are invisible today. */}
        {/* ⚠️ The card's figure first, the typed one second — the same
            precedence the server applies when it costs a sale. A row showing
            the old typed number beside a dish whose card says otherwise is the
            drift the cards exist to end. */}
        {item.recipeCost || item.cost ? (
          <span
            className={`block text-xs ${
              item.price > (item.recipeCost || item.cost || 0)
                ? "text-ink-muted"
                : "text-danger"
            }`}
          >
            {item.price > (item.recipeCost || item.cost || 0)
              ? t.menu.marginShort(
                  Math.round(
                    ((item.price - (item.recipeCost || item.cost || 0)) /
                      item.price) *
                      100,
                  ),
                )
              : t.menu.belowCost}
          </span>
        ) : null}
      </div>
      {onToggleSoldOut && item.isAvailable && posLocked && (
        <span
          title={t.stopList.posLocked}
          className="text-xs text-ink-muted/70"
        >
          {t.stopList.posBadge}
        </span>
      )}
      {onToggleSoldOut && item.isAvailable && !posLocked && (
        <button
          type="button"
          onClick={onToggleSoldOut}
          title={t.menu.soldOutHint}
          className={`rounded-full border px-2.5 py-1 text-xs font-semibold transition-colors ${
            soldOut
              ? "border-amber-500/50 bg-amber-500/10 text-amber-700 dark:text-amber-300"
              : "border-line-strong text-ink-muted hover:border-brand hover:text-brand"
          }`}
        >
          {soldOut ? t.menu.backInStock : t.menu.markSoldOut}
        </button>
      )}
      <button
        type="button"
        data-help="edit"
        onClick={onEdit}
        className="text-sm text-brand hover:underline"
      >
        {t.common.edit}
      </button>
      <button
        type="button"
        onClick={onDelete}
        className="text-sm text-ink-muted/70 hover:text-brand"
      >
        {t.common.delete}
      </button>
    </div>
  );
}
