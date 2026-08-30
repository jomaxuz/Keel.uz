"use client";

// Building a menu from a link instead of typing it in.
//
// ⚠️ **Typing a menu in is the longest job in setting a restaurant up** — a
// hundred dishes, each with a name, a price, a description and a photograph —
// and it is why a signed customer takes three weeks to go live. Almost all of
// it already exists on their old site or on an aggregator's page.
//
// ⚠️ **Two steps, never one.** The page is read and proposed; nothing reaches
// the menu until the owner has looked at the list and pressed apply. An import
// that wrote a hundred and twenty dishes on one press would be a mistake nobody
// undoes by hand — and mistakes are certain, because the input is somebody
// else's page.
//
// ⚠️ **Imported dishes arrive hidden.** They are switched off until the owner
// has been through them: a price read off another restaurant's page and put
// straight in front of guests is a dish sold at whatever that page said.

import { useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { formatPrice } from "@/lib/format";
import type { ImportedDish } from "@/lib/types";

export default function MenuImport({
  onDone,
  onClose,
}: {
  onDone: () => void;
  onClose: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  // ⚠️ UZS rather than the restaurant's setting, and it is not a shortcut: this
  // line is a readability aid beside a box the owner is about to edit, and
  // fetching the profile to draw it would be a request on a page that does not
  // otherwise need one. Every price in this product is whole so'm (CLAUDE.md).

  const [url, setUrl] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [dishes, setDishes] = useState<ImportedDish[] | null>(null);
  const [picked, setPicked] = useState<Set<number>>(new Set());
  const [withImages, setWithImages] = useState(true);
  // ⚠️ **Off by default, and it stays off by default.** These prices came off
  // somebody else's page; switching ninety dishes on unread sells them at
  // whatever that page happened to say, and the argument is at the till with a
  // cashier who has never seen the number. An owner importing their own menu
  // presses this once.
  const [active, setActive] = useState(false);
  const [guessed, setGuessed] = useState(false);
  const [readerLabel, setReaderLabel] = useState("");
  const [progress, setProgress] = useState<{
    done: number;
    total: number;
    percent: number;
  } | null>(null);
  const [done, setDone] = useState<{
    created: number;
    skipped: number;
    active: boolean;
  } | null>(null);

  /** Wait for a job, moving the bar as the server reports it.
   *
   * ⚠️ **Shared by both halves of the import** — reading a page and writing the
   * dishes — because the waiting is identical and two loops would drift.
   * ⚠️ **The bar is the server's own count, never an animation.** A bar that
   * fills at a fixed rate says "nearly done" while ninety photographs are still
   * downloading, and the owner closes the tab at 95%. */
  async function follow(jobId: string, fallbackTotal: number) {
    for (;;) {
      await new Promise((r) => setTimeout(r, 900));
      const j = await api.importJob(jobId);
      if (!j.finished) {
        setProgress({
          done: j.done ?? 0,
          total: j.total ?? fallbackTotal,
          percent: j.percent ?? 0,
        });
        continue;
      }
      if (j.error) throw new ApiError(0, j.error);
      // ⚠️ No result and no error means the job aged out or the container
      // restarted. Reported as "we lost track", not as a failure: the work may
      // well have finished, and "it failed" sends the owner to press the button
      // again and import the whole menu twice.
      return j.result ?? null;
    }
  }

  async function read() {
    setBusy(true);
    setError("");
    setProgress({ done: 0, total: 4, percent: 0 });
    try {
      const { jobId } = await api.menuImportPreview(url.trim());
      const res = await follow(jobId, 4);
      if (!res?.dishes) {
        setError(t.menuImport.lostTrack);
        return;
      }
      setDishes(res.dishes);
      setGuessed(!!res.guessed);
      setReaderLabel(res.readerLabel ?? "");
      // ⚠️ Everything ticked except what the menu already has. The common case
      // is "import all of it", and a list of ninety unticked rows is ninety
      // clicks before the button does anything.
      setPicked(
        new Set(
          res.dishes.map((d, i) => (d.exists ? -1 : i)).filter((i) => i >= 0),
        ),
      );
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.loadFailed);
    } finally {
      setBusy(false);
      setProgress(null);
    }
  }

  /** Start the import and follow it.
   *
   * ⚠️ **The bar is the server's own count, not an animation.** A fake bar that
   * fills at a fixed rate is worse than a spinner: it says "eleven seconds
   * left" while ninety photographs are still downloading, and the owner closes
   * the tab at 95%.
   *
   * ⚠️ **Closing the tab does not stop it either.** The work runs on the
   * server; this loop only watches. That is the whole reason the button no
   * longer returns a gateway error over a menu that was importing fine. */
  async function apply() {
    if (!dishes) return;
    setBusy(true);
    setError("");
    setProgress({ done: 0, total: picked.size, percent: 0 });
    try {
      const chosen = dishes.filter((_, i) => picked.has(i));
      const { jobId } = await api.menuImportApply(chosen, withImages, active);
      const res = await follow(jobId, chosen.length);
      if (!res || res.created === undefined) {
        setError(t.menuImport.lostTrack);
        return;
      }
      setDone({
        created: res.created,
        skipped: res.skipped ?? 0,
        active: !!res.active,
      });
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
      setProgress(null);
    }
  }

  function toggle(i: number) {
    const next = new Set(picked);
    if (next.has(i)) next.delete(i);
    else next.add(i);
    setPicked(next);
  }

  function edit(i: number, patch: Partial<ImportedDish>) {
    if (!dishes) return;
    setDishes(dishes.map((d, di) => (di === i ? { ...d, ...patch } : d)));
  }

  if (done) {
    return (
      <div className="space-y-4">
        <p className="text-sm">
          {t.menuImport.created(done.created)}
          {done.skipped > 0 && ` · ${t.menuImport.skipped(done.skipped)}`}
        </p>
        {/* ⚠️ Said here rather than left to be discovered, and it says which
            of the two things happened. Dishes on the menu and invisible to
            guests is deliberate and is exactly the kind of thing an owner
            reports as "the import did not work"; dishes that went live
            unchecked is the thing they need to know before a guest orders. */}
        <p className="rounded-xl border border-line bg-page px-3 py-2 text-sm text-ink-soft">
          {done.active ? t.menuImport.liveNote : t.menuImport.hiddenNote}
        </p>
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

  return (
    <div className="space-y-4">
      {!dishes && (
        <>
          <p className="text-sm text-ink-soft">{t.menuImport.lead}</p>
          <label className="block text-sm">
            <span className="font-medium">{t.menuImport.urlLabel}</span>
            <input
              className="input mt-1"
              placeholder="https://express24.uz/..."
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && url.trim() && !busy) void read();
              }}
            />
            <span className="mt-1 block text-xs text-ink-muted">
              {t.menuImport.urlHint}
            </span>
          </label>
        </>
      )}

      {error && <p className="text-sm text-danger">{error}</p>}

      {dishes && (
        <>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm font-medium">
              {t.menuImport.found(dishes.length, picked.size)}
              {/* ⚠️ Which of the four readers understood the page. It tells the
                  owner how much checking the list deserves, and when nothing
                  worked it is the difference between "broken" and "this page
                  publishes nothing — use the file import". */}
              {readerLabel && (
                <span className="ml-2 font-normal text-ink-muted">
                  · {readerLabel}
                </span>
              )}
            </p>
            <div className="flex gap-3 text-sm">
              <button
                type="button"
                className="text-brand hover:underline"
                onClick={() => setPicked(new Set(dishes.map((_, i) => i)))}
              >
                {t.menuImport.pickAll}
              </button>
              <button
                type="button"
                className="text-ink-muted hover:underline"
                onClick={() => setPicked(new Set())}
              >
                {t.menuImport.pickNone}
              </button>
            </div>
          </div>

          {/* ⚠️ Said out loud when the page published nothing structured and the
              assistant had to read it. The list is worth the same amount of
              checking either way, but an owner who knows a machine read a
              picture of a menu checks the prices; one who thinks it came from a
              database does not. */}
          {guessed && (
            <p className="rounded-xl border border-amber-400/50 bg-amber-400/10 px-3 py-2 text-sm">
              {t.menuImport.guessedNote}
            </p>
          )}

          <div className="max-h-[52vh] space-y-2 overflow-y-auto pr-1">
            {dishes.map((d, i) => (
              <div
                key={i}
                className={`rounded-xl border p-2.5 ${
                  picked.has(i)
                    ? "border-brand/50 bg-brand/[0.03]"
                    : "border-line"
                }`}
              >
                <div className="flex items-start gap-3">
                  <input
                    type="checkbox"
                    className="mt-2"
                    checked={picked.has(i)}
                    onChange={() => toggle(i)}
                  />
                  {d.imageUrl && (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={d.imageUrl}
                      alt=""
                      className="h-14 w-14 shrink-0 rounded-lg object-cover"
                      // A photograph that will not load here will not download
                      // on apply either, and an empty frame says so quietly.
                      onError={(e) => {
                        e.currentTarget.style.display = "none";
                      }}
                    />
                  )}
                  <div className="grid min-w-0 flex-1 gap-2 sm:grid-cols-[1fr_9rem_9rem]">
                    <input
                      className="input"
                      value={d.name}
                      onChange={(e) => edit(i, { name: e.target.value })}
                    />
                    <input
                      className="input"
                      inputMode="numeric"
                      placeholder={t.menuImport.noPrice}
                      value={d.price || ""}
                      onChange={(e) =>
                        edit(i, { price: Number(e.target.value) || 0 })
                      }
                    />
                    <input
                      className="input"
                      placeholder={t.menuImport.category}
                      value={d.category ?? ""}
                      onChange={(e) => edit(i, { category: e.target.value })}
                    />
                  </div>
                </div>
                <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 pl-7 text-xs text-ink-muted">
                  {d.price > 0 && (
                    <span>{formatPrice(d.price, "UZS", lang)}</span>
                  )}
                  {/* ⚠️ Named, because it is the row somebody would otherwise
                      import twice. */}
                  {d.exists && (
                    <span className="rounded bg-page px-1.5 py-0.5">
                      {t.menuImport.exists}
                    </span>
                  )}
                  {d.description && (
                    <span className="truncate">{d.description}</span>
                  )}
                </div>
              </div>
            ))}
          </div>

          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={withImages}
              onChange={(e) => setWithImages(e.target.checked)}
            />
            <span>
              {t.menuImport.withImages}
              <span className="mt-0.5 block text-xs text-ink-muted">
                {t.menuImport.withImagesHint}
              </span>
            </span>
          </label>

          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-1"
              checked={active}
              onChange={(e) => setActive(e.target.checked)}
            />
            <span>
              {t.menuImport.active}
              <span className="mt-0.5 block text-xs text-ink-muted">
                {active ? t.menuImport.activeOn : t.menuImport.activeOff}
              </span>
            </span>
          </label>
        </>
      )}

      {progress && (
        <div>
          <div className="flex items-baseline justify-between text-sm">
            <span>{t.menuImport.importing}</span>
            {/* ⚠️ The count beside the percentage. "43%" alone does not say
                whether that is 43 dishes or 430, and the owner is deciding
                whether to wait. */}
            <span className="tabular-nums text-ink-muted">
              {progress.done} / {progress.total} · {progress.percent}%
            </span>
          </div>
          <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-ink/10">
            <div
              className="h-full rounded-full bg-brand transition-[width] duration-300"
              style={{ width: `${progress.percent}%` }}
            />
          </div>
          <p className="mt-1 text-xs text-ink-muted">{t.menuImport.keepOpen}</p>
        </div>
      )}

      <div className="flex justify-end gap-3">
        <button type="button" className="btn-ghost px-4 py-2" onClick={onClose}>
          {t.common.cancel}
        </button>
        {!dishes ? (
          <button
            type="button"
            className="btn-primary px-4 py-2 disabled:opacity-60"
            disabled={busy || !url.trim()}
            onClick={read}
          >
            {busy ? t.menuImport.reading : t.menuImport.read}
          </button>
        ) : (
          <button
            type="button"
            className="btn-primary px-4 py-2 disabled:opacity-60"
            disabled={busy || picked.size === 0}
            onClick={apply}
          >
            {busy ? t.common.saving : t.menuImport.apply(picked.size)}
          </button>
        )}
      </div>
    </div>
  );
}
