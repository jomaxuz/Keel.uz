import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
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
import type { Check, FloorTable, TableZone } from "@/lib/types";

import { Chip } from "./menu";
import { money } from "./money";
import { usePrefs } from "./prefs";
import { Tap } from "./press";
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
  const [zones, setZones] = useState<TableZone[]>([]);
  const [zone, setZone] = useState("");
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
      setZones(b.booking?.zones ?? []);
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

  /** The tabs across the top: the room's own parts, plus the unnamed one.
   *
   *  ⚠️ **An unzoned table is not a table without a home.** A restaurant that
   *  has never split its room keeps every table under no zone at all — the
   *  ordinary state — so a strip built only from the zone list would be empty
   *  on exactly those branches, and a strip that hid unzoned tables would hide
   *  the whole room. The unnamed part appears only when something is in it.
   *
   *  ⚠️ And with no zones at all there are no tabs: one room needs no label,
   *  and a single tab reading "the room" is a control that answers nothing. */
  const parts = useMemo(() => {
    const used = new Set(tables?.map((tb) => tb.zoneId ?? "") ?? []);
    const list = zones
      .filter((z) => used.has(z.id))
      .sort((a, b) => a.sort - b.sort)
      .map((z) => ({ id: z.id, name: z.name }));
    if (used.has("") && list.length > 0) {
      list.unshift({ id: "", name: t.floor.unzoned });
    }
    return list;
  }, [zones, tables, t.floor.unzoned]);

  const shown = useMemo(
    () =>
      parts.length === 0
        ? (tables ?? [])
        : (tables ?? []).filter((tb) => (tb.zoneId ?? "") === zone),
    [tables, parts.length, zone],
  );

  // ⚠️ The first tab is selected once the room is known, not on every render:
  // resetting the tab on each poll would send a waiter back to the first zone
  // every fifteen seconds while they were looking at the second.
  useEffect(() => {
    if (parts.length > 0 && !parts.some((p) => p.id === zone)) {
      setZone(parts[0].id);
    }
  }, [parts, zone]);

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
        <Tap onPress={() => void load()} hitSlop={12}>
          <Feather name="refresh-cw" size={19} color={theme.muted} />
        </Tap>
      </View>

      {error !== "" && <Text style={s.error}>{error}</Text>}

      {/* ⚠️ **A fixed height and no shrinking**, the same lesson the menu's
          category strip taught: a row of chips inside a column collapses the
          moment the list beside it grows, and it collapses hardest on the
          rooms with the most tables. */}
      {parts.length > 0 && (
        <View style={local.zonesRow}>
          <ScrollView
            horizontal
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={local.zones}
          >
            {parts.map((p) => (
              <Chip
                key={p.id || "none"}
                label={p.name}
                on={p.id === zone}
                onPress={() => setZone(p.id)}
              />
            ))}
          </ScrollView>
        </View>
      )}

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
        {shown.map((tb) => {
          const check = byTable.get(tb.id);
          return (
            <Tap
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
            </Tap>
          );
        })}
        {shown.length === 0 && <Text style={s.muted}>{t.floor.empty}</Text>}
      </ScrollView>
    </View>
  );
}

const local = StyleSheet.create({
  zonesRow: { height: 54, flexShrink: 0, justifyContent: "center" },
  zones: { gap: 8, paddingHorizontal: 16 },
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
