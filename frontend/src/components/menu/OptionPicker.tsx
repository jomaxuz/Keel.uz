"use client";

// Renders a dish's option groups: radio-style for single-choice groups,
// checkbox-style for `multiple` ones. The selection is lifted to the parent
// (AddToCartControl), which turns it into a cart line.

import { formatPrice } from "@/lib/format";
import { useI18n } from "@/lib/i18n/client";
import { contentName } from "@/lib/i18n/content";
import type { SelectedOption } from "@/lib/cart";
import type { MenuOption } from "@/lib/types";

export default function OptionPicker({
  options,
  selected,
  onChange,
  currency,
  missing,
}: {
  options: MenuOption[];
  selected: SelectedOption[];
  onChange: (next: SelectedOption[]) => void;
  currency: string;
  // Names of required groups the customer has not answered yet.
  missing: string[];
}) {
  const { lang, t } = useI18n();

  function isChecked(group: MenuOption, choiceName: string): boolean {
    return selected.some(
      (s) => s.group.name === group.name && s.choice.name === choiceName,
    );
  }

  function toggle(group: MenuOption, choiceName: string) {
    const choice = group.choices.find((c) => c.name === choiceName);
    if (!choice) return;
    const entry: SelectedOption = {
      group: { name: group.name, nameRu: group.nameRu, nameEn: group.nameEn },
      choice: {
        name: choice.name,
        nameRu: choice.nameRu,
        nameEn: choice.nameEn,
      },
      priceDelta: choice.priceDelta,
    };

    if (group.multiple) {
      onChange(
        isChecked(group, choiceName)
          ? selected.filter(
              (s) =>
                !(s.group.name === group.name && s.choice.name === choiceName),
            )
          : [...selected, entry],
      );
      return;
    }
    // Single choice: replace whatever was picked in this group. Tapping the
    // current choice clears it, unless the group is required.
    const rest = selected.filter((s) => s.group.name !== group.name);
    if (isChecked(group, choiceName) && !group.required) {
      onChange(rest);
      return;
    }
    onChange([...rest, entry]);
  }

  return (
    <div className="space-y-5">
      {options.map((group) => {
        const isMissing = missing.includes(group.name);
        return (
          <fieldset key={group.name}>
            <legend className="flex flex-wrap items-center gap-2 text-sm font-semibold">
              {contentName(group, lang)}
              <span
                className={`text-xs font-medium ${
                  isMissing ? "text-red-600" : "text-ink-muted"
                }`}
              >
                {group.required
                  ? `· ${t.item.required}`
                  : `· ${group.multiple ? t.item.pickAny : t.item.pickOne}`}
              </span>
            </legend>

            <div className="mt-2 flex flex-wrap gap-2">
              {group.choices.map((choice) => {
                const active = isChecked(group, choice.name);
                return (
                  <button
                    key={choice.name}
                    type="button"
                    aria-pressed={active}
                    onClick={() => toggle(group, choice.name)}
                    className={`rounded-full border px-4 py-2 text-sm transition-colors ${
                      active
                        ? "border-brand bg-brand text-white"
                        : isMissing
                          ? "border-red-400 text-ink-soft hover:border-brand"
                          : "border-line-strong text-ink-soft hover:border-brand hover:text-brand"
                    }`}
                  >
                    {contentName(choice, lang)}
                    {choice.priceDelta !== 0 && (
                      <span
                        className={`ml-1.5 text-xs ${active ? "text-white/80" : "text-ink-muted"}`}
                      >
                        {choice.priceDelta > 0 ? "+" : "−"}
                        {formatPrice(Math.abs(choice.priceDelta), currency, lang)}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>

            {isMissing && (
              <p className="mt-1.5 text-xs text-red-600">
                {t.item.missingRequired(contentName(group, lang))}
              </p>
            )}
          </fieldset>
        );
      })}
    </div>
  );
}
