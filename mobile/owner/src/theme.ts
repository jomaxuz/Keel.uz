import { useColorScheme } from "react-native";

// Colours, in two schemes.
//
// ⚠️ **Named by role, not by shade.** `surface` and `ink` mean the same thing in
// both schemes and different things to the eye — which is the only way one set
// of screens can be written once and read correctly in both. A screen that
// referred to "the light grey" would have to be edited for the dark one, and
// the edit that gets forgotten is the one nobody looks at in daylight.
//
// ⚠️ **The brand orange does not flip.** It is the same colour on the till, on
// the site and on the paper receipt; a phone that lightened it in the dark
// would be the one Keel that is a different orange, and a waiter comparing a
// screen against a printed check would see two products.

export type Scheme = "light" | "dark";

export interface Theme {
  scheme: Scheme;
  bg: string;
  surface: string;
  surfaceAlt: string;
  ink: string;
  inkSoft: string;
  muted: string;
  line: string;
  accent: string;
  accentSoft: string;
  onAccent: string;
  danger: string;
  /** Better than yesterday, and worse. ⚠️ Not decoration: this app is read at
   *  a glance and the only question it answers is which direction a number
   *  moved. */
  ok: string;
  warn: string;
  warnSoft: string;
  /** The brand's own navy — the icon's background, and the one place the app
   *  shows its own identity rather than the restaurant's. */
  navy: string;
}

const light: Theme = {
  scheme: "light",
  bg: "#faf7f2",
  surface: "#ffffff",
  surfaceAlt: "#f3efe8",
  ink: "#2a2521",
  inkSoft: "#57504a",
  muted: "#8a8178",
  line: "#e7e0d6",
  accent: "#e2590d",
  accentSoft: "#fdf0e6",
  onAccent: "#ffffff",
  danger: "#c0392b",
  ok: "#0f8a5f",
  warn: "#b7791f",
  warnSoft: "#fdf3e0",
  navy: "#000b1c",
};

// ⚠️ Not the light palette inverted. A dining room at night is dim and the
// phone is the brightest thing in a waiter's hand — so the background is a warm
// near-black rather than pure black (which smears on OLED while scrolling) and
// the text is off-white rather than white (which glares).
const dark: Theme = {
  scheme: "dark",
  bg: "#14110e",
  surface: "#1e1a16",
  surfaceAlt: "#272220",
  ink: "#f2ece4",
  inkSoft: "#c9c0b6",
  muted: "#948a80",
  line: "#332c26",
  accent: "#e2590d",
  accentSoft: "#3a2313",
  onAccent: "#ffffff",
  danger: "#e56a5c",
  ok: "#3ec98e",
  warn: "#e0b062",
  warnSoft: "#332616",
  navy: "#000b1c",
};

export const THEMES: Record<Scheme, Theme> = { light, dark };

/** What the phone itself is set to. `null` while it is unknown, which React
 *  Native reports briefly on some Android versions — treated as light rather
 *  than flashing dark at somebody. */
export function useSystemScheme(): Scheme {
  return useColorScheme() === "dark" ? "dark" : "light";
}
