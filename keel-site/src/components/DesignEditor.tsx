"use client";

// The page-layout constructor.
//
// A layout is a list of bands. Each band names a block (which decides what it
// shows and where the content comes from), a variant (how it looks) and a width
// in **columns of twelve**.
//
// ⚠️ **Columns, never pixels, and that is the one decision everything else
// follows from.** A block dragged to a pixel on a 1400 px screen cannot be
// re-placed on a 380 px phone — no arithmetic recovers what the designer meant —
// and the same layout has to serve the Telegram mini app. In columns the phone
// rule is one line: everything becomes twelve. So the preview below shows both,
// side by side, and the phone view is not an afterthought — it is the same list
// with the same rule applied.
//
// Two more things this screen is careful about:
//
//   • **Saving is not publishing.** The draft is invisible to guests; an operator
//     mid-layout must not be showing a half-drawn page to a restaurant's
//     customers.
//   • **A template is a copy.** Applying one starts a drawing; it does not link
//     to it. Editing a template later must never redraw ten live sites.

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import {
  deleteDesignTemplate,
  designTemplates,
  publishTenantDesign,
  revertTenantDesign,
  saveDesignTemplate,
  saveTenantDesign,
  tenantDesign,
  type DesignSection,
  type DesignState,
  type DesignTemplate,
} from "@/lib/api";

/** What each block shows, in the operator's words. The content itself always
 *  comes from the restaurant's own data — a band has no text of its own, which is
 *  also what makes a layout portable between customers. */
const BLOCK_LABEL: Record<string, string> = {
  hero: "Hero — nom, shior, muqova",
  perks: "Uch afzallik kartochkasi",
  categories: "Menyu kategoriyalari",
  "menu-grid": "Taomlar panjarasi",
  search: "Qidiruv qatori",
  reviews: "Mehmonlar fikri",
  "hours-address": "Ish vaqti va manzil",
  about: "Biz haqimizda",
  gallery: "Galereya",
  cta: "Buyurtma/bron chaqiruvi",
};

const VARIANTS: Record<string, string[]> = {
  hero: ["full", "split", "compact"],
  perks: ["cards", "inline"],
  categories: ["tiles", "list"],
  "menu-grid": ["cards", "rows"],
  search: ["bar", "big"],
  reviews: ["cards"],
  "hours-address": ["map", "plain"],
  about: ["text", "text-image"],
  gallery: ["grid", "strip"],
  cta: ["banner", "buttons"],
};

const TONES = ["", "surface", "raised", "charcoal", "brand"];
const PADS = ["", "sm", "md", "lg"];

/** The page every restaurant has today, as bands.
 *
 *  Mirrors `models.DefaultSections()` and `DEFAULT_SECTIONS` in the tenant app.
 *  Offered as the starting point because a blank canvas is the wrong first
 *  screen: the operator is redrawing a page that already works, not inventing
 *  one. */
const TEMPLATE_DEFAULT: DesignSection[] = [
  { type: "hero", variant: "full", span: 12 },
  { type: "perks", variant: "cards", span: 12 },
  { type: "categories", variant: "tiles", span: 12 },
  {
    type: "menu-grid",
    variant: "cards",
    span: 12,
    binding: { popularOnly: true, limit: 8 },
  },
  { type: "reviews", variant: "cards", span: 12 },
  { type: "hours-address", variant: "map", span: 12 },
];

export default function DesignEditor({
  tenantId,
  slug,
}: {
  tenantId: string;
  slug: string;
}) {
  const [state, setState] = useState<DesignState | null>(null);
  const [sections, setSections] = useState<DesignSection[]>([]);
  const [templates, setTemplates] = useState<DesignTemplate[]>([]);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [dirty, setDirty] = useState(false);

  const load = useCallback(async () => {
    try {
      const s = await tenantDesign(tenantId);
      setState(s);
      setSections(s.draft.sections ?? s.live.sections ?? []);
      setDirty(false);
      setTemplates((await designTemplates()).items);
      setError("");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  function edit(next: DesignSection[]) {
    setSections(next);
    setDirty(true);
    setMessage("");
  }

  const patch = (i: number, part: Partial<DesignSection>) =>
    edit(sections.map((s, j) => (i === j ? { ...s, ...part } : s)));

  const move = (i: number, dir: -1 | 1) => {
    const j = i + dir;
    if (j < 0 || j >= sections.length) return;
    const next = [...sections];
    [next[i], next[j]] = [next[j], next[i]];
    edit(next);
  };

  async function run(fn: () => Promise<string>) {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      setMessage(await fn());
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (!state) {
    return (
      <section className="card">
        <p className="text-sm text-ink-muted">{error || "Yuklanmoqda…"}</p>
      </section>
    );
  }

  return (
    <section className="card">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-sm font-semibold text-ink">Sayt maketi</p>
        {/* ⚠️ The real editor is its own page: full width, the elements of a
            freely drawn band, and the customer's actual site in an iframe with the
            unpublished draft applied. This panel stays as the quick view — "what
            does this customer have now" — because that question gets asked while
            looking at their invoices, not while drawing. */}
        <Link
          href={`/console/tenants/${tenantId}/design`}
          className="rounded-xl bg-ink px-3 py-1.5 text-xs font-semibold text-surface"
        >
          Konstruktorni ochish →
        </Link>
        {state.published ? (
          <span className="rounded-full bg-emerald-500/15 px-2.5 py-0.5 text-xs font-semibold text-emerald-700 dark:text-emerald-300">
            chop etilgan
          </span>
        ) : (
          <span className="rounded-full bg-ink/5 px-2.5 py-0.5 text-xs text-ink-muted">
            standart shablon
          </span>
        )}
      </div>
      <p className="mt-1 text-xs text-ink-muted">
        Kenglik ustunlarda (12 = to&apos;liq). Telefonda har bir band to&apos;liq
        kenglikka tushadi — shuning uchun alohida mobil maket chizish kerak emas.
        Bloklar matnni saqlamaydi: kontent restoranning o&apos;z ma&apos;lumotidan
        keladi.
      </p>

      {/* ---- the bands ---- */}
      <div className="mt-4 space-y-2">
        {sections.map((s, i) => (
          <div key={i} className="rounded-2xl border border-line bg-raised p-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-semibold">
                {BLOCK_LABEL[s.type] ?? s.type}
              </span>
              <span className="ml-auto flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => move(i, -1)}
                  className="btn-ghost px-2 py-1 text-xs"
                  aria-label="Yuqoriga"
                >
                  ↑
                </button>
                <button
                  type="button"
                  onClick={() => move(i, 1)}
                  className="btn-ghost px-2 py-1 text-xs"
                  aria-label="Pastga"
                >
                  ↓
                </button>
                {/* Hidden rather than deleted: an operator trying a layout takes
                    a band out and puts it back, and deleting loses everything
                    tuned on it. */}
                <button
                  type="button"
                  onClick={() => patch(i, { hidden: !s.hidden })}
                  className="btn-ghost px-2 py-1 text-xs"
                >
                  {s.hidden ? "ko‘rsatish" : "yashirish"}
                </button>
                <button
                  type="button"
                  onClick={() => edit(sections.filter((_, j) => j !== i))}
                  className="btn-ghost px-2 py-1 text-xs text-rose-600"
                >
                  o‘chirish
                </button>
              </span>
            </div>

            <div className="mt-2 grid grid-cols-2 gap-2 sm:grid-cols-4">
              <label className="text-xs">
                <span className="block text-ink-muted">Ko‘rinish</span>
                <select
                  className="input mt-1"
                  value={s.variant ?? ""}
                  onChange={(e) => patch(i, { variant: e.target.value })}
                >
                  {(VARIANTS[s.type] ?? []).map((v) => (
                    <option key={v} value={v}>
                      {v}
                    </option>
                  ))}
                </select>
              </label>
              <label className="text-xs">
                <span className="block text-ink-muted">Kenglik (ustun)</span>
                <select
                  className="input mt-1"
                  value={s.span}
                  onChange={(e) => patch(i, { span: Number(e.target.value) })}
                >
                  {[12, 9, 8, 6, 4, 3].map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </select>
              </label>
              <label className="text-xs">
                <span className="block text-ink-muted">Fon</span>
                <select
                  className="input mt-1"
                  value={s.style?.tone ?? ""}
                  onChange={(e) =>
                    patch(i, { style: { ...s.style, tone: e.target.value } })
                  }
                >
                  {TONES.map((v) => (
                    <option key={v} value={v}>
                      {v || "standart"}
                    </option>
                  ))}
                </select>
              </label>
              <label className="text-xs">
                <span className="block text-ink-muted">Bo‘shliq</span>
                <select
                  className="input mt-1"
                  value={s.style?.padding ?? ""}
                  onChange={(e) =>
                    patch(i, { style: { ...s.style, padding: e.target.value } })
                  }
                >
                  {PADS.map((v) => (
                    <option key={v} value={v}>
                      {v || "standart"}
                    </option>
                  ))}
                </select>
              </label>
            </div>

            {s.type === "menu-grid" && (
              <label className="mt-2 flex items-center gap-2 text-xs">
                <input
                  type="checkbox"
                  checked={!!s.binding?.popularOnly}
                  onChange={(e) =>
                    patch(i, {
                      binding: { ...s.binding, popularOnly: e.target.checked },
                    })
                  }
                />
                faqat mashhur taomlar
              </label>
            )}
          </div>
        ))}
        {sections.length === 0 && (
          <p className="rounded-2xl border border-dashed border-line p-4 text-center text-sm text-ink-muted">
            Band yo‘q — pastdan qo‘shing yoki standart shablondan boshlang.
          </p>
        )}
      </div>

      {/* ---- palette ---- */}
      <div className="mt-3 flex flex-wrap gap-2">
        {state.blocks.map((b) => (
          <button
            key={b}
            type="button"
            onClick={() =>
              edit([
                ...sections,
                { type: b, variant: VARIANTS[b]?.[0], span: 12 },
              ])
            }
            className="rounded-full border border-line px-3 py-1 text-xs hover:border-brand hover:text-brand"
          >
            + {BLOCK_LABEL[b] ?? b}
          </button>
        ))}
      </div>

      {/* ---- preview: the same list, twice ---- */}
      <div className="mt-5 grid gap-4 lg:grid-cols-[1fr_200px]">
        <Preview sections={sections} phone={false} />
        <Preview sections={sections} phone />
      </div>

      {/* ---- actions ---- */}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={busy || !dirty}
          onClick={() =>
            run(async () => {
              await saveTenantDesign(tenantId, sections);
              return "Qoralama saqlandi (jonli sayt o‘zgarmadi)";
            })
          }
          className="btn-ghost px-4 py-2 text-sm disabled:opacity-40"
        >
          Qoralamani saqlash
        </button>
        <button
          type="button"
          disabled={busy || sections.length === 0}
          onClick={() =>
            run(async () => {
              if (dirty) await saveTenantDesign(tenantId, sections);
              const res = await publishTenantDesign(tenantId);
              return `Chop etildi (${res.published} band). ${res.note}`;
            })
          }
          className="btn-primary px-4 py-2 text-sm disabled:opacity-40"
        >
          Chop etish
        </button>
        {state.published && (
          <button
            type="button"
            disabled={busy}
            onClick={() =>
              run(async () => {
                await revertTenantDesign(tenantId);
                return "Standart shablonga qaytarildi (chizma saqlanib qoldi)";
              })
            }
            className="btn-ghost px-4 py-2 text-sm disabled:opacity-40"
          >
            Shablonga qaytarish
          </button>
        )}
        <a
          href={`https://${slug}.keel.uz`}
          target="_blank"
          rel="noreferrer"
          className="ml-auto text-sm font-semibold text-brand hover:underline"
        >
          Jonli saytni ochish →
        </a>
      </div>

      {/* ---- templates: one drawing, several customers ---- */}
      <div className="mt-5 border-t border-line pt-4">
        <p className="text-sm font-semibold">Shablonlar</p>
        <p className="mt-1 text-xs text-ink-muted">
          Qo‘llash <b>nusxa oladi</b>: shablonni keyin tahrirlash bu mijozning
          saytini o‘zgartirmaydi.
        </p>
        <div className="mt-2 flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => edit(TEMPLATE_DEFAULT)}
            className="chip"
          >
            Standart maket
          </button>
          {templates.map((tpl) => (
            <span key={tpl.id} className="flex items-center gap-1">
              <button
                type="button"
                onClick={() => edit(tpl.sections)}
                className="chip"
              >
                {tpl.name}
              </button>
              <button
                type="button"
                onClick={() =>
                  run(async () => {
                    await deleteDesignTemplate(tpl.id);
                    return "Shablon o‘chirildi";
                  })
                }
                className="text-xs text-ink-muted hover:text-rose-600"
                aria-label="O‘chirish"
              >
                ×
              </button>
            </span>
          ))}
          <button
            type="button"
            disabled={busy || sections.length === 0}
            onClick={() => {
              const name = window.prompt("Shablon nomi");
              if (!name) return;
              run(async () => {
                await saveDesignTemplate(name, sections);
                return "Shablon saqlandi";
              });
            }}
            className="chip disabled:opacity-40"
          >
            + Hozirgini shablon qilish
          </button>
        </div>
      </div>

      {error && <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">{error}</p>}
      {message && (
        <p className="mt-3 text-sm font-semibold text-emerald-700 dark:text-emerald-300">
          {message}
        </p>
      )}
    </section>
  );
}

/** The layout as boxes.
 *
 *  ⚠️ A schematic, not a screenshot — and deliberately so. What an operator has to
 *  judge before publishing is the **structure**: which bands, in what order, how
 *  wide, and what that becomes on a phone. A pixel-accurate preview would answer
 *  a question the live site answers better, and would need the draft published to
 *  do it.
 *
 *  The phone column is the same list with the one rule applied: every band full
 *  width, in the drawn order. Seeing it next to the desktop view is what stops a
 *  layout being designed for one screen. */
function Preview({
  sections,
  phone,
}: {
  sections: DesignSection[];
  phone: boolean;
}) {
  const visible = sections.filter((s) => !s.hidden);
  return (
    <div>
      <p className="mb-1 text-xs font-semibold uppercase tracking-wider text-ink-muted">
        {phone ? "Telefon (mini app)" : "Kompyuter"}
      </p>
      <div
        className={`rounded-2xl border border-line bg-raised p-2 ${
          phone ? "mx-auto max-w-[200px]" : ""
        }`}
      >
        <div className="grid grid-cols-12 gap-1">
          {visible.map((s, i) => (
            <div
              key={i}
              className="rounded-lg bg-brand/15 px-1 py-2 text-center text-[10px] leading-tight text-ink-soft"
              style={{ gridColumn: `span ${phone ? 12 : s.span} / span ${phone ? 12 : s.span}` }}
            >
              {s.type}
              {!phone && s.span !== 12 && (
                <span className="block text-ink-muted">{s.span}/12</span>
              )}
            </div>
          ))}
          {visible.length === 0 && (
            <div className="col-span-12 py-6 text-center text-xs text-ink-muted">
              bo‘sh
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
