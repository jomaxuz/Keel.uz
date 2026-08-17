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
    <div className="min-h-0 flex-1 overflow-y-auto p-4">
      {/* ---- The counter ---- */}
      <div className="mb-4">
        <h2 className="mb-1.5 till-label">{t.till.counter}</h2>
        <div className="grid grid-cols-4 gap-1.5 xl:grid-cols-6 2xl:grid-cols-8">
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
          <button
            onClick={() => onNewCheck("")}
            className="min-h-[4.25rem] rounded-[10px] border border-dashed border-line-strong text-xl font-bold text-ink-muted transition active:scale-[0.97]"
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
            <div className="mb-1.5 flex gap-1 overflow-x-auto">
              {tabs.map((z) => (
                <button
                  key={z.id}
                  onClick={() => setZone(z.id)}
                  className={`min-h-11 shrink-0 rounded-[10px] px-4 text-sm font-semibold transition ${
                    z.id === current
                      ? "bg-charcoal text-white"
                      : "bg-ink/[0.05] text-ink-soft hover:bg-ink/10"
                  }`}
                >
                  {z.name}
                </button>
              ))}
            </div>
          )}
          {tabs.length === 1 && (
            <h2 className="mb-1.5 till-label">{tabs[0].name}</h2>
          )}
          <div className="grid grid-cols-4 gap-1.5 xl:grid-cols-6 2xl:grid-cols-8">
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
  return (
    <button
      onClick={onClick}
      // ⚠️ Occupied tables are filled, free ones outlined. One glance has to
      // separate them across a room, and colour alone would not survive a
      // cashier who cannot see it — the fill does.
      className={`flex min-h-[4.25rem] flex-col justify-between rounded-[10px] border p-2 text-left transition active:scale-[0.97] ${
        open
          ? "border-transparent bg-brand text-white"
          : "border-line bg-surface hover:bg-ink/[0.03]"
      }`}
    >
      <div className="flex items-baseline justify-between gap-2">
        <span className="font-display text-xl font-bold leading-none">
          {label}
        </span>
        {seats ? (
          <span className={`text-[11px] ${open ? "opacity-70" : "text-ink-muted"}`}>
            {seats}
          </span>
        ) : null}
        {sub ? (
          <span className="truncate text-[10px] font-medium opacity-60">
            {sub}
          </span>
        ) : null}
      </div>
      {open ? (
        <div>
          <div className="text-[13px] font-bold tabular-nums">
            {formatPrice(check!.total, currency, lang)}
          </div>
          {/* The age, not the amount, is what says nobody has looked at this
              table in an hour. */}
          <div className="text-[11px] opacity-75">
            {check!.openMin} {t.till.minShort}
            {check!.unfired > 0 ? ` · ${t.till.pendingLabel}` : ""}
          </div>
        </div>
      ) : (
        <span className="text-[11px] text-ink-muted">{t.till.free}</span>
      )}
    </button>
  );
}
