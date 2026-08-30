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
  const [done, setDone] = useState<{
    created: number;
    skipped: number;
    active: boolean;
  } | null>(null);

  async function read() {
    setBusy(true);
    setError("");
    try {
      const res = await api.menuImportPreview(url.trim());
      setDishes(res.dishes);
      setGuessed(res.guessed);
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
    }
  }

  async function apply() {
    if (!dishes) return;
    setBusy(true);
    setError("");
    try {
      const chosen = dishes.filter((_, i) => picked.has(i));
      const res = await api.menuImportApply(chosen, withImages, active);
      setDone({
        created: res.created,
        skipped: res.skipped,
        active: res.active,
      });
      onDone();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
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
