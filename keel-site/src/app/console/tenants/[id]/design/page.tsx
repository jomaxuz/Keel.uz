"use client";

// The layout editor, as its own page.
//
// It started life as a panel inside the tenant card, and that was wrong for a
// simple reason: this is the screen somebody sits at for an hour with a customer's
// screenshot open beside it. A panel between an invoice list and a container log
// gets a third of the width and none of the attention.
//
// The shape is the one every layout editor converges on, and each half earns its
// place:
//
//   • **Left: what the page is made of.** Bands in order, and inside a freely
//     drawn band, its elements. Selecting one opens its settings underneath —
//     rather than in a dialog, because the next thing after changing a setting is
//     always changing another one.
//
//   • **Right: the real site.** ⚠️ An iframe of the tenant's own domain with the
//     unpublished draft applied, not a schematic. A schematic can show that a band
//     is 6 columns wide; it cannot answer "does this look like the picture the
//     customer sent us", and that is the entire job. The fonts, the photographs,
//     the real dish names and the restaurant's accent are what make a layout look
//     right or wrong.
//
//   • **A phone width beside the desktop one**, switchable. Not decoration:
//     freely placed elements have a **separate phone layout**, and the whole
//     failure mode of free placement is a composition nobody checked at 390px.
//
// ⚠️ The preview is reloaded on demand rather than on every keystroke. It is a
// full page render of somebody's real site — reloading it per drag would make the
// editor unusable and the tenant's container busy for no reason.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import EditorCanvas from "@/components/design/EditorCanvas";
import PreviewOverlay from "@/components/design/PreviewOverlay";
import NavEditor from "@/components/design/NavEditor";
import { useT } from "@/lib/i18n/client";
import {
  BAND_LABELS,
  ELEMENT_LABELS,
  editorDict,
  type EditorDict,
} from "@/lib/i18n/editor";
import {
  previewTenantDesign,
  publishTenantDesign,
  revertTenantDesign,
  saveTenantDesign,
  tenantDesign,
  tenant as tenantApi,
  type DesignBox,
  type DesignElement,
  type DesignSection,
  type DesignState,
  type DesignTemplate,
  type NavLink,
  type SiteThemePatch,
  type StylePreset,
  designSchema,
  designTemplates,
} from "@/lib/api";
import SchemaSettings, {
  Control,
  defFor,
  type SectionDef,
} from "@/components/design/SchemaSettings";

const VARIANTS: Record<string, string[]> = {
  hero: ["full", "split", "compact"],
  perks: ["cards", "inline"],
  categories: ["tiles", "list"],
  "menu-grid": ["cards", "rows"],
  // "bar" is the line under the hero; "big" is a band of its own with a heading.
  search: ["bar", "big"],
  "hours-address": ["map", "plain"],
  about: ["text", "text-image"],
  gallery: ["grid", "strip"],
  cta: ["banner", "buttons"],
  navbar: ["classic", "centered", "minimal", "transparent"],
  footer: ["columns", "compact", "centered"],
  canvas: ["free"],
  popup: ["center", "bottom"],
};

const TONES = ["", "surface", "raised", "charcoal", "brand", "accent", "ink"];

/** How wide a band's contents may run. ⚠️ Empty is the page column, which is
 *  what every band of every existing design means — including the five built-in
 *  templates. Mirrors `designWidths` on the tenant. */
const WIDTHS = ["", "wide", "full"] as const;

/** Which corners the radius rounds, and how far. ⚠️ The two shapes every shop
 *  reference is built from and the editor could not draw: a panel rounded on
 *  the edge that faces the page, and a rail of words turned a quarter turn.
 *  Mirror `elementCorners` / `elementRotations` / `elementRadii`. */
const CORNERS = ["", "left", "right", "top", "bottom"] as const;
const ROTATIONS = ["", "-90", "90"] as const;
const RADII = ["", "sm", "md", "lg", "xl", "2xl", "full"] as const;

/** The icons an `icon` element may be. Mirrors `elementIcons`; the second row
 *  is the shop set — the first twelve were drawn for a restaurant. */
const ICONS = [
  "", "star", "clock", "phone", "pin", "fire", "leaf", "truck", "check",
  "heart", "cart", "chef",
  "play", "search", "user", "bag", "arrow-up", "arrow-down", "arrow-right",
  "arrow-left", "plus", "minus",
] as const;
const COLORS = ["", "ink", "soft", "muted", "white", "brand", "surface", "charcoal"];
const LINKS = ["", "/", "/menu", "/cart", "/checkout", "/bron", "/about", "/profile"];

/** A new element, sized so it is visible the moment it appears. An element added
 *  at 0×0 is an element the operator has to hunt for. */
function newElement(type: string): DesignElement {
  const box: DesignBox = { x: 10, y: 20, w: 40, h: 20, z: 1 };
  if (type === "text") {
    return { type, box: { ...box, h: 12 }, text: { uz: "Matn", ru: "", en: "" }, style: { size: 4, weight: "bold" } };
  }
  if (type === "button") {
    return { type, box: { ...box, w: 22, h: 8 }, text: { uz: "Buyurtma", ru: "", en: "" }, link: "/menu" };
  }
  if (type === "box") return { type, box, style: { tone: "brand", opacity: 20, rounded: true } };
  if (type === "divider") return { type, box: { ...box, h: 4 } };
  if (type === "image") return { type, box: { ...box, w: 45, h: 50 }, style: { rounded: true } };
  if (type === "carousel") return { type, box: { ...box, w: 60, h: 45 }, images: [] };
  if (type === "icon") return { type, box: { x: 10, y: 20, w: 6, h: 10, z: 1 }, icon: "star" };
  if (type === "badge") return { type, box: { ...box, w: 16, h: 6 }, text: { uz: "Yangi", ru: "", en: "" } };
  if (type === "quote") {
    return { type, box: { ...box, w: 45, h: 20 }, text: { uz: "Ajoyib taomlar", ru: "", en: "" }, subtext: { uz: "Mijoz", ru: "", en: "" } };
  }
  if (type === "rating") return { type, box: { ...box, w: 20, h: 8 }, value: 5 };
  if (type === "stat") {
    return { type, box: { ...box, w: 22, h: 16 }, text: { uz: "12", ru: "", en: "" }, subtext: { uz: "yillik tajriba", ru: "", en: "" } };
  }
  if (type === "list") {
    return { type, box: { ...box, w: 35, h: 25 }, text: { uz: "Birinchi qator\nIkkinchi qator", ru: "", en: "" } };
  }
  return { type, box: { ...box, w: 80, h: 60 } };
}

export default function DesignEditorPage() {
  const params = useParams<{ id: string }>();
  const tenantId = params.id;
  // The console's own language, the same cookie every other screen reads.
  const { lang } = useT();
  const d = editorDict(lang);
  const BAND_LABEL = BAND_LABELS[lang] ?? BAND_LABELS.uz;
  const ELEMENT_LABEL = ELEMENT_LABELS[lang] ?? ELEMENT_LABELS.uz;

  const [state, setState] = useState<DesignState | null>(null);
  const [sections, setSections] = useState<DesignSection[]>([]);
  const [css, setCss] = useState("");
  const [slug, setSlug] = useState("");
  const [pick, setPick] = useState<{ band: number; el: number | null }>({ band: 0, el: null });
  const [device, setDevice] = useState<"desktop" | "phone">("desktop");
  // ⚠️ The phone layout is edited separately, and the switch says which one the
  // boxes below are describing. Without this the same two numbers would mean two
  // different things depending on a toggle somewhere else on screen.
  const [editing, setEditing] = useState<"desktop" | "mobile">("desktop");
  const [previewUrl, setPreviewUrl] = useState("");
  const [pane, setPane] = useState<"canvas" | "site">("canvas");
  const [presets, setPresets] = useState<StylePreset[]>([]);
  // Which section of the left column the rail is showing. A single scrolling
  // column worked with five bands and stops working at fifteen: the inspector
  // ends up below the fold exactly when an element is selected.
  const [tab, setTab] = useState<
    "layers" | "element" | "styles" | "css" | "nav" | "templates"
  >("layers");
  // The site's navigation bar. ⚠️ Empty is "leave the header alone", never "a
  // bar with no links" — every tenant on the platform has this unset, and the
  // other reading would empty the header of every site at once.
  const [nav, setNav] = useState<NavLink[]>([]);
  // The palette this design is drawn with.
  //
  // ⚠️ **Nobody could set it, and that was two bugs meeting.** The console has
  // written `page_design.theme` since the constructor shipped and the site read
  // it from nowhere; publishing a design also sets `designLocked`, which
  // switches the owner's own theme editor off. So a customer with a drawn
  // design had no one at all who could change its colours. The site reads it
  // now (handlers/public.go, mergeTheme) and this is where it is chosen.
  const [theme, setTheme] = useState<SiteThemePatch>({});
  const [templates, setTemplates] = useState<DesignTemplate[]>([]);
  const [schema, setSchema] = useState<SectionDef[]>([]);
  // Which repeatable item inside the band is being edited. Separate from the
  // element selection: a band has blocks *or* freely drawn elements, never both.
  const [pickBlock, setPickBlock] = useState<number | null>(null);
  const [zoom, setZoom] = useState(0.7);
  // ⚠️ Editing **on the live preview**: handles drawn over the iframe, using the
  // geometry the site reports. Off by default — the preview is also the pane
  // somebody uses to simply look, and invisible drag targets over a page you are
  // reading is how an element gets moved by accident.
  const [liveEdit, setLiveEdit] = useState(true);
  // ⚠️ Undo is not a nicety in a direct-manipulation editor: the whole way of
  // working is "try it and see", and a drag that cannot be taken back makes
  // trying it expensive. History holds whole section lists — they are small, and
  // a diff-based history would be a second model to keep correct.
  const history = useRef<DesignSection[][]>([]);
  const future = useRef<DesignSection[][]>([]);
  const [busy, setBusy] = useState("");
  const [note, setNote] = useState("");
  const frame = useRef<HTMLIFrameElement>(null);
  // The live value, for callbacks that must not be re-created on every keystroke
  // (the drag handler subscribes to window events).
  const sectionsRef = useRef<DesignSection[]>([]);
  useEffect(() => {
    sectionsRef.current = sections;
  }, [sections]);
  // True between pointerdown and pointerup on the canvas. One history entry per
  // drag, not per pixel.
  const dragging = useRef(false);

  useEffect(() => {
    void (async () => {
      try {
        const [d, t] = await Promise.all([tenantDesign(tenantId), tenantApi(tenantId)]);
        setState(d);
        setSections(d.draft.sections ?? d.live.sections ?? []);
        setCss(d.draft.customCss ?? "");
        setPresets(d.draft.stylePresets ?? []);
        setNav(d.draft.nav ?? d.live.nav ?? []);
        setTheme(d.draft.theme ?? d.live.theme ?? {});
        try {
          const [g, sc] = await Promise.all([designTemplates(), designSchema()]);
          setTemplates(g.items);
          setSchema((sc.sections ?? []) as SectionDef[]);
        } catch {
          // A gallery that failed to load must not stop somebody editing what
          // they already have.
        }
        setSlug(t.tenant.slug);
      } catch (e) {
        setNote(e instanceof Error ? e.message : "yuklanmadi");
      }
    })();
  }, [tenantId]);

  const band = sections[pick.band];
  const bandDef = defFor(schema, band);
  const element =
    band?.canvas?.elements && pick.el != null ? band.canvas.elements[pick.el] : null;

  /** Records the current state so the next change can be undone.
   *
   *  ⚠️ Called on the **start** of a change rather than after it, and skipped
   *  while a drag is in flight (see `dragging`): a drag fires dozens of updates,
   *  and one undo per pixel is an undo stack nobody can walk back out of. */
  const remember = useCallback(() => {
    history.current = [...history.current.slice(-49), sectionsRef.current];
    future.current = [];
  }, []);

  const update = useCallback(
    (i: number, patch: Partial<DesignSection>) => {
      remember();
      setSections((prev) => prev.map((s, k) => (k === i ? { ...s, ...patch } : s)));
    },
    [remember],
  );

  function undo() {
    const prev = history.current.pop();
    if (!prev) return;
    future.current = [sectionsRef.current, ...future.current.slice(0, 49)];
    setSections(prev);
  }

  function redo() {
    const next = future.current.shift();
    if (!next) return;
    history.current = [...history.current, sectionsRef.current];
    setSections(next);
  }

  const updateElement = useCallback(
    (bandIdx: number, elIdx: number, patch: Partial<DesignElement>) => {
      remember();
      setSections((prev) =>
        prev.map((s, k) => {
          if (k !== bandIdx || !s.canvas?.elements) return s;
          const elements = s.canvas.elements.map((e, j) =>
            j === elIdx ? { ...e, ...patch } : e,
          );
          return { ...s, canvas: { ...s.canvas, elements } };
        }),
      );
    },
    [remember],
  );

  /** Moves the box the operator is currently editing — desktop or phone. */
  const moveBox = useCallback(
    (bandIdx: number, elIdx: number, patch: Partial<DesignBox>) => {
      if (!dragging.current) remember();
      setSections((prev) =>
        prev.map((s, k) => {
          if (k !== bandIdx || !s.canvas?.elements) return s;
          const elements = s.canvas.elements.map((e, j) => {
            if (j !== elIdx) return e;
            if (editing === "mobile") {
              // First edit of the phone layout starts from the desktop box, not
              // from zero: a designer adjusting one element does not want to place
              // it from scratch.
              return { ...e, mobile: { ...(e.mobile ?? e.box), ...patch } };
            }
            return { ...e, box: { ...e.box, ...patch } };
          });
          return { ...s, canvas: { ...s.canvas, elements } };
        }),
      );
    },
    [editing, remember],
  );

  async function save() {
    setBusy("save");
    setNote("");
    try {
      await saveTenantDesign(tenantId, sections, css, presets, nav, theme);
      setNote(d.savedDraft);
      await refreshPreview();
    } catch (e) {
      setNote(e instanceof Error ? e.message : "saqlanmadi");
    } finally {
      setBusy("");
    }
  }

  /** Saves and lets the real page redraw. Called when a drag on the live preview
   *  finishes — never during one: a save per pointer move is a page render per
   *  pixel on somebody's live site. */
  async function commitLive() {
    try {
      await saveTenantDesign(tenantId, sectionsRef.current, css, presets, nav, theme);
      // ⚠️ The token is reused rather than minted again. A new token per drag would
      // leave a trail of live preview links, each valid for two hours.
      if (previewUrl) {
        const base = previewUrl.split("&_=")[0];
        setPreviewUrl(`${base}&_=${Date.now()}`);
      }
    } catch (e) {
      setNote(e instanceof Error ? e.message : "saqlanmadi");
    }
  }

  async function refreshPreview() {
    try {
      const res = await previewTenantDesign(tenantId);
      // Cache-busted: the tenant's pages sit behind a 30-second edge cache, and a
      // preview that shows the previous draft is worse than no preview — it looks
      // like the change did not save.
      setPreviewUrl(`${res.url}&_=${Date.now()}`);
    } catch (e) {
      setNote(e instanceof Error ? e.message : "ko'rinish olinmadi");
    }
  }

  async function publish() {
    setBusy("publish");
    try {
      await saveTenantDesign(tenantId, sections, css, presets, nav, theme);
      const res = await publishTenantDesign(tenantId);
      setNote(res.note);
      setState(await tenantDesign(tenantId));
    } catch (e) {
      setNote(e instanceof Error ? e.message : "chop etilmadi");
    } finally {
      setBusy("");
    }
  }

  async function revert() {
    setBusy("revert");
    try {
      await revertTenantDesign(tenantId);
      setNote(d.reverted);
      setState(await tenantDesign(tenantId));
    } catch (e) {
      setNote(e instanceof Error ? e.message : "qaytarilmadi");
    } finally {
      setBusy("");
    }
  }

  const previewWidth = device === "phone" ? 390 : 1280;

  const frameStyle = useMemo(
    () => ({
      width: previewWidth,
      // Scaled to fit rather than shrunk: a preview at 60% still answers "does the
      // composition hold", and one squeezed into the pane answers nothing.
      transform: device === "phone" ? "none" : "scale(0.62)",
      transformOrigin: "top left",
      height: device === "phone" ? 780 : 1400,
    }),
    [device, previewWidth],
  );

  return (
    // Fills what the console header leaves, and scrolls inside its own panes: a
    // page-level scrollbar here would move the canvas out from under the cursor.
    <div className="flex h-[calc(100vh-4rem)] flex-col">
      <header className="flex flex-wrap items-center gap-3 border-b border-line px-4 py-3">
        <Link href={`/console/tenants/${tenantId}`} className="text-sm text-ink-soft hover:text-ink">
          ← {slug || "mijoz"}
        </Link>
        <h1 className="text-sm font-bold text-ink">{d.title}</h1>
        {state?.published && (
          <span className="rounded-full bg-signal-500/15 px-2 py-0.5 text-[11px] font-semibold text-signal-600">
            {d.live}
          </span>
        )}
        <div className="ml-auto flex flex-wrap items-center gap-2">
          <div className="flex overflow-hidden rounded-xl border border-line text-xs">
            {(["desktop", "phone"] as const).map((dev) => (
              <button
                key={dev}
                type="button"
                onClick={() => setDevice(dev)}
                className={`px-3 py-1.5 font-semibold ${
                  device === dev ? "bg-raised text-ink" : "text-ink-soft"
                }`}
              >
                {dev === "desktop" ? d.desktop : d.phone}
              </button>
            ))}
          </div>
          {/* The canvas width, stated. ⚠️ Two devices rather than three: the site
              has one breakpoint that matters (`lg`), so a tablet button would
              imply a third layout that does not exist and cannot be drawn. The
              number says which width is on screen, which is the part a tablet
              button was standing in for. */}
          <span className="rounded-xl border border-line px-2.5 py-1.5 text-[11px] tabular-nums text-ink-muted">
            {device === "phone" ? "390" : "1280"} px
          </span>
          {/* Opens the draft on the real domain, in a tab. The inline preview is
              for glancing; this is for handing to somebody, or for scrolling the
              whole page — which an iframe two thirds the height cannot do. */}
          <button
            type="button"
            onClick={async () => {
              await save();
              const url = previewUrl || "";
              if (url) window.open(url, "_blank", "noopener");
            }}
            title={d.viewHint}
            className="rounded-xl border border-line px-3 py-1.5 text-xs font-semibold text-ink-soft"
          >
            👁 {d.view}
          </button>
          <button
            type="button"
            onClick={() => void save()}
            disabled={busy !== ""}
            className="rounded-xl border border-line px-3 py-1.5 text-xs font-semibold text-ink disabled:opacity-40"
          >
            {busy === "save" ? d.saving : d.saveDraft}
          </button>
          <button
            type="button"
            onClick={() => void publish()}
            disabled={busy !== ""}
            className="rounded-xl bg-ink px-3 py-1.5 text-xs font-semibold text-surface disabled:opacity-40"
          >
            {busy === "publish" ? d.publishing : d.publish}
          </button>
          {state?.published && (
            <button
              type="button"
              onClick={() => void revert()}
              disabled={busy !== ""}
              className="rounded-xl border border-line px-3 py-1.5 text-xs font-semibold text-ink-soft disabled:opacity-40"
            >
              {d.revert}
            </button>
          )}
        </div>
      </header>

      {note && (
        <p className="border-b border-line bg-raised px-4 py-2 text-xs text-ink-soft">{note}</p>
      )}

      <div className="flex flex-1 flex-col lg:flex-row">
        {/* Left: the structure. */}
        {/* The rail: one icon per section of the left column.
            ⚠️ Not decoration. The column held everything at once — bands,
            settings, elements, inspector, CSS — and a design with fifteen bands
            pushed the inspector below the fold at the moment an element was
            selected. Four groups, one visible, and the canvas switches to the
            inspector when something is clicked. */}
        <nav className="flex shrink-0 gap-1 border-b border-line px-2 py-2 lg:flex-col lg:border-b-0 lg:border-r lg:px-2 lg:py-3">
          {(
            [
              { id: "layers", icon: "▤", title: d.tabLayers },
              { id: "element", icon: "◫", title: d.tabElement },
              { id: "styles", icon: "◐", title: d.tabStyles },
              { id: "css", icon: "{ }", title: d.tabCss },
              // ⚠️ Its own tab rather than a band, because it is not one: the
              // bar sits above every page of the site, not inside the home
              // page's list of bands. Put among the bands it would be a band
              // an operator could drag into the middle of the page.
              { id: "nav", icon: "☰", title: d.tabNav },
              { id: "templates", icon: "▢", title: d.tabTemplates },
            ] as const
          ).map((s) => (
            <button
              key={s.id}
              type="button"
              title={s.title}
              onClick={() => setTab(s.id)}
              className={`flex h-9 w-9 items-center justify-center rounded-xl text-sm transition ${
                tab === s.id ? "bg-raised text-ink" : "text-ink-muted hover:text-ink"
              }`}
            >
              {s.icon}
            </button>
          ))}
        </nav>

        <aside className="w-full shrink-0 space-y-3 overflow-auto border-b border-line p-4 lg:w-80 lg:border-b-0 lg:border-r">
          {tab === "layers" && (
          <BandList
            sections={sections}
            pick={pick}
            setPick={setPick}
            setSections={setSections}
          />

          )}

          {tab === "layers" && band && (
            <BandSettings
              band={band}
              index={pick.band}
              update={update}
              setSections={setSections}
              setPick={setPick}
              schema={schema}
              categories={state?.categories}
            />
          )}

          {/* Repeatable items: gallery photos, perk cards, slides.
              ⚠️ Blocks rather than numbered settings (`perk1Title`…): numbered
              fields fix the count, fill the panel with empty inputs, and cannot be
              reordered without retyping. */}
          {tab === "layers" && bandDef?.blocks?.length ? (
            <BlockList
              band={band}
              def={bandDef}
              index={pick.band}
              picked={pickBlock}
              setPicked={setPickBlock}
              update={update}
              lang={lang}
            />
          ) : null}

          {tab === "layers" && band?.canvas && (
            <ElementList
              band={band}
              bandIndex={pick.band}
              pick={pick}
              setPick={setPick}
              setSections={setSections}
            />
          )}

          {tab === "element" && !element && (
            <p className="text-xs text-ink-muted">
              {d.pickElement}
            </p>
          )}

          {tab === "element" && element && pick.el != null && (
            <ElementSettings
              el={element}
              editing={editing}
              setEditing={setEditing}
              onStyle={(patch) =>
                updateElement(pick.band, pick.el!, {
                  style: { ...(element.style ?? {}), ...patch },
                })
              }
              onElement={(patch) => updateElement(pick.band, pick.el!, patch)}
              onBox={(patch) => moveBox(pick.band, pick.el!, patch)}
            />
          )}

          {tab === "templates" && (
            <div className="space-y-2 rounded-2xl border border-line p-3">
              <p className="text-xs font-bold text-ink">{d.tabTemplates}</p>
              <p className="text-[11px] leading-relaxed text-ink-muted">{d.templatesHint}</p>
              <ul className="space-y-1.5">
                {templates.map((tpl) => (
                  <li key={tpl.id}>
                    <button
                      type="button"
                      onClick={() => {
                        // ⚠️ A copy, always. Applying must not link this customer's
                        // page to a gallery entry — an improved template would
                        // otherwise redraw sites that were already approved.
                        remember();
                        setSections(JSON.parse(JSON.stringify(tpl.sections)));
                        // ⚠️ **The palette and the bar come with it.** A
                        // template is a design; copying only the bands landed a
                        // layout drawn around a yellow panel and a lowercase
                        // catalogue bar as grey rectangles under the
                        // restaurant's own header, and left the operator
                        // retyping per customer what the gallery already knew.
                        // ⚠️ Only when the template carries them: one that
                        // chooses no colours must not wipe the ones this
                        // customer already has.
                        if (tpl.theme && Object.keys(tpl.theme).length > 0) {
                          setTheme(JSON.parse(JSON.stringify(tpl.theme)));
                        }
                        if (tpl.nav && tpl.nav.length > 0) {
                          setNav(JSON.parse(JSON.stringify(tpl.nav)));
                        }
                        setPick({ band: 0, el: null });
                        setTab("layers");
                        setNote(d.templateApplied(tpl.name));
                      }}
                      className="w-full rounded-xl border border-line px-3 py-2 text-left hover:border-signal-500"
                    >
                      <span className="block text-xs font-bold text-ink">{tpl.name}</span>
                      {tpl.note && (
                        <span className="mt-0.5 block text-[11px] leading-relaxed text-ink-muted">
                          {tpl.note}
                        </span>
                      )}
                      <span className="mt-1 block text-[11px] text-ink-muted">
                        {tpl.sections.length} band
                        {tpl.builtin ? "" : ` · ${tpl.createdBy ?? ""}`}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {tab === "styles" && (
            <ThemePanel theme={theme} setTheme={setTheme} d={d} />
          )}

          {tab === "styles" && (
            <PresetPanel
              presets={presets}
              setPresets={setPresets}
              current={element?.style ?? null}
              onApply={(style: StylePreset["style"]) => {
                if (pick.el == null) return;
                // ⚠️ A **copy**, not a reference. Editing this preset next month
                // must not repaint elements on a page a customer already approved
                // — the same rule the template gallery follows.
                updateElement(pick.band, pick.el, { style: { ...style } });
              }}
              canApply={pick.el != null}
            />
          )}

          {tab === "nav" && <NavEditor nav={nav} setNav={setNav} d={d} />}

          {tab === "css" && (
          <>
          {/* ⚠️ The escape hatch, and the reason the constructor can answer a brief
              it was not designed for. Refused outright by the backend if it
              contains anything that could close a `<style>` element — so a
              rejected stylesheet comes back empty rather than half-applied. */}
          <div className="rounded-2xl border border-line p-3">
            <p className="text-xs font-bold text-ink">{d.cssTitle}</p>
            <p className="mt-1 text-[11px] leading-relaxed text-ink-muted">
              {d.cssHint}
            </p>
            <textarea
              value={css}
              onChange={(e) => setCss(e.target.value)}
              rows={6}
              spellCheck={false}
              className="mt-2 w-full rounded-xl border border-line bg-surface p-2 font-mono text-[11px] text-ink"
              placeholder=".hero h1 { letter-spacing: -0.02em }"
            />
          </div>
          </>
          )}
        </aside>

        {/* Middle and right in one pane, switched rather than side by side.
            ⚠️ Two surfaces of the same page at once is a screen where neither is
            big enough to work on, and they answer different questions anyway: the
            canvas is "where is this element", the site is "does it look like the
            picture". Somebody drags on one and checks on the other. */}
        <section className="flex min-w-0 flex-1 flex-col bg-raised">
          <div className="flex flex-wrap items-center gap-2 border-b border-line px-4 py-2">
            <div className="flex overflow-hidden rounded-xl border border-line text-xs">
              {(["canvas", "site"] as const).map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => {
                    setPane(p);
                    if (p === "site" && !previewUrl) void refreshPreview();
                  }}
                  className={`px-3 py-1.5 font-semibold ${
                    pane === p ? "bg-surface text-ink" : "text-ink-soft"
                  }`}
                >
                  {p === "canvas" ? d.canvas : d.site}
                </button>
              ))}
            </div>

            {pane === "canvas" && (
              <>
                <div className="flex items-center gap-1 text-xs text-ink-soft">
                  <button type="button" onClick={() => setZoom((z) => Math.max(0.3, +(z - 0.1).toFixed(2)))} className="rounded-lg border border-line px-2 py-1">−</button>
                  <span className="w-10 text-center tabular-nums">{Math.round(zoom * 100)}%</span>
                  <button type="button" onClick={() => setZoom((z) => Math.min(1, +(z + 0.1).toFixed(2)))} className="rounded-lg border border-line px-2 py-1">+</button>
                </div>
                {/* ⚠️ Undo belongs beside the canvas, not in a menu: the way of
                    working here is "drag it and see", and that only works if
                    taking it back is as cheap as trying it. */}
                <button type="button" onClick={undo} className="rounded-lg border border-line px-2 py-1 text-xs text-ink-soft">↶ {d.undo}</button>
                <button type="button" onClick={redo} className="rounded-lg border border-line px-2 py-1 text-xs text-ink-soft">↷ {d.redo}</button>
                <span className="text-[11px] text-ink-muted">
                  Sudrab ko'chiring · burchaklardan o'lchang · strelkalar bilan
                  suring (Shift — 5%)
                </span>
              </>
            )}
            {pane === "site" && (
              <>
                <button type="button" onClick={() => void refreshPreview()} className="rounded-lg border border-line px-2 py-1 text-xs text-ink-soft">
                  {d.refresh}
                </button>
                <Seg
                  value={liveEdit ? "on" : "off"}
                  options={[
                    { v: "on", label: d.liveEditOn },
                    { v: "off", label: d.liveEditOff },
                  ]}
                  onChange={(v) => setLiveEdit(v === "on")}
                />
                <span className="text-[11px] text-ink-muted">{d.liveEditHint}</span>
              </>
            )}
          </div>

          <div className="flex-1 overflow-auto p-4">
            {pane === "canvas" ? (
              band?.canvas ? (
                <div
                  onPointerDown={() => {
                    dragging.current = true;
                  }}
                  onPointerUp={() => {
                    dragging.current = false;
                  }}
                >
                  <EditorCanvas
                    band={band}
                    device={device}
                    editing={editing}
                    zoom={zoom}
                    selected={pick.el}
                    onSelect={(el) => {
                      setPick({ band: pick.band, el });
                      // ⚠️ Selecting on the canvas switches the left column to the
                      // inspector. Without it the settings for the thing just
                      // clicked sit behind another click, which is the one moment
                      // they are certainly wanted.
                      if (el != null) setTab("element");
                    }}
                    onBox={(i, patch) => moveBox(pick.band, i, patch)}
                  />
                </div>
              ) : (
                // ⚠️ Only freely drawn bands have a canvas. Saying so beats
                // showing an empty rectangle, which reads as a broken editor.
                <p className="mx-auto max-w-sm pt-16 text-center text-sm text-ink-muted">
                  {d.fixedBand}
                </p>
              )
            ) : previewUrl ? (
              <div className="mx-auto overflow-hidden rounded-2xl border border-line bg-surface" style={{ width: device === "phone" ? 390 : "100%" }}>
                <iframe
                  ref={frame}
                  src={previewUrl}
                  title={d.site}
                  style={frameStyle}
                  className="block border-0"
                />
                {liveEdit && band?.canvas && (
                  <PreviewOverlay
                    frame={frame}
                    zoom={device === "phone" ? 1 : 0.62}
                    activeBand={pick.band}
                    selected={pick.el}
                    onSelect={(el) => {
                      setPick({ band: pick.band, el });
                      if (el != null) setTab("element");
                    }}
                    onPick={(bandIdx: number, el: number | null) => {
                      setPick({ band: bandIdx, el });
                      setTab(el != null ? "element" : "layers");
                    }}
                    onBox={(i, patch) => moveBox(pick.band, i, patch)}
                    onCommit={() => void commitLive()}
                    boxOf={(i) => {
                      const el = band?.canvas?.elements?.[i];
                      if (!el) return null;
                      return editing === "mobile" ? (el.mobile ?? el.box) : el.box;
                    }}
                  />
                )}
              </div>
            ) : (
              <div className="flex h-full items-center justify-center">
                <button
                  type="button"
                  onClick={() => void refreshPreview()}
                  className="rounded-xl bg-ink px-4 py-2 text-sm font-semibold text-surface"
                >
                  {d.openPreview}
                </button>
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}

/** Bands the schema knows about, plus the labels for them. Built from the schema so
 *  adding a section server-side puts it in this list with no editor change — which
 *  is the point of the schema. */
function BandList({
  sections,
  pick,
  setPick,
  setSections,
}: {
  sections: DesignSection[];
  pick: { band: number; el: number | null };
  setPick: (p: { band: number; el: number | null }) => void;
  setSections: React.Dispatch<React.SetStateAction<DesignSection[]>>;
}) {
  const { lang } = useT();
  const d = editorDict(lang);
  const BAND_LABEL = BAND_LABELS[lang] ?? BAND_LABELS.uz;
  const ELEMENT_LABEL = ELEMENT_LABELS[lang] ?? ELEMENT_LABELS.uz;
  function add(type: string) {
    // ⚠️ A new band starts with sensible text rather than empty.
    // An empty section renders nothing, and a band that appears in the list while
    // the page does not change reads as the button not working.
    const seeded: Record<string, Record<string, unknown>> = {
      hero: { heading: { uz: "Sarlavha", ru: "Заголовок", en: "Heading" }, height: 80, overlay: 40 },
      "rich-text": { heading: { uz: "Sarlavha", ru: "Заголовок", en: "Heading" }, align: "center", tone: "surface" },
      "image-text": { heading: { uz: "Sarlavha", ru: "Заголовок", en: "Heading" }, round: "lg" },
      banner: { heading: { uz: "Aksiya", ru: "Акция", en: "Offer" }, tone: "charcoal", overlay: 55 },
      "menu-grid": { popularOnly: true, limit: 8 },
    };
    const section: DesignSection = {
      type,
      variant: VARIANTS[type]?.[0] ?? "",
      span: 12,
      settings: seeded[type],
      ...(type === "canvas" || type === "popup"
        ? { canvas: { height: 60, elements: [] } }
        : {}),
    };
    setSections((prev) => [...prev, section]);
    setPick({ band: sections.length, el: null });
  }

  function move(i: number, dir: -1 | 1) {
    setSections((prev) => {
      const next = [...prev];
      const j = i + dir;
      if (j < 0 || j >= next.length) return prev;
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
    setPick({ band: i + dir, el: null });
  }

  return (
    <div className="rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{d.bands}</p>
      {/* ⚠️ Dragged, not nudged with arrows.
          Moving a band from the bottom of a fifteen-band page to the top took
          fourteen presses, each one re-rendering the list under the cursor. HTML5
          drag-and-drop rather than a library: the whole gesture is "pick up a row,
          drop it on another", and a dependency for that is a dependency to keep in
          step with React for years. */}
      <ul className="mt-2 space-y-1">
        {sections.map((s, i) => (
          <li
            key={i}
            draggable
            onDragStart={(e) => {
              e.dataTransfer.setData("text/plain", String(i));
              e.dataTransfer.effectAllowed = "move";
            }}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              const from = Number(e.dataTransfer.getData("text/plain"));
              if (Number.isNaN(from) || from === i) return;
              setSections((prev) => {
                const next = [...prev];
                const [moved] = next.splice(from, 1);
                next.splice(i, 0, moved);
                return next;
              });
              // Follow the band that moved: losing the selection mid-reorder means
              // finding it again in a list that just changed shape.
              setPick({ band: i, el: null });
            }}
            className="flex cursor-grab items-center gap-1 active:cursor-grabbing"
          >
            <span className="select-none px-1 text-ink-muted" aria-hidden>⠿</span>
            <button
              type="button"
              onClick={() => setPick({ band: i, el: null })}
              className={`flex-1 rounded-lg px-2 py-1.5 text-left text-xs font-semibold ${
                pick.band === i ? "bg-raised text-ink" : "text-ink-soft"
              } ${s.hidden ? "line-through opacity-50" : ""}`}
            >
              {BAND_LABEL[s.type] ?? s.type}
              {s.canvas?.elements?.length ? ` · ${s.canvas.elements.length}` : ""}
            </button>
            {/* Kept beside the drag handle: a list that can only be reordered by
                dragging cannot be reordered with a keyboard, and one of the two
                people who will use this screen works that way. */}
            <button type="button" onClick={() => move(i, -1)} className="px-1 text-ink-muted" aria-label="↑">↑</button>
            <button type="button" onClick={() => move(i, 1)} className="px-1 text-ink-muted" aria-label="↓">↓</button>
          </li>
        ))}
      </ul>
      <div className="mt-2 flex flex-wrap gap-1">
        {["hero", "rich-text", "image-text", "menu-grid", "search", "banner", "gallery", "hours-address", "canvas", "popup", "navbar", "footer", "categories", "perks", "about", "cta"].map((type) => (
          <button
            key={type}
            type="button"
            onClick={() => add(type)}
            className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft hover:text-ink"
          >
            + {BAND_LABEL[type]}
          </button>
        ))}
      </div>
    </div>
  );
}

function BandSettings({
  band,
  index,
  update,
  setSections,
  setPick,
  schema,
  categories,
}: {
  band: DesignSection;
  index: number;
  update: (i: number, patch: Partial<DesignSection>) => void;
  setSections: React.Dispatch<React.SetStateAction<DesignSection[]>>;
  setPick: (p: { band: number; el: number | null }) => void;
  schema: SectionDef[];
  /** The restaurant's own categories, for the settings that pick among them. */
  categories?: { id: string; name: string }[];
}) {
  const { lang } = useT();
  const d = editorDict(lang);
  const BAND_LABEL = BAND_LABELS[lang] ?? BAND_LABELS.uz;
  const ELEMENT_LABEL = ELEMENT_LABELS[lang] ?? ELEMENT_LABELS.uz;
  const canvas = band.canvas;
  const def = defFor(schema, band);
  return (
    <div className="space-y-2 rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{BAND_LABEL[band.type] ?? band.type}</p>

      {/* ⚠️ Declared settings first, and the old fixed controls after.
          A section that the schema describes is edited entirely through what it
          declared; the variant/span/tone rows below stay for the bands that predate
          the schema, so no existing design loses its controls mid-migration. */}
      {def && def.settings.length > 0 && (
        <SchemaSettings
          def={def}
          values={band.settings ?? {}}
          lang={lang}
          categories={categories}
          onChange={(key, value) =>
            update(index, { settings: { ...(band.settings ?? {}), [key]: value } })
          }
        />
      )}

      {!def?.settings.length && (
      <Row label={d.variant}>
        <select
          value={band.variant ?? ""}
          onChange={(e) => update(index, { variant: e.target.value })}
          className="select"
        >
          {(VARIANTS[band.type] ?? [""]).map((v) => (
            <option key={v} value={v}>{v}</option>
          ))}
        </select>
      </Row>
      )}

      {!canvas && (
        <Row label={d.span}>
          <input
            type="number"
            min={1}
            max={12}
            value={band.span}
            onChange={(e) => update(index, { span: Number(e.target.value) })}
            className="input"
          />
        </Row>
      )}

      <Row label={d.background}>
        <select
          value={band.style?.tone ?? ""}
          onChange={(e) => update(index, { style: { ...(band.style ?? {}), tone: e.target.value } })}
          className="select"
        >
          {TONES.map((v) => <option key={v} value={v}>{v || "sahifa foni"}</option>)}
        </select>
      </Row>

      {/* ⚠️ **On every band, including the navbar.** The bar reads its width
          from here too (lib/siteChrome.ts), which is what lets a shop's header
          run edge to edge while the bands under it stay in the column. */}
      <Row label={d.width}>
        <select
          value={band.style?.width ?? ""}
          onChange={(e) =>
            update(index, { style: { ...(band.style ?? {}), width: e.target.value } })
          }
          className="select"
        >
          {WIDTHS.map((v) => (
            <option key={v} value={v}>
              {v === "" ? d.widthColumn : v === "wide" ? d.widthWide : d.widthFull}
            </option>
          ))}
        </select>
      </Row>

      {canvas && (
        <>
          <Row label={d.height}>
            <input
              type="number"
              min={10}
              max={200}
              value={canvas.height ?? 60}
              onChange={(e) =>
                update(index, { canvas: { ...canvas, height: Number(e.target.value) } })
              }
              className="input"
            />
          </Row>
          {/* ⚠️ A separate phone height, because the desktop proportions rarely
              survive: a 90vh hero on a phone pushes everything below three
              scrolls of empty space. 0 means "same as desktop". */}
          <Row label={d.heightMobile}>
            <input
              type="number"
              min={0}
              max={200}
              value={canvas.heightMobile ?? 0}
              onChange={(e) =>
                update(index, { canvas: { ...canvas, heightMobile: Number(e.target.value) } })
              }
              className="input"
            />
          </Row>
        </>
      )}

      <div className="flex flex-wrap gap-2 pt-1">
        <button
          type="button"
          onClick={() => update(index, { hidden: !band.hidden })}
          className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft"
        >
          {band.hidden ? d.show : d.hide}
        </button>
        {/* Hide rather than delete is the default above; delete is here because an
            experiment eventually has to be thrown away. */}
        <button
          type="button"
          onClick={() => {
            setSections((prev) => prev.filter((_, k) => k !== index));
            setPick({ band: 0, el: null });
          }}
          className="rounded-lg border border-line px-2 py-1 text-[11px] text-hot-600"
        >
          {d.remove}
        </button>
      </div>
    </div>
  );
}

function ElementList({
  band,
  bandIndex,
  pick,
  setPick,
  setSections,
}: {
  band: DesignSection;
  bandIndex: number;
  pick: { band: number; el: number | null };
  setPick: (p: { band: number; el: number | null }) => void;
  setSections: React.Dispatch<React.SetStateAction<DesignSection[]>>;
}) {
  const { lang } = useT();
  const d = editorDict(lang);
  const BAND_LABEL = BAND_LABELS[lang] ?? BAND_LABELS.uz;
  const ELEMENT_LABEL = ELEMENT_LABELS[lang] ?? ELEMENT_LABELS.uz;
  const elements = band.canvas?.elements ?? [];

  function add(type: string) {
    setSections((prev) =>
      prev.map((s, k) =>
        k === bandIndex && s.canvas
          ? { ...s, canvas: { ...s.canvas, elements: [...(s.canvas.elements ?? []), newElement(type)] } }
          : s,
      ),
    );
    setPick({ band: bandIndex, el: elements.length });
  }

  return (
    <div className="rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{d.elements}</p>
      <ul className="mt-2 space-y-1">
        {elements.map((e, i) => (
          <li key={i} className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => setPick({ band: bandIndex, el: i })}
              className={`flex-1 rounded-lg px-2 py-1.5 text-left text-xs ${
                pick.el === i ? "bg-raised font-semibold text-ink" : "text-ink-soft"
              }`}
            >
              {ELEMENT_LABEL[e.type] ?? e.type}
              {e.text?.uz ? ` · ${e.text.uz.slice(0, 18)}` : ""}
            </button>
            <button
              type="button"
              onClick={() =>
                setSections((prev) =>
                  prev.map((s, k) =>
                    k === bandIndex && s.canvas
                      ? {
                          ...s,
                          canvas: {
                            ...s.canvas,
                            elements: (s.canvas.elements ?? []).filter((_, j) => j !== i),
                          },
                        }
                      : s,
                  ),
                )
              }
              className="px-1 text-hot-600"
              aria-label="o'chirish"
            >
              ×
            </button>
          </li>
        ))}
      </ul>
      <div className="mt-2 flex flex-wrap gap-1">
        {Object.keys(ELEMENT_LABEL).map((type) => (
          <button
            key={type}
            type="button"
            onClick={() => add(type)}
            className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft hover:text-ink"
          >
            + {ELEMENT_LABEL[type]}
          </button>
        ))}
      </div>
    </div>
  );
}

function ElementSettings({
  el,
  editing,
  setEditing,
  onStyle,
  onElement,
  onBox,
}: {
  el: DesignElement;
  editing: "desktop" | "mobile";
  setEditing: (v: "desktop" | "mobile") => void;
  onStyle: (patch: Record<string, unknown>) => void;
  onElement: (patch: Partial<DesignElement>) => void;
  onBox: (patch: Partial<DesignBox>) => void;
}) {
  const { lang } = useT();
  const d = editorDict(lang);
  const BAND_LABEL = BAND_LABELS[lang] ?? BAND_LABELS.uz;
  const ELEMENT_LABEL = ELEMENT_LABELS[lang] ?? ELEMENT_LABELS.uz;
  const box = editing === "mobile" ? (el.mobile ?? el.box) : el.box;
  // Everything that carries typed words. ⚠️ Listed rather than defaulted: an
  // element that shows text but is missing from here has a text field nobody can
  // reach, and it looks like the element is broken.
  const isText = ["text", "button", "badge", "quote", "stat", "list"].includes(el.type);

  return (
    <div className="space-y-2 rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{ELEMENT_LABEL[el.type] ?? el.type}</p>

      {/* ⚠️ Which layout the numbers below describe. Free placement has exactly one
          failure mode — a composition nobody checked on a phone — and it is
          prevented by making the phone layout a thing you switch to and edit,
          rather than something you hope works. */}
      <div className="flex overflow-hidden rounded-xl border border-line text-[11px]">
        {(["desktop", "mobile"] as const).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => setEditing(m)}
            className={`flex-1 px-2 py-1.5 font-semibold ${
              editing === m ? "bg-raised text-ink" : "text-ink-soft"
            }`}
          >
            {m === "desktop" ? d.layoutDesktop : d.layoutMobile}
          </button>
        ))}
      </div>
      {editing === "mobile" && !el.mobile && (
        <p className="text-[11px] text-ink-muted">
          {d.mobileNotDrawn}
        </p>
      )}

      <div className="grid grid-cols-2 gap-2">
        {(["x", "y", "w", "h"] as const).map((k) => (
          <Row key={k} label={`${k.toUpperCase()} (%)`}>
            <input
              type="number"
              value={box[k]}
              onChange={(e) => onBox({ [k]: Number(e.target.value) } as Partial<DesignBox>)}
              className="input"
            />
          </Row>
        ))}
        <Row label={d.layer}>
          <input
            type="number"
            min={0}
            max={20}
            value={box.z ?? 0}
            onChange={(e) => onBox({ z: Number(e.target.value) })}
            className="input"
          />
        </Row>
      </div>

      {isText && (
        <>
          {/* Type style, grouped and named — the reference's "Text Style" block.
              Weight and alignment are segmented because there are three and four
              of them: a dropdown for four options hides them behind a click. */}
          <p className="pt-1 text-[11px] font-bold uppercase tracking-wider text-ink-muted">
            {d.textStyle}
          </p>
          <div className="grid grid-cols-2 gap-2">
            <Row label={d.font}>
              <Seg
                value={(el.style?.font ?? "") as string}
                options={[
                  { v: "", label: d.fontBase },
                  { v: "display", label: d.fontDisplay },
                ]}
                onChange={(v) => onStyle({ font: v })}
              />
            </Row>
            <Row label={d.weight}>
              <Seg
                value={(el.style?.weight ?? "") as string}
                options={[
                  { v: "", label: d.weightNormal },
                  { v: "bold", label: d.weightBold },
                  { v: "black", label: d.weightBlack },
                ]}
                onChange={(v) => onStyle({ weight: v })}
              />
            </Row>
          </div>
          <Row label={d.align}>
            <Seg
              value={(el.style?.align ?? "") as string}
              options={[
                { v: "", label: "◧ " + d.alignLeft },
                { v: "center", label: "▣ " + d.alignCenter },
              ]}
              onChange={(v) => onStyle({ align: v })}
            />
          </Row>
          <Row label={d.color}>
            <Swatches
              value={el.style?.color ?? ""}
              options={COLORS}
              preview={COLOR_PREVIEW}
              onChange={(v) => onStyle({ color: v })}
            />
          </Row>
          <Row label="Matn (uz)">
            <input
              value={el.text?.uz ?? ""}
              onChange={(e) => onElement({ text: { uz: e.target.value, ru: el.text?.ru ?? "", en: el.text?.en ?? "" } })}
              className="input"
            />
          </Row>
          <Row label="ru">
            <input
              value={el.text?.ru ?? ""}
              onChange={(e) => onElement({ text: { uz: el.text?.uz ?? "", ru: e.target.value, en: el.text?.en ?? "" } })}
              className="input"
            />
          </Row>
          <Row label="en">
            <input
              value={el.text?.en ?? ""}
              onChange={(e) => onElement({ text: { uz: el.text?.uz ?? "", ru: el.text?.ru ?? "", en: e.target.value } })}
              className="input"
            />
          </Row>
          {/* Size as a step on the type scale, with buttons: the value is nudged
              far more often than it is typed, and the steps are what keep a
              headline proportional when the theme's root size changes. */}
          <Row label={d.size}>
            <div className="flex items-center gap-1">
              <button type="button" onClick={() => onStyle({ size: Math.max(-2, (el.style?.size ?? 0) - 1) })} className="rounded-lg border border-line px-2 py-1 text-xs">−</button>
              <span className="w-8 text-center text-xs tabular-nums text-ink">{el.style?.size ?? 0}</span>
              <button type="button" onClick={() => onStyle({ size: Math.min(8, (el.style?.size ?? 0) + 1) })} className="rounded-lg border border-line px-2 py-1 text-xs">+</button>
            </div>
          </Row>
        </>
      )}

      {el.type === "button" && (
        <>
          {/* ⚠️ **A box, with the site's own pages as one-tap chips beside it.**
              It was a `<select>` of nine paths, which is right while a button
              can only go where every restaurant goes — and wrong for a shop,
              whose buttons go to the section it is divided into
              ("/menu?cat=ayollar"), to its Telegram channel, to the lookbook on
              YouTube. The server checks the scheme, not the destination. */}
          <Row label={d.link}>
            <input
              value={el.link ?? ""}
              onChange={(e) => onElement({ link: e.target.value })}
              placeholder="/menu?cat=ayollar"
              spellCheck={false}
              className="input font-mono text-[11px]"
            />
          </Row>
          <div className="flex flex-wrap gap-1">
            {LINKS.filter(Boolean).map((l) => (
              <button
                key={l}
                type="button"
                onClick={() => onElement({ link: l })}
                className="rounded-full border border-line px-2 py-0.5 font-mono text-[10px] text-ink-muted hover:border-signal-500 hover:text-ink"
              >
                {l}
              </button>
            ))}
          </div>
          {/* Only on an address that actually leaves the site: a new tab on a
              path of ours lands the guest in a second copy of the shop with an
              empty basket, and the server clears it anyway. */}
          {(el.link ?? "").startsWith("http") && (
            <Toggle
              value={!!el.linkExternal}
              label={d.linkExternal}
              onChange={(v) => onElement({ linkExternal: v })}
            />
          )}
        </>
      )}

      {(el.type === "quote" || el.type === "stat") && (
        <Row label={d.second}>
          <input
            value={el.subtext?.uz ?? ""}
            onChange={(e) =>
              onElement({
                subtext: { uz: e.target.value, ru: el.subtext?.ru ?? "", en: el.subtext?.en ?? "" },
              })
            }
            className="input"
          />
        </Row>
      )}

      {el.type === "icon" && (
        <Row label={d.icon}>
          <select
            value={el.icon ?? "star"}
            onChange={(e) => onElement({ icon: e.target.value })}
            className="select"
          >
            {ICONS.filter(Boolean).map((i) => (
              <option key={i} value={i}>{i}</option>
            ))}
          </select>
        </Row>
      )}

      {el.type === "rating" && (
        <Row label={d.stars}>
          <Seg
            value={String(el.value ?? 5)}
            options={["1", "2", "3", "4", "5"].map((v) => ({ v, label: v }))}
            onChange={(v) => onElement({ value: Number(v) })}
          />
        </Row>
      )}

      {el.type === "carousel" && (
        <Row label={d.images}>
          <textarea
            rows={4}
            value={(el.images ?? []).join("\n")}
            onChange={(e) =>
              onElement({
                images: e.target.value.split("\n").map((l) => l.trim()).filter(Boolean),
              })
            }
            className="input font-mono text-[11px]"
            placeholder="/uploads/a.jpg"
          />
        </Row>
      )}

      {el.type === "image" && (
        <Row label={d.image}>
          <input
            value={el.image ?? ""}
            onChange={(e) => onElement({ image: e.target.value })}
            placeholder="/uploads/abc.jpg"
            className="input"
          />
        </Row>
      )}

      {(el.type === "box" || el.type === "image") && (
        <Row label={d.opacity}>
          <input
            type="number"
            min={0}
            max={100}
            value={el.style?.opacity ?? 100}
            onChange={(e) => onStyle({ opacity: Number(e.target.value) })}
            className="input"
          />
        </Row>
      )}

      {(el.type === "box" || isText) && (
        <Row label={d.background}>
          <Swatches
            value={el.style?.tone ?? ""}
            options={TONES}
            preview={TONE_PREVIEW}
            onChange={(v) => onStyle({ tone: v })}
          />
        </Row>
      )}

      <p className="pt-1 text-[11px] font-bold uppercase tracking-wider text-ink-muted">
        {d.look}
      </p>

      {/* ⚠️ **The two shapes a shop reference is actually built from.** A panel
          rounded on the edge that faces the page, with a category list on it,
          and a rail of words turned a quarter turn down the side. Neither was
          reachable: the editor could round all four corners or none, and could
          not turn anything at all — so the way to approximate the first was to
          push the box off the edge of the band and hope. */}
      <Row label={d.radius}>
        <select
          value={el.style?.radius ?? ""}
          onChange={(e) => onStyle({ radius: e.target.value })}
          className="select"
        >
          {RADII.map((v) => (
            <option key={v} value={v}>{v || d.radiusNone}</option>
          ))}
        </select>
      </Row>
      {el.style?.radius && (
        <Row label={d.corner}>
          <select
            value={el.style?.corner ?? ""}
            onChange={(e) => onStyle({ corner: e.target.value })}
            className="select"
          >
            {CORNERS.map((v) => (
              <option key={v} value={v}>{v || d.cornerAll}</option>
            ))}
          </select>
        </Row>
      )}
      <Row label={d.rotate}>
        <Seg
          value={el.style?.rotate ?? ""}
          options={ROTATIONS.map((v) => ({
            v,
            label: v === "" ? d.rotateNone : v === "-90" ? "↑" : "↓",
          }))}
          onChange={(v) => onStyle({ rotate: v })}
        />
      </Row>

      <Toggle
        value={!!el.style?.rounded}
        label={d.rounded}
        onChange={(v) => onStyle({ rounded: v })}
      />
      <Toggle
        value={!!el.style?.shadow}
        label={d.shadow}
        onChange={(v) => onStyle({ shadow: v })}
      />

      <div className="flex flex-wrap gap-2 pt-1">
        <button
          type="button"
          onClick={() => onElement({ hiddenMobile: !el.hiddenMobile })}
          className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft"
        >
          {el.hiddenMobile ? d.showMobile : d.hideMobile}
        </button>
        {el.mobile && (
          <button
            type="button"
            onClick={() => onElement({ mobile: null })}
            className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft"
          >
            {d.clearMobile}
          </button>
        )}
      </div>
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="text-[11px] font-semibold text-ink-muted">{label}</span>
      <div className="mt-0.5">{children}</div>
    </label>
  );
}

/** A segmented control: the reference's align and padding rows, and the right
 *  shape for a small closed set. A `<select>` hides the options behind a click and
 *  makes "which of these three" a two-step question. */
function Seg<T extends string>({
  value,
  options,
  onChange,
}: {
  value: T;
  options: { v: T; label: string; title?: string }[];
  onChange: (v: T) => void;
}) {
  return (
    <div className="flex overflow-hidden rounded-xl border border-line">
      {options.map((o) => (
        <button
          key={o.v}
          type="button"
          title={o.title}
          onClick={() => onChange(o.v)}
          className={`flex-1 px-2 py-1.5 text-[11px] font-semibold transition ${
            value === o.v ? "bg-raised text-ink" : "text-ink-soft hover:text-ink"
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

/** Colour as swatches.
 *
 *  ⚠️ **Swatches of the design system's tokens, not a hex picker**, and the
 *  difference is not cosmetic. A hex colour is frozen: it stays that value in dark
 *  mode, where a "#1a1a1a" headline disappears, and it stops following the accent
 *  the restaurant chose — so a design drawn in one palette silently contradicts the
 *  brand it was drawn for. The tokens are the same names the rest of the site
 *  paints with, so a band drawn today still looks right after the owner changes
 *  their accent tomorrow.
 *
 *  Shown as colour, though: reading "charcoal" and picturing it is the thing a
 *  swatch removes. */
function Swatches({
  value,
  options,
  onChange,
  preview,
}: {
  value: string;
  options: string[];
  onChange: (v: string) => void;
  preview: Record<string, string>;
}) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {options.map((o) => (
        <button
          key={o}
          type="button"
          title={o || "standart"}
          onClick={() => onChange(o)}
          style={{ background: preview[o] ?? "transparent" }}
          className={`h-6 w-6 rounded-md border ${
            value === o ? "border-signal-500 ring-2 ring-signal-500/40" : "border-line-strong"
          }`}
        >
          {/* The empty token is "inherit", which has no colour to show. */}
          {o === "" && <span className="text-[9px] text-ink-muted">—</span>}
        </button>
      ))}
    </div>
  );
}

/** A switch, for the things that are genuinely on or off (shadow, rounding). */
function Toggle({
  value,
  label,
  onChange,
}: {
  value: boolean;
  label: string;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onChange(!value)}
      className="flex w-full items-center justify-between gap-2 py-1"
    >
      <span className="text-[11px] font-semibold text-ink-muted">{label}</span>
      <span
        className={`relative h-5 w-9 rounded-full transition ${
          value ? "bg-signal-500" : "bg-line-strong"
        }`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-surface transition ${
            value ? "left-4" : "left-0.5"
          }`}
        />
      </span>
    </button>
  );
}

/** What each token looks like, for the swatches. Kept beside them rather than
 *  imported from the tenant's stylesheet: this is the console's own theme, and a
 *  swatch that changed colour with the console's mode would be lying about the
 *  site. */
const TONE_PREVIEW: Record<string, string> = {
  "": "transparent",
  surface: "#ffffff",
  raised: "#f4f1ea",
  charcoal: "#20201e",
  brand: "#e2483d",
  // ⚠️ The second colour. Shown as a neutral chip rather than a guess at the
  // customer's own accent: this swatch is a legend, and a swatch that claimed a
  // colour the site does not use would be worse than one that claims none.
  accent: "#f5c542",
  ink: "#111111",
};

const COLOR_PREVIEW: Record<string, string> = {
  "": "transparent",
  ink: "#20201e",
  soft: "#4b4a45",
  muted: "#8a8880",
  white: "#ffffff",
  brand: "#e2483d",
  surface: "#ffffff",
  charcoal: "#20201e",
};

/** Saved styles: name one, reuse it.
 *
 *  ⚠️ **This is the feature that makes a large design finishable.** A page drawn
 *  from a picture has one heading style, one caption style and one label style,
 *  repeated across fifteen bands — and setting font, weight, size, colour and
 *  alignment by hand each time is both slow and how a page ends up with four
 *  slightly different headings. That inconsistency is exactly what makes a design
 *  look homemade, and it is invisible while drawing: each element looked right on
 *  its own.
 *
 *  Applying copies. Editing a preset later changes nothing already drawn — see
 *  models.StylePreset for why that is the right way round. */
function PresetPanel({
  presets,
  setPresets,
  current,
  onApply,
  canApply,
}: {
  presets: StylePreset[];
  setPresets: React.Dispatch<React.SetStateAction<StylePreset[]>>;
  current: StylePreset["style"] | null;
  onApply: (style: StylePreset["style"]) => void;
  canApply: boolean;
}) {
  const { lang } = useT();
  const d = editorDict(lang);
  const BAND_LABEL = BAND_LABELS[lang] ?? BAND_LABELS.uz;
  const ELEMENT_LABEL = ELEMENT_LABELS[lang] ?? ELEMENT_LABELS.uz;
  const [name, setName] = useState("");

  return (
    <div className="space-y-2 rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{d.presets}</p>
      <p className="text-[11px] leading-relaxed text-ink-muted">
        {d.presetsHint}
      </p>

      {presets.length === 0 && (
        <p className="text-[11px] text-ink-muted">{d.presetsEmpty}</p>
      )}

      <ul className="space-y-1">
        {presets.map((p, i) => (
          <li key={i} className="flex items-center gap-1">
            <button
              type="button"
              disabled={!canApply}
              onClick={() => onApply(p.style)}
              title={canApply ? d.presetApply : d.presetNeedElement}
              className="flex-1 rounded-lg border border-line px-2 py-1.5 text-left text-xs font-semibold text-ink-soft hover:text-ink disabled:opacity-40"
            >
              {p.name}
              <span className="ml-2 font-normal text-ink-muted">
                {describe(p.style)}
              </span>
            </button>
            <button
              type="button"
              onClick={() => setPresets((prev) => prev.filter((_, k) => k !== i))}
              className="px-1 text-hot-600"
              aria-label="o'chirish"
            >
              ×
            </button>
          </li>
        ))}
      </ul>

      <div className="flex gap-1 pt-1">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={d.presetName}
          className="input h-9 flex-1 px-3 py-1 text-xs"
        />
        <button
          type="button"
          disabled={!current || !name.trim()}
          onClick={() => {
            if (!current || !name.trim()) return;
            setPresets((prev) => [
              ...prev.filter((p) => p.name !== name.trim()),
              { name: name.trim(), style: { ...current } },
            ]);
            setName("");
          }}
          className="rounded-xl bg-ink px-3 py-1.5 text-xs font-semibold text-surface disabled:opacity-40"
        >
          {d.presetSave}
        </button>
      </div>
      {!current && (
        <p className="text-[11px] text-ink-muted">
          {d.presetNeedSelection}
        </p>
      )}
    </div>
  );
}

/** A one-line summary of a style, so a list of names is not a list of guesses. */
function describe(style: StylePreset["style"]): string {
  if (!style) return "";
  const bits: string[] = [];
  if (style.size) bits.push(`${style.size > 0 ? "+" : ""}${style.size}`);
  if (style.weight) bits.push(style.weight === "black" ? "juda qalin" : "qalin");
  if (style.color) bits.push(style.color);
  if (style.align === "center") bits.push("o'rta");
  return bits.join(" · ");
}


/** The repeatable items inside a section: add, remove, reorder, edit.
 *
 *  Everything here is generated from the schema's block definitions, so a new block
 *  type is a schema entry rather than another panel in this file — the same reason
 *  the section settings are generated. */
function BlockList({
  band,
  def,
  index,
  picked,
  setPicked,
  update,
  lang,
}: {
  band: DesignSection;
  def: SectionDef;
  index: number;
  picked: number | null;
  setPicked: (i: number | null) => void;
  update: (i: number, patch: Partial<DesignSection>) => void;
  lang: string;
}) {
  const blocks = band.blocks ?? [];
  const defs = def.blocks ?? [];

  function write(next: NonNullable<DesignSection["blocks"]>) {
    update(index, { blocks: next });
  }

  function add(type: string) {
    const d = defs.find((b) => b.type === type);
    // Seeded from the schema's defaults, so a new card is visible on the page
    // immediately: a block that renders nothing looks like the button failed.
    const settings: Record<string, unknown> = {};
    for (const s of d?.settings ?? []) {
      if (s.default !== undefined) settings[s.key] = s.default;
      else if (s.localized) settings[s.key] = { uz: "", ru: "", en: "" };
    }
    write([...blocks, { type, settings }]);
    setPicked(blocks.length);
  }

  const active = picked != null ? blocks[picked] : null;
  const activeDef = active ? defs.find((b) => b.type === active.type) : null;

  return (
    <div className="space-y-2 rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">
        {def.blocks?.map((b) => b.name[lang] ?? b.name.uz).join(" · ")}
      </p>

      <ul className="space-y-1">
        {blocks.map((b, i) => (
          <li
            key={i}
            draggable
            onDragStart={(e) => e.dataTransfer.setData("text/plain", String(i))}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => {
              e.preventDefault();
              const from = Number(e.dataTransfer.getData("text/plain"));
              if (Number.isNaN(from) || from === i) return;
              const next = [...blocks];
              const [moved] = next.splice(from, 1);
              next.splice(i, 0, moved);
              write(next);
              setPicked(i);
            }}
            className="flex cursor-grab items-center gap-1"
          >
            <span className="select-none px-1 text-ink-muted" aria-hidden>⠿</span>
            <button
              type="button"
              onClick={() => setPicked(i)}
              className={`flex-1 truncate rounded-lg px-2 py-1.5 text-left text-xs ${
                picked === i ? "bg-raised font-semibold text-ink" : "text-ink-soft"
              } ${b.hidden ? "line-through opacity-50" : ""}`}
            >
              {blockLabel(b, i, lang)}
            </button>
            <button
              type="button"
              onClick={() => write(blocks.map((x, k) => (k === i ? { ...x, hidden: !x.hidden } : x)))}
              className="px-1 text-[11px] text-ink-muted"
              aria-label="hidden"
            >
              {b.hidden ? "○" : "●"}
            </button>
            <button
              type="button"
              onClick={() => {
                write(blocks.filter((_, k) => k !== i));
                setPicked(null);
              }}
              className="px-1 text-hot-600"
              aria-label="×"
            >
              ×
            </button>
          </li>
        ))}
      </ul>

      <div className="flex flex-wrap gap-1">
        {defs.map((b) => (
          <button
            key={b.type}
            type="button"
            onClick={() => add(b.type)}
            className="rounded-lg border border-line px-2 py-1 text-[11px] text-ink-soft hover:text-ink"
          >
            + {b.name[lang] ?? b.name.uz}
          </button>
        ))}
      </div>

      {active && activeDef && (
        <div className="space-y-2.5 border-t border-line pt-2">
          {activeDef.settings.map((s) => (
            <Control
              key={s.key}
              def={s}
              value={(active.settings ?? {})[s.key]}
              lang={lang}
              onChange={(value) =>
                write(
                  blocks.map((x, k) =>
                    k === picked ? { ...x, settings: { ...(x.settings ?? {}), [s.key]: value } } : x,
                  ),
                )
              }
            />
          ))}
        </div>
      )}
    </div>
  );
}

/** A block's own words where it has any, its position where it does not — a list of
 *  identical rows is a list nobody can navigate. */
function blockLabel(
  b: NonNullable<DesignSection["blocks"]>[number],
  i: number,
  lang: string,
): string {
  const bag = (b.settings ?? {}) as Record<string, unknown>;
  for (const key of ["title", "caption", "heading"]) {
    const v = bag[key];
    if (typeof v === "string" && v) return v;
    if (v && typeof v === "object") {
      const t = v as Record<string, string>;
      const text = t[lang] || t.uz;
      if (text) return text;
    }
  }
  const img = bag.image;
  if (typeof img === "string" && img) return img.split("/").pop() ?? img;
  return `${i + 1}`;
}

/** The palette a design is drawn with.
 *
 *  ⚠️ **Colours here are hex and everywhere else they are tokens, and that is
 *  not an inconsistency.** This panel sets what the tokens *mean* — it is the
 *  one place a real colour belongs, and it is why every band and element can
 *  stay an enum. A band painted `#f5c542` would keep that colour in dark mode
 *  and ignore whatever the shop chose; a band painted `accent` follows this.
 *
 *  ⚠️ **Empty means "leave it alone", per field.** A design that only
 *  rearranges bands must not repaint a restaurant that spent an afternoon
 *  choosing its accent — the tenant merges this field by field (mergeTheme). */
function ThemePanel({
  theme,
  setTheme,
  d,
}: {
  theme: SiteThemePatch;
  setTheme: (t: SiteThemePatch) => void;
  d: EditorDict;
}) {
  const set = (k: keyof SiteThemePatch, v: string) =>
    setTheme({ ...theme, [k]: v });

  const Colour = ({
    k,
    label,
    hint,
  }: {
    k: "brand" | "accent";
    label: string;
    hint: string;
  }) => (
    <div>
      <span className="text-[11px] font-semibold text-ink-muted">{label}</span>
      <div className="mt-0.5 flex items-center gap-2">
        {/* ⚠️ A colour well **and** a text box. The well cannot express "not
            set" — it always shows something — so the box beside it is what
            makes clearing a colour possible, and clearing is how a customer
            goes back to the palette they had. */}
        <input
          type="color"
          value={theme[k] || "#e2590d"}
          onChange={(e) => set(k, e.target.value)}
          className="h-8 w-10 shrink-0 cursor-pointer rounded-lg border border-line bg-surface"
          aria-label={label}
        />
        <input
          value={theme[k] ?? ""}
          onChange={(e) => set(k, e.target.value.trim())}
          placeholder={d.themeUnset}
          spellCheck={false}
          className="input font-mono text-[11px]"
        />
      </div>
      <p className="mt-1 text-[11px] leading-relaxed text-ink-muted">{hint}</p>
    </div>
  );

  return (
    <div className="space-y-3 rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{d.themeTitle}</p>
      <p className="text-[11px] leading-relaxed text-ink-muted">{d.themeHint}</p>
      <Colour k="brand" label={d.themeBrand} hint={d.themeBrandHint} />
      <Colour k="accent" label={d.themeAccent} hint={d.themeAccentHint} />
    </div>
  );
}
