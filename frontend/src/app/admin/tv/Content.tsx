"use client";

// The playlist those televisions loop.
//
// ⚠️ **One list per branch, and that is the whole shape of this screen.** Two
// televisions in one room show the same restaurant's food; what differs between
// them is whether the order board is on, and that is set on the screen itself
// next door. A playlist per screen would mean uploading the same video four
// times and remembering to change it four times — and the copy nobody remembers
// is the one still showing last month's promotion.
//
// ⚠️ **Reordered with two buttons, not by dragging.** The list is short, the
// person editing it is often on a laptop trackpad in a back office, and a drag
// that drops in the wrong place on a touch screen is a change nobody notices
// they made. Up and down say what they did.

import { useCallback, useEffect, useRef, useState } from "react";

import { ApiError, api, imageUrl, uploadImage, uploadTVVideo } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useAsk } from "@/components/ui/Ask";
import type { TVSlide } from "@/lib/types";

/** What the wall is doing with this row right now.
 *
 *  ⚠️ **Drawn, because every one of these is invisible otherwise.** A slide
 *  whose window closed on Saturday looks exactly like one that is playing —
 *  same row, same picture, same everything — and the question this page exists
 *  to answer is "what is on the screen at the moment". The same rule the server
 *  and the television apply (models.TVSlidePlayable); it is stated three times
 *  because three places need it, and the wording of the three has to match. */
function slideStatus(s: TVSlide): "on" | "off" | "early" | "late" {
  if (!s.active) return "off";
  const now = Date.now();
  if (s.startsAt && now < new Date(s.startsAt).getTime()) return "early";
  if (s.endsAt && now > new Date(s.endsAt).getTime()) return "late";
  return "on";
}

export default function TVContent({ branchId }: { branchId: string }) {
  const t = useAdminT();
  const { ask } = useAsk();

  const [slides, setSlides] = useState<TVSlide[]>([]);
  const [limit, setLimit] = useState(0);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const imageInput = useRef<HTMLInputElement>(null);
  const videoInput = useRef<HTMLInputElement>(null);

  const load = useCallback(() => {
    if (!branchId) return;
    setLoading(true);
    api
      .tvSlides(branchId)
      .then((d) => {
        setSlides(d.slides);
        setLimit(d.limit);
        setError("");
      })
      .catch((e) =>
        setError(e instanceof ApiError ? e.message : t.common.loadFailed),
      )
      .finally(() => setLoading(false));
  }, [branchId, t]);

  useEffect(load, [load]);

  async function add(file: File, kind: "image" | "video") {
    setUploading(true);
    setError("");
    setNotice("");
    try {
      const url =
        kind === "video" ? await uploadTVVideo(file) : await uploadImage(file);
      // ⚠️ Named from the file, not left blank. Six rows called "" is a list in
      // which nobody can find the one they came to switch off — the same
      // argument as the screen names next door.
      await api.createTVSlide(branchId, {
        kind,
        url,
        name: file.name.replace(/\.[^.]+$/, "").slice(0, 80),
      });
      setNotice(t.tv.slideAdded);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setUploading(false);
    }
  }

  async function save(
    slide: TVSlide,
    body: Parameters<typeof api.updateTVSlide>[2],
  ) {
    setBusy(true);
    setError("");
    try {
      await api.updateTVSlide(branchId, slide.id, body);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy(false);
    }
  }

  async function remove(slide: TVSlide) {
    if (
      !(await ask({
        title: t.tv.slideRemoveConfirm(slide.name || t.tv.kindImage),
        danger: true,
      }))
    )
      return;
    setBusy(true);
    try {
      await api.removeTVSlide(branchId, slide.id);
      setNotice(t.tv.slideRemoved);
      load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.deleteFailed);
    } finally {
      setBusy(false);
    }
  }

  /** Move one row, and send the whole order back.
   *
   *  ⚠️ **The list is reordered on screen first and the server is told after.**
   *  A button that does nothing until a round trip completes gets pressed
   *  again, and the second press is applied to an order the person cannot see
   *  yet. */
  async function move(index: number, delta: number) {
    const next = [...slides];
    const to = index + delta;
    if (to < 0 || to >= next.length) return;
    [next[index], next[to]] = [next[to], next[index]];
    setSlides(next);
    setBusy(true);
    try {
      await api.reorderTVSlides(
        branchId,
        next.map((s) => s.id),
      );
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
      load();
    } finally {
      setBusy(false);
    }
  }

  const fieldCls =
    "rounded-xl border border-line-strong bg-surface px-2 py-1.5 text-xs outline-none focus:border-brand";

  return (
    <div className="space-y-5">
      <section className="rounded-3xl border border-line bg-surface p-5 shadow-card">
        <h2 className="font-display text-lg font-bold">{t.tv.tabContent}</h2>
        <p className="mt-1 max-w-2xl text-sm text-ink-muted">
          {t.tv.contentIntro}
        </p>

        <div className="mt-4 flex flex-wrap items-center gap-3">
          <input
            ref={imageInput}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              e.target.value = "";
              if (f) void add(f, "image");
            }}
          />
          <input
            ref={videoInput}
            type="file"
            accept="video/mp4,video/webm"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              e.target.value = "";
              if (f) void add(f, "video");
            }}
          />
          <button
            className="btn btn-primary"
            disabled={uploading}
            onClick={() => imageInput.current?.click()}
          >
            {uploading ? t.tv.uploading : t.tv.addImage}
          </button>
          <button
            className="btn-ghost border border-line-strong px-4 py-2 text-sm"
            disabled={uploading}
            onClick={() => videoInput.current?.click()}
          >
            {t.tv.addVideo}
          </button>
          <span className="text-xs text-ink-muted">{t.tv.videoHint}</span>
        </div>
        {/* ⚠️ Said out loud, because it is the answer to the question an owner
            asks next — "and if the internet drops?" — and to the one they do
            not ask, which is why a hundred-megabyte cap is reasonable. */}
        <p className="mt-2 text-xs text-ink-muted">{t.tv.videoNote}</p>
      </section>

      {error && <p className="text-sm text-danger">{error}</p>}
      {notice && <p className="text-sm text-success">{notice}</p>}

      <section>
        <h3 className="font-display text-lg font-bold">
          {t.tv.tabContent}{" "}
          <span className="text-sm font-normal text-ink-muted">
            {t.tv.contentCount(slides.length, limit)}
          </span>
        </h3>

        {loading ? (
          <p className="mt-3 text-sm text-ink-muted">{t.common.loading}</p>
        ) : slides.length === 0 ? (
          // ⚠️ Said, not left blank: an empty list and a list that failed to
          // load look identical otherwise.
          <p className="mt-3 rounded-2xl border border-line bg-surface p-6 text-sm text-ink-muted">
            {t.tv.contentEmpty}
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {slides.map((s, i) => {
              const status = slideStatus(s);
              return (
                <li
                  key={s.id}
                  className="flex flex-wrap items-start gap-3 rounded-2xl border border-line bg-surface p-3"
                >
                  {/* The preview. ⚠️ A width is asked for on pictures and never
                      on videos: `?w=` makes the backend try to resize, and a
                      file it cannot decode is served whole — which for a
                      hundred-megabyte clip is a hundred megabytes, to draw a
                      thumbnail. */}
                  <div className="h-16 w-28 shrink-0 overflow-hidden rounded-xl bg-ink/5">
                    {s.kind === "video" ? (
                      <video
                        className="h-full w-full object-cover"
                        src={imageUrl(s.url) ?? undefined}
                        preload="metadata"
                        muted
                        playsInline
                      />
                    ) : (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        className="h-full w-full object-cover"
                        src={imageUrl(s.url, 300) ?? undefined}
                        alt=""
                      />
                    )}
                  </div>

                  <div className="min-w-48 flex-1 space-y-2">
                    <input
                      className={`${fieldCls} w-full text-sm`}
                      defaultValue={s.name}
                      placeholder={t.tv.itemNamePlaceholder}
                      disabled={busy}
                      onBlur={(e) => {
                        const next = e.target.value.trim();
                        if (next !== s.name) void save(s, { name: next });
                      }}
                    />
                    <div className="flex flex-wrap items-center gap-2 text-xs text-ink-muted">
                      <span className="rounded-lg bg-ink/5 px-2 py-0.5">
                        {s.kind === "video" ? t.tv.kindVideo : t.tv.kindImage}
                      </span>
                      <span
                        className={
                          status === "on"
                            ? "text-success"
                            : status === "off"
                              ? "text-ink-muted"
                              : "text-danger"
                        }
                      >
                        {status === "on"
                          ? t.tv.onAir
                          : status === "off"
                            ? t.tv.offAir
                            : status === "early"
                              ? t.tv.notYet
                              : t.tv.expired}
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-wrap items-center gap-2">
                    {/* Only for pictures: a video is as long as it is, and a
                        clip cut off at ten seconds because somebody left the
                        default alone is not a setting worth offering. */}
                    {s.kind === "image" && (
                      <label className="text-xs text-ink-muted">
                        <span className="mr-1">{t.tv.secondsLabel}</span>
                        <input
                          type="number"
                          min={3}
                          max={120}
                          className={`${fieldCls} w-16`}
                          defaultValue={s.seconds}
                          disabled={busy}
                          onBlur={(e) => {
                            const n = Number(e.target.value);
                            if (n && n !== s.seconds)
                              void save(s, { seconds: n });
                          }}
                        />
                      </label>
                    )}
                    <label className="text-xs text-ink-muted">
                      <span className="mr-1">{t.tv.fromDate}</span>
                      <input
                        type="date"
                        className={fieldCls}
                        defaultValue={s.startsOn ?? ""}
                        disabled={busy}
                        onChange={(e) =>
                          void save(s, { startsOn: e.target.value })
                        }
                      />
                    </label>
                    <label className="text-xs text-ink-muted">
                      <span className="mr-1">{t.tv.toDate}</span>
                      <input
                        type="date"
                        className={fieldCls}
                        defaultValue={s.endsOn ?? ""}
                        disabled={busy}
                        onChange={(e) =>
                          void save(s, { endsOn: e.target.value })
                        }
                      />
                    </label>
                  </div>

                  <div className="flex items-center gap-1">
                    <button
                      className="btn-ghost px-2 text-sm"
                      disabled={busy || i === 0}
                      title={t.tv.moveUp}
                      onClick={() => void move(i, -1)}
                    >
                      ↑
                    </button>
                    <button
                      className="btn-ghost px-2 text-sm"
                      disabled={busy || i === slides.length - 1}
                      title={t.tv.moveDown}
                      onClick={() => void move(i, 1)}
                    >
                      ↓
                    </button>
                    <button
                      className="btn-ghost px-2 text-sm"
                      disabled={busy}
                      onClick={() => void save(s, { active: !s.active })}
                    >
                      {s.active ? t.tv.turnOff : t.tv.turnOn}
                    </button>
                    <button
                      className="btn-ghost px-2 text-sm text-danger"
                      disabled={busy}
                      onClick={() => void remove(s)}
                    >
                      {t.common.delete}
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
        <p className="mt-2 text-xs text-ink-muted">
          {t.tv.dateHint} {t.tv.secondsHint}
        </p>
      </section>
    </div>
  );
}
