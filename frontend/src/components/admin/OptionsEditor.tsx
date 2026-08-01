"use client";

// Editor for a dish's option groups ("Hajm", "Qo'shimcha", ...). Each group
// holds choices with a price delta that is added to the dish price. Prices are
// kept as strings while typing (so "" and "-" are valid intermediate states)
// and converted on save by `fromOptionDrafts`.

import { useAdminT } from "@/lib/i18n/admin";
import type { MenuOption } from "@/lib/types";

export interface ChoiceDraft {
  name: string;
  nameRu: string;
  nameEn: string;
  priceDelta: string;
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
}: {
  groups: OptionGroupDraft[];
  onChange: (next: OptionGroupDraft[]) => void;
}) {
  const t = useAdminT();
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
                  { name: "", nameRu: "", nameEn: "", priceDelta: "0" },
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
                </div>
              ))}
              <button
                type="button"
                className="text-sm font-medium text-brand hover:underline"
                onClick={() =>
                  updateGroup(gi, {
                    choices: [
                      ...group.choices,
                      { name: "", nameRu: "", nameEn: "", priceDelta: "0" },
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
