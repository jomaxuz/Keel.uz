"use client";

// Turning the plan into a campaign, and what happens to it afterwards.
//
// ⚠️ **The budget is typed in the ad account's currency, not in so'm.** The
// plan proposes so'm because that is what the restaurant counts in; Meta bills
// the account in its own currency, almost never the som, and the owner's card
// is charged in that one. Showing the proposal beside a field in dollars is the
// honest shape — converting it for them would mean inventing an exchange rate
// and putting the result on the screen as though we knew it.
//
// ⚠️ **The ceiling is asked for before the campaign is made, not after.** It is
// the answer to "how much am I willing to lose in a day", and it binds the
// owner's own later edits as well as the rules — otherwise it is decoration.
//
// ⚠️ **Paused by default.** The switch is a separate press, which leaves one
// moment in which a mistake is still free.

import { useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import type { AdminDict } from "@/lib/i18n/admin";
import type {
  AdsIGPost,
  AdsAreaPick,
  AdsBudgetPick,
  AdsCampaign,
  AdsCampaignList,
  AdsDishPick,
  AdsEstimate,
  AdsSettingsView,
  AdsTextPick,
} from "@/lib/types";

/** Minor units as the owner reads them. ⚠️ `unit` comes from the server — the
 *  browser holds no currency table, because a second copy of the one that
 *  decides how much money moves is a copy that drifts. */
function fromMinor(minor: number, unit: number): string {
  if (unit <= 1) return String(Math.round(minor));
  return (minor / unit).toFixed(2);
}

function som(n: number): string {
  return new Intl.NumberFormat("uz-UZ").format(Math.round(n));
}

export function AdsCreate({
  t,
  settings,
  dish,
  area,
  budget,
  text,
  onCreated,
}: {
  t: AdminDict;
  settings: AdsSettingsView;
  dish: AdsDishPick | null;
  area: AdsAreaPick | null;
  budget: AdsBudgetPick | null;
  text: AdsTextPick | null;
  onCreated: () => void;
}) {
  const unit = settings.unit ?? 100;
  const [daily, setDaily] = useState("");
  const [cap, setCap] = useState("");
  const [days, setDays] = useState("");
  const [start, setStart] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [made, setMade] = useState<AdsCampaign | null>(null);
  const [estimate, setEstimate] = useState<AdsEstimate | null>(null);

  // ---- Two ways to make an advert ----
  //
  // ⚠️ **Assembled or borrowed, never both.** One builds a creative out of a
  // dish photograph and a wording the owner chose; the other puts money behind
  // a post their followers have already reacted to — and Meta keeps those likes
  // and comments, which is the only reason anybody asks for it. They are
  // different objects at Meta and they chase different outcomes, so the choice
  // is made here rather than folded into one form with optional halves.
  // ⚠️ **The wording is editable here, not fixed by the plan.** What the model
  // wrote is a starting point; the owner knows the shop's own voice, and a
  // screen that will only run a sentence a machine produced has turned a
  // suggestion into a rule. It is also the only way to advertise a dish the
  // plan wrote no wording for.
  //
  // ⚠️ **Meta cannot edit a creative after it is made**, so this is the last
  // moment the words can change — which is exactly why the field is here and
  // not on a screen that comes after.
  const [headline, setHeadline] = useState("");
  const [body, setBody] = useState("");
  useEffect(() => {
    if (!text) return;
    setHeadline(text.headline);
    setBody(text.body);
  }, [text]);

  const [mode, setMode] = useState<"plan" | "post">("plan");
  const [posts, setPosts] = useState<AdsIGPost[] | null>(null);
  const [igConnected, setIgConnected] = useState(true);
  const [post, setPost] = useState<AdsIGPost | null>(null);

  useEffect(() => {
    if (mode !== "post" || posts) return;
    api
      .adsInstagram()
      .then((r) => {
        setPosts(r.posts);
        setIgConnected(r.connected);
      })
      .catch(() => {
        setPosts([]);
        setIgConnected(false);
      });
  }, [mode, posts]);

  const ready =
    Number(daily) > 0 &&
    (mode === "post"
      ? Boolean(post)
      : Boolean(dish && headline.trim() && body.trim()));

  // ---- What Meta thinks this buys ----
  //
  // ⚠️ **Asked after the typing stops, not on every keystroke.** Each estimate
  // spends one of the ad account's hourly calls, and on a Limited-tier account
  // that ceiling is shared with everything else this restaurant does at Meta.
  //
  // ⚠️ **Cleared while a new one is in flight**, so a figure never sits under a
  // budget it was not calculated for — an owner reading last number against
  // this number is the whole failure mode of a live forecast.
  const radiusKm = area?.radiusKm;
  useEffect(() => {
    const amount = Number(daily);
    if (!amount || amount <= 0) {
      setEstimate(null);
      return;
    }
    setEstimate(null);
    const timer = window.setTimeout(() => {
      api
        .adsEstimate({ radiusKm, daily: amount })
        .then(setEstimate)
        // Quiet: a forecast that could not be fetched is a missing extra, not
        // a reason to put an error over a form that still works.
        .catch(() => setEstimate(null));
    }, 700);
    return () => window.clearTimeout(timer);
  }, [daily, radiusKm]);

  async function create() {
    if (!ready) return;
    setBusy(true);
    setError("");
    try {
      const row = await api.adsCreateCampaign(
        mode === "post"
          ? {
              sourcePostId: post?.id,
              dishName: dish?.name ?? "",
              areaLabel: area?.label,
              radiusKm: area?.radiusKm,
              daily: Number(daily),
              cap: cap ? Number(cap) : undefined,
              days: days ? Number(days) : budget?.days,
              start,
            }
          : {
              dishName: dish?.name ?? "",
              why: dish?.why,
              areaLabel: area?.label,
              radiusKm: area?.radiusKm,
              daily: Number(daily),
              cap: cap ? Number(cap) : undefined,
              days: days ? Number(days) : budget?.days,
              headline: headline.trim(),
              body: body.trim(),
              start,
            },
      );
      setMade(row);
      onCreated();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.campaign.failed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-3">
      {/* ⚠️ A pair of buttons rather than a dropdown: these are two different
          products of this section, and one of them is the reason a restaurant
          that posts its food every day bought it. */}
      <div className="flex flex-wrap gap-2">
        <ModeTab
          on={mode === "plan"}
          onClick={() => setMode("plan")}
          label={t.ads.campaign.modePlan}
        />
        <ModeTab
          on={mode === "post"}
          onClick={() => setMode("post")}
          label={t.ads.campaign.modePost}
        />
      </div>

      {mode === "post" && (
        <IGPicker
          t={t}
          posts={posts}
          connected={igConnected}
          chosen={post}
          onPick={setPost}
        />
      )}

      {mode === "plan" && !dish && (
        <p className="text-sm text-ink-muted">{t.ads.campaign.needPicks}</p>
      )}
      {/* ⚠️ **What is about to be created, in one block.** The four choices were
          made a screen and a half further up; an owner typing a budget here
          cannot see them, and the one thing they must not do is spend money on
          a campaign they have misremembered. */}
      {mode === "plan" && dish && (
      <dl className="grid gap-x-4 gap-y-1 rounded-xl bg-ink/5 p-3 text-sm sm:grid-cols-2">
        <Line k={t.ads.plan.dishes} v={dish.name} />
        <Line
          k={t.ads.plan.areas}
          v={
            area
              ? area.radiusKm
                ? `${area.label} · ${area.radiusKm} km`
                : area.label
              : t.ads.campaign.areaDefault
          }
        />

        {/* The proposal in so'm, beside a field in the account's currency.
            ⚠️ Shown rather than converted: we do not know the rate the owner's
            bank used, and a made-up one would read as a fact. */}
        {budget && (
          <Line
            k={t.ads.campaign.proposed}
            v={`${som(budget.daily)} ${t.ads.plan.perDay} · ${budget.days} ${
              t.ads.plan.days
            }`}
          />
        )}
      </dl>
      )}
      {mode === "plan" && dish && (
        <div className="grid gap-3">
          <label className="block text-sm">
            <span className="text-ink-soft">{t.ads.campaign.headline}</span>
            <input
              className="input mt-1 w-full"
              value={headline}
              maxLength={90}
              onChange={(e) => setHeadline(e.target.value)}
            />
          </label>
          <label className="block text-sm">
            <span className="text-ink-soft">{t.ads.campaign.bodyText}</span>
            <textarea
              className="input mt-1 w-full"
              rows={3}
              value={body}
              onChange={(e) => setBody(e.target.value)}
            />
          </label>
          <p className="text-xs text-ink-muted">{t.ads.campaign.textNote}</p>
        </div>
      )}

      {(mode === "post" ? post : dish) && (
      <>
          <div className="grid gap-3 sm:grid-cols-3">
            <Field
              label={`${t.ads.campaign.daily} (${settings.currency ?? ""})`}
              value={daily}
              onChange={setDaily}
            />
            <Field
              label={`${t.ads.campaign.cap} (${settings.currency ?? ""})`}
              value={cap}
              onChange={setCap}
            />
            <Field
              label={t.ads.campaign.days}
              value={days}
              onChange={setDays}
            />
          </div>
          {settings.minDailyBudget ? (
            <p className="text-xs text-ink-muted">
              {t.ads.campaign.minNote
                .replace("{amount}", fromMinor(settings.minDailyBudget, unit))
                .replace("{currency}", settings.currency ?? "")}
            </p>
          ) : null}
          {estimate && (
            <div className="rounded-xl border border-ink/10 p-3">
              <p className="text-xs font-semibold text-ink-soft">
                {t.ads.campaign.forecast}
              </p>
              {estimate.ready && estimate.reach ? (
                <dl className="mt-1 grid grid-cols-2 gap-x-4 gap-y-1 text-sm sm:grid-cols-3">
                  <Stat
                    k={t.ads.campaign.reach}
                    v={`${som(estimate.reach)}${
                      estimate.impressions
                        ? ` · ${som(estimate.impressions)} ${t.ads.campaign.impressions}`
                        : ""
                    }`}
                  />
                  {typeof estimate.results === "number" && (
                    <Stat
                      k={
                        estimate.goal === "OFFSITE_CONVERSIONS"
                          ? t.ads.campaign.resultsOrders
                          : t.ads.campaign.resultsClicks
                      }
                      v={som(estimate.results)}
                    />
                  )}
                  {typeof estimate.audience === "number" &&
                    estimate.audience > 0 && (
                      <Stat
                        k={t.ads.campaign.audience}
                        v={som(estimate.audience)}
                      />
                    )}
                </dl>
              ) : (
                <p className="mt-1 text-sm text-ink-muted">
                  {t.ads.campaign.forecastNotReady}
                </p>
              )}
              {/* ⚠️ Whose estimate it is, said every time it is shown. Our own
                  planner is forbidden from predicting results; this number is
                  Meta's, and an owner has to be able to tell them apart. */}
              <p className="mt-1 text-xs text-ink-muted">
                {t.ads.campaign.forecastNote}
              </p>
            </div>
          )}

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={start}
              onChange={(e) => setStart(e.target.checked)}
            />
            <span>{t.ads.campaign.startNow}</span>
          </label>
          <p className="max-w-2xl text-xs text-ink-muted">
            {t.ads.campaign.capNote}
          </p>
          {error && <p className="text-sm text-danger">{error}</p>}
          {made && (
            <p className="text-sm text-brand-dark">
              {t.ads.campaign.created} · {made.name}
            </p>
          )}
          <button
            type="button"
            onClick={create}
            disabled={busy || !ready}
            className="btn btn-primary disabled:opacity-40"
          >
            {busy ? t.ads.campaign.creating : t.ads.campaign.create}
          </button>
      </>
      )}
    </div>
  );
}

function Line({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex gap-2">
      <dt className="shrink-0 text-ink-muted">{k}:</dt>
      <dd className="min-w-0 font-medium">{v}</dd>
    </div>
  );
}

export function AdsCampaigns({
  t,
  list,
  reload,
}: {
  t: AdminDict;
  list: AdsCampaignList | null;
  reload: () => void;
}) {
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const unit = list?.unit ?? 100;
  const statuses = t.ads.campaign.status as Record<string, string>;

  async function act(id: string, action: string, daily?: number) {
    setBusy(id + action);
    setError("");
    try {
      await api.adsUpdateCampaign(id, { action, daily });
      reload();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.campaign.failed);
    } finally {
      setBusy("");
    }
  }

  const rows = list?.campaigns ?? [];
  if (rows.length === 0) return null;

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold">{t.ads.campaign.listTitle}</h3>
      {error && <p className="text-sm text-danger">{error}</p>}
      <div className="space-y-3">
        {rows.map((c) => (
          <div key={c.id} className="rounded-xl border border-ink/10 p-3">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <p className="text-sm font-semibold">{c.name}</p>
              <span className="text-xs text-ink-muted">
                {statuses[c.status] ?? c.status}
              </span>
            </div>
            {c.why && <p className="mt-1 text-xs text-ink-muted">{c.why}</p>}

            {/* ⚠️ Meta's own word is shown beside ours, untranslated, and only
                when it disagrees: "is my advert running" is not the same
                question as which object in Meta's chain is paused. */}
            {c.reviewNote && (
              <p className="mt-1 text-xs text-danger">{c.reviewNote}</p>
            )}

            <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-4">
              <Stat
                k={t.ads.campaign.budget}
                v={`${fromMinor(c.dailyMinor, unit)} ${c.currency ?? ""}`}
              />
              <Stat
                k={t.ads.campaign.spend}
                v={`${c.spend.toFixed(2)} ${c.currency ?? ""}`}
              />
              <Stat k={t.ads.campaign.orders} v={String(c.purchases)} />
              <Stat
                k={t.ads.campaign.revenue}
                v={`${som(c.revenue)} UZS`}
              />
            </dl>

            {/* What the rules did, and why. ⚠️ Shown on the campaign rather
                than only in a log: an automatic pause the owner discovers as
                "my advert stopped" costs more trust than the money it saved. */}
            {c.decisions && c.decisions.length > 0 && (
              <ul className="mt-2 space-y-0.5 text-xs text-ink-muted">
                {c.decisions.slice(-3).map((d, i) => (
                  <li key={i}>
                    {new Date(d.at).toLocaleDateString()} · {d.why}
                  </li>
                ))}
              </ul>
            )}

            <div className="mt-2 flex flex-wrap gap-2">
              {c.status !== "active" && c.status !== "stopped" && (
                <Act
                  label={t.ads.campaign.actions.start}
                  busy={busy === c.id + "start"}
                  onClick={() => act(c.id, "start")}
                />
              )}
              {c.status === "active" && (
                <Act
                  label={t.ads.campaign.actions.pause}
                  busy={busy === c.id + "pause"}
                  onClick={() => act(c.id, "pause")}
                />
              )}
              {c.status !== "stopped" && (
                <Act
                  label={t.ads.campaign.actions.stop}
                  busy={busy === c.id + "stop"}
                  onClick={() => act(c.id, "stop")}
                />
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function Stat({ k, v }: { k: string; v: string }) {
  return (
    <div>
      <dt className="text-ink-muted">{k}</dt>
      <dd className="font-medium">{v}</dd>
    </div>
  );
}

function Act({
  label,
  busy,
  onClick,
}: {
  label: string;
  busy: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={busy}
      className="btn btn-ghost text-xs disabled:opacity-40"
    >
      {label}
    </button>
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

function ModeTab({
  on,
  onClick,
  label,
}: {
  on: boolean;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={on}
      className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
        on
          ? "border-brand bg-brand-tint font-semibold text-brand-dark"
          : "border-ink/10 text-ink-soft hover:border-brand/40"
      }`}
    >
      {label}
    </button>
  );
}

// The restaurant's own posts, with Meta's verdict on each.
//
// ⚠️ **What cannot be boosted is shown and refused, not hidden.** An owner
// looking for the post they remember and not finding it assumes the feature is
// broken; a post greyed out with Meta's reason beside it is an answer.
function IGPicker({
  t,
  posts,
  connected,
  chosen,
  onPick,
}: {
  t: AdminDict;
  posts: AdsIGPost[] | null;
  connected: boolean;
  chosen: AdsIGPost | null;
  onPick: (p: AdsIGPost) => void;
}) {
  if (!connected) {
    return <p className="text-sm text-ink-muted">{t.ads.campaign.noInstagram}</p>;
  }
  if (posts === null) {
    return <p className="text-sm text-ink-muted">{t.ads.campaign.loadingPosts}</p>;
  }
  if (posts.length === 0) {
    return <p className="text-sm text-ink-muted">{t.ads.campaign.noPosts}</p>;
  }
  return (
    <div className="grid grid-cols-3 gap-2 sm:grid-cols-6">
      {posts.map((p) => {
        const ok = !p.blocked;
        const on = chosen?.id === p.id;
        return (
          <button
            key={p.id}
            type="button"
            disabled={!ok}
            onClick={() => onPick(p)}
            aria-pressed={on}
            title={ok ? (p.caption ?? "") : (p.why ?? "")}
            className={`relative aspect-square overflow-hidden rounded-lg border transition ${
              on ? "border-brand ring-2 ring-brand" : "border-ink/10"
            } ${ok ? "hover:border-brand/40" : "cursor-not-allowed opacity-40"}`}
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={p.image}
              alt={p.caption?.slice(0, 60) ?? ""}
              className="h-full w-full object-cover"
            />
            {/* Which network it came from. ⚠️ Shown because the two behave
                differently once promoted, and an owner who cannot tell them
                apart cannot tell why the results differ. */}
            <span className="absolute left-1 top-1 rounded bg-ink/70 px-1 text-[10px] text-white">
              {p.source === "instagram" ? "IG" : "FB"}
            </span>
            {!ok && (
              <span className="absolute inset-x-0 bottom-0 bg-ink/70 px-1 py-0.5 text-[10px] text-white">
                {t.ads.campaign.notBoostable}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
