"use client";

// The call log, and the filters that make it worth keeping.
//
// Two of these filters are the reason the log exists at all — "who still has to
// be rung back" and "which calls became orders" — so they are buttons rather
// than options buried in a dropdown. The rest (search, direction, operator,
// dates) are there for the question that comes after a complaint: "who took
// that call, and what did they say?"
//
// Callbacks sort soonest-first and overdue ones are marked, because a promise
// nobody can see the deadline of is not a promise.

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { formatDateTime, formatUzPhone } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { ListScroll, Pager, usePaged } from "@/components/admin/PagedList";
import type { Call, CallOutcome } from "@/lib/types";

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

const TONE: Record<CallOutcome, string> = {
  order: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  booking: "bg-sky-500/15 text-sky-700 dark:text-sky-300",
  info: "bg-ink/5 text-ink-muted",
  complaint: "bg-rose-500/15 text-rose-700 dark:text-rose-300",
  callback: "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  refused: "bg-ink/5 text-ink-muted",
  missed: "bg-ink/5 text-ink-muted",
  spam: "bg-ink/5 text-ink-muted",
};

// The tint of the whole row, by outcome. Same idea as the orders board and the
// same restraint: a wash behind the text, not a block of colour.
//
// The palette says what to do about it rather than what happened: green earned
// money, red needs answering, amber leaves work behind. Everything that needs
// nothing stays plain — most of a day's calls are questions, and colouring
// those in would drown the handful that matter.
const ROW_TONE: Record<CallOutcome, string> = {
  order: "bg-emerald-500/[0.12]",
  booking: "bg-sky-500/[0.10]",
  info: "",
  complaint: "bg-rose-500/[0.12]",
  callback: "bg-amber-500/[0.13]",
  refused: "",
  // A missed call is a customer nobody spoke to — red, like a complaint.
  missed: "bg-rose-500/[0.12]",
  spam: "",
};

// Nobody has written this call up yet: it is work outstanding, so it wears the
// same amber as a promised callback.
const PENDING_TONE = "bg-amber-500/[0.13]";

export function OutcomeBadge({ outcome }: { outcome: CallOutcome }) {
  const t = useAdminT();
  return (
    <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${TONE[outcome]}`}>
      {t.calls.outcomes[outcome]}
    </span>
  );
}

export default function CallLog({
  /** Bumped by the desk after a call is saved, so the log reloads itself. */
  refreshKey,
  onPick,
}: {
  refreshKey?: number;
  /** Clicking a row loads that number back into the desk above. */
  onPick?: (phone: string) => void;
}) {
  const t = useAdminT();
  const [calls, setCalls] = useState<Call[]>([]);
  const [loading, setLoading] = useState(true);

  const [q, setQ] = useState("");
  const [outcome, setOutcome] = useState("");
  const [direction, setDirection] = useState<"" | "in" | "out">("");
  const [callbacksOnly, setCallbacksOnly] = useState(false);
  // Calls the exchange recorded that nobody has written up. On a PBX install
  // this is the end-of-shift list: every one of them is a conversation whose
  // outcome nobody knows.
  const [pendingOnly, setPendingOnly] = useState(false);
  // Which row's recording is being fetched. The link is asked for on demand —
  // onlinePBX signs them, so a stored one quietly stops working.
  const [playing, setPlaying] = useState("");
  const [audio, setAudio] = useState<{ id: string; url: string } | null>(null);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const paged = usePaged(calls, 15);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const rows = await api.adminCalls({
          q: q.trim() || undefined,
          outcome: outcome || undefined,
          direction: direction || undefined,
          callback: callbacksOnly ? "open" : undefined,
          from: from || undefined,
          to: to || undefined,
        limit: 200,
      });
      setCalls(pendingOnly ? rows.filter((c) => !c.outcome) : rows);
    } catch {
      setCalls([]);
    } finally {
      setLoading(false);
    }
  }, [q, outcome, direction, callbacksOnly, pendingOnly, from, to]);

  useEffect(() => {
    // Typing in the search box must not fire a request per keystroke.
    const timer = setTimeout(load, 300);
    return () => clearTimeout(timer);
  }, [load, refreshKey]);

  async function markDone(call: Call) {
    await api.updateCall(call.id, { callbackDone: true });
    load();
  }

  const now = Date.now();

  return (
    <div className="card">
      <div className="border-b border-line p-4">
        <h2 className="font-display text-lg font-bold">{t.calls.logTitle}</h2>
        <div className="mt-3 flex flex-wrap gap-2">
          <input
            className="input w-full sm:w-72"
            placeholder={t.calls.searchPlaceholder}
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <select
            className="input w-auto"
            value={outcome}
            onChange={(e) => setOutcome(e.target.value)}
          >
            <option value="">{t.calls.allOutcomes}</option>
            {OUTCOMES.map((o) => (
              <option key={o} value={o}>
                {t.calls.outcomes[o]}
              </option>
            ))}
          </select>
          <select
            className="input w-auto"
            value={direction}
            onChange={(e) => setDirection(e.target.value as "" | "in" | "out")}
          >
            <option value="">{t.calls.allDirections}</option>
            <option value="in">{t.calls.incoming}</option>
            <option value="out">{t.calls.outgoing}</option>
          </select>
          <input
            type="date"
            className="input w-auto"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            aria-label={t.calls.from}
          />
          <input
            type="date"
            className="input w-auto"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            aria-label={t.calls.to}
          />
          <button
            type="button"
            onClick={() => setCallbacksOnly((v) => !v)}
            className={`chip ${callbacksOnly ? "bg-brand text-white" : ""}`}
          >
            {t.calls.onlyCallbacks}
          </button>
          <button
            type="button"
            onClick={() => setPendingOnly((v) => !v)}
            className={`chip ${pendingOnly ? "bg-brand text-white" : ""}`}
          >
            {t.calls.onlyPending}
          </button>
        </div>
      </div>

      {loading ? (
        <p className="p-4 text-sm text-ink-muted">{t.common.loading}</p>
      ) : calls.length === 0 ? (
        <p className="p-4 text-sm text-ink-muted">{t.calls.empty}</p>
      ) : (
        <>
          <ListScroll max="max-h-[60vh]">
            <ul className="divide-y divide-line">
              {paged.pageItems.map((c) => {
                const overdue =
                  !!c.callbackAt &&
                  !c.callbackDone &&
                  new Date(c.callbackAt).getTime() <= now;
                return (
                  <li
                    key={c.id}
                    className={`p-4 ${
                      c.outcome ? ROW_TONE[c.outcome] : PENDING_TONE
                    }`}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <button
                        type="button"
                        onClick={() => onPick?.(c.phone)}
                        className="font-semibold hover:text-brand"
                      >
                        {formatUzPhone(c.phone)}
                      </button>
                      {c.customerName && (
                        <span className="text-sm text-ink-muted">
                          {c.customerName}
                        </span>
                      )}
                      {c.outcome ? (
                        <OutcomeBadge outcome={c.outcome} />
                      ) : (
                        <span className="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs font-semibold text-amber-700 dark:text-amber-300">
                          {t.calls.notWrittenUp}
                        </span>
                      )}
                      {c.source === "pbx" && (
                        <span className="rounded-full bg-ink/5 px-2 py-0.5 text-xs text-ink-muted">
                          {t.calls.fromPbx}
                        </span>
                      )}
                      <span className="text-xs text-ink-muted">
                        {c.direction === "out"
                          ? t.calls.outgoing
                          : t.calls.incoming}
                      </span>
                      <span className="ml-auto text-xs text-ink-muted">
                        {formatDateTime(c.createdAt)}
                      </span>
                    </div>

                    {c.note && <p className="mt-1 text-sm">{c.note}</p>}

                    <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-ink-muted">
                      {c.operatorName && (
                        <span>
                          {t.calls.operator}: {c.operatorName}
                        </span>
                      )}
                      {c.seconds > 0 && (
                        <span>
                          {t.calls.duration}: {Math.floor(c.seconds / 60)}:
                          {String(c.seconds % 60).padStart(2, "0")}
                        </span>
                      )}
                      {c.hasRecording && (
                        <button
                          type="button"
                          disabled={playing === c.id}
                          onClick={async () => {
                            setPlaying(c.id);
                            try {
                              const res = await api.callRecording(c.id);
                              setAudio(
                                res.url ? { id: c.id, url: res.url } : null,
                              );
                            } finally {
                              setPlaying("");
                            }
                          }}
                          className="font-semibold text-brand hover:underline"
                        >
                          ▶ {t.calls.recording}
                        </button>
                      )}
                      {c.orderNumber && (
                        <Link
                          href={`/admin/orders?q=${c.orderNumber}`}
                          className="font-semibold text-brand hover:underline"
                        >
                          #{c.orderNumber}
                        </Link>
                      )}
                      {c.reservationNumber && (
                        <Link
                          href="/admin/reservations"
                          className="font-semibold text-brand hover:underline"
                        >
                          #{c.reservationNumber}
                        </Link>
                      )}
                    </div>

                    {audio?.id === c.id && (
                      <audio controls src={audio.url} className="mt-2 w-full" />
                    )}

                    {c.callbackAt && !c.callbackDone && (
                      <div className="mt-2 flex flex-wrap items-center gap-2">
                        <span
                          className={`text-xs font-semibold ${
                            overdue ? "text-brand" : "text-amber-700 dark:text-amber-300"
                          }`}
                        >
                          {t.calls.callbackDue(formatDateTime(c.callbackAt))}
                          {overdue && ` · ${t.calls.overdue}`}
                        </span>
                        <button
                          type="button"
                          onClick={() => markDone(c)}
                          className="chip"
                        >
                          {t.calls.markDone}
                        </button>
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          </ListScroll>
          <Pager {...paged} onPage={paged.setPage} />
        </>
      )}
    </div>
  );
}
