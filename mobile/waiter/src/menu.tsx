import { useMemo, useState } from "react";
import {
  FlatList,
  Image,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import Feather from "@expo/vector-icons/Feather";

import { imageUrl } from "@/lib/api";
import { contentName } from "@/lib/i18n/content";
import type { MenuGroup, MenuItem } from "@/lib/types";

import { money } from "./money";
import { usePrefs } from "./prefs";
import { useUI } from "./ui";

// The menu, as a waiter reads it.
//
// ⚠️ **The names are the restaurant's own text, and they have three versions.**
// `contentName` is the site's rule, imported unchanged: Uzbek is the base and a
// missing translation falls back to it, which is what stops a half-translated
// menu from showing empty rows. It was a rule worth importing and a wording
// worth not — see `src/i18n.ts` on why the panel's dictionary stayed behind.

export function MenuList({
  groups,
  onCheck,
  busy,
  onAdd,
  footer,
}: {
  groups: MenuGroup[];
  /** How many of each dish are already on the check. */
  onCheck: Map<string, number>;
  busy: boolean;
  onAdd: (item: MenuItem) => void;
  /** Space at the foot for the send button, which floats over this. */
  footer: number;
}) {
  const { t, lang, view, setView } = usePrefs();
  const { theme, s } = useUI();
  const [category, setCategory] = useState(0);
  const [query, setQuery] = useState("");
  const [onlyAdded, setOnlyAdded] = useState(false);

  const searching = query.trim() !== "";

  /** What to draw.
   *
   *  ⚠️ **Searching crosses categories, browsing does not.** A waiter typing a
   *  name is answering a guest and does not know or care which section it is
   *  filed under; a waiter tapping through sections is reading the menu the way
   *  it is laid out. Making search obey the selected category would hide the
   *  dish from the person who asked for it by name — which reads as the dish
   *  not existing.
   *
   *  ⚠️ **Matched on the translated name, not the base one.** A Russian-speaking
   *  waiter types what they see; searching the Uzbek text would find nothing
   *  and the menu would appear to be missing its own dishes. The base is
   *  matched too, so a name that has no translation is still reachable. */
  const items = useMemo(() => {
    const q = query.trim().toLowerCase();
    const pool = searching
      ? groups.flatMap((g) => g.items)
      : (groups[category]?.items ?? []);
    return pool.filter((it) => {
      if (onlyAdded && !onCheck.get(it.id)) return false;
      if (!q) return true;
      return (
        contentName(it, lang).toLowerCase().includes(q) ||
        it.name.toLowerCase().includes(q)
      );
    });
  }, [groups, category, query, searching, onlyAdded, onCheck, lang]);

  const views: { key: typeof view; icon: keyof typeof Feather.glyphMap }[] = [
    { key: "list", icon: "list" },
    { key: "cards", icon: "square" },
    { key: "photos", icon: "image" },
  ];

  return (
    <View style={{ flex: 1 }}>
      <View style={local.tools}>
        <View style={[local.search, { borderColor: theme.line, backgroundColor: theme.surface }]}>
          <Feather name="search" size={16} color={theme.muted} />
          <TextInput
            style={[local.searchInput, { color: theme.ink }]}
            value={query}
            onChangeText={setQuery}
            placeholder={t.menu.search}
            placeholderTextColor={theme.muted}
            autoCapitalize="none"
            autoCorrect={false}
            returnKeyType="search"
          />
          {searching && (
            <Pressable onPress={() => setQuery("")} hitSlop={10}>
              <Feather name="x" size={16} color={theme.muted} />
            </Pressable>
          )}
        </View>
        {/* ⚠️ Cycled by one button rather than three: this is a setting somebody
            changes once and then never, and three permanent controls beside a
            search box is a row of things to press instead of a menu. */}
        <Pressable
          style={[local.iconBtn, { borderColor: theme.line, backgroundColor: theme.surface }]}
          onPress={() => {
            const i = views.findIndex((v) => v.key === view);
            setView(views[(i + 1) % views.length].key);
          }}
        >
          <Feather
            name={views.find((v) => v.key === view)?.icon ?? "list"}
            size={17}
            color={theme.ink}
          />
        </Pressable>
      </View>

      {/* ⚠️ **A fixed height and no shrinking.** As a row of chips inside a
          column, this collapsed the moment the list beside it grew — so on
          exactly the categories with the most dishes, the strip naming them
          disappeared. It is the one control that says where you are. */}
      {!searching && (
        <View style={local.catsRow}>
          <FlatList
            horizontal
            data={groups}
            keyExtractor={(g) => g.category.id}
            showsHorizontalScrollIndicator={false}
            contentContainerStyle={local.cats}
            renderItem={({ item: g, index }) => (
              <Chip
                label={contentName(g.category, lang)}
                on={index === category}
                onPress={() => setCategory(index)}
              />
            )}
          />
        </View>
      )}

      <View style={local.filters}>
        {searching ? (
          <Text style={s.muted}>{t.menu.found(items.length)}</Text>
        ) : (
          <Chip
            label={t.menu.onCheck}
            icon="check"
            on={onlyAdded}
            onPress={() => setOnlyAdded((v) => !v)}
          />
        )}
      </View>

      {/* ⚠️ **FlatList, not ScrollView, and not FlashList either.** A ScrollView
          renders every child, which is what a menu of two hundred dishes with
          photographs does to a cheap Android. FlatList virtualises and is built
          in — FlashList is faster still and is a native dependency, which is a
          new build and a new thing to be wrong; it goes in when a real menu is
          measured and found wanting, not before. */}
      <FlatList
        data={items}
        keyExtractor={(it) => it.id}
        numColumns={view === "list" ? 1 : 2}
        // ⚠️ Remounts the list when the column count changes; without it React
        // Native throws rather than re-laying out.
        key={view === "list" ? "list" : "grid"}
        contentContainerStyle={[s.list, { paddingBottom: footer }]}
        columnWrapperStyle={view === "list" ? undefined : { gap: 10 }}
        // A screenful either side: enough that a thumb-flick does not reach
        // blank space, small enough that a photographed menu is not all decoded
        // at once.
        windowSize={5}
        removeClippedSubviews
        renderItem={({ item: it }) => (
          <Row
            item={it}
            count={onCheck.get(it.id) ?? 0}
            view={view}
            busy={busy}
            onAdd={onAdd}
          />
        )}
        ListEmptyComponent={
          <Text style={[s.muted, { textAlign: "center", padding: 20 }]}>
            {searching ? t.menu.nothingFound : t.check.noItems}
          </Text>
        }
      />
    </View>
  );
}

function Row({
  item,
  count,
  view,
  busy,
  onAdd,
}: {
  item: MenuItem;
  count: number;
  view: "list" | "cards" | "photos";
  busy: boolean;
  onAdd: (it: MenuItem) => void;
}) {
  const { lang } = usePrefs();
  const { theme, s } = useUI();
  const name = contentName(item, lang);
  // ⚠️ 300px wide, never the original. The server resizes on request and the
  // full-size photograph of a plate is measured in megabytes — on a
  // restaurant's connection that is the difference between a menu that opens
  // and one somebody stops using.
  const photo = view === "photos" ? imageUrl(item.imageUrl, 300) : null;

  if (view === "list") {
    return (
      <Pressable style={s.row} disabled={busy} onPress={() => onAdd(item)}>
        <Text style={[s.body, { flex: 1 }]}>{name}</Text>
        <Text style={s.num}>{money(item.price)}</Text>
        {count > 0 && <Count n={count} />}
        <Feather name="plus" size={18} color={theme.accent} />
      </Pressable>
    );
  }

  return (
    <Pressable
      style={[
        local.card,
        { backgroundColor: theme.surface, borderColor: count > 0 ? theme.accent : theme.line },
      ]}
      disabled={busy}
      onPress={() => onAdd(item)}
    >
      {view === "photos" &&
        (photo ? (
          <Image
            source={{ uri: photo }}
            style={local.photo}
            // The menu is browsed by flicking; decoding every plate at full
            // fidelity is what makes that stutter.
            resizeMode="cover"
          />
        ) : (
          // ⚠️ A named gap rather than an empty box: a restaurant that has
          // photographed half its menu should not have the other half look
          // broken.
          <View style={[local.photo, local.noPhoto, { backgroundColor: theme.surfaceAlt }]}>
            <Feather name="image" size={20} color={theme.muted} />
          </View>
        ))}
      <View style={local.cardBody}>
        <Text style={s.body} numberOfLines={2}>
          {name}
        </Text>
        <View style={local.cardFoot}>
          <Text style={s.num}>{money(item.price)}</Text>
          {count > 0 ? <Count n={count} /> : (
            <Feather name="plus" size={17} color={theme.accent} />
          )}
        </View>
      </View>
    </Pressable>
  );
}

function Count({ n }: { n: number }) {
  const { theme } = useUI();
  return (
    <View style={[local.count, { backgroundColor: theme.accent }]}>
      <Text style={[local.countText, { color: theme.onAccent }]}>{n}</Text>
    </View>
  );
}

export function Chip({
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
  const { theme } = useUI();
  return (
    <Pressable
      style={[
        local.chip,
        {
          backgroundColor: on ? theme.accentSoft : theme.surface,
          borderColor: on ? theme.accent : theme.line,
        },
      ]}
      onPress={onPress}
    >
      {icon && <Feather name={icon} size={14} color={on ? theme.accent : theme.muted} />}
      <Text
        style={{
          fontSize: 14,
          color: on ? theme.ink : theme.muted,
          fontWeight: on ? "600" : "400",
        }}
      >
        {label}
      </Text>
    </Pressable>
  );
}

const local = StyleSheet.create({
  tools: { flexDirection: "row", gap: 8, paddingHorizontal: 16, paddingTop: 10 },
  search: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 12,
    height: 44,
  },
  searchInput: { flex: 1, fontSize: 15, padding: 0 },
  iconBtn: {
    width: 44,
    height: 44,
    borderWidth: 1,
    borderRadius: 12,
    alignItems: "center",
    justifyContent: "center",
  },
  // The height is fixed on purpose — see the note at the call site.
  catsRow: { height: 52, flexShrink: 0, justifyContent: "center" },
  cats: { gap: 8, paddingHorizontal: 16 },
  filters: { paddingHorizontal: 16, paddingBottom: 4, minHeight: 26, justifyContent: "center" },
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingHorizontal: 14,
    paddingVertical: 9,
    borderRadius: 999,
    borderWidth: 1,
  },
  card: { flex: 1, borderWidth: 1, borderRadius: 16, overflow: "hidden" },
  photo: { width: "100%", height: 104 },
  noPhoto: { alignItems: "center", justifyContent: "center" },
  cardBody: { padding: 12, gap: 8, flex: 1, justifyContent: "space-between" },
  cardFoot: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  count: {
    minWidth: 24,
    height: 24,
    borderRadius: 12,
    paddingHorizontal: 7,
    alignItems: "center",
    justifyContent: "center",
  },
  countText: { fontSize: 13, fontWeight: "700" },
});
