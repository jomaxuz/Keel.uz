"use client";

// The card as a sheet of paper, and the few decisions worth offering about it.
//
// ⚠️ **A preview, not a promise.** Everything on this screen is drawn by the
// same function that writes the file — the picture below *is* the download,
// scaled to fit. A preview built any other way is a preview that stops being
// true the first time either half is edited.
//
// ⚠️ **The language is the sheet's, not the panel's.** A Russian-speaking
// accountant is sent a card by an Uzbek-speaking manager every day of the week;
// tying the paper to whoever pressed the button would make that a translation
// job. The file is named in the same language for the same reason.

import { useCallback, useEffect, useRef, useState } from "react";

import {
  CARD_WORDS,
  DEFAULT_DESIGN,
  cardFileName,
  drawTechCard,
  type CardColumn,
  type CardData,
  type CardDesign,
} from "@/lib/techCardPng";
import { useAdminT } from "@/lib/i18n/admin";
import Modal from "@/components/admin/Modal";

const LANGS: CardDesign["lang"][] = ["uz", "ru", "en"];
const ACCENTS = ["#e2590d", "#1f2937", "#0f766e", "#7c3aed", "#b91c1c"];
/** ⚠️ The name is not offered: a card with no ingredient column is not a card,
 *  and a switch that can produce a useless sheet is a switch somebody finds by
 *  accident. */
const OPTIONAL: CardColumn[] = ["no", "qty", "rate", "cost"];

export default function TechCardPrint({
  data,
  onClose,
}: {
  data: CardData;
  onClose: () => void;
}) {
  const t = useAdminT();
  const [design, setDesign] = useState<CardDesign>(DEFAULT_DESIGN);
  const [preview, setPreview] = useState("");
  const [busy, setBusy] = useState(false);
  // ⚠️ Revoked when it is replaced. Every redraw makes another object URL, and
  // a screen somebody fiddles with for two minutes would hold thirty pictures
  // in memory on a machine that is also running a till.
  const url = useRef("");

  const redraw = useCallback(async () => {
    // Drawn at screen resolution: the preview is looked at, not printed, and
    // rendering 300 DPI on every keystroke is a screen that stutters.
    const blobs = await drawTechCard(data, { ...design, dpi: 150 });
    if (!blobs[0]) return;
    if (url.current) URL.revokeObjectURL(url.current);
    url.current = URL.createObjectURL(blobs[0]);
    setPreview(url.current);
  }, [data, design]);

  useEffect(() => {
    void redraw();
  }, [redraw]);

  useEffect(
    () => () => {
      if (url.current) URL.revokeObjectURL(url.current);
    },
    [],
  );

  async function download() {
    setBusy(true);
    try {
      const blobs = await drawTechCard(data, design);
      blobs.forEach((blob, i) => {
        const href = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = href;
        a.download = cardFileName(data.dish, design.lang, i + 1, blobs.length);
        a.click();
        // ⚠️ Released on the next tick rather than immediately: revoking the
        // URL before the browser has started the download cancels it, and the
        // failure is silent — a button that does nothing on some machines.
        setTimeout(() => URL.revokeObjectURL(href), 30_000);
      });
    } finally {
      setBusy(false);
    }
  }

  const toggle = (col: CardColumn) =>
    setDesign({
      ...design,
      columns: design.columns.includes(col)
        ? design.columns.filter((c) => c !== col)
        : // ⚠️ Put back in the canonical order, never appended: a column turned
          // off and on again would otherwise move to the end of the table.
          (["no", "name", "qty", "rate", "cost"] as CardColumn[]).filter(
            (c) => c === col || design.columns.includes(c),
          ),
    });

  return (
    <Modal wide onClose={onClose}>
      <h2 className="text-lg font-semibold">{t.techCards.printTitle}</h2>
      <p className="mt-1 text-xs text-ink-muted">{t.techCards.printHint}</p>

      <div className="mt-4 grid gap-6 lg:grid-cols-[18rem_1fr]">
        <div className="space-y-5">
          <div>
            <p className="text-sm font-medium">{t.techCards.printLang}</p>
            <div className="mt-2 flex gap-2">
              {LANGS.map((l) => (
                <button
                  key={l}
                  type="button"
                  onClick={() => setDesign({ ...design, lang: l })}
                  className={`rounded-lg border px-3 py-1.5 text-sm font-semibold uppercase ${
                    l === design.lang
                      ? "border-brand bg-brand/10"
                      : "border-line text-ink-muted"
                  }`}
                >
                  {l}
                </button>
              ))}
            </div>
            {/* The file lands under the name the sheet is written in — said
                here because it is the one thing the preview cannot show. */}
            <p className="mt-1.5 text-xs text-ink-muted">
              {cardFileName(data.dish, design.lang)}
            </p>
          </div>

          <div>
            <p className="text-sm font-medium">{t.techCards.printAccent}</p>
            <div className="mt-2 flex gap-2">
              {ACCENTS.map((a) => (
                <button
                  key={a}
                  type="button"
                  aria-label={a}
                  onClick={() => setDesign({ ...design, accent: a })}
                  style={{ background: a }}
                  className={`h-7 w-7 rounded-full ring-offset-2 ${
                    a === design.accent ? "ring-2 ring-ink" : ""
                  }`}
                />
              ))}
            </div>
          </div>

          <div>
            <p className="text-sm font-medium">{t.techCards.printColumns}</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {OPTIONAL.map((col) => (
                <button
                  key={col}
                  type="button"
                  onClick={() => toggle(col)}
                  className={`rounded-full border px-3 py-1 text-xs font-semibold ${
                    design.columns.includes(col)
                      ? "border-brand bg-brand/10"
                      : "border-line text-ink-muted"
                  }`}
                >
                  {CARD_WORDS[design.lang][col]}
                </button>
              ))}
            </div>
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={design.logo}
              onChange={(e) => setDesign({ ...design, logo: e.target.checked })}
            />
            {t.techCards.printLogo}
          </label>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={design.signatures}
              onChange={(e) =>
                setDesign({ ...design, signatures: e.target.checked })
              }
            />
            {t.techCards.printSignatures}
          </label>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={design.dpi === 300}
              onChange={(e) =>
                setDesign({ ...design, dpi: e.target.checked ? 300 : 150 })
              }
            />
            {t.techCards.printHiRes}
          </label>
        </div>

        {/* ⚠️ The preview is the file. Shown on a grey ground so the sheet's
            own white edge is visible — a white page on a white card is a
            picture nobody can tell is A4. */}
        <div className="rounded-2xl bg-ink/[0.06] p-4">
          {preview ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={preview}
              alt=""
              className="mx-auto max-h-[70vh] w-auto rounded-lg shadow-lg"
            />
          ) : (
            <p className="py-20 text-center text-sm text-ink-muted">
              {t.common.loading}
            </p>
          )}
        </div>
      </div>

      <div className="mt-5 flex justify-end gap-2">
        <button className="btn-ghost px-4 py-2 text-sm" onClick={onClose}>
          {t.common.cancel}
        </button>
        <button
          className="btn-primary px-4 py-2 text-sm"
          disabled={busy}
          onClick={() => void download()}
        >
          {t.techCards.printDownload}
        </button>
      </div>
    </Modal>
  );
}
