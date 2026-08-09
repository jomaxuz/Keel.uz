"use client";

// The banner strip, edited by the restaurant.
//
// ⚠️ **In settings rather than as its own page**, because it is part of how the site looks
// rather than something that happens: an owner opens it a few times a year, and a top-level
// menu item competes for attention with the screens they open every hour.
//
// ⚠️ **The link is a choice, not a text field.** A banner is the most-clicked thing on the
// home page, and "any address" there is a way to send a restaurant's own guests somewhere
// else — pasted wrong by accident, or on purpose by whoever gets the panel password next.
// The server refuses anything outside its own pages either way; the select is so nobody
// types something that will be silently dropped.

import { useCallback, useEffect, useState } from "react";
import { api, imageUrl } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import ImageUpload from "@/components/admin/ImageUpload";
import type { Banner } from "@/lib/types";

const LINKS = ["", "/menu", "/bron", "/about", "/filiallar", "/ish"];

export default function BannersEditor() {
  const t = useAdminT();
  const [rows, setRows] = useState<Banner[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [draft, setDraft] = useState<{ imageUrl: string; link: string; title: string }>({
    imageUrl: "",
    link: "",
    title: "",
  });

  const load = useCallback(() => {
    api
      .adminBanners()
      .then(setRows)
      .catch((e) => setError(e instanceof Error ? e.message : ""));
  }, []);

  useEffect(load, [load]);

  async function add() {
    if (!draft.imageUrl) return;
    setBusy(true);
    setError("");
    try {
      await api.createBanner({
        imageUrl: draft.imageUrl,
        link: draft.link,
        title: { uz: draft.title, ru: "", en: "" },
        sortOrder: rows.length,
        isActive: true,
      });
      setDraft({ imageUrl: "", link: "", title: "" });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "");
    } finally {
      setBusy(false);
    }
  }

  async function patch(b: Banner, body: Partial<Banner>) {
    try {
      // ⚠️ The image is sent every time. An empty one means "keep it" on the server, and
      // sending the current value keeps the two ends agreeing about what is stored.
      await api.updateBanner(b.id, { imageUrl: b.imageUrl, link: b.link, title: b.title, sortOrder: b.sortOrder, ...body });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "");
    }
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-ink-muted">{t.banners.hint}</p>
      {error && <p className="text-sm text-brand">{error}</p>}

      <ul className="space-y-3">
        {rows.map((b, i) => (
          <li key={b.id} className="card flex flex-wrap items-center gap-4 p-3">
            <img
              src={imageUrl(b.imageUrl, 300) ?? ""}
              alt=""
              className="h-16 w-28 rounded-xl object-cover"
            />
            <div className="min-w-[180px] flex-1">
              <input
                className="input h-9 px-3 py-1 text-xs"
                placeholder={t.banners.caption}
                defaultValue={b.title?.uz ?? ""}
                onBlur={(e) =>
                  void patch(b, { title: { uz: e.target.value, ru: b.title?.ru ?? "", en: b.title?.en ?? "" } })
                }
              />
              <select
                className="select mt-1.5 h-9 py-1 text-xs"
                value={b.link ?? ""}
                onChange={(e) => void patch(b, { link: e.target.value })}
              >
                {LINKS.map((l) => (
                  <option key={l} value={l}>
                    {l || t.banners.noLink}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => void patch(b, { sortOrder: Math.max(0, b.sortOrder - 1) })}
                className="rounded-lg border border-line px-2 py-1 text-xs"
                aria-label="↑"
              >
                ↑
              </button>
              <button
                type="button"
                onClick={() => void patch(b, { sortOrder: b.sortOrder + 1 })}
                className="rounded-lg border border-line px-2 py-1 text-xs"
                aria-label="↓"
              >
                ↓
              </button>
              <button
                type="button"
                onClick={() => void patch(b, { isActive: !b.isActive })}
                className="rounded-lg border border-line px-2 py-1 text-xs text-ink-soft"
              >
                {b.isActive ? t.banners.hide : t.banners.show}
              </button>
              <button
                type="button"
                onClick={() => {
                  void api.deleteBanner(b.id).then(load);
                }}
                className="rounded-lg border border-line px-2 py-1 text-xs text-brand"
              >
                {t.common.delete}
              </button>
            </div>
            {!b.isActive && (
              <span className="text-xs text-ink-muted">{t.banners.hidden}</span>
            )}
            <span className="sr-only">{i}</span>
          </li>
        ))}
      </ul>

      <div className="card space-y-3 p-4">
        <p className="text-sm font-semibold text-ink">{t.banners.add}</p>
        <ImageUpload
          value={draft.imageUrl}
          onChange={(url) => setDraft({ ...draft, imageUrl: url })}
        />
        <input
          className="input"
          placeholder={t.banners.caption}
          value={draft.title}
          onChange={(e) => setDraft({ ...draft, title: e.target.value })}
        />
        <select
          className="select"
          value={draft.link}
          onChange={(e) => setDraft({ ...draft, link: e.target.value })}
        >
          {LINKS.map((l) => (
            <option key={l} value={l}>
              {l || t.banners.noLink}
            </option>
          ))}
        </select>
        <button
          type="button"
          onClick={() => void add()}
          disabled={busy || !draft.imageUrl}
          className="btn btn-dark disabled:opacity-40"
        >
          {t.common.add}
        </button>
      </div>
    </div>
  );
}
