"use client";

// The standing instruction: what happens while nobody is watching.
//
// ⚠️ **Every field here is a ceiling, never a target.** The rules can stop a
// campaign and can move a budget downwards; the only upward move is inside the
// daily ceiling typed on this screen. That asymmetry is the whole safety
// property — the worst the engine can do is spend exactly what was authorised.
//
// ⚠️ **Off by default, and it stays off until a ceiling is typed.** A rule with
// no number behind it is a machine acting on its own judgement with somebody
// else's card.

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import type { AdminDict } from "@/lib/i18n/admin";
import type { AdsSettingsView } from "@/lib/types";

export default function AdsRules({
  t,
  settings,
  reload,
}: {
  t: AdminDict;
  settings: AdsSettingsView;
  reload: () => void;
}) {
  const unit = settings.unit ?? 100;
  const asUnits = (minor?: number) =>
    minor ? String(unit <= 1 ? minor : minor / unit) : "";

  const [on, setOn] = useState(settings.rules.on);
  const [tune, setTune] = useState(Boolean(settings.rules.tune));
  const [maxDaily, setMaxDaily] = useState(asUnits(settings.rules.maxDailyMinor));
  const [noResult, setNoResult] = useState(asUnits(settings.rules.noResultMinor));
  const [maxCost, setMaxCost] = useState(asUnits(settings.rules.maxCostMinor));
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  const toMinor = (v: string) => Math.round((Number(v) || 0) * unit);

  async function save() {
    setBusy(true);
    setError("");
    try {
      await api.adsRules({
        on,
        tune,
        maxDailyMinor: toMinor(maxDaily),
        noResultMinor: toMinor(noResult),
        maxCostMinor: toMinor(maxCost),
      });
      setSaved(true);
      window.setTimeout(() => setSaved(false), 2000);
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.rules.failed);
    } finally {
      setBusy(false);
    }
  }

  const cur = settings.currency ?? "";
  // ⚠️ **Closed, and below the figures.** These are ceilings for campaigns that
  // are already running; asked of somebody who has never run one they read as
  // required setup — three money fields in a foreign currency, in front of a
  // feature they have not used yet.
  return (
    <details className="rounded-xl border border-ink/10 p-3 open:pb-4">
      <summary className="cursor-pointer select-none text-sm font-semibold">
        {t.ads.rules.title}
      </summary>
      <div className="mt-2 space-y-3">
      <p className="max-w-2xl text-sm text-ink-soft">{t.ads.rules.lead}</p>

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={on}
          onChange={(e) => setOn(e.target.checked)}
        />
        <span>{t.ads.rules.on}</span>
      </label>

      <div className="grid gap-3 sm:grid-cols-3">
        <Field
          label={`${t.ads.rules.maxDaily} (${cur})`}
          value={maxDaily}
          onChange={setMaxDaily}
        />
        <Field
          label={`${t.ads.rules.noResult} (${cur})`}
          value={noResult}
          onChange={setNoResult}
        />
        <Field
          label={`${t.ads.rules.maxCost} (${cur})`}
          value={maxCost}
          onChange={setMaxCost}
        />
      </div>

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={tune}
          onChange={(e) => setTune(e.target.checked)}
        />
        <span>{t.ads.rules.tune}</span>
      </label>

      <p className="max-w-2xl text-xs text-ink-muted">{t.ads.rules.note}</p>
      {error && <p className="text-sm text-danger">{error}</p>}

      <button
        type="button"
        onClick={save}
        disabled={busy}
        className="btn btn-dark disabled:opacity-40"
      >
        {saved ? t.ads.rules.saved : t.ads.rules.save}
      </button>
      </div>
    </details>
  );
}

function Field({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <label className="block text-sm">
      <span className="text-ink-soft">{label}</span>
      <input
        className="input mt-1 w-full"
        inputMode="decimal"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  );
}
