import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { api } from "@/lib/api";
import { HELP } from "@/lib/help/articles";
import { searchHelp } from "@/lib/help/search";
import type { SupportMessage, SupportThread } from "@/lib/types";

import { usePrefs } from "./prefs";
import { Tap } from "./press";
import { useUI } from "./ui";

// Talking to us, from the phone.
//
// ⚠️ **The help is searched before an operator is offered**, exactly as in the
// panel. Most support questions have been asked before and are answered in a
// paragraph; a screen that opens straight onto "write to us" turns every one of
// them into a person waiting for a person. The escalation is one press away and
// never hidden — somebody who has read the article and is still stuck must not
// have to search their way out of the help.
//
// ⚠️ **The articles are the panel's own file, imported rather than copied.**
// They are what makes the assistant answer at all — it is given only these —
// and a second copy written for the phone would be the copy that stops
// describing this build. It costs about 25 KB, which is why this is affordable
// where importing the panel's dictionaries was not.
//
// ⚠️ **Polled, not a socket.** The panel holds a WebSocket open because it sits
// on a desk all day. A phone suspends the moment it goes in a pocket, and a
// socket that dies silently is worse than no socket: the screen keeps saying
// "connected" and nothing arrives. This screen is open for a minute at a time,
// so it asks every five seconds while it is, and stops when it is closed.

const POLL = 5_000;

export function SupportScreen({ onClose }: { onClose: () => void }) {
  const { t, lang } = usePrefs();
  const { theme, s } = useUI();
  // ⚠️ **The composer sits at the bottom of an edge-to-edge screen, which is
  // where Android draws its own back/home buttons.** Without this inset the
  // send button and the text field are underneath them: aiming at the input
  // presses the system bar instead, and the app closes. Reported from a real
  // phone — an emulator with gesture navigation has no bar to collide with.
  const insets = useSafeAreaInsets();

  const [ask, setAsk] = useState("");
  const [opened, setOpened] = useState<string | null>(null);
  const articles = HELP[lang] ?? HELP.uz;
  const hits = searchHelp(articles, ask);

  const [threads, setThreads] = useState<SupportThread[] | null>(null);
  // `null` is the list, `""` is a question not sent yet, an id is a
  // conversation. ⚠️ The empty string matters: the escalation opens the
  // composer before anything exists to load, and asking the server about it
  // would be a 400 the owner reads as a failure.
  const [active, setActive] = useState<string | null>(null);
  const [messages, setMessages] = useState<SupportMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [failed, setFailed] = useState(false);

  const loadThreads = useCallback(async () => {
    try {
      const res = await api.supportThreads();
      setThreads(res.threads ?? []);
    } catch {
      // ⚠️ An empty list rather than an error. This screen is reached when
      // something else is already wrong, and "could not load your history" on
      // top of that is a second problem in front of the first.
      setThreads([]);
    }
  }, []);

  useEffect(() => {
    void loadThreads();
  }, [loadThreads]);

  const openThread = useCallback(async (id: string) => {
    setActive(id);
    setMessages([]);
    if (!id) return;
    try {
      const res = await api.supportThread(id);
      setMessages(res.messages ?? []);
    } catch {
      setMessages([]);
    }
  }, []);

  // ⚠️ The poll is tied to the open conversation, so closing it stops the
  // requests — an app left on this screen in a pocket must not spend a
  // restaurant's data all evening.
  const activeRef = useRef<string | null>(null);
  activeRef.current = active;
  useEffect(() => {
    if (!active) return;
    const timer = setInterval(() => {
      const id = activeRef.current;
      if (!id) return;
      api
        .supportThread(id)
        .then((r) => setMessages(r.messages ?? []))
        .catch(() => {});
    }, POLL);
    return () => clearInterval(timer);
  }, [active]);

  async function send() {
    const text = draft.trim();
    if (text === "" || sending) return;
    setSending(true);
    setFailed(false);
    try {
      // ⚠️ The same ranked articles the panel sends: the assistant answers
      // only from these, so a phone that sent none would be a phone whose
      // questions always wait for a human.
      const candidates = searchHelp(articles, text)
        .slice(0, 4)
        .map((h) => ({ title: h.article.title, body: h.article.body }));
      const res = await api.supportAsk({
        threadId: active || undefined,
        text,
        lang,
        articles: candidates,
      });
      setDraft("");
      setActive(res.threadId);
      // Shown immediately rather than waiting for the next poll: the line the
      // owner just pressed send on has to appear, or they press send again.
      setMessages((m) => [...m, res.message]);
      void loadThreads();
    } catch {
      setFailed(true);
    } finally {
      setSending(false);
    }
  }

  return (
    <Modal animationType="slide" onRequestClose={onClose}>
      <KeyboardAvoidingView
        style={s.screen}
        behavior={Platform.OS === "ios" ? "padding" : undefined}
      >
        <View style={s.header}>
          <Tap
            onPress={() => (active === null ? onClose() : setActive(null))}
            hitSlop={10}
          >
            <Feather name="chevron-left" size={24} color={theme.ink} />
          </Tap>
          <Text style={[s.h2, { flex: 1 }]}>{t.support.title}</Text>
          <Tap onPress={onClose} hitSlop={10}>
            <Feather name="x" size={22} color={theme.muted} />
          </Tap>
        </View>

        {active === null ? (
          <ScrollView
            contentContainerStyle={[
              s.list,
              { paddingBottom: 32 + insets.bottom },
            ]}
            keyboardDismissMode="on-drag"
            keyboardShouldPersistTaps="handled"
          >
            <Text style={s.muted}>{t.support.lead}</Text>

            <TextInput
              style={s.input}
              value={ask}
              onChangeText={setAsk}
              placeholder={t.support.searchPlaceholder}
              placeholderTextColor={theme.muted}
            />

            {/* The answers first. */}
            {ask.trim() !== "" && (
              <>
                <Text style={s.muted}>
                  {hits.length > 0 ? t.support.found : t.support.noAnswer}
                </Text>
                {hits.slice(0, 5).map((h) => (
                  <Tap
                    key={h.article.id}
                    style={[s.card, { gap: 6 }]}
                    onPress={() =>
                      setOpened(opened === h.article.id ? null : h.article.id)
                    }
                  >
                    <Text style={s.h2}>{h.article.title}</Text>
                    {opened === h.article.id && (
                      <Text style={s.soft}>{h.article.body}</Text>
                    )}
                  </Tap>
                ))}
              </>
            )}

            {/* ⚠️ Always visible, never behind "did that help?". Somebody who
                has read the paragraph and is still stuck is the person this
                button exists for, and hiding it makes them search for a way to
                reach a human while something in their restaurant is broken. */}
            <Tap style={s.primary} onPress={() => void openThread("")}>
              <Feather name="message-square" size={18} color={theme.onAccent} />
              <Text style={s.primaryText}>{t.support.askOperator}</Text>
            </Tap>

            {threads === null && (
              <View style={{ paddingVertical: 20 }}>
                <ActivityIndicator color={theme.accent} />
              </View>
            )}

            {(threads?.length ?? 0) > 0 && (
              <>
                <Text style={[s.h2, { marginTop: 6 }]}>{t.support.history}</Text>
                {(threads ?? []).map((th) => (
                  <Tap
                    key={th.id}
                    style={[s.card, { gap: 4 }]}
                    onPress={() => void openThread(th.id)}
                  >
                    <View style={local.row}>
                      <Text style={[s.body, { flex: 1 }]} numberOfLines={1}>
                        {th.subject || th.lastText}
                      </Text>
                      {th.unreadForOwner > 0 && (
                        <View
                          style={[local.badge, { backgroundColor: theme.accent }]}
                        >
                          <Text style={{ color: theme.onAccent, fontSize: 11 }}>
                            {th.unreadForOwner}
                          </Text>
                        </View>
                      )}
                    </View>
                    <Text style={s.muted}>
                      {th.status === "closed"
                        ? t.support.closed
                        : th.status === "open"
                          ? t.support.answered
                          : t.support.waiting}
                    </Text>
                  </Tap>
                ))}
              </>
            )}
          </ScrollView>
        ) : (
          <>
            <FlatList
              data={messages}
              keyExtractor={(m) => m.id}
              contentContainerStyle={s.list}
              keyboardDismissMode="on-drag"
              keyboardShouldPersistTaps="handled"
              ListEmptyComponent={
                <Text style={s.muted}>{t.support.placeholder}</Text>
              }
              renderItem={({ item }) => (
                <View
                  style={[
                    s.card,
                    {
                      gap: 3,
                      // Ours on the right, theirs on the left — the one visual
                      // rule every chat on this phone already follows.
                      alignSelf: item.from === "owner" ? "flex-end" : "flex-start",
                      maxWidth: "88%",
                      backgroundColor:
                        item.from === "owner" ? theme.accentSoft : theme.surface,
                    },
                  ]}
                >
                  <Text style={s.muted}>
                    {item.from === "owner"
                      ? t.support.you
                      : item.from === "assistant"
                        ? t.support.assistant
                        : t.support.operator}
                  </Text>
                  <Text style={s.body}>{item.text}</Text>
                </View>
              )}
            />

            {failed && <Text style={s.error}>{t.support.failed}</Text>}

            <View
              style={[
                local.composer,
                {
                  paddingBottom: Math.max(insets.bottom, 12),
                  borderTopColor: theme.line,
                },
              ]}
            >
              <TextInput
                style={[s.input, { flex: 1, maxWidth: undefined }]}
                value={draft}
                onChangeText={setDraft}
                multiline
                placeholder={t.support.placeholder}
                placeholderTextColor={theme.muted}
              />
              <Tap
                style={[
                  s.primary,
                  { paddingHorizontal: 18 },
                  draft.trim() === "" || sending ? { opacity: 0.5 } : null,
                ]}
                disabled={draft.trim() === "" || sending}
                onPress={() => void send()}
              >
                <Feather name="send" size={18} color={theme.onAccent} />
              </Tap>
            </View>
          </>
        )}
      </KeyboardAvoidingView>
    </Modal>
  );
}

const local = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
  badge: {
    minWidth: 20,
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 999,
    alignItems: "center",
  },
  composer: {
    flexDirection: "row",
    alignItems: "flex-end",
    gap: 8,
    padding: 12,
    // A line, because the composer is now flush with the screen edge and
    // without it the input floats on the same surface as the last message.
    borderTopWidth: 1,
  },
});
