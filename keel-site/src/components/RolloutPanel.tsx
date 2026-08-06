"use client";

// "How many customers are running old code, and one button to fix it."
//
// This sits on the overview page because that is where somebody looks right
// after a deploy. The count is the honest half: it comes from live container
// image ids, not from whether a deploy reported success — a green pipeline and
// fifty containers on last month's image are the same picture from here.
//
// Deliberately quiet when there is nothing to do. A permanent "0 stale" row is
// read as decoration within a week, and then the day it says 12 nobody notices.

import { useCallback, useEffect, useRef, useState } from "react";
import { useT } from "@/lib/i18n/client";
import { rollout, startRollout, shortDate, type RolloutState } from "@/lib/api";

export default function RolloutPanel() {
  const { t } = useT();
  const [state, setState] = useState<RolloutState | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const load = useCallback(async () => {
    try {
      setState(await rollout());
    } catch {
      /* the overview's own error already says the console is unreachable */
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // Poll only while something is moving. A rollout of fifty tenants takes
  // minutes and each one waits on a health check, so five seconds is plenty —
  // and an idle console has no reason to ask at all.
  const running = state?.running ?? false;
  useEffect(() => {
    if (!running) return;
    timer.current = setTimeout(load, 5000);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [running, state, load]);

  async function run() {
    setBusy(true);
    setError("");
    try {
      await startRollout();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!state) return null;
  if (!state.enabled) return null;

  const stale = state.staleCount ?? 0;
  const last = state.last;
  // Nothing to do and nothing in flight: stay out of the way.
  if (stale === 0 && !running && !error) return null;

  return (
    <div className="rounded-2xl border border-line bg-surface p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-semibold text-ink">{t.dash.rolloutTitle}</p>
          <p className="mt-1 text-sm text-ink-soft">
            {stale > 0 ? t.dash.rolloutStale(stale) : t.dash.rolloutUpToDate}
          </p>
        </div>
        <button
          type="button"
          onClick={run}
          disabled={busy || running}
          className="shrink-0 rounded-xl bg-ink px-4 py-2 text-sm font-semibold text-surface disabled:opacity-50"
        >
          {running || busy ? t.dash.rolloutRunning : t.dash.rolloutRun}
        </button>
      </div>

      <p className="mt-2 text-xs text-ink-muted">{t.dash.rolloutWhy}</p>

      {error && (
        <p className="mt-2 text-sm text-rose-600 dark:text-rose-400">{error}</p>
      )}
      {state.error && (
        <p className="mt-2 text-sm text-rose-600 dark:text-rose-400">
          {state.error}
        </p>
      )}

      {last && (
        <div className="mt-3 border-t border-line pt-3 text-xs text-ink-soft">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span className="tabular-nums font-semibold">
              {t.dash.rolloutProgress(last.done, last.total)}
            </span>
            {/* A stuck rollout names the customer holding it up. A percentage
                would not tell anyone which container to go and look at. */}
            {running && last.current && (
              <span>{t.dash.rolloutCurrent(last.current)}</span>
            )}
            {!running && last.status === "done" && (
              <span className="text-emerald-700 dark:text-emerald-300">
                {t.dash.rolloutDone}
              </span>
            )}
            {!running && last.status !== "done" && last.status !== "running" && (
              <span className="text-amber-700 dark:text-amber-300">
                {t.dash.rolloutAborted}
              </span>
            )}
            {last.failed > 0 && (
              <span className="text-rose-600 dark:text-rose-400">
                {t.dash.rolloutFailedN(last.failed)}
              </span>
            )}
            {last.finishedAt && !running && (
              <span className="text-ink-muted">
                {t.dash.rolloutLast}: {shortDate(last.finishedAt)}
              </span>
            )}
          </div>
          {last.note && <p className="mt-1 text-ink-muted">{last.note}</p>}

          {last.items?.length > 0 && (
            <>
              <button
                type="button"
                className="mt-2 underline underline-offset-2"
                onClick={() => setOpen((v) => !v)}
              >
                {open ? "▾" : "▸"} {last.items.length}
              </button>
              {open && (
                <ul className="mt-2 space-y-1">
                  {last.items.map((it) => (
                    <li key={it.slug} className="flex gap-2">
                      <span className="w-40 shrink-0 truncate font-medium text-ink">
                        {it.name || it.slug}
                      </span>
                      <span
                        className={
                          it.status === "failed"
                            ? "text-rose-600 dark:text-rose-400"
                            : "text-ink-muted"
                        }
                      >
                        {it.status === "updated" && t.dash.rolloutItemUpdated}
                        {it.status === "current" && t.dash.rolloutItemCurrent}
                        {it.status === "skipped" && t.dash.rolloutItemSkipped}
                        {it.status === "failed" && t.dash.rolloutItemFailed}
                        {it.note ? ` — ${it.note}` : ""}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
