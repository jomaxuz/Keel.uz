import { useState } from "react";
import { Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";
import * as Location from "expo-location";

import { api, ApiError } from "@/lib/api";
import type { Shift } from "@/lib/types";

import { usePrefs } from "./prefs";
import { Tap } from "./press";
import { useUI } from "./ui";

// Starting and ending a shift, from the thing that has the GPS.
//
// ⚠️ **The phone is where this belongs and it was the last screen to get it.**
// The server requires a position and checks it against the branch's own
// (`geofenceBlocked`) — so an employee was opening their shift on some other
// screen and then working from this one. The device that can answer "where am
// I" is in their hand.
//
// ⚠️ **Permission is asked when the button is pressed, not at launch.** A
// location prompt on first run, before anybody knows what the app is for, is
// answered "no" — and on both platforms a refused location is awkward to
// recover. Asked at the moment it is obviously needed, it is a question with a
// visible reason.

export function ClockButton({
  open,
  onChanged,
}: {
  /** The shift already running, if there is one. */
  open: boolean;
  onChanged: (s: Shift | null) => void;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function punch() {
    setBusy(true);
    setError("");
    try {
      const perm = await Location.requestForegroundPermissionsAsync();
      if (!perm.granted) {
        // ⚠️ Named plainly rather than as a failure: refusing is a choice, and
        // the way back is the system settings, which is what the sentence says.
        setError(t.clock.needLocation);
        return;
      }
      // ⚠️ **Balanced accuracy, not the highest.** The branch's radius is 50
      // metres — chosen because a phone's GPS is 10–30 outdoors and worse
      // inside — so the extra seconds the highest setting spends do not change
      // the answer, and they are spent with somebody standing at a door.
      const pos = await Location.getCurrentPositionAsync({
        accuracy: Location.Accuracy.Balanced,
      });
      const shift = await api.staffClock(open ? "out" : "in", {
        lat: pos.coords.latitude,
        lng: pos.coords.longitude,
        accuracy: pos.coords.accuracy ?? 0,
      });
      onChanged(open ? null : shift);
    } catch (e) {
      // The server's own words: "you are 400 m from the branch" is a sentence
      // somebody can act on, and it is the one refusal that is not a fault.
      setError(e instanceof ApiError ? e.message : t.clock.failed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <View style={{ gap: 8 }}>
      <Tap
        style={[
          s.primary,
          open ? { backgroundColor: theme.surface, borderWidth: 1, borderColor: theme.line } : null,
        ]}
        disabled={busy}
        onPress={() => void punch()}
      >
        <Feather
          name={open ? "log-out" : "log-in"}
          size={18}
          color={open ? theme.ink : theme.onAccent}
        />
        <Text style={[s.primaryText, open ? { color: theme.ink } : null]}>
          {busy ? t.common.loading : open ? t.clock.out : t.clock.in}
        </Text>
      </Tap>
      {error !== "" && <Text style={s.error}>{error}</Text>}
    </View>
  );
}
