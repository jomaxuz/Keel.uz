"use client";

// Advertising: Meta campaigns, decided from this restaurant's own figures.
//
// ⚠️ **The money is the restaurant's and goes straight to Meta.** The ad account
// is theirs and the card on it is theirs; what is sold here is the work around
// it. The screen says so in a line, because an owner who suspects we spent their
// money stops trusting every other number on the panel — and that suspicion is
// impossible to argue with after the fact.
//
// ⚠️ **Four stages in the order they happen, not four cards in a column.** Work
// out what to advertise, connect the account that will pay for it, launch, then
// read what it brought back. The first version stacked these as equals, which
// put a live "create a campaign" card above a plan that did not exist yet —
// saying only "choose a dish first" to somebody who had nothing to choose from.
//
// ⚠️ **The plan comes first because it is the half that works without Meta.**
// Deciding what to advertise needs only this kitchen's own week; connecting an
// account is what the *launch* needs. Leading with the connection would put a
// setup chore in front of the thing the section is bought for.
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
import { CustomRow, MiniField, PickGroup, Pick, PlanFacts } from "./plan";
import AdsReportCard from "./report";
import AdsRules from "./rules";
import Step from "./step";

function som(n: number): string {
  return new Intl.NumberFormat("uz-UZ").format(Math.round(n));
}

export default function AdminAdsPage() {
  const t = useAdminT();
  const [state, setState] = useState<AdsState | null>(null);
  const [error, setError] = useState("");

  const [answer, setAnswer] = useState<AdsPlanAnswer | null>(null);
  const [building, setBuilding] = useState(false);

  // The owner's own choices. ⚠️ Held on the screen and nowhere else, on
  // purpose: they exist only until the campaign is created, and a "saved"
  // choice that no campaign ever reads is a promise the product does not keep.
  const [dish, setDish] = useState<AdsDishPick | null>(null);
  const [area, setArea] = useState<AdsAreaPick | null>(null);
  const [budget, setBudget] = useState<AdsBudgetPick | null>(null);
  const [text, setText] = useState<AdsTextPick | null>(null);

  // What the owner typed instead of taking a proposal. ⚠️ Held beside the
  // picks rather than inside them: clearing the field has to give the three
  // cards back, and a custom value written over `area` would have nothing to
  // go back to.
  const [ownKm, setOwnKm] = useState("");
  const [ownDaily, setOwnDaily] = useState("");
  const [ownDays, setOwnDays] = useState("");

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
  const hasPlan = dishes.length > 0;

  // ⚠️ **Only the wordings written for the chosen dish.** A wording names the
  // dish in its own text, so offering the soup's line for an osh campaign is
  // offering an advert for food the owner has just decided not to advertise.
  // A plan cached before wordings carried a dish has none of them, and there
  // the honest fallback is to show all rather than none.
  const filed = texts.some((x) => x.dish);
  const dishTexts = filed
    ? texts.filter((x) => !dish || x.dish === dish.name)
    : texts;
  // A plan written before wordings carried a dish, or one where the model
  // filed none under the dish that was chosen. ⚠️ Said rather than silently
  // showing somebody else's wording.
  const textsAreStale = filed && dish !== null && dishTexts.length === 0;

  const campaigns = list?.campaigns ?? [];
  const ready = Boolean(state?.ready);

  if (state && !state.on) {
    return (
      <Head t={t}>
        <div className="card space-y-2 p-4">
          <p className="font-semibold">{t.ads.offTitle}</p>
          <p className="text-sm text-ink-soft">{t.ads.off}</p>
        </div>
      </Head>
    );
  }

  return (
    <Head t={t}>
      {error && <p className="text-sm text-danger">{error}</p>}

      {/* ---- ① What to advertise ---- */}
      <Step
        n={1}
        title={t.ads.plan.title}
        lead={t.ads.plan.lead}
        state={hasPlan ? "done" : "now"}
        aside={
          <button
            type="button"
            onClick={build}
            disabled={building}
            className="btn btn-primary disabled:opacity-40"
          >
            {building
              ? t.ads.plan.building
              : hasPlan
                ? t.ads.plan.rebuild
                : t.ads.plan.build}
          </button>
        }
      >
        <div className="space-y-4">
          {answer?.off && <p className="text-sm text-ink-soft">{t.ads.off}</p>}
          {answer?.capped && (
            <p className="text-sm text-ink-soft">{t.ads.plan.capped}</p>
          )}
          {answer?.error && (
            <p className="text-sm text-danger">{answer.error}</p>
          )}

          {!answer && !building && (
            <p className="max-w-2xl text-sm text-ink-muted">
              {t.ads.plan.empty}
            </p>
          )}

          {facts && (
            <PlanFacts
              t={t}
              facts={facts}
              asOf={answer?.asOf}
              cached={answer?.cached}
            />
          )}

          {hasPlan && (
            <div className="space-y-4 border-t border-ink/5 pt-4">
              <PickGroup
                title={t.ads.plan.dishes}
                chosen={dish?.name}
                none={t.ads.plan.notChosen}
              >
                {dishes.map((d) => (
                  <Pick
                    key={d.name}
                    head={d.name}
                    why={d.why}
                    chosen={dish?.name === d.name}
                    note={
                      facts?.dishes.find((f) => f.name === d.name)?.photo ===
                      false
                        ? t.ads.plan.noPhoto
                        : undefined
                    }
                    onPick={() => {
                      setDish(d);
                      // ⚠️ The wording belonged to the dish that was chosen
                      // before; keeping it would launch an advert for the old
                      // one under the new one's name.
                      setText(null);
                    }}
                    chosenLabel={t.ads.plan.chosen}
                  />
                ))}
              </PickGroup>

              {areas.length > 0 && (
                <PickGroup
                  title={t.ads.plan.areas}
                  chosen={area?.label}
                  none={t.ads.plan.notChosen}
                >
                  {areas.map((a) => (
                    <Pick
                      key={a.label}
                      head={
                        a.radiusKm ? `${a.label} · ${a.radiusKm} km` : a.label
                      }
                      why={a.why}
                      chosen={area?.label === a.label}
                      onPick={() => setArea(a)}
                      chosenLabel={t.ads.plan.chosen}
                    />
                  ))}
                  <div className="sm:col-span-3">
                    <CustomRow
                      label={t.ads.plan.ownArea}
                      on={Boolean(ownKm) && area?.label === t.ads.plan.ownArea}
                      onClear={() => {
                        setOwnKm("");
                        setArea(null);
                      }}
                    >
                      <MiniField
                        value={ownKm}
                        suffix="km"
                        width="w-20"
                        onChange={(v) => {
                          setOwnKm(v);
                          const km = Number(v);
                          setArea(
                            km > 0
                              ? {
                                  label: t.ads.plan.ownArea,
                                  radiusKm: km,
                                  why: t.ads.plan.ownWhy,
                                }
                              : null,
                          );
                        }}
                      />
                    </CustomRow>
                  </div>
                </PickGroup>
              )}

              {budgets.length > 0 && (
                <PickGroup
                  title={t.ads.plan.budgets}
                  chosen={
                    budget
                      ? `${som(budget.daily)} ${t.ads.plan.perDay}`
                      : undefined
                  }
                  none={t.ads.plan.notChosen}
                >
                  {budgets.map((b) => (
                    <Pick
                      key={`${b.daily}-${b.days}`}
                      head={`${som(b.daily)} ${t.ads.plan.perDay}`}
                      body={`${b.days} ${t.ads.plan.days}`}
                      why={b.why}
                      chosen={
                        budget?.daily === b.daily && budget?.days === b.days
                      }
                      onPick={() => setBudget(b)}
                      chosenLabel={t.ads.plan.chosen}
                    />
                  ))}
                  <div className="sm:col-span-3">
                    <CustomRow
                      label={t.ads.plan.ownBudget}
                      on={
                        Boolean(ownDaily) &&
                        budget?.why === t.ads.plan.ownWhy
                      }
                      onClear={() => {
                        setOwnDaily("");
                        setOwnDays("");
                        setBudget(null);
                      }}
                    >
                      <MiniField
                        value={ownDaily}
                        suffix={t.ads.plan.perDay}
                        onChange={(v) => {
                          setOwnDaily(v);
                          const daily = Number(v);
                          setBudget(
                            daily > 0
                              ? {
                                  daily,
                                  days: Number(ownDays) || 7,
                                  why: t.ads.plan.ownWhy,
                                }
                              : null,
                          );
                        }}
                      />
                      <MiniField
                        value={ownDays}
                        suffix={t.ads.plan.days}
                        width="w-16"
                        onChange={(v) => {
                          setOwnDays(v);
                          const daily = Number(ownDaily);
                          if (daily > 0) {
                            setBudget({
                              daily,
                              days: Number(v) || 7,
                              why: t.ads.plan.ownWhy,
                            });
                          }
                        }}
                      />
                    </CustomRow>
                  </div>
                </PickGroup>
              )}

              {textsAreStale && (
                <p className="rounded-xl bg-ink/5 p-3 text-sm text-ink-soft">
                  {t.ads.plan.textsStale}
                </p>
              )}

              {dishTexts.length > 0 && (
                <PickGroup
                  title={t.ads.plan.texts}
                  chosen={text?.headline}
                  none={dish ? t.ads.plan.notChosen : t.ads.plan.pickDishFirst}
                >
                  {dishTexts.map((x) => (
                    <Pick
                      key={x.headline}
                      head={x.headline}
                      body={x.body}
                      why={x.why}
                      chosen={text?.headline === x.headline}
                      onPick={() => setText(x)}
                      chosenLabel={t.ads.plan.chosen}
                    />
                  ))}
                </PickGroup>
              )}
            </div>
          )}
        </div>
      </Step>

      {/* ---- ② The account that pays ---- */}
      {state && (
        <AdsConnect
          t={t}
          state={state}
          reload={reload}
          step={ready ? "done" : "now"}
        />
      )}

      {/* ---- ③ Launch ---- */}
      <Step
        n={3}
        title={t.ads.campaign.title}
        lead={t.ads.campaign.lead}
        // ⚠️ **Open as soon as the account is connected, not once the plan has
        // been picked from.** The other way of making an advert — putting money
        // behind a post the restaurant already published — needs no plan at
        // all, and gating the whole stage on the plan hid it from exactly the
        // restaurants that post their food every day.
        state={ready ? "now" : "later"}
        waiting={t.ads.campaign.waitingConnect}
      >
        {state && ready && (
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
      </Step>

      {/* ---- ④ What came back ---- */}
      <Step
        n={4}
        title={t.ads.report.title}
        lead={t.ads.report.lead}
        state={campaigns.length > 0 ? "now" : "later"}
        waiting={t.ads.report.waiting}
      >
        <div className="space-y-4">
          <AdsReportCard t={t} report={report} />
          <AdsCampaigns t={t} list={list} reload={reload} />
          {state && ready && <AdsRules t={t} settings={state.settings} reload={reload} />}
        </div>
      </Step>
    </Head>
  );
}

function Head({
  t,
  children,
}: {
  t: ReturnType<typeof useAdminT>;
  children: React.ReactNode;
}) {
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
      {children}
    </div>
  );
}
