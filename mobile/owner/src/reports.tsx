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
import type { AdminStats, PayrollResponse, ShoppingRow } from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { Tap } from "./press";
import { useUI } from "./ui";

// The three questions an owner asks about a period, and nothing else.
//
// ⚠️ **Read-only, deliberately.** Everything here has a screen in the panel
// that can also *change* something — reprice a dish, pay a wage, write off a
// kilo — and every one of those is a decision made sitting down. A phone that
// offered them would be offering them at a traffic light.
//
// ⚠️ **Three periods, no date picker.** "Today", "this week", "this month" is
// what somebody standing up asks; an arbitrary range is a keyboard, two
// calendars and a screen nobody uses twice. The panel has the calendar.

type Period = "today" | "week" | "month";

/** The range for a period, in local days — the same boundary every report in
 *  this product uses. ⚠️ Built from the device's own calendar rather than from
 *  a string the server sends, so a phone in another timezone still asks about
 *  the restaurant's day. */
function rangeFor(period: Period): { from: string; to: string } {
  const to = new Date();
  const from = new Date();
  if (period === "week") from.setDate(from.getDate() - 6);
  if (period === "month") from.setDate(from.getDate() - 29);
  return { from: day(from), to: day(to) };
}

function day(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate(),
  ).padStart(2, "0")}`;
}

export function ReportsScreen({ branchId }: { branchId: string }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [period, setPeriod] = useState<Period>("today");
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [payroll, setPayroll] = useState<PayrollResponse | null>(null);
  const [shopping, setShopping] = useState<ShoppingRow[] | null>(null);
  const [error, setError] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      setStats(await api.adminStats(rangeFor(period)));
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
    // ⚠️ **Separately, and failures are swallowed.** Payroll and the stock list
    // are extras on this screen; a restaurant that does not use the store, or a
    // manager whose account may not read wages, must still get the takings —
    // which is what the screen is opened for.
    try {
      setPayroll(await api.adminPayroll());
    } catch {
      setPayroll(null);
    }
    try {
      // ⚠️ The list comes grouped by supplier — that is how shopping is
      // actually done — and this screen flattens it: an owner reading a phone
      // wants "what is running out", and who to ring is the panel's question.
      const res = await api.adminShoppingList();
      setShopping((res.groups ?? []).flatMap((g) => g.rows ?? []));
    } catch {
      setShopping(null);
    }
  }, [period, t.common.loadFailed]);

  useEffect(() => {
    void load();
  }, [load, branchId]);

  const p = stats?.period;
  const orders = p?.orders ?? 0;
  const revenue = p?.revenue ?? 0;

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Text style={s.h1}>{t.reports.title}</Text>
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
        <View style={local.periods}>
          {(["today", "week", "month"] as Period[]).map((key) => (
            <Tap
              key={key}
              onPress={() => setPeriod(key)}
              style={[
                local.period,
                {
                  borderColor: key === period ? theme.accent : theme.line,
                  backgroundColor:
                    key === period ? theme.accent : theme.surface,
                },
              ]}
            >
              <Text
                style={{
                  fontWeight: "700",
                  color: key === period ? theme.onAccent : theme.ink,
                }}
              >
                {t.reports[key]}
              </Text>
            </Tap>
          ))}
        </View>

        {stats === null && error === "" && (
          <View style={{ paddingVertical: 30 }}>
            <ActivityIndicator color={theme.accent} />
          </View>
        )}
        {error !== "" && <Text style={s.error}>{error}</Text>}

        {stats && (
          <>
            <View style={[s.card, { gap: 4 }]}>
              <Text style={s.muted}>{t.reports.revenue}</Text>
              <Text style={s.big}>{money(revenue)}</Text>
              <Text style={s.muted}>
                {orders} · {t.reports.average}{" "}
                {money(orders > 0 ? Math.round(revenue / orders) : 0)}
              </Text>
            </View>

            {(stats.top ?? []).length > 0 && (
              <View style={[s.card, { gap: 8 }]}>
                <Text style={s.muted}>{t.reports.topDishes}</Text>
                {(stats.top ?? []).slice(0, 5).map((d) => (
                  <View key={d.name} style={local.row}>
                    <Text style={[s.body, { flex: 1 }]} numberOfLines={1}>
                      {d.name}
                    </Text>
                    <Text style={s.num}>{d.qty}</Text>
                    <Text style={[s.muted, { width: 96, textAlign: "right" }]}>
                      {money(d.total)}
                    </Text>
                  </View>
                ))}
              </View>
            )}

            {(payroll?.rows ?? []).length > 0 && (
              <View style={[s.card, { gap: 8 }]}>
                <Text style={s.muted}>{t.reports.staff}</Text>
                {(payroll?.rows ?? []).slice(0, 6).map((row) => (
                  <View key={row.staffId} style={local.row}>
                    <Text style={[s.body, { flex: 1 }]} numberOfLines={1}>
                      {row.name}
                    </Text>
                    <Text style={s.muted}>
                      {t.reports.hours((row.worked / 60).toFixed(1))}
                    </Text>
                    <Text style={[s.num, { width: 110, textAlign: "right" }]}>
                      {money(row.due)}
                    </Text>
                  </View>
                ))}
              </View>
            )}

            {shopping !== null && (
              <View style={[s.card, { gap: 8 }]}>
                <Text style={s.muted}>{t.reports.stock}</Text>
                {shopping.length === 0 ? (
                  <Text style={s.soft}>{t.reports.stockEmpty}</Text>
                ) : (
                  shopping.slice(0, 8).map((row) => (
                    <View key={row.ingredientId} style={local.row}>
                      <Text style={[s.body, { flex: 1 }]} numberOfLines={1}>
                        {row.name}
                      </Text>
                      {/* ⚠️ What is left, in the unit it is bought in — the
                          number somebody can act on standing in a shop. */}
                      <Text style={[s.num, { color: theme.warn }]}>
                        {row.onHand} {row.unit}
                      </Text>
                    </View>
                  ))
                )}
              </View>
            )}
          </>
        )}
      </ScrollView>
    </View>
  );
}

const local = StyleSheet.create({
  periods: { flexDirection: "row", gap: 8 },
  period: {
    flex: 1,
    borderWidth: 1,
    borderRadius: 14,
    paddingVertical: 12,
    alignItems: "center",
  },
  row: { flexDirection: "row", alignItems: "center", gap: 10 },
});
