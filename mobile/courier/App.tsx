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
import { EarningsScreen } from "./src/earnings";
import { NoticeProvider } from "./src/notice";
import { OrdersScreen } from "./src/orders";
import { PrefsProvider, usePrefs } from "./src/prefs";
import { SettingsScreen } from "./src/settings";
import { usePush } from "./src/push";
import { useSession } from "./src/session";
import { hydrateTokens } from "./src/tokens";
import { stopBackgroundUpdates } from "./src/background";
import { useTracking } from "./src/tracking";
import { useUI } from "./src/ui";

// Keel Courier.
//
// ⚠️ **Four states, not a boolean.** "Which restaurant" and "who is signed in"
// are separate questions with separate answers: a courier who changes employer
// forgets the restaurant, and the end of a shift does not. Collapsing them
// would make the common action cost the rare one's setup.

export default function App() {
  // ⚠️ **Nothing renders until what was saved has been read**, and this is a
  // bug the waiter app shipped first: `PrefsProvider` picks the language and
  // the theme in a `useState` initialiser, which runs the moment it mounts. If
  // hydration is started by a screen *below* it, that initialiser reads an
  // empty store and every launch opens in Uzbek on the light theme, however
  // many times somebody chose otherwise.
  const [ready, setReady] = useState(false);
  useEffect(() => {
    void hydrateTokens().then(() => setReady(true));
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

type Tab = "orders" | "earnings" | "settings";

function Root() {
  const { session, useServer, signIn, signOut, forgetServer, setStatus, refresh } =
    useSession();
  const { lang } = usePrefs();
  const { theme, s } = useUI();
  const [tab, setTab] = useState<Tab>("orders");

  // ⚠️ **The tracker lives here, above the tabs, and that is deliberate.**
  // Mounted inside the orders screen it would be torn down and restarted every
  // time a courier looked at their earnings — losing the buffered fixes and,
  // worse, the permission-granted state that opens the delivered button. The
  // shift is a property of the session, so the stream that follows it is too.
  const onShift = session.state === "ready" && session.courier.status !== "off";
  const tracking = useTracking(onShift);

  // ⚠️ **Registered once signed in, not at launch**, and re-registered when the
  // language changes: a notification is written by the server, so the language
  // travels with the token. A permission prompt on the first screen is asked
  // before anybody knows what the app is for, and the answer to a question you
  // do not understand is "no".
  const push = usePush(
    session.state === "ready",
    lang,
    // Every one of these messages is about an order, so a tap lands on the
    // list — which is also where somebody who ignored the tap will look.
    useCallback(() => setTab("orders"), []),
  );

  // ⚠️ **The two ways out both drop the phone first.** A push token left behind
  // sends the next rider's addresses — names, phone numbers, doors — to whoever
  // now holds this handset, and the foreground service would go on reporting
  // somebody's position after they had signed out.
  const leave = useCallback(
    async (after: () => void) => {
      await push.forget();
      await stopBackgroundUpdates();
      after();
    },
    [push],
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
            {tab === "orders" && (
              <OrdersScreen
                courier={session.courier}
                tracking={tracking}
                onStatus={setStatus}
                onRefreshCourier={refresh}
              />
            )}
            {tab === "earnings" && <EarningsScreen />}
            {tab === "settings" && (
              <SettingsScreen
                courier={session.courier}
                address={session.address}
                pushState={push.state}
                pushDetail={push.detail}
                onRetryPush={push.retry}
                onSignOut={() => leave(() => signOut(session.address))}
                onForgetServer={() => leave(forgetServer)}
              />
            )}
          </View>
          <Tabs tab={tab} onTab={setTab} onShift={onShift} />
        </>
      )}
    </View>
  );
}

function Tabs({
  tab,
  onTab,
  onShift,
}: {
  tab: Tab;
  onTab: (t: Tab) => void;
  /** Drawn as a dot on the orders tab. ⚠️ The one piece of state worth carrying
   *  into the bar: a courier reading their earnings has no other way to see
   *  that they are still on shift, and "why did nobody send me anything" is
   *  answered by that dot more often than by anything else here. */
  onShift: boolean;
}) {
  const { t } = usePrefs();
  const { theme } = useUI();
  // ⚠️ The home indicator and the gesture bar sit under this. Without the inset
  // the last row of a list is unreachable on exactly the phones people carry.
  const insets = useSafeAreaInsets();

  const items: { key: Tab; icon: keyof typeof Feather.glyphMap; label: string }[] = [
    { key: "orders", icon: "package", label: t.tabs.orders },
    { key: "earnings", icon: "dollar-sign", label: t.tabs.earnings },
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
            <View>
              <Feather
                name={it.icon}
                size={21}
                color={on ? theme.accent : theme.muted}
              />
              {it.key === "orders" && onShift && (
                <View style={[local.dot, { backgroundColor: theme.ok }]} />
              )}
            </View>
            {/* ⚠️ Labelled, not icons alone. Three glyphs with no words is a
                guess every new courier has to make on their first evening. */}
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
  tabs: {
    flexDirection: "row",
    borderTopWidth: 1,
    paddingTop: 10,
  },
  tab: { flex: 1, alignItems: "center", gap: 3 },
  dot: {
    position: "absolute",
    top: -1,
    right: -3,
    width: 8,
    height: 8,
    borderRadius: 4,
  },
});
