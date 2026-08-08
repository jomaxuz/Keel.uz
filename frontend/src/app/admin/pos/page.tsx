"use client";

// Mapping our dishes to the till's products.
//
// This is the one screen the whole integration rests on: an order can only be
// pushed when every line on it has an id over there. So the page is built
// around the question "what is still unmapped?" rather than around the list —
// a menu of 48 dishes with three gaps is three orders that will fail at the
// worst moment, and finding them by scrolling is how they stay unfixed.
//
// Hence: the count of what is missing sits at the top, a filter shows only
// those, and a "match by name" pass fills in the obvious ones so the operator
// is left with the handful that genuinely need a decision.

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { contentName } from "@/lib/i18n/content";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import { useAdminScope } from "@/lib/adminScope";
import type { Category, MenuItem, POSProduct, POSSettings } from "@/lib/types";

export default function AdminPOSPage() {
  const t = useAdminT();
  const { lang } = useI18n();
  // Only a chain sees the copy block, and only an admin who may reach more than
  // one branch: a pinned manager is refused by the server anyway.
  const { brandBranches, branch, pinned } = useAdminScope();

  const [settings, setSettings] = useState<POSSettings | null>(null);
  const [menu, setMenu] = useState<MenuItem[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [products, setProducts] = useState<POSProduct[]>([]);
  // menuItemId → POS product id. The single piece of state the page edits.
  const [links, setLinks] = useState<Record<string, string>>({});
  const [dirty, setDirty] = useState(false);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [onlyMissing, setOnlyMissing] = useState(false);
  const [q, setQ] = useState("");
  const [copyFrom, setCopyFrom] = useState("");
  const [copyOverwrite, setCopyOverwrite] = useState(false);
  const [copying, setCopying] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [s, m, c, mapping] = await Promise.all([
        api.adminPOS(),
        api.adminMenu(),
        api.adminCategories(),
        api.posMapping(),
      ]);
      setSettings(s);
      setMenu(m);
      setCategories(c);
      setLinks(
        Object.fromEntries(mapping.map((x) => [x.menuItemId, x.posProductId])),
      );
      if (s.provider && s.enabled) {
        try {
          setProducts(await api.posProducts());
        } catch (e) {
          // The till being unreachable must not blank the mapping already
          // saved — it is still readable and still correct.
          setError(e instanceof Error ? e.message : t.common.loadFailed);
        }
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    load();
  }, [load]);

  const byId = useMemo(
    () => Object.fromEntries(products.map((p) => [p.id, p])),
    [products],
  );
  const categoryName = useMemo(
    () => Object.fromEntries(categories.map((c) => [c.id, contentName(c, lang)])),
    [categories, lang],
  );

  const rows = useMemo(() => {
    const needle = q.trim().toLowerCase();
    return menu
      .filter((d) => !onlyMissing || !links[d.id])
      .filter(
        (d) =>
          !needle ||
          contentName(d, lang).toLowerCase().includes(needle) ||
          d.name.toLowerCase().includes(needle),
      );
  }, [menu, links, onlyMissing, q, lang]);

  const paged = usePaged(rows, 25);
  const missing = menu.filter((d) => !links[d.id]).length;

  /** Fills in every dish whose name matches a POS product exactly, case and
   *  spacing aside. Only the obvious ones — anything ambiguous is left for a
   *  human, because a wrong mapping is worse than a missing one. */
  function matchByName() {
    const index = new Map<string, POSProduct[]>();
    for (const p of products) {
      const key = p.name.trim().toLowerCase();
      index.set(key, [...(index.get(key) ?? []), p]);
    }
    let filled = 0;
    const next = { ...links };
    for (const dish of menu) {
      if (next[dish.id]) continue;
      const hits = index.get(dish.name.trim().toLowerCase());
      // Exactly one candidate, or it is a guess.
      if (hits && hits.length === 1) {
        next[dish.id] = hits[0].id;
        filled++;
      }
    }
    setLinks(next);
    setDirty(filled > 0);
    setMessage(t.pos.matched(filled));
  }

  async function save() {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const items = menu.map((d) => ({
        menuItemId: d.id,
        posProductId: links[d.id] ?? "",
        posProductName: byId[links[d.id] ?? ""]?.name ?? "",
      }));
      await api.savePOSMapping(items);
      setDirty(false);
      setMessage(t.pos.saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  /** Copies another branch's links server-side, then rereads them.
   *
   *  Disabled while there are unsaved edits: the reread is what makes the copy
   *  visible, and it would take the half-finished mapping on screen with it. */
  async function copyMapping() {
    if (!copyFrom) return;
    setCopying(true);
    setError("");
    setMessage("");
    try {
      const res = await api.copyPOSMapping(copyFrom, copyOverwrite);
      setMessage(t.pos.copyDone(res.copied, res.skipped));
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setCopying(false);
    }
  }

  if (loading) {
    return <p className="text-sm text-ink-muted">{t.common.loading}</p>;
  }

  if (!settings?.provider || !settings.enabled) {
    return (
      <div className="card p-6">
        <h1 className="font-display text-xl font-bold">{t.pos.title}</h1>
        <p className="mt-2 text-sm text-ink-muted">{t.pos.notConnected}</p>
        <Link href="/admin/settings" className="btn btn-primary mt-4 inline-flex">
          {t.pos.openSettings}
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-bold">{t.pos.title}</h1>
          <p className="text-sm text-ink-muted">{t.pos.mappingHint}</p>
        </div>
        <Link href="/admin/settings" className="text-sm font-semibold text-brand hover:underline">
          {t.pos.openSettings}
        </Link>
      </div>

      {/* What is still missing, and therefore which orders will fail. */}
      <div
        className={`card p-4 ${missing > 0 ? "border-amber-500/40 bg-amber-500/5" : ""}`}
      >
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div className="font-display text-2xl font-bold">
              {menu.length - missing} / {menu.length}
            </div>
            <div className="text-xs text-ink-muted">{t.pos.mappedCount}</div>
          </div>
          {missing > 0 && (
            <p className="max-w-md text-sm text-amber-700 dark:text-amber-300">
              {t.pos.missingWarn(missing)}
            </p>
          )}
        </div>
      </div>

      {/* ---- copy from a sister branch ----
           A single-branch restaurant never sees this, which is the whole point
           of the branch feature: the complexity only appears once it is real. */}
      {!pinned && brandBranches.length > 1 && (
        <div className="card p-4">
          <h2 className="font-semibold">{t.pos.copyTitle}</h2>
          <p className="mt-1 text-xs text-ink-muted">{t.pos.copyHint}</p>
          <div className="mt-3 flex flex-wrap items-end gap-3">
            <label className="text-sm">
              <span className="mb-1 block text-xs text-ink-muted">
                {t.pos.copyFrom}
              </span>
              <select
                className="input w-full sm:w-64"
                value={copyFrom}
                onChange={(e) => setCopyFrom(e.target.value)}
              >
                <option value="">—</option>
                {brandBranches
                  .filter((b) => b.id !== branch?.id)
                  .map((b) => (
                    <option key={b.id} value={b.id}>
                      {b.name}
                    </option>
                  ))}
              </select>
            </label>
            <button
              type="button"
              onClick={copyMapping}
              disabled={!copyFrom || copying || dirty}
              className="btn btn-primary disabled:opacity-40"
            >
              {copying ? t.pos.copying : t.pos.copyRun}
            </button>
          </div>
          <label className="mt-3 flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-0.5"
              checked={copyOverwrite}
              onChange={(e) => setCopyOverwrite(e.target.checked)}
            />
            <span>
              {t.pos.copyOverwrite}
              <span className="block text-xs text-ink-muted">
                {t.pos.copyOverwriteHint}
              </span>
            </span>
          </label>
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <input
          className="input w-full sm:w-64"
          placeholder={t.pos.searchDish}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <button
          type="button"
          onClick={() => setOnlyMissing((v) => !v)}
          className={`chip ${onlyMissing ? "bg-brand text-white" : ""}`}
        >
          {t.pos.onlyMissing}
        </button>
        <button type="button" onClick={matchByName} className="chip">
          {t.pos.matchByName}
        </button>
        <button
          type="button"
          onClick={save}
          disabled={saving || !dirty}
          className="btn btn-primary ml-auto disabled:opacity-40"
        >
          {saving ? t.common.saving : t.common.save}
        </button>
      </div>

      {error && <p className="text-sm text-brand">{error}</p>}
      {message && (
        <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
          {message}
        </p>
      )}

      <div className="card">
        <ListScroll max="max-h-[65vh]">
          <table className="w-full text-sm">
            <thead className="sticky top-0 bg-surface">
              <tr className="border-b border-line text-left text-xs text-ink-muted">
                <th className="px-4 py-2">{t.pos.ourDish}</th>
                <th className="px-4 py-2">{t.pos.posProduct}</th>
              </tr>
            </thead>
            <tbody>
              {paged.pageItems.map((dish) => {
                const chosen = links[dish.id] ?? "";
                const product = byId[chosen];
                return (
                  <tr key={dish.id} className="border-b border-line last:border-0">
                    <td className="px-4 py-2">
                      <div className="font-medium">{contentName(dish, lang)}</div>
                      <div className="text-xs text-ink-muted">
                        {categoryName[dish.categoryId]} ·{" "}
                        {formatPrice(dish.price, "UZS", lang)}
                      </div>
                    </td>
                    <td className="px-4 py-2">
                      <select
                        className="input"
                        value={chosen}
                        onChange={(e) => {
                          setLinks({ ...links, [dish.id]: e.target.value });
                          setDirty(true);
                        }}
                      >
                        <option value="">— {t.pos.notMapped} —</option>
                        {products.map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.name}
                            {p.category ? ` · ${p.category}` : ""}
                            {p.unavailable ? ` · ${t.pos.stopped}` : ""}
                          </option>
                        ))}
                      </select>
                      {/* A dish mapped to something the till has stopped is
                          the failure worth catching here, not on an order. */}
                      {product?.unavailable && (
                        <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                          {t.pos.stoppedWarn}
                        </p>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </ListScroll>
        <Pager {...paged} onPage={paged.setPage} />
      </div>
    </div>
  );
}
