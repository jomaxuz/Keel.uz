"use client";

import { useMemo, useState } from "react";
// One icon at a time (`react-icons/lu`): the top-level entry point is an index
// of several thousand.
import { LuLayoutGrid, LuList, LuSearch, LuUsers, LuX } from "react-icons/lu";

import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { Lang } from "@/lib/i18n/dictionaries";
import type {
  Check,
  FloorShape,
  FloorTable,
  TableZone,
} from "@/lib/types";

import { LATE_MIN, tableState } from "@/lib/tableState";

import TableObject from "./TableObject";
import TillFloorPlan from "./TillFloorPlan";

/** The three questions a room gets asked, and the three ways of answering. */
type Mode = "plan" | "grid" | "waiters";

/**
 * The room, and it is the first thing anybody sees after their PIN.
 *
 * ⚠️ **A till opens on the floor, not on the menu.** The first question of
 * every order is "who is this for" — a table, or somebody at the counter — and
 * a screen that starts with dishes makes that question something you answer
 * afterwards, by remembering. That is how a round of drinks ends up on the
 * wrong bill.
 *
 * ⚠️ **Occupied and free tables are on the same grid**, distinguished by what
 * they say rather than by being hidden. A waiter who cannot find table 7
 * assumes the screen is broken and opens a counter check instead, which is how
 * one party ends up on two bills. An occupied tile carries the two numbers
 * worth knowing at a glance: what the table owes and how long it has been
 * sitting.
 *
 * ⚠️ **The counter tile is first-class, not a fallback.** Half the places that
 * would buy this sell over a counter, and a till that insists on a table
 * number is a till they cannot use.
 *
 * ⚠️ **Zones are tabs, and only when there is more than one.** A takeaway
 * counter numbers its checks 100–130, and thirty numbers poured into the hall's
 * grid make table 7 something you scroll for. But a restaurant with one room
 * has one zone and must not be handed a tab strip with a single tab — the same
 * rule as the brand switcher, which is not drawn for a single brand.
 */
export default function TablesScreen({
  tables,
  zones,
  shapes,
  planWidth,
  planHeight,
  checks,
  currency,
  onOpenCheck,
  onNewCheck,
}: {
  tables: FloorTable[];
  /** Empty on a restaurant that has not split its room, which is most of them. */
  zones: TableZone[];
  /** Walls and named areas, as drawn in the panel. */
  shapes: FloorShape[];
  planWidth: number;
  planHeight: number;
  checks: Check[];
  currency: string;
  /** An existing check was tapped. */
  onOpenCheck: (check: Check) => void;
  /** A free table (or the counter) was tapped — open a new one. */
  onNewCheck: (tableId: string) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [zone, setZone] = useState("");
  const [waiter, setWaiter] = useState("");
  /** What is being looked for. ⚠️ Kept here rather than in the grid: the same
   *  question is asked of the plan and of the waiter cards, and a box that
   *  emptied itself when the view changed would be a box that has to be typed
   *  into twice. */
  const [query, setQuery] = useState("");

  const byTable = new Map<string, Check>();
  const counter: Check[] = [];
  for (const c of checks) {
    if (c.tableId) byTable.set(c.tableId, c);
    else counter.push(c);
  }

  const activeAll = useMemo(() => tables.filter((tb) => tb.isActive), [tables]);

  // ⚠️ **The counter's numbers come from the settings, like everything else
  // about the room.** A zone marked "list" in Sozlamalar → Stol bron qilish is
  // exactly that: a takeaway counter numbering its orders 100–130, seats and
  // coordinates never filled in because there is no table to sit at. Those
  // numbers belong beside the counter, not scattered through the hall's grid —
  // and never on the floor plan, where they would all pile up at 0,0.
  const listZoneIDs = useMemo(
    () => new Set(zones.filter((z) => z.layout === "list").map((z) => z.id)),
    [zones],
  );
  // ⚠️ **Every table the owner made, whatever kind of zone it is in.** The
  // counter's numbers used to be split off here into a scrolling strip above
  // the room, which had three consequences and all of them were bugs: the zone
  // they belong to never got a tab (the tab strip is built from this list), its
  // name was never printed anywhere, and `isList` below could never be true —
  // so the branch written to draw a counter zone was dead code. A restaurant
  // that made a hall and a counter saw neither name, and concluded the zones it
  // had just set up had not saved.
  const active = activeAll;
  // The tables that carry no shape, only a number — checked when deciding
  // whether the floor plan is worth offering.
  const planTables = useMemo(
    () => activeAll.filter((tb) => !listZoneIDs.has(tb.zoneId ?? "")),
    [activeAll, listZoneIDs],
  );

  // ⚠️ **The plan is only offered when there is one.** Coordinates default to
  // zero, so a restaurant that filled in table numbers and never opened the
  // floor-plan editor has every table stacked in the top-left corner — a room
  // that reads as broken. Those branches get the grid and never see the tab.
  const hasPlan = useMemo(
    () => planTables.some((tb) => tb.x !== 0 || tb.y !== 0) || shapes.length > 0,
    [planTables, shapes],
  );
  // ⚠️ **Derived, not stored-then-corrected.** The plan is the better first
  // screen where one exists — it is the room the person is standing in rather
  // than a list of its names — but whether one exists is only known once the
  // profile has loaded. An effect that flipped the view after mount swapped the
  // grid out from under a finger that was already on its way down, and the tap
  // landed on a tile being replaced. So the view is a pure function of the data
  // and the one choice the user has actually made.
  const [picked, setPicked] = useState<Mode | null>(null);
  const chosen: Mode = picked ?? (hasPlan ? "plan" : "grid");

  /** The tabs, in the order the owner arranged them.
   *
   *  ⚠️ A zone with no tables is not drawn: it is a half-finished setting, and
   *  an empty tab teaches a cashier that tabs can be empty — after which they
   *  stop trusting the one that is merely scrolled. */
  const tabs = useMemo(() => {
    const used = new Set(active.map((tb) => tb.zoneId ?? ""));
    const list = zones
      .filter((z) => used.has(z.id))
      .slice()
      .sort((a, b) => a.sort - b.sort);
    // ⚠️ Tables drawn before zones existed carry no id, and they are the whole
    // room at every restaurant upgrading into this. They get the first tab,
    // named for what it is rather than left blank.
    if (used.has("")) {
      list.unshift({ id: "", name: t.till.tables, bookable: true, sort: -1 });
    }
    return list;
  }, [active, zones, t.till.tables]);

  const current = tabs.some((z) => z.id === zone) ? zone : (tabs[0]?.id ?? "");
  // A list zone is a wall of numbers with no shape: seats and coordinates were
  // never filled in, because there is no table to sit at.
  const isList = tabs.find((z) => z.id === current)?.layout === "list";
  // ⚠️ **A counter zone forces the grid.** Its tables sit at 0,0 — that is what
  // makes it a list — so the plan would stack thirty numbered squares in the
  // top-left corner. The tab is not taken away from the person, it simply
  // cannot answer "where is 112" because nobody drew it anywhere.
  const view: Mode = isList && chosen === "plan" ? "grid" : chosen;
  const setMode = setPicked;
  const q = query.trim().toLowerCase();
  /** ⚠️ **A search crosses the zones.** Somebody hunting for table 27 does not
   *  know which room the owner filed it under — that is the whole reason they
   *  are searching — so the tabs stop filtering while the box has something in
   *  it, and the header says how many were found. */
  const shown = q
    ? active.filter((tb) => matches(tb, byTable.get(tb.id), q))
    : active.filter((tb) => (tb.zoneId ?? "") === current);

  /** Who is serving what, for the waiter view.
   *
   *  ⚠️ **Counted from the checks, not from a staff list.** The question this
   *  view answers is "who is holding tables right now", and a waiter with none
   *  is not part of it — while a name that appears on a check but has since
   *  been deactivated very much is. */
  const waiters = useMemo(() => {
    const map = new Map<string, { name: string; count: number; sum: number }>();
    for (const c of checks) {
      const name = c.serverName || "—";
      const row = map.get(name) ?? { name, count: 0, sum: 0 };
      row.count += 1;
      row.sum += c.total;
      map.set(name, row);
    }
    return [...map.values()].sort((a, b) => b.sum - a.sum);
  }, [checks]);

  const waiterChecks = (
    waiter ? checks.filter((c) => (c.serverName || "—") === waiter) : checks
  ).filter(
    (c) =>
      !q ||
      (c.tableNumber || "").toLowerCase().includes(q) ||
      (c.serverName || "").toLowerCase().includes(q),
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* ---- The bar over the room ----

          ⚠️ **Two groups, not one row of buttons.** On the left, how to look at
          the room; on the right, the two things you do to it — find a table, or
          start a check that has no table. They used to be interleaved, and
          "+ Peshtaxta" sat a finger's width from the view switcher, which is
          how a waiter reaching for "Ro'yxat" opened a counter check instead.

          ⚠️ Three views of one fact, because three different questions get
          asked of it: "where is table 7" (the plan), "which tables are free"
          (the grid), and "what is on table 7 without walking to it" (the
          cards). A till that only draws the plan makes the third question a
          walk across the room.

          ⚠️ **Underlined tabs for the views, pills for the zones.** They were
          both segmented pills, side by side — two identical controls, one
          choosing a way of looking and one choosing a room, and the shape said
          nothing about which was which until both had been read. */}
      {/* ⚠️ **One row, and it may not wrap.** On the 1024px monoblock the room
          gets about 600px of bar once the rail and the check panel have taken
          theirs, and `flex-wrap` answered that by dropping "+ Peshtaxta" onto a
          second line — which pushed the room down and moved the most-pressed
          button on this screen somewhere it had never been. Everything here
          shrinks instead, and the search is the piece that gives way first. */}
      <div className="flex shrink-0 items-center gap-x-1 border-b border-line bg-surface px-3">
        <div className="flex min-w-0 items-center">
          {hasPlan && !isList && (
            <ViewTab
              on={view === "plan"}
              onClick={() => setMode("plan")}
              icon={<LuLayoutGrid className="h-4 w-4" aria-hidden />}
              label={t.till.planView}
            />
          )}
          <ViewTab
            on={view === "grid"}
            onClick={() => setMode("grid")}
            icon={<LuList className="h-4 w-4" aria-hidden />}
            label={t.till.gridView}
          />
          <ViewTab
            on={view === "waiters"}
            onClick={() => setMode("waiters")}
            icon={<LuUsers className="h-4 w-4" aria-hidden />}
            label={t.till.waiterView}
          />
        </div>

        {/* Zones filter the room, so they belong beside the room views and not
            beside the card view, which is grouped by person instead. */}
        {view !== "waiters" && tabs.length > 1 && (
          <div className="till-seg-track no-scrollbar my-2 ml-2 overflow-x-auto">
            {tabs.map((z) => {
              const busy = active.filter(
                (tb) => (tb.zoneId ?? "") === z.id && byTable.has(tb.id),
              ).length;
              return (
                <button
                  key={z.id}
                  onClick={() => setZone(z.id)}
                  className={z.id === current ? "till-seg-on" : "till-seg"}
                >
                  {z.name}
                  {busy > 0 && (
                    <span className="rounded-full bg-ink/[0.06] px-1.5 text-[11px] font-bold text-ink-muted">
                      {busy}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        )}

        <div className="flex-1" />

        {/* ---- Finding one table ----

            ⚠️ **A number, not a name.** In a forty-table room the question is
            always "where is 27", asked by somebody holding a bill with 27 on
            it, and the tile they want is two screens down a grid they are
            scrolling one-handed. The waiter's name matches too, because the
            other half of that question is "which of these are mine".

            ⚠️ **It narrows the room, it does not empty it.** An empty box
            leaves every table drawn — a search that starts by clearing the
            screen is a search that gets used once. */}
        <TableSearch value={query} onChange={setQuery} />

        {/* ⚠️ **The counter is first-class, not a fallback.** Half the places
            that would buy this sell over a counter, and a till that insists on
            a table number is a till they cannot use. It is reachable from every
            view, because "one coffee to take away" arrives while you are
            looking at whatever you happen to be looking at. */}
        <button
          className="till-btn-accent my-2 ml-1.5 shrink-0"
          onClick={() => onNewCheck("")}
        >
          + {t.till.counter}
        </button>
      </div>

      {/* ---- Checks with no table at all ----

          ⚠️ **Only the anonymous ones now.** The numbered counter slots used to
          be pinned up here too, which is what hid their zone: they were taken
          out of the table list, so the zone had no tab, no name, and the code
          written to draw it could never run. They live in their own zone tab
          with every other table; what is left here is the check somebody opened
          with "+ Peshtaxta" — it belongs to nobody's table, so there is nowhere
          else it can go. */}
      {counter.length > 0 && (
        <div className="flex shrink-0 gap-2 overflow-x-auto border-b border-line px-3 py-2">
          {counter.map((c, i) => (
            <button
              key={c.id}
              onClick={() => onOpenCheck(c)}
              className="till-tile min-w-[9rem] shrink-0 justify-between p-2.5"
            >
              <span className="flex items-baseline justify-between gap-2">
                <span className="text-[15px] font-bold">#{i + 1}</span>
                <span className="till-num text-[11px] text-[rgb(var(--till-dim))]">
                  {c.openMin} {t.till.minShort}
                </span>
              </span>
              <span className="till-num text-[15px] font-bold">
                {formatPrice(c.total, currency, lang)}
              </span>
            </button>
          ))}
        </div>
      )}

      {/* ---- The room ---- */}
      {view === "plan" && (
        // The same tinted ground as the grid: the tables are white objects, and
        // a white room turns them into edges rather than shapes.
        <div className="min-h-0 flex-1 overflow-auto bg-[rgb(var(--till-floor))] p-3">
          <TillFloorPlan
            width={planWidth || 1000}
            height={planHeight || 700}
            shapes={shapes}
            tables={shown}
            byTable={byTable}
            currency={currency}
            onPick={(tb, check) =>
              check ? onOpenCheck(check) : onNewCheck(tb.id)
            }
          />
        </div>
      )}

      {view === "grid" && (
        // ⚠️ **A tinted ground, not the page's cream.** The tables are white
        // objects and a white room makes them edges rather than shapes — which
        // is the whole difference between scanning a floor and reading a list.
        <div className="min-h-0 flex-1 overflow-y-auto bg-[rgb(var(--till-floor))] p-3">
          {/* ⚠️ Wider columns than the old card grid. A drawn table is a wide
              object — chairs, a body, a badge — and squeezing eight into a
              1024px monoblock produced a row of grey slots with unreadable
              numbers in them. */}
          {/* ⚠️ **A counter packs tighter than a room.** Its slots are numbers,
              not furniture — no chairs, nothing to walk between — so they sit
              close and in more columns. A hall drawn at the same density loses
              the space that makes the tables read as separate objects. */}
          <div
            className={
              isList
                ? "grid grid-cols-4 gap-1.5 md:grid-cols-6 xl:grid-cols-8 2xl:grid-cols-10"
                : "grid grid-cols-3 gap-x-2 gap-y-1 md:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-7"
            }
          >
            {shown.map((tb) => {
              const c = byTable.get(tb.id);
              return (
                <TableObject
                  key={tb.id}
                  label={tb.number}
                  seats={isList ? 0 : tb.seats}
                  compact={isList}
                  check={c}
                  currency={currency}
                  onClick={() => (c ? onOpenCheck(c) : onNewCheck(tb.id))}
                />
              );
            })}
          </div>
          {/* ⚠️ Said rather than left blank: a restaurant that has not drawn
              its room yet sees an empty screen and concludes the till is
              broken. The counter above still works, which is the part that
              matters. */}
          {active.length === 0 && (
            <p className="py-6 text-center text-sm text-ink-muted">
              {t.till.noTables}
            </p>
          )}
          {/* ⚠️ A search with no hits says so. An empty grid under a box with
              text in it is indistinguishable from a room that has not loaded,
              and the second thing tried is a reload. */}
          {active.length > 0 && shown.length === 0 && (
            <p className="py-6 text-center text-sm text-ink-muted">
              {t.till.noTablesFound}
            </p>
          )}
        </div>
      )}

      {/* ---- By waiter ----

          ⚠️ **The cards show what is on the check.** The question a manager
          asks at eight o'clock is not "is table 7 taken" — the room answers
          that — it is "what is table 7 waiting for", and every till that makes
          you open a check to find out gets one opened by somebody who did not
          mean to edit it. */}
      {view === "waiters" && (
        <div className="flex min-h-0 flex-1">
          <div className="w-44 shrink-0 overflow-y-auto border-r border-line p-2">
            <button
              onClick={() => setWaiter("")}
              className={`${waiter === "" ? "till-row-on" : "till-row"} mb-1 flex-col items-stretch py-2`}
            >
              <span className="text-[13px] font-bold">{t.till.allWaiters}</span>
              <span className="till-num text-[11px] text-ink-muted">
                {checks.length}
              </span>
            </button>
            {waiters.map((wt) => (
              <button
                key={wt.name}
                onClick={() => setWaiter(wt.name)}
                className={`${waiter === wt.name ? "till-row-on" : "till-row"} mb-1 flex-col items-stretch py-2`}
              >
                <span className="truncate text-[13px] font-semibold">
                  {wt.name}
                </span>
                <span className="flex items-baseline justify-between gap-2">
                  <span className="till-num text-[11px] text-ink-muted">
                    {wt.count}
                  </span>
                  <span className="till-num text-[12px] font-semibold">
                    {formatPrice(wt.sum, currency, lang)}
                  </span>
                </span>
              </button>
            ))}
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto p-3">
            {waiterChecks.length === 0 ? (
              <p className="py-8 text-center text-sm text-ink-muted">
                {t.till.noOpenChecks}
              </p>
            ) : (
              <div className="grid grid-cols-2 gap-2 xl:grid-cols-3 2xl:grid-cols-4">
                {waiterChecks.map((c) => (
                  <CheckCard
                    key={c.id}
                    check={c}
                    currency={currency}
                    lang={lang}
                    onClick={() => onOpenCheck(c)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

/** One open check, with enough of its contents to answer "what is on it?".
 *
 *  ⚠️ **Capped at six lines.** A card that grows with the order turns the grid
 *  into a column, and the question this view answers is asked of the room at a
 *  glance, not of one table in detail. */
function CheckCard({
  check,
  currency,
  lang,
  onClick,
}: {
  check: Check;
  currency: string;
  lang: Lang;
  onClick: () => void;
}) {
  const t = useAdminT();
  const live = check.lines.filter((l) => !l.void);
  const shown = live.slice(0, 6);
  const rest = live.length - shown.length;
  const state = tableState(check);
  const billed = state === "billed";
  const late = state === "late";

  return (
    <button
      onClick={onClick}
      className={`till-tile p-0 ${
        billed
          ? "till-tile-billed"
          : late
            ? "till-tile-late"
            : "till-tile-busy"
      }`}
      aria-label={`${check.tableNumber || t.till.counter} · ${formatPrice(check.total, currency, lang)}`}
    >
      <span className="flex items-center justify-between gap-2 border-b border-line px-2.5 py-2">
        <span className="text-[17px] font-bold">
          {check.tableNumber || t.till.counter}
        </span>
        <span
          className={`till-num text-[12px] font-semibold ${
            late ? "text-danger" : "text-[rgb(var(--till-dim))]"
          }`}
        >
          {check.openMin} {t.till.minShort}
        </span>
      </span>

      <span className="flex min-h-[5.5rem] flex-1 flex-col gap-0.5 px-2.5 py-2">
        {shown.map((l) => (
          <span key={l.lineId} className="flex items-baseline gap-1.5">
            {/* The count in its own column, and coloured by whether the kitchen
                has it: on this card that is the whole state of the table. */}
            <span
              className="till-num min-w-[1.25rem] rounded-[5px] px-1 text-center text-[11px] font-bold"
              style={{
                background: l.fired
                  ? "rgb(var(--till-info) / 0.12)"
                  : "rgb(var(--till-accent) / 0.18)",
                color: l.fired
                  ? "rgb(var(--till-info))"
                  : "rgb(var(--till-accent-ink))",
              }}
            >
              {l.qty}
            </span>
            <span className="truncate text-[12px]">{l.name}</span>
          </span>
        ))}
        {rest > 0 && (
          <span className="text-[11px] text-[rgb(var(--till-dim))]">
            +{rest}
          </span>
        )}
      </span>

      <span className="flex items-baseline justify-between gap-2 border-t border-line px-2.5 py-1.5">
        {check.serverName ? (
          <span className="truncate text-[11px] text-[rgb(var(--till-dim))]">
            {check.serverName}
          </span>
        ) : (
          <span />
        )}
        <span className="till-num text-[15px] font-bold">
          {formatPrice(check.total, currency, lang)}
        </span>
      </span>
    </button>
  );
}


/** One way of looking at the room.
 *
 *  ⚠️ **Icon and word, never the icon alone.** Icon-only tabs are learnable in
 *  a week and unusable on the first evening — which is the evening a new waiter
 *  is standing in front of this during service. Same rule as the rail's. */
function ViewTab({
  on,
  onClick,
  icon,
  label,
}: {
  on: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      aria-current={on ? "page" : undefined}
      // The underline is drawn on the tab, inside the bar's own bottom border,
      // so the active tab reads as attached to what is under it.
      className={`-mb-px flex items-center gap-1.5 whitespace-nowrap border-b-2 px-3 py-3 text-[13px] font-semibold transition ${
        on
          ? "border-[rgb(var(--till-accent))] text-ink"
          : "border-transparent text-ink-muted hover:text-ink-soft"
      }`}
    >
      {icon}
      {label}
    </button>
  );
}

/** Does this table answer what was typed?
 *
 *  ⚠️ **The waiter's name counts as an answer.** Half of what this box gets
 *  asked is "where is 27" and the other half is "which of these are mine", and
 *  a search that only knew about numbers sent the second question to the
 *  waiter view — a different screen, laid out differently, for a question the
 *  person was already looking at the right screen for. */
function matches(tb: FloorTable, check: Check | undefined, q: string): boolean {
  if (tb.number.toLowerCase().includes(q)) return true;
  return (check?.serverName || "").toLowerCase().includes(q);
}

/** Finding one table in a room of forty.
 *
 *  ⚠️ **A button until it is needed, a field once it is.** The bar on a 1024px
 *  monoblock has about 600px to spend and an always-open search takes a sixth
 *  of it from the tabs and the counter button — the two things pressed every
 *  few minutes, against one pressed a few times a shift. Collapsed it is an
 *  icon; opened it takes the room it needs and gives it straight back.
 *
 *  ⚠️ **It does not collapse while it has text in it.** A box that closed on
 *  blur would throw the search away every time the waiter tapped the table they
 *  had just found — and the tap that follows a search is always a tap on a
 *  table. */
function TableSearch({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  const t = useAdminT();
  const [open, setOpen] = useState(false);
  const showing = open || value.length > 0;

  if (!showing) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={t.till.searchTables}
        title={t.till.searchTables}
        className="my-2 flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] border border-line bg-surface text-[rgb(var(--till-mid))] hover:bg-ink/[0.04]"
      >
        <LuSearch className="h-4 w-4" aria-hidden />
      </button>
    );
  }

  return (
    <span className="relative my-2 flex shrink items-center">
      <LuSearch
        className="pointer-events-none absolute left-2.5 h-4 w-4 text-[rgb(var(--till-dim))]"
        aria-hidden
      />
      <input
        autoFocus
        className="till-input h-9 w-[8.5rem] py-0 pl-8 pr-8 text-[13px] xl:w-[11rem]"
        placeholder={t.till.searchTables}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onBlur={() => {
          if (!value) setOpen(false);
        }}
      />
      <button
        type="button"
        onClick={() => {
          onChange("");
          setOpen(false);
        }}
        aria-label={t.till.clearSearch}
        className="absolute right-1.5 flex h-6 w-6 items-center justify-center rounded-full text-[rgb(var(--till-dim))] hover:bg-ink/[0.06]"
      >
        <LuX className="h-3.5 w-3.5" aria-hidden />
      </button>
    </span>
  );
}
