import type { Config } from "tailwindcss";

// Semantic colours are CSS variables (see globals.css) so the whole UI can flip
// between light and dark by toggling the `dark` class on <html>.
const v = (name: string) => `rgb(var(${name}) / <alpha-value>)`;

const config: Config = {
  darkMode: "class",
  content: [
    "./src/app/**/*.{ts,tsx}",
    "./src/components/**/*.{ts,tsx}",
    "./src/lib/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Warm "freshly baked" accent — per-restaurant tunable.
        brand: {
          DEFAULT: v("--brand"),
          dark: v("--brand-dark"),
          light: v("--brand-light"),
          tint: v("--brand-tint"),
        },
        // Foreground scale (also used for hairline borders via /10 alpha).
        ink: {
          DEFAULT: v("--fg"),
          soft: v("--fg-soft"),
          muted: v("--fg-muted"),
        },
        cream: v("--bg"), // page background
        // Hairline borders/dividers. Already include their alpha, so they are
        // plain vars rather than the rgb(var(--x) / <alpha-value>) helper.
        line: {
          DEFAULT: "var(--line)",
          strong: "var(--line-strong)",
        },
        surface: v("--surface"), // cards, header/footer of admin, inputs
        // Always-dark surfaces (hero, site footer, dark CTA) — identical in
        // both themes, they carry white text by design.
        charcoal: {
          DEFAULT: "#1c1917",
          soft: "#292524",
        },
      },
      borderRadius: {
        // Driven by CSS vars so the restaurant can restyle the whole site from
        // the admin panel. `rounded-full` stays a true pill.
        md: "var(--radius-md)",
        lg: "var(--radius-lg)",
        xl: "var(--radius-md)",
        "2xl": "var(--radius-lg)",
        "3xl": "var(--radius-xl)",
      },
      fontFamily: {
        sans: ["var(--font-sans)", "system-ui", "sans-serif"],
        display: ["var(--font-display)", "Georgia", "serif"],
      },
      boxShadow: {
        // Driven by CSS vars so the admin panel can dial card depth up or down.
        card: "var(--shadow-card)",
        "card-hover": "var(--shadow-card-hover)",
      },
      keyframes: {
        "fade-up": {
          "0%": { opacity: "0", transform: "translateY(12px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
      },
      animation: {
        "fade-up": "fade-up 0.5s ease-out both",
      },
    },
  },
  plugins: [],
};

export default config;
