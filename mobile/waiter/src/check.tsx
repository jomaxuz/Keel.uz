import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";

import { api, ApiError } from "@/lib/api";
import type { Check, MenuGroup, MenuItem } from "@/lib/types";

import { money } from "./money";
import { theme } from "./theme";

// One table's check: what is on it, and how a dish gets added.
//
// ⚠️ **This is what "the table opened and nothing happened" was.** Opening a
// check is the *start* of the waiter's job, and the first slice had nowhere for
// it to go — so the one action the app offered ended on the screen it started
// on. A table that opens and then does nothing reads as a broken app, which is
// exactly what it was told.
//
// ⚠️ **Adding is not ordering.** Lines land unfired, the waiter reads the table
// back, corrects what they misheard, and *then* sends it. That is the whole
// reason a till is faster than shouting through a hatch, and it is the rule the
// server already enforces — repeated here because a screen that sent on every
// tap would make the rule unreachable.

export function CheckScreen({
  checkId,
  branchId,
  onBack,
}: {
  checkId: string;
  /** Which branch's menu to price against — the waiter's own. */
  branchId: string;
  onBack: () => void;
}) {
  const [check, setCheck] = useState<Check | null>(null);
  const [groups, setGroups] = useState<MenuGroup[] | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [tab, setTab] = useState<"check" | "menu">("check");
  const [category, setCategory] = useState(0);

  const load = useCallback(async () => {
    try {
      setCheck(await api.tillCheck(checkId));
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Yuklab bo'lmadi");
    }
  }, [checkId]);

  useEffect(() => {
    void load();
  }, [load]);

  // ⚠️ Loaded once, beside the check rather than on the way into the menu tab:
  // the wait belongs to opening the table, when somebody is already standing
  // at it, not to the moment a guest has just said what they want.
  useEffect(() => {
    void api
      .getMenu({ branchId })
      .then(setGroups)
      .catch(() => setGroups([]));
  }, [branchId]);

  const lines = useMemo(
    () => (check?.lines ?? []).filter((l) => !l.void),
    [check],
  );
  const unfired = lines.filter((l) => !l.fired).length;

  async function add(item: MenuItem) {
    // ⚠️ No optimism. The price, the stop list, the batch limit and the brand
    // check all live on the server, and a dish that appears and then vanishes
    // is worse than one that takes a moment to appear.
    setBusy(true);
    try {
      setCheck(await api.tillAddLines(checkId, [{ menuItemId: item.id, qty: 1 }]));
      setError("");
    } catch (e) {
      // The server's own words: "lag'mon bugun tugadi" is an answer a waiter
      // can take back to the table.
      setError(e instanceof ApiError ? e.message : "Qo'shib bo'lmadi");
    } finally {
      setBusy(false);
    }
  }

  async function fire() {
    setBusy(true);
    try {
      setCheck(await api.tillFire(checkId));
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Yuborib bo'lmadi");
    } finally {
      setBusy(false);
    }
  }

  if (!check) {
    return (
      <View style={styles.centered}>
        {error !== "" ? (
          <>
            <Text style={styles.error}>{error}</Text>
            <Pressable onPress={onBack}>
              <Text style={styles.link}>Orqaga</Text>
            </Pressable>
          </>
        ) : (
          <ActivityIndicator />
        )}
      </View>
    );
  }

  const items = groups?.[category]?.items ?? [];

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Pressable onPress={onBack} hitSlop={12}>
          <Text style={styles.link}>‹ Zal</Text>
        </Pressable>
        <Text style={styles.headerTitle}>
          {check.tableNumber ? `${check.tableNumber}-stol` : check.number}
        </Text>
        <Text style={styles.total}>{money(check.total)}</Text>
      </View>

      <View style={styles.tabs}>
        <Tab
          label={`Chek (${lines.length})`}
          on={tab === "check"}
          onPress={() => setTab("check")}
        />
        <Tab label="Menyu" on={tab === "menu"} onPress={() => setTab("menu")} />
      </View>

      {error !== "" && <Text style={styles.error}>{error}</Text>}

      {tab === "check" ? (
        <ScrollView contentContainerStyle={styles.list}>
          {lines.map((l) => (
            <View key={l.lineId} style={styles.row}>
              <Text style={styles.rowName}>
                {l.name}
                {l.qty > 1 ? ` × ${l.qty}` : ""}
              </Text>
              {/* ⚠️ An unfired line says so. The kitchen has not seen it, and
                  the difference is the one thing on this screen a guest may be
                  waiting on. */}
              {!l.fired && <Text style={styles.pending}>yuborilmagan</Text>}
              <Text style={styles.rowSum}>{money(l.sum)}</Text>
            </View>
          ))}
          {lines.length === 0 && (
            <Text style={styles.muted}>Chek bo&apos;sh — menyudan tanlang</Text>
          )}
        </ScrollView>
      ) : (
        <>
          <ScrollView horizontal style={styles.cats} showsHorizontalScrollIndicator={false}>
            {(groups ?? []).map((g, i) => (
              <Tab
                key={g.category.id}
                label={g.category.name || "—"}
                on={i === category}
                onPress={() => setCategory(i)}
              />
            ))}
          </ScrollView>
          <ScrollView contentContainerStyle={styles.list}>
            {items.map((it) => (
              <Pressable
                key={it.id}
                style={styles.row}
                disabled={busy}
                onPress={() => void add(it)}
              >
                <Text style={styles.rowName}>{it.name}</Text>
                <Text style={styles.rowSum}>{money(it.price)}</Text>
              </Pressable>
            ))}
            {groups !== null && items.length === 0 && (
              <Text style={styles.muted}>Bu bo&apos;limda taom yo&apos;q</Text>
            )}
          </ScrollView>
        </>
      )}

      {/* ⚠️ Shown only when there is something the kitchen has not seen. A
          permanent button invites being pressed on a table that is already
          cooking, and the second press is a ticket nobody asked for. */}
      {unfired > 0 && (
        <Pressable style={styles.fire} disabled={busy} onPress={() => void fire()}>
          <Text style={styles.fireText}>
            Oshxonaga yuborish ({unfired})
          </Text>
        </Pressable>
      )}
    </View>
  );
}

function Tab({
  label,
  on,
  onPress,
}: {
  label: string;
  on: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable style={[styles.tab, on && styles.tabOn]} onPress={onPress}>
      <Text style={[styles.tabText, on && styles.tabTextOn]}>{label}</Text>
    </Pressable>
  );
}


const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: theme.bg },
  centered: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 10,
    backgroundColor: theme.bg,
  },
  header: {
    paddingTop: 56,
    paddingHorizontal: 16,
    paddingBottom: 10,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
  },
  headerTitle: { fontSize: 17, fontWeight: "600", color: theme.ink },
  total: { fontSize: 17, fontWeight: "600", color: theme.ink },
  link: { fontSize: 15, color: theme.accent },
  tabs: { flexDirection: "row", gap: 8, paddingHorizontal: 16 },
  cats: { flexGrow: 0, paddingHorizontal: 16, paddingVertical: 8 },
  tab: {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: 999,
    backgroundColor: theme.surface,
    borderWidth: 1,
    borderColor: theme.line,
    marginRight: 8,
  },
  tabOn: { backgroundColor: theme.accentSoft, borderColor: theme.accent },
  tabText: { fontSize: 14, color: theme.muted },
  tabTextOn: { color: theme.ink, fontWeight: "600" },
  list: { padding: 16, gap: 8 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    backgroundColor: theme.surface,
    borderWidth: 1,
    borderColor: theme.line,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 14,
  },
  rowName: { flex: 1, fontSize: 15, color: theme.ink },
  rowSum: { fontSize: 15, color: theme.ink, fontVariant: ["tabular-nums"] },
  pending: { fontSize: 12, color: theme.accent },
  muted: { fontSize: 13, color: theme.muted, textAlign: "center", padding: 16 },
  error: { fontSize: 13, color: theme.danger, textAlign: "center", padding: 8 },
  fire: {
    margin: 16,
    backgroundColor: theme.accent,
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: "center",
  },
  fireText: { color: "#fff", fontSize: 16, fontWeight: "600" },
});
