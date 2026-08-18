"use client";

// Designing the three receipts.
//
// ⚠️ **The preview is rendered on the server**, by the same function that drives
// the printer. Drawing it here would be a second implementation of the same
// character grid, and the two would drift — with the difference discovered by a
// restaurant whose paper does not look like the thing they designed. The same
// rule as the reports: one calculation, two outputs.
//
// ⚠️ **Three designs, not one.** They are read by three people under three
// pressures: a cook at a pass with three tickets waiting, the person holding
// the drawer, and a guest checking what they paid. The kitchen ticket carries
// no prices at all, and there is deliberately no switch to add them.

import { useCallback, useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import type { Printer, ReceiptPreview, ReceiptTemplate } from "@/lib/types";
import PrintersEditor from "./PrintersEditor";

type Kind = "kitchen" | "till" | "customer";

/** Which optional lines each receipt offers.
 *
 *  ⚠️ Per receipt, never a shared list. "Show the cashier" is meaningless at the
 *  pass and "show the dish comment" is the entire reason the kitchen ticket
 *  exists — a single list of checkboxes would offer both to both and teach the
 *  owner that half the switches do nothing. */
const FIELDS: Record<Kind, string[]> = {
  kitchen: ["comment", "time", "server"],
  till: ["time", "cashier", "change"],
  customer: ["time", "server", "address", "phone", "change"],
};

export default function ReceiptEditor() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [draft, setDraft] = useState<Record<Kind, ReceiptTemplate> | null>(
    null,
  );
  const [preview, setPreview] = useState<ReceiptPreview | null>(null);
  const [kind, setKind] = useState<Kind>("customer");
  const [saving, setSaving] = useState(false);
  // ⚠️ Loaded and saved with the templates, because they are one setting: the
  // paper width a template is designed for belongs to the printer it comes out
  // of, and splitting them into two screens is how they drift apart.
  const [printers, setPrinters] = useState<Printer[]>([]);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api
      .adminReceipts()
      .then((d) => {
        setDraft({ kitchen: d.kitchen, till: d.till, customer: d.customer });
        // ⚠️ `?? []` — a branch that has never had a printer sends null, and
        // the editor maps over this. The same JSON trap as the floor plan's
        // slices, one collection along.
        setPrinters(d.printers ?? []);
      })
      .catch(() => setError(t.common.loadFailed));
  }, [t, scope.scopeKey]);

  // ⚠️ Redrawn from the server on every edit, debounced. A local preview would
  // be faster and would be a different receipt.
  const refresh = useCallback(async (d: Record<Kind, ReceiptTemplate>) => {
    try {
      setPreview(await api.previewReceipts(d));
    } catch {
      // A preview that could not be fetched is left as it was rather than
      // blanked: an empty paper reads as a broken design.
    }
  }, []);

  useEffect(() => {
    if (!draft) return;
    const id = setTimeout(() => void refresh(draft), 300);
    return () => clearTimeout(id);
  }, [draft, refresh]);

  if (!draft) {
    return <p className="text-sm text-ink-muted">{t.common.loading}</p>;
  }

  const tpl = draft[kind];
  const patch = (p: Partial<ReceiptTemplate>) =>
    setDraft({ ...draft, [kind]: { ...tpl, ...p } });

  async function save() {
    if (!draft) return;
    setSaving(true);
    setError("");
    setNote("");
    try {
      await api.saveReceipts({ ...draft, printers });
      setNote(t.receipts.saved);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <p className="text-xs leading-relaxed text-ink-muted">
        {t.receipts.intro}
      </p>

      <div className="mt-3 flex flex-wrap gap-2">
        {(["customer", "till", "kitchen"] as Kind[]).map((k) => (
          <button
            key={k}
            type="button"
            onClick={() => setKind(k)}
            className={
              kind === k
                ? "rounded-xl bg-ink px-3 py-2 text-xs font-semibold text-surface"
                : "rounded-xl border border-line px-3 py-2 text-xs font-semibold text-ink-soft hover:bg-ink/5"
            }
          >
            {t.receipts.kinds[k]}
          </button>
        ))}
      </div>

      <div className="mt-4 grid gap-5 lg:grid-cols-[1fr_auto]">
        <div className="space-y-3">
          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={tpl.enabled}
              onChange={(e) => patch({ enabled: e.target.checked })}
            />
            <span>
              {t.receipts.enabled}
              <span className="mt-0.5 block text-xs text-ink-muted">
                {t.receipts.enabledHint[kind]}
              </span>
            </span>
          </label>

          <label className="block text-sm">
            <span className="font-medium">{t.receipts.width}</span>
            <select
              className="input mt-1"
              value={tpl.widthMm}
              onChange={(e) => patch({ widthMm: Number(e.target.value) })}
            >
              <option value={80}>80 mm</option>
              <option value={58}>58 mm</option>
            </select>
            {/* ⚠️ Said out loud, because nothing in the software can see the
                paper: an 80 mm design on a 58 mm roll loses the right-hand end
                of every line, and the totals column with it. */}
            <span className="mt-1 block text-xs text-ink-muted">
              {t.receipts.widthHint}
            </span>
          </label>

          <label className="block text-sm">
            <span className="font-medium">{t.receipts.header}</span>
            <input
              className="input mt-1"
              value={tpl.header}
              onChange={(e) => patch({ header: e.target.value })}
            />
          </label>

          <label className="block text-sm">
            <span className="font-medium">{t.receipts.footer}</span>
            <input
              className="input mt-1"
              value={tpl.footer}
              onChange={(e) => patch({ footer: e.target.value })}
            />
          </label>

          <div className="space-y-1">
            <span className="text-sm font-medium">{t.receipts.fields}</span>
            {FIELDS[kind].map((f) => (
              <label key={f} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  // ⚠️ A missing key means shown — matching the server, where
                  // an absent field on an old template must not silently drop
                  // a line that was printing.
                  checked={tpl.fields?.[f] !== false}
                  onChange={(e) =>
                    patch({
                      fields: { ...(tpl.fields ?? {}), [f]: e.target.checked },
                    })
                  }
                />
                <span>{t.receipts.fieldNames[f] ?? f}</span>
              </label>
            ))}
          </div>

          <label className="block text-sm">
            <span className="font-medium">{t.receipts.feed}</span>
            <input
              className="input mt-1 w-24"
              inputMode="numeric"
              value={tpl.feedLines}
              onChange={(e) =>
                patch({
                  feedLines: Number(e.target.value.replace(/\D/g, "")) || 0,
                })
              }
            />
            <span className="mt-1 block text-xs text-ink-muted">
              {t.receipts.feedHint}
            </span>
          </label>
        </div>

        {/* ---- The paper ---- */}
        <div>
          <div className="text-xs text-ink-muted">{t.receipts.preview}</div>
          {/* Monospace and exactly as wide as the paper, so a line that will be
              cut on the printer is visibly cut here. */}
          <pre
            className="mt-1 overflow-x-auto rounded-xl border border-line bg-white p-3 font-mono text-[11px] leading-tight text-black"
            style={{ width: `${tpl.widthMm === 58 ? 32 : 48}ch` }}
          >
            {preview?.[kind]?.join("\n") ?? ""}
          </pre>
        </div>
      </div>

      {/* ⚠️ On the same page as the templates, and saved by the same button.
          The paper width a template is designed for belongs to the printer it
          comes out of — two screens is how a 58 mm design ends up pointed at an
          80 mm roll. */}
      <PrintersEditor printers={printers} onChange={setPrinters} />

      {note && <p className="mt-3 text-sm text-success">{note}</p>}
      {error && <p className="mt-3 text-sm text-danger">{error}</p>}

      <button
        type="button"
        className="btn btn-primary mt-4"
        disabled={saving}
        onClick={() => void save()}
      >
        {saving ? t.common.saving : t.receipts.save}
      </button>
    </div>
  );
}
