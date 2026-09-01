import { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Linking,
  Modal,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api, ApiError } from "@/lib/api";
import { formatPrice, formatUzPhone } from "@/lib/format";
import type { Courier, CourierStatus, Order } from "@/lib/types";

import { arrivalGate } from "./gate";
import { useNotice } from "./notice";
import { usePrefs } from "./prefs";
import { ShiftCard } from "./shift";
import type { Tracking } from "./tracking";
import { useUI } from "./ui";

// The courier's working screen.
//
// ⚠️ **One list, and the next step is a button on the card.** Everything a
// courier does in an evening is "picked it up" and "handed it over"; a screen
// with a detail view behind a tap would put a second press between a bike and
// the road, twice per order.

/** How often the list is re-asked. ⚠️ Not a socket: the phone is on mobile
 *  data, moving between cells, and a poll that fails is a poll that simply
 *  happens again in twenty seconds. A dropped socket needs reconnect logic
 *  nobody would be watching. */
const POLL_MS = 20000;

export function OrdersScreen({
  courier,
  tracking,
  onStatus,
  onRefreshCourier,
}: {
  courier: Courier;
  tracking: Tracking;
  onStatus: (s: CourierStatus) => Promise<void>;
  onRefreshCourier: () => Promise<void>;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const notice = useNotice();

  const [orders, setOrders] = useState<Order[] | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [routeFor, setRouteFor] = useState<Order | null>(null);
  // The branch's arrival radius, in metres. ⚠️ Asked from the server rather
  // than hard-coded: a city-centre restaurant may want 100 m and a village one
  // 500, and the number here has to be the number the server will judge by.
  const [radius, setRadius] = useState(0);

  const load = useCallback(async () => {
    try {
      setOrders(await api.courierOrders());
    } catch {
      // Keep whatever is on screen. A courier under a bridge should not have
      // their list emptied by a failed poll — that reads as "the order was
      // taken off me".
      setOrders((o) => o ?? []);
    }
    // The status can change without this phone doing anything: closing the last
    // order frees the courier server-side.
    await onRefreshCourier();
  }, [onRefreshCourier]);

  useEffect(() => {
    void api
      .getRestaurant()
      .then((r) => setRadius(r.restaurant.delivery.arrivalRadiusM ?? 0))
      .catch(() => setRadius(0));
  }, []);

  useEffect(() => {
    void load();
    const timer = setInterval(() => void load(), POLL_MS);
    return () => clearInterval(timer);
  }, [load]);

  async function advance(o: Order) {
    const next = o.status === "on_the_way" ? "delivered" : "on_the_way";
    setBusyId(o.id);
    try {
      await api.courierAdvanceOrder(o.id, next);
      await load();
    } catch (e) {
      // ⚠️ **The server's own sentence, in a sheet.** This is the one refusal
      // that arrives while a customer is watching, and it is usually the
      // arrival check on a position the server has not received yet — a
      // message the courier can act on ("wait, keep the app open") rather than
      // a red line under a button.
      notice({
        kind: "warn",
        title: t.orders.failed,
        body: e instanceof ApiError ? e.message : undefined,
      });
    } finally {
      setBusyId(null);
    }
  }

  const onShift = courier.status !== "off";
  const list = orders ?? [];

  return (
    <View style={s.screen}>
      <FlatList
        data={list}
        keyExtractor={(o) => o.id}
        contentContainerStyle={s.list}
        // ⚠️ FlatList rather than ScrollView, from the first version: an
        // evening's orders is a short list, and the day a restaurant assigns
        // forty to one courier is the day a ScrollView renders forty cards at
        // once on the cheapest phone in the fleet.
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
        ListHeaderComponent={
          <View style={{ gap: 14 }}>
            <View style={local.hello}>
              <View style={{ flex: 1 }}>
                <Text style={s.h1} numberOfLines={1}>
                  {courier.name}
                </Text>
                <Text style={s.muted}>{t.orders.title(list.length)}</Text>
              </View>
            </View>
            <ShiftCard
              status={courier.status}
              onStatus={(next) => void onStatus(next)}
              tracking={tracking}
            />
          </View>
        }
        ListEmptyComponent={
          orders === null ? (
            <View style={{ paddingVertical: 40 }}>
              <ActivityIndicator color={theme.accent} />
            </View>
          ) : (
            <View style={s.empty}>
              <Feather
                name={onShift ? "inbox" : "power"}
                size={26}
                color={theme.muted}
              />
              <Text style={[s.h2, { textAlign: "center" }]}>
                {onShift ? t.orders.empty : t.orders.offEmpty}
              </Text>
              <Text style={[s.muted, { textAlign: "center" }]}>
                {onShift ? t.orders.emptyHint : t.orders.offEmptyHint}
              </Text>
            </View>
          )
        }
        renderItem={({ item }) => (
          <OrderCard
            order={item}
            radius={radius}
            tracking={tracking}
            busy={busyId === item.id}
            onAdvance={() => void advance(item)}
            onRoute={() => setRouteFor(item)}
          />
        )}
      />

      {routeFor && (
        <RouteSheet order={routeFor} onClose={() => setRouteFor(null)} />
      )}
    </View>
  );
}

function OrderCard({
  order,
  radius,
  tracking,
  busy,
  onAdvance,
  onRoute,
}: {
  order: Order;
  radius: number;
  tracking: Tracking;
  busy: boolean;
  onAdvance: () => void;
  onRoute: () => void;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();

  const onTheWay = order.status === "on_the_way";
  const gate = onTheWay
    ? arrivalGate(order, tracking.fix, radius, t)
    : { applies: false, meters: null, reason: null };
  const blocked = gate.reason !== null;
  const atDoor = gate.applies && !blocked;

  // ⚠️ A stripe rather than a badge: the card is read at a glance from a
  // pocket, and the colour has to survive being half out of it.
  const stripe = onTheWay ? (atDoor ? theme.ok : theme.warn) : theme.accent;

  return (
    <View style={[s.card, { padding: 0, overflow: "hidden" }]}>
      <View style={[local.stripe, { backgroundColor: stripe }]} />
      <View style={{ padding: 14, gap: 10 }}>
        <View style={local.topRow}>
          <Text style={[s.h2, { flex: 1 }]}>#{order.number}</Text>
          {gate.meters !== null && (
            <View
              style={[
                local.distance,
                { backgroundColor: atDoor ? theme.okSoft : theme.warnSoft },
              ]}
            >
              <Feather
                name={atDoor ? "check" : "navigation"}
                size={12}
                color={atDoor ? theme.ok : theme.warn}
              />
              <Text
                style={[
                  local.distanceText,
                  { color: atDoor ? theme.ok : theme.warn },
                ]}
              >
                {atDoor ? t.orders.atDoor : t.orders.away(gate.meters)}
              </Text>
            </View>
          )}
          <Text style={s.money}>{formatPrice(order.total)}</Text>
        </View>

        {order.address?.text ? (
          <View style={local.line}>
            <Feather name="map-pin" size={14} color={theme.muted} />
            <Text style={[s.soft, { flex: 1 }]}>
              {order.address.text}
              {order.address.comment ? ` · ${order.address.comment}` : ""}
            </Text>
          </View>
        ) : null}

        <View style={local.line}>
          <Feather name="user" size={14} color={theme.muted} />
          <Text style={[s.soft, { flex: 1 }]}>{order.customer.name}</Text>
        </View>

        {/* What the courier owes the restaurant at the end of the evening, said
            on every card. ⚠️ Cash and card are the same screen otherwise, and
            handing back the wrong amount is discovered a day later. */}
        <View style={local.line}>
          <Feather
            name={order.paymentMethod === "cash" ? "dollar-sign" : "credit-card"}
            size={14}
            color={order.paymentMethod === "cash" ? theme.accent : theme.muted}
          />
          <Text
            style={[
              s.soft,
              order.paymentMethod === "cash"
                ? { color: theme.ink, fontWeight: "600" }
                : null,
            ]}
          >
            {order.paymentMethod === "cash"
              ? t.orders.cash(formatPrice(order.total))
              : t.orders.paid}
          </Text>
        </View>

        {order.items.some((i) => i.comment) && (
          <View style={[local.note, { backgroundColor: theme.warnSoft }]}>
            <Feather name="edit-3" size={13} color={theme.warn} />
            <Text style={[s.muted, { flex: 1, color: theme.warn }]}>
              {order.items
                .filter((i) => i.comment)
                .map((i) => `${i.name}: ${i.comment}`)
                .join(" · ")}
            </Text>
          </View>
        )}

        {/* ⚠️ **The reason lives above the button it is holding shut.** A
            disabled control with the explanation elsewhere on the screen is a
            control somebody presses repeatedly, in front of a customer, before
            going looking for the reason. */}
        {blocked && (
          <View style={[local.note, { backgroundColor: theme.warnSoft }]}>
            <Feather name="alert-circle" size={13} color={theme.warn} />
            <Text style={[s.muted, { flex: 1, color: theme.warn }]}>
              {gate.reason}
            </Text>
          </View>
        )}

        <View style={local.actions}>
          <Pressable
            style={[s.ghost, local.action]}
            onPress={() =>
              void Linking.openURL(
                `tel:+${order.customer.phone.replace(/\D/g, "")}`,
              )
            }
          >
            <Feather name="phone" size={16} color={theme.ink} />
            <Text style={s.ghostText}>{t.orders.call}</Text>
          </Pressable>

          {order.address?.lat ? (
            <Pressable style={[s.ghost, local.action]} onPress={onRoute}>
              <Feather name="navigation" size={16} color={theme.ink} />
              <Text style={s.ghostText}>{t.orders.route}</Text>
            </Pressable>
          ) : null}
        </View>

        <Pressable
          style={[
            s.primary,
            blocked || busy ? { opacity: 0.45 } : null,
            onTheWay && atDoor ? { backgroundColor: theme.ok } : null,
          ]}
          disabled={blocked || busy}
          onPress={onAdvance}
        >
          <Feather
            name={onTheWay ? "check-circle" : "package"}
            size={18}
            color={theme.onAccent}
          />
          <Text style={s.primaryText}>
            {busy ? "…" : onTheWay ? t.orders.deliver : t.orders.pickUp}
          </Text>
        </Pressable>
      </View>
    </View>
  );
}

/** Which navigator to open.
 *
 *  ⚠️ **A picker rather than one hard-coded app**, and the same three links the
 *  site gives a guest — pointed the other way. Couriers already have a
 *  navigator they trust, and the one that gets them there fastest is the one
 *  they know. Note the coordinate order: Yandex and Google take lat,lng and
 *  2GIS takes lng,lat, which is the mistake that puts a courier in the Aral
 *  Sea and looks like bad data.
 *
 *  The origin is left empty in every link on purpose: each app uses the phone's
 *  own position, which it knows better than we do. */
function RouteSheet({ order, onClose }: { order: Order; onClose: () => void }) {
  const { t } = usePrefs();
  const { theme, s, bottom } = useUI();
  const { lat, lng, text } = order.address;
  const ll = `${lat},${lng}`;

  const links: { label: string; url: string }[] = [
    { label: "Yandex", url: `https://yandex.uz/maps/?rtext=~${ll}&rtt=auto&z=16` },
    {
      label: "Google",
      url: `https://www.google.com/maps/dir/?api=1&destination=${ll}&travelmode=driving`,
    },
    { label: "2GIS", url: `https://2gis.uz/directions/points/%7C${lng}%2C${lat}` },
  ];

  return (
    <Modal transparent animationType="slide" onRequestClose={onClose}>
      <Pressable style={local.backdrop} onPress={onClose}>
        <Pressable
          style={[
            local.sheet,
            { backgroundColor: theme.surface, paddingBottom: bottom + 14 },
          ]}
          onPress={(e) => e.stopPropagation()}
        >
          <View style={local.grab} />
          <Text style={s.h2}>{t.orders.routeTitle}</Text>
          {text ? <Text style={s.muted}>{text}</Text> : null}
          {links.map((l) => (
            <Pressable
              key={l.label}
              style={[s.row, { marginTop: 2 }]}
              onPress={() => {
                void Linking.openURL(l.url);
                onClose();
              }}
            >
              <Feather name="navigation" size={18} color={theme.accent} />
              <Text style={[s.body, { flex: 1 }]}>{l.label}</Text>
              <Feather name="external-link" size={16} color={theme.muted} />
            </Pressable>
          ))}
          <Pressable style={[s.ghost, { marginTop: 4 }]} onPress={onClose}>
            <Text style={s.ghostText}>{t.common.close}</Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const local = StyleSheet.create({
  hello: { flexDirection: "row", alignItems: "center", gap: 12, paddingTop: 42 },
  stripe: { height: 4, width: "100%" },
  topRow: { flexDirection: "row", alignItems: "center", gap: 8 },
  line: { flexDirection: "row", alignItems: "flex-start", gap: 8 },
  distance: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    borderRadius: 999,
    paddingHorizontal: 9,
    paddingVertical: 4,
  },
  distanceText: { fontSize: 12, fontWeight: "700" },
  note: {
    flexDirection: "row",
    alignItems: "flex-start",
    gap: 8,
    borderRadius: 12,
    paddingHorizontal: 10,
    paddingVertical: 9,
  },
  actions: { flexDirection: "row", gap: 8 },
  action: { flex: 1, paddingVertical: 12 },
  backdrop: { flex: 1, backgroundColor: "rgba(0,0,0,0.5)", justifyContent: "flex-end" },
  sheet: {
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    padding: 20,
    gap: 10,
  },
  grab: {
    alignSelf: "center",
    width: 44,
    height: 4,
    borderRadius: 2,
    backgroundColor: "rgba(128,128,128,0.35)",
    marginBottom: 6,
  },
});
