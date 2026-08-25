import { ActivityIndicator, View } from "react-native";
import { StatusBar } from "expo-status-bar";

import { useSession } from "./src/session";
import { LoginScreen, ServerScreen, TablesScreen } from "./src/screens";

// Keel Waiter.
//
// ⚠️ **Four states, not a boolean.** "Which restaurant" and "who is signed in"
// are separate questions with separate answers: a phone that moves to another
// job forgets the restaurant, and the end of a shift does not. Collapsing them
// would make the common action cost the rare one's setup.

export default function App() {
  const { session, useServer, signIn, signOut, forgetServer } = useSession();

  return (
    <>
      <StatusBar style="dark" />
      {session.state === "loading" && (
        <View style={{ flex: 1, alignItems: "center", justifyContent: "center" }}>
          <ActivityIndicator />
        </View>
      )}

      {session.state === "noServer" && <ServerScreen onChosen={useServer} />}

      {session.state === "signedOut" && (
        <LoginScreen
          address={session.address}
          onSignIn={(u, p) => signIn(session.address, u, p)}
          onForget={forgetServer}
        />
      )}

      {session.state === "ready" && (
        <TablesScreen
          staff={session.staff}
          onSignOut={() => signOut(session.address)}
        />
      )}
    </>
  );
}
