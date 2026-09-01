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
  TextInput,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api, ApiError } from "@/lib/api";
import { formatUzPhone } from "@/lib/format";
import { timeAgo } from "@/lib/orderFlow";
import type { Order } from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// The orders that are still in flight, and the two things an owner does about
// one from a phone.
//
// ⚠️ **Accept and cancel, and nothing else.** Everything the panel offers on an
// order — a courier, a discount, an address correction, a receipt — is done
// sitting down, with a keyboard, by somebody who is at work. What is done from
// a phone is what cannot wait for that: an order placed at eleven that nobody
// has accepted, and one that has to be stopped before the kitchen starts it.
//
// ⚠️ **A cancellation asks for a reason, and the server requires one anyway.**
// The guest reads it on their tracking page — it is the only sentence they get
// — so a cancel button that skipped it would be a button that sends "cancelled"
// with no explanation to somebody who has already paid.

export function OrdersScreen({ branchId }: { branchId: string }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [orders, setOrders] = useState<Order[] | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [cancelling, setCancelling] = useState<Order | null>(null);
  const [reason, setReason] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      // ⚠️ **Everything the list returns, filtered here to what is still in
      // flight.** The endpoint answers "the latest orders" and the panel draws
      // them all; on a phone a finished order is a row somebody has to read
      // past to find the one that needs them. Not "today's" either — an order
      // from last night that nobody closed is exactly the one to show.
      const rows = await api.adminOrders({ limit: 50 });
      setOrders(
        rows.filter(
          (o) => o.status !== "delivered" && o.status !== "cancelled",
        ),
      );
      setError("");
    } catch (e) {
      setOrders((o) => o ?? []);
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [t.common.loadFailed]);

  useEffect(() => {
    void load();
    const timer = setInterval(() => void load(), 20_000);
    return () => clearInterval(timer);
  }, [load, branchId]);

  async function accept(o: Order) {
    setBusyId(o.id);
    try {
      await api.updateOrderStatus(o.id, "confirmed");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.orders.failed);
    } finally {
      setBusyId(null);
    }
  }

  async function cancel() {
    if (!cancelling || reason.trim() === "") return;
    setBusyId(cancelling.id);
    try {
      await api.updateOrderStatus(cancelling.id, "cancelled", reason.trim());
      setCancelling(null);
      setReason("");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.orders.failed);
    } finally {
      setBusyId(null);
    }
  }

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Text style={s.h1}>{t.orders.title}</Text>
      </View>

      <FlatList
        data={orders ?? []}
        keyExtractor={(o) => o.id}
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
        ListHeaderComponent={
          error !== "" ? <Text style={s.error}>{error}</Text> : null
        }
        ListEmptyComponent={
          orders === null ? (
            <View style={{ paddingVertical: 40 }}>
              <ActivityIndicator color={theme.accent} />
            </View>
          ) : (
            <View style={s.empty}>
              <Feather name="inbox" size={26} color={theme.muted} />
              <Text style={[s.h2, { textAlign: "center" }]}>
                {t.orders.empty}
              </Text>
              <Text style={[s.muted, { textAlign: "center" }]}>
                {t.orders.emptyHint}
              </Text>
            </View>
          )
        }
        renderItem={({ item }) => {
          const pending = item.status === "pending";
          return (
            <View style={[s.card, { gap: 8 }]}>
              <View style={local.row}>
                <Text style={[s.h2, { flex: 1 }]}>#{item.number}</Text>
                {/* ⚠️ An unaccepted order wears the only colour on this list.
                    It is the one state that costs money by standing still. */}
                {pending && (
                  <View style={[local.chip, { backgroundColor: theme.warnSoft }]}>
                    <Text style={{ color: theme.warn, fontWeight: "700", fontSize: 12 }}>
                      {t.orders.confirm}
                    </Text>
                  </View>
                )}
                <Text style={s.money}>{money(item.total)}</Text>
              </View>

              <Text style={s.muted}>
                {[
                  item.customer?.name,
                  item.address?.text,
                  timeAgo(item.createdAt, t.common.timeAgo),
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </Text>

              <View style={local.actions}>
                {item.customer?.phone ? (
                  <Pressable
                    style={[s.ghost, local.action]}
                    onPress={() =>
                      void Linking.openURL(
                        `tel:+${item.customer.phone.replace(/\D/g, "")}`,
                      )
                    }
                  >
                    <Feather name="phone" size={16} color={theme.ink} />
                    <Text style={s.ghostText}>
                      {formatUzPhone(item.customer.phone)}
                    </Text>
                  </Pressable>
                ) : null}
                <Pressable
                  style={[s.ghost, local.action, { borderColor: theme.danger }]}
                  disabled={busyId === item.id}
                  onPress={() => {
                    setReason("");
                    setCancelling(item);
                  }}
                >
                  <Text style={[s.ghostText, { color: theme.danger }]}>
                    {t.orders.cancel}
                  </Text>
                </Pressable>
              </View>

              {pending && (
                <Pressable
                  style={[s.primary, busyId === item.id ? { opacity: 0.5 } : null]}
                  disabled={busyId === item.id}
                  onPress={() => void accept(item)}
                >
                  <Feather name="check" size={18} color={theme.onAccent} />
                  <Text style={s.primaryText}>{t.orders.confirm}</Text>
                </Pressable>
              )}
            </View>
          );
        }}
      />

      {cancelling && (
        <Modal transparent animationType="fade" onRequestClose={() => setCancelling(null)}>
          <Pressable style={local.backdrop} onPress={() => setCancelling(null)}>
            <Pressable
              style={[local.sheet, { backgroundColor: theme.surface }]}
              onPress={(e) => e.stopPropagation()}
            >
              <Text style={s.h2}>#{cancelling.number}</Text>
              {/* ⚠️ Asked before the button rather than refused after it: the
                  server requires a reason, and a dialog that fails on press has
                  already cost the tap. */}
              <TextInput
                style={s.input}
                value={reason}
                onChangeText={setReason}
                placeholder={t.orders.cancelReason}
                placeholderTextColor={theme.muted}
              />
              <Pressable
                style={[
                  s.primary,
                  { backgroundColor: theme.danger },
                  reason.trim() === "" ? { opacity: 0.5 } : null,
                ]}
                disabled={reason.trim() === "" || busyId !== null}
                onPress={() => void cancel()}
              >
                <Text style={s.primaryText}>{t.orders.cancelDo}</Text>
              </Pressable>
              <Pressable style={s.ghost} onPress={() => setCancelling(null)}>
                <Text style={s.ghostText}>{t.orders.cancelBack}</Text>
              </Pressable>
            </Pressable>
          </Pressable>
        </Modal>
      )}
    </View>
  );
}

const local = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 8 },
  chip: { borderRadius: 999, paddingHorizontal: 10, paddingVertical: 4 },
  actions: { flexDirection: "row", gap: 8 },
  action: { flex: 1, paddingVertical: 12 },
  backdrop: {
    flex: 1,
    backgroundColor: "rgba(0,0,0,0.5)",
    alignItems: "center",
    justifyContent: "center",
    padding: 24,
  },
  sheet: {
    width: "100%",
    maxWidth: 380,
    borderRadius: 22,
    padding: 20,
    gap: 12,
    alignItems: "center",
  },
});
