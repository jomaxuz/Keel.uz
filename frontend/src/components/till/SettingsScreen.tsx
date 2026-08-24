"use client";

import { useEffect, useState } from "react";
import { LuMonitor, LuPrinter } from "react-icons/lu";
import { useAdminT } from "@/lib/i18n/admin";
import { bridge, type Status } from "@/lib/tillBridge";
import PrinterSettings from "./PrinterSettings";

// The till's own settings.
//
// ⚠️ **Everything here is about this machine, and nothing here is about the
// restaurant.** Prices, the menu, staff, receipt templates and the branch's
// shared printers are the panel's — an owner sets them once from an office and
// they apply to every screen in the building. What is left is what only the
// person standing at this monoblock can answer: which printer is plugged into
// it, and what it is bound to. A till that could also edit the restaurant would
// be a panel login on the counter, which is the thing the PIN screen exists to
// avoid.
//
// ⚠️ **Behind the same permission as retiring the screen** (`canExit`, which is
// `void` on the server). Not a new permission, and that is the judgement the
// exit button already made and wrote down: a seventh switch would sit unticked
// in every restaurant until each one discovered it — a settings screen that
// exists for nobody. The set of people is already exactly right: every seeded
// management role holds it (Ish boshqaruvchi, Menejer, Zal administratori) and
// Kassir, Ofitsiant, Barmen and Xostes do not.
//
// ⚠️ **The screen only hides it; the server refuses it.** The printer calls go
// to the Go side of this machine rather than to the API, so there is no request
// to refuse — which is precisely why the *destination* is gated in the rail
// rather than the buttons being disabled inside it.
export default function SettingsScreen({ version }: { version?: string }) {
  const t = useAdminT();
  const [status, setStatus] = useState<Status | null>(null);

  useEffect(() => {
    const b = bridge();
    if (!b) return;
    void (async () => {
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
        <Section icon={<LuPrinter />} title={t.till.settings.printer.title}>
          <PrinterSettings />
        </Section>

        <Section icon={<LuMonitor />} title={t.till.settings.device.title}>
          {status ? (
            <dl className="space-y-2 text-sm">
              <Row label={t.till.settings.device.branch} value={status.branchName} />
              <Row label={t.till.settings.device.server} value={status.server} />
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
