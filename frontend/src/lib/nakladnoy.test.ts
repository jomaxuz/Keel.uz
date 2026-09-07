import { describe, expect, it } from "vitest";

import { adminEn, adminRu, adminUz } from "@/lib/i18n/admin";
import { printNakladnoy } from "@/lib/nakladnoy";
import type { Dispatch } from "@/lib/types";

function vanTo(branch: string, lines: number): Dispatch {
  return {
    id: `id-${branch}`,
    fromBranchId: "central",
    toBranchId: branch,
    number: `07.09/${lines}`,
    at: "2026-09-07T08:00:00+05:00",
    value: 100000,
    by: "Yo'ldashev.Z",
    driver: "01 A 777 AA",
    lines: Array.from({ length: lines }, (_, i) => ({
      ingredientId: `ing-${i}`,
      name: i === 0 ? "Go'sht (marinovka)" : `Mahsulot ${i}`,
      unit: "kg",
      qty: 10 + i,
    })),
  };
}

/** ⚠️ The page is cleared first and the *last* frame is read: the module leaves
 *  its frame in the document for a minute on purpose (some engines print a
 *  blank page if it is torn down under the dialog), so a test reading the first
 *  one reads the previous test's paper. */
function print(
  slips: { dispatch: Dispatch; to: string; from: string }[],
  t = adminUz,
) {
  document.body.innerHTML = "";
  const printed = printNakladnoy(slips, t);
  const frames = document.querySelectorAll("iframe");
  const frame = frames[frames.length - 1] as HTMLIFrameElement | undefined;
  return { printed, doc: frame?.contentDocument ?? null };
}

describe("the slip that travels with the van", () => {
  // ⚠️ **The empty rows are not padding.** A form with exactly as many lines as
  // the system knows about cannot take the two crates a branch rings about
  // while the van is loading — and then the addition is written in the margin,
  // or not written at all. The paper form this copies has them, and so does
  // this.
  it("prints empty numbered rows under what the system knows", () => {
    const { doc } = print([{ dispatch: vanTo("a", 3), to: "Sergili", from: "Markaz" }]);
    const rows = doc?.querySelectorAll("tbody tr") ?? [];
    expect(rows.length).toBeGreaterThan(10);
    // The last row is blank but numbered, so a handwritten crate lands in a
    // numbered row and the count at the bottom stays checkable.
    const last = rows[rows.length - 1];
    expect(last.children[0].textContent).toBe(String(rows.length));
    expect(last.children[1].textContent).toBe("");
  });

  // ⚠️ **Four slips to a sheet, and a slip is never split across two pages** —
  // a slip split in half is a slip nobody can sign. The fifth branch starts a
  // new sheet.
  it("puts four slips on a sheet and starts a new one for the fifth", () => {
    const slips = ["a", "b", "c", "d", "e"].map((b) => ({
      dispatch: vanTo(b, 2),
      to: b,
      from: "Markaz",
    }));
    const { doc } = print(slips);
    const sheets = doc?.querySelectorAll(".sheet") ?? [];
    expect(sheets.length).toBe(2);
    expect(sheets[0].querySelectorAll(".slip").length).toBe(4);
    expect(sheets[1].querySelectorAll(".slip").length).toBe(1);
  });

  // ⚠️ **The language is the receiver's, not the panel's.** The storekeeper
  // works in one language and the branch foreman signing at the far end may
  // read another, and that changes from van to van — so the dictionary is an
  // argument rather than a hook.
  it("prints in whichever language was asked for", () => {
    for (const [dict, word] of [
      [adminUz, adminUz.dispatch.slip.storekeeper],
      [adminRu, adminRu.dispatch.slip.storekeeper],
      [adminEn, adminEn.dispatch.slip.storekeeper],
    ] as const) {
      const { doc } = print(
        [{ dispatch: vanTo("a", 1), to: "Sergili", from: "Markaz" }],
        dict,
      );
      expect(doc?.body.textContent).toContain(word);
    }
  });

  // ⚠️ **Three signatures, and each one is a different question**: what was
  // loaded, who took it, what arrived. A slip missing one of them cannot say
  // who to ask when a crate is short, which is the whole reason the paper
  // exists.
  it("carries all three signature lines", () => {
    const { doc } = print([{ dispatch: vanTo("a", 2), to: "Sergili", from: "Markaz" }]);
    const text = doc?.body.textContent ?? "";
    for (const label of [
      adminUz.dispatch.slip.storekeeper,
      adminUz.dispatch.slip.driver,
      adminUz.dispatch.slip.receiver,
    ]) {
      expect(text).toContain(label);
    }
    // And the names the system already knows are printed beside them — the pen
    // only adds the signature.
    expect(text).toContain("Yo'ldashev.Z");
    expect(text).toContain("01 A 777 AA");
  });

  // ⚠️ **A dish called "<b>" is a dish, not markup.** The rows carry the
  // restaurant's own ingredient names, which is exactly the text nobody
  // sanitises — the receipt printer follows the same rule.
  it("writes names as text rather than as markup", () => {
    const van = vanTo("a", 1);
    van.lines[0].name = "<img src=x onerror=alert(1)>";
    const { doc } = print([{ dispatch: van, to: "Sergili", from: "Markaz" }]);
    expect(doc?.querySelectorAll("img").length ?? 0).toBe(0);
    expect(doc?.body.textContent).toContain("<img src=x onerror=alert(1)>");
  });

  // Nothing to print is not an empty sheet: a printer that spits out a blank
  // page reads as a broken button.
  it("prints nothing when there is nothing to print", () => {
    expect(printNakladnoy([], adminUz)).toBe(false);
  });
});
