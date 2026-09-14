"use client";

// Uzum Tezkor: what the owner hands to the marketplace's manager.
//
// Uzum Tezkor calls this restaurant's server — it takes a token, POSTs each
// order and polls its status. So the owner's whole job here is to give them
// four things: the host, a client id, a secret, and the id of every branch
// (the `restaurantId` each order is addressed to). This card is those four
// things, ready to copy.
//
// ⚠️ **The secret is shown once, right after it is made, and never again.**
// The server keeps only its hash. A secret the panel could show later would be
// in every screenshot and browser cache from then on — and "make a new one" is
// the honest answer to "we lost it", because it also ends every token issued
// under the old one.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/orderFlow";
import type { UzumTezkorSettings } from "@/lib/types";

export default function UzumTezkorCard() {
  const t = useAdminT();
  const [s, setS] = useState<UzumTezkorSettings | null>(null);
  const [secret, setSecret] = useState("");
  const [copied, setCopied] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api
      .adminUzumTezkor()
      .then(setS)
      .catch((e: unknown) =>
        setError(e instanceof Error ? e.message : t.payments.uzumLoadFailed),
      );
  }, [t.payments.uzumLoadFailed]);

  // The host is this panel's own origin: the API is served from the same
  // domain, and it is the domain — never an IP — that Uzum accepts.
  const host =
    s && typeof window !== "undefined" ? window.location.origin + s.basePath : "";

  async function generate() {
    // ⚠️ Asked only when a secret already exists: replacing it stops orders
    // until Uzum types the new one in.
    if (s?.hasSecret && !window.confirm(t.payments.uzumRotateConfirm)) return;
    setBusy(true);
    setError("");
    try {
      const r = await api.generateUzumTezkorCredentials();
      setSecret(r.clientSecret);
      setS((prev) =>
        prev
          ? { ...prev, enabled: true, clientId: r.clientId, hasSecret: true, rotatedAt: r.rotatedAt }
          : prev,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : t.payments.uzumLoadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function toggle(enabled: boolean) {
    setBusy(true);
    setError("");
    try {
      await api.setUzumTezkorEnabled(enabled);
      setS((prev) => (prev ? { ...prev, enabled } : prev));
    } catch (e) {
      setError(e instanceof Error ? e.message : t.payments.uzumLoadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function copy(key: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(key);
      window.setTimeout(() => setCopied((c) => (c === key ? "" : c)), 1500);
    } catch {
      // A browser that refuses the clipboard still shows the value to select.
    }
  }

  const row = (key: string, label: string, value: string, mono = true) => (
    <div className="flex flex-wrap items-center gap-2">
      <span className="w-40 shrink-0 text-xs text-ink-muted">{label}</span>
      <code
        className={`min-w-0 flex-1 break-all rounded-lg bg-ink/[0.04] px-2 py-1 text-xs ${
          mono ? "font-mono" : ""
        }`}
      >
        {value}
      </code>
      <button
        type="button"
        className="btn px-2 py-1 text-xs"
        onClick={() => void copy(key, value)}
      >
        {copied === key ? t.payments.uzumCopied : t.payments.uzumCopy}
      </button>
    </div>
  );

  if (!s) {
    return error ? <p className="text-sm text-danger">{error}</p> : null;
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-ink-soft">{t.payments.uzumIntro}</p>
      {error && <p className="text-sm text-danger">{error}</p>}

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={s.enabled}
          disabled={busy || !s.hasSecret}
          onChange={(e) => void toggle(e.target.checked)}
        />
        {t.payments.uzumEnabled}
      </label>

      <div className="space-y-2">
        {row("host", t.payments.uzumHost, host)}
        {s.clientId && row("clientId", t.payments.uzumClientId, s.clientId)}
        {secret ? (
          <>
            {row("secret", t.payments.uzumSecret, secret)}
            <p className="rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
              {t.payments.uzumSecretOnce}
            </p>
          </>
        ) : (
          <div className="flex flex-wrap items-center gap-2">
            <span className="w-40 shrink-0 text-xs text-ink-muted">
              {t.payments.uzumSecret}
            </span>
            <span className="text-xs text-ink-muted">
              {s.hasSecret ? t.payments.uzumSecretSaved : t.payments.uzumSecretNone}
            </span>
          </div>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          className="btn-primary px-4 py-2"
          disabled={busy}
          onClick={() => void generate()}
        >
          {s.hasSecret ? t.payments.uzumRotate : t.payments.uzumGenerate}
        </button>
        {s.rotatedAt && (
          <span className="text-xs text-ink-muted">
            {t.payments.uzumRotated(formatDateTime(s.rotatedAt))}
          </span>
        )}
      </div>

      <div>
        <p className="text-sm font-medium">{t.payments.uzumStores}</p>
        <p className="mt-0.5 text-xs text-ink-muted">{t.payments.uzumStoresHint}</p>
        <div className="mt-2 space-y-2">
          {s.stores.map((b) => (
            <div key={b.id}>{row(`store-${b.id}`, b.name, b.id)}</div>
          ))}
        </div>
      </div>
    </div>
  );
}
