import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api } from "@/lib/api";
import { timeAgo } from "@/lib/orderFlow";
import type { AdminAlerts, LossAlert } from "@/lib/types";

import { money } from "./money";
import type { Dict } from "./i18n";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// What needs the owner now.
//
// ⚠️ **Two lists that look alike and are not.** The bell counts are things
// waiting to be done — an order nobody accepted, a booking nobody confirmed, a
// dish the till cannot ring up. The loss alerts are things that already
// happened and cannot be undone: a discount taken after the bill was printed, a
// till short at the close. The first is a queue, the second is a record, and
// mixing them would let a fixable job hide a fact somebody has to ask about.
//
// ⚠️ **This screen states, it never concludes.** Every kind here has an
// ordinary explanation that happens weekly in a busy restaurant — a guest who
// complained, a regular given something off, a till short because somebody paid
// a courier out of it. A screen that decided anything would be wrong often
// enough to be resented, and resented alerts get muted rather than argued with.
// The same rule the Telegram text has followed since it existed.

export function AlertsScreen({ branchId }: { branchId: string }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [bell, setBell] = useState<AdminAlerts | null>(null);
  const [loss, setLoss] = useState<LossAlert[] | null>(null);
  const [error, setError] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      const [a, l] = await Promise.all([
        api.adminAlerts(),
        api.adminLossAlerts(),
      ]);
      setBell(a);
      setLoss(l.alerts);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [t.common.loadFailed]);

  useEffect(() => {
    void load();
    // ⚠️ Polled, like every other screen in this product. A socket here would
    // be the one exception nobody maintains — and the events this counts are
    // minutes apart, not seconds.
    const timer = setInterval(() => void load(), 30_000);
    return () => clearInterval(timer);
  }, [load, branchId]);

  // Only what is above zero: a list of zeroes is a list nobody reads, and the
  // one line that matters would be the seventh row of it.
  const jobs: { icon: keyof typeof Feather.glyphMap; text: string; warn?: boolean }[] = [];
  if (bell) {
    if (bell.orders.pending > 0)
      jobs.push({
        icon: "shopping-bag",
        text: t.alerts.pendingOrders(bell.orders.pending),
        warn: true,
      });
    if ((bell.preorders?.upcoming ?? 0) > 0)
      jobs.push({
        icon: "clock",
        text: t.alerts.preorders(bell.preorders?.upcoming ?? 0),
      });
    if (bell.reservations.pending > 0)
      jobs.push({
        icon: "calendar",
        text: t.alerts.reservations(bell.reservations.pending),
      });
    if ((bell.pos?.unaccepted ?? 0) > 0)
      jobs.push({
        icon: "monitor",
        text: t.alerts.posUnaccepted(bell.pos?.unaccepted ?? 0),
        warn: true,
      });
    if ((bell.pos?.failed ?? 0) > 0)
      jobs.push({
        icon: "alert-triangle",
        text: t.alerts.posFailed(bell.pos?.failed ?? 0),
        warn: true,
      });
    if ((bell.pos?.unmapped ?? 0) > 0)
      jobs.push({
        icon: "link-2",
        text: t.alerts.posUnmapped(bell.pos?.unmapped ?? 0),
      });
    if ((bell.print?.failed ?? 0) > 0)
      jobs.push({
        icon: "printer",
        text: t.alerts.printFailed(bell.print?.failed ?? 0),
      });
  }

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Text style={s.h1}>{t.alerts.title}</Text>
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
        {bell === null && error === "" && (
          <View style={{ paddingVertical: 40 }}>
            <ActivityIndicator color={theme.accent} />
          </View>
        )}
        {error !== "" && <Text style={s.error}>{error}</Text>}

        {bell !== null && jobs.length === 0 && (loss?.length ?? 0) === 0 && (
          <View style={s.empty}>
            <Feather name="check-circle" size={26} color={theme.ok} />
            <Text style={[s.h2, { textAlign: "center" }]}>{t.alerts.empty}</Text>
            <Text style={[s.muted, { textAlign: "center" }]}>
              {t.alerts.emptyHint}
            </Text>
          </View>
        )}

        {jobs.map((j, i) => (
          <View key={i} style={[s.row, { gap: 10 }]}>
            <Feather
              name={j.icon}
              size={18}
              color={j.warn ? theme.warn : theme.muted}
            />
            <Text style={[s.body, { flex: 1 }]}>{j.text}</Text>
          </View>
        ))}

        {(loss?.length ?? 0) > 0 && (
          <>
            <Text style={[s.h2, { marginTop: 6 }]}>{t.alerts.lossTitle}</Text>
            {(loss ?? []).map((a) => (
              <View key={a.id} style={[s.card, { gap: 3 }]}>
                <View style={local.row}>
                  <Text style={[s.h2, { flex: 1 }]}>{kindLabel(a.kind, t)}</Text>
                  {/* ⚠️ Zero prints nothing rather than "0 so'm": some kinds
                      have no meaningful figure — a recipe change costs whatever
                      gets sold — and a zero reads as a bug in the alert. */}
                  {a.amount !== 0 && (
                    <Text style={[s.money, { color: theme.warn }]}>
                      {money(a.amount)}
                    </Text>
                  )}
                </View>
                <Text style={s.muted}>
                  {[
                    a.subject,
                    a.by ? t.alerts.by(a.by) : null,
                    timeAgo(a.at, t.common.timeAgo),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </Text>
                {a.reason ? <Text style={s.soft}>“{a.reason}”</Text> : null}
              </View>
            ))}
          </>
        )}
      </ScrollView>
    </View>
  );
}

/** ⚠️ **The kind, in this app's words, and the fallback is the kind itself.**
 *  The server may grow a new one before the phone is updated, and a row that
 *  said nothing would be worse than one saying `stock_short`: an owner can ask
 *  about a word and cannot ask about a blank. */
function kindLabel(kind: string, dict: Dict): string {
  return dict.alerts.kinds[kind as keyof Dict["alerts"]["kinds"]] ?? kind;
}

const local = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
});
