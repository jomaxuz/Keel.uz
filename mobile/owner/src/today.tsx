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
import type { AdminStats, Branch } from "@/lib/types";

import { money } from "./money";
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
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [before, setBefore] = useState<AdminStats | null>(null);
  const [at, setAt] = useState<Date | null>(null);
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

  // Reloaded when the lens changes: the numbers are about a branch.
  useEffect(() => {
    void load();
  }, [load, branchId]);

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
              void load().finally(() => setRefreshing(false));
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
              <Text style={local.big}>{money(revenue)}</Text>
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
      <Text style={[local.mid, tone ? { color: tone } : null]}>{value}</Text>
    </View>
  );
}

/** ⚠️ The device's own clock and its own locale: this is "when did the phone
 *  last ask", not a fact about the restaurant, so it is the one time on these
 *  screens that does not come from the server. */
function clock(d: Date): string {
  return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
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
  big: { fontSize: 30, fontWeight: "800", fontVariant: ["tabular-nums"] },
  mid: { fontSize: 20, fontWeight: "700", fontVariant: ["tabular-nums"] },
});
