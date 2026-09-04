"use client";

// Stop list: what is off sale at this branch right now, and why.
//
// The menu screen already has a "sold out" toggle on every row, and it stays
// there — mid-service the counter is usually already looking at the dish. This
// screen exists for the other direction: the owner who wants to see *only* what
// is off, in one place, without reading past two hundred dishes that are fine.
//
// It also has the second half of the feature, which has no other home. A
// restaurant running iiko or Poster stops a dish once, in the till, and the
// mirror brings it here (backend: handlers/posstop.go). Those rows cannot be
// lifted from here — the switch is over there — and the screen says so instead
// of offering a toggle that springs back three minutes later.

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { formatDateTime, formatPrice } from "@/lib/format";
import { ListScroll } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import type { StopList, StopListItem } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

export default function AdminStopListPage() {
  const t = useAdminT();
  const { tell } = useAsk();
  const scope = useAdminScope();
  const branch = scope.branch;

  const [data, setData] = useState<StopList | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [offOnly, setOffOnly] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [syncingStock, setSyncingStock] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);

  const load = useCallback(() => {
    if (!branch) {
      setData(null);
      setLoading(false);
      return;
    }
    setLoading(true);
    api
      .adminStopList()
      .then((res) => {
        setData(res);
        setError("");
      })
      .catch((e: unknown) =>
        setError(e instanceof Error ? e.message : String(e)),
      )
      .finally(() => setLoading(false));
  }, [branch]);

  useEffect(load, [load, scope.scopeKey]);

  // Optimistic, like the menu screen's toggle: this is pressed while somebody is
  // waiting at the counter, and a row that only changes after a round trip gets
  // pressed twice.
  async function toggle(row: StopListItem) {
    if (!branch || row.pos || row.stock) return;
    const next = !row.manual;
    setBusy(row.menuItemId);
    setData((cur) =>
      cur
        ? {
            ...cur,
            items: cur.items.map((i) =>
              i.menuItemId === row.menuItemId ? { ...i, manual: next } : i,
            ),
          }
        : cur,
    );
    try {
      await api.setSoldOut(branch.id, row.menuItemId, next);
      scope.reload();
    } catch (e: unknown) {
      setData((cur) =>
        cur
          ? {
              ...cur,
              items: cur.items.map((i) =>
                i.menuItemId === row.menuItemId ? { ...i, manual: !next } : i,
              ),
            }
          : cur,
      );
      void tell({
        title: e instanceof Error ? e.message : t.common.saveFailed,
      });
    } finally {
      setBusy(null);
    }
  }

  async function syncNow() {
    setSyncing(true);
    try {
      const res = await api.syncPOSStopList();
      if (!res.ok) void tell({ title: res.message ?? t.common.saveFailed });
      else if (typeof res.stopped === "number")
        void tell({ title: t.stopList.posSynced(res.stopped) });
      load();
      scope.reload();
    } catch (e: unknown) {
      void tell({
        title: e instanceof Error ? e.message : t.common.saveFailed,
      });
    } finally {
      setSyncing(false);
    }
  }

  // ⚠️ Its own button and its own busy flag: the two syncs talk to completely
  // different things — somebody else's till, and our own arithmetic — and one
  // spinner over both would leave the owner unable to tell which one is slow.
  async function syncStockNow() {
    setSyncingStock(true);
    try {
      const res = await api.syncStockStopList();
      if (!res.ok) void tell({ title: res.message ?? t.common.saveFailed });
      else if (typeof res.stopped === "number")
        void tell({ title: t.stopList.stockSynced(res.stopped) });
      load();
      scope.reload();
    } catch (e: unknown) {
      void tell({
        title: e instanceof Error ? e.message : t.common.saveFailed,
      });
    } finally {
      setSyncingStock(false);
    }
  }

  async function toggleStockStop(enabled: boolean) {
    setSyncingStock(true);
    try {
      await api.setStockStop(enabled);
      load();
      scope.reload();
    } catch (e: unknown) {
      void tell({
        title: e instanceof Error ? e.message : t.common.saveFailed,
      });
    } finally {
      setSyncingStock(false);
    }
  }

  const items = useMemo(() => data?.items ?? [], [data]);
  const offCount = useMemo(
    () => items.filter((i) => i.manual || i.pos || i.stock).length,
    [items],
  );
  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter((i) => {
      if (offOnly && !i.manual && !i.pos && !i.stock) return false;
      if (q && !i.name.toLowerCase().includes(q)) return false;
      return true;
    });
  }, [items, query, offOnly]);

  if (!branch) {
    return (
      <div>
        <h1 className="text-2xl font-bold">{t.stopList.title}</h1>
        <p className="mt-4 rounded-2xl border border-line bg-surface px-4 py-3 text-sm text-ink-muted">
          {t.stopList.branchNeeded}
        </p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">{t.stopList.title}</h1>
          <p className="mt-1 max-w-2xl text-sm text-ink-muted">
            {t.stopList.subtitle}
          </p>
        </div>
        <span className="rounded-full bg-amber-500/15 px-3 py-1 text-sm font-semibold text-amber-700 dark:text-amber-300">
          {t.stopList.offNow(offCount)}
        </span>
      </div>

      <PosPanel data={data} syncing={syncing} onSync={syncNow} t={t} />
      <StockPanel
        data={data}
        syncing={syncingStock}
        onSync={syncStockNow}
        onToggle={toggleStockStop}
        t={t}
      />

      <div className="mt-6 flex flex-wrap items-center gap-2">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t.stopList.search}
          className="min-w-56 flex-1 rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
        />
        {/* Two buttons rather than a dropdown: "what is off right now" is the
            reason the screen was opened, and it should be one tap away. */}
        <FilterButton active={!offOnly} onClick={() => setOffOnly(false)}>
          {t.stopList.showAll}
        </FilterButton>
        <FilterButton active={offOnly} onClick={() => setOffOnly(true)}>
          {t.stopList.showOff}
        </FilterButton>
      </div>

      {error && (
        <p className="mt-4 rounded-xl bg-rose-500/10 px-4 py-3 text-sm text-rose-700 dark:text-rose-300">
          {error}
        </p>
      )}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : shown.length === 0 ? (
        <p className="mt-6 rounded-2xl border border-line bg-surface px-4 py-6 text-center text-sm text-ink-muted">
          {items.length === 0
            ? t.stopList.noItems
            : offOnly
              ? t.stopList.nothingOff
              : t.common.notFound}
        </p>
      ) : (
        <ListScroll
          className="mt-4 divide-y divide-line rounded-3xl border border-line bg-surface shadow-card"
          max="max-h-[42rem]"
        >
          {shown.map((row) => (
            <Row
              key={row.menuItemId}
              row={row}
              busy={busy === row.menuItemId}
              onToggle={() => toggle(row)}
              t={t}
            />
          ))}
        </ListScroll>
      )}
    </div>
  );
}

function FilterButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full border px-3 py-1.5 text-sm font-medium transition-colors ${
        active
          ? "border-brand bg-brand/10 text-brand"
          : "border-line-strong text-ink-muted hover:border-brand hover:text-brand"
      }`}
    >
      {children}
    </button>
  );
}

// The till half of the screen.
//
// ⚠️ The useful line here is **when it was last read**, not whether it is
// connected. Credentials that were right this morning are right now too, and the
// only way to tell a working mirror from one that stopped an hour ago is a time.
function PosPanel({
  data,
  syncing,
  onSync,
  t,
}: {
  data: StopList | null;
  syncing: boolean;
  onSync: () => void;
  t: ReturnType<typeof useAdminT>;
}) {
  if (!data) return null;
  const pos = data.pos;
  if (!pos.connected) {
    return (
      <p className="mt-4 rounded-2xl border border-line bg-surface px-4 py-3 text-sm text-ink-muted">
        {t.stopList.posOff}
      </p>
    );
  }
  return (
    <div className="mt-4 rounded-2xl border border-line bg-surface px-4 py-3 shadow-card">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="text-sm">
          <p className="font-semibold">
            {t.stopList.posTitle}
            <span className="ml-2 font-normal text-ink-muted">
              {t.stopList.posConnected(pos.provider)}
            </span>
          </p>
          <p className="mt-0.5 text-ink-muted">
            {pos.syncedAt
              ? t.stopList.posSyncedAt(formatDateTime(pos.syncedAt))
              : t.stopList.posNever}
            {" · "}
            {t.stopList.posEvery(pos.everyMins)}
          </p>
          {/* Named, rather than left to look stale. The timestamp above is the
              honest one; this is the missing half of its explanation. */}
          {pos.paused && (
            <p className="mt-0.5 text-ink-muted">{t.stopList.posPaused}</p>
          )}
        </div>
        <button
          type="button"
          onClick={onSync}
          disabled={syncing}
          className="btn-primary px-4 py-2 text-sm disabled:opacity-60"
        >
          {syncing ? t.stopList.posSyncing : t.stopList.posSyncNow}
        </button>
      </div>
      {pos.syncError && (
        <p className="mt-2 rounded-xl bg-rose-500/10 px-3 py-2 text-sm text-rose-700 dark:text-rose-300">
          {pos.syncError}
        </p>
      )}
      {pos.mappedItem === 0 && (
        <p className="mt-2 rounded-xl bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-300">
          {t.stopList.posNoMapping}
        </p>
      )}
    </div>
  );
}

/** The store's half of the same screen.
 *
 *  ⚠️ **The timestamp is the useful line, not the switch.** A stored "on" flag
 *  goes stale the moment the clock passes it, and a stock stop list that stopped
 *  being worked out at lunchtime looks exactly like one with nothing stopped —
 *  the same rule as the POS panel above it.
 *
 *  ⚠️ Nothing is drawn at all where the branch has not switched it on, which is
 *  every branch by default: a panel explaining a feature that is doing nothing
 *  is a panel that teaches people to skim this screen. */
function StockPanel({
  data,
  onSync,
  onToggle,
  syncing,
  t,
}: {
  data: StopList | null;
  onSync: () => void;
  onToggle: (enabled: boolean) => void;
  syncing: boolean;
  t: ReturnType<typeof useAdminT>;
}) {
  if (!data?.stock) return null;
  const stock = data.stock;

  // ⚠️ **Off is a state with an explanation, not an absent panel.** Refusing a
  // sale is the most expensive thing this system can do, so it is opt-in — and
  // an owner who has never seen the switch cannot opt in. What they get here is
  // the one sentence that makes the decision: it only works if the deliveries
  // and the counts are actually being entered.
  if (!stock.enabled) {
    return (
      <div className="mt-3 rounded-2xl border border-line bg-surface px-4 py-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="text-sm">
            <p className="font-semibold">{t.stopList.stockTitle}</p>
            <p className="mt-0.5 text-xs text-ink-muted">
              {t.stopList.stockOffHint}
            </p>
          </div>
          <button
            type="button"
            onClick={() => onToggle(true)}
            disabled={syncing}
            className="btn px-4 py-2 text-sm disabled:opacity-60"
          >
            {t.stopList.stockEnable}
          </button>
        </div>
      </div>
    );
  }
  return (
    <div className="mt-3 rounded-2xl border border-line bg-surface px-4 py-3 shadow-card">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="text-sm">
          <p className="font-semibold">{t.stopList.stockTitle}</p>
          <p className="mt-0.5 text-ink-muted">
            {stock.syncedAt
              ? t.stopList.stockSyncedAt(formatDateTime(stock.syncedAt))
              : t.stopList.stockNever}
            {" · "}
            {t.stopList.posEvery(stock.everyMins)}
          </p>
          {/* ⚠️ Said every time, not once in a tooltip. The figure behind this
              is an estimate — the last count plus deliveries less what the
              cards account for — and an owner who forgets that will read a
              stopped dish as a fact about the shelf rather than about the
              paperwork. */}
          <p className="mt-0.5 text-xs text-ink-muted">
            {t.stopList.stockHint}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => onToggle(false)}
            disabled={syncing}
            className="btn-ghost px-3 py-2 text-sm disabled:opacity-60"
          >
            {t.stopList.stockDisable}
          </button>
          <button
            type="button"
            onClick={onSync}
            disabled={syncing}
            className="btn px-4 py-2 text-sm disabled:opacity-60"
          >
            {syncing ? t.stopList.posSyncing : t.stopList.stockSyncNow}
          </button>
        </div>
      </div>
    </div>
  );
}

function Row({
  row,
  busy,
  onToggle,
  t,
}: {
  row: StopListItem;
  busy: boolean;
  onToggle: () => void;
  t: ReturnType<typeof useAdminT>;
}) {
  const off = row.manual || row.pos || row.stock;
  return (
    <div className="flex items-center gap-3 p-3">
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className={`truncate font-medium ${off ? "" : "text-ink"}`}>
            {row.name}
          </span>
          {row.pos && (
            <span className="rounded-full bg-rose-500/15 px-1.5 text-xs font-semibold text-rose-700 dark:text-rose-300">
              {t.stopList.posBadge}
            </span>
          )}
          {/* ⚠️ Its own badge, not the till's. The two are lifted in
              completely different places — one at the counter over there, this
              one by recording a delivery or counting a shelf — and a cashier
              who cannot tell them apart goes to the wrong screen. */}
          {row.stock && (
            <span className="rounded-full bg-sky-500/15 px-1.5 text-xs font-semibold text-sky-700 dark:text-sky-300">
              {t.stopList.stockBadge}
            </span>
          )}
          {row.manual && (
            <span className="rounded-full bg-amber-500/15 px-1.5 text-xs font-semibold text-amber-700 dark:text-amber-300">
              {/* ⚠️ **The deadline replaces the word where there is one.** The
                  counter can stop a dish for two hours, and an owner reading
                  "stopped" here would have no way to tell that from a dish
                  taken off for good — and would go and ask. */}
              {row.until
                ? t.stopList.manualUntil(
                    new Date(row.until).toLocaleTimeString([], {
                      hour: "2-digit",
                      minute: "2-digit",
                    }),
                  )
                : t.stopList.manualBadge}
            </span>
          )}
          {row.hidden && (
            <span className="rounded-full bg-ink/5 px-1.5 text-xs text-ink-muted">
              {t.stopList.hiddenBadge}
            </span>
          )}
        </div>
        <p className="truncate text-xs text-ink-muted/70">
          {row.category}
          {row.posProduct ? ` · ${row.posProduct}` : ""}
        </p>
      </div>
      <span className="font-semibold">{formatPrice(row.price)}</span>
      {row.pos || row.stock ? (
        // No toggle at all, and the reason next to it. A disabled button with no
        // explanation reads as a bug in the panel rather than a fact about the
        // till — and a toggle that worked and then sprang back within minutes
        // would be worse than either.
        <span
          title={row.pos ? t.stopList.posLocked : t.stopList.stockLocked}
          className="max-w-56 text-right text-xs text-ink-muted"
        >
          {row.pos ? t.stopList.posLocked : t.stopList.stockLocked}
        </span>
      ) : (
        <button
          type="button"
          onClick={onToggle}
          disabled={busy}
          className={`rounded-full border px-3 py-1 text-xs font-semibold transition-colors disabled:opacity-60 ${
            row.manual
              ? "border-amber-500/50 bg-amber-500/10 text-amber-700 dark:text-amber-300"
              : "border-line-strong text-ink-muted hover:border-brand hover:text-brand"
          }`}
        >
          {row.manual ? t.stopList.unstop : t.stopList.stop}
        </button>
      )}
    </div>
  );
}
