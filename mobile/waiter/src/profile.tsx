import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
// ⚠️ **From the family's own path, not the package index.** The index
// re-exports every icon set it ships — AntDesign, MaterialIcons, Ionicons
// and a dozen more — and each carries a glyph map, so importing one name
// from it pulls all of them into the bundle. Measured on these exact
// screens: 2.0 MB and 688 modules from the index, 1.6 MB and 634 from here.
import Feather from "@expo/vector-icons/Feather";

import { api, ApiError } from "@/lib/api";
import { formatDuration, STATUS_COLOR } from "@/lib/attendance";
import type { Staff, StaffDay, StaffReport } from "@/lib/types";

import { ClockButton } from "./clock";
import { money } from "./money";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// How much somebody has worked, and when.
//
// ⚠️ **The same question the panel answers, from the same endpoint.**
// `/staff/report` is the employee's own screen and already returns the day, its
// status and what it was against the roster — so this is a second *drawing* of
// one answer, never a second calculation. A phone that added up its own hours
// would disagree with the payroll screen, and the disagreement would be found
// on pay day.
//
// ⚠️ **And the colours are not chosen here.** `STATUS_COLOR` lives in
// `lib/attendance.ts` beside the panel's Tailwind classes, so a shift that is
// amber in the office cannot be green in somebody's hand. Only the label is
// this app's.

export function ProfileScreen({ staff }: { staff: Staff }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [report, setReport] = useState<StaffReport | null>(null);
  const [error, setError] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      setReport(await api.staffReport());
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.floor.failedLoad);
    }
  }, [t.floor.failedLoad]);

  useEffect(() => {
    void load();
  }, [load]);

  const dur = useCallback(
    (m: number) => formatDuration(m, t.profile.hour, t.profile.minute),
    [t.profile.hour, t.profile.minute],
  );

  /** The days, padded so the first one lands under its weekday.
   *
   *  ⚠️ **Monday first.** The roster is written a week at a time and a
   *  Sunday-first grid puts the end of the week at its start — the panel's
   *  calendar reads Monday first for the same reason, and two calendars that
   *  disagree about where a week begins are two calendars nobody trusts. */
  const grid = useMemo(() => {
    const days = report?.days ?? [];
    if (days.length === 0) return [];
    const first = new Date(days[0].date + "T00:00:00");
    // getDay(): 0 is Sunday. Monday-first means Sunday is the seventh column.
    const lead = (first.getDay() + 6) % 7;
    return [...Array<StaffDay | null>(lead).fill(null), ...days];
  }, [report]);

  if (!report) {
    return (
      <View style={s.centered}>
        {error !== "" ? (
          <>
            <Text style={s.error}>{error}</Text>
            <Pressable onPress={() => void load()}>
              <Text style={s.link}>{t.common.retry}</Text>
            </Pressable>
          </>
        ) : (
          <ActivityIndicator color={theme.accent} />
        )}
      </View>
    );
  }

  const openDay = report.days.find((d) => d.open);

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <View style={{ flex: 1 }}>
          <Text style={s.h2}>{staff.name}</Text>
          <Text style={s.muted}>{staff.position || t.profile.title}</Text>
        </View>
        {/* ⚠️ Named, not only coloured: "on shift" is the one state on this
            screen somebody might act on, and a dot alone says nothing to
            whoever cannot separate these hues. */}
        {openDay && (
          <View style={[local.badge, { backgroundColor: theme.accentSoft }]}>
            <View style={[local.dot, { backgroundColor: theme.accent }]} />
            <Text style={{ color: theme.accent, fontSize: 12 }}>
              {t.profile.onShift}
            </Text>
          </View>
        )}
      </View>

      <ScrollView
        contentContainerStyle={s.list}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            tintColor={theme.accent}
            onRefresh={async () => {
              setRefreshing(true);
              await load();
              setRefreshing(false);
            }}
          />
        }
      >
        {/* ⚠️ First on the screen, because it is the only thing here somebody
            *does*. The hours below are a record; this is the act that creates
            them, and burying it under a calendar would put the daily action
            beneath the monthly reading. */}
        <ClockButton open={!!openDay} onChanged={() => void load()} />

        <View style={local.trends}>
          <Trend label={t.profile.today} value={dur(report.today.current)} />
          <Trend label={t.profile.week} value={dur(report.week.current)} />
          <Trend label={t.profile.month} value={dur(report.month.current)} />
        </View>

        <View style={s.card}>
          <Line
            label={t.profile.worked}
            value={dur(report.totals.worked)}
          />
          <Line
            label={t.profile.expected}
            value={dur(report.totals.expected)}
          />
          <Line
            label={t.profile.diff}
            value={dur(report.totals.diff)}
            tone={
              report.totals.diff >= 0
                ? STATUS_COLOR.ok
                : STATUS_COLOR.under
            }
          />
          {report.periodPay > 0 && (
            <Line label="—" value={money(report.periodPay)} />
          )}
        </View>

        <Text style={[s.h2, { marginTop: 6 }]}>{t.profile.calendar}</Text>
        <View style={local.calendar}>
          {grid.map((day, i) =>
            day === null ? (
              <View key={`pad-${i}`} style={local.cell} />
            ) : (
              <Day key={day.date} day={day} />
            ),
          )}
        </View>
        {report.days.length === 0 && (
          <Text style={[s.muted, { textAlign: "center" }]}>
            {t.profile.noData}
          </Text>
        )}

        {/* The legend, because eight colours is more than anybody memorises —
            and only the ones this period actually contains. */}
        <View style={local.legend}>
          {[...new Set(report.days.map((d) => d.status))].map((st) => (
            <View key={st} style={local.legendItem}>
              <View
                style={[local.dot, { backgroundColor: STATUS_COLOR[st] }]}
              />
              <Text style={s.muted}>{t.profile.status[st]}</Text>
            </View>
          ))}
        </View>
      </ScrollView>
    </View>
  );
}

function Day({ day }: { day: StaffDay }) {
  const { theme } = useUI();
  const colour = STATUS_COLOR[day.status];
  // ⚠️ A wash rather than a fill: a grid of eight saturated squares is a
  // pattern nobody reads, and the number on top has to stay legible in both
  // schemes. The border carries the colour at full strength.
  const soft = theme.scheme === "dark" ? "22" : "1f";
  const plain = day.status === "off" || day.status === "upcoming";
  return (
    <View
      style={[
        local.cell,
        {
          backgroundColor: plain ? "transparent" : colour + soft,
          borderColor: plain ? theme.line : colour,
          borderStyle: day.status === "upcoming" ? "dashed" : "solid",
        },
      ]}
    >
      <Text style={{ fontSize: 13, color: theme.ink }}>
        {Number(day.date.slice(8, 10))}
      </Text>
      {day.worked > 0 && (
        <Text style={{ fontSize: 9, color: theme.muted }}>
          {Math.round((day.worked / 60) * 10) / 10}
        </Text>
      )}
    </View>
  );
}

function Trend({ label, value }: { label: string; value: string }) {
  const { s } = useUI();
  return (
    <View style={[s.card, { flex: 1, padding: 12, alignItems: "center" }]}>
      <Text style={s.muted}>{label}</Text>
      <Text style={[s.h2, { marginTop: 2 }]}>{value}</Text>
    </View>
  );
}

function Line({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: string;
}) {
  const { s, theme } = useUI();
  return (
    <View style={local.line}>
      <Text style={s.soft}>{label}</Text>
      <Text style={[s.num, tone ? { color: tone } : null]}>{value}</Text>
    </View>
  );
}

const local = StyleSheet.create({
  badge: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 999,
  },
  dot: { width: 8, height: 8, borderRadius: 4 },
  trends: { flexDirection: "row", gap: 10 },
  line: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingVertical: 7,
  },
  calendar: { flexDirection: "row", flexWrap: "wrap", gap: 6 },
  cell: {
    // ⚠️ Seven per row on the narrowest phone this is sold onto. A percentage
    // width would reflow to six on a small screen and the calendar would stop
    // being a calendar.
    width: 40,
    height: 44,
    borderRadius: 10,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  legend: { flexDirection: "row", flexWrap: "wrap", gap: 12, marginTop: 4 },
  legendItem: { flexDirection: "row", alignItems: "center", gap: 6 },
});
