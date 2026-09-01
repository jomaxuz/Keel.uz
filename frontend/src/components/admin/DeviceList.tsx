"use client";

// Which phone an account is signed in on, and the button that releases it.
//
// ⚠️ **The release is not a convenience beside the lock — it is what makes the
// lock safe to have.** An install id changes when the app is reinstalled, a
// lost phone never comes back, and a screen breaks on a Friday night. Without
// somebody in the building who can lift a binding, the feature that stops two
// people sharing a login also stops one person doing their job, and the answer
// would be a phone call to us.
//
// ⚠️ **The IP is shown because it is the one thing that answers "is this really
// them".** A courier who signs in from the restaurant's wifi every morning and
// then from another city is a question worth asking; the binding alone cannot
// raise it.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/orderFlow";
import type { LoginDevice } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

export default function DeviceList({
  kind,
  subjectId,
  className = "",
}: {
  kind: "admin" | "staff" | "courier";
  subjectId: string;
  className?: string;
}) {
  const t = useAdminT();
  const { ask } = useAsk();
  const [rows, setRows] = useState<LoginDevice[] | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api.adminDevices(kind, subjectId);
      setRows(res.devices ?? []);
      setError("");
    } catch (e) {
      // ⚠️ A manager looking at somebody they may not see gets a 404, the same
      // as everywhere else in the panel — and an empty list rather than an
      // error, because the row itself already told them what they may see.
      setRows([]);
      if (e instanceof ApiError && e.status !== 404) setError(e.message);
    }
  }, [kind, subjectId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function release(id: string) {
    if (!(await ask({ title: t.devices.releaseConfirm, danger: true }))) return;
    setBusy(id);
    try {
      await api.adminDeleteDevice(id);
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(null);
    }
  }

  if (rows === null) return null;

  return (
    <section className={className}>
      <h3 className="text-sm font-semibold">{t.devices.title}</h3>
      <p className="mt-0.5 text-xs text-ink-muted">{t.devices.hint}</p>

      {error && <p className="mt-2 text-sm text-red-600">{error}</p>}

      {rows.length === 0 ? (
        <p className="mt-3 rounded-2xl border border-dashed border-line-strong p-4 text-center text-sm text-ink-muted">
          {t.devices.empty}
        </p>
      ) : (
        <ul className="mt-3 space-y-2">
          {rows.map((d) => (
            <li
              key={d.id}
              className="flex flex-wrap items-center gap-3 rounded-2xl border border-line bg-surface p-3 shadow-card"
            >
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold">
                  {t.devices.app[d.app as keyof typeof t.devices.app] ?? d.app}
                  {d.name ? ` · ${d.name}` : ""}
                </p>
                {/* ⚠️ The id in full, in a monospace face: it is what somebody
                    compares against the phone in their hand when two rows look
                    alike, and a truncated one cannot be compared at all. */}
                <p className="break-all font-mono text-[11px] text-ink-muted">
                  {d.deviceId}
                </p>
                <p className="text-xs text-ink-muted">
                  {[
                    d.ip ? `IP ${d.ip}` : null,
                    d.platform,
                    t.devices.lastSeen(formatDateTime(d.lastSeenAt)),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </p>
              </div>
              <button
                type="button"
                disabled={busy === d.id}
                onClick={() => void release(d.id)}
                className="btn-ghost shrink-0 px-3 py-1.5 text-xs text-red-600 disabled:opacity-50"
              >
                {t.devices.release}
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
