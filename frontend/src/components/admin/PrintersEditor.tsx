"use client";

// The printers a branch has, and which receipt comes out of each.
//
// ⚠️ **One address box, not a form of five.** The person setting this up is
// standing next to the printer reading its self-test page, or looking at the
// Windows printer list. Every field they have to translate into is a field they
// can get wrong, and the difference between one box and five is whether a
// restaurant finishes the install or phones us.
//
// ⚠️ **The test button is the most useful control here**, for the reason the
// SMS page's is: the address can be typed perfectly and the printer still be
// switched off, on another subnet, or shared under a different name — and from
// a form all four look identical until a real ticket fails at eight o'clock.

import { useEffect, useState } from "react";

import { api, ApiError } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import PrintQueuePanel from "@/components/admin/PrintQueuePanel";
import type { Printer } from "@/lib/types";

/** What a printer can be asked to print. The order is the order of the day:
 *  the kitchen ticket, then the bill, then the receipt. */
// ⚠️ **"label" is here and last on purpose.** It is a different machine with a
// different roll — a shop's sticker printer — and putting it beside the receipt
// kinds is what stops somebody ticking it on the till's printer and answering a
// price change with a receipt-shaped strip of stickers.
const KINDS = ["kitchen", "precheck", "till", "customer", "label"] as const;

export default function PrintersEditor({
  printers,
  onChange,
}: {
  printers: Printer[];
  onChange: (next: Printer[]) => void;
}) {
  const t = useAdminT();
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  /** The menu's sections, so a printer can be told which ones it takes.
   *
   *  ⚠️ Loaded once with the editor rather than when a printer is expanded: the
   *  wait belongs to opening a settings page, not to the moment somebody has
   *  decided which roll the drinks come off. */
  const [categories, setCategories] = useState<{ id: string; name: string }[]>(
    [],
  );

  useEffect(() => {
    void api
      .adminCategories()
      .then((rows) =>
        setCategories(rows.map((c) => ({ id: c.id, name: c.name }))),
      )
      .catch(() => setCategories([]));
  }, []);

  function patch(id: string, p: Partial<Printer>) {
    onChange(printers.map((x) => (x.id === id ? { ...x, ...p } : x)));
  }

  function add() {
    onChange([
      ...printers,
      {
        // A local id until the server mints one on save; it only has to be
        // unique within this form.
        id: `new-${Date.now()}`,
        name: "",
        target: "",
        // ⚠️ Nothing chosen: a printer that started out printing everything
        // would put every bill in the building on the pass's roll the moment
        // somebody added it and walked away.
        kinds: [],
        charset: "latin",
        cut: true,
        copies: 1,
      },
    ]);
  }

  async function test(p: Printer) {
    setBusy(p.id);
    setNote("");
    setError("");
    try {
      const res = await api.testPrint(p.id);
      setNote(res.queued > 0 ? t.printers.testQueued : t.printers.testNoAgent);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : t.common.saveFailed);
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="mt-6">
      <h3 className="text-sm font-semibold">{t.printers.title}</h3>
      <p className="mt-1 text-xs leading-relaxed text-ink-muted">
        {t.printers.intro}
      </p>
      <p className="mt-1 text-xs leading-relaxed text-ink-muted">
        {t.printers.agentHint}
      </p>

      <div className="mt-3 space-y-3">
        {printers.map((p) => (
          <div key={p.id} className="rounded-2xl border border-line p-3">
            <div className="flex flex-wrap gap-2">
              <label className="min-w-[10rem] flex-1 text-xs">
                <span className="text-ink-muted">{t.printers.name}</span>
                <input
                  className="input mt-1"
                  value={p.name}
                  onChange={(e) => patch(p.id, { name: e.target.value })}
                />
              </label>
              <label className="min-w-[14rem] flex-[2] text-xs">
                <span className="text-ink-muted">{t.printers.target}</span>
                <input
                  className="input mt-1 font-mono"
                  value={p.target}
                  placeholder="tcp://192.168.1.50:9100"
                  onChange={(e) => patch(p.id, { target: e.target.value })}
                />
              </label>
            </div>
            <p className="mt-1 font-mono text-[11px] text-ink-muted">
              {t.printers.targetHint}
            </p>

            <div className="mt-3 flex flex-wrap items-center gap-2">
              <span className="text-xs text-ink-muted">
                {t.printers.kinds}:
              </span>
              {KINDS.map((k) => {
                const on = p.kinds.includes(k);
                return (
                  <button
                    key={k}
                    type="button"
                    onClick={() =>
                      patch(p.id, {
                        kinds: on
                          ? p.kinds.filter((x) => x !== k)
                          : [...p.kinds, k],
                      })
                    }
                    className={
                      on
                        ? "rounded-lg bg-ink px-2.5 py-1.5 text-xs font-semibold text-surface"
                        : "rounded-lg border border-line px-2.5 py-1.5 text-xs font-semibold text-ink-soft hover:bg-ink/5"
                    }
                  >
                    {t.receipts.kinds[k === "precheck" ? "customer" : k]}
                    {k === "precheck" ? " · " + t.till.precheckShort : ""}
                  </button>
                );
              })}
            </div>

            {/* ⚠️ **Only shown for a printer that takes kitchen tickets.** A
                bill is the whole bill and a guest's copy is the whole meal —
                splitting either across two printers gives somebody half a
                receipt. Routing is a kitchen question, so the control appears
                where the question exists and nowhere else. */}
            {p.kinds.includes("kitchen") && (
              <div className="mt-3">
                <span className="text-xs text-ink-muted">
                  {t.printers.categories}
                </span>
                <p className="mt-0.5 text-[11px] text-ink-muted">
                  {/* ⚠️ The default is spelled out, because "none selected"
                      reads as "prints nothing" — and here it means the
                      opposite, which is what every restaurant with one printer
                      relies on. */}
                  {p.categories?.length
                    ? t.printers.categoriesSome
                    : t.printers.categoriesAll}
                </p>
                <div className="mt-1.5 flex flex-wrap gap-1.5">
                  {categories.map((c) => {
                    const on = p.categories?.includes(c.id) ?? false;
                    return (
                      <button
                        key={c.id}
                        type="button"
                        onClick={() =>
                          patch(p.id, {
                            categories: on
                              ? (p.categories ?? []).filter((x) => x !== c.id)
                              : [...(p.categories ?? []), c.id],
                          })
                        }
                        className={
                          on
                            ? "rounded-lg bg-brand px-2.5 py-1.5 text-xs font-semibold text-white"
                            : "rounded-lg border border-line px-2.5 py-1.5 text-xs text-ink-soft hover:bg-ink/5"
                        }
                      >
                        {c.name}
                      </button>
                    );
                  })}
                </div>
              </div>
            )}

            <div className="mt-3 flex flex-wrap items-center gap-4 text-xs">
              <label>
                <span className="text-ink-muted">{t.printers.charset}</span>
                <select
                  className="input mt-1"
                  value={p.charset ?? "latin"}
                  onChange={(e) => patch(p.id, { charset: e.target.value })}
                >
                  <option value="latin">{t.printers.latin}</option>
                  <option value="cyrillic">{t.printers.cyrillic}</option>
                </select>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={!!p.cut}
                  onChange={(e) => patch(p.id, { cut: e.target.checked })}
                />
                {t.printers.cut}
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={!!p.drawer}
                  onChange={(e) => patch(p.id, { drawer: e.target.checked })}
                />
                {t.printers.drawer}
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={!!p.disabled}
                  onChange={(e) => patch(p.id, { disabled: e.target.checked })}
                />
                {t.printers.disabled}
              </label>
              <div className="ml-auto flex gap-2">
                <button
                  type="button"
                  className="btn"
                  disabled={busy === p.id || !p.target}
                  onClick={() => void test(p)}
                >
                  {t.printers.test}
                </button>
                <button
                  type="button"
                  className="btn-ghost text-danger"
                  onClick={() =>
                    onChange(printers.filter((x) => x.id !== p.id))
                  }
                >
                  {t.printers.remove}
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      <button type="button" className="btn mt-3" onClick={add}>
        {t.printers.add}
      </button>

      {/* ⚠️ Under the printers rather than on a page of its own: the person who
          asks "why did nothing come out" is already here, looking at the
          address they typed. A separate screen would be found by whoever went
          looking for it, which is nobody. */}
      <PrintQueuePanel />
      {note && <p className="mt-2 text-xs text-ink-soft">{note}</p>}
      {error && <p className="mt-2 text-xs text-danger">{error}</p>}
    </div>
  );
}
