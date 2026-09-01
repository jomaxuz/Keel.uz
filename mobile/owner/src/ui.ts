import { useMemo } from "react";
import { StyleSheet } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { usePrefs } from "./prefs";
import type { Theme } from "./theme";

// The shapes every screen is built from.
//
// ⚠️ **Built from the theme rather than declared once at module level.** A
// `StyleSheet.create` at the top of a file is evaluated when the module loads,
// which is before anybody has chosen light or dark — so the colours would be
// frozen to whichever scheme happened to be first. Rebuilt per theme, memoised
// per theme, which is two objects for the life of the process.
//
// ⚠️ **This app is read standing up, between two other things.** Not in a
// dining room and not on a bike — in a car, in a queue, at a table with guests.
// So the numbers are large, the rows are calm, and nothing here is denser than
// it has to be: an owner who has to squint at a figure will stop checking it.

export function useUI() {
  const { theme } = usePrefs();
  const s = useMemo(() => build(theme), [theme]);
  // ⚠️ Handed out with the styles because every screen already asks for those,
  // and a screen that has to remember a second import is a screen that forgets
  // it — which is how the bottom of four apps ended up under Android's buttons.
  return { theme, s, bottom: useBottomInset() };
}

/** How much room Android's own buttons need at the bottom of this screen.
 *
 * ⚠️ **A tap meant for our button lands on the system's.** These apps draw
 * edge to edge, so the layout extends underneath the navigation bar: a control
 * placed at the bottom of a sheet or a screen is *behind* Back and Home, and
 * the finger aimed at "send to the kitchen" goes back a screen with the order
 * unsent. Reported from real phones, on the support chat first.
 *
 * ⚠️ **Measured, never guessed.** The sheets here carried `paddingBottom: 34`,
 * which happens to clear a gesture pill and does not clear a three-button bar —
 * so the bug was invisible on exactly the phones the app was written on. The
 * system reports the strip it actually drew; the floor of 12 is for the phones
 * that report nothing, where the number is spacing rather than safety.
 */
export function useBottomInset(): number {
  return Math.max(useSafeAreaInsets().bottom, 12);
}


function build(c: Theme) {
  return StyleSheet.create({
    screen: { flex: 1, backgroundColor: c.bg },
    centered: {
      flex: 1,
      alignItems: "center",
      justifyContent: "center",
      padding: 24,
      gap: 12,
      backgroundColor: c.bg,
    },

    // ---- Type ----
    h1: { fontSize: 24, fontWeight: "700", color: c.ink },
    h2: { fontSize: 17, fontWeight: "600", color: c.ink },
    body: { fontSize: 15, color: c.ink },
    soft: { fontSize: 14, color: c.inkSoft },
    muted: { fontSize: 13, color: c.muted },
    error: { fontSize: 13, color: c.danger, textAlign: "center" },
    link: { fontSize: 15, color: c.accent, fontWeight: "500" },
    // ⚠️ Tabular figures wherever a number is read against another number: a
    // price list whose digits shift width is a list that cannot be scanned.
    num: { fontSize: 15, color: c.ink, fontVariant: ["tabular-nums"] },
    money: {
      fontSize: 17,
      fontWeight: "700",
      color: c.ink,
      fontVariant: ["tabular-nums"],
    },
    // ⚠️ **The headline figures live here, with a colour, and that is the
    // whole point.** They were declared in each screen's own module-level
    // `StyleSheet.create`, where a theme colour cannot be reached — so they
    // had none, React Native defaulted them to black, and the one number this
    // app exists to show was invisible on a dark screen. Reported from a real
    // phone; the light theme is where it was written and where it was tested.
    big: {
      fontSize: 30,
      fontWeight: "800",
      color: c.ink,
      fontVariant: ["tabular-nums"],
    },
    mid: {
      fontSize: 20,
      fontWeight: "700",
      color: c.ink,
      fontVariant: ["tabular-nums"],
    },

    // ---- Surfaces ----
    card: {
      backgroundColor: c.surface,
      borderRadius: 18,
      borderWidth: 1,
      borderColor: c.line,
      padding: 14,
    },
    row: {
      flexDirection: "row",
      alignItems: "center",
      gap: 12,
      backgroundColor: c.surface,
      borderRadius: 14,
      borderWidth: 1,
      borderColor: c.line,
      paddingHorizontal: 14,
      paddingVertical: 14,
    },

    // ---- Controls ----
    input: {
      width: "100%",
      maxWidth: 340,
      borderWidth: 1,
      borderColor: c.line,
      borderRadius: 14,
      paddingHorizontal: 16,
      paddingVertical: 14,
      fontSize: 16,
      backgroundColor: c.surface,
      color: c.ink,
    },
    primary: {
      backgroundColor: c.accent,
      borderRadius: 14,
      paddingHorizontal: 24,
      paddingVertical: 15,
      alignItems: "center",
      flexDirection: "row",
      justifyContent: "center",
      gap: 8,
    },
    primaryText: { color: c.onAccent, fontSize: 16, fontWeight: "600" },
    ghost: {
      borderRadius: 14,
      borderWidth: 1,
      borderColor: c.line,
      backgroundColor: c.surface,
      paddingHorizontal: 18,
      paddingVertical: 14,
      alignItems: "center",
      flexDirection: "row",
      justifyContent: "center",
      gap: 8,
    },
    ghostText: { color: c.ink, fontSize: 15, fontWeight: "600" },

    header: {
      paddingTop: 54,
      paddingHorizontal: 16,
      paddingBottom: 12,
      flexDirection: "row",
      alignItems: "center",
      justifyContent: "space-between",
      gap: 12,
    },
    list: { padding: 16, gap: 14, paddingBottom: 32 },

    // ⚠️ An empty list is a state, not an absence. A courier who sees nothing
    // needs to know whether that is "no orders yet" or "the app is broken", and
    // a blank screen answers the second.
    empty: {
      alignItems: "center",
      gap: 8,
      paddingVertical: 44,
      paddingHorizontal: 24,
      borderRadius: 18,
      borderWidth: 1,
      borderStyle: "dashed",
      borderColor: c.line,
    },
  });
}
