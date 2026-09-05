"use client";

// How this branch's scales lay out a printed barcode.
//
// ⚠️ **This is the one setting in the product that gets money wrong in
// silence.** Every other field here is visibly wrong when it is wrong: a bad
// phone number does not ring, a bad address is on the map. A bad scale layout
// beeps, shows the right product and prints a receipt with the wrong quantity —
// and nobody finds out until a stocktake, thousands of sales later.
//
// ⚠️ **So the form decodes a real label in front of whoever is filling it in.**
// The numbers here mean nothing to a person; a decoded weight next to the
// sticker in their hand is something they can check. A settings screen that only
// accepted the values would have been correct and useless.

import { useState } from "react";

import { useAdminT } from "@/lib/i18n/admin";
import type { ScaleLabel } from "@/lib/types";

/** The same rule the server applies, so the preview is not a second opinion.
 *
 *  ⚠️ **A second implementation, written down rather than hidden** — the
 *  original is `models.ScaleLabel.Read` and there is nothing to import across
 *  that boundary. It exists only to preview; the till never decodes a label
 *  itself, so a drift here shows a wrong preview rather than charging a wrong
 *  amount. That is the cheap direction to be wrong in, and it is why the preview
 *  is here and the decision is there. */
function preview(s: ScaleLabel, code: string): { item: string; value: number } | null {
  const prefix = s.prefix || "2";
  const itemLen = s.itemLen || 6;
  const valueLen = s.valueLen || 5;
  const trimmed = code.trim();
  if (!trimmed.startsWith(prefix)) return null;
  if (trimmed.length !== prefix.length + itemLen + valueLen + 1) return null;
  if (!/^\d+$/.test(trimmed)) return null;
  const at = prefix.length;
  const raw = Number(trimmed.slice(at + itemLen, at + itemLen + valueLen));
  if (!Number.isFinite(raw)) return null;
  return {
    item: trimmed.slice(at, at + itemLen),
    value: s.value === "price" ? raw : raw / 1000,
  };
}

export default function ScaleSettings({
  value,
  onChange,
}: {
  value: ScaleLabel | undefined;
  onChange: (next: ScaleLabel) => void;
}) {
  const t = useAdminT();
  const s: ScaleLabel = value ?? { enabled: false };
  const [sample, setSample] = useState("");
  const set = (patch: Partial<ScaleLabel>) => onChange({ ...s, ...patch });

  const read = s.enabled && sample ? preview(s, sample) : null;
  const total = prefix(s).length + (s.itemLen || 6) + (s.valueLen || 5) + 1;

  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          className="h-5 w-5"
          checked={s.enabled}
          onChange={(e) => set({ enabled: e.target.checked })}
        />
        <span className="font-medium">{t.scale.enabled}</span>
      </label>
      <p className="text-xs text-ink-muted">{t.scale.enabledHint}</p>

      {s.enabled && (
        <>
          <div className="grid gap-3 sm:grid-cols-4">
            <Num label={t.scale.prefix} value={prefix(s)} onChange={(v) => set({ prefix: v })} text />
            <Num
              label={t.scale.itemLen}
              value={String(s.itemLen || 6)}
              onChange={(v) => set({ itemLen: Number(v) || 0 })}
            />
            <Num
              label={t.scale.valueLen}
              value={String(s.valueLen || 5)}
              onChange={(v) => set({ valueLen: Number(v) || 0 })}
            />
            <label className="block text-sm">
              <span className="font-medium">{t.scale.value}</span>
              <select
                className="mt-1 w-full rounded-xl border border-line bg-raised px-3 py-2"
                value={s.value || "weight"}
                onChange={(e) => set({ value: e.target.value })}
              >
                <option value="weight">{t.scale.valueWeight}</option>
                <option value="price">{t.scale.valuePrice}</option>
              </select>
            </label>
          </div>

          {/* ⚠️ **The length is stated, because it is the thing that is wrong
              most often and the hardest to see.** A scale prints an EAN-13 and
              nothing else; a layout adding up to twelve decodes no real label at
              all, and the symptom is "weighed goods do not work" with no clue
              where to look. */}
          <p className={`text-xs ${total === 13 ? "text-ink-muted" : "text-warn"}`}>
            {t.scale.totalLen(total)}
          </p>

          <div className="rounded-2xl border border-line bg-raised p-3">
            <label className="block text-sm">
              <span className="font-medium">{t.scale.sample}</span>
              <input
                className="mt-1 w-full rounded-xl border border-line bg-surface px-3 py-2"
                value={sample}
                inputMode="numeric"
                autoComplete="off"
                spellCheck={false}
                placeholder="2123456012503"
                onChange={(e) => setSample(e.target.value.trim())}
              />
            </label>
            <p className="mt-2 text-sm">
              {!sample ? (
                <span className="text-ink-muted">{t.scale.sampleHint}</span>
              ) : read ? (
                <span>
                  {t.scale.sampleItem(read.item)} ·{" "}
                  <b>
                    {s.value === "price"
                      ? t.scale.samplePrice(read.value)
                      : t.scale.sampleKg(read.value)}
                  </b>
                </span>
              ) : (
                // ⚠️ Not an error: a code that does not decode is usually a
                // layout that does not match this shop's scales, which is
                // exactly what this form is for.
                <span className="text-warn">{t.scale.sampleNo}</span>
              )}
            </p>
          </div>
        </>
      )}
    </div>
  );
}

function prefix(s: ScaleLabel): string {
  return s.prefix || "2";
}

function Num({
  label,
  value,
  onChange,
  text,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  text?: boolean;
}) {
  return (
    <label className="block text-sm">
      <span className="font-medium">{label}</span>
      <input
        className="mt-1 w-full rounded-xl border border-line bg-raised px-3 py-2"
        value={value}
        inputMode="numeric"
        onChange={(e) =>
          onChange(text ? e.target.value.replace(/\D/g, "") : e.target.value.replace(/\D/g, ""))
        }
      />
    </label>
  );
}
