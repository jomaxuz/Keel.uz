"use client";

import { useMemo, useState } from "react";
import { useT } from "@/lib/i18n/client";
import { TELEGRAM } from "@/lib/links";

/** What a month actually costs, on one screen.
 *
 *  ⚠️ **Both halves of the bill, not the till alone.** The page charges for two
 *  different things in two different shapes — a monthly rate per branch for the
 *  counter, and a per-order rate for online orders — and they are presented in
 *  two sections a long way apart. A visitor who reads only one of them is doing
 *  the wrong arithmetic, and the number they arrive at is the one they compare
 *  with a competitor's. This block is the only place the two are added up.
 *
 *  ⚠️ **Every price is read from the dictionary the tables above are drawn
 *  from**, never re-typed here. A calculator that quietly disagrees with the
 *  table three screens up is worse than no calculator: it is the page arguing
 *  with itself in front of the customer, and the one number they would remember
 *  is whichever is higher. */

/** Order bands. The thresholds are logic — the prose beside them lives in the
 *  dictionary, the rate comes from `pricing.tiers[i]`. The band applies to the
 *  whole month's volume, which is what `tiersNote` promises. */
const BANDS = [3_000, 15_000, 50_000, Infinity];

/** The warehouse module: one charge for the whole company, not per branch, and
 *  already inside Pro. Stated in `till.addonDesc` in prose only, which is why
 *  it is the one figure on this screen written as a number. */
const STOCK_ADDON = 290_000;

/** Branch pricing: full for the first, −30% from the second, −40% from the
 *  fifth (`till.chainDesc`). */
function branchFactor(n: number) {
  let total = 1;
  for (let i = 2; i <= n; i++) total += i >= 5 ? 0.6 : 0.7;
  return total;
}

const ORDER_STEPS = [
  0, 100, 300, 600, 1_000, 1_500, 2_000, 3_000, 5_000, 8_000, 12_000, 15_000,
  20_000, 30_000, 50_000,
];

function money(n: number) {
  return String(Math.round(n)).replace(/\B(?=(\d{3})+(?!\d))/g, " ");
}

/** "450 000" and "2 500 000 dan" and "800 so'm" all mean one number here. */
function num(s: string) {
  return Number(s.replace(/[^\d]/g, "")) || 0;
}

export default function Calculator() {
  const { t } = useT();
  const c = t.calc;

  // ⚠️ Enterprise is deliberately not offered: its price is negotiated, so the
  // honest output for it is a sentence rather than a total, and a select whose
  // last option empties the answer is a broken calculator. The note under the
  // result says where that conversation goes instead.
  // ⚠️ **Which ladder the visitor is pricing.** The hero above now names two
  // businesses and two entry prices; a calculator that silently answers with
  // restaurant rungs would quote a grocery three times its real bill and lose
  // it at the only screen where it was doing arithmetic about us.
  const [shop, setShop] = useState(false);
  const plans = (shop ? t.till.shopPlans : t.till.plans).slice(0, 3);
  const [plan, setPlan] = useState(1); // Standard: the middle of the table.
  const [branches, setBranches] = useState(1);
  const [stock, setStock] = useState(false);
  const [step, setStep] = useState(7); // 3 000 orders — the first band edge.

  const orders = ORDER_STEPS[step];

  const sum = useMemo(() => {
    const till = plan < 0 ? 0 : num(plans[plan].price) * branchFactor(branches);
    // Pro carries the warehouse module; below it, only if asked for.
    //
    // ⚠️ **Every shop rung carries it**, which is the one place the two
    // ladders differ in kind rather than in price: what a shop sells is what is
    // on its shelf, so a shop till that cannot count is not a cheaper product,
    // it is a different one. Charging the add-on here would quote 149 000 plus
    // 290 000 for something already included.
    const stockIncluded = shop ? plan >= 0 : plan === 2;
    const stockCost = stockIncluded || !stock ? 0 : STOCK_ADDON;
    const band = BANDS.findIndex((max) => orders <= max);
    const rate = num(t.pricing.tiers[band]?.price ?? t.pricing.tiers[0].price);
    return {
      till,
      stockIncluded,
      stockCost,
      rate,
      ordersCost: orders * rate,
      total: till + stockCost + orders * rate,
    };
  }, [plan, plans, shop, branches, stock, orders, t.pricing.tiers]);

  return (
    <div className="grid items-start gap-6 lg:grid-cols-[1.05fr_.95fr]">
      {/* ---- The knobs ---- */}
      <div className="rounded-3xl border border-line bg-surface p-6 sm:p-8">
        {/* ⚠️ **The kind of business first**, because it changes every number
            under it. Two tabs rather than a select: there are exactly two
            ladders and both fit on a line. */}
        <fieldset className="mb-7">
          <legend className="text-sm font-semibold text-ink">{c.kind}</legend>
          <div className="mt-3 grid grid-cols-2 gap-2">
            {[
              { label: c.kindRestaurant, on: !shop },
              { label: c.kindShop, on: shop },
            ].map((k) => (
              <button
                key={k.label}
                type="button"
                onClick={() => setShop(k.label === c.kindShop)}
                aria-pressed={k.on}
                className={`rounded-xl border px-4 py-3 text-center font-display text-sm font-semibold transition ${
                  k.on
                    ? "border-signal-500 bg-signal-500/10 text-ink"
                    : "border-line text-ink-muted hover:border-line-strong"
                }`}
              >
                {k.label}
              </button>
            ))}
          </div>
        </fieldset>

        <fieldset>
          <legend className="text-sm font-semibold text-ink">{c.plan}</legend>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            {[{ name: c.planNone, registers: c.planNoneNote, i: -1 }].concat(
              plans.map((p, i) => ({ name: p.name, registers: p.registers, i })),
            ).map((p) => (
              <button
                key={p.name}
                type="button"
                onClick={() => setPlan(p.i)}
                aria-pressed={plan === p.i}
                className={`rounded-xl border px-4 py-3 text-left transition ${
                  plan === p.i
                    ? "border-signal-500 bg-signal-500/10"
                    : "border-line hover:border-line-strong"
                }`}
              >
                <span className="block font-display text-sm font-semibold text-ink">
                  {p.name}
                </span>
                <span className="block text-xs text-ink-muted">{p.registers}</span>
              </button>
            ))}
          </div>
        </fieldset>

        {/* Branches. A stepper rather than a slider: this is a number an owner
            knows exactly, and nobody drags to find out they have three. */}
        <div className="mt-7 flex items-center justify-between gap-4">
          <div>
            <p className="text-sm font-semibold text-ink">{c.branches}</p>
            <p className="mt-0.5 text-xs text-ink-muted">{c.branchesNote}</p>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <Step label="−" onClick={() => setBranches((n) => Math.max(1, n - 1))} />
            <span className="w-10 text-center font-display text-lg tabular-nums text-ink">
              {branches}
            </span>
            <Step label="+" onClick={() => setBranches((n) => Math.min(30, n + 1))} />
          </div>
        </div>

        <div className="mt-7">
          <div className="flex items-baseline justify-between gap-4">
            <p className="text-sm font-semibold text-ink">{c.orders}</p>
            <p className="font-display text-lg tabular-nums text-ink">{money(orders)}</p>
          </div>
          {/* Stepped rather than continuous: the steps land on the band edges,
              so dragging shows the rate dropping at exactly the volumes the
              table above names. A linear 0–50 000 slider spends nine tenths of
              its travel in a band almost nobody is in. */}
          <input
            type="range"
            min={0}
            max={ORDER_STEPS.length - 1}
            value={step}
            onChange={(e) => setStep(Number(e.target.value))}
            aria-label={c.orders}
            className="mt-3 w-full accent-signal-500"
          />
          <p className="mt-2 text-xs text-ink-muted">{c.ordersNote}</p>
        </div>

        <label
          className={`mt-7 flex items-start gap-3 rounded-xl border p-4 ${
            sum.stockIncluded ? "border-line bg-raised" : "border-line-strong"
          }`}
        >
          <input
            type="checkbox"
            checked={sum.stockIncluded || stock}
            disabled={sum.stockIncluded}
            onChange={(e) => setStock(e.target.checked)}
            className="mt-0.5 h-4 w-4 shrink-0 accent-signal-500"
          />
          <span className="min-w-0">
            <span className="block text-sm font-semibold text-ink">{c.stock}</span>
            <span className="mt-0.5 block text-xs text-ink-muted">
              {sum.stockIncluded ? c.stockIncluded : c.stockNote}
            </span>
          </span>
        </label>
      </div>

      {/* ---- The answer ---- */}
      {/* ⚠️ `self-start` and sticky, not stretched to the knobs' height. Filled
          to match, this card has an empty third in it wherever the button is
          put — at the bottom the gap lands above it, in the flow it lands
          below, and either way it reads as a panel still being built. Sized to
          its own content it is simply the answer; sticking it means the answer
          stays on screen while the knobs it belongs to are being turned, which
          is the whole point of a calculator. */}
      <div className="relative self-start overflow-hidden rounded-3xl border border-line-strong bg-surface p-6 sm:p-8 lg:sticky lg:top-24">
        <div className="pointer-events-none absolute -right-16 -top-16 h-52 w-52 rounded-full bg-signal-500/10 blur-2xl" />
        <div className="relative">
          <p className="eyebrow">{c.total}</p>
          <p className="h-display mt-3 text-4xl leading-none tabular-nums sm:text-5xl">
            {money(sum.total)}{" "}
            <span className="text-2xl font-semibold text-ink-soft sm:text-3xl">
              {t.pricing.unit}
            </span>
          </p>

          <dl className="mt-7 space-y-3 border-t border-line pt-6 text-sm">
            <Row
              k={c.tillLine}
              v={plan < 0 ? "—" : `${money(sum.till)} ${t.pricing.unit}`}
            />
            {sum.stockCost > 0 && (
              <Row k={c.stockLine} v={`${money(sum.stockCost)} ${t.pricing.unit}`} />
            )}
            <Row
              k={`${c.ordersLine} · ${money(sum.rate)} ${t.pricing.unit}`}
              v={`${money(sum.ordersCost)} ${t.pricing.unit}`}
            />
          </dl>

          {branches > 1 && plan >= 0 && (
            <p className="mt-5 inline-flex items-center gap-1.5 rounded-lg bg-emerald-500/10 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-400">
              {c.discount}
            </p>
          )}

          <a href={TELEGRAM} className="btn-primary mt-7 w-full px-6 py-3.5 text-base">
            {c.cta}
          </a>
          <p className="mt-3 text-xs leading-relaxed text-ink-muted">{c.note}</p>
        </div>
      </div>
    </div>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <dt className="text-ink-soft">{k}</dt>
      <dd className="shrink-0 tabular-nums font-semibold text-ink">{v}</dd>
    </div>
  );
}

function Step({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      className="grid h-9 w-9 place-items-center rounded-xl border border-line-strong text-lg text-ink transition hover:bg-raised"
    >
      {label}
    </button>
  );
}
