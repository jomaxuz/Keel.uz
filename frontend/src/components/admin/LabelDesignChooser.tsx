"use client";

// Choosing what a label looks like.
//
// ⚠️ **The shop picks by looking.** A dropdown of six words — "shelf",
// "compact", "full" — asks somebody to imagine paper, and the person setting
// this up is standing at a counter with a roll in their hand. So all six are
// drawn at once, at the width of the paper, and the choice is made the way it
// would be made off a shelf.
//
// ⚠️ **Rendered by the server, by the printer's own layout code.** Drawing the
// six here would be a second implementation of the same character grid, and the
// two would drift — with the difference found by a shop whose stickers do not
// look like the design they picked. The same rule the receipt preview follows.

import { useCallback, useEffect, useRef, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { LABEL_STYLES } from "@/lib/types";
import type { LabelDesign, LabelDesignView, LabelStyle } from "@/lib/types";

/** Which optional lines the chooser offers. ⚠️ The three the renderer reads —
 *  a switch that changed nothing would teach the shop that none of them work. */
const FIELDS = ["shop", "unit", "date"] as const;

/** The emphasis markers escpos reads, taken off the front of a preview line.
 *
 *  ⚠️ **Shown as weight and size rather than dropped.** They are control
 *  characters, so a preview that printed them raw shows nothing at all — and
 *  this whole screen is about which line is the big one. */
function marked(line: string): { text: string; bold: boolean; big: boolean } {
  const head = line.charCodeAt(0);
  if (head === 1) return { text: line.slice(1), bold: true, big: false };
  if (head === 2) return { text: line.slice(1), bold: false, big: true };
  if (head === 3) return { text: line.slice(1), bold: true, big: true };
  return { text: line, bold: false, big: false };
}

export default function LabelDesignChooser() {
  const t = useAdminT();
  const scope = useAdminScope();

  const [view, setView] = useState<LabelDesignView | null>(null);
  const [draft, setDraft] = useState<LabelDesign | null>(null);
  const [saving, setSaving] = useState(false);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  // ⚠️ What is on screen, so the preview of an edit that has been superseded is
  // dropped rather than painted over the newer one.
  const seq = useRef(0);

  useEffect(() => {
    api
      .adminLabelDesign()
      .then((d) => {
        setView(d);
        setDraft(d.design);
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      );
  }, [t.common.loadFailed, scope.scopeKey]);

  // Redrawn by the server on every edit, debounced. ⚠️ A local preview would be
  // faster and would be a different label.
  const redraw = useCallback((next: LabelDesign) => {
    const mine = ++seq.current;
    const id = setTimeout(() => {
      void api
        .previewLabelDesign(next)
        .then((d) => {
          if (mine === seq.current) setView(d);
        })
        .catch(() => {});
    }, 250);
    return () => clearTimeout(id);
  }, []);

  function patch(p: Partial<LabelDesign>) {
    if (!draft) return;
    const next = { ...draft, ...p };
    setDraft(next);
    setNote("");
    redraw(next);
  }

  async function save() {
    if (!draft) return;
    setSaving(true);
    try {
      const d = await api.saveLabelDesign(draft);
      setView(d);
      setDraft(d.design);
      setNote(t.labels.design.saved);
      setError("");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  if (error !== "" && !draft) {
    return <p className="text-sm text-danger">{error}</p>;
  }
  if (!draft || !view) return null;

  const cols = draft.widthMm === 58 ? 32 : 48;

  return (
    <section className="space-y-4 rounded-2xl border border-line p-4">
      <div>
        <h2 className="text-base font-semibold">{t.labels.design.title}</h2>
        <p className="mt-1 text-xs text-ink-muted">{t.labels.design.hint}</p>
      </div>

      {/* The six, at the width of the paper. */}
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {LABEL_STYLES.map((style) => {
          const drawn = view.options.find((o) => o.style === style);
          const on = draft.style === style;
          return (
            <button
              key={style}
              type="button"
              onClick={() => patch({ style })}
              aria-pressed={on}
              className={`flex flex-col gap-2 rounded-xl border p-3 text-left transition-colors ${
                on
                  ? "border-brand bg-brand/5"
                  : "border-line hover:border-brand"
              }`}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm font-semibold">
                  {t.labels.design.style[style]}
                </span>
                <span
                  className={`shrink-0 text-xs ${
                    on ? "font-semibold text-brand" : "text-ink-muted"
                  }`}
                >
                  {on ? t.labels.design.chosen : t.labels.design.choose}
                </span>
              </div>

              {/* Monospace and exactly as wide as the paper, so a line the
                  printer would cut is visibly cut here too. */}
              <div
                className="overflow-x-auto rounded-lg border border-line bg-white p-2 font-mono text-[10px] leading-tight text-black"
                style={{ width: `${cols}ch`, maxWidth: "100%" }}
              >
                {(drawn?.lines ?? []).map((line, i) => {
                  const m = marked(line);
                  return (
                    <div
                      key={i}
                      className={`${m.bold ? "font-bold" : ""} ${
                        m.big ? "text-[15px] leading-tight tracking-tight" : ""
                      } whitespace-pre`}
                    >
                      {m.text || " "}
                    </div>
                  );
                })}
                {/* ⚠️ Sketched, never a real barcode: the bars are drawn by the
                    printer's firmware from the code, and a picture of them here
                    would be a promise about a scan we cannot make. What the
                    shop needs from this box is "there is a code on it, and it
                    is this big". */}
                {drawn?.bars ? (
                  <div className="mt-1">
                    <div
                      className="h-6 w-full"
                      style={{
                        backgroundImage:
                          "repeating-linear-gradient(90deg,#000 0 1px,transparent 1px 3px,#000 3px 5px,transparent 5px 8px)",
                      }}
                    />
                    <div className="text-center text-[9px]">
                      {view.sample.barcode}
                    </div>
                  </div>
                ) : (
                  <div className="mt-1 text-center text-[9px] text-ink-muted">
                    {t.labels.design.noBars}
                  </div>
                )}
              </div>

              <p className="text-xs text-ink-muted">
                {t.labels.design.styleHint[style]}
              </p>
            </button>
          );
        })}
      </div>

      <p className="text-xs text-ink-muted">
        {t.labels.design.sample(view.sample.name)} · {t.labels.design.sampleHint}
      </p>

      {/* What the design does not decide. */}
      <div className="grid gap-3 sm:grid-cols-3">
        <label className="block text-sm">
          <span className="text-xs text-ink-muted">
            {t.labels.design.width}
          </span>
          <select
            className="input mt-1 w-full"
            value={draft.widthMm}
            onChange={(e) => patch({ widthMm: Number(e.target.value) })}
          >
            <option value={58}>58 mm</option>
            <option value={80}>80 mm</option>
          </select>
        </label>

        <label className="block text-sm">
          <span className="text-xs text-ink-muted">{t.labels.design.lang}</span>
          <select
            className="input mt-1 w-full"
            value={draft.lang ?? ""}
            onChange={(e) => patch({ lang: e.target.value })}
          >
            <option value="">O&apos;zbekcha</option>
            <option value="ru">Русский</option>
            <option value="en">English</option>
          </select>
        </label>

        <label className="block text-sm">
          <span className="text-xs text-ink-muted">{t.labels.design.feed}</span>
          <input
            className="input mt-1 w-full"
            inputMode="numeric"
            value={draft.feedLines}
            onChange={(e) =>
              patch({ feedLines: Number(e.target.value.replace(/\D/g, "")) || 0 })
            }
          />
          <span className="mt-1 block text-xs text-ink-muted">
            {t.labels.design.feedHint}
          </span>
        </label>
      </div>

      <div>
        <p className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-ink-muted">
          {t.labels.design.fields}
        </p>
        <div className="flex flex-wrap gap-1.5">
          {FIELDS.map((f) => {
            // ⚠️ **Missing means shown.** A shop that has never opened this has
            // no entry for any of them, and drawing that as "off" would show
            // three switches turned off beside a preview that prints the lines.
            const on = draft.fields?.[f] !== false;
            return (
              <button
                key={f}
                type="button"
                aria-pressed={on}
                onClick={() =>
                  patch({ fields: { ...(draft.fields ?? {}), [f]: !on } })
                }
                className={`rounded-full border px-3 py-1.5 text-sm font-medium transition-colors ${
                  on
                    ? "border-brand bg-brand/10 text-brand"
                    : "border-line-strong text-ink-muted hover:border-brand"
                }`}
              >
                {t.labels.design.field[f]}
              </button>
            );
          })}
        </div>
      </div>

      <div className="flex items-center gap-3">
        <button
          className="btn-primary px-4 py-2 text-sm"
          disabled={saving}
          onClick={() => void save()}
        >
          {t.labels.design.save}
        </button>
        {note && <span className="text-sm text-success">{note}</span>}
        {error !== "" && <span className="text-sm text-danger">{error}</span>}
      </div>
    </section>
  );
}
