import { createContext, createElement, useCallback, useContext, useState, type ReactNode } from "react";
import { Modal, Pressable, StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// Saying something that has to be read.
//
// ⚠️ **A line of small text at the bottom of a screen is not a notification.**
// The waiter app learned this with "no printer took it": it appeared under a
// scrolling list, below the buttons, in the size everything else on the screen
// is not — so the one message that changes what somebody does next was the one
// nobody saw. It is a sheet: it interrupts, it says what happened, and it is
// dismissed deliberately.
//
// ⚠️ **Only for what changes the next action.** Everything that can be shown in
// place stays in place — the arrival gate is written on the card, beside the
// button it is holding shut, because that is where the courier is looking. A
// modal for each of those would be a modal people learn to tap through without
// reading, and then this one goes with them.

type Kind = "ok" | "warn" | "error";

interface Note {
  kind: Kind;
  title: string;
  body?: string;
}

const Ctx = createContext<((n: Note) => void) | null>(null);

export function NoticeProvider({ children }: { children: ReactNode }) {
  const [note, setNote] = useState<Note | null>(null);
  const show = useCallback((n: Note) => setNote(n), []);
  return createElement(
    Ctx.Provider,
    { value: show },
    children,
    note ? createElement(Sheet, { note, onClose: () => setNote(null) }) : null,
  );
}

export function useNotice() {
  const show = useContext(Ctx);
  if (!show) throw new Error("NoticeProvider yo'q");
  return show;
}

function Sheet({ note, onClose }: { note: Note; onClose: () => void }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const tone =
    note.kind === "error"
      ? theme.danger
      : note.kind === "warn"
        ? theme.warn
        : theme.ok;
  const icon =
    note.kind === "error"
      ? "alert-circle"
      : note.kind === "warn"
        ? "alert-triangle"
        : "check-circle";

  return (
    <Modal transparent animationType="fade" onRequestClose={onClose}>
      {/* ⚠️ The backdrop dismisses. A message with one way out is a message
          somebody is trapped by, and this one interrupts a person carrying
          plates. */}
      <Pressable style={local.backdrop} onPress={onClose}>
        <Pressable
          style={[local.card, { backgroundColor: theme.surface }]}
          onPress={(e) => e.stopPropagation()}
        >
          <View style={[local.badge, { backgroundColor: tone + "22" }]}>
            <Feather name={icon} size={26} color={tone} />
          </View>
          <Text style={[s.h2, { textAlign: "center" }]}>{note.title}</Text>
          {note.body ? (
            <Text style={[s.muted, { textAlign: "center" }]}>{note.body}</Text>
          ) : null}
          <Pressable style={[s.primary, { alignSelf: "stretch" }]} onPress={onClose}>
            <Text style={s.primaryText}>{t.notice.ok}</Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const local = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: "rgba(0,0,0,0.5)",
    alignItems: "center",
    justifyContent: "center",
    padding: 28,
  },
  card: {
    width: "100%",
    maxWidth: 340,
    borderRadius: 22,
    padding: 22,
    gap: 12,
    alignItems: "center",
  },
  badge: {
    width: 56,
    height: 56,
    borderRadius: 28,
    alignItems: "center",
    justifyContent: "center",
  },
});
