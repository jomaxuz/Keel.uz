"use client";

import { useEffect, useState } from "react";
import { LuMonitor, LuPrinter } from "react-icons/lu";
import { useAdminT } from "@/lib/i18n/admin";
import { bridge, type Status } from "@/lib/tillBridge";
import PrinterList from "./PrinterList";
import PrinterSettings from "./PrinterSettings";

// The till's own settings.
//
// ⚠️ **The line is what a person standing here can answer, not what belongs to
// this machine.** Prices, the menu, staff and the receipt *templates* stay the
// panel's: an owner decides them once from an office and they apply to every
// screen in the building. Printers cross that line in the other direction — a
// printer is connected by whoever is holding the box, in the restaurant, and
// the machine they are standing at is this one. So the branch's printer list is
// editable here while what a receipt *says* is not, and a till that could edit
// both would be a panel login on the counter — the thing the PIN screen exists
// to avoid.
//
// ⚠️ **Behind the same permission as retiring the screen** (`canExit`, which is
// `void` on the server). Not a new permission, and that is the judgement the
// exit button already made and wrote down: a seventh switch would sit unticked
// in every restaurant until each one discovered it — a settings screen that
// exists for nobody. The set of people is already exactly right: every seeded
// management role holds it (Ish boshqaruvchi, Menejer, Zal administratori) and
// Kassir, Ofitsiant, Barmen and Xostes do not.
//
// ⚠️ **Two halves, guarded differently, and the difference is worth knowing.**
// The branch printer list is server data, so `/staff/printers` refuses the same
// permission on its own — hiding the rail item there is politeness. The
// machine-local block below talks to this computer's Go side and never reaches
// the API, so for that half there is no request for a server to refuse and
// hiding the *destination* is the enforcement. Both are gated the same way so
// nobody has to remember which is which.
export default function SettingsScreen({
  version,
  onError,
}: {
  version?: string;
  onError?: (msg: string) => void;
}) {
  const t = useAdminT();
  const [status, setStatus] = useState<Status | null>(null);
  // ⚠️ Empty in a browser, which is correct: there is no till build there to
  // report, and an empty row is better than a number that describes something
  // else.
  const [tillVersion, setTillVersion] = useState("");

  useEffect(() => {
    const b = bridge();
    if (!b) return;
    void (async () => {
      try {
        // ⚠️ Asked for separately from the status: an older shell has no
        // `TillVersion` binding, and one missing method must not take the
        // branch and the server rows down with it.
        setTillVersion(await b.TillVersion());
      } catch {
        // An older build of the shell. The row simply does not appear.
      }
      try {
        setStatus(await b.Status());
      } catch {
        // ⚠️ Swallowed: this block is a support answer ("which branch is this
        // machine?"), not a control. A failure to read it must not take the
        // printer settings down with it — those are the reason somebody opened
        // this screen.
      }
    })();
  }, []);

  return (
    // ⚠️ Scrolls inside itself, like every other till destination: the window is
    // 768px tall on the hardware this is sold onto and the page must never be
    // what grows.
    <div className="min-h-0 flex-1 overflow-y-auto p-4">
      <div className="mx-auto w-full max-w-[36rem] space-y-4">
        {/* ⚠️ **The branch's printers first, this machine's second, and the
            order is the answer to "which one do I want".** The list above is
            the restaurant's real printing — every till prints to it and it is
            what puts a ticket at the pass. The block below is the fallback for
            a monoblock whose own printer nobody has added yet; putting it first
            invited somebody to configure the machine and wonder why the kitchen
            never printed. */}
        <Section
          icon={<LuPrinter />}
          title={t.till.settings.printer.shared.title}
        >
          <PrinterList onError={onError} />
        </Section>

        <Section icon={<LuPrinter />} title={t.till.settings.printer.local.title}>
          <p className="mb-3 text-xs text-ink-muted">
            {t.till.settings.printer.local.hint}
          </p>
          <PrinterSettings />
        </Section>

        <Section icon={<LuMonitor />} title={t.till.settings.device.title}>
          {status ? (
            <dl className="space-y-2 text-sm">
              <Row label={t.till.settings.device.branch} value={status.branchName} />
              <Row label={t.till.settings.device.server} value={status.server} />
              {/* ⚠️ **The build on this counter, not the one the server is
                  serving.** These are two different programs updated on two
                  different days: the screen comes from the restaurant's own
                  server, the shell is an installer somebody ran here. This row
                  used to print the first while everybody read it as the second,
                  so a monoblock that had just been updated reported the version
                  it had before — and the update looked as if it had not
                  happened. Both are shown now, because a support call needs
                  both and neither can be derived from the other. */}
              {tillVersion && (
                <Row label={t.till.settings.device.tillVersion} value={tillVersion} />
              )}
              {version && (
                <Row label={t.till.settings.device.version} value={version} />
              )}
              {/* ⚠️ **The relay, named in words.** When it is not running no
                  receipt reaches the branch's own printers and nothing else on
                  any screen says so — "the printer is broken" is the first
                  theory, and it sends somebody to the wrong end of the
                  building. */}
              <Row
                label={t.till.settings.device.agent}
                value={
                  status.agent
                    ? t.till.settings.device.agentOn
                    : t.till.settings.device.agentOff
                }
                warn={!status.agent}
              />
            </dl>
          ) : (
            <p className="text-sm text-ink-soft">
              {t.till.settings.device.browser}
            </p>
          )}
        </Section>
      </div>
    </div>
  );
}

function Section({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="card p-4">
      <h2 className="mb-3 flex items-center gap-2 text-sm font-medium text-ink">
        <span className="text-ink-muted" aria-hidden>
          {icon}
        </span>
        {title}
      </h2>
      {children}
    </section>
  );
}

function Row({
  label,
  value,
  warn,
}: {
  label: string;
  value: string;
  warn?: boolean;
}) {
  if (!value) return null;
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="shrink-0 text-ink-muted">{label}</dt>
      {/* Breaks anywhere: a server address is one long word and would otherwise
          push this row wider than the window. */}
      <dd className={`break-all text-right ${warn ? "text-danger" : "text-ink"}`}>
        {value}
      </dd>
    </div>
  );
}
