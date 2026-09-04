import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
// ⚠️ From the family's own path, not the package index — see profile.tsx.
import Feather from "@expo/vector-icons/Feather";

import { api, ApiError } from "@/lib/api";
import type {
  ShoppingCatalogRow,
  ShoppingDraftRow,
  ShoppingOrder,
  Staff,
} from "@/lib/types";

import { usePrefs } from "./prefs";
import { Tap } from "./press";
import { useTopInset, useUI } from "./ui";

// Writing the list somebody is sent to the market with.
//
// ⚠️ **The other half of the buying, and the reason it is a separate
// permission.** One person decides what the restaurant needs; another goes and
// gets it and writes down what it cost. Held by the same account the list stops
// being a check on the trip and becomes a note the buyer wrote to themselves.
//
// ⚠️ **The form starts from the shortage the store already worked out.** A
// blank page would be a second, competing shopping list — one the arithmetic
// produces and one a person types — and the buyer would have no way to tell
// which is real.
//
// ⚠️ **The unit is shown and never chosen.** A market sells mint in bunches and
// flour in sacks; "5" typed into a field measured in kilos is five kilos on the
// shelf instead of a quarter of one, and the gap surfaces a month later at a
// count as a shortfall somebody is asked to explain.

/** Whether this account writes shopping lists **in this app**.
 *
 * ⚠️ **Narrower than the permission, and deliberately so.** A cashier holds
 * `buyorder` because the till is where they first hear that something has run
 * out — but their phone is not where a restaurant's buying gets planned, and a
 * section they never use is a section that teaches them to ignore the app.
 *
 * ⚠️ **Expressed in permissions, never in the role's name.** The spelling of a
 * job title grants nothing (models/staffrole.go), so "management or the
 * storekeeper" is asked as "may void, or counts the store" — the same test
 * `tillPersonView.CanExit` already uses to mean management.
 */
export function canWriteHere(staff: Staff): boolean {
  const perms = staff.perms ?? [];
  return (
    perms.includes("buyorder") &&
    (perms.includes("void") || perms.includes("stock"))
  );
}

type Draft = {
  key: string;
  ingredientId?: string;
  name: string;
  unit: string;
  qty: string;
  onHand?: number;
  packName?: string;
  packQty?: number;
  /** Whether the number typed counts packs. ⚠️ A flag, not a converted figure —
   *  the server owns the arithmetic, because the result is what somebody is
   *  sent to buy. */
  pack?: boolean;
};

export function ZakupScreen() {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  // ⚠️ The screen has no header of its own, so nothing else clears the status
  // bar — the first line sat under the clock. See useTopInset.
  const top = useTopInset();

  const [suggested, setSuggested] = useState<ShoppingDraftRow[]>([]);
  /** ⚠️ Needed as well as the shortage: a list can ask for something above its
   *  minimum, and without the catalogue the writer had to type the name — which
   *  creates a second ingredient no tech card points at. */
  const [catalog, setCatalog] = useState<ShoppingCatalogRow[]>([]);
  const [orders, setOrders] = useState<ShoppingOrder[]>([]);
  const [lines, setLines] = useState<Draft[]>([]);
  const [query, setQuery] = useState("");
  const [forDate, setForDate] = useState(() => {
    // Tomorrow: today's shopping has been done by the time anybody writes this.
    const d = new Date();
    d.setDate(d.getDate() + 1);
    return d.toISOString().slice(0, 10);
  });
  const [preview, setPreview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      const [draft, sent] = await Promise.all([
        api.staffBuyOrderDraft(),
        api.staffBuyOrders(),
      ]);
      setSuggested(draft.rows);
      setCatalog(draft.catalog ?? []);
      setOrders(sent.orders);
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.zakup.loadFailed);
    }
  }, [t.zakup.loadFailed]);

  useEffect(() => {
    void load();
  }, [load]);

  const chosen = useMemo(
    () => new Set(lines.map((l) => l.ingredientId).filter(Boolean)),
    [lines],
  );
  /** ⚠️ Short things first, then the rest of the catalogue once something is
   *  typed. With nothing typed only the shortage shows, or the screen opens as
   *  two hundred rows nobody scrolls. */
  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    const short = suggested.filter(
      (r) =>
        !chosen.has(r.ingredientId) &&
        (q === "" || r.name.toLowerCase().includes(q)),
    );
    if (q === "") return short;
    const ids = new Set(short.map((r) => r.ingredientId));
    const rest = catalog
      .filter(
        (c) =>
          !chosen.has(c.ingredientId) &&
          !ids.has(c.ingredientId) &&
          c.name.toLowerCase().includes(q),
      )
      .slice(0, 20)
      .map((c) => ({ ...c, qty: 0, onHand: 0 }) as ShoppingDraftRow);
    return [...short, ...rest];
  }, [suggested, catalog, chosen, query]);

  /** A name the store does not know. ⚠️ Offered rather than refused: a list
   *  somebody cannot finish writing is a list they write on paper instead, and
   *  then nothing here sees it. */
  const unknown = useMemo(() => {
    const q = query.trim();
    if (q.length < 2) return "";
    return catalog.some((r) => r.name.toLowerCase() === q.toLowerCase())
      ? ""
      : q;
  }, [catalog, query]);

  function add(row: Partial<Draft> & { name: string }) {
    setDone("");
    setLines((cur) => [
      ...cur,
      {
        key: row.ingredientId ?? `new-${Date.now()}`,
        ingredientId: row.ingredientId,
        name: row.name,
        unit: row.unit ?? "",
        qty: row.qty ?? "",
        onHand: row.onHand,
        packName: row.packName,
        packQty: row.packQty,
      },
    ]);
    setQuery("");
  }

  const ready = lines.filter((l) => Number(l.qty) > 0);

  async function send() {
    setBusy(true);
    setError("");
    try {
      await api.staffCreateBuyOrder({
        forDate,
        lines: ready.map((l) => ({
          ingredientId: l.ingredientId,
          name: l.name,
          qty: Number(l.qty),
          pack: l.pack,
        })),
      });
      setLines([]);
      setPreview(false);
      setDone(t.zakup.sent(ready.length));
      void load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.zakup.sendFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <ScrollView
      style={s.screen}
      contentContainerStyle={[local.body, { paddingTop: top + 12 }]}
      keyboardShouldPersistTaps="handled"
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
      {error !== "" && <Text style={[s.error, local.gap]}>{error}</Text>}
      {done !== "" && (
        <Text style={[local.done, { color: theme.accent }]}>{done}</Text>
      )}

      <Text style={[s.h2, local.gap]}>{t.zakup.forDate}</Text>
      <TextInput
        style={s.input}
        value={forDate}
        placeholder="2026-09-05"
        placeholderTextColor={theme.muted}
        onChangeText={setForDate}
      />

      {lines.length > 0 && (
        <>
          <Text style={[s.h2, local.gap]}>{t.zakup.listTitle}</Text>
          {lines.map((l) => (
            <View key={l.key} style={[local.line, { borderColor: theme.line }]}>
              <View style={local.head}>
                <Text style={[s.body, { flex: 1 }]}>{l.name}</Text>
                <Pressable
                  onPress={() =>
                    setLines((cur) => cur.filter((x) => x.key !== l.key))
                  }
                  hitSlop={12}
                >
                  <Feather name="x" size={18} color={theme.muted} />
                </Pressable>
              </View>
              <View style={local.head}>
                <TextInput
                  style={[s.input, { flex: 1 }]}
                  keyboardType="decimal-pad"
                  value={l.qty}
                  placeholder={t.zakup.qty}
                  placeholderTextColor={theme.muted}
                  onChangeText={(v) =>
                    setLines((cur) =>
                      cur.map((x) =>
                        x.key === l.key
                          ? { ...x, qty: v.replace(",", ".") }
                          : x,
                      ),
                    )
                  }
                />
                {/* ⚠️ Tapped where a market packaging exists, a label where it
                    does not. The unit is never free text: "5" typed into a
                    field measured in kilos is five kilos on a shelf. */}
                {l.packName && l.packQty ? (
                  <Pressable
                    onPress={() =>
                      setLines((cur) =>
                        cur.map((x) =>
                          x.key === l.key ? { ...x, pack: !x.pack } : x,
                        ),
                      )
                    }
                  >
                    <Text
                      style={[
                        s.muted,
                        { width: 64, textAlign: "center", color: theme.accent },
                      ]}
                    >
                      {l.pack ? l.packName : l.unit} ⇄
                    </Text>
                  </Pressable>
                ) : (
                  <Text style={[s.muted, { width: 64, textAlign: "center" }]}>
                    {l.unit}
                  </Text>
                )}
              </View>
            </View>
          ))}
        </>
      )}

      <Text style={[s.h2, local.gap]}>{t.zakup.shortTitle}</Text>
      <TextInput
        style={s.input}
        value={query}
        placeholder={t.zakup.search}
        placeholderTextColor={theme.muted}
        onChangeText={setQuery}
      />
      {shown.length === 0 && unknown === "" ? (
        <Text style={s.muted}>{t.zakup.nothingShort}</Text>
      ) : (
        <>
          {shown.map((row) => (
            <Tap
              key={row.ingredientId}
              onPress={() =>
                add({
                  ingredientId: row.ingredientId,
                  name: row.name,
                  unit: row.unit,
                  qty: row.qty > 0 ? String(row.qty) : "",
                  onHand: row.onHand,
                })
              }
              style={[local.row, { borderColor: theme.line }]}
            >
              <View style={{ flex: 1 }}>
                <Text style={s.body}>{row.name}</Text>
                {/* A catalogue row has no shortage figures — only its unit,
                    which is what the writer needs before typing a number. */}
                <Text style={s.muted}>
                  {row.qty > 0
                    ? `${t.zakup.onHand(row.onHand, row.unit)} · ${t.zakup.need(row.qty, row.unit)}`
                    : t.zakup.unitIs(row.unit)}
                </Text>
              </View>
              <Feather name="plus" size={20} color={theme.accent} />
            </Tap>
          ))}
          {unknown !== "" && (
            <Tap
              onPress={() => add({ name: unknown })}
              style={[local.row, { borderColor: theme.line }]}
            >
              <Feather name="plus-circle" size={18} color={theme.accent} />
              <Text style={[s.body, { flex: 1, marginLeft: 8 }]}>
                {t.zakup.addNew(unknown)}
              </Text>
            </Tap>
          )}
        </>
      )}

      {ready.length > 0 && (
        <Tap
          onPress={() => setPreview(true)}
          style={[local.send, { backgroundColor: theme.accent }]}
        >
          <Text style={local.sendText}>{t.zakup.review}</Text>
        </Tap>
      )}

      {orders.length > 0 && (
        <>
          <Text style={[s.h2, local.gap]}>{t.zakup.sentTitle}</Text>
          {orders.slice(0, 8).map((o) => (
            <View key={o.id} style={[local.row, { borderColor: theme.line }]}>
              <View style={{ flex: 1 }}>
                <Text style={s.body}>{o.forDate}</Text>
                {/* ⚠️ Asked and brought together: "asked for ten, brought six"
                    is the sentence this document exists to make possible. */}
                <Text style={s.muted}>
                  {t.zakup.progress(
                    o.lines.filter((l) => l.gotAt && !l.missing).length,
                    o.lines.length,
                  )}
                  {o.createdBy ? ` · ${o.createdBy}` : ""}
                </Text>
              </View>
              <Text style={s.muted}>
                {o.status === "done" ? t.zakup.statusDone : t.zakup.statusSent}
              </Text>
            </View>
          ))}
        </>
      )}

      {/* ⚠️ **A list is somebody else's morning.** The buyer will not be able to
          ask what "5" meant, so the numbers and the units are read back once on
          one page, in the words they will arrive in. */}
      {preview && (
        <View style={local.sheet}>
          <View style={[local.card, { backgroundColor: theme.surface }]}>
            <Text style={s.h1}>{t.zakup.previewTitle}</Text>
            <Text style={s.muted}>{t.zakup.previewBody(forDate)}</Text>
            {ready.map((l) => (
              <View key={l.key} style={local.head}>
                <Text style={[s.body, { flex: 1 }]}>{l.name}</Text>
                {/* ⚠️ Read back in both where they differ: the list travels to
                    somebody else's morning and "2" has to be unambiguous before
                    it leaves. */}
                <Text style={s.body}>
                  {l.pack && l.packQty
                    ? `${l.qty} ${l.packName} = ${Number(l.qty) * l.packQty} ${l.unit}`
                    : `${l.qty} ${l.unit}`}
                </Text>
              </View>
            ))}
            <View style={local.head}>
              <Tap
                onPress={() => setPreview(false)}
                style={[local.half, { borderWidth: 1, borderColor: theme.line }]}
              >
                <Text style={s.body}>{t.zakup.back}</Text>
              </Tap>
              <Tap
                onPress={() => void send()}
                disabled={busy}
                style={[local.half, { backgroundColor: theme.accent }]}
              >
                {busy ? (
                  <ActivityIndicator color="#fff" />
                ) : (
                  <Text style={local.sendText}>{t.zakup.send}</Text>
                )}
              </Tap>
            </View>
          </View>
        </View>
      )}
    </ScrollView>
  );
}

const local = StyleSheet.create({
  body: { padding: 16, paddingBottom: 40, gap: 8 },
  gap: { marginTop: 12 },
  done: { marginTop: 12, fontWeight: "600" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  line: { borderWidth: 1, borderRadius: 12, padding: 12, gap: 8 },
  head: { flexDirection: "row", alignItems: "center", gap: 10 },
  send: {
    marginTop: 12,
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: "center",
  },
  sendText: { color: "#fff", fontSize: 16, fontWeight: "700" },
  sheet: {
    position: "absolute",
    inset: 0,
    justifyContent: "flex-end",
    backgroundColor: "rgba(0,0,0,0.4)",
    padding: 16,
  },
  card: { borderRadius: 18, padding: 16, gap: 10 },
  half: { flex: 1, borderRadius: 12, paddingVertical: 14, alignItems: "center" },
});
