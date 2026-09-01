"use client";

// Connecting the restaurant's phone system.
//
// Two things on this screen matter more than the credentials, and both are
// about the same failure: an owner who has typed everything correctly and still
// gets nothing, because onlinePBX was never told where to call.
//
//   • the **webhook address**, shown ready to copy — it cannot be configured
//     without being visible, so hiding it would only make it harder to paste;
//   • **when an event last arrived**, which is the only thing that
//     distinguishes "the key is right" from "the address was actually pasted
//     in". A connection check proves the first and says nothing about the
//     second.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/format";
import type { PBXSettings, PBXSettingsInput } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

const EMPTY: PBXSettingsInput = {
  enabled: false,
  domain: "",
  defaultExtension: "",
};

export default function PBXEditor() {
  const t = useAdminT();
  const { ask } = useAsk();
  const [form, setForm] = useState<PBXSettingsInput>(EMPTY);
  const [stored, setStored] = useState<PBXSettings | null>(null);
  const [origin, setOrigin] = useState("");
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [check, setCheck] = useState<{ ok: boolean; message: string } | null>(
    null,
  );
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    setOrigin(window.location.origin);
    api
      .adminPBX()
      .then((s) => {
        setStored(s);
        setForm({
          enabled: s.enabled,
          domain: s.domain,
          defaultExtension: s.defaultExtension,
        });
      })
      .catch(() => setError(t.common.loadFailed));
  }, [t]);

  async function save(rotate = false) {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const saved = await api.updatePBX({ ...form, rotateToken: rotate });
      setStored(saved);
      // The key goes back to blank: it is stored now, and leaving it in the
      // input only invites a second submit that re-sends it.
      setForm((f) => ({ ...f, apiKey: "" }));
      setMessage(t.pbx.saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function ping() {
    setChecking(true);
    setCheck(null);
    try {
      setCheck(await api.pingPBX());
    } catch (e) {
      setCheck({ ok: false, message: e instanceof Error ? e.message : "" });
    } finally {
      setChecking(false);
    }
  }

  const webhookURL = stored?.webhookPath
    ? `${origin}${stored.webhookPath}`
    : "";

  return (
    <div className="space-y-5">
      <p className="text-sm text-ink-muted">{t.pbx.intro}</p>

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={form.enabled}
          onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
        />
        <span>{t.pbx.enabled}</span>
      </label>

      <div>
        <label className="text-sm font-medium">{t.pbx.domain}</label>
        <input
          className="input mt-1"
          value={form.domain}
          placeholder="example.onpbx.ru"
          onChange={(e) => setForm({ ...form, domain: e.target.value })}
        />
        <p className="mt-1 text-xs text-ink-muted">{t.pbx.domainHint}</p>
      </div>

      <div>
        <label className="text-sm font-medium">
          {t.pbx.apiKey}{" "}
          {stored?.hasApiKey && (
            <span className="text-xs font-normal text-emerald-700 dark:text-emerald-300">
              · {t.payments.stored}
            </span>
          )}
        </label>
        <input
          className="input mt-1"
          type="password"
          autoComplete="new-password"
          value={form.apiKey ?? ""}
          placeholder={stored?.hasApiKey ? t.payments.keepStored : ""}
          onChange={(e) => setForm({ ...form, apiKey: e.target.value })}
        />
        <p className="mt-1 text-xs text-ink-muted">{t.pbx.apiKeyHint}</p>
      </div>

      <div>
        <label className="text-sm font-medium">{t.pbx.defaultExtension}</label>
        <input
          className="input mt-1"
          value={form.defaultExtension}
          onChange={(e) =>
            setForm({ ...form, defaultExtension: e.target.value })
          }
        />
        <p className="mt-1 text-xs text-ink-muted">
          {t.pbx.defaultExtensionHint}
        </p>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={() => save()}
          disabled={saving}
          className="btn btn-primary"
        >
          {saving ? t.common.saving : t.common.save}
        </button>
        <button
          type="button"
          onClick={ping}
          disabled={checking}
          className="btn btn-ghost"
        >
          {checking ? t.pbx.checking : t.pbx.check}
        </button>
        {message && (
          <span className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
            {message}
          </span>
        )}
        {error && <span className="text-sm text-brand">{error}</span>}
      </div>

      {check && (
        <p
          className={`rounded-xl p-3 text-sm ${
            check.ok
              ? "bg-emerald-500/10 font-semibold text-emerald-700 dark:text-emerald-300"
              : "bg-rose-500/10 text-rose-700 dark:text-rose-300"
          }`}
        >
          {check.ok ? `✓ ${check.message}` : check.message}
        </p>
      )}
      {!check && stored?.lastCheckAt && (
        <p className="text-xs text-ink-muted">
          {t.pos.lastCheck(formatDateTime(stored.lastCheckAt))}:{" "}
          {stored.lastCheckOk ? `✓ ${stored.lastCheck}` : stored.lastCheck}
        </p>
      )}

      {/* The half onlinePBX needs from us. */}
      {webhookURL && (
        <div className="rounded-xl bg-ink/5 p-3">
          <p className="text-xs font-semibold">{t.pbx.webhookTitle}</p>
          <div className="mt-2 flex items-center gap-2">
            <code className="min-w-0 flex-1 truncate rounded-lg bg-surface px-2 py-1 text-xs">
              {webhookURL}
            </code>
            <button
              type="button"
              className="chip shrink-0"
              onClick={() => {
                navigator.clipboard.writeText(webhookURL);
                setCopied(true);
                setTimeout(() => setCopied(false), 1500);
              }}
            >
              {copied ? t.payments.copied : t.payments.copy}
            </button>
          </div>
          <p className="mt-2 text-xs text-ink-muted">{t.pbx.webhookHint}</p>

          {/* Whether anything has ever come back. Credentials can be perfect
              and this still say "never" — which is the actual diagnosis. */}
          <p className="mt-2 text-xs">
            {stored?.lastEventAt ? (
              <span className="text-emerald-700 dark:text-emerald-300">
                ✓ {t.pbx.lastEvent(formatDateTime(stored.lastEventAt))}
              </span>
            ) : (
              <span className="text-amber-700 dark:text-amber-300">
                {t.pbx.noEvents}
              </span>
            )}
          </p>

          <button
            type="button"
            onClick={async () => {
              if (await ask({ title: t.pbx.rotateConfirm, danger: true }))
                save(true);
            }}
            className="mt-3 text-xs font-semibold text-brand hover:underline"
          >
            {t.pbx.rotate}
          </button>
          <p className="mt-1 text-xs text-ink-muted">{t.pbx.rotateHint}</p>
        </div>
      )}
    </div>
  );
}

/** The operator's own handset number. Lives on the account page rather than in
 *  settings: it is per-person, and every operator has to set their own. */
export function MyExtension({ initial }: { initial?: string }) {
  const t = useAdminT();
  const [value, setValue] = useState(initial ?? "");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function save() {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      await api.setMyExtension(value.trim());
      setMessage(t.pbx.extensionSaved);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <label className="text-sm font-medium">{t.pbx.myExtension}</label>
      <div className="mt-1 flex gap-2">
        <input
          className="input"
          value={value}
          placeholder="101"
          onChange={(e) => setValue(e.target.value)}
        />
        <button
          type="button"
          onClick={save}
          disabled={saving}
          className="btn btn-ghost shrink-0"
        >
          {saving ? t.common.saving : t.common.save}
        </button>
      </div>
      <p className="mt-1 text-xs text-ink-muted">{t.pbx.myExtensionHint}</p>
      {message && (
        <p className="mt-1 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
          {message}
        </p>
      )}
      {error && <p className="mt-1 text-xs text-brand">{error}</p>}
    </div>
  );
}
