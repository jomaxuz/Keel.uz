import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
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
import type { Check, MenuGroup, MenuItem } from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// One table's check: what is on it, and how a dish gets added.
//
// ⚠️ **Adding is not ordering.** Lines land unfired, the waiter reads the table
// back, corrects what they misheard, and *then* sends it. That is the whole
// reason a till is faster than shouting through a hatch, and it is the rule the
// server already enforces — repeated on the screen because a phone that sent on
// every tap would put the rule out of reach.

export function CheckScreen({
  checkId,
  branchId,
  onBack,
}: {
  checkId: string;
  branchId: string;
  onBack: () => void;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
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
      setError(e instanceof ApiError ? e.message : t.floor.failedLoad);
    }
  }, [checkId, t.floor.failedLoad]);

  useEffect(() => {
    void load();
  }, [load]);

  // ⚠️ Loaded beside the check rather than on the way into the menu tab: the
  // wait belongs to opening the table, when somebody is already standing at it,
  // not to the moment a guest has just said what they want.
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
      setCheck(
        await api.tillAddLines(checkId, [{ menuItemId: item.id, qty: 1 }]),
      );
      setError("");
    } catch (e) {
      // The server's own words: "lag'mon bugun tugadi" is an answer a waiter
      // can take back to the table.
      setError(e instanceof ApiError ? e.message : t.check.failedAdd);
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
      setError(e instanceof ApiError ? e.message : t.check.failedFire);
    } finally {
      setBusy(false);
    }
  }

  if (!check) {
    return (
      <View style={s.centered}>
        {error !== "" ? (
          <>
            <Text style={s.error}>{error}</Text>
            <Pressable onPress={onBack}>
              <Text style={s.link}>{t.check.back}</Text>
            </Pressable>
          </>
        ) : (
          <ActivityIndicator color={theme.accent} />
        )}
      </View>
    );
  }

  const items = groups?.[category]?.items ?? [];

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Pressable onPress={onBack} hitSlop={12} style={local.back}>
          <Feather name="chevron-left" size={20} color={theme.accent} />
          <Text style={s.link}>{t.check.back}</Text>
        </Pressable>
        <Text style={s.h2}>
          {check.tableNumber ? t.check.table(check.tableNumber) : check.number}
        </Text>
        <Text style={[s.h2, { fontVariant: ["tabular-nums"] }]}>
          {money(check.total)}
        </Text>
      </View>

      <View style={local.tabs}>
        <Chip
          label={`${t.check.tab} · ${lines.length}`}
          icon="file-text"
          on={tab === "check"}
          onPress={() => setTab("check")}
        />
        <Chip
          label={t.check.menu}
          icon="grid"
          on={tab === "menu"}
          onPress={() => setTab("menu")}
        />
      </View>

      {error !== "" && <Text style={[s.error, { paddingTop: 8 }]}>{error}</Text>}

      {tab === "check" ? (
        <ScrollView contentContainerStyle={s.list}>
          {lines.map((l) => (
            <View key={l.lineId} style={s.row}>
              <View style={{ flex: 1 }}>
                <Text style={s.body}>
                  {l.name}
                  {l.qty > 1 ? ` × ${l.qty}` : ""}
                </Text>
                {/* ⚠️ An unfired line says so. The kitchen has not seen it, and
                    that is the one thing here a guest may be waiting on. */}
                {!l.fired && (
                  <Text style={[s.muted, { color: theme.accent }]}>
                    {t.check.pending}
                  </Text>
                )}
              </View>
              <Text style={s.num}>{money(l.sum)}</Text>
            </View>
          ))}
          {lines.length === 0 && (
            <Text style={[s.muted, local.empty]}>{t.check.empty}</Text>
          )}
        </ScrollView>
      ) : (
        <>
          <ScrollView
            horizontal
            style={local.cats}
            contentContainerStyle={{ gap: 8, paddingHorizontal: 16 }}
            showsHorizontalScrollIndicator={false}
          >
            {(groups ?? []).map((g, i) => (
              <Chip
                key={g.category.id}
                label={g.category.name || "—"}
                on={i === category}
                onPress={() => setCategory(i)}
              />
            ))}
          </ScrollView>
          <ScrollView contentContainerStyle={s.list}>
            {items.map((it) => (
              <Pressable
                key={it.id}
                style={s.row}
                disabled={busy}
                onPress={() => void add(it)}
              >
                <Text style={[s.body, { flex: 1 }]}>{it.name}</Text>
                <Text style={s.num}>{money(it.price)}</Text>
                <Feather name="plus" size={18} color={theme.accent} />
              </Pressable>
            ))}
            {groups !== null && items.length === 0 && (
              <Text style={[s.muted, local.empty]}>{t.check.noItems}</Text>
            )}
          </ScrollView>
        </>
      )}

      {/* ⚠️ Shown only when there is something the kitchen has not seen. A
          permanent button invites being pressed on a table that is already
          cooking, and the second press is a ticket nobody asked for. */}
      {unfired > 0 && (
        <Pressable
          style={[s.primary, local.fire]}
          disabled={busy}
          onPress={() => void fire()}
        >
          <Feather name="send" size={18} color={theme.onAccent} />
          <Text style={s.primaryText}>{t.check.fire(unfired)}</Text>
        </Pressable>
      )}
    </View>
  );
}

export function Chip({
  label,
  icon,
  on,
  onPress,
}: {
  label: string;
  icon?: keyof typeof Feather.glyphMap;
  on: boolean;
  onPress: () => void;
}) {
  const { theme } = useUI();
  return (
    <Pressable
      style={[
        local.chip,
        {
          backgroundColor: on ? theme.accentSoft : theme.surface,
          borderColor: on ? theme.accent : theme.line,
        },
      ]}
      onPress={onPress}
    >
      {icon && (
        <Feather
          name={icon}
          size={14}
          color={on ? theme.accent : theme.muted}
        />
      )}
      <Text
        style={{
          fontSize: 14,
          color: on ? theme.ink : theme.muted,
          fontWeight: on ? "600" : "400",
        }}
      >
        {label}
      </Text>
    </Pressable>
  );
}

const local = StyleSheet.create({
  back: { flexDirection: "row", alignItems: "center", gap: 2 },
  tabs: { flexDirection: "row", gap: 8, paddingHorizontal: 16 },
  cats: { flexGrow: 0, paddingVertical: 10 },
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 9,
    borderRadius: 999,
    borderWidth: 1,
  },
  empty: { textAlign: "center", padding: 20 },
  fire: { margin: 16, marginTop: 4 },
});
