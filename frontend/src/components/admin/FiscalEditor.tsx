"use client";

// Fiscalisation: registering each sale with the tax committee.
//
// The restaurant signs its own contract with a virtual cash register from the
// registry and types the credentials here — the same trade as the payment
// providers and the SMS gateway. We never file anything ourselves and this
// screen must never look as though we do.
//
// Four things shape it:
//
//   • **Providers with no adapter yet are still listed**, marked, and can have
//     their credentials saved. An owner scanning for the name they already
//     know and not finding it concludes we do not support their till; "we
//     support it, we are waiting on your contract's documentation" is an
//     answer a salesperson can give and a blank list cannot.
//   • **Saving and enabling are different acts.** The form is filled in over
//     days, between a contract and a visit to the tax office. Enabling is a
//     claim that sales are being registered, and a false one is discovered by
//     an inspector rather than by us — so the server checks it and this screen
//     explains the refusal rather than hiding the switch.
//   • **The VAT rate has to be *given*.** Empty and 0 are different answers,
//     and an unset rate defaulting to zero would declare the restaurant exempt
//     on every receipt while looking perfectly well-formed.
//   • **"Last receipt filed" is the most useful line here**, not the green tick
//     from the check button. The tick says the credentials worked when it was
//     pressed; the timestamp says whether sales are being registered now, and a
//     stored flag goes stale the moment the hour moves past it.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/format";
import type {
  FiscalCredsInput,
  FiscalProvider,
  FiscalProviderInfo,
  FiscalSettings,
  FiscalSettingsInput,
} from "@/lib/types";

const EMPTY_CREDS: FiscalCredsInput = {
  login: "",
  password: "",
  token: "",
  registerId: "",
  baseUrl: "",
};

const EMPTY: FiscalSettingsInput = {
  provider: "",
  enabled: false,
  tin: "",
  vatPercent: null,
  creds: {},
};

/** The drawer for one provider, defaulted.
 *
 *  ⚠️ A provider the owner has only just selected has no stored drawer, and
 *  reading `form.creds[id].login` on it throws while they are typing. */
function drawerOf(form: FiscalSettingsInput, id: string): FiscalCredsInput {
  return form.creds[id] ?? EMPTY_CREDS;
}

export default function FiscalEditor() {
  const t = useAdminT();
  const [form, setForm] = useState<FiscalSettingsInput>(EMPTY);
  const [stored, setStored] = useState<FiscalSettings | null>(null);
  const [providers, setProviders] = useState<FiscalProviderInfo[]>([]);
  // ⚠️ A string, so that "" and "0" stay distinguishable. A number input would
  // collapse "not declared" into "exempt" — the one distinction this field
  // exists to carry.
  const [vat, setVat] = useState("");
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [minting, setMinting] = useState(false);
  // Held only until the page is left. ⚠️ Never re-fetched and never stored: the
  // server keeps it to compare against and will not hand it back, which is what
  // makes rotating it a revocation rather than a second working key.
  const [agentToken, setAgentToken] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  function load(s: FiscalSettings) {
    setStored(s);
    setVat(s.vatPercent == null ? "" : String(s.vatPercent));
    const drawer = (id: string): FiscalCredsInput => ({
      login: s.creds?.[id]?.login ?? "",
      // Secrets are never returned. Empty here means "keep the stored one",
      // which is why they are not pre-filled with a placeholder.
      password: "",
      token: "",
      registerId: s.creds?.[id]?.registerId ?? "",
      baseUrl: s.creds?.[id]?.baseUrl ?? "",
    });
    // ⚠️ Built from what the server sent, not from a list written here. A
    // provider added on the server and forgotten in this file is a provider
    // whose stored credentials never reach the form — the owner sees empty
    // fields above a connection that is working, retypes them, and the second
    // save is the one that breaks it.
    const creds: Record<string, FiscalCredsInput> = {};
    for (const id of Object.keys(s.creds ?? {})) creds[id] = drawer(id);
    setForm({
      provider: s.provider,
      enabled: s.enabled,
      tin: s.tin ?? "",
      vatPercent: s.vatPercent,
      creds,
    });
  }

  useEffect(() => {
    api
      .fiscalSettings()
      .then(load)
      .catch(() => setError(t.common.loadFailed));
    api
      .fiscalProviders()
      .then(setProviders)
      .catch(() => {});
  }, [t]);

  const chosen = form.provider === "" ? null : form.provider;
  const info = providers.find((p) => p.id === form.provider) ?? null;
  const ready = info?.ready ?? false;
  // A register on the restaurant's own network. Everything below that changes
  // shape does so because of this one fact — see the note it renders.
  const local = info?.local ?? false;

  function setCreds(patch: Partial<FiscalCredsInput>) {
    if (!chosen) return;
    setForm((f) => ({
      ...f,
      creds: {
        ...f.creds,
        // ⚠️ Defaulted, because a provider the owner has just picked has no
        // stored drawer yet and spreading `undefined` loses every keystroke
        // after the first.
        [chosen]: { ...(f.creds[chosen] ?? EMPTY_CREDS), ...patch },
      },
    }));
  }

  async function save(next?: Partial<FiscalSettingsInput>) {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const body: FiscalSettingsInput = {
        ...form,
        // ⚠️ Empty stays null. Number("") is 0, and that one coercion would
        // mark the restaurant VAT-exempt on every receipt it ever files.
        vatPercent: vat.trim() === "" ? null : Number(vat),
        ...next,
      };
      load(await api.updateFiscal(body));
      setMessage(t.fiscal.saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  // Checks what is *stored*, not the draft in the form: the server is what
  // files receipts, and testing an unsaved draft would pass on credentials the
  // restaurant does not actually use. Same rule as the SMS test button.
  async function mintToken() {
    setMinting(true);
    setError("");
    try {
      const res = await api.fiscalAgentToken();
      setAgentToken(res.token);
      // Reload so the "last seen" line resets with the token it belonged to —
      // a rotated relay that still looks connected is exactly the wrong thing
      // to show at the moment somebody needs to know the new key arrived.
      load(await api.fiscalSettings());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setMinting(false);
    }
  }

  /** Where the relay should point. Taken from the configured API base so the
   *  owner copies a working command rather than transcribing a hostname —
   *  transcription is where the wrong address creeps in. */
  function apiBase() {
    return process.env.NEXT_PUBLIC_API_URL || "";
  }

  async function ping() {
    setTesting(true);
    setError("");
    setMessage("");
    try {
      const res = await api.pingFiscal();
      if (res.ok) setMessage(res.message);
      else setError(res.message);
      api
        .fiscalSettings()
        .then(load)
        .catch(() => {});
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setTesting(false);
    }
  }

  return (
    <div className="space-y-5">
      <p className="text-sm text-ink-muted">{t.fiscal.intro}</p>

      {/* The state of the thing, before the fields. An owner opening this page
          is usually here because a receipt did not come out. */}
      {stored?.enabled && (
        <div className="rounded-xl border border-line bg-surface p-3 text-sm">
          <div className="font-medium">{t.fiscal.liveTitle}</div>
          {/* ⚠️ The timestamp, not the tick. See the note at the top. */}
          <div className="mt-1 text-ink-soft">
            {stored.lastReceiptAt
              ? `${t.fiscal.lastReceipt}: ${formatDateTime(stored.lastReceiptAt)}`
              : t.fiscal.noReceiptsYet}
          </div>
          {stored.lastErrorAt && (
            <div className="mt-1 text-danger">
              {formatDateTime(stored.lastErrorAt)} — {stored.lastError}
            </div>
          )}
        </div>
      )}

      <label className="block text-sm">
        <span className="font-medium">{t.fiscal.provider}</span>
        <select
          className="input mt-1"
          value={form.provider}
          onChange={(e) =>
            setForm({ ...form, provider: e.target.value as FiscalProvider })
          }
        >
          <option value="">{t.fiscal.none}</option>
          {providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
              {p.ready ? "" : ` — ${t.fiscal.notReadyTag}`}
            </option>
          ))}
        </select>
        {info && (
          <span className="mt-1 block text-xs text-ink-muted">{info.note}</span>
        )}
      </label>

      {/* Named as our gap, not theirs. An owner told "not connected" without a
          reason re-checks credentials that were never wrong. */}
      {chosen && !ready && (
        <div className="rounded-xl border border-line bg-surface p-3 text-sm text-ink-soft">
          {t.fiscal.notReadyNote}
        </div>
      )}

      {/* ⚠️ Said here, on the settings page, because this is where the owner
          decides — and the requirement is a fact about the building rather than
          about a password. A restaurant that learns at the counter that the
          till tablet has to be on the register's own network learns it during
          a shift. */}
      {chosen && local && (
        <div className="rounded-xl border border-line bg-surface p-3 text-sm text-ink-soft">
          {t.fiscal.localNote}
        </div>
      )}

      {chosen && (
        <>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium">{t.fiscal.tin}</span>
              <input
                className="input mt-1"
                inputMode="numeric"
                value={form.tin}
                onChange={(e) => setForm({ ...form, tin: e.target.value })}
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.fiscal.tinHint}
              </span>
            </label>

            <label className="block text-sm">
              <span className="font-medium">{t.fiscal.vat}</span>
              <input
                className="input mt-1"
                inputMode="numeric"
                placeholder={t.fiscal.vatPh}
                value={vat}
                onChange={(e) => setVat(e.target.value)}
              />
              {/* Says both halves out loud, because the difference between an
                  empty box and a zero is invisible and expensive. */}
              <span className="mt-1 block text-xs text-ink-muted">
                {t.fiscal.vatHint}
              </span>
            </label>

            {/* ⚠️ Hidden entirely for a local register, not disabled: it has
                no account. Multikassa's agent authenticates nobody — it is
                protected by being unreachable from outside the building — so
                four empty boxes labelled "login" and "password" would invite an
                owner to invent credentials and then wonder why nothing worked. */}
            {!local && (
              <>
                <label className="block text-sm">
                  <span className="font-medium">{t.fiscal.login}</span>
                  <input
                    className="input mt-1"
                    value={drawerOf(form, chosen).login}
                    onChange={(e) => setCreds({ login: e.target.value })}
                  />
                </label>

                <label className="block text-sm">
                  <span className="font-medium">{t.fiscal.registerId}</span>
                  <input
                    className="input mt-1"
                    value={drawerOf(form, chosen).registerId}
                    onChange={(e) => setCreds({ registerId: e.target.value })}
                  />
                  <span className="mt-1 block text-xs text-ink-muted">
                    {t.fiscal.registerIdHint}
                  </span>
                </label>

                <label className="block text-sm">
                  <span className="font-medium">{t.fiscal.password}</span>
                  <input
                    className="input mt-1"
                    type="password"
                    autoComplete="new-password"
                    placeholder={
                      stored?.creds?.[chosen]?.hasSecret
                        ? t.fiscal.secretSaved
                        : ""
                    }
                    value={drawerOf(form, chosen).password ?? ""}
                    onChange={(e) => setCreds({ password: e.target.value })}
                  />
                  <span className="mt-1 block text-xs text-ink-muted">
                    {t.fiscal.secretHint}
                  </span>
                </label>

                <label className="block text-sm">
                  <span className="font-medium">{t.fiscal.token}</span>
                  <input
                    className="input mt-1"
                    type="password"
                    autoComplete="new-password"
                    placeholder={
                      stored?.creds?.[chosen]?.hasSecret
                        ? t.fiscal.secretSaved
                        : ""
                    }
                    value={drawerOf(form, chosen).token ?? ""}
                    onChange={(e) => setCreds({ token: e.target.value })}
                  />
                  <span className="mt-1 block text-xs text-ink-muted">
                    {t.fiscal.tokenHint}
                  </span>
                </label>
              </>
            )}

            <label className="block text-sm sm:col-span-2">
              <span className="font-medium">
                {local ? t.fiscal.agentUrl : t.fiscal.baseUrl}
              </span>
              <input
                className="input mt-1"
                placeholder={local ? "http://192.168.1.50:9090" : ""}
                value={drawerOf(form, chosen).baseUrl}
                onChange={(e) => setCreds({ baseUrl: e.target.value })}
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {local ? t.fiscal.agentUrlHint : t.fiscal.baseUrlHint}
              </span>
            </label>
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.enabled}
              // Not hidden when the adapter is missing — a control that
              // vanishes teaches an owner the page is broken. It refuses with a
              // reason instead, which is the same thing the server says.
              disabled={!ready}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
            />
            <span className={ready ? "" : "text-ink-muted"}>
              {t.fiscal.enable}
            </span>
          </label>

          {/* ---- The relay ----
              Only for a local register, because it exists for exactly one
              problem: a browser cannot be relied on to reach an address inside
              the restaurant. Offering it beside a cloud provider would be
              offering a second answer to a question that was never asked. */}
          {local && (
            <div className="rounded-xl border border-line bg-surface p-3">
              <p className="text-sm font-medium">{t.fiscal.agentTitle}</p>
              <p className="mt-1 text-xs text-ink-soft">{t.fiscal.agentHint}</p>

              {/* ⚠️ A timestamp, never a green "connected" badge. The relay's
                  failure mode is going quiet — the PC reboots for Windows
                  updates and nothing changes on any screen — so a saved flag
                  would read "connected" a week after it stopped. The same
                  lesson as lastEventAt and lastUpdateAt. */}
              <p className="mt-2 text-xs text-ink-muted">
                {stored?.agentSeenAt
                  ? `${t.fiscal.agentSeen}: ${formatDateTime(stored.agentSeenAt)}`
                  : t.fiscal.agentNever}
              </p>

              <button
                className="btn mt-3"
                disabled={minting}
                onClick={() => mintToken()}
              >
                {stored?.agentSeenAt || agentToken
                  ? t.fiscal.agentRotate
                  : t.fiscal.agentCreate}
              </button>

              {agentToken && (
                <div className="mt-3 rounded-lg border border-line bg-ink/5 p-3">
                  {/* ⚠️ Said before the token, not after: it is shown once and
                      never again, and an owner who closes the page without
                      copying it has to reinstall on the register's PC. */}
                  <p className="text-xs font-medium">{t.fiscal.agentOnce}</p>
                  <code className="mt-2 block break-all font-mono text-xs">
                    {agentToken}
                  </code>
                  <p className="mt-3 text-xs text-ink-muted">
                    {t.fiscal.agentRun}
                  </p>
                  <code className="mt-1 block break-all font-mono text-xs">
                    fiscalagent -server {apiBase()} -token {agentToken}
                  </code>
                </div>
              )}
            </div>
          )}
        </>
      )}

      {message && <p className="text-sm text-success">{message}</p>}
      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="flex flex-wrap gap-2">
        <button
          className="btn btn-primary"
          disabled={saving}
          onClick={() => save()}
        >
          {saving ? t.common.saving : t.common.save}
        </button>
        {/* ⚠️ Not offered for a local register, and not merely disabled: this
            page is very often open on a laptop somewhere else entirely, where
            the check would fail truthfully and mean nothing. The button lives
            on the till screen, which is the only machine whose answer is
            evidence. The line below says where. */}
        {chosen && ready && !local && (
          <button className="btn" disabled={testing} onClick={ping}>
            {testing ? t.fiscal.checking : t.fiscal.check}
          </button>
        )}
      </div>

      {stored?.lastCheckAt && (
        <p className="text-xs text-ink-muted">
          {formatDateTime(stored.lastCheckAt)} —{" "}
          {stored.lastCheck || (stored.lastCheckOk ? "ok" : "")}
        </p>
      )}
    </div>
  );
}
