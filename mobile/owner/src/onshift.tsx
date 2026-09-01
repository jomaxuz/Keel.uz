import { useCallback, useEffect, useState } from "react";
import {
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api } from "@/lib/api";
import type { StaffRow } from "@/lib/types";

import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// Who is actually in the building.
//
// ⚠️ **A "now" question, which is why it sits on the first screen and not in
// the reports.** Hours worked, overtime and what is owed are a month's
// arithmetic done at a desk; "is the second cook in yet" is asked in a car at
// half past ten, and until now the only way to answer it was to ring somebody
// and ask — which tells the person being checked that they were checked.
//
// ⚠️ **One line normally, the list only on a tap.** Six names on the screen an
// owner opens for the takings is six names in the way. The line carries the two
// numbers that decide whether the list is worth opening: how many are in, and
// how many were expected and are not.

/** Today, as the day boundary every report in this product uses. */
function today(): { from: string; to: string } {
  const d = new Date();
  const day = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate(),
  ).padStart(2, "0")}`;
  return { from: day, to: day };
}

export function OnShiftCard({ branchId }: { branchId: string }) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [rows, setRows] = useState<StaffRow[] | null>(null);
  const [open, setOpen] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await api.adminStaff(today());
      setRows(res ?? []);
    } catch {
      // ⚠️ Silent, like the briefing beside it: this is an addition to the
      // screen, and an error banner over working takings is a worse morning
      // than a missing line nobody was promised.
      setRows(null);
    }
  }, []);

  useEffect(() => {
    void load();
    // Polled slowly. People clock in and out a handful of times a day, and this
    // sits under a number that is already refreshing.
    const timer = setInterval(() => void load(), 120_000);
    return () => clearInterval(timer);
  }, [load, branchId]);

  if (rows === null || rows.length === 0) return null;

  const onShift = rows.filter((r) => r.onShift);
  // ⚠️ **"Absent" is the server's word for it, not this app's arithmetic.**
  // Whether somebody was expected today is a question about the roster, days
  // off and the shift that started yesterday and ended at two — and a phone
  // that recomputed it would be the second definition that drifts.
  const absent = rows.filter((r) => r.todayStatus === "absent");

  return (
    <>
      <Pressable style={s.row} onPress={() => setOpen(true)}>
        <Feather name="users" size={18} color={theme.muted} />
        <Text style={[s.body, { flex: 1 }]}>{t.who.now(onShift.length)}</Text>
        {absent.length > 0 && (
          <Text style={[s.muted, { color: theme.warn }]}>
            {t.who.absent(absent.length)}
          </Text>
        )}
        <Feather name="chevron-right" size={18} color={theme.muted} />
      </Pressable>

      {open && (
        <Modal transparent animationType="fade" onRequestClose={() => setOpen(false)}>
          <Pressable style={local.backdrop} onPress={() => setOpen(false)}>
            <Pressable
              style={[local.sheet, { backgroundColor: theme.surface }]}
              onPress={(e) => e.stopPropagation()}
            >
              <Text style={s.h2}>{t.who.title}</Text>

              {onShift.length === 0 && (
                <Text style={s.muted}>{t.who.nobody}</Text>
              )}

              <ScrollView style={{ maxHeight: 380 }}>
                {/* ⚠️ On shift first, then everybody else. The list is read
                    top-down for one purpose — who is here — and sorting it by
                    name would put the person who did not come in above the
                    four who did. */}
                {[...onShift, ...rows.filter((r) => !r.onShift)].map((r) => (
                  <View key={r.id} style={local.line}>
                    <View
                      style={[
                        local.dot,
                        {
                          backgroundColor: r.onShift
                            ? theme.ok
                            : r.todayStatus === "absent"
                              ? theme.warn
                              : theme.line,
                        },
                      ]}
                    />
                    <View style={{ flex: 1 }}>
                      <Text style={s.body}>{r.name}</Text>
                      {r.position ? (
                        <Text style={s.muted}>{r.position}</Text>
                      ) : null}
                    </View>
                    <Text style={s.muted}>{when(r, t)}</Text>
                  </View>
                ))}
              </ScrollView>

              <Pressable style={s.ghost} onPress={() => setOpen(false)}>
                <Text style={s.ghostText}>{t.common.close}</Text>
              </Pressable>
            </Pressable>
          </Pressable>
        </Modal>
      )}
    </>
  );
}

/** What this person's day looks like so far, in one short phrase.
 *
 *  ⚠️ **The times come from the server as "HH:MM" strings**, already in the
 *  restaurant's own hours. Anything parsed from a timestamp here would be in
 *  the phone's timezone — which is the trap this codebase has written up twice,
 *  and it is silent: the shift simply reads five hours early. */
function when(r: StaffRow, t: ReturnType<typeof usePrefs>["t"]): string {
  if (r.onShift) return r.todayIn ? t.who.since(r.todayIn) : "";
  if (r.todayIn && r.todayOut) return t.who.done(r.todayIn, r.todayOut);
  if (r.todayStatus === "absent") return t.who.notYet;
  if (r.todayStatus === "off") return t.who.off;
  return "";
}

const local = StyleSheet.create({
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
    padding: 18,
    gap: 12,
  },
  line: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    paddingVertical: 9,
  },
  dot: { width: 8, height: 8, borderRadius: 4 },
});
