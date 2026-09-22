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
import { usePanelWords, type PanelWords } from "@/lib/panelWords";
import { useAdminScope } from "@/lib/adminScope";
import type { StopList, StopListItem } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";
import Modal from "@/components/admin/Modal";
import {
  holdLeft,
  stopHoldBody,
  typedHold,
  STOP_HOLD_PRESETS,
  type StopHold,
} from "@/lib/stopHold";

export default function AdminStopListPage() {
  const t = useAdminT();
  const w = usePanelWords();
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
  /** The dish whose deadline is being chosen. ⚠️ Only ever set on the way
   *  *off*: putting a dish back has no duration, and a dialog in front of it
   *  would be a question nobody asked. */
  const [holding, setHolding] = useState<StopListItem | null>(null);

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

  /** Keep the dish off, but say when it comes back.
   *
   *  ⚠️ **Not the toggle with a flag.** Sending `soldOut: false` and stopping it
   *  again would put the dish on sale for as long as the round trip takes, and
   *  that window is exactly when a guest orders it. The server has always
   *  accepted a deadline on a dish it was already holding; only the screens
   *  refused to ask. */
  async function hold(row: StopListItem, chosen: StopHold) {
    if (!branch) return;
    setHolding(null);
    setBusy(row.menuItemId);
    try {
      await api.setSoldOut(branch.id, row.menuItemId, true, stopHoldBody(chosen));
      load();
    } catch (e: unknown) {
      void tell({ title: e instanceof Error ? e.message : t.common.saveFailed });
    } finally {
      setBusy(null);
    }
  }

  // Optimistic, like the menu screen's toggle: this is pressed while somebody is
  // waiting at the counter, and a row that only changes after a round trip gets
  // pressed twice.
  /** The click. Opens the question rather than doing the thing. */
  function press(row: StopListItem) {
    if (!branch || row.pos || row.stock) return;
    // ⚠️ **The dialog opens either way now.** A dish already off is the case
    // the timer is most used for — the kitchen has been rung, the answer is
    // "fifteen minutes", and until this was here the only way to record it was
    // to put the dish back on sale and stop it again.
    setHolding(row);
  }

  async function toggle(row: StopListItem, hold?: StopHold) {
    if (!branch || row.pos || row.stock) return;
    setHolding(null);
    const next = !row.manual;
    setBusy(row.menuItemId);
    setData((cur) =>
      cur
        ? {
            ...cur,
            items: cur.items.map((i) =>
              i.menuItemId === row.menuItemId
                ? // ⚠️ The deadline is cleared rather than guessed. It is the
                  // server's arithmetic, and a badge drawn from a guess would
                  // name a time the dish does not actually come back.
                  { ...i, manual: next, until: undefined }
                : i,
            ),
          }
        : cur,
    );
    try {
      await api.setSoldOut(
        branch.id,
        row.menuItemId,
        next,
        stopHoldBody(hold ?? null),
      );
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
        void tell({ title: w.stopPosSynced(res.stopped) });
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
        void tell({ title: w.stopSynced(res.stopped) });
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
            {w.stopLead}
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
          placeholder={w.search}
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
            ? w.stopNoItems
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
              onToggle={() => press(row)}
              onExpired={load}
              t={t}
              w={w}
            />
          ))}
        </ListScroll>
      )}

      {holding && (
        <HoldDialog
          row={holding}
          onCancel={() => setHolding(null)}
          onConfirm={(chosen) => void toggle(holding, chosen)}
          onHold={(chosen) => void hold(holding, chosen)}
        />
      )}
    </div>
  );
}

/**
 * How long the stop should hold.
 *
 * ⚠️ **The same choices the counter offers**, and the rule behind them is the
 * same module (`lib/stopHold`). Two readings of "2 hours" in one product is how
 * a restaurant ends up unable to say which screen is wrong — the reason
 * `soldOutHeldBy` is one function on the server too. What differs here is only
 * the markup: the till draws controls sized for a thumb on a monoblock, this
 * draws a form.
 *
 * ⚠️ **Open-ended is preselected**, because that is what this button did before
 * deadlines existed. A duration chosen on somebody's behalf would put a dish
 * back on the menu that is genuinely gone, and nobody notices until a guest
 * orders it.
 */
function HoldDialog({
  row,
  onCancel,
  onConfirm,
  onHold,
}: {
  row: StopListItem;
  onCancel: () => void;
  /** Stop it, or put it back — the toggle, with an optional deadline. */
  onConfirm: (hold: StopHold) => void;
  /** Keep it off and only say when it returns. Only for a dish already off. */
  onHold: (hold: StopHold) => void;
}) {
  const t = useAdminT();
  const stopping = !row.manual;
  const [hold, setHold] = useState<StopHold>(null);
  /** ⚠️ Kept beside `hold` rather than derived from it: the presets write
   *  numbers there too, and a box that filled itself in when somebody tapped
   *  "1 soat" would leave the next keystroke editing a figure they never
   *  typed. */
  const [minutes, setMinutes] = useState("");

  return (
    <Modal onClose={onCancel}>
      <h2 className="text-lg font-semibold">
        {stopping ? t.stopList.stop : t.stopList.unstop}
      </h2>
      <p className="mt-1 text-sm text-ink-muted">
        {stopping ? t.stopList.holdBody(row.name) : t.stopList.backBody(row.name)}
      </p>

      {/* ⚠️ **Two different questions behind one control.** Stopping asks how
          long the dish is off; a dish already off asks when it is ready, which
          is what somebody just rang the kitchen to find out. */}
      <h3 className="mt-4 text-sm font-semibold text-ink">
        {stopping ? t.till.stopHoldTitle : t.till.stopReadyTitle}
      </h3>
      <div className="mt-2 flex flex-wrap gap-1.5">
        {(stopping
          ? // ⚠️ Open-ended only when stopping. Offering it to a dish already
            // off would be a button that changes nothing.
            [null as StopHold, ...STOP_HOLD_PRESETS]
          : STOP_HOLD_PRESETS
        ).map((value) => (
          <button
            key={String(value)}
            type="button"
            onClick={() => {
              setHold(value);
              setMinutes("");
            }}
            className={`rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors ${
              hold === value
                ? "border-brand bg-brand/10 text-brand"
                : "border-line-strong text-ink-muted hover:border-brand hover:text-brand"
            }`}
          >
            {value === null
              ? t.till.stopHoldOpen
              : value === "close"
                ? t.till.stopHoldClose
                : value < 60
                  ? t.till.stopHoldMins(value)
                  : t.till.stopHoldHours(value / 60)}
          </button>
        ))}
      </div>

      {/* ⚠️ No "≈ 21:35" preview here either. The instant is the server's
          arithmetic — the panel's clock is more trustworthy than a monoblock's,
          but showing a figure from one screen that the other computes is how
          the two start disagreeing about the same stop. */}
      <label className="mt-2 flex items-center gap-2 text-sm text-ink-muted">
        {t.till.stopHoldOr}
        <input
          className="input w-20 text-center"
          inputMode="numeric"
          value={minutes}
          placeholder={t.till.stopHoldMinutesPh}
          onChange={(e) => {
            const { text, hold: next } = typedHold(e.target.value);
            setMinutes(text);
            setHold(next);
          }}
        />
        {t.till.stopHoldMinutes}
      </label>

      <div className="mt-5 flex flex-wrap justify-end gap-2">
        <button type="button" className="btn-ghost" onClick={onCancel}>
          {t.common.cancel}
        </button>
        {/* ⚠️ **Two outcomes that far apart do not share a button.** One keeps
            the dish off and only records when it returns; the other puts it on
            sale now. */}
        {!stopping && (
          <button
            type="button"
            className="btn-primary"
            disabled={hold === null}
            onClick={() => onHold(hold)}
          >
            {t.till.stopReadySave}
          </button>
        )}
        <button
          type="button"
          className={stopping ? "btn-primary" : "btn-ghost"}
          onClick={() => onConfirm(hold)}
        >
          {stopping ? t.stopList.stop : t.stopList.unstop}
        </button>
      </div>
    </Modal>
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
  onExpired,
  t,
  w,
}: {
  row: StopListItem;
  busy: boolean;
  onToggle: () => void;
  /** A timed stop reached zero. ⚠️ The list is refetched rather than flipped
   *  here: the server is what lifts the stop, and a row that decided on its own
   *  would be a second opinion about whether a dish is on sale. */
  onExpired: () => void;
  t: ReturnType<typeof useAdminT>;
  /** What this business calls what it sells — see lib/panelWords.ts. */
  w: PanelWords;
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
              {row.until ? (
                <Countdown until={row.until} onDone={onExpired} />
              ) : (
                t.stopList.manualBadge
              )}
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
          title={row.pos ? w.stopPosLocked : t.stopList.stockLocked}
          className="max-w-56 text-right text-xs text-ink-muted"
        >
          {row.pos ? w.stopPosLocked : t.stopList.stockLocked}
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

/**
 * How long is left of a timed stop, ticking.
 *
 * ⚠️ **Its own component so one second does not redraw the whole list.** This
 * page is left open on an office screen, and a tick in the parent would
 * re-render two hundred rows a second to move one number.
 *
 * ⚠️ **Counted against the server's instant**, never decremented locally: a page
 * open all afternoon drifts, and a drifting counter would say "2 min" about a
 * dish that came back ten minutes ago — worse than no counter, because somebody
 * would act on it.
 */
function Countdown({ until, onDone }: { until: string; onDone: () => void }) {
  const t = useAdminT();
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  const left = holdLeft(until, now);
  // Fired from an effect rather than during render: calling the parent's
  // setState while rendering is the warning React shows and the update it drops.
  useEffect(() => {
    if (left?.done) onDone();
  }, [left?.done, onDone]);

  if (!left || left.done) return null;
  return (
    <>
      {left.hours > 0
        ? t.till.stopLeftHm(left.hours, left.minutes)
        : left.minutes > 0
          ? t.till.stopLeftM(left.minutes)
          : t.till.stopLeftS(left.seconds)}
    </>
  );
}
