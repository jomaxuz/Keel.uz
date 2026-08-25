import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";

import { api, ApiError } from "@/lib/api";
import type { Check, FloorTable, Staff } from "@/lib/types";

import { money } from "./money";
import { theme } from "./theme";

// The three screens the first slice needs, in the order somebody meets them.
//
// ⚠️ **Deliberately plain.** What this slice has to answer is whether the rules
// the counter runs on give the same answers here, against a real server — not
// what the app will look like. Drawing it properly before that is answered is
// work that might be thrown away.

// ---- Which restaurant ----

export function ServerScreen({
  onChosen,
}: {
  onChosen: (address: string) => boolean;
}) {
  const [address, setAddress] = useState("");
  const [bad, setBad] = useState(false);

  return (
    <View style={styles.centered}>
      <Text style={styles.title}>Keel Waiter</Text>
      <Text style={styles.muted}>Restoran manzili</Text>
      <TextInput
        style={styles.input}
        value={address}
        onChangeText={(v) => {
          setBad(false);
          setAddress(v);
        }}
        placeholder="osh"
        autoCapitalize="none"
        autoCorrect={false}
        keyboardType="url"
      />
      {/* ⚠️ The short form is the example, because it is what somebody knows.
          A placeholder showing "https://osh.keel.uz/api/v1" would teach the
          wrong answer to everybody who reads it. */}
      <Text style={styles.hint}>
        Restoraningizning qisqa nomi yoki to&apos;liq manzili
      </Text>
      {bad && <Text style={styles.error}>Bu manzilga o&apos;xshamaydi</Text>}
      <Pressable
        style={styles.button}
        onPress={() => {
          if (!onChosen(address)) setBad(true);
        }}
      >
        <Text style={styles.buttonText}>Davom etish</Text>
      </Pressable>
    </View>
  );
}

// ---- Who is signing in ----

export function LoginScreen({
  address,
  onSignIn,
  onForget,
}: {
  address: string;
  onSignIn: (username: string, password: string) => Promise<void>;
  onForget: () => void;
}) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    setBusy(true);
    setError("");
    try {
      await onSignIn(username.trim(), password);
    } catch (e) {
      // ⚠️ The server's own words. It distinguishes a wrong password from a
      // switched-off account, and those send somebody to two different people.
      setError(e instanceof ApiError ? e.message : "Kirib bo'lmadi");
      setBusy(false);
    }
  }

  return (
    <View style={styles.centered}>
      <Text style={styles.title}>Kirish</Text>
      <Text style={styles.muted}>{address}</Text>
      <TextInput
        style={styles.input}
        value={username}
        onChangeText={setUsername}
        placeholder="Login"
        autoCapitalize="none"
        autoCorrect={false}
      />
      <TextInput
        style={styles.input}
        value={password}
        onChangeText={setPassword}
        placeholder="Parol"
        secureTextEntry
        autoCapitalize="none"
      />
      {error !== "" && <Text style={styles.error}>{error}</Text>}
      <Pressable style={styles.button} disabled={busy} onPress={submit}>
        <Text style={styles.buttonText}>{busy ? "…" : "Kirish"}</Text>
      </Pressable>
      <Pressable onPress={onForget}>
        <Text style={styles.link}>Boshqa restoran</Text>
      </Pressable>
    </View>
  );
}

// ---- The room ----

export function TablesScreen({
  staff,
  onSignOut,
  onOpenCheck,
}: {
  staff: Staff;
  onSignOut: () => void;
  /** Where a tapped table goes. ⚠️ Opening a check is the *start* of the job:
   *  a screen that opened one and stayed put is a table that does nothing. */
  onOpenCheck: (checkId: string, branchId: string) => void;
}) {
  const [tables, setTables] = useState<FloorTable[] | null>(null);
  const [checks, setChecks] = useState<Check[]>([]);
  const [branch, setBranch] = useState("");
  const [branchId, setBranchId] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");

  const load = useCallback(async () => {
    try {
      const [b, c] = await Promise.all([api.tillBranch(), api.tillChecks()]);
      setBranch(b.name);
      setBranchId(b.id);
      setTables(b.booking?.tables ?? []);
      setChecks(c.checks);
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Yuklab bo'lmadi");
      setTables([]);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  /** Which table each open check is sitting at.
   *
   *  ⚠️ Built once per change rather than searched per tile: a room has forty
   *  tables and this is the kind of scan that is invisible until somebody is
   *  scrolling on a four-year-old phone. */
  const byTable = useMemo(() => {
    const m = new Map<string, Check>();
    for (const c of checks) if (c.tableId) m.set(c.tableId, c);
    return m;
  }, [checks]);

  async function open(table: FloorTable) {
    // ⚠️ An occupied table is opened, not refused: the whole reason a waiter
    // taps a table that already has a check is to add to it.
    const existing = byTable.get(table.id);
    if (existing) {
      onOpenCheck(existing.id, branchId);
      return;
    }
    setBusy(table.id);
    try {
      const check = await api.tillOpenCheck({ tableId: table.id, guests: 0 });
      onOpenCheck(check.id, branchId);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Ochib bo'lmadi");
    } finally {
      setBusy("");
    }
  }

  if (tables === null) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator />
      </View>
    );
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <View>
          <Text style={styles.headerTitle}>{branch || "Zal"}</Text>
          <Text style={styles.muted}>{staff.name}</Text>
        </View>
        <Pressable onPress={onSignOut}>
          <Text style={styles.link}>Chiqish</Text>
        </Pressable>
      </View>

      {error !== "" && <Text style={styles.error}>{error}</Text>}

      {/* ⚠️ A ScrollView is right for this slice and wrong for the app: it
          renders every child. It stays until there is a real room to measure,
          and then becomes a FlashList — the note is here so the next person
          meets the decision rather than the oversight. */}
      <ScrollView contentContainerStyle={styles.grid}>
        {tables.map((t) => {
          const check = byTable.get(t.id);
          return (
            <Pressable
              key={t.id}
              style={[styles.table, check ? styles.tableBusy : null]}
              disabled={busy === t.id}
              onPress={() => void open(t)}
            >
              <Text style={styles.tableNumber}>{t.number}</Text>
              <Text style={styles.tableNote}>
                {check ? money(check.total) : "bo'sh"}
              </Text>
            </Pressable>
          );
        })}
        {tables.length === 0 && (
          <Text style={styles.muted}>Bu filialda stol qo&apos;shilmagan</Text>
        )}
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: theme.bg },
  centered: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    padding: 24,
    gap: 10,
    backgroundColor: theme.bg,
  },
  title: { fontSize: 22, fontWeight: "600", color: theme.ink },
  muted: { fontSize: 13, color: theme.muted },
  hint: { fontSize: 12, color: theme.muted, textAlign: "center" },
  error: { fontSize: 13, color: theme.danger, textAlign: "center" },
  link: { fontSize: 14, color: theme.accent },
  input: {
    width: "100%",
    maxWidth: 320,
    borderWidth: 1,
    borderColor: theme.line,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 16,
    backgroundColor: theme.surface,
    color: theme.ink,
  },
  button: {
    marginTop: 4,
    backgroundColor: theme.accent,
    borderRadius: 12,
    paddingHorizontal: 24,
    paddingVertical: 12,
  },
  buttonText: { color: "#fff", fontSize: 16, fontWeight: "600" },
  header: {
    paddingTop: 56,
    paddingHorizontal: 16,
    paddingBottom: 12,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
  },
  headerTitle: { fontSize: 18, fontWeight: "600", color: theme.ink },
  grid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 10,
    padding: 16,
  },
  table: {
    width: 96,
    height: 96,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: theme.line,
    backgroundColor: theme.surface,
    alignItems: "center",
    justifyContent: "center",
    gap: 4,
  },
  // ⚠️ Not colour alone: an occupied table also says what it owes. Green
  // against red is a distinction roughly one man in twelve cannot make, and
  // the lock screen already learned this.
  tableBusy: { borderColor: theme.accent, backgroundColor: theme.accentSoft },
  tableNumber: { fontSize: 20, fontWeight: "600", color: theme.ink },
  tableNote: { fontSize: 12, color: theme.muted },
});
