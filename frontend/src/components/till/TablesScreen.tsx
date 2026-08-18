"use client";

import { useMemo, useState } from "react";

import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { Lang } from "@/lib/i18n/dictionaries";
import type { Check, FloorTable, TableZone } from "@/lib/types";

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
  checks,
  currency,
  onOpenCheck,
  onNewCheck,
}: {
  tables: FloorTable[];
  /** Empty on a restaurant that has not split its room, which is most of them. */
  zones: TableZone[];
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

  const byTable = new Map<string, Check>();
  const counter: Check[] = [];
  for (const c of checks) {
    if (c.tableId) byTable.set(c.tableId, c);
    else counter.push(c);
  }

  const active = useMemo(() => tables.filter((tb) => tb.isActive), [tables]);

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

  return (
    <div className="min-h-0 flex-1 overflow-y-auto p-3.5">
      {/* ---- The counter ---- */}
      <div className="mb-5">
        <div className="mb-2 flex items-baseline gap-2">
          <h2 className="till-label">{t.till.counter}</h2>
          {counter.length > 0 && (
            <span className="till-chip till-chip-warn">{counter.length}</span>
          )}
        </div>
        <div className="grid grid-cols-4 gap-2 xl:grid-cols-6 2xl:grid-cols-8">
          {/* ⚠️ **Not the order number.** It is drawn from crypto/rand and
              reads "6XGC-ZNHV" — eight characters that mean nothing across a
              room and cannot be told apart at a glance from the one beside it.
              The tile is numbered by position instead, and the real number is
              kept underneath, small, because that is the one printed on the
              guest's receipt when they come back to ask. */}
          {counter.map((c, i) => (
            <Tile
              key={c.id}
              label={`#${i + 1}`}
              sub={c.number}
              check={c}
              currency={currency}
              lang={lang}
              onClick={() => onOpenCheck(c)}
            />
          ))}
          {/* ⚠️ Dashed, and it is the only dashed thing on the screen: an empty
              slot that looks like a tile is a tile a cashier taps expecting a
              check. */}
          <button
            onClick={() => onNewCheck("")}
            aria-label={t.till.newCheck}
            className="flex min-h-[4.5rem] items-center justify-center rounded-[12px] border border-dashed border-line-strong text-2xl font-bold text-ink-muted transition hover:border-ink/30 hover:text-ink-soft active:scale-[0.97]"
          >
            +
          </button>
        </div>
      </div>

      {/* ---- The room ---- */}
      {tabs.length > 0 && (
        <div>
          {/* One zone means no strip: the tab would name what the whole screen
              already is. */}
          {tabs.length > 1 && (
            <div className="till-seg-track no-scrollbar mb-3 self-start overflow-x-auto">
              {tabs.map((z) => {
                // How many of this zone's tables are sitting. ⚠️ On the tab,
                // because the zone you are not looking at is exactly the one
                // you forget: a waiter watching the hall cannot see that the
                // takeaway counter has four checks waiting.
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
                      <span
                        className={`rounded-full px-1.5 text-[11px] font-bold ${
                          z.id === current
                            ? "bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                            : "bg-ink/[0.06] text-ink-muted"
                        }`}
                      >
                        {busy}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          )}
          {tabs.length === 1 && (
            <h2 className="mb-2 till-label">{tabs[0].name}</h2>
          )}
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
        </div>
      )}

      {/* ⚠️ Said rather than left blank: a restaurant that has not drawn its
          room yet sees an empty screen and concludes the till is broken. The
          counter above still works, which is the part that matters. */}
      {active.length === 0 && (
        <p className="py-6 text-center text-sm text-ink-muted">
          {t.till.noTables}
        </p>
      )}
    </div>
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
  const late = open && check!.openMin >= LATE_MIN;

  return (
    <button
      onClick={onClick}
      // Named by the table and its state: the tile's own text runs the number,
      // the seats and the money together into one unreadable string.
      aria-label={`${label} · ${open ? t.till.busyLabel : t.till.free}`}
      // ⚠️ **Tinted, not filled.** An occupied tile used to be solid `brand`
      // with white text — unreadable on half the accents an owner can pick, and
      // the first things to go were the two numbers the tile exists for. A tint
      // with dark text survives every accent, and the dot carries the state at
      // full strength where nothing has to be read on top of it.
      className={`till-tile h-[9.25rem] justify-between p-3.5 ${
        late ? "till-tile-late" : open ? "till-tile-busy" : ""
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
            background: late
              ? "rgb(var(--till-late))"
              : open
                ? "rgb(var(--till-accent))"
                : "rgb(var(--till-ok))",
          }}
        />
      </span>

      <span className="text-[13px] text-[rgb(var(--till-mid))]">
        {seats ? `${seats} ${t.till.seatsShort} · ` : ""}
        {open ? t.till.busyLabel : t.till.free}
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
