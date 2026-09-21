import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

// Every link into the catalogue has to use the address this business answers at.
//
// ⚠️ **A link written `/menu` is not merely untidy on a shop — it is a full
// page load.** `/menu` redirects to `/catalog` there, and a `<Link>` that meets
// a server redirect stops being a client navigation: the browser throws the
// application away and loads a document. It showed up as "why does picking a
// size hard-refresh", and the same mistake was on every card in the catalogue
// and on the home page's main button.
//
// Nothing at runtime can see it: the link works, the page appears, and only the
// flash tells you. So the rule is enforced on the source.

/** Files that draw the public site. */
function siteFiles(): string[] {
  const roots = ["src/app/(site)", "src/components/menu", "src/components/site", "src/components/design"];
  const out: string[] = [];
  const walk = (dir: string) => {
    for (const name of readdirSync(dir)) {
      const path = join(dir, name);
      if (statSync(path).isDirectory()) walk(path);
      else if (/\.tsx?$/.test(name) && !name.includes(".test.")) out.push(path);
    }
  };
  for (const r of roots) walk(r);
  return out;
}

/** `/menu` written as an address a guest is sent to. */
const SENT_TO_MENU = [
  /href=\{?["'`]\/menu/,          // href="/menu" or href={`/menu/...`}
  /push\(["'`]\/menu/,             // router.push("/menu…")
  /replace\(["'`]\/menu/,
];

describe("links into the catalogue", () => {
  it("never hard-code /menu, which redirects on a shop", () => {
    const offenders: string[] = [];
    for (const file of siteFiles()) {
      const src = readFileSync(file, "utf8");
      src.split("\n").forEach((line, i) => {
        if (line.includes("catalogRoute") || line.trimStart().startsWith("//") || line.trimStart().startsWith("*")) return;
        if (SENT_TO_MENU.some((re) => re.test(line))) offenders.push(`${file}:${i + 1}  ${line.trim()}`);
      });
    }
    expect(
      offenders,
      `these send a guest to /menu, which is a redirect — and a redirect is a full page load:\n${offenders.join("\n")}`,
    ).toEqual([]);
  });

  it("is looking at real files", () => {
    // A walker that found nothing would make the assertion above pass forever.
    expect(siteFiles().length).toBeGreaterThan(20);
  });
});
