"use client";

// Advertising: Meta campaigns, decided from this restaurant's own figures.
//
// ⚠️ **The money is the restaurant's and goes straight to Meta.** The ad account
// is theirs and the card on it is theirs; what is sold here is the work around
// it. The screen says so in a line, because an owner who suspects we spent their
// money stops trusting every other number on the panel — and that suspicion is
// impossible to argue with after the fact.
//
// ⚠️ **The plan and the campaign are separate halves, and the plan works
// without Meta.** Deciding what to advertise needs only this restaurant's own
// week; creating the advert needs a connected account. So the screen draws the
// plan for everybody and the campaign section only once the connection can
// actually carry one — and when it cannot, it says which of the five steps is
// missing rather than "not connected".
//
// ⚠️ **The bars are drawn from the server's arithmetic, never from the plan.**
// Every percentage on this screen comes out of `facts`, which the restaurant's
// own server computed; the model's words sit beside them. So a model that got
// carried away can write a poor reason but cannot move a bar — which is the
// difference between advice an owner can check and advice they have to trust.
//
// ⚠️ The section itself is gated as a bought module (`modulegate.go` on the
// server, `PANEL_ROUTES` here), so this page only ever draws for a restaurant
// that has it. The states below are about Meta, not about the invoice.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type {
  AdsAreaPick,
  AdsBudgetPick,
  AdsCampaignList,
  AdsDishPick,
  AdsPlanAnswer,
  AdsReport,
  AdsState,
  AdsTextPick,
} from "@/lib/types";

import AdsConnect from "./connect";
import { AdsCampaigns, AdsCreate } from "./campaigns";
import AdsReportCard from "./report";
import AdsRules from "./rules";

// So'm, grouped. ⚠️ Whole so'm everywhere in this product — there are no tiyin.
function som(n: number): string {
  return new Intl.NumberFormat("uz-UZ").format(Math.round(n));
}

export default function AdminAdsPage() {
  const t = useAdminT();
  const [state, setState] = useState<AdsState | null>(null);
  const [error, setError] = useState("");

  const [answer, setAnswer] = useState<AdsPlanAnswer | null>(null);
  const [building, setBuilding] = useState(false);

  // The owner's own choices. ⚠️ Held on the screen and nowhere else, on purpose:
  // until a campaign can actually be created there is nothing to save them
  // *into*, and a "saved" choice that no campaign ever reads is a promise the
  // product does not keep.
  const [dish, setDish] = useState<AdsDishPick | null>(null);
  const [area, setArea] = useState<AdsAreaPick | null>(null);
  const [budget, setBudget] = useState<AdsBudgetPick | null>(null);
  const [text, setText] = useState<AdsTextPick | null>(null);
  const [copied, setCopied] = useState(false);

  const [list, setList] = useState<AdsCampaignList | null>(null);
  const [report, setReport] = useState<AdsReport | null>(null);

  const reload = useCallback(() => {
    api
      .adsState()
      .then(setState)
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.ads.loadFailed),
      );
    // ⚠️ Both are allowed to fail quietly: a restaurant with no campaigns yet
    // is the ordinary case, and an error banner over an empty list would
    // describe nothing anybody can act on.
    api.adsCampaigns().then(setList).catch(() => {});
    api.adsReport().then(setReport).catch(() => {});
  }, [t.ads.loadFailed]);

  useEffect(() => {
    reload();
  }, [reload]);

  async function build() {
    setBuilding(true);
    setError("");
    try {
      setAnswer(await api.adsPlan());
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.ads.plan.failed);
    } finally {
      setBuilding(false);
    }
  }

  const facts = answer?.facts;
  const plan = answer?.plan;
  // ⚠️ `?? []` on all four: the server sends what the model returned, and a plan
  // missing one group is a plan, not a crash. The Go-side trap this codebase has
  // been bitten by twice (a nil slice arriving as `null`) lands here too.
  const dishes = plan?.dishes ?? [];
  const areas = plan?.areas ?? [];
  const budgets = plan?.budgets ?? [];
  const texts = plan?.texts ?? [];

  async function copyText() {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(`${text.headline}\n\n${text.body}`);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      // A clipboard the browser refused is not worth an error banner: the
      // wording is on the screen and can be selected by hand.
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.ads.title}</h1>
        <p className="mt-1 max-w-2xl text-sm text-ink-soft">{t.ads.lead}</p>
        {/* ⚠️ Beside the lead rather than in a footnote: this is the sentence
            that decides whether the next screen is read as ours or as theirs. */}
        <p className="mt-2 max-w-2xl text-xs text-ink-muted">
          {t.ads.spendNote}
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {state && !state.on && (
        <div className="card space-y-2 p-4">
          <p className="font-semibold">{t.ads.offTitle}</p>
          <p className="text-sm text-ink-soft">{t.ads.off}</p>
        </div>
      )}

      {state?.on && <AdsConnect t={t} state={state} reload={reload} />}

      {/* ---- The plan ---- */}
      {state?.on && (
        <div className="card space-y-4 p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="font-semibold">{t.ads.plan.title}</p>
              <p className="mt-1 max-w-2xl text-sm text-ink-soft">
                {t.ads.plan.lead}
              </p>
            </div>
            <button
              type="button"
              onClick={build}
              disabled={building}
              className="btn btn-primary disabled:opacity-40"
            >
              {plan ? t.ads.plan.rebuild : t.ads.plan.build}
            </button>
          </div>

          {answer?.off && <p className="text-sm text-ink-soft">{t.ads.off}</p>}
          {answer?.capped && (
            <p className="text-sm text-ink-soft">{t.ads.plan.capped}</p>
          )}
          {answer?.error && (
            <p className="text-sm text-danger">{answer.error}</p>
          )}

          {/* ---- What the week actually looks like ---- */}
          {facts && facts.dishes.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm text-ink-soft">
                {t.ads.plan.weekTotal}: {som(facts.weekTotal)}{" "}
                {facts.currency}
              </p>
              {facts.dishes.map((d) => (
                <div key={d.name}>
                  <div className="flex items-baseline justify-between gap-2 text-sm">
                    <span className="font-medium">{d.name}</span>
                    <span className="text-xs text-ink-muted">
                      {d.share}% · {t.ads.plan.share}
                    </span>
                  </div>
                  {/* ⚠️ Width straight from the server's `share`. The bar is the
                      figure, not an impression of it. */}
                  <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-ink/10">
                    <div
                      className="h-full rounded-full bg-brand transition-[width] duration-300"
                      style={{ width: `${Math.min(100, d.share)}%` }}
                    />
                  </div>
                  <p className="mt-1 text-xs text-ink-muted">
                    {som(d.money)} {facts.currency} · {t.ads.plan.lastWeek}:{" "}
                    {som(d.lastWeek)}
                  </p>
                </div>
              ))}
            </div>
          )}

          {answer?.asOf && (
            <p className="text-xs text-ink-muted">
              {t.ads.plan.asOf}: {new Date(answer.asOf).toLocaleString()}
              {answer.cached ? ` · ${t.ads.plan.cached}` : ""}
            </p>
          )}

          {/* ---- The proposals, one group at a time ---- */}
          {dishes.length > 0 && (
            <Group title={t.ads.plan.dishes}>
              {dishes.map((d) => (
                <Pick
                  key={d.name}
                  head={d.name}
                  why={d.why}
                  chosen={dish?.name === d.name}
                  onPick={() => setDish(d)}
                  choose={t.ads.plan.choose}
                  chosenLabel={t.ads.plan.chosen}
                />
              ))}
            </Group>
          )}

          {areas.length > 0 && (
            <Group title={t.ads.plan.areas}>
              {areas.map((a) => (
                <Pick
                  key={a.label}
                  head={
                    a.radiusKm ? `${a.label} · ${a.radiusKm} km` : a.label
                  }
                  why={a.why}
                  chosen={area?.label === a.label}
                  onPick={() => setArea(a)}
                  choose={t.ads.plan.choose}
                  chosenLabel={t.ads.plan.chosen}
                />
              ))}
            </Group>
          )}

          {budgets.length > 0 && (
            <Group title={t.ads.plan.budgets}>
              {budgets.map((b) => (
                <Pick
                  key={`${b.daily}-${b.days}`}
                  head={`${som(b.daily)} ${t.ads.plan.perDay} · ${b.days} ${
                    t.ads.plan.days
                  }`}
                  why={b.why}
                  chosen={budget?.daily === b.daily && budget?.days === b.days}
                  onPick={() => setBudget(b)}
                  choose={t.ads.plan.choose}
                  chosenLabel={t.ads.plan.chosen}
                />
              ))}
            </Group>
          )}

          {texts.length > 0 && (
            <Group title={t.ads.plan.texts}>
              {texts.map((x) => (
                <Pick
                  key={x.headline}
                  head={x.headline}
                  body={x.body}
                  why={x.why}
                  chosen={text?.headline === x.headline}
                  onPick={() => setText(x)}
                  choose={t.ads.plan.choose}
                  chosenLabel={t.ads.plan.chosen}
                />
              ))}
            </Group>
          )}

          {/* ---- What the owner chose ---- */}
          {(dish || area || budget || text) && (
            <div className="rounded-xl bg-ink/5 p-3">
              <p className="text-sm font-semibold">{t.ads.plan.picked}</p>
              <ul className="mt-1 space-y-0.5 text-sm text-ink-soft">
                {dish && <li>{dish.name}</li>}
                {area && (
                  <li>
                    {area.label}
                    {area.radiusKm ? ` · ${area.radiusKm} km` : ""}
                  </li>
                )}
                {budget && (
                  <li>
                    {som(budget.daily)} {t.ads.plan.perDay} · {budget.days}{" "}
                    {t.ads.plan.days}
                  </li>
                )}
                {text && <li>{text.headline}</li>}
              </ul>
              {text && (
                <button
                  type="button"
                  onClick={copyText}
                  className="btn btn-dark mt-3 disabled:opacity-40"
                >
                  {copied ? t.ads.plan.copied : t.ads.plan.copy}
                </button>
              )}
              {/* ⚠️ Said only while the connection cannot carry a campaign.
                  This is the moment the owner has a plan in hand and looks for
                  the button that runs it — leaving them to find out there is
                  none is how a working feature reads as broken. Once the
                  account is connected the button is right below, and this
                  sentence would be a lie. */}
              {!state?.ready && (
                <p className="mt-2 max-w-2xl text-xs text-ink-muted">
                  {t.ads.plan.handNote}
                </p>
              )}
            </div>
          )}
        </div>
      )}

      {/* ---- From the plan to a campaign ---- */}
      {state?.ready && (
        <AdsCreate
          t={t}
          settings={state.settings}
          dish={dish}
          area={area}
          budget={budget}
          text={text}
          onCreated={reload}
        />
      )}

      {state?.connected && <AdsCampaigns t={t} list={list} reload={reload} />}
      {state?.connected && <AdsReportCard t={t} report={report} />}
      {state?.ready && (
        <AdsRules t={t} settings={state.settings} reload={reload} />
      )}
    </div>
  );
}

function Group({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-2">
      <p className="text-sm font-semibold">{title}</p>
      <div className="grid gap-2 sm:grid-cols-3">{children}</div>
    </div>
  );
}

// One proposal, with the reason it was proposed.
//
// ⚠️ **The reason is not a tooltip.** It is the whole basis on which the owner
// is choosing, and a choice made without it is a guess with extra steps.
function Pick({
  head,
  body,
  why,
  chosen,
  onPick,
  choose,
  chosenLabel,
}: {
  head: string;
  body?: string;
  why: string;
  chosen: boolean;
  onPick: () => void;
  choose: string;
  chosenLabel: string;
}) {
  return (
    <button
      type="button"
      onClick={onPick}
      aria-pressed={chosen}
      className={`rounded-xl border p-3 text-left transition-colors ${
        chosen
          ? "border-brand bg-brand-tint"
          : "border-ink/10 hover:border-brand/40"
      }`}
    >
      <p className="text-sm font-semibold">{head}</p>
      {body && <p className="mt-1 text-sm text-ink-soft">{body}</p>}
      <p className="mt-1 text-xs text-ink-muted">{why}</p>
      <span className="mt-2 inline-block text-xs font-semibold text-brand-dark">
        {chosen ? chosenLabel : choose}
      </span>
    </button>
  );
}
