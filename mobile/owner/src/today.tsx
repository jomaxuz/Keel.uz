import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api } from "@/lib/api";
import type { AdminStats, Branch, BriefingCard } from "@/lib/types";

import { money } from "./money";
import { OnShiftCard } from "./onshift";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// The screen an owner opens twenty times a day.
//
// ⚠️ **One question, answered before the phone is fully out of the pocket:
// how is today going.** Everything else in the panel is a decision somebody
// sits down to make; this is the number they check between two other things,
// standing up, and it has to be readable in three seconds.
//
// ⚠️ **A figure with nothing beside it is not information.** "12 400 000" is
// neither good nor bad — it is only an answer next to yesterday's, which is why
// the comparison is fetched at the same time and shown in the same block. The
// panel's dashboard makes the same argument with a chart; a phone has room for
// one sentence.

/** Local midnight, as the day boundary every report in this product uses. */
function dayRange(offset = 0): { from: string; to: string } {
  const d = new Date();
  d.setDate(d.getDate() + offset);
  const day = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate(),
  ).padStart(2, "0")}`;
  return { from: day, to: day };
}

export function TodayScreen({
  branches,
  branchId,
  onBranch,
}: {
  branches: Branch[];
  branchId: string;
  onBranch: (id: string) => void;
}) {
  const { t, lang } = usePrefs();
  const { theme, s } = useUI();
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [before, setBefore] = useState<AdminStats | null>(null);
  const [at, setAt] = useState<Date | null>(null);
  // ⚠️ Held apart from the figures and allowed to fail on its own. The takings
  // do not depend on the assistant, and a morning where the briefing could not
  // be written is not a morning without a dashboard — the panel learned this
  // the same way, and draws nothing rather than an error over working numbers.
  const [cards, setCards] = useState<BriefingCard[]>([]);
  const [locked, setLocked] = useState(false);
  const [error, setError] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      // ⚠️ Two requests rather than one: the server answers a period, and
      // "yesterday" is a different period. Asking for a week and slicing it
      // here would be this app deciding what a day is — which is exactly the
      // second definition that drifts from the panel's.
      const [now, prev] = await Promise.all([
        api.adminStats(dayRange(0)),
        api.adminStats(dayRange(-1)),
      ]);
      setStats(now);
      setBefore(prev);
      setAt(new Date());
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [t.common.loadFailed]);

  // ⚠️ **The briefing is asked for separately, and never in the same
  // `Promise.all` as the takings.** It is written once a day and read from the
  // database afterwards, so it is not the slow part — but it is the part that
  // can answer "your plan does not include this", and one rejected request
  // must not take today's revenue down with it.
  const loadBriefing = useCallback(async () => {
    // ⚠️ The language goes in the query rather than being left to the cookie:
    // there is no cookie on a phone, and the server would otherwise write
    // this owner's morning in Uzbek because that is the base.
    const qs = new URLSearchParams({ lang });
    if (branchId) qs.set("branchId", branchId);
    try {
      const res = await api.adminInsights(`?${qs.toString()}`);
      setCards(res.cards ?? []);
      setLocked(res.entitled === false);
    } catch {
      setCards([]);
    }
  }, [lang, branchId]);

  // Reloaded when the lens changes: the numbers are about a branch.
  useEffect(() => {
    void load();
    void loadBriefing();
  }, [load, loadBriefing, branchId]);

  const period = stats?.period;
  const yesterday = before?.period;
  const revenue = period?.revenue ?? 0;
  const orders = period?.orders ?? 0;
  const average = orders > 0 ? Math.round(revenue / orders) : 0;
  const change =
    yesterday && yesterday.revenue > 0
      ? Math.round(((revenue - yesterday.revenue) / yesterday.revenue) * 100)
      : null;

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Text style={s.h1}>{t.today.title}</Text>
        {at && <Text style={s.muted}>{t.today.updated(clock(at))}</Text>}
      </View>

      <ScrollView
        contentContainerStyle={s.list}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            tintColor={theme.accent}
            colors={[theme.accent]}
            onRefresh={() => {
              setRefreshing(true);
              void Promise.all([load(), loadBriefing()]).finally(() =>
                setRefreshing(false),
              );
            }}
          />
        }
      >
        {/* ⚠️ **The lens first, because every number under it is about one
            branch.** A single-branch restaurant never sees this row: one option
            is not a choice, and a control with nothing to choose is a control
            that teaches people to look for meaning that is not there. */}
        {branches.length > 1 && (
          <ScrollView horizontal showsHorizontalScrollIndicator={false}>
            <View style={local.lens}>
              {[{ id: "", name: t.today.branchAll }, ...branches].map((b) => (
                <Pressable
                  key={b.id || "all"}
                  onPress={() => onBranch(b.id)}
                  style={[
                    local.chip,
                    {
                      borderColor: b.id === branchId ? theme.accent : theme.line,
                      backgroundColor:
                        b.id === branchId ? theme.accent : theme.surface,
                    },
                  ]}
                >
                  <Text
                    style={{
                      fontSize: 13,
                      fontWeight: "600",
                      color: b.id === branchId ? theme.onAccent : theme.ink,
                    }}
                  >
                    {b.name}
                  </Text>
                </Pressable>
              ))}
            </View>
          </ScrollView>
        )}

        {stats === null && error === "" && (
          <View style={{ paddingVertical: 40 }}>
            <ActivityIndicator color={theme.accent} />
          </View>
        )}
        {error !== "" && <Text style={s.error}>{error}</Text>}

        {stats && (
          <>
            {/* The one number, in the size it deserves. */}
            <View style={[s.card, { gap: 4 }]}>
              <Text style={s.muted}>{t.today.revenue}</Text>
              <Text style={s.big}>{money(revenue)}</Text>
              <Text
                style={[
                  s.muted,
                  change !== null
                    ? { color: change >= 0 ? theme.ok : theme.danger }
                    : null,
                ]}
              >
                {change === null
                  ? t.today.noYesterday
                  : t.today.vsYesterday(change)}
              </Text>
            </View>

            <View style={local.grid}>
              <Tile label={t.today.orders} value={String(orders)} />
              <Tile label={t.today.average} value={money(average)} />
              <Tile
                label={t.today.delivered}
                value={String(period?.delivered ?? 0)}
              />
              <Tile
                label={t.today.cancelled}
                value={String(period?.cancelled ?? 0)}
                tone={(period?.cancelled ?? 0) > 0 ? theme.warn : undefined}
              />
            </View>
          </>
        )}

        {/* Who is in the building — one line, the list on a tap. */}
        <OnShiftCard branchId={branchId} />

        {/* ⚠️ **Under the figures, not above them.** This screen is opened
            twenty times a day for one number and read once a morning for the
            briefing; putting the rarer thing first would push the common one
            below the fold on the smaller half of the phones in this market. */}
        {cards.length > 0 && (
          <>
            <Text style={[s.h2, { marginTop: 6 }]}>{t.today.briefing}</Text>
            {cards.map((c) => (
              <View key={c.key} style={[s.card, { gap: 4 }]}>
                <View style={local.card}>
                  {/* The area as a stripe rather than a word: five categories
                      named in three languages is fifteen strings for something
                      the sentence already says. */}
                  <View
                    style={[
                      local.tint,
                      { backgroundColor: areaTint(c.area, theme.accent) },
                    ]}
                  />
                  <View style={{ flex: 1, gap: 3 }}>
                    <Text style={s.h2}>{c.title}</Text>
                    <Text style={s.soft}>{c.body}</Text>
                    {/* ⚠️ **The figures beside the sentence, because the words
                        come from a model and the numbers do not.** An owner who
                        wants to check can, in the same glance — a card that
                        showed only prose would be asking for trust it gave no
                        way to verify. */}
                    {c.numbers && Object.keys(c.numbers).length > 0 && (
                      <Text style={[s.muted, { fontVariant: ["tabular-nums"] }]}>
                        {Object.entries(c.numbers)
                          .map(([k, v]) => `${k}: ${v.toLocaleString("ru-RU")}`)
                          .join("  ·  ")}
                      </Text>
                    )}
                  </View>
                </View>
              </View>
            ))}
          </>
        )}

        {/* ⚠️ Something the restaurant can buy is worth one line; a quiet
            morning is not. A card that said "nothing today" every day would be
            trained out of an owner's attention within a week — and would then
            be invisible on the morning it had four things to say. */}
        {cards.length === 0 && locked && (
          <Text style={s.muted}>{t.today.briefingLocked}</Text>
        )}
      </ScrollView>
    </View>
  );
}

function Tile({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: string;
}) {
  const { s } = useUI();
  return (
    <View style={[s.card, local.tile]}>
      <Text style={s.muted}>{label}</Text>
      <Text style={[s.mid, tone ? { color: tone } : null]}>{value}</Text>
    </View>
  );
}

/** ⚠️ The device's own clock and its own locale: this is "when did the phone
 *  last ask", not a fact about the restaurant, so it is the one time on these
 *  screens that does not come from the server. */
function clock(d: Date): string {
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}

/** The five areas the assistant writes about, in this app's palette. ⚠️ An
 *  unknown one falls back to the accent rather than to nothing: the server may
 *  grow a sixth before the phone is updated, and a card with no stripe reads as
 *  a card that failed to draw. */
function areaTint(area: string, fallback: string): string {
  return (
    {
      guests: "#0ea5e9",
      menu: "#f59e0b",
      stock: "#10b981",
      team: "#8b5cf6",
      money: "#f43f5e",
    }[area] ?? fallback
  );
}

const local = StyleSheet.create({
  lens: { flexDirection: "row", gap: 8, paddingBottom: 2 },
  chip: {
    borderWidth: 1,
    borderRadius: 999,
    paddingHorizontal: 14,
    paddingVertical: 8,
  },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 12 },
  tile: { flexGrow: 1, flexBasis: "46%", gap: 2 },
  card: { flexDirection: "row", gap: 10 },
  tint: { width: 3, borderRadius: 2, alignSelf: "stretch" },
});
