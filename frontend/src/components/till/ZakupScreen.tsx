"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  LuCheck,
  LuChevronLeft,
  LuChevronRight,
  LuSearch,
  LuX,
} from "react-icons/lu";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import type {
  ShoppingCatalogRow,
  ShoppingDraftRow,
  ShoppingOrder,
  ShoppingSource,
} from "@/lib/types";

// The shopping list, written where the news arrives.
//
// ⚠️ **Its own section rather than a corner of the stop list.** The two screens
// answer opposite questions — "this is off the menu now" and "buy this
// tomorrow" — and a cashier reaching for one at eight in the evening must not
// land on the other. They share a subject and nothing else.
//
// ⚠️ **Three steps, because it used to be one column.** The date, the chosen
// lines, the shortage list, the search box, the quantity fields and two buttons
// were on screen at once; the person using it is standing at a counter during
// service, and every one of those was a decision competing with the others. Now
// each screen asks one question — what do we need, how much of it, and is this
// right — and the answer to the one before is visible while the next is being
// given.
//
// ⚠️ **The form starts from the shortage the store already worked out**, not
// from a blank page. A restaurant with two shopping lists — one the arithmetic
// produces and one a person types — gives the buyer no way to tell which is
// real, and the one they follow will be whichever they saw last.
//
// ⚠️ **The unit is never editable.** A market sells mint in bunches and flour
// in sacks; "5" typed into a field measured in kilos is five kilos on the shelf
// instead of a quarter of one, the figure is then twenty times too high, the
// stop list never fires, and the gap surfaces a month later at a count as an
// unexplained shortfall.
//
// ⚠️ **What was already sent is a tab, not a section.** "What did I send" and
// "sign for what arrived" are a different errand from writing tomorrow's list,
// and they were below it — so the list somebody was writing scrolled away to
// reach them.

type Draft = {
  key: string;
  ingredientId?: string;
  name: string;
  unit: string;
  qty: string;
  onHand?: number;
  /** How the market sells it, when somebody wrote it down. */
  packName?: string;
  packQty?: number;
  /** Whether the number typed counts packs. ⚠️ A flag, not a converted figure:
   *  the factor is a fact about the ingredient and the result is what somebody
   *  is sent to buy, so the server does the arithmetic. */
  pack?: boolean;
  /** Where the line will be answered from. ⚠️ **Shown, never chosen here.** The
   *  catalogue decides — sorting a list into "buy" and "fetch" is knowledge
   *  about the store, which is not the job of the person noticing the bar is
   *  empty. But it is shown, because a line filed wrongly otherwise sits all
   *  morning on a phone belonging to somebody who was never going to answer
   *  it. */
  source?: ShoppingSource;
};

/** The word for where a line goes, in the language the screen is in. */
function sourceLabel(
  source: ShoppingSource | undefined,
  t: { zakup: { fromMarket: string; fromStore: string } },
) {
  return source === "store" ? t.zakup.fromStore : t.zakup.fromMarket;
}

/** Which question is being asked. */
type Step = 1 | 2 | 3;

export default function ZakupScreen({
  onError,
}: {
  onError: (message: string) => void;
}) {
  const t = useAdminT();

  const [tab, setTab] = useState<"new" | "sent">("new");
  const [step, setStep] = useState<Step>(1);
  const [lines, setLines] = useState<Draft[]>([]);
  const [orders, setOrders] = useState<ShoppingOrder[]>([]);
  const [suggested, setSuggested] = useState<ShoppingDraftRow[]>([]);
  /** Everything the store already knows about. ⚠️ Needed as well as the
   *  shortage: a list can ask for something above its minimum, and without the
   *  catalogue the writer had to type the name — which creates a second
   *  ingredient no tech card points at. */
  const [catalog, setCatalog] = useState<ShoppingCatalogRow[]>([]);
  const [forDate, setForDate] = useState(() => {
    // ⚠️ Tomorrow, because that is what a shopping list is for. Today's has
    // already been shopped by the time anybody is standing at a till.
    const d = new Date();
    d.setDate(d.getDate() + 1);
    return d.toISOString().slice(0, 10);
  });
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState("");
  const [loadErr, setLoadErr] = useState("");
  /** The list somebody is standing over with a bag in front of them. */
  const [accepting, setAccepting] = useState<ShoppingOrder | null>(null);

  const load = useCallback(() => {
    api
      .staffBuyOrderDraft()
      .then((res) => {
        setSuggested(res.rows);
        setCatalog(res.catalog ?? []);
        setLoadErr("");
      })
      // ⚠️ **Said, not swallowed.** A refused or failed load used to leave the
      // list empty, which on this screen is indistinguishable from "the store
      // is fully stocked" — so a permission problem, an old server and a
      // healthy restaurant all looked the same, and the only one of the three
      // that needs no action is the one people assumed.
      .catch((e) =>
        setLoadErr(e instanceof ApiError ? e.message : t.zakup.loadFailed),
      );
    api
      .staffBuyOrders()
      .then((res) => setOrders(res.orders))
      .catch(() => setOrders([]));
  }, [t.zakup.loadFailed]);
  useEffect(load, [load]);

  const chosen = useMemo(
    () => new Set(lines.map((l) => l.ingredientId).filter(Boolean)),
    [lines],
  );

  /** What the picker offers: short things first, then the whole catalogue.
   *
   *  ⚠️ **Everything, without typing.** The catalogue used to appear only once
   *  somebody typed — which meant a restaurant that has never set a minimum on
   *  anything (most of them: the shortage list is opt-in per ingredient) opened
   *  this screen to an empty list and no way to discover the ingredients were
   *  there at all.
   *
   *  ⚠️ **Nothing is capped.** A cap is the same failure wearing a different
   *  hat: the one ingredient somebody cannot find is the one they type by hand,
   *  and that creates a duplicate no tech card points at. */
  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    const short = suggested.filter(
      (row) => q === "" || row.name.toLowerCase().includes(q),
    );
    const shortIds = new Set(short.map((r) => r.ingredientId));
    const rest = catalog
      .filter(
        (c) =>
          !shortIds.has(c.ingredientId) &&
          (q === "" || c.name.toLowerCase().includes(q)),
      )
      .map((c) => ({ ...c, qty: 0, onHand: 0 }) as ShoppingDraftRow);
    return [...short, ...rest];
  }, [suggested, catalog, query]);

  /** Whether what was typed names nothing the store knows about.
   *
   *  ⚠️ **Offered rather than refused.** A list somebody cannot finish writing
   *  is a list they write on paper instead, and then nothing here sees it —
   *  the lesson this product paid for with the supplier field and the void
   *  reason. The line carries the typed name and the delivery invents the
   *  ingredient later, marked for somebody to finish. */
  const unknown = useMemo(() => {
    const q = query.trim();
    if (q.length < 2) return "";
    return catalog.some((r) => r.name.toLowerCase() === q.toLowerCase())
      ? ""
      : q;
  }, [catalog, query]);

  function add(row: Partial<Draft> & { name: string }) {
    setDone("");
    setLines((cur) => [
      ...cur,
      {
        key: row.ingredientId ?? `new-${Date.now()}`,
        ingredientId: row.ingredientId,
        name: row.name,
        unit: row.unit ?? "",
        qty: row.qty ?? "",
        onHand: row.onHand,
        packName: row.packName,
        packQty: row.packQty,
        source: row.source,
      },
    ]);
    setQuery("");
  }

  /** ⚠️ **One tap on, one tap off.** The picker used to only add: taking
   *  something back meant finding it again in a second list further down the
   *  same column, which is why lists arrived with things nobody wanted. */
  function toggle(row: ShoppingDraftRow) {
    if (chosen.has(row.ingredientId)) {
      setLines((cur) => cur.filter((l) => l.ingredientId !== row.ingredientId));
      return;
    }
    add({
      ingredientId: row.ingredientId,
      name: row.name,
      unit: row.unit,
      // ⚠️ Pre-filled with what is short, not locked to it: a manager who knows
      // a holiday is coming buys more.
      qty: row.qty > 0 ? String(row.qty) : "",
      onHand: row.onHand,
      packName: row.packName,
      packQty: row.packQty,
      source: row.source,
    });
  }

  const ready = lines.filter((l) => Number(l.qty) > 0);
  const storeCount = ready.filter((l) => l.source === "store").length;
  const marketCount = ready.length - storeCount;
  const waiting = orders.filter((o) => o.status === "shipped").length;

  async function send() {
    if (ready.length === 0) {
      onError(t.zakup.nothingToSend);
      return;
    }
    setBusy(true);
    try {
      await api.staffCreateBuyOrder({
        forDate,
        lines: ready.map((l) => ({
          ingredientId: l.ingredientId,
          name: l.name,
          qty: Number(l.qty),
          pack: l.pack,
        })),
      });
      setLines([]);
      setStep(1);
      // ⚠️ **The split is said out loud after sending.** The writer chose none
      // of it, so a plain "sent" would leave them with no way to notice that
      // the lemons they meant for the market went to a storekeeper — which is
      // the one mistake this routing can make, and the one a person can fix in
      // the catalogue in ten seconds if they are told.
      setDone(
        storeCount > 0 && marketCount > 0
          ? t.zakup.sentSplit(marketCount, storeCount)
          : t.zakup.sent(ready.length),
      );
      load();
    } catch (e) {
      onError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  // ---- What the step's forward button says, and whether it may be pressed ----
  const canGo =
    step === 1
      ? lines.length > 0
      : step === 2
        ? ready.length > 0
        : ready.length > 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col bg-[rgb(var(--till-floor))]">
      {/* ---- Which errand ----
          ⚠️ Two tabs rather than two sections: writing tomorrow's list and
          signing for this morning's delivery are different jobs, and the second
          used to sit below the first — so the list being written scrolled away
          to reach it. */}
      {/* ⚠️ **One column, centred, like the checks screen.** On a 1920px till
          the list, the steps and the send bar ran edge to edge — a quantity box
          a metre from the name it belonged to, and a single "+" per row far to
          the right of everything. Every band below keeps its full-width
          background and puts its contents in this column. */}
      <header className="border-b border-line bg-surface px-3 py-2">
        <div className="mx-auto flex w-full max-w-3xl items-center gap-2">
          <div className="till-seg-track">
            <button
              className={tab === "new" ? "till-seg-on" : "till-seg"}
              onClick={() => setTab("new")}
            >
              {t.zakup.tabNew}
            </button>
            <button
              className={tab === "sent" ? "till-seg-on" : "till-seg"}
              onClick={() => setTab("sent")}
            >
              {t.zakup.tabSent}
              {/* The one number worth carrying on a tab: a delivery standing on
                the counter waiting to be signed for. */}
              {waiting > 0 && (
                <span className="ml-1.5 rounded-full bg-[rgb(var(--till-accent))] px-1.5 text-[12px] font-bold text-white">
                  {waiting}
                </span>
              )}
            </button>
          </div>

          {tab === "new" && (
            <label className="ml-auto flex items-center gap-2 text-[13px] text-ink-muted">
              {t.zakup.forDate}
              <input
                type="date"
                className="till-input h-11 w-auto"
                value={forDate}
                onChange={(e) => setForDate(e.target.value)}
              />
            </label>
          )}
        </div>
      </header>

      {tab === "sent" ? (
        <SentList orders={orders} t={t} onAccept={setAccepting} />
      ) : (
        <>
          {/* ---- Where we are ----
              ⚠️ Numbered and named, and the steps already answered are
              pressable: a person who mistyped a quantity should not have to
              send the list to fix it. The one ahead is not, because its answer
              depends on this one. */}
          <nav className="border-b border-line bg-surface px-3 py-2">
            <div className="mx-auto flex w-full max-w-3xl items-center gap-1">
              {([1, 2, 3] as Step[]).map((n, i) => (
                <button
                  key={n}
                  disabled={n > step}
                  onClick={() => setStep(n)}
                  className={`flex items-center gap-2 rounded-[12px] px-2.5 py-1.5 text-[13px] font-semibold transition ${
                    n === step
                      ? "bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                      : n < step
                        ? "text-ink-soft hover:bg-ink/[0.05]"
                        : "text-ink-muted/60"
                  }`}
                >
                  <span
                    className={`grid h-6 w-6 place-items-center rounded-full text-[12px] ${
                      n < step
                        ? "bg-[rgb(var(--till-accent))] text-white"
                        : n === step
                          ? "bg-[rgb(var(--till-accent-ink))] text-white"
                          : "bg-ink/[0.08] text-ink-muted"
                    }`}
                  >
                    {n < step ? <LuCheck className="h-3.5 w-3.5" /> : n}
                  </span>
                  {n === 1
                    ? t.zakup.step1
                    : n === 2
                      ? t.zakup.step2
                      : t.zakup.step3}
                  {i < 2 && <span className="ml-1 text-ink-muted/40">›</span>}
                </button>
              ))}
            </div>
          </nav>

          {done !== "" && (
            <p className="border-b border-line bg-[rgb(var(--till-accent-tint))] px-3 py-2 text-sm font-semibold text-[rgb(var(--till-accent-ink))]">
              <span className="mx-auto block w-full max-w-3xl">{done}</span>
            </p>
          )}

          <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
            <div className="mx-auto w-full max-w-3xl">
              {step === 1 && (
                <>
                  {/* ⚠️ The search is the first thing under the thumb: the
                    shortage list answers most mornings, and the rest of the
                    catalogue is reached by typing three letters. */}
                  <div className="relative mb-3">
                    <LuSearch className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-muted" />
                    <input
                      className="till-input h-12 w-full pl-9"
                      placeholder={t.zakup.search}
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                    />
                  </div>

                  {loadErr !== "" ? (
                    <p className="py-4 text-sm text-danger">{loadErr}</p>
                  ) : shown.length === 0 && unknown === "" ? (
                    <p className="py-4 text-sm text-ink-muted">
                      {t.zakup.nothingShort}
                    </p>
                  ) : (
                    <>
                      <p className="mb-2 text-[13px] text-ink-muted">
                        {t.zakup.pickHint}
                      </p>
                      {/* Two to a row on a monoblock: the names are short and a
                        single column made the shortage list four screens long. */}
                      <ul className="grid gap-2 sm:grid-cols-2">
                        {shown.map((row) => {
                          const on = chosen.has(row.ingredientId);
                          return (
                            <li key={row.ingredientId}>
                              <button
                                onClick={() => toggle(row)}
                                aria-pressed={on}
                                className={`flex w-full items-center gap-2.5 rounded-[14px] border p-3 text-left transition ${
                                  on
                                    ? "border-[rgb(var(--till-accent))] bg-[rgb(var(--till-accent-tint))]"
                                    : "border-line bg-surface hover:border-ink/20"
                                }`}
                              >
                                <span className="min-w-0 flex-1">
                                  <span className="flex items-center gap-1.5">
                                    <span className="truncate text-[15px] font-semibold">
                                      {row.name}
                                    </span>
                                    {row.qty > 0 && (
                                      <span className="shrink-0 rounded-full bg-danger/10 px-1.5 py-0.5 text-[11px] font-semibold text-danger">
                                        {t.zakup.shortBadge}
                                      </span>
                                    )}
                                  </span>
                                  <span className="mt-0.5 block text-[13px] text-ink-muted">
                                    {/* A row that came from the catalogue rather
                                      than the shortage has no figures to show —
                                      only its unit, which is what the writer
                                      needs before typing a number into it. */}
                                    {row.qty > 0
                                      ? `${t.zakup.onHand(row.onHand, row.unit)} · ${t.zakup.need(row.qty, row.unit)}`
                                      : t.zakup.unitIs(row.unit)}
                                  </span>
                                </span>
                                <span
                                  className={`grid h-8 w-8 shrink-0 place-items-center rounded-full text-[18px] ${
                                    on
                                      ? "bg-[rgb(var(--till-accent))] text-white"
                                      : "bg-ink/[0.06] text-ink-soft"
                                  }`}
                                >
                                  {on ? <LuCheck className="h-4 w-4" /> : "+"}
                                </span>
                              </button>
                            </li>
                          );
                        })}
                        {unknown !== "" && (
                          <li className="sm:col-span-2">
                            <button
                              className="w-full rounded-[14px] border border-dashed border-line bg-surface p-3 text-left text-[15px]"
                              onClick={() => add({ name: unknown })}
                            >
                              {t.zakup.addNew(unknown)}
                            </button>
                          </li>
                        )}
                      </ul>
                    </>
                  )}
                </>
              )}

              {step === 2 && (
                <>
                  <p className="mb-2 text-[13px] text-ink-muted">
                    {t.zakup.qtyHint}
                  </p>
                  {lines.length === 0 ? (
                    <p className="py-4 text-sm text-ink-muted">
                      {t.zakup.nothingChosen}
                    </p>
                  ) : (
                    <ul className="space-y-2">
                      {lines.map((l) => (
                        <li
                          key={l.key}
                          className="flex items-center gap-2.5 rounded-[14px] border border-line bg-surface p-3"
                        >
                          <span className="min-w-0 flex-1">
                            <span className="block truncate text-[15px] font-semibold">
                              {l.name}
                            </span>
                            <span className="text-[13px] text-ink-muted">
                              {/* ⚠️ Where it goes is on every row, not only where
                                it is surprising: a badge that appears sometimes
                                is one people stop reading, and the row it is
                                missing from is the one that needed it. */}
                              {sourceLabel(l.source, t)}
                              {l.onHand !== undefined &&
                                ` · ${t.zakup.onHand(l.onHand, l.unit)}`}
                            </span>
                          </span>
                          <input
                            className="till-input h-12 w-24 text-center text-[17px] font-semibold"
                            inputMode="decimal"
                            autoFocus={lines.length === 1}
                            value={l.qty}
                            onChange={(e) =>
                              setLines((cur) =>
                                cur.map((x) =>
                                  x.key === l.key
                                    ? {
                                        ...x,
                                        qty: e.target.value.replace(",", "."),
                                      }
                                    : x,
                                ),
                              )
                            }
                          />
                          {/* ⚠️ **Tapped, not typed.** Where a market packaging is
                            written down the unit becomes a two-way switch —
                            kilos or bunches — and the conversion is the
                            server's. Where none is, this is a label and the unit
                            stays the store's, which is the whole reason the
                            field is never free text. */}
                          {l.packName && l.packQty ? (
                            <button
                              className="h-12 w-[4.5rem] shrink-0 rounded-[12px] bg-ink/[0.06] px-1 text-[13px] font-semibold text-ink-soft"
                              onClick={() =>
                                setLines((cur) =>
                                  cur.map((x) =>
                                    x.key === l.key
                                      ? { ...x, pack: !x.pack }
                                      : x,
                                  ),
                                )
                              }
                            >
                              {l.pack ? l.packName : l.unit}
                            </button>
                          ) : (
                            <span className="w-[4.5rem] shrink-0 text-center text-[13px] text-ink-muted">
                              {l.unit}
                            </span>
                          )}
                          <button
                            className="grid h-10 w-10 shrink-0 place-items-center rounded-full text-ink-muted hover:bg-ink/[0.06]"
                            aria-label={l.name}
                            onClick={() =>
                              setLines((cur) =>
                                cur.filter((x) => x.key !== l.key),
                              )
                            }
                          >
                            <LuX className="h-4 w-4" />
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </>
              )}

              {step === 3 && (
                /* ---- The last look before it is sent ----
                 ⚠️ **A list is somebody else's morning.** The buyer will not be
                 able to ask what "5" meant, so the numbers and the units are
                 read back once, in the words they will arrive in. */
                <>
                  <p className="mb-1 text-[15px] font-semibold">
                    {t.zakup.previewBody(forDate)}
                  </p>
                  {storeCount > 0 && marketCount > 0 && (
                    <p className="mb-2 text-[13px] text-ink-muted">
                      {t.zakup.splitNote(marketCount, storeCount)}
                    </p>
                  )}
                  {ready.length === 0 ? (
                    <p className="py-4 text-sm text-ink-muted">
                      {t.zakup.nothingToSend}
                    </p>
                  ) : (
                    <ul className="mt-2 overflow-hidden rounded-[14px] border border-line bg-surface">
                      {ready.map((l) => (
                        <li
                          key={l.key}
                          className="flex items-center justify-between gap-2 border-b border-line px-3 py-2.5 last:border-b-0"
                        >
                          <span className="min-w-0">
                            <span className="block truncate text-[15px]">
                              {l.name}
                            </span>
                            <span className="text-[12px] text-ink-muted">
                              {sourceLabel(l.source, t)}
                            </span>
                          </span>
                          {/* ⚠️ Read back in both, where they differ: the list
                            travels to somebody else's morning and "2" has to be
                            unambiguous before it leaves. */}
                          <span className="shrink-0 text-[15px] font-bold tabular-nums">
                            {l.pack && l.packQty
                              ? `${l.qty} ${l.packName} = ${Number(l.qty) * l.packQty} ${l.unit}`
                              : `${l.qty} ${l.unit}`}
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </>
              )}
            </div>
          </div>

          {/* ---- One way forward ----
              ⚠️ Exactly one accent control (globals.css), and it is always the
              thing that moves this screen on. Back is quiet and on the left,
              where a thumb expects it. */}
          <div className="border-t border-line bg-surface px-3 py-2.5">
            <div className="mx-auto flex w-full max-w-3xl items-center gap-3">
              {step > 1 ? (
                <button
                  className="till-btn flex items-center gap-1.5 px-4"
                  onClick={() => setStep((s) => (s - 1) as Step)}
                >
                  <LuChevronLeft className="h-4 w-4" />
                  {t.zakup.back}
                </button>
              ) : (
                <span />
              )}
              <span className="ml-auto text-sm text-ink-muted">
                {t.zakup.chosen(step === 1 ? lines.length : ready.length)}
              </span>
              {step < 3 ? (
                <button
                  className="till-btn-accent flex items-center gap-1.5 px-5"
                  disabled={!canGo}
                  onClick={() => setStep((s) => (s + 1) as Step)}
                >
                  {t.zakup.next}
                  <LuChevronRight className="h-4 w-4" />
                </button>
              ) : (
                <button
                  className="till-btn-accent px-6"
                  disabled={!canGo || busy}
                  onClick={() => void send()}
                >
                  {t.zakup.send}
                </button>
              )}
            </div>
          </div>
        </>
      )}

      {accepting && (
        <AcceptDialog
          order={accepting}
          onClose={() => setAccepting(null)}
          onDone={() => {
            setAccepting(null);
            load();
          }}
          onError={onError}
        />
      )}
    </div>
  );
}

/** ---- Lists already sent ----
 *
 *  ⚠️ Its own tab, and its own component: it answers "what happened to what I
 *  asked for", which is a different errand from writing tomorrow's list. Below
 *  it, as a section, it pushed the list being written off the screen. */
function SentList({
  orders,
  t,
  onAccept,
}: {
  orders: ShoppingOrder[];
  t: ReturnType<typeof useAdminT>;
  onAccept: (o: ShoppingOrder) => void;
}) {
  if (orders.length === 0) {
    return (
      <p className="p-6 text-center text-sm text-ink-muted">{t.zakup.noSent}</p>
    );
  }
  return (
    <ul className="mx-auto min-h-0 w-full max-w-3xl flex-1 space-y-2 overflow-y-auto p-3">
      {orders.slice(0, 20).map((o) => (
        <li
          key={o.id}
          className="rounded-[14px] border border-line bg-surface p-3 text-[14px]"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="font-semibold">
              {o.forDate}
              <span className="ml-2 text-[13px] font-normal text-ink-muted">
                {sourceLabel(o.source, t)}
              </span>
            </span>
            <span
              className={
                o.status === "done"
                  ? "text-[13px] text-ink-muted"
                  : "text-[13px] font-semibold text-[rgb(var(--till-accent-ink))]"
              }
            >
              {o.status === "done"
                ? t.zakup.statusDone
                : o.status === "shipped"
                  ? t.zakup.statusShipped
                  : t.zakup.statusSent}
            </span>
          </div>
          {/* ⚠️ "Asked for ten, brought six" is the sentence this whole document
              exists to make possible, so the counts are shown together rather
              than only the result. */}
          <p className="text-[13px] text-ink-muted">
            {t.zakup.progress(
              o.lines.filter((l) => l.gotAt && !l.missing).length,
              o.lines.length,
            )}
            {o.createdBy ? ` · ${o.createdBy}` : ""}
          </p>
          {/* ⚠️ **The button only exists on the list that is waiting for it.**
              "Signed for" is the state this whole feature was built to make
              reachable, and an accept button on a list nobody has shopped yet
              would answer a question nobody has asked. */}
          {o.status === "shipped" && (
            <button
              className="mt-2 rounded-[10px] bg-[rgb(var(--till-accent-tint))] px-3 py-1.5 text-[13px] font-semibold text-[rgb(var(--till-accent-ink))]"
              onClick={() => onAccept(o)}
            >
              {t.zakup.accept}
            </button>
          )}
        </li>
      ))}
    </ul>
  );
}

/** ---- Counting what turned up ----
 *
 *  ⚠️ **What reaches the shelf is what the restaurant counted**, not what the
 *  buyer says he handed over. Where the two differ, the difference is the
 *  record — it is the thing that had nowhere to be written before, and the
 *  reason a purchase is created here rather than at the market.
 *
 *  ⚠️ **A row left alone is accepted as sent.** Somebody who signs without
 *  retyping anything is saying "this is right", which is the ordinary case; a
 *  form that demanded every figure again would be a form people close. */
function AcceptDialog({
  order,
  onClose,
  onDone,
  onError,
}: {
  order: ShoppingOrder;
  onClose: () => void;
  onDone: () => void;
  onError: (message: string) => void;
}) {
  const t = useAdminT();
  const [counted, setCounted] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  /** ⚠️ Minted once per dialog, not per attempt: a phone that retried after a
   *  timeout must not write a second delivery. */
  const [clientId] = useState(
    () => `acc-${order.id}-${Math.random().toString(36).slice(2, 10)}`,
  );

  const sent = order.lines.filter((l) => l.gotAt && !l.missing);

  async function send() {
    setBusy(true);
    try {
      await api.staffAcceptBuyOrder(order.id, {
        clientId,
        // Only what differs. A row nobody touched keeps what it was told.
        lines: sent
          .filter((l) => counted[l.id] !== undefined && counted[l.id] !== "")
          .map((l) => ({ lineId: l.id, qty: Number(counted[l.id]) }))
          .filter((l) => !Number.isNaN(l.qty) && l.qty >= 0),
      });
      onDone();
    } catch (e) {
      onError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog max-h-[85dvh] w-full max-w-sm overflow-y-auto p-4">
        <h2 className="font-display text-lg font-bold">
          {t.zakup.acceptTitle}
        </h2>
        <p className="mt-1 text-sm text-ink-soft">{t.zakup.acceptBody}</p>
        <ul className="mt-3 divide-y divide-line text-sm">
          {sent.map((l) => (
            <li key={l.id} className="flex items-center gap-2 py-1.5">
              <span className="min-w-0 flex-1 truncate">
                {l.name}
                <span className="ml-1 text-xs text-ink-muted">
                  {l.gotQty} {l.unit}
                  {/* The price is the buyer's own note and is shown, not
                      editable: a figure retyped at a back door would rewrite
                      the price history of every dish the ingredient is in. */}
                  {l.price ? ` · ${formatPrice(l.price)}` : ""}
                </span>
              </span>
              <input
                className="till-input h-11 w-24 text-center"
                inputMode="decimal"
                placeholder={String(l.gotQty ?? "")}
                value={counted[l.id] ?? ""}
                onChange={(e) =>
                  setCounted((cur) => ({
                    ...cur,
                    [l.id]: e.target.value.replace(",", "."),
                  }))
                }
              />
            </li>
          ))}
        </ul>
        <div className="mt-5 flex gap-2">
          <button className="till-btn flex-1" autoFocus onClick={onClose}>
            {t.zakup.back}
          </button>
          <button
            className="till-btn-primary flex-1"
            disabled={busy}
            onClick={() => void send()}
          >
            {t.zakup.acceptSend}
          </button>
        </div>
      </div>
    </div>
  );
}
