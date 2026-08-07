"use client";

// Online payment credentials.
//
// The restaurant's owner fills this in once, from whatever the bank emailed
// them, and never looks at it again. Two things follow from that:
//
//   • **the webhook URLs are the important half of this screen.** Payme, Click
//     and Uzum each need to be *told* where to call, in their own cabinet, and
//     nothing works until they have been. So the URLs are shown ready to copy
//     rather than left for the owner to work out from a manual.
//   • **a saved secret is never shown again.** The field says "saved" and stays
//     empty; typing into it replaces the stored value, leaving it alone keeps
//     it. A form that rendered the merchant key would put it in every
//     screenshot and browser cache from then on.
//
// The keys are also never sent back by the API — the panel only ever learns
// whether one is stored.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { PaymentSettings, PaymentSettingsInput } from "@/lib/types";

const EMPTY: PaymentSettingsInput = {
  returnUrl: "",
  payme: {
    enabled: false,
    merchantId: "",
    testMode: false,
    accountField: "order_id",
  },
  click: {
    enabled: false,
    serviceId: "",
    merchantId: "",
    merchantUserId: "",
  },
  uzum: {
    enabled: false,
    serviceId: "",
    login: "",
    accountField: "order_id",
  },
  atmos: {
    enabled: false,
    storeId: "",
    baseUrl: "",
  },
};

export default function PaymentsEditor() {
  const t = useAdminT();
  const [form, setForm] = useState<PaymentSettingsInput>(EMPTY);
  const [stored, setStored] = useState<PaymentSettings | null>(null);
  const [origin, setOrigin] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    setOrigin(window.location.origin);
    api
      .adminPaymentSettings()
      .then((s) => {
        setStored(s);
        setForm({
          returnUrl: s.returnUrl,
          payme: {
            enabled: s.payme.enabled,
            merchantId: s.payme.merchantId,
            testMode: s.payme.testMode,
            accountField: s.payme.accountField || "order_id",
          },
          click: {
            enabled: s.click.enabled,
            serviceId: s.click.serviceId,
            merchantId: s.click.merchantId,
            merchantUserId: s.click.merchantUserId,
          },
          uzum: {
            enabled: s.uzum.enabled,
            serviceId: s.uzum.serviceId,
            login: s.uzum.login,
            accountField: s.uzum.accountField || "order_id",
          },
          atmos: {
            enabled: s.atmos.enabled,
            storeId: s.atmos.storeId,
            baseUrl: s.atmos.baseUrl,
          },
        });
      })
      .catch(() => setError(t.common.loadFailed));
  }, [t]);

  async function save() {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const saved = await api.updatePaymentSettings(form);
      setStored(saved);
      // The secrets go back to blank after saving: they are stored now, and
      // leaving them in the inputs would only invite a second submit that
      // re-sends them.
      setForm((f) => ({
        ...f,
        payme: { ...f.payme, key: "", testKey: "" },
        click: { ...f.click, secretKey: "" },
        uzum: { ...f.uzum, password: "" },
        atmos: { ...f.atmos, consumerKey: "", consumerSecret: "", apiKey: "" },
      }));
      setMessage(t.payments.saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  // The base the provider cabinets are given. The panel may be open on an IP or
  // a tunnel, so the field is editable — a webhook pointing at 192.168.x.x is
  // one the bank can never reach.
  const base = form.returnUrl.trim() || origin;
  const hook = (path: string) => `${base.replace(/\/$/, "")}/api/v1${path}`;

  return (
    <div className="space-y-6">
      <p className="text-sm text-ink-muted">{t.payments.intro}</p>

      <div>
        <label className="text-sm font-medium">{t.payments.siteUrl}</label>
        <input
          className="input mt-1"
          value={form.returnUrl}
          placeholder={origin}
          onChange={(e) => setForm({ ...form, returnUrl: e.target.value })}
        />
        <p className="mt-1 text-xs text-ink-muted">{t.payments.siteUrlHint}</p>
      </div>

      {/* ---- Payme ---- */}
      <Provider
        title="Payme"
        enabled={form.payme.enabled}
        onToggle={(enabled) => setForm({ ...form, payme: { ...form.payme, enabled } })}
        hooks={[{ label: t.payments.hookUrl, url: hook("/payments/payme") }]}
        t={t}
      >
        <Field
          label={t.payments.merchantId}
          value={form.payme.merchantId}
          onChange={(v) => setForm({ ...form, payme: { ...form.payme, merchantId: v } })}
        />
        <Secret
          label={t.payments.paymeKey}
          stored={stored?.payme.hasKey}
          value={form.payme.key ?? ""}
          onChange={(v) => setForm({ ...form, payme: { ...form.payme, key: v } })}
          t={t}
        />
        <Secret
          label={t.payments.paymeTestKey}
          stored={stored?.payme.hasTestKey}
          value={form.payme.testKey ?? ""}
          onChange={(v) => setForm({ ...form, payme: { ...form.payme, testKey: v } })}
          t={t}
        />
        <Field
          label={t.payments.accountField}
          value={form.payme.accountField}
          hint={t.payments.accountFieldHint}
          onChange={(v) =>
            setForm({ ...form, payme: { ...form.payme, accountField: v } })
          }
        />
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.payme.testMode}
            onChange={(e) =>
              setForm({ ...form, payme: { ...form.payme, testMode: e.target.checked } })
            }
          />
          <span>{t.payments.testMode}</span>
        </label>
        {form.payme.testMode && (
          <p className="text-xs text-amber-700 dark:text-amber-300">
            {t.payments.testModeWarning}
          </p>
        )}
      </Provider>

      {/* ---- Click ---- */}
      <Provider
        title="Click"
        enabled={form.click.enabled}
        onToggle={(enabled) => setForm({ ...form, click: { ...form.click, enabled } })}
        hooks={[
          { label: t.payments.prepareUrl, url: hook("/payments/click/prepare") },
          { label: t.payments.completeUrl, url: hook("/payments/click/complete") },
        ]}
        t={t}
      >
        <Field
          label={t.payments.serviceId}
          value={form.click.serviceId}
          onChange={(v) => setForm({ ...form, click: { ...form.click, serviceId: v } })}
        />
        <Field
          label={t.payments.merchantId}
          value={form.click.merchantId}
          onChange={(v) => setForm({ ...form, click: { ...form.click, merchantId: v } })}
        />
        <Field
          label={t.payments.merchantUserId}
          value={form.click.merchantUserId}
          onChange={(v) =>
            setForm({ ...form, click: { ...form.click, merchantUserId: v } })
          }
        />
        <Secret
          label={t.payments.secretKey}
          stored={stored?.click.hasSecretKey}
          value={form.click.secretKey ?? ""}
          onChange={(v) => setForm({ ...form, click: { ...form.click, secretKey: v } })}
          t={t}
        />
      </Provider>

      {/* ---- Uzum ---- */}
      <Provider
        title="Uzum"
        enabled={form.uzum.enabled}
        onToggle={(enabled) => setForm({ ...form, uzum: { ...form.uzum, enabled } })}
        hooks={[{ label: t.payments.hookBase, url: hook("/payments/uzum") }]}
        hookNote={t.payments.uzumHookNote}
        t={t}
      >
        <Field
          label={t.payments.serviceId}
          value={form.uzum.serviceId}
          onChange={(v) => setForm({ ...form, uzum: { ...form.uzum, serviceId: v } })}
        />
        <Field
          label={t.payments.login}
          value={form.uzum.login}
          onChange={(v) => setForm({ ...form, uzum: { ...form.uzum, login: v } })}
        />
        <Secret
          label={t.payments.password}
          stored={stored?.uzum.hasPassword}
          value={form.uzum.password ?? ""}
          onChange={(v) => setForm({ ...form, uzum: { ...form.uzum, password: v } })}
          t={t}
        />
        <Field
          label={t.payments.accountField}
          value={form.uzum.accountField}
          hint={t.payments.accountFieldHint}
          onChange={(v) => setForm({ ...form, uzum: { ...form.uzum, accountField: v } })}
        />
      </Provider>

      {/* ---- ATMOS ---- */}
      <Provider
        title="ATMOS"
        enabled={form.atmos.enabled}
        onToggle={(enabled) => setForm({ ...form, atmos: { ...form.atmos, enabled } })}
        hooks={[{ label: t.payments.hookBase, url: hook("/payments/atmos") }]}
        hookNote={t.payments.atmosHookNote}
        t={t}
      >
        <Field
          label={t.payments.storeId}
          value={form.atmos.storeId}
          onChange={(v) => setForm({ ...form, atmos: { ...form.atmos, storeId: v } })}
        />
        <Secret
          label={t.payments.consumerKey}
          stored={stored?.atmos.hasConsumerKey}
          value={form.atmos.consumerKey ?? ""}
          onChange={(v) => setForm({ ...form, atmos: { ...form.atmos, consumerKey: v } })}
          t={t}
        />
        <Secret
          label={t.payments.consumerSecret}
          stored={stored?.atmos.hasConsumerSecret}
          value={form.atmos.consumerSecret ?? ""}
          onChange={(v) => setForm({ ...form, atmos: { ...form.atmos, consumerSecret: v } })}
          t={t}
        />
        {/* Kept last and labelled apart from the OAuth pair on purpose: this is
            the one ATMOS signs its callbacks with, and pasting the consumer
            secret here produces a gateway that looks configured, sends the
            guest to a real payment page, and then refuses the confirmation. */}
        <Secret
          label={t.payments.atmosApiKey}
          stored={stored?.atmos.hasApiKey}
          value={form.atmos.apiKey ?? ""}
          onChange={(v) => setForm({ ...form, atmos: { ...form.atmos, apiKey: v } })}
          t={t}
        />
        <Field
          label={t.payments.baseUrl}
          value={form.atmos.baseUrl}
          hint={t.payments.baseUrlHint}
          onChange={(v) => setForm({ ...form, atmos: { ...form.atmos, baseUrl: v } })}
        />
      </Provider>

      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={save}
          disabled={saving}
          className="btn btn-primary"
        >
          {saving ? t.common.saving : t.common.save}
        </button>
        {message && (
          <span className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
            {message}
          </span>
        )}
        {error && <span className="text-sm text-brand">{error}</span>}
      </div>
    </div>
  );
}

function Provider({
  title,
  enabled,
  onToggle,
  hooks,
  hookNote,
  children,
  t,
}: {
  title: string;
  enabled: boolean;
  onToggle: (v: boolean) => void;
  hooks: { label: string; url: string }[];
  hookNote?: string;
  children: React.ReactNode;
  t: ReturnType<typeof useAdminT>;
}) {
  return (
    <div className="rounded-2xl border border-line p-4">
      <label className="flex items-center gap-3">
        <input
          type="checkbox"
          checked={enabled}
          onChange={(e) => onToggle(e.target.checked)}
        />
        <span className="font-display text-lg font-bold">{title}</span>
      </label>

      {enabled && (
        <div className="mt-4 space-y-3">
          {children}

          {/* The half the bank needs from us. Nothing works until these are
              pasted into the provider's own cabinet, so they are not hidden
              behind a "developer" heading. */}
          <div className="rounded-xl bg-ink/5 p-3">
            <p className="text-xs font-semibold">{t.payments.hooksTitle}</p>
            {hooks.map((h) => (
              <CopyRow key={h.url} label={h.label} value={h.url} t={t} />
            ))}
            {hookNote && (
              <p className="mt-2 text-xs text-ink-muted">{hookNote}</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function CopyRow({
  label,
  value,
  t,
}: {
  label: string;
  value: string;
  t: ReturnType<typeof useAdminT>;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="mt-2">
      <div className="text-xs text-ink-muted">{label}</div>
      <div className="mt-1 flex items-center gap-2">
        <code className="min-w-0 flex-1 truncate rounded-lg bg-surface px-2 py-1 text-xs">
          {value}
        </code>
        <button
          type="button"
          className="chip shrink-0"
          onClick={() => {
            navigator.clipboard.writeText(value);
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
          }}
        >
          {copied ? t.payments.copied : t.payments.copy}
        </button>
      </div>
    </div>
  );
}

function Field({
  label,
  value,
  hint,
  onChange,
}: {
  label: string;
  value: string;
  hint?: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <label className="text-sm font-medium">{label}</label>
      <input
        className="input mt-1"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      {hint && <p className="mt-1 text-xs text-ink-muted">{hint}</p>}
    </div>
  );
}

/** A secret field. Empty means "keep what is stored", which is why the
 *  placeholder says so rather than leaving the box looking unset. */
function Secret({
  label,
  stored,
  value,
  onChange,
  t,
}: {
  label: string;
  stored?: boolean;
  value: string;
  onChange: (v: string) => void;
  t: ReturnType<typeof useAdminT>;
}) {
  return (
    <div>
      <label className="text-sm font-medium">
        {label}{" "}
        {stored && (
          <span className="text-xs font-normal text-emerald-700 dark:text-emerald-300">
            · {t.payments.stored}
          </span>
        )}
      </label>
      <input
        className="input mt-1"
        type="password"
        autoComplete="new-password"
        value={value}
        placeholder={stored ? t.payments.keepStored : ""}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}
