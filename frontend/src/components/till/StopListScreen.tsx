"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError, imageUrl } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useTillWords } from "@/lib/tillWords";
import {
  holdLeft,
  stopHoldBody,
  typedHold,
  STOP_HOLD_PRESETS,
  type StopHold,
} from "@/lib/stopHold";
import type { StopListItem } from "@/lib/types";

/**
 * What has run out, on the counter's own screen.
 *
 * ⚠️ **This screen exists because the list had only a panel door.** Lag'mon runs
 * out at eight, the kitchen tells whoever is nearest, and the only way to act on
 * it was an owner's login on a machine in the office. So mostly nobody did, and
 * the dish went on selling until a guest was told twenty minutes later that
 * their order is not coming.
 *
 * ⚠️ **Stopping asks first, and that is not politeness.** The two directions are
 * not symmetric. Putting a dish back is self-correcting — somebody orders it and
 * the kitchen makes it. Taking one off is *silent and permanent*: it vanishes
 * from the site, the bot and the grid, so nobody misses it, nobody complains,
 * and no screen ever mentions it again. A dish mis-tapped during service stays
 * off the menu for months, and the restaurant loses that revenue without ever
 * learning why. One tap is the right cost for an action somebody notices; it is
 * the wrong cost for one nobody does.
 *
 * ⚠️ **The grid is cards, not rows.** A cashier is looking for a dish by sight,
 * at arm's length, mid-service — a picture and a name found in one glance beats
 * a column of text read line by line. Categories are buttons for the same
 * reason: a 200-dish menu is unusable as one scroll, and the kitchen names the
 * section ("issiq taomlar") before it names the dish.
 */
export default function StopListScreen({
  onError,
}: {
  onError: (msg: string) => void;
}) {
  const t = useAdminT();
  const w = useTillWords();
  const [items, setItems] = useState<StopListItem[]>([]);
  const [query, setQuery] = useState("");
  const [cat, setCat] = useState("");
  const [only, setOnly] = useState<"" | "off" | "on">("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<Record<string, boolean>>({});
  /** The dish waiting on a yes. Null when nothing is being asked. */
  const [asking, setAsking] = useState<StopListItem | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await api.tillStopList();
      setItems(res.items);
    } catch (e) {
      onError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [onError]);

  useEffect(() => {
    void load();
  }, [load]);

  const offCount = useMemo(
    () => items.filter((i) => i.manual || i.pos || i.stock).length,
    [items],
  );

  // ⚠️ **Categories come from the menu, not from a fixed list.** A restaurant
  // with no desserts must not see a "Shirinliklar" button that filters to
  // nothing — an empty control teaches a room that the screen is decoration.
  // Same rule as the public menu's facets.
  const cats = useMemo(() => {
    const seen = new Map<string, string>();
    for (const i of items) {
      if (i.categoryId && !seen.has(i.categoryId)) {
        seen.set(i.categoryId, i.category);
      }
    }
    return [...seen.entries()].map(([id, name]) => ({ id, name }));
  }, [items]);

  // ⚠️ Filtered in the browser, never re-fetched. The whole menu is already
  // here, and a request per keystroke on a counter's connection turns the
  // fastest screen in the building into the slowest. Same reasoning as the
  // public menu search.
  const shown = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter((i) => {
      if (cat && i.categoryId !== cat) return false;
      const off = i.manual || i.pos || i.stock;
      if (only === "off" && !off) return false;
      if (only === "on" && off) return false;
      if (!q) return true;
      return (
        i.name.toLowerCase().includes(q) || i.category.toLowerCase().includes(q)
      );
    });
  }, [items, query, cat, only]);

  /** How many of this dish today.
   *
   *  ⚠️ **Not optimistic, unlike the stop toggle beside it.** The server
   *  recomputes the stop from the orders when a limit changes, so what comes
   *  back is a fact this screen cannot work out: whether raising the number put
   *  the dish back on sale. Guessing it would show a dish as available that the
   *  next guest is refused. */
  async function setLimit(row: StopListItem, limit: number) {
    setAsking(null);
    setBusy((b) => ({ ...b, [row.menuItemId]: true }));
    try {
      const res = await api.tillSetDailyLimit(row.menuItemId, limit);
      setItems((list) =>
        list.map((i) =>
          i.menuItemId === row.menuItemId
            ? { ...i, limit: res.limit, sold: res.sold, limitOff: res.limitOff }
            : i,
        ),
      );
    } catch (e) {
      onError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy((b) => {
        const copy = { ...b };
        delete copy[row.menuItemId];
        return copy;
      });
    }
  }

  /** Keep the dish off, but say when it comes back.
   *
   *  ⚠️ **The half this shipped without, and the commonest use of it.** A
   *  cashier stops somsa, then rings the kitchen: "how long?" — "fifteen
   *  minutes". Until now the only way to record that answer was to put the dish
   *  back on sale and stop it again, which is a dish briefly orderable and
   *  nobody would do it twice. The server already accepted a deadline on a dish
   *  it was already holding; only the screen refused to ask. */
  async function hold(row: StopListItem, chosen: StopHold) {
    setAsking(null);
    setBusy((b) => ({ ...b, [row.menuItemId]: true }));
    try {
      const res = await api.tillSetSoldOut(
        row.menuItemId,
        true,
        stopHoldBody(chosen),
      );
      setItems((list) =>
        list.map((i) =>
          i.menuItemId === row.menuItemId
            ? { ...i, manual: true, until: res.until }
            : i,
        ),
      );
    } catch (e) {
      onError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy((b) => {
        const copy = { ...b };
        delete copy[row.menuItemId];
        return copy;
      });
    }
  }

  /** The tap. Opens the question rather than doing the thing. */
  function ask(row: StopListItem) {
    if (busy[row.menuItemId]) return;
    // Held by the till system or the stockroom: the card is already disabled,
    // this covers the keyboard and says why rather than doing nothing.
    if (row.pos || row.stock) {
      onError(t.till.stopHint);
      return;
    }
    // ⚠️ A dish stopped by its own limit still opens the dialog: this is the
    // one screen that can lift it, by raising the number. Treating it like the
    // till's or the store's list would leave a stop with no way back.
    setAsking(row);
  }

  /** The answer. Optimistic, and it puts the card back if the server refuses —
   *  a till on a slow line that waited would be tapped again, and the second
   *  tap is the one that undoes the first. */
  async function apply(
    row: StopListItem,
    hold?: { minutes?: number; untilClose?: boolean },
  ) {
    const next = !row.manual;
    setAsking(null);
    setBusy((b) => ({ ...b, [row.menuItemId]: true }));
    setItems((list) =>
      list.map((i) =>
        i.menuItemId === row.menuItemId
          ? // ⚠️ The optimistic row clears `until` rather than guessing it. The
            // deadline is the server's arithmetic, and a countdown drawn from a
            // guess would tick down to a moment the dish does not come back.
            { ...i, manual: next, until: undefined }
          : i,
      ),
    );
    try {
      const res = await api.tillSetSoldOut(row.menuItemId, next, hold);
      // The moment the server settled on, so the card counts down from it.
      setItems((list) =>
        list.map((i) =>
          i.menuItemId === row.menuItemId ? { ...i, until: res.until } : i,
        ),
      );
    } catch (e) {
      setItems((list) =>
        list.map((i) =>
          i.menuItemId === row.menuItemId
            ? { ...i, manual: !next, until: row.until }
            : i,
        ),
      );
      onError(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy((b) => {
        const copy = { ...b };
        delete copy[row.menuItemId];
        return copy;
      });
    }
  }

  const chip = (on: boolean) =>
    `shrink-0 rounded-[12px] px-3.5 py-2 text-[14px] font-semibold transition ${
      on
        ? "bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
        : "bg-ink/[0.05] text-ink-soft hover:bg-ink/[0.09]"
    }`;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="border-b border-line px-3 py-3">
        <div className="flex flex-wrap items-center gap-3">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={w.tillStopSearch}
            className="till-input h-11 min-w-0 flex-1"
          />
          {/* The answer to "is anything off tonight", so it does not require
              reading the grid. */}
          <span className="shrink-0 text-sm font-semibold text-ink-muted">
            {offCount > 0
              ? w.tillStopOffCount(offCount)
              : t.till.stopNothingOff}
          </span>
        </div>

        {/* State first, then category: "what is off" is the question this
            screen is opened with, and it must not be behind a section. */}
        <div className="mt-2.5 flex gap-2 overflow-x-auto pb-0.5">
          <button className={chip(only === "")} onClick={() => setOnly("")}>
            {t.till.stopAll}
          </button>
          <button
            className={chip(only === "off")}
            onClick={() => setOnly("off")}
          >
            {t.till.stopOnlyOff}
          </button>
          <button className={chip(only === "on")} onClick={() => setOnly("on")}>
            {t.till.stopOnlySelling}
          </button>
        </div>

        {cats.length > 1 && (
          <div className="mt-2 flex gap-2 overflow-x-auto pb-0.5">
            <button className={chip(cat === "")} onClick={() => setCat("")}>
              {t.till.stopAll}
            </button>
            {cats.map((c) => (
              <button
                key={c.id}
                className={chip(cat === c.id)}
                onClick={() => setCat(c.id)}
              >
                {c.name}
              </button>
            ))}
          </div>
        )}
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
        {loading ? (
          <p className="px-2 py-6 text-sm text-ink-muted">{t.till.loading}</p>
        ) : shown.length === 0 ? (
          <p className="px-2 py-6 text-sm text-ink-muted">{w.tillStopEmpty}</p>
        ) : (
          <ul className="grid grid-cols-2 gap-2.5 sm:grid-cols-3 xl:grid-cols-4">
            {shown.map((row) => {
              const held = row.pos || row.stock;
              // ⚠️ `off` is what the card looks like; `held` is what it refuses.
              // A limit makes a dish look stopped and still opens — raising the
              // number is the only way back, and it is on the other side of
              // this tap.
              const off = row.manual || held || row.limitOff;
              return (
                <li key={row.menuItemId}>
                  <button
                    onClick={() => ask(row)}
                    disabled={held || busy[row.menuItemId]}
                    className={`flex h-full w-full flex-col overflow-hidden rounded-[14px] border text-left transition disabled:opacity-60 ${
                      off
                        ? "border-rose-500/45 bg-rose-500/[0.07]"
                        : "border-line bg-surface hover:border-line-strong"
                    }`}
                  >
                    <span className="relative block aspect-[4/3] w-full bg-ink/[0.05]">
                      {row.imageUrl && (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={imageUrl(row.imageUrl, 300) ?? undefined}
                          alt=""
                          className={`h-full w-full object-cover ${
                            off ? "grayscale" : ""
                          }`}
                        />
                      )}
                      {/* ⚠️ Named by who stopped it, over the picture. "Off"
                          alone would send the cashier to press a card that
                          refuses, and the refusal reads as a broken screen. */}
                      {off && (
                        <span className="absolute left-2 top-2 rounded-[9px] bg-rose-600 px-2 py-1 text-[12px] font-bold text-white">
                          {row.pos
                            ? t.till.stopByPOS
                            : row.stock
                              ? t.till.stopByStock
                              : row.limitOff
                                ? t.till.stopByLimit
                                : // ⚠️ **A running countdown replaces the word,
                                  // it does not sit beside it.** A badge has one
                                  // line of room, and "14 daq" answers the
                                  // question "is somsa coming back" that "off"
                                  // only restates — which is the question the
                                  // cashier just rang the kitchen to ask.
                                  row.manual && row.until
                                  ? null
                                  : t.till.stopOff}
                          {row.manual && row.until && (
                            <Countdown until={row.until} onDone={load} />
                          )}
                        </span>
                      )}
                    </span>
                    <span className="flex min-h-0 flex-1 flex-col px-2.5 py-2">
                      <span className="line-clamp-2 text-[14px] font-semibold leading-tight text-ink">
                        {row.name}
                      </span>
                      <span className="mt-auto truncate pt-1 text-[12px] text-ink-muted">
                        {/* ⚠️ **The two numbers replace the category, not sit
                            under it.** A card on a monoblock has one line of
                            room here, and "7 / 10 sotildi" is the only thing on
                            this screen that changes during service — the
                            category never does. It is drawn only when there is
                            a limit, so a kitchen that sets none sees exactly
                            what it saw before. */}
                        {row.limit > 0
                          ? t.till.limitSold(row.sold, row.limit)
                          : row.category}
                      </span>
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      <p className="border-t border-line px-4 py-2.5 text-[13px] text-ink-muted">
        {t.till.stopHint}
      </p>

      {asking && (
        <ConfirmStop
          row={asking}
          onCancel={() => setAsking(null)}
          onConfirm={(chosen) => void apply(asking, chosen)}
          onHold={(chosen) => void hold(asking, chosen)}
          onLimit={(n) => void setLimit(asking, n)}
        />
      )}
    </div>
  );
}

/**
 * How long is left of a timed stop, ticking.
 *
 * ⚠️ **Its own component so one second does not redraw two hundred cards.** The
 * grid is a monoblock's whole screen and this runs all evening; a tick in the
 * parent would re-render every dish on the menu once a second to move one
 * number. Here React re-renders the badge and nothing else.
 *
 * ⚠️ **Counted against the server's instant, never decremented locally.** A
 * screen left open for a shift drifts, and a drifting counter would say "2 daq"
 * about a dish that came back ten minutes ago — worse than no counter, because
 * somebody would act on it.
 *
 * ⚠️ **Reaching zero asks the list to reload rather than deciding for itself.**
 * The server is what lifts the stop, and a card that flipped on its own would
 * be a second opinion about whether a dish is on sale — the disagreement the
 * whole stop list is built to avoid.
 */
function Countdown({ until, onDone }: { until: string; onDone: () => void }) {
  const t = useAdminT();
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  const left = holdLeft(until, now);
  // ⚠️ Fired from an effect rather than during render: calling the parent's
  // setState while rendering is the warning React shows and the update it
  // then drops.
  useEffect(() => {
    if (left?.done) onDone();
  }, [left?.done, onDone]);

  if (!left || left.done) return null;
  const label =
    left.hours > 0
      ? t.till.stopLeftHm(left.hours, left.minutes)
      : left.minutes > 0
        ? t.till.stopLeftM(left.minutes)
        : // ⚠️ The last minute counts in seconds. This is the minute somebody
          // is standing there waiting for, and "0 daq" for sixty of them reads
          // as a stuck screen.
          t.till.stopLeftS(left.seconds);
  return <>{label}</>;
}

/**
 * The question before a dish leaves the menu.
 *
 * ⚠️ **It names the dish and the consequence, not just "are you sure".** A
 * confirm that only asks for a second tap is one people learn to answer without
 * reading, and then it costs a tap and prevents nothing. Naming the dish is what
 * catches the actual error — the wrong card, tapped in a hurry.
 *
 * ⚠️ **No reason field.** Unlike a void, this destroys nothing and takes no
 * money out, and a required box on the busiest screen in the building is how a
 * feature stops being used at all. The action itself is already written to the
 * activity log with the name of whoever tapped it.
 */
function ConfirmStop({
  row,
  onCancel,
  onConfirm,
  onHold,
  onLimit,
}: {
  row: StopListItem;
  onCancel: () => void;
  onConfirm: (hold?: { minutes?: number; untilClose?: boolean }) => void;
  /** Keep the dish off and say when it comes back. Only ever called for a dish
   *  that is already stopped — the other case is `onConfirm` with a hold. */
  onHold: (hold: StopHold) => void;
  /** Set today's batch size, or 0 to remove the limit. */
  onLimit: (limit: number) => void;
}) {
  const t = useAdminT();
  const w = useTillWords();
  const stopping = !row.manual;
  const [limit, setLimitValue] = useState(row.limit ? String(row.limit) : "");
  /** How long the stop should hold. ⚠️ **Open-ended is the default and stays
   *  the default.** That is what this button has always done, and a deadline
   *  chosen for somebody would put a dish back on the menu that is genuinely
   *  gone — the failure nobody notices until a guest orders it. */
  const [hold, setHold] = useState<StopHold>(null);
  /** What was typed into the minutes box, as text.
   *
   *  ⚠️ **Kept beside `hold` rather than derived from it.** The presets write
   *  numbers into `hold` too, and a box that re-rendered "60" the moment
   *  somebody tapped "1 soat" would look like it had been filled in for them —
   *  and the next tap would be editing a figure they did not type. */
  const [minutes, setMinutes] = useState("");

  /** What the box does to the choice. ⚠️ The rule is shared with the panel's
   *  copy of this control — see lib/stopHold.ts — because two readings of an
   *  emptied field is how the same word starts meaning two things. */
  function typeMinutes(text: string) {
    const { text: digits, hold: next } = typedHold(text);
    setMinutes(digits);
    setHold(next);
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-sm p-4">
        <h2 className="font-display text-lg font-bold">
          {stopping ? w.tillStopConfirmOff : t.till.stopConfirmOnTitle}
        </h2>
        <p className="mt-2 text-sm text-ink-soft">
          {stopping
            ? t.till.stopConfirmOffBody(row.name)
            : t.till.stopConfirmOnBody(row.name)}
        </p>
        {/* ---- How long ----
            ⚠️ **Shown for both halves, and they are different questions.**
            Stopping asks "how long is this off"; a dish already stopped asks
            "when is it ready", which is what the cashier just rang the kitchen
            to find out. Until this was here the only way to record "fifteen
            minutes" for a dish already off was to put it back on sale and stop
            it again — briefly orderable, and nobody does it twice.

            ⚠️ **Buttons rather than a number field.** The dish in front of
            somebody who opened this dialog has just run out, they are standing
            at a counter, and "how many minutes" is arithmetic nobody wants to
            do at eight in the evening. The presets are what a kitchen actually
            says: "fifteen minutes", "half an hour", "till we close". */}
        <div className="mt-4">
          <h3 className="text-[13px] font-semibold text-ink-soft">
            {stopping ? t.till.stopHoldTitle : t.till.stopReadyTitle}
          </h3>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {(stopping
              ? // ⚠️ Open-ended only when stopping, and preselected there: it
                // is what this button has always done. Offering it to a dish
                // that is already off would be a button that changes nothing.
                [null as StopHold, ...STOP_HOLD_PRESETS]
              : STOP_HOLD_PRESETS
            ).map((value) => (
              <button
                key={String(value)}
                onClick={() => {
                  setHold(value);
                  setMinutes("");
                }}
                className={`rounded-[11px] px-3 py-2 text-[13px] font-semibold transition ${
                  hold === value
                    ? "bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                    : "bg-ink/[0.05] text-ink-soft"
                }`}
              >
                {value === null
                  ? t.till.stopHoldOpen
                  : value === "close"
                    ? t.till.stopHoldClose
                    : value < 60
                      ? t.till.stopHoldMins(value)
                      : t.till.stopHoldHours(value / 60)}
              </button>
            ))}
          </div>
            {/* ⚠️ **The presets are the fast path, this is the honest one.** A
                kitchen says "twenty minutes" as often as it says "an hour", and
                a screen offering only round numbers makes somebody pick the
                wrong one and then forget why the dish came back early.

                ⚠️ **No "≈ 21:35" preview.** The clock that would compute it is
                the till's, and this whole feature sends a duration precisely
                because that clock cannot be trusted. The badge shows the
                server's answer a moment later, which is the one that is true. */}
            <label className="mt-2 flex items-center gap-2 text-[13px] text-ink-muted">
              {t.till.stopHoldOr}
              <input
                className="till-input h-10 w-20 text-center"
                inputMode="numeric"
                value={minutes}
                placeholder={t.till.stopHoldMinutesPh}
                onChange={(e) => typeMinutes(e.target.value)}
              />
              {t.till.stopHoldMinutes}
            </label>
            {/* ⚠️ **Its own button for a dish that is already off**, because the
                action is not the one below it: this keeps the dish stopped and
                only says when it returns, while the button below puts it back on
                sale now. Two outcomes that far apart must not share a control. */}
            {!stopping && (
              <button
                className="till-btn-primary mt-3 w-full"
                disabled={hold === null}
                onClick={() => onHold(hold)}
              >
                {t.till.stopReadySave}
              </button>
            )}
          </div>

        <div className="mt-5 flex gap-2">
          {/* ⚠️ Cancel first and autofocused. The dangerous half of this dialog
              is the one reached by a second reflexive tap in the same place the
              card was, so the confirm is deliberately not under the finger. */}
          <button className="till-btn flex-1" autoFocus onClick={onCancel}>
            {t.till.stopCancel}
          </button>
          <button
            className="till-btn-primary flex-1"
            onClick={() => onConfirm(stopHoldBody(hold))}
          >
            {stopping ? t.till.stopConfirmYesOff : t.till.stopConfirmYesOn}
          </button>
        </div>

        {/* ---- Today's batch ----
            ⚠️ **Below the stop, behind a divider, and never the first thing a
            finger lands on.** The dish in front of somebody who opened this
            dialog has usually just run out; the limit is the other errand —
            planning, done once in the morning — and putting a number field
            above the button that answers tonight's question would slow down the
            thing this screen exists for. */}
        <div className="mt-5 border-t border-line pt-4">
          <h3 className="text-sm font-semibold text-ink">{t.till.limitTitle}</h3>
          <p className="mt-1 text-[13px] text-ink-muted">{t.till.limitHint}</p>
          {row.limit > 0 && (
            <p className="mt-2 text-[13px] text-ink-soft">
              {t.till.limitSold(row.sold, row.limit)}
            </p>
          )}
          <div className="mt-2 flex gap-2">
            <input
              className="input w-24"
              inputMode="numeric"
              value={limit}
              placeholder={t.till.limitNone}
              onChange={(e) => setLimitValue(e.target.value.replace(/\D/g, ""))}
            />
            <button
              className="till-btn-primary flex-1"
              onClick={() => onLimit(Number(limit) || 0)}
            >
              {/* ⚠️ An emptied field says "never mind", and the button says so
                  rather than looking like it will save a limit of nothing. */}
              {limit === "" && row.limit > 0
                ? t.till.limitClear
                : t.till.limitSave}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
