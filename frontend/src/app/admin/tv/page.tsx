"use client";

// The televisions on the restaurant's walls.
//
// ⚠️ **A screen is added by typing the code it is showing.** Not by opening a
// link on it, not by signing in on it: a television is driven with a remote
// control, and typing a password one letter at a time across an on-screen
// keyboard, in front of a dining room, is a setup nobody finishes. Reading six
// characters aloud is — which is why the code is on the wall and the form is
// here.
//
// ⚠️ **The code rotates every few seconds**, so this form is filled in while
// somebody is looking at the screen. It says so, and it says what to do when
// the code has moved on — that refusal is the ordinary case, not an error.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminScope } from "@/lib/adminScope";
import { useAdminT } from "@/lib/i18n/admin";
import { timeAgo } from "@/lib/orderFlow";
import { useAsk } from "@/components/ui/Ask";
import type { TVScreen, TVScreenMode } from "@/lib/types";

/** How long a screen may be silent before the panel stops calling it alive.
 *
 *  ⚠️ **Generous on purpose.** The app calls home about once a minute; a
 *  restaurant's wifi drops a call regularly, and a screen marked "offline" the
 *  moment one poll is missed teaches a manager to ignore the marker entirely —
 *  after which it says nothing about the screen that has been dark since
 *  Tuesday. */
const ONLINE_WINDOW_MS = 5 * 60 * 1000;

function isOnline(screen: TVScreen): boolean {
  if (!screen.lastSeenAt) return false;
  return Date.now() - new Date(screen.lastSeenAt).getTime() < ONLINE_WINDOW_MS;
}

export default function AdminTVPage() {
  const t = useAdminT();
  const { ask } = useAsk();
  const scope = useAdminScope();
  const branchId = scope.branch?.id ?? "";

  const [screens, setScreens] = useState<TVScreen[]>([]);
  const [limit, setLimit] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);

  // The pairing form.
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [mode, setMode] = useState<TVScreenMode>("content");
  // Which row is being renamed, and what has been typed into it.
  const [editing, setEditing] = useState<string | null>(null);
  const [draftName, setDraftName] = useState("");

  const load = useCallback(() => {
    if (!branchId) {
      setScreens([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    api
      .tvScreens(branchId)
      .then((d) => {
        setScreens(d.screens);
        setLimit(d.limit);
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [branchId, t]);

  useEffect(load, [load]);

  // ⚠️ **Refreshed while the page is open**, because "is this screen alive" is
  // the question it exists to answer and the answer changes on its own. Slow
  // enough to be free: a screen that came back thirty seconds ago is news that
  // can wait thirty seconds.
  useEffect(() => {
    const id = setInterval(load, 30_000);
    return () => clearInterval(id);
  }, [load]);

  async function pair() {
    if (!branchId || !code.trim()) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await api.claimTVScreen({
        code: code.trim(),
        branchId,
        name: name.trim(),
        mode,
      });
      setCode("");
      setName("");
      setNotice(t.tv.paired);
      load();
    } catch (e) {
      // ⚠️ The server's own words. "That code is unknown or expired — type the
      // new one shown on the screen" names the next step; a generic failure
      // sends somebody to check their internet connection instead of the wall.
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  // ⚠️ Renamed in place rather than in a browser prompt. `window.prompt` is
  // the browser's own box — unstyled, untranslatable, and it carries the site's
  // domain over a panel a restaurant pays for. Same rule as components/ui/Ask.
  async function rename(screen: TVScreen) {
    const next = draftName.trim();
    setEditing(null);
    if (!next || next === screen.name) return;
    await save(screen, { name: next });
  }

  async function save(
    screen: TVScreen,
    body: { name?: string; mode?: TVScreenMode },
  ) {
    setBusy(true);
    setError("");
    try {
      await api.updateTVScreen(branchId, screen.id, body);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function remove(screen: TVScreen) {
    if (!(await ask({ title: t.tv.removeConfirm(screen.name), danger: true })))
      return;
    setBusy(true);
    setError("");
    try {
      await api.removeTVScreen(branchId, screen.id);
      setNotice(t.tv.removed);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.deleteFailed);
    } finally {
      setBusy(false);
    }
  }

  async function revokeAll() {
    if (
      !(await ask({
        title: t.tv.revokeConfirm,
        body: t.tv.revokeBody,
        danger: true,
      }))
    )
      return;
    setBusy(true);
    setError("");
    try {
      await api.revokeTVScreens(branchId);
      setNotice(t.tv.revoked);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

  return (
    <div className="space-y-5">
      <header>
        <h1 className="font-display text-2xl font-bold">{t.tv.title}</h1>
        <p className="mt-1 max-w-2xl text-sm text-ink-muted">{t.tv.intro}</p>
      </header>

      {/* ⚠️ **A branch, not the whole company.** A television hangs on one
          wall, and a list mixing three restaurants' screens is a list in which
          nobody can find the one they are standing under. */}
      {!branchId ? (
        <p className="rounded-2xl border border-line bg-surface p-6 text-sm text-ink-muted">
          {t.tv.pickBranch}
        </p>
      ) : (
        <>
          <section className="rounded-3xl border border-line bg-surface p-5 shadow-card">
            <h2 className="font-display text-lg font-bold">{t.tv.addTitle}</h2>
            {/* The three steps, in the order somebody standing in the room does
                them. Written out because this is the one screen in the panel
                whose other half is a television across the room. */}
            <ol className="mt-2 list-decimal space-y-1 pl-5 text-sm text-ink-muted">
              <li>{t.tv.step1}</li>
              <li>{t.tv.step2}</li>
              <li>{t.tv.step3}</li>
            </ol>

            <div className="mt-4 grid gap-3 sm:grid-cols-3">
              <label className="block text-sm">
                <span className="font-medium">{t.tv.code}</span>
                <input
                  className={`${inputCls} font-mono text-lg tracking-[0.3em] uppercase`}
                  value={code}
                  placeholder="K7P4RM"
                  maxLength={12}
                  onChange={(e) => setCode(e.target.value)}
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t.tv.name}</span>
                <input
                  className={inputCls}
                  value={name}
                  placeholder={t.tv.namePlaceholder}
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t.tv.mode}</span>
                <select
                  className={inputCls}
                  value={mode}
                  onChange={(e) => setMode(e.target.value as TVScreenMode)}
                >
                  <option value="content">{t.tv.modeContent}</option>
                  <option value="board">{t.tv.modeBoard}</option>
                  <option value="split">{t.tv.modeSplit}</option>
                </select>
              </label>
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-3">
              <button
                className="btn btn-primary"
                disabled={busy || !code.trim()}
                onClick={() => void pair()}
              >
                {busy ? t.common.saving : t.tv.pair}
              </button>
              <span className="text-xs text-ink-muted">{t.tv.codeHint}</span>
            </div>
          </section>

          {error && <p className="text-sm text-danger">{error}</p>}
          {notice && <p className="text-sm text-success">{notice}</p>}

          <section>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="font-display text-lg font-bold">
                {t.tv.listTitle}{" "}
                <span className="text-sm font-normal text-ink-muted">
                  {limit > 0
                    ? t.tv.countOf(screens.length, limit)
                    : t.tv.count(screens.length)}
                </span>
              </h2>
              {screens.length > 0 && (
                <button
                  className="btn-ghost px-3 py-1.5 text-sm text-danger"
                  disabled={busy}
                  onClick={() => void revokeAll()}
                >
                  {t.tv.revoke}
                </button>
              )}
            </div>

            {loading ? (
              <p className="mt-3 text-sm text-ink-muted">{t.common.loading}</p>
            ) : screens.length === 0 ? (
              // ⚠️ Said, not left blank: an empty list and a list that failed to
              // load look identical otherwise.
              <p className="mt-3 rounded-2xl border border-line bg-surface p-6 text-sm text-ink-muted">
                {t.tv.empty}
              </p>
            ) : (
              <ul className="mt-3 space-y-2">
                {screens.map((s) => (
                  <li
                    key={s.id}
                    className="flex flex-wrap items-center gap-3 rounded-2xl border border-line bg-surface px-4 py-3"
                  >
                    <span
                      className={`h-2.5 w-2.5 shrink-0 rounded-full ${
                        isOnline(s) ? "bg-success" : "bg-ink-muted/40"
                      }`}
                      aria-hidden
                    />
                    <div className="min-w-40 flex-1">
                      {editing === s.id ? (
                        <input
                          className="w-full rounded-xl border border-line-strong bg-surface px-2 py-1 text-sm outline-none focus:border-brand"
                          value={draftName}
                          autoFocus
                          onChange={(e) => setDraftName(e.target.value)}
                          onBlur={() => void rename(s)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") void rename(s);
                            if (e.key === "Escape") setEditing(null);
                          }}
                        />
                      ) : (
                        <p className="font-medium">{s.name}</p>
                      )}
                      <p className="text-xs text-ink-muted">
                        {/* ⚠️ A time, not a stored flag: "online" that was
                            written down once is a screen that has been dark
                            since Tuesday and still says it is fine. */}
                        {s.lastSeenAt
                          ? t.tv.lastSeen(
                              timeAgo(s.lastSeenAt, t.common.timeAgo),
                            )
                          : t.tv.neverSeen}
                        {s.appVersion ? ` · v${s.appVersion}` : ""}
                      </p>
                    </div>

                    <select
                      className="rounded-xl border border-line-strong bg-surface px-2 py-1.5 text-xs outline-none focus:border-brand"
                      value={s.mode}
                      disabled={busy}
                      onChange={(e) =>
                        void save(s, { mode: e.target.value as TVScreenMode })
                      }
                    >
                      <option value="content">{t.tv.modeContent}</option>
                      <option value="board">{t.tv.modeBoard}</option>
                      <option value="split">{t.tv.modeSplit}</option>
                    </select>

                    <button
                      className="btn-ghost px-2 text-sm"
                      disabled={busy}
                      onClick={() => {
                        setDraftName(s.name);
                        setEditing(s.id);
                      }}
                    >
                      {t.common.edit}
                    </button>
                    <button
                      className="btn-ghost px-2 text-sm text-danger"
                      disabled={busy}
                      onClick={() => void remove(s)}
                    >
                      {t.tv.unpair}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      )}
    </div>
  );
}
