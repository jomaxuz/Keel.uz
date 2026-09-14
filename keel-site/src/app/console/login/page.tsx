"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Logo } from "@/components/Logo";
import LangSwitch from "@/components/LangSwitch";
import ThemeToggle from "@/components/ThemeToggle";
import { useT } from "@/lib/i18n/client";
import { login, me } from "@/lib/api";
import { consoleHome } from "@/lib/consoleHome";

export default function LoginPage() {
  const { t } = useT();
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await login(username, password);
      // Straight to the screen this account works on, not via the overview.
      const home = await me().then(consoleHome).catch(() => "/console");
      router.replace(home);
    } catch (err) {
      setError(err instanceof Error ? err.message : "…");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="hull-glow grid min-h-screen place-items-center px-5">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center justify-between">
          <Logo />
          <div className="flex items-center gap-2">
            <LangSwitch />
            <ThemeToggle />
          </div>
        </div>
        <form onSubmit={submit} className="card space-y-4">
          <div>
            <h1 className="h-display text-xl">{t.dash.title}</h1>
            <p className="mt-1 text-sm text-ink-muted">{t.dash.login}</p>
          </div>
          <div>
            <label className="text-sm font-medium">{t.dash.username}</label>
            <input
              className="input mt-1"
              value={username}
              autoComplete="username"
              onChange={(e) => setUsername(e.target.value)}
            />
          </div>
          <div>
            <label className="text-sm font-medium">{t.dash.password}</label>
            <input
              className="input mt-1"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>
          {error && <p className="text-sm text-rose-600 dark:text-rose-400">{error}</p>}
          <button type="submit" disabled={busy} className="btn-primary w-full">
            {busy ? t.dash.loading : t.dash.signIn}
          </button>
        </form>
      </div>
    </div>
  );
}
