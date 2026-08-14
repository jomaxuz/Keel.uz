"use client";

// Arranging the front page.
//
// Twenty figures, of which roughly none are useful to everybody: an owner opens
// this screen for takings and the average bill, a branch manager opens it to
// see what is unconfirmed. A dashboard where the number you came for is
// fourteenth down is one people stop reading — and then stop trusting, because
// they form their impression of the month from the orders list instead.
//
// ⚠️ **Switches, not a blank canvas.** The list says what is *off*, and
// everything starts on. A screen that opened empty and asked the owner to build
// their dashboard would be abandoned halfway by most of them, leaving a worse
// dashboard than the one they started with.

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { DASHBOARD_TILES, TILE_GROUPS, tileById } from "@/lib/dashboardTiles";
import type { DashboardPrefs } from "@/lib/types";

export default function DashboardCustomiser({
  prefs,
  onClose,
  onSaved,
}: {
  prefs: DashboardPrefs;
  onClose: () => void;
  onSaved: (next: DashboardPrefs) => void;
}) {
  const t = useAdminT();
  // Edited locally and sent on save, rather than a request per checkbox: an
  // owner tidying the page toggles six things in ten seconds, and six requests
  // racing each other decide the layout by whichever the network delivers last.
  const [hidden, setHidden] = useState<string[]>(prefs.hidden);
  const [order, setOrder] = useState<string[]>(
    // Seeded with the full resolved order, so dragging one tile up does not
    // have to reason about "where do the unnamed ones go".
    prefs.visible.length ? fullOrder(prefs) : DASHBOARD_TILES.map((x) => x.id),
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  // Escape closes it. This panel covers the numbers the reader came for, and a
  // dialog that can only be dismissed by finding its button is one people learn
  // to avoid opening.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  function toggle(id: string) {
    setHidden((h) => (h.includes(id) ? h.filter((x) => x !== id) : [...h, id]));
  }

  function move(id: string, delta: number) {
    setOrder((current) => {
      const i = current.indexOf(id);
      const j = i + delta;
      if (i < 0 || j < 0 || j >= current.length) return current;
      const next = [...current];
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
  }

  async function save() {
    setSaving(true);
    setError("");
    try {
      await api.adminSaveDashboard({ hidden, order });
      // Re-read rather than assume: the server drops ids it does not know, and
      // the page must draw what was actually stored, not what was sent.
      onSaved(await api.adminDashboardPrefs());
      onClose();
    } catch {
      setError(t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  const d = t.dashboard.customise;

  return (
    <section className="mt-4 rounded-3xl border border-line bg-surface p-5 shadow-card">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-bold">{d.title}</h2>
          <p className="mt-1 text-xs text-ink-muted">{d.hint}</p>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={onClose} className="btn btn-ghost">
            {t.common.cancel}
          </button>
          <button
            type="button"
            onClick={save}
            disabled={saving}
            className="btn btn-primary disabled:opacity-60"
          >
            {saving ? t.common.saving : t.common.save}
          </button>
        </div>
      </div>

      {error && <p className="mt-3 text-sm text-brand">{error}</p>}

      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        {TILE_GROUPS.map((group) => {
          const tiles = order
            .map(tileById)
            .filter((tile) => tile && tile.group === group);
          if (!tiles.length) return null;
          return (
            <div key={group}>
              <h3 className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
                {t.dashboard[`group${cap(group)}` as keyof typeof t.dashboard] as string}
              </h3>
              <ul className="mt-2 divide-y divide-line rounded-2xl border border-line">
                {tiles.map((tile) => {
                  const id = tile!.id;
                  const off = hidden.includes(id);
                  return (
                    <li key={id} className="flex items-center gap-2 px-3 py-2">
                      <label className="flex flex-1 cursor-pointer items-center gap-2">
                        <input
                          type="checkbox"
                          checked={!off}
                          onChange={() => toggle(id)}
                          className="h-4 w-4 accent-brand"
                        />
                        <span className={`text-sm ${off ? "text-ink-muted line-through" : "text-ink"}`}>
                          {tile!.label(t)}
                        </span>
                      </label>
                      {/* Arrows rather than drag-and-drop: no library, works
                          with a keyboard, and works on the tablet the manager
                          actually uses. */}
                      <button
                        type="button"
                        onClick={() => move(id, -1)}
                        aria-label={d.moveUp}
                        className="rounded-lg border border-line px-2 py-1 text-xs text-ink-soft hover:bg-ink/5"
                      >
                        ↑
                      </button>
                      <button
                        type="button"
                        onClick={() => move(id, 1)}
                        aria-label={d.moveDown}
                        className="rounded-lg border border-line px-2 py-1 text-xs text-ink-soft hover:bg-ink/5"
                      >
                        ↓
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          );
        })}
      </div>

      <button
        type="button"
        onClick={() => {
          setHidden([]);
          setOrder(DASHBOARD_TILES.map((x) => x.id));
        }}
        className="mt-4 text-xs text-ink-muted underline hover:text-ink"
      >
        {d.reset}
      </button>
    </section>
  );
}

/** The admin's full order including the tiles they have hidden.
 *
 *  `visible` leaves the hidden ones out, and editing from it would lose their
 *  positions — switch a tile back on and it would jump to the end, which reads
 *  as the setting having forgotten something. */
function fullOrder(prefs: DashboardPrefs): string[] {
  const named = prefs.order.filter((id) => tileById(id));
  const rest = DASHBOARD_TILES.map((x) => x.id).filter((id) => !named.includes(id));
  return [...named, ...rest];
}

function cap(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
