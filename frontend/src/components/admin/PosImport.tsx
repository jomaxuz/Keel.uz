"use client";

// Moving in from iiko, r_keeper, Clopos, Poster or Jowi.
//
// ⚠️ **This is the objection, not a nicety.** A restaurant on one of those does
// not stay because it prefers the software — it stays because the ingredient
// list, the tech cards and the stock are in there, and moving them by hand is
// weeks of somebody's time. "Hamma narsa boshqa POS'da" is the last thing
// between a signed customer and a live one.
//
// ⚠️ **A file rather than an integration.** None of those systems documents an
// API for reading tech cards, and a restaurant on its way out has usually lost
// its API access anyway — the licence is the reseller's. All of them export to
// Excel.
//
// ⚠️ **Two steps, and here the reason is sharper than for the menu importer.**
// A wrong tech card is not visibly wrong: every dish still has a cost, the
// reports still add up, and the mistake is found at a stocktake weeks later. So
// nothing is written until the owner has read the list, and every line the
// import could not understand is shown at the top rather than skipped.

import { useState } from "react";

import { api, ApiError, posImportPreview, type PosImportLine } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";

type Kind = "ingredients" | "recipes" | "stock";

// The fields the owner can point at a column. ⚠️ Per kind, because offering
// "which column is the dish" on an ingredient list is offering a question with
// no answer, and a mapping screen full of irrelevant rows is one nobody reads.
const FIELDS: Record<Kind, string[]> = {
  ingredients: ["name", "unit", "price", "category", "note"],
  recipes: ["dish", "name", "qty", "unit"],
  stock: ["name", "qty", "unit"],
};

export default function PosImport({
  onDone,
  onClose,
}: {
  onDone: () => void;
  onClose: () => void;
}) {
  const t = useAdminT();
  const [kind, setKind] = useState<Kind>("ingredients");
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const [header, setHeader] = useState<string[]>([]);
  const [columns, setColumns] = useState<Record<string, number>>({});
  const [lines, setLines] = useState<PosImportLine[] | null>(null);
  const [picked, setPicked] = useState<Set<number>>(new Set());
  const [done, setDone] = useState<string | null>(null);

  async function read(cols?: Record<string, number>) {
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      const res = await posImportPreview(kind, file, cols);
      setHeader(res.header);
      setColumns(res.columns);
      setLines(res.lines);
      // ⚠️ Only the lines that can actually be imported. Ticking a line with a
      // problem would mean the apply silently drops it, and the count at the
      // end would not match what the owner selected.
      setPicked(
        new Set(
          res.lines
            .map((l, i) => (l.problem || l.exists ? -1 : i))
            .filter((i) => i >= 0),
        ),
      );
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  async function apply() {
    if (!lines) return;
    setBusy(true);
    setError("");
    try {
      const chosen = lines.filter((_, i) => picked.has(i));
      const res = await api.posImportApply(kind, chosen);
      const parts: string[] = [];
      if (res.created) parts.push(t.posImport.created(res.created));
      if (res.updated) parts.push(t.posImport.updated(res.updated));
      if (res.skipped) parts.push(t.posImport.skipped(res.skipped));
      // ⚠️ Named, not counted. "Twelve ingredients were not found" is a number
      // somebody has to go and reconstruct; the names are the work itself.
      for (const n of res.missingProduct ?? [])
        parts.push(t.posImport.noProduct(n));
      for (const n of res.missingDish ?? []) parts.push(t.posImport.noDish(n));
      setDone(parts.join(" · ") || t.posImport.nothing);
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  function setColumn(field: string, idx: number) {
    const next = { ...columns, [field]: idx };
    setColumns(next);
    void read(next);
  }

  if (done) {
    return (
      <div className="space-y-4">
        <p className="text-sm">{done}</p>
        <div className="flex justify-end">
          <button
            type="button"
            className="btn-primary px-4 py-2"
            onClick={onClose}
          >
            {t.common.close}
          </button>
        </div>
      </div>
    );
  }

  const problems = (lines ?? []).filter((l) => l.problem).length;

  return (
    <div className="space-y-4">
      {!lines && (
        <>
          <p className="text-sm text-ink-soft">{t.posImport.lead}</p>

          <div className="flex flex-wrap gap-2">
            {(["ingredients", "recipes", "stock"] as Kind[]).map((k) => (
              <button
                key={k}
                type="button"
                onClick={() => setKind(k)}
                className={`rounded-xl px-3 py-2 text-sm ${
                  kind === k
                    ? "bg-ink font-medium text-surface"
                    : "border border-line text-ink-muted"
                }`}
              >
                {t.posImport.kinds[k]}
              </button>
            ))}
          </div>
          <p className="text-xs text-ink-muted">
            {t.posImport.kindHints[kind]}
          </p>

          {/* ⚠️ The order matters and is said out loud: tech cards point at
              ingredients by name, and a stock sheet counts them. Importing
              either before the ingredient list means every line comes back
              "not found", which reads as the file being wrong. */}
          <p className="rounded-xl border border-line bg-page px-3 py-2 text-xs text-ink-soft">
            {t.posImport.order}
          </p>

          <label className="block text-sm">
            <span className="font-medium">{t.posImport.fileLabel}</span>
            <input
              type="file"
              accept=".csv,.xlsx,.xlsm,text/csv"
              className="mt-1 block w-full text-sm"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
            />
          </label>
        </>
      )}

      {error && <p className="text-sm text-danger">{error}</p>}

      {lines && (
        <>
          {/* ---- Which column is which ---- */}
          <div className="rounded-xl border border-line p-3">
            <p className="text-sm font-medium">{t.posImport.mapping}</p>
            <p className="mt-0.5 text-xs text-ink-muted">
              {t.posImport.mappingHint}
            </p>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              {FIELDS[kind].map((f) => (
                <label key={f} className="flex items-center gap-2 text-sm">
                  <span className="w-28 shrink-0 text-ink-muted">
                    {t.posImport.fields[f] ?? f}
                  </span>
                  <select
                    className="input flex-1"
                    value={columns[f] ?? -1}
                    onChange={(e) => setColumn(f, Number(e.target.value))}
                  >
                    <option value={-1}>{t.posImport.noColumn}</option>
                    {header.map((h, i) => (
                      <option key={i} value={i}>
                        {h || `#${i + 1}`}
                      </option>
                    ))}
                  </select>
                </label>
              ))}
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm font-medium">
              {t.posImport.found(lines.length, picked.size)}
              {problems > 0 && (
                <span className="ml-2 text-danger">
                  {t.posImport.problems(problems)}
                </span>
              )}
            </p>
            <button
              type="button"
              className="text-sm text-brand hover:underline"
              onClick={() =>
                setPicked(
                  new Set(
                    lines
                      .map((l, i) => (l.problem ? -1 : i))
                      .filter((i) => i >= 0),
                  ),
                )
              }
            >
              {t.posImport.pickAll}
            </button>
          </div>

          <div className="max-h-[46vh] space-y-1.5 overflow-y-auto pr-1">
            {lines.map((l, i) => (
              <div
                key={i}
                className={`rounded-lg border px-2.5 py-2 text-sm ${
                  l.problem
                    ? "border-danger/40 bg-danger/5"
                    : picked.has(i)
                      ? "border-brand/50 bg-brand/[0.03]"
                      : "border-line"
                }`}
              >
                <div className="flex items-start gap-2.5">
                  <input
                    type="checkbox"
                    className="mt-1"
                    checked={picked.has(i)}
                    disabled={!!l.problem}
                    onChange={() => {
                      const next = new Set(picked);
                      if (next.has(i)) next.delete(i);
                      else next.add(i);
                      setPicked(next);
                    }}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-baseline gap-x-2">
                      {l.dish && (
                        <span className="font-medium">{l.dish} ←</span>
                      )}
                      <span className={l.dish ? "" : "font-medium"}>
                        {l.name}
                      </span>
                      {l.qty ? (
                        <span className="text-ink-muted">{l.qty}</span>
                      ) : null}
                      {l.unit && (
                        <span className="text-ink-muted">{l.unit}</span>
                      )}
                      {l.price ? (
                        <span className="text-ink-muted">{l.price}</span>
                      ) : null}
                    </div>
                    {l.problem && (
                      <p className="mt-0.5 text-xs text-danger">
                        {t.posImport.row(l.row)}: {l.problem}
                      </p>
                    )}
                    {/* ⚠️ Said on the row, because it changes how the price
                        beside it must be read — and getting that wrong makes
                        the ingredient a thousand times too cheap. */}
                    {l.smallUnit && !l.problem && (
                      <p className="mt-0.5 text-xs text-amber-700 dark:text-amber-400">
                        {t.posImport.smallUnit}
                      </p>
                    )}
                    {l.exists && !l.problem && (
                      <p className="mt-0.5 text-xs text-ink-muted">
                        {t.posImport.exists}
                      </p>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      <div className="flex justify-end gap-3">
        <button type="button" className="btn-ghost px-4 py-2" onClick={onClose}>
          {t.common.cancel}
        </button>
        {!lines ? (
          <button
            type="button"
            className="btn-primary px-4 py-2 disabled:opacity-60"
            disabled={busy || !file}
            onClick={() => read()}
          >
            {busy ? t.posImport.reading : t.posImport.read}
          </button>
        ) : (
          <button
            type="button"
            className="btn-primary px-4 py-2 disabled:opacity-60"
            disabled={busy || picked.size === 0}
            onClick={apply}
          >
            {busy ? t.common.saving : t.posImport.apply(picked.size)}
          </button>
        )}
      </div>
    </div>
  );
}
