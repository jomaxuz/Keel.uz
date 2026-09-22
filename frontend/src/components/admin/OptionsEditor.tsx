"use client";

// Editor for a dish's option groups ("Hajm", "Qo'shimcha", ...). Each group
// holds choices with a price delta that is added to the dish price. Prices are
// kept as strings while typing (so "" and "-" are valid intermediate states)
// and converted on save by `fromOptionDrafts`.

import { useState } from "react";

import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { usePanelWords } from "@/lib/panelWords";
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

export function toOptionDrafts(
  options: MenuOption[] | null,
): OptionGroupDraft[] {
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

/** Which groups would be thrown away by `fromOptionDrafts`, and why.
 *
 * ⚠️ **This exists because the dropping used to be silent, and that is exactly
 * what it looked like from the owner's side: fill in the sizes, press Saqlash,
 * the dish saves, the variants are gone.** Nothing errored — the filter below
 * is correct, an unnamed question with no answers is not a question — but a
 * correct filter applied without telling anybody is indistinguishable from a
 * save that did not work.
 *
 * The empty state now shows a worked example ("Kichik / Katta"), which made
 * this far more likely to bite: somebody reads the example, types the two
 * answers, and never notices that the box asking what the *question* is was
 * left blank. */
export function optionProblems(
  drafts: OptionGroupDraft[],
): { index: number; needName: boolean; needChoice: boolean }[] {
  const out: { index: number; needName: boolean; needChoice: boolean }[] = [];
  drafts.forEach((g, index) => {
    const needName = !g.name.trim();
    const needChoice = !g.choices.some((c) => c.name.trim());
    // ⚠️ A group that is *entirely* blank is not a problem — it is the row the
    // "add question" button just created, and complaining about it would mean
    // refusing to save a dish because somebody opened a form and changed their
    // mind. Only a half-filled question is a question about to be lost.
    if (needName && needChoice) return;
    if (needName || needChoice) out.push({ index, needName, needChoice });
  });
  return out;
}

// Drops groups/choices left blank so an accidentally added empty row never
// reaches the menu. ⚠️ Callers check `optionProblems` first — see the note on it.
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
  price = 0,
  ingredients = [],
}: {
  groups: OptionGroupDraft[];
  onChange: (next: OptionGroupDraft[]) => void;
  /** The dish's own price, so a choice can show what the guest actually pays.
   *
   *  ⚠️ **A signed delta is not a price, and an owner reads it as one.** "+5000"
   *  next to a field labelled "price" is the single most reported confusion on
   *  this screen: half the people typed the *final* price into it and made a
   *  50 000 so'm dish cost 95 000. Printing "mijoz 50 000 so'm to'laydi" beside
   *  the box removes the ambiguity without removing the field. */
  price?: number;
  /** For the per-choice tech card. ⚠️ Empty until the restaurant has entered
   *  any ingredients, and then the card is not offered at all: a "recipe"
   *  button with nothing to put in it is a control that teaches people this
   *  screen has settings they cannot use. */
  ingredients?: Ingredient[];
}) {
  const t = useAdminT();
  const w = usePanelWords();
  // Which choice has its card open. ⚠️ One at a time, and closed by default:
  // most choices are only a price, and a card unfolded under every row would
  // bury the fields this editor is actually for.
  const [openCard, setOpenCard] = useState<string | null>(null);
  // ⚠️ **Translations are folded away per question, and that is not tidiness.**
  // They were three of the four columns on every row, so the field an owner
  // came here to fill — the name — was a quarter of the width, and most
  // restaurants never fill RU or EN at all. Folded, the row is a name and a
  // price, which is what this screen is.
  const [openTranslations, setOpenTranslations] = useState<number | null>(null);

  // Which questions are half-filled, recomputed as they are typed: the warning
  // has to disappear the moment it is answered, or it teaches people to ignore
  // it.
  const problems = optionProblems(groups);
  const problem = (i: number) => problems.find((p) => p.index === i);

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

  const blankChoice = (): ChoiceDraft => ({
    name: "",
    nameRu: "",
    nameEn: "",
    priceDelta: "0",
    recipe: [],
  });

  return (
    <div className="rounded-2xl border border-line bg-ink/[0.02] p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-semibold">{t.options.title}</p>
          <p className="mt-0.5 text-xs text-ink-muted">{t.options.hint}</p>
        </div>
        <button
          type="button"
          className="btn-ghost shrink-0 px-3 py-1.5 text-sm"
          onClick={() =>
            onChange([
              ...groups,
              {
                name: "",
                nameRu: "",
                nameEn: "",
                required: true,
                multiple: false,
                choices: [blankChoice(), blankChoice()],
              },
            ])
          }
        >
          {t.options.addGroup}
        </button>
      </div>

      {/* ⚠️ **The explanation is two sentences and one worked example, and it
          only shows when there is nothing yet.** The old copy said "(ixtiyoriy
          — masalan hajm yoki qo'shimcha)" beside a heading called "Variantlar",
          which names the feature without saying what it is for. An owner does
          not think "option group with choices"; they think "menda kichik va
          katta bor, kattasi besh ming qimmat". */}
      {groups.length === 0 && (
        <div className="mt-3 rounded-xl border border-dashed border-line-strong p-3">
          <p className="text-sm leading-relaxed text-ink-soft">
            {w.optionsLead}
          </p>
          <p className="mt-1.5 text-sm text-ink-muted">{t.options.example}</p>
        </div>
      )}

      <div className="mt-3 space-y-4">
        {groups.map((group, gi) => (
          <div
            key={gi}
            className="rounded-xl border border-line bg-surface p-3.5"
          >
            <div className="flex flex-wrap items-end gap-3">
              <label className="block min-w-[14rem] flex-1 text-sm">
                <span className="font-medium">{t.options.groupName}</span>
                <input
                  className={inputCls}
                  value={group.name}
                  placeholder={t.options.groupPh}
                  onChange={(e) => updateGroup(gi, { name: e.target.value })}
                />
                {/* ⚠️ Said on the row, not only at save. By the time the alert
                    appears the owner has to work out which of four questions it
                    means; here the answer is the field it is under. */}
                {problem(gi)?.needName && (
                  <span className="mt-1 block text-xs text-danger">
                    {t.options.needName}
                  </span>
                )}
              </label>
              <button
                type="button"
                className="pb-2 text-sm text-ink-muted hover:text-red-600"
                onClick={() => onChange(groups.filter((_, i) => i !== gi))}
              >
                {t.options.deleteGroup}
              </button>
            </div>

            <div className="mt-2.5 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
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
            </div>

            {/* ---- The answers ---- */}
            <div className="mt-3 space-y-2">
              <div className="hidden gap-3 px-1 text-xs font-medium text-ink-muted sm:grid sm:grid-cols-[1fr_9rem_11rem_2rem]">
                <span>{t.options.choice}</span>
                <span>{t.options.priceDelta}</span>
                <span />
                <span />
              </div>
              {group.choices.map((choice, ci) => {
                const delta = Number(choice.priceDelta) || 0;
                const total = price + delta;
                return (
                  <div
                    key={ci}
                    className="rounded-lg border border-line/70 p-2 sm:border-0 sm:p-0"
                  >
                    <div className="grid gap-3 sm:grid-cols-[1fr_9rem_11rem_2rem] sm:items-center">
                      <input
                        className={smallInput}
                        value={choice.name}
                        placeholder={t.options.choicePh}
                        onChange={(e) =>
                          updateChoice(gi, ci, { name: e.target.value })
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
                      {/* ⚠️ The answer to "what will this actually cost", beside
                          the box that asks for a difference. Shown only once the
                          answer has a name, so an empty row is not decorated
                          with a price nobody set. */}
                      <span className="text-xs text-ink-muted">
                        {choice.name.trim()
                          ? t.options.resultPrice(formatPrice(total))
                          : ""}
                      </span>
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
                    </div>

                    {/* ---- What this choice pours ----

                        ⚠️ **Per choice, not per dish.** A vodka at 40, 50 and
                        100 ml is one dish whose stock differs entirely by which
                        measure was ticked — the price already varied, and until
                        this existed the store did not. The dish's own card stays
                        where it is: the tonic and the lemon go in whichever
                        measure of gin does. */}
                    {ingredients.length > 0 && choice.name.trim() !== "" && (
                      <div className="mt-1.5">
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
                              // what the extra measure costs, and comparing it
                              // to the whole drink would call every pour a loss.
                              price={delta}
                              onChange={(recipe) =>
                                updateChoice(gi, ci, { recipe })
                              }
                            />
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
              <button
                type="button"
                className="text-sm font-medium text-brand hover:underline"
                onClick={() =>
                  updateGroup(gi, {
                    choices: [...group.choices, blankChoice()],
                  })
                }
              >
                {t.options.addChoice}
              </button>
              {problem(gi)?.needChoice && (
                <p className="pt-1 text-xs text-danger">
                  {t.options.needChoice}
                </p>
              )}
              <p className="pt-1 text-xs text-ink-muted">
                {t.options.priceHint}
              </p>
            </div>

            {/* ---- Translations, folded ---- */}
            <div className="mt-3 border-t border-line pt-2.5">
              <button
                type="button"
                onClick={() =>
                  setOpenTranslations(openTranslations === gi ? null : gi)
                }
                className="text-xs font-medium text-ink-muted hover:text-brand"
              >
                {openTranslations === gi ? "− " : "+ "}
                {t.options.translations}
              </button>
              {openTranslations === gi && (
                <div className="mt-2 space-y-2">
                  <p className="text-xs text-ink-muted">
                    {t.options.translationsHint}
                  </p>
                  <div className="grid gap-2 sm:grid-cols-2">
                    <input
                      className={smallInput}
                      value={group.nameRu}
                      placeholder={`RU · ${group.name || t.options.groupPh}`}
                      onChange={(e) =>
                        updateGroup(gi, { nameRu: e.target.value })
                      }
                    />
                    <input
                      className={smallInput}
                      value={group.nameEn}
                      placeholder={`EN · ${group.name || t.options.groupPh}`}
                      onChange={(e) =>
                        updateGroup(gi, { nameEn: e.target.value })
                      }
                    />
                  </div>
                  {group.choices.map((choice, ci) => (
                    <div key={ci} className="grid gap-2 sm:grid-cols-2">
                      <input
                        className={smallInput}
                        value={choice.nameRu}
                        placeholder={`RU · ${choice.name || t.options.choicePh}`}
                        onChange={(e) =>
                          updateChoice(gi, ci, { nameRu: e.target.value })
                        }
                      />
                      <input
                        className={smallInput}
                        value={choice.nameEn}
                        placeholder={`EN · ${choice.name || t.options.choicePh}`}
                        onChange={(e) =>
                          updateChoice(gi, ci, { nameEn: e.target.value })
                        }
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* ---- What the guest sees ----

                ⚠️ **The form is abstract and this is not.** An owner filling in
                "required", "multiple" and a signed delta cannot picture the
                result, and the only way they used to find out was to save, open
                the site, and look. One line of preview turns the whole block
                into something checkable in place. */}
            {group.name.trim() !== "" &&
              group.choices.some((c) => c.name.trim() !== "") && (
                <div className="mt-3 rounded-lg bg-ink/[0.03] px-3 py-2">
                  <p className="text-xs font-medium text-ink-muted">
                    {t.options.preview}
                  </p>
                  <p className="mt-1 text-sm text-ink">
                    <span className="font-semibold">{group.name}</span>
                    {group.required ? " *" : ""} ·{" "}
                    {group.choices
                      .filter((c) => c.name.trim())
                      .map(
                        (c) =>
                          c.name +
                          (Number(c.priceDelta)
                            ? ` (${Number(c.priceDelta) > 0 ? "+" : ""}${formatPrice(
                                Number(c.priceDelta),
                              )})`
                            : ""),
                      )
                      .join(" · ")}
                  </p>
                </div>
              )}
          </div>
        ))}
      </div>

      {groups.length > 0 && (
        <p className="mt-3 text-xs text-ink-muted/80">{t.options.note}</p>
      )}
    </div>
  );
}
