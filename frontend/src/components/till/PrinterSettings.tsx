"use client";

import { useCallback, useEffect, useState } from "react";
import { LuCheck, LuPrinter, LuRefreshCw } from "react-icons/lu";
import { useAdminT } from "@/lib/i18n/admin";
import { bridge, type PrintConfig } from "@/lib/tillBridge";

// Choosing the printer this machine puts receipts out of.
//
// ⚠️ **A list, never a text field.** The spooler is addressed by the printer's
// name exactly as Windows spells it, and that spelling is not what is written
// on the box: a driver installs "XP-58", the second one on the counter becomes
// "XP-58 (Copy 1)", and a Russian driver names itself in Cyrillic. A name one
// character out is a receipt that never prints and no error anybody sees — the
// spooler discards the job. Choosing from what the machine itself reports
// removes the whole class of failure.
//
// ⚠️ **One component, two places.** It is the last step of pairing a monoblock
// (the installer is standing at the printer, which is the only moment a test
// print can be checked by the person who pressed it) and it is the printer
// section of the till's own settings. A second copy for the second place would
// be two printer screens within a month, and the one being described on the
// phone would be the other one.
export default function PrinterSettings() {
  const t = useAdminT();
  const [cfg, setCfg] = useState<PrintConfig | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // ⚠️ **"Sent", not "printed".** The spooler accepts a job for a printer that
  // is switched off, out of paper or unplugged, and reports nothing — so the
  // only honest thing this screen can say is that it handed the job over.
  const [sent, setSent] = useState(false);

  const load = useCallback(() => {
    const b = bridge();
    if (!b) return;
    void (async () => {
      try {
        setCfg(await b.PrintConfig());
      } catch (err) {
        setError(String(err));
      }
    })();
  }, []);
  useEffect(load, [load]);

  async function save(next: PrintConfig) {
    const b = bridge();
    if (!b) return;
    setCfg(next);
    setSent(false);
    setBusy(true);
    setError("");
    try {
      await b.SavePrintConfig(next);
      // Re-read rather than trusting what was sent: `effective` is the Go
      // side's answer to "which printer will actually be used", and after
      // clearing the choice it is the Windows default — a name this screen
      // cannot work out for itself.
      setCfg(await b.PrintConfig());
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  }

  async function test() {
    const b = bridge();
    if (!b) return;
    setBusy(true);
    setError("");
    setSent(false);
    try {
      await b.TestPrint();
      setSent(true);
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  }

  // ⚠️ Not an error, and worded so. A till open in a browser is a tablet on the
  // floor or a laptop at a demo — it sells perfectly well and prints through
  // the browser's dialog. Telling somebody their install is broken when it is
  // not is how a working screen gets "fixed".
  if (!bridge()) {
    return (
      <p className="rounded-xl bg-ink/5 p-3 text-sm text-ink-soft">
        {t.till.settings.windowsOnly}
      </p>
    );
  }
  if (!cfg) return null; // one frame, not worth a spinner

  const none = cfg.printers.length === 0;

  return (
    <div className="space-y-4">
      {/* ⚠️ The effective printer is stated in words before any control.
          Somebody opening this asks "where do my receipts go", and a select
          showing "Windows standarti" answers a different question — it names
          the setting, not the printer. */}
      <p className="text-sm text-ink-soft">
        {cfg.effective
          ? t.till.settings.printer.current(cfg.effective)
          : t.till.settings.printer.nowhere}
      </p>

      {none ? (
        <p className="rounded-xl bg-ink/5 p-3 text-sm text-ink-soft">
          {t.till.settings.printer.none}
        </p>
      ) : (
        <label className="block">
          <span className="mb-1 block text-xs text-ink-muted">
            {t.till.settings.printer.pick}
          </span>
          <select
            className="input w-full"
            value={cfg.target}
            disabled={busy}
            onChange={(e) => void save({ ...cfg, target: e.target.value })}
          >
            {/* ⚠️ **An explicit "Windows standarti" row, kept first.** The empty
                value is the setting most tills should have — a monoblock's
                receipt printer is the machine's default — and leaving it a
                blank row would read as "not chosen yet" and invite somebody to
                pick a name they do not need. */}
            <option value="">{t.till.settings.printer.systemDefault}</option>
            {cfg.printers.map((p) => (
              <option key={p.name} value={`usb://${p.name}`}>
                {p.name}
                {p.default ? ` — ${t.till.settings.printer.isDefault}` : ""}
              </option>
            ))}
          </select>
        </label>
      )}

      <div className="space-y-2">
        <Toggle
          label={t.till.settings.printer.cut}
          hint={t.till.settings.printer.cutHint}
          on={cfg.cut === true}
          disabled={busy}
          onChange={(v) => void save({ ...cfg, cut: v })}
        />
        <Toggle
          label={t.till.settings.printer.drawer}
          hint={t.till.settings.printer.drawerHint}
          on={cfg.drawer === true}
          disabled={busy}
          onChange={(v) => void save({ ...cfg, drawer: v })}
        />
        <Toggle
          label={t.till.settings.printer.cyrillic}
          hint={t.till.settings.printer.cyrillicHint}
          on={cfg.charset === "cyrillic"}
          disabled={busy}
          onChange={(v) => void save({ ...cfg, charset: v ? "cyrillic" : "latin" })}
        />
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}
      {sent && (
        <p className="flex items-center gap-2 text-sm text-ink-soft">
          <LuCheck className="h-4 w-4 shrink-0" aria-hidden />
          {t.till.settings.printer.sent}
        </p>
      )}

      <div className="flex gap-2">
        <button
          type="button"
          className="btn btn-ghost flex-1 gap-2"
          onClick={load}
          disabled={busy}
        >
          <LuRefreshCw className="h-4 w-4" aria-hidden />
          {t.till.settings.printer.refresh}
        </button>
        {/* ⚠️ **The test is the main button here**, because every setting above
            can be right and still produce nothing: the printer is off, the
            paper is out, or the name belongs to a driver whose printer left the
            building. Only paper proves it, and the person who can look at the
            printer is standing in front of this screen. */}
        <button
          type="button"
          className="btn btn-primary flex-1 gap-2"
          onClick={() => void test()}
          disabled={busy || none}
        >
          <LuPrinter className="h-4 w-4" aria-hidden />
          {t.till.settings.printer.test}
        </button>
      </div>
    </div>
  );
}

function Toggle({
  label,
  hint,
  on,
  disabled,
  onChange,
}: {
  label: string;
  hint: string;
  on: boolean;
  disabled: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-start gap-3">
      <input
        type="checkbox"
        className="mt-1 h-5 w-5 shrink-0 accent-brand"
        checked={on}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="min-w-0">
        <span className="block text-sm text-ink">{label}</span>
        <span className="block text-xs text-ink-muted">{hint}</span>
      </span>
    </label>
  );
}
