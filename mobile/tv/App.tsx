import { StatusBar } from "expo-status-bar";
import { useKeepAwake } from "expo-keep-awake";
import { View } from "react-native";

import {
  LoadingScreen,
  PairedScreen,
  PairingScreen,
  ServerScreen,
} from "./src/screens";
import { BoardScreen, BoardStrip, useTVBoard } from "./src/board";
import { PlayerScreen } from "./src/player";
import { useTVPlaylist } from "./src/playlist";
import { useTVSession } from "./src/session";

// Keel TV — the screen on the restaurant's wall.
//
// ⚠️ **The fifth app, and the only one nobody holds.** Every other app in this
// repository is picked up by a person who can react to it: a waiter reads an
// error, a cashier presses a button again, a courier restarts something. This
// one hangs above head height in a room full of guests, and the only person who
// can act on anything it says is not in the room. That single fact decides the
// whole design:
//
//   - it never shows an error to the room — a dropped connection is silence,
//     not a red box in front of forty people eating;
//   - it needs no input at all after setup, because there is no keyboard and
//     the remote is in a drawer;
//   - it is paired by a code it *shows*, not by credentials it asks for;
//   - and being unpaired from the panel has to reach it, which is why it says
//     hello every minute rather than trusting a token issued a year ago.
//
// ⚠️ **The version string is read from here rather than from a build flag**, so
// the panel's "this screen is running an old build" is a fact about the JavaScript
// actually running — which, with over-the-air updates, is the only version that
// answers "why is this television behaving differently from the others".
const APP_VERSION = "1.2.1";

export default function App() {
  // ⚠️ **A television must never sleep, and Android will put it to sleep.**
  // Without this the dining room's screen shows a screensaver an hour after
  // opening, and the restaurant reports the app as broken — correctly, from
  // where they are standing.
  useKeepAwake();

  const { state, useServer } = useTVSession(APP_VERSION);

  // ⚠️ **The playlist is asked for even while the screen is offline**, because
  // the answer usually comes off this set's own disk: the files were downloaded
  // when the panel last changed something, and a dropped connection is not a
  // reason for a dining room to go dark. The version is what the heartbeat
  // carries; `null` means it has not landed yet, and the stored list plays.
  const paired = state.state === "paired" || state.state === "offline";
  const { items, downloading } = useTVPlaylist(
    paired ? state.contentVersion : null,
  );

  const screen = paired ? state.screen : null;

  // ⚠️ **The board is polled only by the screens that draw one.** A television
  // in the dining room set to `content` has no use for the counter's numbers,
  // and every screen in the chain asking for them every ten seconds is a load
  // the restaurant pays for in nothing.
  const wantsBoard = screen?.mode === "board" || screen?.mode === "split";
  const board = useTVBoard(paired && wantsBoard);

  // ⚠️ **A screen set to the order board does not play the loop.** The mode is
  // the wall's answer, not the playlist's: one television by the counter shows
  // numbers all day while the one in the dining room plays the menu, and both
  // read the same branch's playlist.
  const playing = paired && screen?.mode !== "board" && items.length > 0;
  // ⚠️ **`split` with an empty playlist falls back to the whole board**, not to
  // a placeholder. Split means "the loop, with the numbers under it"; with no
  // loop there is still a counter, and the numbers are the half of that screen
  // somebody in the room is actually waiting on.
  const showBoard =
    paired &&
    (screen?.mode === "board" || (screen?.mode === "split" && !playing));

  return (
    <View style={{ flex: 1, backgroundColor: "#000b1c" }}>
      {/* Hidden: a status bar on a wall-mounted screen is a strip of somebody
          else's operating system across a restaurant's own display. */}
      <StatusBar hidden />
      {state.state === "loading" && <LoadingScreen />}
      {state.state === "noServer" && <ServerScreen onSubmit={useServer} />}
      {state.state === "unreachable" && (
        <ServerScreen
          onSubmit={useServer}
          initial={state.address}
          unreachable
          answered={state.answered}
        />
      )}
      {state.state === "pairing" && (
        <PairingScreen code={state.code} expiresAt={state.expiresAt} />
      )}
      {showBoard && (
        <BoardScreen board={board} branchName={screen?.branchName} />
      )}
      {playing && <PlayerScreen items={items} />}
      {/* ⚠️ Over the video, not beside it: the strip is `split` mode, and it
          draws only when the counter has something ready — an empty bar across
          a restaurant's own promotional video is furniture. */}
      {playing && screen?.mode === "split" && <BoardStrip board={board} />}
      {state.state === "paired" && !playing && !showBoard && (
        <PairedScreen
          screen={state.screen}
          offline={false}
          downloading={downloading}
          // ⚠️ `null` is "the first heartbeat has not landed", which is a
          // freshly paired screen's first minute — not an empty playlist. The
          // two used to be shown with the same sentence.
          waiting={state.contentVersion === null}
        />
      )}
      {state.state === "offline" && !playing && !showBoard && (
        <PairedScreen
          screen={state.screen}
          offline
          downloading={false}
          waiting={false}
        />
      )}
    </View>
  );
}
