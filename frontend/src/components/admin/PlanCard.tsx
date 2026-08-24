"use client";

// What this restaurant is on, what it costs, and when the next payment is due.
//
// ⚠️ **The screen an owner opens right before paying us**, which is why every
// number on it is printed rather than calculated. Branch discounts, add-ons and
// any negotiated price are already applied by the console; a panel that
// multiplied a rung's list price by a branch count would quote a figure the
// invoice disagrees with, to the one person who checks.
//
// ⚠️ **A date, not a flag.** "Obuna faol" goes stale at midnight with nobody
// watching — the lesson this codebase records under provisionStatus,
// lastEventAt and lastUpdateAt. So the card shows the date and counts to it,
// and the countdown is the thing that changes colour.
//
// ⚠️ **It never blocks and never nags.** The subscription corner on the till
// already warns in the last week; this is the reference card, and a second
// escalating warning in the panel would teach an owner to skip both.

import { MODULE_LABEL, PLAN_LABEL, money } from "@/components/admin/UpgradeCta";
import { TELEGRAM } from "@/lib/links";
import { useAdminT } from "@/lib/i18n/admin";
import { useSubscription } from "@/lib/subscription";

/** Whole days from today to a "YYYY-MM-DD" the server already localised.
 *
 *  ⚠️ Both sides are pinned to midnight UTC from the date parts, so this is a
 *  difference of calendar days and never of hours — a countdown that ticks over
 *  at four in the afternoon because that is when somebody paid is a countdown
 *  nobody believes. The string is already local (the server formats it), so no
 *  timezone conversion belongs here either. */
function daysUntil(date: string): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!m) return null;
  const end = Date.UTC(+m[1], +m[2] - 1, +m[3]);
  const now = new Date();
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((end - today) / 86_400_000);
}

export default function PlanCard() {
  const t = useAdminT();
  const { sub } = useSubscription();

  // Still in flight. Nothing rather than a skeleton: this card sits under the
  // password form and a box that flashes in is a box that draws the eye away
  // from the thing the page is actually for.
  if (!sub) return null;

  // ⚠️ **No plan is an ordinary state, not an error.** A restaurant paying per
  // order for its website has no counter and never will, and telling that
  // owner something is wrong would be false. It says what a subscription would
  // open and stops there.
  if (!sub.enabled || !sub.plan) {
    return (
      <section className="mt-6 rounded-2xl border border-line bg-surface p-6 shadow-card">
        <h2 className="font-semibold text-ink">{t.account.planTitle}</h2>
        <p className="mt-2 text-sm text-ink-soft">{t.account.planNone}</p>
        <p className="mt-1 text-sm text-ink-muted">{t.account.planNoneHint}</p>
        <TelegramLink label={t.account.planChangeCta} />
      </section>
    );
  }

  const planName = PLAN_LABEL[sub.plan] ?? sub.plan;
  const left = sub.paidUntil ? daysUntil(sub.paidUntil) : null;

  // The countdown, in the words an owner would use. ⚠️ Overdue is stated in
  // days rather than as "expired": how far past it is decides whether this is
  // a note to self or a phone call this morning.
  let due = t.account.planNoDate;
  let tone = "text-ink-muted";
  if (left !== null) {
    if (left < 0) {
      due = t.account.planOverdue(-left);
      tone = "text-rose-600 dark:text-rose-400";
    } else if (left === 0) {
      due = t.account.planDueToday;
      tone = "text-rose-600 dark:text-rose-400";
    } else {
      due = t.account.planDaysLeft(left);
      tone = left <= 7 ? "text-amber-600 dark:text-amber-400" : "text-ink-muted";
    }
  }

  // ⚠️ 0 is "agreed separately", never "free". An Enterprise price is settled
  // per customer and the console deliberately mirrors nothing rather than
  // invent one — printing "0 so'm" would be a quote we never gave.
  const hasPrice = (sub.monthly ?? 0) > 0;

  return (
    <section className="mt-6 rounded-2xl border border-line bg-surface p-6 shadow-card">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="font-semibold text-ink">{t.account.planTitle}</h2>
          <p className="mt-1 text-xl font-bold text-ink">{planName}</p>
          <p className="mt-1 text-sm text-ink-muted">
            {t.account.planRegisters(sub.registers ?? 0)}
            {(sub.branches ?? 0) > 1 && (
              <> · {t.account.planBranches(sub.branches ?? 0)}</>
            )}
          </p>
        </div>

        <div className="text-right">
          <p className="text-xs uppercase tracking-wider text-ink-muted">
            {t.account.planMonthly}
          </p>
          {hasPrice ? (
            <p className="text-xl font-bold text-ink">
              {money(sub.monthly ?? 0)}{" "}
              <span className="text-sm font-medium text-ink-muted">so&apos;m</span>
            </p>
          ) : (
            <p className="mt-1 text-sm text-ink-soft">
              {t.account.planIndividual}
            </p>
          )}
        </div>
      </div>

      {/* The two halves of "when do I pay": the date it runs to, and how far
          away that is. The date alone makes the owner do the arithmetic; the
          countdown alone cannot be written on a calendar. */}
      <div className="mt-5 flex flex-wrap items-baseline justify-between gap-3 rounded-xl border border-line bg-raised px-4 py-3">
        <span className="text-sm text-ink-soft">
          {t.account.planNextPayment}
        </span>
        <span className="text-sm">
          {sub.paidUntil && (
            <span className="font-semibold text-ink">{sub.paidUntil}</span>
          )}{" "}
          <span className={tone}>{due}</span>
        </span>
      </div>

      {/* ⚠️ Under the date rather than beside the price. The question this
          answers ("how do I move up, or drop the add-on") is asked *after*
          reading what is currently paid for, and a link level with the figure
          reads as a way to change the figure itself. */}
      <div className="mt-4 flex flex-wrap items-baseline justify-between gap-2">
        <span className="text-sm text-ink-soft">{t.account.planChange}</span>
        <TelegramLink label={t.account.planChangeCta} />
      </div>

      {sub.modules.length > 0 && (
        <div className="mt-4">
          <p className="text-xs uppercase tracking-wider text-ink-muted">
            {t.account.planModules}
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            {sub.modules.map((m) => (
              <span
                key={m}
                className="rounded-lg bg-ink/[0.05] px-2.5 py-1 text-xs font-medium text-ink-soft"
              >
                {MODULE_LABEL[m] ?? m}
                {/* Bought on top rather than included in the rung. Worth
                    marking: it is the line an owner questions on an invoice,
                    and it is the one they can drop without changing plan. */}
                {sub.addons?.includes(m) && (
                  <span className="ml-1 text-ink-muted">
                    · {t.account.planAddons}
                  </span>
                )}
              </span>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

/** The one way to change a plan, and it is a conversation.
 *
 *  ⚠️ No self-serve upgrade button, deliberately: switching a rung changes what
 *  the customer is billed and the money is still collected by hand. A control
 *  that looked like it would upgrade the account and instead sent a message
 *  would be the worst of both, so this one says what it is.
 *
 *  ⚠️ `target="_blank"` with `rel="noopener"`: the panel is a working screen
 *  with unsaved forms on other tabs, and navigating it away to Telegram to ask
 *  a question is a way to lose them. */
function TelegramLink({ label }: { label: string }) {
  return (
    <a
      href={TELEGRAM}
      target="_blank"
      rel="noopener noreferrer"
      className="mt-2 inline-flex items-center gap-1.5 text-sm font-semibold text-brand hover:underline"
    >
      {/* Telegram's mark, inline: an emoji renders differently on the Windows
          monoblock this panel is often opened on, and a remote image is a
          request that can fail on a restaurant's connection. */}
      <svg viewBox="0 0 24 24" className="h-4 w-4" fill="currentColor" aria-hidden>
        <path d="M21.9 4.3 18.8 19c-.2 1-.9 1.3-1.7.8l-4.8-3.5-2.3 2.2c-.3.3-.5.5-1 .5l.3-4.9 8.9-8c.4-.3-.1-.5-.6-.2L6.7 13 2 11.5c-1-.3-1-1 .2-1.5l18.4-7.1c.9-.3 1.6.2 1.3 1.4Z" />
      </svg>
      {label}
    </a>
  );
}
