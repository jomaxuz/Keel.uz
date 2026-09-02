import { StatusBar } from "expo-status-bar";
import { useKeepAwake } from "expo-keep-awake";
import { View } from "react-native";

import {
  LoadingScreen,
  PairedScreen,
  PairingScreen,
  ServerScreen,
} from "./src/screens";
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
const APP_VERSION = "1.0.2";

export default function App() {
  // ⚠️ **A television must never sleep, and Android will put it to sleep.**
  // Without this the dining room's screen shows a screensaver an hour after
  // opening, and the restaurant reports the app as broken — correctly, from
  // where they are standing.
  useKeepAwake();

  const { state, useServer } = useTVSession(APP_VERSION);

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
      {state.state === "paired" && (
        <PairedScreen screen={state.screen} offline={false} />
      )}
      {state.state === "offline" && (
        <PairedScreen screen={state.screen} offline />
      )}
    </View>
  );
}
