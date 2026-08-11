"use client";

// The home page's card strip, edited by the restaurant itself.
//
// ⚠️ **The three built-in cards are promises somebody else wrote.** "Delivered
// in 30–45 minutes", "bought at the market every morning", "cash or card" — true
// of the sample restaurant, and a lie on a bakery that does not deliver. Until
// now the only way to change them was the console, which means our operator had
// to be asked; a restaurant that cannot correct a claim made in its own name
// eventually stops believing the rest of the page.
//
// Two controls, because there are two different wishes: replace the words, or
// have no strip at all.
//
//   • empty list  → the built-in copy, exactly as before. That is what every
//     existing restaurant has, so nothing changes for anyone who never opens
//     this section.
//   • hide        → the band draws nothing. Deleting the last card cannot mean
//     this: an empty list is "never touched", and reading it as "wanted blank"
//     would empty the strip on every site the day it shipped.

import { PERK_ICON_NAMES, PerkIcon } from "@/components/design/perkIcons";
import LocalizedField from "@/components/admin/LocalizedField";
import { useAdminT } from "@/lib/i18n/admin";
import { EMPTY_LOCALIZED } from "@/lib/i18n/site-content";
import type { PerkCard } from "@/lib/types";

// The strip is a three-column grid; more than six cards is two full rows of
// small print nobody reads, and the cap is what stops this from becoming one.
const MAX_CARDS = 6;

export default function PerksEditor({
  cards,
  hidden,
  onChange,
}: {
  cards: PerkCard[];
  hidden: boolean;
  onChange: (next: { perks: PerkCard[]; hidePerks: boolean }) => void;
}) {
  const t = useAdminT();

  const patchCard = (i: number, p: Partial<PerkCard>) =>
    onChange({
      perks: cards.map((c, idx) => (idx === i ? { ...c, ...p } : c)),
      hidePerks: hidden,
    });

  const setCards = (next: PerkCard[]) =>
    onChange({ perks: next, hidePerks: hidden });

  return (
    <div className="rounded-2xl border border-line bg-ink/[0.02] p-4">
      <span className="text-sm font-semibold">{t.settings.perksTitle}</span>
      <p className="mt-1 text-xs text-ink-muted">{t.settings.perksHint}</p>

      <label className="mt-3 flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={!hidden}
          onChange={(e) =>
            onChange({ perks: cards, hidePerks: !e.target.checked })
          }
        />
        <span>{t.settings.perksShow}</span>
      </label>

      {!hidden && (
        <>
          {cards.length === 0 && (
            <p className="mt-3 rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:bg-amber-500/10 dark:text-amber-300">
              {t.settings.perksDefaults}
            </p>
          )}

          <div className="mt-4 space-y-4">
            {cards.map((card, i) => (
              <div key={i} className="rounded-2xl border border-line bg-surface p-4">
                <div className="flex items-center justify-between gap-3">
                  <span className="text-xs font-semibold uppercase text-ink-muted">
                    {t.settings.perksCard(i + 1)}
                  </span>
                  <button
                    type="button"
                    onClick={() => setCards(cards.filter((_, idx) => idx !== i))}
                    className="text-xs text-ink-muted hover:text-red-600"
                  >
                    {t.common.delete}
                  </button>
                </div>

                {/* An icon list, not a name to type: the drawings are a fixed
                    set, and a typed name that matches nothing would silently
                    become a star on the live site. */}
                <div className="mt-3 flex flex-wrap gap-2">
                  {PERK_ICON_NAMES.map((name) => (
                    <button
                      key={name}
                      type="button"
                      title={name}
                      onClick={() => patchCard(i, { icon: name })}
                      className={`flex h-10 w-10 items-center justify-center rounded-xl border ${
                        card.icon === name
                          ? "border-brand bg-brand-tint"
                          : "border-line hover:border-line-strong"
                      }`}
                    >
                      <PerkIcon name={name} className="h-5 w-5 text-brand" />
                    </button>
                  ))}
                </div>

                <div className="mt-4 space-y-3">
                  <LocalizedField
                    plain
                    label={t.settings.perksCardTitle}
                    value={card.title ?? EMPTY_LOCALIZED}
                    onChange={(v) => patchCard(i, { title: v })}
                  />
                  <LocalizedField
                    plain
                    multiline
                    rows={2}
                    label={t.settings.perksCardText}
                    value={card.text ?? EMPTY_LOCALIZED}
                    onChange={(v) => patchCard(i, { text: v })}
                  />
                </div>
              </div>
            ))}
          </div>

          {cards.length < MAX_CARDS && (
            <button
              type="button"
              onClick={() =>
                setCards([
                  ...cards,
                  {
                    icon: PERK_ICON_NAMES[cards.length % PERK_ICON_NAMES.length],
                    title: EMPTY_LOCALIZED,
                    text: EMPTY_LOCALIZED,
                  },
                ])
              }
              className="btn-ghost mt-4 px-3 py-2 text-sm"
            >
              {t.settings.perksAdd}
            </button>
          )}
        </>
      )}
    </div>
  );
}
