"use client";

// Telling search engines the site exists.
//
// ⚠️ **The honest part of this screen is the part that says what cannot be
// done.** The obvious feature is one button that submits every page to Google.
// There is no public way to do that: Google's Indexing API is documented for
// job postings and live streams only, and the sitemap ping endpoint was
// withdrawn in 2023. A button that quietly did nothing would be worse than no
// button — somebody would press it and stop asking why nothing was indexed.
//
// So the screen splits into what is pushed and what is pulled. IndexNow pushes,
// and covers Yandex — which in Uzbekistan is not the consolation prize. Google
// pulls, from a sitemap it re-reads on its own once, and the one manual step is
// spelled out here rather than assumed.

import { useCallback, useEffect, useState } from "react";
import { seoPing, seoStatus, type SeoPingResult, type SeoStatus } from "@/lib/api";
import { useT } from "@/lib/i18n/client";
import { growthDict } from "@/lib/i18n/growth";

export default function SeoPage() {
  const { lang } = useT();
  const d = growthDict(lang).seo;

  const [status, setStatus] = useState<SeoStatus | null>(null);
  const [result, setResult] = useState<SeoPingResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState("");

  const load = useCallback(() => {
    seoStatus()
      .then(setStatus)
      .catch((e) => setError(String(e.message ?? e)));
  }, []);
  useEffect(load, [load]);

  async function push() {
    setBusy(true);
    setError("");
    setResult(null);
    try {
      setResult(await seoPing());
      load();
    } catch (e) {
      setError(String((e as Error).message ?? e));
    } finally {
      setBusy(false);
    }
  }

  async function copy(what: string) {
    try {
      await navigator.clipboard.writeText(what);
      setCopied(what);
      window.setTimeout(() => setCopied(""), 1800);
    } catch {
      setError(d.copyFailed);
    }
  }

  const last = status?.last;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="h-display text-2xl">{d.title}</h1>
        <p className="mt-1 max-w-2xl text-sm text-ink-muted">
          {d.leadA} <strong className="text-ink">{d.leadSitemap}</strong>
          {d.leadB}
        </p>
      </div>

      {error && (
        <p className="rounded-xl bg-signal-500/10 px-4 py-2 text-sm text-signal-600 dark:text-signal-400">
          {error}
        </p>
      )}

      {/* ---- What we have ---- */}
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="card">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            {d.cardUrls}
          </p>
          <p className="h-display mt-1 text-2xl">{status?.urls ?? "…"}</p>
          <p className="mt-1 text-xs text-ink-muted">{d.cardUrlsHint}</p>
        </div>
        <div className="card">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            {d.cardKey}
          </p>
          <p className="h-display mt-1 text-2xl">
            {status ? (status.hasKey ? d.keyYes : d.keyNo) : "…"}
          </p>
          <p className="mt-1 text-xs text-ink-muted">
            {status?.hasKey ? d.keyHintYes : d.keyHintNo}
          </p>
        </div>
        <div className="card">
          <p className="text-xs uppercase tracking-wide text-ink-muted">
            {d.cardLast}
          </p>
          <p className="h-display mt-1 text-2xl">
            {last ? new Date(last.lastPingAt).toLocaleDateString() : "—"}
          </p>
          <p className="mt-1 text-xs text-ink-muted">
            {last
              ? `${last.lastCount} ${d.lastUrls} ${last.lastStatus}`
              : d.lastNever}
          </p>
        </div>
      </div>

      {/* ---- Push: Yandex and Bing ---- */}
      <div className="card">
        <h2 className="h-display text-lg">{d.pushTitle}</h2>
        <p className="mt-1 max-w-3xl text-sm text-ink-muted">
          {d.pushLeadA} <strong className="text-ink">{d.pushLeadPush}</strong>{" "}
          {d.pushLeadB}
        </p>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button
            className="btn-primary"
            disabled={busy || !status?.hasKey || !status?.urls}
            onClick={push}
          >
            {busy ? d.pushBusy : d.pushButton}
          </button>
          {!status?.hasKey && (
            <span className="text-sm text-ink-muted">
              {d.needKeyA} <code className="text-ink">INDEXNOW_KEY</code>{" "}
              {d.needKeyB} <code className="text-ink">{status?.keyLocation}</code>{" "}
              {d.needKeyC}
            </span>
          )}
        </div>

        {result && (
          <div className="mt-4 rounded-2xl border border-line bg-raised px-4 py-3 text-sm">
            <p className="font-medium text-ink">
              {result.ok
                ? `${result.urls} ${d.resultOk}`
                : `${d.resultFailed} (${result.status})`}
            </p>
            {/* ⚠️ The engine's own status, not a green tick. 403 means the key
                file is unreadable, 422 means a URL is not on this host — the
                fixes are different, and one flat "failed" names neither. */}
            {!result.ok && (
              <p className="mt-1 text-ink-muted">
                {result.status === 403
                  ? d.reason403
                  : result.status === 422
                    ? d.reason422
                    : result.message || d.reasonNone}
              </p>
            )}
          </div>
        )}
      </div>

      {/* ---- Pull: Google ---- */}
      <div className="card">
        <h2 className="h-display text-lg">{d.googleTitle}</h2>
        {/* ⚠️ This paragraph is the feature. It is what stops somebody looking
            for a button that does not exist and concluding the site is
            broken. */}
        <p className="mt-1 max-w-3xl text-sm text-ink-muted">
          {d.googleLeadA} <strong className="text-ink">{d.googleLeadPush}</strong>{" "}
          {d.googleLeadB} <strong className="text-ink">{d.googleLeadOnce}</strong>{" "}
          {d.googleLeadC}
        </p>
        <ol className="mt-4 space-y-2.5 text-sm text-ink-soft">
          <li className="flex gap-3">
            <span className="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
              1
            </span>
            <span>
              {d.step1}{" "}
              <a
                href="https://search.google.com/search-console"
                target="_blank"
                rel="noreferrer"
                className="text-signal-600 underline-offset-4 hover:underline dark:text-signal-400"
              >
                search.google.com/search-console
              </a>
            </span>
          </li>
          <li className="flex gap-3">
            <span className="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
              2
            </span>
            <span>
              {d.step2A}{" "}
              <strong className="text-ink">{d.step2Section}</strong> {d.step2B}
            </span>
          </li>
          <li className="flex gap-3">
            <span className="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-full bg-signal-500/15 text-xs font-bold text-signal-600 dark:text-signal-400">
              3
            </span>
            <span>{d.step3}</span>
          </li>
        </ol>
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <code className="rounded-xl border border-line bg-raised px-3 py-2 text-sm text-ink">
            {status?.sitemap ?? "…"}
          </code>
          <button
            className="btn-ghost text-sm"
            disabled={!status?.sitemap}
            onClick={() => copy(status!.sitemap)}
          >
            {copied === status?.sitemap ? d.copied : d.copy}
          </button>
          <a
            href={status?.sitemap}
            target="_blank"
            rel="noreferrer"
            className="btn-ghost text-sm"
          >
            {d.open}
          </a>
        </div>
      </div>

      {/* ---- What is already done ---- */}
      <div className="card">
        <h2 className="h-display text-lg">{d.doneTitle}</h2>
        <ul className="mt-3 space-y-2 text-sm text-ink-soft">
          <li>
            • <strong className="text-ink">{d.done1Term}</strong> {d.done1}
          </li>
          <li>
            • <strong className="text-ink">{d.done2Term}</strong> {d.done2}
          </li>
          <li>
            • <strong className="text-ink">{d.done3Term}</strong> {d.done3}
          </li>
          <li>
            • <strong className="text-ink">{d.done4Term}</strong> {d.done4}
          </li>
        </ul>
      </div>
    </div>
  );
}
