"use client";

// What the restaurant knows about a customer, beyond what they ordered.
//
// This is the block whoever answers the phone reads before speaking: the note
// is for the things that go wrong otherwise ("allergic to nuts", "always asks
// for extra bread"), the tags are for sorting regulars the way this particular
// restaurant thinks about them.
//
// The name and the phone are deliberately not editable here. The phone was
// proved by SMS and the whole account hangs off it; changing it from the panel
// would quietly detach a customer from their own history.

import { useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import type { SiteUser } from "@/lib/types";

const SOURCES = ["site", "qr", "instagram", "referral", "phone"] as const;

export default function CustomerNotes({
  user,
  knownTags,
  onSaved,
}: {
  user: SiteUser;
  /** Tags already in use, offered so a typo cannot create a second "VIP ". */
  knownTags: string[];
  onSaved: (next: SiteUser) => void;
}) {
  const t = useAdminT();
  const [note, setNote] = useState(user.note ?? "");
  const [tags, setTags] = useState<string[]>(user.tags ?? []);
  const [source, setSource] = useState(user.source ?? "");
  // The input holds "MM-DD"; a date picker would demand a year the restaurant
  // has no use for.
  const [birthday, setBirthday] = useState(user.birthday ?? "");
  const [newTag, setNewTag] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  function toggleTag(x: string) {
    setTags((cur) =>
      cur.includes(x) ? cur.filter((c) => c !== x) : [...cur, x],
    );
  }

  async function save() {
    setError(null);
    setSaving(true);
    try {
      const next = await api.updateAdminUser(user.id, {
        note,
        tags,
        source,
        birthday,
      });
      onSaved(next);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  const suggestions = knownTags.filter((x) => !tags.includes(x));

  return (
    <section className="mt-6 rounded-3xl border border-line bg-surface p-6 shadow-card">
      <h2 className="text-lg font-bold">{t.users.notesTitle}</h2>
      <p className="mt-1 text-sm text-ink-muted">{t.users.notesHint}</p>

      <label className="mt-4 block text-sm">
        <span className="font-medium">{t.users.note}</span>
        <textarea
          className="input mt-1 w-full"
          rows={3}
          maxLength={500}
          placeholder={t.users.notePh}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      </label>

      <div className="mt-4">
        <span className="text-sm font-medium">{t.users.tags}</span>
        <div className="mt-2 flex flex-wrap gap-2">
          {tags.map((x) => (
            <button
              key={x}
              type="button"
              onClick={() => toggleTag(x)}
              className="rounded-full border border-brand bg-brand-tint px-3 py-1 text-xs font-semibold text-brand-dark"
            >
              {x} ✕
            </button>
          ))}
          {suggestions.map((x) => (
            <button
              key={x}
              type="button"
              onClick={() => toggleTag(x)}
              className="rounded-full border border-line-strong px-3 py-1 text-xs text-ink-muted hover:border-brand hover:text-brand"
            >
              + {x}
            </button>
          ))}
        </div>
        <div className="mt-2 flex gap-2">
          <input
            className="input flex-1 text-sm"
            placeholder={t.users.newTagPh}
            maxLength={30}
            value={newTag}
            onChange={(e) => setNewTag(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                const x = newTag.trim();
                if (x && !tags.includes(x)) setTags([...tags, x]);
                setNewTag("");
              }
            }}
          />
          <button
            type="button"
            onClick={() => {
              const x = newTag.trim();
              if (x && !tags.includes(x)) setTags([...tags, x]);
              setNewTag("");
            }}
            disabled={!newTag.trim()}
            className="btn-ghost px-4 py-2 text-sm disabled:opacity-50"
          >
            {t.users.addTag}
          </button>
        </div>
      </div>

      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <label className="block text-sm">
          <span className="font-medium">{t.users.birthday}</span>
          <input
            className="input mt-1 w-full"
            placeholder="MM-DD"
            maxLength={5}
            value={birthday}
            onChange={(e) => setBirthday(e.target.value)}
          />
          <span className="mt-1 block text-xs text-ink-muted">
            {t.users.birthdayHint}
          </span>
        </label>
        <label className="block text-sm">
          <span className="font-medium">{t.users.source}</span>
          <select
            className="input mt-1 w-full"
            value={source}
            onChange={(e) => setSource(e.target.value)}
          >
            <option value="">—</option>
            {SOURCES.map((sc) => (
              <option key={sc} value={sc}>
                {t.users.sourceLabel[sc]}
              </option>
            ))}
          </select>
        </label>
      </div>

      {error && (
        <p className="mt-3 rounded-lg bg-rose-50 px-3 py-2 text-sm text-brand dark:bg-rose-500/10 dark:text-rose-300">
          {error}
        </p>
      )}

      <div className="mt-4 flex items-center gap-3">
        <button
          type="button"
          onClick={save}
          disabled={saving}
          className="btn-primary px-5 py-2 text-sm disabled:opacity-60"
        >
          {saving ? t.common.saving : t.common.save}
        </button>
        {saved && (
          <span className="text-sm text-emerald-600 dark:text-emerald-400">
            {t.settings.saved}
          </span>
        )}
      </div>
    </section>
  );
}
