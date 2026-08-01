"use client";

// Admin activity log: who did what, and when. Owner-only.
//
// The panel is shared by the owner and their managers, so every action that
// changes something — an order accepted or cancelled, a courier or an admin
// added or removed, the menu or the settings edited — leaves a line here. Rows
// are never edited or deleted.

import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { ListScroll } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/orderFlow";
import type { AdminLog, AdminUser } from "@/lib/types";

const PAGE = 100;

// Filter groups match the action id prefixes the backend writes.
const GROUPS = [
  "",
  "order",
  "courier",
  "staff",
  "admin",
  "menu",
  "category",
  "provider",
  "settings",
] as const;

export default function AdminLogsPage() {
  const t = useAdminT();
  const [logs, setLogs] = useState<AdminLog[]>([]);
  const [admins, setAdmins] = useState<AdminUser[]>([]);
  const [group, setGroup] = useState<string>("");
  const [adminId, setAdminId] = useState<string>("");
  // Free text: operators paste an order number straight off a receipt to see
  // everything that was ever done to that order.
  const [q, setQ] = useState("");
  const [loading, setLoading] = useState(true);
  const [more, setMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (before?: string) => {
      setLoading(true);
      try {
        const rows = await api.adminLogs({
          action: group || undefined,
          adminId: adminId || undefined,
          q: q.trim() || undefined,
          limit: PAGE,
          before,
        });
        setLogs((prev) => (before ? [...prev, ...rows] : rows));
        setMore(rows.length === PAGE);
        setError(null);
      } catch (e) {
        setError(e instanceof ApiError ? e.message : t.common.loadFailed);
      } finally {
        setLoading(false);
      }
    },
    // t only supplies the error string; re-running on a language switch would
    // reload the list for no reason.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [group, adminId, q],
  );

  useEffect(() => {
    // Debounced: the search box reloads as it is typed into.
    const id = setTimeout(() => load(), 250);
    return () => clearTimeout(id);
  }, [load]);

  useEffect(() => {
    api
      .adminAccounts()
      .then(setAdmins)
      .catch(() => setAdmins([]));
  }, []);

  // Unknown ids (an action added later, an old row) still read sensibly.
  const actionLabel = (action: string) =>
    t.logs.actions[action as keyof typeof t.logs.actions] ?? action;

  return (
    <div>
      <h1 className="font-display text-2xl font-bold">{t.logs.title}</h1>
      <p className="mt-1 text-sm text-ink-muted">{t.logs.hint}</p>

      <div className="mt-5 flex flex-wrap items-center gap-2">
        {GROUPS.map((g) => (
          <button
            key={g || "all"}
            type="button"
            onClick={() => setGroup(g)}
            className={`rounded-full px-3.5 py-1.5 text-sm font-semibold transition-colors ${
              group === g
                ? "bg-brand text-white"
                : "border border-line bg-surface text-ink-soft hover:border-brand hover:text-brand"
            }`}
          >
            {g === "" ? t.logs.groupAll : t.logs.groups[g]}
          </button>
        ))}
        <input
          className="ml-auto w-full max-w-xs rounded-xl border border-line-strong bg-surface px-3 py-1.5 text-sm outline-none focus:border-brand"
          placeholder={t.logs.searchPh}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select
          value={adminId}
          onChange={(e) => setAdminId(e.target.value)}
          className="rounded-xl border border-line-strong bg-surface px-3 py-1.5 text-sm outline-none focus:border-brand"
        >
          <option value="">{t.logs.everyone}</option>
          {admins.map((a) => (
            <option key={a.id} value={a.id}>
              {a.username}
            </option>
          ))}
        </select>
      </div>

      {error && <p className="mt-4 text-sm text-red-600">{error}</p>}

      <div className="mt-5 overflow-hidden rounded-3xl border border-line bg-surface shadow-card">
        <ListScroll className="overflow-x-auto" max="max-h-[68vh]">
        <table className="w-full min-w-[640px] text-sm">
          <thead className="sticky top-0 z-10 bg-surface">
            <tr className="border-b border-line text-left text-xs uppercase tracking-wide text-ink-muted">
              <th className="px-4 py-3 font-medium">{t.logs.when}</th>
              <th className="px-4 py-3 font-medium">{t.logs.who}</th>
              <th className="px-4 py-3 font-medium">{t.logs.what}</th>
              <th className="px-4 py-3 font-medium">{t.logs.target}</th>
            </tr>
          </thead>
          <tbody>
            {logs.map((l) => (
              <tr key={l.id} className="border-b border-line last:border-0">
                <td className="whitespace-nowrap px-4 py-2.5 text-ink-muted">
                  {formatDateTime(l.at)}
                </td>
                <td className="px-4 py-2.5">
                  <span className="font-medium">{l.adminName || "—"}</span>
                  {l.adminRole && (
                    <span className="ml-1 text-xs text-ink-muted/70">
                      {l.adminRole === "owner"
                        ? t.admins.roleOwner
                        : t.admins.roleManager}
                    </span>
                  )}
                </td>
                <td className="px-4 py-2.5">{actionLabel(l.action)}</td>
                <td className="px-4 py-2.5">
                  {l.targetLabel && (
                    <span className="font-medium">{l.targetLabel}</span>
                  )}
                  {l.details && (
                    <span className="block text-xs text-ink-muted">
                      {l.details}
                    </span>
                  )}
                </td>
              </tr>
            ))}
            {!loading && logs.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-10 text-center text-ink-muted/70">
                  {t.logs.empty}
                </td>
              </tr>
            )}
          </tbody>
        </table>
        </ListScroll>
      </div>

      <div className="mt-4 flex items-center justify-center gap-3">
        {loading && (
          <span className="text-sm text-ink-muted/70">{t.common.loading}</span>
        )}
        {!loading && more && (
          <button
            type="button"
            onClick={() => load(logs[logs.length - 1]?.at)}
            className="btn-ghost px-4 py-2 text-sm"
          >
            {t.logs.loadMore}
          </button>
        )}
      </div>
    </div>
  );
}
