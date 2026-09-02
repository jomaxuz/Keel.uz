"use client";

// Panel accounts. Owner-only (the API enforces it too).
//
// An admin is always an existing site customer: someone who signed in on the
// site with their phone and an SMS code. The owner finds that person here, hands
// them a login and a temporary password, and the account is forced to change it
// on first sign-in. That way every panel account has a verified human behind it.

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import Modal from "@/components/admin/Modal";
import { ListScroll } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/orderFlow";
import { formatUzPhone } from "@/lib/format";
import type { AdminUser, AdminUserRow, PanelRole } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

export default function AdminAccountsPage() {
  const t = useAdminT();
  const { ask, tell } = useAsk();
  const [accounts, setAccounts] = useState<AdminUser[]>([]);
  const [me, setMe] = useState<AdminUser | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  // Account whose password is being reset.
  const [resetting, setResetting] = useState<AdminUser | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([api.adminAccounts(), api.me()])
      .then(([list, self]) => {
        setAccounts(list);
        setMe(self);
        setError(null);
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(load, [load]);

  async function remove(a: AdminUser) {
    if (
      !(await ask({
        title: t.admins.confirmDelete(a.username),
        danger: true,
      }))
    )
      return;
    try {
      await api.deleteAdminAccount(a.id);
      load();
    } catch (e) {
      void tell({
        title: e instanceof ApiError ? e.message : t.common.deleteFailed,
      });
    }
  }

  async function changeRole(a: AdminUser, role: PanelRole) {
    try {
      await api.updateAdminAccount(a.id, { role });
      load();
    } catch (e) {
      void tell({
        title: e instanceof ApiError ? e.message : t.common.saveFailed,
      });
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-bold">{t.admins.title}</h1>
          <p className="mt-1 text-sm text-ink-muted">{t.admins.hint}</p>
        </div>
        <button
          type="button"
          onClick={() => setAdding(true)}
          className="btn-primary px-4 py-2 text-sm"
        >
          {t.admins.add}
        </button>
      </div>

      {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">
          {t.common.loading}
        </p>
      ) : (
        <ListScroll className="mt-5 space-y-3 pr-1" max="max-h-[70vh]">
          {accounts.map((a) => (
            <div
              key={a.id}
              className="rounded-3xl border border-line bg-surface p-4 shadow-card"
            >
              <div className="flex flex-wrap items-center gap-3">
                <div className="min-w-[200px] flex-1">
                  <p className="font-semibold">
                    {a.username}
                    {me?.id === a.id && (
                      <span className="ml-2 badge bg-ink/10 text-ink-soft">
                        {t.admins.you}
                      </span>
                    )}
                    {a.mustChangePassword && (
                      <span className="badge-brand ml-2">
                        {t.admins.pendingChange}
                      </span>
                    )}
                  </p>
                  <p className="mt-0.5 text-sm text-ink-muted">
                    {a.name || "—"}
                    {a.phone ? ` · ${formatUzPhone(a.phone)}` : ""}
                  </p>
                  <p className="mt-0.5 text-xs text-ink-muted/70">
                    {t.admins.createdLine(
                      formatDateTime(a.createdAt),
                      a.createdBy || "—",
                    )}
                    {a.lastLoginAt
                      ? ` · ${t.admins.lastLogin(formatDateTime(a.lastLoginAt))}`
                      : ` · ${t.admins.neverLoggedIn}`}
                  </p>
                </div>

                <select
                  value={a.role}
                  onChange={(e) => changeRole(a, e.target.value as PanelRole)}
                  className="rounded-xl border border-line-strong bg-surface px-2 py-1.5 text-xs outline-none focus:border-brand"
                >
                  <option value="owner">{t.admins.roleOwner}</option>
                  <option value="manager">{t.admins.roleManager}</option>
                  <option value="operator">{t.admins.roleOperator}</option>
                </select>

                <button
                  type="button"
                  onClick={() => setResetting(a)}
                  className="btn-ghost px-3 py-1.5 text-xs"
                >
                  {t.admins.resetPassword}
                </button>
                {me?.id !== a.id && (
                  <button
                    type="button"
                    onClick={() => remove(a)}
                    className="px-2 text-sm text-ink-muted hover:text-red-600"
                  >
                    {t.common.delete}
                  </button>
                )}
              </div>
            </div>
          ))}
        </ListScroll>
      )}

      {adding && (
        <AddAdminModal
          onClose={() => setAdding(false)}
          onDone={() => {
            setAdding(false);
            load();
          }}
        />
      )}
      {resetting && (
        <ResetPasswordModal
          account={resetting}
          onClose={() => setResetting(null)}
          onDone={() => {
            setResetting(null);
            load();
          }}
        />
      )}
    </div>
  );
}

// ---- Add: pick a site customer, then hand out credentials ----

function AddAdminModal({
  onClose,
  onDone,
}: {
  onClose: () => void;
  onDone: () => void;
}) {
  const t = useAdminT();
  const [q, setQ] = useState("");
  const [users, setUsers] = useState<AdminUserRow[]>([]);
  const [searching, setSearching] = useState(false);
  const [picked, setPicked] = useState<AdminUserRow | null>(null);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<Exclude<PanelRole, "stock">>("manager");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Search the site customers — an admin must be one of them.
  useEffect(() => {
    if (picked) return;
    setSearching(true);
    const id = setTimeout(() => {
      api
        .adminUsers({ q: q.trim() || undefined })
        .then((rows) => setUsers(rows.slice(0, 8)))
        .catch(() => setUsers([]))
        .finally(() => setSearching(false));
    }, 300);
    return () => clearTimeout(id);
  }, [q, picked]);

  async function submit() {
    if (!picked) return;
    setBusy(true);
    setError(null);
    try {
      await api.createAdminAccount({
        userId: picked.id,
        username,
        password,
        role,
      });
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

  return (
    <Modal onClose={onClose}>
      <h3 className="font-display text-lg font-bold">{t.admins.addTitle}</h3>
      <p className="mt-1 text-sm text-ink-muted">{t.admins.addHint}</p>

      {!picked ? (
        <>
          <input
            className={inputCls}
            autoFocus
            placeholder={t.admins.searchPh}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <ul className="mt-3 divide-y divide-line rounded-2xl border border-line">
            {searching && users.length === 0 && (
              <li className="px-3 py-3 text-sm text-ink-muted/70">
                {t.common.loading}
              </li>
            )}
            {!searching && users.length === 0 && (
              <li className="px-3 py-3 text-sm text-ink-muted/70">
                {t.admins.noUsers}
              </li>
            )}
            {users.map((u) => (
              <li key={u.id}>
                <button
                  type="button"
                  onClick={() => {
                    setPicked(u);
                    // A sensible default login: their phone without the code.
                    setUsername(u.phone.slice(-9));
                  }}
                  className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left hover:bg-ink/[0.03]"
                >
                  <span>
                    <span className="block text-sm font-medium">
                      {[u.firstName, u.lastName].filter(Boolean).join(" ") ||
                        t.admins.noName}
                    </span>
                    <span className="block text-xs text-ink-muted">
                      {formatUzPhone(u.phone)}
                    </span>
                  </span>
                  <span className="text-xs text-brand">{t.admins.pick}</span>
                </button>
              </li>
            ))}
          </ul>
        </>
      ) : (
        <>
          <div className="mt-4 flex items-center justify-between gap-3 rounded-2xl border border-line bg-ink/[0.02] px-3 py-2">
            <span className="text-sm">
              <span className="font-medium">
                {[picked.firstName, picked.lastName]
                  .filter(Boolean)
                  .join(" ") || t.admins.noName}
              </span>
              <span className="block text-xs text-ink-muted">
                {formatUzPhone(picked.phone)}
              </span>
            </span>
            <button
              type="button"
              onClick={() => setPicked(null)}
              className="text-xs text-brand hover:underline"
            >
              {t.admins.changeUser}
            </button>
          </div>

          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium">{t.admins.login}</span>
              <input
                className={inputCls}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.admins.tempPassword}</span>
              <input
                className={inputCls}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </label>
          </div>
          <label className="mt-3 block text-sm">
            <span className="font-medium">{t.admins.role}</span>
            <select
              className={inputCls}
              value={role}
              onChange={(e) =>
                setRole(e.target.value as Exclude<PanelRole, "stock">)
              }
            >
              <option value="manager">{t.admins.roleManager}</option>
              <option value="owner">{t.admins.roleOwner}</option>
              <option value="operator">{t.admins.roleOperator}</option>
            </select>
          </label>
          {/* ⚠️ Said where the choice is made, not in a help article. An owner
              picking "Operator" is deciding what a temp may see of their
              business, and the answer has to be on the screen at that moment. */}
          {role === "operator" && (
            <p className="mt-2 text-xs text-ink-muted">
              {t.admins.roleOperatorNote}
            </p>
          )}
          <p className="mt-2 rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
            {t.admins.tempPasswordNote}
          </p>
        </>
      )}

      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}

      <div className="mt-6 flex justify-end gap-3">
        <button type="button" onClick={onClose} className="btn-ghost px-4 py-2">
          {t.common.cancel}
        </button>
        <button
          type="button"
          disabled={
            busy || !picked || username.length < 3 || password.length < 6
          }
          onClick={submit}
          className="btn-primary px-5 py-2 disabled:opacity-50"
        >
          {busy ? "..." : t.admins.create}
        </button>
      </div>
    </Modal>
  );
}

// ---- Reset someone's password (they must change it again on next sign-in) ----

function ResetPasswordModal({
  account,
  onClose,
  onDone,
}: {
  account: AdminUser;
  onClose: () => void;
  onDone: () => void;
}) {
  const t = useAdminT();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    setBusy(true);
    setError(null);
    try {
      await api.updateAdminAccount(account.id, { password });
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal onClose={onClose}>
      <h3 className="font-display text-lg font-bold">
        {t.admins.resetTitle(account.username)}
      </h3>
      <label className="mt-4 block text-sm">
        <span className="font-medium">{t.admins.tempPassword}</span>
        <input
          className="mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand"
          autoFocus
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </label>
      <p className="mt-2 text-xs text-ink-muted">{t.admins.tempPasswordNote}</p>
      {error && <p className="mt-3 text-sm text-red-600">{error}</p>}
      <div className="mt-6 flex justify-end gap-3">
        <button type="button" onClick={onClose} className="btn-ghost px-4 py-2">
          {t.common.cancel}
        </button>
        <button
          type="button"
          disabled={busy || password.length < 6}
          onClick={submit}
          className="btn-primary px-5 py-2 disabled:opacity-50"
        >
          {busy ? "..." : t.common.save}
        </button>
      </div>
    </Modal>
  );
}
