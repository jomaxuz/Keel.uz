"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError, setToken } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { homeFor } from "@/lib/panelRole";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";
import ForgotPassword from "@/components/admin/ForgotPassword";

export default function AdminLoginPage() {
  const router = useRouter();
  const t = useAdminT();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [forgot, setForgot] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await api.login(username, password);
      setToken(res.token);
      // ⚠️ **A storekeeper lands in the store, not on the dashboard.** `/admin`
      // is the sales dashboard and their token is refused every figure on it —
      // so the first thing a technologist would ever see is a page of errors,
      // which is how somebody decides an account does not work.
      // ⚠️ **Each role lands where it may actually work.** The dashboard is the
      // company's numbers, so it answers forbidden for the two limited roles —
      // and a sign-in that ends on an error reads as a broken account.
      router.replace(homeFor(res.user?.role));
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 401
          ? t.login.wrong
          : t.login.failed,
      );
    } finally {
      setLoading(false);
    }
  }

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand focus:ring-1 focus:ring-brand";

  const header = (
    <div className="flex items-start justify-between gap-3">
      <div>
        <h1 className="text-xl font-bold">
          {forgot ? t.login.forgotTitle : t.login.title}
        </h1>
        <p className="mt-1 text-sm text-ink-muted">
          {forgot ? t.login.forgotSubtitle : t.login.subtitle}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <LangSwitch />
        <ThemeToggle />
      </div>
    </div>
  );

  if (forgot) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-cream p-4">
        <div className="w-full max-w-sm rounded-3xl border border-line bg-surface p-8 shadow-card">
          {header}
          <ForgotPassword onClose={() => setForgot(false)} />
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-cream p-4">
      <form
        onSubmit={onSubmit}
        className="w-full max-w-sm rounded-3xl border border-line bg-surface shadow-card p-8 shadow-sm"
      >
        {header}

        <label className="mt-6 block text-sm">
          <span className="font-medium">{t.login.username}</span>
          <input
            className={inputCls}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
            autoFocus
          />
        </label>

        <label className="mt-4 block text-sm">
          <span className="font-medium">{t.login.password}</span>
          <input
            type="password"
            className={inputCls}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
        </label>

        {error && (
          <p className="mt-4 rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand">
            {error}
          </p>
        )}

        <button
          type="submit"
          disabled={loading}
          className="btn-primary mt-6 w-full px-6 py-2.5 disabled:opacity-60"
        >
          {loading ? t.login.submitting : t.login.submit}
        </button>
        <button
          type="button"
          onClick={() => setForgot(true)}
          className="mt-3 w-full text-sm text-ink-muted hover:text-brand"
        >
          {t.login.forgot}
        </button>
      </form>
    </div>
  );
}
