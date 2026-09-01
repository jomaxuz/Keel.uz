import { StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { Tap } from "./press";
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
  /** ⚠️ **A card is half a screen wide and the full stepper does not fit.** It
   *  overflowed to the right and left the count and the plus off the edge — so
   *  the control that says how many were added was the part that disappeared.
   *  Compact is the same control at the size the space allows, not a different
   *  one: same order, same glyphs, same behaviour. */
  compact = false,
}: {
  value: number;
  onMinus: () => void;
  onPlus: () => void;
  disabled?: boolean;
  removeAtZero?: boolean;
  compact?: boolean;
}) {
  const { theme } = useUI();
  const atFloor = value <= 1 && !removeAtZero;
  const btn = compact ? local.btnSmall : local.btn;
  const size = compact ? 15 : 17;
  return (
    <View style={compact ? local.rowSmall : local.row}>
      <Tap
        style={[btn, { borderColor: theme.line }]}
        disabled={disabled || atFloor}
        hitSlop={6}
        onPress={onMinus}
      >
        {/* ⚠️ The bin rather than a minus at one, when minus means removal. A
            minus that deletes is the same glyph doing two different things, and
            the second one cannot be undone from this screen. */}
        <Feather
          name={removeAtZero && value <= 1 ? "trash-2" : "minus"}
          size={size}
          color={
            atFloor
              ? theme.muted
              : removeAtZero && value <= 1
                ? theme.danger
                : theme.ink
          }
        />
      </Tap>
      <Text
        style={[
          compact ? local.valueSmall : local.value,
          { color: theme.ink, backgroundColor: theme.accentSoft },
        ]}
      >
        {value}
      </Text>
      <Tap
        style={[btn, { borderColor: theme.line }]}
        disabled={disabled || value >= 99}
        // ⚠️ A larger slop on the compact one: the button shrinks, the thumb
        // does not.
        hitSlop={compact ? 10 : 6}
        onPress={onPlus}
      >
        <Feather name="plus" size={size} color={theme.accent} />
      </Tap>
    </View>
  );
}

const local = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: 6 },
  rowSmall: { flexDirection: "row", alignItems: "center", gap: 3 },
  btn: {
    width: 40,
    height: 40,
    borderRadius: 12,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  btnSmall: {
    width: 30,
    height: 30,
    borderRadius: 9,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  valueSmall: {
    minWidth: 24,
    textAlign: "center",
    fontSize: 13,
    fontWeight: "700",
    paddingVertical: 6,
    borderRadius: 8,
    overflow: "hidden",
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
