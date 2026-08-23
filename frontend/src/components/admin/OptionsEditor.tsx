"use client";

// Editor for a dish's option groups ("Hajm", "Qo'shimcha", ...). Each group
// holds choices with a price delta that is added to the dish price. Prices are
// kept as strings while typing (so "" and "-" are valid intermediate states)
// and converted on save by `fromOptionDrafts`.

import { useState } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import RecipeEditor from "@/components/admin/RecipeEditor";
import type { Ingredient, MenuOption, RecipeLine } from "@/lib/types";

export interface ChoiceDraft {
  name: string;
  nameRu: string;
  nameEn: string;
  priceDelta: string;
  /** What this choice alone takes out of the store, per portion.
   *
   *  ⚠️ **The bar sells one bottle in three measures.** A vodka poured at 40,
   *  50 and 100 ml is one dish with a "Hajm" group, and until this existed
   *  every one of them took the dish's own recipe out of the store — so the
   *  price varied and the stock did not, and the difference surfaced once a
   *  month as a shortfall nobody could attribute. */
  recipe: RecipeLine[];
}

export interface OptionGroupDraft {
  name: string;
  nameRu: string;
  nameEn: string;
  required: boolean;
  multiple: boolean;
  choices: ChoiceDraft[];
}

export function toOptionDrafts(options: MenuOption[] | null): OptionGroupDraft[] {
  return (options ?? []).map((g) => ({
    name: g.name,
    nameRu: g.nameRu ?? "",
    nameEn: g.nameEn ?? "",
    required: g.required ?? false,
    multiple: g.multiple ?? false,
    choices: (g.choices ?? []).map((c) => ({
      name: c.name,
      nameRu: c.nameRu ?? "",
      nameEn: c.nameEn ?? "",
      priceDelta: String(c.priceDelta ?? 0),
      recipe: c.recipe ?? [],
    })),
  }));
}

// Drops groups/choices left blank so an accidentally added empty row never
// reaches the menu.
export function fromOptionDrafts(drafts: OptionGroupDraft[]): MenuOption[] {
  return drafts
    .map((g) => ({
      name: g.name.trim(),
      nameRu: g.nameRu.trim(),
      nameEn: g.nameEn.trim(),
      required: g.required,
      multiple: g.multiple,
      choices: g.choices
        .filter((c) => c.name.trim())
        .map((c) => ({
          name: c.name.trim(),
          nameRu: c.nameRu.trim(),
          nameEn: c.nameEn.trim(),
          priceDelta: Number(c.priceDelta) || 0,
          // ⚠️ Only lines that name an ingredient and take something: a half
          // filled row would be a card that silently accounts for nothing.
          recipe: (c.recipe ?? []).filter((l) => l.ingredientId && l.qty > 0),
        })),
    }))
    .filter((g) => g.name && g.choices.length > 0);
}

const inputCls =
  "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";
const smallInput =
  "w-full rounded-lg border border-line-strong bg-surface px-2.5 py-1.5 text-sm outline-none focus:border-brand";

export default function OptionsEditor({
  groups,
  onChange,
  ingredients = [],
}: {
  groups: OptionGroupDraft[];
  onChange: (next: OptionGroupDraft[]) => void;
  /** For the per-choice tech card. ⚠️ Empty until the restaurant has entered
   *  any ingredients, and then the card is not offered at all: a "recipe"
   *  button with nothing to put in it is a control that teaches people this
   *  screen has settings they cannot use. */
  ingredients?: Ingredient[];
}) {
  const t = useAdminT();
  // Which choice has its card open. ⚠️ One at a time, and closed by default:
  // most choices are only a price, and a card unfolded under every row would
  // bury the three name fields this editor is actually for.
  const [openCard, setOpenCard] = useState<string | null>(null);
  function updateGroup(i: number, patch: Partial<OptionGroupDraft>) {
    onChange(groups.map((g, gi) => (gi === i ? { ...g, ...patch } : g)));
  }

  function updateChoice(gi: number, ci: number, patch: Partial<ChoiceDraft>) {
    updateGroup(gi, {
      choices: groups[gi].choices.map((c, i) =>
        i === ci ? { ...c, ...patch } : c,
      ),
    });
  }

  return (
    <div className="sm:col-span-2 rounded-2xl border border-line bg-ink/[0.02] p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm font-semibold">
          {t.options.title}{" "}
          <span className="font-normal text-ink-muted">{t.options.hint}</span>
        </p>
        <button
          type="button"
          className="btn-ghost px-3 py-1.5 text-sm"
          onClick={() =>
            onChange([
              ...groups,
              {
                name: "",
                nameRu: "",
                nameEn: "",
                required: true,
                multiple: false,
                choices: [
                  {
                    name: "",
                    nameRu: "",
                    nameEn: "",
                    priceDelta: "0",
                    recipe: [],
                  },
                ],
              },
            ])
          }
        >
          {t.options.addGroup}
        </button>
      </div>

      {groups.length === 0 && (
        <p className="mt-3 text-sm text-ink-muted/70">
          {t.options.empty}
        </p>
      )}

      <div className="mt-3 space-y-4">
        {groups.map((group, gi) => (
          <div
            key={gi}
            className="rounded-xl border border-line bg-surface p-3"
          >
            <div className="grid gap-3 sm:grid-cols-3">
              <label className="block text-sm">
                <span className="font-medium">{t.options.groupName}</span>
                <input
                  className={inputCls}
                  value={group.name}
                  placeholder="Hajm"
                  onChange={(e) => updateGroup(gi, { name: e.target.value })}
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">RU</span>
                <input
                  className={inputCls}
                  value={group.nameRu}
                  placeholder={group.name}
                  onChange={(e) => updateGroup(gi, { nameRu: e.target.value })}
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">EN</span>
                <input
                  className={inputCls}
                  value={group.nameEn}
                  placeholder={group.name}
                  onChange={(e) => updateGroup(gi, { nameEn: e.target.value })}
                />
              </label>
            </div>

            <div className="mt-3 flex flex-wrap items-center gap-4 text-sm">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={group.required}
                  onChange={(e) =>
                    updateGroup(gi, { required: e.target.checked })
                  }
                />
                {t.options.required}
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={group.multiple}
                  onChange={(e) =>
                    updateGroup(gi, { multiple: e.target.checked })
                  }
                />
                {t.options.multiple}
              </label>
              <button
                type="button"
                className="ml-auto text-sm text-ink-muted hover:text-red-600"
                onClick={() => onChange(groups.filter((_, i) => i !== gi))}
              >
                {t.options.deleteGroup}
              </button>
            </div>

            {/* Choices */}
            <div className="mt-3 space-y-2">
              <div className="hidden gap-2 text-xs font-medium text-ink-muted sm:grid sm:grid-cols-[1fr_1fr_1fr_130px_32px]">
                <span>{t.options.choice}</span>
                <span>RU</span>
                <span>EN</span>
                <span>{t.options.priceDelta}</span>
                <span />
              </div>
              {group.choices.map((choice, ci) => (
                <div
                  key={ci}
                  className="grid gap-2 sm:grid-cols-[1fr_1fr_1fr_130px_32px] sm:items-center"
                >
                  <input
                    className={smallInput}
                    value={choice.name}
                    placeholder="Katta"
                    onChange={(e) =>
                      updateChoice(gi, ci, { name: e.target.value })
                    }
                  />
                  <input
                    className={smallInput}
                    value={choice.nameRu}
                    placeholder={choice.name}
                    onChange={(e) =>
                      updateChoice(gi, ci, { nameRu: e.target.value })
                    }
                  />
                  <input
                    className={smallInput}
                    value={choice.nameEn}
                    placeholder={choice.name}
                    onChange={(e) =>
                      updateChoice(gi, ci, { nameEn: e.target.value })
                    }
                  />
                  <input
                    type="number"
                    className={smallInput}
                    value={choice.priceDelta}
                    placeholder="0"
                    onChange={(e) =>
                      updateChoice(gi, ci, { priceDelta: e.target.value })
                    }
                  />
                  <button
                    type="button"
                    aria-label={t.options.deleteChoice}
                    className="justify-self-start text-ink-muted/70 hover:text-red-600 sm:justify-self-center"
                    onClick={() =>
                      updateGroup(gi, {
                        choices: group.choices.filter((_, i) => i !== ci),
                      })
                    }
                  >
                    ✕
                  </button>
                  {/* ---- What this choice pours ----

                      ⚠️ **Per choice, not per dish.** A vodka at 40, 50 and
                      100 ml is one dish whose stock differs entirely by which
                      measure was ticked — the price already varied, and until
                      this existed the store did not. The dish's own card stays
                      where it is: the tonic and the lemon go in whichever
                      measure of gin does.

                      ⚠️ Folded away, and the row says whether there is
                      anything behind it. Most choices are only a price, and a
                      card unfolded under every one of them would bury the
                      fields this editor exists for. */}
                  {ingredients.length > 0 && (
                    <div className="sm:col-span-5">
                      <button
                        type="button"
                        onClick={() =>
                          setOpenCard(
                            openCard === `${gi}:${ci}` ? null : `${gi}:${ci}`,
                          )
                        }
                        className="text-xs font-medium text-ink-muted hover:text-brand"
                      >
                        {choice.recipe.length > 0
                          ? t.options.recipeSet(choice.recipe.length)
                          : t.options.recipeAdd}
                      </button>
                      {openCard === `${gi}:${ci}` && (
                        <div className="mt-2 rounded-xl border border-line bg-surface p-3">
                          <p className="mb-2 text-xs text-ink-muted">
                            {t.options.recipeHint}
                          </p>
                          <RecipeEditor
                            lines={choice.recipe}
                            ingredients={ingredients}
                            // ⚠️ The margin is shown against the choice's own
                            // surcharge, not the dish price: this card costs
                            // what the extra measure costs, and comparing it to
                            // the whole drink would call every pour a loss.
                            price={Number(choice.priceDelta) || 0}
                            onChange={(recipe) =>
                              updateChoice(gi, ci, { recipe })
                            }
                          />
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
              <button
                type="button"
                className="text-sm font-medium text-brand hover:underline"
                onClick={() =>
                  updateGroup(gi, {
                    choices: [
                      ...group.choices,
                      {
                    name: "",
                    nameRu: "",
                    nameEn: "",
                    priceDelta: "0",
                    recipe: [],
                  },
                    ],
                  })
                }
              >
                {t.options.addChoice}
              </button>
            </div>

            <p className="mt-2 text-xs text-ink-muted/70">
              {t.options.note}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}
