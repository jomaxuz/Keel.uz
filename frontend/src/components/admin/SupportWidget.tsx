"use client";

/**
 * The help button, and the conversation behind it.
 *
 * ⚠️ **It floats over the panel and never pushes it.** This is the same lesson
 * the till's keyboard cost: a panel that reflows when a helper opens is a panel
 * that moves the row somebody was reading, and what they report is "the screen
 * jumps". The card is fixed, z-indexed above everything, and the page under it
 * does not know it exists.
 *
 * ⚠️ **The live channel is a WebSocket to this restaurant's own server**, not
 * to the platform. Same origin — no CORS, no preflight, and no platform
 * credential in a browser on a customer's domain. That server holds one request
 * open against the control plane and pushes what comes back; see the Go side
 * for why a ticket rather than the session token opens the socket.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { LuHeadset, LuSend, LuX, LuChevronLeft } from "react-icons/lu";

import { api, API_URL } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { SupportMessage, SupportThread } from "@/lib/types";

export default function SupportWidget() {
  const t = useAdminT();
  const [open, setOpen] = useState(false);
  const [threads, setThreads] = useState<SupportThread[] | null>(null);
  const [active, setActive] = useState<string | null>(null);
  const [messages, setMessages] = useState<SupportMessage[]>([]);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [failed, setFailed] = useState(false);
  const [live, setLive] = useState(false);
  // ⚠️ Unread is held here rather than derived from `threads`, because the
  // badge has to survive the list being reloaded — and it is the only thing on
  // screen when the widget is shut.
  const [unread, setUnread] = useState(0);

  const bottom = useRef<HTMLDivElement>(null);
  const socket = useRef<WebSocket | null>(null);
  // ⚠️ Kept in a ref as well as in state: the socket handler is created once
  // and would otherwise close over the thread that was open when it connected.
  const activeRef = useRef<string | null>(null);
  activeRef.current = active;

  const loadThreads = useCallback(async () => {
    try {
      const res = await api.supportThreads();
      setThreads(res.threads);
      setUnread(res.threads.reduce((n, x) => n + (x.unreadForOwner || 0), 0));
    } catch {
      setThreads([]);
    }
  }, []);

  const openThread = useCallback(async (id: string) => {
    setActive(id);
    try {
      const res = await api.supportThread(id);
      setMessages(res.messages);
    } catch {
      setMessages([]);
    }
  }, []);

  // ---- The socket ----
  //
  // ⚠️ **Opened once for the session, not per conversation.** A reply may land
  // on a thread the owner is not looking at — that is exactly when the badge
  // matters — so the channel is per restaurant and the screen decides what to
  // do with each message.
  useEffect(() => {
    let stopped = false;
    let retry: ReturnType<typeof setTimeout> | undefined;
    let wait = 1000;

    async function connect() {
      if (stopped) return;
      try {
        const { ticket, url: given } = await api.supportTicket();
        // ⚠️ **The server says where its socket is, and the fallback is only
        // for an install that has not set `PUBLIC_BASE_URL`.** In production
        // the panel and the API share an origin and either would work; in
        // development `/api/*` reaches the backend through a Next rewrite,
        // which proxies HTTP and does not upgrade a WebSocket. Building the
        // address here from `location` fails in development only — with a
        // handshake error that looks like a bug in this file.
        const base = API_URL.startsWith("http")
          ? API_URL
          : `${location.origin}${API_URL}`;
        const socketBase =
          given || base.replace(/^http/, "ws") + "/admin/support/ws";
        const url = `${socketBase}?ticket=${encodeURIComponent(ticket)}`;
        const ws = new WebSocket(url);
        socket.current = ws;
        ws.onopen = () => {
          setLive(true);
          // A dropped connection that reconnects immediately is the common
          // case; the backoff only matters when the server is actually down.
          wait = 1000;
        };
        ws.onmessage = (e) => {
          let payload: { messages?: SupportMessage[] };
          try {
            payload = JSON.parse(e.data as string);
          } catch {
            return;
          }
          const fresh = (payload.messages ?? []).filter(
            (m) => m.from !== "owner",
          );
          if (fresh.length === 0) return;
          // ⚠️ The owner's own lines are filtered out above: they were already
          // put on screen when they were sent, and echoing them back is a
          // message that appears twice.
          const here = fresh.filter((m) => m.threadId === activeRef.current);
          if (here.length > 0) {
            setMessages((prev) => [
              ...prev,
              ...here.filter((m) => !prev.some((p) => p.id === m.id)),
            ]);
          }
          setUnread((n) => n + fresh.length - here.length);
          void loadThreads();
        };
        ws.onclose = () => {
          setLive(false);
          socket.current = null;
          if (stopped) return;
          retry = setTimeout(connect, wait);
          // ⚠️ Capped. A tab left open overnight against a server that is down
          // must not turn into a request every second until morning.
          wait = Math.min(wait * 2, 30_000);
        };
        ws.onerror = () => ws.close();
      } catch {
        if (stopped) return;
        retry = setTimeout(connect, wait);
        wait = Math.min(wait * 2, 30_000);
      }
    }
    void connect();
    return () => {
      stopped = true;
      if (retry) clearTimeout(retry);
      socket.current?.close();
    };
  }, [loadThreads]);

  useEffect(() => {
    if (open) void loadThreads();
  }, [open, loadThreads]);

  useEffect(() => {
    bottom.current?.scrollIntoView({ block: "end" });
  }, [messages]);

  async function send() {
    const text = draft.trim();
    if (!text || sending) return;
    setSending(true);
    setFailed(false);
    try {
      const res = await api.supportAsk({ threadId: active ?? "", text });
      setDraft("");
      setActive(res.threadId);
      // Shown immediately rather than waiting for the socket to echo it: the
      // person pressed send and the line has to appear.
      setMessages((prev) => [...prev, res.message]);
      void loadThreads();
    } catch {
      setFailed(true);
    } finally {
      setSending(false);
    }
  }

  return (
    <>
      {/* ⚠️ Above the panel and below a modal. The till learned this the hard
          way: a helper that shares a stacking level with dialogs is a helper
          that covers the button it was opened to explain. */}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label={t.support.open}
        className="fixed bottom-5 right-5 z-40 grid h-14 w-14 place-items-center rounded-full bg-brand text-white shadow-xl shadow-ink/20 transition hover:brightness-110"
      >
        {open ? <LuX className="h-6 w-6" /> : <LuHeadset className="h-6 w-6" />}
        {!open && unread > 0 && (
          <span className="absolute -right-0.5 -top-0.5 grid h-6 min-w-6 place-items-center rounded-full bg-rose-500 px-1.5 text-xs font-bold text-white">
            {unread}
          </span>
        )}
      </button>

      {open && (
        <div className="fixed bottom-24 right-5 z-40 flex h-[32rem] max-h-[calc(100dvh-8rem)] w-[min(24rem,calc(100vw-2.5rem))] flex-col overflow-hidden rounded-2xl border border-line bg-surface shadow-2xl shadow-ink/25">
          <header className="flex items-center gap-2 border-b border-line px-4 py-3">
            {active && (
              <button
                type="button"
                onClick={() => {
                  setActive(null);
                  setMessages([]);
                }}
                aria-label={t.support.back}
                className="grid h-8 w-8 place-items-center rounded-lg text-ink-muted hover:bg-raised"
              >
                <LuChevronLeft className="h-4 w-4" />
              </button>
            )}
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-semibold text-ink">
                {t.support.title}
              </p>
              <p className="truncate text-xs text-ink-muted">
                {/* ⚠️ The connection state is shown, quietly. A chat that has
                    silently lost its socket looks identical to a chat nobody
                    has answered yet, and the two need different patience. */}
                {live ? t.support.live : t.support.offline}
              </p>
            </div>
          </header>

          {!active ? (
            <div className="flex-1 overflow-y-auto p-3">
              <p className="px-1 pb-3 text-sm leading-relaxed text-ink-muted">
                {t.support.lead}
              </p>
              {threads?.length === 0 && (
                <p className="px-1 py-4 text-sm text-ink-muted">{t.support.empty}</p>
              )}
              <ul className="space-y-2">
                {(threads ?? []).map((th) => (
                  <li key={th.id}>
                    <button
                      type="button"
                      onClick={() => void openThread(th.id)}
                      className="w-full rounded-xl border border-line px-3 py-2.5 text-left hover:bg-raised"
                    >
                      <span className="flex items-center gap-2">
                        <span className="min-w-0 flex-1 truncate text-sm font-medium text-ink">
                          {th.subject}
                        </span>
                        {th.unreadForOwner > 0 && (
                          <span className="grid h-5 min-w-5 place-items-center rounded-full bg-rose-500 px-1 text-[11px] font-bold text-white">
                            {th.unreadForOwner}
                          </span>
                        )}
                      </span>
                      <span className="mt-1 block truncate text-xs text-ink-muted">
                        {th.lastText}
                      </span>
                      <span className="mt-1 block text-[11px] text-ink-muted">
                        {th.status === "closed"
                          ? t.support.closed
                          : th.status === "open"
                            ? t.support.answered
                            : t.support.waiting}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <div className="flex-1 space-y-2 overflow-y-auto p-3">
              {messages.map((m) => (
                <div
                  key={m.id}
                  className={m.from === "owner" ? "flex justify-end" : "flex"}
                >
                  <div
                    className={`max-w-[85%] rounded-2xl px-3 py-2 text-sm leading-relaxed ${
                      m.from === "owner"
                        ? "bg-brand text-white"
                        : "border border-line bg-raised text-ink"
                    }`}
                  >
                    {m.from !== "owner" && (
                      <p className="mb-0.5 text-[11px] font-semibold text-ink-muted">
                        {m.from === "assistant"
                          ? t.support.assistant
                          : m.author || t.support.operator}
                      </p>
                    )}
                    <p className="whitespace-pre-wrap break-words">{m.text}</p>
                  </div>
                </div>
              ))}
              <div ref={bottom} />
            </div>
          )}

          <div className="border-t border-line p-3">
            {failed && (
              <p className="pb-2 text-xs text-rose-600">{t.support.failed}</p>
            )}
            <div className="flex items-end gap-2">
              <textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                  // Enter sends, shift+enter breaks the line — what everybody
                  // already expects from a chat box.
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    void send();
                  }
                }}
                rows={2}
                placeholder={t.support.placeholder}
                className="min-h-[2.75rem] flex-1 resize-none rounded-xl border border-line bg-page px-3 py-2 text-sm text-ink outline-none placeholder:text-ink-muted focus:border-brand"
              />
              <button
                type="button"
                onClick={() => void send()}
                disabled={sending || !draft.trim()}
                aria-label={t.support.send}
                className="grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-brand text-white disabled:opacity-40"
              >
                <LuSend className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
