"use client";

// Phone + one-time SMS code sign-in.
//
// Step 1: the customer types their number → POST /auth/phone/request.
// Step 2: they type the 6-digit code → POST /auth/phone/verify → user JWT.
//
// With SMS_PROVIDER=demo the backend returns the code in the response and it is
// shown on screen, so the flow can be tested without a paid SMS account.

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useUser } from "@/lib/user";
import { useI18n } from "@/lib/i18n/client";

export default function PhoneLogin({ onSuccess }: { onSuccess?: () => void }) {
  const { login } = useUser();
  const { t } = useI18n();

  const [step, setStep] = useState<"phone" | "code">("phone");
  const [phone, setPhone] = useState("");
  const [name, setName] = useState("");
  const [code, setCode] = useState("");
  const [demoCode, setDemoCode] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function requestCode() {
    setBusy(true);
    setError(null);
    try {
      const res = await api.phoneRequestCode(phone);
      setPhone(res.phone);
      setDemoCode(res.code ?? null);
      setStep("code");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.login.error);
    } finally {
      setBusy(false);
    }
  }

  async function verify() {
    setBusy(true);
    setError(null);
    try {
      const res = await api.phoneVerify(phone, code, name.trim() || undefined);
      login(res.token, res.user);
      onSuccess?.();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.login.error);
    } finally {
      setBusy(false);
    }
  }

  if (step === "phone") {
    return (
      <form
        className="space-y-4 text-left"
        onSubmit={(e) => {
          e.preventDefault();
          requestCode();
        }}
      >
        <label className="block text-sm">
          <span className="font-medium">{t.login.phoneLabel}</span>
          <input
            className="input mt-1"
            type="tel"
            inputMode="tel"
            autoComplete="tel"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="+998 90 123 45 67"
            autoFocus
          />
          <span className="mt-1 block text-xs text-ink-muted">
            {t.login.phoneHint}
          </span>
        </label>

        <label className="block text-sm">
          <span className="font-medium">{t.login.nameLabel}</span>
          <input
            className="input mt-1"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t.login.namePh}
          />
        </label>

        {error && <p className="text-sm text-brand">{error}</p>}

        <button
          type="submit"
          disabled={busy}
          className="btn-primary w-full px-6 py-3"
        >
          {busy ? t.login.sending : t.login.sendCode}
        </button>
      </form>
    );
  }

  return (
    <form
      className="space-y-4 text-left"
      onSubmit={(e) => {
        e.preventDefault();
        verify();
      }}
    >
      <label className="block text-sm">
        <span className="font-medium">{t.login.codeLabel}</span>
        <input
          className="input mt-1 text-center text-lg font-bold tracking-[0.4em]"
          inputMode="numeric"
          autoComplete="one-time-code"
          maxLength={6}
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
          autoFocus
        />
        <span className="mt-1 block text-xs text-ink-muted">
          {t.login.codeHint(phone)}
        </span>
      </label>

      {demoCode && (
        <p className="rounded-xl bg-brand-tint px-3 py-2 text-xs font-semibold text-brand-dark">
          {t.login.demoNotice(demoCode)}
        </p>
      )}

      {error && <p className="text-sm text-brand">{error}</p>}

      <button
        type="submit"
        disabled={busy || code.length < 4}
        className="btn-primary w-full px-6 py-3"
      >
        {busy ? t.login.verifying : t.login.verify}
      </button>

      <div className="flex items-center justify-between text-xs">
        <button
          type="button"
          onClick={() => {
            setStep("phone");
            setCode("");
            setDemoCode(null);
            setError(null);
          }}
          className="text-ink-muted hover:text-brand"
        >
          {t.login.changePhone}
        </button>
        <button
          type="button"
          onClick={requestCode}
          disabled={busy}
          className="font-semibold text-brand hover:underline disabled:opacity-50"
        >
          {t.login.resend}
        </button>
      </div>
    </form>
  );
}
