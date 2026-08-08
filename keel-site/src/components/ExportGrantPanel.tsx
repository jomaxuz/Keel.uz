"use client";

// Opening the customer's "download everything" button.
//
// **Their data is theirs and they must be able to leave with it.** What this
// screen decides is not whether they may have it, but for how long the button
// exists — because the archive is one file holding every one of their
// customers' names, phone numbers and addresses along with the whole
// commercial history of the business. A permanent button on their own settings
// page turns any borrowed session — a manager who left, a laptop in a back
// office, a password that has been round three people — into a silent, complete
// copy of the restaurant.
//
// So the shape of this panel is the policy:
//
//   • the reason is a required field, not a note, because the person who asks
//     about this a year later is usually the customer;
//   • the window is days, capped, because the way this leaks is a switch left
//     on after a migration everybody has forgotten;
//   • closing it needs no reason at all — the safe direction is never gated;
//   • what was actually downloaded stays on screen after the grant expires,
//     because "did they take a copy" is the first question after a leak and it
//     must outlive the permission.

import { useCallback, useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import {
  bytes,
  exportGrant as fetchGrant,
  setExportGrant,
  type ExportGrant,
} from "@/lib/api";

export default function ExportGrantPanel({ tenantId }: { tenantId: string }) {
  const { t } = useT();
  const [grant, setGrant] = useState<ExportGrant | null>(null);
  const [reason, setReason] = useState("");
  const [days, setDays] = useState(7);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setGrant(await fetchGrant(tenantId));
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  async function save(enabled: boolean) {
    setBusy(true);
    setError("");
    try {
      await setExportGrant(tenantId, enabled ? { enabled, reason, days } : { enabled });
      setReason("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!grant) return null;

  return (
    <section className="card">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-sm font-semibold text-ink">{t.dash.exportTitle}</p>
        {grant.active && (
          <span className="rounded-full bg-amber-500/15 px-2.5 py-0.5 text-xs font-semibold text-amber-700 dark:text-amber-300">
            {t.dash.exportOpen}
          </span>
        )}
      </div>
      <p className="mt-1 text-xs text-ink-muted">{t.dash.exportHint}</p>

      {grant.active ? (
        <div className="mt-4 rounded-2xl border border-line bg-raised p-4 text-sm">
          <p className="text-ink">
            {t.dash.exportGrantedBy(grant.grantedBy ?? "?", dateTime(grant.expiresAt))}
          </p>
          <p className="mt-1 text-ink-soft">“{grant.reason}”</p>
          <button
            type="button"
            disabled={busy}
            onClick={() => save(false)}
            className="btn-ghost mt-3 px-3 py-2 text-xs disabled:opacity-40"
          >
            {t.dash.exportClose}
          </button>
        </div>
      ) : (
        <div className="mt-4 rounded-2xl border border-line bg-raised p-4">
          <label className="text-sm font-medium">{t.dash.exportReason}</label>
          <input
            className="input mt-1"
            value={reason}
            placeholder={t.dash.exportReasonPh}
            onChange={(e) => setReason(e.target.value)}
          />
          <div className="mt-3 flex flex-wrap items-end gap-3">
            <label className="text-sm">
              <span className="mb-1 block text-xs text-ink-muted">
                {t.dash.exportDays}
              </span>
              <select
                className="input w-32"
                value={days}
                onChange={(e) => setDays(Number(e.target.value))}
              >
                {[1, 3, 7, 14, 30].map((d) => (
                  <option key={d} value={d}>
                    {d}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="button"
              // Guarded here as well as on the server, so the button explains
              // itself rather than answering with a 400.
              disabled={busy || reason.trim().length < 5}
              onClick={() => save(true)}
              className="btn-primary px-4 py-2 text-sm disabled:opacity-40"
            >
              {t.dash.exportOpenBtn}
            </button>
          </div>
        </div>
      )}

      {error && <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">{error}</p>}

      {/* Kept visible whether or not a grant is open. */}
      {grant.downloads.length > 0 && (
        <div className="mt-4">
          <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
            {t.dash.exportDownloads}
          </p>
          <ul className="mt-2 divide-y divide-line text-sm">
            {[...grant.downloads].reverse().map((d, i) => (
              <li key={i} className="flex flex-wrap items-center justify-between gap-2 py-2">
                <span className="text-ink-soft">
                  {dateTime(d.at)} · {d.by}
                </span>
                <span className="tabular-nums text-ink-muted">
                  {d.files} · {bytes(d.bytes)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}

/** 24-hour, hand-formatted — `toLocaleString` follows the device language and
 *  would put "1:16 PM" next to the 24-hour times everywhere else. */
function dateTime(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()} ${p(d.getHours())}:${p(d.getMinutes())}`;
}
