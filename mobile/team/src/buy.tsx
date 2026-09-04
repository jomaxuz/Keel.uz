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
  AdvanceBalance,
  BuyCatalogRow,
  BuyLineInput,
  ShoppingGroup,
  ShoppingOrder,
} from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { Tap } from "./press";
import { useUI } from "./ui";

// The market run, written at the stall.
//
// ⚠️ **The delivery a restaurant actually has, and the one that was recorded
// worst.** A supplier sends an invoice; a market run sends somebody with cash
// at six in the morning, and what came back was written on a scrap of paper,
// carried to an office and typed in by whoever had time — a day late, and
// sometimes not at all. Everything downstream stands on it: the shelf, the
// shopping list that decides the *next* run, the cost of every dish, the stop
// list.
//
// ⚠️ **Nothing here computes anything the server has not.** The shopping list
// arrives from `/staff/buy/list` — the same function the panel's screen reads —
// because a buyer at a market and an owner in an office must not disagree about
// whether the kitchen is out of beef. That argument happens after the money is
// spent.
//
// ⚠️ **The last price is beside every box on purpose.** It is the only guard
// the figure has: a price typed here reprices every dish that uses the
// ingredient, and `9 000` entered as `90 000` looks like an ordinary number
// forever afterwards. Nothing downstream can tell them apart. A person standing
// at the stall can — if the last one is in front of them.

/** One line being built. Kept as text, not numbers: a half-typed "12" must not
 *  become 12 and then 120 as somebody types 120, and an emptied box has to stay
 *  empty rather than snapping back to 0. */
type Draft = {
  key: string;
  ingredientId?: string;
  name: string;
  unit: string;
  lastPrice: number;
  qty: string;
  price: string;
  /** How the market sells it, when somebody wrote it down. */
  packName?: string;
  packQty?: number;
  /** Whether the figures typed count packs. ⚠️ A flag, not a converted number:
   *  the factor is a fact about the ingredient and the result lands on a shelf,
   *  so the server does the arithmetic. */
  pack?: boolean;
};

export function BuyScreen() {
  const { t } = usePrefs();
  const { theme, s } = useUI();

  const [purse, setPurse] = useState<AdvanceBalance | null>(null);
  /** The list somebody sent this buyer. ⚠️ When there is one it **replaces**
   *  the computed shortage: two shopping lists on one screen is a buyer with no
   *  way to tell which one the restaurant actually asked for, and the one they
   *  follow will be whichever they saw last. */
  const [order, setOrder] = useState<ShoppingOrder | null>(null);
  const [groups, setGroups] = useState<ShoppingGroup[]>([]);
  const [catalog, setCatalog] = useState<BuyCatalogRow[]>([]);
  const [lines, setLines] = useState<Draft[]>([]);
  const [query, setQuery] = useState("");
  const [supplier, setSupplier] = useState("");
  const [error, setError] = useState("");
  const [done, setDone] = useState("");
  const [busy, setBusy] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      const [list, cat, bal, sent] = await Promise.all([
        api.staffBuyList(),
        api.staffBuyCatalog(),
        api.staffBuyBalance(),
        api.staffBuyOrders(true),
      ]);
      setGroups(list.groups);
      setCatalog(cat.ingredients);
      setPurse(bal);
      setOrder(sent.orders[0] ?? null);
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.buy.loadFailed);
    }
  }, [t.buy.loadFailed]);

  useEffect(() => {
    void load();
  }, [load]);

  /** Everything short, flattened. ⚠️ The grouping by supplier is the panel's
   *  question — who to ring — and a buyer already standing at a market has one
   *  list to walk, not five. */
  const shortlist = useMemo(
    () => groups.flatMap((g) => g.rows),
    [groups],
  );

  const add = useCallback((row: Partial<Draft> & { name: string }) => {
    setDone("");
    setLines((cur) => {
      // ⚠️ Twice is one line, not two: the same tile pressed again at a stall
      // is somebody checking, not a second bag of the same thing.
      if (row.ingredientId && cur.some((l) => l.ingredientId === row.ingredientId)) {
        return cur;
      }
      return [
        ...cur,
        {
          key: row.ingredientId ?? `new-${Date.now()}`,
          ingredientId: row.ingredientId,
          name: row.name,
          unit: row.unit ?? "",
          lastPrice: row.lastPrice ?? 0,
          qty: row.qty ?? "",
          price: row.price ? String(row.price) : "",
          packName: row.packName,
          packQty: row.packQty,
        },
      ];
    });
  }, []);

  const total = useMemo(
    () =>
      lines.reduce(
        (sum, l) => sum + (Number(l.qty) || 0) * (Number(l.price) || 0),
        0,
      ),
    [lines],
  );

  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return [];
    return catalog
      .filter((c) => c.name.toLowerCase().includes(q))
      .slice(0, 12);
  }, [catalog, query]);

  /** Whether what was typed is a name nothing in the catalogue answers to.
   *
   *  ⚠️ **Offered rather than refused.** A buyer who cannot record half a run
   *  stops recording any of it — the lesson this product paid for with the
   *  supplier field and the void reason. The server matches the name against
   *  what exists before inventing anything, and marks whatever it does invent
   *  for somebody to finish. */
  const unknown = useMemo(() => {
    const q = query.trim();
    if (q.length < 2) return "";
    return catalog.some((c) => c.name.toLowerCase() === q.toLowerCase())
      ? ""
      : q;
  }, [catalog, query]);

  /** What came back for one line of the list somebody sent.
   *
   *  ⚠️ **The shelf does not move here.** Ticking is a fact about the list; the
   *  stock only changes when the trip is finished. A line that raised stock on
   *  a tick would put food on the shelf while the buyer was still at the
   *  market, and an untick would then have to take it off — a correction
   *  nothing downstream could tell from a theft. */
  async function mark(
    lineId: string,
    body: { qty?: number; price?: number; missing?: boolean; clear?: boolean },
  ) {
    if (!order) return;
    try {
      setOrder(await api.staffMarkBuyOrderLine(order.id, lineId, body));
      setDone("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.buy.sendFailed);
    }
  }

  /** Close the trip: everything ticked becomes one delivery. */
  async function finish() {
    if (!order) return;
    setBusy(true);
    setError("");
    try {
      const res = await api.staffFinishBuyOrder(order.id, {
        // ⚠️ Minted here and kept for the retry, exactly as a free-form run
        // does: a market has worse signal than a dining room, and without it a
        // resend after a timeout is a second delivery.
        clientId: `ord-${order.id}-${Date.now().toString(36)}`,
        supplier: supplier.trim(),
      });
      setSupplier("");
      setDone(
        res.already
          ? t.buy.alreadySent
          : t.buy.sent(money(res.purchase?.total ?? 0)),
      );
      void load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.buy.sendFailed);
    } finally {
      setBusy(false);
    }
  }

  async function send() {
    const ready: BuyLineInput[] = lines
      .filter((l) => Number(l.qty) > 0)
      .map((l) => ({
        ingredientId: l.ingredientId,
        newName: l.ingredientId ? undefined : l.name,
        qty: Number(l.qty),
        price: Number(l.price) || 0,
        pack: l.pack,
      }));
    if (ready.length === 0) {
      setError(t.buy.nothingToSend);
      return;
    }
    setBusy(true);
    setError("");
    try {
      // ⚠️ **The id is minted here and kept for the retry.** A market has worse
      // signal than a dining room; without it a resend after a timeout is a
      // second delivery — the shelf raised twice and the invoice paid twice.
      const res = await api.staffBuyCreate({
        clientId: `buy-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        supplier: supplier.trim(),
        lines: ready,
      });
      setLines([]);
      setSupplier("");
      setQuery("");
      setDone(
        res.already
          ? t.buy.alreadySent
          : t.buy.sent(money(res.purchase.total ?? 0)),
      );
      void load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.buy.sendFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <ScrollView
      style={s.screen}
      contentContainerStyle={local.body}
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
      {/* ---- What is still in the buyer's pocket ----
          ⚠️ **First, and before the list.** It is the number that decides
          whether this trip happens at all, and somebody who has to ring the
          office to find out how much they are carrying will guess instead —
          which is the state this ledger exists to end.

          ⚠️ **Negative is shown rather than clamped.** A buyer who ran out and
          paid for the last crate themselves is owed money, and a purse that
          stopped at zero would be silent about exactly the debt they are
          waiting on. */}
      {purse !== null && (
        <View style={[local.purse, { borderColor: theme.line }]}>
          <Text style={s.muted}>{t.buy.purse}</Text>
          <Text
            style={[
              local.purseValue,
              { color: purse.balance < 0 ? theme.danger : theme.ink },
            ]}
          >
            {money(purse.balance)}
          </Text>
          {purse.balance < 0 && (
            <Text style={s.muted}>{t.buy.purseOwed}</Text>
          )}
          {purse.issued > 0 && (
            <Text style={s.muted}>
              {t.buy.purseOf(money(purse.issued), money(purse.spent))}
            </Text>
          )}
        </View>
      )}

      {error !== "" && <Text style={[s.error, local.gap]}>{error}</Text>}
      {done !== "" && (
        <Text style={[local.done, { color: theme.accent }]}>{done}</Text>
      )}

      {/* ---- The list somebody sent ----
          ⚠️ **It replaces the computed shortage rather than sitting beside it.**
          Two shopping lists on one screen leaves the buyer no way to tell which
          one the restaurant actually asked for — and the one they follow will
          be whichever they saw last.

          ⚠️ **Asked and brought are shown together.** "Asked for ten, brought
          six" is the sentence this whole document exists to make possible; a
          screen showing only the result would leave the same silence the buying
          had before. */}
      {order !== null && (
        <>
          <Text style={[s.h2, local.gap]}>{t.buy.orderTitle(order.forDate)}</Text>
          {order.createdBy ? (
            <Text style={s.muted}>{t.buy.orderFrom(order.createdBy)}</Text>
          ) : null}
          {order.lines.map((l) => (
            <OrderLine
              key={l.id}
              line={l}
              // ⚠️ Read off the catalogue rather than stored on the line: the
              // list holds one number in the unit the store keeps, and how a
              // market happens to sell it is a fact about the ingredient.
              pack={catalog.find((c) => c.id === l.ingredientId)}
              onMark={(body) => void mark(l.id, body)}
            />
          ))}
          <Text style={[s.h2, local.gap]}>{t.buy.whereTitle}</Text>
          <TextInput
            style={s.input}
            value={supplier}
            placeholder={t.buy.wherePlaceholder}
            placeholderTextColor={theme.muted}
            onChangeText={setSupplier}
          />
          <Tap
            onPress={() => void finish()}
            disabled={busy}
            style={[local.send, { backgroundColor: theme.accent }]}
          >
            {busy ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={local.sendText}>{t.buy.finish}</Text>
            )}
          </Tap>
          <Text style={[s.muted, local.gap]}>{t.buy.sendHint}</Text>
        </>
      )}

      {/* ---- What the kitchen is short of ----
          ⚠️ The list the buyer leaves with is the list the owner reads, from one
          function on the server. Drawn only when nobody sent a list. */}
      {order === null && lines.length === 0 && (
        <>
          <Text style={[s.h2, local.gap]}>{t.buy.shortTitle}</Text>
          {shortlist.length === 0 ? (
            <Text style={s.muted}>{t.buy.nothingShort}</Text>
          ) : (
            shortlist.map((row) => (
              <Tap
                key={row.ingredientId}
                onPress={() =>
                  add({
                    ingredientId: row.ingredientId,
                    name: row.name,
                    unit: row.unit,
                    lastPrice: row.price,
                    // ⚠️ Pre-filled with what is short, **not** locked to it:
                    // a market sells what it has, and a buyer who came back
                    // with more than the list asked for must be able to say so.
                    qty: String(row.suggested),
                    price: String(row.price),
                  })
                }
                style={[local.row, { borderColor: theme.line }]}
              >
                <View style={{ flex: 1 }}>
                  <Text style={s.body}>{row.name}</Text>
                  <Text style={s.muted}>
                    {t.buy.onHand(row.onHand, row.unit)} ·{" "}
                    {t.buy.need(row.suggested, row.unit)}
                  </Text>
                </View>
                <Feather name="plus" size={20} color={theme.accent} />
              </Tap>
            ))
          )}
        </>
      )}

      {/* ---- The run being written ---- */}
      {lines.length > 0 && (
        <>
          <Text style={[s.h2, local.gap]}>{t.buy.basketTitle}</Text>
          {lines.map((l) => (
            <View key={l.key} style={[local.line, { borderColor: theme.line }]}>
              <View style={local.lineHead}>
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
              <View style={local.fields}>
                <View style={{ flex: 1 }}>
                  {/* ⚠️ **Tapped, not typed.** Where a market packaging is
                      written down the unit becomes a two-way switch — kilos or
                      bunches — and the conversion is the server's. Where none
                      is, this is a label: the unit stays the store's, which is
                      the whole reason it is never free text. */}
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
                      <Text style={[s.muted, { color: theme.accent }]}>
                        {t.buy.qty(l.pack ? l.packName : l.unit)} ⇄
                      </Text>
                    </Pressable>
                  ) : (
                    <Text style={s.muted}>{t.buy.qty(l.unit)}</Text>
                  )}
                  <TextInput
                    style={[s.input, local.field]}
                    keyboardType="decimal-pad"
                    value={l.qty}
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
                </View>
                <View style={{ flex: 1 }}>
                  <Text style={s.muted}>{t.buy.price}</Text>
                  <TextInput
                    style={[s.input, local.field]}
                    keyboardType="number-pad"
                    value={l.price}
                    placeholderTextColor={theme.muted}
                    onChangeText={(v) =>
                      setLines((cur) =>
                        cur.map((x) =>
                          x.key === l.key
                            ? { ...x, price: v.replace(/\D/g, "") }
                            : x,
                        ),
                      )
                    }
                  />
                  {/* ⚠️ **The guard, and the only one this figure has.** */}
                  {l.lastPrice > 0 && (
                    <Text style={s.muted}>
                      {t.buy.lastPrice(money(l.lastPrice))}
                    </Text>
                  )}
                </View>
              </View>
            </View>
          ))}
        </>
      )}

      {/* ---- Anything else the market had ----
          ⚠️ Hidden while a list is open: a buyer recording the same crate twice
          — once as a ticked line and once as a free-form row — would raise the
          shelf twice, and nothing on any screen would say so. */}
      {order === null && (
        <>
      <Text style={[s.h2, local.gap]}>{t.buy.addTitle}</Text>
      <TextInput
        style={s.input}
        value={query}
        placeholder={t.buy.searchPlaceholder}
        placeholderTextColor={theme.muted}
        onChangeText={setQuery}
      />
      {shown.map((c) => (
        <Tap
          key={c.id}
          onPress={() => {
            add({
              ingredientId: c.id,
              name: c.name,
              unit: c.unit,
              lastPrice: c.lastPrice,
              price: c.lastPrice > 0 ? String(c.lastPrice) : "",
              packName: c.packName,
              packQty: c.packQty,
            });
            setQuery("");
          }}
          style={[local.row, { borderColor: theme.line }]}
        >
          <Text style={[s.body, { flex: 1 }]}>{c.name}</Text>
          {c.lastPrice > 0 && (
            <Text style={s.muted}>{money(c.lastPrice)}</Text>
          )}
        </Tap>
      ))}
      {unknown !== "" && (
        <Tap
          onPress={() => {
            add({ name: unknown });
            setQuery("");
          }}
          style={[local.row, { borderColor: theme.line }]}
        >
          <Feather name="plus-circle" size={18} color={theme.accent} />
          <Text style={[s.body, { flex: 1, marginLeft: 8 }]}>
            {t.buy.addNew(unknown)}
          </Text>
        </Tap>
      )}

      {lines.length > 0 && (
        <>
          <Text style={[s.h2, local.gap]}>{t.buy.whereTitle}</Text>
          <TextInput
            style={s.input}
            value={supplier}
            placeholder={t.buy.wherePlaceholder}
            placeholderTextColor={theme.muted}
            onChangeText={setSupplier}
          />

          <View style={[local.total, { borderColor: theme.line }]}>
            <Text style={s.body}>{t.buy.total}</Text>
            <Text style={[s.body, local.totalValue]}>{money(total)}</Text>
          </View>

          <Tap
            onPress={() => void send()}
            disabled={busy}
            style={[local.send, { backgroundColor: theme.accent }]}
          >
            {busy ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={local.sendText}>{t.buy.send}</Text>
            )}
          </Tap>
          {/* ⚠️ Said before the button rather than after the fact: this raises
              the shelf and rewrites prices the moment it lands, and the person
              pressing it should know that is what they are doing. */}
          <Text style={[s.muted, local.gap]}>{t.buy.sendHint}</Text>
        </>
      )}
        </>
      )}
    </ScrollView>
  );
}

/** One line of the list somebody sent, and what came back of it.
 *
 * ⚠️ **The unit is shown and never chosen.** A market sells mint in bunches and
 * flour in sacks; "5" typed into a field measured in kilos puts five kilos on
 * the shelf instead of a quarter of one, the figure is then twenty times too
 * high, the stop list never fires, and the gap surfaces a month later at a
 * count as a shortfall the person holding the clipboard is asked to explain.
 *
 * ⚠️ **"Could not get it" is its own answer, not a quantity of zero.** A line
 * nobody touched and one somebody looked for and could not find are different
 * facts, and only the second is worth ringing a supplier about.
 */
function OrderLine({
  line,
  pack,
  onMark,
}: {
  line: ShoppingOrder["lines"][number];
  pack?: BuyCatalogRow;
  onMark: (body: {
    qty?: number;
    price?: number;
    pack?: boolean;
    missing?: boolean;
    clear?: boolean;
  }) => void;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [qty, setQty] = useState(line.gotQty ? String(line.gotQty) : "");
  const [price, setPrice] = useState(line.price ? String(line.price) : "");
  /** ⚠️ Off by default even where a packaging exists: the list asked in the
   *  store's unit, so the figure in front of the buyer is in that unit until
   *  they say otherwise. */
  const [inPacks, setInPacks] = useState(false);
  const hasPack = Boolean(pack?.packName && pack.packQty);

  const settled = Boolean(line.gotAt);
  return (
    <View
      style={[
        local.line,
        { borderColor: settled ? theme.accent : theme.line },
      ]}
    >
      <View style={local.lineHead}>
        <Text style={[s.body, { flex: 1 }]}>{line.name}</Text>
        <Text style={s.muted}>
          {t.buy.asked(line.qty, line.unit ?? "")}
        </Text>
      </View>

      {line.missing ? (
        <View style={local.lineHead}>
          <Text style={[s.muted, { flex: 1 }]}>{t.buy.wasMissing}</Text>
          <Pressable onPress={() => onMark({ clear: true })} hitSlop={10}>
            <Text style={[s.muted, { color: theme.accent }]}>{t.buy.undo}</Text>
          </Pressable>
        </View>
      ) : (
        <>
          <View style={local.fields}>
            <View style={{ flex: 1 }}>
              {hasPack ? (
                <Pressable onPress={() => setInPacks((v) => !v)}>
                  <Text style={[s.muted, { color: theme.accent }]}>
                    {t.buy.qty(inPacks ? pack!.packName! : line.unit ?? "")} ⇄
                  </Text>
                </Pressable>
              ) : (
                <Text style={s.muted}>{t.buy.qty(line.unit ?? "")}</Text>
              )}
              <TextInput
                style={[s.input, local.field]}
                keyboardType="decimal-pad"
                value={qty}
                placeholderTextColor={theme.muted}
                onChangeText={(v) => setQty(v.replace(",", "."))}
              />
            </View>
            <View style={{ flex: 1 }}>
              <Text style={s.muted}>{t.buy.price}</Text>
              <TextInput
                style={[s.input, local.field]}
                keyboardType="number-pad"
                value={price}
                placeholderTextColor={theme.muted}
                onChangeText={(v) => setPrice(v.replace(/\D/g, ""))}
              />
            </View>
          </View>
          <View style={local.fields}>
            <Tap
              onPress={() =>
                onMark({
                  qty: Number(qty) || 0,
                  price: Number(price) || 0,
                  pack: inPacks,
                })
              }
              disabled={!(Number(qty) > 0)}
              style={[local.mark, { backgroundColor: theme.accent }]}
            >
              <Text style={local.sendText}>
                {settled ? t.buy.changed : t.buy.got}
              </Text>
            </Tap>
            <Tap
              onPress={() => onMark({ missing: true })}
              style={[local.mark, { borderWidth: 1, borderColor: theme.line }]}
            >
              <Text style={s.muted}>{t.buy.noneLeft}</Text>
            </Tap>
          </View>
        </>
      )}
    </View>
  );
}

const local = StyleSheet.create({
  body: { padding: 16, paddingBottom: 40, gap: 8 },
  gap: { marginTop: 12 },
  done: { marginTop: 12, fontWeight: "600" },
  purse: { borderWidth: 1, borderRadius: 14, padding: 14, gap: 2 },
  purseValue: { fontSize: 26, fontWeight: "700" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  line: { borderWidth: 1, borderRadius: 12, padding: 12, gap: 8 },
  lineHead: { flexDirection: "row", alignItems: "center", gap: 8 },
  fields: { flexDirection: "row", gap: 10 },
  field: { marginTop: 4 },
  total: {
    flexDirection: "row",
    justifyContent: "space-between",
    borderTopWidth: 1,
    paddingTop: 12,
    marginTop: 8,
  },
  totalValue: { fontWeight: "700" },
  send: {
    marginTop: 12,
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: "center",
  },
  sendText: { color: "#fff", fontSize: 16, fontWeight: "700" },
  mark: { flex: 1, borderRadius: 12, paddingVertical: 12, alignItems: "center" },
});
