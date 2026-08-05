"use client";

// Connecting the branch to the till it already runs.
//
// The shape of this screen follows one fact: an owner filling it in has a page
// of credentials from their POS provider and no way to tell whether they typed
// them correctly. So the **check button is the main control**, not an
// afterthought — it dials the till and answers with the name of the
// organisation it reached ("Maracanda · terminal ishlayapti"), which is the
// only thing that distinguishes configured from configured correctly.
//
// Secrets follow the payment-keys rule: shown as "saved", never rendered, and
// an empty field means keep the stored one.

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/format";
import type { POSProvider, POSSettings, POSSettingsInput } from "@/lib/types";

const EMPTY: POSSettingsInput = {
  provider: "",
  enabled: false,
  autoSend: true,
  iiko: {
    organizationId: "",
    terminalGroup: "",
    orderTypeId: "",
    paymentTypeId: "",
    baseUrl: "",
  },
  syrve: {
    organizationId: "",
    terminalGroup: "",
    orderTypeId: "",
    paymentTypeId: "",
    baseUrl: "",
  },
  poster: { spotId: 0, baseUrl: "" },
  clopos: {
    clientId: "",
    brand: "",
    integratorId: "",
    venueId: 0,
    saleTypeId: 0,
    baseUrl: "",
  },
  rkeeper: { url: "", login: "", station: "", anchor: "" },
};

const PROVIDERS: { id: POSProvider; label: string }[] = [
  { id: "", label: "—" },
  { id: "iiko", label: "iiko" },
  { id: "syrve", label: "Syrve" },
  { id: "poster", label: "Poster" },
  { id: "clopos", label: "Clopos" },
  { id: "rkeeper", label: "r_keeper 7" },
];

export default function POSEditor() {
  const t = useAdminT();
  const [form, setForm] = useState<POSSettingsInput>(EMPTY);
  const [stored, setStored] = useState<POSSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [check, setCheck] = useState<{ ok: boolean; message: string } | null>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api
      .adminPOS()
      .then((s) => {
        setStored(s);
        setForm({
          provider: s.provider,
          enabled: s.enabled,
          autoSend: s.autoSend,
          iiko: {
            organizationId: s.iiko.organizationId,
            terminalGroup: s.iiko.terminalGroup,
            orderTypeId: s.iiko.orderTypeId,
            paymentTypeId: s.iiko.paymentTypeId,
            baseUrl: s.iiko.baseUrl,
          },
          syrve: {
            organizationId: s.syrve.organizationId,
            terminalGroup: s.syrve.terminalGroup,
            orderTypeId: s.syrve.orderTypeId,
            paymentTypeId: s.syrve.paymentTypeId,
            baseUrl: s.syrve.baseUrl,
          },
          poster: { spotId: s.poster.spotId, baseUrl: s.poster.baseUrl },
          clopos: {
            clientId: s.clopos.clientId,
            brand: s.clopos.brand,
            integratorId: s.clopos.integratorId,
            venueId: s.clopos.venueId,
            saleTypeId: s.clopos.saleTypeId,
            baseUrl: s.clopos.baseUrl,
          },
          rkeeper: {
            url: s.rkeeper.url,
            login: s.rkeeper.login,
            station: s.rkeeper.station,
            anchor: s.rkeeper.anchor,
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
      const saved = await api.updatePOS(form);
      setStored(saved);
      // Blank the secrets after saving: they are stored now, and leaving them
      // in the inputs only invites a second submit that re-sends them.
      setForm((f) => ({
        ...f,
        iiko: { ...f.iiko, apiLogin: "" },
        syrve: { ...f.syrve, apiLogin: "" },
        poster: { ...f.poster, token: "" },
        clopos: { ...f.clopos, clientSecret: "" },
        rkeeper: { ...f.rkeeper, password: "", token: "" },
      }));
      setMessage(t.pos.saved);
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
      setCheck(await api.pingPOS());
    } catch (e) {
      setCheck({ ok: false, message: e instanceof Error ? e.message : "" });
    } finally {
      setChecking(false);
    }
  }

  const p = form.provider;

  return (
    <div className="space-y-5">
      <p className="text-sm text-ink-muted">{t.pos.intro}</p>

      <div>
        <label className="text-sm font-medium">{t.pos.provider}</label>
        <select
          className="input mt-1"
          value={p}
          onChange={(e) =>
            setForm({ ...form, provider: e.target.value as POSProvider })
          }
        >
          {PROVIDERS.map((x) => (
            <option key={x.id} value={x.id}>
              {x.label}
            </option>
          ))}
        </select>
      </div>

      {p && (
        <>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
            />
            <span>{t.pos.enabled}</span>
          </label>
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={form.autoSend}
              onChange={(e) => setForm({ ...form, autoSend: e.target.checked })}
            />
            <span>
              {t.pos.autoSend}
              <span className="block text-xs text-ink-muted">
                {t.pos.autoSendHint}
              </span>
            </span>
          </label>
        </>
      )}

      {/* iiko and Syrve are the same product under two names, so they get the
          same five fields. Rendered from one component rather than copied:
          a form duplicated once is a form that drifts. */}
      {(p === "iiko" || p === "syrve") && (
        <IikoFields
          t={t}
          value={p === "syrve" ? form.syrve : form.iiko}
          hasApiLogin={
            p === "syrve" ? stored?.syrve.hasApiLogin : stored?.iiko.hasApiLogin
          }
          onChange={(patch) =>
            setForm((f) =>
              p === "syrve"
                ? { ...f, syrve: { ...f.syrve, ...patch } }
                : { ...f, iiko: { ...f.iiko, ...patch } },
            )
          }
        />
      )}

      {p === "poster" && (
        <div className="space-y-3 rounded-2xl border border-line p-4">
          <Secret
            label={t.pos.posterToken}
            hint={t.pos.posterTokenHint}
            stored={stored?.poster.hasToken}
            value={form.poster.token ?? ""}
            onChange={(v) => setForm({ ...form, poster: { ...form.poster, token: v } })}
            t={t}
          />
          <Field
            label={t.pos.posterSpot}
            hint={t.pos.posterSpotHint}
            value={String(form.poster.spotId || "")}
            onChange={(v) =>
              setForm({
                ...form,
                poster: { ...form.poster, spotId: Number(v.replace(/\D/g, "")) || 0 },
              })
            }
          />
          <p className="rounded-xl bg-amber-500/10 p-3 text-xs text-amber-800 dark:text-amber-200">
            {t.pos.posterNote}
          </p>
        </div>
      )}

      {p === "clopos" && (
        <div className="space-y-3 rounded-2xl border border-line p-4">
          <Field
            label="client_id"
            value={form.clopos.clientId}
            onChange={(v) => setForm({ ...form, clopos: { ...form.clopos, clientId: v } })}
          />
          <Secret
            label="client_secret"
            stored={stored?.clopos.hasSecret}
            value={form.clopos.clientSecret ?? ""}
            onChange={(v) =>
              setForm({ ...form, clopos: { ...form.clopos, clientSecret: v } })
            }
            t={t}
          />
          <Field
            label={t.pos.cloposBrand}
            value={form.clopos.brand}
            onChange={(v) => setForm({ ...form, clopos: { ...form.clopos, brand: v } })}
          />
          <Field
            label="integrator_id"
            hint={t.pos.cloposIntegratorHint}
            value={form.clopos.integratorId}
            onChange={(v) =>
              setForm({ ...form, clopos: { ...form.clopos, integratorId: v } })
            }
          />
          <Field
            label={t.pos.cloposVenue}
            value={String(form.clopos.venueId || "")}
            onChange={(v) =>
              setForm({
                ...form,
                clopos: { ...form.clopos, venueId: Number(v) || 0 },
              })
            }
          />
          <Field
            label={t.pos.cloposSaleType}
            hint={t.pos.cloposSaleTypeHint}
            value={String(form.clopos.saleTypeId || "")}
            onChange={(v) =>
              setForm({
                ...form,
                clopos: { ...form.clopos, saleTypeId: Number(v) || 0 },
              })
            }
          />
        </div>
      )}

      {p === "rkeeper" && (
        <div className="space-y-3 rounded-2xl border border-line p-4">
          {/* The thing an owner has to know before filling anything in. */}
          <div className="rounded-xl border border-amber-500/40 bg-amber-500/10 p-3 text-sm">
            <div className="font-semibold text-amber-700 dark:text-amber-300">
              {t.pos.rkeeperWarnTitle}
            </div>
            <p className="mt-1 text-ink-muted">{t.pos.rkeeperWarn}</p>
          </div>
          <Field
            label={t.pos.rkeeperUrl}
            hint="https://192.168.1.10:9000/rk7api/v0/xmlinterface.xml"
            value={form.rkeeper.url}
            onChange={(v) => setForm({ ...form, rkeeper: { ...form.rkeeper, url: v } })}
          />
          <Field
            label={t.pos.rkeeperLogin}
            hint={t.pos.rkeeperLoginHint}
            value={form.rkeeper.login}
            onChange={(v) => setForm({ ...form, rkeeper: { ...form.rkeeper, login: v } })}
          />
          <Secret
            label={t.payments.password}
            stored={stored?.rkeeper.hasPassword}
            value={form.rkeeper.password ?? ""}
            onChange={(v) =>
              setForm({ ...form, rkeeper: { ...form.rkeeper, password: v } })
            }
            t={t}
          />
          <Field
            label={t.pos.rkeeperStation}
            hint={t.pos.rkeeperStationHint}
            value={form.rkeeper.station}
            onChange={(v) =>
              setForm({ ...form, rkeeper: { ...form.rkeeper, station: v } })
            }
          />
          <Field
            label={t.pos.rkeeperAnchor}
            hint={t.pos.rkeeperLicenceHint}
            value={form.rkeeper.anchor}
            onChange={(v) => setForm({ ...form, rkeeper: { ...form.rkeeper, anchor: v } })}
          />
          <Secret
            label={t.pos.rkeeperToken}
            stored={stored?.rkeeper.hasToken}
            value={form.rkeeper.token ?? ""}
            onChange={(v) => setForm({ ...form, rkeeper: { ...form.rkeeper, token: v } })}
            t={t}
          />
        </div>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <button type="button" onClick={save} disabled={saving} className="btn btn-primary">
          {saving ? t.common.saving : t.common.save}
        </button>
        {p && (
          <button
            type="button"
            onClick={ping}
            disabled={checking}
            className="btn btn-ghost"
          >
            {checking ? t.pos.checking : t.pos.check}
          </button>
        )}
        {p && (
          <Link href="/admin/pos" className="text-sm font-semibold text-brand hover:underline">
            {t.pos.openMapping}
          </Link>
        )}
        {message && (
          <span className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
            {message}
          </span>
        )}
        {error && <span className="text-sm text-brand">{error}</span>}
      </div>

      {/* The answer from the till: the name it reached, or why it could not. */}
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
    </div>
  );
}

/** The credentials iiko and Syrve share. `patch` rather than a full value so
 *  the caller decides which of the two drawers it lands in. */
function IikoFields({
  value,
  hasApiLogin,
  onChange,
  t,
}: {
  value: {
    apiLogin?: string;
    organizationId: string;
    terminalGroup: string;
    orderTypeId: string;
    paymentTypeId: string;
    baseUrl: string;
  };
  hasApiLogin?: boolean;
  onChange: (patch: Partial<{ [K in string]: string }>) => void;
  t: ReturnType<typeof useAdminT>;
}) {
  return (
    <div className="space-y-3 rounded-2xl border border-line p-4">
      <Secret
        label={t.pos.iikoApiLogin}
        hint={t.pos.iikoApiLoginHint}
        stored={hasApiLogin}
        value={value.apiLogin ?? ""}
        onChange={(v) => onChange({ apiLogin: v })}
        t={t}
      />
      <Field
        label={t.pos.iikoOrg}
        value={value.organizationId}
        onChange={(v) => onChange({ organizationId: v })}
      />
      <Field
        label={t.pos.iikoTerminal}
        hint={t.pos.iikoTerminalHint}
        value={value.terminalGroup}
        onChange={(v) => onChange({ terminalGroup: v })}
      />
      <Field
        label={t.pos.iikoOrderType}
        hint={t.pos.optionalHint}
        value={value.orderTypeId}
        onChange={(v) => onChange({ orderTypeId: v })}
      />
      <Field
        label={t.pos.iikoPaymentType}
        hint={t.pos.iikoPaymentTypeHint}
        value={value.paymentTypeId}
        onChange={(v) => onChange({ paymentTypeId: v })}
      />
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

function Secret({
  label,
  hint,
  stored,
  value,
  onChange,
  t,
}: {
  label: string;
  hint?: string;
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
      {hint && <p className="mt-1 text-xs text-ink-muted">{hint}</p>}
    </div>
  );
}
