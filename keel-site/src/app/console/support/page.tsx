"use client";

// The support queue, and the conversation an operator is in.
//
// ⚠️ **Two panes, not two pages.** An operator answers a question, glances at
// the next one, and comes back — a queue that navigates away every time a
// thread is opened is a queue somebody loses their place in twenty times a
// morning. The list stays; the right pane changes.
//
// ⚠️ **The customer record is on the screen while the answer is being typed.**
// Half of support is "is their container up", "what plan are they on", "who is
// this person" — and an operator who has to open a second tab to find out will
// answer without looking.

import { useCallback, useEffect, useRef, useState } from "react";

import { useT } from "@/lib/i18n/client";
import {
  supportList,
  supportReply,
  supportThread,
  type SupportMessageRow,
  type SupportTenantCard,
  type SupportThreadRow,
} from "@/lib/api";

// ⚠️ **Read from the dictionary at render, not built once at module load.**
// A `const` map here is evaluated when the module is imported, which is before
// the language is known and never again after it changes — so the queue would
// keep its first language for the life of the tab.
const FILTER_IDS = ["waiting", "open", "closed", "all"] as const;

// ⚠️ **The queue refreshes itself, because nobody presses reload on a queue.**
// Ten seconds is short enough that a waiting restaurant is noticed within one
// and long enough that thirty operators do not become three requests a second.
const REFRESH_MS = 10_000;

export default function SupportPage() {
  const { t: d } = useT();
  const [status, setStatus] = useState("waiting");
  const [q, setQ] = useState("");
  const [rows, setRows] = useState<SupportThreadRow[]>([]);
  const [waiting, setWaiting] = useState(0);
  const [error, setError] = useState("");

  const [openID, setOpenID] = useState<string | null>(null);
  const [thread, setThread] = useState<SupportThreadRow | null>(null);
  const [messages, setMessages] = useState<SupportMessageRow[]>([]);
  const [tenant, setTenant] = useState<SupportTenantCard | null>(null);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const bottom = useRef<HTMLDivElement>(null);

  const load = useCallback(async () => {
    try {
      const r = await supportList({
        status,
        q: q.trim() || undefined,
      });
      setRows(r.threads);
      setWaiting(r.waiting);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "yuklanmadi");
    }
  }, [status, q]);

  // ⚠️ Debounced, because this runs on every keystroke in the search box and
  // the search is a text index over every thread the platform has received.
  useEffect(() => {
    const timer = window.setTimeout(load, 250);
    return () => window.clearTimeout(timer);
  }, [load]);

  useEffect(() => {
    const timer = window.setInterval(load, REFRESH_MS);
    return () => window.clearInterval(timer);
  }, [load]);

  const open = useCallback(async (id: string) => {
    setOpenID(id);
    try {
      const r = await supportThread(id);
      setThread(r.thread);
      setMessages(r.messages);
      setTenant(r.tenant);
    } catch (e) {
      setError(e instanceof Error ? e.message : "ochilmadi");
    }
  }, []);

  // The open conversation follows the queue's refresh, so an operator sitting
  // on a thread sees the customer's follow-up without pressing anything.
  useEffect(() => {
    if (!openID) return;
    const timer = window.setInterval(() => void open(openID), REFRESH_MS);
    return () => window.clearInterval(timer);
  }, [openID, open]);

  useEffect(() => {
    bottom.current?.scrollIntoView({ block: "end" });
  }, [messages]);

  async function send(close: boolean) {
    if (!openID || sending) return;
    const text = draft.trim();
    if (!text && !close) return;
    setSending(true);
    try {
      await supportReply(openID, { text, close });
      setDraft("");
      await open(openID);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "yuborilmadi");
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="grid gap-5 lg:grid-cols-[22rem_1fr]">
      {/* ---- The queue ---- */}
      <section className="flex flex-col gap-3">
        <div className="flex items-center justify-between gap-3">
          <h1 className="h-display text-xl">{d.console.support.title}</h1>
          {waiting > 0 && (
            <span className="rounded-lg bg-rose-500 px-2 py-0.5 text-xs font-bold text-white">
              {waiting}
            </span>
          )}
        </div>
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={d.console.support.search}
          className="input"
        />
        <div className="flex flex-wrap gap-1.5">
          {FILTER_IDS.map((f) => (
            <button
              key={f}
              type="button"
              onClick={() => setStatus(f)}
              className={`rounded-lg px-2.5 py-1.5 text-xs font-semibold transition ${
                status === f
                  ? "bg-ink text-page"
                  : "border border-line text-ink-muted hover:text-ink"
              }`}
            >
              {d.console.support.filters[f]}
            </button>
          ))}
        </div>
        {error && <p className="text-sm text-rose-600">{error}</p>}
        <ul className="space-y-2 lg:max-h-[calc(100vh-19rem)] lg:overflow-y-auto lg:pr-1">
          {rows.length === 0 && (
            <li className="rounded-xl border border-dashed border-line px-3 py-6 text-center text-sm text-ink-muted">
              {status === "waiting"
                ? d.console.support.emptyQueue
                : d.console.support.emptyOther}
            </li>
          )}
          {rows.map((th) => (
            <li key={th.id}>
              <button
                type="button"
                onClick={() => void open(th.id)}
                className={`w-full rounded-xl border px-3 py-2.5 text-left transition ${
                  openID === th.id
                    ? "border-signal-500 bg-signal-500/10"
                    : "border-line hover:bg-raised"
                }`}
              >
                <span className="flex items-center gap-2">
                  <span className="min-w-0 flex-1 truncate text-sm font-semibold text-ink">
                    {th.restaurant || th.slug}
                  </span>
                  {th.unreadForUs > 0 && (
                    <span className="grid h-5 min-w-5 place-items-center rounded-full bg-rose-500 px-1 text-[11px] font-bold text-white">
                      {th.unreadForUs}
                    </span>
                  )}
                </span>
                <span className="mt-0.5 block truncate text-sm text-ink-soft">
                  {th.subject}
                </span>
                <span className="mt-1 flex items-center gap-2 text-[11px] text-ink-muted">
                  <span>
                    {d.console.support.status[
                      th.status as keyof typeof d.console.support.status
                    ] ?? th.status}
                  </span>
                  <span>·</span>
                  <span>{ago(th.lastAt, d.console.support)}</span>
                  {th.operatorName && (
                    <>
                      <span>·</span>
                      <span className="truncate">{th.operatorName}</span>
                    </>
                  )}
                </span>
              </button>
            </li>
          ))}
        </ul>
      </section>

      {/* ---- The conversation ---- */}
      {/* ⚠️ **A fixed height, not a minimum.** With `min-h` the card grew with
          the conversation: `overflow-y-auto` inside it never had a boundary to
          scroll against, so a long answer stretched the whole page and the
          operator scrolled the browser to reach the reply box — past the
          queue, past the header, further with every message. A chat pane has
          to be the thing that scrolls, which means it has to be the thing with
          a height. `min-h-0` on the list below is the other half: a flex child
          refuses to shrink past its content without it, which quietly restores
          the same bug. */}
      <section className="flex h-[calc(100vh-9rem)] min-h-[28rem] flex-col overflow-hidden rounded-2xl border border-line bg-surface">
        {!thread ? (
          <p className="grid h-full place-items-center p-8 text-sm text-ink-muted">
            Chapdan suhbatni tanlang.
          </p>
        ) : (
          <div className="flex h-full flex-col">
            <header className="shrink-0 border-b border-line p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="font-display text-lg font-semibold text-ink">
                    {thread.restaurant || thread.slug}
                  </p>
                  <p className="mt-0.5 text-sm text-ink-muted">
                    {thread.subject}
                  </p>
                  <p className="mt-1 text-xs text-ink-muted">
                    {thread.askedBy}
                    {thread.askedRole ? ` · ${thread.askedRole}` : ""} ·{" "}
                    {d.console.support.status[
                      thread.status as keyof typeof d.console.support.status
                    ] ?? thread.status}
                  </p>
                </div>
                {tenant && <TenantCard tenant={tenant} />}
              </div>
            </header>

            <div className="min-h-0 flex-1 space-y-2.5 overflow-y-auto p-4">
              {messages.map((m) => (
                <div
                  key={m.id}
                  className={m.from === "owner" ? "flex" : "flex justify-end"}
                >
                  <div
                    className={`max-w-[80%] rounded-2xl px-3.5 py-2.5 text-sm leading-relaxed ${
                      m.from === "owner"
                        ? "border border-line bg-raised text-ink"
                        : m.from === "assistant"
                          ? "border border-dashed border-line-strong bg-page text-ink"
                          : "bg-ink text-page"
                    }`}
                  >
                    <p className="mb-0.5 text-[11px] font-semibold opacity-70">
                      {m.from === "assistant"
                        ? d.console.support.assistant
                        : m.author}
                      {" · "}
                      {when(m.at)}
                    </p>
                    <p className="whitespace-pre-wrap break-words">{m.text}</p>
                  </div>
                </div>
              ))}
              <div ref={bottom} />
            </div>

            <div className="shrink-0 border-t border-line p-4">
              <textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                    e.preventDefault();
                    void send(false);
                  }
                }}
                rows={3}
                placeholder={d.console.support.reply}
                className="input min-h-[5rem] resize-none"
              />
              <div className="mt-2 flex items-center justify-end gap-2">
                {/* ⚠️ **Answering and closing is one press.** A separate close
                    button means half the queue stays open for ever: the
                    operator sends the answer, the thread scrolls away, and
                    nobody comes back to tidy it. */}
                <button
                  type="button"
                  onClick={() => void send(true)}
                  disabled={sending}
                  className="btn-ghost text-sm"
                >
                  Javob berib yopish
                </button>
                <button
                  type="button"
                  onClick={() => void send(false)}
                  disabled={sending || !draft.trim()}
                  className="btn-primary text-sm"
                >
                  Yuborish
                </button>
              </div>
            </div>
          </div>
        )}
      </section>
    </div>
  );
}

/** Who is asking, in one line an operator reads without leaving the answer.
 *
 * ⚠️ The prop is `tenant`, not `t`. It was `t`, which shadowed the dictionary
 * the moment this file learned to speak three languages — the kind of collision
 * that compiles happily anywhere the two happen to share a field name. */
function TenantCard({ tenant }: { tenant: SupportTenantCard }) {
  const { t } = useT();
  // ⚠️ **The container's state is the loudest thing here.** "The panel is
  // blank" and "the panel is down" are the same sentence from a customer and
  // completely different answers from us.
  const down = tenant.container && tenant.container !== "running";
  return (
    <div className="rounded-xl border border-line bg-page px-3 py-2 text-xs">
      <p className="font-semibold text-ink">{tenant.slug}</p>
      {tenant.owner && (
        <p className="mt-0.5 text-ink-muted">
          {tenant.owner}
          {tenant.phone ? ` · ${tenant.phone}` : ""}
        </p>
      )}
      <p className="mt-0.5 text-ink-muted">
        {tenant.till
          ? `${t.console.support.tillLabel}: ${tenant.till}`
          : t.console.support.noTill}
        {tenant.free ? ` · ${t.console.support.free}` : ""}
      </p>
      {down && (
        <p className="mt-1 font-semibold text-rose-600">
          {t.console.support.container}: {tenant.container}
        </p>
      )}
    </div>
  );
}

/** "12 daq" — how long a restaurant has been waiting, which is the number an
 *  operator sorts by in their head.
 *
 *  ⚠️ Takes the unit words rather than reading them itself: it is a plain
 *  function, not a component, and calling a hook here would be a hook called
 *  outside a render. */
function ago(iso: string, w: SupportWords): string {
  const then = new Date(iso).getTime();
  if (!then) return "";
  const mins = Math.max(0, Math.round((Date.now() - then) / 60000));
  if (mins < 60) return w.mins(mins);
  const hours = Math.round(mins / 60);
  if (hours < 24) return w.hours(hours);
  return w.days(Math.round(hours / 24));
}

type SupportWords = {
  mins: (n: number) => string;
  hours: (n: number) => string;
  days: (n: number) => string;
};

function when(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString("uz-UZ", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}
