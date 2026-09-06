"use client";

// Scanning marked goods as they arrive.
//
// ⚠️ **The whole screen is about *where* a refusal lands.** A marked bottle
// bought outside the system, or one whose code has already been withdrawn, is
// refused by the tax register at the moment of sale — in front of a customer,
// with the cashier's only remedy being to scan it again, which cannot help.
// Scanned here, the same fact turns up in the store room with the box still open
// and the supplier's number still on the invoice.
//
// ⚠️ **The scanner is a keyboard.** There is no driver and no device API: the
// code arrives as typed text ending in Enter. So the whole screen is one focused
// box that never loses focus, and everything else on it is a list of what has
// already gone in — a form somebody has to click back into after every bottle is
// a form that gets abandoned at the fifth.
//
// ⚠️ **This is not an Asl Belgisi API.** Whether a code was ever issued is the
// national system's answer, given through the fiscal receipt. What is kept here
// is a fact about this shop: these are the bottles we took in.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import { useAdminScope } from "@/lib/adminScope";
import { useAsk } from "@/components/ui/Ask";
import { ListScroll } from "@/components/admin/PagedList";
import type { MenuItem } from "@/lib/types";

export default function AdminMarkingPage() {
  const t = useAdminT();
  const { tell } = useAsk();
  const scope = useAdminScope();

  const [items, setItems] = useState<MenuItem[]>([]);
  const [itemId, setItemId] = useState("");
  const [held, setHeld] = useState<number | null>(null);
  /** What has been scanned but not saved. ⚠️ Kept in the order it was scanned:
   *  the person unpacking is comparing it against a box, top to bottom. */
  const [codes, setCodes] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const box = useRef<HTMLInputElement>(null);

  useEffect(() => {
    void api
      .adminMenu()
      // ⚠️ Only marked products: a list of the whole menu would bury the six
      // things this screen is for under four hundred that carry no code.
      .then((rows) => setItems(rows.filter((m) => m.marked)))
      .catch(() => setItems([]));
  }, [scope.scopeKey]);

  const loadHeld = useCallback(async () => {
    if (!itemId) {
      setHeld(null);
      return;
    }
    try {
      const res = await api.adminMarkStock(itemId);
      setHeld(res.held);
    } catch {
      setHeld(null);
    }
  }, [itemId]);

  useEffect(() => {
    void loadHeld();
  }, [loadHeld, scope.scopeKey]);

  // ⚠️ **Focus is taken back after every render.** The scanner types into
  // whatever has focus, and a code that landed in the product picker is a code
  // nobody saved and nobody can see was lost.
  useEffect(() => {
    box.current?.focus();
  });

  const chosen = useMemo(
    () => items.find((m) => m.id === itemId),
    [items, itemId],
  );

  function scanned(raw: string) {
    const code = raw.trim();
    if (code === "") return;
    setCodes((cur) => {
      // ⚠️ A repeat is dropped here rather than sent: the pistol beeps, nobody
      // is sure it took, it is scanned again — and the server would answer with
      // a duplicate somebody then has to read past.
      if (cur.includes(code)) return cur;
      return [...cur, code];
    });
  }

  async function save() {
    if (codes.length === 0) return;
    setBusy(true);
    try {
      const res = await api.adminReceiveMarks({
        menuItemId: itemId || undefined,
        codes,
      });
      const parts = [t.marking.added(res.added)];
      if (res.duplicates.length > 0) {
        parts.push(t.marking.dup(res.duplicates.length));
      }
      if (res.bad.length > 0) parts.push(t.marking.bad(res.bad.length));
      void tell({ title: parts.join(" · ") });
      setCodes([]);
      await loadHeld();
    } catch (e) {
      void tell({
        title: e instanceof ApiError ? e.message : t.common.saveFailed,
      });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-semibold">{t.marking.title}</h1>
        <p className="text-xs text-ink-muted">{t.marking.hint}</p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="block text-sm">
          <span className="text-xs text-ink-muted">{t.marking.product}</span>
          <select
            className="input mt-1 w-full"
            value={itemId}
            onChange={(e) => setItemId(e.target.value)}
          >
            <option value="">{t.marking.productAny}</option>
            {items.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </select>
          {/* ⚠️ Said plainly: the code's own GTIN names a product in the
              national catalogue, which is not this restaurant's menu — matching
              them would be a mapping nobody has filled in. */}
          <span className="mt-1 block text-xs text-ink-muted">
            {t.marking.productHint}
          </span>
        </label>

        {chosen && held !== null && (
          <div className="rounded-xl border border-line p-3">
            <p className="text-xs text-ink-muted">{t.marking.held}</p>
            <p className="text-2xl font-semibold tabular-nums">{held}</p>
          </div>
        )}
      </div>

      <label className="block text-sm">
        <span className="text-xs text-ink-muted">{t.marking.scan}</span>
        <input
          ref={box}
          className="input mt-1 w-full font-mono"
          placeholder={t.marking.scanPlaceholder}
          autoFocus
          onKeyDown={(e) => {
            if (e.key !== "Enter") return;
            e.preventDefault();
            scanned(e.currentTarget.value);
            e.currentTarget.value = "";
          }}
        />
      </label>

      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-ink-muted">
          {t.marking.pending(codes.length)}
        </p>
        <div className="flex gap-2">
          {codes.length > 0 && (
            <button
              className="btn-ghost px-3 py-2 text-sm"
              onClick={() => setCodes([])}
            >
              {t.marking.clear}
            </button>
          )}
          <button
            className="btn-primary px-4 py-2 text-sm"
            disabled={busy || codes.length === 0}
            onClick={() => void save()}
          >
            {t.marking.save}
          </button>
        </div>
      </div>

      <ListScroll>
        <div className="space-y-1">
          {codes.map((c, i) => (
            <div
              key={c}
              className="flex items-center gap-3 rounded-lg border border-line px-3 py-2"
            >
              <span className="w-6 text-xs text-ink-muted tabular-nums">
                {i + 1}
              </span>
              {/* ⚠️ Shown in full and monospaced: the person is comparing it
                  against a sticker in their hand, and a truncated code is one
                  they cannot check. */}
              <span className="flex-1 break-all font-mono text-xs">{c}</span>
              <button
                className="text-xs text-ink-muted underline"
                onClick={() => setCodes((cur) => cur.filter((x) => x !== c))}
              >
                {t.marking.remove}
              </button>
            </div>
          ))}
        </div>
      </ListScroll>
    </div>
  );
}
