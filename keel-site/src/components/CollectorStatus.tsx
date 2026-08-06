"use client";

// Why the charts are empty.
//
// The overview could show an empty 30-day chart, an empty top-customers list
// and zeroes across the board, and there was **no way at all to tell "nobody
// has ordered yet" from "the collector has never successfully run"**. Both look
// identical, and only one of them is somebody's problem — which is exactly the
// class of silent failure this system keeps having to design against.
//
// So the collector writes down what it did, and this line reads it back:
// when it ran, how many customer databases it reached, how many rows that
// produced, and which ones failed. Plus the button that answers the question in
// one press instead of waiting an hour for the next tick.
//
// Shown only when it has something to explain — a run that reached every
// database and produced rows needs no commentary, and a permanent status line
// is read as decoration within a week.

import { useState } from "react";
import { useT } from "@/lib/i18n/client";
import { collectNow, shortDate, type CollectorRun } from "@/lib/api";

export default function CollectorStatus({
  run,
  onDone,
}: {
  run: CollectorRun | null;
  onDone: () => void;
}) {
  const { t } = useT();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function collect() {
    setBusy(true);
    setError("");
    try {
      const out = await collectNow();
      if (out.error) setError(out.error);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const never = !run;
  const noRows = !!run && run.tenants > 0 && run.rows === 0;
  const failed = !!run && run.failed > 0;

  // Nothing to explain: it ran, it reached everyone, it found data.
  if (!never && !noRows && !failed && !error) return null;

  return (
    <div className="rounded-2xl border border-line bg-surface p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0 text-sm">
          {never ? (
            <p className="text-ink">{t.dash.collectorNever}</p>
          ) : (
            <>
              {/* The sentence that resolves the ambiguity, first. */}
              {noRows && <p className="text-ink">{t.dash.collectorNoRows}</p>}
              {failed && (
                <p className="text-rose-600 dark:text-rose-400">
                  {t.dash.collectorFailed(run.failed)}
                </p>
              )}
              <p className="mt-1 text-xs text-ink-muted">
                {t.dash.collectorAt(shortDate(run.at))} ·{" "}
                {t.dash.collectorReached(run.ok, run.tenants)} ·{" "}
                {t.dash.collectorRows(run.rows)}
              </p>
            </>
          )}
          {/* The useful sentence is almost never ours — it is a database
              refusing a connection. Shown verbatim. */}
          {run?.errors?.map((e) => (
            <p key={e} className="mt-1 break-words text-xs text-ink-muted">
              {e}
            </p>
          ))}
          {error && (
            <p className="mt-1 text-xs text-rose-600 dark:text-rose-400">{error}</p>
          )}
        </div>
        <button
          type="button"
          onClick={collect}
          disabled={busy}
          className="shrink-0 rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface disabled:opacity-50"
        >
          {busy ? t.dash.collectorRunning : t.dash.collectorRun}
        </button>
      </div>
    </div>
  );
}
