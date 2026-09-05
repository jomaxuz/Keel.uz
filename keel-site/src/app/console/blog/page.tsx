"use client";

// Writing a post.
//
// ⚠️ **Three languages side by side, each written by hand.** A post shaped into
// Russian by a machine reads like a machine to the person deciding whether to
// buy a till from us, and this is the one page where that decides something. A
// language left empty is a language the post does not appear in — which is the
// honest outcome, and better than a card that opens onto somebody else's words.
//
// ⚠️ **Draft until somebody says otherwise.** A post is written over several
// sittings; one that went live on the first save would put a half-finished
// paragraph on the front of the company.

import { useCallback, useEffect, useRef, useState } from "react";

import {
  blogDelete,
  blogList,
  blogSave,
  blogUpload,
  type BlogPost,
  type BlogText,
} from "@/lib/api";
import { useT } from "@/lib/i18n/client";

const LANGS = ["uz", "ru", "en"] as const;
type L = (typeof LANGS)[number];

const emptyText = (): BlogText => ({ title: "", excerpt: "", body: "" });
const emptyPost = (): BlogPost => ({
  slug: "",
  uz: emptyText(),
  ru: emptyText(),
  en: emptyText(),
  published: false,
  views: 0,
});

export default function ConsoleBlog() {
  const { t } = useT();
  const [posts, setPosts] = useState<BlogPost[]>([]);
  const [draft, setDraft] = useState<BlogPost | null>(null);
  const [lang, setLang] = useState<L>("uz");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const file = useRef<HTMLInputElement>(null);
  const coverFile = useRef<HTMLInputElement>(null);
  const bodyRef = useRef<HTMLTextAreaElement>(null);

  const load = useCallback(() => {
    blogList()
      .then((d) => setPosts(d.posts))
      .catch(() => setError(t.dash.loadFailed));
  }, [t.dash.loadFailed]);

  useEffect(load, [load]);

  async function save() {
    if (!draft) return;
    setBusy(true);
    setError("");
    try {
      await blogSave(draft);
      setDraft(null);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : t.dash.loadFailed);
    } finally {
      setBusy(false);
    }
  }

  /** The picture at the top of the post and on its card. */
  async function uploadCover(f: File) {
    if (!draft) return;
    setBusy(true);
    setError("");
    try {
      const { url } = await blogUpload(f);
      setDraft({ ...draft, cover: url });
    } catch (e) {
      setError(e instanceof Error ? e.message : t.dash.loadFailed);
    } finally {
      setBusy(false);
      // ⚠️ Cleared, or choosing the same file twice fires no change event and
      // the button looks broken to the person who just used it.
      if (coverFile.current) coverFile.current.value = "";
    }
  }

  /** Put an uploaded picture where the cursor is.
   *
   *  ⚠️ **At the cursor, not at the end.** A writer uploads a picture while
   *  looking at the paragraph it belongs under; appending it to the bottom
   *  means finding it again and moving it, every time. */
  async function upload(f: File) {
    setBusy(true);
    setError("");
    try {
      const { url } = await blogUpload(f);
      const el = bodyRef.current;
      const md = `\n![](${url})\n`;
      if (draft && el) {
        const at = el.selectionStart ?? el.value.length;
        const body = el.value.slice(0, at) + md + el.value.slice(at);
        setDraft({ ...draft, [lang]: { ...draft[lang], body } });
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : t.dash.loadFailed);
    } finally {
      setBusy(false);
      if (file.current) file.current.value = "";
    }
  }

  const text = draft?.[lang] ?? emptyText();
  const setText = (patch: Partial<BlogText>) =>
    draft && setDraft({ ...draft, [lang]: { ...draft[lang], ...patch } });

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="h-display text-2xl">{t.blogAdmin.title}</h1>
        <button
          type="button"
          className="rounded-lg bg-signal-500 px-4 py-2 text-sm font-semibold text-white"
          onClick={() => {
            setDraft(emptyPost());
            setLang("uz");
          }}
        >
          {t.blogAdmin.add}
        </button>
      </div>

      {error && <p className="mt-4 text-sm text-rose-600">{error}</p>}

      {draft && (
        <div className="mt-6 rounded-3xl border border-line bg-surface p-5">
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium">{t.blogAdmin.slug}</span>
              <input
                className="mt-1 w-full rounded-lg border border-line bg-surface px-3 py-2"
                value={draft.slug}
                placeholder={t.blogAdmin.slugPh}
                onChange={(e) => setDraft({ ...draft, slug: e.target.value })}
              />
              <span className="mt-1 block text-xs text-ink-muted">
                {t.blogAdmin.slugHint}
              </span>
            </label>
            {/* ⚠️ **Chosen from the computer, not typed.** A cover was an
                address box, and the only address that works is one this editor
                produced a minute earlier — so the field asked the writer to go
                and find it, and the honest answer to "where do I get that" was
                "upload it somewhere else first". */}
            <div className="block text-sm">
              <span className="font-medium">{t.blogAdmin.cover}</span>
              <div className="mt-1 flex items-center gap-3">
                {draft.cover ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={draft.cover}
                    alt=""
                    className="h-16 w-28 rounded-lg border border-line object-cover"
                  />
                ) : (
                  <div className="grid h-16 w-28 place-items-center rounded-lg border border-dashed border-line text-xs text-ink-muted">
                    {t.blogAdmin.noCover}
                  </div>
                )}
                <input
                  ref={coverFile}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={(e) => {
                    const f = e.target.files?.[0];
                    if (f) void uploadCover(f);
                  }}
                />
                <button
                  type="button"
                  className="rounded-lg border border-line px-3 py-2 text-sm font-semibold"
                  disabled={busy}
                  onClick={() => coverFile.current?.click()}
                >
                  {draft.cover ? t.blogAdmin.coverChange : t.blogAdmin.coverPick}
                </button>
                {draft.cover && (
                  <button
                    type="button"
                    className="text-sm text-ink-muted underline"
                    onClick={() => setDraft({ ...draft, cover: "" })}
                  >
                    {t.blogAdmin.coverClear}
                  </button>
                )}
              </div>
            </div>
          </div>

          {/* ⚠️ **A tab per language, and each one is written.** Nothing here
              copies from another: a translated-by-machine paragraph is worse
              than no page in that language at all. */}
          <div className="mt-5 flex gap-2">
            {LANGS.map((l) => (
              <button
                key={l}
                type="button"
                onClick={() => setLang(l)}
                className={`rounded-lg border px-3 py-1.5 text-sm font-semibold uppercase ${
                  l === lang
                    ? "border-signal-500 bg-signal-500/10"
                    : "border-line text-ink-muted"
                } ${draft[l].title ? "" : "opacity-60"}`}
              >
                {l}
              </button>
            ))}
          </div>

          <div className="mt-4 space-y-3">
            <input
              className="w-full rounded-lg border border-line bg-surface px-3 py-2 text-lg font-semibold"
              placeholder={t.blogAdmin.postTitle}
              value={text.title}
              onChange={(e) => setText({ title: e.target.value })}
            />
            <textarea
              className="w-full rounded-lg border border-line bg-surface px-3 py-2 text-sm"
              rows={2}
              placeholder={t.blogAdmin.excerpt}
              value={text.excerpt}
              onChange={(e) => setText({ excerpt: e.target.value })}
            />
            <textarea
              ref={bodyRef}
              className="w-full rounded-lg border border-line bg-surface px-3 py-2 font-mono text-sm"
              rows={18}
              placeholder={t.blogAdmin.body}
              value={text.body}
              onChange={(e) => setText({ body: e.target.value })}
            />
            <p className="text-xs text-ink-muted">{t.blogAdmin.bodyHint}</p>
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <input
              ref={file}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) void upload(f);
              }}
            />
            <button
              type="button"
              className="rounded-lg border border-line px-3 py-2 text-sm font-semibold"
              disabled={busy}
              onClick={() => file.current?.click()}
            >
              {t.blogAdmin.uploadImage}
            </button>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={draft.published}
                onChange={(e) =>
                  setDraft({ ...draft, published: e.target.checked })
                }
              />
              {t.blogAdmin.publish}
            </label>
            <div className="ml-auto flex gap-2">
              <button
                type="button"
                className="rounded-lg border border-line px-4 py-2 text-sm"
                onClick={() => setDraft(null)}
              >
                {t.blogAdmin.cancel}
              </button>
              <button
                type="button"
                className="rounded-lg bg-signal-500 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
                disabled={busy}
                onClick={() => void save()}
              >
                {t.blogAdmin.save}
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="mt-8 divide-y divide-line rounded-3xl border border-line bg-surface">
        {posts.length === 0 && (
          <p className="p-5 text-sm text-ink-muted">{t.blogAdmin.empty}</p>
        )}
        {posts.map((p) => (
          <div key={p.id} className="flex flex-wrap items-center gap-3 p-4">
            <div className="min-w-0 flex-1">
              <p className="truncate font-semibold">{p.uz.title || p.slug}</p>
              <p className="text-xs text-ink-muted">
                /{p.slug} ·{" "}
                {p.published ? t.blogAdmin.live : t.blogAdmin.draft} ·{" "}
                {t.blogAdmin.views(p.views)} ·{" "}
                {LANGS.filter((l) => p[l].title).join(" ").toUpperCase()}
              </p>
            </div>
            <button
              type="button"
              className="rounded-lg border border-line px-3 py-1.5 text-sm"
              onClick={() => {
                setDraft(p);
                setLang("uz");
              }}
            >
              {t.blogAdmin.edit}
            </button>
            <button
              type="button"
              className="rounded-lg border border-line px-3 py-1.5 text-sm text-rose-600"
              onClick={async () => {
                if (!p.id) return;
                await blogDelete(p.id);
                load();
              }}
            >
              {t.blogAdmin.delete}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
