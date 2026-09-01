import { useEffect, useRef, useState } from "react";
import { AppState, Pressable, StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { usePrefs } from "./prefs";
import { Tap } from "./press";
import { useUI } from "./ui";

// Opened with no network.
//
// ⚠️ **This screen exists because the login screen was standing in for it.** A
// launch asks the server who is signed in; when the request never arrives the
// answer used to be "signed out", so a waiter in a basement, on a dead wifi or
// out of data met a password field. They type the right password, it fails, and
// the app says "could not sign in" — so they type it again, blaming themselves
// for a network they cannot see. None of that is fixable by signing in, and a
// screen that offers a way out it does not have is worse than one that says
// plainly what is wrong.
//
// ⚠️ **It retries by itself.** The ordinary way out is the network coming back
// — walking out of the cellar, the router rebooting — and a screen that only
// cleared on a tap would keep somebody locked out of a shift they are standing
// in the middle of. The button stays because five seconds is a long time when
// you are holding a tray.

/** How often the launch question is asked again. Short enough that walking
 *  back into signal clears the screen while the phone is still in the hand,
 *  long enough to be nothing on a battery. */
const RETRY_MS = 5000;

export function OfflineScreen({
  address,
  onRetry,
}: {
  /** Which restaurant this phone is trying to reach. ⚠️ Shown because the
   *  other cause of this screen is a wrong or dead address, and the person who
   *  can tell is the one reading it. */
  address: string;
  onRetry: () => void;
}) {
  const { t } = usePrefs();
  const { theme, s, bottom } = useUI();
  const [tries, setTries] = useState(0);
  // ⚠️ Held in a ref: the timer below is set up once, and a callback captured
  // in it would go on calling the first render's `onRetry` for ever.
  const latest = useRef(onRetry);
  latest.current = onRetry;

  useEffect(() => {
    const timer = setInterval(() => {
      setTries((n) => n + 1);
      latest.current();
    }, RETRY_MS);
    // ⚠️ And immediately when the app comes back to the front: a phone brought
    // out of a pocket in a place with signal should not wait out a tick.
    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active") latest.current();
    });
    return () => {
      clearInterval(timer);
      sub.remove();
    };
  }, []);

  return (
    <View style={[s.centered, { paddingBottom: bottom + 24 }]}>
      <View style={[local.badge, { backgroundColor: theme.accentSoft }]}>
        <Feather name="wifi-off" size={30} color={theme.accent} />
      </View>
      <Text style={[s.h1, { textAlign: "center" }]}>{t.offline.title}</Text>
      <Text style={[s.muted, { textAlign: "center" }]}>{t.offline.body}</Text>
      <View style={local.chip}>
        <Feather name="home" size={13} color={theme.muted} />
        <Text style={s.muted}>{address}</Text>
      </View>
      <Tap
        style={[s.primary, { alignSelf: "stretch", maxWidth: 340 }]}
        onPress={() => {
          setTries((n) => n + 1);
          onRetry();
        }}
      >
        <Feather name="refresh-cw" size={18} color={theme.onAccent} />
        <Text style={s.primaryText}>{t.common.retry}</Text>
      </Tap>
      {/* ⚠️ Said, so the screen does not look frozen: it is checking, and a
          person watching a still screen assumes it is not. */}
      <Text style={s.muted}>{t.offline.retrying(tries)}</Text>
    </View>
  );
}

const local = StyleSheet.create({
  badge: {
    width: 68,
    height: 68,
    borderRadius: 34,
    alignItems: "center",
    justifyContent: "center",
  },
  chip: { flexDirection: "row", alignItems: "center", gap: 6 },
});
