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
import type { LabelDesign, LabelDesignView } from "@/lib/types";
import LabelPaper from "./LabelPaper";

/** Which optional lines the chooser offers. ⚠️ The three the renderer reads —
 *  a switch that changed nothing would teach the shop that none of them work. */
const FIELDS = ["shop", "unit", "date"] as const;

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

  return (
    <section className="space-y-4 rounded-2xl border border-line p-4">
      <div>
        <h2 className="text-base font-semibold">{t.labels.design.title}</h2>
        <p className="mt-1 text-xs text-ink-muted">{t.labels.design.hint}</p>
      </div>

      {/* The six, as cards: a picture of the paper, its name, and what it is
          for. ⚠️ The picture is the control — a shop chooses a label the way it
          would choose one off a shelf, not from a dropdown of six words. */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {LABEL_STYLES.map((style) => {
          const drawn = view.options.find((o) => o.style === style);
          const on = draft.style === style;
          return (
            <button
              key={style}
              type="button"
              onClick={() => patch({ style })}
              aria-pressed={on}
              className={`group flex flex-col overflow-hidden rounded-2xl border text-left transition-all ${
                on
                  ? "border-brand ring-2 ring-brand/30"
                  : "border-line hover:border-brand hover:shadow-card"
              }`}
            >
              {/* The paper, on a surface that is not the paper — a white
                  sticker on a white card has no edges, and its width is half of
                  what is being judged here. */}
              <div
                className={`flex justify-center px-4 py-5 transition-colors ${
                  on ? "bg-brand/10" : "bg-ink/5 group-hover:bg-brand/5"
                }`}
              >
                <LabelPaper
                  lines={drawn?.lines ?? []}
                  bars={drawn?.bars ?? false}
                  code={view.sample.barcode}
                  widthMm={draft.widthMm}
                  feedLines={draft.feedLines}
                />
              </div>

              <div className="flex flex-1 flex-col gap-1 border-t border-line p-3">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-sm font-semibold">
                    {t.labels.design.style[style]}
                  </span>
                  <span
                    className={`shrink-0 rounded-full px-2 py-0.5 text-xs ${
                      on
                        ? "bg-brand text-white"
                        : "border border-line-strong text-ink-muted"
                    }`}
                  >
                    {on ? t.labels.design.chosen : t.labels.design.choose}
                  </span>
                </div>
                <p className="text-xs leading-relaxed text-ink-muted">
                  {t.labels.design.styleHint[style]}
                </p>
                {!drawn?.bars && (
                  <p className="text-xs font-medium text-ink-muted">
                    · {t.labels.design.noBars}
                  </p>
                )}
              </div>
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
