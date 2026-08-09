"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import { me, type Me } from "@/lib/api";
import { AttentionBadge, Field, StatusBadge, statusLabel } from "@/components/dash";
import { BreakdownChart } from "@/components/Charts";
import AdminCredentials from "@/components/AdminCredentials";
import {
  createTenant,
  dayLabel,
  dayShort,
  money,
  moneyShort,
  shortDate,
  tenants,
  updateTenant,
  type TenantRow,
  type TenantStatus,
} from "@/lib/api";

const STATUSES: TenantStatus[] = ["active", "trial", "suspended", "deleted"];

/** Wrapped because `useSearchParams` suspends: the overview links here with
 *  `?attention=…`, and without a boundary Next refuses to render the page. */
export default function TenantsPage() {
  return (
    <Suspense fallback={null}>
      <TenantsList />
    </Suspense>
  );
}

function TenantsList() {
  const { t } = useT();
  const params = useSearchParams();
  const [rows, setRows] = useState<TenantRow[] | null>(null);
  // What this account may do. ⚠️ Asked rather than assumed: the server refuses the
  // rest either way, and a page of buttons that all answer "no permission" teaches
  // somebody their tool is broken.
  const [can, setCan] = useState<Me["can"] | null>(null);
  const canSeeAll = can?.allTenants !== false;
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("");
  // Seeded from the URL so the overview's "3 demo tugagan" lands on exactly
  // those three, and so the filtered view can be sent to somebody as a link.
  const [attention, setAttention] = useState(params.get("attention") ?? "");
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);

  const load = useCallback(() => {
    tenants({ q, status, attention })
      .then((r) => setRows(r.items))
      .catch(() => setError(t.dash.loadFailed));
  }, [q, status, attention, t]);

  // Debounced: typing a shop name should not fire a request per keystroke.
  useEffect(() => {
    me().then((u) => setCan(u.can)).catch(() => setCan(null));
  }, []);

  useEffect(() => {
    const id = setTimeout(load, 250);
    return () => clearTimeout(id);
  }, [load]);

  // Switching a customer off from the list, without opening the card.
  //
  // Confirmed every time, and the question names the site and says what its
  // visitors will see: this button takes a working restaurant offline in the
  // middle of a lunch service if it is pressed on the wrong row, and the rows
  // are one line apart.
  const [busyId, setBusyId] = useState("");
  async function setStatus_(id: string, name: string, next: TenantStatus, ask: string) {
    if (!confirm(ask.replace("{name}", name))) return;
    setBusyId(id);
    setError("");
    try {
      await updateTenant(id, { status: next });
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "…");
    } finally {
      setBusyId("");
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center gap-3">
        <input
          className="input max-w-xs"
          placeholder={t.dash.search}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select
          className="input max-w-[180px]"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="">{t.dash.allStatuses}</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {statusLabel(s, t)}
            </option>
          ))}
        </select>
        <button type="button" className="btn-primary ml-auto" onClick={() => setCreating(true)}>
          {t.dash.newTenant}
        </button>
      </div>

      {/* Buttons rather than another dropdown, for the same reason the call log
          puts its two filters in the open: these are the questions the list
          exists to answer — who runs out this week, who already has, and who
          owes us money — and a question hidden inside a select is a question
          nobody asks. */}
      <div className="flex flex-wrap items-center gap-2">
        {(
          [
            ["down", t.dash.downFilter],
            ["trial_ending", t.dash.trialEndingFilter],
            ["trial_expired", t.dash.trialExpiredFilter],
            ["suspended", t.dash.unpaidFilter],
            ["invoice_due", t.dash.invoiceDueFilter],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            type="button"
            onClick={() => setAttention((a) => (a === key ? "" : key))}
            className={
              attention === key
                ? // `text-surface`, not `text-cream`: cream is a token of the
                  // *tenant* app's palette and does not exist here, so the
                  // class did nothing and the label inherited ink — black on a
                  // black button. A selected filter looked like a blank chip.
                  "rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface"
                : "rounded-xl border border-line px-3 py-2 text-xs font-semibold text-ink-soft hover:bg-raised"
            }
          >
            {label}
          </button>
        ))}
        {attention && (
          <button
            type="button"
            onClick={() => setAttention("")}
            className="text-xs text-ink-muted hover:text-ink"
          >
            {t.dash.clearFilter}
          </button>
        )}
      </div>

      {error && <p className="text-sm text-rose-600 dark:text-rose-400">{error}</p>}

      {creating && (
        <NewTenantForm commercial={!!can?.billing}
          onClose={() => setCreating(false)}
          onCreated={() => {
            setCreating(false);
            load();
          }}
        />
      )}

      {!rows ? (
        <p className="text-sm text-ink-muted">{t.dash.loading}</p>
      ) : rows.length === 0 ? (
        <p className="text-sm text-ink-muted">{t.dash.empty}</p>
      ) : (
        <>
        {/* Above the table, and only when there is something to compare.
            The table answers "what does this customer owe"; the chart answers
            "who are the customers" — and with fifty rows that second question
            is unanswerable by scrolling. Sorted by the same figure the table
            is read for, so the two cannot tell different stories. */}
        {rows.length > 1 && (
          <section className="rounded-2xl border border-line bg-surface p-5">
            <p className="text-sm font-semibold text-ink">{t.dash.billable}</p>
            <div className="mt-4">
              <BreakdownChart
                labels={chartRows(rows).map((r) => r.tenant.name)}
                data={chartRows(rows).map((r) => r.billable)}
              />
            </div>
          </section>
        )}
        <div className="overflow-x-auto rounded-2xl border border-line bg-surface">
          <table className="w-full min-w-[720px] text-sm">
            <thead className="sticky top-0 bg-raised text-left text-xs uppercase tracking-wider text-ink-muted">
              <tr>
                <th className="px-4 py-3">{t.dash.name}</th>
                <th className="px-4 py-3">{t.dash.status}</th>
                {/* The customer's own period, not the calendar month: this is
                    the number read out when a restaurant asks what it owes. */}
                <th className="px-4 py-3 text-right">{t.dash.period}</th>
                <th className="px-4 py-3 text-right">{t.dash.billable}</th>
                {/* Fee ÷ takings. Neither half means anything alone: 12 mln
                    so'm is either 1% of a business or 20% of it, and only one
                    of those customers is about to phone about the price. */}
                <th className="px-4 py-3 text-right">{t.dash.share}</th>
                <th className="px-4 py-3 text-right">{t.dash.lifetime}</th>
                {/* "Ochilgan" moved under the name. Seven columns, three of
                    them long so'm figures, do not fit a 1180px page — and the
                    opening date is the one nobody scans for. */}
                <th className="px-4 py-3 text-right">{t.dash.actions}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {rows.map(({ tenant: x, period, attention: warn, orders, billable, share, lifetime }) => (
                <tr key={x.id} className="hover:bg-raised/60">
                  <td className="px-4 py-3">
                    <Link
                      href={`/console/tenants/${x.id}`}
                      className="font-semibold text-ink hover:underline"
                    >
                      {x.name}
                    </Link>
                    <div className="text-xs text-ink-muted">
                      {x.slug}
                      {x.kind && ` · ${x.kind}`}
                      {` · ${shortDate(x.createdAt)}`}
                      {x.hideWatermark && ` · ${t.dash.watermarkOff}`}
                      {/* ⚠️ Who brought them in, on the row rather than only on the
                          card: the question "which of my people signed this up" is
                          asked while scanning the list, and an answer one click away
                          is an answer nobody reads. Absent on customers created
                          before accounts existed, and absent from an agent's own list
                          where every row would say the same name. */}
                      {x.createdBy && canSeeAll && ` · ${x.createdBy}`}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-col items-start gap-1">
                      <StatusBadge status={x.status} label={statusLabel(x.status, t)} />
                      {/* Shown without filtering, so scanning the list is
                          enough — a warning you have to go looking for is a
                          warning that arrives late. */}
                      <AttentionBadge attention={warn} t={t} />
                      {/* Free terms, on the row rather than only on the card:
                          an operator scanning for who to chase needs to see
                          who is deliberately not being chased, or they chase
                          them. */}
                      {x.free && (
                        <span className="rounded-lg bg-ink/10 px-2 py-0.5 text-[11px] font-semibold text-ink-soft">
                          {x.freeUntil
                            ? t.dash.freeBadgeUntil(dayLabel(x.freeUntil.slice(0, 10)))
                            : t.dash.freeBadge}
                        </span>
                      )}
                      {!x.free && x.discountPercent > 0 && (
                        <span className="rounded-lg bg-ink/10 px-2 py-0.5 text-[11px] font-semibold text-ink-soft">
                          −{x.discountPercent}%
                        </span>
                      )}
                    </div>
                  </td>
                  {/* whitespace-nowrap belt-and-braces: the separator is
                      already non-breaking, and a money column must not be one
                      CSS change away from stacking again. */}
                  {/* Shortened, with the exact figure on hover and in full on
                      the customer's own card. A rounded number is for scanning
                      a list, never for quoting to a customer. */}
                  <td className="whitespace-nowrap px-4 py-3 text-right">
                    <div className="tabular-nums" title={money(orders)}>
                      {moneyShort(orders)}
                    </div>
                    {/* Which days that number covers. Without it the column is
                        a figure with no question attached to it. */}
                    {/* Years dropped: both ends are almost always this year,
                        and the full form was the widest thing in the row while
                        being the least important. */}
                    {period && (
                      <div className="whitespace-nowrap text-xs tabular-nums text-ink-muted">
                        {dayShort(period.from)} — {dayShort(period.to)}
                      </div>
                    )}
                  </td>
                  <td
                    className="whitespace-nowrap px-4 py-3 text-right tabular-nums"
                    title={money(billable)}
                  >
                    {moneyShort(billable)}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums">
                    <ShareCell share={share} t={t} />
                  </td>
                  <td
                    className="whitespace-nowrap px-4 py-3 text-right tabular-nums text-ink-muted"
                    title={money(lifetime?.orders ?? 0)}
                  >
                    {moneyShort(lifetime?.orders ?? 0)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {/* ⚠️ Suspending a customer takes their site down, so the button
                        exists only for the roles allowed to do it. The server refuses
                        the rest regardless — this is so an agent is not shown a
                        control whose only outcome is a refusal. */}
                    {!can?.provision ? null : x.status === "suspended" || x.status === "deleted" ? (
                      <button
                        type="button"
                        disabled={busyId === x.id}
                        onClick={() => setStatus_(x.id, x.name, "active", t.dash.confirmResume)}
                        className="rounded-xl border border-line px-3 py-2 text-xs font-semibold text-ink-soft hover:bg-raised disabled:opacity-50"
                      >
                        {x.status === "deleted" ? t.dash.restore : t.dash.resume}
                      </button>
                    ) : (
                      <button
                        type="button"
                        disabled={busyId === x.id}
                        onClick={() => setStatus_(x.id, x.name, "suspended", t.dash.confirmSuspend)}
                        className="rounded-xl border border-line px-3 py-2 text-xs font-semibold text-ink-soft hover:bg-raised disabled:opacity-50"
                      >
                        {t.dash.suspend}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        </>
      )}
    </div>
  );
}

/** Our fee as a share of the customer's takings.
 *
 *  Coloured by band rather than shown as a bare number, because the number
 *  only means something against a threshold: under ~2% a restaurant never
 *  counts it, over 3% they start. An operator scanning the list should see
 *  which customers are drifting towards that conversation without doing
 *  division in their head.
 *
 *  Blank when there are no takings to compare against — a share of zero
 *  revenue is not 0%, it is unknown, and printing "0%" would read as the
 *  cheapest customer on the list. */
function ShareCell({ share, t }: { share: number; t: ReturnType<typeof useT>["t"] }) {
  if (!share) return <span className="text-ink-muted">—</span>;
  const tone =
    share >= 3
      ? "text-rose-600 dark:text-rose-400"
      : share >= 2
        ? "text-amber-700 dark:text-amber-300"
        : "text-ink-muted";
  return (
    <span className={tone} title={t.dash.shareHint}>
      {share.toFixed(share < 10 ? 2 : 1)}%
    </span>
  );
}

function NewTenantForm({
  onClose,
  onCreated,
  commercial,
}: {
  onClose: () => void;
  onCreated: () => void;
  /** ⚠️ An agent's form is the short one: name, owner, login, password, slug and
   *  domain. The trial and the note are commercial, and the server clears them for an
   *  agent regardless — the request is JSON, so a hidden field would prove nothing.
   *  Hiding them keeps the form honest about what this account can decide. */
  commercial?: boolean;
}) {
  const { t } = useT();
  const [form, setForm] = useState({
    slug: "",
    name: "",
    kind: "",
    domain: "",
    ownerName: "",
    ownerPhone: "",
    adminUsername: "admin",
    adminPassword: "",
    note: "",
  });
  // Default on: a customer created without anybody thinking about it gets an
  // evaluation, not a bill. The opposite default would quietly start charging
  // a restaurant that had only agreed to look at the thing.
  const [trial, setTrial] = useState(true);
  const [trialDays, setTrialDays] = useState("14");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await createTenant({
        ...form,
        trial,
        ...(trial ? { trialDays: Number(trialDays) || 0 } : {}),
      });
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "…");
    } finally {
      setBusy(false);
    }
  }

  const set = (k: keyof typeof form) => (v: string) => setForm((f) => ({ ...f, [k]: v }));

  return (
    <form onSubmit={submit} className="card space-y-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label={t.dash.name} value={form.name} onChange={set("name")} />
        <Field
          label={t.dash.slug}
          hint={t.dash.slugHint}
          value={form.slug}
          onChange={(v) => set("slug")(v.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
        />
        <Field label={t.dash.kind} value={form.kind} onChange={set("kind")} />
        <Field label={t.dash.domains} value={form.domain} onChange={set("domain")} />
        <Field label={t.dash.ownerName} value={form.ownerName} onChange={set("ownerName")} />
        <Field label={t.dash.ownerPhone} value={form.ownerPhone} onChange={set("ownerPhone")} />
      </div>
      {/* A demo or a paying customer from day one — decided once, here.
          The two are not a label: a trial carries a deadline and is switched
          off when it passes, a subscription carries a billing anchor and is
          not. */}
      <div
        className={`rounded-2xl border border-line bg-raised/50 p-4 ${
          commercial ? "" : "hidden"
        }`}
      >
        <label className="flex items-start gap-2 text-sm font-medium">
          <input
            type="checkbox"
            className="mt-1"
            checked={trial}
            onChange={(e) => setTrial(e.target.checked)}
          />
          <span>{t.dash.giveTrial}</span>
        </label>
        {trial ? (
          <div className="mt-3 max-w-[180px]">
            <label className="text-sm font-medium">{t.dash.trialDays}</label>
            <input
              className="input mt-1"
              inputMode="numeric"
              value={trialDays}
              onChange={(e) => setTrialDays(e.target.value.replace(/\D/g, ""))}
            />
          </div>
        ) : (
          <p className="mt-2 text-xs text-ink-muted">{t.dash.trialHint}</p>
        )}
      </div>

      <AdminCredentials
        username={form.adminUsername}
        password={form.adminPassword}
        onUsername={set("adminUsername")}
        onPassword={set("adminPassword")}
      />
      {commercial && (
        <Field label={t.dash.note} value={form.note} onChange={set("note")} />
      )}
      {error && <p className="text-sm text-rose-600 dark:text-rose-400">{error}</p>}
      <div className="flex gap-3">
        <button type="submit" disabled={busy} className="btn-primary">
          {busy ? t.dash.saving : t.dash.create}
        </button>
        <button type="button" onClick={onClose} className="btn-ghost">
          {t.dash.back}
        </button>
      </div>
    </form>
  );
}

/** The customers the chart draws: the biggest ten by what they are billed.
 *
 *  ⚠️ Capped and sorted here rather than drawn from the table's own order. A
 *  bar per customer stops being readable somewhere around fifteen, and the
 *  table is sorted by when they joined — which is the right order to read a
 *  list in and the wrong one to compare magnitudes in. */
function chartRows(rows: TenantRow[]): TenantRow[] {
  return [...rows].sort((a, b) => b.billable - a.billable).slice(0, 10);
}
