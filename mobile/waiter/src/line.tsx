import { useState } from "react";
import { Modal, Pressable, StyleSheet, Text, TextInput, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { api, ApiError } from "@/lib/api";
import type { Check, CheckLine } from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// Correcting a line that has already been added.
//
// ⚠️ **Without this the app can make a mistake it cannot fix.** Adding a dish is
// one tap, so mis-taps are common — and until now the only way back was to walk
// to the till. An app that creates errors and sends somebody else to correct
// them adds work to a shift rather than removing it.
//
// ⚠️ **Fired and unfired are two different acts, and the screen says which.**
// Before the kitchen has seen a line, removing it is a typo being corrected and
// costs nothing. Afterwards the food has been cooked, somebody paid for it in
// ingredients, and taking it off the bill is a write-off — the server asks for
// a reason and may ask for a manager's code. That is the oldest way money
// leaves a restaurant, and the difference is not ours to blur.

export function LineDialog({
  checkId,
  line,
  onDone,
  onClose,
}: {
  checkId: string;
  line: CheckLine;
  onDone: (c: Check) => void;
  onClose: () => void;
}) {
  const { t } = usePrefs();
  const { theme, s, bottom } = useUI();
  const [qty, setQty] = useState(line.qty);
  const [comment, setComment] = useState(line.comment ?? "");
  const [reason, setReason] = useState("");
  const [pin, setPin] = useState("");
  const [needPin, setNeedPin] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const fired = !!line.fired;

  async function run(what: () => Promise<Check>) {
    setBusy(true);
    setError("");
    try {
      onDone(await what());
      onClose();
    } catch (e) {
      const err = e as ApiError;
      // ⚠️ **A refusal that asks for a manager is not a failure.** The server
      // decides who may write off cooked food — the permission may have changed
      // since this person signed in — and the answer is a code, not an error
      // message. Same shape the till's override dialog takes.
      if (err.status === 403) setNeedPin(true);
      setError(err instanceof ApiError ? err.message : t.line.failed);
      setBusy(false);
    }
  }

  return (
    <Modal transparent animationType="fade" onRequestClose={onClose}>
      <Pressable style={local.backdrop} onPress={onClose}>
        <Pressable
          // ⚠️ The sheet's own floor was a constant that cleared a gesture
          // pill and not a three-button bar; the system's measurement clears
          // both. See `useBottomInset`.
          style={[
            local.sheet,
            { backgroundColor: theme.surface, paddingBottom: bottom + 14 },
          ]}
          onPress={(e) => e.stopPropagation()}
        >
          <Text style={s.h2}>{line.name}</Text>
          <Text style={s.muted}>
            {money(line.price)} · {fired ? t.line.fired : t.check.pending}
          </Text>

          {/* ⚠️ Quantity only while the kitchen has not seen it. After that the
              ticket at the pass names a number, and changing it here would
              leave the paper and the screen disagreeing about one dish. */}
          {!fired && (
            <View style={local.stepper}>
              <Pressable
                style={[local.step, { borderColor: theme.line }]}
                disabled={busy || qty <= 1}
                onPress={() => setQty((n) => Math.max(1, n - 1))}
              >
                <Feather name="minus" size={20} color={qty <= 1 ? theme.muted : theme.ink} />
              </Pressable>
              <Text style={[s.h1, { minWidth: 56, textAlign: "center" }]}>{qty}</Text>
              <Pressable
                style={[local.step, { borderColor: theme.line }]}
                disabled={busy || qty >= 99}
                onPress={() => setQty((n) => Math.min(99, n + 1))}
              >
                <Feather name="plus" size={20} color={theme.ink} />
              </Pressable>
            </View>
          )}

          {!fired && (
            <TextInput
              style={s.input}
              value={comment}
              onChangeText={setComment}
              placeholder={t.line.commentPlaceholder}
              placeholderTextColor={theme.muted}
            />
          )}

          {/* ⚠️ The reason is asked **before** the button, not after it: a
              dialog that refuses on press has already cost the tap, and the
              server requires one on a fired line. */}
          {fired && (
            <TextInput
              style={s.input}
              value={reason}
              onChangeText={setReason}
              placeholder={t.line.reasonPlaceholder}
              placeholderTextColor={theme.muted}
            />
          )}

          {needPin && (
            <TextInput
              style={s.input}
              value={pin}
              onChangeText={setPin}
              placeholder={t.line.pinPlaceholder}
              placeholderTextColor={theme.muted}
              keyboardType="number-pad"
              secureTextEntry
            />
          )}

          {error !== "" && <Text style={s.error}>{error}</Text>}

          <View style={local.actions}>
            <Pressable
              style={[local.ghost, { borderColor: theme.line }]}
              disabled={busy}
              onPress={() =>
                void run(async () => {
                  let c = await api.tillVoidLine(checkId, line.lineId, {
                    ...(fired ? { reason: reason.trim() } : {}),
                    ...(pin ? { pin } : {}),
                  });
                  return c;
                })
              }
            >
              <Feather name="trash-2" size={17} color={theme.danger} />
              <Text style={{ color: theme.danger, fontSize: 15 }}>
                {fired ? t.line.writeOff : t.line.remove}
              </Text>
            </Pressable>

            {!fired && (
              <Pressable
                style={[s.primary, { flex: 1 }]}
                disabled={busy}
                onPress={() =>
                  void run(async () => {
                    let c: Check | null = null;
                    if (qty !== line.qty) {
                      c = await api.tillLineQty(checkId, line.lineId, qty);
                    }
                    if ((comment.trim() || "") !== (line.comment ?? "")) {
                      c = await api.tillCommentLine(
                        checkId,
                        line.lineId,
                        comment.trim(),
                      );
                    }
                    // ⚠️ Nothing changed is not an error and not a request: the
                    // dialog is also how somebody looks at a line.
                    return c ?? (await api.tillCheck(checkId));
                  })
                }
              >
                <Text style={s.primaryText}>{t.line.save}</Text>
              </Pressable>
            )}
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const local = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: "rgba(0,0,0,0.45)",
    justifyContent: "flex-end",
  },
  sheet: {
    padding: 20,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    gap: 12,
  },
  stepper: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: 16 },
  step: {
    width: 52,
    height: 52,
    borderRadius: 16,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  actions: { flexDirection: "row", gap: 10, alignItems: "center" },
  ghost: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    borderWidth: 1,
    borderRadius: 14,
    paddingHorizontal: 16,
    paddingVertical: 15,
  },
});
