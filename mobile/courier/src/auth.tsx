import { useState } from "react";
import {
  KeyboardAvoidingView,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
// ⚠️ **From the family's own path, not the package index.** The index
// re-exports every icon set it ships — AntDesign, MaterialIcons, Ionicons and a
// dozen more — and each carries a glyph map, so importing one name from it
// pulls all of them into the bundle. Measured on the waiter app's screens:
// 2.0 MB and 688 modules from the index, 1.6 MB and 634 from here.
import Feather from "@expo/vector-icons/Feather";

import { ApiError } from "@/lib/api";

import { KeelMark } from "./mark";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// Getting in: which restaurant, and who.

export function ServerScreen({
  onChosen,
}: {
  onChosen: (address: string) => boolean;
}) {
  const { t } = usePrefs();
  const { theme, s, bottom } = useUI();
  const [address, setAddress] = useState("");
  const [bad, setBad] = useState(false);

  return (
    <KeyboardAvoidingView
      style={{ flex: 1 }}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
    >
      <View style={[s.centered, { paddingBottom: bottom + 24 }]}>
        {/* The one place the app shows its own identity rather than the
            restaurant's — after this every screen belongs to the restaurant. */}
        <View style={[local.mark, { backgroundColor: theme.navy }]}>
          <KeelMark size={38} colour="#fea204" />
        </View>
        <Text style={s.h1}>{t.appName}</Text>
        <Text style={[s.muted, { marginBottom: 8 }]}>{t.server.title}</Text>
        <TextInput
          style={s.input}
          value={address}
          onChangeText={(v) => {
            setBad(false);
            setAddress(v);
          }}
          placeholder={t.server.placeholder}
          placeholderTextColor={theme.muted}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="url"
        />
        {/* ⚠️ The short form is the example, because it is what somebody knows.
            A placeholder showing the full URL teaches the wrong answer to
            everybody who reads it. */}
        <Text style={[s.muted, { textAlign: "center" }]}>{t.server.hint}</Text>
        {bad && <Text style={s.error}>{t.server.bad}</Text>}
        <Pressable
          style={[s.primary, { marginTop: 6, alignSelf: "stretch", maxWidth: 340 }]}
          onPress={() => {
            if (!onChosen(address)) setBad(true);
          }}
        >
          <Text style={s.primaryText}>{t.server.next}</Text>
          <Feather name="arrow-right" size={18} color={theme.onAccent} />
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}

export function LoginScreen({
  address,
  onSignIn,
  onForget,
}: {
  address: string;
  onSignIn: (username: string, password: string) => Promise<void>;
  onForget: () => void;
}) {
  const { t } = usePrefs();
  const { theme, s, bottom } = useUI();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [show, setShow] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    setBusy(true);
    setError("");
    try {
      await onSignIn(username.trim(), password);
    } catch (e) {
      // ⚠️ The server's own words. It tells a wrong password apart from a
      // switched-off account, and those send somebody to two different people.
      //
      // ⚠️ **And a network failure is neither.** "Could not sign in" under a
      // correct password is what makes somebody type it a third time; this is
      // the same distinction the launch screen makes, said in one line because
      // here there is only one line to say it in.
      setError(
        e instanceof ApiError
          ? e.message
          : `${t.offline.title} — ${t.common.retry.toLowerCase()}`,
      );
      setBusy(false);
    }
  }

  return (
    <KeyboardAvoidingView
      style={{ flex: 1 }}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
    >
      <View style={[s.centered, { paddingBottom: bottom + 24 }]}>
        <Text style={s.h1}>{t.login.title}</Text>
        <View style={local.chip}>
          <Feather name="home" size={13} color={theme.muted} />
          <Text style={s.muted}>{address}</Text>
        </View>
        <TextInput
          style={s.input}
          value={username}
          onChangeText={setUsername}
          placeholder={t.login.username}
          placeholderTextColor={theme.muted}
          autoCapitalize="none"
          autoCorrect={false}
          textContentType="username"
        />
        <View style={local.passwordRow}>
          <TextInput
            style={[s.input, { paddingRight: 46 }]}
            value={password}
            onChangeText={setPassword}
            placeholder={t.login.password}
            placeholderTextColor={theme.muted}
            secureTextEntry={!show}
            autoCapitalize="none"
            textContentType="password"
          />
          {/* ⚠️ A password typed on a phone, outdoors, by somebody in a hurry —
              the eye is what stops the third failed attempt. */}
          <Pressable
            style={local.eye}
            hitSlop={10}
            onPress={() => setShow((v) => !v)}
          >
            <Feather
              name={show ? "eye-off" : "eye"}
              size={18}
              color={theme.muted}
            />
          </Pressable>
        </View>
        {error !== "" && <Text style={s.error}>{error}</Text>}
        <Pressable
          style={[s.primary, { alignSelf: "stretch", maxWidth: 340 }]}
          disabled={busy}
          onPress={() => void submit()}
        >
          <Text style={s.primaryText}>{busy ? "…" : t.login.submit}</Text>
        </Pressable>
        <Pressable onPress={onForget} hitSlop={10}>
          <Text style={s.link}>{t.login.other}</Text>
        </Pressable>
      </View>
    </KeyboardAvoidingView>
  );
}

const local = StyleSheet.create({
  mark: {
    width: 72,
    height: 72,
    borderRadius: 22,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: 4,
  },
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    marginBottom: 6,
  },
  passwordRow: { width: "100%", maxWidth: 340, justifyContent: "center" },
  eye: { position: "absolute", right: 14 },
});
