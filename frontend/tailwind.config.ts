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
        // ⚠️ **The second colour, and it is a surface rather than an accent.**
        // Every shop reference has a primary that sells (the cart button) and a
        // secondary that organises — the panel a category list sits on. `ink`
        // is what stays readable on it, computed from its luminance in
        // theme-css.ts rather than chosen: a yellow panel needs black words and
        // a deep green one needs white, and a designer picking both by hand
        // gets it right for the colour in front of them and wrong for the next
        // customer's.
        accent: {
          DEFAULT: v("--accent"),
          ink: v("--accent-ink"),
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
        // ⚠️ **Added late, and it was already in use.** `text-danger` appears on
        // every error line in the till, the admin panel and the dialogs — and
        // Tailwind generates nothing for a colour that was never declared, so
        // all of them have been rendering in ordinary ink. Nothing looked
        // broken: the sentence was there, correct, and quiet.
        danger: v("--danger"),
        // Always-dark surfaces (hero, site footer, dark CTA) — identical in
        // both themes, they carry white text by design.
        charcoal: {
          DEFAULT: "#1c1917",
          soft: "#292524",
        },
        // ⚠️ **Keel's own colour, and deliberately not `brand`.** `brand` is the
        // restaurant's accent, chosen by the owner in the panel and different in
        // every install — our mark drawn in it would be a different logo per
        // customer. Fixed here, taken from logos/keel-mark.svg.
        //
        // `deep` is the same amber a step darker: at #F5A524 the mark on the
        // till's near-white lock screen is a pale smear, and the one place it
        // appears is the screen the machine sits on all day.
        keel: {
          DEFAULT: "#F5A524",
          deep: "#D2870F",
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
        // Our own wordmark only — never the restaurant's copy, which follows
        // --font-sans/--font-display and is the owner's to change.
        poppins: ["var(--font-poppins)", "system-ui", "sans-serif"],
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
