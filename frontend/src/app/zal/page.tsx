"use client";

// The floor screen: my tables, and what is on them.
//
// ⚠️ **Not the till with the payment hidden.** A waiter carries this between
// tables; the till stands on a counter ending sales. The two share the server
// and the shared components in components/till, and nothing else — see
// docs/pos-reja.md §3 for why this one is the screen that moves to a phone.
//
// ⚠️ **"My tables" is the default, and the whole floor is one tap away.** A
// waiter's screen showing everybody's tables is a list they have to read past;
// one that *only* shows theirs strands a table when somebody goes home early.
// So: mine first, all available.

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import {
  api,
  ApiError,
  clearTillToken,
  hasTillDevice,
  setTillDeviceToken,
} from "@/lib/api";
import { contentName } from "@/lib/i18n/content";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { useStaff } from "@/lib/staff";
import LangSwitch from "@/components/site/LangSwitch";
import PinPad from "@/components/till/PinPad";
import ShiftGate, { useShift } from "@/components/till/ShiftGate";
import TablesScreen from "@/components/till/TablesScreen";
import MenuGrid from "@/components/till/MenuGrid";
import OptionDialog from "@/components/till/OptionDialog";
import OrderPanel from "./OrderPanel";
import type {
  OrderItemOption,
  Check,
  FloorTable,
  TableZone,
  MenuGroup,
  MenuItem,
  TillPerson,
} from "@/lib/types";

/** Idle lock, matching the till. A tablet left on a table hands the next person
 *  the last one's name, which is the thing the PIN exists to prevent. */
const IDLE_LOCK_MS = 3 * 60 * 1000;

export default function FloorPage() {
  const t = useAdminT();
  const { lang } = useI18n();
  const router = useRouter();
  const { staff, loading: authLoading, logout } = useStaff();

  const [device, setDevice] = useState<boolean | null>(null);
  const [person, setPerson] = useState<TillPerson | null>(null);
  const [pinsUsed, setPinsUsed] = useState<boolean | null>(null);

  const [checks, setChecks] = useState<Check[]>([]);
  const [tables, setTables] = useState<FloorTable[]>([]);
  const [zones, setZones] = useState<TableZone[]>([]);
  const [menu, setMenu] = useState<MenuGroup[]>([]);
  const [currency, setCurrency] = useState("UZS");
  const [active, setActive] = useState<Check | null>(null);
  const [view, setView] = useState<"tables" | "order" | "menu">("tables");
  // ⚠️ The waiter needs this at least as much as the cashier: the question
  // "which size" is asked at the table, by the person holding this screen.
  const [picking, setPicking] = useState<MenuItem | null>(null);
  // Guards the same double-add as the till's: one tap, one line.
  const [adding, setAdding] = useState(false);
  // ⚠️ The waiter sees the same gate as the cashier, and for the same reason:
  // an order fired before the shift is open is food cooked against no count.
  // The screen has no payment button, but it opens the check that will be paid.
  const shift = useShift(!!person || (!!staff && pinsUsed === false));
  const [mine, setMine] = useState(true);
  const [catID, setCatID] = useState("");
  const [query, setQuery] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const url = new URL(window.location.href);
    const tok = url.searchParams.get("t");
    if (tok) {
      setTillDeviceToken(tok);
      url.searchParams.delete("t");
      window.history.replaceState({}, "", url.pathname + url.search);
    }
    setDevice(hasTillDevice());
  }, []);

  useEffect(() => {
    if (device === null) return;
    if (!authLoading && !staff && !device) {
      router.replace("/staff/login?next=/zal");
    }
  }, [staff, authLoading, device, router]);

  useEffect(() => {
    if (!staff && !device) return;
    api
      .tillSession()
      .then((r) => setPinsUsed(r.pinsUsed))
      .catch(() => setPinsUsed(false));
  }, [staff, device]);

  // ⚠️ Same idle lock as the till, and it matters more here: a tablet left on
  // a table is a screen anybody can pick up, and every order taken on it would
  // carry the last waiter's name.
  useEffect(() => {
    if (!person) return;
    let timer: ReturnType<typeof setTimeout>;
    const arm = () => {
      clearTimeout(timer);
      timer = setTimeout(() => {
        clearTillToken();
        setPerson(null);
        setActive(null);
        setView("tables");
      }, IDLE_LOCK_MS);
    };
    arm();
    const events = ["pointerdown", "keydown"] as const;
    for (const e of events) window.addEventListener(e, arm);
    return () => {
      clearTimeout(timer);
      for (const e of events) window.removeEventListener(e, arm);
    };
  }, [person]);

  const refresh = useCallback(async () => {
    try {
      const res = await api.tillChecks(mine);
      setChecks(res.checks);
      // Keep the open check in step with the server without replacing what the
      // waiter is looking at: the id is the same row, the numbers are fresher.
      setActive((cur) =>
        cur ? (res.checks.find((c) => c.id === cur.id) ?? cur) : cur,
      );
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) setError(err.message);
    }
  }, [mine]);

  useEffect(() => {
    if (!person) return;
    void refresh();
    // Slower than the till's: a waiter is looking at one table, not watching
    // the room, and a tablet on battery does not need a poll every few seconds.
    const id = setInterval(() => void refresh(), 30000);
    return () => clearInterval(id);
  }, [person, refresh]);

  useEffect(() => {
    if (!person) return;
    Promise.all([api.getMenu(), api.getRestaurant()])
      .then(([m, r]) => {
        setMenu(m);
        setCatID((c) => c || (m[0]?.category.id ?? ""));
        setTables(r.restaurant.booking?.tables ?? []);
        // ⚠️ Nil slices arrive as null, not [] — the tab strip maps over this.
        setZones(r.restaurant.booking?.zones ?? []);
        setCurrency(r.restaurant.currency || "UZS");
      })
      .catch(() => setError(t.till.retry));
  }, [person, t]);

  const items = useMemo(() => {
    const q = query.trim().toLowerCase();
    const pool: MenuItem[] = q
      ? menu.flatMap((g) => g.items)
      : (menu.find((g) => g.category.id === catID)?.items ?? []);
    const visible = pool.filter((it) => it.isAvailable);
    if (!q) return visible;
    return visible.filter((it) =>
      contentName(it, lang).toLowerCase().includes(q),
    );
  }, [menu, catID, query, lang]);

  async function openCheck(tableId: string) {
    try {
      const check = await api.tillOpenCheck({ tableId, guests: 0 });
      setActive(check);
      // ⚠️ Straight to the menu, not to an empty check. A waiter opening a
      // table is standing beside it about to be told what they want, and a
      // screen that stops to show an empty list first is a tap that buys
      // nothing.
      setView("menu");
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.till.retry);
    }
  }

  async function addDish(
    item: MenuItem,
    options?: OrderItemOption[],
    qty = 1,
  ) {
    if (!active) return;
    setAdding(true);
    try {
      setActive(
        await api.tillAddLines(active.id, [
          { menuItemId: item.id, qty, ...(options?.length ? { options } : {}) },
        ]),
      );
      setPicking(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setAdding(false);
    }
  }

  if (device === null || pinsUsed === null) return null;
  if (!device && (authLoading || !staff)) return null;
  if (!person && (device || pinsUsed)) {
    return <PinPad onUnlock={setPerson} />;
  }

  const canWaiter = person
    ? person.canWaiter || person.canCashier
    : !!staff && (staff.canWaiter || staff.canCashier);
  if (canWaiter && !shift.loading && !shift.shift) {
    return (
      <main className="till flex h-dvh flex-col overflow-hidden bg-cream">
        <ShiftGate state={shift} currency={currency} />
      </main>
    );
  }

  if (!canWaiter) {
    return (
      <main className="till flex min-h-dvh items-center justify-center bg-cream p-6">
        <p className="max-w-sm text-center text-ink-muted">{t.till.noAccess}</p>
      </main>
    );
  }

  return (
    // The same frame as the till: a waiter who moves between the two screens
    // during a shift must not have to relearn where anything is.
    <main className="till flex h-dvh flex-col overflow-hidden bg-cream">
      <header className="till-chrome flex h-12 shrink-0 items-center gap-2 px-3">
        <span className="truncate text-sm font-medium">
          {person?.name ?? staff?.name}
        </span>
        {view === "tables" && (
          // ⚠️ Mine by default, all one tap away: a screen showing everybody's
          // tables is a list to read past, one showing only mine strands a
          // table when somebody goes home early.
          <button
            className={`ml-1 min-h-8 rounded-[8px] px-2.5 text-xs font-semibold transition ${
              mine
                ? "bg-brand text-white"
                : "border border-white/20 text-white/60"
            }`}
            onClick={() => setMine(!mine)}
          >
            {mine ? t.till.myTables : t.till.allTables}
          </button>
        )}
        <div className="ml-auto flex items-center gap-1">
          {/* ⚠️ No theme toggle: these screens are always light (forcedLight
              in lib/theme.tsx). A control that does nothing is worse than an
              absent one — the cashier presses it, nothing happens, and the next
              button that genuinely fails gets pressed twice too. */}
          <LangSwitch />
          <button
            className="till-btn-dark px-3"
            onClick={() => {
              if (device) {
                clearTillToken();
                setPerson(null);
                setActive(null);
                setView("tables");
              } else {
                logout();
              }
            }}
          >
            {device ? t.till.lock : t.till.logout}
          </button>
        </div>
      </header>

      {error && (
        <div className="shrink-0 bg-brand/15 px-3 py-2 text-sm">
          {error}
          <button className="ml-3 underline" onClick={() => setError(null)}>
            ✕
          </button>
        </div>
      )}

      {view === "tables" && (
        <TablesScreen
          tables={tables}
          zones={zones}
          checks={checks}
          currency={currency}
          onOpenCheck={(c) => {
            setActive(c);
            setView("order");
          }}
          onNewCheck={(tableId) => void openCheck(tableId)}
        />
      )}

      {view === "menu" && active && (
        <>
          <div className="flex shrink-0 items-center gap-2 px-3 py-2">
            <button className="btn h-11 px-4" onClick={() => setView("order")}>
              ← {active.tableNumber || t.till.counter}
            </button>
            <input
              className="input h-11 flex-1"
              placeholder={t.till.search}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
          <MenuGrid
            menu={menu}
            items={items}
            categoryID={catID}
            onCategory={setCatID}
            query={query}
            // ⚠️ Colour, never photographs, on the floor screen: it runs on a
            // tablet on mobile data being carried around, and a grid of images
            // is the one thing that makes it feel slow in a guest's presence.
            showImages={false}
            currency={currency}
            disabled={false}
            onPick={(it) => {
              if ((it.options?.length ?? 0) > 0) {
                setPicking(it);
                return;
              }
              void addDish(it);
            }}
          />
        </>
      )}

      {view === "order" && active && (
        <>
          <div className="flex shrink-0 items-center justify-between gap-2 border-b border-line px-3 py-2">
            <span className="font-display text-lg font-bold">
              {active.tableNumber
                ? `${active.tableNumber}-${t.till.table.toLowerCase()}`
                : t.till.counter}
            </span>
            <button
              className="till-btn-primary px-4"
              onClick={() => setView("menu")}
            >
              + {t.till.menu}
            </button>
          </div>
          <OrderPanel
            check={active}
            currency={currency}
            tables={tables}
            busyTables={
              checks.map((c) => c.tableId).filter(Boolean) as string[]
            }
            onChange={(next) => {
              setActive(next);
              void refresh();
            }}
            onBack={() => {
              setActive(null);
              setView("tables");
              void refresh();
            }}
            onError={setError}
          />
        </>
      )}

      {picking && (
        <OptionDialog
          item={picking}
          currency={currency}
          busy={adding}
          onCancel={() => setPicking(null)}
          onAdd={(options, qty) => void addDish(picking, options, qty)}
        />
      )}
    </main>
  );
}
