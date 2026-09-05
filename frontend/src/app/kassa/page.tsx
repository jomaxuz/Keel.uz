"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
// One icon at a time (`react-icons/lu`): the top-level entry point is an index
// of several thousand.
import {
  LuArrowRightLeft,
  LuLayoutGrid,
  LuReceipt,
  LuBan,
  LuShoppingCart,
  LuSettings,
  LuMerge,
  LuSplit,
  LuUtensils,
  LuBike,
  LuWallet,
  LuX,
} from "react-icons/lu";

import {
  api,
  ApiError,
  clearTillDeviceToken,
  clearTillToken,
  setTillDeviceToken,
  hasTillDevice,
} from "@/lib/api";
import { isNetworkError } from "@/lib/offline/sales";
import {
  addLocalLine,
  fireLocal,
  isLocal,
  openLocalCheck,
  openLocalChecks,
  removeLocalLine,
  setLocalQty,
  type LocalCheck,
} from "@/lib/offline/checks";
import { useOffline } from "@/lib/offline/useOffline";
import { printReceipt } from "@/lib/print";
import { unpair } from "@/lib/tillBridge";
import { VERSION } from "@/lib/version";
import { useStaff } from "@/lib/staff";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import { contentName } from "@/lib/i18n/content";
import { roleLabelOf } from "@/lib/roleName";
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

import CheckPanel from "./CheckPanel";
import ScanPanel from "@/components/till/ScanPanel";
import WeightDialog, { byWeight } from "@/components/till/WeightDialog";
import UnfiledPanel from "./UnfiledPanel";
import CloseDayButton from "./CloseDayButton";
import CashShiftPanel from "./CashShiftPanel";
import ChecksScreen from "@/components/till/ChecksScreen";
import DebtsPanel from "./DebtsPanel";
import Toasts, { type Toast } from "@/components/till/Toasts";
import PinPad from "@/components/till/PinPad";
import BookingsStrip from "@/components/till/BookingsStrip";
import TillChrome from "@/components/till/TillChrome";
import SettingsScreen from "@/components/till/SettingsScreen";
import TillNav from "@/components/till/TillNav";
import StopListScreen from "@/components/till/StopListScreen";
import ZakupScreen from "@/components/till/ZakupScreen";
import OnlineScreen from "@/components/till/OnlineScreen";
import CourseTabs from "@/components/till/CourseTabs";
import MoveLinesDialog from "@/components/till/MoveLinesDialog";
import MergeDialog from "@/components/till/MergeDialog";
import MenuGrid from "@/components/till/MenuGrid";
import ShiftGate, { useShift } from "@/components/till/ShiftGate";
import ClockGate, { useClockRefusal } from "@/components/till/ClockGate";
import OptionDialog from "@/components/till/OptionDialog";
import ScanDialog from "@/components/till/ScanDialog";
import TablesScreen from "@/components/till/TablesScreen";
import NewCheckDialog from "@/components/till/NewCheckDialog";
import { useAsk } from "@/components/ui/Ask";

/**
 * The till.
 *
 * Three columns, and the order of them is the order of the job: **which table**
 * (left), **what they want** (middle), **what they owe** (right). A cashier
 * standing at a counter reads left to right once and does not scroll; every
 * layout that puts the total in the middle turns the last step into a hunt.
 *
 * What it deliberately does not do:
 *
 *   - **No basket step.** Tapping a dish puts it on the check immediately. A
 *     confirm-then-add flow doubles every tap on the screen that is used more
 *     than any other in the building.
 *   - **No auto-refresh of the check you are editing.** The list of tables
 *     polls; the open check does not. A poll that overwrites the line a cashier
 *     is halfway through voiding is how a till loses trust in one shift.
 */
/** How long the screen stays unlocked with nobody touching it.
 *
 *  ⚠️ Short enough that walking away hands the till back, long enough that a
 *  cashier taking an order at the table is not locked out on their way back.
 *  Three minutes is the number every POS lands on for the same reason. */
const IDLE_LOCK_MS = 3 * 60 * 1000;

/** Where a cashier can be: the room, the menu, the sales list, the drawer.
 *
 *  ⚠️ The list is its own destination rather than a fourth mode of the floor.
 *  The floor answers "where is table 7"; this answers "find me the check that
 *  just left" — and until it existed the answer was a manager's login on a
 *  machine standing in the dining room. */
type View =
  | "tables"
  | "order"
  | "checks"
  | "cash"
  | "stop"
  | "zakup"
  | "settings"
  | "online";

/** Where this monoblock remembers whether it draws photographs. */
const IMAGES_KEY = "keel_till_images";

export default function TillPage() {
  const router = useRouter();
  const { ask } = useAsk();
  const { staff, loading: authLoading, logout } = useStaff();

  const t = useAdminT();

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
  //
  // ⚠️ **It ends on a full page load, and that is the fix rather than a
  // flourish.** Clearing the tokens and calling `logout()` left the screen
  // exactly where it was: `device` is React state read once on mount, so it
  // stayed `true`, the redirect below never fired, and the till came back with
  // the PIN pad — which reads as "it only signed the employee out". Everything
  // that makes this screen a till also lives in memory by then (the menu, the
  // floor plan, the branch, the open checks, the session), and none of it is
  // reset by clearing a token.
  //
  // A hard navigation is the only reset that cannot miss a piece: the tab is
  // rebuilt with no tokens, which is byte for byte the state a Keel till is in
  // the first time it is switched on. It lands where a never-bound till lands.
  //
  // ⚠️ **The offline queue is deliberately left alone.** Unsent sales live in
  // IndexedDB and they are money that has not reached the server yet; a "sign
  // out" that quietly deleted them would destroy a shift's takings to tidy up
  // a screen. They are still there, and still flush, when the machine is bound
  // again.
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
    // Screen-local preferences go too: they belong to the machine that was
    // just retired, and the next restaurant to bind this tablet should not
    // inherit somebody else's dish-picture setting.
    try {
      window.localStorage.removeItem(IMAGES_KEY);
    } catch {
      // A browser with storage blocked has nothing to clear. Not a reason to
      // strand somebody on a screen they asked to leave.
    }
    // ⚠️ **Inside the Windows application there is no URL to go to.** The till
    // is a bundled app, so navigating leaves it: the monoblock ends up showing
    // a bare webview pointed at localhost, which in a restaurant is a black
    // screen with an address bar and nobody able to guess what to do next. The
    // application has its own way back — the setup screen — and `unpair()`
    // clears the pairing and reloads into it, returning false in a browser so
    // the line below still runs there.
    if (await unpair()) return;
    // `location.replace`, not `href`: the retired till must not be one Back
    // press away from a screen whose tokens are gone — that lands on a broken
    // half-loaded till rather than on the login.
    window.location.replace("/staff/login?next=/kassa");
  }, [t, logout]);
  const { lang } = useI18n();

  const [menu, setMenu] = useState<MenuGroup[]>([]);
  const [tables, setTables] = useState<FloorTable[]>([]);
  const [zones, setZones] = useState<TableZone[]>([]);
  // The room as it was drawn: walls, named areas and the plan's own size. The
  // till renders the same coordinates the booking page does.
  const [shapes, setShapes] = useState<FloorShape[]>([]);
  const [plan, setPlan] = useState({ w: 1000, h: 700 });
  const [currency, setCurrency] = useState("UZS");
  // Which counter this is. ⚠️ In the header because a chain's cashier can be
  // moved between branches in a week, and every till looks identical.
  const [branchName, setBranchName] = useState("");
  // ⚠️ Kept on the device so a check opened during an outage charges what the
  // same table would have been charged a minute earlier. The server owns the
  // number online; this is the copy the offline path needs.
  const [servicePercent, setServicePercent] = useState(0);
  /** Whether this counter sells the goods it bought — a shop rather than a
   *  kitchen. ⚠️ False until the branch has answered, so a slow network opens
   *  on the familiar screen rather than flashing a scanner at a waiter. */
  const [sellsGoods, setSellsGoods] = useState(false);
  // ⚠️ **Starts true**, like the branch response's absent field: the floor plan
  // is what a till has unless it is told otherwise, and a flicker that removes
  // the room for a moment on every boot is a waiter's tap landing on nothing.
  const [hasTables, setHasTables] = useState(true);
  // ⚠️ Starts true for the same reason, and asked separately: a fast food has
  // a kitchen and no tables.
  const [hasKitchen, setHasKitchen] = useState(true);
  const [scalePort, setScalePort] = useState("");
  const [checks, setChecks] = useState<Check[]>([]);
  // ⚠️ **Checks this device owns.** They were opened while the server was not
  // there, so nothing else in the building knows about them — not the kitchen
  // screen, not the panel, not the other till. They are drawn beside the
  // server's own so a waiter looking for table 7 finds it, and marked so nobody
  // wonders why the pass has not started cooking.
  const [locals, setLocals] = useState<LocalCheck[]>([]);
  const [active, storeActive] = useState<Check | null>(null);
  /** The same check, readable from a callback that has gone stale.
   *
   *  ⚠️ **A scanner outruns React, and a closure remembers what it was told.**
   *  Two scans in quick succession run through handlers created by the same
   *  render, so the second one reads `active` as null however long it waited —
   *  and opens a second check for a customer standing at one counter with one
   *  basket. State is what the screen draws; this is what the next line asks. */
  const activeRef = useRef<Check | null>(null);
  /** ⚠️ **The only writer**, so the ref cannot fall behind the state. Eighteen
   *  call sites set the open check; a mirror kept by an effect would be correct
   *  on screen and one turn late for the next scan, which is exactly the gap
   *  being closed here. */
  const setActive = useCallback((next: Check | null) => {
    activeRef.current = next;
    storeActive(next);
  }, []);
  const [catID, setCatID] = useState<string>("");
  // ⚠️ **Per device, in localStorage.** Photographs help on a bright 15" panel
  // with a decent processor and hurt on the four-gigabyte monoblock next to it;
  // that is a fact about the machine, so the machine remembers it. The company
  // default belongs with the receipt-design settings and is not built yet.
  const [showImages, setShowImages] = useState(false);

  // ⚠️ **The binding link is consumed and wiped from the address bar**, the
  // same way the kiosk screen does it: a long-lived branch token sitting in a
  // URL ends up in a browser history, a screenshot and a bookmark bar.
  useEffect(() => {
    const url = new URL(window.location.href);
    const tok = url.searchParams.get("t");
    if (tok) {
      setTillDeviceToken(tok);
      url.searchParams.delete("t");
      window.history.replaceState({}, "", url.pathname + url.search);
    }
    setDevice(hasTillDevice());
    setShowImages(window.localStorage.getItem(IMAGES_KEY) === "1");
  }, []);
  useEffect(() => {
    window.localStorage.setItem(IMAGES_KEY, showImages ? "1" : "0");
  }, [showImages]);
  const [query, setQuery] = useState("");
  const [opening, setOpening] = useState(false);
  /** Who is already on the table this cashier just walked into. */
  const [heldWarning, setHeldWarning] = useState("");
  // The table the floor tap chose, carried into the dialog so it only has to
  // ask how many guests.
  const [preTable, setPreTable] = useState("");
  // The dish waiting on an answer about its options. null = nothing is being
  // asked, which is the state a till spends almost all of its time in.
  const [picking, setPicking] = useState<MenuItem | null>(null);
  /** A weighed product tapped from the grid, waiting for its weight. */
  const [weighing, setWeighing] = useState<MenuItem | null>(null);
  // A marked dish waiting for its bottle to be scanned, with the choices
  // already made — the option dialog runs first, because what is being sold has
  // to be settled before the thing itself is identified.
  const [scanning, setScanning] = useState<{
    item: MenuItem;
    options?: OrderItemOption[];
  } | null>(null);
  // ⚠️ A second tap while the first request is in flight adds the dish twice —
  // the exact "two lines on the check" the dialog exists to prevent, arriving
  // through the dialog itself. On a monoblock over a restaurant's wifi the
  // window is wide enough to hit by accident.
  const [adding, setAdding] = useState(false);
  // Whether this branch files receipts at all. ⚠️ Asked once rather than
  // guessed from the check list: a restaurant with no register must not be
  // shown a "close the tax day" button, which would answer a question it has
  // never been asked and cannot act on.
  const [fiscalOn, setFiscalOn] = useState(false);
  // Who is standing here, once somebody has tapped their PIN.
  //
  // ⚠️ **null means locked**, and the screen shows nothing but the pad. The
  // whole point is that a check cannot be opened, voided or paid without a name
  // attached — a till that keeps working while locked would be the old
  // behaviour with an extra screen in front of it.
  const [person, setPerson] = useState<TillPerson | null>(null);
  // ⚠️ **What this monoblock knows about itself, and it is read while locked.**
  // Whether the branch uses PINs at all (restaurants that have set none keep
  // working rather than being locked out by an upgrade), plus the three things
  // the lock screen puts on itself: the brand, the branch and the pictures the
  // owner chose — see StaffTillSession.
  const [session, setSession] = useState<TillSession | null>(null);
  // ⚠️ **Whether the last question got an answer**, which is the one fact on a
  // locked till that can be wrong. Not `navigator.onLine`, which answers "is
  // there a wifi association" — true throughout an outage of the internet
  // behind the restaurant's own router. Same rule as lib/offline/useOffline,
  // arrived at from the request rather than from the browser.
  const [linkUp, setLinkUp] = useState(true);
  const pinsUsed = session ? session.pinsUsed : null;
  // Whether this monoblock is bound to a branch. ⚠️ null while we have not
  // looked: rendering the login redirect before knowing would bounce a
  // perfectly good till to a form it never needs.
  const [device, setDevice] = useState<boolean | null>(null);
  // ⚠️ **The room first, the menu second.** The first question of every order
  // is who it is for; a screen that opens on dishes makes that something you
  // answer afterwards, by remembering, and that is how a round of drinks lands
  // on the wrong bill.
  const [view, setView] = useState<View>("tables");
  // ⚠️ Lifted out of the check panel because the buttons that open them are now
  // on the bottom bar, which the panel does not own. The dialogs themselves
  // stay where the logic is.
  const [moving, setMoving] = useState(false);
  // ⚠️ **Which guest and which course the next dish belongs to.** They live on
  // the page rather than in the check panel because the *menu* needs them: the
  // tab is where the dish goes, not a filter over a list that is already there.
  const [guest, setGuest] = useState(0);
  const [course, setCourse] = useState(0);
  // Ticking dishes onto another check — a party that split, or joined.
  const [movingLines, setMovingLines] = useState(false);
  const [merging, setMerging] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  // How many sales are waiting on the tax register, for the rail's dot.
  const [unfiled, setUnfiled] = useState(0);
  // The drawer. Asked only once somebody is unlocked: a locked till has nobody
  // to answer for a shift, and asking anyway would spend a request per idle
  // monoblock every time the screen woke up.
  // ⚠️ **Who is standing here, whatever kind of till this is.** A bound
  // monoblock has a device token and **no staff account at all** — that is the
  // whole point of binding it from the panel with a link — so anything gated on
  // `staff` simply never runs on the ordinary installation. It did: the menu,
  // the floor plan and the open-checks poll were all behind `if (!staff)`, so a
  // bound till drew an empty room ("stollar chizilmagan"), an empty rail and no
  // dishes, while every request it did make succeeded. The screen looked set up
  // wrong rather than broken, which is the worst place for the bug to point.
  // ⚠️ **One condition, and it is the same one the pad answers.** This used to
  // add "…or a staff account on a branch with no PINs", which is what let the
  // whole lock screen be skipped there — two places deciding whether the till
  // was open, and they could disagree. Now nothing loads, polls or opens a
  // drawer until somebody has come through the pad, whichever way they came.
  const unlocked = !!person;
  const shift = useShift(unlocked);
  // ⚠️ **The network, as the till experiences it.** Not `navigator.onLine`,
  // which answers a different question — see lib/offline/useOffline.
  const net = useOffline(unlocked);
  // Whether this machine's clock may be stamped from at all.
  const clock = useClockRefusal(unlocked);
  // Said in the ordinary colour, not as an error: the sale is fine, we are not.
  // ⚠️ Messages, not state: they appear in the corner and take themselves away
  // (see components/till/Toasts). The offline banner below is deliberately not
  // one of these — a lost connection is still true a minute later.
  const [toasts, setToasts] = useState<Toast[]>([]);
  const say = useCallback((text: string, kind: Toast["kind"] = "info") => {
    if (!text) return;
    setToasts((list) => [
      ...list,
      { id: Date.now() + Math.random(), text, kind },
    ]);
  }, []);
  const setNotice = useCallback(
    (text: string | null) => say(text ?? "", "info"),
    [say],
  );
  const setError = useCallback(
    (text: string | null) => say(text ?? "", "error"),
    [say],
  );
  const [ready, setReady] = useState(false);

  // The id of the check on screen, read inside the poll without making the poll
  // depend on it — otherwise every keystroke would restart the timer.
  const activeID = useRef<string | null>(null);
  activeID.current = active?.id ?? null;

  // ---- Access ----
  //
  // The screen is hidden from someone who may not use it, but that is only the
  // courtesy half: every endpoint below checks the same permission, so a hidden
  // button is not what keeps a dishwasher out of the till.
  useEffect(() => {
    // ⚠️ A bound monoblock never sees the login form. That is the whole point:
    // typing a username and a password between two guests is what this replaces.
    if (device === null) return;
    if (!authLoading && !staff && !device) {
      router.replace("/staff/login?next=/kassa");
    }
  }, [staff, authLoading, device, router]);

  // ---- One-time loads ----
  useEffect(() => {
    if (!unlocked) return;
    let alive = true;
    (async () => {
      try {
        // ⚠️ The room comes from **this till's branch**, not from the public
        // profile: that one answers which branch a *visitor* is served from,
        // and on a company with two of them the counter drew the other room.
        // ⚠️ **The branch first, then its menu.** The stop list is the
        // branch's — a dish sold out here is on sale two kilometres away — and
        // asking for the menu without saying which branch this is returns
        // somebody else's answer, which is how a cashier ends up pressing a
        // dish the kitchen ran out of an hour ago.
        const branch = await api.tillBranch();
        const groups = await api.getMenu({ branchId: branch.id });
        if (!alive) return;
        setMenu(groups);
        setCatID(groups[0]?.category.id ?? "");
        const booking = branch.booking;
        // The floor plan is the serving branch's, and so is layered onto the
        // profile by the server — the same answer the booking page reads, so
        // the till cannot disagree with it about which tables exist.
        setTables(booking?.tables ?? []);
        // ⚠️ Nil slices arrive as null, not [] — the tab strip maps over this.
        setZones(booking?.zones ?? []);
        // ⚠️ Nil slices arrive as null, not [] — everything below maps over it.
        setShapes(booking?.shapes ?? []);
        setPlan({ w: booking?.width || 1000, h: booking?.height || 700 });
        setCurrency(branch.currency || "UZS");
        setBranchName(branch.name ?? "");
        setServicePercent(branch.servicePercent ?? 0);
        // ⚠️ **A capability, read from a call the till already makes.** The
        // counter needs to know whether to open on a scanner; it does not need
        // to know the difference between a pharmacy and a flower shop, and a
        // screen that switched on the business type would need editing every
        // time a type is added.
        setSellsGoods(branch.sellsGoods === true);
        // ⚠️ **Absent means yes.** Every till that exists today is a
        // restaurant's, and a server that has not been updated yet sends no
        // such field — read as `=== true` it would take the floor plan away
        // from a working dining room during a rollout.
        setHasTables(branch.hasTables !== false);
        setHasKitchen(branch.hasKitchen !== false);
        setScalePort(branch.scalePort ?? "");
        // ⚠️ **The counter opens on itself in a shop.** The room-first rule is
        // a restaurant's, and it exists because a line has to belong to a
        // table; in a shop the first action is a scan, and a cashier who has to
        // navigate to it before every sale would navigate to it three hundred
        // times a day.
        if (branch.sellsGoods === true) setView("order");
      } catch {
        // The menu failing is worth saying out loud — a till with no dishes on
        // it looks like a restaurant with no menu, and the cashier's next move
        // is to reboot the tablet.
        if (alive) setError(t.till.retry);
      } finally {
        if (alive) setReady(true);
      }
    })();
    return () => {
      alive = false;
    };
  }, [unlocked, t.till.retry]);

  // ---- The open-checks list ----
  const reloadLocals = useCallback(async () => {
    setLocals(await openLocalChecks());
  }, []);

  /** Mark the menu with what the branch cannot sell right now.
   *
   *  ⚠️ **Replaces rather than adds.** A dish put back on sale has to come back
   *  — the counter's toggle, a delivery recorded, a limit raised — and a flag
   *  that could only ever be set would leave it grey until the next restart,
   *  which is the same defect in the other direction. */
  const applySoldOut = useCallback((ids: string[]) => {
    const off = new Set(ids);
    setMenu((groups) =>
      groups.map((g) => ({
        ...g,
        items: g.items.map((it) =>
          !!it.soldOut === off.has(it.id)
            ? it
            : { ...it, soldOut: off.has(it.id) },
        ),
      })),
    );
  }, []);

  const refreshChecks = useCallback(async () => {
    try {
      const res = await api.tillChecks();
      net.seen(true);
      setChecks(res.checks);
      // ⚠️ **The menu's sold-out flags come from here**, not from the menu
      // itself: the menu is fetched once when the screen opens, so a dish that
      // ran out afterwards stayed pressable until the till was restarted — and
      // a dish that reached its daily batch never greyed out at all. The list
      // arrives with a poll that was already running.
      if (res.soldOut) applySoldOut(res.soldOut);
      // Keep the open check in step with the server, but only when nothing is
      // being typed into it: the panel below owns its own copy while it is
      // being edited.
      const id = activeID.current;
      // ⚠️ **A check this device owns is not in that list and never will be.**
      // The poll clears the open check when the server stops listing it —
      // right, because somebody else closed it — but a local check lives here,
      // so the same rule would wipe the table a cashier is standing in front
      // of, one poll after they opened it.
      if (id && !id.startsWith("local:")) {
        const fresh = res.checks.find((c) => c.id === id);
        if (!fresh) setActive(null);
      }
    } catch (err) {
      // ⚠️ The poll is the till's heartbeat: it runs every fifteen seconds
      // whatever else is happening, which makes it the cheapest honest answer
      // to "can we reach the server right now".
      if (!(err instanceof ApiError)) net.seen(false);
      if (err instanceof ApiError && err.status === 403) setError(err.message);
    }
    // ⚠️ **`net.seen`, not `net`.** This callback holds the poll below, so
    // anything unstable here restarts the poll — and the poll's own setState
    // then causes the render that restarts it again. `seen` is the only part of
    // `net` used here and it never changes identity; depending on the whole
    // object would tie the till's heartbeat to a badge counter.
  }, [net.seen, applySoldOut]);

  useEffect(() => {
    if (!unlocked) return;
    void reloadLocals();
    void refreshChecks();
    // 15s, the same beat as the panel's alert poll. A till is not a chat: the
    // thing that changes underneath you is another waiter opening a table, and
    // fifteen seconds is faster than anybody can walk there.
    const timer = setInterval(() => void refreshChecks(), 15_000);
    return () => clearInterval(timer);
  }, [unlocked, refreshChecks, reloadLocals]);

  /** Let go of the check on screen.
   *
   *  ⚠️ **Told to the server, not only forgotten here.** Opening a check takes
   *  a hold so two screens cannot edit one table, and a hold that is only
   *  dropped locally would leave the table looking busy to everybody else until
   *  it expired — which is two minutes of a colleague being told to wait for
   *  somebody who has already walked away.
   *
   *  ⚠️ Fired and not awaited: this runs on the way out of a screen, and a
   *  waiter must never watch a spinner to leave a table. A release that does
   *  not arrive costs nothing — the hold expires by itself, which is the half
   *  that actually makes this safe.
   */
  /** Enter a table, saying so first when somebody else is on it.
   *
   *  ⚠️ **Warned at the door, not at the button.** The server refuses the edit
   *  either way — with the holder's name — but a cashier who has already walked
   *  into a table, chosen a payment method and pressed pay is being told at the
   *  worst possible moment, in front of the guest. The name is on the check
   *  before any of that, and only while the hold is fresh, so it is a fact
   *  about right now.
   *
   *  ⚠️ **Warned, not blocked.** A cashier legitimately opens a waiter's table:
   *  to answer for a bill over the phone, to take payment while the waiter is
   *  in the kitchen, to look. Refusing outright would make the till less
   *  capable than the room it serves; the warning is what makes it a decision
   *  rather than an accident. */
  const enter = useCallback((c: Check) => {
    if (c.heldBy) setHeldWarning(c.heldBy);
    setActive(c);
    setView("order");
  }, []);

  const release = useCallback(() => {
    const id = activeID.current;
    setActive(null);
    if (id && !id.startsWith("local:"))
      void api.tillReleaseCheck(id).catch(() => {});
  }, []);

  // ---- Menu view ----
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

  // Does this screen lock, and what does it put on itself while it does?
  //
  // ⚠️ **Polled while locked, once unlocked.** A monoblock stands locked for
  // most of an afternoon, and this is where the banners, the branch name and
  // the connection light come from — a lock screen that answered with whatever
  // was true when the browser last started would show a banner the owner
  // deleted a week ago and a green light through an outage.
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
          // bound monoblock and show a pad that nothing can unlock.
          setDevice(hasTillDevice());
        })
        .catch(() => {
          if (!alive) return;
          setLinkUp(false);
          // ⚠️ **A failure leaves the till usable.** One that cannot reach the
          // server must still be able to sell, so the first failure answers
          // "no PINs" and lets the screen through rather than holding it on a
          // blank frame forever. A later success replaces this wholesale.
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

  // ⚠️ **Idle lock, and this is what makes the PIN mean anything.** Without it
  // one unlock at six covers the whole evening and every void is attributed to
  // whoever happened to arrive first — exactly the failure the PIN exists to
  // fix, with an extra screen in front of it.
  useEffect(() => {
    if (!person) return;
    let timer: ReturnType<typeof setTimeout>;
    const arm = () => {
      clearTimeout(timer);
      timer = setTimeout(() => {
        clearTillToken();
        setPerson(null);
        setActive(null);
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

  useEffect(() => {
    api
      .tillFiscalStatus()
      .then((f) => setFiscalOn(f.enabled))
      // Silent: a till whose fiscal status could not be read still sells food,
      // and the button simply stays hidden — the honest default when we do not
      // know whether a register exists.
      .catch(() => {});
  }, []);

  /** Ask the server for a receipt's lines and hand them to the printer.
   *
   *  ⚠️ The layout is the server's, character by character — the same code that
   *  draws the preview the owner approved in the settings. */
  const [printing, setPrinting] = useState(false);
  async function print(kind: "kitchen" | "till" | "customer" | "precheck") {
    if (!active) return;
    setPrinting(true);
    try {
      const res = await api.tillPrint(active.id, kind);
      setActive(res.check);
      // ⚠️ **The browser only prints when the restaurant's own printer did
      // not.** A branch with a printer at the counter gets paper without a
      // dialog; one with none gets the browser's print window, which is how
      // every first evening goes. The cashier never has to know which they are.
      // ⚠️ **The drawer opens for the till's own copy and nothing else** — the
      // rule the print queue already applies to the branch's printers
      // (printqueue.go). A bill or a kitchen ticket that kicks the drawer is a
      // drawer standing open through the evening, which is both a theft risk
      // and the reason somebody wedges it shut.
      if (res.queued === 0)
        printReceipt(res.lines, res.widthMM, res.logoUrl, {
          drawer: kind === "till",
        });
      void refreshChecks();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setPrinting(false);
    }
  }

  // ⚠️ **Returns the check rather than only storing it.** A shop's first scan
  // opens the sale and adds a line in one action, and the line cannot wait for
  // a state update that lands after this function returns — that race put the
  // first scanned packet of every sale on nothing at all.
  async function openCheck(
    tableId: string,
    guests: number,
  ): Promise<Check | null> {
    setOpening(false);
    setGuest(0);
    setCourse(0);
    try {
      const check = await api.tillOpenCheck({ tableId, guests });
      net.seen(true);
      setActive(check);
      setView("order");
      await refreshChecks();
      return check;
    } catch (err) {
      // ⚠️ **A table is opened locally rather than refused.** The guests are
      // sitting down; a till that cannot start their order over a wifi drop is
      // a till the restaurant keeps a paper pad beside.
      if (isNetworkError(err)) {
        net.seen(false);
        const table = tables.find((tb) => tb.id === tableId);
        const check = await openLocalCheck(
          tableId,
          table?.number ?? "",
          guests,
          person?.name ?? staff?.name ?? "",
          servicePercent,
        );
        if (check) {
          setActive(check);
          setView("order");
          await reloadLocals();
          // ⚠️ No notice bar here. The check panel already carries this
          // sentence, attached to the check it is about and for as long as the
          // check is local — a banner saying the same thing at the top is the
          // same warning twice, and the one that can be dismissed teaches
          // people to dismiss the one that cannot.
          return check;
        }
        setError(t.till.offlineNoStore);
        return null;
      }
      setError(err instanceof ApiError ? err.message : t.till.retry);
    }
    return null;
  }

  /** The sale a shop's counter starts by scanning.
   *
   *  ⚠️ **No table and one guest, deliberately.** A counter sale is the case
   *  `StaffOpenCheck` already allows without a table; a dialog asking a shop
   *  cashier how many people are in their party would be asked three hundred
   *  times a day and answered wrongly once.
   *
   *  ⚠️ **One at a time, and the second caller waits for the first.** Two
   *  scans a fraction of a second apart — a scanner that sent its code twice,
   *  two taps on a card — both read `active` as null, because state has not
   *  landed yet, and both open a check. The customer's two items end up on two
   *  bills, one of which is paid and one of which stays open on the counter
   *  for the rest of the day. Holding the promise rather than a boolean is what
   *  makes the second line land on the same check instead of being dropped.
   *
   *  ⚠️ A ref, not state: this must be true for the *next line of this
   *  function*, and a state update is not. That gap is the whole bug. */
  const openingCounter = useRef<Promise<Check | null> | null>(null);
  function openCounterCheck(): Promise<Check | null> {
    if (!openingCounter.current) {
      openingCounter.current = openCheck("", 1).finally(() => {
        openingCounter.current = null;
      });
    }
    return openingCounter.current;
  }

  async function addDish(
    item: MenuItem,
    options?: OrderItemOption[],
    qty = 1,
    markCode?: string,
    portion?: number,
  ) {
    // ⚠️ **A shop's check opens itself on the first scan.** There is no table
    // to tap and no party to count, so requiring an open check would make the
    // opening move of every sale a step the cashier performs on a screen that
    // has nothing else on it. A restaurant is unchanged: no check, no line.
    const check =
      activeRef.current ?? (sellsGoods ? await openCounterCheck() : null);
    if (!check) return;
    // ⚠️ **The scan is asked for here rather than at the tile**, because this is
    // the one path every way of adding a dish goes through — a plain tap, a tap
    // that opened the option dialog, and a repeat of a line. A check at the
    // tile would be one the next caller forgets.
    if (item.marked && !markCode) {
      setPicking(null);
      setScanning({ item, options });
      return;
    }
    // A check this device owns is edited here; there is nothing to ask.
    if (isLocal(check)) {
      setAdding(true);
      try {
        const next = await addLocalLine(
          check as LocalCheck,
          item,
          qty,
          options,
          guest,
          course,
          markCode,
          portion,
        );
        setActive(next);
        setPicking(null);
        await reloadLocals();
      } finally {
        setAdding(false);
      }
      return;
    }
    setAdding(true);
    try {
      // Optimism is wrong here: the price, the sold-out list and the brand check
      // all live on the server, and a line that appears and then vanishes is
      // worse than one that takes 200ms to appear.
      const next = await api.tillAddLines(check.id, [
        {
          menuItemId: item.id,
          qty,
          ...(options?.length ? { options } : {}),
          // Zero is the ordinary case — one bill, one service — and is left off
          // the wire entirely so a counter's requests look exactly as they did.
          ...(guest ? { guest } : {}),
          ...(course ? { course } : {}),
          ...(markCode ? { markCode } : {}),
          // A whole portion is left off the wire entirely: it is what every
          // line was before parts existed, and the server stores it as absent.
          ...(portion && portion !== 100 ? { portion } : {}),
        },
      ]);
      setActive(next);
      setPicking(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setAdding(false);
    }
  }

  /** One tap for a plain dish, a dialog for one that asks a question.
   *
   *  ⚠️ The branch is on **whether the dish has groups**, not on whether any is
   *  required. An optional "qo'shimcha pishloq" that can only be chosen by
   *  editing the line afterwards is a group the till cannot sell either — and
   *  the guest asked for it at the counter, not later. */
  function pick(item: MenuItem) {
    // ⚠️ **A kilo is not a piece.** Tapped from the grid, a product measured by
    // weight has no quantity yet — adding it as one would sell a kilo of
    // anything for the price of one unit, on the receipt, silently. The scan
    // path already asks; this is the same question asked from the other side.
    if (sellsGoods && byWeight(item)) {
      setWeighing(item);
      return;
    }
    // ⚠️ A dish that can be sold in parts asks the same question a dish with
    // options does — which one of these am I selling — so it opens the same
    // dialog. Without this the only way to sell half a loaf would be to add a
    // whole one and correct it, which is two mistakes waiting to be made.
    if ((item.options?.length ?? 0) > 0 || (item.portions?.length ?? 0) > 0) {
      setPicking(item);
      return;
    }
    void addDish(item);
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
  }

  // ⚠️ Permissions come from whoever is unlocked, never from the device — the
  // monoblock has none of its own, and reading them off the account that
  // happens to be signed in is exactly the mix-up the PIN exists to end.
  const canCashier = person ? person.canCashier : !!staff?.canCashier;
  // ⚠️ From the unlocked person, like every other permission here: the
  // monoblock has none of its own. Falls back to the signed-in account for a
  // till installed before PIN sessions existed.
  const canBuyOrder = person
    ? !!person.canBuyOrder
    : !!staff?.perms?.includes("buyorder");
  // The role's own name in this screen's language — see lib/roleName.ts.
  const roleLabel = roleLabelOf(person, staff, lang);
  const canTill = person
    ? person.canWaiter || person.canCashier
    : !!staff && (staff.canWaiter || staff.canCashier);
  // ⚠️ **The drawer before the room.** A check opened with no shift open
  // belongs to no count, so the money is taken and the evening's figure is
  // quietly short — with nothing on any screen saying so. The gate is drawn
  // instead of the floor rather than above it: a banner on a working till is a
  // banner that gets worked past, because the first guest is already standing
  // there.
  // ⚠️ **Before the drawer, because it is a worse fault than a missing shift.**
  // A check opened with no shift is money the count is short by; a check
  // stamped from a clock that has gone backwards is a tax document with the
  // wrong date on it, written by a machine that will keep doing it all evening.
  // See lib/offline/clock.ts.
  if (canTill && clock.refusal) {
    return (
      <main className="till flex h-dvh flex-col overflow-hidden bg-cream">
        <ClockGate refusal={clock.refusal} onRecheck={clock.recheck} />
      </main>
    );
  }

  if (canTill && !shift.loading && !shift.shift) {
    return (
      <main className="till flex h-dvh flex-col overflow-hidden bg-cream">
        <ShiftGate state={shift} currency={currency} />
      </main>
    );
  }

  if (!canTill) {
    return (
      <main className="till flex min-h-dvh items-center justify-center bg-cream p-6">
        <p className="max-w-sm text-center text-ink-muted">{t.till.noAccess}</p>
      </main>
    );
  }

  return (
    // ⚠️ `till` re-declares the theme variables it inherits: this screen must
    // not change shape because the owner tried a new look on the website, and a
    // scaled-up root font size overflows the 1024×768 monoblock most of these
    // run on.
    <main className="till flex h-dvh flex-col overflow-hidden bg-cream">
      <TillChrome
        title="Keel POS"
        personName={person?.name ?? staff?.name ?? ""}
        // ⚠️ **The role's own name when there is one.** This printed "Kassir"
        // for anybody who could work a till, so a manager and a cashier at the
        // same monoblock read as the same person — and the name in the corner
        // is how the room knows who is unlocked and who a void will be
        // recorded against. The old two-way label stays as the fallback for an
        // account with no role, which is every install that predates them.
        roleLabel={
          roleLabel ?? (canCashier ? t.till.roleCashier : t.till.roleWaiter)
        }
        branchName={branchName}
        shiftOpenedAt={shift.shift?.openedAt}
        device={!!device || pinsUsed === true}
        subscription={session?.subscription}
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
      />

      {/* ⚠️ **The room keeps working, and the banner says so.** A till that
          announced a lost connection as an error would have a cashier stop and
          call somebody — while the kitchen is cooking, the drawer is opening
          and the only thing that has actually failed is our end of a wire. */}
      {(!net.online || net.pending > 0) && (
        <div className="flex shrink-0 items-center gap-3 border-b border-[rgb(var(--till-accent-line))] bg-[rgb(var(--till-accent-tint))] px-3 py-2 text-sm">
          <span className="font-semibold text-[rgb(var(--till-accent-ink))]">
            {net.online
              ? t.till.offlinePending(net.pending)
              : t.till.offlineTitle}
          </span>
          <span className="hidden min-w-0 flex-1 truncate text-[13px] text-ink-muted lg:block">
            {t.till.offlineHint}
          </span>
          {net.pending > 0 && (
            <button
              className="till-btn shrink-0"
              onClick={() => void net.flush()}
            >
              {t.till.offlineSend}
            </button>
          )}
        </div>
      )}

      {/* ⚠️ In the corner and gone in five seconds — see components/till/Toasts.
          These used to be strips above the room: reaching for a table, the
          cashier watched the grid jump a row and pressed the tile that had
          moved into their finger. */}
      <Toasts
        items={toasts}
        onDismiss={(id) => setToasts((l) => l.filter((x) => x.id !== id))}
      />

      <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
        {/* ---- Where you are ----

            ⚠️ **Three destinations, and the room is the first.** The first
            question of every order is who it is for; a till that opens on
            dishes makes that something you answer afterwards, by remembering,
            and that is how a round of drinks lands on the wrong bill. */}
        <TillNav
          items={[
            // ⚠️ **Removed rather than disabled where there is no room.** The
            // rule elsewhere on this screen is the opposite — a control that
            // vanishes is a control people hunt for — but that rule is about
            // things that are temporarily unavailable. A shop has no floor
            // plan and never will, and a permanently grey first destination is
            // a shop cashier's first impression of the till.
            ...(hasTables
              ? [
                  {
                    id: "tables",
                    icon: <LuLayoutGrid />,
                    label: t.till.tables,
                    // Somewhere in the room a check has lines the kitchen has
                    // not been told about — the one thing that goes quietly
                    // wrong.
                    dot: checks.some((c) => c.unfired > 0),
                  },
                ]
              : []),
            {
              id: "order",
              icon: <LuUtensils />,
              label: t.till.menu,
              // ⚠️ Disabled rather than hidden: a menu with nothing to add a
              // dish to is a screen that answers every tap with silence, and a
              // control that vanishes is a control people hunt for.
              //
              // ⚠️ **Never disabled in a shop**, where this is not a menu but
              // the counter itself: the scan opens its own check, so there is
              // nothing to wait for and nothing to disable it against.
              disabled: !active && !sellsGoods,
            },
            {
              // ⚠️ **Not behind `canCashier`.** The person told that lag'mon
              // has run out is whoever is nearest the kitchen door, and that is
              // usually a waiter. Stopping a dish takes no money out and
              // destroys no record — the test the server applies too — so the
              // button lives where the news arrives.
              id: "stop",
              icon: <LuBan />,
              label: t.till.stopList,
            },
            // ⚠️ **Its own section, not a corner of the stop list.** The two
            // answer opposite questions — "this is off the menu now" and "buy
            // this tomorrow" — and somebody reaching for one at eight in the
            // evening must not land on the other.
            //
            // ⚠️ Drawn from the permission, never from the role's name: the
            // spelling of a job title grants nothing (models/staffrole.go).
            ...(canBuyOrder
              ? [
                  {
                    id: "zakup",
                    icon: <LuShoppingCart />,
                    label: t.zakup.title,
                  },
                ]
              : []),
            ...(canCashier
              ? [
                  {
                    id: "checks",
                    icon: <LuReceipt />,
                    label: t.till.check,
                  },
                  {
                    // ⚠️ **Beside the drawer, not among the tables.** A table
                    // is something you serve; an online order is money you
                    // either collect or do not, so it belongs next to the
                    // question "what is in the till" rather than next to
                    // "who is sitting where".
                    id: "online",
                    icon: <LuBike />,
                    label: t.online.title,
                  },
                  {
                    id: "cash",
                    icon: <LuWallet />,
                    label: t.cash.title,
                    dot: fiscalOn && unfiled > 0,
                  },
                ]
              : []),
            // ⚠️ **Behind the same permission as the exit button** (`canExit`,
            // which is `void` on the server) and for the reason written there:
            // a seventh permission would sit unticked in every restaurant until
            // each one found it, while this set of people is already exactly
            // right — every seeded management role holds it and no cashier,
            // waiter, barman or host does.
            //
            // ⚠️ **Last, and not near the exit button.** The exit is pinned past
            // a divider at the far end because it takes the machine out of
            // service; this changes which printer a receipt comes out of. Two
            // controls a thumb-width apart, one recoverable and one not, is the
            // arrangement that rail was rearranged to avoid.
            ...(person?.canExit
              ? [
                  {
                    id: "settings",
                    icon: <LuSettings />,
                    label: t.till.settings.title,
                  },
                ]
              : []),
          ]}
          // ⚠️ **Here rather than in the top bar**, and pinned to the far end
          // of the rail. Beside the padlock it was two similar buttons a
          // thumb-width apart, one of which locks the screen for a second and
          // one of which takes the machine out of service until somebody with
          // a panel login walks over with a fresh link. Distance is the cheapest
          // guard there is.
          //
          // ⚠️ Still `undefined` rather than disabled for anybody who may not —
          // a greyed-out control is one people keep pressing. The server
          // refuses the call either way.
          onExit={person?.canExit ? exitScreen : undefined}
          exitLabel={t.till.exit}
          value={view}
          // ⚠️ **Leaving the check lets go of it.**
          //
          // It used to stay: a cashier rang two dishes onto table 4, walked to
          // the drawer or back to the room, and the check was still sitting in
          // the right-hand column — so the next person to walk up, tap "Menyu"
          // and press a dish put it on table 4. Nothing warned anybody, the
          // line looked ordinary on the bill, and the guest who paid for it was
          // at a different table. That is the exact failure the room-first rule
          // exists to prevent, arriving through the rail instead of the menu.
          //
          // ⚠️ **The dish screen is the one exception**, because it is the same
          // piece of work: it exists to add lines to the check on the right, and
          // releasing on the way there would leave nothing to add them to.
          //
          // ⚠️ Nothing is lost by letting go. Every line is already on the check
          // on the server — including the ones the kitchen has not been told
          // about — so re-tapping the table brings all of it back with the send
          // button still there, and the room marks that table with a dot until
          // somebody does.
          onPick={(id) => {
            setView(id as View);
            if (id !== "order") release();
          }}
        />

        {/* ---- The work area ---- */}
        <section className="flex min-h-0 min-w-0 flex-1 flex-col">
          {view === "tables" && (
            <>
              <BookingsStrip active={view === "tables"} />
              <TablesScreen
                tables={tables}
                zones={zones}
                shapes={shapes}
                planWidth={plan.w}
                planHeight={plan.h}
                checks={[...checks, ...locals]}
                currency={currency}
                onOpenCheck={enter}
                onNewCheck={(tableId) => {
                  // ⚠️ The counter opens its dialog (it asks how many guests);
                  // a tapped table already answered the only question there was.
                  setOpening(true);
                  setPreTable(tableId);
                }}
              />
            </>
          )}

          {/* ⚠️ The drawer is a destination, not a panel stacked over the
              check. It used to sit above the bill in the right-hand column,
              which meant the number a guest is waiting for was pushed down the
              screen by a form nobody opens twice a day. */}
          {view === "checks" && (
            <ChecksScreen
              // ⚠️ Including the ones this device is holding offline: a check
              // the server has never heard of is still a table with people at
              // it, and a list that leaves it out is a list that is wrong on
              // exactly the day the network is.
              open={[...checks, ...locals]}
              currency={currency}
              onError={setError}
              onOpenCheck={enter}
            />
          )}

          {view === "stop" && <StopListScreen onError={setError} />}
          {view === "zakup" && <ZakupScreen onError={setError} />}
          {view === "online" && <OnlineScreen onError={setError} />}

          {/* ⚠️ Named rather than "somebody is editing this": a name sends the
          cashier to the colleague two metres away, and the anonymous version
          sends them to look for a manager. */}
          {heldWarning !== "" && (
            <div className="fixed inset-0 z-50 grid place-items-center bg-black/40 p-6">
              <div className="card w-full max-w-sm space-y-3 p-5 text-center">
                <p className="text-lg font-semibold text-amber-600">
                  {t.till.heldTitle(heldWarning)}
                </p>
                <p className="text-sm text-ink-soft">{t.till.heldBody}</p>
                <button
                  type="button"
                  className="till-btn-accent w-full"
                  onClick={() => setHeldWarning("")}
                >
                  {t.till.gotIt}
                </button>
              </div>
            </div>
          )}

          {/* ⚠️ Guarded here as well as in the rail. `view` is state, and a
              person who opened this and then locked the screen would hand the
              next cashier a settings screen that was already on it. */}
          {view === "settings" && person?.canExit && (
            <SettingsScreen version={VERSION} onError={setError} />
          )}

          {view === "cash" && (
            <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-4">
              <div className="mx-auto max-w-2xl space-y-3">
                {/* ⚠️ First, and only when it has something to say. Sales that
                    took money with no tax receipt behind them are invisible by
                    nature — the guest has gone and nothing looks wrong. */}
                <UnfiledPanel
                  currency={currency}
                  onError={setError}
                  onCount={setUnfiled}
                />
                <CashShiftPanel
                  currency={currency}
                  onError={setError}
                  onChanged={shift.reload}
                />
                {/* ⚠️ Under the drawer, not beside the checks: this is money
                    arriving for something that was sold days ago, so it belongs
                    with the shift's figures rather than with tonight's tables.
                    A cashier looking for it is looking at the drawer. */}
                <DebtsPanel currency={currency} onError={setError} />
                {/* ⚠️ Below the unfiled list: ending the tax day is refused
                    while any receipt is outstanding, so the thing that has to
                    be dealt with first is shown first — otherwise the cashier
                    meets a refusal before seeing its cause. */}
                {fiscalOn && (
                  <CloseDayButton currency={currency} onError={setError} />
                )}
              </div>
            </div>
          )}

          {/* ⚠️ **One pane differs, and only this one.** The check, the payment
              dialog, the cash shift and the navigation are the same objects in a
              shop as in a restaurant — a second till would be two of each, and
              two tills drift. What changes is how a line gets onto the check:
              tapped from a grid, or scanned. */}
          {/* ⚠️ **The scanner does not replace the cards, it sits above
              them.** A shop rings up by scanning, but a scanner that has
              stopped reading is an ordinary morning — a cable, a dead battery,
              a packet whose label has been rubbed off by the freezer — and a
              counter with nothing but a dead input on it cannot sell anything
              until somebody arrives with a new one. The same list, tapped. */}
          {view === "order" && sellsGoods && (
            <ScanPanel
              scalePort={scalePort}
              onAdd={async (item, qty) => {
                await addDish(item, undefined, qty);
              }}
            />
          )}

          {view === "order" && (
            <>
              <div className="flex shrink-0 items-center gap-2 border-b border-line bg-surface px-3 py-2.5">
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
                {/* ⚠️ **A device setting, not a company one.** Whether
                    photographs help depends on the screen and the processor in
                    front of you — facts about this monoblock, not about the
                    restaurant. A weak till turns them off without changing
                    anything for the branch next door. */}
                {/* ⚠️ Beside the search rather than in the check: it belongs
                    to the dish about to be added, and a control for the next
                    tap has to be where the next tap is.

                    ⚠️ **A kitchen's idea, so it is absent from a shop.** A
                    course is "bring this after that" said to a cook; nothing
                    behind a shop counter is served in an order, and a control
                    that cannot mean anything is a control somebody presses
                    once and then distrusts the row it sits in. */}
                {!sellsGoods && <CourseTabs value={course} onPick={setCourse} />}
                <button
                  className="till-btn w-11 shrink-0 px-0 text-base"
                  onClick={() => setShowImages(!showImages)}
                  title={t.till.toggleImages}
                  aria-pressed={showImages}
                >
                  {showImages ? "🖼" : "▦"}
                </button>
              </div>
              <MenuGrid
                menu={menu}
                items={items}
                categoryID={catID}
                onCategory={setCatID}
                query={query}
                showImages={showImages}
                currency={currency}
                // ⚠️ Never disabled in a shop: the tap opens its own check,
                // exactly as the scan does.
                disabled={!active && !sellsGoods}
                onPick={pick}
              />
            </>
          )}

          {/* ---- What can be done to this check ----

              ⚠️ **A bar along the bottom, not four more buttons under the
              total.** These are rare, deliberate actions on a check that is
              already open; stacked in the right-hand column they were the same
              size and shape as the one button the check is actually waiting
              for. Disabled until there is a check, because that is what they
              act on. */}
          <div className="no-scrollbar flex min-h-[3.75rem] shrink-0 items-center gap-2 overflow-x-auto border-t border-line bg-surface px-3">
            {/* ⚠️ **Short labels, and the row scrolls.** These were five full
                sentences in one row: on a 1024px monoblock every button wrapped
                to three lines and grew out through the bottom of the bar, over
                the dish grid. The full wording is the tooltip and the
                accessible name — length is free there. */}
            <button
              className="till-btn-quiet"
              disabled={!active || printing}
              onClick={() => void print("precheck")}
              title={t.till.precheck}
              aria-label={t.till.precheck}
            >
              <LuReceipt className="h-4 w-4" aria-hidden />
              {t.till.precheckShort}
            </button>
            <button
              className="till-btn-quiet"
              disabled={!active}
              onClick={() => setMoving(true)}
              title={t.till.moveTable}
              aria-label={t.till.moveTable}
            >
              <LuArrowRightLeft className="h-4 w-4" aria-hidden />
              {t.till.moveTableShort}
            </button>
            <button
              className="till-btn-quiet"
              // ⚠️ No longer needs a second check to exist: the first
              // destination in the dialog is a new one, which is what
              // splitting a bill is.
              disabled={!active}
              onClick={() => setMovingLines(true)}
              title={t.till.moveLines}
              aria-label={t.till.moveLines}
            >
              <LuSplit className="h-4 w-4" aria-hidden />
              {t.till.moveLinesShort}
            </button>
            {/* ⚠️ Only when there is somewhere to land. A control that opens
                onto "no other checks" teaches people it is decorative — and
                this row is read at speed with a tray in one hand. */}
            <button
              className="till-btn-quiet"
              disabled={!active || checks.length < 2}
              onClick={() => setMerging(true)}
              title={t.till.merge}
              aria-label={t.till.merge}
            >
              <LuMerge className="h-4 w-4" aria-hidden />
              {t.till.merge}
            </button>
            {canCashier && (
              <button
                className="till-btn-quiet"
                disabled={!active}
                onClick={() => setCancelling(true)}
                title={t.till.cancelCheck}
                aria-label={t.till.cancelCheck}
              >
                <LuX className="h-4 w-4" aria-hidden />
                {t.till.cancelShort}
              </button>
            )}
            <div className="flex-1" />
            {canCashier && (
              <button
                className="till-btn-quiet shrink-0 text-[rgb(var(--till-accent-ink))]"
                style={{
                  background: "rgb(var(--till-accent-tint))",
                  borderColor: "rgb(var(--till-accent-line))",
                }}
                onClick={() => setView("cash")}
              >
                {t.cash.title}
              </button>
            )}
          </div>
        </section>

        {/* ---- The check ----

            ⚠️ **On screen whenever there is one, and gone when there is not.**
            Two rules, and they were arrived at from opposite directions.

            It has to be there while a check is open, on every view: it used to
            be drawn only on the menu, so the running total — the number the
            guest is waiting to be told — disappeared the moment the cashier
            looked at the room.

            ⚠️ And it has to be **absent** otherwise, rather than sitting there
            saying "chek bo'sh". An empty column is a quarter of a 1024px
            monoblock spent on a sentence: the room loses the width its tables
            are laid out in, and the panel reads as a thing that is still
            holding something. Releasing the check now actually looks like
            releasing it. */}
        {active && (
          <aside className="flex w-full shrink-0 border-t border-line bg-surface lg:w-[20rem] lg:border-l lg:border-t-0 xl:w-[23rem] 2xl:w-[28rem]">
            <CheckPanel
              // ⚠️ **Without this a shop's counter offers to send packets to a
              // kitchen.** Lines are "unfired" until somebody fires them, and
              // nobody ever does behind a counter — so the button stayed, and
              // the button that takes the money stayed grey behind it.
              kitchen={hasKitchen}
              check={active}
              currency={currency}
              canCashier={canCashier}
              tables={tables}
              busyTables={
                checks.map((c) => c.tableId).filter(Boolean) as string[]
              }
              guest={guest}
              onGuest={setGuest}
              moving={moving}
              onMoving={setMoving}
              cancelling={cancelling}
              onCancelling={setCancelling}
              onChange={(next) => {
                setActive(next);
                void refreshChecks();
              }}
              onClosed={() => {
                setActive(null);
                setGuest(0);
                setCourse(0);
                setView("tables");
                void refreshChecks();
              }}
              onError={setError}
              onOffline={setNotice}
              onSeen={net.seen}
              onLocalFire={async () => {
                if (!isLocal(active)) return;
                setActive(await fireLocal(active as LocalCheck));
                await reloadLocals();
              }}
              onLocalQty={async (lineId, qty) => {
                if (!isLocal(active)) return;
                setActive(await setLocalQty(active as LocalCheck, lineId, qty));
                await reloadLocals();
              }}
              onLocalRemove={async (lineId) => {
                if (!isLocal(active)) return;
                setActive(await removeLocalLine(active as LocalCheck, lineId));
                await reloadLocals();
              }}
            />
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
            void addDish(picking, options, qty, undefined, portion)
          }
        />
      )}

      {scanning && active && (
        <ScanDialog
          item={scanning.item}
          // ⚠️ Every code already on this check, so the same bottle cannot be
          // scanned twice. The server checks it again — this is where it can be
          // answered by scanning the other bottle instead.
          existing={active.lines
            .filter((l) => !l.void)
            .map((l) => l.markCode ?? "")}
          busy={adding}
          onCancel={() => setScanning(null)}
          onScanned={(code) => {
            const pending = scanning;
            setScanning(null);
            void addDish(pending.item, pending.options, 1, code);
          }}
        />
      )}

      {merging && active && (
        <MergeDialog
          check={active}
          others={checks.filter((c) => c.id !== active.id)}
          currency={currency}
          busy={adding}
          onCancel={() => setMerging(false)}
          onMerge={async (intoId) => {
            setMerging(false);
            try {
              // ⚠️ The **surviving** check comes back and becomes the active
              // one: the check that was merged away no longer exists as a bill,
              // and leaving the screen on it would show a waiter an empty table
              // that still had food on it a second ago.
              setActive(await api.tillMerge(active.id, intoId));
              await refreshChecks();
            } catch (err) {
              setError(err instanceof ApiError ? err.message : t.till.retry);
            }
          }}
        />
      )}

      {movingLines && active && (
        <MoveLinesDialog
          check={active}
          others={checks.filter((c) => c.id !== active.id)}
          currency={currency}
          busy={adding}
          onCancel={() => setMovingLines(false)}
          onMove={async (lineIds, toCheckId, pin) => {
            try {
              // ⚠️ Empty means "onto a new check" — the split. The source stays
              // on screen either way: the waiter is standing at that table, and
              // a screen that jumps to the other half after dividing a bill
              // loses the person's place in the meal.
              if (toCheckId === "") {
                const res = await api.tillSplit(active.id, lineIds);
                setActive(res.check);
              } else {
                setActive(
                  await api.tillMoveLines(active.id, lineIds, toCheckId, pin),
                );
              }
              setMovingLines(false);
              await refreshChecks();
            } catch (err) {
              // ⚠️ **The dialog stays open when a manager is needed.** It is
              // holding the ticked lines, and closing it would make the code
              // cost the waiter the whole selection — which is how people learn
              // to fetch the manager *before* choosing anything, or to stop
              // using the feature.
              if (err instanceof ApiError && err.needsOverride) throw err;
              setMovingLines(false);
              setError(err instanceof ApiError ? err.message : t.till.retry);
            }
          }}
        />
      )}

      {weighing && (
        <WeightDialog
          item={weighing}
          scalePort={scalePort}
          onCancel={() => setWeighing(null)}
          onConfirm={(kg) => {
            const item = weighing;
            setWeighing(null);
            void addDish(item, undefined, kg);
          }}
        />
      )}

      {opening && (
        <NewCheckDialog
          tables={tables}
          initialTable={preTable}
          busy={checks.map((c) => c.tableId).filter(Boolean) as string[]}
          onCancel={() => setOpening(false)}
          onOpen={openCheck}
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
    roleRu: staff.roleNameRu,
    roleEn: staff.roleNameEn,
    // ⚠️ **Carried over, and it was being dropped.** On a till signed in with
    // a staff login rather than a bound monoblock, this object *is* the person
    // — so leaving `canExit` undefined hid the exit button from a manager on
    // every unbound screen. The server refuses the call regardless, so the bug
    // was invisible except as a button that was never there.
    //
    // Read from the resolved role, which is what `/staff/me` returns; the
    // legacy booleans have no `void` at all and would answer no for everybody.
    canExit: staff.perms?.includes("void"),
  };
}

/** How often a locked till asks the server what it should be showing.
 *
 *  ⚠️ Slow, because it is the whole cost of the lock screen and it runs on
 *  every idle monoblock in the country. Fifteen seconds is fast enough for a
 *  connection light to be believed and far too slow to matter. */
const LOCK_POLL_MS = 15_000;

/** What a till assumes about itself when it cannot ask.
 *
 *  ⚠️ `pinsUsed: false` on purpose: a till that cannot reach the server must
 *  still be able to sell, so the unanswerable question falls the way that lets
 *  the person through rather than the way that locks a working restaurant out
 *  of its own evening. */
const OFFLINE_SESSION: TillSession = {
  pinsUsed: false,
  brandName: "",
  branchName: "",
  banners: [],
};
