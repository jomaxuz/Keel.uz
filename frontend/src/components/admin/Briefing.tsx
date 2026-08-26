"use client";

// The morning briefing: two to four things worth doing before service.
//
// ⚠️ **Every card prints its own figures beside the sentence.** The words come
// from a model and the numbers do not, and showing them together is the whole
// basis for trusting the words: an owner who wants to check can, in the same
// glance, without leaving the page. A card that showed only prose would be
// asking for trust it had given no way to verify.

import { useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import type { BriefingCard as Card } from "@/lib/types";
import { useAdminT } from "@/lib/i18n/admin";

// Where each action sends the owner. ⚠️ Named here rather than sent by the
// server so a card can never point at a screen this build does not have.
const DESTINATIONS: Record<string, string> = {
  campaign: "/admin/campaigns",
  shopping: "/admin/shopping",
  stocktake: "/admin/stocktake",
  menu: "/admin/menu",
  team: "/admin/reports",
  reports: "/admin/reports",
};

const AREA_TINT: Record<string, string> = {
  guests: "bg-sky-500",
  menu: "bg-amber-500",
  stock: "bg-emerald-500",
  team: "bg-violet-500",
  money: "bg-rose-500",
};

export default function Briefing({ scope }: { scope?: string }) {
  const t = useAdminT();
  const [cards, setCards] = useState<Card[] | null>(null);
  // ⚠️ Held separately from the cards so "you do not have this yet" is not
  // confused with "there is nothing to say today". They look identical on
  // screen — an empty panel — and only one of them is something to sell.
  const [offer, setOffer] = useState<{ monthly?: number } | null>(null);
  const [failed, setFailed] = useState("");

  useEffect(() => {
    let alive = true;
    api
      .adminInsights(scope)
      .then((r) => {
        if (!alive) return;
        setCards(r.cards ?? []);
        if (r.entitled === false) setOffer({ monthly: r.monthly });
        if (r.error) setFailed(r.error);
      })
      // ⚠️ A failure here draws nothing at all. The dashboard's own numbers do
      // not depend on this, and an error banner over a working dashboard is a
      // worse morning than a missing panel nobody was promised.
      .catch(() => alive && setCards([]));
    return () => {
      alive = false;
    };
  }, [scope]);

  // Nothing to say, still loading, or switched off all render as absence: this
  // is an addition to the page, never a hole in it.
  // ⚠️ Something the restaurant can buy is worth a line; a quiet morning is
  // not. A panel that said "no insights today" every day would be trained out
  // of an owner's attention within a week, and would then be invisible on the
  // morning it had four.
  if (offer) {
    return (
      <section className="mt-6 rounded-2xl border border-dashed border-line bg-surface-soft p-4">
        <h2 className="font-semibold">{t.briefing.title}</h2>
        <p className="mt-1 text-sm text-ink-soft">{t.briefing.locked}</p>
        {offer.monthly ? (
          <p className="mt-2 text-sm font-semibold">
            {offer.monthly.toLocaleString("ru-RU")} {t.briefing.perMonth}
          </p>
        ) : null}
      </section>
    );
  }
  // ⚠️ **A failure says so, for the owner only.** This is the third time this
  // feature has been reported as broken while working exactly as written, and
  // every time the missing piece was a sentence rather than a fix. A rate limit
  // that clears in eighteen seconds and a key that was never configured look
  // identical from an empty dashboard.
  if (failed) {
    return (
      <section className="mt-6 rounded-2xl border border-line bg-surface-soft p-4">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-muted">
          {t.briefing.title}
        </h2>
        <p className="mt-1 text-sm text-ink-soft">{t.briefing.failed}</p>
        <p className="mt-1 text-xs text-ink-muted">{failed}</p>
      </section>
    );
  }
  if (!cards || cards.length === 0) return null;

  return (
    <section className="mt-6">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-muted">
        {t.briefing.title}
      </h2>
      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        {cards.map((c) => (
          <article
            key={c.key}
            className="relative overflow-hidden rounded-2xl border border-line bg-surface p-4"
          >
            <span
              className={`absolute inset-y-0 left-0 w-1 ${
                AREA_TINT[c.area] ?? "bg-brand"
              }`}
              aria-hidden
            />
            <h3 className="pl-2 font-semibold">{c.title}</h3>
            <p className="mt-1 pl-2 text-sm text-ink-soft">{c.body}</p>

            {/* The exact figures, beside the words that were written about
                them. Small on purpose — they are for checking, not reading. */}
            {c.numbers && Object.keys(c.numbers).length > 0 && (
              <dl className="mt-2 flex flex-wrap gap-x-4 gap-y-1 pl-2 text-xs text-ink-muted">
                {Object.entries(c.numbers).map(([k, v]) => (
                  <div key={k} className="flex gap-1">
                    <dt>{t.briefing.numbers[k] ?? k}:</dt>
                    <dd className="font-medium tabular-nums text-ink">
                      {typeof v === "number" ? v.toLocaleString("ru-RU") : String(v)}
                    </dd>
                  </div>
                ))}
              </dl>
            )}

            {c.action && DESTINATIONS[c.action] && (
              <Link
                href={DESTINATIONS[c.action]}
                className="mt-3 ml-2 inline-block text-sm font-semibold text-brand hover:underline"
              >
                {t.briefing.actions[c.action] ?? t.briefing.open} →
              </Link>
            )}
          </article>
        ))}
      </div>
    </section>
  );
}
