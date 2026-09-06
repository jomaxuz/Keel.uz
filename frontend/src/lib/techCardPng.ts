// A technical card as a sheet of A4, drawn to a canvas.
//
// ⚠️ **Drawn rather than screenshotted.** Turning the panel's own markup into
// an image needs a library that re-implements a browser's layout, and what
// comes out depends on the fonts, the theme and the zoom of whoever pressed
// the button — three cards printed from three machines that do not match. A
// canvas is the same picture everywhere, at a size measured in millimetres
// rather than in whatever the screen happened to be.
//
// ⚠️ **Nothing here computes a cost the server has not.** Line rates come from
// `ratePerUnit`, the one rule the editor already prices with — a second
// implementation is how a printed card and a report end up disagreeing about
// the same dish, on paper, in front of a supplier.
//
// ⚠️ **Millimetres, then pixels.** Every position below is in millimetres of a
// real A4 page and multiplied once by a scale. Sizing in pixels means a card
// that looks right at 150 DPI and wrong at 300, and the second one is the one
// somebody prints.

/** A4 in millimetres. */
const A4 = { w: 210, h: 297 };
const MARGIN = 14;

export type CardColumn = "no" | "name" | "qty" | "rate" | "cost";

export type CardDesign = {
  /** Which language the sheet is written in. */
  lang: "uz" | "ru" | "en";
  /** The colour of the rules and the heading. */
  accent: string;
  logo: boolean;
  columns: CardColumn[];
  /** Lines for a signature and a date at the foot — what makes it a document
   *  somebody signs rather than a printout. */
  signatures: boolean;
  /** Dots per inch. 150 is a good screen and a fine print; 300 is a print
   *  somebody is going to frame. */
  dpi: 150 | 300;
};

export const DEFAULT_DESIGN: CardDesign = {
  lang: "uz",
  accent: "#e2590d",
  logo: true,
  columns: ["no", "name", "qty", "rate", "cost"],
  signatures: true,
  dpi: 150,
};

export type CardLine = {
  name: string;
  qty: number;
  /** "g", "ml", "pcs" — already the recipe unit, not the purchase unit. */
  unit: string;
  /** What one of that unit costs. From the server's rate; see the note above. */
  rate: number;
};

export type CardData = {
  /** The brand, as it appears on the sheet. */
  brand: string;
  logoUrl?: string;
  dish: string;
  /** "Chiqishi: 320 g" — already formatted, because the unit words are the
   *  caller's dictionary rather than this file's. */
  yield?: string;
  lines: CardLine[];
  /** The total the server computed. ⚠️ Passed in, never summed here: the
   *  rounding rule is the server's and a sheet that rounded per line would
   *  print a different total from every report. */
  cost: number;
  /** What the dish sells for, when it is a dish. */
  price?: number;
};

/** The words on the sheet. ⚠️ Here rather than in the panel's dictionary: this
 *  file draws them, and a sheet that half-translated because a key was missing
 *  would be a document in two languages. */
export const CARD_WORDS = {
  uz: {
    title: "Texnologik karta",
    no: "№",
    name: "Nomi",
    qty: "Miqdori",
    rate: "Narxi",
    cost: "Summa",
    total: "Jami tannarx",
    price: "Sotuv narxi",
    margin: "Ustama",
    yield: "Chiqishi",
    date: "Sana",
    signedBy: "Tuzdi",
    approvedBy: "Tasdiqladi",
    page: "bet",
    file: "Texnologik-karta",
  },
  ru: {
    title: "Технологическая карта",
    no: "№",
    name: "Наименование",
    qty: "Количество",
    rate: "Цена",
    cost: "Сумма",
    total: "Себестоимость",
    price: "Цена продажи",
    margin: "Наценка",
    yield: "Выход",
    date: "Дата",
    signedBy: "Составил",
    approvedBy: "Утвердил",
    page: "стр.",
    file: "Тех-карта",
  },
  en: {
    title: "Technical card",
    no: "#",
    name: "Ingredient",
    qty: "Quantity",
    rate: "Rate",
    cost: "Cost",
    total: "Cost price",
    price: "Sale price",
    margin: "Margin",
    yield: "Yield",
    date: "Date",
    signedBy: "Prepared by",
    approvedBy: "Approved by",
    page: "page",
    file: "Tech-card",
  },
} as const;

/** Whole so'm, grouped. ⚠️ Not `Intl`: ICU data differs between browsers and
 *  builds, so the same card would print two spellings of one number. */
function money(n: number): string {
  const s = Math.round(n).toString();
  let out = "";
  for (let i = 0; i < s.length; i++) {
    if (i > 0 && (s.length - i) % 3 === 0) out += " ";
    out += s[i];
  }
  return out;
}

/** A quantity, keeping a fraction only where there is one. */
function qty(n: number): string {
  const r = Math.round(n * 100) / 100;
  return r === Math.floor(r) ? String(r) : String(r).replace(/0+$/, "");
}

async function loadLogo(url?: string): Promise<HTMLImageElement | null> {
  if (!url) return null;
  return new Promise((resolve) => {
    const img = new Image();
    // ⚠️ **Anonymous, or the canvas is tainted and `toBlob` throws.** The logo
    // is served from the same origin today; a restaurant that moves its
    // uploads to a CDN tomorrow would otherwise turn a working button into an
    // error nobody could explain.
    img.crossOrigin = "anonymous";
    img.onload = () => resolve(img);
    // A missing logo is a sheet without a logo, never a failed download.
    img.onerror = () => resolve(null);
    img.src = url;
  });
}

/** How many ingredient rows fit on one page at this row height. */
function rowsPerPage(mm: (n: number) => number, rowH: number): number {
  const top = 52; // everything above the table, in millimetres
  const bottom = 42; // totals and signatures
  return Math.max(1, Math.floor((A4.h - top - bottom) / rowH));
}

/** Draw every page of one card. Returns one blob per A4 sheet. */
export async function drawTechCard(
  data: CardData,
  design: CardDesign,
): Promise<Blob[]> {
  const w = CARD_WORDS[design.lang];
  const scale = design.dpi / 25.4; // pixels per millimetre
  const mm = (n: number) => n * scale;
  const logo = design.logo ? await loadLogo(data.logoUrl) : null;

  // ⚠️ **The row height shrinks to fit before a second page is started.** A
  // card of eighteen lines on two sheets is a card somebody loses half of; one
  // slightly tighter sheet is what anybody would have done by hand.
  let rowH = 8;
  while (rowH > 6 && data.lines.length > rowsPerPage(mm, rowH)) rowH -= 0.5;
  const perPage = rowsPerPage(mm, rowH);
  const pages = Math.max(1, Math.ceil(data.lines.length / perPage));

  const out: Blob[] = [];
  for (let page = 0; page < pages; page++) {
    const canvas = document.createElement("canvas");
    canvas.width = Math.round(mm(A4.w));
    canvas.height = Math.round(mm(A4.h));
    const c = canvas.getContext("2d");
    if (!c) continue;

    c.fillStyle = "#ffffff";
    c.fillRect(0, 0, canvas.width, canvas.height);
    c.textBaseline = "middle";

    const left = MARGIN;
    const right = A4.w - MARGIN;
    let y = MARGIN;

    // ---- The head ----
    if (logo) {
      // Fitted into a box rather than scaled to a width: a tall logo and a wide
      // one both have to leave the heading where it is.
      const box = 16;
      const ratio = logo.width / logo.height;
      const lw = ratio > 1 ? box : box * ratio;
      const lh = ratio > 1 ? box / ratio : box;
      c.drawImage(logo, mm(left), mm(y), mm(lw), mm(lh));
    }
    const textLeft = logo ? left + 20 : left;
    c.fillStyle = "#111111";
    c.font = `bold ${mm(5.5)}px sans-serif`;
    c.fillText(data.brand, mm(textLeft), mm(y + 5));
    c.fillStyle = design.accent;
    c.font = `bold ${mm(7)}px sans-serif`;
    c.fillText(w.title, mm(textLeft), mm(y + 13));

    y += 22;
    c.strokeStyle = design.accent;
    c.lineWidth = Math.max(1, mm(0.6));
    c.beginPath();
    c.moveTo(mm(left), mm(y));
    c.lineTo(mm(right), mm(y));
    c.stroke();

    y += 8;
    c.fillStyle = "#111111";
    c.font = `bold ${mm(6)}px sans-serif`;
    c.fillText(data.dish, mm(left), mm(y));

    y += 8;
    c.fillStyle = "#555555";
    c.font = `${mm(3.6)}px sans-serif`;
    const head: string[] = [];
    if (data.yield) head.push(`${w.yield}: ${data.yield}`);
    head.push(`${w.date}: ${new Date().toLocaleDateString("ru-RU")}`);
    if (pages > 1) head.push(`${w.page} ${page + 1}/${pages}`);
    c.fillText(head.join("   ·   "), mm(left), mm(y));

    // ---- The table ----
    y += 8;
    const cols = design.columns;
    const widths: Record<CardColumn, number> = {
      no: 10,
      name: 0, // whatever is left
      qty: 26,
      rate: 30,
      cost: 32,
    };
    const fixed = cols
      .filter((k) => k !== "name")
      .reduce((s, k) => s + widths[k], 0);
    widths.name = right - left - fixed;

    const xs: Record<string, number> = {};
    let x = left;
    for (const k of cols) {
      xs[k] = x;
      x += widths[k];
    }

    c.fillStyle = "#f4f4f5";
    c.fillRect(mm(left), mm(y), mm(right - left), mm(rowH));
    c.fillStyle = "#333333";
    c.font = `bold ${mm(3.4)}px sans-serif`;
    for (const k of cols) {
      const label = w[k as keyof typeof w] as string;
      const rightAligned = k !== "name" && k !== "no";
      c.textAlign = rightAligned ? "right" : "left";
      c.fillText(
        label,
        mm(rightAligned ? xs[k] + widths[k] - 2 : xs[k] + 2),
        mm(y + rowH / 2),
      );
    }
    y += rowH;

    const slice = data.lines.slice(page * perPage, (page + 1) * perPage);
    c.font = `${mm(3.4)}px sans-serif`;
    slice.forEach((line, i) => {
      const n = page * perPage + i + 1;
      // ⚠️ Every other row shaded rather than a rule under each: a grid of
      // lines on a printer that streaks is unreadable, and shading survives it.
      if (i % 2 === 1) {
        c.fillStyle = "#fafafa";
        c.fillRect(mm(left), mm(y), mm(right - left), mm(rowH));
      }
      c.fillStyle = "#111111";
      for (const k of cols) {
        const rightAligned = k !== "name" && k !== "no";
        c.textAlign = rightAligned ? "right" : "left";
        const tx = mm(rightAligned ? xs[k] + widths[k] - 2 : xs[k] + 2);
        const ty = mm(y + rowH / 2);
        if (k === "no") c.fillText(String(n), tx, ty);
        if (k === "name") c.fillText(fit(c, line.name, mm(widths.name - 4)), tx, ty);
        if (k === "qty") c.fillText(`${qty(line.qty)} ${line.unit}`, tx, ty);
        if (k === "rate") c.fillText(money(line.rate), tx, ty);
        if (k === "cost") c.fillText(money(line.rate * line.qty), tx, ty);
      }
      y += rowH;
    });

    c.textAlign = "left";
    c.strokeStyle = "#dddddd";
    c.lineWidth = Math.max(1, mm(0.3));
    c.beginPath();
    c.moveTo(mm(left), mm(y));
    c.lineTo(mm(right), mm(y));
    c.stroke();

    // ---- The foot, on the last page only ----
    if (page === pages - 1) {
      y += 10;
      c.textAlign = "right";
      c.fillStyle = "#111111";
      c.font = `bold ${mm(4.6)}px sans-serif`;
      c.fillText(`${w.total}: ${money(data.cost)}`, mm(right), mm(y));
      if (data.price && data.price > 0) {
        y += 7;
        c.font = `${mm(3.8)}px sans-serif`;
        c.fillStyle = "#555555";
        const margin = data.price > 0
          ? Math.round(((data.price - data.cost) / data.price) * 100)
          : 0;
        c.fillText(
          `${w.price}: ${money(data.price)}   ·   ${w.margin}: ${margin}%`,
          mm(right),
          mm(y),
        );
      }
      c.textAlign = "left";

      if (design.signatures) {
        y += 16;
        c.strokeStyle = "#999999";
        c.fillStyle = "#555555";
        c.font = `${mm(3.4)}px sans-serif`;
        const half = (right - left) / 2 - 6;
        for (const [i, label] of [w.signedBy, w.approvedBy].entries()) {
          const sx = left + i * (half + 12);
          c.beginPath();
          c.moveTo(mm(sx), mm(y));
          c.lineTo(mm(sx + half), mm(y));
          c.stroke();
          c.fillText(label, mm(sx), mm(y + 5));
        }
      }
    }

    const blob = await new Promise<Blob | null>((res) =>
      canvas.toBlob(res, "image/png"),
    );
    if (blob) out.push(blob);
  }
  return out;
}

/** Cut a name to the width it has, with an ellipsis where it was cut. */
function fit(c: CanvasRenderingContext2D, text: string, max: number): string {
  if (c.measureText(text).width <= max) return text;
  let s = text;
  while (s.length > 1 && c.measureText(`${s}…`).width > max) s = s.slice(0, -1);
  return `${s}…`;
}

/** What the file is called when it lands in somebody's downloads.
 *
 *  ⚠️ **Named in the language the sheet is written in**, because the sheet and
 *  its file are one thing: a Russian card arriving as "Texnologik-karta" is a
 *  file somebody renames before sending it on.
 *
 *  ⚠️ **Only the characters a filesystem accepts.** A slash in a dish name is a
 *  directory on one operating system and an error on another, and "Choy 1/2"
 *  is an ordinary thing to call a dish. */
export function cardFileName(
  dish: string,
  lang: CardDesign["lang"],
  page?: number,
  pages?: number,
): string {
  const clean = dish
    .replace(/[/\\:*?"<>|]+/g, " ")
    .trim()
    .replace(/\s+/g, "-");
  const base = `${CARD_WORDS[lang].file}-${clean}`;
  const suffix = pages && pages > 1 ? `-${page}` : "";
  return `${base}${suffix}.png`.slice(0, 120);
}
