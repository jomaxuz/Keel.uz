"use client";

// "I forgot the password."
//
// Every install runs on the restaurant's own VPS, so this used to mean an SSH
// session and a command-line tool — a support call from someone who has never
// opened a terminal. Three steps instead: the username, the code texted to the
// number on the account, and the new password.
//
// The code only works for this flow (the server issues it for a separate
// purpose), so the owner's ordinary customer login on the same phone cannot be
// mistaken for it — or used as one.

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";

type Step = "username" | "code" | "done";

export default function ForgotPassword({ onClose }: { onClose: () => void }) {
  const t = useAdminT();
  const [step, setStep] = useState<Step>("username");
  const [username, setUsername] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [maskedPhone, setMaskedPhone] = useState("");
  // The demo SMS provider hands the code back instead of sending it; showing it
  // is the only way to test a fresh install before an SMS contract exists.
  const [demoCode, setDemoCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand focus:ring-1 focus:ring-brand";

  async function requestCode(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      const res = await api.forgotPassword(username.trim());
      setMaskedPhone(res.phone);
      setDemoCode(res.code ?? "");
      setStep("code");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function submitReset(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      await api.resetPassword(username.trim(), code.trim(), password);
      setStep("done");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  if (step === "done") {
    return (
      <div className="mt-6">
        <p className="rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">
          {t.login.resetDone}
        </p>
        <button
          type="button"
          onClick={onClose}
          className="btn-primary mt-4 w-full px-6 py-2.5"
        >
          {t.login.backToLogin}
        </button>
      </div>
    );
  }

  return (
    <form onSubmit={step === "username" ? requestCode : submitReset} className="mt-6">
      {step === "username" ? (
        <>
          <p className="text-sm text-ink-muted">{t.login.forgotHint}</p>
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.login.username}</span>
            <input
              className={inputCls}
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              autoFocus
            />
          </label>
        </>
      ) : (
        <>
          <p className="text-sm text-ink-muted">
            {t.login.codeSentTo(maskedPhone)}
          </p>
          {demoCode && (
            <p className="mt-2 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
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
              autoFocus
            />
          </label>
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.login.newPassword}</span>
            <input
              type="password"
              className={inputCls}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
            />
          </label>
        </>
      )}

      {error && (
        <p className="mt-4 rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand dark:bg-rose-500/10 dark:text-rose-300">
          {error}
        </p>
      )}

      <button
        type="submit"
        disabled={busy}
        className="btn-primary mt-6 w-full px-6 py-2.5 disabled:opacity-60"
      >
        {busy
          ? t.common.saving
          : step === "username"
            ? t.login.sendCode
            : t.login.resetSubmit}
      </button>
      <button
        type="button"
        onClick={onClose}
        className="mt-3 w-full text-sm text-ink-muted hover:text-ink"
      >
        {t.login.backToLogin}
      </button>
    </form>
  );
}
