"use client";

// The week this kitchen actually had, and the four choices drawn from it.
//
// ⚠️ **The figures are the server's arithmetic; the words beside them are the
// model's.** Every percentage here comes out of `facts`, which the restaurant's
// own server computed — so a model that got carried away can write a poor
// reason but cannot move a number.
//
// ⚠️ **A bar only where a bar says something.** The first version drew a
// full-width bar for every dish, which on a real week meant one bar at 99% and
// four empty troughs labelled "0%" — four rows of furniture carrying no
// information, and a dish that sold 45 000 so'm reading as nothing at all. The
// money is the figure that always means something, so it leads; the share sits
// beside it and the bar is a narrow gauge rather than the width of the screen.
//
// ⚠️ **Each group says what has been chosen from it.** Four groups of three
// cards is twelve near-identical boxes; without a "chosen: Osh" line the owner
// has to remember what they clicked two screens up, and the launch button below
// refuses for a reason they cannot see.

import type { AdminDict } from "@/lib/i18n/admin";
import type {
  AdsAreaPick,
  AdsBudgetPick,
  AdsDishPick,
  AdsFacts,
  AdsTextPick,
} from "@/lib/types";

function som(n: number): string {
  return new Intl.NumberFormat("uz-UZ").format(Math.round(n));
}

/** What the week looked like. */
export function PlanFacts({
  t,
  facts,
  asOf,
  cached,
}: {
  t: AdminDict;
  facts: AdsFacts;
  asOf?: string;
  cached?: boolean;
}) {
  if (facts.dishes.length === 0) return null;
  return (
    <div className="space-y-3">
      <div className="rounded-xl bg-ink/5 px-3 py-2">
        <p className="text-xs text-ink-muted">{t.ads.plan.weekTotal}</p>
        <p className="text-lg font-semibold tabular-nums">
          {som(facts.weekTotal)}{" "}
          <span className="text-sm font-normal text-ink-soft">
            {facts.currency}
          </span>
        </p>
      </div>

      <ul className="divide-y divide-ink/5">
        {facts.dishes.map((d) => (
          <li key={d.name} className="flex items-center gap-3 py-1.5">
            <span className="min-w-0 flex-1 truncate text-sm font-medium">
              {d.name}
            </span>
            {/* The gauge is fixed and narrow: it compares, it does not
                decorate. A dish under one per cent gets a visible sliver
                rather than an empty trough. */}
            <span
              aria-hidden
              className="hidden h-1.5 w-24 shrink-0 overflow-hidden rounded-full bg-ink/10 sm:block"
            >
              <span
                className="block h-full rounded-full bg-brand"
                style={{ width: `${Math.max(2, Math.min(100, d.share))}%` }}
              />
            </span>
            <span className="w-10 shrink-0 text-right text-xs tabular-nums text-ink-muted">
              {d.share < 1 ? "<1%" : `${d.share}%`}
            </span>
            <span className="w-32 shrink-0 text-right text-sm tabular-nums">
              {som(d.money)}
            </span>
            <span className="hidden w-32 shrink-0 text-right text-xs tabular-nums text-ink-muted sm:block">
              {t.ads.plan.lastWeek}: {som(d.lastWeek)}
            </span>
          </li>
        ))}
      </ul>

      {asOf && (
        <p className="text-xs text-ink-muted">
          {t.ads.plan.asOf}: {new Date(asOf).toLocaleString()}
          {cached ? ` · ${t.ads.plan.cached}` : ""}
        </p>
      )}
    </div>
  );
}

/** One group of proposals, with what was taken from it. */
export function PickGroup({
  title,
  chosen,
  none,
  children,
}: {
  title: string;
  chosen?: string;
  none: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-baseline justify-between gap-x-3">
        <h3 className="text-sm font-semibold">{title}</h3>
        <p
          className={`text-xs ${
            chosen ? "font-medium text-brand-dark" : "text-ink-muted"
          }`}
        >
          {chosen ?? none}
        </p>
      </div>
      <div className="grid gap-2 sm:grid-cols-3">{children}</div>
    </div>
  );
}

// One proposal, with the reason it was proposed.
//
// ⚠️ **The reason is not a tooltip.** It is the whole basis on which the owner
// is choosing, and a choice made without it is a guess with extra steps.
//
// ⚠️ **The whole card is the control.** "Tanlash" used to be a blue word in the
// corner — the same weight as a footnote, on the one element of this page that
// is a decision. Now the card is the button and the mark says which one is
// taken.
export function Pick({
  head,
  body,
  why,
  chosen,
  onPick,
  chosenLabel,
}: {
  head: string;
  body?: string;
  why: string;
  chosen: boolean;
  onPick: () => void;
  chosenLabel: string;
}) {
  return (
    <button
      type="button"
      onClick={onPick}
      aria-pressed={chosen}
      className={`flex h-full flex-col rounded-xl border p-3 text-left transition-colors ${
        chosen
          ? "border-brand bg-brand-tint ring-1 ring-brand"
          : "border-ink/10 hover:border-brand/40 hover:bg-ink/[0.02]"
      }`}
    >
      <span className="flex items-start gap-2">
        <span
          aria-hidden
          className={`mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border text-[10px] leading-none ${
            chosen
              ? "border-brand bg-brand text-white"
              : "border-ink/25 text-transparent"
          }`}
        >
          ✓
        </span>
        <span className="text-sm font-semibold">{head}</span>
      </span>
      {body && <span className="mt-1 text-sm text-ink-soft">{body}</span>}
      {/* Pushed to the bottom so a row of cards ends on one line however long
          the reasons are — ragged card bottoms read as a broken grid. */}
      <span className="mt-2 flex-1 text-xs text-ink-muted">{why}</span>
      {chosen && (
        <span className="mt-2 text-xs font-semibold text-brand-dark">
          {chosenLabel}
        </span>
      )}
    </button>
  );
}

export type Picks = {
  dish: AdsDishPick | null;
  area: AdsAreaPick | null;
  budget: AdsBudgetPick | null;
  text: AdsTextPick | null;
};
