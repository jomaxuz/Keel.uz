"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import {
  api,
  ApiError,
  clearTillToken,
  setTillDeviceToken,
  hasTillDevice,
} from "@/lib/api";
import { useStaff } from "@/lib/staff";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice, formatTime } from "@/lib/format";
import { contentName } from "@/lib/i18n/content";
import LangSwitch from "@/components/site/LangSwitch";
import type {
  OrderItemOption,
  Check,
  FloorTable,
  TableZone,
  MenuGroup,
  MenuItem,
  TillPerson,
} from "@/lib/types";

import CheckPanel from "./CheckPanel";
import UnfiledPanel from "./UnfiledPanel";
import CloseDayButton from "./CloseDayButton";
import CashShiftPanel from "./CashShiftPanel";
import PinPad from "@/components/till/PinPad";
import MenuGrid from "@/components/till/MenuGrid";
import ShiftGate, { useShift } from "@/components/till/ShiftGate";
import OptionDialog from "@/components/till/OptionDialog";
import TablesScreen from "@/components/till/TablesScreen";
import NewCheckDialog from "@/components/till/NewCheckDialog";

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

/** Where this monoblock remembers whether it draws photographs. */
const IMAGES_KEY = "keel_till_images";

export default function TillPage() {
  const router = useRouter();
  const { staff, loading: authLoading, logout } = useStaff();
  const t = useAdminT();
  const { lang } = useI18n();

  const [menu, setMenu] = useState<MenuGroup[]>([]);
  const [tables, setTables] = useState<FloorTable[]>([]);
  const [zones, setZones] = useState<TableZone[]>([]);
  const [currency, setCurrency] = useState("UZS");
  const [checks, setChecks] = useState<Check[]>([]);
  const [active, setActive] = useState<Check | null>(null);
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
  // The table the floor tap chose, carried into the dialog so it only has to
  // ask how many guests.
  const [preTable, setPreTable] = useState("");
  // The dish waiting on an answer about its options. null = nothing is being
  // asked, which is the state a till spends almost all of its time in.
  const [picking, setPicking] = useState<MenuItem | null>(null);
  // ⚠️ A second tap while the first request is in flight adds the dish twice —
  // the exact "two lines on the check" the dialog exists to prevent, arriving
  // through the dialog itself. On a monoblock over a restaurant's wifi the
  // window is wide enough to hit by accident.
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState<string | null>(null);
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
  // Whether this branch uses PINs at all. Restaurants that have set none keep
  // working exactly as before rather than being locked out by an upgrade.
  const [pinsUsed, setPinsUsed] = useState<boolean | null>(null);
  // Whether this monoblock is bound to a branch. ⚠️ null while we have not
  // looked: rendering the login redirect before knowing would bounce a
  // perfectly good till to a form it never needs.
  const [device, setDevice] = useState<boolean | null>(null);
  // ⚠️ **The room first, the menu second.** The first question of every order
  // is who it is for; a screen that opens on dishes makes that something you
  // answer afterwards, by remembering, and that is how a round of drinks lands
  // on the wrong bill.
  const [view, setView] = useState<"tables" | "order">("tables");
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
  const unlocked = !!person || (!!staff && pinsUsed === false);
  const shift = useShift(unlocked);
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
        const [groups, restaurant] = await Promise.all([
          api.getMenu(),
          api.getRestaurant(),
        ]);
        if (!alive) return;
        setMenu(groups);
        setCatID(groups[0]?.category.id ?? "");
        // The floor plan is the serving branch's, and so is layered onto the
        // profile by the server — the same answer the booking page reads, so
        // the till cannot disagree with it about which tables exist.
        setTables(restaurant.restaurant.booking?.tables ?? []);
        // ⚠️ Nil slices arrive as null, not [] — the tab strip maps over this.
        setZones(restaurant.restaurant.booking?.zones ?? []);
        setCurrency(restaurant.restaurant.currency || "UZS");
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
  const refreshChecks = useCallback(async () => {
    try {
      const res = await api.tillChecks();
      setChecks(res.checks);
      // Keep the open check in step with the server, but only when nothing is
      // being typed into it: the panel below owns its own copy while it is
      // being edited.
      const id = activeID.current;
      if (id) {
        const fresh = res.checks.find((c) => c.id === id);
        if (!fresh) setActive(null);
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) setError(err.message);
    }
  }, []);

  useEffect(() => {
    if (!unlocked) return;
    void refreshChecks();
    // 15s, the same beat as the panel's alert poll. A till is not a chat: the
    // thing that changes underneath you is another waiter opening a table, and
    // fifteen seconds is faster than anybody can walk there.
    const timer = setInterval(() => void refreshChecks(), 15_000);
    return () => clearInterval(timer);
  }, [unlocked, refreshChecks]);

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

  // Does this screen lock? Asked once, and a failure leaves it unlocked — a
  // till that cannot reach the server must still be able to sell.
  useEffect(() => {
    if (!staff && !device) return;
    api
      .tillSession()
      .then((r) => setPinsUsed(r.pinsUsed))
      .catch(() => setPinsUsed(false));
  }, [staff, device]);

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

  async function openCheck(tableId: string, guests: number) {
    setOpening(false);
    try {
      const check = await api.tillOpenCheck({ tableId, guests });
      setActive(check);
      setView("order");
      await refreshChecks();
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
      // Optimism is wrong here: the price, the sold-out list and the brand check
      // all live on the server, and a line that appears and then vanishes is
      // worse than one that takes 200ms to appear.
      const next = await api.tillAddLines(active.id, [
        { menuItemId: item.id, qty, ...(options?.length ? { options } : {}) },
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
    if ((item.options?.length ?? 0) > 0) {
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
  // ⚠️ On a bound monoblock the pad is the **only** way in, whether or not
  // anybody has been given a PIN yet: there is no staff account behind the
  // screen to fall back to. On an older till signed in with a staff login the
  // pad appears once somebody has a code, and not before.
  if (!person && (device || pinsUsed)) {
    return <PinPad onUnlock={setPerson} />;
  }

  // ⚠️ Permissions come from whoever is unlocked, never from the device — the
  // monoblock has none of its own, and reading them off the account that
  // happens to be signed in is exactly the mix-up the PIN exists to end.
  const canCashier = person ? person.canCashier : !!staff?.canCashier;
  const canTill = person
    ? person.canWaiter || person.canCashier
    : !!staff && (staff.canWaiter || staff.canCashier);
  // ⚠️ **The drawer before the room.** A check opened with no shift open
  // belongs to no count, so the money is taken and the evening's figure is
  // quietly short — with nothing on any screen saying so. The gate is drawn
  // instead of the floor rather than above it: a banner on a working till is a
  // banner that gets worked past, because the first guest is already standing
  // there.
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
      {/* ⚠️ Dark chrome, light work area. The frame is identical under both
          themes, so the dish grid is the only thing that changes brightness and
          the controls stay where the eye learned them at eleven at night.

          ⚠️ Compact on purpose. A monoblock is usually 768px tall and every row
          here is a row the dish grid does not get — the header is chrome, and
          chrome does not sell food. */}
      <header className="till-chrome flex h-12 shrink-0 items-center gap-3 px-3">
        <h1 className="font-display text-base font-bold tracking-tight">
          {t.till.title}
        </h1>
        {/* Whoever is unlocked, not whoever set the monoblock up — that name
            is the one the journal will carry. */}
        <span className="truncate text-sm text-white/60">
          {person?.name ?? staff?.name}
        </span>
        {/* ⚠️ The open shift, named where it cannot be missed. A cashier who
            cannot see which shift they are selling into finds out at the count,
            and by then the answer is a discrepancy rather than a fact. */}
        {shift.shift && (
          <span className="hidden rounded-full bg-white/10 px-2.5 py-1 text-xs font-semibold text-white/70 sm:inline">
            {t.cash.openedAt} {formatTime(shift.shift.openedAt)}
          </span>
        )}
        <div className="ml-auto flex items-center gap-1.5">
          {/* ⚠️ No theme toggle: these screens are always light (forcedLight
              in lib/theme.tsx). A control that does nothing is worse than an
              absent one — the cashier presses it, nothing happens, and the next
              button that genuinely fails gets pressed twice too. */}
          <LangSwitch />
          {/* ⚠️ On a bound monoblock this locks rather than logs out — there
              is no account to sign out of, and clearing the device token would
              mean fetching a new link from the panel to sell anything. */}
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
        <div className="shrink-0 bg-brand/15 px-3 py-2 text-sm text-ink">
          {error}
          <button
            className="ml-3 underline"
            onClick={() => setError(null)}
            aria-label={t.till.back}
          >
            ✕
          </button>
        </div>
      )}

      <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
        {/* ---- Open checks ----

            ⚠️ **A rail, not a stack of cards.** Rounded panels with their own
            borders made twelve open checks into two screens of scrolling, and
            the cashier's question here is never "tell me about this check" — it
            is "which one is table 7". Rows answer that; cards make you read.

            ⚠️ Dark, because it is chrome. The dish grid is the only thing on
            this screen worth looking at directly. */}
        <aside className="till-chrome-soft flex w-full shrink-0 flex-col lg:w-52">
          <div className="flex items-center justify-between gap-2 px-3 pb-1 pt-2.5">
            <h2 className="till-label-on-dark">{t.till.openChecks}</h2>
            <button
              className="till-btn-primary min-h-9 px-3 text-base leading-none"
              onClick={() => {
                setPreTable("");
                setOpening(true);
              }}
              aria-label={t.till.newCheck}
            >
              +
            </button>
          </div>
          <ul className="flex gap-1 overflow-x-auto p-2 pt-1 lg:flex-col lg:overflow-y-auto">
            {checks.length === 0 && ready && (
              <li className="px-1 py-2 text-sm text-white/40">
                {t.till.noChecks}
              </li>
            )}
            {checks.map((c) => (
              <li key={c.id} className="shrink-0 lg:shrink">
                <button
                  onClick={() => {
                    setActive(c);
                    setView("order");
                  }}
                  className={`${
                    c.id === active?.id ? "till-row-on" : "till-row"
                  } min-w-36 flex-col items-stretch py-2`}
                >
                  <span className="flex items-baseline justify-between gap-2">
                    {/* The table number is what the cashier is looking for and
                        it is read from a step away — everything else on the row
                        is context. */}
                    <span className="font-display text-base font-bold leading-none">
                      {c.tableNumber
                        ? `${c.tableNumber}-${t.till.table.toLowerCase()}`
                        : t.till.counter}
                    </span>
                    {/* The most useful number on this list is the age, not the
                        amount: a table open for ninety minutes is the one
                        nobody is looking at. */}
                    <span className="text-[11px] opacity-60">
                      {c.openMin} {t.till.minShort}
                    </span>
                  </span>
                  <span className="mt-0.5 flex items-baseline justify-between gap-2">
                    <span className="text-sm opacity-80">
                      {formatPrice(c.total, currency, lang)}
                    </span>
                    {/* ⚠️ A dot, not a sentence. The rail is glanced at, and
                        "kutmoqda: 2" on every second row is a rail nobody
                        reads — the count is on the check itself. */}
                    {c.unfired > 0 && (
                      <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-brand" />
                    )}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </aside>

        {/* ---- The work area ----

            ⚠️ **The room first, the menu second, and this is the whole reason
            the screen is laid out this way.** The first question of every order
            is who it is for; a till that opens on dishes makes that something
            you answer afterwards, by remembering, and that is how a round of
            drinks lands on the wrong bill. The menu replaces the room only once
            a check is open — at which point "who is this for" has an answer
            printed at the top of the check panel. */}
        <section className="flex min-h-0 min-w-0 flex-1 flex-col border-line lg:border-r">
          {view === "tables" ? (
            <TablesScreen
              tables={tables}
              zones={zones}
              checks={checks}
              currency={currency}
              onOpenCheck={(c) => {
                setActive(c);
                setView("order");
              }}
              onNewCheck={(tableId) => {
                // ⚠️ The counter opens its dialog (it asks how many guests);
                // a tapped table already answered the only question there was.
                setOpening(true);
                setPreTable(tableId);
              }}
            />
          ) : (
            <>
          <div className="flex shrink-0 items-center gap-1.5 px-2.5 pb-1.5 pt-2">
            {/* ⚠️ Back to the room is a button, not the browser's. A monoblock
                runs fullscreen with no chrome, and a waiter who cannot get back
                to the floor opens a second check for the same table. */}
            <button
              className="till-btn w-11 shrink-0 px-0 text-base"
              onClick={() => setView("tables")}
              aria-label={t.till.floor}
            >
              ←
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
              >
                ✕
              </button>
            )}
            {/* ⚠️ **A device setting, not a company one.** Whether photographs
                help depends on the screen and the processor in front of you —
                facts about this monoblock, not about the restaurant. A weak
                till turns them off without changing anything for the branch
                next door. */}
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
            disabled={!active}
            onPick={pick}
          />
            </>
          )}
        </section>

        {/* ---- The check ---- */}
        <div className="w-full shrink-0 space-y-3 overflow-y-auto p-3 lg:w-80">
          {/* ⚠️ Above the check, and only when it has something to say. Sales
              that took money with no tax receipt behind them are invisible by
              nature — the guest has gone and nothing looks wrong — so the one
              place a cashier already looks is where it has to appear. It
              renders nothing at all when the list is empty. */}
          <UnfiledPanel currency={currency} onError={setError} />
          {/* ⚠️ Below the unfiled list on purpose. Ending the tax day is
              refused while any receipt is outstanding, so the thing that has
              to be dealt with first is the thing shown first — otherwise the
              cashier meets a refusal before seeing its cause. Cashiers only:
              a waiter cannot end a day. */}
          {/* ⚠️ Above the Z-report, because the drawer is counted first and the
              tax day is ended after: closing the register's day is refused
              while receipts are unfiled, and the shift close is what asks for
              it. Reversed, the cashier meets a refusal before doing the thing
              that triggers it. */}
          {canCashier && (
            <CashShiftPanel
              currency={currency}
              onError={setError}
              onChanged={shift.reload}
            />
          )}
          {canCashier && fiscalOn && (
            <CloseDayButton currency={currency} onError={setError} />
          )}
          {/* ⚠️ Not drawn on the floor view. Its empty state says "pick a dish
              from the menu", and on the room screen there is no menu to pick
              from — an instruction that cannot be followed teaches people to
              stop reading the panel that will later carry the total. */}
          {view === "order" && (
          <CheckPanel
            check={active}
            currency={currency}
            canCashier={canCashier}
            tables={tables}
            busyTables={
              checks.map((c) => c.tableId).filter(Boolean) as string[]
            }
            onChange={(next) => {
              setActive(next);
              void refreshChecks();
            }}
            onClosed={() => {
              setActive(null);
              void refreshChecks();
            }}
            onError={setError}
          />
          )}
        </div>
      </div>

      {picking && (
        <OptionDialog
          item={picking}
          currency={currency}
          busy={adding}
          onCancel={() => setPicking(null)}
          onAdd={(options, qty) => void addDish(picking, options, qty)}
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
