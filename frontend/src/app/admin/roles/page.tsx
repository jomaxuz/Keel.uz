"use client";

// Roles: the job titles this restaurant hands out, and what each one may do.
//
// ⚠️ **The tension this screen has to respect: too many permissions produce one
// shared PIN.** A till where every button answers "you may not" teaches a room
// to solve it once — the manager's code gets told to everybody — and after that
// every void in the journal carries the same name. So there are six switches,
// not thirty, and the screen says what each one costs rather than listing
// capabilities.
//
// ⚠️ **A role is a record, not a typed word.** `Staff.position` is free text and
// the system must never read it, because treating a typed job title as a
// permission hands a key to whoever spells it the same way. This screen is the
// answer to that, not an exception to it.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { PermOption, StaffRole } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

export default function AdminRolesPage() {
  const t = useAdminT();
  const { ask } = useAsk();
  const [roles, setRoles] = useState<StaffRole[]>([]);
  const [perms, setPerms] = useState<PermOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [draft, setDraft] = useState<Draft | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminRoles()
      .then((d) => {
        setRoles(d.roles);
        setPerms(d.perms);
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [t]);

  useEffect(load, [load]);

  async function save() {
    if (!draft || !draft.name.trim()) return;
    setBusy(true);
    setError("");
    try {
      const body = { name: draft.name.trim(), perms: draft.perms };
      if (draft.id) await api.updateRole(draft.id, body);
      else await api.createRole(body);
      setDraft(null);
      setNotice(t.roles.saved);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function remove(role: StaffRole) {
    if (!(await ask({ title: t.roles.deleteConfirm(role.name), danger: true })))
      return;
    setBusy(true);
    setError("");
    try {
      await api.deleteRole(role.id);
      setNotice(t.roles.deleted);
      load();
    } catch (e) {
      // ⚠️ The server's own words. "This role is held by staff — reassign them
      // first" names the next step; a generic failure would leave the owner
      // guessing why a delete button did nothing.
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <p className="text-ink-muted">{t.common.loading}</p>;

  return (
    <div className="space-y-5">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-bold">{t.roles.title}</h1>
          <p className="mt-1 max-w-2xl text-sm text-ink-muted">
            {t.roles.intro}
          </p>
        </div>
        <button
          className="btn btn-primary"
          onClick={() => setDraft({ id: "", name: "", perms: [] })}
        >
          + {t.roles.add}
        </button>
      </header>

      {error && <p className="text-sm text-danger">{error}</p>}
      {notice && <p className="text-sm text-success">{notice}</p>}

      <div className="overflow-x-auto">
        <table className="w-full min-w-[44rem] text-sm">
          <thead className="text-left text-xs text-ink-muted">
            <tr className="border-b border-line">
              <th className="py-2 pr-3 font-medium">{t.roles.name}</th>
              <th className="py-2 pr-3 font-medium">{t.roles.perms}</th>
              <th className="py-2 pr-3 text-right font-medium">
                {t.roles.staffCount}
              </th>
              <th className="py-2" />
            </tr>
          </thead>
          <tbody>
            {roles.map((role) => (
              <tr key={role.id} className="border-b border-line/60 align-top">
                <td className="py-3 pr-3 font-medium">{role.name}</td>
                <td className="py-3 pr-3">
                  {role.perms.length === 0 ? (
                    // ⚠️ Said, not left blank. A role with no till permissions
                    // is a real answer — a hostess has an account and no
                    // business at the drawer — and an empty cell reads as a
                    // row that failed to load.
                    <span className="text-xs text-ink-muted">
                      {t.roles.noPerms}
                    </span>
                  ) : (
                    <div className="flex flex-wrap gap-1">
                      {role.perms.map((p) => (
                        <span key={p} className="chip text-xs">
                          {perms.find((x) => x.id === p)?.name ?? p}
                        </span>
                      ))}
                    </div>
                  )}
                </td>
                <td className="py-3 pr-3 text-right tabular-nums">
                  {role.staffCount}
                </td>
                <td className="py-3 text-right">
                  <button
                    className="btn-ghost px-2 text-sm"
                    onClick={() =>
                      setDraft({
                        id: role.id,
                        name: role.name,
                        perms: [...role.perms],
                      })
                    }
                  >
                    {t.common.edit}
                  </button>
                  <button
                    className="btn-ghost px-2 text-sm text-danger"
                    disabled={busy}
                    onClick={() => void remove(role)}
                  >
                    {t.common.delete}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {draft && (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
          <div className="w-full max-w-md rounded-3xl border border-line bg-surface p-5 shadow-card">
            <h2 className="font-display text-xl font-bold">
              {draft.id ? t.roles.editTitle : t.roles.add}
            </h2>

            <label className="mt-4 block text-sm">
              <span className="text-ink-muted">{t.roles.name}</span>
              <input
                className="input mt-1"
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
            </label>

            <div className="mt-4 space-y-2">
              {perms.map((p) => (
                <label key={p.id} className="flex items-start gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="mt-1"
                    checked={draft.perms.includes(p.id)}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        perms: e.target.checked
                          ? [...draft.perms, p.id]
                          : draft.perms.filter((x) => x !== p.id),
                      })
                    }
                  />
                  <span>
                    {p.name}
                    <span className="mt-0.5 block text-xs text-ink-muted">
                      {t.roles.hints[p.id] ?? ""}
                    </span>
                  </span>
                </label>
              ))}
            </div>

            {/* ⚠️ Said on the screen where the switches are, because it is the
                thing that makes restrictive roles usable — and an owner who
                does not know it will grant everybody everything. */}
            <p className="mt-4 rounded-xl bg-ink/5 p-3 text-xs text-ink-soft">
              {t.roles.overrideNote}
            </p>

            <div className="mt-5 flex justify-end gap-2">
              <button className="btn-ghost px-4" onClick={() => setDraft(null)}>
                {t.common.cancel}
              </button>
              <button
                className="btn btn-primary"
                disabled={busy || !draft.name.trim()}
                onClick={() => void save()}
              >
                {busy ? t.common.saving : t.common.save}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

interface Draft {
  id: string;
  name: string;
  perms: string[];
}
