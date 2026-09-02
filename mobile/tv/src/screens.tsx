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
  initial = "",
  unreachable = false,
  answered = false,
}: {
  onSubmit: (address: string) => boolean;
  /** ⚠️ Comes back **filled in** after a failed attempt. Retyping a word on a
   *  D-pad is four button presses a character, and the likeliest fix is one
   *  wrong letter — an empty field would make the correction cost more than
   *  the mistake. */
  initial?: string;
  /** The address was accepted and did not lead to a working Keel server. */
  unreachable?: boolean;
  /** Whether the server answered at all. ⚠️ It changes who has to act: an
   *  answer means the address is right and that restaurant's server is behind
   *  — nothing on this remote control can fix that. */
  answered?: boolean;
}) {
  const [value, setValue] = useState(initial);
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
      <Text style={[styles.hint, (bad || unreachable) && styles.bad]}>
        {bad
          ? "Manzil noto'g'ri — restoran nomini yozing, masalan: osh"
          : unreachable
            ? answered
              ? // The server is there and does not know about televisions yet.
                // Said plainly, because the next step is ours and not theirs.
                "Server javob berdi, lekin TV bo'limi yo'q — restoran serveri yangilanishi kerak."
              : // ⚠️ The causes in the order they actually happen, because the
                // person reading this is standing on a chair with a remote and
                // has no other way to find out which one it is.
                "Bu manzilda javob yo'q. Nomini tekshiring, internetni tekshiring — ulanish tiklansa ekran o'zi davom etadi."
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
          do is report a broken screen.

          ⚠️ And it counts the *real* window. It used to show the server's
          ninety seconds while the app quietly fetched a new code every ten —
          so the number on the wall was wrong by a factor of nine, and the first
          person to try pairing a television could not finish typing before it
          moved. */}
      <Text style={styles.hint}>
        {left > 0
          ? `Kod ${left} soniyadan keyin yangilanadi`
          : "Kod yangilanmoqda…"}
      </Text>
    </View>
  );
}

/** A paired screen with nothing to play.
 *
 *  ⚠️ **A placeholder that says which screen this is**, because the next thing
 *  somebody does after pairing is walk to the other televisions and pair those
 *  — and four identical black screens is how two of them end up named the same.
 *  Once the branch has a playlist this is replaced by it; what is left here is
 *  the three cases where there is nothing to draw, and each of them names the
 *  person who can fix it. */
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
          : screen?.mode === "board"
            ? "Buyurtma tablosi keyingi bosqichda."
            : // ⚠️ It names where to go, because the person reading it is
              // standing in front of the television having just paired it and
              // the answer is on a laptop in the back office.
              "Kontent yo'q — Keel panelida: TV ekranlar → Kontent"}
      </Text>
    </View>
  );
}

/** The frame between launch and knowing anything.
 *
 *  ⚠️ **It says something, and that is not decoration.** This drew an empty
 *  view once, and the first time the app got stuck here — one missing call
 *  after the address was typed — the result was a black rectangle on a wall
 *  with no code, no message and nothing to press. A blank screen and a crashed
 *  app are the same picture; a screen with the product's name on it is at least
 *  a screen that is running. */
export function LoadingScreen() {
  return (
    <View style={styles.center}>
      <Text style={styles.h1}>Keel TV</Text>
      <Text style={styles.lead}>Ulanmoqda…</Text>
    </View>
  );
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
