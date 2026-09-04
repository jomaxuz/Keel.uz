import { useCallback, useEffect, useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider, useSafeAreaInsets } from "react-native-safe-area-context";
// ⚠️ **From the family's own path, not the package index.** The index
// re-exports every icon set it ships, each with its own glyph map, so importing
// one name from it pulls all of them into the bundle — measured at 2.0 MB
// against 1.6 MB on the waiter app's screens.
import Feather from "@expo/vector-icons/Feather";

import { LoginScreen, ServerScreen } from "./src/auth";
import { BuyScreen } from "./src/buy";
import { ZakupScreen, canWriteHere } from "./src/zakup";
import { NoticeProvider } from "./src/notice";
import { OfflineScreen } from "./src/offlinescreen";
import { PrefsProvider, usePrefs } from "./src/prefs";
import { ProfileScreen } from "./src/profile";
import { SettingsScreen } from "./src/settings";
import { usePushRegistration } from "./src/push";
import { useSession } from "./src/session";
import { initDevice } from "./src/device";
import { hydrateTokens } from "./src/tokens";
import { useUI } from "./src/ui";

// Keel Team — the app everybody in the restaurant has.
//
// ⚠️ **The third phone app, and the smallest on purpose.** A waiter has the
// floor, a courier has the road; a cook, a barman, a dishwasher and a cleaner
// have one thing the system needs from them and one thing they need from it:
// the shift starts, the shift ends, and this is what I have worked. Everything
// else on their phone would be somebody else's screen.
//
// ⚠️ **It exists because attendance was the one thing with no phone at all.**
// The clock-in lives on a web page (`/staff`), so an employee had to be told a
// URL, keep it in a browser tab and find it again every morning — and the till
// now refuses a PIN without an open shift, which turns "I could not find the
// page" into "I cannot start work".

export default function App() {
  // ⚠️ **Nothing renders until what was saved has been read**, and this is a
  // bug the waiter app shipped first: `PrefsProvider` picks the language and
  // the theme in a `useState` initialiser, which runs the moment it mounts. If
  // hydration is started by a screen *below* it, that initialiser reads an
  // empty store and every launch opens in Uzbek on the light theme, however
  // many times somebody chose otherwise.
  const [ready, setReady] = useState(false);
  useEffect(() => {
    void hydrateTokens().then(() => {
      // ⚠️ After the store is hydrated and before the first request: the id
      // lives beside the tokens, and the login is the call that needs it.
      initDevice("team");
      setReady(true);
    });
  }, []);

  if (!ready) {
    // One frame on a fast phone, and the brand's own colour rather than white:
    // a white flash between the splash and the app is the thing that reads as
    // two applications starting.
    return <View style={{ flex: 1, backgroundColor: "#000b1c" }} />;
  }

  return (
    <SafeAreaProvider>
      <PrefsProvider>
        <NoticeProvider>
          <Root />
        </NoticeProvider>
      </PrefsProvider>
    </SafeAreaProvider>
  );
}

type Tab = "profile" | "buy" | "zakup" | "settings";

function Root() {
  const { session, useServer, signIn, signOut, forgetServer, retry } =
    useSession();
  const { lang } = usePrefs();
  const { theme, s } = useUI();
  const [tab, setTab] = useState<Tab>("profile");

  // ⚠️ Registered once signed in, not at launch: a permission prompt on the
  // first screen is asked before anybody knows what the app is for, and the
  // answer to a question you do not understand is "no" — which on iOS is close
  // to permanent.
  //
  // ⚠️ **A tap has nowhere else to go here.** In the waiter app it opens the
  // table the kitchen finished; this app has one screen, and the honest
  // behaviour is to land on it rather than to invent a destination.
  const push = usePushRegistration(
    session.state === "ready",
    lang,
    useCallback(() => setTab("profile"), []),
  );

  return (
    <View style={s.screen}>
      {/* The bar's contrast follows the scheme, or it disappears into it. */}
      <StatusBar style={theme.scheme === "dark" ? "light" : "dark"} />

      {session.state === "loading" && (
        <View style={s.centered}>
          <ActivityIndicator color={theme.accent} />
        </View>
      )}

      {session.state === "noServer" && <ServerScreen onChosen={useServer} />}

      {/* ⚠️ Before the login screen, not an error on it: a launch with no
          network used to land on the password field, where the right password
          fails and the app blames the person for a network they cannot see. */}
      {session.state === "offline" && (
        <OfflineScreen address={session.address} onRetry={() => void retry()} />
      )}

      {session.state === "signedOut" && (
        <LoginScreen
          address={session.address}
          onSignIn={(u, p) => signIn(session.address, u, p)}
          onForget={forgetServer}
        />
      )}

      {session.state === "ready" && (
        <>
          <View style={{ flex: 1 }}>
            {tab === "profile" && <ProfileScreen staff={session.staff} />}
            {tab === "buy" && <BuyScreen />}
            {tab === "zakup" && <ZakupScreen />}
            {tab === "settings" && (
              <SettingsScreen
                staff={session.staff}
                address={session.address}
                pushState={push.state}
                onRetryPush={push.retry}
                // ⚠️ The phone is dropped **before** the token is cleared, or
                // the request goes out unauthenticated and the row stays —
                // sending somebody else's pay slip to a phone that has left.
                onSignOut={async () => {
                  await push.forget();
                  signOut(session.address);
                }}
                onForgetServer={async () => {
                  await push.forget();
                  forgetServer();
                }}
              />
            )}
          </View>
          {/* ⚠️ **The tab exists only for the account that may use it**, the
              same way the pass screen does for a waiter. A buyer is one job in
              a restaurant, and a cook opening this app should not be shown a
              screen that would refuse them — a button that says no teaches a
              room to stop reading the app. */}
          <Tabs
            tab={tab}
            onTab={setTab}
            canBuy={(session.staff.perms ?? []).includes("buy")}
            // ⚠️ Narrower than the permission on purpose — see canWriteHere.
            canOrder={canWriteHere(session.staff)}
          />
        </>
      )}
    </View>
  );
}

function Tabs({
  tab,
  onTab,
  canBuy,
  canOrder,
}: {
  tab: Tab;
  onTab: (t: Tab) => void;
  canBuy: boolean;
  canOrder: boolean;
}) {
  const { t } = usePrefs();
  const { theme } = useUI();
  // ⚠️ The home indicator and the gesture bar sit under this. Without the inset
  // the last row of a list is unreachable on exactly the phones people carry.
  const insets = useSafeAreaInsets();

  const items: { key: Tab; icon: keyof typeof Feather.glyphMap; label: string }[] =
    [
      { key: "profile", icon: "clock", label: t.tabs.profile },
      ...(canBuy
        ? ([{ key: "buy", icon: "shopping-bag", label: t.tabs.buy }] as const)
        : []),
      ...(canOrder
        ? ([{ key: "zakup", icon: "clipboard", label: t.tabs.zakup }] as const)
        : []),
      { key: "settings", icon: "settings", label: t.tabs.settings },
    ];

  return (
    <View
      style={[
        local.tabs,
        {
          backgroundColor: theme.surface,
          borderTopColor: theme.line,
          paddingBottom: Math.max(insets.bottom, 10),
        },
      ]}
    >
      {items.map((it) => {
        const on = it.key === tab;
        return (
          <Pressable key={it.key} style={local.tab} onPress={() => onTab(it.key)}>
            <Feather
              name={it.icon}
              size={21}
              color={on ? theme.accent : theme.muted}
            />
            {/* ⚠️ Labelled, not icons alone. Two glyphs with no words is a
                guess every new employee has to make on their first morning. */}
            <Text
              style={{
                fontSize: 11,
                color: on ? theme.accent : theme.muted,
                fontWeight: on ? "600" : "400",
              }}
            >
              {it.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

const local = StyleSheet.create({
  tabs: { flexDirection: "row", borderTopWidth: 1, paddingTop: 10 },
  tab: { flex: 1, alignItems: "center", gap: 3 },
});
