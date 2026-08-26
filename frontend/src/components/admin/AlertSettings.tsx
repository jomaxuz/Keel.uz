"use client";

// Where the restaurant says what counts as unusual, and where the owner links
// the chat these arrive in.
//
// ⚠️ **The thresholds are the feature, not the switch.** A hundred thousand
// so'm off a bill is a rounding error in one restaurant and a week's profit in
// another. A figure we chose would be wrong in one of them every day, and a
// notification channel that is wrong every day is muted in its first week —
// along with the one message that mattered.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { AlertSettings as Settings } from "@/lib/types";

export default function AlertSettings() {
  const t = useAdminT();
  const [set, setSet] = useState<Settings | null>(null);
  const [linked, setLinked] = useState(false);
  const [hasChannel, setHasChannel] = useState(true);
  const [link, setLink] = useState("");
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);

  const load = () =>
    api
      .adminAlertSettings()
      .then((r) => {
        setSet(r.settings);
        setLinked(r.linked);
        setHasChannel(r.hasChannel);
        setLink(r.link);
      })
      .catch(() => setSet(null));

  useEffect(() => {
    load();
  }, []);

  if (!set) return null;

  const patch = (p: Partial<Settings>) => {
    setSet({ ...set, ...p });
    setSaved(false);
  };

  const money = (
    key: "voidFrom" | "discountFrom" | "cashShortFrom" | "stockShortFrom",
  ) => (
    <label className="block text-sm">
      <span className="font-medium">{t.alerts.fields[key]}</span>
      <input
        type="number"
        min={0}
        step={10000}
        className="input mt-1 w-44"
        value={set[key]}
        onChange={(e) => patch({ [key]: Number(e.target.value) || 0 })}
      />
      <span className="mt-1 block text-xs text-ink-muted">
        {t.alerts.hints[key]}
      </span>
    </label>
  );

  return (
    <div className="card space-y-4 p-4">
      <div>
        <h2 className="font-semibold">{t.alerts.title}</h2>
        <p className="mt-1 text-sm text-ink-soft">{t.alerts.intro}</p>
      </div>

      {/* ⚠️ **Above the switch, because it is the reason the switch will not
          help.** Somebody who set up a Telegram group, sent a test message and
          saw it arrive has every reason to think this is working — and the one
          thing that would tell them otherwise is a sentence right here. */}
      {set.enabled && !hasChannel && (
        <p className="rounded-xl border border-amber-400/60 bg-amber-50 p-3 text-sm dark:bg-amber-950/30">
          {t.alerts.noChannel}
        </p>
      )}

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={set.enabled}
          onChange={(e) => patch({ enabled: e.target.checked })}
        />
        <span>{t.alerts.enable}</span>
      </label>

      {set.enabled && (
        <>
          {/* ⚠️ The chat comes before the thresholds: without it the settings
              below configure a channel that reaches nobody, and an owner who
              tuned four numbers and then heard nothing would reasonably decide
              the feature does not work. */}
          <div className="rounded-xl border border-line p-3">
            <p className="text-sm font-medium">{t.alerts.channel}</p>
            {linked ? (
              <div className="mt-2 flex flex-wrap items-center gap-3">
                <span className="text-sm text-emerald-700 dark:text-emerald-400">
                  {t.alerts.linked}
                </span>
                <button
                  type="button"
                  className="btn btn-ghost text-sm"
                  onClick={async () => {
                    await api.adminUnlinkAlerts().catch(() => {});
                    load();
                  }}
                >
                  {t.alerts.unlink}
                </button>
              </div>
            ) : link ? (
              <a
                href={link}
                target="_blank"
                rel="noreferrer"
                className="mt-2 inline-block text-sm font-semibold text-brand hover:underline"
              >
                {t.alerts.linkBtn} →
              </a>
            ) : (
              /* ⚠️ Said plainly rather than drawing a link to nowhere: with no
                 bot configured there is nothing to tap, and a dead link is how
                 a working feature gets reported as broken. */
              <p className="mt-2 text-sm text-ink-muted">{t.alerts.noBot}</p>
            )}
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            {money("voidFrom")}
            {money("discountFrom")}
            {money("cashShortFrom")}
            {money("stockShortFrom")}
          </div>

          <label className="block text-sm">
            <span className="font-medium">{t.alerts.fields.dailyMax}</span>
            <input
              type="number"
              min={1}
              max={50}
              className="input mt-1 w-28"
              value={set.dailyMax}
              onChange={(e) =>
                patch({ dailyMax: Math.max(1, Number(e.target.value) || 1) })
              }
            />
            {/* ⚠️ The most important field on this page, and the one nobody
                would think to ask for. A bad night would otherwise send forty
                messages, and forty messages is silence. */}
            <span className="mt-1 block text-xs text-ink-muted">
              {t.alerts.hints.dailyMax}
            </span>
          </label>
        </>
      )}

      <div className="flex items-center gap-3">
        <button
          type="button"
          className="btn-primary px-4 py-2"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            try {
              const r = await api.adminSaveAlertSettings(set);
              setSet(r);
              setSaved(true);
            } finally {
              setBusy(false);
            }
          }}
        >
          {busy ? t.common.saving : t.common.save}
        </button>
        {/* A tick rather than a word: there is no "saved" string in the
            dictionary, and adding one in three languages to say what a tick
            already says is three more things to keep in step. */}
        {saved && <span className="text-sm text-emerald-600">✓</span>}
      </div>

      <p className="text-xs text-ink-muted">{t.alerts.footnote}</p>
    </div>
  );
}
