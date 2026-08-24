"use client";

import { useCallback, useEffect, useState } from "react";
import { LuCable, LuNetwork, LuPlus, LuPrinter, LuUsb } from "react-icons/lu";
import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import {
  buildTarget,
  connectionOf,
  DEFAULT_PORT,
  readTarget,
  type Connection,
} from "@/lib/printerTarget";
import { bridge, type Installed } from "@/lib/tillBridge";
import type { Printer } from "@/lib/types";

// The branch's printers, connected from the counter.
//
// ⚠️ **The same records the panel edits** (`receipt_settings.printers`) — what
// the queue reads and the agent prints from. A printer added here works for
// every till in the branch and prints kitchen tickets. A till-local list would
// be a second answer to a question that already has one owner, and the first
// symptom would be a kitchen ticket that prints from one screen and not the
// next.
//
// ⚠️ **This section works in a browser too**, unlike the machine-local one
// below it: the data is the server's. Only the USB list needs the Windows
// application, and only because Windows is the one that knows the names.
export default function PrinterList({
  onError,
}: {
  onError?: (msg: string) => void;
}) {
  const t = useAdminT();
  const s = t.till.settings.printer.shared;
  const [printers, setPrinters] = useState<Printer[] | null>(null);
  const [kinds, setKinds] = useState<string[]>([]);
  const [editing, setEditing] = useState<Printer | null>(null);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState("");

  const load = useCallback(() => {
    void (async () => {
      try {
        const res = await api.tillPrinters();
        setPrinters(res.printers);
        setKinds(res.kinds);
      } catch (e) {
        // ⚠️ An empty list rather than nothing: a manager who cannot reach the
        // server should still see the section and its message, not a screen
        // that renders as if the feature were missing.
        setPrinters([]);
        onError?.(e instanceof ApiError ? e.message : String(e));
      }
    })();
  }, [onError]);
  useEffect(load, [load]);

  async function save(next: Printer[]) {
    setBusy(true);
    setNote("");
    try {
      const res = await api.tillSavePrinters(next);
      // ⚠️ The server's copy, not the one just sent: it drops an address that
      // would not parse and mints ids for new rows. Keeping the local list
      // would show a printer that was never stored.
      setPrinters(res.printers);
      setEditing(null);
    } catch (e) {
      onError?.(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function test(p: Printer) {
    setBusy(true);
    setNote("");
    try {
      const res = await api.tillTestPrinter(p.id);
      // ⚠️ **Zero queued is not success.** A printer that prints no kind of
      // receipt accepts the button and does nothing, which reads as a broken
      // printer rather than as an unfinished setting one tap away.
      setNote(res.queued > 0 ? s.queued : s.notQueued);
    } catch (e) {
      onError?.(e instanceof ApiError ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!printers) return null; // one frame, not worth a spinner

  return (
    <div className="space-y-3">
      <p className="text-xs text-ink-muted">{s.hint}</p>

      {printers.length === 0 ? (
        <p className="rounded-xl bg-ink/5 p-3 text-sm text-ink-soft">{s.none}</p>
      ) : (
        <ul className="space-y-2">
          {printers.map((p) => (
            <li
              key={p.id}
              className="flex flex-wrap items-center gap-2 rounded-xl border border-line p-3"
            >
              <ConnectionIcon target={p.target} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="truncate text-sm text-ink">{p.name || p.target}</span>
                  {p.disabled && (
                    <span className="badge shrink-0">{s.disabled}</span>
                  )}
                </div>
                {/* The address in full: it is what somebody compares against
                    the sticker on the printer when a ticket stops coming out. */}
                <div className="truncate text-xs text-ink-muted">{p.target}</div>
                <div className="truncate text-xs text-ink-muted">
                  {p.kinds.length
                    ? p.kinds.map((k) => kindLabel(t, k)).join(" · ")
                    : s.kindsEmpty}
                </div>
              </div>
              <button
                type="button"
                className="btn btn-ghost shrink-0 text-sm"
                disabled={busy}
                onClick={() => void test(p)}
              >
                {s.test}
              </button>
              <button
                type="button"
                className="btn btn-ghost shrink-0 text-sm"
                disabled={busy}
                onClick={() => setEditing(p)}
              >
                {s.edit}
              </button>
            </li>
          ))}
        </ul>
      )}

      {note && <p className="text-sm text-ink-soft">{note}</p>}

      <button
        type="button"
        className="btn btn-ghost w-full gap-2"
        disabled={busy}
        onClick={() =>
          setEditing({
            id: "",
            name: "",
            target: "",
            // ⚠️ **Nothing ticked by default**, matching the server: a printer
            // somebody added and did not finish must not start taking every
            // bill in the building onto the pass's roll.
            kinds: [],
            copies: 1,
            cut: true,
          })
        }
      >
        <LuPlus className="h-4 w-4" aria-hidden />
        {s.add}
      </button>

      {editing && (
        <PrinterForm
          value={editing}
          kinds={kinds}
          busy={busy}
          onCancel={() => setEditing(null)}
          onRemove={
            editing.id
              ? () => {
                  if (!window.confirm(s.removeConfirm(editing.name || editing.target)))
                    return;
                  void save(printers.filter((p) => p.id !== editing.id));
                }
              : undefined
          }
          onSave={(next) =>
            void save(
              next.id
                ? printers.map((p) => (p.id === next.id ? next : p))
                : [...printers, next],
            )
          }
        />
      )}
    </div>
  );
}

function kindLabel(t: ReturnType<typeof useAdminT>, kind: string): string {
  const k = t.till.settings.printer.shared.kind;
  switch (kind) {
    case "kitchen":
      return k.kitchen;
    case "till":
      return k.till;
    case "customer":
      return k.customer;
    case "precheck":
      return k.precheck;
    default:
      // ⚠️ Shown as it came rather than dropped: a kind this build does not
      // know is one a newer server added, and hiding it would make the row look
      // like it prints less than it does.
      return kind;
  }
}

/** ⚠️ Derived from the address, never stored beside it: a second copy is a
 *  second fact that disagrees the day somebody edits the line. */
function ConnectionIcon({ target }: { target: string }) {
  const how = connectionOf(target);
  const cls = "h-5 w-5 shrink-0 text-ink-muted";
  if (how === "usb") return <LuUsb className={cls} aria-hidden />;
  if (how === "lan") return <LuNetwork className={cls} aria-hidden />;
  return <LuCable className={cls} aria-hidden />;
}

function PrinterForm({
  value,
  kinds,
  busy,
  onSave,
  onCancel,
  onRemove,
}: {
  value: Printer;
  kinds: string[];
  busy: boolean;
  onSave: (p: Printer) => void;
  onCancel: () => void;
  onRemove?: () => void;
}) {
  const t = useAdminT();
  const s = t.till.settings.printer.shared;
  const parsed = readTarget(value.target);

  const [name, setName] = useState(value.name);
  const [how, setHow] = useState<Connection>(parsed.how);
  const [usbName, setUsbName] = useState(parsed.name);
  const [ip, setIp] = useState(parsed.ip);
  const [port, setPort] = useState(parsed.port);
  const [raw, setRaw] = useState(parsed.raw);
  const [checked, setChecked] = useState<string[]>(value.kinds ?? []);
  const [copies, setCopies] = useState(value.copies ?? 1);
  const [cut, setCut] = useState(value.cut ?? false);
  const [drawer, setDrawer] = useState(value.drawer ?? false);
  const [disabled, setDisabled] = useState(value.disabled ?? false);
  const [error, setError] = useState("");

  // What this machine has, when there is a machine to ask. ⚠️ Only the names
  // are needed and only on Windows; a browser till still reaches every other
  // field, because the address can be typed.
  const [installed, setInstalled] = useState<Installed[]>([]);
  useEffect(() => {
    const b = bridge();
    if (!b) return;
    void b
      .Printers()
      .then(setInstalled)
      .catch(() => {
        // A machine whose spooler refused is one where the name is typed. Not
        // worth a message: the field below is already the answer.
      });
  }, []);

  function submit() {
    const target = buildTarget(how, { name: usbName, ip, port, raw });
    if (!name.trim()) return setError(s.nameRequired);
    // ⚠️ Refused here rather than saved and dropped by the server. An address
    // the server discards leaves somebody looking at a list that did not change
    // and no reason why.
    if (!target) return setError(how === "lan" ? s.ipRequired : s.nameRequired);
    onSave({
      ...value,
      name: name.trim(),
      target,
      kinds: checked,
      copies,
      cut,
      drawer,
      disabled,
    });
  }

  return (
    <div className="space-y-3 rounded-xl border border-line-strong p-3">
      <Field label={s.name}>
        <input
          className="input w-full"
          value={name}
          placeholder={s.namePlaceholder}
          onChange={(e) => setName(e.target.value)}
        />
      </Field>

      <Field label={s.how}>
        <select
          className="input w-full"
          value={how}
          onChange={(e) => setHow(e.target.value as Connection)}
        >
          <option value="usb">{s.usb}</option>
          <option value="lan">{s.lan}</option>
          <option value="other">{s.other}</option>
        </select>
      </Field>

      {how === "usb" &&
        (installed.length > 0 ? (
          <Field label={s.usbPick}>
            <select
              className="input w-full"
              value={usbName}
              onChange={(e) => setUsbName(e.target.value)}
            >
              <option value="">—</option>
              {installed.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.name}
                </option>
              ))}
              {/* ⚠️ A name saved on another machine is kept in the list, or
                  editing a kitchen printer from a second till would silently
                  clear it. */}
              {usbName && !installed.some((p) => p.name === usbName) && (
                <option value={usbName}>{usbName}</option>
              )}
            </select>
          </Field>
        ) : (
          <Field label={s.usbManual} hint={s.usbNone}>
            <input
              className="input w-full"
              value={usbName}
              onChange={(e) => setUsbName(e.target.value)}
            />
          </Field>
        ))}

      {how === "lan" && (
        <div className="flex gap-2">
          <div className="flex-1">
            <Field label={s.ip}>
              <input
                className="input w-full"
                inputMode="decimal"
                placeholder="192.168.1.50"
                value={ip}
                onChange={(e) => setIp(e.target.value)}
              />
            </Field>
          </div>
          <div className="w-28">
            <Field label={s.port} hint={s.portHint}>
              <input
                className="input w-full"
                inputMode="numeric"
                value={port}
                onChange={(e) => setPort(Number(e.target.value) || DEFAULT_PORT)}
              />
            </Field>
          </div>
        </div>
      )}

      {how === "other" && (
        <Field label={s.target} hint={s.targetHint}>
          <input
            className="input w-full"
            value={raw}
            onChange={(e) => setRaw(e.target.value)}
          />
        </Field>
      )}

      <Field label={s.prints} hint={checked.length ? undefined : s.kindsEmpty}>
        <div className="flex flex-wrap gap-2">
          {kinds.map((k) => (
            <label
              key={k}
              className="flex cursor-pointer items-center gap-2 rounded-lg border border-line px-3 py-2 text-sm"
            >
              <input
                type="checkbox"
                className="h-4 w-4 accent-brand"
                checked={checked.includes(k)}
                onChange={(e) =>
                  setChecked(
                    e.target.checked
                      ? [...checked, k]
                      : checked.filter((x) => x !== k),
                  )
                }
              />
              {kindLabel(t, k)}
            </label>
          ))}
        </div>
      </Field>

      <div className="flex gap-2">
        <div className="w-28">
          <Field label={s.copies}>
            <input
              className="input w-full"
              inputMode="numeric"
              value={copies}
              onChange={(e) => setCopies(Math.max(1, Number(e.target.value) || 1))}
            />
          </Field>
        </div>
      </div>

      <Check label={t.till.settings.printer.cut} on={cut} onChange={setCut} />
      <Check
        label={t.till.settings.printer.drawer}
        on={drawer}
        onChange={setDrawer}
      />
      <Check label={s.off} hint={s.offHint} on={disabled} onChange={setDisabled} />

      {error && <p className="text-sm text-danger">{error}</p>}

      <div className="flex gap-2">
        {onRemove && (
          <button
            type="button"
            className="btn btn-ghost text-danger"
            disabled={busy}
            onClick={onRemove}
          >
            {s.remove}
          </button>
        )}
        <button
          type="button"
          className="btn btn-ghost flex-1"
          disabled={busy}
          onClick={onCancel}
        >
          {s.cancel}
        </button>
        <button
          type="button"
          className="btn btn-primary flex-1"
          disabled={busy}
          onClick={submit}
        >
          {s.save}
        </button>
      </div>
    </div>
  );
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs text-ink-muted">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-ink-muted">{hint}</span>}
    </label>
  );
}

function Check({
  label,
  hint,
  on,
  onChange,
}: {
  label: string;
  hint?: string;
  on: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-start gap-3">
      <input
        type="checkbox"
        className="mt-1 h-5 w-5 shrink-0 accent-brand"
        checked={on}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="min-w-0">
        <span className="block text-sm text-ink">{label}</span>
        {hint && <span className="block text-xs text-ink-muted">{hint}</span>}
      </span>
    </label>
  );
}

export { LuPrinter };
