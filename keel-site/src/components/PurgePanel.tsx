"use client";

// The irreversible half of closing an account.
//
// ⚠️ **Its own panel, at the bottom, and only for a customer already switched
// off.** Beside the status buttons it would be one mis-click from "suspend",
// which is the click somebody makes forty times a year — and the two actions
// are not neighbours in consequence even though they are neighbours in wording.
//
// The confirmation is a **typed slug**, not a dialog. A dialog is dismissed by
// the same reflex that opened it, and it cannot show what is about to be
// destroyed; typing `b5somsa` means reading the identifier that actually decides
// which database is dropped. The reason field is required for the same reason it
// is on the server: months later somebody looks at the row and asks why.

import { useState } from "react";
import { useT } from "@/lib/i18n/client";
import { purgeTenant, type PurgeStep, type Tenant } from "@/lib/api";

export default function PurgePanel({
  tenant,
  onDone,
}: {
  tenant: Tenant;
  onDone: () => void;
}) {
  const { t } = useT();
  const [confirm, setConfirm] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [steps, setSteps] = useState<PurgeStep[] | null>(null);

  // Already gone: the panel becomes the record of it. Nothing to press, and the
  // date, the operator and the reason are what somebody opening this row later
  // actually needs.
  if (tenant.purgedAt) {
    return (
      <section className="card border-rose-500/30">
        <p className="text-sm font-semibold text-ink">{t.dash.purgeTitle}</p>
        <p className="mt-2 text-sm text-ink-soft">
          {t.dash.purgedNote(
            tenant.purgedAt.slice(0, 10),
            tenant.purgedBy || "—",
            tenant.purgeReason || "—",
          )}
        </p>
      </section>
    );
  }

  const offline = tenant.status === "suspended" || tenant.status === "deleted";
  const ready = offline && confirm.trim() === tenant.slug && reason.trim().length > 0;

  async function run() {
    setBusy(true);
    setError("");
    try {
      const res = await purgeTenant(tenant.id, confirm.trim(), reason.trim());
      setSteps(res.steps);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : "xato");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="card border-rose-500/30">
      <p className="text-sm font-semibold text-rose-600 dark:text-rose-400">
        {t.dash.purgeTitle}
      </p>
      {/* What goes and what stays, before the fields rather than after them.
          The distinction is the one thing an operator must have straight: the
          restaurant's data is destroyed, our invoices are not. */}
      <p className="mt-2 whitespace-pre-line text-sm text-ink-soft">{t.dash.purgeWhat}</p>

      {!offline ? (
        // Not an error — an instruction. The customer is still live, and the
        // step before this one is the reversible one that should happen first.
        <p className="mt-3 rounded-xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-ink-soft">
          {t.dash.purgeNeedsSuspend}
        </p>
      ) : (
        <div className="mt-4 space-y-3">
          <label className="block">
            <span className="text-xs font-semibold text-ink-muted">
              {t.dash.purgeConfirmLabel(tenant.slug)}
            </span>
            <input
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              placeholder={tenant.slug}
              className="input mt-1"
              autoComplete="off"
            />
          </label>
          <label className="block">
            <span className="text-xs font-semibold text-ink-muted">{t.dash.purgeReason}</span>
            <input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              className="input mt-1"
            />
          </label>
          <button
            type="button"
            onClick={run}
            disabled={!ready || busy}
            className="rounded-xl bg-rose-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-40"
          >
            {busy ? t.dash.purgeRunning : t.dash.purgeButton}
          </button>
        </div>
      )}

      {error && <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">{error}</p>}

      {/* Per step, because they fail independently and their fixes differ. A
          single "done" would hide the one line that matters: whether the
          database is actually gone. */}
      {steps && (
        <ul className="mt-4 space-y-1 text-sm">
          {steps.map((s) => (
            <li key={s.step} className={s.ok ? "text-ink-soft" : "text-rose-600 dark:text-rose-400"}>
              {s.ok ? "✓" : "✕"} {t.dash.purgeStep(s.step)}
              {s.error ? ` — ${s.error}` : ""}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
