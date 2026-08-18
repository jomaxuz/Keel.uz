"use client";

// What the printers were asked to do, and what came back.
//
// ⚠️ **A failed print is the quietest failure in the whole system.** Everything
// else that goes wrong is visible to somebody: an unfiled receipt raises an
// alert, an order the till never sent shows a red badge, a card payment that
// fails leaves a guest standing at the counter. A kitchen ticket that never
// printed leaves no trace at all — the order is on the screen, the sale is in
// the reports, and the only symptom is a plate nobody made, noticed twenty
// minutes later by the person waiting for it.
//
// The queue has recorded every attempt and the printer's own words since the
// day it was written. Nobody could read them.
//
// ⚠️ **Failures first, and folded shut when there are none.** A list sorted by
// time answers "what did we print today", which nobody asks. The question this
// exists for is "what did *not* print", and on a busy evening that is three
// rows among four hundred.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatTime } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { ListScroll } from "@/components/admin/PagedList";
import type { PrintJobRow } from "@/lib/types";

export default function PrintQueuePanel() {
  const t = useAdminT();
  const [jobs, setJobs] = useState<PrintJobRow[]>([]);
  const [failed, setFailed] = useState(0);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api.adminPrintJobs();
      setJobs(res.jobs);
      setFailed(res.failed);
    } catch {
      // Silent. This is a background reading on a settings page somebody
      // opened to do something else.
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function retry(job: PrintJobRow) {
    setBusy(job.id);
    setError("");
    try {
      await api.adminRetryPrintJob(job.id);
      // Re-read rather than patched here: whether the agent picks it up is the
      // server's answer, and a row that marked itself sent would hide a
      // printer that is still unplugged.
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy("");
    }
  }

  const shown = open ? jobs : jobs.filter((j) => j.failed);
  if (jobs.length === 0) return null;

  return (
    <div className="mt-4 rounded-2xl border border-line p-3">
      <div className="flex items-center justify-between gap-2">
        <div>
          <span className="text-sm font-medium">{t.printers.queueTitle}</span>
          {/* ⚠️ The count is the failures, not the queue length: "412 jobs"
              says the printers are busy, which is not a problem. */}
          {failed > 0 && (
            <span className="ml-2 text-sm font-medium text-danger">
              {t.printers.queueFailed(failed)}
            </span>
          )}
        </div>
        <button
          type="button"
          className="btn-ghost px-2 py-1 text-xs"
          onClick={() => setOpen(!open)}
        >
          {open ? t.printers.queueOnlyFailed : t.printers.queueAll}
        </button>
      </div>

      {failed === 0 && !open && (
        <p className="mt-1 text-xs text-ink-muted">{t.printers.queueOk}</p>
      )}
      {error && <p className="mt-1 text-xs text-danger">{error}</p>}

      {shown.length > 0 && (
        <ListScroll max="max-h-72" className="mt-2">
          <ul className="space-y-1">
            {shown.map((j) => (
              <li
                key={j.id}
                className="flex items-center justify-between gap-2 rounded-xl bg-ink/[0.03] px-2.5 py-1.5 text-xs"
              >
                <div className="min-w-0">
                  <div className="truncate">
                    {t.printers.kindLabels[j.kind] ?? j.kind}
                    {j.number ? ` · ${j.number}` : ""}
                    {j.printerName ? ` · ${j.printerName}` : ""}
                  </div>
                  <div className="truncate text-ink-muted">
                    {formatTime(j.createdAt)}
                    {/* The printer's own words: they usually name something
                        fixable in seconds — no paper, wrong name, unplugged. */}
                    {j.error ? ` — ${j.error}` : ""}
                    {!j.error && j.working
                      ? ` — ${t.printers.queueWorking}`
                      : ""}
                    {!j.error && j.doneAt ? ` — ${t.printers.queueDone}` : ""}
                  </div>
                </div>
                {/* Only what was given up on: a job still being tried does not
                    need a button, and one that printed must never be sent
                    again from here. */}
                {j.failed && (
                  <button
                    type="button"
                    className="btn-ghost shrink-0 px-2 py-1 text-xs"
                    disabled={busy !== ""}
                    onClick={() => retry(j)}
                  >
                    {t.printers.queueRetry}
                  </button>
                )}
              </li>
            ))}
          </ul>
        </ListScroll>
      )}
    </div>
  );
}
