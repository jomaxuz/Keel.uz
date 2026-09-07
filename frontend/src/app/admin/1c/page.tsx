"use client";

// The door an accountant's 1C knocks on.
//
// ⚠️ **The whole screen is one address and two credentials**, because that is
// all the accountant has to type — into 1C, once. Everything else on the page
// exists to answer the only question an owner asks afterwards: has it actually
// knocked? A settings form with no evidence of traffic is a form that gets
// filled in wrongly and left, and nobody finds out for a month.
//
// ⚠️ **1C starts the exchange and we never start it.** The office machine is
// usually unreachable from the internet, which is exactly why the published
// protocol works this way. Nothing here can push anything into 1C — and a
// button that claimed to would be a button that does nothing on most days.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError, downloadOneCXML } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type { OneCSettings } from "@/lib/types";

export default function OneCPage() {
  const t = useAdminT();
  const [s, setS] = useState<OneCSettings | null>(null);
  const [form, setForm] = useState({
    login: "",
    password: "",
    days: 31,
    enabled: false,
  });
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminOneC()
      .then((d) => {
        setS(d);
        setForm({
          login: d.login,
          password: "",
          days: d.days,
          enabled: d.enabled,
        });
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [t.common.loadFailed]);

  useEffect(load, [load]);

  async function save() {
    setBusy(true);
    setNote("");
    try {
      const saved = await api.adminSaveOneC({
        enabled: form.enabled,
        login: form.login,
        // ⚠️ Sent only when it was typed: an empty field keeps the stored
        // password, which is the rule on every settings page here.
        password: form.password || undefined,
        days: form.days,
      });
      setS(saved);
      setForm((f) => ({ ...f, password: "" }));
      setNote(t.common.save);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <h1 className="text-2xl font-bold">{t.onec.title}</h1>
      <p className="mt-2 max-w-3xl text-sm text-ink-muted">{t.onec.intro}</p>

      {loading && (
        <p className="mt-6 text-sm text-ink-muted">{t.common.loading}</p>
      )}
      {error && <p className="mt-4 text-sm text-danger">{error}</p>}
      {note && <p className="mt-4 text-sm text-emerald-600">{note}</p>}

      {s && (
        <>
          {/* ⚠️ **The address is given, not described.** It is the one value
              that must be exact, and an owner asked to assemble it from a base
              URL and a path gets it wrong once in three — with a 404 that reads
              as "the integration does not work". */}
          <div className="mt-5 rounded-2xl border border-line bg-surface p-4 shadow-card">
            <div className="text-xs uppercase text-ink-muted/70">
              {t.onec.urlLabel}
            </div>
            <code className="mt-1 block break-all text-sm font-semibold">
              {s.url}
            </code>
            <p className="mt-2 text-xs text-ink-muted">{t.onec.urlHint}</p>
          </div>

          <div className="mt-5 grid gap-3 sm:grid-cols-2">
            <label className="flex items-center gap-2 text-sm sm:col-span-2">
              <input
                type="checkbox"
                checked={form.enabled}
                onChange={(e) =>
                  setForm((f) => ({ ...f, enabled: e.target.checked }))
                }
              />
              {t.onec.enabled}
            </label>
            <label className="block text-sm">
              <span className="text-xs text-ink-muted">{t.onec.login}</span>
              <input
                value={form.login}
                onChange={(e) =>
                  setForm((f) => ({ ...f, login: e.target.value }))
                }
                className="mt-1 w-full rounded-lg border border-line bg-surface px-3 py-2"
              />
            </label>
            <label className="block text-sm">
              <span className="text-xs text-ink-muted">{t.onec.password}</span>
              <input
                type="password"
                value={form.password}
                onChange={(e) =>
                  setForm((f) => ({ ...f, password: e.target.value }))
                }
                className="mt-1 w-full rounded-lg border border-line bg-surface px-3 py-2"
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {s.hasPassword ? t.onec.stored : t.onec.passwordHint}
              </span>
            </label>
            <label className="block text-sm">
              <span className="text-xs text-ink-muted">{t.onec.days}</span>
              <input
                type="number"
                min={1}
                max={366}
                value={form.days}
                onChange={(e) =>
                  setForm((f) => ({ ...f, days: Number(e.target.value) || 31 }))
                }
                className="mt-1 w-full rounded-lg border border-line bg-surface px-3 py-2"
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.onec.daysHint}
              </span>
            </label>
          </div>

          <button
            type="button"
            onClick={save}
            disabled={busy}
            className="mt-4 rounded-lg bg-brand px-4 py-2 text-sm font-semibold text-white disabled:opacity-40"
          >
            {busy ? t.common.saving : t.common.save}
          </button>

          {/* ⚠️ **Evidence of traffic, not a green tick.** "Connected" would be
              a stored flag, and a stored flag goes stale the moment anything
              changes — the lesson `provisionStatus` taught this codebase. A
              date does not go stale. */}
          <div className="mt-6 rounded-2xl border border-line bg-surface p-4 text-sm shadow-card">
            {s.lastSeenAt ? (
              <>
                <div className="font-semibold">
                  {t.onec.lastSeen(formatDate(s.lastSeenAt))}
                </div>
                <div className="mt-1 text-xs text-ink-muted">
                  {t.onec.lastResult(s.lastType || "—", s.lastMode || "—")}
                </div>
                {!!s.lastImported && (
                  <div className="mt-1 text-xs text-ink-muted">
                    {t.onec.imported(s.lastImported)}
                  </div>
                )}
                {!!s.lastExported && (
                  <div className="mt-1 text-xs text-ink-muted">
                    {t.onec.exported(s.lastExported)}
                  </div>
                )}
                {s.lastError && (
                  <div className="mt-2 text-xs text-amber-700">
                    {s.lastError}
                  </div>
                )}
              </>
            ) : (
              <span className="text-ink-muted">{t.onec.never}</span>
            )}
          </div>

          {/* The same XML by hand, for the accountant who will never switch the
              automatic exchange on. */}
          <div className="mt-5">
            <button
              type="button"
              onClick={() =>
                downloadOneCXML(form.days).catch((e) =>
                  setError(
                    e instanceof ApiError ? e.message : t.common.loadFailed,
                  ),
                )
              }
              className="rounded-lg border border-line px-4 py-2 text-sm font-semibold"
            >
              {t.onec.download}
            </button>
            <p className="mt-2 max-w-2xl text-xs text-ink-muted">
              {t.onec.downloadHint}
            </p>
          </div>
        </>
      )}
    </div>
  );
}
