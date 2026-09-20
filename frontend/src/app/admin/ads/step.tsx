"use client";

// One stage of the advertising section, drawn as a stage.
//
// ⚠️ **The section is a pipeline and it has to look like one.** Connect, plan,
// launch, measure: four things that happen in an order, where the second is
// worthless without the first. Drawn as four equal cards stacked down a page,
// an owner meets them as four unrelated settings screens — and the first
// version of this page did exactly that, with a live "create a campaign" card
// sitting above a plan that did not exist yet, saying only "choose a dish
// first".
//
// ⚠️ **A stage that cannot be acted on yet shows what it will do, never its
// controls.** Disabled inputs read as broken ones; a sentence reads as a
// promise. That is the whole difference between a page that explains itself
// and a page somebody has to be taught.

import type { ReactNode } from "react";

export type StepState = "done" | "now" | "later";

export default function Step({
  n,
  title,
  lead,
  state,
  /** One line shown instead of the body while the stage is still out of
   *  reach — what will happen here, not why it is disabled. */
  waiting,
  aside,
  children,
}: {
  n: number;
  title: string;
  lead?: string;
  state: StepState;
  waiting?: string;
  aside?: ReactNode;
  children?: ReactNode;
}) {
  const later = state === "later";
  return (
    <section
      className={`card p-4 transition-opacity sm:p-5 ${
        later ? "opacity-60" : ""
      }`}
      aria-current={state === "now" ? "step" : undefined}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          {/* The number is the whole navigation aid: it says there is an order,
              and which part of it you are looking at. */}
          <span
            aria-hidden
            className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-sm font-semibold ${
              state === "done"
                ? "bg-brand text-white"
                : state === "now"
                  ? "bg-brand-tint text-brand-dark ring-2 ring-brand"
                  : "bg-ink/10 text-ink-muted"
            }`}
          >
            {state === "done" ? "✓" : n}
          </span>
          <div className="min-w-0">
            <h2 className="text-base font-semibold">{title}</h2>
            {lead && (
              <p className="mt-0.5 max-w-2xl text-sm text-ink-soft">{lead}</p>
            )}
          </div>
        </div>
        {aside && <div className="shrink-0">{aside}</div>}
      </div>

      <div className="mt-3 sm:pl-10">
        {later ? (
          waiting && <p className="text-sm text-ink-muted">{waiting}</p>
        ) : (
          children
        )}
      </div>
    </section>
  );
}

/** A row of facts that belong on one line — the shape a summary wants when the
 *  stage behind it is finished and folded away. */
export function Summary({ items }: { items: (string | undefined)[] }) {
  const shown = items.filter(Boolean) as string[];
  if (shown.length === 0) return null;
  return (
    <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
      {shown.map((s, i) => (
        <span key={i} className="flex items-center gap-2">
          {i > 0 && <span className="text-ink-muted">·</span>}
          <span>{s}</span>
        </span>
      ))}
    </p>
  );
}
