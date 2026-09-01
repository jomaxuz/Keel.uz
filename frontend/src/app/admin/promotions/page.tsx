"use client";

// Campaigns and promo codes.
//
// One screen for both, because they are one rule with two triggers: a code the
// guest types, or a window of time that sets itself off. Splitting them would
// mean two forms asking the same eight questions.
//
// The form leads with the trigger, then the discount, then the conditions —
// which is the order an owner thinks in: "a code… for 15% off… on Tuesdays".

import { useEffect, useMemo, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { formatDateTime, formatPrice } from "@/lib/format";
import Modal from "@/components/admin/Modal";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { useAsk } from "@/components/ui/Ask";
import type {
  Category,
  Promotion,
  PromotionKind,
  PromotionScope,
  PromotionTrigger,
  PromotionUsage,
  PromotionStatus,
} from "@/lib/types";

const ORDER_TYPES = ["delivery", "pickup", "dinein"] as const;
// Monday-first, matching the working-hours editor (backend day: 0=Sunday).
const DAY_ORDER = [1, 2, 3, 4, 5, 6, 0];

function emptyDraft(trigger: PromotionTrigger): Promotion {
  return {
    id: "",
    name: "",
    trigger,
    code: "",
    kind: "percent",
    value: 10,
    maxDiscount: 0,
    minOrder: 0,
    scope: "order",
    categoryIds: [],
    days: [],
    timeFrom: "",
    timeTo: "",
    orderTypes: [],
    usageLimit: 0,
    perUserLimit: 0,
    firstOrderOnly: false,
    usedCount: 0,
    isActive: true,
    sortOrder: 0,
  };
}

export default function AdminPromotionsPage() {
  const t = useAdminT();
  const { tell } = useAsk();
  const scope = useAdminScope();
  const [rows, setRows] = useState<Promotion[]>([]);
  const [cats, setCats] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [draft, setDraft] = useState<Promotion | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<PromotionTrigger>("auto");
  // "How many people used this code" — the question the redemption counter
  // cannot answer, because one regular using it five times is one person.
  const [usageOf, setUsageOf] = useState<Promotion | null>(null);
  const [usage, setUsage] = useState<PromotionUsage | null>(null);

  function openUsage(p: Promotion) {
    setUsageOf(p);
    setUsage(null);
    api
      .promotionUsage(p.id)
      .then(setUsage)
      .catch(() => setUsage(null));
  }

  function load() {
    setLoading(true);
    Promise.all([
      api.adminPromotions().catch(() => [] as Promotion[]),
      api.adminCategories().catch(() => [] as Category[]),
    ])
      .then(([p, c]) => {
        setRows(p);
        setCats(c);
      })
      .finally(() => setLoading(false));
  }
  useEffect(load, []);

  const visible = useMemo(
    () => rows.filter((r) => r.trigger === tab),
    [rows, tab],
  );
  const paged = usePaged(visible, 10);

  async function save() {
    if (!draft) return;
    setError(null);
    setSaving(true);
    try {
      const body: Partial<Promotion> = {
        ...draft,
        // Dates arrive from <input type="date"> as "YYYY-MM-DD"; the API wants
        // an instant or nothing at all.
        startsAt: draft.startsAt
          ? new Date(draft.startsAt).toISOString()
          : null,
        endsAt: draft.endsAt ? new Date(draft.endsAt).toISOString() : null,
      };
      if (draft.id) await api.updatePromotion(draft.id, body);
      else await api.createPromotion(body);
      setDraft(null);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function remove(p: Promotion) {
    if (!confirm(t.promo.confirmDelete(p.name))) return;
    try {
      await api.deletePromotion(p.id);
      load();
    } catch {
      void tell({ title: t.common.deleteFailed });
    }
  }

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand";

  // What the rule takes off, in words, for the list.
  function describe(p: Promotion): string {
    switch (p.kind) {
      case "percent":
        return `−${p.value}%${p.maxDiscount ? ` (${t.promo.upTo} ${formatPrice(p.maxDiscount)})` : ""}`;
      case "fixed":
        return `−${formatPrice(p.value)}`;
      default:
        return t.promo.kindFreeDelivery;
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">{t.promo.title}</h1>
        <button
          type="button"
          onClick={() => setDraft(emptyDraft(tab))}
          className="btn-primary px-4 py-2"
        >
          {tab === "code" ? t.promo.addCode : t.promo.addCampaign}
        </button>
      </div>
      <p className="mt-2 text-sm text-ink-muted">{t.promo.hint}</p>

      <div className="mt-4 flex gap-2">
        {(["auto", "code"] as PromotionTrigger[]).map((k) => (
          <button
            key={k}
            type="button"
            onClick={() => setTab(k)}
            className={`rounded-full border px-4 py-1.5 text-sm font-semibold transition-colors ${
              tab === k
                ? "border-brand bg-brand-tint text-brand-dark"
                : "border-line-strong text-ink-soft hover:border-brand"
            }`}
          >
            {k === "auto" ? t.promo.tabCampaigns : t.promo.tabCodes}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : visible.length === 0 ? (
        <p className="mt-6 rounded-2xl border border-line bg-surface p-6 text-sm text-ink-muted">
          {tab === "code" ? t.promo.emptyCodes : t.promo.emptyCampaigns}
        </p>
      ) : (
        <>
          <ListScroll
            className="mt-6 divide-y divide-line rounded-3xl border border-line bg-surface shadow-card"
            max="max-h-[32rem]"
          >
            {paged.pageItems.map((p: Promotion) => (
              <div key={p.id} className="flex items-center gap-3 p-3 text-sm">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="truncate font-medium">{p.name}</span>
                    {p.code && (
                      <span className="rounded-full bg-brand/10 px-2 py-0.5 font-mono text-xs font-bold text-brand">
                        {p.code}
                      </span>
                    )}
                    <StatusBadge
                      status={p.status ?? (p.isActive ? "running" : "off")}
                      t={t}
                    />
                  </div>
                  <p className="text-xs text-ink-muted">
                    {describe(p)}
                    {p.minOrder
                      ? ` · ${t.promo.from} ${formatPrice(p.minOrder)}`
                      : ""}
                    {p.usageLimit
                      ? ` · ${p.usedCount}/${p.usageLimit}`
                      : p.usedCount
                        ? ` · ${t.promo.used} ${p.usedCount}`
                        : ""}
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() =>
                    setDraft({
                      ...p,
                      startsAt: p.startsAt ? p.startsAt.slice(0, 10) : null,
                      endsAt: p.endsAt ? p.endsAt.slice(0, 10) : null,
                    })
                  }
                  className="text-brand hover:underline"
                >
                  {t.common.edit}
                </button>
                <button
                  type="button"
                  onClick={() => openUsage(p)}
                  className="text-ink-muted hover:text-brand"
                >
                  {t.promo.usage}
                </button>
                <button
                  type="button"
                  onClick={() => remove(p)}
                  className="text-ink-muted/70 hover:text-brand"
                >
                  {t.common.delete}
                </button>
              </div>
            ))}
          </ListScroll>
          <Pager
            page={paged.page}
            pageCount={paged.pageCount}
            from={paged.from}
            to={paged.to}
            total={paged.total}
            onPage={paged.setPage}
          />
        </>
      )}

      {draft && (
        <Modal onClose={() => setDraft(null)} wide>
          <h2 className="text-lg font-bold">
            {draft.id ? t.promo.editTitle : t.promo.newTitle}
          </h2>
          <div className="mt-4 space-y-4">
            <label className="block text-sm">
              <span className="font-medium">{t.promo.name}</span>
              <input
                className={inputCls}
                placeholder={t.promo.namePh}
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.promo.nameHint}
              </span>
            </label>

            {draft.trigger === "code" && (
              <label className="block text-sm">
                <span className="font-medium">{t.promo.code}</span>
                <input
                  className={`${inputCls} font-mono uppercase`}
                  placeholder="SALOM15"
                  value={draft.code}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      code: e.target.value
                        .toUpperCase()
                        .replace(/[^A-Z0-9]/g, ""),
                    })
                  }
                />
              </label>
            )}

            {/* What it takes off */}
            <div className="rounded-2xl border border-line p-3">
              <span className="text-sm font-medium">{t.promo.kind}</span>
              <div className="mt-2 flex flex-wrap gap-2">
                {(["percent", "fixed", "freeDelivery"] as PromotionKind[]).map(
                  (k) => (
                    <button
                      key={k}
                      type="button"
                      onClick={() => setDraft({ ...draft, kind: k })}
                      className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                        draft.kind === k
                          ? "border-brand bg-brand-tint text-brand-dark"
                          : "border-line-strong text-ink-soft hover:border-brand"
                      }`}
                    >
                      {k === "percent"
                        ? t.promo.kindPercent
                        : k === "fixed"
                          ? t.promo.kindFixed
                          : t.promo.kindFreeDelivery}
                    </button>
                  ),
                )}
              </div>
              {draft.kind !== "freeDelivery" && (
                <div className="mt-3 grid gap-3 sm:grid-cols-2">
                  <label className="block text-sm">
                    <span className="font-medium">
                      {draft.kind === "percent"
                        ? t.promo.percent
                        : t.promo.amount}
                    </span>
                    <input
                      type="number"
                      className={inputCls}
                      value={draft.value}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          value: Number(e.target.value) || 0,
                        })
                      }
                    />
                  </label>
                  {draft.kind === "percent" && (
                    <label className="block text-sm">
                      <span className="font-medium">{t.promo.maxDiscount}</span>
                      <input
                        type="number"
                        className={inputCls}
                        value={draft.maxDiscount}
                        onChange={(e) =>
                          setDraft({
                            ...draft,
                            maxDiscount: Number(e.target.value) || 0,
                          })
                        }
                      />
                      <span className="mt-1 block text-xs text-ink-muted">
                        {t.promo.maxDiscountHint}
                      </span>
                    </label>
                  )}
                </div>
              )}
            </div>

            {/* What it applies to */}
            <div className="rounded-2xl border border-line p-3">
              <span className="text-sm font-medium">{t.promo.scope}</span>
              <div className="mt-2 flex flex-wrap gap-2">
                {(["order", "category"] as PromotionScope[]).map((sc) => (
                  <button
                    key={sc}
                    type="button"
                    onClick={() => setDraft({ ...draft, scope: sc })}
                    className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                      draft.scope === sc
                        ? "border-brand bg-brand-tint text-brand-dark"
                        : "border-line-strong text-ink-soft hover:border-brand"
                    }`}
                  >
                    {sc === "order"
                      ? t.promo.scopeOrder
                      : t.promo.scopeCategory}
                  </button>
                ))}
              </div>
              {draft.scope === "category" && (
                <div className="mt-3 flex flex-wrap gap-2">
                  {cats.map((c) => {
                    const on = (draft.categoryIds ?? []).includes(c.id);
                    return (
                      <button
                        key={c.id}
                        type="button"
                        onClick={() =>
                          setDraft({
                            ...draft,
                            categoryIds: on
                              ? (draft.categoryIds ?? []).filter(
                                  (x) => x !== c.id,
                                )
                              : [...(draft.categoryIds ?? []), c.id],
                          })
                        }
                        className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                          on
                            ? "border-brand bg-brand-tint text-brand-dark"
                            : "border-line-strong text-ink-soft"
                        }`}
                      >
                        {c.name}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            {/* When it runs */}
            <div className="rounded-2xl border border-line p-3">
              <span className="text-sm font-medium">{t.promo.when}</span>
              <div className="mt-2 grid gap-3 sm:grid-cols-2">
                <label className="block text-sm">
                  <span className="text-ink-muted">{t.promo.startsAt}</span>
                  <input
                    type="date"
                    className={inputCls}
                    value={draft.startsAt ?? ""}
                    onChange={(e) =>
                      setDraft({ ...draft, startsAt: e.target.value || null })
                    }
                  />
                </label>
                <label className="block text-sm">
                  <span className="text-ink-muted">{t.promo.endsAt}</span>
                  <input
                    type="date"
                    className={inputCls}
                    value={draft.endsAt ?? ""}
                    onChange={(e) =>
                      setDraft({ ...draft, endsAt: e.target.value || null })
                    }
                  />
                </label>
                <label className="block text-sm">
                  <span className="text-ink-muted">{t.promo.timeFrom}</span>
                  <input
                    type="time"
                    className={inputCls}
                    value={draft.timeFrom ?? ""}
                    onChange={(e) =>
                      setDraft({ ...draft, timeFrom: e.target.value })
                    }
                  />
                </label>
                <label className="block text-sm">
                  <span className="text-ink-muted">{t.promo.timeTo}</span>
                  <input
                    type="time"
                    className={inputCls}
                    value={draft.timeTo ?? ""}
                    onChange={(e) =>
                      setDraft({ ...draft, timeTo: e.target.value })
                    }
                  />
                </label>
              </div>
              <p className="mt-3 text-xs text-ink-muted">{t.promo.daysHint}</p>
              <div className="mt-1 flex flex-wrap gap-1.5">
                {DAY_ORDER.map((d) => {
                  const on = (draft.days ?? []).includes(d);
                  return (
                    <button
                      key={d}
                      type="button"
                      onClick={() =>
                        setDraft({
                          ...draft,
                          days: on
                            ? (draft.days ?? []).filter((x) => x !== d)
                            : [...(draft.days ?? []), d],
                        })
                      }
                      className={`rounded-lg border px-2.5 py-1 text-xs transition-colors ${
                        on
                          ? "border-brand bg-brand-tint text-brand-dark"
                          : "border-line-strong text-ink-soft"
                      }`}
                    >
                      {t.promo.dayShort[d]}
                    </button>
                  );
                })}
              </div>
              <p className="mt-3 text-xs text-ink-muted">
                {t.promo.orderTypesHint}
              </p>
              <div className="mt-1 flex flex-wrap gap-1.5">
                {ORDER_TYPES.map((ot) => {
                  const on = (draft.orderTypes ?? []).includes(ot);
                  return (
                    <button
                      key={ot}
                      type="button"
                      onClick={() =>
                        setDraft({
                          ...draft,
                          orderTypes: on
                            ? (draft.orderTypes ?? []).filter((x) => x !== ot)
                            : [...(draft.orderTypes ?? []), ot],
                        })
                      }
                      className={`rounded-lg border px-2.5 py-1 text-xs transition-colors ${
                        on
                          ? "border-brand bg-brand-tint text-brand-dark"
                          : "border-line-strong text-ink-soft"
                      }`}
                    >
                      {t.promo.orderType[ot]}
                    </button>
                  );
                })}
              </div>
            </div>

            <label className="block text-sm">
              <span className="font-medium">{t.promo.minOrder}</span>
              <input
                type="number"
                className={inputCls}
                value={draft.minOrder}
                onChange={(e) =>
                  setDraft({ ...draft, minOrder: Number(e.target.value) || 0 })
                }
              />
            </label>

            {/* Limits only make sense for a code: a campaign applies to whoever
                qualifies, and "used up" is not a thing it can be. */}
            {draft.trigger === "code" && (
              <div className="rounded-2xl border border-line p-3">
                <span className="text-sm font-medium">{t.promo.limits}</span>
                <div className="mt-2 grid gap-3 sm:grid-cols-2">
                  <label className="block text-sm">
                    <span className="text-ink-muted">{t.promo.usageLimit}</span>
                    <input
                      type="number"
                      className={inputCls}
                      value={draft.usageLimit}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          usageLimit: Number(e.target.value) || 0,
                        })
                      }
                    />
                  </label>
                  <label className="block text-sm">
                    <span className="text-ink-muted">
                      {t.promo.perUserLimit}
                    </span>
                    <input
                      type="number"
                      className={inputCls}
                      value={draft.perUserLimit}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          perUserLimit: Number(e.target.value) || 0,
                        })
                      }
                    />
                  </label>
                </div>
                <label className="mt-3 flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={draft.firstOrderOnly}
                    onChange={(e) =>
                      setDraft({ ...draft, firstOrderOnly: e.target.checked })
                    }
                  />
                  <span>{t.promo.firstOrderOnly}</span>
                </label>
                <p className="mt-2 text-xs text-ink-muted">
                  {t.promo.limitsHint}
                </p>
              </div>
            )}

            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={draft.isActive}
                onChange={(e) =>
                  setDraft({ ...draft, isActive: e.target.checked })
                }
              />
              <span className="font-medium">{t.promo.active}</span>
            </label>

            {error && (
              <p className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand dark:bg-rose-500/10 dark:text-rose-300">
                {error}
              </p>
            )}
          </div>

          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setDraft(null)}
              className="px-4 py-2 text-sm text-ink-muted hover:text-ink"
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

      {usageOf && (
        <Modal onClose={() => setUsageOf(null)}>
          <h2 className="text-lg font-bold">{usageOf.name}</h2>
          {usageOf.code && (
            <p className="mt-1 font-mono text-sm text-brand">{usageOf.code}</p>
          )}
          {!usage ? (
            <p className="py-8 text-center text-ink-muted/70">
              {t.common.loading}
            </p>
          ) : (
            <>
              <div className="mt-4 grid grid-cols-3 gap-3 text-center">
                <Stat
                  label={t.promo.statPeople}
                  value={String(usage.stats.people)}
                />
                <Stat
                  label={t.promo.statRedemptions}
                  value={String(usage.stats.redemptions)}
                />
                <Stat
                  label={t.promo.statDiscounted}
                  value={formatPrice(usage.stats.discounted)}
                />
              </div>
              {usage.orders.length === 0 ? (
                <p className="mt-4 text-sm text-ink-muted">
                  {t.promo.notUsedYet}
                </p>
              ) : (
                <ListScroll
                  className="mt-4 divide-y divide-line rounded-2xl border border-line"
                  max="max-h-72"
                >
                  {usage.orders.map((o) => (
                    <div
                      key={o.orderId}
                      className="flex items-center gap-3 px-3 py-2 text-sm"
                    >
                      <div className="min-w-0 flex-1">
                        <p className="truncate font-medium">
                          #{o.number} · {o.customer}
                        </p>
                        <p className="text-xs text-ink-muted">
                          {formatDateTime(o.createdAt)} · {o.phone}
                        </p>
                      </div>
                      <span className="shrink-0 tabular-nums text-emerald-700 dark:text-emerald-400">
                        −{formatPrice(o.amount)}
                      </span>
                    </div>
                  ))}
                </ListScroll>
              )}
            </>
          )}
          <div className="mt-6 flex justify-end">
            <button
              type="button"
              onClick={() => setUsageOf(null)}
              className="btn-ghost px-4 py-2 text-sm"
            >
              {t.common.close}
            </button>
          </div>
        </Modal>
      )}

      {/* Campaigns are brand-wide; the switcher above says which brand. */}
      {scope.multi && (
        <p className="mt-4 text-xs text-ink-muted">
          {t.promo.brandNote(scope.brand?.name ?? "")}
        </p>
      )}
    </div>
  );
}

// StatusBadge says what the rule is doing, not just whether a checkbox is
// ticked: "expired", "used up" and "not today" are all different reasons a code
// is quietly doing nothing, and the owner needs to tell them apart.
function StatusBadge({
  status,
  t,
}: {
  status: PromotionStatus;
  t: ReturnType<typeof useAdminT>;
}) {
  const tone: Record<PromotionStatus, string> = {
    running: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
    idle: "bg-ink/5 text-ink-muted",
    scheduled: "bg-sky-500/15 text-sky-700 dark:text-sky-300",
    expired: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
    usedUp: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
    off: "bg-ink/5 text-ink-muted",
  };
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-xs font-semibold ${tone[status]}`}
    >
      {t.promo.status[status]}
    </span>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-line p-3">
      <p className="text-lg font-bold">{value}</p>
      <p className="text-xs text-ink-muted">{label}</p>
    </div>
  );
}
