import { StatusBar } from "expo-status-bar";
import { useKeepAwake } from "expo-keep-awake";
import { View } from "react-native";

import {
  LoadingScreen,
  PairedScreen,
  PairingScreen,
  ServerScreen,
} from "./src/screens";
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
const APP_VERSION = "1.1.0";

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
  const { items } = useTVPlaylist(paired ? state.contentVersion : null);

  // ⚠️ **A screen set to the order board does not play the loop**, and an empty
  // loop is not a black rectangle. The board itself is the next stage; until
  // then those screens keep the placeholder that at least names the room, which
  // is what somebody pairing the other televisions needs to see.
  const screen = paired ? state.screen : null;
  const playing = paired && screen?.mode !== "board" && items.length > 0;

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
      {playing && <PlayerScreen items={items} />}
      {state.state === "paired" && !playing && (
        <PairedScreen screen={state.screen} offline={false} />
      )}
      {state.state === "offline" && !playing && (
        <PairedScreen screen={state.screen} offline />
      )}
    </View>
  );
}
