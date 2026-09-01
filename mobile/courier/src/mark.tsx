import { View } from "react-native";

// The Keel mark, drawn out of two views.
//
// ⚠️ **No SVG, and that is a size decision rather than a stylistic one.**
// `react-native-svg` is a native dependency: a new build, a new thing that can
// be the wrong version, and about a megabyte — for one shape that appears on
// exactly two screens. The mark is a hull and a fin, and a bordered box with
// two rounded bottom corners *is* a hull.
//
// ⚠️ **The fin stays long.** It is the part that reads at 24 points; shortened,
// the mark becomes an anonymous curve, which is the one thing `logos/README.txt`
// asks not to do to it.

export function KeelMark({
  size,
  colour,
}: {
  /** Width of the hull. The whole mark is about 1.3× this tall. */
  size: number;
  colour: string;
}) {
  const stroke = Math.max(2, Math.round(size * 0.12));
  return (
    <View style={{ alignItems: "center" }}>
      <View
        style={{
          width: size,
          // The source curve runs from y=6 to y=20.5 across a width of 22 —
          // two thirds, not a half circle.
          height: size * 0.66,
          borderColor: colour,
          borderWidth: stroke,
          borderTopWidth: 0,
          borderBottomLeftRadius: size / 2,
          borderBottomRightRadius: size / 2,
        }}
      />
      <View
        style={{
          width: stroke,
          height: size * 0.38,
          backgroundColor: colour,
          borderRadius: stroke / 2,
          // Pulled up by half a stroke so the fin grows out of the hull rather
          // than hanging under it with a seam between them.
          marginTop: -stroke / 2,
        }}
      />
    </View>
  );
}
