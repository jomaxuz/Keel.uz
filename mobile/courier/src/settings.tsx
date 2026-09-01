import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import Feather from "@expo/vector-icons/Feather";
import * as Application from "expo-application";

import { formatUzPhone } from "@/lib/format";
import type { Courier } from "@/lib/types";

import { LANGS, DICTS, type Lang } from "./i18n";
import type { PushState } from "./push";
import { usePrefs, type ThemeChoice } from "./prefs";
import { useUI } from "./ui";

// Language, appearance, and the two ways out.

export function SettingsScreen({
  courier,
  address,
  onSignOut,
  onForgetServer,
  pushState,
  onRetryPush,
}: {
  courier: Courier;
  address: string;
  /** ⚠️ Async: the push registration is dropped **before** the token is
   *  cleared, or the request goes out unauthenticated and the row stays —
   *  sending the next rider's addresses to a phone that has left. */
  onSignOut: () => void | Promise<void>;
  onForgetServer: () => void | Promise<void>;
  /** ⚠️ Shown rather than hidden: "they said they sent me an order and nothing
   *  arrived" has five causes, and this is the only place on the phone that
   *  tells them apart. */
  pushState: PushState;
  onRetryPush: () => void;
}) {
  const { t, lang, setLang, choice, setChoice } = usePrefs();
  const { theme, s } = useUI();

  const themes: { key: ThemeChoice; label: string; icon: keyof typeof Feather.glyphMap }[] = [
    { key: "system", label: t.settings.themeSystem, icon: "smartphone" },
    { key: "light", label: t.settings.themeLight, icon: "sun" },
    { key: "dark", label: t.settings.themeDark, icon: "moon" },
  ];

  return (
    <View style={s.screen}>
      <View style={s.header}>
        <Text style={s.h1}>{t.settings.title}</Text>
      </View>

      <ScrollView contentContainerStyle={s.list}>
        <Section title={t.settings.language} icon="globe">
          {LANGS.map((l) => (
            <Choice
              key={l}
              // ⚠️ Each language names itself, in itself. A list that said
              // "Russian" in Uzbek is a list a Russian speaker has to decode
              // before they can leave the language they cannot read.
              label={DICTS[l].lang}
              on={lang === l}
              onPress={() => setLang(l as Lang)}
            />
          ))}
        </Section>

        <Section title={t.settings.theme} icon="moon">
          {themes.map((x) => (
            <Choice
              key={x.key}
              label={x.label}
              icon={x.icon}
              on={choice === x.key}
              onPress={() => setChoice(x.key)}
            />
          ))}
        </Section>

        <Section title={t.push.title} icon="bell">
          <View style={local.choice}>
            <Feather
              name={pushState === "working" ? "check-circle" : "alert-circle"}
              size={16}
              color={pushState === "working" ? theme.ok : theme.warn}
            />
            <View style={{ flex: 1 }}>
              <Text style={s.body}>{t.push[pushState]}</Text>
              <Text style={s.muted}>{t.push.hint[pushState]}</Text>
            </View>
          </View>
          {pushState !== "working" && (
            <Pressable style={local.choice} onPress={onRetryPush}>
              <Feather name="refresh-cw" size={16} color={theme.accent} />
              <Text style={[s.body, { color: theme.accent }]}>
                {t.common.retry}
              </Text>
            </Pressable>
          )}
        </Section>

        <Section title={t.settings.account} icon="user">
          <Row label={courier.name} value={courier.vehicle} />
          {courier.phone ? (
            <Row label={t.settings.phone} value={formatUzPhone(courier.phone)} />
          ) : null}
          <Row label={t.settings.restaurant} value={address} />
          <Row
            label={t.settings.version}
            value={Application.nativeApplicationVersion ?? "—"}
          />
        </Section>

        {/* ⚠️ Two ways out, kept apart on purpose. A shift ends every evening; a
            phone changes restaurant once, if ever. One button doing both would
            make the daily action cost the rare one's setup — and the rare one
            is destructive in a way the daily one is not. */}
        <Pressable
          style={[s.row, { marginTop: 8 }]}
          onPress={() => void onSignOut()}
        >
          <Feather name="log-out" size={18} color={theme.ink} />
          <Text style={[s.body, { flex: 1 }]}>{t.settings.signOut}</Text>
        </Pressable>

        <Pressable style={s.row} onPress={() => void onForgetServer()}>
          <Feather name="home" size={18} color={theme.danger} />
          <View style={{ flex: 1 }}>
            <Text style={[s.body, { color: theme.danger }]}>
              {t.settings.changeServer}
            </Text>
            <Text style={s.muted}>{t.settings.changeServerHint}</Text>
          </View>
        </Pressable>
      </ScrollView>
    </View>
  );
}

function Section({
  title,
  icon,
  children,
}: {
  title: string;
  icon: keyof typeof Feather.glyphMap;
  children: React.ReactNode;
}) {
  const { theme, s } = useUI();
  return (
    <View style={{ gap: 8 }}>
      <View style={local.sectionTitle}>
        <Feather name={icon} size={14} color={theme.muted} />
        <Text style={s.muted}>{title}</Text>
      </View>
      <View style={[s.card, { padding: 6, gap: 2 }]}>{children}</View>
    </View>
  );
}

function Choice({
  label,
  icon,
  on,
  onPress,
}: {
  label: string;
  icon?: keyof typeof Feather.glyphMap;
  on: boolean;
  onPress: () => void;
}) {
  const { theme, s } = useUI();
  return (
    <Pressable
      style={[local.choice, on ? { backgroundColor: theme.accentSoft } : null]}
      onPress={onPress}
    >
      {icon && <Feather name={icon} size={16} color={theme.muted} />}
      <Text style={[s.body, { flex: 1 }]}>{label}</Text>
      {/* ⚠️ A tick rather than colour alone: the chosen row has to be
          identifiable without relying on a wash somebody may not see. */}
      {on && <Feather name="check" size={18} color={theme.accent} />}
    </Pressable>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  const { s } = useUI();
  return (
    <View style={local.choice}>
      <Text style={[s.soft, { flex: 1 }]}>{label}</Text>
      <Text style={s.muted}>{value}</Text>
    </View>
  );
}

const local = StyleSheet.create({
  sectionTitle: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingLeft: 4,
  },
  choice: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    paddingHorizontal: 12,
    paddingVertical: 13,
    borderRadius: 12,
  },
});
