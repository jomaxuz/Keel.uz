"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { api, ApiError, imageUrl } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
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

  /** The deadline as a wall clock, in the viewer's own timezone.
   *
   *  ⚠️ **Formatted from the absolute instant the server sent**, never from a
   *  duration it computed: "90 minutes left" is stale the moment it is drawn,
   *  and this screen stays open for an entire shift. */
  function clockOf(until: string): string {
    const at = new Date(until);
    return Number.isNaN(at.getTime())
      ? ""
      : at.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
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
            placeholder={t.till.stopSearch}
            className="till-input h-11 min-w-0 flex-1"
          />
          {/* The answer to "is anything off tonight", so it does not require
              reading the grid. */}
          <span className="shrink-0 text-sm font-semibold text-ink-muted">
            {offCount > 0
              ? t.till.stopOffCount(offCount)
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
          <p className="px-2 py-6 text-sm text-ink-muted">{t.till.stopEmpty}</p>
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
                                : // ⚠️ **The deadline replaces the word, it does
                                  // not sit beside it.** A badge with one line of
                                  // room says either "off" or when it comes back,
                                  // and the second answers the first. A cashier
                                  // asked "is lag'mon coming back?" reads the
                                  // card rather than reopening the dialog.
                                  row.manual && row.until
                                  ? t.till.stopUntil(clockOf(row.until))
                                  : t.till.stopOff}
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
          onConfirm={(hold) => void apply(asking, hold)}
          onLimit={(n) => void setLimit(asking, n)}
        />
      )}
    </div>
  );
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
  onLimit,
}: {
  row: StopListItem;
  onCancel: () => void;
  onConfirm: (hold?: { minutes?: number; untilClose?: boolean }) => void;
  /** Set today's batch size, or 0 to remove the limit. */
  onLimit: (limit: number) => void;
}) {
  const t = useAdminT();
  const stopping = !row.manual;
  const [limit, setLimitValue] = useState(row.limit ? String(row.limit) : "");
  /** How long the stop should hold. ⚠️ **Open-ended is the default and stays
   *  the default.** That is what this button has always done, and a deadline
   *  chosen for somebody would put a dish back on the menu that is genuinely
   *  gone — the failure nobody notices until a guest orders it. */
  const [hold, setHold] = useState<number | "close" | null>(null);

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-ink/40 p-4 sm:items-center">
      <div className="till-dialog w-full max-w-sm p-4">
        <h2 className="font-display text-lg font-bold">
          {stopping ? t.till.stopConfirmOffTitle : t.till.stopConfirmOnTitle}
        </h2>
        <p className="mt-2 text-sm text-ink-soft">
          {stopping
            ? t.till.stopConfirmOffBody(row.name)
            : t.till.stopConfirmOnBody(row.name)}
        </p>
        {/* ---- How long ----
            ⚠️ **Only when stopping.** Putting a dish back has no duration, and
            a row of times above the button would be a question nobody asked.

            ⚠️ **Buttons rather than a number field.** The dish in front of
            somebody who opened this dialog has just run out, they are standing
            at a counter, and "how many minutes" is arithmetic nobody wants to
            do at eight in the evening. Four presets and the honest default
            cover what a kitchen actually says: "for an hour", "till we close",
            "until I say so". */}
        {stopping && (
          <div className="mt-4">
            <h3 className="text-[13px] font-semibold text-ink-soft">
              {t.till.stopHoldTitle}
            </h3>
            <div className="mt-2 flex flex-wrap gap-1.5">
              {(
                [
                  [null, t.till.stopHoldOpen],
                  [60, t.till.stopHoldHours(1)],
                  [120, t.till.stopHoldHours(2)],
                  [240, t.till.stopHoldHours(4)],
                  ["close", t.till.stopHoldClose],
                ] as const
              ).map(([value, label]) => (
                <button
                  key={String(value)}
                  onClick={() => setHold(value)}
                  className={`rounded-[11px] px-3 py-2 text-[13px] font-semibold transition ${
                    hold === value
                      ? "bg-[rgb(var(--till-accent-tint))] text-[rgb(var(--till-accent-ink))]"
                      : "bg-ink/[0.05] text-ink-soft"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>
        )}

        <div className="mt-5 flex gap-2">
          {/* ⚠️ Cancel first and autofocused. The dangerous half of this dialog
              is the one reached by a second reflexive tap in the same place the
              card was, so the confirm is deliberately not under the finger. */}
          <button className="till-btn flex-1" autoFocus onClick={onCancel}>
            {t.till.stopCancel}
          </button>
          <button
            className="till-btn-primary flex-1"
            onClick={() =>
              onConfirm(
                hold === null
                  ? undefined
                  : hold === "close"
                    ? { untilClose: true }
                    : { minutes: hold },
              )
            }
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
