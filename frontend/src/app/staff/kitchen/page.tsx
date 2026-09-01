"use client";

// The kitchen screen (KDS) — a tablet stood on end by the pass.
//
// This is not the orders page with bigger text. The owner's list answers "what
// is happening in my business"; a cook answers one question all evening: **what
// do I make next.** So the screen carries the ticket, the wait and one button,
// and everything else is left out on purpose — no money, no customer, no
// address, no filters. A cook who has to read past a total to find the dish is a
// cook who stops reading the screen.
//
// The design decisions that matter:
//
//   • **Oldest first, always, with no sort control.** The ticket about to become
//     a complaint is the one that has waited longest, and any other order
//     starves it.
//   • **The wait, in minutes, large.** It is the only number on the screen, and
//     it comes from the server — a tablet's clock is often wrong.
//   • **Line comments are impossible to miss.** "piyozsiz" changes what is
//     cooked; it is the one field here that is not decoration.
//   • **Ready tickets leave immediately.** A screen that keeps finished work
//     visible becomes a screen nobody trusts to be current.
//   • **It polls, it does not stream.** Every screen in this system polls
//     (AlertBell, the calls panel); a socket here would be the one exception
//     nobody maintains.

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, api } from "@/lib/api";
import { useStaff } from "@/lib/staff";
import { useAdminT } from "@/lib/i18n/admin";
import { formatTime } from "@/lib/format";
import { timeAgo } from "@/lib/orderFlow";
import {
  LuBike,
  LuShoppingBag,
  LuUtensils,
  LuClock,
  LuFlame,
  LuCheck,
  LuMessageSquare,
} from "react-icons/lu";
import type { KitchenTicket } from "@/lib/types";

/** How long a ticket may sit before the card starts saying so. Minutes, and
 *  deliberately generous: a kitchen where everything is red has a screen that
 *  says nothing. */
const LATE_MIN = 20;
const VERY_LATE_MIN = 35;

export default function KitchenPage() {
  const router = useRouter();
  const { staff, workplace, loading, logout } = useStaff();
  const t = useAdminT();

  const [tickets, setTickets] = useState<KitchenTicket[] | null>(null);
  const [error, setError] = useState("");
  // ⚠️ A refusal is not a transient error and must not be polled at. Left in
  // `error` it would re-appear every ten seconds looking like a screen that
  // keeps failing, when in fact nothing is going to change until somebody in
  // the panel ticks a box.
  const [denied, setDenied] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  // Which tickets were on screen last time, so a new one can announce itself.
  const seen = useRef<Set<string>>(new Set());
  const [sound, setSound] = useState(true);

  const load = useCallback(async () => {
    try {
      const rows = await api.staffKitchen();
      setTickets(rows);
      setError("");

      const fresh = rows.filter((r) => !seen.current.has(r.id));
      // Not on the first load: a screen that chimes once for every ticket
      // already in the kitchen when the tablet wakes up is a screen somebody
      // switches the sound off on, permanently.
      if (seen.current.size > 0 && fresh.length > 0 && sound) chime();
      seen.current = new Set(rows.map((r) => r.id));
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        router.replace("/staff/login?next=/staff/kitchen");
        return;
      }
      if (e instanceof ApiError && e.status === 403) {
        // The server's own words: it distinguishes "not given access" from
        // "account switched off", and those send the reader to different
        // people.
        setDenied(e.message);
        return;
      }
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [router, sound, t]);

  useEffect(() => {
    if (loading) return;
    if (!staff) {
      router.replace("/staff/login?next=/staff/kitchen");
      return;
    }
    if (denied) return;
    load();
    // 10 seconds: fast enough that a cook never waits for the screen, slow
    // enough to be nothing on a tablet that stays open all day.
    const id = setInterval(load, 10_000);
    return () => clearInterval(id);
  }, [loading, staff, load, router, denied]);

  /** One dish, ticked or put back.
   *
   *  ⚠️ **Written into the ticket on screen before the poll comes back.** The
   *  press and the next refresh are up to ten seconds apart on a pass; a tick
   *  that does not appear immediately is a tick somebody presses again, and the
   *  second press is an untick.
   */
  async function tickDish(ticket: KitchenTicket, index: number, ready: boolean) {
    const item = ticket.items[index];
    const ref = item.lineId ? { lineId: item.lineId } : { index };
    setTickets((prev) =>
      (prev ?? []).map((x) =>
        x.id !== ticket.id
          ? x
          : {
              ...x,
              items: x.items.map((it, i) =>
                i === index
                  ? { ...it, readyAt: ready ? new Date().toISOString() : undefined }
                  : it,
              ),
            },
      ),
    );
    try {
      const res = await api.staffKitchenItem(ticket.id, ref, ready);
      // The whole ticket is finished: it leaves the pass now rather than at the
      // next poll, for the same reason the whole-ticket button removes it.
      if (res.allReady) {
        setTickets((prev) => (prev ?? []).filter((x) => x.id !== ticket.id));
      }
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
      await load();
    }
  }

  async function act(id: string, action: "start" | "ready") {
    setBusy(id);
    try {
      await api.staffKitchenAction(id, action);
      // Removed locally as well as reloaded: on a busy pass the press and the
      // next poll are seconds apart, and a button that stays lit gets pressed
      // again.
      if (action === "ready") {
        setTickets((prev) => (prev ?? []).filter((x) => x.id !== id));
      }
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
      await load();
    } finally {
      setBusy(null);
    }
  }

  if (loading || !staff) {
    return <p className="p-6 text-lg text-ink-muted">{t.common.loading}</p>;
  }

  return (
    <div className="min-h-screen bg-cream p-4">
      <header className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-bold">{t.kitchen.title}</h1>
          <p className="text-sm text-ink-muted">
            {workplace?.name ?? ""} · {staff.name}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setSound((v) => !v)}
            className="chip"
            aria-label={t.kitchen.soundToggle}
          >
            {sound ? "🔔" : "🔕"}
          </button>
          <button type="button" onClick={() => router.push("/staff")} className="chip">
            {t.kitchen.backToClock}
          </button>
          <button type="button" onClick={logout} className="chip">
            {t.staff.logout}
          </button>
        </div>
      </header>

      {error && <p className="mb-3 text-base font-semibold text-brand">{error}</p>}

      {denied ? (
        // Its own screen rather than a red line above an empty pass: an empty
        // kitchen and a kitchen you may not see look identical otherwise, and
        // the first reads as good news.
        <div className="rounded-3xl border border-line bg-surface p-10 text-center">
          <p className="text-xl font-semibold">{t.kitchen.denied}</p>
          <p className="mt-2 text-base text-ink-muted">{denied}</p>
          <button
            type="button"
            onClick={() => router.push("/staff")}
            className="btn-primary mt-5 px-5 py-2.5"
          >
            {t.kitchen.backToClock}
          </button>
        </div>
      ) : tickets === null ? (
        <p className="text-lg text-ink-muted">{t.common.loading}</p>
      ) : tickets.length === 0 ? (
        // An empty kitchen is good news and should read like it, not like a
        // screen that failed to load.
        <div className="rounded-3xl border border-line bg-surface p-10 text-center">
          <p className="text-xl font-semibold">{t.kitchen.empty}</p>
          <p className="mt-1 text-sm text-ink-muted">{t.kitchen.emptyHint}</p>
        </div>
      ) : (
        // ⚠️ **Four across on a kitchen screen, not three.** A KDS is a fixed
        // monitor showing everything at once — scrolling is the failure mode,
        // because the ticket that scrolled off is the one that has been
        // waiting longest. The cards were sized like a web page's and held
        // six; the point is to hold the pass.
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          {tickets.map((ticket) => (
            <Ticket
              key={ticket.id}
              ticket={ticket}
              busy={busy === ticket.id}
              onAct={act}
              onTick={tickDish}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function Ticket({
  ticket,
  busy,
  onAct,
  onTick,
}: {
  ticket: KitchenTicket;
  busy: boolean;
  onAct: (id: string, action: "start" | "ready") => void;
  onTick: (ticket: KitchenTicket, index: number, ready: boolean) => void;
}) {
  const t = useAdminT();
  const late = ticket.waitingMin >= VERY_LATE_MIN;
  const warn = !late && ticket.waitingMin >= LATE_MIN;
  const started = ticket.status === "preparing";
  const done = ticket.items.filter((i) => i.readyAt).length;

  const tone = late
    ? { edge: "border-rose-500", strip: "bg-rose-500", text: "text-rose-50" }
    : warn
      ? { edge: "border-amber-500", strip: "bg-amber-500", text: "text-amber-50" }
      : { edge: "border-line-strong", strip: "bg-ink/85", text: "text-cream" };

  // ⚠️ **An icon per channel, and it is not decoration.** A cook plates a
  // delivery differently from a table — one goes in a box now, the other goes
  // out on a plate when the runner is free — and the word for it was set in the
  // same small grey as everything else on the card.
  const Channel =
    ticket.type === "delivery"
      ? LuBike
      : ticket.type === "pickup"
        ? LuShoppingBag
        : LuUtensils;

  return (
    <article
      className={`overflow-hidden rounded-2xl border-2 bg-surface shadow-card ${tone.edge}`}
    >
      {/* The header is a solid strip, so the number and the wait can be read
          from the far side of a kitchen without either of them competing with
          the food underneath. */}
      <header className={`flex items-center gap-2 px-3 py-2 ${tone.strip} ${tone.text}`}>
        <Channel className="h-5 w-5 shrink-0 opacity-90" aria-hidden />
        <span className="font-display text-lg font-bold leading-none">
          #{ticket.number}
        </span>
        {ticket.tableNumber && (
          <span className="rounded-md bg-white/20 px-1.5 py-0.5 text-sm font-semibold leading-none">
            {ticket.tableNumber}
          </span>
        )}
        {/* ⚠️ How much of the ticket is done, on the strip a cook reads from
            across the kitchen. Without it a half-finished order looks the same
            as one nobody has touched. */}
        {done > 0 && (
          <span className="rounded-md bg-white/20 px-1.5 py-0.5 text-sm font-semibold leading-none tabular-nums">
            {t.kitchen.readyOf(done, ticket.items.length)}
          </span>
        )}
        <span className="ml-auto flex items-baseline gap-1 leading-none">
          <span className="font-display text-2xl font-bold tabular-nums">
            {ticket.waitingMin}
          </span>
          <span className="text-xs opacity-80">{t.kitchen.min}</span>
        </span>
      </header>

      {ticket.scheduledAt && (
        // Wanted at a set time. It only reaches this screen once its lead time
        // has arrived, but the hour still belongs on the card: it is what the
        // cook plates to.
        <p className="flex items-center gap-1.5 bg-brand-tint px-3 py-1 text-sm font-bold text-brand-dark">
          <LuClock className="h-4 w-4" aria-hidden />
          {formatTime(ticket.scheduledAt)}
        </p>
      )}

      <ul className="space-y-1.5 px-3 py-2.5">
        {ticket.items.map((item, i) => (
          <li key={item.lineId ?? i}>
            {/* ⚠️ **The whole row is the tick.** A cook presses this with the
                back of a wrist, a knuckle or a gloved thumb; a checkbox-sized
                target beside the name is the one thing on this screen that
                would need care to hit. */}
            <button
              type="button"
              onClick={() => onTick(ticket, i, !item.readyAt)}
              className={`flex w-full items-baseline gap-2 rounded-lg px-1 py-1 text-left transition-colors ${
                item.readyAt ? "bg-emerald-500/10" : "hover:bg-ink/[0.04]"
              }`}
              aria-pressed={!!item.readyAt}
              title={item.readyAt ? t.kitchen.dishUndo : t.kitchen.dishReady}
            >
              <span
                className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-md border-2 ${
                  item.readyAt
                    ? "border-emerald-600 bg-emerald-600 text-white"
                    : "border-line-strong"
                }`}
                aria-hidden
              >
                {item.readyAt && <LuCheck className="h-4 w-4" />}
              </span>
              {/* ⚠️ The quantity in a chip rather than as text. It is the one
                  number a cook counts against what is on the bench, and beside
                  a long dish name it used to disappear into the sentence. */}
              <span className="min-w-[1.75rem] shrink-0 rounded-md bg-ink/[0.07] px-1.5 py-0.5 text-center font-display text-base font-bold tabular-nums">
                {item.qty}
              </span>
              <span
                className={`text-base font-semibold leading-snug ${
                  item.readyAt ? "text-ink-muted line-through" : ""
                }`}
              >
                {item.name}
              </span>
            </button>
            {item.readyAt && (
              <p className="ml-[4.5rem] text-sm font-medium text-emerald-700 dark:text-emerald-400">
                {t.till.readyAgo(timeAgo(item.readyAt, t.common.timeAgo))}
              </p>
            )}
            {item.options && item.options.length > 0 && (
              <p className="ml-[4.5rem] text-sm text-ink-soft">
                {item.options.map((o) => o.choice).join(" · ")}
              </p>
            )}
            {/* The line that changes what is cooked. Loud on purpose, and now
                marked as speech rather than as another grey note. */}
            {item.comment && (
              <p className="ml-[4.5rem] mt-1 flex items-start gap-1.5 rounded-lg bg-amber-500/15 px-2 py-1 text-sm font-semibold text-amber-800 dark:text-amber-200">
                <LuMessageSquare className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
                {item.comment}
              </p>
            )}
          </li>
        ))}
      </ul>

      {ticket.comment && (
        <p className="mx-3 mb-2 rounded-lg border border-line px-2.5 py-1.5 text-sm text-ink-soft">
          {ticket.comment}
        </p>
      )}

      <div className="flex gap-2 border-t border-line p-2">
        {!started && (
          <button
            type="button"
            disabled={busy}
            onClick={() => onAct(ticket.id, "start")}
            className="btn btn-dark flex flex-1 items-center justify-center gap-1.5 py-2.5 text-sm disabled:opacity-40"
          >
            <LuFlame className="h-4 w-4" aria-hidden />
            {t.kitchen.start}
          </button>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => onAct(ticket.id, "ready")}
          className="btn btn-primary flex flex-1 items-center justify-center gap-1.5 py-2.5 text-sm disabled:opacity-40"
        >
          <LuCheck className="h-4 w-4" aria-hidden />
          {t.kitchen.ready}
        </button>
      </div>
    </article>
  );
}

/** Two notes, synthesised — no audio file to 404 on a customer's VPS.
 *
 *  Deliberately the same pair the panel uses for a new order: a restaurant
 *  where the office and the kitchen chime differently for the same event is one
 *  where somebody learns to ignore one of them. */
function chime() {
  try {
    const Ctor =
      window.AudioContext ??
      (window as unknown as { webkitAudioContext?: typeof AudioContext })
        .webkitAudioContext;
    if (!Ctor) return;
    const ctx = new Ctor();
    [880, 1175].forEach((hz, i) => {
      const osc = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.frequency.value = hz;
      osc.connect(gain);
      gain.connect(ctx.destination);
      const at = ctx.currentTime + i * 0.18;
      gain.gain.setValueAtTime(0.0001, at);
      gain.gain.exponentialRampToValueAtTime(0.2, at + 0.02);
      gain.gain.exponentialRampToValueAtTime(0.0001, at + 0.16);
      osc.start(at);
      osc.stop(at + 0.18);
    });
    setTimeout(() => ctx.close(), 800);
  } catch {
    // A browser that will not make a sound before the screen is touched is
    // normal; the tickets are still on it.
  }
}
