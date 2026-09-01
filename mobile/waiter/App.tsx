import { useCallback, useEffect, useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider, useSafeAreaInsets } from "react-native-safe-area-context";
// ⚠️ **From the family's own path, not the package index.** The index
// re-exports every icon set it ships — AntDesign, MaterialIcons, Ionicons
// and a dozen more — and each carries a glyph map, so importing one name
// from it pulls all of them into the bundle. Measured on these exact
// screens: 2.0 MB and 688 modules from the index, 1.6 MB and 634 from here.
import Feather from "@expo/vector-icons/Feather";

import { CheckScreen } from "./src/check";
import { FloorScreen } from "./src/floor";
import { LoginScreen, ServerScreen } from "./src/auth";
import { NoticeProvider } from "./src/notice";
import { OfflineScreen } from "./src/offlinescreen";
import { PrefsProvider, usePrefs } from "./src/prefs";
import { ProfileScreen } from "./src/profile";
import { SettingsScreen } from "./src/settings";
import { usePushRegistration } from "./src/push";
import { useSession } from "./src/session";
import { openOfflineStore } from "./src/offline";
import { hydrateTokens } from "./src/tokens";
import { useUI } from "./src/ui";

// Keel Waiter.
//
// ⚠️ **Four states, not a boolean.** "Which restaurant" and "who is signed in"
// are separate questions with separate answers: a phone that moves to another
// job forgets the restaurant, and the end of a shift does not. Collapsing them
// would make the common action cost the rare one's setup.

export default function App() {
  // ⚠️ **Nothing renders until what was saved has been read, and that ordering
  // is a bug this app already shipped.** `PrefsProvider` picks the language and
  // the theme in a `useState` initialiser — which runs the moment it mounts. If
  // hydration is started by a screen *below* it, that initialiser reads an
  // empty store and every launch opens in Uzbek on the light theme, however
  // many times somebody chose otherwise. The setting was being saved correctly
  // the whole time; it was being read too early.
  //
  // ⚠️ Gating on it rather than re-reading afterwards: a provider that adopted
  // the saved values on a later tick would paint one frame of the wrong
  // language, and a screen that changes language while somebody is looking at
  // it reads as a fault rather than as a preference.
  const [ready, setReady] = useState(false);
  useEffect(() => {
    // ⚠️ Both before the first render, and the disk before the tokens is not
    // an ordering that matters — what matters is that neither is behind a
    // screen. A sale queued while the store was still unset would be written to
    // nowhere and reported as saved.
    void Promise.all([hydrateTokens(), openOfflineStore()]).then(() =>
      setReady(true),
    );
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

type Tab = "floor" | "profile" | "settings";

function Root() {
  const { session, useServer, signIn, signOut, forgetServer, retry } =
    useSession();
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [tab, setTab] = useState<Tab>("floor");
  // ⚠️ **One level of navigation, held here, rather than a router.** The check
  // is the only place a tab leads to, and the phone's back button has nothing
  // else to mean. A navigation library at this size is a dependency carrying
  // one decision; it goes in the moment there is a third destination.
  const [open, setOpen] = useState<{ checkId: string; branchId: string } | null>(
    null,
  );
  const branchId = session.state === "ready" ? (session.staff.branchId ?? "") : "";
  // ⚠️ Registered once signed in, not at launch: a permission prompt on the
  // first screen is asked before anybody knows what the app is for, and the
  // answer to a question you do not understand is "no" — which on iOS is close
  // to permanent.
  const push = usePushRegistration(
    session.state === "ready",
    useCallback(
      (checkId: string) => setOpen({ checkId, branchId }),
      [branchId],
    ),
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

      {/* ⚠️ **Before the login screen, not instead of an error on it.** A
          launch with no network used to land on the password field, where the
          right password fails and the app blames the person for a network they
          cannot see. */}
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

      {session.state === "ready" && open !== null && (
        <CheckScreen
          checkId={open.checkId}
          branchId={open.branchId}
          onBack={() => setOpen(null)}
        />
      )}

      {session.state === "ready" && open === null && (
        <>
          <View style={{ flex: 1 }}>
            {tab === "floor" && (
              <FloorScreen
                onOpenCheck={(checkId, branchId) =>
                  setOpen({ checkId, branchId })
                }
              />
            )}
            {tab === "profile" && <ProfileScreen staff={session.staff} />}
            {tab === "settings" && (
              <SettingsScreen
                staff={session.staff}
                address={session.address}
                // ⚠️ The phone is dropped **before** the token is cleared, or
                // the request goes out unauthenticated and the row stays —
                // sending the next evening's tables to whoever went home.
                pushState={push.state}
                onRetryPush={push.retry}
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
      { key: "floor", icon: "grid", label: t.tabs.floor },
      { key: "profile", icon: "user", label: t.tabs.profile },
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
          <Pressable
            key={it.key}
            style={local.tab}
            onPress={() => onTab(it.key)}
          >
            <Feather
              name={it.icon}
              size={21}
              color={on ? theme.accent : theme.muted}
            />
            {/* ⚠️ Labelled, not icons alone. Three glyphs with no words is a
                guess every new waiter has to make on their first evening. */}
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
});
