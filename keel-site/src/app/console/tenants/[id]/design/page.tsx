"use client";

// The layout editor, as its own page.
//
// It started life as a panel inside the tenant card, and that was wrong for a
// simple reason: this is the screen somebody sits at for an hour with a customer's
// screenshot open beside it. A panel between an invoice list and a container log
// gets a third of the width and none of the attention.
//
// ⚠️ **The shell was rebuilt once the tools were all there**, and the reason is
// the sentence the person who uses it said: *it is hard*. Everything worked —
// bands, free placement, live editing, a real preview — and it was reached
// through a rail of six unlabelled glyphs (▤ ◫ ◐ { } ☰ ▢) in front of a column
// that held every panel at once. Two changes fixed most of it:
//
//   • **Two named tabs instead of six glyphs.** «Sahifa» is what the page is
//     made of; «Dizayn» is everything that applies to the whole site — colours,
//     the bar, templates, saved styles, CSS. Nobody has to learn which square
//     means which.
//   • **The inspector is where the thing is, not where a tab is.** Click
//     something on the page and its settings open on the right, titled with what
//     was clicked. The old editor put them behind a second click on an icon —
//     at the exact moment they were certainly wanted.
//
// The rest of the shape is the one every layout editor converges on:
//
//   • **Left: what the page is made of.** Bands in order, and inside a freely
//     drawn band, its elements — one tree, so "what is on this page" is one
//     list rather than three panels.
//
//   • **Middle: the real site.** ⚠️ An iframe of the tenant's own domain with the
//     unpublished draft applied, not a schematic. A schematic can show that a band
//     is 6 columns wide; it cannot answer "does this look like the picture the
//     customer sent us", and that is the entire job.
//
//   • **A phone width beside the desktop one**, switchable. Not decoration:
//     freely placed elements have a **separate phone layout**, and the whole
//     failure mode of free placement is a composition nobody checked at 390px.
//
// ⚠️ **Nothing reloads the page any more if it does not have to.** Typing a word
// and dragging a box are sent into the live preview as a patch (see
// PreviewBridge) and the draft is saved quietly behind it. Before this, every
// letter typed and every box dragged re-pointed the iframe at the tenant's real
// site: a full navigation, with the white flash and the jump to the top of the
// page that come with it. A reload still happens for changes a patch cannot
// express — a new band, a different variant — and even then the operator is put
// back where they were scrolled to.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import EditorCanvas from "@/components/design/EditorCanvas";
import PreviewOverlay, { type Incoming } from "@/components/design/PreviewOverlay";
import NavEditor from "@/components/design/NavEditor";
import { useT } from "@/lib/i18n/client";
import { diffDesign } from "@/lib/designDiff";
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
  uploadTenantImage,
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

/** Which elements have words a double-click can open.
 *
 *  ⚠️ **A list, not "has a text field".** Every element type carries `text` in
 *  the model; only these render it. A caret blinking inside a photograph or a
 *  coloured panel is an editor claiming something it cannot do, and the
 *  operator finds out by typing into it and watching nothing change. */
const TEXTUAL = new Set([
  "text", "button", "badge", "quote", "stat", "list",
]);

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
// ⚠️ `/catalog` beside `/menu`: a shop's catalogue answers there and `/menu`
// redirects to it, so either works — but a button written with the shop's own
// address sends the guest straight there instead of through a redirect.
const LINKS = ["", "/", "/menu", "/catalog", "/cart", "/checkout", "/bron", "/about", "/profile"];

/** The sections somebody can add, grouped and in the order they are usually
 *  reached for.
 *
 *  ⚠️ **Grouped rather than a wall of sixteen buttons**, and the groups are the
 *  question being asked: "the top of the page", "something with words in it",
 *  "draw it myself", "the bar and the footer". The old list was one flat row of
 *  chips in which `navbar` sat between `popup` and `categories` — which is how
 *  an operator ends up adding a second header to a page. */
const BAND_GROUPS: { key: "basic" | "content" | "free" | "shell"; types: string[] }[] = [
  { key: "basic", types: ["hero", "menu-grid", "categories", "perks", "search"] },
  { key: "content", types: ["rich-text", "image-text", "banner", "gallery", "about", "cta", "hours-address"] },
  { key: "free", types: ["canvas", "popup"] },
  { key: "shell", types: ["navbar", "footer"] },
];

/** The elements a free band can hold, in the same shape and for the same
 *  reason. `widget-*` is last because it is the one group that is not a drawing
 *  — it is a working part of the site being placed. */
const ELEMENT_GROUPS: { key: "text" | "media" | "shape" | "widget"; types: string[] }[] = [
  { key: "text", types: ["text", "button", "badge", "list", "quote", "stat"] },
  { key: "media", types: ["image", "carousel", "icon", "rating"] },
  { key: "shape", types: ["box", "divider"] },
  {
    key: "widget",
    types: ["widget-menu", "widget-categories", "widget-hours", "widget-map", "widget-cart", "widget-social"],
  },
];

/** One glyph per element type, for the palette cards.
 *
 *  ⚠️ **Beside the name, never instead of it.** The glyph makes a grid of cards
 *  scannable once somebody knows the editor; the word is what makes it usable
 *  the first time. The rail this screen replaced had the glyph alone. */
const ELEMENT_GLYPH: Record<string, string> = {
  text: "T", button: "▭", badge: "◔", list: "≡", quote: "❝", stat: "12",
  image: "▣", carousel: "▥", icon: "★", rating: "★★",
  box: "■", divider: "—",
  "widget-menu": "▤", "widget-categories": "▦", "widget-hours": "◷",
  "widget-map": "◎", "widget-cart": "▾", "widget-social": "◈",
};

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

/** A new band, seeded so that adding it changes the page.
 *
 *  ⚠️ An empty section renders nothing, and a band that appears in the list
 *  while the page does not change reads as the button not working. */
function newBand(type: string): DesignSection {
  const seeded: Record<string, Record<string, unknown>> = {
    hero: { heading: { uz: "Sarlavha", ru: "Заголовок", en: "Heading" }, height: 80, overlay: 40 },
    "rich-text": { heading: { uz: "Sarlavha", ru: "Заголовок", en: "Heading" }, align: "center", tone: "surface" },
    "image-text": { heading: { uz: "Sarlavha", ru: "Заголовок", en: "Heading" }, round: "lg" },
    banner: { heading: { uz: "Aksiya", ru: "Акция", en: "Offer" }, tone: "charcoal", overlay: 55 },
    "menu-grid": { popularOnly: true, limit: 8 },
  };
  return {
    type,
    variant: VARIANTS[type]?.[0] ?? "",
    span: 12,
    settings: seeded[type],
    ...(type === "canvas" || type === "popup"
      ? { canvas: { height: 60, elements: [] } }
      : {}),
  };
}

/** Keeps a box inside the band it was dropped into. A drop near the right edge
 *  must not put half the element outside the page. */
function clampBox(box: DesignBox, x: number, y: number): DesignBox {
  return {
    ...box,
    x: Math.max(0, Math.min(100 - box.w, Math.round(x - box.w / 2))),
    y: Math.max(0, Math.min(100 - box.h, Math.round(y - box.h / 2))),
  };
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
  // ⚠️ **The real page, not the schematic, and it opens on it.** The schematic
  // shows grey rectangles where the customer's photographs and prices are, so
  // the one question the operator actually has cannot be asked from it.
  const [pane, setPane] = useState<"canvas" | "site">("site");
  const [presets, setPresets] = useState<StylePreset[]>([]);
  // ⚠️ **Two tabs, and they answer two different questions.** «Sahifa» is what
  // this page is made of, band by band; «Dizayn» is what applies to the whole
  // site — colours, the bar, templates, saved styles, the stylesheet. The six
  // unlabelled icons this replaced made both questions into a guess.
  const [side, setSide] = useState<"page" | "design">("page");
  // Whether the section picker is open. A sheet rather than a permanent row of
  // chips: sixteen sections is a wall, and it is needed for ten seconds.
  const [adding, setAdding] = useState(false);
  // The site's navigation bar. ⚠️ Empty is "leave the header alone", never "a
  // bar with no links" — every tenant on the platform has this unset, and the
  // other reading would empty the header of every site at once.
  const [nav, setNav] = useState<NavLink[]>([]);
  // The palette this design is drawn with. The site reads it (handlers/public.go,
  // mergeTheme) and this is where it is chosen.
  const [theme, setTheme] = useState<SiteThemePatch>({});
  const [templates, setTemplates] = useState<DesignTemplate[]>([]);
  const [schema, setSchema] = useState<SectionDef[]>([]);
  // Which repeatable item inside the band is being edited. Separate from the
  // element selection: a band has blocks *or* freely drawn elements, never both.
  const [pickBlock, setPickBlock] = useState<number | null>(null);
  // ⚠️ The **schematic** canvas's zoom. The live pane has its own (`siteZoom`)
  // and it is measured rather than chosen.
  const [zoom, setZoom] = useState(0.7);
  // ⚠️ Editing **on the live preview**: handles drawn over the iframe, using the
  // geometry the site reports.
  const [liveEdit, setLiveEdit] = useState(true);
  // What is being dragged off the palette right now, if anything. Held here
  // rather than in the browser's drag data because the drop target has to know
  // *while* the pointer is moving — see PreviewOverlay's `Incoming`.
  const [incoming, setIncoming] = useState<Incoming | null>(null);
  // ⚠️ Undo is not a nicety in a direct-manipulation editor: the whole way of
  // working is "try it and see", and a drag that cannot be taken back makes
  // trying it expensive.
  const history = useRef<DesignSection[][]>([]);
  const future = useRef<DesignSection[][]>([]);
  const [busy, setBusy] = useState("");
  const [note, setNote] = useState("");
  // What the draft is doing right now, for the one word in the header that says
  // whether the work is safe. ⚠️ It replaced a «Save» button that had to be
  // pressed: an editor whose changes are lost by closing the tab is an editor
  // people do not trust, and the one before this had exactly that shape.
  const [saveState, setSaveState] = useState<"clean" | "dirty" | "saving" | "error">("clean");
  const frame = useRef<HTMLIFrameElement>(null);
  const sectionsRef = useRef<DesignSection[]>([]);
  useEffect(() => {
    sectionsRef.current = sections;
  }, [sections]);
  // True between pointerdown and pointerup, on **either** surface.
  //
  // ⚠️ **It was only ever set by the schematic pane**, and the live page is now
  // the one people work on — so every pixel of every drag there pushed a copy
  // of the whole document onto the undo stack, ran a full diff of it
  // (`JSON.stringify` per band and per element) and asked the site to measure
  // itself again. That is the stutter: not one slow thing, three cheap things
  // multiplied by a mouse.
  const dragging = useRef(false);
  /** Which element the drag is moving, so the watcher can skip the diff: during
   *  a drag we already know what changed. */
  const dragTarget = useRef<{ band: number; index: number } | null>(null);

  // ---- What a change costs ----
  //
  // ⚠️ **Two kinds of change, and telling them apart is the whole fix.** Most
  // edits can be written straight into the rendered page — a box that moved, a
  // colour, a size, a heading — because `canvasStyle` decides what an element
  // looks like and the site's bridge can run the same code in the browser.
  // What cannot is structure: a band added, a variant changed, a binding that
  // decides which dishes are shown. The editor used to treat *every* change as
  // structure, so a single typed letter navigated the iframe to the customer's
  // real site.
  //
  // ⚠️ **The decision is a diff of the document, not a flag set by whoever made
  // the change** (`lib/designDiff.ts`). A flag has to be remembered at every
  // call site, and the one that forgets it is the one that shows the operator a
  // preview that disagrees with what they are editing.
  const loaded = useRef(false);
  /** The sections as the preview currently shows them. */
  const shown = useRef<DesignSection[]>([]);
  /** Set by a change the page cannot be told about: it has to be rendered. */
  const staleView = useRef(false);
  const saveTimer = useRef<number | undefined>(undefined);
  /** Where the preview was scrolled to, reported by the site itself. Kept so a
   *  reload can put the operator back rather than at the top of the page. */
  const scrollY = useRef(0);

  useEffect(() => {
    function onMessage(e: MessageEvent) {
      const data = e.data as { type?: string; scrollY?: number };
      if (data?.type === "keel:geometry" && typeof data.scrollY === "number") {
        scrollY.current = data.scrollY;
      }
    }
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, []);

  /** Tells the live page what changed, without reloading it. Silent when the
   *  preview is not open — the draft is still saved either way. */
  const pushPatch = useCallback(
    (
      change: { elements: { band: number; index: number }[]; bands: number[] },
      from: DesignSection[],
      /** Whether the page should re-measure afterwards. ⚠️ False during a drag:
       *  the console already knows where the box is — it is the one drawing it —
       *  and a measurement per pixel is a round trip per pixel for an answer
       *  nobody reads. */
      measure = true,
    ) => {
      const win = frame.current?.contentWindow;
      if (!win) return;
      // ⚠️ The **whole** element, not the fields that changed: only the site
      // knows which class name a style key produces, and it is the same module
      // that rendered the page. See PreviewBridge.
      const elements = change.elements.flatMap(({ band, index }) => {
        const el = from[band]?.canvas?.elements?.[index];
        return el ? [{ band, index, el }] : [];
      });
      if (elements.length > 0) {
        win.postMessage({ type: "keel:patch", items: elements, measure }, "*");
      }
      const bands = change.bands.flatMap((band) =>
        from[band] ? [{ band, section: from[band] }] : [],
      );
      if (bands.length > 0) {
        win.postMessage({ type: "keel:band", items: bands }, "*");
      }
    },
    [],
  );

  useEffect(() => {
    void (async () => {
      try {
        const [des, t] = await Promise.all([tenantDesign(tenantId), tenantApi(tenantId)]);
        setState(des);
        const opening = des.draft.sections ?? des.live.sections ?? [];
        setSections(opening);
        // What the preview will be showing once it loads.
        shown.current = opening;
        setCss(des.draft.customCss ?? "");
        setPresets(des.draft.stylePresets ?? []);
        setNav(des.draft.nav ?? des.live.nav ?? []);
        setTheme(des.draft.theme ?? des.live.theme ?? {});
        try {
          const [g, sc] = await Promise.all([designTemplates(), designSchema()]);
          setTemplates(g.items);
          setSchema((sc.sections ?? []) as SectionDef[]);
        } catch {
          // A gallery that failed to load must not stop somebody editing what
          // they already have.
        }
        setSlug(t.tenant.slug);
        await refreshPreview();
        // ⚠️ Last, and after a tick: everything above is a `setState`, and the
        // autosave watcher must not read the document arriving as a change the
        // operator made.
        window.setTimeout(() => {
          loaded.current = true;
        }, 0);
      } catch (e) {
        setNote(e instanceof Error ? e.message : "yuklanmadi");
      }
    })();
  }, [tenantId]);

  const band = sections[pick.band];
  const bandDef = defFor(schema, band);
  const element =
    band?.canvas?.elements && pick.el != null ? band.canvas.elements[pick.el] : null;

  const persist = useCallback(async () => {
    // ⚠️ **Never in the middle of a drag.** A save scheduled by the edit before
    // this one comes due while the hand is still moving, and if that edit needed
    // the page rendered again the iframe reloads **under the cursor** — which is
    // the "it hard-refreshes while I resize" nobody could place, because the
    // change that caused it was the one before.
    if (dragging.current) {
      window.clearTimeout(saveTimer.current);
      saveTimer.current = window.setTimeout(() => void persistRef.current(), 400);
      return;
    }
    setSaveState("saving");
    try {
      await saveTenantDesign(tenantId, sectionsRef.current, css, presets, nav, theme);
      setSaveState("clean");
      if (staleView.current && !dragging.current) {
        staleView.current = false;
        reloadPreview();
      }
    } catch (e) {
      setSaveState("error");
      setNote(e instanceof Error ? e.message : "saqlanmadi");
    }
    // `reloadPreview` is declared below and stable for the life of the page.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId, css, presets, nav, theme]);

  // ⚠️ The two callbacks are read through refs so that the watcher below can
  // depend on the **document** alone. With them in the dependency list, merely
  // switching between the desktop and the phone layout — which changes
  // `pushPatch` — counted as an edit, and an edit that changed nothing still
  // saved the draft and reloaded the page.
  const persistRef = useRef(persist);
  persistRef.current = persist;
  const pushPatchRef = useRef(pushPatch);
  pushPatchRef.current = pushPatch;

  // The autosave watcher. ⚠️ It decides between a patch and a reload, and it
  // debounces: a save per keystroke would be a write per keystroke on somebody's
  // database, and a reload per keystroke is what made this editor painful.
  //
  // ⚠️ **The pending save is not cancelled on unmount.** Letting the timer fire
  // after the operator has navigated away is what saves the last sentence they
  // typed; cancelling it loses up to a second of work at the one moment nobody
  // is watching the screen to notice.
  useEffect(() => {
    if (!loaded.current) return;
    // ⚠️ Compared against what the preview is **showing**, not against the
    // previous render: several edits can land between two reloads, and a diff
    // against the last keystroke would call the second one "nothing changed".
    // ⚠️ **A drag skips the diff entirely.** Comparing two documents means
    // stringifying every band and every element, and a drag asks for that sixty
    // times a second while the operator is watching the thing they are dragging.
    // What changed is not in doubt here — it is the box under the cursor.
    if (dragging.current && dragTarget.current) {
      shown.current = sections;
      pushPatchRef.current({ elements: [dragTarget.current], bands: [] }, sections, false);
      return;
    }
    const change = sections === shown.current
      ? ({ kind: "reload" } as const) // the stylesheet, the bar or the palette
      : diffDesign(shown.current, sections);
    shown.current = sections;
    if (change.kind === "patch") pushPatchRef.current(change, sections);
    else if (change.kind === "reload") staleView.current = true;
    setSaveState("dirty");
    window.clearTimeout(saveTimer.current);
    saveTimer.current = window.setTimeout(() => void persistRef.current(), 900);
  }, [sections, css, presets, nav, theme]);

  /** Records the current state so the next change can be undone. Called on the
   *  **start** of a change, and skipped while a drag is in flight. */
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

  /** A word changed. ⚠️ Its own path because it is the change people make most,
   *  and the one the live page can be told about — routing it through
   *  `updateElement` would mark the view stale and reload the site per letter. */
  const setElementText = useCallback(
    (bandIdx: number, elIdx: number, value: { uz: string; ru: string; en: string }) => {
      updateElement(bandIdx, elIdx, { text: value });
    },
    [updateElement],
  );

  /** Moves the box the operator is currently editing — desktop or phone. */
  const moveBox = useCallback(
    (bandIdx: number, elIdx: number, patch: Partial<DesignBox>) => {
      if (!dragging.current) remember();
      dragTarget.current = { band: bandIdx, index: elIdx };
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

  /** Puts a new element into a band, at the point it was dropped. */
  const placeElement = useCallback(
    (bandIdx: number, type: string, x: number, y: number) => {
      const target = sectionsRef.current[bandIdx];
      if (!target?.canvas) return;
      const el = newElement(type);
      const box = clampBox(el.box, x, y);
      const placed: DesignElement =
        editing === "mobile" ? { ...el, box, mobile: box } : { ...el, box };
      remember();
      setSections((prev) =>
        prev.map((s, k) =>
          k === bandIdx && s.canvas
            ? { ...s, canvas: { ...s.canvas, elements: [...(s.canvas.elements ?? []), placed] } }
            : s,
        ),
      );
      setPick({ band: bandIdx, el: (target.canvas.elements ?? []).length });
      setSide("page");
    },
    [editing, remember],
  );

  /** Puts a new band at an index — dropped between two bands on the page, or
   *  appended when it was clicked rather than dragged. */
  const placeBand = useCallback(
    (type: string, at: number) => {
      remember();
      const index = Math.max(0, Math.min(sectionsRef.current.length, at));
      setSections((prev) => {
        const next = [...prev];
        next.splice(index, 0, newBand(type));
        return next;
      });
      setPick({ band: index, el: null });
      setAdding(false);
      setNote("");
    },
    [remember],
  );

  async function save() {
    setBusy("save");
    setNote("");
    window.clearTimeout(saveTimer.current);
    await persist();
    setBusy("");
  }

  /** Re-renders the real page, and puts the operator back where they were.
   *
   *  ⚠️ **The token is reused rather than minted again.** A new token per reload
   *  would leave a trail of live preview links, each valid for two hours. */
  function reloadPreview() {
    shown.current = sectionsRef.current;
    setPreviewUrl((url) => (url ? `${url.split("&_=")[0]}&_=${Date.now()}` : url));
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
      window.clearTimeout(saveTimer.current);
      await saveTenantDesign(tenantId, sectionsRef.current, css, presets, nav, theme);
      setSaveState("clean");
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

  // How big the preview pane actually is, measured.
  //
  // ⚠️ **The scale used to be a constant, and the pane had a hole in it.** A
  // 1280px page at 0.62 is 794px wide whatever the window is, and `transform`
  // does not change layout — so on any wider pane the site sat in the top-left
  // corner with bare panel down the right.
  const paneRef = useRef<HTMLDivElement>(null);
  const [paneSize, setPaneSize] = useState({ w: 0, h: 0 });
  useEffect(() => {
    const el = paneRef.current;
    if (!el) return;
    const read = () => setPaneSize({ w: el.clientWidth, h: el.clientHeight });
    read();
    // ⚠️ A ResizeObserver rather than a window listener: the pane also changes
    // width when the inspector opens, which the window never hears about — and a
    // stale scale puts the drag handles somewhere the element is not.
    const ro = new ResizeObserver(read);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  /** The scale the preview is drawn at. ⚠️ Never above 1: blowing a 390px phone
   *  page up to fill a desktop pane is a preview of something nobody will see. */
  const siteZoom = useMemo(() => {
    if (!paneSize.w) return device === "phone" ? 1 : 0.62;
    return Math.min(1, paneSize.w / previewWidth);
  }, [paneSize.w, previewWidth, device]);

  const frameStyle = useMemo(
    () => ({
      width: previewWidth,
      transform: `scale(${siteZoom})`,
      transformOrigin: "top left",
      height: paneSize.h ? Math.round(paneSize.h / siteZoom) : 1400,
    }),
    [previewWidth, siteZoom, paneSize.h],
  );

  const canDrawIn = useCallback(
    (i: number) => !!sectionsRef.current[i]?.canvas,
    [],
  );

  return (
    // Fills what the console header leaves, and scrolls inside its own panes: a
    // page-level scrollbar here would move the canvas out from under the cursor.
    <div className="flex h-[calc(100vh-4rem)] flex-col bg-surface">
      <header className="flex flex-wrap items-center gap-3 border-b border-line px-4 py-2.5">
        <Link
          href={`/console/tenants/${tenantId}`}
          className="text-sm font-semibold text-ink-soft hover:text-ink"
        >
          ← {slug || "mijoz"}
        </Link>
        <span className="h-4 w-px bg-line" />
        <h1 className="text-sm font-bold text-ink">{d.title}</h1>
        {state?.published && (
          <span className="rounded-full bg-signal-500/15 px-2 py-0.5 text-[11px] font-semibold text-signal-600">
            {d.live}
          </span>
        )}
        {/* ⚠️ The one word that says whether the work is safe, and it is in the
            header rather than in a toast: a message that fades is a message
            somebody was not looking at. */}
        <SaveState state={saveState} d={d} />

        <div className="ml-auto flex flex-wrap items-center gap-2">
          <Seg
            value={device}
            options={[
              { v: "desktop", label: d.desktop },
              { v: "phone", label: `${d.phone} · 390` },
            ]}
            onChange={(v) => setDevice(v as "desktop" | "phone")}
          />
          <span className="flex overflow-hidden rounded-xl border border-line">
            <button
              type="button"
              onClick={undo}
              title={d.undo}
              className="px-2.5 py-1.5 text-xs text-ink-soft hover:text-ink"
            >
              ↶
            </button>
            <button
              type="button"
              onClick={redo}
              title={d.redo}
              className="border-l border-line px-2.5 py-1.5 text-xs text-ink-soft hover:text-ink"
            >
              ↷
            </button>
          </span>
          {/* Opens the draft on the real domain, in a tab. The inline preview is
              for glancing; this is for handing to somebody, or for scrolling the
              whole page — which an iframe two thirds the height cannot do. */}
          <button
            type="button"
            onClick={async () => {
              await save();
              if (previewUrl) window.open(previewUrl, "_blank", "noopener");
            }}
            title={d.viewHint}
            className="rounded-xl border border-line px-3 py-1.5 text-xs font-semibold text-ink-soft hover:text-ink"
          >
            👁 {d.view}
          </button>
          <button
            type="button"
            onClick={() => void publish()}
            disabled={busy !== ""}
            className="rounded-xl bg-ink px-3.5 py-1.5 text-xs font-semibold text-surface disabled:opacity-40"
          >
            {busy === "publish" ? d.publishing : d.publish}
          </button>
          {state?.published && (
            <button
              type="button"
              onClick={() => void revert()}
              disabled={busy !== ""}
              className="rounded-xl border border-line px-3 py-1.5 text-xs font-semibold text-ink-muted disabled:opacity-40"
            >
              {d.revert}
            </button>
          )}
        </div>
      </header>

      {note && (
        <p className="border-b border-line bg-raised px-4 py-2 text-xs text-ink-soft">{note}</p>
      )}

      <div className="flex min-h-0 flex-1 flex-col lg:flex-row">
        {/* ---- Left: what the page is made of ---- */}
        <aside className="flex w-full shrink-0 flex-col border-b border-line lg:w-72 lg:border-b-0 lg:border-r">
          <div className="border-b border-line p-2">
            <Seg
              value={side}
              options={[
                { v: "page", label: d.tabPage },
                { v: "design", label: d.tabDesign },
              ]}
              onChange={(v) => setSide(v as "page" | "design")}
            />
          </div>

          <div className="min-h-0 flex-1 space-y-3 overflow-auto p-3">
            {side === "page" && (
              <>
                <Outline
                  sections={sections}
                  pick={pick}
                  setPick={(p) => {
                    setPick(p);
                    setPickBlock(null);
                  }}
                  setSections={setSections}
                  remember={remember}
                  d={d}
                  bandLabel={BAND_LABEL}
                  elementLabel={ELEMENT_LABEL}
                />
                <button
                  type="button"
                  onClick={() => setAdding((v) => !v)}
                  className="w-full rounded-2xl border border-dashed border-line-strong py-2.5 text-xs font-bold text-ink-soft hover:border-signal-500 hover:text-ink"
                >
                  + {d.addBand}
                </button>
                {adding && (
                  <Palette
                    title={d.addBand}
                    hint={d.addDragHint}
                    groups={BAND_GROUPS.map((g) => ({
                      title: {
                        basic: d.groupBasic,
                        content: d.groupContent,
                        free: d.groupFree,
                        shell: d.groupShell,
                      }[g.key],
                      items: g.types.map((t) => ({ type: t, label: BAND_LABEL[t] ?? t })),
                    }))}
                    onDragStart={(type) => setIncoming({ kind: "band", type })}
                    onDragEnd={() => setIncoming(null)}
                    onPick={(type) => placeBand(type, sections.length)}
                  />
                )}

                {band?.canvas && (
                  <Palette
                    title={d.addElement}
                    hint={d.addDragHint}
                    groups={ELEMENT_GROUPS.map((g) => ({
                      title: {
                        text: d.elGroupText,
                        media: d.elGroupMedia,
                        shape: d.elGroupShape,
                        widget: d.elGroupWidget,
                      }[g.key],
                      items: g.types.map((t) => ({
                        type: t,
                        label: ELEMENT_LABEL[t] ?? t,
                        glyph: ELEMENT_GLYPH[t],
                      })),
                    }))}
                    onDragStart={(type) => setIncoming({ kind: "element", type })}
                    onDragEnd={() => setIncoming(null)}
                    // Clicked rather than dragged: the middle of the band, which
                    // is where somebody looking for it will look.
                    onPick={(type) => placeElement(pick.band, type, 50, 50)}
                  />
                )}
              </>
            )}

            {side === "design" && (
              <>
                <Fold title={d.designColors} open>
                  <ThemePanel theme={theme} setTheme={setTheme} d={d} />
                </Fold>
                <Fold title={d.designNav}>
                  <NavEditor nav={nav} setNav={setNav} d={d} />
                </Fold>
                <Fold title={d.designTemplatesTitle}>
                  <TemplateList
                    templates={templates}
                    d={d}
                    onApply={(tpl) => {
                      // ⚠️ A copy, always. Applying must not link this customer's
                      // page to a gallery entry — an improved template would
                      // otherwise redraw sites that were already approved.
                      remember();
                      setSections(JSON.parse(JSON.stringify(tpl.sections)));
                      // ⚠️ **The palette and the bar come with it**, and only when
                      // the template carries them: one that chooses no colours must
                      // not wipe the ones this customer already has.
                      if (tpl.theme && Object.keys(tpl.theme).length > 0) {
                        setTheme(JSON.parse(JSON.stringify(tpl.theme)));
                      }
                      if (tpl.nav && tpl.nav.length > 0) {
                        setNav(JSON.parse(JSON.stringify(tpl.nav)));
                      }
                      setPick({ band: 0, el: null });
                      setSide("page");
                      setNote(d.templateApplied(tpl.name));
                    }}
                  />
                </Fold>
                <Fold title={d.designPresets}>
                  <PresetPanel
                    presets={presets}
                    setPresets={setPresets}
                    current={element?.style ?? null}
                    onApply={(style: StylePreset["style"]) => {
                      if (pick.el == null) return;
                      // ⚠️ A **copy**, not a reference — the same rule the
                      // template gallery follows.
                      updateElement(pick.band, pick.el, { style: { ...style } });
                    }}
                    canApply={pick.el != null}
                  />
                </Fold>
                <Fold title={d.designCss}>
                  {/* ⚠️ The escape hatch, and the reason the constructor can answer
                      a brief it was not designed for. Refused outright by the
                      backend if it contains anything that could close a `<style>`
                      element — so a rejected stylesheet comes back empty rather
                      than half-applied. */}
                  <div className="space-y-2">
                    <p className="text-[11px] leading-relaxed text-ink-muted">{d.cssHint}</p>
                    <textarea
                      value={css}
                      onChange={(e) => setCss(e.target.value)}
                      rows={8}
                      spellCheck={false}
                      className="w-full rounded-xl border border-line bg-surface p-2 font-mono text-[11px] text-ink"
                      placeholder=".hero h1 { letter-spacing: -0.02em }"
                    />
                  </div>
                </Fold>
              </>
            )}
          </div>

          <p className="border-t border-line px-3 py-2 text-[11px] leading-relaxed text-ink-muted">
            {d.autosaveHint}
          </p>
        </aside>

        {/* ---- Middle: the page itself ---- */}
        <section className="flex min-h-0 min-w-0 flex-1 flex-col bg-raised">
          <div className="flex flex-wrap items-center gap-2 border-b border-line px-4 py-2">
            <Seg
              value={pane}
              options={[
                { v: "site", label: d.site },
                { v: "canvas", label: d.canvas },
              ]}
              onChange={(v) => {
                const p = v as "canvas" | "site";
                setPane(p);
                if (p === "site" && !previewUrl) void refreshPreview();
              }}
            />

            {pane === "canvas" && (
              <>
                <div className="flex items-center gap-1 text-xs text-ink-soft">
                  <button type="button" onClick={() => setZoom((z) => Math.max(0.3, +(z - 0.1).toFixed(2)))} className="rounded-lg border border-line px-2 py-1">−</button>
                  <span className="w-10 text-center tabular-nums">{Math.round(zoom * 100)}%</span>
                  <button type="button" onClick={() => setZoom((z) => Math.min(1, +(z + 0.1).toFixed(2)))} className="rounded-lg border border-line px-2 py-1">+</button>
                </div>
                <span className="text-[11px] text-ink-muted">{d.dragHint}</span>
              </>
            )}
            {pane === "site" && (
              <>
                <Seg
                  value={liveEdit ? "on" : "off"}
                  options={[
                    { v: "on", label: d.liveEditOn },
                    { v: "off", label: d.liveEditOff },
                  ]}
                  onChange={(v) => setLiveEdit(v === "on")}
                />
                <button
                  type="button"
                  onClick={() => reloadPreview()}
                  title={d.refresh}
                  className="rounded-lg border border-line px-2 py-1 text-xs text-ink-soft hover:text-ink"
                >
                  ⟳
                </button>
                <span className="text-[11px] text-ink-muted">{d.liveEditHint}</span>
              </>
            )}
          </div>

          <div ref={paneRef} className="min-h-0 flex-1 overflow-auto p-4">
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
                    dropping={incoming?.kind === "element"}
                    onDropAt={(x, y) => {
                      if (incoming?.kind !== "element") return;
                      placeElement(pick.band, incoming.type, x, y);
                      setIncoming(null);
                    }}
                    onSelect={(el) => setPick({ band: pick.band, el })}
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
              // ⚠️ **Sized to the scaled result, not to the raw page.** A
              // transform does not change layout, so a box left at 100% wide
              // around a 794px rendering is a box with bare panel inside it.
              <div
                className="mx-auto overflow-hidden rounded-2xl border border-line bg-surface"
                style={{
                  width: Math.round(previewWidth * siteZoom),
                  height: paneSize.h ? paneSize.h - 8 : undefined,
                }}
              >
                <iframe
                  ref={frame}
                  src={previewUrl}
                  title={d.site}
                  style={frameStyle}
                  className="block border-0"
                  onLoad={() => {
                    // ⚠️ Back to where the operator was. A reload is sometimes
                    // unavoidable, and the part that hurts is not the wait — it
                    // is landing at the top of a page you were working halfway
                    // down.
                    const win = frame.current?.contentWindow;
                    if (!win || !scrollY.current) return;
                    win.postMessage({ type: "keel:scrollto", y: scrollY.current }, "*");
                  }}
                />
                {/* ⚠️ **Mounted whenever live editing is on**, not only when the
                    selected band happens to be a freely drawn one: clicking the
                    thing you want to change is *how* you find it. Handles still
                    appear only where there is something to drag. */}
                {liveEdit && (
                  <PreviewOverlay
                    frame={frame}
                    // ⚠️ The same number the iframe is drawn at. Two scales here
                    // is handles that sit where the element is not.
                    zoom={siteZoom}
                    activeBand={pick.band}
                    selected={pick.el}
                    incoming={incoming}
                    canDrawIn={canDrawIn}
                    onDropElement={(bandIdx, x, y) => {
                      placeElement(bandIdx, incoming?.type ?? "text", x, y);
                      setIncoming(null);
                    }}
                    onDropBand={(at) => {
                      if (incoming?.kind !== "band") return;
                      placeBand(incoming.type, at);
                      setIncoming(null);
                    }}
                    onSelect={(el) => setPick({ band: pick.band, el })}
                    onPick={(bandIdx: number, el: number | null) => {
                      setPick({ band: bandIdx, el });
                      setSide("page");
                    }}
                    onBox={(i, patch) => moveBox(pick.band, i, patch)}
                    onDragState={(on) => {
                      // ⚠️ **The undo entry is taken here, at the start.** While
                      // the flag is up `moveBox` takes none — that is the point
                      // — so without this a drag would be unundoable rather than
                      // one step.
                      if (on && !dragging.current) remember();
                      dragging.current = on;
                      if (on) return;
                      // ⚠️ The end of a drag is where a drag's whole cost is
                      // paid: one undo entry was taken at the start, and now one
                      // save and one measurement. Everything in between was a
                      // box moving under a cursor.
                      dragTarget.current = null;
                      setSaveState("dirty");
                      window.clearTimeout(saveTimer.current);
                      saveTimer.current = window.setTimeout(
                        () => void persistRef.current(),
                        600,
                      );
                      frame.current?.contentWindow?.postMessage(
                        { type: "keel:measure" },
                        "*",
                      );
                    }}
                    // ⚠️ Nothing to do on drag end any more. The draft saves
                    // itself and the page was already told about the box — the
                    // reload this used to trigger is the thing that was fixed.
                    onCommit={() => {}}
                    boxOf={(i) => {
                      const el = band?.canvas?.elements?.[i];
                      if (!el) return null;
                      return editing === "mobile" ? (el.mobile ?? el.box) : el.box;
                    }}
                    // ⚠️ **Null for anything without words**, which is what
                    // decides whether a double-click opens a typing box.
                    textOf={(i) => {
                      const el = band?.canvas?.elements?.[i];
                      if (!el || !TEXTUAL.has(el.type)) return null;
                      return el.text?.uz ?? "";
                    }}
                    onText={(i, value) =>
                      setElementText(pick.band, i, {
                        uz: value,
                        ru: band?.canvas?.elements?.[i]?.text?.ru ?? "",
                        en: band?.canvas?.elements?.[i]?.text?.en ?? "",
                      })
                    }
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

        {/* ---- Right: the settings of whatever is selected ---- */}
        <aside className="w-full shrink-0 space-y-3 overflow-auto border-t border-line p-3 lg:w-80 lg:border-l lg:border-t-0">
          {!band && <p className="text-xs text-ink-muted">{d.inspectorEmpty}</p>}

          {band && element && pick.el != null ? (
            <>
              <InspectorHead
                title={`${ELEMENT_LABEL[element.type] ?? element.type}`}
                sub={BAND_LABEL[band.type] ?? band.type}
                onClose={() => setPick({ band: pick.band, el: null })}
                d={d}
              />
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
                onText={(value) => setElementText(pick.band, pick.el!, value)}
                onBox={(patch) => moveBox(pick.band, pick.el!, patch)}
                tenantId={tenantId}
              />
            </>
          ) : band ? (
            <>
              <InspectorHead
                title={BAND_LABEL[band.type] ?? band.type}
                sub={d.sectionSettings}
                d={d}
              />
              <BandSettings
                band={band}
                index={pick.band}
                update={update}
                setSections={setSections}
                setPick={setPick}
                schema={schema}
                categories={state?.categories}
              />
              {/* Repeatable items: gallery photos, perk cards, slides.
                  ⚠️ Blocks rather than numbered settings (`perk1Title`…):
                  numbered fields fix the count, fill the panel with empty inputs,
                  and cannot be reordered without retyping. */}
              {bandDef?.blocks?.length ? (
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
            </>
          ) : null}
        </aside>
      </div>
    </div>
  );
}

/** The draft's state, in one word.
 *
 *  ⚠️ **A dot and a word, not a spinner.** The question is "is my work safe",
 *  and a spinner answers "something is happening", which is a different
 *  question and the one nobody asked. */
function SaveState({
  state,
  d,
}: {
  state: "clean" | "dirty" | "saving" | "error";
  d: EditorDict;
}) {
  const look = {
    clean: { dot: "bg-signal-500", text: d.stateSaved, tone: "text-ink-muted" },
    dirty: { dot: "bg-ink-muted", text: d.stateDirty, tone: "text-ink-muted" },
    saving: { dot: "bg-ink-muted animate-pulse", text: d.stateSaving, tone: "text-ink-muted" },
    error: { dot: "bg-hot-600", text: d.stateDirty, tone: "text-hot-600" },
  }[state];
  return (
    <span className={`flex items-center gap-1.5 text-[11px] font-semibold ${look.tone}`}>
      <span className={`h-1.5 w-1.5 rounded-full ${look.dot}`} />
      {look.text}
    </span>
  );
}

/** The title above the inspector: what is selected, and what it is part of.
 *
 *  ⚠️ **Named rather than implied.** The panel used to open with no heading, so
 *  a column of X/Y/W/H fields was the only clue about which of forty things on
 *  the page was about to change. */
function InspectorHead({
  title,
  sub,
  onClose,
  d,
}: {
  title: string;
  sub: string;
  onClose?: () => void;
  d: EditorDict;
}) {
  return (
    <div className="flex items-start gap-2">
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-bold text-ink">{title}</p>
        <p className="truncate text-[11px] text-ink-muted">{sub}</p>
      </div>
      {onClose && (
        <button
          type="button"
          onClick={onClose}
          title={d.closeLabel}
          className="rounded-lg border border-line px-2 py-0.5 text-xs text-ink-muted hover:text-ink"
        >
          ×
        </button>
      )}
    </div>
  );
}

/** A named, collapsible group. ⚠️ The replacement for the icon rail: the tools
 *  that apply to the whole site are all in one column with their names on, and
 *  the one being used is the one that is open. */
function Fold({
  title,
  open = false,
  children,
}: {
  title: string;
  open?: boolean;
  children: React.ReactNode;
}) {
  const [on, setOn] = useState(open);
  return (
    <div className="overflow-hidden rounded-2xl border border-line">
      <button
        type="button"
        onClick={() => setOn((v) => !v)}
        className="flex w-full items-center justify-between px-3 py-2.5 text-xs font-bold text-ink"
      >
        {title}
        <span className="text-ink-muted">{on ? "−" : "+"}</span>
      </button>
      {on && <div className="border-t border-line p-3">{children}</div>}
    </div>
  );
}

/** The page as a tree: bands in order, and the elements inside a free band.
 *
 *  ⚠️ **One list, not three panels.** Bands, the band's settings and its
 *  elements used to be three stacked cards in the same scrolling column, so a
 *  design with fifteen bands pushed the elements below the fold exactly when one
 *  was selected. The settings moved to the right; what is left here is the
 *  answer to one question — what is this page made of, and in what order. */
function Outline({
  sections,
  pick,
  setPick,
  setSections,
  remember,
  d,
  bandLabel,
  elementLabel,
}: {
  sections: DesignSection[];
  pick: { band: number; el: number | null };
  setPick: (p: { band: number; el: number | null }) => void;
  setSections: React.Dispatch<React.SetStateAction<DesignSection[]>>;
  remember: () => void;
  d: EditorDict;
  bandLabel: Record<string, string>;
  elementLabel: Record<string, string>;
}) {
  function move(i: number, dir: -1 | 1) {
    const j = i + dir;
    if (j < 0 || j >= sections.length) return;
    remember();
    setSections((prev) => {
      const next = [...prev];
      [next[i], next[j]] = [next[j], next[i]];
      return next;
    });
    setPick({ band: j, el: null });
  }

  if (sections.length === 0) {
    return <p className="px-1 text-xs leading-relaxed text-ink-muted">{d.emptyPage}</p>;
  }

  return (
    <ul className="space-y-1">
      {sections.map((s, i) => {
        const active = pick.band === i;
        const elements = s.canvas?.elements ?? [];
        return (
          <li key={i}>
            {/* ⚠️ Dragged, not nudged with arrows. Moving a band from the bottom
                of a fifteen-band page to the top took fourteen presses, each one
                re-rendering the list under the cursor. The arrows stay beside the
                handle: a list that can only be reordered by dragging cannot be
                reordered with a keyboard. */}
            <div
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
                remember();
                setSections((prev) => {
                  const next = [...prev];
                  const [moved] = next.splice(from, 1);
                  next.splice(i, 0, moved);
                  return next;
                });
                // Follow the band that moved: losing the selection mid-reorder
                // means finding it again in a list that just changed shape.
                setPick({ band: i, el: null });
              }}
              className={`group flex items-center gap-1 rounded-xl px-1.5 py-1 ${
                active ? "bg-raised" : "hover:bg-raised/60"
              }`}
            >
              <span className="cursor-grab select-none px-0.5 text-ink-muted active:cursor-grabbing" aria-hidden>
                ⠿
              </span>
              <button
                type="button"
                onClick={() => setPick({ band: i, el: null })}
                className={`min-w-0 flex-1 truncate text-left text-xs font-semibold ${
                  active ? "text-ink" : "text-ink-soft"
                } ${s.hidden ? "line-through opacity-50" : ""}`}
              >
                {bandLabel[s.type] ?? s.type}
                {elements.length ? (
                  <span className="ml-1 font-normal text-ink-muted">· {elements.length}</span>
                ) : null}
              </button>
              <button
                type="button"
                onClick={() => {
                  remember();
                  setSections((prev) =>
                    prev.map((x, k) => (k === i ? { ...x, hidden: !x.hidden } : x)),
                  );
                }}
                title={s.hidden ? d.show : d.hide}
                className="px-1 text-[11px] text-ink-muted opacity-0 transition group-hover:opacity-100"
              >
                {s.hidden ? "○" : "●"}
              </button>
              <button type="button" onClick={() => move(i, -1)} title={d.bandUp} className="px-0.5 text-[11px] text-ink-muted opacity-0 transition group-hover:opacity-100">↑</button>
              <button type="button" onClick={() => move(i, 1)} title={d.bandDown} className="px-0.5 text-[11px] text-ink-muted opacity-0 transition group-hover:opacity-100">↓</button>
            </div>

            {/* The elements of the selected free band, nested under it. Only the
                selected one: every band's elements at once is the wall of rows
                this screen was rebuilt to remove. */}
            {active && s.canvas && (
              <ul className="ml-5 mt-0.5 space-y-0.5 border-l border-line pl-2">
                {elements.length === 0 && (
                  <li className="py-1 text-[11px] leading-relaxed text-ink-muted">
                    {d.emptyElements}
                  </li>
                )}
                {elements.map((e, j) => (
                  <li key={j} className="group flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setPick({ band: i, el: j })}
                      className={`min-w-0 flex-1 truncate rounded-lg px-1.5 py-1 text-left text-[11px] ${
                        pick.el === j ? "bg-raised font-semibold text-ink" : "text-ink-soft"
                      }`}
                    >
                      <span className="mr-1 text-ink-muted">{ELEMENT_GLYPH[e.type] ?? "◦"}</span>
                      {e.text?.uz ? e.text.uz.slice(0, 20) : (elementLabel[e.type] ?? e.type)}
                    </button>
                    <button
                      type="button"
                      title={d.duplicate}
                      onClick={() => {
                        remember();
                        setSections((prev) =>
                          prev.map((x, k) =>
                            k === i && x.canvas
                              ? {
                                  ...x,
                                  canvas: {
                                    ...x.canvas,
                                    elements: [
                                      ...(x.canvas.elements ?? []),
                                      // Offset, so the copy is not hidden exactly
                                      // behind the original.
                                      {
                                        ...e,
                                        box: { ...e.box, x: Math.min(95, e.box.x + 3), y: Math.min(95, e.box.y + 3) },
                                      },
                                    ],
                                  },
                                }
                              : x,
                          ),
                        );
                        setPick({ band: i, el: elements.length });
                      }}
                      className="px-1 text-[11px] text-ink-muted opacity-0 transition group-hover:opacity-100"
                    >
                      ⧉
                    </button>
                    <button
                      type="button"
                      title={d.remove}
                      onClick={() => {
                        remember();
                        setSections((prev) =>
                          prev.map((x, k) =>
                            k === i && x.canvas
                              ? {
                                  ...x,
                                  canvas: {
                                    ...x.canvas,
                                    elements: (x.canvas.elements ?? []).filter((_, m) => m !== j),
                                  },
                                }
                              : x,
                          ),
                        );
                        setPick({ band: i, el: null });
                      }}
                      className="px-1 text-[11px] text-hot-600 opacity-0 transition group-hover:opacity-100"
                    >
                      ×
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </li>
        );
      })}
    </ul>
  );
}

/** Cards that can be dragged onto the page, grouped and named.
 *
 *  ⚠️ **Dragged, and that is the point.** A click appends to the end of the
 *  band, which is what the editor could do before and what made placing a
 *  headline a three-step job: add it, find it, drag it to where it was wanted.
 *  A card dropped on the page lands where it was dropped — the gesture every
 *  page builder people have used works this way, and its absence was read as
 *  the editor not being one. */
function Palette({
  title,
  hint,
  groups,
  onDragStart,
  onDragEnd,
  onPick,
}: {
  title: string;
  hint: string;
  groups: { title: string; items: { type: string; label: string; glyph?: string }[] }[];
  onDragStart: (type: string) => void;
  onDragEnd: () => void;
  onPick: (type: string) => void;
}) {
  return (
    <div className="space-y-2 rounded-2xl border border-line p-3">
      <p className="text-xs font-bold text-ink">{title}</p>
      <p className="text-[11px] leading-relaxed text-ink-muted">{hint}</p>
      {groups.map((g) => (
        <div key={g.title} className="space-y-1">
          <p className="text-[10px] font-bold uppercase tracking-wider text-ink-muted">
            {g.title}
          </p>
          <div className="grid grid-cols-2 gap-1">
            {g.items.map((it) => (
              <button
                key={it.type}
                type="button"
                draggable
                onDragStart={(e) => {
                  e.dataTransfer.setData("text/plain", it.type);
                  e.dataTransfer.effectAllowed = "copy";
                  onDragStart(it.type);
                }}
                onDragEnd={onDragEnd}
                onClick={() => onPick(it.type)}
                className="flex cursor-grab items-center gap-1.5 rounded-xl border border-line px-2 py-1.5 text-left text-[11px] font-semibold text-ink-soft transition hover:border-signal-500 hover:text-ink active:cursor-grabbing"
              >
                {it.glyph && (
                  <span className="w-4 shrink-0 text-center text-ink-muted">{it.glyph}</span>
                )}
                <span className="truncate">{it.label}</span>
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

/** The gallery of finished layouts. */
function TemplateList({
  templates,
  d,
  onApply,
}: {
  templates: DesignTemplate[];
  d: EditorDict;
  onApply: (tpl: DesignTemplate) => void;
}) {
  return (
    <div className="space-y-2">
      <p className="text-[11px] leading-relaxed text-ink-muted">{d.templatesHint}</p>
      <ul className="space-y-1.5">
        {templates.map((tpl) => (
          <li key={tpl.id}>
            <button
              type="button"
              onClick={() => onApply(tpl)}
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
    <div className="space-y-2">

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


function ElementSettings({
  el,
  editing,
  setEditing,
  onStyle,
  onElement,
  onText,
  onBox,
  tenantId,
}: {
  el: DesignElement;
  editing: "desktop" | "mobile";
  setEditing: (v: "desktop" | "mobile") => void;
  onStyle: (patch: Record<string, unknown>) => void;
  onElement: (patch: Partial<DesignElement>) => void;
  /** ⚠️ Words go through their own path, not through `onElement`. A word is the
   *  one change the live page can be *told* about, and routing it with the rest
   *  would reload somebody's real site on every letter — which is exactly what
   *  this editor used to do. */
  onText: (value: { uz: string; ru: string; en: string }) => void;
  onBox: (patch: Partial<DesignBox>) => void;
  /** Whose uploads a chosen photograph is written into. */
  tenantId: string;
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
    <div className="space-y-2">

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
              onChange={(e) => onText({ uz: e.target.value, ru: el.text?.ru ?? "", en: el.text?.en ?? "" })}
              className="input"
            />
          </Row>
          <Row label="ru">
            <input
              value={el.text?.ru ?? ""}
              onChange={(e) => onText({ uz: el.text?.uz ?? "", ru: e.target.value, en: el.text?.en ?? "" })}
              className="input"
            />
          </Row>
          <Row label="en">
            <input
              value={el.text?.en ?? ""}
              onChange={(e) => onText({ uz: el.text?.uz ?? "", ru: el.text?.ru ?? "", en: e.target.value })}
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
          <ImageField
            value={el.image ?? ""}
            onChange={(v) => onElement({ image: v })}
            tenantId={tenantId}
            d={d}
          />
        </Row>
      )}

      {(el.type === "image" || el.type === "carousel") && (
        <>
          {/* ⚠️ **The setting a whole design language needed.** Every shop
              reference we are sent is built from shoes cut out of their
              backdrops, floating on panels and discs — and a cut-out is a
              shape, so filling the box crops the toe off it. Until this
              existed the only way to place one was a hand-written stylesheet,
              which is the console telling a designer to write CSS for the
              commonest thing they do. */}
          <Row label={d.fit}>
            <Seg
              value={(el.style?.fit ?? "") as string}
              options={[
                { v: "", label: d.fitCover },
                { v: "contain", label: d.fitContain },
              ]}
              onChange={(v) => onStyle({ fit: v })}
            />
          </Row>
          <p className="text-[11px] leading-relaxed text-ink-muted">{d.fitHint}</p>
        </>
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
    <div className="space-y-2">
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
    <div className="space-y-3">
      <p className="text-[11px] leading-relaxed text-ink-muted">{d.themeHint}</p>
      <Colour k="brand" label={d.themeBrand} hint={d.themeBrandHint} />
      <Colour k="accent" label={d.themeAccent} hint={d.themeAccentHint} />
    </div>
  );
}

/** A photograph: chosen from the operator's own machine, or a path typed by
 *  hand.
 *
 *  ⚠️ **Both, and the box stays.** Choosing a file is what somebody wants
 *  nineteen times out of twenty — the brief is a folder of pictures and the
 *  customer has uploaded nothing yet, because the site is being drawn before
 *  they have ever logged in. The box is for the twentieth: a photograph the
 *  restaurant already has, whose path is copied out of their own panel, which
 *  no upload button can reach.
 *
 *  ⚠️ **The file goes into the customer's own uploads, not ours.** A design
 *  pointing at a picture on keel.uz would be a page that breaks the day we
 *  move a file, on a site we do not own — and it would serve every one of that
 *  restaurant's visitors from our bandwidth. */
function ImageField({
  value,
  onChange,
  tenantId,
  d,
}: {
  value: string;
  onChange: (v: string) => void;
  tenantId: string;
  d: EditorDict;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function choose(file: File | undefined) {
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      const res = await uploadTenantImage(tenantId, file);
      onChange(res.path);
    } catch (e) {
      // ⚠️ The server's own sentence, not "yuklanmadi". It is the only thing
      // that distinguishes a file that is too large from one that is a PDF,
      // and both are things the operator can fix in ten seconds if told.
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-1.5">
      {/* What is actually there, if anything. A path is not a picture, and a
          field that shows only a path is a field somebody has to publish to
          check. */}
      {value ? (
        <div className="overflow-hidden rounded-xl border border-line bg-raised">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={`https://${tenantSlugHost()}${value}`}
            alt=""
            className="h-24 w-full object-cover"
            onError={(e) => {
              (e.currentTarget as HTMLImageElement).style.display = "none";
            }}
          />
        </div>
      ) : null}
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="/uploads/abc.jpg"
        spellCheck={false}
        className="input font-mono text-[11px]"
      />
      <label className="flex cursor-pointer items-center justify-center rounded-xl border border-dashed border-line py-2 text-[11px] font-semibold text-ink-muted hover:border-signal-500 hover:text-ink">
        <input
          type="file"
          accept="image/jpeg,image/png,image/webp,image/avif,image/gif"
          className="hidden"
          disabled={busy}
          onChange={(e) => {
            void choose(e.target.files?.[0]);
            // ⚠️ Cleared, so choosing the same file twice fires again. Without
            // it a failed upload cannot be retried with the same picture, and
            // the button reads as dead.
            e.target.value = "";
          }}
        />
        {busy ? d.imageUploading : d.imageChoose}
      </label>
      {error && <p className="text-[11px] text-hot-600">{error}</p>}
    </div>
  );
}

/** The host a chosen photograph will be served from.
 *
 *  ⚠️ The preview iframe already points at the tenant, so the slug is known —
 *  but reading it out of the iframe would tie this component to the pane beside
 *  it. It is taken from the preview URL the editor already holds, and when
 *  there is none the thumbnail simply does not draw, which is the honest
 *  outcome rather than a broken image. */
function tenantSlugHost(): string {
  const frame = typeof document === "undefined" ? null : document.querySelector("iframe");
  const src = frame?.getAttribute("src") ?? "";
  try {
    return new URL(src).host;
  } catch {
    return "";
  }
}
