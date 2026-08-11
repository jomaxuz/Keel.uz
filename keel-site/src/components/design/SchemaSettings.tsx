"use client";

// The settings panel, drawn from the schema.
//
// ⚠️ **This is the change the editor needed most.** Every control used to be
// written by hand per element type, so adding a section meant editing the editor —
// and the panel drifted into a long form of number fields that described a page
// rather than a tool for building one. A section now declares what it can be asked
// (served by the control plane) and this file renders the controls: one place, one
// set of behaviours, and a new section is a schema entry.
//
// It is the arrangement Shopify's theme editor uses, and their setting types are
// the ones worth having: text, textarea, range, select, checkbox, image, url,
// alignment, colour. The two differences here are deliberate:
//
//   • **Colour is a token, not a hex value.** A frozen colour keeps its value in
//     dark mode, where a near-black heading disappears, and stops following the
//     accent the restaurant chose.
//   • **Text is three fields, not one.** The site is trilingual; a single field
//     would produce a page that is correct in one language and empty in two.

import type { DesignSection } from "@/lib/api";

/** One of the restaurant's own records a band can be pointed at. */
export interface PickRecord {
  id: string;
  name: string;
}

export interface SettingDef {
  key: string;
  type: string;
  label: Record<string, string>;
  /** A line under the control. Used where the empty value means something an
   *  operator would otherwise have to discover by saving and looking. */
  help?: Record<string, string>;
  localized?: boolean;
  options?: { value: string; label: Record<string, string> }[];
  min?: number;
  max?: number;
  step?: number;
  default?: unknown;
}

export interface BlockDef {
  type: string;
  name: Record<string, string>;
  settings: SettingDef[];
}

export interface SectionDef {
  type: string;
  name: Record<string, string>;
  note?: Record<string, string>;
  settings: SettingDef[];
  blocks?: BlockDef[];
}

const TONES = ["", "surface", "raised", "charcoal", "brand"];
const TONE_PREVIEW: Record<string, string> = {
  "": "transparent",
  surface: "#ffffff",
  raised: "#f4f1ea",
  charcoal: "#20201e",
  brand: "#e2483d",
};
const LINKS = ["", "/", "/menu", "/cart", "/bron", "/about", "/profile"];

/** The one string this file says on its own. Everything else it draws is named by
 *  the schema, which is where wording belongs — but "clear" is a property of the
 *  control, not of the setting, so a schema entry for it would be repeated on
 *  every picker and could disagree with itself. */
const CLEAR_LABEL: Record<string, string> = {
  uz: "Tanlovni tozalash (hammasi)",
  ru: "Очистить выбор (все)",
  en: "Clear selection (all)",
};

type Bag = Record<string, unknown>;

function label(map: Record<string, string> | undefined, lang: string): string {
  return map?.[lang] ?? map?.uz ?? "";
}

/** Renders every setting a section declares. */
export default function SchemaSettings({
  def,
  values,
  lang,
  categories,
  onChange,
}: {
  def: SectionDef;
  values: Bag;
  lang: string;
  /** The restaurant's own categories, for the settings that pick among them. */
  categories?: PickRecord[];
  onChange: (key: string, value: unknown) => void;
}) {
  if (def.settings.length === 0) return null;
  return (
    <div className="space-y-2.5">
      {def.settings.map((s) => (
        <Control
          key={s.key}
          def={s}
          value={values[s.key]}
          lang={lang}
          categories={categories}
          onChange={(v) => onChange(s.key, v)}
        />
      ))}
    </div>
  );
}

export function Control({
  def,
  value,
  lang,
  categories,
  onChange,
}: {
  def: SettingDef;
  value: unknown;
  lang: string;
  categories?: PickRecord[];
  onChange: (value: unknown) => void;
}) {
  const name = label(def.label, lang);

  if (def.type === "categories") {
    const chosen = Array.isArray(value)
      ? (value as unknown[]).filter((v): v is string => typeof v === "string")
      : [];
    const list = categories ?? [];
    // ⚠️ Nothing at all rather than an empty box: a restaurant with no menu yet
    // is a normal state during setup, and a control with no options teaches the
    // operator that the panel is broken.
    if (list.length === 0) return null;
    return (
      <Field name={name} help={label(def.help, lang)}>
        <div className="flex flex-wrap gap-1.5">
          {list.map((c) => {
            const on = chosen.includes(c.id);
            return (
              <button
                key={c.id}
                type="button"
                // Toggles rather than a multi-select list: the question is "these
                // ones", and a ctrl-click list answers it by making the operator
                // remember what is already selected while they click.
                onClick={() =>
                  onChange(
                    on ? chosen.filter((id) => id !== c.id) : [...chosen, c.id],
                  )
                }
                className={`rounded-lg border px-2 py-1 text-[11px] font-semibold ${
                  on
                    ? "border-signal-500 bg-signal-500/10 text-ink"
                    : "border-line text-ink-soft"
                }`}
              >
                {c.name}
              </button>
            );
          })}
        </div>
        {chosen.length > 0 && (
          // The way back to "all". Without it, clearing a selection means
          // remembering which chips are lit and clicking each one off.
          <button
            type="button"
            onClick={() => onChange([])}
            className="mt-1.5 text-[11px] font-semibold text-ink-muted underline"
          >
            {label(CLEAR_LABEL, lang)}
          </button>
        )}
      </Field>
    );
  }

  if (def.type === "checkbox") {
    const on = typeof value === "boolean" ? value : Boolean(def.default);
    return (
      <button
        type="button"
        onClick={() => onChange(!on)}
        className="flex w-full items-center justify-between gap-2 py-1"
      >
        <span className="text-[11px] font-semibold text-ink-muted">{name}</span>
        <span className={`relative h-5 w-9 rounded-full transition ${on ? "bg-signal-500" : "bg-line-strong"}`}>
          <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-surface transition ${on ? "left-4" : "left-0.5"}`} />
        </span>
      </button>
    );
  }

  if (def.type === "range") {
    // ⚠️ A slider **with** its number. The slider is how a value is found and the
    // number is how it is repeated on the next customer.
    const n = typeof value === "number" ? value : Number(def.default ?? def.min ?? 0);
    return (
      <Field name={name}>
        <div className="flex items-center gap-2">
          <input
            type="range"
            min={def.min ?? 0}
            max={def.max ?? 100}
            step={def.step ?? 1}
            value={n}
            onChange={(e) => onChange(Number(e.target.value))}
            className="h-1.5 flex-1 accent-signal-500"
          />
          <span className="w-9 text-right text-[11px] tabular-nums text-ink">{n}</span>
        </div>
      </Field>
    );
  }

  if (def.type === "select") {
    const v = typeof value === "string" ? value : String(def.default ?? "");
    const opts = def.options ?? [];
    // Few options are worth showing at once; many belong behind a click.
    if (opts.length > 0 && opts.length <= 4) {
      return (
        <Field name={name}>
          <div className="flex overflow-hidden rounded-xl border border-line">
            {opts.map((o) => (
              <button
                key={o.value}
                type="button"
                onClick={() => onChange(o.value)}
                className={`flex-1 px-2 py-1.5 text-[11px] font-semibold ${
                  v === o.value ? "bg-raised text-ink" : "text-ink-soft"
                }`}
              >
                {label(o.label, lang)}
              </button>
            ))}
          </div>
        </Field>
      );
    }
    return (
      <Field name={name}>
        <select value={v} onChange={(e) => onChange(e.target.value)} className="select">
          {opts.map((o) => (
            <option key={o.value} value={o.value}>{label(o.label, lang)}</option>
          ))}
        </select>
      </Field>
    );
  }

  if (def.type === "alignment") {
    const v = typeof value === "string" ? value : String(def.default ?? "left");
    return (
      <Field name={name}>
        <div className="flex overflow-hidden rounded-xl border border-line">
          {[
            { v: "left", icon: "◧" },
            { v: "center", icon: "▣" },
          ].map((o) => (
            <button
              key={o.v}
              type="button"
              onClick={() => onChange(o.v)}
              className={`flex-1 px-2 py-1.5 text-[11px] ${v === o.v ? "bg-raised text-ink" : "text-ink-soft"}`}
            >
              {o.icon}
            </button>
          ))}
        </div>
      </Field>
    );
  }

  if (def.type === "tone") {
    const v = typeof value === "string" ? value : String(def.default ?? "");
    return (
      <Field name={name}>
        <div className="flex flex-wrap gap-1.5">
          {TONES.map((t) => (
            <button
              key={t}
              type="button"
              title={t || "—"}
              onClick={() => onChange(t)}
              style={{ background: TONE_PREVIEW[t] }}
              className={`h-6 w-6 rounded-md border ${
                v === t ? "border-signal-500 ring-2 ring-signal-500/40" : "border-line-strong"
              }`}
            />
          ))}
        </div>
      </Field>
    );
  }

  if (def.type === "link") {
    const v = typeof value === "string" ? value : String(def.default ?? "");
    return (
      <Field name={name}>
        <select value={v} onChange={(e) => onChange(e.target.value)} className="select">
          {LINKS.map((l) => (
            <option key={l} value={l}>{l || "—"}</option>
          ))}
        </select>
      </Field>
    );
  }

  if (def.type === "image") {
    const v = typeof value === "string" ? value : "";
    return (
      <Field name={name}>
        <input
          value={v}
          onChange={(e) => onChange(e.target.value)}
          placeholder="/uploads/…"
          className="input h-9 px-3 py-1 text-xs"
        />
      </Field>
    );
  }

  // text / textarea, localised into three fields.
  const bag = (value && typeof value === "object" ? value : {}) as Record<string, string>;
  const flat = typeof value === "string" ? value : "";
  const set = (l: string, s: string) =>
    onChange({ uz: bag.uz ?? flat, ru: bag.ru ?? "", en: bag.en ?? "", [l]: s });
  const Input = def.type === "textarea" ? "textarea" : "input";

  return (
    <Field name={name}>
      <div className="space-y-1">
        {(["uz", "ru", "en"] as const).map((l) => (
          <div key={l} className="flex items-start gap-1.5">
            <span className="w-5 pt-2 text-[10px] font-bold uppercase text-ink-muted">{l}</span>
            <Input
              value={bag[l] ?? (l === "uz" ? flat : "")}
              onChange={(e: React.ChangeEvent<HTMLInputElement & HTMLTextAreaElement>) =>
                set(l, e.target.value)
              }
              rows={def.type === "textarea" ? 2 : undefined}
              className="input min-h-9 flex-1 px-3 py-1.5 text-xs"
            />
          </div>
        ))}
      </div>
    </Field>
  );
}

function Field({
  name,
  help,
  children,
}: {
  name: string;
  help?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="text-[11px] font-semibold text-ink-muted">{name}</span>
      <div className="mt-0.5">{children}</div>
      {help && <span className="mt-0.5 block text-[10px] text-ink-muted">{help}</span>}
    </label>
  );
}

/** The section this band is, from the schema. */
export function defFor(schema: SectionDef[], section: DesignSection | undefined): SectionDef | null {
  if (!section) return null;
  return schema.find((s) => s.type === section.type) ?? null;
}
