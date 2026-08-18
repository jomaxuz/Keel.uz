"use client";

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import OverrideDialog from "@/components/till/OverrideDialog";
import { printReceipt } from "@/lib/print";
import type { CashFigures, CashShift } from "@/lib/types";

/**
 * The drawer, on the screen standing in front of it.
 *
 * ⚠️ **Counting cash belongs here, not in the admin panel.** The alternative was
 * giving every cashier a panel login — the customer base, the payment keys, the
 * reports — or having the manager come in next morning to count a drawer
 * somebody else emptied. Both are worse than what they avoided.
 *
 * ⚠️ **The expected figure is not shown until a number has been typed.** Putting
 * it beside an empty box invites it to be copied, and a count that agrees
 * because it was read off the screen is the one record this whole thing exists
 * to prevent. It is the same rule the panel's screen follows, for the same
 * reason, and it is the only part of this that is not shared code.
 */
export default function CashShiftPanel({
  currency,
  onError,
  onChanged,
}: {
  currency: string;
  onError: (msg: string) => void;
  /** Told when the drawer opened or closed, so the screen's own gate agrees
   *  with this panel. ⚠️ Two readers of one fact drift: without this the
   *  cashier closes the shift here and keeps taking orders on a floor that
   *  still believes it is open. */
  onChanged?: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [shift, setShift] = useState<CashShift | null>(null);
  const [figures, setFigures] = useState<CashFigures | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  const [float_, setFloat] = useState("");
  const [counted, setCounted] = useState("");
  const [varianceNote, setVarianceNote] = useState("");

  // The action a manager was asked to authorise, kept so the retry sends the
  // same numbers — retyping a count is how a drawer gets closed at a figure
  // nobody approved.
  const [override, setOverride] = useState<Pending | null>(null);
  const [overrideError, setOverrideError] = useState("");

  const money = (n: number) => formatPrice(n, "UZS", lang);

  const load = useCallback(async () => {
    try {
      const d = await api.tillCashShift();
      setShift(d.open);
      setFigures(d.figures ?? null);
    } catch {
      // Silent: a till that could not read the drawer still sells food, and
      // the panel simply stays collapsed.
    } finally {
      setLoaded(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function act(kind: Pending["kind"], pin: string) {
    setBusy(true);
    setOverrideError("");
    try {
      if (kind === "open") {
        await api.tillOpenCashShift({
          openingFloat: Number(float_) || 0,
          pin: pin || undefined,
        });
        setFloat("");
      } else {
        const res = await api.tillCloseCashShift({
          counted: Number(counted) || 0,
          varianceNote: varianceNote.trim() || undefined,
          pin: pin || undefined,
        });
        setCounted("");
        setVarianceNote("");
        // ⚠️ Printed here, immediately, and not offered as a button on a screen
        // the cashier is about to leave. The Z report is the paper for the
        // shift that just ended; asking somebody at 2am to remember one more
        // tap produces days with no Z report and no way to make one.
        if (res.lines?.length) printReceipt(res.lines, res.widthMM);
        // ⚠️ The register's day may have refused to end — usually because
        // receipts are still unfiled. Reported rather than swallowed: the
        // count succeeded either way, and the cashier is standing next to the
        // machine that can fix it.
        if (res.fiscalNote) onError(res.fiscalNote);
      }
      setOverride(null);
      await load();
      onChanged?.();
    } catch (err) {
      const perm = err instanceof ApiError ? err.needsOverride : null;
      if (perm) {
        if (pin) setOverrideError(t.till.overrideWrong);
        setOverride({
          kind,
          permissionName: err instanceof ApiError ? err.permissionName : "",
        });
      } else {
        onError(err instanceof ApiError ? err.message : t.till.retry);
        setOverride(null);
      }
    } finally {
      setBusy(false);
    }
  }

  // ⚠️ The X report changes nothing, so it needs no permission beyond seeing
  // the drawer — it is the numbers already on this screen, on paper. Requiring
  // a manager to print what is already displayed teaches people that
  // permissions are theatre.
  async function printX() {
    setBusy(true);
    try {
      const res = await api.tillShiftReport();
      printReceipt(res.lines, res.widthMM);
    } catch (err) {
      onError(err instanceof ApiError ? err.message : t.till.retry);
    } finally {
      setBusy(false);
    }
  }

  if (!loaded) return null;

  const typed = counted !== "";
  const variance = (Number(counted) || 0) - (figures?.expected ?? 0);
  const needsNote = typed && variance !== 0 && !varianceNote.trim();

  return (
    <div className="rounded-2xl border border-line bg-surface p-3">
      <button
        className="flex w-full items-center justify-between gap-2 text-left"
        onClick={() => setOpen(!open)}
      >
        <span className="text-sm font-medium">
          {shift ? t.till.shiftOpen : t.till.shiftClosed}
        </span>
        <span className="text-xs text-ink-muted">{open ? "▲" : "▼"}</span>
      </button>

      {/* Even collapsed, the one number worth a glance: what should be in the
          drawer right now. */}
      {shift && figures && !open && (
        <div className="mt-1 text-xs text-ink-muted">
          {t.cash.expected}: {money(figures.expected)}
        </div>
      )}

      {open && !shift && (
        <div className="mt-3">
          <label className="block text-sm">
            <span className="text-ink-muted">{t.cash.openingFloat}</span>
            <input
              className="till-input mt-1 h-11"
              inputMode="numeric"
              value={float_}
              onChange={(e) => setFloat(e.target.value.replace(/\D/g, ""))}
            />
          </label>
          <button
            className="till-btn-primary mt-2.5 w-full"
            disabled={busy}
            onClick={() => void act("open", "")}
          >
            {t.cash.openShift}
          </button>
        </div>
      )}

      {open && shift && figures && (
        <div className="mt-3 space-y-2 text-sm">
          <Row
            label={t.cash.openingFloat}
            value={money(figures.openingFloat)}
          />
          <Row label={t.cash.counterCash} value={money(figures.counterCash)} />
          {figures.settlements > 0 && (
            <Row
              label={t.cash.settlements}
              value={money(figures.settlements)}
            />
          )}
          {/* ⚠️ Apart from the list, because it is **not** in the total: cash a
              courier is still carrying is real money that is not in this
              drawer. Adding it would double every delivery. */}
          {figures.withCouriers > 0 && (
            <div className="rounded-xl bg-ink/5 p-2 text-xs">
              {t.cash.withCouriers}: {money(figures.withCouriers)}
            </div>
          )}

          {/* ⚠️ **Above the count, not beside the close button.** X is the
              report somebody prints *before* touching the drawer — to check
              the till mid-shift, or to see what the numbers say before they
              count. Putting it next to "close the shift" is where a tired
              cashier presses the wrong one. */}
          <button
            className="till-btn w-full"
            disabled={busy}
            onClick={() => void printX()}
          >
            {t.till.xReport}
          </button>

          <label className="block pt-2">
            <span className="text-ink-muted">{t.cash.counted}</span>
            <input
              className="till-input mt-1 h-11"
              inputMode="numeric"
              value={counted}
              onChange={(e) => setCounted(e.target.value.replace(/\D/g, ""))}
            />
          </label>

          {typed && (
            <div
              className={`rounded-xl p-2 text-sm ${
                variance === 0 ? "bg-success/10" : "bg-danger/10"
              }`}
            >
              <div className="flex justify-between text-xs text-ink-muted">
                <span>{t.cash.expected}</span>
                <span>{money(figures.expected)}</span>
              </div>
              <div className="mt-1 flex justify-between font-medium">
                <span>{t.cash.variance}</span>
                <span>
                  {variance > 0 ? "+" : ""}
                  {money(variance)}
                </span>
              </div>
            </div>
          )}

          {typed && variance !== 0 && (
            <input
              className="till-input h-11"
              placeholder={t.cash.varianceNote}
              value={varianceNote}
              onChange={(e) => setVarianceNote(e.target.value)}
            />
          )}

          <button
            className="till-btn-primary w-full"
            disabled={busy || !typed || needsNote}
            onClick={() => void act("close", "")}
          >
            {t.cash.closeShift}
          </button>
        </div>
      )}

      {override && (
        <OverrideDialog
          permissionName={override.permissionName}
          busy={busy}
          error={overrideError}
          onCancel={() => {
            setOverride(null);
            setOverrideError("");
          }}
          onSubmit={(pin) => void act(override.kind, pin)}
        />
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-2">
      <span className="text-ink-muted">{label}</span>
      <span>{value}</span>
    </div>
  );
}

interface Pending {
  kind: "open" | "close";
  permissionName: string;
}
