"use client";

// One message to one segment.
//
// The segments already existed and led nowhere: the panel could say that eleven
// VIPs had gone quiet and then offer nothing to do about it. This is the other
// half — and because it spends real money on real people's phones, most of the
// screen is there to slow the owner down at the right moments.
//
//   • **Two numbers per segment, always**: how many are in it and how many can
//     actually be messaged. A badge saying 24 next to a send of 19 is a screen
//     people stop believing, and the gap is the useful part.
//   • **The cost is spelled out before sending**, in messages rather than
//     recipients. Non-Latin text is 70 characters per part, not 160, and it is
//     billed per part — so a polite closing sentence can double the bill with
//     nothing on screen changing.
//   • **Sending is a second, separate press** with the count in the button.
//     There is no recalling an SMS.
//   • **Opted-out guests are never in the audience**, and the screen says so
//     rather than hiding it: it is the number that keeps the restaurant welcome.

import { useCallback, useEffect, useState } from "react";
import CustomerPicker from "@/components/admin/CustomerPicker";
import ImageUpload from "@/components/admin/ImageUpload";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import RfmGrid from "@/components/admin/RfmGrid";
import { formatDateTime } from "@/lib/format";
import { ListScroll } from "@/components/admin/PagedList";
import type { Campaign, CampaignPreview, SegmentRow } from "@/lib/types";

type SegKey = keyof ReturnType<typeof useAdminT>["users"]["segment"];

export default function AdminCampaignsPage() {
  const t = useAdminT();
  const [segments, setSegments] = useState<SegmentRow[] | null>(null);
  const [history, setHistory] = useState<Campaign[]>([]);
  const [segment, setSegment] = useState("");
  const [text, setText] = useState("");
  // ⚠️ The channel is part of the message, not a setting: the same words cost money
  // as an SMS and nothing through the bot, and they reach different people — a guest
  // who signed in through Telegram may have no phone number at all.
  // Segment or one named person. ⚠️ Two ways of choosing an audience rather than two
  // features: everything after this — the opt-out rule, the channel rules, the entry
  // in the campaign log — is identical, and a second send path would eventually
  // forget one of them.
  const [target, setTarget] = useState<"segment" | "one">("segment");
  const [person, setPerson] = useState<{ id: string; label: string } | null>(null);
  const [picking, setPicking] = useState(false);
  const [channel, setChannel] = useState("sms");
  // Telegram only. An SMS has no such thing, and offering the field for one would
  // be offering something that silently does nothing.
  const [image, setImage] = useState("");
  const [preview, setPreview] = useState<CampaignPreview | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const load = useCallback(async () => {
    try {
      const [segs, camps] = await Promise.all([
        api.adminSegments(),
        api.adminCampaigns(),
      ]);
      setSegments(segs);
      setHistory(camps);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [t]);

  useEffect(() => {
    load();
  }, [load]);

  // While something is sending, the history is the progress bar — so it is
  // reread until nothing is in flight and then left alone.
  useEffect(() => {
    if (!history.some((c) => c.status === "sending")) return;
    const id = setInterval(() => {
      api.adminCampaigns().then(setHistory).catch(() => {});
    }, 3000);
    return () => clearInterval(id);
  }, [history]);

  // Any edit invalidates the preview. Sending against a stale count is exactly
  // the mistake the preview exists to prevent.
  function edit(next: { segment?: string; text?: string; channel?: string; image?: string }) {
    if (next.segment !== undefined) setSegment(next.segment);
    if (next.text !== undefined) setText(next.text);
    if (next.channel !== undefined) setChannel(next.channel);
    if (next.image !== undefined) setImage(next.image);
    setPreview(null);
    setMessage("");
  }

  async function runPreview() {
    setBusy(true);
    setError("");
    try {
      setPreview(
        await api.campaignPreview(
          target === "one"
            ? { text, channel, image, userId: person?.id }
            : { segment, text, channel, image },
        ),
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function send() {
    setBusy(true);
    setError("");
    try {
      const res = await api.sendCampaign(
        target === "one"
          ? { text, channel, image, userId: person?.id }
          : { segment, text, channel, image },
      );
      setMessage(t.campaigns.started(res.recipients));
      setText("");
      setPreview(null);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const chosen = segments?.find((s) => s.segment === segment) ?? null;
  const canPreview =
    text.trim().length > 0 && !busy && (target === "one" ? !!person : !!segment);

  return (
    <div className="space-y-5">
      {picking && (
        <CustomerPicker
          channel={channel}
          onClose={() => setPicking(false)}
          onPick={(u) => {
            setPerson({ id: u.id, label: u.firstName || u.phone });
            setPicking(false);
            setPreview(null);
          }}
        />
      )}
      <div>
        <h1 className="font-display text-2xl font-bold">{t.campaigns.title}</h1>
        <p className="text-sm text-ink-muted">{t.campaigns.hint}</p>
      </div>

      {/* ---- who ---- */}
      <section className="card p-5">
        <h2 className="font-semibold">{t.campaigns.pickSegment}</h2>
        {segments === null ? (
          <p className="mt-3 text-sm text-ink-muted">{t.common.loading}</p>
        ) : (
          <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
            {/* ⚠️ The rule segments only. The RFM cells arrive in the same
                list from the server — they are audiences in exactly the same
                sense — but they are picked in their own panel below, which can
                show what each cell is *worth*. Fifteen identical buttons in one
                grid would bury the distinction between "crossed a line" and
                "ranked against everybody else". */}
            {segments
              .filter((s) => !s.segment.startsWith("rfm:"))
              .map((s) => {
              const active = s.segment === segment;
              return (
                <button
                  key={s.segment}
                  type="button"
                  // A segment nobody can be messaged in is not a choice; it is
                  // disabled rather than hidden so the owner can see it exists
                  // and is empty.
                  disabled={s.reachable === 0}
                  onClick={() => edit({ segment: s.segment })}
                  className={`rounded-2xl border p-3 text-left transition-colors disabled:opacity-40 ${
                    active
                      ? "border-brand bg-brand/5"
                      : "border-line bg-surface hover:border-brand"
                  }`}
                >
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="font-semibold">
                      {t.users.segment[s.segment as SegKey] ?? s.segment}
                    </span>
                    <span className="font-display text-xl font-bold tabular-nums">
                      {s.reachable}
                    </span>
                  </div>
                  <p className="mt-0.5 text-xs text-ink-muted">
                    {t.users.segHint[s.segment as SegKey] ?? ""}
                  </p>
                  {(s.optedOut > 0 || s.noPhone > 0) && (
                    <p className="mt-1 text-xs text-ink-muted">
                      {t.campaigns.excluded(s.optedOut, s.noPhone)}
                    </p>
                  )}
                </button>
              );
            })}
          </div>
        )}

        {/* The same choice, ranked. Clicking a cell sets the audience exactly
            as the buttons above do — one piece of state, so the two pickers
            cannot both look selected. */}
        <div className="mt-5 border-t border-line pt-4">
          <h3 className="text-sm font-semibold">{t.campaigns.rfm.title}</h3>
          <div className="mt-3">
            <RfmGrid onPick={(seg) => edit({ segment: seg })} />
          </div>
        </div>
      </section>

      {/* ---- what ---- */}
      <section className="card p-5">
        <h2 className="font-semibold">{t.campaigns.text}</h2>

        {/* Segment, or one named person. */}
        <div className="mt-3 flex flex-wrap gap-2">
          {(["segment", "one"] as const).map((v) => (
            <button
              key={v}
              type="button"
              onClick={() => {
                setTarget(v);
                setPreview(null);
                setMessage("");
              }}
              className={`rounded-full border px-3 py-1.5 text-xs font-semibold ${
                target === v
                  ? "border-brand bg-brand-tint text-brand-dark"
                  : "border-line text-ink-soft"
              }`}
            >
              {v === "segment" ? t.campaigns.toSegment : t.campaigns.toOne}
            </button>
          ))}
          {target === "one" && (
            <button
              type="button"
              onClick={() => setPicking(true)}
              className="rounded-full border border-line px-3 py-1.5 text-xs font-semibold text-ink"
            >
              {person ? person.label : t.campaigns.chooseCustomer}
            </button>
          )}
        </div>

        {/* ⚠️ The channel comes before the message, because it changes what the
            message can be: a photograph and two buttons through the bot, 70 or 160
            characters and a bill through SMS. Choosing it afterwards means writing
            for one and sending through the other. */}
        <div className="mt-3 flex flex-wrap gap-2">
          {(["sms", "telegram", "push"] as const).map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => edit({ channel: c })}
              className={`rounded-full border px-3 py-1.5 text-xs font-semibold ${
                channel === c
                  ? "border-brand bg-brand-tint text-brand-dark"
                  : "border-line text-ink-soft"
              }`}
            >
              {t.campaigns.channelName[c]}
            </button>
          ))}
        </div>
        <p className="mt-1 text-xs text-ink-muted">
          {t.campaigns.channelHint[channel as "sms" | "telegram" | "push"] ??
            t.campaigns.smsHint}
        </p>

        <textarea
          className="input mt-2 min-h-28"
          maxLength={480}
          value={text}
          placeholder={t.campaigns.textPh}
          onChange={(e) => edit({ text: e.target.value })}
        />
        <p className="mt-1 text-xs text-ink-muted">{t.campaigns.textHint}</p>

        {channel === "telegram" && (
          <label className="mt-3 block">
            <span className="text-xs font-semibold text-ink-muted">
              {t.campaigns.image}
            </span>
            {/* ⚠️ The upload control, not a path field.
                A field expecting "/uploads/…" asks the operator to know where our
                files live and to have put one there already — which is the same
                control every other image in this panel uses, and there was no reason
                for this one to be different. */}
            <div className="mt-1">
              <ImageUpload value={image} onChange={(url) => edit({ image: url })} />
            </div>
            <span className="mt-1 block text-xs text-ink-muted">
              {t.campaigns.imageHint}
            </span>
          </label>
        )}

        <div className="mt-3 flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={runPreview}
            disabled={!canPreview}
            className="btn btn-dark disabled:opacity-40"
          >
            {t.campaigns.check}
          </button>

          {preview && (
            <>
              {/* The one number that matters, and it is not "recipients".
                  ⚠️ Through the bot there is no per-part cost at all, so the SMS
                  summary would print a price for something free — which is the kind
                  of wrong number that stops an owner using a feature. */}
              <p className="text-sm text-ink-soft">
                {/* Free through the bot and free through push. Printing the SMS
                    price line for either would put a cost on something that has
                    none — the kind of wrong number that stops an owner using a
                    feature at all. */}
                {channel === "sms"
                  ? t.campaigns.summary(
                      preview.recipients,
                      preview.parts,
                      preview.messages,
                    )
                  : t.campaigns.summaryFree(preview.recipients)}
              </p>
              {preview.blocked && (
                <p className="text-xs text-brand">{preview.blocked}</p>
              )}
              {channel === "telegram" && (preview.noTelegram ?? 0) > 0 && (
                <p className="text-xs text-ink-muted">
                  {t.campaigns.noTelegram(preview.noTelegram ?? 0)}
                </p>
              )}
              {/* ⚠️ Its own line, not folded into "no phone". The three
                  exclusions have three different fixes — collect a number,
                  invite them to the bot, ask them on the site — and one merged
                  count tells the owner none of them. */}
              {channel === "push" && (preview.noPush ?? 0) > 0 && (
                <p className="text-xs text-ink-muted">
                  {t.campaigns.noPush(preview.noPush ?? 0)}
                </p>
              )}
              <button
                type="button"
                onClick={send}
                disabled={
                  busy ||
                  preview.recipients === 0 ||
                  // ⚠️ Each channel has its own precondition, and the SMS one must
                  // not block a Telegram send: a restaurant with a bot and no SMS
                  // contract is a normal state, not a broken one.
                  // ⚠️ Each channel has its own precondition and they must not
                  // block each other. Push has none: the keys are generated on
                  // first use, so the only thing that can make it reach nobody
                  // is that nobody subscribed — which `recipients === 0`
                  // already catches, above.
                  (channel === "push"
                    ? false
                    : channel === "telegram"
                      ? preview.ready === false
                      : preview.demo)
                }
                className="btn btn-primary disabled:opacity-40"
              >
                {t.campaigns.send(preview.recipients)}
              </button>
            </>
          )}
        </div>

        {/* Said before the press, not discovered after: with no gateway the
            campaign would "succeed" and reach nobody. */}
        {preview?.demo && (
          <p className="mt-3 rounded-xl border border-amber-500/40 bg-amber-500/5 px-3 py-2 text-sm text-amber-800 dark:text-amber-200">
            {t.campaigns.noGateway}
          </p>
        )}
        {error && <p className="mt-3 text-sm text-brand">{error}</p>}
        {message && (
          <p className="mt-3 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
            {message}
          </p>
        )}
        {chosen && (
          <p className="mt-3 text-xs text-ink-muted">{t.campaigns.optOutNote}</p>
        )}
      </section>

      {/* ---- what was sent ---- */}
      <section className="card p-5">
        <h2 className="font-semibold">{t.campaigns.history}</h2>
        {history.length === 0 ? (
          <p className="mt-3 text-sm text-ink-muted">{t.campaigns.historyEmpty}</p>
        ) : (
          <ListScroll max="max-h-[420px]">
            <ul className="mt-3 divide-y divide-line">
              {history.map((c) => (
                <li key={c.id} className="py-3">
                  <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="font-semibold">
                      {t.users.segment[c.segment as SegKey] ?? c.segment}
                    </span>
                    <span className="text-xs text-ink-muted">
                      {formatDateTime(c.createdAt)} · {c.createdBy}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-ink-soft">{c.text}</p>
                  <p className="mt-1 text-xs tabular-nums text-ink-muted">
                    {c.status === "sending"
                      ? t.campaigns.progress(c.sent + c.failed, c.total)
                      : t.campaigns.result(c.sent, c.failed)}
                  </p>
                  {/* The gateway's own sentence. "84 failed" with no reason
                      leaves three different next steps indistinguishable. */}
                  {c.error && (
                    <p className="mt-1 text-xs text-brand">{c.error}</p>
                  )}
                </li>
              ))}
            </ul>
          </ListScroll>
        )}
      </section>
    </div>
  );
}
