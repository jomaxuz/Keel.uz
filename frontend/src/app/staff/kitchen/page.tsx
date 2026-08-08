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
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    }
  }, [router, sound, t]);

  useEffect(() => {
    if (loading) return;
    if (!staff) {
      router.replace("/staff/login?next=/staff/kitchen");
      return;
    }
    load();
    // 10 seconds: fast enough that a cook never waits for the screen, slow
    // enough to be nothing on a tablet that stays open all day.
    const id = setInterval(load, 10_000);
    return () => clearInterval(id);
  }, [loading, staff, load, router]);

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

      {tickets === null ? (
        <p className="text-lg text-ink-muted">{t.common.loading}</p>
      ) : tickets.length === 0 ? (
        // An empty kitchen is good news and should read like it, not like a
        // screen that failed to load.
        <div className="rounded-3xl border border-line bg-surface p-10 text-center">
          <p className="text-xl font-semibold">{t.kitchen.empty}</p>
          <p className="mt-1 text-sm text-ink-muted">{t.kitchen.emptyHint}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {tickets.map((ticket) => (
            <Ticket
              key={ticket.id}
              ticket={ticket}
              busy={busy === ticket.id}
              onAct={act}
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
}: {
  ticket: KitchenTicket;
  busy: boolean;
  onAct: (id: string, action: "start" | "ready") => void;
}) {
  const t = useAdminT();
  const late = ticket.waitingMin >= VERY_LATE_MIN;
  const warn = !late && ticket.waitingMin >= LATE_MIN;
  const started = ticket.status === "preparing";

  return (
    <article
      className={`rounded-3xl border-2 bg-surface p-4 shadow-card ${
        late
          ? "border-rose-500"
          : warn
            ? "border-amber-500"
            : "border-line-strong"
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-display text-xl font-bold">#{ticket.number}</p>
          <p className="text-sm text-ink-soft">
            {t.kitchen.type[ticket.type as "delivery" | "pickup" | "dinein"] ??
              ticket.type}
            {ticket.tableNumber ? ` · ${t.kitchen.table(ticket.tableNumber)}` : ""}
          </p>
        </div>
        {/* The one number on the card, sized to be read across a kitchen. */}
        <div
          className={`shrink-0 text-right ${
            late
              ? "text-rose-600 dark:text-rose-400"
              : warn
                ? "text-amber-700 dark:text-amber-300"
                : "text-ink-soft"
          }`}
        >
          <span className="font-display text-3xl font-bold tabular-nums">
            {ticket.waitingMin}
          </span>
          <span className="ml-1 text-sm">{t.kitchen.min}</span>
        </div>
      </div>

      <ul className="mt-3 space-y-2 border-t border-line pt-3">
        {ticket.items.map((item, i) => (
          <li key={i}>
            <div className="flex items-baseline gap-2">
              <span className="font-display text-xl font-bold tabular-nums">
                {item.qty}×
              </span>
              <span className="text-lg font-semibold">{item.name}</span>
            </div>
            {item.options && item.options.length > 0 && (
              <p className="ml-8 text-sm text-ink-soft">
                {item.options.map((o) => o.choice).join(" · ")}
              </p>
            )}
            {/* The line that changes what is cooked. Loud on purpose. */}
            {item.comment && (
              <p className="ml-8 mt-1 rounded-lg bg-amber-500/15 px-2 py-1 text-base font-semibold text-amber-800 dark:text-amber-200">
                {item.comment}
              </p>
            )}
          </li>
        ))}
      </ul>

      {ticket.comment && (
        <p className="mt-3 rounded-xl border border-line px-3 py-2 text-sm text-ink-soft">
          {ticket.comment}
        </p>
      )}

      <div className="mt-4 flex gap-2">
        {!started && (
          <button
            type="button"
            disabled={busy}
            onClick={() => onAct(ticket.id, "start")}
            className="btn btn-dark flex-1 py-3 text-base disabled:opacity-40"
          >
            {t.kitchen.start}
          </button>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => onAct(ticket.id, "ready")}
          className="btn btn-primary flex-1 py-3 text-base disabled:opacity-40"
        >
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
