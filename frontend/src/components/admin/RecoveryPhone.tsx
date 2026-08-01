"use client";

// The number a forgotten password is recovered through.
//
// Without one, "forgot my password" falls back to an SSH session and
// `cmd/adminreset` on the server — which is exactly the support call the reset
// flow exists to remove. The seeded first owner has no phone at all, so this is
// the one setting worth nagging about.
//
// The number is confirmed by SMS before it is stored: a mistyped digit here
// would only be discovered on the day it is needed, which is the worst possible
// day to discover it.

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatUzPhone } from "@/lib/format";

export default function RecoveryPhone({
  phone,
  onSaved,
}: {
  /** The number currently on the account, if any. */
  phone?: string;
  onSaved: (phone: string) => void;
}) {
  const t = useAdminT();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [code, setCode] = useState("");
  const [sent, setSent] = useState(false);
  const [demoCode, setDemoCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand";

  function reset() {
    setEditing(false);
    setSent(false);
    setDraft("");
    setCode("");
    setDemoCode("");
    setError(null);
  }

  async function sendCode() {
    setError(null);
    setBusy(true);
    try {
      const res = await api.adminPhoneRequest(draft);
      setDemoCode(res.code ?? "");
      setSent(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function verify() {
    setError(null);
    setBusy(true);
    try {
      const res = await api.adminPhoneVerify(draft, code.trim());
      onSaved(res.phone);
      reset();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-6 rounded-3xl border border-line bg-surface p-6 shadow-card">
      <h2 className="font-semibold">{t.account.recoveryTitle}</h2>
      <p className="mt-1 text-sm text-ink-muted">{t.account.recoveryHint}</p>

      {!editing && (
        <div className="mt-4 flex flex-wrap items-center gap-3">
          {phone ? (
            <span className="font-medium">{formatUzPhone(phone)}</span>
          ) : (
            // Worth calling out: without a number the owner is one forgotten
            // password away from needing a terminal.
            <span className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
              {t.account.recoveryMissing}
            </span>
          )}
          <button
            type="button"
            onClick={() => setEditing(true)}
            className="btn-ghost px-4 py-2 text-sm"
          >
            {phone ? t.account.recoveryChange : t.account.recoveryAdd}
          </button>
        </div>
      )}

      {editing && (
        <div className="mt-4">
          <label className="block text-sm">
            <span className="font-medium">{t.account.recoveryPhone}</span>
            <input
              className={inputCls}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="+998 90 123 45 67"
              inputMode="tel"
              disabled={sent}
              autoFocus
            />
          </label>

          {sent && (
            <>
              {demoCode && (
                <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
                  {t.login.demoCode(demoCode)}
                </p>
              )}
              <label className="mt-4 block text-sm">
                <span className="font-medium">{t.login.code}</span>
                <input
                  className={inputCls}
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  inputMode="numeric"
                  autoComplete="one-time-code"
                />
              </label>
            </>
          )}

          {error && (
            <p className="mt-3 rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand dark:bg-rose-500/10 dark:text-rose-300">
              {error}
            </p>
          )}

          <div className="mt-4 flex gap-3">
            <button
              type="button"
              onClick={sent ? verify : sendCode}
              disabled={busy || (sent ? !code.trim() : !draft.trim())}
              className="btn-primary px-5 py-2 text-sm disabled:opacity-60"
            >
              {busy
                ? t.common.saving
                : sent
                  ? t.account.recoveryConfirm
                  : t.login.sendCode}
            </button>
            <button
              type="button"
              onClick={reset}
              className="px-4 py-2 text-sm text-ink-muted hover:text-ink"
            >
              {t.common.cancel}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
