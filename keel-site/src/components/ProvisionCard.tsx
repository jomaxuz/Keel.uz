"use client";

import { useState } from "react";
import { useT } from "@/lib/i18n/client";
import { provisionTenant, type Tenant } from "@/lib/api";

/** What happened when we last tried to bring this tenant up, and a button to
 *  try again.
 *
 *  Provisioning is deliberately allowed to fail without failing the tenant: a
 *  customer recorded but not started is one button away from working, whereas
 *  a create that errors halfway leaves an operator guessing which half
 *  happened. The reason is shown in full — a retry button with no reason above
 *  it gets pressed twice and then abandoned. */
export default function ProvisionCard({
  tenant,
  onDone,
}: {
  tenant: Tenant;
  onDone: (t: Tenant) => void;
}) {
  const { t } = useT();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function retry() {
    setBusy(true);
    setError("");
    try {
      const out = await provisionTenant(tenant.id);
      onDone(out.tenant);
    } catch (e) {
      setError(e instanceof Error ? e.message : "…");
    } finally {
      setBusy(false);
    }
  }

  const ok = tenant.provisionStatus === "ready";
  const failed = tenant.provisionStatus === "failed";

  const container =
    tenant.containerStatus === "running"
      ? t.dash.provRunning
      : tenant.containerStatus === "stopped"
        ? t.dash.provStopped
        : tenant.containerStatus === "absent"
          ? t.dash.provAbsent
          : null;

  return (
    <div className="rounded-2xl border border-line bg-surface p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2.5">
          <span
            className={`h-2.5 w-2.5 rounded-full ${
              ok ? "bg-emerald-500" : failed ? "bg-rose-500" : "bg-ink-muted"
            }`}
          />
          <span className="text-sm font-semibold text-ink">
            {ok ? t.dash.provReady : failed ? t.dash.provFailed : t.dash.provPending}
          </span>
          {container && <span className="text-xs text-ink-muted">· {container}</span>}
        </div>
        <button type="button" onClick={retry} disabled={busy} className="btn-ghost px-3 py-2 text-xs">
          {busy ? t.dash.saving : t.dash.provRetry}
        </button>
      </div>

      {tenant.provisionError && (
        <pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-words rounded-xl bg-rose-500/10 p-3 text-xs text-rose-700 dark:text-rose-300">
          {tenant.provisionError}
        </pre>
      )}
      {error && <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">{error}</p>}
      {/* Absent container status means this deployment has no Docker socket —
          worth saying, because otherwise "not provisioned" looks like a bug. */}
      {tenant.containerStatus === undefined && (
        <p className="mt-3 text-xs text-ink-muted">{t.dash.provOff}</p>
      )}
    </div>
  );
}
