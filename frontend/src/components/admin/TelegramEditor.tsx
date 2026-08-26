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
  // ⚠️ Held as text, not as a number. A group id starts with "-100…" and a
  // half-typed "-" is not a number — a numeric state would swallow the minus
  // as the owner types it, and a group id without its minus is a chat that
  // does not exist.
  const [alertChat, setAlertChat] = useState("");
  const [feedbackChat, setFeedbackChat] = useState("");
  const [notifyLang, setNotifyLang] = useState("uz");
  const [saving, setSaving] = useState(false);
  const [checking, setChecking] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  function load(s: TelegramSettings) {
    setStored(s);
    setEnabled(s.enabled);
    setAlertChat(s.alertChatId ? String(s.alertChatId) : "");
    setFeedbackChat(s.feedbackChatId ? String(s.feedbackChatId) : "");
    // ⚠️ Empty from the server means "never chosen", and the messages are then
    // written in Uzbek — so that is what the dropdown must show, or the setting
    // displays one thing and the group receives another.
    setNotifyLang(s.notifyLang || "uz");
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
      load(
        await api.updateTelegram({
          enabled,
          botToken,
          // An empty box is a decision — "stop sending there" — so it goes as
          // zero rather than being left out.
          alertChatId: Number(alertChat.trim()) || 0,
          feedbackChatId: Number(feedbackChat.trim()) || 0,
          notifyLang,
        }),
      );
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

      {/* ⚠️ **Groups rather than a person's chat, and two of them.** An owner's
          own chat is one person, one phone and one holiday away from nobody
          reading any of it; a group survives them changing their number and
          keeps a history somebody can search. Two, because the suspicious-events
          one names employees and the feedback one does not — a floor manager can
          be given the second without the first. */}
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm">
          <span className="font-medium">{t.telegram.alertChat}</span>
          <input
            className="input mt-1"
            value={alertChat}
            inputMode="text"
            placeholder="-1001234567890"
            onChange={(e) => setAlertChat(e.target.value)}
          />
          <span className="mt-1 block text-xs text-ink-muted">
            {t.telegram.alertChatHint}
          </span>
        </label>

        <label className="block text-sm">
          <span className="font-medium">{t.telegram.feedbackChat}</span>
          <input
            className="input mt-1"
            value={feedbackChat}
            inputMode="text"
            placeholder="-1001234567890"
            onChange={(e) => setFeedbackChat(e.target.value)}
          />
          <span className="mt-1 block text-xs text-ink-muted">
            {t.telegram.feedbackChatHint}
          </span>
        </label>
      </div>

      {/* ⚠️ Beside the two ids, not under the bot token, because it is a
          property of what those groups receive rather than of the bot. An
          owner setting up a Russian-speaking accountant's group is thinking
          about that group while they are looking at its id. */}
      <label className="block text-sm sm:max-w-xs">
        <span className="font-medium">{t.telegram.notifyLang}</span>
        <select
          className="input mt-1"
          value={notifyLang}
          onChange={(e) => setNotifyLang(e.target.value)}
        >
          {(stored?.notifyLangs ?? ["uz", "ru", "en"]).map((l) => (
            <option key={l} value={l}>
              {t.telegram.langs[l] ?? l}
            </option>
          ))}
        </select>
        <span className="mt-1 block text-xs text-ink-muted">
          {t.telegram.notifyLangHint}
        </span>
      </label>

      {/* ⚠️ Written out rather than left as "find your chat id somewhere": the
          id is the one thing on this page an owner cannot work out by looking,
          and a setting nobody can fill in is a setting nobody turns on. */}
      <p className="rounded-xl border border-line bg-surface-soft p-3 text-xs text-ink-soft">
        {t.telegram.chatIdHow}
      </p>

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

      {/* ⚠️ The question the check button cannot answer.
          A bot is two halves: we call Telegram (proved above), and Telegram calls
          us. Only the second one makes the bot *reply*, and when it is missing
          the panel shows a perfectly connected bot while a guest pressing Start
          gets silence. So the fact is shown as what it is — a timestamp, which
          cannot go stale the way a stored "ok" flag does. */}
      {stored?.hasToken && (
        <div className="rounded-2xl border border-line bg-cream p-4">
          <p className="text-sm font-bold text-ink">{t.telegram.incomingTitle}</p>
          <p className="mt-1 text-xs text-ink-muted">{t.telegram.incomingHint}</p>
          <p
            className={`mt-2 text-xs font-semibold ${
              stored.lastUpdateAt && !stored.lastUpdateAt.startsWith("0001")
                ? "text-emerald-700 dark:text-emerald-300"
                : "text-ink-soft"
            }`}
          >
            {stored.lastUpdateAt && !stored.lastUpdateAt.startsWith("0001")
              ? t.telegram.incomingLast(formatDateTime(stored.lastUpdateAt))
              : t.telegram.incomingNever}
          </p>
          {stored.webhookAt && !stored.webhookAt.startsWith("0001") && (
            <p className="mt-1 text-xs text-ink-muted">
              {t.telegram.webhookAt(formatDateTime(stored.webhookAt))}
            </p>
          )}
        </div>
      )}

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
