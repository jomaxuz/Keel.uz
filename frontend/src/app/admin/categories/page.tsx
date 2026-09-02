"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import ImageUpload from "@/components/admin/ImageUpload";
import Modal from "@/components/admin/Modal";
import { ListScroll } from "@/components/admin/PagedList";
import { useAdminT } from "@/lib/i18n/admin";
import type { Category } from "@/lib/types";
import { useAsk } from "@/components/ui/Ask";

function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/['`]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

const empty: Category = {
  id: "",
  name: "",
  nameRu: "",
  nameEn: "",
  slug: "",
  sortOrder: 0,
  isActive: true,
  imageUrl: "",
};

export default function AdminCategoriesPage() {
  const [cats, setCats] = useState<Category[]>([]);
  const { ask, tell } = useAsk();
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Category | null>(null);
  const [saving, setSaving] = useState(false);
  const t = useAdminT();

  function load() {
    setLoading(true);
    api
      .adminCategories()
      .then(setCats)
      .catch(() => setCats([]))
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  async function save() {
    if (!editing || !editing.name.trim()) return;
    setSaving(true);
    const payload: Category = {
      ...editing,
      slug: editing.slug.trim() || slugify(editing.name),
    };
    try {
      if (editing.id) {
        await api.updateCategory(editing.id, payload);
      } else {
        await api.createCategory(payload);
      }
      setEditing(null);
      load();
    } catch {
      void tell({ title: t.common.saveFailed });
    } finally {
      setSaving(false);
    }
  }

  async function remove(c: Category) {
    if (
      !(await ask({
        title: t.categories.confirmDeleteFull(c.name),
        danger: true,
      }))
    )
      return;
    try {
      await api.deleteCategory(c.id);
      load();
    } catch {
      void tell({ title: t.common.deleteFailed });
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t.categories.title}</h1>
        <button
          type="button"
          onClick={() => setEditing({ ...empty })}
          className="btn-primary px-4 py-2"
        >
          {t.categories.addNew}
        </button>
      </div>

      <div className="mt-6 overflow-hidden rounded-3xl border border-line bg-surface shadow-card">
        {loading ? (
          <p className="py-10 text-center text-ink-muted/70">
            {t.common.loading}
          </p>
        ) : cats.length === 0 ? (
          <p className="py-10 text-center text-ink-muted/70">
            {t.categories.empty}
          </p>
        ) : (
          <ListScroll max="max-h-[70vh]">
            <ul className="divide-y divide-line">
              {cats.map((c) => (
                <li key={c.id} className="flex items-center gap-4 p-4">
                  <span className="w-8 text-center text-sm text-ink-muted/70">
                    {c.sortOrder}
                  </span>
                  <span className="flex-1 font-medium">{c.name}</span>
                  {!c.isActive && (
                    <span className="rounded-full bg-ink/5 px-2 py-0.5 text-xs text-ink-muted">
                      {t.menu.hidden}
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => setEditing({ ...c })}
                    className="text-sm text-brand hover:underline"
                  >
                    {t.common.edit}
                  </button>
                  <button
                    type="button"
                    onClick={() => remove(c)}
                    className="text-sm text-ink-muted/70 hover:text-brand"
                  >
                    {t.common.delete}
                  </button>
                </li>
              ))}
            </ul>
          </ListScroll>
        )}
      </div>

      {editing && (
        <Modal onClose={() => setEditing(null)}>
          <h2 className="text-lg font-bold">
            {editing.id ? t.categories.editTitle : t.categories.newTitle}
          </h2>

          <label className="mt-4 block text-sm">
            <span className="font-medium">{t.categories.name}</span>
            <input
              className="input mt-1"
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
              autoFocus
            />
          </label>

          {/* Optional translations — empty falls back to the Uzbek name. */}
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium">{t.categories.nameRu}</span>
              <input
                className="input mt-1"
                value={editing.nameRu ?? ""}
                placeholder={editing.name}
                onChange={(e) =>
                  setEditing({ ...editing, nameRu: e.target.value })
                }
              />
            </label>
            <label className="block text-sm">
              <span className="font-medium">{t.categories.nameEn}</span>
              <input
                className="input mt-1"
                value={editing.nameEn ?? ""}
                placeholder={editing.name}
                onChange={(e) =>
                  setEditing({ ...editing, nameEn: e.target.value })
                }
              />
            </label>
          </div>

          <div className="mt-4 grid grid-cols-2 gap-4">
            <label className="block text-sm">
              <span className="font-medium">{t.categories.sortOrder}</span>
              <input
                type="number"
                className="mt-1 w-full rounded-xl border border-line-strong px-3 py-2 text-sm outline-none focus:border-brand"
                value={editing.sortOrder}
                onChange={(e) =>
                  setEditing({
                    ...editing,
                    sortOrder: Number(e.target.value) || 0,
                  })
                }
              />
            </label>
            <label className="mt-6 flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={editing.isActive}
                onChange={(e) =>
                  setEditing({ ...editing, isActive: e.target.checked })
                }
              />
              <span className="font-medium">{t.categories.isActiveLabel}</span>
            </label>
          </div>

          <div className="mt-4">
            <span className="text-sm font-medium">
              {t.categories.imageOptional}
            </span>
            <div className="mt-1">
              <ImageUpload
                value={editing.imageUrl}
                onChange={(url) => setEditing({ ...editing, imageUrl: url })}
              />
            </div>
          </div>

          <div className="mt-6 flex justify-end gap-3">
            <button
              type="button"
              onClick={() => setEditing(null)}
              className="btn-ghost px-4 py-2"
            >
              {t.common.cancel}
            </button>
            <button
              type="button"
              onClick={save}
              disabled={saving}
              className="btn-primary px-4 py-2 disabled:opacity-60"
            >
              {saving ? t.common.saving : t.common.save}
            </button>
          </div>
        </Modal>
      )}
    </div>
  );
}
