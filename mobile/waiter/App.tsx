import { useEffect, useState } from "react";
import { StatusBar } from "expo-status-bar";
import { StyleSheet, Text, View } from "react-native";

import { qtyNumber } from "@/lib/qty";
import { serviceOn } from "@/lib/offline/checks";
import { checkMark, normalizeMark } from "@/lib/marking";

import { hydrateTokens } from "./src/tokens";

// The first screen, and it exists to prove one thing: the rules the counter
// runs on are the rules this phone runs on — the same files, not a copy.
//
// ⚠️ It is deliberately arithmetic rather than a table of data. A screen that
// fetched would prove the network works; what has to be proven first is that a
// module written for a browser evaluates here at all, because that is the
// question the whole approach rests on.

export default function App() {
  const [ready, setReady] = useState(false);

  // ⚠️ Before anything is fetched, not alongside it: a request made while the
  // token store was still the browser's would read every token as absent and
  // send somebody to a login they had already passed.
  useEffect(() => {
    void hydrateTokens().then(() => setReady(true));
  }, []);

  return (
    <View style={styles.screen}>
      <Text style={styles.title}>Keel Waiter</Text>
      <Text style={styles.line}>9,4 → {qtyNumber("9,4")}</Text>
      <Text style={styles.line}>
        216 000 + 10% xizmat → {serviceOn(216000, 10)}
      </Text>
      <Text style={styles.line}>
        shtrix-kod markirovka emasmi →{" "}
        {checkMark(normalizeMark("4607034170203")) ?? "—"}
      </Text>
      <Text style={styles.muted}>
        {ready ? "tokenlar o'qildi" : "o'qilmoqda…"}
      </Text>
      <StatusBar style="auto" />
    </View>
  );
}

const styles = StyleSheet.create({
  screen: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "#faf7f2",
    gap: 8,
  },
  title: { fontSize: 22, fontWeight: "600", marginBottom: 8 },
  line: { fontSize: 15 },
  muted: { fontSize: 13, color: "#8a8178", marginTop: 12 },
});
