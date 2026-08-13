"use client";

import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import ImageUpload from "@/components/admin/ImageUpload";
import Modal from "@/components/admin/Modal";
import { ListScroll } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import OptionsEditor, {
  fromOptionDrafts,
  toOptionDrafts,
  type OptionGroupDraft,
} from "@/components/admin/OptionsEditor";
import ComboEditor from "@/components/admin/ComboEditor";
import type { Category, ComboLine, MenuItem } from "@/lib/types";

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
  imageUrl: string;
  isAvailable: boolean;
  isPopular: boolean;
  sortOrder: number;
  tags: string;
  ikpu: string;
  options: OptionGroupDraft[];
  comboItems: ComboLine[];
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
    imageUrl: m.imageUrl,
    isAvailable: m.isAvailable,
    isPopular: m.isPopular,
    sortOrder: m.sortOrder,
    tags: (m.tags ?? []).join(", "),
    ikpu: m.ikpu ?? "",
    options: toOptionDrafts(m.options),
    comboItems: m.comboItems ?? [],
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
    imageUrl: "",
    isAvailable: true,
    isPopular: false,
    sortOrder: 0,
    tags: "",
    ikpu: "",
    options: [],
    comboItems: [],
    kind: "dish",
  };
}

export default function AdminMenuPage() {
  const [cats, setCats] = useState<Category[]>([]);
  const [items, setItems] = useState<MenuItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [saving, setSaving] = useState(false);
  const t = useAdminT();
  const scope = useAdminScope();
  // What has run out **at this branch** today. The menu itself belongs to the
  // brand, so this is the one thing on this page that is not shared.
  const branch = scope.branch;
  const [soldOut, setSoldOut] = useState<Set<string>>(new Set());
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
      alert(t.common.saveFailed);
    }
  }

  function load() {
    setLoading(true);
    Promise.all([api.adminCategories(), api.adminMenu()])
      .then(([c, m]) => {
        setCats(c);
        setItems(m);
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
      alert(t.menu.nameRequired);
      return;
    }
    const price = Number(draft.price);
    if (!Number.isFinite(price) || price < 0) {
      alert(t.menu.priceInvalid);
      return;
    }
    // ⚠️ A set with nothing in it is a dish, and the server would file it as
    // one — silently, because empty `comboItems` is exactly how a dish is
    // stored. Saying so here is the only place it can be said: afterwards the
    // form has closed and the item looks saved, which it is, as the wrong kind
    // of thing.
    if (draft.kind === "combo" && draft.comboItems.length === 0) {
      alert(t.menu.comboEmpty);
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
      imageUrl: draft.imageUrl,
      images: [],
      isAvailable: draft.isAvailable,
      isPopular: draft.isPopular,
      sortOrder: draft.sortOrder,
      options: fromOptionDrafts(draft.options),
      comboItems: draft.comboItems,
      tags: draft.tags
        .split(",")
        .map((t) => t.trim())
        .filter(Boolean),
      // Sent as typed. ⚠️ The server is the one that decides what a code is —
      // it keeps only 17 digits and clears anything else — so cleaning it here
      // too would be a second rule to keep in step, and the browser's copy is
      // the one that would drift.
      ikpu: draft.ikpu.trim(),
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
      alert(t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function remove(m: MenuItem) {
    if (!confirm(`"${m.name}" taomini o'chirasizmi?`)) return;
    try {
      await api.deleteMenuItem(m.id);
      load();
    } catch {
      alert(t.common.deleteFailed);
    }
  }

  const byCat = cats.map((c) => ({
    category: c,
    items: items.filter((i) => i.categoryId === c.id),
  }));
  const orphans = items.filter((i) => !cats.some((c) => c.id === i.categoryId));

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand";

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t.menu.title}</h1>
        <button
          type="button"
          onClick={() => setDraft(emptyDraft(cats[0]?.id ?? ""))}
          disabled={cats.length === 0}
          className="btn-primary px-4 py-2 disabled:opacity-60"
          title={cats.length === 0 ? t.menu.needCategory : ""}
        >
          {t.menu.addNew}
        </button>
      </div>

      {cats.length === 0 && !loading && (
        <p className="mt-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-700">
          {t.menu.needCategoryNotice}
        </p>
      )}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>
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
                  {list.map((m) => (
                    <MenuRow
                      key={m.id}
                      item={m}
                      soldOut={soldOut.has(m.id) || posSoldOut.has(m.id)}
                      posLocked={posSoldOut.has(m.id)}
                      onToggleSoldOut={branch ? () => toggleSoldOut(m) : undefined}
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
                {orphans.map((m) => (
                  <MenuRow
                    key={m.id}
                    item={m}
                    soldOut={soldOut.has(m.id) || posSoldOut.has(m.id)}
                    posLocked={posSoldOut.has(m.id)}
                    onToggleSoldOut={branch ? () => toggleSoldOut(m) : undefined}
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
            <div className="rounded-2xl border border-line p-4">
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => setDraft({ ...draft, kind: "dish", comboItems: [] })}
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
                  onClick={() => setDraft({ ...draft, kind: "combo", options: [] })}
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
                    onChange={(comboItems) => setDraft({ ...draft, comboItems })}
                  />
                ) : (
                  <OptionsEditor
                    groups={draft.options}
                    onChange={(options) => setDraft({ ...draft, options })}
                  />
                )}
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
              <span className="mt-1 block text-xs text-ink-muted">{t.menu.ikpuHint}</span>
            </label>

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
      <span className="font-semibold">{formatPrice(item.price)}</span>
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
