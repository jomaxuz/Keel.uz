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
// ⚠️ One icon at a time (`react-icons/lu`): the top-level entry point is an
// index of several thousand.
import { LuLockOpen, LuLogOut } from "react-icons/lu";

import {
  api,
  ApiError,
  clearTillDeviceToken,
  clearTillToken,
  hasTillDevice,
  setTillDeviceToken,
} from "@/lib/api";
import { contentName } from "@/lib/i18n/content";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { useStaff } from "@/lib/staff";
import PinPad from "@/components/till/PinPad";
import BookingsStrip from "@/components/till/BookingsStrip";
import TillChrome from "@/components/till/TillChrome";
import ShiftGate, { useShift } from "@/components/till/ShiftGate";
import ClockGate, { useClockRefusal } from "@/components/till/ClockGate";
import TablesScreen from "@/components/till/TablesScreen";
import CourseTabs from "@/components/till/CourseTabs";
import MenuGrid from "@/components/till/MenuGrid";
// One icon at a time (`react-icons/lu`): the top-level entry point is an index
// of several thousand.
import { LuLayoutGrid } from "react-icons/lu";
import OptionDialog from "@/components/till/OptionDialog";
import OrderPanel from "./OrderPanel";
import Toasts, { type Toast } from "@/components/till/Toasts";
import { useAsk } from "@/components/ui/Ask";
import type {
  OrderItemOption,
  Check,
  FloorShape,
  FloorTable,
  TableZone,
  MenuGroup,
  MenuItem,
  Staff,
  TillPerson,
  TillSession,
} from "@/lib/types";

/** Idle lock, matching the till. A tablet left on a table hands the next person
 *  the last one's name, which is the thing the PIN exists to prevent. */
const IDLE_LOCK_MS = 3 * 60 * 1000;

export default function FloorPage() {
  const t = useAdminT();
  const { ask } = useAsk();
  const { lang } = useI18n();
  const router = useRouter();
  const { staff, loading: authLoading, logout } = useStaff();

  // ⚠️ **Retiring the screen, which is not logging out of it.**
  //
  // Locking hands the screen to the next person; this stops the machine being a
  // till at all, and getting it back means somebody with a panel login fetching
  // a fresh link. So it asks first — the one control on these screens that
  // does, because it is the one whose mistake cannot be undone from the room it
  // was made in.
  //
  // ⚠️ The server is told **before** the local tokens are cleared. The other
  // order looks tidier and loses the register slot: the device row would stay
  // counted against the plan with no machine left able to name it, and the
  // restaurant would be at its cap with a till nobody can find.
  const exitScreen = useCallback(async () => {
    if (!(await ask({ title: t.till.exitConfirm, danger: true }))) return;
    try {
      await api.tillUnbind();
    } catch {
      // ⚠️ Swallowed, and the screen still goes. A manager who pressed this has
      // decided the machine is leaving; refusing to release it because our end
      // of a wire is down would strand them holding a till they cannot use and
      // cannot retire. The row is removable from the panel afterwards.
    }
    clearTillToken();
    clearTillDeviceToken();
    logout();
    // ⚠️ **Same bug as the till had, same fix.** `device` is state read once on
    // mount, so clearing tokens left the tablet sitting on the PIN pad with a
    // whole floor plan still in memory — which reads as "it only signed the
    // employee out". A full page load is the only reset that cannot miss a
    // piece, and it lands where a never-bound screen lands.
    //
    // The offline queue is deliberately untouched: unsent sales are money that
    // has not reached the server, and tidying a screen is not a reason to
    // delete them.
    window.location.replace("/staff/login?next=/zal");
  }, [t, logout]);

  const [device, setDevice] = useState<boolean | null>(null);
  const [person, setPerson] = useState<TillPerson | null>(null);
  // ⚠️ **What this tablet knows about itself, and it is read while locked.**
  // Whether the branch uses PINs at all (restaurants that have set none keep
  // working rather than being locked out by an upgrade), plus the three things
  // the lock screen puts on itself: the brand, the branch and the pictures the
  // owner chose — see StaffTillSession.
  const [session, setSession] = useState<TillSession | null>(null);
  // ⚠️ **Whether the last question got an answer**, which is the one fact on a
  // locked screen that can be wrong. Not `navigator.onLine`, which answers "is
  // there a wifi association" — true throughout an outage of the internet
  // behind the restaurant's own router. Same rule as lib/offline/useOffline,
  // arrived at from the request rather than from the browser.
  const [linkUp, setLinkUp] = useState(true);
  const pinsUsed = session ? session.pinsUsed : null;

  const [checks, setChecks] = useState<Check[]>([]);
  const [tables, setTables] = useState<FloorTable[]>([]);
  const [zones, setZones] = useState<TableZone[]>([]);
  const [shapes, setShapes] = useState<FloorShape[]>([]);
  const [plan, setPlan] = useState({ w: 1000, h: 700 });
  const [menu, setMenu] = useState<MenuGroup[]>([]);
  const [currency, setCurrency] = useState("UZS");
  const [branchName, setBranchName] = useState("");
  const [active, setActive] = useState<Check | null>(null);
  // ⚠️ Two panes now, not three screens: the left one is either the room or
  // the menu, and the order is always in the column beside it.
  const [view, setView] = useState<"tables" | "menu">("tables");
  // ⚠️ The waiter needs this at least as much as the cashier: the question
  // "which size" is asked at the table, by the person holding this screen.
  const [picking, setPicking] = useState<MenuItem | null>(null);
  // Which guest the next dish is for, and which course it goes out with. On the
  // page because the menu needs both — the tab is where the dish goes.
  const [guest, setGuest] = useState(0);
  const [course, setCourse] = useState(0);
  // Guards the same double-add as the till's: one tap, one line.
  const [adding, setAdding] = useState(false);
  // ⚠️ The waiter sees the same gate as the cashier, and for the same reason:
  // an order fired before the shift is open is food cooked against no count.
  // The screen has no payment button, but it opens the check that will be paid.
  // ⚠️ **Who is standing here, whichever kind of tablet this is.** A bound
  // tablet has a device token and no staff account; an older one has a staff
  // login and, if the branch has set no codes, no `person` either. Anything
  // gated on one of the two silently draws an empty room on the other — and an
  // empty room reads as "the tables have not been drawn yet", which sends the
  // waiter to a settings page that is already correct.
  // ⚠️ **One condition, and it is the same one the pad answers.** This used to
  // add "…or a staff account on a branch with no PINs", which is what let the
  // whole lock screen be skipped there — two places deciding whether the till
  // was open, and they could disagree. Now nothing loads, polls or opens a
  // drawer until somebody has come through the pad, whichever way they came.
  const unlocked = !!person;
  const shift = useShift(unlocked);
  // ⚠️ The floor opens checks too, so it stamps too — and a clock that has gone
  // backwards is the same fault on this screen as on the till.
  const clock = useClockRefusal(unlocked);
  const [mine, setMine] = useState(true);
  /** Let go of the table on screen.
   *
   *  ⚠️ **Told to the server, not only forgotten here.** Opening a check takes
   *  a hold so two screens cannot edit one table, and a hold dropped only
   *  locally leaves the table looking busy to everybody else until it expires —
   *  two minutes of a colleague waiting for somebody who has already walked
   *  away. Fired and not awaited: a waiter must never watch a spinner to leave
   *  a table, and a release that never arrives costs nothing because the hold
   *  expires by itself. */
  const release = useCallback(() => {
    setActive((cur) => {
      if (cur && !cur.id.startsWith("local:")) {
        void api.tillReleaseCheck(cur.id).catch(() => {});
      }
      return null;
    });
  }, []);
  const [catID, setCatID] = useState("");
  const [query, setQuery] = useState("");
  // ⚠️ Messages in the corner rather than a strip that moves the room down —
  // a waiter reaching for a table watched the tiles jump and pressed the one
  // that had moved into their finger. See components/till/Toasts.
  const [toasts, setToasts] = useState<Toast[]>([]);
  const setError = useCallback((text: string | null) => {
    if (!text) return;
    setToasts((l) => [
      ...l,
      { id: Date.now() + Math.random(), text, kind: "error" as const },
    ]);
  }, []);

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

  // Does this screen lock, and what does it put on itself while it does?
  //
  // ⚠️ **Polled while locked, once unlocked.** A tablet sits locked on a
  // side table for most of an afternoon, and this is where the banners, the
  // branch name and the connection light come from — a lock screen that
  // answered with whatever was true when the browser last started would show a
  // banner the owner deleted a week ago and a green light through an outage.
  useEffect(() => {
    if (!staff && !device) return;
    let alive = true;
    const ask = () =>
      api
        .tillSession()
        .then((r) => {
          if (!alive) return;
          setSession(r);
          setLinkUp(true);
          // ⚠️ Asked again because the call itself may have dropped a dead
          // device token: the screen would otherwise keep believing it is a
          // bound tablet and show a pad that nothing can unlock.
          setDevice(hasTillDevice());
        })
        .catch(() => {
          if (!alive) return;
          setLinkUp(false);
          // ⚠️ **A failure leaves the screen usable.** One that cannot reach
          // the server must still be able to open a check, so the first failure
          // answers "no PINs" and lets the waiter through rather than holding
          // them on a blank frame. A later success replaces this wholesale.
          setSession((s) => s ?? OFFLINE_SESSION);
        });
    void ask();
    if (person)
      return () => {
        alive = false;
      };
    const timer = setInterval(ask, LOCK_POLL_MS);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [staff, device, person]);

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
    if (!unlocked) return;
    void refresh();
    // Slower than the till's: a waiter is looking at one table, not watching
    // the room, and a tablet on battery does not need a poll every few seconds.
    const id = setInterval(() => void refresh(), 30000);
    return () => clearInterval(id);
  }, [unlocked, refresh]);

  useEffect(() => {
    if (!unlocked) return;
    // The room is this tablet's branch, not the site's default one.
    // ⚠️ The branch first, then **its** menu: the stop list belongs to this
    // kitchen, and a menu fetched without a branch answers for whichever one
    // the site serves by default — a waiter would be offering a dish that ran
    // out here an hour ago.
    api
      .tillBranch()
      .then(async (r) => [await api.getMenu({ branchId: r.id }), r] as const)
      .then(([m, r]) => {
        setMenu(m);
        setCatID((c) => c || (m[0]?.category.id ?? ""));
        setTables(r.booking?.tables ?? []);
        // ⚠️ Nil slices arrive as null, not [] — the tab strip maps over this.
        setZones(r.booking?.zones ?? []);
        setShapes(r.booking?.shapes ?? []);
        setPlan({ w: r.booking?.width || 1000, h: r.booking?.height || 700 });
        setCurrency(r.currency || "UZS");
        setBranchName(r.name ?? "");
      })
      .catch(() => setError(t.till.retry));
  }, [unlocked, t]);

  // Free tables right now: drawn tables that no open check is sitting on.
  const freeCount = useMemo(() => {
    const taken = new Set(checks.map((c) => c.tableId).filter(Boolean));
    return tables.filter((tb) => tb.isActive && !taken.has(tb.id)).length;
  }, [tables, checks]);

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
      setGuest(0);
      setCourse(0);
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
    portion?: number,
  ) {
    if (!active) return;
    setAdding(true);
    try {
      setActive(
        await api.tillAddLines(active.id, [
          {
            menuItemId: item.id,
            qty,
            ...(options?.length ? { options } : {}),
            ...(guest ? { guest } : {}),
            ...(course ? { course } : {}),
            // Off the wire for a whole one, which is what nearly every line is.
            ...(portion && portion !== 100 ? { portion } : {}),
          },
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
  // ⚠️ Locked until somebody names themselves. Nothing below renders — a till
  // that stayed usable while locked would be the old behaviour with a pad in
  // front of it.
  //
  // ⚠️ **Every till, every time — including a branch that has issued no codes.**
  // The pad used to be skipped entirely there, so the screen that asks "who is
  // standing here" was absent from exactly the machines nobody had set up
  // properly, and appeared for the first time as a surprise on the day somebody
  // set a code. `staffAsPerson` is the way through: the person is already
  // signed in with a real login, so the till knows the name and is only missing
  // the four digits — see components/till/PinPad.
  //
  // ⚠️ On a bound monoblock there is **no** fallback and none is offered: there
  // is no staff account behind the screen to name, so the PIN is the only way
  // in, which is what binding a monoblock from the panel means.
  if (!person) {
    return (
      <PinPad
        onUnlock={setPerson}
        session={session}
        online={linkUp}
        fallback={
          staff && pinsUsed === false
            ? {
                name: staff.name,
                onContinue: () => setPerson(staffAsPerson(staff)),
              }
            : undefined
        }
      />
    );
  } // ⚠️ Locked until somebody names themselves. Nothing below renders — a till
  // that stayed usable while locked would be the old behaviour with a pad in
  // front of it.
  //
  // ⚠️ **Every till, every time — including a branch that has issued no codes.**
  // The pad used to be skipped entirely there, so the screen that asks "who is
  // standing here" was absent from exactly the machines nobody had set up
  // properly, and appeared for the first time as a surprise on the day somebody
  // set a code. `staffAsPerson` is the way through: the person is already
  // signed in with a real login, so the till knows the name and is only missing
  // the four digits — see components/till/PinPad.
  //
  // ⚠️ On a bound monoblock there is **no** fallback and none is offered: there
  // is no staff account behind the screen to name, so the PIN is the only way
  // in, which is what binding a monoblock from the panel means.
  if (!person) {
    return (
      <PinPad
        onUnlock={setPerson}
        fallback={
          staff && pinsUsed === false
            ? {
                name: staff.name,
                onContinue: () => setPerson(staffAsPerson(staff)),
              }
            : undefined
        }
      />
    );
  }

  const canWaiter = person
    ? person.canWaiter || person.canCashier
    : !!staff && (staff.canWaiter || staff.canCashier);
  if (canWaiter && clock.refusal) {
    return (
      <main className="till flex h-dvh flex-col overflow-hidden bg-cream">
        <ClockGate refusal={clock.refusal} onRecheck={clock.recheck} />
      </main>
    );
  }

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
      <TillChrome
        title={`Keel · ${t.till.floor}`}
        personName={person?.name ?? staff?.name ?? ""}
        // ⚠️ The role's own name when there is one. A hardcoded "Ofitsiant"
        // named the *screen* rather than the person standing at it, so a
        // manager covering the floor read as a waiter — and the name in this
        // corner is how the room knows who is unlocked.
        roleLabel={person?.role ?? staff?.roleName ?? t.till.roleWaiter}
        branchName={branchName}
        shiftOpenedAt={shift.shift?.openedAt}
        device={!!device || pinsUsed === true}
        subscription={session?.subscription}
        // ⚠️ **Undefined, not disabled, for anybody who may not.** Passing a
        // handler that refuses would draw a button beside the padlock that does
        // nothing when pressed — and the person pressing it is standing in
        // front of a guest. Absent is the honest shape; the server refuses the
        // call as well, for whoever gets past the missing button.
        onExit={person?.canExit ? exitScreen : undefined}
        onLock={() => {
          // ⚠️ Lock, not sign out, whenever the screen can lock: the pad
          // comes back and the next person names themselves. Signing out of a
          // shared account mid-service is a different and worse thing.
          if (device || pinsUsed) {
            clearTillToken();
            setPerson(null);
            setActive(null);
            setView("tables");
          } else {
            logout();
          }
        }}
      >
        {view === "tables" && (
          <>
            {/* ⚠️ Mine by default, the whole floor one tap away: a screen
                showing everybody's tables is a list to read past, one showing
                only mine strands a table when somebody goes home early. */}
            <div className="till-seg-track">
              {/* ⚠️ Short here, full in the tooltip: at 1024px the two long
                  labels wrapped to two lines each and grew the header out over
                  the room below it. */}
              <button
                className={mine ? "till-seg-on" : "till-seg"}
                onClick={() => setMine(true)}
                title={t.till.myTables}
                aria-label={t.till.myTables}
              >
                {t.till.myTablesShort}
              </button>
              <button
                className={!mine ? "till-seg-on" : "till-seg"}
                onClick={() => setMine(false)}
                title={t.till.allTables}
                aria-label={t.till.allTables}
              >
                {t.till.allTablesShort}
              </button>
            </div>
            {/* The room in three numbers. ⚠️ Free first: it is the one a waiter
                walking in from the door is actually looking for. */}
            <div className="ml-1 hidden items-center gap-4 text-[13px] text-[rgb(var(--till-mid))] xl:flex">
              <span className="flex items-center gap-1.5">
                <span
                  className="h-2 w-2 rounded-full"
                  style={{ background: "rgb(var(--till-ok))" }}
                />
                {t.till.free} {freeCount}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="h-2 w-2 rounded-full"
                  style={{ background: "rgb(var(--till-accent))" }}
                />
                {t.till.busyLabel} {checks.length}
              </span>
            </div>
          </>
        )}
      </TillChrome>

      <Toasts
        items={toasts}
        onDismiss={(id) => setToasts((l) => l.filter((x) => x.id !== id))}
      />

      {/* ⚠️ **The room and the order, side by side.** They used to be two
          full-screen views: opening a table hid the floor, and the order hid
          both — so a waiter answering "what did table 3 order?" while standing
          beside table 9 had to leave the room to find out and then find their
          way back. The design keeps the check in a column that never leaves,
          and switches the left pane between the room and the menu. */}
      <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
        <section className="flex min-h-0 min-w-0 flex-1 flex-col">
          {view === "menu" && active ? (
            <>
              <div className="flex shrink-0 items-center gap-2 border-b border-line bg-surface px-3 py-2.5">
                {/* ⚠️ **Going back to the room lets go of the table.** It
                    used to keep it: a waiter added two dishes, pressed this,
                    and the order stayed in the right-hand column — so the next
                    person to pick up the tablet and press a dish put it on
                    somebody else's bill. The same gesture as the panel's own
                    "Orqaga", and it has to mean the same thing.

                    ⚠️ Nothing is lost: the lines are on the check already,
                    unsent ones included, and the table keeps its dot until
                    somebody tells the kitchen. */}
                <button
                  className="till-btn h-11 px-4"
                  onClick={() => {
                    release();
                    setView("tables");
                  }}
                >
                  <LuLayoutGrid className="h-4 w-4" aria-hidden />
                  {t.till.tables}
                </button>
                <input
                  className="till-input h-11 flex-1"
                  placeholder={t.till.search}
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
                {query && (
                  <button
                    className="till-btn w-11 shrink-0 px-0"
                    onClick={() => setQuery("")}
                    aria-label={t.till.back}
                  >
                    ✕
                  </button>
                )}
                {/* The course this dish goes out with — beside the search,
                    where the next tap already is. */}
                <CourseTabs value={course} onPick={setCourse} />
              </div>
              <MenuGrid
                menu={menu}
                items={items}
                categoryID={catID}
                onCategory={setCatID}
                query={query}
                // ⚠️ Colour, never photographs, on the floor screen: it runs on
                // a tablet on mobile data being carried around, and a grid of
                // images is the one thing that makes it feel slow in a guest's
                // presence.
                showImages={false}
                currency={currency}
                disabled={false}
                onPick={(it) => {
                  // A dish sold in parts asks the same question as one with
                  // options — which of these am I selling — and gets the same
                  // dialog.
                  if (
                    (it.options?.length ?? 0) > 0 ||
                    (it.portions?.length ?? 0) > 0
                  ) {
                    setPicking(it);
                    return;
                  }
                  void addDish(it);
                }}
              />
            </>
          ) : (
            <>
              <BookingsStrip active={view === "tables"} />
              <TablesScreen
                tables={tables}
                zones={zones}
                shapes={shapes}
                planWidth={plan.w}
                planHeight={plan.h}
                checks={checks}
                currency={currency}
                onOpenCheck={(c) => {
                  setActive(c);
                  setView("tables");
                }}
                onNewCheck={(tableId) => void openCheck(tableId)}
              />
            </>
          )}
        </section>

        {/* ---- This table's order ----

            ⚠️ **Drawn only while there is one.** It used to stand there empty
            with "pick a table" in it, which cost the room a fifth of a tablet
            for a sentence — and, worse, made letting go of a table look like
            nothing had happened: the column stayed exactly where it was, so a
            waiter could not tell a released check from a held one at a glance.
            The room now takes the whole screen when nobody is being served,
            which is also the screen this app spends most of its time on. */}
        {active && (
          <aside className="flex w-full shrink-0 flex-col border-t border-line bg-surface lg:w-[21rem] lg:border-l lg:border-t-0 xl:w-[24rem]">
            {active ? (
              <>
                <div className="flex shrink-0 items-center justify-between gap-2 border-b border-line px-4 py-3.5">
                  <span className="text-[21px] font-bold tracking-tight">
                    {active.tableNumber
                      ? `${active.tableNumber}-${t.till.table.toLowerCase()}`
                      : t.till.counter}
                    {active.guests ? (
                      <span className="text-ink-muted"> · {active.guests}</span>
                    ) : null}
                  </span>
                  {/* The state of this table in one word, where the design puts
                    it: amber while something is still a draft on the tablet,
                    quiet once the kitchen has all of it. */}
                  <div className="flex items-center gap-1.5">
                    {/* ⚠️ **What is standing at the pass, first.** A waiter reads
                      this header to decide whether to walk to the kitchen, and
                      until this number existed the only way to know was to go
                      and look. It outranks the draft chip during service: cold
                      food is a complaint, an unsent line is a delay. */}
                    {(active.readyWaiting ?? 0) > 0 && (
                      <span className="till-chip bg-emerald-600 text-white">
                        {t.till.waitingCount(active.readyWaiting ?? 0)}
                      </span>
                    )}
                    <span
                      className={`till-chip ${
                        active.unfired > 0 ? "till-chip-warn" : "till-chip-info"
                      }`}
                    >
                      {active.unfired > 0
                        ? t.till.pendingLabel
                        : t.till.firedLabel}
                    </span>
                  </div>
                </div>
                <OrderPanel
                  check={active}
                  currency={currency}
                  tables={tables}
                  busyTables={
                    checks.map((c) => c.tableId).filter(Boolean) as string[]
                  }
                  // ⚠️ Everybody's open checks, not only this waiter's: a table
                  // splitting the bill with the party next to them, or joining
                  // it, does not care whose section either table is in — and the
                  // room's default view is "mine", which would have made half
                  // the destinations invisible.
                  otherChecks={checks.filter((c) => c.id !== active.id)}
                  guest={guest}
                  onGuest={setGuest}
                  onAddDish={() => setView("menu")}
                  onChange={(next) => {
                    setActive(next);
                    void refresh();
                  }}
                  onBack={() => {
                    release();
                    setView("tables");
                    void refresh();
                  }}
                  onError={setError}
                />
              </>
            ) : (
              // Unreachable while the column is only drawn for an open check,
              // and kept as the branch's other half rather than deleted: the
              // panel is one `active` away from needing it again.
              <div className="flex flex-1 items-center justify-center p-8">
                <p className="max-w-[14rem] text-center text-[15px] leading-relaxed text-[rgb(var(--till-dim))]">
                  {t.till.selectTable}
                </p>
              </div>
            )}
          </aside>
        )}
      </div>

      {picking && (
        <OptionDialog
          item={picking}
          currency={currency}
          busy={adding}
          onCancel={() => setPicking(null)}
          onAdd={(options, qty, portion) =>
            void addDish(picking, options, qty, portion)
          }
        />
      )}
    </main>
  );
}

/** The signed-in staff account, read as the person at the till.
 *
 *  ⚠️ **Only ever used on a branch with no PINs**, and it is a name rather than
 *  a credential: the account was authenticated at the login screen, and the
 *  four digits this stands in for say *who*, never *whether*. Mapping it here
 *  rather than leaving `person` null is what keeps the rest of the screen on
 *  one path — the idle lock, the permissions, the header name and the lock
 *  button all read `person`, and a second "or the staff account" clause in each
 *  of them is four places to forget. */
function staffAsPerson(staff: Staff): TillPerson {
  return {
    id: staff.id,
    name: staff.name,
    position: staff.position,
    canWaiter: staff.canWaiter,
    canCashier: staff.canCashier,
    role: staff.roleName,
    // Same omission as the till's copy had, and the same consequence: on an
    // unbound tablet this object *is* the person, so a manager lost the exit
    // button entirely. Read from the resolved role — the legacy booleans carry
    // no `void` and would answer no for everybody.
    canExit: staff.perms?.includes("void"),
  };
}

/** How often a locked screen asks the server what it should be showing.
 *
 *  ⚠️ Slow, because it is the whole cost of the lock screen and it runs on
 *  every idle tablet in the country. Fifteen seconds is fast enough for a
 *  connection light to be believed and far too slow to matter. */
const LOCK_POLL_MS = 15_000;

/** What a screen assumes about itself when it cannot ask.
 *
 *  ⚠️ `pinsUsed: false` on purpose: one that cannot reach the server must still
 *  be able to open a check, so the unanswerable question falls the way that
 *  lets the person through rather than the way that locks a working restaurant
 *  out of its own evening. */
const OFFLINE_SESSION: TillSession = {
  pinsUsed: false,
  brandName: "",
  branchName: "",
  banners: [],
};
