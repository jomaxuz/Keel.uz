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
import {
  seoLlmsPing,
  seoPing,
  seoStatus,
  type LlmsStatus,
  type SeoPingResult,
  type SeoStatus,
} from "@/lib/api";
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
  const [aiBusy, setAiBusy] = useState(false);

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

  async function pingAi() {
    setAiBusy(true);
    setError("");
    try {
      const llms = await seoLlmsPing();
      setStatus((s) => (s ? { ...s, llms } : s));
    } catch (e) {
      setError(String((e as Error).message ?? e));
    } finally {
      setAiBusy(false);
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

      {/* ---- For AI assistants: llms.txt ----

          ⚠️ The paragraph under the title is the honest half: there is no ping
          a language model listens to. What this card pings is IndexNow, and it
          says so — a button promising "tell ChatGPT" would be a button that
          lies about what it did. */}
      <LlmsCard
        d={d}
        llms={status?.llms}
        hasKey={!!status?.hasKey}
        busy={aiBusy}
        copied={copied}
        onCopy={copy}
        onPing={pingAi}
      />

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

function when(v?: string | null): string {
  return v ? new Date(v).toLocaleString() : "";
}

function LlmsCard({
  d,
  llms,
  hasKey,
  busy,
  copied,
  onCopy,
  onPing,
}: {
  d: ReturnType<typeof growthDict>["seo"];
  llms?: LlmsStatus;
  hasKey: boolean;
  busy: boolean;
  copied: string;
  onCopy: (what: string) => void;
  onPing: () => void;
}) {
  const ping = llms?.lastPing;
  const pingOk = ping && (ping.status === 200 || ping.status === 202);
  return (
    <div className="card">
      <h2 className="h-display text-lg">{d.aiTitle}</h2>
      <p className="mt-1 max-w-3xl text-sm text-ink-muted">{d.aiLead}</p>
      <p className="mt-2 max-w-3xl text-xs text-ink-muted">
        {d.aiPingNote.replace("{n}", String(llms?.everyMin ?? 15))}
      </p>

      {/* One row per language: the index, the full text, how big the full
          text is. Both are links, because the first thing anybody does with
          a file like this is open it and read what the machines will read. */}
      <div className="mt-4 space-y-2">
        {(llms?.files ?? []).map((f) => (
          <div
            key={f.lang}
            className="flex flex-wrap items-center gap-2 rounded-2xl border border-line bg-raised px-3 py-2 text-sm"
          >
            <span className="w-8 font-semibold uppercase text-ink">{f.lang}</span>
            {[
              { label: d.aiIndex, url: f.index },
              { label: d.aiFull, url: f.full },
            ].map((x) => (
              <span key={x.url} className="flex items-center gap-1">
                <a
                  href={x.url}
                  target="_blank"
                  rel="noreferrer"
                  className="text-signal-600 underline-offset-4 hover:underline dark:text-signal-400"
                >
                  {x.label}
                </a>
                <button className="btn-ghost px-2 py-1 text-xs" onClick={() => onCopy(x.url)}>
                  {copied === x.url ? d.copied : d.copy}
                </button>
              </span>
            ))}
            {f.pages > 0 && (
              <span className="ml-auto text-xs text-ink-muted">
                {f.pages} {d.aiPages} · {Math.round(f.bytes / 1024)} KB
              </span>
            )}
          </div>
        ))}
      </div>

      <div className="mt-4 grid gap-3 text-sm sm:grid-cols-3">
        <div>
          <p className="text-xs uppercase tracking-wide text-ink-muted">{d.aiChecked}</p>
          <p className="mt-0.5 text-ink">{when(llms?.checkedAt) || d.aiNever}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wide text-ink-muted">{d.aiChanged}</p>
          <p className="mt-0.5 text-ink">{when(llms?.changedAt) || "—"}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-wide text-ink-muted">{d.aiLastPing}</p>
          <p className="mt-0.5 text-ink">
            {ping
              ? `${when(ping.at)} · ${ping.count} · ${ping.status} · ${ping.auto ? d.aiAuto : d.aiManual}`
              : d.aiNoPing}
          </p>
          {ping && !pingOk && (
            <p className="mt-0.5 text-xs text-signal-600 dark:text-signal-400">
              {ping.status === 403 ? d.reason403 : ping.status === 422 ? d.reason422 : ping.message || d.reasonNone}
            </p>
          )}
        </div>
      </div>

      {llms?.error && (
        <p className="mt-3 rounded-xl bg-signal-500/10 px-3 py-2 text-sm text-signal-600 dark:text-signal-400">
          {llms.error}
        </p>
      )}
      {!!llms?.unsent && (
        <p className="mt-3 text-sm text-ink-muted">
          {llms.unsent} {d.aiUnsent}
        </p>
      )}

      {!!llms?.changed?.length && (
        <details className="mt-3 text-sm">
          <summary className="cursor-pointer text-ink-soft">
            {d.aiChangedList} · {llms.changed.length}
          </summary>
          <ul className="mt-2 space-y-1 text-xs text-ink-muted">
            {llms.changed.map((u) => (
              <li key={u} className="break-all">
                {u}
              </li>
            ))}
          </ul>
        </details>
      )}

      <div className="mt-4">
        <button className="btn-primary" disabled={busy || !hasKey} onClick={onPing}>
          {busy ? d.aiBusy : d.aiButton}
        </button>
      </div>
    </div>
  );
}
