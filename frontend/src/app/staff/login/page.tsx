"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { useStaff } from "@/lib/staff";
import { useAdminT } from "@/lib/i18n/admin";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";

export default function StaffLoginPage() {
  const router = useRouter();
  // Where to land after signing in. The till and the floor screen share this
  // form and this token — a second login page would be a second place for the
  // two to disagree about what a valid session is.
  const params = useSearchParams();
  const next = safeNext(params.get("next"));
  const { staff, loading, login } = useStaff();
  const t = useAdminT();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!loading && staff) router.replace(next);
  }, [staff, loading, router, next]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.staffLogin(username.trim().toLowerCase(), password);
      login(res.token, res.staff);
      router.replace(next);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.staff.loginFailed);
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
              {t.staff.loginTitle}
            </h1>
            <p className="mt-1 text-sm text-ink-muted">{t.staff.loginHint}</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <LangSwitch />
            <ThemeToggle />
          </div>
        </div>

        <label className="mt-6 block text-sm">
          <span className="font-medium">{t.staff.username}</span>
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
          <span className="font-medium">{t.staff.password}</span>
          <input
            className="input mt-1"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>

        {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

        <button type="submit" disabled={busy} className="btn-primary mt-6 w-full py-3">
          {busy ? t.staff.signingIn : t.staff.signIn}
        </button>
      </form>
    </main>
  );
}

/** Only our own screens, and only by path.
 *
 *  ⚠️ A `next` taken from the address bar is attacker-controlled: an absolute
 *  URL there turns the restaurant's own login into an open redirect, and the
 *  page it lands on can ask for the password again while looking exactly like
 *  this one. A leading `//` is an absolute URL too — that is the form the naive
 *  check misses. */
function safeNext(raw: string | null): string {
  if (!raw || !raw.startsWith("/") || raw.startsWith("//")) return "/staff";
  return raw;
}
