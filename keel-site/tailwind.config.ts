import type { Config } from "tailwindcss";

// Keel's own identity — deliberately not the tenant palette.
//
// A restaurant picks its own brand colour for its own site; this palette is
// only ever seen on keel.uz and in the dashboard. The one place the two meet
// is the "Powered by Keel" line in a customer's footer, and that is drawn in
// currentColor precisely so none of this leaks onto somebody else's page.
const config: Config = {
  darkMode: "class",
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        // Deep marine — the hull below the waterline.
        hull: {
          950: "#05101A",
          900: "#0A1A28",
          800: "#0F2536",
          700: "#16344A",
          600: "#1F4763",
        },
        // The signal light. Warm on purpose: an all-blue marine palette reads
        // cold, and the product is sold to people who run warm little shops.
        signal: {
          400: "#FFC46B",
          500: "#F5A524",
          600: "#D2870F",
        },
        // Semantic surfaces, switched by the dark class in globals.css.
        page: "rgb(var(--page) / <alpha-value>)",
        surface: "rgb(var(--surface) / <alpha-value>)",
        raised: "rgb(var(--raised) / <alpha-value>)",
        ink: "rgb(var(--ink) / <alpha-value>)",
        "ink-soft": "rgb(var(--ink-soft) / <alpha-value>)",
        "ink-muted": "rgb(var(--ink-muted) / <alpha-value>)",
        line: "rgb(var(--line) / <alpha-value>)",
        "line-strong": "rgb(var(--line-strong) / <alpha-value>)",
      },
      fontFamily: {
        sans: ["var(--font-sans)", "system-ui", "sans-serif"],
        display: ["var(--font-display)", "var(--font-sans)", "sans-serif"],
      },
      borderRadius: { xl: "14px", "2xl": "20px", "3xl": "28px" },
      maxWidth: { page: "1180px" },
      keyframes: {
        rise: {
          from: { opacity: "0", transform: "translateY(14px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        drift: {
          "0%,100%": { transform: "translateY(0)" },
          "50%": { transform: "translateY(-10px)" },
        },
      },
      animation: {
        rise: "rise .7s cubic-bezier(.2,.7,.3,1) both",
        drift: "drift 9s ease-in-out infinite",
      },
    },
  },
  plugins: [],
};

export default config;
