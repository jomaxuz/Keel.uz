"use client";

// Console accounts: who works here, and what each of them may see.
//
// ⚠️ **Owner only, and the server says so too.** Hiding the page is a convenience;
// the refusal lives in the handlers. An account that could grant itself more would not
// be a boundary at all.
//
// The one number worth putting beside a name is **how many customers they brought
// in** — it is what a sales account is for, and it is the column an owner opens this
// page to read.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  consoleLog,
  createStaff,
  me as fetchMe,
  staffList,
  updateStaff,
  type ConsoleLogRow,
  type Me,
  type StaffRow,
} from "@/lib/api";

const ROLE_LABEL: Record<string, string> = {
  owner: "Owner (hammasi)",
  admin: "Admin (platforma)",
  manager: "Sotuv menejeri",
  agent: "Agent",
};

export default function StaffPage() {
  const [me, setMe] = useState<Me | null>(null);
  const [rows, setRows] = useState<StaffRow[]>([]);
  const [log, setLog] = useState<ConsoleLogRow[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState({
    username: "",
    password: "",
    name: "",
    phone: "",
    role: "agent",
  });

  const load = useCallback(() => {
    staffList()
      .then((r) => setRows(r.items))
      .catch((e) => setError(e instanceof Error ? e.message : "yuklanmadi"));
    consoleLog()
      .then((r) => setLog(r.items))
      .catch(() => setLog([]));
  }, []);

  useEffect(() => {
    fetchMe().then(setMe).catch(() => setMe(null));
    load();
  }, [load]);

  async function add() {
    setBusy(true);
    setError("");
    try {
      await createStaff(form);
      setForm({ username: "", password: "", name: "", phone: "", role: "agent" });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "saqlanmadi");
    } finally {
      setBusy(false);
    }
  }

  async function patch(id: string, body: Record<string, unknown>) {
    try {
      await updateStaff(id, body);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "saqlanmadi");
    }
  }

  if (me && !me.can.staff) {
    return (
      <div className="card p-6">
        <p className="text-sm text-ink-muted">Bu bo&apos;lim faqat owner uchun.</p>
        <Link href="/console" className="mt-3 inline-block text-sm text-signal-600">
          ← Konsol
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-bold text-ink">Xodimlar</h1>
      {error && <p className="text-sm text-hot-600">{error}</p>}

      <section className="card p-5">
        <h2 className="text-sm font-semibold text-ink">Yangi hisob</h2>
        <p className="mt-1 text-xs text-ink-muted">
          Agent faqat o&apos;zi jalb qilgan mijozlarni ko&apos;radi va restoran
          statistikasini umuman ko&apos;rmaydi. Sotuv menejeri hamma mijozni
          ko&apos;radi, lekin server va hisob-kitobga tegmaydi.
        </p>
        <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
          <input
            className="input"
            placeholder="login"
            value={form.username}
            onChange={(e) => setForm({ ...form, username: e.target.value })}
          />
          <input
            className="input"
            placeholder="ism"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <input
            className="input"
            placeholder="telefon"
            value={form.phone}
            onChange={(e) => setForm({ ...form, phone: e.target.value })}
          />
          <select
            className="select"
            value={form.role}
            onChange={(e) => setForm({ ...form, role: e.target.value })}
          >
            {Object.keys(ROLE_LABEL).map((r) => (
              <option key={r} value={r}>
                {ROLE_LABEL[r]}
              </option>
            ))}
          </select>
          <input
            className="input"
            placeholder="parol (8+)"
            value={form.password}
            onChange={(e) => setForm({ ...form, password: e.target.value })}
          />
        </div>
        <button
          type="button"
          onClick={() => void add()}
          disabled={busy || form.username.length < 3 || form.password.length < 8}
          className="mt-3 rounded-xl bg-ink px-4 py-2 text-sm font-semibold text-surface disabled:opacity-40"
        >
          Qo&apos;shish
        </button>
      </section>

      <section className="card overflow-hidden p-0">
        <table className="w-full text-sm">
          <thead className="border-b border-line bg-raised text-left text-xs text-ink-muted">
            <tr>
              <th className="p-3">Kim</th>
              <th className="p-3">Rol</th>
              <th className="p-3">Mijozlar</th>
              <th className="p-3">Oxirgi kirish</th>
              <th className="p-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {rows.map((s) => (
              <tr key={s.id} className={s.isActive === false ? "opacity-50" : ""}>
                <td className="p-3">
                  <span className="block font-semibold text-ink">{s.name || s.username}</span>
                  <span className="block text-xs text-ink-muted">
                    {s.username}
                    {s.phone ? ` · ${s.phone}` : ""}
                  </span>
                </td>
                <td className="p-3">
                  <select
                    className="select h-9 py-1 text-xs"
                    value={s.role || "owner"}
                    onChange={(e) => void patch(s.id, { role: e.target.value })}
                  >
                    {Object.keys(ROLE_LABEL).map((r) => (
                      <option key={r} value={r}>
                        {ROLE_LABEL[r]}
                      </option>
                    ))}
                  </select>
                </td>
                {/* The column this page exists for. */}
                <td className="p-3 tabular-nums">{s.tenants}</td>
                <td className="p-3 text-xs text-ink-muted">
                  {s.lastLoginAt ? new Date(s.lastLoginAt).toLocaleString() : "—"}
                </td>
                <td className="p-3 text-right">
                  <button
                    type="button"
                    onClick={() => void patch(s.id, { isActive: s.isActive === false })}
                    className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft"
                  >
                    {s.isActive === false ? "Yoqish" : "O'chirish"}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {/* ⚠️ The log is here rather than on its own page: the question it answers —
          "who signed this customer up, and who changed what" — is asked while looking
          at the list of people it names. */}
      <section className="card p-5">
        <h2 className="text-sm font-semibold text-ink">Amallar jurnali</h2>
        <ul className="mt-2 divide-y divide-line text-xs">
          {log.slice(0, 60).map((l) => (
            <li key={l.id} className="flex flex-wrap items-baseline gap-2 py-1.5">
              <span className="text-ink-muted">{new Date(l.at).toLocaleString()}</span>
              <span className="font-semibold text-ink">{l.actor}</span>
              <span className="text-ink-soft">{l.action}</span>
              {l.target && <span className="text-ink-muted">{l.target}</span>}
              {l.detail && <span className="text-ink-muted">· {l.detail}</span>}
            </li>
          ))}
          {log.length === 0 && <li className="py-2 text-ink-muted">Hali yozuv yo&apos;q.</li>}
        </ul>
      </section>
    </div>
  );
}
