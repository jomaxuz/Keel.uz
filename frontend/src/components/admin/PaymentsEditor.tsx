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
import type {
  InStoreProvider,
  PaymentSettings,
  PaymentSettingsInput,
} from "@/lib/types";

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
  aggregators: [],
  tillMethods: [],
};

/** The marketplaces offered before anybody types one.
 *
 *  ⚠️ **Known by id, not free text.** Sales are attributed to a rail by this
 *  id, and a marketplace spelled two ways is two rails with half a balance
 *  each — which reads, on the payouts screen, exactly like an aggregator
 *  underpaying. */
const KNOWN_AGGREGATORS = [
  { id: "yandex_eats", name: "Yandex Eats" },
  { id: "uzum_tezkor", name: "Uzum Tezkor" },
];

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
          // ⚠️ Seeded from what the server sent, every rail, including the ones
          // with no adapter: a form that only carried the rails it draws today
          // would post a map missing the rest, and the server's "a rail the
          // panel did not mention keeps its drawer" rule would be the only
          // thing standing between an owner and lost credentials. Belt and
          // braces, on the half where the mistake is invisible.
          inStore: { rails: railsOf(s.inStore?.providers ?? []) },
          // ⚠️ Seeded with every known marketplace, switched on or not, so the
          // form always has both rows to draw — and the ones already stored
          // keep their own name and rate.
          aggregators: KNOWN_AGGREGATORS.map((k) => {
            const saved = (s.aggregators ?? []).find((a) => a.id === k.id);
            return saved ?? { ...k, enabled: false };
          }),
          tillMethods: s.tillMethods ?? [],
        });
      })
      .catch(() => setError(t.common.loadFailed));
  }, [t]);

  async function save() {
    const buttons = form.tillMethods ?? [];
    // ⚠️ Said here, on the rows, rather than as a 400 after the press: a till
    // with no button and a custom button with no name are both refused by the
    // server, and a refusal with the form scrolled away is a mystery.
    if (buttons.length > 0 && !buttons.some((b) => b.enabled)) {
      setError(t.payments.tillMethodsNone);
      return;
    }
    if (buttons.some((b) => !isDefaultButton(b.id) && !b.name.trim())) {
      setError(t.payments.tillMethodNameRequired);
      return;
    }
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
        inStore: f.inStore && {
          rails: Object.fromEntries(
            Object.entries(f.inStore.rails).map(([id, r]) => [
              id,
              { ...r, secretKey: "" },
            ]),
          ),
        },
      }));
      // ⚠️ The buttons come back with the ids the server gave new rows. Keeping
      // the form's own copy would send those rows again as new ones on the
      // next save — two "Humo" buttons, then three.
      setForm((f) => ({ ...f, tillMethods: saved.tillMethods ?? f.tillMethods }));
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

      {/* ---- The counter rails ----

          ⚠️ **A different question from everything above, and it is worth
          saying on screen.** The four providers above are how somebody pays
          from a sofa; these are how somebody pays while standing at the
          counter, and a restaurant can perfectly well want one and not the
          other. Kept in the same page because it is the same set of bank
          contracts, and separated by a heading because it is not the same
          feature. */}
      {(stored?.inStore?.providers?.length ?? 0) > 0 && (
        <div className="space-y-3">
          <div>
            <h3 className="font-display text-lg font-bold">
              {t.payments.inStoreTitle}
            </h3>
            <p className="mt-1 text-sm text-ink-muted">
              {t.payments.inStoreIntro}
            </p>
          </div>
          {stored!.inStore.providers.map((p) => (
            <Rail
              key={p.id}
              provider={p}
              value={
                form.inStore?.rails[p.id] ?? {
                  enabled: false,
                  serviceId: "",
                  userId: "",
                  baseUrl: "",
                }
              }
              onChange={(next) =>
                setForm((f) => ({
                  ...f,
                  inStore: {
                    rails: { ...(f.inStore?.rails ?? {}), [p.id]: next },
                  },
                }))
              }
              t={t}
            />
          ))}
        </div>
      )}

      {/* ---- The marketplaces ----

          ⚠️ **A payment method, not a delivery service**, and the page says so.
          The delivery-providers screen answers "who carries the food"; this
          answers "who holds the money". Yandex can be both at once, and one
          record for both would give a restaurant that merely hires the courier
          fleet a balance it does not have.

          ⚠️ **Switched on here because the till only offers what is on.** Two
          extra payment buttons on every counter in the country, for the
          restaurants that have never heard of Uzum Tezkor, is how a payment
          screen becomes something cashiers guess at. */}
      {/* ---- The till's own buttons ----

          ⚠️ **A name over a kind.** The owner names the buttons the cashier
          presses — "Humo terminal", "Beznal (hisob raqam)" — and picks which of
          the three kinds the money is: the drawer counts cash, the reports split
          card from transfer, and nothing downstream has to learn the name. */}
      <div className="space-y-3">
        <div>
          <h3 className="font-display text-lg font-bold">
            {t.payments.tillMethodsTitle}
          </h3>
          <p className="mt-1 text-sm text-ink-muted">{t.payments.tillMethodsIntro}</p>
        </div>
        {(form.tillMethods ?? []).map((m, i) => (
          <div
            key={m.id || `new-${i}`}
            className="flex flex-wrap items-center gap-3 rounded-2xl border border-line p-3"
          >
            <input
              type="checkbox"
              aria-label={t.payments.tillMethodEnabled}
              checked={m.enabled}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  tillMethods: (f.tillMethods ?? []).map((x, j) =>
                    j === i ? { ...x, enabled: e.target.checked } : x,
                  ),
                }))
              }
            />
            <input
              className="input min-w-[10rem] flex-1"
              value={m.name}
              maxLength={40}
              placeholder={
                isDefaultButton(m.id)
                  ? defaultButtonName(m.kind, t.till)
                  : t.payments.tillMethodNamePlaceholder
              }
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  tillMethods: (f.tillMethods ?? []).map((x, j) =>
                    j === i ? { ...x, name: e.target.value } : x,
                  ),
                }))
              }
            />
            <select
              className="input w-auto"
              value={m.kind}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  tillMethods: (f.tillMethods ?? []).map((x, j) =>
                    j === i
                      ? { ...x, kind: e.target.value as "cash" | "card" | "transfer" }
                      : x,
                  ),
                }))
              }
            >
              <option value="cash">{t.payments.tillKindCash}</option>
              <option value="card">{t.payments.tillKindCard}</option>
              <option value="transfer">{t.payments.tillKindTransfer}</option>
            </select>
            {!isDefaultButton(m.id) && (
              <button
                type="button"
                className="text-sm text-danger"
                onClick={() =>
                  setForm((f) => ({
                    ...f,
                    tillMethods: (f.tillMethods ?? []).filter((_, j) => j !== i),
                  }))
                }
              >
                {t.payments.tillMethodRemove}
              </button>
            )}
          </div>
        ))}
        <button
          type="button"
          className="btn"
          onClick={() =>
            setForm((f) => ({
              ...f,
              tillMethods: [
                ...(f.tillMethods ?? []),
                { id: "", name: "", kind: "card", enabled: true },
              ],
            }))
          }
        >
          {t.payments.tillMethodAdd}
        </button>
        <p className="text-xs text-ink-muted">{t.payments.tillMethodsHint}</p>
      </div>

      <div className="space-y-3">
        <div>
          <h3 className="font-display text-lg font-bold">
            {t.payments.aggregatorsTitle}
          </h3>
          <p className="mt-1 text-sm text-ink-muted">
            {t.payments.aggregatorsIntro}
          </p>
        </div>
        {(form.aggregators ?? []).map((a, i) => (
          <div
            key={a.id}
            className="flex flex-wrap items-center gap-3 rounded-2xl border border-line p-3"
          >
            <label className="flex items-center gap-2 text-sm font-medium">
              <input
                type="checkbox"
                checked={a.enabled}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    aggregators: (f.aggregators ?? []).map((x, j) =>
                      j === i ? { ...x, enabled: e.target.checked } : x,
                    ),
                  }))
                }
              />
              {a.name}
            </label>
            <label className="ml-auto flex items-center gap-2 text-sm text-ink-muted">
              {t.payments.aggregatorCommission}
              <input
                className="input w-20"
                inputMode="decimal"
                value={a.commissionPercent ?? ""}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    aggregators: (f.aggregators ?? []).map((x, j) =>
                      j === i
                        ? {
                            ...x,
                            commissionPercent:
                              Number(e.target.value.replace(/[^\d.]/g, "")) || 0,
                          }
                        : x,
                    ),
                  }))
                }
              />
              %
            </label>
          </div>
        ))}
        {/* ⚠️ Said out loud, because the number above looks like it does
            something: the rate only prefills the payout form. Money is only
            ever what the statement says. */}
        <p className="text-xs text-ink-muted">{t.payments.aggregatorRateHint}</p>
      </div>

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

/** The rails as the form holds them, seeded from what the server sent. */
function railsOf(providers: InStoreProvider[]) {
  return Object.fromEntries(
    providers.map((p) => [
      p.id,
      {
        enabled: p.enabled,
        serviceId: p.serviceId,
        userId: p.userId,
        baseUrl: p.baseUrl,
      },
    ]),
  );
}

/** One counter rail.
 *
 *  ⚠️ **A rail with no adapter can be filled in and cannot be switched on.**
 *  The row is drawn anyway — an owner who has been sold Payme GO, or a Rahmat
 *  terminal, scans this list for the name and needs to read *why* rather than
 *  conclude we have never heard of it. The toggle is disabled here and the
 *  server refuses it as well, because a screen is not a rule. */
function Rail({
  provider,
  value,
  onChange,
  t,
}: {
  provider: InStoreProvider;
  value: {
    enabled: boolean;
    serviceId: string;
    userId: string;
    secretKey?: string;
    baseUrl: string;
  };
  onChange: (v: {
    enabled: boolean;
    serviceId: string;
    userId: string;
    secretKey?: string;
    baseUrl: string;
  }) => void;
  t: ReturnType<typeof useAdminT>;
}) {
  const needs = (box: string) => provider.needs.includes(box);
  return (
    <div className="rounded-2xl border border-line p-4">
      <label className="flex items-center gap-3">
        <input
          type="checkbox"
          checked={value.enabled && provider.ready}
          disabled={!provider.ready}
          onChange={(e) => onChange({ ...value, enabled: e.target.checked })}
        />
        <span className="font-display text-lg font-bold">{provider.name}</span>
      </label>
      <p className="mt-1 text-xs text-ink-muted">{provider.note}</p>
      {!provider.ready && (
        <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
          {t.payments.inStoreNotReady}
        </p>
      )}

      {/* ⚠️ Shown for a rail with no adapter too, and deliberately: the usual
          case is an owner filling this in from a contract days before the
          integration exists, and a drawer that only opens once we are ready
          makes them come back and re-find the page. */}
      <div className="mt-4 space-y-3">
        {needs("serviceId") && (
          <Field
            label={t.payments.inStoreServiceId}
            value={value.serviceId}
            onChange={(v) => onChange({ ...value, serviceId: v })}
          />
        )}
        {needs("userId") && (
          <Field
            label={t.payments.inStoreUserId}
            hint={t.payments.inStoreUserIdHint}
            value={value.userId}
            onChange={(v) => onChange({ ...value, userId: v })}
          />
        )}
        {needs("secretKey") && (
          <Secret
            label={t.payments.inStoreSecret}
            stored={provider.hasSecretKey}
            value={value.secretKey ?? ""}
            onChange={(v) => onChange({ ...value, secretKey: v })}
            t={t}
          />
        )}
        {needs("baseUrl") && (
          <Field
            label={t.payments.inStoreBaseUrl}
            value={value.baseUrl}
            onChange={(v) => onChange({ ...value, baseUrl: v })}
          />
        )}
      </div>
    </div>
  );
}

/** The three buttons a till always had. Their names may be left empty, and
 *  they cannot be removed — only switched off. */
function isDefaultButton(id: string): boolean {
  return id === "cash" || id === "card" || id === "transfer";
}

function defaultButtonName(
  kind: "cash" | "card" | "transfer",
  t: { methodCash: string; methodCard: string; methodTransfer: string },
): string {
  return kind === "cash" ? t.methodCash : kind === "card" ? t.methodCard : t.methodTransfer;
}
