"use client";

// The shift, asked about the moment somebody's PIN is accepted.
//
// ⚠️ **A till with no open shift is not a till.** Every check opened before the
// drawer is opened belongs to no shift, which means it belongs to no count at
// the end of the evening — the money is taken, the receipt prints, and the
// figure the manager reconciles against is quietly short by exactly that
// amount. Nothing on the screen says so, because nothing went wrong.
//
// ⚠️ **So it is a gate, not a banner.** A note at the top of a working screen
// is a note that gets worked past: the first guest is already standing there,
// and opening the drawer is a thing you will do in a minute. The room is not
// drawn until the shift is open.
//
// ⚠️ **Opening it is one number.** What was in the drawer at the start — and
// that is the whole form. A cashier arriving at eleven at night to a queue will
// answer one question and abandon three.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import LangSwitch from "@/components/site/LangSwitch";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import type { CashShift } from "@/lib/types";

import OverrideDialog from "./OverrideDialog";

export type ShiftState = {
  shift: CashShift | null;
  canShift: boolean;
  reload: () => void;
};

/** Ask about the drawer, and keep asking when told to.
 *
 *  Lives in a hook rather than the gate so the header can show which shift is
 *  open and offer to close it — the same fact, read once. */
export function useShift(active: boolean): ShiftState & { loading: boolean } {
  const [shift, setShift] = useState<CashShift | null>(null);
  const [canShift, setCanShift] = useState(false);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);

  const reload = useCallback(() => setTick((n) => n + 1), []);

  useEffect(() => {
    if (!active) return;
    let alive = true;
    (async () => {
      try {
        const r = await api.tillCashShift();
        if (!alive) return;
        setShift(r.open);
        setCanShift(r.canShift);
      } catch {
        // ⚠️ A failed question is not "no shift is open": answering it that way
        // would offer to open a second drawer over the top of a live one, and
        // the server would refuse — after the cashier had typed the float.
        // Left as it was; the gate keeps waiting.
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, [active, tick]);

  return { shift, canShift, loading, reload };
}

export default function ShiftGate({
  state,
  currency,
}: {
  state: ShiftState & { loading: boolean };
  currency: string;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const [float, setFloat] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // A code from somebody who may open the drawer, when this person may not.
  const [override, setOverride] = useState<{ permissionName: string } | null>(
    null,
  );
  const [overrideError, setOverrideError] = useState("");

  async function open(pin?: string) {
    setBusy(true);
    setError(null);
    try {
      await api.tillOpenCashShift({
        openingFloat: Number(float) || 0,
        ...(pin ? { pin } : {}),
      });
      setOverride(null);
      state.reload();
    } catch (err) {
      // The server, not the screen, decides that a manager is needed: the
      // permission may have changed since this person's PIN was accepted.
      const perm = err instanceof ApiError ? err.needsOverride : null;
      if (perm) {
        // A second refusal means the code was wrong, not that the permission
        // changed — said inside the dialog, where the pad still is.
        if (pin) setOverrideError(t.till.overrideWrong);
        setOverride({
          permissionName: err instanceof ApiError ? err.permissionName : "",
        });
      } else {
        setError(err instanceof ApiError ? err.message : t.till.retry);
        setOverride(null);
      }
    } finally {
      setBusy(false);
    }
  }

  if (state.loading) {
    return (
      <div className="flex flex-1 items-center justify-center text-sm text-ink-muted">
        …
      </div>
    );
  }

  return (
    <div className="relative flex flex-1 flex-col items-center justify-center gap-4 p-6">
      {/* Same reason as the lock screen's: this gate replaces the whole till,
          header included, so without it the language switch is unreachable for
          exactly as long as the drawer stays closed. */}
      <div className="absolute right-4 top-4">
        <LangSwitch />
      </div>
      <div className="till-dialog w-full max-w-sm p-5 text-center">
        <p className="text-lg font-semibold">{t.till.shiftClosed}</p>
        <p className="mt-1 text-sm text-ink-muted">{t.till.shiftClosedHint}</p>

        <label className="mt-4 block text-left text-sm">
          <span className="font-medium">{t.till.openingFloatLabel}</span>
          <input
            className="till-input mt-1 w-full text-lg"
            inputMode="numeric"
            autoFocus
            placeholder="0"
            value={float}
            onChange={(e) => setFloat(e.target.value.replace(/\D/g, ""))}
          />
        </label>
        {/* Shown back in words, because the digits were typed on a screen with
            a guest waiting and a zero too many is the ordinary mistake. */}
        {float ? (
          <p className="mt-1 text-left text-xs text-ink-muted">
            {formatPrice(Number(float), currency, lang)}
          </p>
        ) : null}

        {error ? <p className="mt-3 text-sm text-danger">{error}</p> : null}

        <button
          className="till-btn-primary mt-4 min-h-12 w-full text-base"
          disabled={busy}
          onClick={() => open()}
        >
          {t.till.openShiftTitle}
        </button>
        {/* ⚠️ Said before the button is pressed, not after it is refused: a
            waiter who cannot open the drawer needs to fetch somebody, and
            learning that from an error costs the walk back. */}
        {!state.canShift && (
          <p className="mt-2 text-xs text-ink-muted">{t.till.shiftNeedsManager}</p>
        )}
      </div>

      {override && (
        <OverrideDialog
          permissionName={override.permissionName}
          busy={busy}
          error={overrideError}
          onCancel={() => setOverride(null)}
          onSubmit={(pin) => open(pin)}
        />
      )}
    </div>
  );
}
