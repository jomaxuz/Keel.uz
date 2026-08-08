"use client";

// One message to one segment.
//
// The segments already existed and led nowhere: the panel could say that eleven
// VIPs had gone quiet and then offer nothing to do about it. This is the other
// half — and because it spends real money on real people's phones, most of the
// screen is there to slow the owner down at the right moments.
//
//   • **Two numbers per segment, always**: how many are in it and how many can
//     actually be messaged. A badge saying 24 next to a send of 19 is a screen
//     people stop believing, and the gap is the useful part.
//   • **The cost is spelled out before sending**, in messages rather than
//     recipients. Non-Latin text is 70 characters per part, not 160, and it is
//     billed per part — so a polite closing sentence can double the bill with
//     nothing on screen changing.
//   • **Sending is a second, separate press** with the count in the button.
//     There is no recalling an SMS.
//   • **Opted-out guests are never in the audience**, and the screen says so
//     rather than hiding it: it is the number that keeps the restaurant welcome.

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/format";
import { ListScroll } from "@/components/admin/PagedList";
import type { Campaign, CampaignPreview, SegmentRow } from "@/lib/types";

type SegKey = keyof ReturnType<typeof useAdminT>["users"]["segment"];

export default function AdminCampaignsPage() {
  const t = useAdminT();
  const [segments, setSegments] = useState<SegmentRow[] | null>(null);
  const [history, setHistory] = useState<Campaign[]>([]);
  const [segment, setSegment] = useState("");
  const [text, setText] = useState("");
  const [preview, setPreview] = useState<CampaignPreview | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    try {
      const [segs, camps] = await Promise.all([
        api.adminSegments(),
        api.adminCampaigns(),
      ]);
      setSegments(segs);
      setHistory(camps);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [t]);

  useEffect(() => {
    load();
  }, [load]);

  // While something is sending, the history is the progress bar — so it is
  // reread until nothing is in flight and then left alone.
  useEffect(() => {
    if (!history.some((c) => c.status === "sending")) return;
    const id = setInterval(() => {
      api.adminCampaigns().then(setHistory).catch(() => {});
    }, 3000);
    return () => clearInterval(id);
  }, [history]);

  // Any edit invalidates the preview. Sending against a stale count is exactly
  // the mistake the preview exists to prevent.
  function edit(next: { segment?: string; text?: string }) {
    if (next.segment !== undefined) setSegment(next.segment);
    if (next.text !== undefined) setText(next.text);
    setPreview(null);
    setMessage("");
  }

  async function runPreview() {
    setBusy(true);
    setError("");
    try {
      setPreview(await api.campaignPreview(segment, text));
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function send() {
    setBusy(true);
    setError("");
    try {
      const res = await api.sendCampaign(segment, text);
      setMessage(t.campaigns.started(res.recipients));
      setText("");
      setPreview(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const chosen = segments?.find((s) => s.segment === segment) ?? null;
  const canPreview = !!segment && text.trim().length > 0 && !busy;

  return (
    <div className="space-y-5">
      <div>
        <h1 className="font-display text-2xl font-bold">{t.campaigns.title}</h1>
        <p className="text-sm text-ink-muted">{t.campaigns.hint}</p>
      </div>

      {/* ---- who ---- */}
      <section className="card p-5">
        <h2 className="font-semibold">{t.campaigns.pickSegment}</h2>
        {segments === null ? (
          <p className="mt-3 text-sm text-ink-muted">{t.common.loading}</p>
        ) : (
          <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
            {segments.map((s) => {
              const active = s.segment === segment;
              return (
                <button
                  key={s.segment}
                  type="button"
                  // A segment nobody can be messaged in is not a choice; it is
                  // disabled rather than hidden so the owner can see it exists
                  // and is empty.
                  disabled={s.reachable === 0}
                  onClick={() => edit({ segment: s.segment })}
                  className={`rounded-2xl border p-3 text-left transition-colors disabled:opacity-40 ${
                    active
                      ? "border-brand bg-brand/5"
                      : "border-line bg-surface hover:border-brand"
                  }`}
                >
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="font-semibold">
                      {t.users.segment[s.segment as SegKey] ?? s.segment}
                    </span>
                    <span className="font-display text-xl font-bold tabular-nums">
                      {s.reachable}
                    </span>
                  </div>
                  <p className="mt-0.5 text-xs text-ink-muted">
                    {t.users.segHint[s.segment as SegKey] ?? ""}
                  </p>
                  {(s.optedOut > 0 || s.noPhone > 0) && (
                    <p className="mt-1 text-xs text-ink-muted">
                      {t.campaigns.excluded(s.optedOut, s.noPhone)}
                    </p>
                  )}
                </button>
              );
            })}
          </div>
        )}
      </section>

      {/* ---- what ---- */}
      <section className="card p-5">
        <h2 className="font-semibold">{t.campaigns.text}</h2>
        <textarea
          className="input mt-2 min-h-28"
          maxLength={480}
          value={text}
          placeholder={t.campaigns.textPh}
          onChange={(e) => edit({ text: e.target.value })}
        />
        <p className="mt-1 text-xs text-ink-muted">{t.campaigns.textHint}</p>

        <div className="mt-3 flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={runPreview}
            disabled={!canPreview}
            className="btn btn-dark disabled:opacity-40"
          >
            {t.campaigns.check}
          </button>

          {preview && (
            <>
              {/* The one number that matters, and it is not "recipients". */}
              <p className="text-sm text-ink-soft">
                {t.campaigns.summary(
                  preview.recipients,
                  preview.parts,
                  preview.messages,
                )}
              </p>
              <button
                type="button"
                onClick={send}
                disabled={busy || preview.recipients === 0 || preview.demo}
                className="btn btn-primary disabled:opacity-40"
              >
                {t.campaigns.send(preview.recipients)}
              </button>
            </>
          )}
        </div>

        {/* Said before the press, not discovered after: with no gateway the
            campaign would "succeed" and reach nobody. */}
        {preview?.demo && (
          <p className="mt-3 rounded-xl border border-amber-500/40 bg-amber-500/5 px-3 py-2 text-sm text-amber-800 dark:text-amber-200">
            {t.campaigns.noGateway}
          </p>
        )}
        {error && <p className="mt-3 text-sm text-brand">{error}</p>}
        {message && (
          <p className="mt-3 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
            {message}
          </p>
        )}
        {chosen && (
          <p className="mt-3 text-xs text-ink-muted">{t.campaigns.optOutNote}</p>
        )}
      </section>

      {/* ---- what was sent ---- */}
      <section className="card p-5">
        <h2 className="font-semibold">{t.campaigns.history}</h2>
        {history.length === 0 ? (
          <p className="mt-3 text-sm text-ink-muted">{t.campaigns.historyEmpty}</p>
        ) : (
          <ListScroll max="max-h-[420px]">
            <ul className="mt-3 divide-y divide-line">
              {history.map((c) => (
                <li key={c.id} className="py-3">
                  <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="font-semibold">
                      {t.users.segment[c.segment as SegKey] ?? c.segment}
                    </span>
                    <span className="text-xs text-ink-muted">
                      {formatDateTime(c.createdAt)} · {c.createdBy}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-ink-soft">{c.text}</p>
                  <p className="mt-1 text-xs tabular-nums text-ink-muted">
                    {c.status === "sending"
                      ? t.campaigns.progress(c.sent + c.failed, c.total)
                      : t.campaigns.result(c.sent, c.failed)}
                  </p>
                  {/* The gateway's own sentence. "84 failed" with no reason
                      leaves three different next steps indistinguishable. */}
                  {c.error && (
                    <p className="mt-1 text-xs text-brand">{c.error}</p>
                  )}
                </li>
              ))}
            </ul>
          </ListScroll>
        )}
      </section>
    </div>
  );
}
