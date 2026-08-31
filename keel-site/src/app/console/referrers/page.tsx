"use client";

// Who sends us customers from outside, and what we owe them.
//
// ⚠️ **This screen exists so an arrangement can be honoured.** The register
// engineer who walks into twenty kitchens a week is not competing with us and
// has no reason to mention us — except that the last restaurant they sent was
// paid for. A referral nobody can count is a referral nobody can pay for, and a
// partner who is not paid in the second month stops sending anybody in the
// third.
//
// ⚠️ **The customers behind a number are named, not counted.** "You are owed
// 1 240 000" is a figure to argue about; "these four restaurants, and this is
// what each of them paid" is a figure to agree on — and every conversation this
// page is for is a conversation about money with somebody outside the company.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  createReferrer,
  referrer as fetchReferrer,
  referrers as fetchReferrers,
  updateReferrer,
  type Referrer,
  type ReferrerTenant,
} from "@/lib/api";
import { ORIGIN } from "@/lib/i18n/url";

const EMPTY = {
  name: "",
  code: "",
  contact: "",
  note: "",
  percent: 15,
  months: 12,
};

/** ⚠️ Spaces, not commas — the same grouping every other screen in the product
 *  uses. `uz-UZ` yields "150,000", and a number that is punctuated differently
 *  from the invoice beside it is a number somebody re-reads to be sure it is
 *  the same kind of thing. (`ru-RU` groups with spaces; the replace catches the
 *  browsers that ignore the locale.) */
function money(n: number): string {
  return n.toLocaleString("ru-RU").replace(/,/g, " ") + " so'm";
}

export default function ReferrersPage() {
  const [rows, setRows] = useState<Referrer[]>([]);
  const [open, setOpen] = useState<string | null>(null);
  const [tenants, setTenants] = useState<ReferrerTenant[]>([]);
  const [form, setForm] = useState<typeof EMPTY & { id?: string }>(EMPTY);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    fetchReferrers()
      .then((d) => setRows(d.referrers))
      .catch((e) => setError(String(e.message ?? e)));
  }, []);
  useEffect(load, [load]);

  async function save() {
    setBusy(true);
    setError("");
    try {
      const body = { ...form, isActive: true };
      if (form.id) await updateReferrer(form.id, body);
      else await createReferrer(body);
      setForm(EMPTY);
      load();
    } catch (e) {
      setError(String((e as Error).message ?? e));
    } finally {
      setBusy(false);
    }
  }

  async function toggle(row: Referrer) {
    setBusy(true);
    try {
      await updateReferrer(row.id, { ...row, isActive: !row.isActive });
      load();
    } catch (e) {
      setError(String((e as Error).message ?? e));
    } finally {
      setBusy(false);
    }
  }

  async function show(id: string) {
    if (open === id) {
      setOpen(null);
      return;
    }
    setOpen(id);
    setTenants([]);
    try {
      const d = await fetchReferrer(id);
      setTenants(d.tenants);
    } catch (e) {
      setError(String((e as Error).message ?? e));
    }
  }

  const owed = rows.reduce((n, r) => n + r.commission, 0);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="h-display text-2xl">Hamkorlar</h1>
          <p className="mt-1 max-w-2xl text-sm text-ink-muted">
            Bizni tavsiya qiladigan tashqi odamlar va firmalar: fiskal kassa
            sotuvchilari, qadoq yetkazib beruvchilari, buxgalterlar. Har biriga
            o'z havolasi beriladi va olib kelgan mijozlari shu bo'yicha
            hisoblanadi.
          </p>
        </div>
        {/* Read first, because it is the number the page is opened for. */}
        <div className="rounded-2xl border border-line bg-surface px-5 py-3">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            Jami to'lanishi kerak
          </p>
          <p className="h-display mt-0.5 text-xl">{money(owed)}</p>
        </div>
      </div>

      {error && (
        <p className="rounded-xl bg-signal-500/10 px-4 py-2 text-sm text-signal-600 dark:text-signal-400">
          {error}
        </p>
      )}

      {/* ---- Add or edit ---- */}
      <div className="card">
        <h2 className="h-display text-lg">
          {form.id ? "Hamkorni tahrirlash" : "Yangi hamkor"}
        </h2>
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <label className="block text-sm">
            <span className="text-ink-muted">Nomi</span>
            <input
              className="input mt-1"
              placeholder="Fiskal Servis MChJ"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">Kod (havolada)</span>
            <input
              className="input mt-1"
              placeholder="fiskal"
              value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value })}
            />
            {/* ⚠️ Said here because the code travels by voice: an engineer
                standing in a kitchen reads it off a leaflet out loud. */}
            <span className="mt-1 block text-xs text-ink-muted">
              {ORIGIN}/h/{form.code || "kod"} — qisqa va aytish oson bo'lsin
            </span>
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">Aloqa</span>
            <input
              className="input mt-1"
              placeholder="+998 90 123 45 67 / @username"
              value={form.contact}
              onChange={(e) => setForm({ ...form, contact: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">Foiz</span>
            <input
              type="number"
              min={0}
              max={50}
              className="input mt-1"
              value={form.percent}
              onChange={(e) =>
                setForm({ ...form, percent: Number(e.target.value) || 0 })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">Necha oy (0 — cheksiz)</span>
            <input
              type="number"
              min={0}
              max={60}
              className="input mt-1"
              value={form.months}
              onChange={(e) =>
                setForm({ ...form, months: Number(e.target.value) || 0 })
              }
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">Izoh</span>
            <input
              className="input mt-1"
              value={form.note}
              onChange={(e) => setForm({ ...form, note: e.target.value })}
            />
          </label>
        </div>

        {/* ⚠️ The rule spelled out where it is being set, because it is what
            the partner will be told on the phone and the two must match. */}
        <p className="mt-3 text-sm text-ink-muted">
          Mijoz <strong className="text-ink">to'lagan</strong> pulning{" "}
          {form.percent}% i, obuna boshlangandan{" "}
          {form.months > 0 ? `${form.months} oy davomida` : "cheksiz muddat"}.
          Hisob-fakturadan emas, <strong className="text-ink">kelgan puldan</strong>{" "}
          — to'lanmagan hisob uchun komissiya bermaymiz.
        </p>

        <div className="mt-4 flex gap-2">
          <button
            className="btn-primary"
            disabled={busy || !form.name.trim() || !form.code.trim()}
            onClick={save}
          >
            {form.id ? "Saqlash" : "Qo'shish"}
          </button>
          {form.id && (
            <button className="btn-ghost" onClick={() => setForm(EMPTY)}>
              Bekor qilish
            </button>
          )}
        </div>
      </div>

      {/* ---- The channels ---- */}
      <div className="space-y-3">
        {rows.map((row) => (
          <div key={row.id} className="card">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <h3 className="h-display text-lg">{row.name}</h3>
                  {!row.isActive && (
                    <span className="rounded-full bg-ink/10 px-2 py-0.5 text-xs text-ink-muted">
                      faol emas
                    </span>
                  )}
                </div>
                <p className="mt-1 text-sm text-ink-muted">
                  {ORIGIN}/h/{row.code} · {row.percent}%
                  {row.months > 0 ? ` · ${row.months} oy` : " · cheksiz"}
                  {row.contact ? ` · ${row.contact}` : ""}
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-5 text-right">
                <div>
                  <p className="text-xs uppercase tracking-wide text-ink-muted">
                    Mijoz
                  </p>
                  {/* ⚠️ Both numbers, always. A channel sending twenty trials
                      and no subscribers is sending the wrong twenty, and one
                      figure cannot say that. */}
                  <p className="h-display text-lg">
                    {row.paying}
                    <span className="text-ink-muted"> / {row.tenants}</span>
                  </p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-ink-muted">
                    Kelgan pul
                  </p>
                  <p className="h-display text-lg">{money(row.collected)}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-ink-muted">
                    Komissiya
                  </p>
                  <p className="h-display text-lg text-signal-600 dark:text-signal-400">
                    {money(row.commission)}
                  </p>
                </div>
              </div>
            </div>

            <div className="mt-4 flex flex-wrap gap-2">
              <button className="btn-ghost text-sm" onClick={() => show(row.id)}>
                {open === row.id ? "Yopish" : "Mijozlari"}
              </button>
              <button
                className="btn-ghost text-sm"
                onClick={() =>
                  setForm({
                    id: row.id,
                    name: row.name,
                    code: row.code,
                    contact: row.contact ?? "",
                    note: row.note ?? "",
                    percent: row.percent,
                    months: row.months,
                  })
                }
              >
                Tahrirlash
              </button>
              <Link
                href={`/console/leaflet?ref=${row.code}`}
                className="btn-ghost text-sm"
              >
                Varaqa
              </Link>
              <button
                className="btn-ghost text-sm"
                disabled={busy}
                onClick={() => toggle(row)}
              >
                {row.isActive ? "O'chirish" : "Yoqish"}
              </button>
            </div>

            {open === row.id && (
              <div className="mt-4 overflow-x-auto border-t border-line pt-4">
                <table className="w-full min-w-[560px] text-sm">
                  <thead>
                    <tr className="text-left text-xs uppercase tracking-wide text-ink-muted">
                      <th className="py-2">Mijoz</th>
                      <th className="py-2">Obuna</th>
                      <th className="py-2">Muddat tugaydi</th>
                      <th className="py-2 text-right">To'lagan</th>
                      <th className="py-2 text-right">Komissiya</th>
                    </tr>
                  </thead>
                  <tbody>
                    {tenants.map((tt) => (
                      <tr key={tt.id} className="border-t border-line/60">
                        <td className="py-2">
                          <Link
                            href={`/console/tenants/${tt.id}`}
                            className="font-medium text-ink hover:underline"
                          >
                            {tt.name}
                          </Link>
                          <span className="ml-1 text-ink-muted">{tt.slug}</span>
                        </td>
                        <td className="py-2 text-ink-muted">
                          {tt.subscribedAt
                            ? tt.subscribedAt.slice(0, 10)
                            : "sinov"}
                        </td>
                        <td className="py-2 text-ink-muted">
                          {tt.windowEndsAt
                            ? tt.windowEndsAt.slice(0, 10)
                            : "cheksiz"}
                        </td>
                        <td className="py-2 text-right tabular-nums">
                          {money(tt.collected)}
                        </td>
                        <td className="py-2 text-right tabular-nums">
                          {money(tt.commission)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {tenants.length === 0 && (
                  <p className="py-4 text-sm text-ink-muted">
                    Hali mijoz yo'q. Havolani hamkorga bering va mijoz
                    yaratganda kodni yozing.
                  </p>
                )}
              </div>
            )}
          </div>
        ))}
        {rows.length === 0 && (
          <p className="card text-sm text-ink-muted">
            Hali hamkor yo'q. Birinchisini yuqorida qo'shing — masalan fiskal
            kassa sotadigan firma.
          </p>
        )}
      </div>
    </div>
  );
}
