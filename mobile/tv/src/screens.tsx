// What the television draws.
//
// ⚠️ **Read from four metres away, by somebody who is not looking for it.**
// Every size here is chosen against that distance, not against a phone: the
// pairing code is the largest thing this app ever draws, because a manager
// reads it aloud across a dining room while typing it into a laptop.
//
// ⚠️ **Overscan.** Older sets cut 3–5% off every edge and there is no way to
// ask how much — so nothing important goes closer than `EDGE` to the border. A
// code that is 40 pixels from the edge is a code with a missing character on
// somebody's television.

import { useEffect, useState } from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
  useWindowDimensions,
} from "react-native";

import type { TVScreenSelf } from "@/lib/types";

const EDGE = 48;

const C = {
  bg: "#000b1c",
  ink: "#ffffff",
  muted: "#8ea3c0",
  accent: "#e2590d",
  line: "#1b2c48",
};

/** The first screen a television ever shows: which restaurant is this.
 *
 *  ⚠️ **One short word, and that is the whole design.** Whoever is installing
 *  knows the restaurant as "osh", not as "https://osh.keel.uz/api/v1" — and on
 *  a remote control every extra character is four button presses. The rule that
 *  turns one into the other is the shared `serverAddress`, so the phone apps,
 *  the Windows till and this set all resolve it identically. */
export function ServerScreen({
  onSubmit,
}: {
  onSubmit: (address: string) => boolean;
}) {
  const [value, setValue] = useState("");
  const [bad, setBad] = useState(false);
  // ⚠️ **Tracked by hand rather than read from Pressable's state.** React
  // Native's core `Pressable` reports `pressed`, and only the TV fork reports
  // `focused` — but a television is driven entirely by focus and never presses
  // anything until the moment it does. `onFocus`/`onBlur` are ordinary View
  // props and work here, which keeps this app on plain react-native.
  const [focused, setFocused] = useState(false);

  return (
    <View style={styles.center}>
      <Text style={styles.h1}>Keel TV</Text>
      <Text style={styles.lead}>Restoran manzilini kiriting</Text>
      <TextInput
        style={styles.input}
        value={value}
        onChangeText={(t) => {
          setValue(t);
          setBad(false);
        }}
        placeholder="osh"
        placeholderTextColor={C.muted}
        autoCapitalize="none"
        autoCorrect={false}
        // ⚠️ Focused on mount: a television has no pointer, and a field nobody
        // can reach with a remote is a screen that cannot be got past.
        autoFocus
        onSubmitEditing={() => {
          if (!onSubmit(value)) setBad(true);
        }}
      />
      <Pressable
        style={({ pressed }) => [
          styles.button,
          (focused || pressed) && styles.buttonFocused,
        ]}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        onPress={() => {
          if (!onSubmit(value)) setBad(true);
        }}
      >
        <Text style={styles.buttonText}>Davom etish</Text>
      </Pressable>
      <Text style={[styles.hint, bad && styles.bad]}>
        {bad
          ? "Manzil noto'g'ri — restoran nomini yozing, masalan: osh"
          : "Masalan: osh — yoki to'liq manzil, agar o'z domeningiz bo'lsa"}
      </Text>
    </View>
  );
}

/** The pairing code, and nothing else.
 *
 *  ⚠️ **No instructions for the guest, and none for the remote.** This screen
 *  hangs in a public room; whoever needs it is holding a laptop and has already
 *  been told what to do with it. What it does say is where to type the code,
 *  because that is the one thing somebody standing in front of it may not
 *  know. */
export function PairingScreen({
  code,
  expiresAt,
}: {
  code: string;
  expiresAt: number;
}) {
  const { width } = useWindowDimensions();
  // ⚠️ Sized from the screen rather than fixed: this app runs on a 32" set in a
  // corner shop and on a 65" in a mall, and a code that fits one is either
  // unreadable or clipped on the other.
  const codeSize = Math.max(72, Math.min(220, Math.round(width / 7)));
  const [left, setLeft] = useState(() =>
    Math.max(0, Math.round((expiresAt - Date.now()) / 1000)),
  );

  useEffect(() => {
    setLeft(Math.max(0, Math.round((expiresAt - Date.now()) / 1000)));
    const id = setInterval(() => {
      setLeft(Math.max(0, Math.round((expiresAt - Date.now()) / 1000)));
    }, 1000);
    return () => clearInterval(id);
  }, [expiresAt]);

  return (
    <View style={styles.center}>
      <Text style={styles.lead}>Bu ekranni ulash uchun</Text>
      {/* Spaced into two halves: six characters read as one word are
          transcribed with a character in the wrong place. */}
      <Text style={[styles.code, { fontSize: codeSize }]}>
        {code.slice(0, 3)} {code.slice(3)}
      </Text>
      <Text style={styles.lead}>
        Keel panelida: TV ekranlar → Ekran qo&apos;shish
      </Text>
      {/* ⚠️ The countdown is told, not hidden: a manager who sees the code
          change mid-typing needs to know that is normal, or the next thing they
          do is report a broken screen. */}
      <Text style={styles.hint}>
        Kod {left} soniyadan keyin yangilanadi — yangisini yozing
      </Text>
    </View>
  );
}

/** A paired screen with nothing to play yet.
 *
 *  ⚠️ **A placeholder that says which screen this is**, because the next thing
 *  somebody does after pairing is walk to the other televisions and pair those
 *  — and four identical black screens is how two of them end up named the same.
 *  The playlist replaces this in the next stage. */
export function PairedScreen({
  screen,
  offline,
}: {
  screen: TVScreenSelf | null;
  offline: boolean;
}) {
  return (
    <View style={styles.center}>
      <Text style={styles.h1}>{screen?.name ?? "Keel TV"}</Text>
      <Text style={styles.lead}>{screen?.branchName ?? ""}</Text>
      <Text style={styles.hint}>
        {offline
          ? // ⚠️ Said quietly and never as an error: the room is open, the
            // guests are eating, and nothing about a dropped wifi is theirs to
            // worry about. It is here at all because it is the first thing
            // somebody checks when the panel says a screen is silent.
            "Aloqa yo'q — ulanish tiklanganda o'zi sinxronlashadi"
          : "Ulandi. Kontent keyingi bosqichda."}
      </Text>
    </View>
  );
}

/** The one frame between launch and knowing anything. */
export function LoadingScreen() {
  return <View style={styles.center} />;
}

const styles = StyleSheet.create({
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: C.bg,
    padding: EDGE,
    gap: 16,
  },
  h1: { color: C.ink, fontSize: 44, fontWeight: "700" },
  lead: { color: C.muted, fontSize: 24 },
  code: {
    color: C.ink,
    fontWeight: "800",
    letterSpacing: 12,
    // A monospaced face so 8 and B cannot trade places at four metres.
    fontFamily: "monospace",
    marginVertical: 12,
  },
  hint: { color: C.muted, fontSize: 18, textAlign: "center" },
  bad: { color: C.accent },
  input: {
    minWidth: 420,
    borderWidth: 2,
    borderColor: C.line,
    borderRadius: 16,
    color: C.ink,
    fontSize: 32,
    paddingHorizontal: 20,
    paddingVertical: 12,
    textAlign: "center",
  },
  button: {
    marginTop: 8,
    paddingHorizontal: 32,
    paddingVertical: 14,
    borderRadius: 16,
    borderWidth: 2,
    borderColor: C.line,
  },
  // ⚠️ **The focus ring is the cursor.** A television has no pointer: the only
  // way to know which control the D-pad is on is that it looks different, and
  // "slightly different" is invisible from the far side of a room.
  buttonFocused: { borderColor: C.accent, backgroundColor: "#1a1005" },
  buttonText: { color: C.ink, fontSize: 24, fontWeight: "600" },
});
