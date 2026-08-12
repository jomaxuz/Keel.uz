"use client";

// "Something new came in" — out loud.
//
// A kitchen does not watch the screen, so a new order or booking has to make a
// noise. The panel polls a tiny endpoint and compares the newest timestamps
// with what it last saw; when either moves, it plays a chime and shows a
// banner until it is dismissed.
//
// Two browser realities shape this:
//   • Sound cannot start on its own: a page has to be interacted with first.
//     So it is *on* by default and unlocked by the operator's first click
//     anywhere in the panel — nobody has to discover a switch to be told that
//     an order arrived. The switch only exists to turn it off, and that choice
//     is remembered.
//   • No audio file: the chime is synthesised with WebAudio. One less asset to
//     ship, and it cannot 404 on a customer's VPS.
//
// Orders and bookings ring differently (two notes vs three): a host across the
// room can tell "someone is waiting to be fed" from "someone wants a table"
// without looking at the screen.
//
// ⚠️ **An order nobody has accepted keeps ringing.** A single chime is a single
// chance to be heard, and a kitchen is loud, the tablet is across the room, and
// whoever was standing by it had their hands full. The order then sits in
// `pending` until somebody happens to look — which is the failure this whole
// component exists to prevent, arriving through the one gap it left open.
//
// So the repeat is driven by **the server's `pending` count, not by a client-side
// "unacknowledged" flag**: it is the same fact as the "Qabul qilish" button, so
// pressing that button on any device stops the sound on all of them, a second
// panel open in the office is not a second alarm nobody can silence, and a
// reloaded tab does not forget what it was ringing about. There is nothing to
// keep in sync because there is only one copy of the truth.
//
// The escape hatch is a five-minute snooze rather than a dismiss, because
// "close" on an alarm about an unaccepted order would be a lie that decays: the
// order is still unaccepted, and the next poll would prove it.
//
// Pre-orders ring twice in their life, and the two are deliberately different
// events. One when it is **placed** — news, somebody has to buy the meat — and
// one when it falls **due**, which is the instruction the whole feature exists
// for: it arrives hours later, with nobody having touched anything, and it is
// the only chime here that is not caused by a person acting right now. So it
// gets a sound of its own (a four-note alternating figure, unmistakably not the
// two-note "an order arrived") and a banner that names what to do.

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatTime } from "@/lib/format";
import { useAdminScope } from "@/lib/adminScope";
import { useAdminT } from "@/lib/i18n/admin";

const POLL_MS = 15000;
// How long the "quiet for a moment" button holds. Deliberately short: the
// reason to press it is real (the operator is on the phone about that very
// order and cannot accept it yet) and it lasts about that long. Anything
// longer becomes a way to turn the alarm off without meaning to.
const SNOOZE_MS = 5 * 60_000;
const SOUND_KEY = "admin_sound";
// The switch is rendered in both the sidebar and the phone bar, while the
// watcher runs once. They talk over this event rather than a context, which
// keeps the switch a plain button anywhere in the tree.
const SOUND_EVENT = "admin-sound-change";

function soundEnabled(): boolean {
  if (typeof window === "undefined") return true;
  return window.localStorage.getItem(SOUND_KEY) !== "0";
}

/**
 * The switch. Safe to render more than once: it owns no polling, and every
 * copy re-reads the shared preference when any of them changes it.
 */
export function SoundToggle() {
  const t = useAdminT();
  const [sound, setSound] = useState(true);

  useEffect(() => {
    setSound(soundEnabled());
    const sync = () => setSound(soundEnabled());
    window.addEventListener(SOUND_EVENT, sync);
    return () => window.removeEventListener(SOUND_EVENT, sync);
  }, []);

  return (
    <button
      type="button"
      onClick={() => {
        const next = !sound;
        window.localStorage.setItem(SOUND_KEY, next ? "1" : "0");
        // Tell the watcher (and the other copy of this switch).
        window.dispatchEvent(
          new CustomEvent(SOUND_EVENT, { detail: { preview: next } }),
        );
        setSound(next);
      }}
      title={t.booking.soundHint}
      className="flex items-center gap-1.5 rounded-full border border-line-strong px-3 py-1.5 text-xs font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand"
    >
      <span aria-hidden>{sound ? "🔔" : "🔕"}</span>
      {sound ? t.booking.soundOn : t.booking.soundOff}
    </button>
  );
}

type ChimeKind = "order" | "booking" | "preorder";

/** How many of each have arrived since the operator last cleared the banner. */
interface Fresh {
  orders: number;
  bookings: number;
  /** New pre-orders placed. */
  preorders: number;
  /** Pre-orders that have just fallen due — the kitchen's cue. */
  due: number;
}
const NOTHING_FRESH: Fresh = { orders: 0, bookings: 0, preorders: 0, due: 0 };

/** The timestamps the panel has already reacted to. */
interface Seen {
  order: string | null;
  booking: string | null;
  preorder: string | null;
  due: string | null;
}

/** The watcher: polls, chimes and shows the banner. Mount exactly once. */
export default function AlertBell() {
  const t = useAdminT();
  // Which branch this panel is listening for.
  const { scopeKey } = useAdminScope();
  // On unless it was explicitly switched off before.
  const [sound, setSound] = useState(true);
  const [fresh, setFresh] = useState<Fresh>(NOTHING_FRESH);
  // Orders nobody has accepted yet, straight from the server. While this is
  // above zero the bell repeats — see the note at the top.
  const [waiting, setWaiting] = useState(0);
  // When the operator asked for quiet. A ref rather than state: the poll reads
  // it and nothing renders from it except the label below, which re-renders on
  // its own schedule anyway.
  const [snoozeUntil, setSnoozeUntil] = useState(0);
  const snoozeRef = useRef(0);
  // What the panel had already seen. `null` until the first poll, so opening
  // the panel never announces the whole backlog.
  const seen = useRef<Seen | null>(null);
  const audioRef = useRef<AudioContext | null>(null);
  // Lets the preference listener above play a preview without depending on
  // `chime` being defined first.
  const chimeRef = useRef<((kind: ChimeKind) => void) | null>(null);

  useEffect(() => {
    setSound(soundEnabled());
    const onChange = (e: Event) => {
      setSound(soundEnabled());
      // Turning it on is a gesture browsers accept: play the chime the
      // operator just enabled so they know it works.
      if ((e as CustomEvent<{ preview?: boolean }>).detail?.preview) {
        chimeRef.current?.("booking");
      }
    };
    window.addEventListener(SOUND_EVENT, onChange);
    return () => window.removeEventListener(SOUND_EVENT, onChange);
  }, []);

  // Browsers keep audio muted until the page has been interacted with. Any
  // click in the panel counts, so the first one quietly unlocks the chime.
  useEffect(() => {
    const unlock = () => {
      try {
        const Ctor =
          window.AudioContext ??
          (window as unknown as { webkitAudioContext?: typeof AudioContext })
            .webkitAudioContext;
        if (!Ctor) return;
        const ctx = (audioRef.current ??= new Ctor());
        void ctx.resume();
      } catch {
        /* nothing to unlock */
      }
    };
    window.addEventListener("pointerdown", unlock, { once: true });
    return () => window.removeEventListener("pointerdown", unlock);
  }, []);

  const chime = useCallback((kind: ChimeKind = "order") => {
    try {
      const Ctor =
        window.AudioContext ??
        (window as unknown as { webkitAudioContext?: typeof AudioContext })
          .webkitAudioContext;
      if (!Ctor) return;
      const ctx = (audioRef.current ??= new Ctor());
      void ctx.resume();
      // Short notes: audible across a room, and nothing like a phone call.
      // Orders ring twice, bookings three times — see the note at the top.
      const notes =
        kind === "booking"
          ? [
              [0, 660],
              [0.16, 880],
              [0.32, 1320],
            ]
          : kind === "preorder"
            ? // A pre-order falling due: four notes alternating rather than
              // climbing. It has to be the one sound nobody mistakes for "an
              // order just came in", because the correct reaction is different
              // — this one means "start cooking something you accepted hours
              // ago", and there is no new row on the screen to explain it.
              [
                [0, 1175],
                [0.14, 880],
                [0.28, 1175],
                [0.42, 880],
              ]
            : [
                [0, 880],
                [0.18, 1175],
              ];
      notes.forEach(([delay, freq]) => {
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = "sine";
        osc.frequency.value = freq;
        const start = ctx.currentTime + delay;
        gain.gain.setValueAtTime(0.0001, start);
        gain.gain.exponentialRampToValueAtTime(0.35, start + 0.02);
        gain.gain.exponentialRampToValueAtTime(0.0001, start + 0.16);
        osc.connect(gain).connect(ctx.destination);
        osc.start(start);
        osc.stop(start + 0.2);
      });
    } catch {
      /* audio blocked — the banner still shows */
    }
  }, []);

  useEffect(() => {
    chimeRef.current = chime;
  }, [chime]);

  useEffect(() => {
    let stopped = false;
    // Switching branch changes which orders are being watched, so the
    // "last seen" marks belong to the branch we just left. Clearing them makes
    // the next poll a fresh baseline — otherwise the other branch's newest
    // order, which nobody here is waiting for, would ring the bell.
    seen.current = null;
    setFresh(NOTHING_FRESH);
    setWaiting(0);

    async function poll() {
      try {
        const a = await api.adminAlerts();
        if (stopped) return;
        const next: Seen = {
          order: a.orders.newestAt,
          booking: a.reservations.newestAt,
          // Absent from an older backend, which reads as "no pre-orders" —
          // the panel keeps working, it simply has nothing to announce.
          preorder: a.preorders?.newestAt ?? null,
          due: a.preorders?.dueAt ?? null,
        };
        // The alarm: orders nobody has accepted, however long ago they landed.
        //
        // ⚠️ Outside the `prev` guard on purpose. That guard exists so opening
        // the panel does not announce the whole backlog — but an unaccepted
        // order **is** the backlog that matters, and a panel opened in the
        // morning onto three orders taken overnight must say so rather than
        // wait for a fourth.
        const unaccepted = a.orders.pending ?? 0;
        setWaiting(unaccepted);

        const prev = seen.current;
        const moved = (k: keyof Seen) =>
          !!prev && !!next[k] && next[k] !== prev[k];
        const hit = {
          orders: moved("order"),
          bookings: moved("booking"),
          preorders: moved("preorder"),
          due: moved("due"),
        };
        // ⚠️ A new arrival ends the snooze. Somebody silenced the alarm about
        // *this* order, usually because they are on the phone about it — that
        // says nothing about the order that just landed, and carrying the quiet
        // over to it is how the five-minute button turns into a missed order.
        // Expiry is handled here too, so the banner returns to normal on its
        // own rather than waiting for something else to re-render it.
        if (snoozeRef.current && (hit.orders || Date.now() >= snoozeRef.current)) {
          snoozeRef.current = 0;
          setSnoozeUntil(0);
        }
        const alarm = unaccepted > 0 && sound && !snoozeRef.current;
        if (hit.orders || hit.bookings || hit.preorders || hit.due) {
          setFresh((f) => ({
            orders: f.orders + (hit.orders ? 1 : 0),
            bookings: f.bookings + (hit.bookings ? 1 : 0),
            preorders: f.preorders + (hit.preorders ? 1 : 0),
            due: f.due + (hit.due ? 1 : 0),
          }));
        }
        if (sound) {
          // Queued rather than played together: two chimes over each other
          // are one noise nobody can tell apart, which is the whole point
          // of giving them different notes. Most urgent first — a pre-order
          // that is due needs somebody at the stove now.
          const queue: ChimeKind[] = [];
          if (hit.due) queue.push("preorder");
          // The alarm already says "an order is waiting", so a new arrival
          // does not get a second chime on top of it — two order chimes in
          // one breath sound like a fault, not like two orders.
          if (alarm || hit.orders) queue.push("order");
          if (hit.preorders && !hit.due) queue.push("preorder");
          if (hit.bookings) queue.push("booking");
          queue.forEach((kind, i) => setTimeout(() => chime(kind), i * 700));
        }
        seen.current = next;
      } catch {
        /* the panel keeps working without alerts */
      }
    }

    poll();
    const id = setInterval(poll, POLL_MS);
    return () => {
      stopped = true;
      clearInterval(id);
    };
  }, [sound, chime, scopeKey]);

  const quiet = Date.now() < snoozeUntil;
  const lines = [
    // "Due now" first and on its own line: it is the only one of these that
    // does not correspond to a row appearing anywhere, so the banner is the
    // only place it is ever explained.
    fresh.due > 0 && `${t.booking.preorderDue} · ${fresh.due}`,
    fresh.orders > 0 && `${t.booking.newOrder} · ${fresh.orders}`,
    fresh.preorders > 0 && `${t.booking.newPreorder} · ${fresh.preorders}`,
    fresh.bookings > 0 && `${t.booking.newBooking} · ${fresh.bookings}`,
  ].filter(Boolean) as string[];

  return (
    // One stack, so the two banners never sit on top of each other. The alarm
    // is listed second and the column is reversed: it belongs at the bottom,
    // nearest the thumb, because it is the one with something to press.
    <div className="pointer-events-none fixed bottom-4 right-4 z-50 flex max-w-xs flex-col-reverse gap-2 [&>*]:pointer-events-auto">
      {/* The alarm has no close button: it is not a notice that something
          happened, it is a statement that something is still waiting — and it
          goes away when that stops being true, which is when somebody presses
          "Qabul qilish". */}
      {waiting > 0 && (
        <div
          className={`rounded-2xl border-2 bg-surface p-4 shadow-card-hover ${
            quiet ? "border-line-strong" : "border-brand"
          }`}
        >
          <p className="text-sm font-bold text-brand">
            {t.booking.waitingOrders(waiting)}
          </p>
          <p className="mt-1 text-xs text-ink-muted">
            {quiet
              ? t.booking.snoozedUntil(formatTime(new Date(snoozeUntil)))
              : t.booking.waitingHint}
          </p>
          <div className="mt-2 flex flex-wrap gap-2 text-xs">
            <Link href="/admin/orders" className="btn-primary px-3 py-1.5">
              {t.orders.title}
            </Link>
            {!quiet && (
              <button
                type="button"
                onClick={() => {
                  const until = Date.now() + SNOOZE_MS;
                  snoozeRef.current = until;
                  setSnoozeUntil(until);
                }}
                className="btn-ghost px-3 py-1.5"
              >
                {t.booking.snooze}
              </button>
            )}
          </div>
        </div>
      )}

      {lines.length > 0 && (
        <div className="rounded-2xl border border-brand/40 bg-surface p-4 shadow-card-hover">
          {lines.map((line, i) => (
            <p key={i} className="text-sm font-semibold">
              {line}
            </p>
          ))}
          <div className="mt-2 flex flex-wrap gap-2 text-xs">
            {(fresh.orders > 0 || fresh.due > 0 || fresh.preorders > 0) && (
              <Link
                href={
                  // Straight to the pre-order tab when that is what rang: the
                  // order is not near the top of the ordinary list — it was
                  // placed hours ago, sorted by when it arrived.
                  fresh.due > 0 || (fresh.preorders > 0 && fresh.orders === 0)
                    ? "/admin/orders?tab=preorders"
                    : "/admin/orders"
                }
                onClick={() =>
                  setFresh((f) => ({ ...f, orders: 0, preorders: 0, due: 0 }))
                }
                className="btn-primary px-3 py-1.5"
              >
                {t.orders.title}
              </Link>
            )}
            {fresh.bookings > 0 && (
              <Link
                href="/admin/reservations"
                onClick={() => setFresh((f) => ({ ...f, bookings: 0 }))}
                className="btn-ghost px-3 py-1.5"
              >
                {t.booking.title}
              </Link>
            )}
            <button
              type="button"
              onClick={() => setFresh(NOTHING_FRESH)}
              className="px-3 py-1.5 text-ink-muted hover:text-ink"
            >
              {t.common.close}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
