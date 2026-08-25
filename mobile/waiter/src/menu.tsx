import { memo, useCallback, useMemo, useState } from "react";
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
import { Stepper } from "./stepper";
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
  onRemove,
  footer,
}: {
  groups: MenuGroup[];
  /** How many of each dish are already on the check. */
  onCheck: Map<string, number>;
  busy: boolean;
  onAdd: (item: MenuItem) => void;
  /** Take one off. ⚠️ Needed here and not only on the check: a waiter who has
   *  just tapped one too many is looking at the menu, and sending them to
   *  another tab to undo a tap they made a second ago is how a wrong count
   *  survives to the kitchen. */
  onRemove: (item: MenuItem) => void;
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
        // ⚠️ Batched and capped: without these React Native renders as much as
        // it can between frames, which on a cheap Android is the pause people
        // feel as the list "catching up" after a flick.
        initialNumToRender={12}
        maxToRenderPerBatch={8}
        updateCellsBatchingPeriod={50}
        removeClippedSubviews
        // ⚠️ `renderItem` is defined per render either way; what has to be
        // stable are the props inside it. `onAdd` and `onRemove` come from the
        // check screen and do not change, and `count` is a number — so a row
        // whose dish and count are unchanged does not redraw.
        renderItem={({ item: it }) => (
          <Row
            item={it}
            count={onCheck.get(it.id) ?? 0}
            view={view}
            busy={busy}
            onAdd={onAdd}
            onRemove={onRemove}
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

/** ⚠️ **Memoised, and this is the scroll stutter.** Every keystroke in the
 *  search box and every reply from the server re-rendered the whole list —
 *  two hundred rows, each decoding a photograph. `memo` means a row is redrawn
 *  only when its own dish or its own count changes. */
const Row = memo(function Row({
  item,
  count,
  view,
  busy,
  onAdd,
  onRemove,
}: {
  item: MenuItem;
  count: number;
  view: "list" | "cards" | "photos";
  busy: boolean;
  onAdd: (it: MenuItem) => void;
  onRemove: (it: MenuItem) => void;
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
    // ⚠️ **The row stops being pressable once there is a stepper on it.** Two
    // ways to add one dish — the row and the plus — differ by a few pixels and
    // by one, and the difference is only discovered at the table.
    const Wrapper = count > 0 ? View : Pressable;
    return (
      <Wrapper
        style={s.row}
        {...(count > 0 ? {} : { disabled: busy, onPress: () => onAdd(item) })}
      >
        <Text style={[s.body, { flex: 1 }]}>{name}</Text>
        <Text style={s.num}>{money(item.price)}</Text>
        {count > 0 ? (
          <Stepper
            value={count}
            disabled={busy}
            removeAtZero
            onMinus={() => onRemove(item)}
            onPlus={() => onAdd(item)}
          />
        ) : (
          <Pressable
            style={[local.plus, { borderColor: theme.line }]}
            disabled={busy}
            hitSlop={6}
            onPress={() => onAdd(item)}
          >
            <Feather name="plus" size={18} color={theme.accent} />
          </Pressable>
        )}
      </Wrapper>
    );
  }

  return (
    <Pressable
      style={[
        local.card,
        { backgroundColor: theme.surface, borderColor: count > 0 ? theme.accent : theme.line },
      ]}
      disabled={busy || count > 0}
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
          {/* ⚠️ The price shrinks rather than the stepper overflowing: on a
              card there is one row of space and the control has to fit in it,
              not past it. */}
          <Text style={[s.num, { fontSize: 13 }]} numberOfLines={1}>
            {money(item.price)}
          </Text>
          {count > 0 ? (
            <Stepper
              value={count}
              disabled={busy}
              removeAtZero
              compact
              onMinus={() => onRemove(item)}
              onPlus={() => onAdd(item)}
            />
          ) : (
            <Pressable
              style={[local.plusSmall, { borderColor: theme.line }]}
              disabled={busy}
              hitSlop={10}
              onPress={() => onAdd(item)}
            >
              <Feather name="plus" size={16} color={theme.accent} />
            </Pressable>
          )}
        </View>
      </View>
    </Pressable>
  );
});

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
  // ⚠️ Air between the three rows. They were flush against one another, which
  // reads as one control that has gone wrong rather than as three that work.
  tools: { flexDirection: "row", gap: 10, paddingHorizontal: 16, paddingTop: 12 },
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
  catsRow: { height: 56, flexShrink: 0, justifyContent: "center", marginTop: 4 },
  cats: { gap: 8, paddingHorizontal: 16 },
  filters: {
    paddingHorizontal: 16,
    paddingTop: 4,
    paddingBottom: 8,
    minHeight: 30,
    justifyContent: "center",
  },
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
  cardFoot: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 6,
  },
  plusSmall: {
    width: 30,
    height: 30,
    borderRadius: 9,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
  plus: {
    width: 40,
    height: 40,
    borderRadius: 12,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center",
  },
});
