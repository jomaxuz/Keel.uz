"use client";

// The SMS gateway that sends login codes.
//
// Chosen and paid for by the restaurant itself: it signs its own contract with
// Eskiz, Play Mobile, getsms.uz or OneSignal, gets its own moderated sender
// name, and types the credentials here. Three things shape this screen:
//
//   • **"Send a test SMS" is the point of the page, not a nicety.** Credentials
//     that look right still hide two failures — a sender name that was never
//     moderated, and an account with no money on it. Both surface at the same
//     moment: the first guest trying to log in, which the restaurant reads as
//     "the site is broken". One button, one message, before that happens.
//   • **demo mode is stated loudly.** A half-filled provider silently falls
//     back to demo on the server, so the panel shows what is *actually*
//     sending rather than what was ticked.
//   • **a saved password is never shown again** — the field says "saved" and
//     stays empty; typing replaces it, leaving it alone keeps it. Same rule as
//     the payment keys, and the same trap if it were not: an owner fixing a
//     typo in their login would blank the password and switch SMS off.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/format";
import type { SMSProvider, SMSSettings, SMSSettingsInput } from "@/lib/types";

const EMPTY: SMSSettingsInput = {
  provider: "demo",
  from: "",
  eskiz: { email: "", baseUrl: "" },
  playmobile: { url: "", login: "" },
  getsms: { url: "", login: "", nickname: "" },
  onesignal: { appId: "", from: "", baseUrl: "" },
};

export default function SmsEditor() {
  const t = useAdminT();
  const [form, setForm] = useState<SMSSettingsInput>(EMPTY);
  const [stored, setStored] = useState<SMSSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [testPhone, setTestPhone] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  function load(s: SMSSettings) {
    setStored(s);
    setForm({
      provider: s.provider,
      from: s.from,
      eskiz: { email: s.eskiz.email, baseUrl: s.eskiz.baseUrl },
      playmobile: { url: s.playmobile.url, login: s.playmobile.login },
      getsms: {
        url: s.getsms.url,
        login: s.getsms.login,
        nickname: s.getsms.nickname,
      },
      onesignal: {
        appId: s.onesignal.appId,
        from: s.onesignal.from,
        baseUrl: s.onesignal.baseUrl,
      },
    });
  }

  useEffect(() => {
    api
      .adminSMSSettings()
      .then(load)
      .catch(() => setError(t.common.loadFailed));
  }, [t]);

  async function save() {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      load(await api.updateSMSSettings(form));
      setMessage(t.sms.saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  // Tests the *stored* settings, not the ones in the form — the server is what
  // sends, and testing an unsaved draft would pass on credentials the site
  // does not actually use.
  async function test() {
    setTesting(true);
    setError("");
    setMessage("");
    try {
      const res = await api.testSMS(testPhone);
      if (res.ok) setMessage(res.message);
      else setError(res.message);
      api.adminSMSSettings().then(load).catch(() => {});
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setTesting(false);
    }
  }

  const providers: SMSProvider[] = stored?.providers?.length
    ? stored.providers
    : ["eskiz", "playmobile", "getsms", "onesignal", "demo"];

  // What is actually sending, which is not always what was chosen.
  const inDemo = stored?.demo ?? true;
  const fellBack = inDemo && form.provider !== "demo";

  return (
    <div className="space-y-5">
      <p className="text-sm text-ink-muted">{t.sms.intro}</p>

      {/* The state of the thing, before any of the fields. An owner who opens
          this page is nearly always here because SMS did not arrive. */}
      {inDemo && (
        <div className="rounded-xl border border-amber-500/40 bg-amber-500/10 p-3">
          <p className="text-sm font-semibold">{t.sms.demoTitle}</p>
          <p className="mt-1 text-xs text-ink-muted">
            {fellBack ? t.sms.fallbackWarning : t.sms.demoHint}
          </p>
          {stored?.missing && (
            <p className="mt-1 text-xs font-medium">
              {t.sms.missing(stored.missing)}
            </p>
          )}
        </div>
      )}
      {stored?.fromEnv && !inDemo && (
        <p className="text-xs text-ink-muted">{t.sms.fromEnvHint}</p>
      )}

      <div>
        <label className="text-sm font-medium">{t.sms.provider}</label>
        <select
          className="input mt-1"
          value={form.provider}
          onChange={(e) =>
            setForm({ ...form, provider: e.target.value as SMSProvider })
          }
        >
          {providers.map((p) => (
            <option key={p} value={p}>
              {t.sms.providerNames[p] ?? p}
            </option>
          ))}
        </select>
      </div>

      {form.provider !== "demo" && (
        <Field
          label={t.sms.from}
          hint={t.sms.fromHint}
          value={form.from}
          onChange={(from) => setForm({ ...form, from })}
        />
      )}

      {form.provider === "eskiz" && (
        <Group>
          <Field
            label={t.sms.eskizEmail}
            value={form.eskiz.email}
            onChange={(email) =>
              setForm({ ...form, eskiz: { ...form.eskiz, email } })
            }
          />
          <Secret
            label={t.sms.eskizPassword}
            stored={stored?.eskiz.hasPassword}
            value={form.eskiz.password ?? ""}
            onChange={(password) =>
              setForm({ ...form, eskiz: { ...form.eskiz, password } })
            }
            t={t}
          />
          <Field
            label={t.sms.eskizBaseUrl}
            value={form.eskiz.baseUrl}
            onChange={(baseUrl) =>
              setForm({ ...form, eskiz: { ...form.eskiz, baseUrl } })
            }
          />
        </Group>
      )}

      {form.provider === "playmobile" && (
        <Group>
          <Field
            label={t.sms.playmobileLogin}
            value={form.playmobile.login}
            onChange={(login) =>
              setForm({ ...form, playmobile: { ...form.playmobile, login } })
            }
          />
          <Secret
            label={t.sms.playmobilePassword}
            stored={stored?.playmobile.hasPassword}
            value={form.playmobile.password ?? ""}
            onChange={(password) =>
              setForm({ ...form, playmobile: { ...form.playmobile, password } })
            }
            t={t}
          />
          <Field
            label={t.sms.playmobileUrl}
            value={form.playmobile.url}
            onChange={(url) =>
              setForm({ ...form, playmobile: { ...form.playmobile, url } })
            }
          />
        </Group>
      )}

      {form.provider === "getsms" && (
        <Group>
          <Field
            label={t.sms.getsmsLogin}
            value={form.getsms.login}
            onChange={(login) =>
              setForm({ ...form, getsms: { ...form.getsms, login } })
            }
          />
          <Secret
            label={t.sms.getsmsPassword}
            stored={stored?.getsms.hasPassword}
            value={form.getsms.password ?? ""}
            onChange={(password) =>
              setForm({ ...form, getsms: { ...form.getsms, password } })
            }
            t={t}
          />
          <Field
            label={t.sms.getsmsNickname}
            hint={t.sms.getsmsNicknameHint}
            value={form.getsms.nickname}
            onChange={(nickname) =>
              setForm({ ...form, getsms: { ...form.getsms, nickname } })
            }
          />
          <Field
            label={t.sms.getsmsUrl}
            value={form.getsms.url}
            onChange={(url) =>
              setForm({ ...form, getsms: { ...form.getsms, url } })
            }
          />
        </Group>
      )}

      {form.provider === "onesignal" && (
        <Group>
          <p className="text-xs text-ink-muted">{t.sms.onesignalNote}</p>
          <Field
            label={t.sms.onesignalAppId}
            value={form.onesignal.appId}
            onChange={(appId) =>
              setForm({ ...form, onesignal: { ...form.onesignal, appId } })
            }
          />
          <Secret
            label={t.sms.onesignalApiKey}
            stored={stored?.onesignal.hasApiKey}
            value={form.onesignal.apiKey ?? ""}
            onChange={(apiKey) =>
              setForm({ ...form, onesignal: { ...form.onesignal, apiKey } })
            }
            t={t}
          />
          <Field
            label={t.sms.onesignalFrom}
            value={form.onesignal.from}
            onChange={(from) =>
              setForm({ ...form, onesignal: { ...form.onesignal, from } })
            }
          />
          <Field
            label={t.sms.onesignalBaseUrl}
            value={form.onesignal.baseUrl}
            onChange={(baseUrl) =>
              setForm({ ...form, onesignal: { ...form.onesignal, baseUrl } })
            }
          />
        </Group>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <button className="btn btn-primary" onClick={save} disabled={saving}>
          {saving ? t.common.saving : t.common.save}
        </button>
        {message && (
          <span className="text-sm text-emerald-700 dark:text-emerald-300">
            {message}
          </span>
        )}
        {error && <span className="text-sm text-red-600">{error}</span>}
      </div>

      {/* ---- Proving it works ---- */}
      <div className="rounded-xl bg-ink/5 p-3">
        <p className="text-xs text-ink-muted">{t.sms.testWhy}</p>
        <div className="mt-2 sm:flex sm:items-end sm:gap-2">
          <div className="flex-1">
            <label className="text-sm font-medium">{t.sms.testPhone}</label>
            <input
              className="input mt-1"
              value={testPhone}
              placeholder="998 90 123 45 67"
              onChange={(e) => setTestPhone(e.target.value)}
            />
          </div>
          <button
            type="button"
            className="btn mt-2 shrink-0 sm:mt-0"
            onClick={test}
            disabled={testing}
          >
            {testing ? t.sms.testing : t.sms.test}
          </button>
        </div>
        <p className="mt-1 text-xs text-ink-muted">{t.sms.testPhoneHint}</p>

        <p className="mt-2 text-xs">
          {stored?.lastTestAt && !stored.lastTestAt.startsWith("0001") ? (
            <span
              className={
                stored.lastTestOk
                  ? "text-emerald-700 dark:text-emerald-300"
                  : "text-red-600"
              }
            >
              {t.sms.lastTest(formatDateTime(stored.lastTestAt))} —{" "}
              {stored.lastTest}
            </span>
          ) : (
            <span className="text-ink-muted">{t.sms.lastTestNever}</span>
          )}
        </p>
      </div>
    </div>
  );
}

function Group({ children }: { children: React.ReactNode }) {
  return (
    <div className="space-y-3 rounded-xl border border-line p-3">{children}</div>
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
