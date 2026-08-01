// Turns the restaurant's theme settings into CSS variable overrides.
//
// The result is inlined into <head> by the root layout, so the restaurant's
// colours are already applied on the first paint — no flash of the default
// orange, and no client-side work.

import type { SiteTheme } from "./types";

// Font pairings the admin panel can pick from. The families themselves are
// loaded by the root layout (next/font), so switching is just a var swap.
export const FONT_PRESETS = {
  classic: { sans: "--font-inter", display: "--font-playfair" },
  modern: { sans: "--font-inter", display: "--font-inter" },
  soft: { sans: "--font-nunito", display: "--font-nunito" },
} as const;

export type FontPreset = keyof typeof FONT_PRESETS;

// Page/card background tones, as "R G B" pairs for light and dark.
export const BACKGROUNDS = {
  warm: { bg: "251 248 244", surface: "255 255 255", darkBg: "16 14 13", darkSurface: "39 35 32" },
  white: { bg: "255 255 255", surface: "255 255 255", darkBg: "12 12 12", darkSurface: "32 32 32" },
  cool: { bg: "246 248 251", surface: "255 255 255", darkBg: "13 15 19", darkSurface: "32 36 43" },
  sand: { bg: "246 241 232", surface: "255 253 249", darkBg: "20 18 14", darkSurface: "43 39 32" },
} as const;

export type BackgroundPreset = keyof typeof BACKGROUNDS;

// "Nothing customised" — every field empty means the built-in design.
export const EMPTY_THEME: SiteTheme = {
  brand: "",
  brandDark: "",
  radius: null,
  buttonShape: "",
  font: "",
  background: "",
  shadow: "",
  buttonStyle: "",
  scale: null,
};

// Card depth presets.
const SHADOWS = {
  none: { card: "none", hover: "none" },
  soft: {
    card: "0 1px 2px rgb(28 25 23 / 0.04), 0 8px 24px -12px rgb(28 25 23 / 0.18)",
    hover: "0 2px 4px rgb(28 25 23 / 0.06), 0 18px 40px -16px rgb(28 25 23 / 0.28)",
  },
  strong: {
    card: "0 2px 4px rgb(28 25 23 / 0.08), 0 16px 40px -14px rgb(28 25 23 / 0.32)",
    hover: "0 4px 10px rgb(28 25 23 / 0.12), 0 30px 60px -18px rgb(28 25 23 / 0.42)",
  },
} as const;

const DARK_SHADOWS = {
  none: { card: "none", hover: "none" },
  soft: {
    card: "0 1px 2px rgb(0 0 0 / 0.35), 0 10px 28px -14px rgb(0 0 0 / 0.6)",
    hover: "0 2px 6px rgb(0 0 0 / 0.45), 0 22px 48px -18px rgb(0 0 0 / 0.75)",
  },
  strong: {
    card: "0 2px 6px rgb(0 0 0 / 0.5), 0 18px 44px -16px rgb(0 0 0 / 0.8)",
    hover: "0 4px 12px rgb(0 0 0 / 0.6), 0 32px 64px -20px rgb(0 0 0 / 0.9)",
  },
} as const;

interface Rgb {
  r: number;
  g: number;
  b: number;
}

function parseHex(hex: string): Rgb | null {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return null;
  const n = parseInt(m[1], 16);
  return { r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 };
}

const clamp = (n: number) => Math.max(0, Math.min(255, Math.round(n)));

// Mix towards white (amount > 0) or black (amount < 0).
function mix(c: Rgb, amount: number): Rgb {
  const target = amount > 0 ? 255 : 0;
  const k = Math.abs(amount);
  return {
    r: clamp(c.r + (target - c.r) * k),
    g: clamp(c.g + (target - c.g) * k),
    b: clamp(c.b + (target - c.b) * k),
  };
}

const triple = (c: Rgb) => `${c.r} ${c.g} ${c.b}`;

// Perceived brightness, used to keep the dark-mode accent readable.
function luminance(c: Rgb): number {
  return (0.299 * c.r + 0.587 * c.g + 0.114 * c.b) / 255;
}

/**
 * Builds the <style> body for a restaurant theme. Returns "" when nothing is
 * customised, so the built-in design stays byte-for-byte the default.
 */
export function themeCss(theme?: SiteTheme | null): string {
  if (!theme) return "";

  const light: string[] = [];
  const dark: string[] = [];

  const brand = parseHex(theme.brand ?? "");
  if (brand) {
    light.push(`--brand: ${triple(brand)}`);
    light.push(`--brand-dark: ${triple(mix(brand, -0.25))}`);
    light.push(`--brand-light: ${triple(mix(brand, 0.35))}`);
    light.push(`--brand-tint: ${triple(mix(brand, 0.9))}`);

    // Dark mode needs a lighter accent or the brand disappears on near-black.
    const explicit = parseHex(theme.brandDark ?? "");
    const darkBrand =
      explicit ?? (luminance(brand) < 0.55 ? mix(brand, 0.3) : brand);
    dark.push(`--brand: ${triple(darkBrand)}`);
    dark.push(`--brand-dark: ${triple(mix(darkBrand, -0.15))}`);
    dark.push(`--brand-light: ${triple(mix(darkBrand, 0.3))}`);
    dark.push(`--brand-tint: ${triple(mix(brand, -0.72))}`);
  }

  if (typeof theme.radius === "number" && theme.radius >= 0) {
    const xl = Math.min(theme.radius, 40);
    light.push(`--radius-xl: ${xl}px`);
    light.push(`--radius-lg: ${Math.round(xl * 0.66)}px`);
    light.push(`--radius-md: ${Math.round(xl * 0.5)}px`);
    if (theme.buttonShape === "match") {
      light.push(`--radius-btn: ${Math.round(xl * 0.5)}px`);
    }
  } else if (theme.buttonShape === "match") {
    light.push(`--radius-btn: var(--radius-md)`);
  }

  // Background tone.
  const bgKey = theme.background as BackgroundPreset | "" | undefined;
  if (bgKey && BACKGROUNDS[bgKey]) {
    const b = BACKGROUNDS[bgKey];
    light.push(`--bg: ${b.bg}`, `--surface: ${b.surface}`);
    dark.push(`--bg: ${b.darkBg}`, `--surface: ${b.darkSurface}`);
  }

  // Card depth.
  const shadowKey = theme.shadow as keyof typeof SHADOWS | "" | undefined;
  if (shadowKey && SHADOWS[shadowKey]) {
    light.push(
      `--shadow-card: ${SHADOWS[shadowKey].card}`,
      `--shadow-card-hover: ${SHADOWS[shadowKey].hover}`,
    );
    dark.push(
      `--shadow-card: ${DARK_SHADOWS[shadowKey].card}`,
      `--shadow-card-hover: ${DARK_SHADOWS[shadowKey].hover}`,
    );
  }

  // Primary button look.
  if (theme.buttonStyle === "outline") {
    light.push(
      "--btn-bg: transparent",
      "--btn-fg: rgb(var(--brand))",
      "--btn-border: rgb(var(--brand))",
      "--btn-bg-hover: rgb(var(--brand))",
      "--btn-fg-hover: #fff",
    );
  } else if (theme.buttonStyle === "soft") {
    light.push(
      "--btn-bg: rgb(var(--brand-tint))",
      "--btn-fg: rgb(var(--brand-dark))",
      "--btn-border: transparent",
      "--btn-bg-hover: rgb(var(--brand))",
      "--btn-fg-hover: #fff",
    );
  }

  // Density: root font size scales text and rem-based spacing together.
  if (typeof theme.scale === "number" && theme.scale >= 14 && theme.scale <= 19) {
    light.push(`--root-size: ${theme.scale}px`);
  }

  const font = FONT_PRESETS[(theme.font || "classic") as FontPreset];
  if (font && theme.font && theme.font !== "classic") {
    light.push(`--font-sans: var(${font.sans})`);
    light.push(`--font-display: var(${font.display})`);
  }

  const blocks: string[] = [];
  if (light.length) blocks.push(`:root{${light.join(";")}}`);
  if (dark.length) blocks.push(`.dark{${dark.join(";")}}`);
  return blocks.join("");
}
