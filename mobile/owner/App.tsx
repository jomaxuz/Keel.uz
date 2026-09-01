import { useCallback, useEffect, useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider, useSafeAreaInsets } from "react-native-safe-area-context";
// ⚠️ **From the family's own path, not the package index**, which re-exports
// every icon set it ships — measured at 2.0 MB against 1.6 MB on the waiter
// app's screens.
import Feather from "@expo/vector-icons/Feather";

import { AlertsScreen } from "./src/alerts";
import { LoginScreen, ServerScreen } from "./src/auth";
import { NoticeProvider } from "./src/notice";
import { OfflineScreen } from "./src/offlinescreen";
import { OrdersScreen } from "./src/orders";
import { PrefsProvider, usePrefs } from "./src/prefs";
import { ReportsScreen } from "./src/reports";
import { SettingsScreen } from "./src/settings";
import { TodayScreen } from "./src/today";
import { usePushRegistration } from "./src/push";
import { useSession } from "./src/session";
import { initDevice } from "./src/device";
import { hydrateTokens } from "./src/tokens";
import { useUI } from "./src/ui";

// Keel Owner.
//
// ⚠️ **This is not the panel on a phone, and the difference is the whole
// design.** The panel is where somebody sits down and decides: a menu, a price,
// a rota, a campaign. Those are keyboard work and they stay there. What an
// owner does with a phone is *watch and react* — how is today going, what needs
// me now, accept that order, why was 400 000 taken off table six — and every
// screen here answers one of those in three seconds, standing up.
//
// ⚠️ **What is deliberately absent**: editing the menu, settings, CRM
// campaigns, stock documents, tech cards. Not because they are hard, but
// because a phone is where they would be done badly, and a screen that can
// change a price at a traffic light is a screen that eventually does.

export default function App() {
  // ⚠️ Nothing renders until what was saved has been read: `PrefsProvider`
  // picks the language and the theme in a `useState` initialiser, and reading
  // an empty store there opens every launch in Uzbek on the light theme.
  const [ready, setReady] = useState(false);
  useEffect(() => {
    void hydrateTokens().then(() => {
      // ⚠️ After the store is hydrated and before the first request: the id
      // lives beside the tokens, and the login is the call that needs it.
      initDevice("owner");
      setReady(true);
    });
  }, []);

  if (!ready) {
    // One frame on a fast phone, and the brand's own colour rather than white:
    // a white flash between the splash and the app reads as two applications
    // starting.
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

type Tab = "today" | "alerts" | "orders" | "reports" | "settings";

function Root() {
  const {
    session,
    useServer,
    signIn,
    signOut,
    forgetServer,
    retry,
    branchId,
    pickBranch,
  } = useSession();
  const { lang } = usePrefs();
  const { theme, s } = useUI();
  const [tab, setTab] = useState<Tab>("today");

  // ⚠️ **A tap lands where the message came from.** An owner who taps a
  // discount warning and arrives on a revenue tile has to go looking for the
  // thing they were just told about — which is the fastest way to teach
  // somebody that the notification was not worth opening.
  const push = usePushRegistration(
    session.state === "ready",
    lang,
    useCallback((type: string | undefined) => {
      setTab(type === "order" ? "orders" : "alerts");
    }, []),
  );

  const branches = session.state === "ready" ? session.branches : [];

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
            {tab === "today" && (
              <TodayScreen
                branches={branches}
                branchId={branchId}
                onBranch={pickBranch}
              />
            )}
            {tab === "alerts" && <AlertsScreen branchId={branchId} />}
            {tab === "orders" && <OrdersScreen branchId={branchId} />}
            {tab === "reports" && <ReportsScreen branchId={branchId} />}
            {tab === "settings" && (
              <SettingsScreen
                admin={session.admin}
                branches={branches}
                branchId={branchId}
                onBranch={pickBranch}
                address={session.address}
                pushState={push.state}
                onRetryPush={push.retry}
                // ⚠️ The phone is dropped **before** the token is cleared, or
                // the request goes out unauthenticated and the row stays —
                // delivering a restaurant's takings to a handset that has left.
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
          <Tabs tab={tab} onTab={setTab} />
        </>
      )}
    </View>
  );
}

function Tabs({ tab, onTab }: { tab: Tab; onTab: (t: Tab) => void }) {
  const { t } = usePrefs();
  const { theme } = useUI();
  // ⚠️ The home indicator and the gesture bar sit under this. Without the inset
  // the last row of a list is unreachable on exactly the phones people carry.
  const insets = useSafeAreaInsets();

  const items: { key: Tab; icon: keyof typeof Feather.glyphMap; label: string }[] =
    [
      { key: "today", icon: "bar-chart-2", label: t.tabs.today },
      { key: "alerts", icon: "bell", label: t.tabs.alerts },
      { key: "orders", icon: "shopping-bag", label: t.tabs.orders },
      { key: "reports", icon: "file-text", label: t.tabs.reports },
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
              size={20}
              color={on ? theme.accent : theme.muted}
            />
            {/* ⚠️ Labelled, not icons alone — five glyphs with no words is five
                guesses, and this app is opened once a day rather than all
                evening, so nobody builds the habit that would replace them. */}
            <Text
              style={{
                fontSize: 10,
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
