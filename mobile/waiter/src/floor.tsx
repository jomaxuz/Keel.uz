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
import type { Check, FloorTable } from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// The room.
//
// ⚠️ **No branch picker, and that is the design.** A monoblock is bound to a
// branch because a machine stands somewhere; a person carries their own, and
// `staff.branchId` is what every till endpoint already reads. Asking a waiter
// which branch they are in is a question whose wrong answer is silent — the
// wrong room's tables, and nothing to say so.

export function FloorScreen({
  onOpenCheck,
}: {
  onOpenCheck: (checkId: string, branchId: string) => void;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [tables, setTables] = useState<FloorTable[] | null>(null);
  const [checks, setChecks] = useState<Check[]>([]);
  const [branch, setBranch] = useState("");
  const [branchId, setBranchId] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      const [b, c] = await Promise.all([api.tillBranch(), api.tillChecks()]);
      setBranch(b.name);
      setBranchId(b.id);
      setTables(b.booking?.tables ?? []);
      setChecks(c.checks);
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.floor.failedLoad);
      setTables((prev) => prev ?? []);
    }
  }, [t.floor.failedLoad]);

  useEffect(() => {
    void load();
  }, [load]);

  /** Which table each open check is sitting at.
   *
   *  ⚠️ Built once per change rather than searched per tile: a room has forty
   *  tables and this is the kind of scan that is invisible until somebody is
   *  scrolling on a four-year-old phone. */
  const byTable = useMemo(() => {
    const m = new Map<string, Check>();
    for (const c of checks) if (c.tableId) m.set(c.tableId, c);
    return m;
  }, [checks]);

  const taken = byTable.size;

  async function open(table: FloorTable) {
    // ⚠️ An occupied table is opened, not refused: the whole reason a waiter
    // taps a table that already has a check is to add to it.
    const existing = byTable.get(table.id);
    if (existing) {
      onOpenCheck(existing.id, branchId);
      return;
    }
    setBusy(table.id);
    try {
      const check = await api.tillOpenCheck({ tableId: table.id, guests: 0 });
      onOpenCheck(check.id, branchId);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.floor.failedOpen);
    } finally {
      setBusy("");
    }
  }

  if (tables === null) {
    return (
      <View style={s.centered}>
        <ActivityIndicator color={theme.accent} />
      </View>
    );
  }

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <View>
          <Text style={s.h2}>{branch || t.floor.title}</Text>
          <Text style={s.muted}>
            {taken} / {tables.length}
          </Text>
        </View>
        <Pressable onPress={() => void load()} hitSlop={12}>
          <Feather name="refresh-cw" size={19} color={theme.muted} />
        </Pressable>
      </View>

      {error !== "" && <Text style={s.error}>{error}</Text>}

      {/* ⚠️ Still a ScrollView, and it renders every child. Right for a room of
          forty and wrong for a chain's biggest hall — it becomes a FlashList
          the day a real room is measured, not before. Pull-to-refresh, because
          the room changes under a waiter while they are walking. */}
      <ScrollView
        contentContainerStyle={local.grid}
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
        {tables.map((tb) => {
          const check = byTable.get(tb.id);
          return (
            <Pressable
              key={tb.id}
              style={[
                local.table,
                {
                  backgroundColor: check ? theme.accentSoft : theme.surface,
                  borderColor: check ? theme.accent : theme.line,
                },
              ]}
              disabled={busy === tb.id}
              onPress={() => void open(tb)}
            >
              <Text style={[local.number, { color: theme.ink }]}>
                {tb.number}
              </Text>
              {/* ⚠️ Not colour alone: an occupied table says what it owes.
                  Green against red is a distinction roughly one man in twelve
                  cannot make, and the lock screen already learned this. */}
              <Text
                style={[
                  local.note,
                  { color: check ? theme.accent : theme.muted },
                ]}
              >
                {check ? money(check.total) : t.floor.free}
              </Text>
              {check && check.unfired > 0 && (
                <View style={[local.dot, { backgroundColor: theme.accent }]} />
              )}
            </Pressable>
          );
        })}
        {tables.length === 0 && <Text style={s.muted}>{t.floor.empty}</Text>}
      </ScrollView>
    </View>
  );
}

const local = StyleSheet.create({
  grid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 10,
    paddingHorizontal: 16,
    paddingBottom: 24,
  },
  table: {
    width: 104,
    height: 104,
    borderRadius: 18,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 4,
  },
  number: { fontSize: 22, fontWeight: "700" },
  note: { fontSize: 12, fontVariant: ["tabular-nums"] },
  // A table with food the kitchen has not seen yet — the one thing on this
  // screen a guest is actively waiting on.
  dot: { position: "absolute", top: 10, right: 10, width: 8, height: 8, borderRadius: 4 },
});
