"use client";

// Connecting the restaurant's own Meta ad account.
//
// ⚠️ **Five facts, drawn as five.** A working connection is a business
// portfolio, an ad account, a Page, the Instagram account beside it and a
// pixel. Meta holds all five, the owner can rarely name any of them, and a
// campaign created with one missing fails at a different step each time — so
// the checklist shows what is still missing rather than "not connected".
//
// ⚠️ **Every tick is a fact read back from Meta**, never a screen the owner
// has visited. A progress bar counting our own steps would sit full over a
// connection that cannot run an advert, which is the one lie this screen
// cannot afford (docs/reklama-reja.md §6).
//
// ⚠️ **Disconnecting does not stop a campaign**, and the button says so. Meta
// keeps spending the owner's card whatever we forget; pretending otherwise
// would be the worst possible lie at exactly the moment somebody is trying to
// make spending stop.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import type { AdminDict } from "@/lib/i18n/admin";
import type { AdsAssets, AdsState } from "@/lib/types";

/** Where this panel wants the owner to land once Meta is done with them.
 *
 *  ⚠️ **Not the address Meta redirects to.** Meta returns to the platform's
 *  single whitelisted URI and the platform forwards here — one entry in Meta's
 *  settings for every restaurant, rather than one per customer that somebody
 *  has to remember to add. */
function returnTo(): string {
  return `${window.location.origin}/admin/ads`;
}

export default function AdsConnect({
  t,
  state,
  reload,
}: {
  t: AdminDict;
  state: AdsState;
  reload: () => void;
}) {
  const [assets, setAssets] = useState<AdsAssets | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const s = state.settings;

  const loadAssets = useCallback(async () => {
    if (!state.connected) return;
    try {
      setAssets(await api.adsAssets());
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.connect.assetsFailed);
    }
  }, [state.connected, t.ads.connect.assetsFailed]);

  useEffect(() => {
    void loadAssets();
  }, [loadAssets]);

  // ---- Coming back from Meta ----
  //
  // ⚠️ **The code is taken out of the address before anything else happens.**
  // It is single-use and short-lived; left in the bar it is re-sent by every
  // refresh, and the second attempt fails with a message about an expired code
  // that describes nothing the owner did.
  useEffect(() => {
    const url = new URL(window.location.href);
    const code = url.searchParams.get("code");
    const denied = url.searchParams.get("error");
    if (!code && !denied) return;
    url.searchParams.delete("code");
    url.searchParams.delete("error");
    url.searchParams.delete("error_reason");
    url.searchParams.delete("error_description");
    url.searchParams.delete("state");
    window.history.replaceState({}, "", url.toString());
    if (denied) {
      setError(t.ads.connect.denied);
      return;
    }
    setBusy(true);
    api
      .adsConnect(code as string)
      .then(() => reload())
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.ads.connect.failed),
      )
      .finally(() => setBusy(false));
  }, [reload, t.ads.connect.denied, t.ads.connect.failed]);

  async function start() {
    setError("");
    try {
      const app = await api.adsApp(returnTo());
      if (
        !app.configured ||
        !app.appId ||
        !app.configId ||
        !app.redirectUri ||
        !app.state
      ) {
        setError(t.ads.connect.notConfigured);
        return;
      }
      const q = new URLSearchParams({
        client_id: app.appId,
        // ⚠️ `config_id` replaces `scope` entirely — which permissions are
        // asked for is decided in Meta's app dashboard, not here.
        config_id: app.configId,
        response_type: "code",
        override_default_response_type: "true",
        redirect_uri: app.redirectUri,
        // ⚠️ Booked by the platform against this restaurant before the dialog
        // opens: it is what lets the single redirect address find its way back
        // to the right panel, and it is checked rather than parsed.
        state: app.state,
      });
      window.location.href = `https://www.facebook.com/${
        app.version ?? "v26.0"
      }/dialog/oauth?${q.toString()}`;
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.connect.failed);
    }
  }

  async function choose(body: Parameters<typeof api.adsChoose>[0]) {
    setBusy(true);
    setError("");
    try {
      await api.adsChoose(body);
      reload();
      await loadAssets();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.connect.failed);
    } finally {
      setBusy(false);
    }
  }

  async function disconnect() {
    setBusy(true);
    try {
      await api.adsDisconnect();
      setAssets(null);
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.connect.failed);
    } finally {
      setBusy(false);
    }
  }

  const stepLabels = t.ads.connect.steps as Record<string, string>;

  return (
    <div className="card space-y-4 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="font-semibold">{t.ads.connect.title}</p>
          <p className="mt-1 max-w-2xl text-sm text-ink-soft">
            {t.ads.connect.lead}
          </p>
        </div>
        <button
          type="button"
          onClick={start}
          disabled={busy}
          className="btn btn-primary disabled:opacity-40"
        >
          {state.connected ? t.ads.connect.again : t.ads.connect.button}
        </button>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {/* ⚠️ A revoked token is the one state that must never be quiet: Meta
          does not stop the campaigns when access is removed, so the money
          keeps moving while we go blind. */}
      {s.status === "revoked" && (
        <p className="rounded-xl bg-danger/10 p-3 text-sm text-danger">
          {t.ads.connect.revoked}
        </p>
      )}

      <ul className="space-y-1.5">
        {state.steps.map((step) => (
          <li key={step.key} className="flex items-baseline gap-2 text-sm">
            <span
              aria-hidden
              className={`inline-block h-2 w-2 shrink-0 rounded-full ${
                step.done ? "bg-brand" : "bg-ink/20"
              }`}
            />
            <span className={step.done ? "" : "text-ink-muted"}>
              {stepLabels[step.key] ?? step.key}
            </span>
            {step.name && (
              <span className="text-xs text-ink-muted">· {step.name}</span>
            )}
          </li>
        ))}
      </ul>

      {assets && (
        <div className="grid gap-3 sm:grid-cols-2">
          <Picker
            label={t.ads.connect.business}
            value={s.businessId ?? ""}
            disabled={busy}
            options={assets.businesses.map((b) => ({
              id: b.id,
              label: b.name,
            }))}
            none={t.ads.connect.none}
            onPick={(id) => choose({ businessId: id })}
          />
          <Picker
            label={t.ads.connect.account}
            value={s.adAccountId ?? ""}
            disabled={busy}
            options={assets.accounts.map((a) => ({
              id: a.id,
              // ⚠️ The currency is in the label because it is the one fact that
              // decides what every budget on the next screen means.
              label: `${a.name} · ${a.currency}${
                a.account_status === 1 ? "" : ` · ${t.ads.connect.inactive}`
              }`,
            }))}
            none={t.ads.connect.none}
            onPick={(id) => choose({ adAccountId: id })}
          />
          <Picker
            label={t.ads.connect.page}
            value={s.pageId ?? ""}
            disabled={busy}
            options={assets.pages.map((p) => ({ id: p.id, label: p.name }))}
            none={t.ads.connect.none}
            onPick={(id) => choose({ pageId: id })}
          />
          <Picker
            label={t.ads.connect.pixel}
            value={s.pixelId ?? ""}
            disabled={busy}
            options={assets.pixels.map((p) => ({ id: p.id, label: p.name }))}
            none={t.ads.connect.none}
            onPick={(id) => choose({ pixelId: id })}
          />
        </div>
      )}

      {/* Said where the pixel is chosen, because this is the sentence that
          explains why it is worth the extra step. */}
      {state.connected && !s.pixelId && (
        <p className="max-w-2xl text-xs text-ink-muted">
          {t.ads.connect.pixelNote}
        </p>
      )}
      {s.currency && s.currency !== "UZS" && (
        <p className="max-w-2xl text-xs text-ink-muted">
          {t.ads.connect.currencyNote.replace("{currency}", s.currency)}
        </p>
      )}

      {state.connected && (
        <div className="border-t border-ink/10 pt-3">
          <button
            type="button"
            onClick={disconnect}
            disabled={busy}
            className="btn btn-ghost text-danger disabled:opacity-40"
          >
            {t.ads.connect.disconnect}
          </button>
          <p className="mt-1 max-w-2xl text-xs text-ink-muted">
            {t.ads.connect.disconnectNote}
          </p>
          {s.lastCheckAt && (
            <p className="mt-1 text-xs text-ink-muted">
              {t.ads.connect.checked}:{" "}
              {new Date(s.lastCheckAt).toLocaleString()}
            </p>
          )}
        </div>
      )}
    </div>
  );
}

function Picker({
  label,
  value,
  options,
  none,
  disabled,
  onPick,
}: {
  label: string;
  value: string;
  options: { id: string; label: string }[];
  none: string;
  disabled: boolean;
  onPick: (id: string) => void;
}) {
  return (
    <label className="block text-sm">
      <span className="text-ink-soft">{label}</span>
      <select
        className="input mt-1 w-full"
        value={value}
        disabled={disabled}
        onChange={(e) => e.target.value && onPick(e.target.value)}
      >
        <option value="">{none}</option>
        {options.map((o) => (
          <option key={o.id} value={o.id}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  );
}
