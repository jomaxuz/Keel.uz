"use client";

// The call centre desk.
//
// One screen, three columns of work: find out who is calling, do the thing they
// rang about, write down what came of it. The operator never leaves it — an
// order, a booking and a callback are all reachable from here, because a call
// that makes somebody open three tabs is a call the guest waits through.
//
// The desk deliberately does **not** have its own role. A manager answering the
// phone during a rush is the same person who will confirm the order two minutes
// later; splitting that in two would mean logging out to do half the job.

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatPrice, formatUzPhone } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import CallerCard from "@/components/admin/CallerCard";
import CallLog, { OutcomeBadge } from "@/components/admin/CallLog";
import OperatorOrderModal, {
  type SeedLine,
} from "@/components/admin/OperatorOrderModal";
import type {
  Call,
  CallOutcome,
  CallStats,
  CallerLookup,
  CallerOrder,
  Order,
} from "@/lib/types";

const OUTCOMES: CallOutcome[] = [
  "order",
  "booking",
  "info",
  "complaint",
  "callback",
  "refused",
  "missed",
  "spam",
];

export default function AdminCallsPage() {
  const t = useAdminT();
  const { lang } = useI18n();

  const [phone, setPhone] = useState("");
  const [lookup, setLookup] = useState<CallerLookup | null>(null);
  const [looking, setLooking] = useState(false);
  const [error, setError] = useState("");

  const [direction, setDirection] = useState<"in" | "out">("in");
  const [outcome, setOutcome] = useState<CallOutcome | "">("");
  const [note, setNote] = useState("");
  const [callbackAt, setCallbackAt] = useState("");
  const [saving, setSaving] = useState(false);
  const [savedMsg, setSavedMsg] = useState("");

  // The call this screen is currently about, once it has been written down.
  // Placing an order afterwards links the two.
  const [callId, setCallId] = useState("");
  const [ordering, setOrdering] = useState(false);
  const [seed, setSeed] = useState<SeedLine[] | undefined>();
  const [createdOrder, setCreatedOrder] = useState<Order | null>(null);

  // The call the exchange says is ringing right now. When one appears, the
  // caller's card opens by itself — the whole point of wiring up a PBX is that
  // the operator is already looking at the customer when they say hello.
  const [live, setLive] = useState<Call | null>(null);
  const [myExtension, setMyExtension] = useState("");
  const [dialing, setDialing] = useState(false);
  const [dialMsg, setDialMsg] = useState("");

  const [stats, setStats] = useState<CallStats | null>(null);
  const [logKey, setLogKey] = useState(0);
  const [currency, setCurrency] = useState("UZS");

  // How long this call has been running. Started when the number is looked up,
  // which is the closest thing the panel has to "the operator picked up".
  const startedAt = useRef<number | null>(null);
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    api
      .getRestaurant()
      .then((r) => setCurrency(r.restaurant.currency || "UZS"))
      .catch(() => {});
  }, []);

  const loadStats = useCallback(() => {
    api.adminCallStats().then(setStats).catch(() => setStats(null));
  }, []);
  useEffect(loadStats, [loadStats]);

  // Polling rather than a socket, like the new-order chime: one small request,
  // no new infrastructure, and nothing to reconnect after a laptop lid closes.
  useEffect(() => {
    let cancelled = false;
    // The id of the ringing call we have already reacted to, so the screen
    // pops once per call rather than every three seconds for its whole life.
    let handled = "";
    async function tick() {
      try {
        const res = await api.liveCall();
        if (cancelled) return;
        setMyExtension(res.extension ?? "");
        if (res.ringing && res.call) {
          setLive(res.call);
          if (res.call.id !== handled) {
            handled = res.call.id;
            // Only take over the screen when the operator is not already in
            // the middle of writing up a different call — losing a half-typed
            // note to somebody else's incoming call would be worse than
            // missing the pop.
            if (!lookup || outcome === "") {
              setCallId(res.call.id);
              setDirection(res.call.direction);
              doLookup(res.call.phone);
            }
          }
        } else {
          setLive(null);
        }
      } catch {
        // A failing poll must never break the desk: an operator can still
        // look a number up by hand.
      }
    }
    tick();
    const timer = setInterval(tick, 3000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lookup, outcome]);

  useEffect(() => {
    if (startedAt.current === null) return;
    const timer = setInterval(() => {
      setElapsed(Math.floor((Date.now() - startedAt.current!) / 1000));
    }, 1000);
    return () => clearInterval(timer);
  }, [lookup]);

  async function doLookup(raw?: string) {
    const value = (raw ?? phone).trim();
    if (value.replace(/\D/g, "").length < 7) {
      setError(t.calls.phoneInvalid);
      return;
    }
    setPhone(value);
    setLooking(true);
    setError("");
    try {
      const data = await api.adminLookup(value);
      setLookup(data);
      startedAt.current = Date.now();
      setElapsed(0);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.loadFailed);
    } finally {
      setLooking(false);
    }
  }

  function reset() {
    setPhone("");
    setLookup(null);
    setOutcome("");
    setNote("");
    setCallbackAt("");
    setCallId("");
    setSeed(undefined);
    setCreatedOrder(null);
    setSavedMsg("");
    setError("");
    startedAt.current = null;
    setElapsed(0);
  }

  async function saveCall() {
    if (!outcome) return;
    if (outcome === "callback" && !callbackAt) {
      setError(t.calls.callbackRequired);
      return;
    }
    setSaving(true);
    setError("");
    try {
      const name = lookup?.user
        ? [lookup.user.firstName, lookup.user.lastName]
            .filter(Boolean)
            .join(" ")
            .trim()
        : "";
      const body = {
        outcome,
        note: note.trim(),
        callbackAt: callbackAt || undefined,
        seconds: elapsed,
        orderId: createdOrder?.id,
      };
      // The row may already exist: an order placed mid-call creates it, and a
      // correction afterwards must edit that row rather than log the call twice.
      const saved = callId
        ? await api.updateCall(callId, { ...body, name })
        : await api.createCall({
            ...body,
            direction,
            phone: lookup?.phone ?? phone,
            userId: lookup?.user?.id,
            name,
          });
      setCallId(saved.id);
      setSavedMsg(t.calls.saved);
      setLogKey((k) => k + 1);
      loadStats();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  async function dial(phone: string) {
    setDialing(true);
    setDialMsg("");
    try {
      const res = await api.dial(phone, callId || undefined);
      setDialMsg(res.ok ? t.calls.dialing : (res.message ?? ""));
      if (res.ok) setDirection("out");
    } catch (e) {
      setDialMsg(e instanceof Error ? e.message : "");
    } finally {
      setDialing(false);
    }
  }

  function repeat(order: CallerOrder) {
    setSeed(
      order.items.map((i) => ({
        menuItemId: i.menuItemId,
        name: i.name,
        qty: i.qty,
        options: i.options,
      })),
    );
    setOrdering(true);
  }

  function orderCreated(order: Order) {
    setCreatedOrder(order);
    setOrdering(false);
    setOutcome("order");
    setLogKey((k) => k + 1);
    loadStats();
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-display text-2xl font-bold">{t.calls.title}</h1>
        <p className="text-sm text-ink-muted">{t.calls.subtitle}</p>
      </div>

      {/* ---- Today, in four numbers ---- */}
      {stats && (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCard label={t.calls.statCalls} value={String(stats.total)} hint={t.calls.statsToday} />
          <StatCard label={t.calls.statOrders} value={String(stats.orders)} hint={t.calls.statsToday} />
          <StatCard
            label={t.calls.statConversion}
            value={`${stats.conversion}%`}
            hint={t.calls.statsToday}
          />
          <StatCard
            label={t.calls.statCallbacks}
            value={String(stats.callbacksOpen)}
            hint={
              stats.callbacksOverdue > 0
                ? t.calls.statOverdue(stats.callbacksOverdue)
                : undefined
            }
            alert={stats.callbacksOverdue > 0}
          />
        </div>
      )}

      {/* The phone is ringing. Loud on purpose: this is the one thing on the
          screen that is time-critical. */}
      {live && (
        <div className="card animate-pulse border-brand bg-brand/5 p-4">
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-lg font-bold text-brand">
              ☎ {t.calls.incomingNow}
            </span>
            <span className="font-display text-xl font-bold">
              {formatUzPhone(live.phone)}
            </span>
            {live.customerName && (
              <span className="text-sm text-ink-muted">{live.customerName}</span>
            )}
            <button
              type="button"
              onClick={() => {
                setCallId(live.id);
                setDirection(live.direction);
                doLookup(live.phone);
              }}
              className="btn btn-primary ml-auto"
            >
              {t.calls.openCaller}
            </button>
          </div>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-[1fr_1fr]">
        {/* ---- Left: who is calling ---- */}
        <div className="space-y-4">
          <div className="card p-4">
            <h2 className="font-display text-lg font-bold">
              {t.calls.lookupTitle}
            </h2>
            <div className="mt-3 flex gap-2">
              {(["in", "out"] as const).map((d) => (
                <button
                  key={d}
                  type="button"
                  onClick={() => setDirection(d)}
                  className={`chip ${direction === d ? "bg-brand text-white" : ""}`}
                >
                  {d === "in" ? t.calls.incoming : t.calls.outgoing}
                </button>
              ))}
            </div>
            <form
              className="mt-3 flex gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                doLookup();
              }}
            >
              <input
                className="input flex-1"
                placeholder={t.calls.phonePlaceholder}
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                inputMode="tel"
                autoFocus
              />
              <button type="submit" disabled={looking} className="btn btn-primary">
                {looking ? t.calls.lookingUp : t.calls.lookup}
              </button>
              {/* Only offered when this operator has a handset: the exchange
                  rings them first, so without an extension there is nothing
                  to ring. */}
              {myExtension && (
                <button
                  type="button"
                  disabled={dialing || !phone.trim()}
                  onClick={() => dial(phone)}
                  className="btn btn-ghost disabled:opacity-40"
                  title={t.calls.dialHint(myExtension)}
                >
                  ☎
                </button>
              )}
            </form>
            {dialMsg && <p className="mt-2 text-sm text-ink-muted">{dialMsg}</p>}
            {error && <p className="mt-2 text-sm text-brand">{error}</p>}

            {lookup && (
              <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-ink-muted">
                <span>
                  {t.calls.onCall}: {Math.floor(elapsed / 60)}:
                  {String(elapsed % 60).padStart(2, "0")}
                </span>
                <button
                  type="button"
                  onClick={reset}
                  className="font-semibold text-brand hover:underline"
                >
                  {t.calls.newCall}
                </button>
              </div>
            )}
          </div>

          {lookup && (
            <CallerCard data={lookup} currency={currency} onRepeat={repeat} />
          )}
        </div>

        {/* ---- Right: what came of it ---- */}
        <div className="space-y-4">
          {lookup && (
            <div className="card p-4">
              <h2 className="font-display text-lg font-bold">
                {t.calls.resultTitle}
              </h2>

              <div className="mt-3 flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setSeed(undefined);
                    setOrdering(true);
                  }}
                  className="btn btn-primary"
                >
                  {t.calls.takeOrder}
                </button>
                <Link href="/admin/reservations" className="btn btn-ghost">
                  {t.calls.makeBooking}
                </Link>
              </div>

              {createdOrder && (
                <p className="mt-3 rounded-lg bg-emerald-500/10 p-2 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                  {t.calls.created(createdOrder.number)} ·{" "}
                  {formatPrice(createdOrder.total, currency, lang)}
                </p>
              )}

              <div className="mt-4">
                <label className="text-xs font-semibold text-ink-muted">
                  {t.calls.outcome}
                </label>
                <div className="mt-1 flex flex-wrap gap-1">
                  {OUTCOMES.map((o) => (
                    <button
                      key={o}
                      type="button"
                      onClick={() => setOutcome(o)}
                      className={`chip ${outcome === o ? "bg-brand text-white" : ""}`}
                    >
                      {t.calls.outcomes[o]}
                    </button>
                  ))}
                </div>
              </div>

              {/* A callback is the one outcome that leaves work behind, so it
                  is the one that has to name a time. */}
              {outcome === "callback" && (
                <div className="mt-3">
                  <label className="text-xs font-semibold text-ink-muted">
                    {t.calls.callbackAt}
                  </label>
                  <input
                    type="datetime-local"
                    className="input mt-1"
                    value={callbackAt}
                    onChange={(e) => setCallbackAt(e.target.value)}
                  />
                </div>
              )}

              <div className="mt-3">
                <label className="text-xs font-semibold text-ink-muted">
                  {t.calls.noteLabel}
                </label>
                <textarea
                  className="input mt-1"
                  rows={3}
                  placeholder={t.calls.notePlaceholder}
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
              </div>

              <div className="mt-3 flex items-center gap-3">
                <button
                  type="button"
                  disabled={!outcome || saving}
                  onClick={saveCall}
                  className="btn btn-primary disabled:opacity-50"
                >
                  {saving ? t.common.saving : t.calls.saveCall}
                </button>
                {savedMsg && (
                  <span className="text-sm font-semibold text-emerald-700 dark:text-emerald-300">
                    {savedMsg}
                  </span>
                )}
                {outcome && <OutcomeBadge outcome={outcome} />}
              </div>
            </div>
          )}

          {!lookup && (
            <div className="card p-6 text-sm text-ink-muted">
              {t.calls.subtitle}
            </div>
          )}
        </div>
      </div>

      <CallLog refreshKey={logKey} onPick={(p) => doLookup(p)} />

      {ordering && lookup && (
        <OperatorOrderModal
          phone={lookup.phone}
          customer={lookup.user}
          seed={seed}
          callId={callId || undefined}
          onClose={() => setOrdering(false)}
          onCreated={orderCreated}
        />
      )}
    </div>
  );
}

function StatCard({
  label,
  value,
  hint,
  alert,
}: {
  label: string;
  value: string;
  hint?: string;
  alert?: boolean;
}) {
  return (
    <div className={`card p-4 ${alert ? "border-brand" : ""}`}>
      <div className="text-xs text-ink-muted">{label}</div>
      <div className="font-display text-2xl font-bold">{value}</div>
      {hint && (
        <div className={`text-xs ${alert ? "text-brand" : "text-ink-muted"}`}>
          {hint}
        </div>
      )}
    </div>
  );
}
