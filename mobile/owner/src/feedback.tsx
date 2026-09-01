import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Linking,
  Modal,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api, ApiError } from "@/lib/api";
import { formatUzPhone } from "@/lib/format";
import { timeAgo } from "@/lib/orderFlow";
import type { Feedback } from "@/lib/types";

import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// What guests said, and the one thing an owner does about it from a phone.
//
// ⚠️ **This screen exists because of when complaints arrive.** A one-star
// review lands at eight in the evening, and until now the only place it could
// be read was the panel — that is, the next morning, from a desk. By then the
// guest has told somebody else instead. The answer to a complaint is a phone
// call, and the phone is already in the owner's hand.
//
// ⚠️ **Opens on the unanswered ones, not on everything.** The list that matters
// is the one that has to reach zero; a chronological feed of every rating puts
// four fives in front of the one that needed somebody.
//
// ⚠️ **Publishing to the public site stays in the panel.** It puts a guest's
// name and their words on the restaurant's website, which is a decision made
// sitting down and looking at the site — not a toggle beside a call button at a
// traffic light. Answering is the reverse: it cannot wait, and it costs one tap
// and a sentence.

type Filter = "unhandled" | "low" | "";

export function FeedbackScreen({ branchId }: { branchId: string }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [filter, setFilter] = useState<Filter>("unhandled");
  const [rows, setRows] = useState<Feedback[] | null>(null);
  const [stats, setStats] = useState<{ average: number; open: number } | null>(
    null,
  );
  const [answering, setAnswering] = useState<Feedback | null>(null);
  const [resolution, setResolution] = useState("");
  const [busy, setBusy] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api.adminFeedback(filter ? { filter } : undefined);
      setRows(res.feedback ?? []);
      setStats(res.stats ?? null);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [filter, t.common.loadFailed]);

  // Reloaded when the lens or the filter changes; the reviews are about a
  // branch the same way the takings are.
  useEffect(() => {
    void load();
  }, [load, branchId]);

  async function answer() {
    if (!answering || resolution.trim() === "") return;
    setBusy(true);
    try {
      await api.handleFeedback(answering.id, resolution.trim());
      setAnswering(null);
      setResolution("");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.feedback.failed);
    } finally {
      setBusy(false);
    }
  }

  const filters: { key: Filter; label: string }[] = [
    { key: "unhandled", label: t.feedback.filterOpen },
    { key: "low", label: t.feedback.filterLow },
    { key: "", label: t.feedback.filterAll },
  ];

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Text style={s.h1}>{t.feedback.title}</Text>
        {/* ⚠️ The average beside the queue, because they answer different
            questions: one is how the restaurant is doing, the other is what is
            waiting. An owner who sees only the second thinks every evening is a
            bad one. */}
        {stats && (
          <Text style={s.muted}>
            {t.feedback.average(stats.average.toFixed(1))}
          </Text>
        )}
      </View>

      <View style={local.filters}>
        {filters.map((f) => (
          <Pressable
            key={f.key || "all"}
            onPress={() => setFilter(f.key)}
            style={[
              local.chip,
              {
                borderColor: f.key === filter ? theme.accent : theme.line,
                backgroundColor:
                  f.key === filter ? theme.accent : theme.surface,
              },
            ]}
          >
            <Text
              style={{
                fontSize: 13,
                fontWeight: "600",
                color: f.key === filter ? theme.onAccent : theme.ink,
              }}
            >
              {f.label}
              {f.key === "unhandled" && (stats?.open ?? 0) > 0
                ? ` · ${stats?.open}`
                : ""}
            </Text>
          </Pressable>
        ))}
      </View>

      <FlatList
        data={rows ?? []}
        keyExtractor={(f) => f.id}
        contentContainerStyle={s.list}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            tintColor={theme.accent}
            colors={[theme.accent]}
            onRefresh={() => {
              setRefreshing(true);
              void load().finally(() => setRefreshing(false));
            }}
          />
        }
        ListHeaderComponent={
          <>
            {rows === null && error === "" && (
              <View style={{ paddingVertical: 40 }}>
                <ActivityIndicator color={theme.accent} />
              </View>
            )}
            {error !== "" && <Text style={s.error}>{error}</Text>}
          </>
        }
        ListEmptyComponent={
          rows === null ? null : (
            <View style={s.empty}>
              <Feather name="message-circle" size={26} color={theme.ok} />
              <Text style={[s.h2, { textAlign: "center" }]}>
                {filter === "" ? t.feedback.emptyAll : t.feedback.empty}
              </Text>
              <Text style={[s.muted, { textAlign: "center" }]}>
                {t.feedback.emptyHint}
              </Text>
            </View>
          )
        }
        renderItem={({ item }) => {
          const low = item.rating <= 3;
          return (
            <View style={[s.card, { gap: 8 }]}>
              <View style={local.row}>
                {/* ⚠️ Stars as characters rather than icons: five glyphs of a
                    vector family cost five nodes per row and say exactly what
                    "★★☆☆☆" says at any size, in any language. */}
                <Text
                  style={[
                    local.stars,
                    { color: low ? theme.danger : theme.ok },
                  ]}
                >
                  {"★".repeat(item.rating)}
                  <Text style={{ color: theme.line }}>
                    {"★".repeat(5 - item.rating)}
                  </Text>
                </Text>
                <Text style={[s.muted, { flex: 1, textAlign: "right" }]}>
                  {timeAgo(item.createdAt, t.common.timeAgo)}
                </Text>
              </View>

              <Text style={s.muted}>
                {[
                  item.orderNumber ? `#${item.orderNumber}` : null,
                  item.customer?.name || null,
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </Text>

              {item.comment ? (
                <Text style={s.body}>“{item.comment}”</Text>
              ) : null}

              {item.handled ? (
                <View style={local.row}>
                  <Feather name="check-circle" size={15} color={theme.ok} />
                  <Text style={[s.muted, { flex: 1 }]}>
                    {item.resolution ||
                      t.feedback.answered(item.handledBy || "")}
                  </Text>
                </View>
              ) : (
                <View style={local.actions}>
                  {/* ⚠️ The call first and on the left: it is what actually
                      settles a complaint, and the note is what is written
                      afterwards. A screen that offered only the note would be
                      a screen for closing tickets. */}
                  {item.customer?.phone ? (
                    <Pressable
                      style={[s.ghost, local.action]}
                      onPress={() =>
                        void Linking.openURL(`tel:${item.customer.phone}`)
                      }
                    >
                      <Feather name="phone" size={16} color={theme.ink} />
                      <Text style={s.ghostText}>{t.feedback.call}</Text>
                    </Pressable>
                  ) : null}
                  <Pressable
                    style={[s.primary, local.action]}
                    onPress={() => {
                      setAnswering(item);
                      setResolution("");
                    }}
                  >
                    <Feather name="edit-2" size={16} color={theme.onAccent} />
                    <Text style={s.primaryText}>{t.feedback.answer}</Text>
                  </Pressable>
                </View>
              )}

              {item.customer?.phone && !item.handled ? (
                <Text style={s.muted}>
                  {formatUzPhone(item.customer.phone)}
                </Text>
              ) : null}
            </View>
          );
        }}
      />

      {answering && (
        <Modal
          transparent
          animationType="fade"
          onRequestClose={() => setAnswering(null)}
        >
          <Pressable style={local.backdrop} onPress={() => setAnswering(null)}>
            <Pressable
              style={[local.sheet, { backgroundColor: theme.surface }]}
              onPress={(e) => e.stopPropagation()}
            >
              <Text style={s.h2}>{t.feedback.answer}</Text>
              {/* ⚠️ The server requires the sentence, and so does the point of
                  the record: "handled" with nothing written is a tick that
                  makes the list shorter without making anybody wiser. */}
              <Text style={s.muted}>{t.feedback.answerHint}</Text>
              <TextInput
                style={[s.input, { minHeight: 88, textAlignVertical: "top" }]}
                value={resolution}
                onChangeText={setResolution}
                multiline
                placeholder={t.feedback.answerPlaceholder}
                placeholderTextColor={theme.muted}
              />
              <Pressable
                style={[
                  s.primary,
                  resolution.trim() === "" || busy ? { opacity: 0.5 } : null,
                ]}
                disabled={resolution.trim() === "" || busy}
                onPress={() => void answer()}
              >
                <Text style={s.primaryText}>{t.feedback.answerDo}</Text>
              </Pressable>
              <Pressable style={s.ghost} onPress={() => setAnswering(null)}>
                <Text style={s.ghostText}>{t.common.close}</Text>
              </Pressable>
            </Pressable>
          </Pressable>
        </Modal>
      )}
    </View>
  );
}

const local = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
  filters: { flexDirection: "row", gap: 8, paddingHorizontal: 16 },
  chip: {
    borderWidth: 1,
    borderRadius: 999,
    paddingHorizontal: 14,
    paddingVertical: 8,
  },
  stars: { fontSize: 18, letterSpacing: 1 },
  actions: { flexDirection: "row", gap: 8 },
  action: { flex: 1, paddingVertical: 12 },
  backdrop: {
    flex: 1,
    backgroundColor: "rgba(0,0,0,0.5)",
    alignItems: "center",
    justifyContent: "center",
    padding: 24,
  },
  sheet: {
    width: "100%",
    maxWidth: 380,
    borderRadius: 22,
    padding: 18,
    gap: 12,
  },
});
