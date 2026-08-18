"use client";

import { useMemo, useState } from "react";

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
  const counterTables = useMemo(
    () => activeAll.filter((tb) => listZoneIDs.has(tb.zoneId ?? "")),
    [activeAll, listZoneIDs],
  );
  const active = useMemo(
    () => activeAll.filter((tb) => !listZoneIDs.has(tb.zoneId ?? "")),
    [activeAll, listZoneIDs],
  );

  // ⚠️ **The plan is only offered when there is one.** Coordinates default to
  // zero, so a restaurant that filled in table numbers and never opened the
  // floor-plan editor has every table stacked in the top-left corner — a room
  // that reads as broken. Those branches get the grid and never see the tab.
  const hasPlan = useMemo(
    () => active.some((tb) => tb.x !== 0 || tb.y !== 0) || shapes.length > 0,
    [active, shapes],
  );
  // ⚠️ **Derived, not stored-then-corrected.** The plan is the better first
  // screen where one exists — it is the room the person is standing in rather
  // than a list of its names — but whether one exists is only known once the
  // profile has loaded. An effect that flipped the view after mount swapped the
  // grid out from under a finger that was already on its way down, and the tap
  // landed on a tile being replaced. So the view is a pure function of the data
  // and the one choice the user has actually made.
  const [picked, setPicked] = useState<Mode | null>(null);
  const view: Mode = picked ?? (hasPlan ? "plan" : "grid");
  const setMode = setPicked;

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
  const shown = active.filter((tb) => (tb.zoneId ?? "") === current);
  // A list zone is a wall of numbers with no shape: seats and coordinates were
  // never filled in, so the tile drops the seat count rather than printing 0.
  const isList = tabs.find((z) => z.id === current)?.layout === "list";

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

  const waiterChecks = waiter
    ? checks.filter((c) => (c.serverName || "—") === waiter)
    : checks;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* ---- How to look at the room ----

          ⚠️ Three views of one fact, because three different questions get
          asked of it: "where is table 7" (the plan), "which tables are free"
          (the grid), and "what is on table 7 without walking to it" (the
          cards). A till that only draws the plan makes the third question a
          walk across the room. */}
      <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-line bg-surface px-3 py-2.5">
        <div className="till-seg-track">
          {hasPlan && (
            <button
              className={view === "plan" ? "till-seg-on" : "till-seg"}
              onClick={() => setMode("plan")}
            >
              {t.till.planView}
            </button>
          )}
          <button
            className={view === "grid" ? "till-seg-on" : "till-seg"}
            onClick={() => setMode("grid")}
          >
            {t.till.gridView}
          </button>
          <button
            className={view === "waiters" ? "till-seg-on" : "till-seg"}
            onClick={() => setMode("waiters")}
          >
            {t.till.waiterView}
          </button>
        </div>

        {/* Zones filter the room, so they belong beside the room views and not
            beside the card view, which is grouped by person instead. */}
        {view !== "waiters" && tabs.length > 1 && (
          <div className="till-seg-track no-scrollbar overflow-x-auto">
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

        {/* ⚠️ **The counter is first-class, not a fallback.** Half the places
            that would buy this sell over a counter, and a till that insists on
            a table number is a till they cannot use. It is reachable from every
            view, because "one coffee to take away" arrives while you are
            looking at whatever you happen to be looking at. */}
        <button className="till-btn-accent" onClick={() => onNewCheck("")}>
          + {t.till.counter}
        </button>
      </div>

      {/* ---- Counter checks ----
          Kept above the room in every view: they belong to nobody's table, so
          there is nowhere else they can appear. */}
      {(counter.length > 0 || counterTables.length > 0) && (
        <div className="flex shrink-0 gap-2 overflow-x-auto border-b border-line px-3 py-2">
          {/* The numbered slots the owner set up, in their own order: 101, 102,
              103 — the number the guest is called back by. Taken ones carry
              their total, free ones are an empty slot to open. */}
          {counterTables.map((tb) => {
            const c = byTable.get(tb.id);
            return (
              <button
                key={tb.id}
                onClick={() => (c ? onOpenCheck(c) : onNewCheck(tb.id))}
                aria-label={`${tb.number} · ${c ? t.till.busyLabel : t.till.free}`}
                className={`till-tile min-w-[6.5rem] shrink-0 justify-between p-2.5 ${
                  c ? "till-tile-busy" : ""
                }`}
              >
                <span className="text-[15px] font-bold">{tb.number}</span>
                <span className="till-num text-[13px] font-semibold">
                  {c ? formatPrice(c.total, currency, lang) : "—"}
                </span>
              </button>
            );
          })}
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
        <div className="min-h-0 flex-1 overflow-auto p-3">
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
        <div className="min-h-0 flex-1 overflow-y-auto p-3">
          <div className="grid grid-cols-4 gap-2 xl:grid-cols-6 2xl:grid-cols-8">
            {shown.map((tb) => {
              const c = byTable.get(tb.id);
              return (
                <Tile
                  key={tb.id}
                  label={tb.number}
                  seats={isList ? 0 : tb.seats}
                  check={c}
                  currency={currency}
                  lang={lang}
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
  const billed = !!check.precheckAt;
  const late = !billed && check.openMin >= LATE_MIN;

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

/** How long a table can sit before the tile stops being ordinary.
 *
 *  ⚠️ **Not a rule about service, a rule about attention.** Forty-five minutes
 *  is a normal lunch and a long wait for a bill, so the colour does not accuse
 *  anybody — it answers the only question this screen is scanned for during a
 *  rush: which table has nobody looking at it. A shorter threshold turns the
 *  whole room red at eight o'clock, and a room that is always red says nothing. */
const LATE_MIN = 45;

function Tile({
  label,
  sub,
  seats,
  check,
  currency,
  lang,
  onClick,
}: {
  label: string;
  /** A quiet second line — the counter check's printed number. */
  sub?: string;
  seats?: number;
  check?: Check;
  currency: string;
  lang: Lang;
  onClick: () => void;
}) {
  const t = useAdminT();
  const open = !!check;
  // ⚠️ The bill outranks the age: a table that has asked to pay is waiting for
  // a person, not for food, and forty minutes of that is a different problem
  // from forty minutes of eating.
  const billed = open && !!check!.precheckAt;
  const late = open && !billed && check!.openMin >= LATE_MIN;

  return (
    <button
      onClick={onClick}
      // Named by the table and its state: the tile's own text runs the number,
      // the seats and the money together into one unreadable string.
      aria-label={`${label} · ${
        billed ? t.till.billed : open ? t.till.busyLabel : t.till.free
      }`}
      // ⚠️ **Tinted, not filled.** An occupied tile used to be solid `brand`
      // with white text — unreadable on half the accents an owner can pick, and
      // the first things to go were the two numbers the tile exists for. A tint
      // with dark text survives every accent, and the dot carries the state at
      // full strength where nothing has to be read on top of it.
      className={`till-tile h-[9.25rem] justify-between p-3.5 ${
        billed
          ? "till-tile-billed"
          : late
            ? "till-tile-late"
            : open
              ? "till-tile-busy"
              : ""
      }`}
    >
      <span className="flex items-start justify-between gap-2">
        <span className="text-[26px] font-bold leading-none tracking-tight">
          {label}
        </span>
        {/* ⚠️ A dot, not a word. The grid is glanced at from across a room, and
            a word on every second tile is a grid nobody reads. */}
        <span
          className="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full"
          style={{
            background: billed
              ? "rgb(var(--till-info))"
              : late
                ? "rgb(var(--till-late))"
                : open
                  ? "rgb(var(--till-accent))"
                  : "rgb(var(--till-ok))",
          }}
        />
      </span>

      <span className="text-[13px] text-[rgb(var(--till-mid))]">
        {seats ? `${seats} ${t.till.seatsShort} · ` : ""}
        {billed ? t.till.billed : open ? t.till.busyLabel : t.till.free}
        {sub ? ` · ${sub}` : ""}
      </span>

      <span className="flex items-baseline justify-between gap-2">
        {/* ⚠️ The age, not the amount, is what says nobody has looked at this
            table in an hour — so it is the one that changes colour. */}
        <span
          className={`till-num text-[13px] ${
            late ? "font-bold text-danger" : "text-ink-muted"
          }`}
        >
          {open ? `${check!.openMin} ${t.till.minShort}` : "—"}
        </span>
        <span className="flex items-baseline gap-1.5">
          {open && check!.unfired > 0 && (
            <span
              className="mb-0.5 h-2 w-2 rounded-full"
              style={{ background: "rgb(var(--till-info))" }}
              title={t.till.pendingLabel}
            />
          )}
          <span className="till-num text-[17px] font-bold">
            {open ? formatPrice(check!.total, currency, lang) : "—"}
          </span>
        </span>
      </span>
    </button>
  );
}
