"use client";

// Lets the restaurant restyle the public site: accent colour, corner roundness,
// button shape and font pairing. The preview below the controls is rendered
// with the very same CSS variables the live site uses, so what the owner sees
// here is what the site will look like after saving.

import { useAdminT } from "@/lib/i18n/admin";
import { EMPTY_THEME, themeCss } from "@/lib/theme-css";
import type { SiteTheme } from "@/lib/types";

const DEFAULT_BRAND = "#e2590d";
const DEFAULT_RADIUS = 24;

// One-click looks. Each preset is just a bundle of the same knobs below, so an
// owner who is not a designer still gets a coherent result in one tap.
const PRESETS: { id: string; label: string; theme: SiteTheme }[] = [
  {
    id: "default",
    label: "Issiq (standart)",
    theme: { ...EMPTY_THEME },
  },
  {
    id: "minimal",
    label: "Minimal",
    theme: {
      ...EMPTY_THEME,
      brand: "#1f2937",
      radius: 6,
      buttonShape: "match",
      font: "modern",
      background: "white",
      shadow: "none",
      buttonStyle: "solid",
    },
  },
  {
    id: "fresh",
    label: "Yashil",
    theme: {
      ...EMPTY_THEME,
      brand: "#16a34a",
      radius: 20,
      font: "soft",
      background: "cool",
      shadow: "soft",
      buttonStyle: "solid",
    },
  },
  {
    id: "elegant",
    label: "Nafis",
    theme: {
      ...EMPTY_THEME,
      brand: "#7c3aed",
      radius: 14,
      buttonShape: "match",
      font: "classic",
      background: "sand",
      shadow: "strong",
      buttonStyle: "outline",
    },
  },
  {
    id: "night",
    label: "Ko'k",
    theme: {
      ...EMPTY_THEME,
      brand: "#2563eb",
      radius: 10,
      buttonShape: "match",
      font: "modern",
      background: "cool",
      shadow: "soft",
      buttonStyle: "soft",
    },
  },
];

const SWATCHES = [
  "#e2590d", // warm orange (built-in)
  "#dc2626", // red
  "#ea580c", // tangerine
  "#ca8a04", // gold
  "#16a34a", // green
  "#0d9488", // teal
  "#2563eb", // blue
  "#7c3aed", // violet
  "#db2777", // pink
  "#1f2937", // graphite
];

export default function DesignEditor({
  theme,
  onChange,
}: {
  theme: SiteTheme;
  onChange: (next: SiteTheme) => void;
}) {
  const t = useAdminT();
  const brand = theme.brand || DEFAULT_BRAND;
  const radius = theme.radius ?? DEFAULT_RADIUS;
  const shape = theme.buttonShape || "pill";
  const font = theme.font || "classic";
  const background = theme.background || "warm";
  const shadow = theme.shadow || "soft";
  const buttonStyle = theme.buttonStyle || "solid";
  const scale = theme.scale ?? 16;

  const set = (patch: Partial<SiteTheme>) => onChange({ ...theme, ...patch });

  // Same generator as the live site, scoped to the preview box.
  const previewCss = themeCss({ ...theme, brand, radius })
    .replace(/:root\{/g, ".theme-preview{")
    .replace(/\.dark\{/g, ".theme-preview.is-dark{");

  return (
    <div className="space-y-6">
      <p className="text-sm text-ink-muted">{t.settings.designHint}</p>

      {/* One-tap looks */}
      <div>
        <span className="text-sm font-medium">{t.settings.presets}</span>
        <div className="mt-2 flex flex-wrap gap-2">
          {PRESETS.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => onChange(p.theme)}
              className="flex items-center gap-2 rounded-full border border-line-strong px-3 py-1.5 text-sm font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand"
            >
              <span
                className="h-4 w-4 rounded-full"
                style={{ background: p.theme.brand || DEFAULT_BRAND }}
              />
              {p.label}
            </button>
          ))}
        </div>
        <p className="mt-1.5 text-xs text-ink-muted">{t.settings.presetsHint}</p>
      </div>

      {/* Accent colour */}
      <div>
        <span className="text-sm font-medium">{t.settings.brandColor}</span>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          {SWATCHES.map((c) => (
            <button
              key={c}
              type="button"
              aria-label={c}
              onClick={() => set({ brand: c })}
              style={{ background: c }}
              className={`h-9 w-9 rounded-full ring-offset-2 ring-offset-surface transition-transform hover:scale-110 ${
                brand.toLowerCase() === c ? "ring-2 ring-ink" : ""
              }`}
            />
          ))}
          <input
            type="color"
            value={brand}
            onChange={(e) => set({ brand: e.target.value })}
            className="h-9 w-12 cursor-pointer rounded-md border border-line bg-surface"
          />
          <input
            className="w-28 rounded-md border border-line-strong bg-surface px-2 py-1.5 font-mono text-sm outline-none focus:border-brand"
            value={brand}
            onChange={(e) => set({ brand: e.target.value })}
          />
        </div>
        <p className="mt-1.5 text-xs text-ink-muted">
          {t.settings.brandColorHint}
        </p>
      </div>

      {/* Corner roundness */}
      <div>
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">{t.settings.radius}</span>
          <span className="text-sm tabular-nums text-ink-muted">{radius}px</span>
        </div>
        <input
          type="range"
          min={0}
          max={28}
          step={1}
          value={radius}
          onChange={(e) => set({ radius: Number(e.target.value) })}
          className="mt-2 w-full accent-brand"
        />
        <p className="mt-1 text-xs text-ink-muted">{t.settings.radiusHint}</p>
      </div>

      {/* Button shape */}
      <div>
        <span className="text-sm font-medium">{t.settings.buttonShape}</span>
        <div className="mt-2 flex flex-wrap gap-2">
          {(
            [
              ["pill", t.settings.buttonPill],
              ["match", t.settings.buttonMatch],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => set({ buttonShape: value })}
              className={`border px-4 py-2 text-sm font-semibold transition-colors ${
                shape === value
                  ? "border-brand bg-brand text-white"
                  : "border-line-strong text-ink-soft hover:border-brand"
              }`}
              style={{
                borderRadius: value === "pill" ? 9999 : Math.round(radius * 0.5),
              }}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Background tone */}
      <div>
        <span className="text-sm font-medium">{t.settings.background}</span>
        <div className="mt-2 flex flex-wrap gap-2">
          {(
            [
              ["warm", t.settings.bgWarm],
              ["white", t.settings.bgWhite],
              ["cool", t.settings.bgCool],
              ["sand", t.settings.bgSand],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => set({ background: value })}
              className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
                background === value
                  ? "border-brand bg-brand-tint/50 text-brand-dark"
                  : "border-line-strong text-ink-soft hover:border-brand"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Card depth */}
      <div>
        <span className="text-sm font-medium">{t.settings.shadow}</span>
        <div className="mt-2 flex flex-wrap gap-2">
          {(
            [
              ["none", t.settings.shadowNone],
              ["soft", t.settings.shadowSoft],
              ["strong", t.settings.shadowStrong],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => set({ shadow: value })}
              className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
                shadow === value
                  ? "border-brand bg-brand-tint/50 text-brand-dark"
                  : "border-line-strong text-ink-soft hover:border-brand"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Primary button fill */}
      <div>
        <span className="text-sm font-medium">{t.settings.buttonStyle}</span>
        <div className="mt-2 flex flex-wrap gap-2">
          {(
            [
              ["solid", t.settings.btnSolid],
              ["outline", t.settings.btnOutline],
              ["soft", t.settings.btnSoft],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => set({ buttonStyle: value })}
              className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
                buttonStyle === value
                  ? "border-brand bg-brand-tint/50 text-brand-dark"
                  : "border-line-strong text-ink-soft hover:border-brand"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Density */}
      <div>
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">{t.settings.scale}</span>
          <span className="text-sm tabular-nums text-ink-muted">{scale}px</span>
        </div>
        <input
          type="range"
          min={14}
          max={18}
          step={1}
          value={scale}
          onChange={(e) => set({ scale: Number(e.target.value) })}
          className="mt-2 w-full accent-brand"
        />
        <p className="mt-1 text-xs text-ink-muted">{t.settings.scaleHint}</p>
      </div>

      {/* Fonts */}
      <div>
        <span className="text-sm font-medium">{t.settings.font}</span>
        <div className="mt-2 grid gap-2 sm:grid-cols-3">
          {(
            [
              ["classic", t.settings.fontClassic, t.settings.fontClassicHint],
              ["modern", t.settings.fontModern, t.settings.fontModernHint],
              ["soft", t.settings.fontSoft, t.settings.fontSoftHint],
            ] as const
          ).map(([value, label, hint]) => (
            <button
              key={value}
              type="button"
              onClick={() => set({ font: value })}
              className={`rounded-2xl border px-4 py-3 text-left transition-colors ${
                font === value
                  ? "border-brand bg-brand-tint/40"
                  : "border-line-strong hover:border-brand"
              }`}
            >
              <span className="block text-sm font-semibold">{label}</span>
              <span className="mt-0.5 block text-xs text-ink-muted">{hint}</span>
            </button>
          ))}
        </div>
      </div>

      {/* Live preview */}
      <div>
        <span className="text-sm font-medium">{t.settings.preview}</span>
        <style dangerouslySetInnerHTML={{ __html: previewCss }} />
        <div className="mt-2 grid gap-3 sm:grid-cols-2">
          <PreviewBox className="theme-preview" t={t} />
          <PreviewBox className="theme-preview is-dark dark" t={t} dark />
        </div>
      </div>

      <button
        type="button"
        onClick={() => onChange({ ...EMPTY_THEME })}
        className="text-sm text-ink-muted hover:text-brand"
      >
        {t.settings.reset}
      </button>
    </div>
  );
}

function PreviewBox({
  className,
  t,
  dark = false,
}: {
  className: string;
  t: ReturnType<typeof useAdminT>;
  dark?: boolean;
}) {
  return (
    <div
      className={`${className} rounded-2xl border border-line p-4 ${
        dark ? "bg-cream text-ink" : "bg-cream text-ink"
      }`}
    >
      <div className="card p-4">
        <p className="font-display text-lg font-bold">{t.settings.previewCard}</p>
        <p className="mt-1 text-sm text-ink-muted">{t.settings.designHint}</p>
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <span className="btn-primary px-4 py-2">{t.settings.previewBtn}</span>
          <span className="btn-ghost px-4 py-2">{t.common.cancel}</span>
          <span className="badge-brand">★</span>
        </div>
      </div>
    </div>
  );
}
