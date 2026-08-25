"use client";

// Three proposed messages, so the owner is choosing between ideas rather than
// staring at an empty box.
//
// ⚠️ **Nothing here sends.** It fills the textarea and stops. The send button is
// the same one it always was, on the same screen, after the same preview — a
// campaign the assistant could send would be a bill and a reputation, and the
// people who received it are precisely the regulars it was written for.

import { useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { CampaignVariant } from "@/lib/types";

export default function CampaignWriter({
  segment,
  channel,
  onPick,
}: {
  segment: string;
  channel: string;
  onPick: (text: string) => void;
}) {
  const t = useAdminT();
  const [offer, setOffer] = useState("");
  const [busy, setBusy] = useState(false);
  const [variants, setVariants] = useState<CampaignVariant[] | null>(null);
  const [note, setNote] = useState("");

  const write = async () => {
    setBusy(true);
    setNote("");
    try {
      const r = await api.adminCampaignText({ segment, channel, offer });
      if (r.entitled === false) {
        setNote(t.briefing.locked);
      } else if (r.capped) {
        setNote(t.campaignWriter.capped);
      } else if (!r.variants?.length) {
        setNote(t.campaignWriter.empty);
      }
      setVariants(r.variants ?? []);
    } catch {
      setNote(t.campaignWriter.failed);
    } finally {
      // ⚠️ In `finally`. A pair of buttons left spinning after a failed request
      // is this codebase's own recent bug, on the till's pairing screen.
      setBusy(false);
    }
  };

  // ⚠️ Offered only once a segment is chosen: there is nothing to write to
  // otherwise, and a button that returns an error when pressed teaches people
  // the feature does not work.
  if (!segment) return null;

  return (
    <div className="mt-3 rounded-2xl border border-line bg-surface-soft p-3">
      <div className="flex flex-wrap items-center gap-2">
        <input
          className="input flex-1 text-sm"
          value={offer}
          placeholder={t.campaignWriter.offerPh}
          onChange={(e) => setOffer(e.target.value)}
        />
        <button
          type="button"
          onClick={write}
          disabled={busy}
          className="btn btn-ghost text-sm"
        >
          {busy ? t.campaignWriter.busy : t.campaignWriter.write}
        </button>
      </div>
      {/* ⚠️ Spelled out under the field, because the alternative is an owner
          discovering it from a guest at the till: the assistant will not invent
          a discount, so an empty box means a message with no offer in it. */}
      <p className="mt-1 text-xs text-ink-muted">{t.campaignWriter.offerHint}</p>

      {note && <p className="mt-2 text-sm text-ink-soft">{note}</p>}

      {variants && variants.length > 0 && (
        <ul className="mt-3 space-y-2">
          {variants.map((v, i) => (
            <li key={i} className="rounded-xl border border-line bg-surface p-3">
              <p className="text-sm">{v.text}</p>
              {v.note && (
                <p className="mt-1 text-xs italic text-ink-muted">{v.note}</p>
              )}
              <div className="mt-2 flex items-center gap-3 text-xs text-ink-muted">
                <span className="tabular-nums">
                  {v.chars} {t.campaignWriter.chars}
                </span>
                {/* ⚠️ The part count, beside the text, before it is chosen. It
                    is what the send is billed as, and two parts is twice the
                    invoice — a fact that otherwise appears for the first time
                    on the bill. */}
                {typeof v.parts === "number" && (
                  <span
                    className={
                      v.parts > 1 ? "font-semibold text-amber-600" : undefined
                    }
                  >
                    {v.parts} {t.campaignWriter.parts}
                  </span>
                )}
                <button
                  type="button"
                  onClick={() => onPick(v.text)}
                  className="ml-auto font-semibold text-brand hover:underline"
                >
                  {t.campaignWriter.use}
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
