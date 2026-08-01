"use client";

// Ratings and complaints.
//
// The screen opens on the **unanswered complaints**, not on the average. An
// average is a number to feel good or bad about; an unanswered complaint is a
// customer about to leave, and it is the only list here that has to reach zero.
//
// Closing one demands a line on what was actually done. "Handled" with nothing
// written down makes the list look better without making the restaurant any
// wiser — and six months later nobody can say whether the guest was called back.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { api, ApiError } from "@/lib/api";
import Modal from "@/components/admin/Modal";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import type { Feedback, FeedbackList } from "@/lib/types";
import { formatDateTime } from "@/lib/format";

type Filter = "unhandled" | "low" | "all";

export default function AdminFeedbackPage() {
  const t = useAdminT();
  const [data, setData] = useState<FeedbackList | null>(null);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<Filter>("unhandled");
  const [q, setQ] = useState("");
  const [open, setOpen] = useState<Feedback | null>(null);
  const [resolution, setResolution] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    api
      .adminFeedback({ filter, q: q.trim() || undefined })
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [filter, q]);

  useEffect(() => {
    const id = setTimeout(load, 250);
    return () => clearTimeout(id);
  }, [load]);

  const rows = data?.feedback ?? [];
  const paged = usePaged(rows, 12);

  async function close() {
    if (!open) return;
    setError(null);
    setSaving(true);
    try {
      await api.handleFeedback(open.id, resolution);
      setOpen(null);
      setResolution("");
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h1 className="font-display text-2xl font-bold">{t.feedback.title}</h1>
        <input
          className="input max-w-xs"
          placeholder={t.feedback.searchPh}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      {data && (
        <div className="mt-6 grid gap-4 sm:grid-cols-3">
          <Stat
            label={t.feedback.statOpen}
            value={String(data.stats.open)}
            tone={data.stats.open > 0 ? "alert" : "ok"}
          />
          <Stat
            label={t.feedback.statAverage}
            value={data.stats.count ? data.stats.average.toFixed(2) : "—"}
          />
          <Stat label={t.feedback.statCount} value={String(data.stats.count)} />
        </div>
      )}

      <div className="mt-4 flex flex-wrap gap-2">
        {(["unhandled", "low", "all"] as Filter[]).map((f) => (
          <button
            key={f}
            type="button"
            onClick={() => setFilter(f)}
            className={`rounded-full border px-4 py-1.5 text-sm font-semibold transition-colors ${
              filter === f
                ? "border-brand bg-brand-tint text-brand-dark"
                : "border-line-strong text-ink-soft hover:border-brand"
            }`}
          >
            {t.feedback.filter[f]}
          </button>
        ))}
      </div>

      {loading ? (
        <p className="py-10 text-center text-ink-muted/70">{t.common.loading}</p>
      ) : rows.length === 0 ? (
        <p className="mt-6 rounded-2xl border border-line bg-surface p-6 text-sm text-ink-muted">
          {filter === "unhandled" ? t.feedback.allClear : t.feedback.empty}
        </p>
      ) : (
        <>
          <ListScroll
            className="mt-6 divide-y divide-line rounded-3xl border border-line bg-surface shadow-card"
            max="max-h-[34rem]"
          >
            {paged.pageItems.map((f: Feedback) => (
              <div key={f.id} className="p-4">
                <div className="flex flex-wrap items-center gap-3">
                  <Stars rating={f.rating} />
                  <Link
                    href={`/admin/orders?q=${f.orderNumber}`}
                    className="text-sm font-semibold hover:text-brand"
                  >
                    #{f.orderNumber}
                  </Link>
                  <span className="text-sm text-ink-muted">
                    {f.customer.name} · {f.customer.phone}
                  </span>
                  <span className="ml-auto text-xs text-ink-muted">
                    {formatDateTime(f.createdAt)}
                  </span>
                </div>

                {f.comment && (
                  <p className="mt-2 rounded-xl bg-ink/[0.03] px-3 py-2 text-sm">
                    {f.comment}
                  </p>
                )}

                {f.handled ? (
                  <p className="mt-2 text-xs text-emerald-700 dark:text-emerald-400">
                    ✓ {f.handledBy} — {f.resolution}
                  </p>
                ) : (
                  f.rating <= 3 && (
                    <button
                      type="button"
                      onClick={() => {
                        setOpen(f);
                        setResolution("");
                      }}
                      className="btn-ghost mt-2 px-4 py-1.5 text-sm"
                    >
                      {t.feedback.handle}
                    </button>
                  )
                )}
              </div>
            ))}
          </ListScroll>
          <Pager
            page={paged.page}
            pageCount={paged.pageCount}
            from={paged.from}
            to={paged.to}
            total={paged.total}
            onPage={paged.setPage}
          />
        </>
      )}

      {open && (
        <Modal onClose={() => setOpen(null)}>
          <h2 className="text-lg font-bold">{t.feedback.handleTitle}</h2>
          <div className="mt-2 flex items-center gap-2">
            <Stars rating={open.rating} />
            <span className="text-sm text-ink-muted">#{open.orderNumber}</span>
          </div>
          {open.comment && (
            <p className="mt-3 rounded-xl bg-ink/[0.03] px-3 py-2 text-sm">
              {open.comment}
            </p>
          )}
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.feedback.resolution}</span>
            <textarea
              className="input mt-1 w-full"
              rows={3}
              maxLength={500}
              placeholder={t.feedback.resolutionPh}
              value={resolution}
              onChange={(e) => setResolution(e.target.value)}
              autoFocus
            />
            <span className="mt-1 block text-xs text-ink-muted">
              {t.feedback.resolutionHint}
            </span>
          </label>
          {error && (
            <p className="mt-3 rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand dark:bg-rose-500/10 dark:text-rose-300">
              {error}
            </p>
          )}
          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setOpen(null)}
              className="px-4 py-2 text-sm text-ink-muted hover:text-ink"
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              onClick={close}
              disabled={saving || !resolution.trim()}
              className="btn-primary px-4 py-2 disabled:opacity-60"
            >
              {saving ? t.common.saving : t.feedback.markHandled}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}

function Stars({ rating }: { rating: number }) {
  return (
    <span
      className={`shrink-0 text-lg ${rating <= 3 ? "text-rose-500" : "text-brand"}`}
      aria-label={`${rating}/5`}
    >
      {"★".repeat(rating)}
      <span className="text-ink-muted/30">{"★".repeat(5 - rating)}</span>
    </span>
  );
}

function Stat({
  label,
  value,
  tone = "plain",
}: {
  label: string;
  value: string;
  tone?: "plain" | "alert" | "ok";
}) {
  const cls =
    tone === "alert"
      ? "border-rose-500/40 bg-rose-500/5"
      : tone === "ok"
        ? "border-emerald-500/40 bg-emerald-500/5"
        : "border-line bg-surface";
  return (
    <div className={`rounded-2xl border p-4 shadow-card ${cls}`}>
      <p className="font-display text-2xl font-bold">{value}</p>
      <p className="text-sm text-ink-muted">{label}</p>
    </div>
  );
}
