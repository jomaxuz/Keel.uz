"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import {
  dayLabel,
  money,
  shortDate,
  tenant as fetchTenant,
  updateTenant,
  type Tenant,
  type TenantDetail,
  type TenantStatus,
} from "@/lib/api";
import { AttentionBadge, Field, StatusBadge, statusLabel } from "@/components/dash";
import AdminCredentials from "@/components/AdminCredentials";
import ProvisionCard from "@/components/ProvisionCard";
import InvoicesPanel from "@/components/InvoicesPanel";
import ExportGrantPanel from "@/components/ExportGrantPanel";
import DesignEditor from "@/components/DesignEditor";
import TenantInsights from "@/components/TenantInsights";

const STATUSES: TenantStatus[] = ["active", "trial", "suspended", "deleted"];

export default function TenantPage() {
  const { t } = useT();
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<TenantDetail | null>(null);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [form, setForm] = useState<Partial<Tenant> | null>(null);
  // Never prefilled from the server — it is never sent back.
  const [adminPassword, setAdminPassword] = useState("");
  // Held apart from the rest of the form: the date input needs a bare local
  // "YYYY-MM-DD", which the server sends as `period.anchor` precisely so the
  // browser never has to slice it out of a UTC timestamp.
  const [subscribedAt, setSubscribedAt] = useState("");

  useEffect(() => {
    fetchTenant(id)
      .then((d) => {
        setData(d);
        setForm(d.tenant);
        setSubscribedAt(d.period?.anchor ?? "");
      })
      .catch(() => setError(t.dash.loadFailed));
  }, [id, t]);

  async function save() {
    if (!form) return;
    setSaving(true);
    setError("");
    setSaved(false);
    try {
      const out = await updateTenant(id, {
        name: form.name,
        kind: form.kind,
        status: form.status,
        pricePerOrder: form.pricePerOrder,
        minMonthly: form.minMonthly ?? 0,
        hideWatermark: form.hideWatermark,
        showcase: form.showcase,
        free: form.free,
        freeReason: form.freeReason ?? "",
        freeUntil: form.freeUntil ? form.freeUntil.slice(0, 10) : "",
        discountPercent: form.discountPercent ?? 0,
        ownerName: form.ownerName,
        ownerPhone: form.ownerPhone,
        note: form.note,
        domains: form.domains,
        adminUsername: form.adminUsername,
        adminPassword,
        // Only when the field actually holds a date. An empty string would be
        // rejected as unparseable, and — worse — omitting the field is what
        // lets the server set the anchor itself when a trial becomes a paying
        // customer. Sending "" would fight that rule.
        ...(subscribedAt ? { subscribedAt } : {}),
      });
      setData(out);
      setForm(out.tenant);
      setSubscribedAt(out.period?.anchor ?? "");
      setAdminPassword("");
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "…");
    } finally {
      setSaving(false);
    }
  }

  async function remove() {
    if (!data) return;
    if (!confirm(t.dash.confirmDelete.replace("{name}", data.tenant.name))) return;
    setSaving(true);
    setError("");
    try {
      const out = await updateTenant(id, { status: "deleted" });
      setData(out);
      setForm(out.tenant);
    } catch (err) {
      setError(err instanceof Error ? err.message : "…");
    } finally {
      setSaving(false);
    }
  }

  if (error && !data) return <p className="text-sm text-rose-600 dark:text-rose-400">{error}</p>;
  if (!data || !form) return <p className="text-sm text-ink-muted">{t.dash.loading}</p>;

  const set = <K extends keyof Tenant>(k: K, v: Tenant[K]) =>
    setForm((f) => ({ ...(f ?? {}), [k]: v }));

  // Read from the server, not summed from `data.days`: that list is capped at
  // 120 rows, so the browser's total quietly becomes wrong the day a customer
  // outlives it — and it becomes wrong in the direction of undercharging.
  const { period, totals, lifetime } = data;
  const isTrial = period?.kind === "trial";

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <Link href="/console/tenants" className="text-sm text-ink-muted hover:text-ink">
          ← {t.dash.back}
        </Link>
        <h1 className="h-display text-2xl">{data.tenant.name}</h1>
        <StatusBadge
          status={data.tenant.status}
          label={statusLabel(data.tenant.status, t)}
        />
        <AttentionBadge attention={data.attention} t={t} />
        <span className="text-sm text-ink-muted">{data.tenant.slug}</span>
      </div>

      <div className="grid gap-px overflow-hidden rounded-2xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-4">
        {[
          {
            k: isTrial ? t.dash.trialPeriod : t.dash.period,
            v: money(totals?.orders ?? 0),
            // Which days the number covers, and — for a paying customer — the
            // date the next invoice falls. That date is the exclusive end of
            // the window, so there is only ever one of them to keep right.
            sub: period
              ? `${dayLabel(period.from)} — ${dayLabel(period.to)}`
              : undefined,
          },
          { k: t.dash.billable, v: money(totals?.billable ?? 0) },
          { k: t.dash.lifetime, v: money(lifetime?.orders ?? 0) },
          {
            k: isTrial ? t.dash.trialEnds : t.dash.nextInvoice,
            v: period ? dayLabel(period.to) : "—",
            sub: `${t.dash.created}: ${shortDate(data.tenant.createdAt)}`,
          },
        ].map(({ k, v, sub }) => (
          <div key={k} className="bg-surface p-5">
            <p className="text-xs uppercase tracking-wider text-ink-muted">{k}</p>
            <p className="h-display mt-2 text-2xl">{v}</p>
            {sub && <p className="mt-1 text-xs tabular-nums text-ink-muted">{sub}</p>}
          </div>
        ))}
      </div>

      {/* Closing is not dropping the database, and the note says so where the
          decision is made — an operator who is not sure what a button destroys
          will either avoid it forever or press it once and find out. */}
      {/* Said on the row somebody is already looking at, because the question
          arrives late: "the site switched itself off" is not an answer, and by
          then the server log has rotated away. */}
      {data.tenant.status === "suspended" && data.tenant.autoSuspendedAt && (
        <p className="rounded-xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-ink-soft">
          {t.dash.autoSuspended} · {shortDate(data.tenant.autoSuspendedAt)}
        </p>
      )}

      {data.tenant.status === "deleted" && (
        <p className="rounded-xl border border-line bg-raised px-4 py-3 text-sm text-ink-soft">
          {t.dash.deletedNote}
        </p>
      )}

      {/* The number above still rests on a date. When that date was never
          recorded it is the day the tenant was opened — a reasonable guess, and
          one an operator correcting an invoice has a right to know about. */}
      {period && !period.anchored && (
        <p className="rounded-xl border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-ink-soft">
          {t.dash.notAnchored}
        </p>
      )}

      <div className="grid gap-6 lg:grid-cols-[1.2fr_.8fr]">
        <section className="card space-y-4">
          <ProvisionCard
            tenant={data.tenant}
            onDone={(x) => {
              setData({ ...data, tenant: x });
              setForm(x);
            }}
          />

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label={t.dash.name} value={form.name ?? ""} onChange={(v) => set("name", v)} />
            <Field label={t.dash.kind} value={form.kind ?? ""} onChange={(v) => set("kind", v)} />
            <Field
              label={t.dash.ownerName}
              value={form.ownerName ?? ""}
              onChange={(v) => set("ownerName", v)}
            />
            <Field
              label={t.dash.ownerPhone}
              value={form.ownerPhone ?? ""}
              onChange={(v) => set("ownerPhone", v)}
            />
            <div>
              <label className="text-sm font-medium">{t.dash.status}</label>
              <select
                className="input mt-1"
                value={form.status}
                onChange={(e) => set("status", e.target.value as TenantStatus)}
              >
                {STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {statusLabel(s, t)}
                  </option>
                ))}
              </select>
            </div>
            <Field
              label={t.dash.pricePerOrder}
              value={String(form.pricePerOrder ?? 0)}
              onChange={(v) => set("pricePerOrder", Number(v.replace(/\D/g, "")) || 0)}
            />
            {/* The other end of the ladder. Next to the per-order price because
                they are one decision: what this customer costs at the bottom
                and what they cost at the top. 0 hands it back to the platform
                default, which is why the hint says so rather than leaving an
                operator to guess whether an empty field means "no floor". */}
            <div>
              <Field
                label={t.dash.minMonthly}
                value={String(form.minMonthly ?? 0)}
                onChange={(v) => set("minMonthly", Number(v.replace(/\D/g, "")) || 0)}
              />
              <p className="mt-1 text-xs text-ink-muted">{t.dash.minMonthlyHint}</p>
            </div>
            {/* The anchor every invoice is counted from. Left empty on a trial
                and filled in by the server the moment that trial becomes a
                paying customer — so the day the money arrived is recorded at
                the one moment anybody knows it. */}
            <div className="sm:col-span-2">
              <label className="text-sm font-medium">{t.dash.subscribedAt}</label>
              <input
                type="date"
                className="input mt-1"
                value={subscribedAt}
                onChange={(e) => setSubscribedAt(e.target.value)}
              />
              <p className="mt-1 text-xs text-ink-muted">{t.dash.subscribedHint}</p>
            </div>
          </div>

          {/* Free terms.
              
              A separate block from the price, and not a price of zero: a zero
              is indistinguishable from a cleared field, produces invoices for
              nothing with no record of why, and does not stop the nightly
              sweep from switching the customer off over a trial date nobody
              meant to apply to them. */}
          <div className="rounded-2xl border border-line bg-raised p-4">
            <label className="flex items-start gap-2 text-sm">
              <input
                type="checkbox"
                className="mt-1"
                checked={!!form.free}
                onChange={(e) => set("free", e.target.checked)}
              />
              <span>
                {t.dash.free}
                <span className="block text-xs text-ink-muted">{t.dash.freeHint}</span>
              </span>
            </label>

            {form.free && (
              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                {/* Required, and refused by the server if empty. An account
                    that pays nothing for a reason nobody wrote down becomes an
                    argument the day somebody asks. */}
                <div className="sm:col-span-2">
                  <label className="text-sm font-medium">{t.dash.freeReason}</label>
                  <input
                    className="input mt-1"
                    value={form.freeReason ?? ""}
                    placeholder={t.dash.freeReasonPlaceholder}
                    onChange={(e) => set("freeReason", e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">{t.dash.freeUntil}</label>
                  <input
                    type="date"
                    className="input mt-1"
                    value={(form.freeUntil ?? "").slice(0, 10)}
                    onChange={(e) => set("freeUntil", e.target.value)}
                  />
                  {/* Empty is forever, and saying so is the point: an anchor
                      customer may well have been promised exactly that. */}
                  <p className="mt-1 text-xs text-ink-muted">{t.dash.freeUntilHint}</p>
                </div>
              </div>
            )}

            {!form.free && (
              <div className="mt-3 max-w-[220px]">
                <label className="text-sm font-medium">{t.dash.discount}</label>
                <input
                  className="input mt-1"
                  inputMode="numeric"
                  value={String(form.discountPercent ?? 0)}
                  onChange={(e) =>
                    set(
                      "discountPercent",
                      Math.min(100, Number(e.target.value.replace(/\D/g, "")) || 0),
                    )
                  }
                />
                <p className="mt-1 text-xs text-ink-muted">{t.dash.discountHint}</p>
              </div>
            )}
          </div>

          <div>
            <label className="text-sm font-medium">{t.dash.domains}</label>
            <textarea
              className="input mt-1 min-h-[84px]"
              value={(form.domains ?? []).join("\n")}
              onChange={(e) => set("domains", e.target.value.split("\n"))}
            />
          </div>

          {/* The paid switch. It lives here and not in the restaurant's own
              settings for the obvious reason: an owner who can turn it off
              will. */}
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={!!form.hideWatermark}
              onChange={(e) => set("hideWatermark", e.target.checked)}
            />
            {/* ⚠️ The price is on the switch, not in a document somewhere.
                It is a charge that appears on the customer's next invoice, and a toggle that
                does not say so reads as a free favour — which is how a three-million-so'm line
                turns into a phone call. The date it was switched on is what makes the first
                invoice fair: the add-on is billed by the day. */}
            <span>
              {t.dash.hideWatermark}
              <span className="mt-0.5 block text-xs text-ink-muted">
                {t.dash.hideWatermarkPrice}
              </span>
            </span>
          </label>

          {/* Off by default, and ticked only after somebody has actually
              asked. A customer who finds their logo on our marketing page
              without being asked is a customer with a complaint. */}
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={!!form.showcase}
              onChange={(e) => set("showcase", e.target.checked)}
            />
            <span>
              {t.dash.showcase}
              <span className="block text-xs text-ink-muted">
                {t.dash.showcaseHint}
              </span>
            </span>
          </label>

          <AdminCredentials
            username={form.adminUsername ?? ""}
            password={adminPassword}
            stored={data.tenant.hasAdminPassword}
            onUsername={(v) => set("adminUsername", v)}
            onPassword={setAdminPassword}
          />

          <Field label={t.dash.note} value={form.note ?? ""} onChange={(v) => set("note", v)} />

          <div className="flex flex-wrap items-center gap-3">
            <button type="button" onClick={save} disabled={saving} className="btn-primary">
              {saving ? t.dash.saving : t.dash.save}
            </button>
            {saved && (
              <span className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                {t.dash.saved}
              </span>
            )}
            {error && <span className="text-sm text-rose-600 dark:text-rose-400">{error}</span>}

            {/* Kept away from Save, and only where a customer is not already
                closed. It is the heaviest thing on the page, so it does not
                also sit in the list one line from "Suspend". */}
            {data.tenant.status !== "deleted" && (
              <button
                type="button"
                onClick={remove}
                disabled={saving}
                className="ml-auto text-sm font-semibold text-rose-600 hover:underline dark:text-rose-400"
              >
                {t.dash.deleteTenant}
              </button>
            )}
          </div>
        </section>

        {/* What is happening at this restaurant right now, and the thirty days
            behind it. Loaded separately from the tenant record: it dials the
            customer's own database, and that must never take down the page
            somebody opened because the container is misbehaving. */}
        <TenantInsights tenantId={data.tenant.id} days={data.days} />

        <section className="card">
          <p className="text-sm font-semibold text-ink">{t.dash.days}</p>
          {data.days.length === 0 ? (
            <p className="mt-3 text-sm text-ink-muted">{t.dash.noData}</p>
          ) : (
            <ul className="mt-4 max-h-[420px] divide-y divide-line overflow-y-auto text-sm">
              {[...data.days].reverse().map((d) => (
                <li key={d.date} className="flex items-center justify-between gap-3 py-2.5">
                  <span className="text-ink-muted">{d.date}</span>
                  <span className="tabular-nums text-ink-soft">
                    {money(d.orders)} · {money(d.billable)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* What this customer was billed and what we actually collected.
            Cash until the MChJ exists — see InvoicesPanel. */}
        <InvoicesPanel tenantId={data.tenant.id} />

        {/* The page-layout constructor. Drawn here because the design is sold as
            a service — the restaurant's own panel has no layout editor, and its
            theme knobs lock once something is published. */}
        {/* ⚠️ The editor is its own page now, and the panel here is a door to it.
            This is the screen somebody sits at for an hour with a customer's
            screenshot open beside it; between an invoice list and a container log
            it got a third of the width and none of the attention. */}
        <DesignEditor tenantId={data.tenant.id} slug={data.tenant.slug} />

        {/* Letting them leave with their data. Last on the page on purpose:
            it is rare, it is dangerous, and it should never be the thing a
            hand lands on while scrolling. */}
        <ExportGrantPanel tenantId={data.tenant.id} />
      </div>
    </div>
  );
}
