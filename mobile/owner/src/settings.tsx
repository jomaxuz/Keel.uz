import { useEffect, useState } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
// ⚠️ **From the family's own path, not the package index.** The index
// re-exports every icon set it ships — AntDesign, MaterialIcons, Ionicons
// and a dozen more — and each carries a glyph map, so importing one name
// from it pulls all of them into the bundle. Measured on these exact
// screens: 2.0 MB and 688 modules from the index, 1.6 MB and 634 from here.
import Feather from "@expo/vector-icons/Feather";
import * as Application from "expo-application";

import { api } from "@/lib/api";
import type { AdminUser, Branch } from "@/lib/types";

import { money } from "./money";

import { LANGS, DICTS, type Lang } from "./i18n";
import type { PushState } from "./push";
import { usePrefs, type ThemeChoice } from "./prefs";
import { useUI } from "./ui";

// Language, appearance, and the two ways out.

export function SettingsScreen({
  admin,
  branches,
  branchId,
  onBranch,
  address,
  onSignOut,
  onForgetServer,
  pushState,
  onRetryPush,
}: {
  admin: AdminUser;
  /** ⚠️ The lens lives here as well as on the first screen: an owner who set
   *  it while looking at today's takings expects the reports to follow, and a
   *  second place to change it is how the two screens end up disagreeing. */
  branches: Branch[];
  branchId: string;
  onBranch: (id: string) => void;
  address: string;
  /** ⚠️ Async, because the phone's push registration is dropped first: a token
   *  left behind sends the next evening's tables to whoever went home. */
  onSignOut: () => void | Promise<void>;
  onForgetServer: () => void | Promise<void>;
  /** ⚠️ Shown rather than hidden: "the kitchen pressed ready and nothing
   *  arrived" has five possible causes, and without this there is no way to
   *  tell them apart from the phone it happened on. */
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
        <Text style={s.h2}>{t.settings.title}</Text>
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

        <Section title={t.settings.notifications} icon="bell">
          <View style={local.choice}>
            <Feather
              name={pushState === "working" ? "check-circle" : "alert-circle"}
              size={16}
              color={pushState === "working" ? theme.accent : theme.danger}
            />
            <View style={{ flex: 1 }}>
              <Text style={s.body}>{t.settings.push[pushState]}</Text>
              <Text style={s.muted}>{t.settings.pushHint[pushState]}</Text>
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

        {branches.length > 1 && (
          <Section title={t.settings.branch} icon="map-pin">
            <Choice
              label={t.today.branchAll}
              on={branchId === ""}
              onPress={() => onBranch("")}
            />
            {branches.map((b) => (
              <Choice
                key={b.id}
                label={b.name}
                on={branchId === b.id}
                onPress={() => onBranch(b.id)}
              />
            ))}
          </Section>
        )}

        <PlanSection />

        <Section title={t.settings.account} icon="user">
          <Row label={admin.name || admin.username} value={admin.role} />
          <Row label={t.settings.restaurant} value={address} />
          <Row
            label={t.settings.version}
            value={Application.nativeApplicationVersion ?? "—"}
          />
        </Section>

        {/* ⚠️ Two ways out, kept apart on purpose. A shift ends every evening;
            a phone changes restaurant once, if ever. One button doing both
            would make the daily action cost the rare one's setup — and the
            rare one is destructive in a way the daily one is not. */}
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

/** What this restaurant pays us, and when the next payment is due.
 *
 *  ⚠️ **A date and a countdown, never a flag.** "Subscription active" goes
 *  stale at midnight with nobody watching — the lesson this codebase already
 *  wrote down about `provisionStatus`. The date cannot go stale, and the
 *  countdown is the half that changes colour.
 *
 *  ⚠️ **It states, it never nags.** The till already warns in the final week
 *  and the panel carries the full card; a third escalating warning, on the
 *  phone, is how an owner learns to ignore all three. It is here because "when
 *  do I pay" is asked away from the desk, by the one person who pays.
 *
 *  ⚠️ **No plan is an ordinary state.** A restaurant paying per order has no
 *  counter and never will, and telling that owner something is wrong would be
 *  false.
 */
function PlanSection() {
  const { t } = usePrefs();
  const { theme, s } = useUI();
  const [sub, setSub] = useState<Awaited<
    ReturnType<typeof api.subscription>
  > | null>(null);

  useEffect(() => {
    let alive = true;
    api
      .subscription()
      .then((r) => alive && setSub(r))
      // Silent: this is an addition to the screen, and the language and theme
      // above it work with or without an answer.
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, []);

  if (!sub) return null;

  if (!sub.enabled || !sub.plan) {
    return (
      <Section title={t.settings.plan} icon="credit-card">
        <View style={local.choice}>
          <View style={{ flex: 1 }}>
            <Text style={s.body}>{t.settings.planNone}</Text>
            <Text style={s.muted}>{t.settings.planNoneHint}</Text>
          </View>
        </View>
      </Section>
    );
  }

  const left = sub.paidUntil ? daysUntil(sub.paidUntil) : null;
  // ⚠️ Overdue is said in days rather than as "expired": how far past it is
  // decides whether this is a note to self or a call this morning.
  let due = t.settings.planNoDate;
  let tone = theme.muted;
  if (left !== null) {
    if (left < 0) {
      due = t.settings.planOverdue(-left);
      tone = theme.danger;
    } else if (left === 0) {
      due = t.settings.planDueToday;
      tone = theme.danger;
    } else {
      due = t.settings.planDaysLeft(left);
      tone = left <= 7 ? theme.warn : theme.muted;
    }
  }

  return (
    <Section title={t.settings.plan} icon="credit-card">
      <View style={local.choice}>
        <View style={{ flex: 1 }}>
          {/* The rung's own name, as it is written on the invoice. */}
          <Text style={s.body}>{planName(sub.plan)}</Text>
          {sub.paidUntil ? (
            <Text style={s.muted}>{t.settings.planUntil(sub.paidUntil)}</Text>
          ) : null}
        </View>
        <Text style={[s.muted, { color: tone }]}>{due}</Text>
      </View>
      <View style={local.choice}>
        <Text style={[s.soft, { flex: 1 }]}>{t.settings.planMonthly}</Text>
        {/* ⚠️ Zero is "agreed separately", never "free": an Enterprise price is
            settled per customer, and printing "0 so'm" would be a quote nobody
            gave. */}
        <Text style={s.muted}>
          {(sub.monthly ?? 0) > 0
            ? money(sub.monthly ?? 0)
            : t.settings.planIndividual}
        </Text>
      </View>
    </Section>
  );
}

/** ⚠️ The rungs are proper names — Start, Standard, Pro, Enterprise — so they
 *  are not translated, and an unknown id prints itself rather than nothing: the
 *  console may sell a rung this build has never heard of. */
function planName(id: string): string {
  return (
    { start: "Start", standard: "Standard", pro: "Pro", enterprise: "Enterprise" }[
      id
    ] ?? id
  );
}

/** Whole days from today to a "YYYY-MM-DD" the server already localised.
 *
 *  ⚠️ Both sides are pinned to midnight UTC from the date parts, so this is a
 *  difference of calendar days and never of hours — and the string is already
 *  local, so no timezone conversion belongs here. A countdown that ticks over
 *  at four in the afternoon because that is when somebody paid is a countdown
 *  nobody believes. */
function daysUntil(date: string): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!m) return null;
  const end = Date.UTC(+m[1], +m[2] - 1, +m[3]);
  const now = new Date();
  const today = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate());
  return Math.round((end - today) / 86_400_000);
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
      style={[
        local.choice,
        on ? { backgroundColor: theme.accentSoft } : null,
      ]}
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
