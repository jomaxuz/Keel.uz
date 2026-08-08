"use client";

// Connecting the restaurant's own Telegram bot.
//
// The bot is the owner's — their name on it, their contract with Telegram, their
// brand on the mini app — the same reasoning that put the SMS gateway and the
// payment providers in their hands rather than the platform's.
//
// The shape of this page follows two facts about how it is actually used:
//
//   • **The check button is the main control**, not the save button. A token
//     that looks right and belongs to a deleted bot is indistinguishable from a
//     working one until the first guest tries to sign in — and the restaurant
//     reads that as "your site is broken". Pressing it answers with the bot's own
//     name, which is the difference between configured and configured correctly.
//
//   • **The link is the deliverable.** Once the bot answers, the operator needs
//     one thing to hand over: the address that opens the mini app. It is built
//     server-side from the username the check found, so nobody assembles it by
//     hand — a mistyped bot name opens somebody else's bot.
//
// The token follows the payment-keys rule: shown as "stored", never rendered,
// and an empty field means keep the stored one.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { formatDateTime } from "@/lib/format";
import type { TelegramSettings } from "@/lib/types";

export default function TelegramEditor() {
  const t = useAdminT();
  const [stored, setStored] = useState<TelegramSettings | null>(null);
  const [enabled, setEnabled] = useState(false);
  const [botToken, setBotToken] = useState("");
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  function load(s: TelegramSettings) {
    setStored(s);
    setEnabled(s.enabled);
    // Never prefilled: the server does not return it, and a blank box that means
    // "keep it" is the only honest thing to show.
    setBotToken("");
  }

  useEffect(() => {
    api
      .adminTelegram()
      .then(load)
      .catch((e) => setError(e instanceof Error ? e.message : ""));
  }, []);

  async function save() {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      load(await api.updateTelegram({ enabled, botToken }));
      setMessage(t.settings.saved);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function check() {
    setChecking(true);
    setError("");
    setMessage("");
    try {
      const res = await api.pingTelegram();
      if (res.ok) setMessage(res.message);
      else setError(res.message);
      // Reread: a successful check is what fills in the username, and the link
      // below is built from it.
      api.adminTelegram().then(load).catch(() => {});
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setChecking(false);
    }
  }

  const live = stored?.enabled && stored?.hasToken;

  return (
    <div className="space-y-5">
      <p className="text-sm text-ink-muted">{t.telegram.intro}</p>

      {/* The state of the thing, before any of the fields — an owner opening
          this page is nearly always here because a guest could not sign in. */}
      {!live && (
        <div className="rounded-xl border border-amber-500/40 bg-amber-500/10 p-3">
          <p className="text-sm font-semibold">{t.telegram.offTitle}</p>
          <p className="mt-1 text-xs text-ink-muted">{t.telegram.offHint}</p>
        </div>
      )}

      <label className="flex items-start gap-2 text-sm">
        <input
          type="checkbox"
          className="mt-0.5"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
        />
        <span>
          {t.telegram.enabled}
          <span className="block text-xs text-ink-muted">
            {t.telegram.enabledHint}
          </span>
        </span>
      </label>

      <div>
        <label className="text-sm font-medium">
          {t.telegram.token}{" "}
          {stored?.hasToken && (
            <span className="text-xs font-normal text-emerald-700 dark:text-emerald-300">
              · {t.telegram.tokenStored}
            </span>
          )}
        </label>
        <input
          className="input mt-1"
          value={botToken}
          autoComplete="off"
          placeholder={stored?.hasToken ? t.telegram.tokenKeep : "123456:AA..."}
          onChange={(e) => setBotToken(e.target.value)}
        />
        <p className="mt-1 text-xs text-ink-muted">{t.telegram.tokenHint}</p>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={save}
          disabled={saving}
          className="btn btn-primary disabled:opacity-40"
        >
          {saving ? t.common.saving : t.common.save}
        </button>
        <button
          type="button"
          onClick={check}
          disabled={checking || !stored?.hasToken}
          className="btn btn-dark disabled:opacity-40"
        >
          {checking ? t.telegram.checking : t.telegram.check}
        </button>
        {stored?.lastCheck && (
          <span
            className={`text-xs ${
              stored.lastCheckOk
                ? "text-emerald-700 dark:text-emerald-300"
                : "text-brand"
            }`}
          >
            {t.telegram.lastCheck(formatDateTime(stored.lastCheckAt))}:{" "}
            {stored.lastCheck}
          </span>
        )}
      </div>

      {/* The one thing the operator hands to the restaurant. Only after a
          successful check, because a link built from an unverified username
          opens another bot — and that failure looks like ours. */}
      {stored?.miniAppUrl && (
        <div className="rounded-xl border border-line bg-surface p-3">
          <p className="text-sm font-semibold">{t.telegram.linkTitle}</p>
          <p className="mt-1 text-xs text-ink-muted">{t.telegram.linkHint}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <code className="rounded-lg bg-ink/5 px-2 py-1 text-sm">
              {stored.miniAppUrl}
            </code>
            <button
              type="button"
              onClick={() => {
                navigator.clipboard?.writeText(stored.miniAppUrl ?? "");
                setCopied(true);
                setTimeout(() => setCopied(false), 2000);
              }}
              className="chip"
            >
              {copied ? t.payments.copied : t.payments.copy}
            </button>
          </div>
        </div>
      )}

      <p className="rounded-xl border border-line px-3 py-2 text-xs text-ink-soft">
        {t.telegram.phoneNote}
      </p>

      {error && <p className="text-sm text-brand">{error}</p>}
      {message && (
        <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
          {message}
        </p>
      )}
    </div>
  );
}
