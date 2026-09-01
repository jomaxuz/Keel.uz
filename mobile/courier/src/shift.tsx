import { Pressable, StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import type { CourierStatus } from "@/lib/types";

import { usePrefs } from "./prefs";
import type { Tracking } from "./tracking";
import { useUI } from "./ui";

// The top of the orders screen: am I working, and does the restaurant know
// where I am.
//
// ⚠️ **Both questions in one card, because they are one question.** Being on
// shift is what starts the location stream, and the location is what opens the
// "delivered" button — a courier who saw only the switch would have no way to
// tell "no orders tonight" from "the dispatcher's map lost me an hour ago".

export function ShiftCard({
  status,
  onStatus,
  tracking,
}: {
  status: CourierStatus;
  onStatus: (s: CourierStatus) => void;
  tracking: Tracking;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();

  const options: { key: CourierStatus; label: string; hint: string; icon: keyof typeof Feather.glyphMap }[] = [
    { key: "off", label: t.shift.off, hint: t.shift.offHint, icon: "power" },
    { key: "free", label: t.shift.free, hint: t.shift.freeHint, icon: "check-circle" },
    { key: "busy", label: t.shift.busy, hint: t.shift.busyHint, icon: "truck" },
  ];

  return (
    <View style={[s.card, { gap: 12 }]}>
      <View style={local.titleRow}>
        <Feather name="clock" size={14} color={theme.muted} />
        <Text style={s.muted}>{t.shift.title}</Text>
      </View>

      <View style={local.switcher}>
        {options.map((o) => {
          const on = o.key === status;
          // ⚠️ Colour by meaning rather than by position: "off" is the state
          // that stops work arriving, and it reads as a warning for exactly as
          // long as it is true.
          const tint =
            o.key === "off" ? theme.warn : o.key === "free" ? theme.ok : theme.accent;
          return (
            <Pressable
              key={o.key}
              onPress={() => onStatus(o.key)}
              style={[
                local.option,
                {
                  borderColor: on ? tint : theme.line,
                  backgroundColor: on ? tint : theme.surface,
                },
              ]}
            >
              <Feather
                name={o.icon}
                size={17}
                color={on ? theme.onAccent : theme.muted}
              />
              <Text
                style={[
                  local.optionLabel,
                  { color: on ? theme.onAccent : theme.ink },
                ]}
              >
                {o.label}
              </Text>
              <Text
                style={[
                  local.optionHint,
                  { color: on ? "rgba(255,255,255,0.85)" : theme.muted },
                ]}
              >
                {o.hint}
              </Text>
            </Pressable>
          );
        })}
      </View>

      {status === "off" ? (
        <Text style={s.muted}>{t.shift.startHint}</Text>
      ) : (
        <LocationRow tracking={tracking} />
      )}
    </View>
  );
}

/** What the location stream is doing, in one line a courier can act on. */
function LocationRow({ tracking }: { tracking: Tracking }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();

  // ⚠️ **Permission first, and as a button rather than as an error.** "Location
  // denied" with nothing to press is a sentence that ends the shift; the way
  // back is a system prompt this app can still raise while `canAskAgain` holds,
  // and the phone's settings after that — which is what the hint says.
  if (tracking.state !== "granted") {
    const denied = tracking.state === "denied";
    return (
      <View
        style={[
          local.geo,
          { backgroundColor: denied ? theme.warnSoft : theme.surfaceAlt },
        ]}
      >
        <Feather name="map-pin" size={16} color={theme.warn} />
        <View style={{ flex: 1, gap: 2 }}>
          <Text style={[s.body, { fontWeight: "600" }]}>{t.geo.denied}</Text>
          <Text style={s.muted}>{t.geo.deniedHint}</Text>
        </View>
        {!denied && (
          <Pressable
            style={[local.allow, { borderColor: theme.accent }]}
            onPress={() => void tracking.request()}
          >
            <Text style={[s.body, { color: theme.accent, fontWeight: "600" }]}>
              {t.geo.allow}
            </Text>
          </Pressable>
        )}
      </View>
    );
  }

  const time =
    tracking.lastSentAt !== null
      ? new Date(tracking.lastSentAt).toLocaleTimeString(undefined, {
          hour: "2-digit",
          minute: "2-digit",
        })
      : null;

  return (
    <View style={[local.geo, { backgroundColor: theme.surfaceAlt }]}>
      {/* A filled dot rather than a tick: this is a state that is true right
          now, and it stops being true while nobody is looking at it. */}
      <View
        style={[
          local.dot,
          { backgroundColor: tracking.live ? theme.ok : theme.warn },
        ]}
      />
      <View style={{ flex: 1, gap: 2 }}>
        <Text style={s.body}>
          {tracking.live ? t.geo.on : `${t.geo.on} · ${t.geo.waiting}`}
        </Text>
        <Text style={s.muted}>
          {[
            time ? t.geo.lastSent(time) : null,
            tracking.pending > 0 ? t.geo.queued(tracking.pending) : null,
          ]
            .filter(Boolean)
            .join(" · ") || " "}
        </Text>
        {/* ⚠️ **Which of the two modes this is, said plainly.** With the
            background service running the phone can go in a pocket; without it
            the screen has to stay on, and a courier who does not know which
            they have will find out at a door with a shut button. */}
        <Text style={s.muted}>
          {tracking.background ? t.geo.inBackground : t.geo.keepOpen}
        </Text>
      </View>
    </View>
  );
}

const local = StyleSheet.create({
  titleRow: { flexDirection: "row", alignItems: "center", gap: 6 },
  switcher: { flexDirection: "row", gap: 8 },
  option: {
    flex: 1,
    borderWidth: 1,
    borderRadius: 14,
    paddingVertical: 12,
    paddingHorizontal: 8,
    alignItems: "center",
    gap: 4,
    // ⚠️ Tall enough to be hit with a glove on. This is the control a courier
    // presses at a kerb, and the three sit side by side.
    minHeight: 92,
    justifyContent: "center",
  },
  optionLabel: { fontSize: 14, fontWeight: "700", textAlign: "center" },
  optionHint: { fontSize: 11, textAlign: "center" },
  geo: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    borderRadius: 14,
    paddingHorizontal: 12,
    paddingVertical: 12,
  },
  dot: { width: 10, height: 10, borderRadius: 5 },
  allow: {
    borderWidth: 1,
    borderRadius: 999,
    paddingHorizontal: 14,
    paddingVertical: 8,
  },
});
