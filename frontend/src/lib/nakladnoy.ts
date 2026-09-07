// The slip that travels with the van.
//
// ⚠️ **This is a piece of paper before it is a screen.** It is printed in the
// store room, carried by a driver, signed three times and kept in a folder —
// so the layout is copied from the form restaurants already use rather than
// designed afresh: four narrow slips across one landscape sheet, one per
// receiving branch, each with the date at the top, a numbered table of raw
// materials, and empty rows underneath for what somebody adds by hand at the
// last minute.
//
// ⚠️ **The empty rows are not padding.** A form with exactly as many lines as
// the system knows about is a form that cannot take the two crates the branch
// rings about while the van is loading — and then the addition is written in
// the margin, or not written at all.
//
// ⚠️ **The language is chosen at the moment of printing, not by the panel.**
// The storekeeper works in Uzbek and the branch foreman signing at the far end
// may read Russian; whose language the paper is in is a question about the
// person receiving it, and the answer changes from van to van. So the print
// button asks, and this module takes the dictionary rather than the hook.
//
// ⚠️ **Built as DOM nodes, never as an HTML string.** The rows carry the
// restaurant's own ingredient names, which is exactly the text nobody
// sanitises — the same rule the receipt printer follows (lib/print.ts).

import type { AdminDict } from "@/lib/i18n/admin";
import type { Dispatch } from "@/lib/types";

/** One slip: a van load and the branch it is for. */
export type Slip = {
  dispatch: Dispatch;
  to: string;
  /** The store it left, printed so a folder of slips says where each came
   *  from without anybody having to remember. */
  from: string;
};

/** How many printed rows a slip has, whatever the load. See the note above. */
const ROWS = 18;

/** How many slips fit across one landscape sheet. Four is what the form this
 *  copies uses, and it is what a chain of four branches needs on one morning. */
const PER_SHEET = 4;

/** "07" SENTABR 2026 — the way the form writes a date, in the chosen
 *  language.
 *
 *  ⚠️ **The month is spelled out and comes from the dictionary**, because
 *  `toLocaleDateString` renders a long Uzbek month as "M09" — which turns the
 *  one line the whole slip is filed by into a serial number. The names are the
 *  ones the calendar screens already use; a second copy here would be a second
 *  thing to keep in step for no gain. */
function slipDate(at: string, t: AdminDict): string {
  const d = new Date(at);
  const day = String(d.getDate()).padStart(2, "0");
  return `" ${day} " ${t.staff.months[d.getMonth()].toUpperCase()} ${d.getFullYear()}`;
}

/**
 * Print delivery notes: one slip per van, four to a sheet.
 *
 * ⚠️ **An off-screen iframe rather than a new window.** A popup is blocked by
 * default in the browser a storekeeper actually has, and the failure looks
 * exactly like a printer that is out of paper. Off-screen rather than
 * `display: none`, because a hidden frame is not printed by every engine.
 */
export function printNakladnoy(slips: Slip[], t: AdminDict): boolean {
  if (slips.length === 0) return false;

  const frame = document.createElement("iframe");
  frame.setAttribute("aria-hidden", "true");
  frame.style.position = "fixed";
  frame.style.right = "100%";
  frame.style.bottom = "100%";
  frame.style.width = "0";
  frame.style.height = "0";
  frame.style.border = "0";
  document.body.appendChild(frame);

  const doc = frame.contentDocument;
  if (!doc) {
    frame.remove();
    return false;
  }
  doc.open();
  doc.write(`<!doctype html><html><head><meta charset="utf-8">
<title>${escapeTitle(t.dispatch.slip.title)}</title>
<style>
  /* ⚠️ Landscape, because four slips across a portrait sheet leaves each one
     too narrow for a name like "Go'sht (marinovka)" — and a name that wraps
     onto two lines costs a row somebody needed. */
  @page { size: A4 landscape; margin: 8mm; }
  html, body { margin: 0; padding: 0; }
  body {
    font-family: Arial, "Helvetica Neue", sans-serif;
    font-size: 9pt;
    color: #000;
  }
  .sheet {
    display: flex;
    gap: 4mm;
    /* ⚠️ Each sheet is its own page, and the break is on the container rather
       than between slips: a slip split across two pages is a slip that cannot
       be signed. */
    page-break-after: always;
    break-after: page;
  }
  .sheet:last-child { page-break-after: auto; break-after: auto; }
  .slip {
    flex: 1 1 0;
    min-width: 0;
    border: 1pt solid #000;
    padding: 2mm;
    display: flex;
    flex-direction: column;
  }
  .head {
    border: 1pt solid #000;
    padding: 1.2mm 1.5mm;
    font-weight: bold;
    font-size: 7.5pt;
    margin-bottom: 1.5mm;
  }
  /* ⚠️ **Two fixed lines, always.** On one line the longest branch name in the
     longest language ("Филиал Beshqayrag'och — дата: ...") runs past the edge
     of a 70mm slip and takes the year with it — and the date is the one thing
     the whole slip is filed by. Two lines never clip, and being fixed they keep
     the four slips lining up across the sheet; a header that wraps on one of
     them and not the next reads as four different forms. */
  .head b, .head span { display: block; white-space: nowrap; overflow: hidden; }
  .head span { font-weight: normal; }
  table { width: 100%; border-collapse: collapse; table-layout: fixed; }
  th, td {
    border: 0.7pt solid #000;
    padding: 0.6mm 0.8mm;
    /* ⚠️ **The name has to fit on one line**, or the row grows and the slips
       stop lining up across the sheet — and a form whose rows are different
       heights is one nobody can run a finger across. "Go'sht (marinovka)" is
       the longest name a somsa shop writes, and it fits at this size. */
    font-size: 7.5pt;
    /* A tall empty row is what makes a handwritten addition legible. */
    height: 5.2mm;
    vertical-align: middle;
    overflow: hidden;
  }
  th { font-weight: bold; text-align: center; font-size: 7pt; }
  td.n, th.n { width: 6mm; text-align: center; }
  td.unit, th.unit { width: 10mm; text-align: center; }
  td.qty, th.qty { width: 11mm; text-align: right; }
  .sign { margin-top: 3mm; font-size: 8pt; }
  .sign div { margin-top: 3.5mm; display: flex; justify-content: space-between; gap: 2mm; }
  .sign b { font-weight: bold; }
  .line { flex: 0 0 22mm; border-bottom: 0.7pt solid #000; }
  .note { margin-top: 1.5mm; font-size: 7.5pt; }
</style></head><body></body></html>`);
  doc.close();

  const body = doc.body;
  for (let i = 0; i < slips.length; i += PER_SHEET) {
    const sheet = doc.createElement("div");
    sheet.className = "sheet";
    for (const slip of slips.slice(i, i + PER_SHEET)) {
      sheet.appendChild(buildSlip(doc, slip, t));
    }
    body.appendChild(sheet);
  }

  frame.contentWindow?.focus();
  frame.contentWindow?.print();
  // ⚠️ Removed on a timer rather than immediately: some engines return from
  // `print()` before the dialog has read the document, and a frame torn down
  // underneath it prints a blank page.
  window.setTimeout(() => frame.remove(), 60_000);
  return true;
}

function buildSlip(doc: Document, slip: Slip, t: AdminDict): HTMLElement {
  const el = doc.createElement("div");
  el.className = "slip";

  const head = doc.createElement("div");
  head.className = "head";
  const who = doc.createElement("b");
  who.textContent = `${t.dispatch.slip.branch} ${slip.to}`;
  const when = doc.createElement("span");
  when.textContent = `${t.dispatch.slip.date}: ${slipDate(slip.dispatch.at, t)}`;
  head.append(who, when);
  el.appendChild(head);

  const table = doc.createElement("table");
  const thead = doc.createElement("thead");
  const hr = doc.createElement("tr");
  for (const [text, cls] of [
    ["№", "n"],
    [t.dispatch.slip.item, "name"],
    [t.dispatch.slip.unit, "unit"],
    [t.dispatch.slip.qty, "qty"],
  ] as const) {
    const th = doc.createElement("th");
    th.textContent = text;
    if (cls) th.className = cls;
    hr.appendChild(th);
  }
  thead.appendChild(hr);
  table.appendChild(thead);

  const tbody = doc.createElement("tbody");
  const lines = slip.dispatch.lines;
  for (let i = 0; i < Math.max(ROWS, lines.length); i++) {
    const tr = doc.createElement("tr");
    const line = lines[i];
    const cells: [string, string][] = line
      ? [
          [String(i + 1), "n"],
          [line.name, ""],
          [line.unit, "unit"],
          [String(line.qty), "qty"],
        ]
      : [
          // ⚠️ The number is printed on the empty rows too: a person writing a
          // crate in by hand at the last minute is writing into a numbered row,
          // and the count at the bottom of the slip stays checkable.
          [String(i + 1), "n"],
          ["", ""],
          ["", "unit"],
          ["", "qty"],
        ];
    for (const [text, cls] of cells) {
      const td = doc.createElement("td");
      td.textContent = text;
      if (cls) td.className = cls;
      tr.appendChild(td);
    }
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);
  el.appendChild(table);

  if (slip.dispatch.note) {
    const note = doc.createElement("div");
    note.className = "note";
    note.textContent = slip.dispatch.note;
    el.appendChild(note);
  }

  // ⚠️ **Three signatures, and each one is a different question.** The
  // storekeeper says what was loaded, the driver says they took it, the branch
  // says what arrived — which is why a discrepancy has somebody to ask rather
  // than being an anonymous shortfall at the next count. The names the system
  // knows are printed; the line beside each is where the pen goes.
  const sign = doc.createElement("div");
  sign.className = "sign";
  const rows: [string, string][] = [
    [t.dispatch.slip.storekeeper, slip.dispatch.by ?? ""],
    [t.dispatch.slip.driver, slip.dispatch.driver ?? ""],
    [t.dispatch.slip.receiver, ""],
  ];
  for (const [label, name] of rows) {
    const row = doc.createElement("div");
    const who = doc.createElement("span");
    const b = doc.createElement("b");
    b.textContent = label;
    who.appendChild(b);
    if (name) who.append(` ${name}`);
    const line = doc.createElement("span");
    line.className = "line";
    row.append(who, line);
    sign.appendChild(row);
  }
  el.appendChild(sign);
  return el;
}

/** The document title goes into an HTML string, so it is the one value here
 *  that has to be escaped rather than set as a text node. */
function escapeTitle(s: string): string {
  return s.replace(/[<>&"]/g, (c) => `&#${c.charCodeAt(0)};`);
}
