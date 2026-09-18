"use client";

// What every part of Keel is running, and the one button that changes it.
//
// It sits inside the server card rather than in a settings page because that is
// where somebody already looks when something is wrong, and "which version is
// this?" is the question immediately after "is it up?".
//
// ⚠️ **The rows are what each part says about itself**, not what the repository
// says they should be. The console's number comes from the binary answering
// this request; the restaurants' servers are asked over their own /health; the
// till's is read from the manifest the monoblocks are actually offered. A panel
// that printed one number would have shown green through every deploy that
// built an image and left the containers running — which has happened here.
//
// ⚠️ **The site's bundle is compared in the browser**, against the constant
// built into this very file. It is the one part no server can report on: if the
// Next container was not replaced, the page doing the asking is the stale
// thing, and only it can say so.

import { useCallback, useEffect, useRef, useState } from "react";
import { useT } from "@/lib/i18n/client";
import { VERSION } from "@/lib/version";
import {
  bumpVersion,
  versionInfo,
  type VersionInfo,
  type VersionPart,
} from "@/lib/api";

/** How often the panel re-asks.
 *
 *  ⚠️ Two speeds, because the question changes. Idle, this is a number that
 *  moves about once a week and the probe costs a request per customer
 *  container. With a release in flight it is the screen somebody is watching,
 *  and five minutes of nothing reads as a release that failed. */
const IDLE_MS = 5 * 60_000;
const WATCHING_MS = 20_000;

const PARTS = ["patch", "minor", "major"] as const;

export default function VersionPanel() {
  const { t } = useT();
  const [info, setInfo] = useState<VersionInfo | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  // ⚠️ **Counted, not derived from `info`.** The next poll is scheduled off
  // this, so it is bumped whether the load succeeded or failed — keying the
  // schedule on the answer instead means one failed request at mount stops the
  // panel asking for the rest of the day, and it stops with an error message
  // that looks like a considered final state.
  const [tick, setTick] = useState(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const load = useCallback(() => {
    versionInfo()
      .then((v) => {
        setInfo(v);
        setError("");
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setTick((n) => n + 1));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // ⚠️ Rescheduled after each answer rather than run on a fixed interval: the
  // rate depends on what the last one said, and an interval set once would keep
  // the slow pace through the whole of the release it is meant to watch.
  useEffect(() => {
    if (tick === 0) return;
    const watching = info?.release?.status === "running";
    timer.current = setTimeout(load, watching ? WATCHING_MS : IDLE_MS);
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, [tick, info, load]);

  async function release(part: (typeof PARTS)[number]) {
    const next = info?.next?.[part];
    if (!next) return;
    // ⚠️ A confirmation, unlike the prune button beside it. That one frees
    // disk; this one replaces every container on the box, and the press is
    // one click away from a number nobody meant to move.
    if (!window.confirm(t.dash.verConfirm(next))) return;
    setBusy(part);
    try {
      const res = await bumpVersion(part);
      setInfo((cur) => (cur ? { ...cur, release: res.release } : cur));
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy("");
    }
  }

  if (!info) {
    return error ? (
      <div className="mt-4 border-t border-line pt-3 text-xs">
        <p className="font-semibold text-ink">{t.dash.verTitle}</p>
        <p className="mt-1 text-rose-600 dark:text-rose-400">{error}</p>
      </div>
    ) : null;
  }

  const rel = info.release;
  // The site's bundle, which only the browser can answer for.
  const siteRow: VersionPart = {
    id: "site",
    version: VERSION,
    match: VERSION === info.version,
  };

  return (
    <div className="mt-4 border-t border-line pt-3 text-xs">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="font-semibold text-ink">{t.dash.verTitle}</p>
        <span className="tabular-nums text-sm font-semibold text-ink">
          {info.version}
          {info.stage ? (
            <span className="ml-1.5 text-[11px] font-normal text-ink-muted">
              {info.stage}
            </span>
          ) : null}
        </span>
      </div>

      <div className="mt-2 space-y-1">
        <Row label={t.dash.verSite} part={siteRow} />
        {info.parts.map((p) => (
          <Row key={p.id} label={t.dash.verParts[p.id] ?? p.id} part={p} />
        ))}
      </div>

      {/* ---- raising it ----
           The buttons name the number they produce, not the word for the step:
           "to v0.2.1" is a decision, "patch" is a thing to look up. The word is
           kept beside it, small, because it is what makes the three tell
           apart at a glance. */}
      {info.mayRelease && info.releaseWired ? (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          {PARTS.map((part) => {
            const next = info.next?.[part];
            if (!next) return null;
            return (
              <button
                key={part}
                type="button"
                onClick={() => void release(part)}
                // ⚠️ Disabled while one is in flight, and by the same fact the
                // server refuses a second on: two releases racing would push
                // onto a tree the first has already moved.
                disabled={!!busy || rel?.status === "running"}
                className="rounded-xl border border-line px-2.5 py-1 text-[11px] font-semibold text-ink-soft hover:text-ink disabled:opacity-40"
              >
                {busy === part ? t.dash.verBumping : t.dash.verBump(next)}
                <span className="ml-1 font-normal text-ink-muted">
                  {t.dash.verPart[part]}
                </span>
              </button>
            );
          })}
        </div>
      ) : info.mayRelease ? (
        // Wired up nowhere on this deployment — worth saying, because the
        // person reading it is the one who would wire it.
        <p className="mt-3 text-ink-muted">{t.dash.verOff}</p>
      ) : null}

      {/* The release's own state. ⚠️ "running" is not a spinner: it says what
          was asked for and that the number here moves by itself, because the
          alternative is somebody pressing the button a second time. */}
      {rel && (
        <p
          className={`mt-2 ${
            rel.status === "stale"
              ? "text-amber-700 dark:text-amber-300"
              : rel.status === "done"
                ? "text-ink-muted"
                : "text-ink-soft"
          }`}
        >
          {rel.status === "running"
            ? t.dash.verRunning(rel.version)
            : rel.status === "done"
              ? t.dash.verDone(rel.version)
              : t.dash.verStale(rel.version)}
          {rel.runUrl && rel.status !== "done" && (
            <>
              {" · "}
              <a
                href={rel.runUrl}
                target="_blank"
                rel="noreferrer"
                className="underline decoration-dotted underline-offset-2"
              >
                {t.dash.verWatch}
              </a>
            </>
          )}
        </p>
      )}

      {error && <p className="mt-2 text-rose-600 dark:text-rose-400">{error}</p>}
    </div>
  );
}

/** One part, its number, and whether it agrees.
 *
 *  ⚠️ Three states, not two. "Behind" is a thing to fix; "unknown" is a thing
 *  that could not be asked — a laptop with no Docker socket, a till release
 *  directory nobody configured — and colouring the second like the first puts a
 *  permanent amber row on a healthy screen, which is how a panel stops being
 *  read. */
function Row({ label, part }: { label: string; part: VersionPart }) {
  const { t } = useT();
  const tone = part.unknown
    ? "text-ink-muted"
    : part.match
      ? "text-ink-soft"
      : "text-amber-700 dark:text-amber-300";
  return (
    <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5">
      <span className="text-ink-soft">{label}</span>
      <span className={`tabular-nums ${tone}`}>
        {part.version || t.dash.verUnknown}
        {!part.unknown && !part.match && (
          <span className="ml-1.5 font-normal">· {t.dash.verBehind}</span>
        )}
      </span>
      {part.note && (
        <span className="w-full text-[11px] text-ink-muted/80">{part.note}</span>
      )}
    </div>
  );
}
