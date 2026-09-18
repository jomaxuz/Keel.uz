"use client";

// The one VPS everything sits on.
//
// The control plane, the edge, Mongo and every customer container share one
// machine, so "how much room is left" is a single question — and it is the one
// that decides when the next customer stops being sellable.
//
// **Disk is the one that bites**, and quietly: uploads only ever grow, every
// deploy leaves another image behind, and a full disk stops Mongo writing
// before anybody notices a graph. So Docker's own usage is shown beside the
// filesystem's, split into what a prune would free and what it would not —
// images and stopped containers are reclaimable in one command; the volumes
// are customers' photographs and are not.
//
// Memory is read as **available**, never as "free": free memory on a healthy
// Linux box is near zero because the page cache holds the rest, and a gauge
// built on it shows every server as dying forever.

import { useCallback, useEffect, useState } from "react";
import { useT } from "@/lib/i18n/client";
import VersionPanel from "@/components/VersionPanel";
import {
  bytes,
  pruneDocker,
  systemStats,
  type BackupStatus,
  type DockerUsage,
  type HostStats,
} from "@/lib/api";

export default function ServerHealth() {
  const { t } = useT();
  const [host, setHost] = useState<HostStats | null>(null);
  const [docker, setDocker] = useState<DockerUsage | null>(null);
  const [backup, setBackup] = useState<BackupStatus | null>(null);
  const [error, setError] = useState("");
  const [pruning, setPruning] = useState(false);
  const [pruneNote, setPruneNote] = useState("");

  /** Frees the reclaimable space and says how much. ⚠️ The figures are replaced from
   *  the response rather than re-fetched: reading them again a moment later would
   *  race Docker's own accounting and could show the old number, which reads as the
   *  button having done nothing. */
  async function prune() {
    setPruning(true);
    setPruneNote("");
    try {
      const res = await pruneDocker();
      setPruneNote(t.dash.serverPruned(bytes(res.freed)));
      if (res.docker) setDocker(res.docker);
      if (res.warning) setPruneNote(t.dash.serverPruned(bytes(res.freed)) + " · " + res.warning);
    } catch (e) {
      setPruneNote(e instanceof Error ? e.message : "xato");
    } finally {
      setPruning(false);
    }
  }

  const load = useCallback(() => {
    systemStats()
      .then((s) => {
        setHost(s.host);
        setDocker(s.docker ?? null);
        setBackup(s.backup ?? null);
        setError("");
      })
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, []);

  useEffect(() => {
    load();
    // Slow on purpose: this is a number a person glances at, and the CPU
    // reading costs the server a 200 ms sample every time it is asked for.
    const id = setInterval(load, 30_000);
    return () => clearInterval(id);
  }, [load]);

  if (error) {
    return (
      <section className="card">
        <p className="text-sm font-semibold text-ink">{t.dash.serverTitle}</p>
        <p className="mt-2 text-sm text-rose-600 dark:text-rose-400">{error}</p>
      </section>
    );
  }
  if (!host) return null;

  return (
    <section className="card">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-sm font-semibold text-ink">{t.dash.serverTitle}</p>
        <span className="text-xs text-ink-muted">
          {t.dash.serverUptime(uptime(host.uptimeSeconds, t))}
        </span>
      </div>

      <div className="mt-4 grid gap-4 sm:grid-cols-3">
        <Gauge
          label={t.dash.serverCpu}
          percent={host.cpuPercent}
          detail={`${host.cores} ${t.dash.serverCores} · ${t.dash.serverLoad} ${host.load1.toFixed(2)}`}
        />
        <Gauge
          label={t.dash.serverMem}
          percent={host.memPercent}
          detail={`${bytes(host.memTotal - host.memAvailable)} / ${bytes(host.memTotal)}`}
        />
        <Gauge
          label={t.dash.serverDisk}
          percent={host.diskPercent}
          detail={`${bytes(host.diskFree)} ${t.dash.serverFree}`}
          note={host.diskPath}
        />
      </div>

      {docker && (
        <div className="mt-4 border-t border-line pt-3 text-xs text-ink-soft">
          <p className="font-semibold text-ink">{t.dash.serverDocker}</p>
          <div className="mt-1.5 flex flex-wrap gap-x-5 gap-y-1">
            <span>
              {t.dash.serverImages}: <b className="tabular-nums">{bytes(docker.images)}</b>
            </span>
            <span>
              {t.dash.serverVolumes}: <b className="tabular-nums">{bytes(docker.volumes)}</b>
            </span>
            {docker.reclaimable > 0 && (
              // The actionable half: what one command would give back.
              <span className="text-amber-700 dark:text-amber-300">
                {t.dash.serverReclaimable}:{" "}
                <b className="tabular-nums">{bytes(docker.reclaimable)}</b>
              </span>
            )}
          </div>
          {docker.reclaimable > 0 && (
            <div className="mt-2 flex flex-wrap items-center gap-2">
              {/* ⚠️ The button sits beside the number it acts on, and it names the
                  amount: "free 2.4 GB" is a decision, "clean up" is a guess. */}
              <button
                type="button"
                onClick={() => void prune()}
                disabled={pruning}
                className="rounded-xl border border-line px-2.5 py-1 text-[11px] font-semibold text-ink-soft hover:text-ink disabled:opacity-40"
              >
                {pruning
                  ? t.dash.serverPruning
                  : t.dash.serverPrune(bytes(docker.reclaimable))}
              </button>
              {pruneNote && <span className="text-[11px] text-ink-muted">{pruneNote}</span>}
            </div>
          )}
          {docker.reclaimable > 1_000_000_000 && (
            <p className="mt-1.5 text-ink-muted">{t.dash.serverPruneHint}</p>
          )}
        </div>
      )}

      {/* ---- last night's backup ----
           Shown even when everything is fine, and shown as an *age*: the only
           failure worth catching here is the silent one, where the copies
           stopped weeks ago and nothing anywhere says so. A row that only
           appears when broken is a row nobody learns to read. */}
      <div className="mt-4 border-t border-line pt-3 text-xs">
        <p className="font-semibold text-ink">{t.dash.backupTitle}</p>
        {!backup?.present ? (
          <p className="mt-1 text-rose-600 dark:text-rose-400">
            {t.dash.backupNone}
          </p>
        ) : (
          <>
            <p
              className={`mt-1 ${
                // Whether this counts as a missed night is the server's call
                // (sysstat.StaleAfter), not this component's: the overview
                // raises an alarm off the same flag, and a page that showed a
                // grey line under a red banner would be arguing with itself.
                backup.stale
                  ? "text-rose-600 dark:text-rose-400"
                  : "text-ink-soft"
              }`}
            >
              {backup.stale
                ? t.dash.backupStale(age(backup.ageHours, t))
                : t.dash.backupOk(
                    age(backup.ageHours, t),
                    backup.files,
                    bytes(backup.bytes),
                  )}
            </p>
            {backup.failures > 0 && (
              <p className="mt-1 text-amber-700 dark:text-amber-300">
                {t.dash.backupFailures(backup.failures)}
              </p>
            )}
          </>
        )}
      </div>

      {/* ---- what everything is running ----
           In this card rather than its own, because the two questions arrive
           together: somebody opens the server panel when something looks wrong,
           and "which version is this?" is the one asked right after "is it
           up?". Its own section would be a second place to look, which in
           practice is a place nobody looks. */}
      <VersionPanel />

      {/* A figure that could not be read is said, not defaulted to zero. */}
      {host.errors?.map((e) => (
        <p key={e} className="mt-2 text-xs text-ink-muted">
          {e}
        </p>
      ))}
    </section>
  );
}

/** The one thing on this page that can be unrecoverable, said where it will be
 *  read: at the top of the overview, filled, before any figure.
 *
 *  ⚠️ **The quiet row was not enough, and that is not a hypothesis.** The card
 *  below already showed this in red, and it showed it for three days in August
 *  2026 while cron silently refused to run the job — small grey-to-red text at
 *  the bottom of a card about disk usage, on a screen whose other rows change
 *  every day. Nobody read it. The signal was correct and useless, which is the
 *  worse of the two failures: it makes the dashboard look like it is watching.
 *
 *  So it lives here as well, and the two readings come from one field on one
 *  response — this is not a second opinion about the same manifest.
 *
 *  Rendered only when something is actually wrong, unlike the row below, which
 *  is always visible. The row's job is to be learnable ("copies are still being
 *  taken"); this one's job is to interrupt, and a banner that is always there
 *  interrupts nobody by the second week. */
export function BackupAlarm({ backup }: { backup?: BackupStatus }) {
  const { t } = useT();
  // Missing entirely means an older control-plane build that does not send the
  // field. Silence is right: inventing an alarm out of an absent field would
  // fire it on every machine the moment this shipped.
  if (!backup) return null;
  const bad = !backup.present || backup.stale || backup.failures > 0;
  if (!bad) return null;

  const line = !backup.present
    ? t.dash.backupNone
    : backup.stale
      ? t.dash.backupStale(age(backup.ageHours, t))
      : t.dash.backupFailures(backup.failures);

  return (
    <div className="rounded-2xl bg-rose-600 p-4 text-white dark:bg-rose-700">
      <p className="text-sm font-semibold">{t.dash.backupAlarm}</p>
      <p className="mt-1 text-sm text-white/90">{line}</p>
      {/* What to type next. An alarm that names the failure and not the check
          leaves the reader where they started — and the check is three
          commands nobody has memorised. */}
      <p className="mt-2 font-mono text-xs text-white/75">{t.dash.backupAlarmHint}</p>
    </div>
  );
}

/** A bar, its number, and the band it falls in.
 *
 *  Coloured only past the thresholds that mean something: green everywhere
 *  under 75% would be decoration, and a gauge that is always the same colour
 *  stops being read. */
function Gauge({
  label,
  percent,
  detail,
  note,
}: {
  label: string;
  percent: number;
  detail: string;
  note?: string;
}) {
  const p = Math.max(0, Math.min(100, percent));
  const tone =
    p >= 90
      ? "bg-rose-500"
      : p >= 75
        ? "bg-amber-500"
        : "bg-emerald-500";
  return (
    <div>
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs uppercase tracking-wider text-ink-muted">{label}</span>
        <span className="tabular-nums text-sm font-semibold text-ink">
          {p.toFixed(0)}%
        </span>
      </div>
      <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-line">
        <div className={`h-full rounded-full ${tone}`} style={{ width: `${p}%` }} />
      </div>
      <p className="mt-1 text-xs text-ink-muted">{detail}</p>
      {note && <p className="truncate text-[11px] text-ink-muted/70">{note}</p>}
    </div>
  );
}

/** Hours as "2 kun 3 soat", reusing the uptime wording rather than a second
 *  vocabulary for the same idea. */
function age(hours: number, t: ReturnType<typeof useT>["t"]): string {
  return uptime(Math.max(0, hours) * 3600, t);
}

function uptime(seconds: number, t: ReturnType<typeof useT>["t"]): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  return d > 0 ? `${d} ${t.dash.serverDays} ${h} ${t.dash.serverHours}` : `${h} ${t.dash.serverHours}`;
}
