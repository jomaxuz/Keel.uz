"use client";

// QR cards: one for the restaurant, or one per table.
//
// A table's card links to the site with `?table=<id>`, which puts the guest "at
// that table" for the rest of the visit — the menu reads normally and checkout
// offers to bring the food to the table instead of asking for an address. The
// restaurant-wide card is the same site without a table, for a window or a
// business card.
//
// Cards are drawn on a canvas (see QrPoster) so what the admin sees is exactly
// what downloads, at print resolution.

import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import QrPoster, {
  drawPoster,
  POSTER_STYLES,
  type PosterContent,
} from "@/components/admin/QrPoster";
import { dataUrlToBlob, dataUrlToBytes, saveBlob, zipFiles } from "@/lib/zip";
import type { BookingSettings, FloorTable, Restaurant } from "@/lib/types";

type Target = { key: string; table: FloorTable | null };

export default function AdminQrPage() {
  const t = useAdminT();
  const scope = useAdminScope();
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);
  const [styleKey, setStyleKey] = useState<keyof typeof POSTER_STYLES>("cream");
  const [title, setTitle] = useState("");
  const [subtitle, setSubtitle] = useState("");
  const [description, setDescription] = useState("");
  const [footer, setFooter] = useState("");
  const [selected, setSelected] = useState<string>("all");
  const [busy, setBusy] = useState(false);
  // The site's own address. Editable: the panel may be opened over an IP or a
  // tunnel while the printed code has to point at the real domain.
  const [origin, setOrigin] = useState("");
  const previewRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    setOrigin(window.location.origin);
    api
      // The plan and the phone number are the selected branch's: table 7 in
      // Samarqand is not table 7 in Toshkent.
      .getRestaurant({ branchId: scope.branch?.id, brand: scope.brand?.id })
      .then((r) => {
        setRestaurant(r.restaurant);
        setTitle((v) => v || r.restaurant.name);
        setSubtitle((v) => v || t.qr.defaultSubtitle);
        setDescription((v) => v || t.qr.defaultDescription);
        setFooter((v) => v || (r.restaurant.phones?.[0] ?? ""));
      })
      .catch(() => setRestaurant(null));
    // Defaults are seeded once; re-running on a language switch would undo the
    // admin's own wording.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scope.branch?.id, scope.brand?.id]);

  const booking: BookingSettings | null = restaurant?.booking ?? null;
  const tables = useMemo(
    () => (booking?.tables ?? []).filter((tb) => tb.isActive),
    [booking],
  );

  const style = POSTER_STYLES[styleKey];

  // The printed link carries the brand and the branch as well as the table.
  // A card is glued to a table for years: it has to keep pointing at the right
  // menu and the right kitchen even after the company adds a second brand.
  const urlFor = (table: FloorTable | null) => {
    const qs = new URLSearchParams();
    if (table) qs.set("table", table.id);
    if (scope.branch?.id) qs.set("branch", scope.branch.id);
    if (scope.brand?.slug || scope.brand?.id) {
      qs.set("brand", scope.brand.slug || scope.brand.id);
    }
    const suffix = qs.toString() ? `?${qs}` : "";
    return `${origin}/menu${suffix}`;
  };

  const contentFor = (table: FloorTable | null): PosterContent => ({
    title,
    subtitle,
    description,
    badge: table ? table.number : "",
    footer,
  });

  const current: Target = useMemo(() => {
    if (selected === "all") return { key: "all", table: null };
    const table = tables.find((tb) => tb.id === selected) ?? null;
    return { key: selected, table };
  }, [selected, tables]);

  function pngBytes(canvas: HTMLCanvasElement): Uint8Array {
    return dataUrlToBytes(canvas.toDataURL("image/png"));
  }

  function downloadCurrent() {
    if (!previewRef.current) return;
    const name = current.table
      ? `qr-stol-${current.table.number}.png`
      : "qr-restoran.png";
    saveBlob(dataUrlToBlob(previewRef.current.toDataURL("image/png")), name);
  }

  /**
   * Every table at once, as a single archive. Firing one download per table
   * makes the browser ask whether it may download multiple files — and drops
   * the rest if the operator hesitates. One file asks nothing.
   */
  async function downloadAll() {
    if (tables.length === 0) return;
    setBusy(true);
    try {
      const canvas = document.createElement("canvas");
      const files = tables.map((table) => {
        drawPoster(canvas, urlFor(table), contentFor(table), style);
        return { name: `qr-stol-${table.number}.png`, data: pngBytes(canvas) };
      });
      // The restaurant-wide card rides along: it is what goes in the window.
      drawPoster(canvas, urlFor(null), contentFor(null), style);
      files.push({ name: "qr-restoran.png", data: pngBytes(canvas) });
      saveBlob(zipFiles(files), "qr-kodlar.zip");
    } finally {
      setBusy(false);
    }
  }

  const inputCls =
    "mt-1 w-full rounded-xl border border-line-strong bg-surface px-3 py-2 text-sm outline-none focus:border-brand";

  return (
    <div>
      <h1 className="text-2xl font-bold">{t.qr.title}</h1>
      <p className="mt-1 max-w-2xl text-sm text-ink-muted">{t.qr.hint}</p>

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_420px]">
        {/* ---- what the card says ---- */}
        <section className="rounded-3xl border border-line bg-surface p-5 shadow-card">
          <h2 className="text-sm font-semibold">{t.qr.whichCode}</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => setSelected("all")}
              className={`rounded-full px-3.5 py-1.5 text-sm font-semibold transition-colors ${
                selected === "all"
                  ? "bg-brand text-white"
                  : "border border-line-strong text-ink-soft hover:border-brand"
              }`}
            >
              {t.qr.wholeRestaurant}
            </button>
            {tables.map((tb) => (
              <button
                key={tb.id}
                type="button"
                onClick={() => setSelected(tb.id)}
                className={`rounded-full px-3.5 py-1.5 text-sm font-semibold transition-colors ${
                  selected === tb.id
                    ? "bg-brand text-white"
                    : "border border-line-strong text-ink-soft hover:border-brand"
                }`}
              >
                {t.qr.tableLabel(tb.number)}
              </button>
            ))}
          </div>
          {tables.length === 0 && (
            <p className="mt-3 rounded-2xl border border-dashed border-line-strong p-4 text-center text-sm text-ink-muted/70">
              {t.qr.noTables}
            </p>
          )}

          <h2 className="mt-6 text-sm font-semibold">{t.qr.background}</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {(
              Object.keys(POSTER_STYLES) as (keyof typeof POSTER_STYLES)[]
            ).map((key) => (
              <button
                key={key}
                type="button"
                onClick={() => setStyleKey(key)}
                style={{ background: POSTER_STYLES[key].bg }}
                className={`h-10 w-16 rounded-xl border-2 transition-colors ${
                  styleKey === key ? "border-brand" : "border-line-strong"
                }`}
                aria-label={key}
              >
                <span
                  className="mx-auto block h-3 w-8 rounded-full"
                  style={{ background: POSTER_STYLES[key].accent }}
                />
              </button>
            ))}
          </div>

          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium">{t.qr.cardTitle}</span>
              <input
                className={inputCls}
                value={title}
                maxLength={40}
                onChange={(e) => setTitle(e.target.value)}
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.qr.cardSubtitle}</span>
              <input
                className={inputCls}
                value={subtitle}
                maxLength={60}
                onChange={(e) => setSubtitle(e.target.value)}
              />
            </label>
          </div>
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.qr.cardDescription}</span>
            <input
              className={inputCls}
              value={description}
              maxLength={140}
              onChange={(e) => setDescription(e.target.value)}
            />
          </label>
          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.qr.cardFooter}</span>
            <input
              className={inputCls}
              value={footer}
              maxLength={60}
              onChange={(e) => setFooter(e.target.value)}
            />
          </label>

          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.qr.siteUrl}</span>
            <input
              className={`${inputCls} font-mono text-xs`}
              value={origin}
              onChange={(e) => setOrigin(e.target.value.replace(/\/+$/, ""))}
            />
            <span className="mt-1 block text-xs text-ink-muted">
              {t.qr.siteUrlHint}
            </span>
          </label>

          <p className="mt-4 break-all rounded-xl bg-ink/[0.03] px-3 py-2 font-mono text-xs text-ink-muted">
            {urlFor(current.table)}
          </p>
        </section>

        {/* ---- the card itself ---- */}
        <aside className="h-fit rounded-3xl border border-line bg-surface p-5 shadow-card">
          <QrPoster
            canvasRef={previewRef}
            url={urlFor(current.table)}
            content={contentFor(current.table)}
            style={style}
          />
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={downloadCurrent}
              className="btn-primary px-4 py-2 text-sm"
            >
              {t.qr.download}
            </button>
            {tables.length > 0 && (
              <button
                type="button"
                disabled={busy}
                onClick={downloadAll}
                className="btn-ghost px-4 py-2 text-sm disabled:opacity-60"
              >
                {busy ? t.qr.downloading : t.qr.downloadAll(tables.length)}
              </button>
            )}
          </div>
          <p className="mt-3 text-xs text-ink-muted">{t.qr.printHint}</p>
        </aside>
      </div>
    </div>
  );
}
