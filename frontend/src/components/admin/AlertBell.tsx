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

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { useAdminScope } from "@/lib/adminScope";
import { useAdminT } from "@/lib/i18n/admin";

const POLL_MS = 15000;
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

/** The watcher: polls, chimes and shows the banner. Mount exactly once. */
export default function AlertBell() {
  const t = useAdminT();
  // Which branch this panel is listening for.
  const { scopeKey } = useAdminScope();
  // On unless it was explicitly switched off before.
  const [sound, setSound] = useState(true);
  const [fresh, setFresh] = useState<{ orders: number; bookings: number }>({
    orders: 0,
    bookings: 0,
  });
  // What the panel had already seen. `null` until the first poll, so opening
  // the panel never announces the whole backlog.
  const seen = useRef<{ order: string | null; booking: string | null } | null>(
    null,
  );
  const audioRef = useRef<AudioContext | null>(null);
  // Lets the preference listener above play a preview without depending on
  // `chime` being defined first.
  const chimeRef = useRef<((kind: "order" | "booking") => void) | null>(null);

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

  const chime = useCallback((kind: "order" | "booking" = "order") => {
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
    setFresh({ orders: 0, bookings: 0 });

    async function poll() {
      try {
        const a = await api.adminAlerts();
        if (stopped) return;
        const nextOrder = a.orders.newestAt;
        const nextBooking = a.reservations.newestAt;
        const prev = seen.current;
        if (prev) {
          const newOrder = !!nextOrder && nextOrder !== prev.order;
          const newBooking = !!nextBooking && nextBooking !== prev.booking;
          if (newOrder || newBooking) {
            setFresh((f) => ({
              orders: f.orders + (newOrder ? 1 : 0),
              bookings: f.bookings + (newBooking ? 1 : 0),
            }));
            if (sound) {
              if (newOrder) chime("order");
              // Both at once: let the order ring first, then the booking.
              if (newBooking) setTimeout(() => chime("booking"), newOrder ? 700 : 0);
            }
          }
        }
        seen.current = { order: nextOrder, booking: nextBooking };
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

  const has = fresh.orders > 0 || fresh.bookings > 0;

  return (
    <>
      {has && (
        <div className="fixed bottom-4 right-4 z-50 max-w-xs rounded-2xl border border-brand/40 bg-surface p-4 shadow-card-hover">
          <p className="text-sm font-semibold">
            {fresh.orders > 0 && `${t.booking.newOrder} · ${fresh.orders}`}
            {fresh.orders > 0 && fresh.bookings > 0 && " · "}
            {fresh.bookings > 0 && `${t.booking.newBooking} · ${fresh.bookings}`}
          </p>
          <div className="mt-2 flex flex-wrap gap-2 text-xs">
            {fresh.orders > 0 && (
              <Link
                href="/admin/orders"
                onClick={() => setFresh((f) => ({ ...f, orders: 0 }))}
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
              onClick={() => setFresh({ orders: 0, bookings: 0 })}
              className="px-3 py-1.5 text-ink-muted hover:text-ink"
            >
              {t.common.close}
            </button>
          </div>
        </div>
      )}
    </>
  );
}
