"use client";

// Console accounts: who works here, and what each of them may see.
//
// ⚠️ **Owner only, and the server says so too.** Hiding the page is a convenience;
// the refusal lives in the handlers. An account that could grant itself more would not
// be a boundary at all.
//
// The one number worth putting beside a name is **how many customers they brought
// in** — it is what a sales account is for, and it is the column an owner opens this
// page to read. Everything else about a person is one click away, on their own page.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  consoleLog,
  createStaff,
  deleteStaff,
  me as fetchMe,
  staffList,
  updateStaff,
  type ConsoleLogRow,
  type Me,
  type StaffRow,
} from "@/lib/api";
import { useT } from "@/lib/i18n/client";
import { LogBlock, RolePicker } from "@/components/console/StaffParts";

export default function StaffPage() {
  const { t } = useT();
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [rows, setRows] = useState<StaffRow[]>([]);
  const [log, setLog] = useState<ConsoleLogRow[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const empty = {
    username: "",
    password: "",
    name: "",
    phone: "",
    roles: ["agent"] as string[],
  };
  const [form, setForm] = useState(empty);

  const load = useCallback(() => {
    staffList()
      .then((r) => setRows(r.items))
      .catch((e) => setError(e instanceof Error ? e.message : "yuklanmadi"));
    consoleLog()
      .then((r) => setLog(r.items))
      .catch(() => setLog([]));
  }, []);

  useEffect(() => {
    fetchMe()
      .then(setMe)
      .catch(() => setMe(null));
    load();
  }, [load]);

  async function add() {
    setBusy(true);
    setError("");
    try {
      await createStaff(form);
      setForm(empty);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "saqlanmadi");
    } finally {
      setBusy(false);
    }
  }

  async function patch(id: string, body: Parameters<typeof updateStaff>[1]) {
    setError("");
    try {
      await updateStaff(id, body);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "saqlanmadi");
    }
  }

  async function remove(s: StaffRow) {
    // ⚠️ Asked every time: this is the one button on the page that cannot be
    // undone, sitting beside the one that can.
    if (!window.confirm(t.console.staff.confirmDelete(s.name || s.username))) return;
    setError("");
    try {
      await deleteStaff(s.id);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "o'chirilmadi");
    }
  }

  if (me && !me.can.staff) {
    return (
      <div className="card p-6">
        <p className="text-sm text-ink-muted">{t.console.staff.ownerOnly}</p>
        <Link
          href="/console"
          className="mt-3 inline-block text-sm text-signal-600"
        >
          {t.console.staff.back}
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-bold text-ink">{t.console.staff.title}</h1>
      {error && <p className="text-sm text-hot-600">{error}</p>}

      <section className="card p-5">
        <h2 className="text-sm font-semibold text-ink">
          {t.console.staff.newAccount}
        </h2>
        <p className="mt-1 text-xs text-ink-muted">{t.console.staff.rolesHint}</p>
        <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <input
            className="input"
            placeholder={t.console.staff.login}
            value={form.username}
            onChange={(e) => setForm({ ...form, username: e.target.value })}
          />
          <input
            className="input"
            placeholder={t.console.staff.name}
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <input
            className="input"
            placeholder={t.console.staff.phone}
            value={form.phone}
            onChange={(e) => setForm({ ...form, phone: e.target.value })}
          />
          <input
            className="input"
            placeholder={t.console.staff.password}
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
          />
        </div>
        <div className="mt-3">
          <RolePicker
            value={form.roles}
            onChange={(roles) => setForm({ ...form, roles })}
          />
        </div>
        <button
          type="button"
          onClick={() => void add()}
          disabled={
            busy ||
            form.username.length < 3 ||
            form.password.length < 8 ||
            form.roles.length === 0
          }
          className="mt-3 rounded-xl bg-ink px-4 py-2 text-sm font-semibold text-surface disabled:opacity-40"
        >
          {t.console.staff.add}
        </button>
      </section>

      <section className="card overflow-x-auto p-0">
        <table className="w-full text-sm">
          <thead className="border-b border-line bg-raised text-left text-xs text-ink-muted">
            <tr>
              <th className="p-3">{t.console.staff.who}</th>
              <th className="p-3">{t.console.staff.role}</th>
              <th className="p-3">{t.console.staff.clients}</th>
              <th className="p-3">{t.console.staff.lastLogin}</th>
              <th className="p-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {rows.map((s) => (
              // ⚠️ The whole row opens the person, and every control inside it
              // stops the click: a role pill that also navigated away would
              // save a change nobody saw happen.
              <tr
                key={s.id}
                onClick={() => router.push(`/console/staff/${s.id}`)}
                className={`cursor-pointer hover:bg-raised ${
                  s.isActive === false ? "opacity-50" : ""
                }`}
              >
                <td className="p-3">
                  <Link
                    href={`/console/staff/${s.id}`}
                    onClick={(e) => e.stopPropagation()}
                    className="block font-semibold text-ink hover:underline"
                  >
                    {s.name || s.username}
                  </Link>
                  <span className="block text-xs text-ink-muted">
                    {s.username}
                    {s.phone ? ` · ${s.phone}` : ""}
                    {s.isActive === false ? ` · ${t.console.staff.inactive}` : ""}
                  </span>
                </td>
                <td className="p-3">
                  <RolePicker
                    value={s.roles?.length ? s.roles : [s.role || "owner"]}
                    onChange={(roles) => void patch(s.id, { roles })}
                  />
                </td>
                {/* The column this page exists for. */}
                <td className="p-3 tabular-nums">{s.tenants}</td>
                <td className="p-3 text-xs text-ink-muted">
                  {s.lastLoginAt
                    ? new Date(s.lastLoginAt).toLocaleString()
                    : "—"}
                </td>
                <td className="p-3 text-right">
                  <div className="flex justify-end gap-1.5">
                    {/* ⚠️ Two buttons, on purpose. Switching off keeps the login
                        for when the person comes back; deleting frees it and
                        cannot be undone. One button that did the first while
                        reading like the second is the bug this replaced. */}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        void patch(s.id, { isActive: s.isActive === false });
                      }}
                      className="whitespace-nowrap rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft"
                    >
                      {s.isActive === false
                        ? t.console.staff.enable
                        : t.console.staff.disable}
                    </button>
                    {s.username !== me?.username && (
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          void remove(s);
                        }}
                        className="whitespace-nowrap rounded-lg border border-hot-600/40 px-2 py-1 text-[11px] text-hot-600"
                      >
                        {t.console.staff.deleteForever}
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {/* ⚠️ The log is here rather than on its own page: the question it answers —
          "who signed this customer up, and who changed what" — is asked while looking
          at the list of people it names.

          ⚠️ **Its own block with its own scroll.** Three hundred rows inline made
          this page as long as the log, and the list of people it is read beside
          scrolled off the top. */}
      <LogBlock rows={log} />
    </div>
  );
}
