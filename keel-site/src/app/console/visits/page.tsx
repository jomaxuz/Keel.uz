"use client";

// An agent's day: where they plan to go, and what happened when they got there.
//
// ⚠️ **Planned first, judged after** — that order is the feature. A list of visits
// made is a report nobody reads; a list of places to visit is the thing an agent opens
// in the morning, and the same rows become the report once they are closed.
//
// The owner sees everyone's, an agent only their own. That is decided by the server
// (see handlers/visits.go); this page renders whatever it was given, so a change to the
// rule cannot leave a stale copy of it here.

import { useCallback, useEffect, useState } from "react";
import {
  createVisit,
  deleteVisit,
  updateVisit,
  visitList,
  type VisitRow,
} from "@/lib/api";
import { useT } from "@/lib/i18n/client";

// ⚠️ Ids here, words from the dictionary at render: a const map of labels is
// evaluated when the module is imported — before the language is known, and
// never again after it changes.
const OUTCOMES = ["positive", "negative", "callback"] as const;

export default function VisitsPage() {
  const { t } = useT();
  const [rows, setRows] = useState<VisitRow[]>([]);
  const [summary, setSummary] = useState({
    planned: 0,
    positive: 0,
    negative: 0,
  });
  const [canSeeAll, setCanSeeAll] = useState(false);
  const [status, setStatus] = useState("");
  const [q, setQ] = useState("");
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    place: "",
    address: "",
    phone: "",
    plannedFor: "",
  });
  // Which row is being closed, and with what. Held here rather than per row so only
  // one form is open at a time: two half-filled outcomes is how the wrong one gets
  // saved.
  const [closing, setClosing] = useState<{
    id: string;
    outcome: string;
    comment: string;
    nextAt: string;
  } | null>(null);

  const load = useCallback(() => {
    visitList({ status: status || undefined, q: q.trim() || undefined })
      .then((r) => {
        setRows(r.items);
        setSummary(r.summary);
        setCanSeeAll(r.canSeeAll);
      })
      .catch((e) =>
        setError(e instanceof Error ? e.message : t.console.visits.loadFailed),
      );
  }, [status, q]);

  useEffect(() => {
    const timer = window.setTimeout(load, 200);
    return () => window.clearTimeout(timer);
  }, [load]);

  async function add() {
    setError("");
    try {
      await createVisit(form);
      setForm({ place: "", address: "", phone: "", plannedFor: "" });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.console.visits.saveFailed);
    }
  }

  async function close() {
    if (!closing) return;
    setError("");
    try {
      await updateVisit(closing.id, {
        status: "done",
        outcome: closing.outcome,
        comment: closing.comment,
        nextAt: closing.nextAt,
      });
      setClosing(null);
      load();
    } catch (e) {
      // ⚠️ The server refuses a negative outcome with no reason, and a callback with
      // no date. Shown as it came: the sentence explains what to do.
      setError(e instanceof Error ? e.message : t.console.visits.saveFailed);
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h1 className="text-xl font-bold text-ink">{t.console.visits.title}</h1>
        <p className="text-xs text-ink-muted">
          {t.console.visits.summary(
            summary.planned,
            summary.positive,
            summary.negative,
          )}
        </p>
      </div>
      {error && <p className="text-sm text-hot-600">{error}</p>}

      <section className="card p-5">
        <h2 className="text-sm font-semibold text-ink">
          {t.console.visits.newTitle}
        </h2>
        <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <input
            className="input"
            placeholder={t.console.visits.place}
            value={form.place}
            onChange={(e) => setForm({ ...form, place: e.target.value })}
          />
          <input
            className="input"
            placeholder={t.console.visits.address}
            value={form.address}
            onChange={(e) => setForm({ ...form, address: e.target.value })}
          />
          <input
            className="input"
            placeholder={t.console.visits.phone}
            value={form.phone}
            onChange={(e) => setForm({ ...form, phone: e.target.value })}
          />
          <input
            className="input"
            type="date"
            value={form.plannedFor}
            onChange={(e) => setForm({ ...form, plannedFor: e.target.value })}
          />
        </div>
        <button
          type="button"
          onClick={() => void add()}
          disabled={form.place.trim().length < 2}
          className="mt-3 rounded-xl bg-ink px-4 py-2 text-sm font-semibold text-surface disabled:opacity-40"
        >
          {t.console.visits.add}
        </button>
      </section>

      <div className="flex flex-wrap items-center gap-2">
        {[
          { v: "", label: t.console.visits.filters.all },
          { v: "planned", label: t.console.visits.filters.planned },
          { v: "done", label: t.console.visits.filters.done },
        ].map((f) => (
          <button
            key={f.v}
            type="button"
            onClick={() => setStatus(f.v)}
            className={`rounded-full border px-3 py-1.5 text-xs font-semibold ${
              status === f.v
                ? "border-signal-500 bg-raised text-ink"
                : "border-line text-ink-soft"
            }`}
          >
            {f.label}
          </button>
        ))}
        <input
          className="input h-9 max-w-xs px-3 py-1 text-xs"
          placeholder={t.console.visits.search}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      <ul className="space-y-2">
        {rows.map((v) => (
          <li key={v.id} className="card p-4">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <span className="font-semibold text-ink">{v.place}</span>
              <span className="text-xs text-ink-muted">
                {v.status === "planned"
                  ? t.console.visits.planned(v.plannedFor)
                  : v.visitedAt?.slice(0, 10)}
                {canSeeAll && ` · ${v.agentName}`}
              </span>
            </div>
            {(v.address || v.phone) && (
              <p className="mt-1 text-xs text-ink-muted">
                {[v.address, v.phone].filter(Boolean).join(" · ")}
              </p>
            )}
            {v.outcome && (
              <p className="mt-2 text-xs">
                <span
                  className={`rounded-full px-2 py-0.5 font-semibold ${
                    v.outcome === "positive"
                      ? "bg-signal-500/15 text-signal-600"
                      : v.outcome === "negative"
                        ? "bg-hot-500/15 text-hot-600"
                        : "bg-raised text-ink-soft"
                  }`}
                >
                  {outcomeLabel(t, v.outcome)}
                </span>
                {v.nextAt && (
                  <span className="ml-2 text-ink-muted">
                    {t.console.visits.again(v.nextAt)}
                  </span>
                )}
              </p>
            )}
            {v.comment && (
              <p className="mt-1.5 text-sm text-ink-soft">{v.comment}</p>
            )}

            {v.status === "planned" && closing?.id !== v.id && (
              <button
                type="button"
                onClick={() =>
                  setClosing({
                    id: v.id,
                    outcome: "positive",
                    comment: "",
                    nextAt: "",
                  })
                }
                className="mt-3 rounded-xl border border-line px-3 py-1.5 text-xs font-semibold text-ink"
              >
                {t.console.visits.went}
              </button>
            )}

            {closing?.id === v.id && (
              <div className="mt-3 space-y-2 border-t border-line pt-3">
                <div className="flex flex-wrap gap-1.5">
                  {OUTCOMES.map((o) => (
                    <button
                      key={o}
                      type="button"
                      onClick={() => setClosing({ ...closing, outcome: o })}
                      className={`rounded-full border px-3 py-1.5 text-xs font-semibold ${
                        closing.outcome === o
                          ? "border-signal-500 bg-raised text-ink"
                          : "border-line text-ink-soft"
                      }`}
                    >
                      {t.console.visits.outcomes[o]}
                    </button>
                  ))}
                </div>
                <textarea
                  className="input min-h-16 text-xs"
                  placeholder={
                    closing.outcome === "negative"
                      ? t.console.visits.whyNo
                      : t.console.visits.note
                  }
                  value={closing.comment}
                  onChange={(e) =>
                    setClosing({ ...closing, comment: e.target.value })
                  }
                />
                {closing.outcome === "callback" && (
                  <input
                    className="input h-9 px-3 py-1 text-xs"
                    type="date"
                    value={closing.nextAt}
                    onChange={(e) =>
                      setClosing({ ...closing, nextAt: e.target.value })
                    }
                  />
                )}
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => void close()}
                    className="rounded-xl bg-ink px-3 py-1.5 text-xs font-semibold text-surface"
                  >
                    {t.console.visits.save}
                  </button>
                  <button
                    type="button"
                    onClick={() => setClosing(null)}
                    className="rounded-xl border border-line px-3 py-1.5 text-xs text-ink-soft"
                  >
                    {t.console.visits.cancel}
                  </button>
                  <button
                    type="button"
                    onClick={() => void deleteVisit(v.id).then(load)}
                    className="ml-auto rounded-xl border border-line px-3 py-1.5 text-xs text-hot-600"
                  >
                    {t.console.visits.remove}
                  </button>
                </div>
              </div>
            )}
          </li>
        ))}
        {rows.length === 0 && (
          <li className="card p-6 text-sm text-ink-muted">
            {t.console.visits.empty}
          </li>
        )}
      </ul>
    </div>
  );
}

/** ⚠️ Falls back to the id: an outcome recorded before this dictionary knew its
 *  name should still say what it was, not go blank. */
function outcomeLabel(
  t: { console: { visits: { outcomes: Record<string, string> } } },
  outcome: string,
) {
  return t.console.visits.outcomes[outcome] ?? outcome;
}
