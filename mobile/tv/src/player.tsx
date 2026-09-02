// The loop on the wall.
//
// ⚠️ **Nothing here is ever interactive.** No controls, no progress bar, no tap
// target: a television in a dining room is driven by nobody, and the remote is
// in a drawer. What this component does is decide which item is on screen and
// when to move on — everything else is the file itself, drawn edge to edge.
//
// ⚠️ **Black behind everything, and `contain` rather than `cover`.** A picture
// cropped to fill a 16:9 wall loses whichever edge the price was on, and the
// restaurant will not find out — nobody looks at the screen from the office.
// Letterboxing on black is invisible on a dark set and never eats content.

import { useEventListener } from "expo";
import { VideoView, useVideoPlayer } from "expo-video";
import { useEffect, useState } from "react";
import { Image, StyleSheet, View } from "react-native";

import type { PlayItem } from "./playlist";

export function PlayerScreen({ items }: { items: PlayItem[] }) {
  const [index, setIndex] = useState(0);

  // ⚠️ **The index is kept inside the list as the list changes.** A playlist
  // edited in the panel while the screen is on item five, down to three items,
  // would otherwise leave the television pointing past the end — a black
  // rectangle that nothing in the app would ever move off.
  const safe = items.length === 0 ? 0 : index % items.length;
  const current: PlayItem | undefined = items[safe];

  useEffect(() => {
    if (index !== safe) setIndex(safe);
  }, [index, safe]);

  // One player for the whole loop, its source replaced as videos come round.
  // ⚠️ Not one per item: creating a player per slide leaks a decoder on cheap
  // hardware, and the symptom is a set that plays fine for an hour and then
  // shows nothing until it is unplugged.
  const player = useVideoPlayer(null, (p) => {
    p.muted = true; // A dining room has its own sound.
    p.loop = false; // The playlist advances; the clip does not repeat itself.
  });

  useEffect(() => {
    if (!current) return;
    if (current.kind !== "video") {
      // ⚠️ Paused rather than left running: audio is muted, but a decoder still
      // working through a clip nobody can see is battery, heat and, on a cheap
      // set, the reason the next video stutters.
      player.pause();
      return;
    }
    player.replace({ uri: current.localUri });
    player.play();
  }, [current, player]);

  // A picture moves on by its own clock; a video moves on when it ends.
  useEffect(() => {
    if (!current || current.kind !== "image") return;
    const id = setTimeout(
      () => setIndex((n) => n + 1),
      Math.max(3, current.seconds) * 1000,
    );
    return () => clearTimeout(id);
  }, [current]);

  useEventListener(player, "playToEnd", () => {
    // ⚠️ With one video in the loop this lands back on the same item, and a
    // player that has played to its end will not start again on its own — so
    // the effect above has to see a change. Replaying here is the whole of it.
    if (items.length <= 1) {
      player.currentTime = 0;
      player.play();
      return;
    }
    setIndex((n) => n + 1);
  });

  // ⚠️ **A video that fails to open must not stop the loop.** A file truncated
  // by a power cut mid-download, or a clip this particular set's decoder will
  // not take, would otherwise be a still black screen for the rest of the day.
  useEventListener(player, "statusChange", ({ status }) => {
    if (status === "error" && items.length > 1) setIndex((n) => n + 1);
  });

  if (!current) return <View style={styles.black} />;

  return (
    <View style={styles.black}>
      {current.kind === "video" ? (
        <VideoView
          style={StyleSheet.absoluteFill}
          player={player}
          contentFit="contain"
          nativeControls={false}
          // Nothing about a wall-mounted screen should offer to go full screen
          // or hand audio to another device: there is nobody there to ask.
          fullscreenOptions={{ enable: false }}
          allowsPictureInPicture={false}
        />
      ) : (
        <Image
          style={StyleSheet.absoluteFill}
          source={{ uri: current.localUri }}
          resizeMode="contain"
          // ⚠️ Keyed by the item, so React replaces the view rather than
          // re-using it with a new source: re-using it shows the previous
          // picture until the next one has decoded, which on a slow set is a
          // visible flash of the wrong slide.
          key={current.id}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  black: { flex: 1, backgroundColor: "#000000" },
});
