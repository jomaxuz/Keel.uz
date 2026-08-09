"use client";

// Hiring, from the restaurant's side: the vacancies, and the people who answered them.
//
// ⚠️ **One screen, not two.** The question an owner has is "who applied for the waiter
// job" — a vacancy list and an application list on separate pages makes that two
// navigations and a mental join. Selecting a vacancy filters the applications beside it.
//
// ⚠️ An application's status is the point of the list. "new" is a person nobody has rung
// yet, and that is the only number on this page worth acting on — so it is what the list
// opens on.

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime, formatUzPhone } from "@/lib/format";
import type { JobApplication, Vacancy } from "@/lib/types";

const STATUSES = ["new", "called", "hired", "refused"] as const;

export default function VacanciesPage() {
  const t = useAdminT();
  const [rows, setRows] = useState<Vacancy[]>([]);
  const [apps, setApps] = useState<JobApplication[]>([]);
  const [pick, setPick] = useState<string>("");
  const [status, setStatus] = useState<string>("new");
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    title: "",
    description: "",
    salary: "",
    employment: "full",
  });

  const load = useCallback(() => {
    api
      .adminVacancies()
      .then(setRows)
      .catch((e) => setError(e instanceof Error ? e.message : ""));
    api
      .jobApplications({ vacancyId: pick || undefined, status: status || undefined })
      .then(setApps)
      .catch(() => setApps([]));
  }, [pick, status]);

  useEffect(load, [load]);

  async function save() {
    if (form.title.trim().length < 2) return;
    setError("");
    try {
      await api.saveVacancy(null, {
        title: { uz: form.title, ru: "", en: "" },
        description: { uz: form.description, ru: "", en: "" },
        salary: form.salary,
        employment: form.employment,
        isActive: true,
        sortOrder: rows.length,
      });
      setForm({ title: "", description: "", salary: "", employment: "full" });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "");
    }
  }

  return (
    <div className="space-y-5">
      <h1 className="font-display text-2xl font-bold">{t.vacancies.title}</h1>
      <p className="text-sm text-ink-muted">{t.vacancies.hint}</p>
      {error && <p className="text-sm text-brand">{error}</p>}

      <section className="card space-y-3 p-5">
        <h2 className="font-semibold">{t.vacancies.add}</h2>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <input
            className="input"
            placeholder={t.vacancies.position}
            value={form.title}
            onChange={(e) => setForm({ ...form, title: e.target.value })}
          />
          <input
            className="input"
            placeholder={t.vacancies.salaryPh}
            value={form.salary}
            onChange={(e) => setForm({ ...form, salary: e.target.value })}
          />
        </div>
        <textarea
          className="input min-h-20"
          placeholder={t.vacancies.descriptionPh}
          value={form.description}
          onChange={(e) => setForm({ ...form, description: e.target.value })}
        />
        <div className="flex flex-wrap items-center gap-2">
          <select
            className="select h-10 max-w-[200px] py-1"
            value={form.employment}
            onChange={(e) => setForm({ ...form, employment: e.target.value })}
          >
            <option value="full">{t.vacancies.full}</option>
            <option value="part">{t.vacancies.part}</option>
            <option value="shift">{t.vacancies.shift}</option>
          </select>
          <button
            type="button"
            onClick={() => void save()}
            disabled={form.title.trim().length < 2}
            className="btn btn-dark disabled:opacity-40"
          >
            {t.common.add}
          </button>
        </div>
      </section>

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        <section className="card p-5">
          <h2 className="font-semibold">{t.vacancies.open}</h2>
          <ul className="mt-3 space-y-2">
            {rows.map((v) => (
              <li
                key={v.id}
                className={`flex flex-wrap items-center gap-2 rounded-xl border p-3 ${
                  pick === v.id ? "border-brand" : "border-line"
                } ${v.isActive ? "" : "opacity-60"}`}
              >
                <button
                  type="button"
                  onClick={() => setPick(pick === v.id ? "" : v.id)}
                  className="min-w-0 flex-1 text-left"
                >
                  <span className="block font-semibold">{v.title?.uz}</span>
                  <span className="block text-xs text-ink-muted">
                    {v.salary || "—"}
                    {/* The count is why this list is worth scanning: it says which job
                        needs closing and which needs advertising harder. */}
                    {typeof v.applications === "number" && ` · ${t.vacancies.applied(v.applications)}`}
                  </span>
                </button>
                <button
                  type="button"
                  onClick={() =>
                    void api
                      .saveVacancy(v.id, { ...v, isActive: !v.isActive })
                      .then(load)
                  }
                  className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft"
                >
                  {v.isActive ? t.vacancies.close : t.vacancies.reopen}
                </button>
                <button
                  type="button"
                  onClick={() => void api.deleteVacancy(v.id).then(load)}
                  className="rounded-lg border border-line px-2 py-1 text-[11px] text-brand"
                >
                  {t.common.delete}
                </button>
              </li>
            ))}
            {rows.length === 0 && (
              <li className="text-sm text-ink-muted">{t.vacancies.none}</li>
            )}
          </ul>
        </section>

        <section className="card p-5">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 className="font-semibold">{t.vacancies.applications}</h2>
            <div className="flex flex-wrap gap-1">
              {["", ...STATUSES].map((s) => (
                <button
                  key={s || "all"}
                  type="button"
                  onClick={() => setStatus(s)}
                  className={`rounded-full border px-2.5 py-1 text-[11px] font-semibold ${
                    status === s ? "border-brand bg-brand-tint text-brand-dark" : "border-line text-ink-soft"
                  }`}
                >
                  {s ? t.vacancies.status[s as (typeof STATUSES)[number]] : t.orders.filterAll}
                </button>
              ))}
            </div>
          </div>

          <ul className="mt-3 divide-y divide-line">
            {apps.map((a) => (
              <li key={a.id} className="py-3">
                <div className="flex flex-wrap items-baseline justify-between gap-2">
                  <span className="font-semibold">{a.name}</span>
                  <a
                    href={`tel:+${a.phone.replace(/\D/g, "")}`}
                    className="text-sm font-semibold text-brand"
                  >
                    {formatUzPhone(a.phone)}
                  </a>
                </div>
                <p className="text-xs text-ink-muted">
                  {a.vacancyTitle} · {formatDateTime(a.createdAt)}
                </p>
                {a.comment && <p className="mt-1 text-sm text-ink-soft">{a.comment}</p>}
                <div className="mt-2 flex flex-wrap gap-1">
                  {STATUSES.map((s) => (
                    <button
                      key={s}
                      type="button"
                      onClick={() => void api.updateApplication(a.id, { status: s }).then(load)}
                      className={`rounded-full border px-2.5 py-1 text-[11px] ${
                        a.status === s
                          ? "border-brand bg-brand-tint text-brand-dark"
                          : "border-line text-ink-soft"
                      }`}
                    >
                      {t.vacancies.status[s]}
                    </button>
                  ))}
                </div>
              </li>
            ))}
            {apps.length === 0 && (
              <li className="py-3 text-sm text-ink-muted">{t.vacancies.noApps}</li>
            )}
          </ul>
        </section>
      </div>
    </div>
  );
}
