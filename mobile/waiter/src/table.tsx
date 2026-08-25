import { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api, ApiError } from "@/lib/api";
import type { Check } from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// The three things a table does that a single check cannot express.
//
// ⚠️ **All three exist because a party is not a bill.** Guests arrive at one
// table and pay in two; two tables are pushed together; somebody sits down at
// the wrong one and the food follows them. A till that can only open and close
// a check sends every one of those to the counter — which is where they used to
// go, and the walk this removes.

/** ⚠️ `menu` is a real state, not the absence of one: the sheet opens on a
 *  choice of four occasional actions, and going straight to any of them would
 *  make three of them unreachable. */
type Job = "menu" | "guests" | "split" | "merge" | "move";

export function TableActions({
  check,
  others,
  onDone,
  onClose,
  job,
  onJob,
}: {
  check: Check;
  /** The branch's other open checks — the only possible destinations. */
  others: Check[];
  onDone: (c: Check) => void;
  onClose: () => void;
  job: Job;
  onJob: (j: Job) => void;
}) {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [picked, setPicked] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const lines = (check.lines ?? []).filter((l) => !l.void);

  async function run(what: () => Promise<Check>) {
    setBusy(true);
    setError("");
    try {
      onDone(await what());
      onClose();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.table.failed);
      setBusy(false);
    }
  }

  const toggle = (id: string) =>
    setPicked((p) => (p.includes(id) ? p.filter((x) => x !== id) : [...p, id]));

  return (
    <Modal transparent animationType="slide" onRequestClose={onClose}>
      <View style={local.backdrop}>
        <View style={[local.sheet, { backgroundColor: theme.surface }]}>
          <View style={local.head}>
            <Text style={s.h2}>{job === "menu" ? t.table.actions : t.table[job]}</Text>
            <Pressable onPress={onClose} hitSlop={12}>
              <Feather name="x" size={20} color={theme.muted} />
            </Pressable>
          </View>

          {job === "menu" && (
            <View style={{ gap: 8 }}>
              {(["guests", "split", "move", "merge"] as const).map((k) => (
                <Pressable key={k} style={s.row} onPress={() => onJob(k)}>
                  <Feather
                    name={
                      k === "guests"
                        ? "users"
                        : k === "split"
                          ? "scissors"
                          : k === "move"
                            ? "corner-up-right"
                            : "git-merge"
                    }
                    size={18}
                    color={theme.accent}
                  />
                  <Text style={[s.body, { flex: 1 }]}>{t.table[k]}</Text>
                  <Feather name="chevron-right" size={16} color={theme.muted} />
                </Pressable>
              ))}
            </View>
          )}

          {job === "guests" && (
            <>
              <Text style={s.muted}>{t.table.guestsHint}</Text>
              <View style={local.numbers}>
                {[1, 2, 3, 4, 5, 6, 8, 10, 12].map((n) => (
                  <Pressable
                    key={n}
                    style={[
                      local.num,
                      {
                        borderColor: check.guests === n ? theme.accent : theme.line,
                        backgroundColor:
                          check.guests === n ? theme.accentSoft : "transparent",
                      },
                    ]}
                    disabled={busy}
                    onPress={() =>
                      void run(() => api.tillUpdateCheck(check.id, { guests: n }))
                    }
                  >
                    <Text style={s.body}>{n}</Text>
                  </Pressable>
                ))}
              </View>
            </>
          )}

          {(job === "split" || job === "move") && (
            <>
              {/* ⚠️ **Lines are picked, never "half".** A guest pays for what
                  they ate; splitting a total down the middle is a different
                  thing that reads the same on a screen and is wrong at the
                  table. */}
              <Text style={s.muted}>
                {job === "split" ? t.table.splitHint : t.table.moveHint}
              </Text>
              <ScrollView style={{ maxHeight: 260 }} contentContainerStyle={{ gap: 8 }}>
                {lines.map((l) => (
                  <Pressable
                    key={l.lineId}
                    style={[
                      s.row,
                      picked.includes(l.lineId)
                        ? { borderColor: theme.accent, backgroundColor: theme.accentSoft }
                        : null,
                    ]}
                    onPress={() => toggle(l.lineId)}
                  >
                    <Feather
                      name={picked.includes(l.lineId) ? "check-square" : "square"}
                      size={18}
                      color={picked.includes(l.lineId) ? theme.accent : theme.muted}
                    />
                    <Text style={[s.body, { flex: 1 }]}>
                      {l.name}
                      {l.qty > 1 ? ` × ${l.qty}` : ""}
                    </Text>
                    <Text style={s.num}>{money(l.sum)}</Text>
                  </Pressable>
                ))}
              </ScrollView>
            </>
          )}

          {(job === "merge" || job === "move") && (
            <>
              <Text style={s.muted}>{t.table.pickTable}</Text>
              <ScrollView style={{ maxHeight: 220 }} contentContainerStyle={{ gap: 8 }}>
                {others.map((o) => (
                  <Pressable
                    key={o.id}
                    style={s.row}
                    disabled={busy || (job === "move" && picked.length === 0)}
                    onPress={() =>
                      void run(async () =>
                        job === "merge"
                          ? await api.tillMerge(check.id, o.id)
                          : await api.tillMoveLines(check.id, picked, o.id),
                      )
                    }
                  >
                    <Feather name="corner-up-right" size={17} color={theme.accent} />
                    <Text style={[s.body, { flex: 1 }]}>
                      {o.tableNumber ? t.check.table(o.tableNumber) : o.number}
                    </Text>
                    <Text style={s.num}>{money(o.total)}</Text>
                  </Pressable>
                ))}
                {others.length === 0 && (
                  // ⚠️ Said rather than shown as an empty list: there being no
                  // other open table is the ordinary state, not a fault.
                  <Text style={[s.muted, { textAlign: "center", padding: 12 }]}>
                    {t.table.noOthers}
                  </Text>
                )}
              </ScrollView>
            </>
          )}

          {error !== "" && <Text style={s.error}>{error}</Text>}

          {job === "split" && (
            <Pressable
              style={s.primary}
              disabled={busy || picked.length === 0}
              onPress={() =>
                void run(async () => (await api.tillSplit(check.id, picked)).check)
              }
            >
              <Text style={s.primaryText}>
                {t.table.splitDo(picked.length)}
              </Text>
            </Pressable>
          )}
        </View>
      </View>
    </Modal>
  );
}

const local = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: "rgba(0,0,0,0.45)", justifyContent: "flex-end" },
  sheet: {
    padding: 20,
    paddingBottom: 34,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    gap: 12,
  },
  head: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  numbers: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  num: {
    width: 56,
    height: 56,
    borderRadius: 16,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
});
