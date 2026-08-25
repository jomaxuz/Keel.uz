import { Pressable, StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { useUI } from "./ui";

// Minus, a number, plus.
//
// ⚠️ **One component, because there are two of them and they must agree.** The
// count beside a dish in the menu and the quantity on the check are the same
// fact seen twice; if they were built separately they would drift in size and
// in behaviour, and a waiter would learn one of them and be surprised by the
// other.
//
// ⚠️ **44px targets.** This is pressed with a thumb, walking, often with
// something in the other hand — and the two buttons sit next to each other, so
// a small target is not merely hard to hit, it is easy to hit *the wrong one*.

export function Stepper({
  value,
  onMinus,
  onPlus,
  disabled,
  /** Whether pressing minus at one removes the dish rather than refusing. */
  removeAtZero = false,
}: {
  value: number;
  onMinus: () => void;
  onPlus: () => void;
  disabled?: boolean;
  removeAtZero?: boolean;
}) {
  const { theme } = useUI();
  const atFloor = value <= 1 && !removeAtZero;
  return (
    <View style={local.row}>
      <Pressable
        style={[local.btn, { borderColor: theme.line }]}
        disabled={disabled || atFloor}
        hitSlop={6}
        onPress={onMinus}
      >
        {/* ⚠️ The bin rather than a minus at one, when minus means removal. A
            minus that deletes is the same glyph doing two different things, and
            the second one cannot be undone from this screen. */}
        <Feather
          name={removeAtZero && value <= 1 ? "trash-2" : "minus"}
          size={17}
          color={
            atFloor
              ? theme.muted
              : removeAtZero && value <= 1
                ? theme.danger
                : theme.ink
          }
        />
      </Pressable>
      <Text
        style={[
          local.value,
          { color: theme.ink, backgroundColor: theme.accentSoft },
        ]}
      >
        {value}
      </Text>
      <Pressable
        style={[local.btn, { borderColor: theme.line }]}
        disabled={disabled || value >= 99}
        hitSlop={6}
        onPress={onPlus}
      >
        <Feather name="plus" size={17} color={theme.accent} />
      </Pressable>
    </View>
  );
}

const local = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 6 },
  btn: {
    width: 40,
    height: 40,
    borderRadius: 12,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  value: {
    minWidth: 34,
    textAlign: "center",
    fontSize: 15,
    fontWeight: "700",
    paddingVertical: 8,
    borderRadius: 10,
    overflow: "hidden",
  },
});
