// keel.uz/status — is the platform working, and was it yesterday.
//
// The trap a status page falls into is being green by default: nothing has
// told it otherwise, so it says all is well. That page is worse than none — it
// gets read as evidence during the one hour it is wrong, and it spends exactly
// the trust it was built to earn.
//
// So everything here is measured. The control plane samples itself once a
// minute — its own database, and whether the customer containers it believes
// should be running actually are — and those samples are what is drawn.
// Three states, not two:
//
//   • **passed** — every sample in that hour was fine
//   • **failed** — at least one was not, and the hour carries the reason
//   • **no data** — nothing was measured; drawn as a gap, never as green and
//     never as red. A page that paints ignorance either colour is one nobody
//     believes the second time.
//
// The control plane being unreachable *is* the status, and the page says so
// rather than failing to render.

import type { Metadata } from "next";
import Header from "@/components/Header";
import { getT } from "@/lib/i18n/server";
import { getStatus, type StatusDay, type StatusHour } from "@/lib/partners";

// Always measured at request time. A cached status page is a page that says
// everything is fine for as long as the cache lives.
export const dynamic = "force-dynamic";

export async function generateMetadata(): Promise<Metadata> {
  const t = await getT();
  return { title: `${t.status.title} · Keel`, robots: { index: false } };
}

export default async function StatusPage() {
  const t = await getT();
  const s = await getStatus();

  return (
    <>
      <Header />
      <main className="container-page py-16 sm:py-20">
        <p className="eyebrow">{t.status.eyebrow}</p>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <h1 className="h-display text-3xl sm:text-4xl">{t.status.title}</h1>
          {/* ⚠️ The version, and what the version *means*.
              "v0.1" alone invites a guess, and the guess somebody makes about a
              platform holding their restaurant's orders is the generous one. The word
              beside it is printed while it is still true, and it comes from the binary
              answering this request rather than from a file that survives a rollback. */}
          {s?.version && (
            <span className="rounded-full border border-line px-3 py-1 text-xs font-semibold text-ink-soft">
              {s.version}
              {s.stage === "test" && ` · ${t.status.stageTest}`}
            </span>
          )}
        </div>

        {/* The headline, in words before colours: somebody reading this is
            often doing so because something looked broken. */}
        <div className="mt-8 rounded-3xl border border-line bg-surface p-6">
          {!s ? (
            <Headline tone="bad" title={t.status.unreachable} note={t.status.unreachableNote} />
          ) : s.lastCheck === null ? (
            <Headline tone="unknown" title={t.status.noData} note={t.status.noDataNote} />
          ) : s.up ? (
            <Headline
              tone="good"
              title={t.status.allGood}
              note={t.status.checkedAt(hourLabel(s.lastCheck))}
            />
          ) : (
            <Headline
              tone="bad"
              title={t.status.degraded}
              note={t.status.checkedAt(hourLabel(s.lastCheck))}
            />
          )}

          {s && s.lastCheck !== null && (
            <p className="mt-4 text-sm text-ink-soft">
              {t.status.uptime90d}{" "}
              <span className="font-display text-lg font-semibold text-ink tabular-nums">
                {s.uptime90d.toFixed(2)}%
              </span>
            </p>
          )}
        </div>

        {s && (
          <>
            <Panel title={t.status.last48h} legend={t}>
              <div className="flex h-16 items-stretch gap-[2px]">
                {s.hours.map((h) => (
                  <span
                    key={h.hour}
                    title={hourTitle(h, t)}
                    className={`flex-1 rounded-[3px] ${barClass(h.checks, h.ok, h.seen)}`}
                  />
                ))}
              </div>
              <Axis
                left={hourLabel(s.hours[0]?.hour ?? "")}
                right={hourLabel(s.hours[s.hours.length - 1]?.hour ?? "")}
              />
            </Panel>

            <Panel title={t.status.last90d} legend={t}>
              <div className="flex h-16 items-stretch gap-[2px]">
                {s.days.map((d: StatusDay) => (
                  <span
                    key={d.day}
                    title={dayTitle(d, t)}
                    className={`flex-1 rounded-[3px] ${barClass(d.checks, d.ok, d.checks > 0)}`}
                  />
                ))}
              </div>
              <Axis
                left={dayLabel(s.days[0]?.day ?? "")}
                right={dayLabel(s.days[s.days.length - 1]?.day ?? "")}
              />
            </Panel>
          </>
        )}

        {/* Said plainly, because the honest limit of a self-check is the thing
            a reader deserves to know before trusting the green. */}
        <p className="mt-8 max-w-2xl text-sm leading-relaxed text-ink-muted">
          {t.status.method}
        </p>
      </main>
    </>
  );
}

function Headline({
  tone,
  title,
  note,
}: {
  tone: "good" | "bad" | "unknown";
  title: string;
  note: string;
}) {
  const dot =
    tone === "good"
      ? "bg-emerald-500"
      : tone === "bad"
        ? "bg-rose-500"
        : "bg-ink-muted";
  return (
    <div className="flex items-start gap-3">
      {/* The dot supplements the sentence; it never carries the meaning on its
          own. */}
      <span className={`mt-1.5 h-3 w-3 shrink-0 rounded-full ${dot}`} aria-hidden />
      <div>
        <p className="font-display text-xl font-semibold text-ink">{title}</p>
        <p className="mt-1 text-sm text-ink-muted">{note}</p>
      </div>
    </div>
  );
}

function Panel({
  title,
  legend,
  children,
}: {
  title: string;
  legend: Awaited<ReturnType<typeof getT>>;
  children: React.ReactNode;
}) {
  return (
    <section className="mt-6 rounded-3xl border border-line bg-surface p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm font-semibold text-ink">{title}</p>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink-muted">
          <Key className="bg-emerald-500" label={legend.status.legendOk} />
          <Key className="bg-rose-500" label={legend.status.legendBad} />
          <Key className="bg-line-strong" label={legend.status.legendNone} />
        </div>
      </div>
      <div className="mt-4">{children}</div>
    </section>
  );
}

function Key({ className, label }: { className: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <i aria-hidden className={`inline-block h-2.5 w-2.5 rounded-[2px] ${className}`} />
      {label}
    </span>
  );
}

function Axis({ left, right }: { left: string; right: string }) {
  return (
    <div className="mt-2 flex justify-between text-xs text-ink-muted">
      <span>{left}</span>
      <span>{right}</span>
    </div>
  );
}

/** Three states, and the third is not a shade of the other two. */
function barClass(checks: number, ok: number, seen: boolean): string {
  if (!seen || checks === 0) return "bg-line-strong/60";
  if (ok === checks) return "bg-emerald-500";
  // A partial hour is still a failed hour: somebody's order did not go
  // through. Drawn at full strength rather than as a paler red, which would
  // read as "nearly fine".
  return "bg-rose-500";
}

function hourTitle(h: StatusHour, t: Awaited<ReturnType<typeof getT>>): string {
  if (!h.seen || h.checks === 0) return `${hourLabel(h.hour)} — ${t.status.legendNone}`;
  const head = `${hourLabel(h.hour)} — ${h.ok}/${h.checks}`;
  return h.note ? `${head}\n${h.note}` : head;
}

function dayTitle(d: StatusDay, t: Awaited<ReturnType<typeof getT>>): string {
  if (d.checks === 0) return `${dayLabel(d.day)} — ${t.status.legendNone}`;
  return `${dayLabel(d.day)} — ${((d.ok / d.checks) * 100).toFixed(2)}%`;
}

/** "2026-08-06T14" → "06.08 14:00". Split rather than parsed: `new Date` on a
 *  bare date string is UTC midnight, which prints the previous day in any
 *  negative offset — the same class of mistake as reading a day boundary in
 *  UTC on the server. */
function hourLabel(key: string): string {
  const [date, hour] = (key ?? "").split("T");
  const [, m, d] = (date ?? "").split("-");
  return m && d ? `${d}.${m} ${hour ?? "00"}:00` : key;
}

function dayLabel(key: string): string {
  const [, m, d] = (key ?? "").split("-");
  return m && d ? `${d}.${m}` : key;
}
