// The order board.
//
// ⚠️ **Read from four metres away by somebody who is looking for one number
// among twenty.** That is the only thing this screen has to do well, and it is
// what every size, colour and position here is chosen against — not density.
// A board that fits more numbers by making them smaller has failed at its job
// while looking like it is doing more of it.
//
// ⚠️ **Ready is on the right and louder.** The two columns are not equals: a
// guest checks "is mine ready" many times and "is mine cooking" once, and the
// answer they came for should not need finding.

import { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, Text, View, useWindowDimensions } from "react-native";

import { api } from "@/lib/api";

import { noteServerTime, serverNow } from "./clock";

export interface Board {
  cooking: string[];
  ready: string[];
}

/** How often the wall asks. ⚠️ Not the heartbeat's minute: a guest standing at
 *  a counter watching their number not appear is the whole experience this
 *  screen delivers, and a minute of that is a complaint. */
const BOARD_POLL_MS = 10_000;

/** How stale an answer may be before the board goes quiet.
 *
 *  ⚠️ **The opposite rule from the playlist, and deliberately so.** A loop that
 *  keeps playing through a dropped connection is showing the restaurant's own
 *  content, one revision behind at worst. A board that keeps showing numbers is
 *  making a claim about food — and a number that says "ready" when it is not
 *  sends a guest to a counter to be told no. Better a screen that says nothing:
 *  they ask a person, which is what they would have done anyway. */
const BOARD_STALE_MS = 2 * 60_000;

const EMPTY: Board = { cooking: [], ready: [] };

/** What the counter is doing, or nothing when we cannot know. */
export function useTVBoard(active: boolean): Board {
  const [board, setBoard] = useState<Board>(EMPTY);
  // When the answer we are holding was true. ⚠️ Server time, not the set's own:
  // a cheap television's clock is months out, and this comparison decides
  // whether a room is shown numbers or nothing.
  const at = useRef(0);

  const poll = useCallback(async () => {
    try {
      const res = await api.tvBoard();
      noteServerTime(res.serverTime);
      at.current = serverNow();
      setBoard({ cooking: res.cooking, ready: res.ready });
    } catch {
      // Silence. Whether this becomes a blank board is decided below, by how
      // old the last good answer is — one dropped request on a restaurant's
      // wifi is not a reason to clear a wall.
      if (serverNow() - at.current > BOARD_STALE_MS) {
        at.current = 0;
        setBoard(EMPTY);
      }
    }
  }, []);

  useEffect(() => {
    if (!active) {
      setBoard(EMPTY);
      at.current = 0;
      return;
    }
    void poll();
    const id = setInterval(() => void poll(), BOARD_POLL_MS);
    return () => clearInterval(id);
  }, [active, poll]);

  return board;
}

/** Whether there is anything worth drawing. A quiet afternoon is not a board. */
export function boardHasAnything(b: Board): boolean {
  return b.cooking.length > 0 || b.ready.length > 0;
}

/** The whole wall: the `board` mode. */
export function BoardScreen({
  board,
  branchName,
}: {
  board: Board;
  branchName?: string;
}) {
  const { width } = useWindowDimensions();
  // ⚠️ Sized from the screen rather than fixed: this app runs on a 32" set by a
  // counter and on a 65" one across a room, and a number tuned for one is
  // unreadable or absurd on the other.
  const numberSize = Math.round(width / 22);

  if (!boardHasAnything(board)) {
    // ⚠️ **Not an empty grid with two headings.** A board drawn with nothing
    // under it reads as broken — and between lunch and dinner it would read
    // that way for hours. The room's own name is a screen that is plainly on.
    return (
      <View style={styles.quiet}>
        <Text style={styles.quietName}>{branchName ?? "Keel"}</Text>
      </View>
    );
  }

  return (
    <View style={styles.board}>
      <Column
        title="Tayyorlanmoqda"
        numbers={board.cooking}
        size={numberSize}
      />
      <Column title="Tayyor" numbers={board.ready} size={numberSize} ready />
    </View>
  );
}

function Column({
  title,
  numbers,
  size,
  ready = false,
}: {
  title: string;
  numbers: string[];
  size: number;
  ready?: boolean;
}) {
  return (
    <View style={[styles.column, ready && styles.columnReady]}>
      <Text style={[styles.columnTitle, ready && styles.columnTitleReady]}>
        {title}
      </Text>
      <View style={styles.numbers}>
        {numbers.map((n) => (
          <Text
            key={n}
            style={[
              styles.number,
              { fontSize: size },
              ready && styles.numberReady,
            ]}
          >
            {n}
          </Text>
        ))}
      </View>
    </View>
  );
}

/** The `split` mode: the loop plays, with the ready numbers along the bottom.
 *
 *  ⚠️ **Ready only, not both columns.** A strip is a glance, and half a metre
 *  of wall cannot carry two lists — whichever one it carries badly is the one
 *  somebody misreads. Ready is the answer people are waiting for; the rest is
 *  on the receipt.
 *
 *  ⚠️ **It disappears when there is nothing to say**, giving the video the
 *  whole wall back. A permanent empty bar across a promotional video is a
 *  restaurant's own screen eaten by furniture. */
export function BoardStrip({ board }: { board: Board }) {
  const { width, height } = useWindowDimensions();
  if (board.ready.length === 0) return null;
  return (
    <View style={[styles.strip, { paddingBottom: Math.round(height * 0.03) }]}>
      <Text style={styles.stripTitle}>Tayyor</Text>
      {board.ready.map((n) => (
        <Text
          key={n}
          style={[styles.stripNumber, { fontSize: Math.round(width / 32) }]}
        >
          {n}
        </Text>
      ))}
    </View>
  );
}

// ⚠️ **Overscan again**: older sets cut 3–5% off every edge, so nothing sits
// closer than EDGE to the border. A number with its first digit off the screen
// is worse than no number — it is a number belonging to somebody else.
const EDGE = 48;

const C = {
  bg: "#000b1c",
  ink: "#ffffff",
  muted: "#8ea3c0",
  ready: "#37d67a",
  line: "#1b2c48",
};

const styles = StyleSheet.create({
  board: {
    flex: 1,
    flexDirection: "row",
    backgroundColor: C.bg,
    padding: EDGE,
    gap: EDGE,
  },
  column: { flex: 1, gap: 20 },
  // The ready column is given the visual weight as well as the better side.
  columnReady: {
    borderLeftWidth: 2,
    borderLeftColor: C.line,
    paddingLeft: EDGE,
  },
  columnTitle: {
    color: C.muted,
    fontSize: 28,
    fontWeight: "600",
    letterSpacing: 2,
    textTransform: "uppercase",
  },
  columnTitleReady: { color: C.ready },
  numbers: { flexDirection: "row", flexWrap: "wrap", gap: 24 },
  number: {
    color: C.muted,
    fontWeight: "800",
    // Monospaced, so 8 and B cannot trade places at four metres.
    fontFamily: "monospace",
  },
  numberReady: { color: C.ink },
  quiet: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: C.bg,
    padding: EDGE,
  },
  quietName: { color: C.muted, fontSize: 40, fontWeight: "700" },
  strip: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    alignItems: "center",
    flexWrap: "wrap",
    gap: 24,
    paddingHorizontal: EDGE,
    paddingTop: 16,
    // ⚠️ Solid, not translucent: a video behind a number is a number that is
    // legible on some frames and not on others, and nobody is going to wait for
    // a darker shot.
    backgroundColor: "#000814",
    borderTopWidth: 2,
    borderTopColor: C.line,
  },
  stripTitle: {
    color: C.ready,
    fontSize: 22,
    fontWeight: "600",
    letterSpacing: 2,
    textTransform: "uppercase",
  },
  stripNumber: { color: C.ink, fontWeight: "800", fontFamily: "monospace" },
});
