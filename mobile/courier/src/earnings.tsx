import { useCallback, useEffect, useState } from "react";
import { ActivityIndicator, FlatList, StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import type { CourierOrderRow, CourierStats } from "@/lib/types";

import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// What the evening was worth, and what is still in a pocket.
//
// ⚠️ **Cash in hand is the number this screen exists for.** The rest is
// encouragement; that one is a debt. It is everything collected less everything
// handed back — not the day's takings, which the web app showed first and which
// is wrong twice over: it stays on screen after the money has been handed in,
// and it misses cash still owed from yesterday.

export function EarningsScreen() {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [stats, setStats] = useState<CourierStats | null>(null);
  const [history, setHistory] = useState<CourierOrderRow[] | null>(null);

  const load = useCallback(() => {
    void api.courierStats().then(setStats).catch(() => setStats(null));
    void api
      .courierHistory()
      .then(setHistory)
      .catch(() => setHistory([]));
  }, []);

  useEffect(load, [load]);

  const periods = [
    { label: t.earnings.today, p: stats?.today },
    { label: t.earnings.week, p: stats?.week },
    { label: t.earnings.month, p: stats?.month },
    { label: t.earnings.all, p: stats?.all },
  ];

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Text style={s.h1}>{t.earnings.title}</Text>
      </View>

      <FlatList
        data={history ?? []}
        keyExtractor={(o) => o.id}
        contentContainerStyle={s.list}
        ListHeaderComponent={
          <View style={{ gap: 14 }}>
            <View style={local.grid}>
              {periods.map(({ label, p }) => (
                <View key={label} style={[s.card, local.tile]}>
                  <Text style={s.muted}>{label}</Text>
                  <Text style={[local.big, { color: theme.accent }]}>
                    {formatPrice(p?.earnings ?? 0)}
                  </Text>
                  <Text style={s.muted}>{t.earnings.deliveries(p?.orders ?? 0)}</Text>
                </View>
              ))}
            </View>

            {stats && stats.cashInHand > 0 && (
              <View style={[local.banner, { backgroundColor: theme.warnSoft }]}>
                <Feather name="dollar-sign" size={16} color={theme.warn} />
                <Text style={[s.body, { flex: 1, color: theme.warn }]}>
                  {t.earnings.cashInHand(formatPrice(stats.cashInHand))}
                </Text>
              </View>
            )}
            {stats && stats.cashInHand === 0 && stats.cashSettled > 0 && (
              <View style={[local.banner, { backgroundColor: theme.okSoft }]}>
                <Feather name="check-circle" size={16} color={theme.ok} />
                <Text style={[s.body, { flex: 1, color: theme.ok }]}>
                  {t.earnings.cashClear}
                </Text>
              </View>
            )}

            <Text style={[s.h2, { marginTop: 4 }]}>{t.earnings.history}</Text>
          </View>
        }
        ListEmptyComponent={
          history === null ? (
            <View style={{ paddingVertical: 30 }}>
              <ActivityIndicator color={theme.accent} />
            </View>
          ) : (
            <View style={s.empty}>
              <Feather name="package" size={24} color={theme.muted} />
              <Text style={s.muted}>{t.earnings.historyEmpty}</Text>
            </View>
          )
        }
        renderItem={({ item }) => (
          <View style={[s.card, { gap: 4 }]}>
            <View style={local.row}>
              <Text style={[s.h2, { flex: 1 }]}>#{item.number}</Text>
              <Text style={[s.money, { color: theme.ok }]}>
                +{formatPrice(item.earned)}
              </Text>
            </View>
            <Text style={s.muted}>
              {when(item.deliveredAt)}
              {item.address?.text ? ` · ${item.address.text}` : ""}
            </Text>
            <Text style={s.muted}>
              {formatPrice(item.total)} ·{" "}
              {item.paymentMethod === "cash" ? t.earnings.cash : t.orders.paid}
            </Text>
          </View>
        )}
      />
    </View>
  );
}

/** ⚠️ The device's own locale for the clock, and the raw string parsed rather
 *  than sliced: the server sends UTC, and cutting the first ten characters off
 *  it — the shortcut this codebase has been bitten by twice — shows the
 *  previous day to every courier working past midnight. */
function when(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? ""
    : d.toLocaleString(undefined, {
        day: "2-digit",
        month: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      });
}

const local = StyleSheet.create({
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 12 },
  tile: { flexGrow: 1, flexBasis: "46%", gap: 2 },
  big: { fontSize: 20, fontWeight: "700", fontVariant: ["tabular-nums"] },
  banner: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    borderRadius: 14,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
});
