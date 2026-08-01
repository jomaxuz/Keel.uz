"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { useCourier } from "@/lib/courier";
import { useAdminT } from "@/lib/i18n/admin";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";

export default function CourierLoginPage() {
  const router = useRouter();
  const { courier, loading, login } = useCourier();
  const t = useAdminT();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!loading && courier) router.replace("/kuryer");
  }, [courier, loading, router]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.courierLogin(username.trim().toLowerCase(), password);
      login(res.token, res.courier);
      router.replace("/kuryer");
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : t.courier.loginFailed,
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-dvh items-center justify-center p-6">
      <form
        onSubmit={submit}
        className="w-full max-w-sm rounded-3xl border border-line bg-surface p-7 shadow-card"
      >
        <div className="flex items-start justify-between gap-3">
          <div>
            <h1 className="font-display text-2xl font-bold">
              {t.courier.loginTitle}
            </h1>
            <p className="mt-1 text-sm text-ink-muted">{t.courier.loginHint}</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <LangSwitch />
            <ThemeToggle />
          </div>
        </div>

        <label className="mt-6 block text-sm">
          <span className="font-medium">{t.courier.username}</span>
          <input
            className="input mt-1"
            value={username}
            autoCapitalize="none"
            autoCorrect="off"
            onChange={(e) => setUsername(e.target.value)}
            required
          />
        </label>

        <label className="mt-4 block text-sm">
          <span className="font-medium">{t.courier.password}</span>
          <input
            className="input mt-1"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>

        {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

        <button
          type="submit"
          disabled={busy}
          className="btn-primary mt-6 w-full py-3"
        >
          {busy ? t.courier.signingIn : t.courier.signIn}
        </button>
      </form>
    </main>
  );
}
