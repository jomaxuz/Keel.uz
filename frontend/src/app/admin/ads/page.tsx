"use client";

// Advertising: Meta campaigns, decided from this restaurant's own figures.
//
// ⚠️ **The money is the restaurant's and goes straight to Meta.** The ad account
// is theirs and the card on it is theirs; what is sold here is the work around
// it. The screen says so in a line, because an owner who suspects we spent their
// money stops trusting every other number on the panel — and that suspicion is
// impossible to argue with after the fact.
//
// ⚠️ **What exists today is the door, not the room.** Connecting an account
// needs an app Meta has reviewed; until that lands the honest screen is one that
// says "not connected" plainly. A page that looked connected would be found out
// by somebody who had just pressed a button expecting an advert to run.
//
// ⚠️ The section itself is gated as a bought module (`modulegate.go` on the
// server, `PANEL_ROUTES` here), so this page only ever draws for a restaurant
// that has it. The states below are about Meta, not about the invoice.

import { useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { AdsState } from "@/lib/types";

export default function AdminAdsPage() {
  const t = useAdminT();
  const [state, setState] = useState<AdsState | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api
      .adsState()
      .then(setState)
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.ads.loadFailed),
      );
  }, [t.ads.loadFailed]);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.ads.title}</h1>
        <p className="mt-1 max-w-2xl text-sm text-ink-soft">{t.ads.lead}</p>
        {/* ⚠️ Beside the lead rather than in a footnote: this is the sentence
            that decides whether the next screen is read as ours or as theirs. */}
        <p className="mt-2 max-w-2xl text-xs text-ink-muted">
          {t.ads.spendNote}
        </p>
      </div>

      {error && <p className="text-sm text-danger">{error}</p>}

      {state && !state.on && (
        <div className="card space-y-2 p-4">
          <p className="font-semibold">{t.ads.offTitle}</p>
          <p className="text-sm text-ink-soft">{t.ads.off}</p>
        </div>
      )}

      {state?.on && !state.connected && (
        <div className="card space-y-2 p-4">
          <p className="font-semibold">{t.ads.notConnectedTitle}</p>
          <p className="max-w-2xl text-sm text-ink-soft">{t.ads.notConnected}</p>
        </div>
      )}
    </div>
  );
}
