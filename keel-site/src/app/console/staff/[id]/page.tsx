"use client";

// One console account and everything it has done.
//
// ⚠️ **Every figure comes from the records the work already writes** — customers by
// who signed them up, visits by agent, support threads by operator, invoices by who
// issued them, the log by actor. There is no per-person counter anywhere, so there is
// nothing here that can disagree with the screens those records live on.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import {
  deleteStaff,
  money,
  staffDetail,
  updateStaff,
  type StaffDetail,
} from "@/lib/api";
import { useT } from "@/lib/i18n/client";
import { LogBlock, RolePicker } from "@/components/console/StaffParts";

export default function StaffMemberPage() {
  const { t } = useT();
  const d = t.console.staff.detail;
  const router = useRouter();
  const { id } = useParams<{ id: string }>();
  const [data, setData] = useState<StaffDetail | null>(null);
  const [error, setError] = useState("");
  const [missing, setMissing] = useState(false);

  const load = useCallback(() => {
    staffDetail(id)
      .then((x) => {
        setData(x);
        setError("");
      })
      .catch((e) => {
        const msg = e instanceof Error ? e.message : "";
        if (msg === "topilmadi") setMissing(true);
        else setError(msg || t.dash.loadFailed);
      });
  }, [id, t]);

  useEffect(load, [load]);

  async function patch(body: Parameters<typeof updateStaff>[1]) {
    setError("");
    try {
      await updateStaff(id, body);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "saqlanmadi");
    }
  }

  async function remove() {
    if (!data) return;
    const a = data.account;
    if (!window.confirm(t.console.staff.confirmDelete(a.name || a.username))) return;
    try {
      await deleteStaff(id);
      router.replace("/console/staff");
    } catch (e) {
      setError(e instanceof Error ? e.message : "o'chirilmadi");
    }
  }

  const back = (
    <Link href="/console/staff" className="text-sm text-ink-muted hover:text-ink">
      {d.back}
    </Link>
  );

  if (missing) {
    return (
      <div className="space-y-4">
        {back}
        <p className="card p-6 text-sm text-ink-muted">{d.notFound}</p>
      </div>
    );
  }
  if (!data) {
    return (
      <div className="space-y-4">
        {back}
        {error ? (
          <p className="text-sm text-hot-600">{error}</p>
        ) : (
          <p className="text-sm text-ink-muted">{t.dash.loading}</p>
        )}
      </div>
    );
  }

  const a = data.account;
  const off = a.isActive === false;
  const when = (iso?: string) => (iso ? new Date(iso).toLocaleString() : "—");

  return (
    <div className="space-y-6">
      {back}
      {error && <p className="text-sm text-hot-600">{error}</p>}

      <section className="card p-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-ink">{a.name || a.username}</h1>
            <p className="mt-0.5 text-sm text-ink-muted">
              {a.username}
              {a.phone ? ` · ${a.phone}` : ""}
              {" · "}
              <span className={off ? "text-hot-600" : ""}>
                {off ? d.inactive : d.active}
              </span>
            </p>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => void patch({ isActive: off })}
              className="rounded-lg border border-line px-3 py-1.5 text-xs text-ink-soft"
            >
              {off ? t.console.staff.enable : t.console.staff.disable}
            </button>
            <button
              type="button"
              onClick={() => void remove()}
              className="rounded-lg border border-hot-600/40 px-3 py-1.5 text-xs text-hot-600"
            >
              {t.console.staff.deleteForever}
            </button>
          </div>
        </div>
        <div className="mt-4">
          <RolePicker value={a.roles} onChange={(roles) => void patch({ roles })} />
        </div>
        <dl className="mt-4 grid gap-3 text-xs sm:grid-cols-3">
          <Fact label={d.created} value={when(a.createdAt)} />
          <Fact label={d.createdBy} value={a.createdBy || "—"} />
          <Fact label={d.lastLogin} value={a.lastLoginAt ? when(a.lastLoginAt) : d.never} />
        </dl>
      </section>

      <div className="grid gap-px overflow-hidden rounded-2xl border border-line bg-line sm:grid-cols-2 lg:grid-cols-5">
        <Stat
          label={d.customers}
          value={data.tenants.total}
          sub={d.thisMonth(data.tenants.thisMonth)}
        />
        <Stat
          label={d.visits}
          value={data.visits.planned + data.visits.done}
          sub={`${d.done}: ${data.visits.done} · ${d.planned}: ${data.visits.planned}`}
        />
        <Stat
          label={d.support}
          value={data.support.total}
          sub={`${d.open}: ${data.support.open + data.support.waiting} · ${d.closed}: ${data.support.closed}`}
        />
        <Stat label={d.resolved} value={data.reports.resolved} />
        <Stat
          label={t.console.staff.detail.actions}
          value={data.log.total}
          sub={d.thisMonth(data.log.thisMonth)}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <section className="card p-5">
          <h2 className="text-sm font-semibold text-ink">{d.recentCustomers}</h2>
          {Object.keys(data.tenants.byStatus).length > 0 && (
            <p className="mt-1 text-xs text-ink-muted">
              {Object.entries(data.tenants.byStatus)
                .map(([k, n]) => `${d.statuses[k] ?? k}: ${n}`)
                .join(" · ")}
            </p>
          )}
          <ul className="mt-3 max-h-80 divide-y divide-line overflow-y-auto text-sm">
            {data.tenants.recent.map((c) => (
              <li key={c.id} className="flex items-center justify-between gap-3 py-2">
                <Link
                  href={`/console/tenants/${c.id}`}
                  className="min-w-0 truncate font-medium text-ink hover:underline"
                >
                  {c.name || c.slug}
                </Link>
                <span className="shrink-0 text-xs text-ink-muted">
                  {d.statuses[c.status] ?? c.status} ·{" "}
                  {new Date(c.createdAt).toLocaleDateString()}
                </span>
              </li>
            ))}
            {data.tenants.recent.length === 0 && (
              <li className="py-2 text-xs text-ink-muted">{d.noCustomers}</li>
            )}
          </ul>
        </section>

        <section className="card p-5">
          <h2 className="text-sm font-semibold text-ink">{d.recentVisits}</h2>
          <p className="mt-1 text-xs text-ink-muted">
            {d.positive}: {data.visits.positive} · {d.negative}: {data.visits.negative} ·{" "}
            {d.callback}: {data.visits.callback}
          </p>
          <ul className="mt-3 max-h-80 divide-y divide-line overflow-y-auto text-sm">
            {data.visits.recent.map((v) => (
              <li key={v.id} className="py-2">
                <div className="flex items-center justify-between gap-3">
                  <span className="min-w-0 truncate font-medium text-ink">{v.place}</span>
                  <span className="shrink-0 text-xs text-ink-muted">
                    {v.status === "done"
                      ? (v.outcome && t.console.visits.outcomes[
                          v.outcome as keyof typeof t.console.visits.outcomes
                        ]) || d.done
                      : `${d.planned} · ${v.plannedFor}`}
                  </span>
                </div>
                {v.comment && <p className="mt-0.5 text-xs text-ink-muted">{v.comment}</p>}
              </li>
            ))}
            {data.visits.recent.length === 0 && (
              <li className="py-2 text-xs text-ink-muted">{d.noVisits}</li>
            )}
          </ul>
        </section>
      </div>

      {data.invoices.issued > 0 && (
        <section className="card p-5">
          <h2 className="text-sm font-semibold text-ink">{d.invoices}</h2>
          <p className="mt-1 text-sm text-ink-soft">
            {d.issued(data.invoices.issued)} · {money(data.invoices.amount)}
          </p>
        </section>
      )}

      <LogBlock rows={data.log.recent} title={d.actions} />
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-ink-muted">{label}</dt>
      <dd className="mt-0.5 text-ink">{value}</dd>
    </div>
  );
}

function Stat({ label, value, sub }: { label: string; value: number; sub?: string }) {
  return (
    <div className="bg-surface p-5">
      <p className="text-xs uppercase tracking-wider text-ink-muted">{label}</p>
      <p className="h-display mt-2 text-2xl tabular-nums">{value}</p>
      {sub && <p className="mt-1 text-xs text-ink-muted">{sub}</p>}
    </div>
  );
}
