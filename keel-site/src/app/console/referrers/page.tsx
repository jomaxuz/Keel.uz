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
import { useT } from "@/lib/i18n/client";
import { growthDict } from "@/lib/i18n/growth";

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
function money(n: number, currency: string): string {
  return n.toLocaleString("ru-RU").replace(/,/g, " ") + " " + currency;
}

export default function ReferrersPage() {
  const { lang } = useT();
  const d = growthDict(lang).ref;
  const sum = (n: number) => money(n, d.currency);

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
          <h1 className="h-display text-2xl">{d.title}</h1>
          <p className="mt-1 max-w-2xl text-sm text-ink-muted">{d.lead}</p>
        </div>
        {/* Read first, because it is the number the page is opened for. */}
        <div className="rounded-2xl border border-line bg-surface px-5 py-3">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            {d.owed}
          </p>
          <p className="h-display mt-0.5 text-xl">{sum(owed)}</p>
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
          {form.id ? d.formEdit : d.formNew}
        </h2>
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <label className="block text-sm">
            <span className="text-ink-muted">{d.name}</span>
            <input
              className="input mt-1"
              placeholder={d.namePh}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">{d.code}</span>
            <input
              className="input mt-1"
              placeholder={d.codePh}
              value={form.code}
              onChange={(e) => setForm({ ...form, code: e.target.value })}
            />
            {/* ⚠️ Said here because the code travels by voice: an engineer
                standing in a kitchen reads it off a leaflet out loud. */}
            <span className="mt-1 block text-xs text-ink-muted">
              {ORIGIN}/h/{form.code || d.codeWord} — {d.codeHint}
            </span>
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">{d.contact}</span>
            <input
              className="input mt-1"
              placeholder={d.contactPh}
              value={form.contact}
              onChange={(e) => setForm({ ...form, contact: e.target.value })}
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-muted">{d.percent}</span>
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
            <span className="text-ink-muted">{d.months}</span>
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
            <span className="text-ink-muted">{d.note}</span>
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
          {d.rule
            .replace("{percent}", String(form.percent))
            .replace(
              "{months}",
              form.months > 0
                ? d.ruleMonths.replace("{n}", String(form.months))
                : d.ruleForever,
            )}{" "}
          {d.ruleNote}
        </p>

        <div className="mt-4 flex gap-2">
          <button
            className="btn-primary"
            disabled={busy || !form.name.trim() || !form.code.trim()}
            onClick={save}
          >
            {form.id ? d.save : d.add}
          </button>
          {form.id && (
            <button className="btn-ghost" onClick={() => setForm(EMPTY)}>
              {d.cancel}
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
                      {d.inactive}
                    </span>
                  )}
                </div>
                <p className="mt-1 text-sm text-ink-muted">
                  {ORIGIN}/h/{row.code} · {row.percent}%
                  {row.months > 0
                    ? ` · ${row.months} ${d.monthsShort}`
                    : ` · ${d.unlimited}`}
                  {row.contact ? ` · ${row.contact}` : ""}
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-5 text-right">
                <div>
                  <p className="text-xs uppercase tracking-wide text-ink-muted">
                    {d.colCustomers}
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
                    {d.colCollected}
                  </p>
                  <p className="h-display text-lg">{sum(row.collected)}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-ink-muted">
                    {d.colCommission}
                  </p>
                  <p className="h-display text-lg text-signal-600 dark:text-signal-400">
                    {sum(row.commission)}
                  </p>
                </div>
              </div>
            </div>

            <div className="mt-4 flex flex-wrap gap-2">
              <button className="btn-ghost text-sm" onClick={() => show(row.id)}>
                {open === row.id ? d.btnClose : d.btnCustomers}
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
                {d.btnEdit}
              </button>
              <Link
                href={`/console/leaflet?ref=${row.code}`}
                className="btn-ghost text-sm"
              >
                {d.btnLeaflet}
              </Link>
              <button
                className="btn-ghost text-sm"
                disabled={busy}
                onClick={() => toggle(row)}
              >
                {row.isActive ? d.btnOff : d.btnOn}
              </button>
            </div>

            {open === row.id && (
              <div className="mt-4 overflow-x-auto border-t border-line pt-4">
                <table className="w-full min-w-[560px] text-sm">
                  <thead>
                    <tr className="text-left text-xs uppercase tracking-wide text-ink-muted">
                      <th className="py-2">{d.thName}</th>
                      <th className="py-2">{d.thSubscribed}</th>
                      <th className="py-2">{d.thWindow}</th>
                      <th className="py-2 text-right">{d.thPaid}</th>
                      <th className="py-2 text-right">{d.thCommission}</th>
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
                            : d.trial}
                        </td>
                        <td className="py-2 text-ink-muted">
                          {tt.windowEndsAt
                            ? tt.windowEndsAt.slice(0, 10)
                            : d.unlimited}
                        </td>
                        <td className="py-2 text-right tabular-nums">
                          {sum(tt.collected)}
                        </td>
                        <td className="py-2 text-right tabular-nums">
                          {sum(tt.commission)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {tenants.length === 0 && (
                  <p className="py-4 text-sm text-ink-muted">{d.noTenants}</p>
                )}
              </div>
            )}
          </div>
        ))}
        {rows.length === 0 && (
          <p className="card text-sm text-ink-muted">{d.empty}</p>
        )}
      </div>
    </div>
  );
}
