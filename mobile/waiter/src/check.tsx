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
import type { Check, CheckLine, MenuGroup, MenuItem } from "@/lib/types";

import { useSafeAreaInsets } from "react-native-safe-area-context";

import { LineDialog } from "./line";
import { TableActions } from "./table";
import { Chip, MenuList } from "./menu";
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
  const insets = useSafeAreaInsets();
  const [editing, setEditing] = useState<CheckLine | null>(null);
  const [note, setNote] = useState("");
  const [job, setJob] = useState<
    "menu" | "guests" | "split" | "merge" | "move" | null
  >(null);
  const [others, setOthers] = useState<Check[]>([]);

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

  // ⚠️ Loaded once with the check rather than when a sheet opens: the wait
  // belongs to arriving at the table, not to the moment somebody has decided
  // to move a dish and is standing between two guests.
  useEffect(() => {
    void api
      .tillChecks()
      .then((r) => setOthers(r.checks.filter((c) => c.id !== checkId)))
      .catch(() => setOthers([]));
  }, [checkId]);

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

  /** How many of each dish are already on this check.
   *
   *  ⚠️ **The menu has to say what has already been added.** Tapping a tile
   *  four times is how a waiter enters four coffees, and with nothing counting
   *  back at them the only way to know whether the third tap registered is to
   *  switch tabs and read the check. That is the moment somebody taps again to
   *  be sure — and the guest is charged for five.
   *
   *  Summed across lines rather than read off one: the server merges a repeat
   *  into an unfired line but starts a new one once the kitchen has seen it, so
   *  two of the same dish can legitimately be two rows. */
  const onCheck = useMemo(() => {
    const m = new Map<string, number>();
    for (const l of lines) {
      if (!l.menuItemId) continue;
      m.set(l.menuItemId, (m.get(l.menuItemId) ?? 0) + l.qty);
    }
    return m;
  }, [lines]);

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

  /** Print the bill for this table.
   *
   *  ⚠️ **Only once everything has been sent.** A bill printed while a dish is
   *  still unfired is a bill that is about to be wrong — and the guest has
   *  already been handed it. The button is hidden rather than disabled: a
   *  greyed-out control invites "why can't I", and the answer is one line up
   *  on the same screen.
   *
   *  ⚠️ **`queued: 0` is not an error and is said plainly.** It means no printer
   *  in the branch took the job, which is how every restaurant's first evening
   *  goes — and the honest next step is the till, not a retry. */
  async function printBill() {
    setBusy(true);
    setNote("");
    try {
      const res = await api.tillPrint(checkId, "precheck");
      setCheck(res.check);
      setNote(t.bill.printed(res.queued));
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.bill.failed);
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
  const pad = unfired > 0 ? 96 + insets.bottom : 24 + insets.bottom;

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
        <View style={local.headRight}>
          <Text style={[s.h2, { fontVariant: ["tabular-nums"] }]}>
            {money(check.total)}
          </Text>
          {/* ⚠️ Behind one button rather than four in the header: these are
              things a table does occasionally, and four controls above the
              check would crowd out the two it does constantly. */}
          <Pressable onPress={() => setJob("menu")} hitSlop={10}>
            <Feather name="more-vertical" size={20} color={theme.muted} />
          </Pressable>
        </View>
      </View>

      {job !== null && (
        <TableActions
          check={check}
          others={others}
          job={job}
          onJob={setJob}
          onDone={setCheck}
          onClose={() => setJob(null)}
        />
      )}

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
        // ⚠️ Room at the foot for the send button, which floats over this. The
        // last dish added is the one somebody is looking for, and it is exactly
        // the one that would sit under the button.
        <ScrollView contentContainerStyle={[s.list, { paddingBottom: pad }]}>
          {lines.map((l) => (
            // ⚠️ The whole row opens the dialog rather than a small edit icon:
            // this is used with a thumb, walking, and a target the size of a
            // glyph is the reason somebody gives up and walks to the till.
            <Pressable
              key={l.lineId}
              style={s.row}
              onPress={() => setEditing(l)}
            >
              <View style={{ flex: 1 }}>
                <Text style={s.body}>
                  {/* ⚠️ The line's own frozen name, not the menu's: it is what
                      the guest agreed to, and a dish renamed at six o'clock
                      must not rewrite a check opened at five. */}
                  {l.name}
                  {l.qty > 1 ? ` × ${l.qty}` : ""}
                </Text>
                {/* ⚠️ An unfired line says so. The kitchen has not seen it, and
                    that is the one thing here a guest may be waiting on. */}
                {l.comment ? (
                  // What the kitchen was told about this dish. Shown because it
                  // is the half of the order a guest will check.
                  <Text style={s.muted}>{l.comment}</Text>
                ) : null}
                {!l.fired && (
                  <Text style={[s.muted, { color: theme.accent }]}>
                    {t.check.pending}
                  </Text>
                )}
              </View>
              <Text style={s.num}>{money(l.sum)}</Text>
              <Feather name="chevron-right" size={16} color={theme.muted} />
            </Pressable>
          ))}
          {lines.length === 0 && (
            <Text style={[s.muted, local.empty]}>{t.check.empty}</Text>
          )}
        </ScrollView>
      ) : (
        <MenuList
          groups={groups ?? []}
          onCheck={onCheck}
          busy={busy}
          onAdd={add}
          footer={pad}
        />
      )}

      {editing && (
        <LineDialog
          checkId={checkId}
          line={editing}
          onDone={setCheck}
          onClose={() => setEditing(null)}
        />
      )}

      {/* ⚠️ **The bill, and it belongs on the check tab only.** Asking for it is
          the end of the waiter's job — the table says "hisob" and somebody goes
          to fetch it. Until now that meant walking to the till, which is the
          walk this whole app exists to remove. Hidden while the menu is open:
          nobody prints a bill in the middle of taking an order. */}
      {tab === "check" && lines.length > 0 && unfired === 0 && (
        <Pressable
          style={[
            s.row,
            local.bill,
            { marginBottom: Math.max(insets.bottom, 12) + 4 },
          ]}
          disabled={busy}
          onPress={() => void printBill()}
        >
          <Feather name="printer" size={18} color={theme.ink} />
          <Text style={[s.body, { flex: 1 }]}>{t.bill.print}</Text>
          <Text style={s.num}>{money(check.total)}</Text>
        </Pressable>
      )}

      {note !== "" && (
        <Text style={[s.muted, { textAlign: "center", paddingBottom: 8 }]}>
          {note}
        </Text>
      )}

      {/* ⚠️ Shown only when there is something the kitchen has not seen. A
          permanent button invites being pressed on a table that is already
          cooking, and the second press is a ticket nobody asked for. */}
      {unfired > 0 && (
        // ⚠️ **Above Android's navigation bar, not under it.** With three-button
        // navigation the bar sits at the bottom of the screen and a button
        // placed at the bottom of the *layout* lands beneath it — so the tap
        // that was meant to send an order to the kitchen presses Back instead,
        // and the waiter is returned to the room with the food unsent. The
        // inset is the system's own measurement of that strip; gesture
        // navigation reports a smaller one, and a fixed margin would be wrong
        // on one of the two.
        <Pressable
          style={[
            s.primary,
            local.fire,
            { marginBottom: Math.max(insets.bottom, 12) + 4 },
          ]}
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

const local = StyleSheet.create({
  back: { flexDirection: "row", alignItems: "center", gap: 2 },
  tabs: { flexDirection: "row", gap: 8, paddingHorizontal: 16 },
  empty: { textAlign: "center", padding: 20 },
  fire: { marginHorizontal: 16, marginTop: 4 },
  bill: { marginHorizontal: 16, marginTop: 4 },
  headRight: { flexDirection: "row", alignItems: "center", gap: 12 },
});
