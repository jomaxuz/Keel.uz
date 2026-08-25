import { useState } from "react";
import { ActivityIndicator, View } from "react-native";
import { StatusBar } from "expo-status-bar";

import { useSession } from "./src/session";
import { CheckScreen } from "./src/check";
import { LoginScreen, ServerScreen, TablesScreen } from "./src/screens";

// Keel Waiter.
//
// ⚠️ **Four states, not a boolean.** "Which restaurant" and "who is signed in"
// are separate questions with separate answers: a phone that moves to another
// job forgets the restaurant, and the end of a shift does not. Collapsing them
// would make the common action cost the rare one's setup.

export default function App() {
  const { session, useServer, signIn, signOut, forgetServer } = useSession();
  // ⚠️ **One level of navigation, held here, rather than a router.** There are
  // two screens behind the sign-in and the phone's back button has nothing else
  // to mean; a navigation library at this size would be a dependency carrying
  // one decision. It goes in the moment there is a third destination.
  const [open, setOpen] = useState<{ checkId: string; branchId: string } | null>(
    null,
  );

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

      {session.state === "ready" && open === null && (
        <TablesScreen
          staff={session.staff}
          onSignOut={() => signOut(session.address)}
          onOpenCheck={(checkId, branchId) => setOpen({ checkId, branchId })}
        />
      )}

      {session.state === "ready" && open !== null && (
        <CheckScreen
          checkId={open.checkId}
          branchId={open.branchId}
          onBack={() => setOpen(null)}
        />
      )}
    </>
  );
}
