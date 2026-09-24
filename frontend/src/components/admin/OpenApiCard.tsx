"use client";

// The open API: keys other programs read the restaurant with, and addresses we
// call when an order moves. See handlers/openapi.go, handlers/webhooks.go and
// docs/open-api.md.
//
// ⚠️ **Both secrets are shown once, right after they are made, and never
// again** — the API key because the server keeps only its hash, the signing
// secret because the panel is never sent it. Same rule as UzumTezkorCard, for
// the same reason: a secret the panel could show later is in every screenshot
// from then on, and "make a new one" is the honest answer to "we lost it".

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/orderFlow";
import type { APIKey, WebhookDelivery, WebhookEndpoint } from "@/lib/types";

function errText(e: unknown, fallback: string) {
  return e instanceof Error ? e.message : fallback;
}

/** One value with a copy button — the key, the secret, the address. */
function CopyRow({ label, value }: { label: string; value: string }) {
  const t = useAdminT().openApi;
  const [copied, setCopied] = useState(false);
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // A browser that refuses the clipboard still shows the value to select.
    }
  }
  return (
    <div className="flex flex-wrap items-center gap-2">
      {/* Its own line on a phone: beside a 64-character key the label left the
          value a column four characters wide. */}
      <span className="w-full shrink-0 text-xs text-ink-muted sm:w-32">{label}</span>
      <code className="min-w-0 flex-1 break-all rounded-lg bg-ink/[0.04] px-2 py-1 font-mono text-xs">
        {value}
      </code>
      <button type="button" className="btn px-2 py-1 text-xs" onClick={() => void copy()}>
        {copied ? t.copied : t.copy}
      </button>
    </div>
  );
}

/** The one-time secret, with the warning that goes with it. */
function OnceSecret({ label, value, warning }: { label: string; value: string; warning: string }) {
  return (
    <div className="space-y-2">
      <CopyRow label={label} value={value} />
      <p className="rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
        {warning}
      </p>
    </div>
  );
}

function Checks({
  options,
  labels,
  value,
  onChange,
}: {
  options: string[];
  labels: Record<string, string>;
  value: string[];
  onChange: (v: string[]) => void;
}) {
  return (
    <div className="mt-1 space-y-1">
      {options.map((o) => (
        <label key={o} className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={value.includes(o)}
            onChange={(e) =>
              onChange(e.target.checked ? [...value, o] : value.filter((x) => x !== o))
            }
          />
          {labels[o] ?? o}
          <code className="text-xs text-ink-muted">{o}</code>
        </label>
      ))}
    </div>
  );
}

export default function OpenApiCard() {
  const t = useAdminT().openApi;
  const [basePath, setBasePath] = useState("");
  const host = typeof window !== "undefined" && basePath ? window.location.origin + basePath : "";
  return (
    <div className="space-y-6">
      <p className="text-sm text-ink-soft">{t.intro}</p>
      {host && <CopyRow label={t.baseUrl} value={host} />}
      <KeysPart onBasePath={setBasePath} />
      <WebhooksPart />
    </div>
  );
}

// ---- Keys ----

function KeysPart({ onBasePath }: { onBasePath: (p: string) => void }) {
  const t = useAdminT().openApi;
  const [keys, setKeys] = useState<APIKey[] | null>(null);
  const [scopes, setScopes] = useState<string[]>([]);
  const [name, setName] = useState("");
  const [picked, setPicked] = useState<string[]>([]);
  const [fresh, setFresh] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(() => {
    api
      .adminAPIKeys()
      .then((r) => {
        setKeys(r.keys);
        setScopes(r.scopes);
        onBasePath(r.basePath);
      })
      .catch((e: unknown) => setError(errText(e, t.loadFailed)));
  }, [onBasePath, t.loadFailed]);
  useEffect(load, [load]);

  async function create() {
    setBusy(true);
    setError("");
    try {
      const r = await api.createAPIKey({ name, scopes: picked });
      setFresh(r.secret);
      setName("");
      setPicked([]);
      setKeys((prev) => [r.key, ...(prev ?? [])]);
    } catch (e) {
      setError(errText(e, t.loadFailed));
    } finally {
      setBusy(false);
    }
  }

  async function revoke(k: APIKey) {
    if (!window.confirm(t.revokeConfirm(k.name))) return;
    setError("");
    try {
      const r = await api.revokeAPIKey(k.id);
      setKeys((prev) => (prev ?? []).map((x) => (x.id === k.id ? r : x)));
    } catch (e) {
      setError(errText(e, t.loadFailed));
    }
  }

  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-semibold">{t.keysTitle}</p>
        <p className="mt-0.5 text-xs text-ink-muted">{t.keysHint}</p>
      </div>
      {error && <p className="text-sm text-danger">{error}</p>}
      {fresh && <OnceSecret label={t.newKey} value={fresh} warning={t.keyOnce} />}

      {keys && keys.length === 0 && <p className="text-xs text-ink-muted">{t.noKeys}</p>}
      <ul className="space-y-2">
        {(keys ?? []).map((k) => (
          <li
            key={k.id}
            className={`rounded-xl border border-line p-3 ${k.revokedAt ? "opacity-60" : ""}`}
          >
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-medium">{k.name}</span>
              <code className="text-xs text-ink-muted">{k.prefix}…</code>
              <span className="text-xs text-ink-muted">
                {k.scopes.map((s) => t.scope[s] ?? s).join(" · ")}
              </span>
              <span className="flex-1" />
              {k.revokedAt ? (
                <span className="text-xs text-ink-muted">{t.revoked}</span>
              ) : (
                <button
                  type="button"
                  className="btn px-2 py-1 text-xs text-danger"
                  onClick={() => void revoke(k)}
                >
                  {t.revoke}
                </button>
              )}
            </div>
            <p className="mt-1 text-xs text-ink-muted">
              {t.createdBy(k.createdBy || "—", formatDateTime(k.createdAt))} ·{" "}
              {k.lastUsedAt ? t.lastUsed(formatDateTime(k.lastUsedAt)) : t.neverUsed}
            </p>
          </li>
        ))}
      </ul>

      <div className="rounded-xl border border-line bg-surface p-3">
        <label className="text-sm font-medium">{t.keyName}</label>
        <input
          className="input mt-1"
          value={name}
          maxLength={80}
          placeholder={t.keyNamePlaceholder}
          onChange={(e) => setName(e.target.value)}
        />
        <p className="mt-3 text-sm font-medium">{t.scopes}</p>
        <Checks options={scopes} labels={t.scope} value={picked} onChange={setPicked} />
        <button
          type="button"
          className="btn-primary mt-3 px-4 py-2"
          disabled={busy || !name.trim() || picked.length === 0}
          onClick={() => void create()}
        >
          {t.createKey}
        </button>
      </div>
    </div>
  );
}

// ---- Webhooks ----

function WebhooksPart() {
  const t = useAdminT().openApi;
  const [endpoints, setEndpoints] = useState<WebhookEndpoint[] | null>(null);
  const [events, setEvents] = useState<string[]>([]);
  const [url, setUrl] = useState("");
  const [picked, setPicked] = useState<string[]>([]);
  // The one-time secret, and which endpoint it belongs to.
  const [fresh, setFresh] = useState<{ id: string; secret: string } | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .adminWebhooks()
      .then((r) => {
        setEndpoints(r.endpoints);
        setEvents(r.events);
        setPicked(r.events);
      })
      .catch((e: unknown) => setError(errText(e, t.loadFailed)));
  }, [t.loadFailed]);

  async function create() {
    setBusy(true);
    setError("");
    try {
      const r = await api.createWebhook({ url: url.trim(), events: picked });
      setFresh({ id: r.endpoint.id, secret: r.secret });
      setUrl("");
      setEndpoints((prev) => [...(prev ?? []), r.endpoint]);
    } catch (e) {
      setError(errText(e, t.loadFailed));
    } finally {
      setBusy(false);
    }
  }

  const replace = (ep: WebhookEndpoint) =>
    setEndpoints((prev) => (prev ?? []).map((x) => (x.id === ep.id ? { ...x, ...ep } : x)));

  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-semibold">{t.webhooksTitle}</p>
        <p className="mt-0.5 text-xs text-ink-muted">{t.webhooksHint}</p>
      </div>
      {error && <p className="text-sm text-danger">{error}</p>}

      <ul className="space-y-2">
        {(endpoints ?? []).map((ep) => (
          <WebhookRow
            key={ep.id}
            ep={ep}
            events={events}
            secret={fresh?.id === ep.id ? fresh.secret : ""}
            onSecret={(secret) => setFresh({ id: ep.id, secret })}
            onChange={replace}
            onDelete={() => setEndpoints((prev) => (prev ?? []).filter((x) => x.id !== ep.id))}
          />
        ))}
      </ul>

      <div className="rounded-xl border border-line bg-surface p-3">
        <label className="text-sm font-medium">{t.url}</label>
        <input
          className="input mt-1"
          value={url}
          inputMode="url"
          placeholder="https://"
          onChange={(e) => setUrl(e.target.value)}
        />
        <p className="mt-3 text-sm font-medium">{t.events}</p>
        <Checks options={events} labels={t.event} value={picked} onChange={setPicked} />
        <button
          type="button"
          className="btn-primary mt-3 px-4 py-2"
          disabled={busy || !url.trim().startsWith("https://") || picked.length === 0}
          onClick={() => void create()}
        >
          {t.addWebhook}
        </button>
      </div>
    </div>
  );
}

function WebhookRow({
  ep,
  events,
  secret,
  onSecret,
  onChange,
  onDelete,
}: {
  ep: WebhookEndpoint;
  events: string[];
  secret: string;
  onSecret: (s: string) => void;
  onChange: (ep: WebhookEndpoint) => void;
  onDelete: () => void;
}) {
  const t = useAdminT().openApi;
  const [editing, setEditing] = useState(false);
  const [url, setUrl] = useState(ep.url);
  const [picked, setPicked] = useState(ep.events);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [deliveries, setDeliveries] = useState<WebhookDelivery[] | null>(null);

  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (e) {
      setError(errText(e, t.loadFailed));
    } finally {
      setBusy(false);
    }
  }

  const save = (enabled: boolean, u = url, ev = picked) =>
    run(async () => {
      onChange(await api.updateWebhook(ep.id, { url: u.trim(), events: ev, enabled }));
      setEditing(false);
    });

  const test = () =>
    run(async () => {
      const r = await api.testWebhook(ep.id);
      setNote(r.ok ? t.testOk(r.status, r.ms) : t.testFail(r.error));
    });

  const rotate = () => {
    if (!window.confirm(t.rotateConfirm)) return;
    void run(async () => onSecret((await api.rotateWebhookSecret(ep.id)).secret));
  };

  const remove = () => {
    if (!window.confirm(t.removeConfirm)) return;
    void run(async () => {
      await api.deleteWebhook(ep.id);
      onDelete();
    });
  };

  const loadDeliveries = () =>
    run(async () => setDeliveries((await api.webhookDeliveries(ep.id)).deliveries));

  const retry = (id: string) =>
    run(async () => {
      await api.retryWebhookDelivery(id);
      setDeliveries((await api.webhookDeliveries(ep.id)).deliveries);
    });

  // One line of health: failing beats succeeding beats nothing yet — a
  // receiver that worked on Monday and has failed since is failing.
  const health =
    ep.consecutiveFailures > 0
      ? t.healthFailing(ep.consecutiveFailures, ep.lastError ?? "")
      : ep.lastSuccessAt
        ? t.healthOk(formatDateTime(ep.lastSuccessAt))
        : t.healthNone;

  return (
    <li className={`rounded-xl border border-line p-3 ${ep.enabled ? "" : "opacity-70"}`}>
      <div className="flex flex-wrap items-center gap-2">
        <code className="min-w-0 flex-1 break-all text-xs">{ep.url}</code>
        <label className="flex items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={ep.enabled}
            disabled={busy}
            onChange={(e) => void save(e.target.checked, ep.url, ep.events)}
          />
          {t.enabled}
        </label>
      </div>
      <p className="mt-1 text-xs text-ink-muted">
        {ep.events.map((e) => t.event[e] ?? e).join(" · ")}
      </p>
      <p
        className={`mt-1 text-xs ${
          ep.consecutiveFailures > 0 ? "text-danger" : "text-ink-muted"
        }`}
      >
        {health}
        {ep.pending > 0 && ` · ${t.pending(ep.pending)}`}
      </p>

      {secret && (
        <div className="mt-2">
          <OnceSecret label={t.secret} value={secret} warning={t.secretOnce} />
        </div>
      )}
      {error && <p className="mt-2 text-sm text-danger">{error}</p>}
      {note && <p className="mt-2 text-xs">{note}</p>}

      {editing && (
        <div className="mt-2 space-y-2">
          <input className="input" value={url} onChange={(e) => setUrl(e.target.value)} />
          <Checks options={events} labels={t.event} value={picked} onChange={setPicked} />
        </div>
      )}

      <div className="mt-2 flex flex-wrap gap-2">
        {editing ? (
          <>
            <button
              type="button"
              className="btn-primary px-3 py-1 text-xs"
              disabled={busy || picked.length === 0}
              onClick={() => void save(ep.enabled)}
            >
              {t.save}
            </button>
            <button type="button" className="btn px-3 py-1 text-xs" onClick={() => setEditing(false)}>
              {t.cancel}
            </button>
          </>
        ) : (
          <>
            <button type="button" className="btn px-3 py-1 text-xs" disabled={busy} onClick={() => void test()}>
              {t.test}
            </button>
            <button
              type="button"
              className="btn px-3 py-1 text-xs"
              disabled={busy}
              onClick={() => (deliveries ? setDeliveries(null) : void loadDeliveries())}
            >
              {deliveries ? t.hideDeliveries : t.deliveries}
            </button>
            <button
              type="button"
              className="btn px-3 py-1 text-xs"
              onClick={() => {
                setUrl(ep.url);
                setPicked(ep.events);
                setEditing(true);
              }}
            >
              {t.edit}
            </button>
            <button type="button" className="btn px-3 py-1 text-xs" disabled={busy} onClick={rotate}>
              {t.rotate}
            </button>
            <button type="button" className="btn px-3 py-1 text-xs text-danger" disabled={busy} onClick={remove}>
              {t.remove}
            </button>
          </>
        )}
      </div>

      {deliveries && (
        <div className="mt-3">
          {deliveries.length === 0 ? (
            <p className="text-xs text-ink-muted">{t.noDeliveries}</p>
          ) : (
            <ul className="divide-y divide-line text-xs">
              {deliveries.map((d) => (
                <li key={d.id} className="flex flex-wrap items-center gap-2 py-1.5">
                  <span
                    className={
                      d.status === "delivered"
                        ? "text-emerald-700 dark:text-emerald-400"
                        : d.status === "failed"
                          ? "text-danger"
                          : "text-ink-muted"
                    }
                  >
                    {t.status[d.status] ?? d.status}
                  </span>
                  <span>{t.event[d.event] ?? d.event}</span>
                  {d.orderNumber && <span className="text-ink-muted">#{d.orderNumber}</span>}
                  <span className="text-ink-muted">{formatDateTime(d.createdAt)}</span>
                  <span className="text-ink-muted">{t.attempts(d.attempts)}</span>
                  {d.lastError && d.status !== "delivered" && (
                    <span className="text-ink-muted">{d.lastError}</span>
                  )}
                  <span className="flex-1" />
                  {(d.status === "failed" || d.status === "skipped") && (
                    <button
                      type="button"
                      className="btn px-2 py-0.5 text-xs"
                      disabled={busy}
                      onClick={() => void retry(d.id)}
                    >
                      {t.retry}
                    </button>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </li>
  );
}
