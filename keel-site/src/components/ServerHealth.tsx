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
import { bytes, systemStats, type DockerUsage, type HostStats } from "@/lib/api";

export default function ServerHealth() {
  const { t } = useT();
  const [host, setHost] = useState<HostStats | null>(null);
  const [docker, setDocker] = useState<DockerUsage | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    systemStats()
      .then((s) => {
        setHost(s.host);
        setDocker(s.docker ?? null);
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
          {docker.reclaimable > 1_000_000_000 && (
            <p className="mt-1.5 text-ink-muted">{t.dash.serverPruneHint}</p>
          )}
        </div>
      )}

      {/* A figure that could not be read is said, not defaulted to zero. */}
      {host.errors?.map((e) => (
        <p key={e} className="mt-2 text-xs text-ink-muted">
          {e}
        </p>
      ))}
    </section>
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

function uptime(seconds: number, t: ReturnType<typeof useT>["t"]): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  return d > 0 ? `${d} ${t.dash.serverDays} ${h} ${t.dash.serverHours}` : `${h} ${t.dash.serverHours}`;
}
