import { forwardRef } from "react";
import {
  Platform,
  Pressable,
  type PressableProps,
  type View,
} from "react-native";

import { useUI } from "./ui";

// How a tap is answered.
//
// ⚠️ **The same complaint as the till's, on a different machine.** The counter
// screens had it first: somebody presses, nothing visibly happens, so they
// press again — and the fault reported is "it freezes", when what is missing
// is the answer to the first press. A plain `Pressable` with a static style
// gives no feedback at all on Android: no ripple, no dimming, nothing until
// the state it changed comes back from a server on a restaurant's wifi.
//
// ⚠️ **Feedback is not the same thing as the work being done**, and separating
// them is the whole point. The ripple is drawn by the platform on the UI
// thread, so it appears while JavaScript is still busy — which is exactly the
// moment a person needs to be told their tap landed.
//
// ⚠️ **A default hit area.** These apps are used walking, one-handed, with a
// tray in the other. Eight points around a control is the difference between a
// plus that works and one that "sometimes does not".
export const Tap = forwardRef<View, PressableProps>(function Tap(
  { style, android_ripple, hitSlop, ...rest },
  ref,
) {
  const { theme } = useUI();
  return (
    <Pressable
      ref={ref}
      hitSlop={hitSlop ?? 8}
      android_ripple={
        android_ripple ?? { color: theme.accentSoft, foreground: true }
      }
      // ⚠️ iOS gets dimming instead: it has no ripple, and a control that
      // answers on one platform and not the other is a control somebody learns
      // to distrust on both.
      style={(state) => [
        typeof style === "function" ? style(state) : style,
        state.pressed && Platform.OS !== "android" ? { opacity: 0.7 } : null,
      ]}
      {...rest}
    />
  );
});
