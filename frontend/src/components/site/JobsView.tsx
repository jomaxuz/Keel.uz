"use client";

// The vacancies, and one form per vacancy.
//
// ⚠️ **Name and number, and nothing else.** A CV upload, an email, a date of birth — each
// one is a reason not to apply from a phone on a bus, and the restaurant is going to ring
// them anyway. The two fields are what makes that call possible.
//
// ⚠️ **The refusals are shown as sentences, not as a failed button.** The server refuses a
// second application to the same vacancy and more than two in a day — and somebody who
// thinks they applied and did not is worse than somebody who is told to wait. So the
// message from the server is printed as it came.

import { useState } from "react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n/client";
import { localized } from "@/lib/i18n/site-content";
import type { Vacancy } from "@/lib/types";

export default function JobsView({ vacancies }: { vacancies: Vacancy[] }) {
  const { t, lang } = useI18n();
  const [open, setOpen] = useState<string>("");
  const [form, setForm] = useState({ name: "", phone: "", comment: "" });
  const [busy, setBusy] = useState(false);
  const [sent, setSent] = useState<Record<string, boolean>>({});
  const [error, setError] = useState("");

  async function apply(id: string) {
    setBusy(true);
    setError("");
    try {
      await api.applyForVacancy(id, form);
      setSent((prev) => ({ ...prev, [id]: true }));
      setForm({ name: "", phone: "", comment: "" });
      setOpen("");
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  if (vacancies.length === 0) {
    return (
      <div className="container-page py-16">
        <p className="text-sm text-ink-muted">{t.jobs.empty}</p>
      </div>
    );
  }

  return (
    <ul className="container-page space-y-4 py-10">
      {vacancies.map((v) => {
        const title = localized(v.title, lang);
        const description = localized(v.description, lang);
        return (
          <li key={v.id} className="card p-5 sm:p-6">
            <div className="flex flex-wrap items-baseline justify-between gap-3">
              <h2 className="font-display text-lg font-bold">{title}</h2>
              {v.salary && (
                <span className="badge-brand">{v.salary}</span>
              )}
            </div>
            {v.employment && (
              <p className="mt-1 text-xs text-ink-muted">
                {t.jobs.employment[v.employment as keyof typeof t.jobs.employment] ??
                  v.employment}
              </p>
            )}
            {description && (
              <p className="mt-3 whitespace-pre-line text-sm text-ink-soft">{description}</p>
            )}

            {sent[v.id] ? (
              <p className="mt-4 text-sm font-semibold text-brand">{t.jobs.sent}</p>
            ) : open === v.id ? (
              <div className="mt-4 space-y-2 border-t border-line pt-4">
                <input
                  className="input"
                  placeholder={t.jobs.name}
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
                <input
                  className="input"
                  inputMode="tel"
                  placeholder="998 90 123 45 67"
                  value={form.phone}
                  onChange={(e) => setForm({ ...form, phone: e.target.value })}
                />
                <textarea
                  className="input min-h-20"
                  maxLength={500}
                  placeholder={t.jobs.commentPh}
                  value={form.comment}
                  onChange={(e) => setForm({ ...form, comment: e.target.value })}
                />
                {error && <p className="text-sm text-brand">{error}</p>}
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => void apply(v.id)}
                    disabled={busy || form.name.trim().length < 2 || form.phone.trim().length < 9}
                    className="btn btn-primary px-5 py-2.5 disabled:opacity-40"
                  >
                    {busy ? t.common.saving : t.jobs.send}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setOpen("");
                      setError("");
                    }}
                    className="btn btn-ghost px-4 py-2.5"
                  >
                    {t.common.close}
                  </button>
                </div>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => {
                  setOpen(v.id);
                  setError("");
                }}
                className="btn btn-primary mt-4 px-5 py-2.5"
              >
                {t.jobs.apply}
              </button>
            )}
          </li>
        );
      })}
    </ul>
  );
}
