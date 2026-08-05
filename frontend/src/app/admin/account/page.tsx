"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import RecoveryPhone from "@/components/admin/RecoveryPhone";
import { MyExtension } from "@/components/admin/PBXEditor";
import type { AdminUser } from "@/lib/types";

export default function AdminAccountPage() {
  const router = useRouter();
  const [user, setUser] = useState<AdminUser | null>(null);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newUsername, setNewUsername] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const t = useAdminT();
  const [done, setDone] = useState(false);

  useEffect(() => {
    api
      .me()
      .then((u) => {
        setUser(u);
        setNewUsername(u.username);
      })
      .catch(() => setUser(null));
  }, []);

  const forced = user?.mustChangePassword ?? false;

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (!currentPassword) {
      setError(t.account.needCurrent);
      return;
    }
    if (newPassword && newPassword.length < 6) {
      setError(t.account.tooShort);
      return;
    }
    if (newPassword && newPassword !== confirm) {
      setError(t.account.mismatch);
      return;
    }
    if (forced && !newPassword) {
      setError(t.account.forcedRequired);
      return;
    }
    const usernameChanged = !!user && newUsername.trim() !== user.username;
    if (!newPassword && !usernameChanged) {
      setError(t.account.nothingToChange);
      return;
    }

    setSaving(true);
    try {
      await api.changeCredentials({
        currentPassword,
        newUsername: usernameChanged ? newUsername.trim() : undefined,
        newPassword: newPassword || undefined,
      });
      setDone(true);
      setCurrentPassword("");
      setNewPassword("");
      setConfirm("");
      if (forced) {
        // First-login change complete → proceed to the dashboard.
        router.replace("/admin");
      } else {
        const u = await api.me();
        setUser(u);
        setTimeout(() => setDone(false), 2500);
      }
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : t.account.failed,
      );
    } finally {
      setSaving(false);
    }
  }

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand";

  return (
    <div className="max-w-lg">
      <h1 className="text-2xl font-bold">{t.account.title}</h1>

      {forced && (
        <div className="mt-4 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-800">
          {t.account.forcedNotice}
        </div>
      )}

      <form
        onSubmit={onSubmit}
        className="mt-6 rounded-3xl border border-line bg-surface shadow-card p-6"
      >
        <label className="block text-sm">
          <span className="font-medium">{t.account.username}</span>
          <input
            className={inputCls}
            value={newUsername}
            onChange={(e) => setNewUsername(e.target.value)}
            autoComplete="username"
          />
        </label>

        <hr className="my-5 border-line" />

        <label className="block text-sm">
          <span className="font-medium">{t.account.currentPassword}</span>
          <input
            type="password"
            className={inputCls}
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
            autoComplete="current-password"
          />
        </label>

        <label className="mt-4 block text-sm">
          <span className="font-medium">
            {t.account.newPassword}
            {forced ? "" : t.account.newPasswordOptional}
          </span>
          <input
            type="password"
            className={inputCls}
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            autoComplete="new-password"
            placeholder={t.account.passwordPh}
          />
        </label>

        <label className="mt-4 block text-sm">
          <span className="font-medium">{t.account.confirmPassword}</span>
          <input
            type="password"
            className={inputCls}
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            autoComplete="new-password"
          />
        </label>

        {error && (
          <p className="mt-4 rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand">
            {error}
          </p>
        )}
        {done && !forced && (
          <p className="mt-4 text-sm text-emerald-600">{t.account.saved}</p>
        )}

        <button
          type="submit"
          disabled={saving}
          className="btn-primary mt-6 w-full px-6 py-2.5 disabled:opacity-60"
        >
          {saving ? t.common.saving : t.common.save}
        </button>
      </form>

      {/* Which handset is this operator's. Here rather than in settings
          because it is per-person: every operator sets their own. */}
      {!forced && (
        <div className="mt-6 rounded-2xl border border-line bg-surface p-6 shadow-card">
          <MyExtension initial={user?.pbxExtension} />
        </div>
      )}

      {/* Hidden during the forced first-login change: one thing at a time. */}
      {!forced && (
        <RecoveryPhone
          phone={user?.phone}
          onSaved={(next) =>
            setUser((u) => (u ? { ...u, phone: next } : u))
          }
        />
      )}
    </div>
  );
}
