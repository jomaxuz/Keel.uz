import { useMemo } from "react";
import { StyleSheet } from "react-native";

import { usePrefs } from "./prefs";
import type { Theme } from "./theme";

// The shapes every screen is built from.
//
// ⚠️ **Built from the theme rather than declared once at module level.** A
// `StyleSheet.create` at the top of a file is evaluated when the module loads,
// which is before anybody has chosen light or dark — so the colours would be
// frozen to whichever scheme happened to be first. Rebuilt per theme, memoised
// per theme, which is two objects for the life of the process.

export function useUI() {
  const { theme } = usePrefs();
  const s = useMemo(() => build(theme), [theme]);
  return { theme, s };
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
    num: { fontSize: 15, color: c.ink, fontVariant: ["tabular-nums"] },

    // ---- Surfaces ----
    card: {
      backgroundColor: c.surface,
      borderRadius: 16,
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
      // ⚠️ 52px of height at least: this is tapped by a thumb, in a moving
      // dining room, by somebody carrying something in the other hand.
      paddingVertical: 15,
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

    header: {
      paddingTop: 54,
      paddingHorizontal: 16,
      paddingBottom: 12,
      flexDirection: "row",
      alignItems: "center",
      justifyContent: "space-between",
      gap: 12,
    },
    list: { padding: 16, gap: 10, paddingBottom: 28 },
  });
}
