// Screenshots for the knowledge base.
//
// ⚠️ **From the demo database, never from a customer.** Every frame here shows
// a real screen with B5 Somsa's generated data: an article illustrated with a
// live restaurant's takings, phone numbers and staff names is a help page that
// leaks its own customers.
//
// ⚠️ **Every frame is taken three times, once per language.** A Russian reader
// looking at a screenshot of an Uzbek panel is being shown a screen that is not
// theirs: the words in the picture are exactly the words they are meant to find
// on their own screen, and if they cannot read them the picture is decoration.
// The panel takes its language from the `lang` cookie — it has no language URL,
// by design — so the cookie is set per browser context and the run repeats.
//
// ⚠️ **The callout coordinates are measured per language too**, and that is not
// caution: "Zagotovkalar · 3" and "Заготовки · 3" are different widths, so the
// button beside them sits somewhere else. One shared set of coordinates would
// put every Russian arrow slightly off — and slightly off is the version nobody
// reports, because it still looks like an arrow.
//
// ⚠️ **The annotations are still not drawn into the image.** The boxes are
// coordinates in `figures.json` and the words live in the article, in the
// language the article is in. Burning captions into the pictures would mean
// three *captioned* frames per screen kept in step by hand, and a translator
// who cannot reach the words at all.
//
// Usage:  cd scripts && npm install && node help-screens.mjs [name ...]
//   with the backend on :8080 (MONGO_DB=demo) and the panel on :3000:
//     go run ./cmd/adminreset -db demo -username admin -password Demo12345
//     MONGO_DB=demo go run ./cmd/server
//     cd frontend && npm run dev
//
// ⚠️ `scripts/` has its own package.json. Playwright is not a dependency of
// frontend/ or keel-site/ because their Dockerfiles run `npm ci`, which
// installs devDependencies — a browser download inside a production image.

import { chromium } from "playwright";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { acceptCookies, LANGS, langCookie, settle, toWebp } from "./shot-lib.mjs";

const BASE = process.env.PANEL ?? "http://localhost:3000";
const USER = process.env.ADMIN_USER ?? "admin";
const PASS = process.env.ADMIN_PASS ?? "Demo12345";

const HERE = path.dirname(new URL(import.meta.url).pathname);
const OUT = path.resolve(HERE, "../keel-site/public/help");
const FIGURES = path.resolve(HERE, "../keel-site/src/lib/help/figures.json");

/** One frame.
 *
 *  `notes` are the elements a callout can point at, given as `data-help`
 *  values. ⚠️ **Not visible text.** Matching a button by the words on it works
 *  perfectly in Uzbek and matches nothing in Russian — and a missed match here
 *  fails by *omitting* a callout, not by erroring, so it would ship. The
 *  attribute is a small, deliberate contract in the panel's own source. */
const SHOTS = [
  // ---- The panel ----
  { name: "dashboard", url: "/admin", notes: ["nav", "period"] },
  { name: "orders", url: "/admin/orders" },
  { name: "menu", url: "/admin/menu", notes: ["add", "uncosted"] },
  { name: "menu-item", url: "/admin/menu", act: "openFirstEdit" },
  { name: "categories", url: "/admin/categories" },
  { name: "stop-list", url: "/admin/stop-list" },
  { name: "reservations", url: "/admin/reservations" },
  { name: "promotions", url: "/admin/promotions" },
  { name: "qr", url: "/admin/qr" },

  // ---- Store ----
  { name: "stock", url: "/admin/stock" },
  { name: "shopping", url: "/admin/shopping" },
  { name: "ingredients", url: "/admin/ingredients", notes: ["unit", "expected"] },
  {
    name: "tech-cards",
    url: "/admin/tech-cards",
    notes: ["tabs", "dishes", "add", "batch", "rate"],
  },
  { name: "tech-cards-dishes", url: "/admin/tech-cards", act: "dishesTab" },
  { name: "tech-card-prep", url: "/admin/tech-cards", act: "openFirstPrep" },
  { name: "purchases", url: "/admin/purchases" },
  { name: "suppliers", url: "/admin/suppliers" },
  { name: "writeoffs", url: "/admin/writeoffs" },
  { name: "transfers", url: "/admin/transfers" },
  { name: "production", url: "/admin/production" },
  { name: "stocktake", url: "/admin/stocktake" },

  // ---- People ----
  { name: "staff", url: "/admin/staff", notes: ["add"] },
  { name: "roles", url: "/admin/roles" },
  { name: "payroll", url: "/admin/payroll" },
  { name: "couriers", url: "/admin/couriers" },
  { name: "admins", url: "/admin/admins" },
  { name: "logs", url: "/admin/logs" },

  // ---- Money and reports ----
  { name: "cash", url: "/admin/cash" },
  { name: "reports", url: "/admin/reports" },
  { name: "checks", url: "/admin/checks" },

  // ---- Customers ----
  { name: "users", url: "/admin/users" },
  { name: "feedback", url: "/admin/feedback" },
  { name: "campaigns", url: "/admin/campaigns" },
  { name: "calls", url: "/admin/calls" },

  // ---- Settings ----
  { name: "settings", url: "/admin/settings" },
  { name: "pos", url: "/admin/pos" },
  { name: "vacancies", url: "/admin/vacancies" },

  // ---- Apps ----
  { name: "kitchen-login", url: "/staff/login", anon: true },
  { name: "courier-login", url: "/kuryer/login", anon: true },
  { name: "kiosk-login", url: "/kiosk/login", anon: true },

  // ---- The guest's side ----
  //
  // ⚠️ `site: true` means the address itself carries the language (`/ru/menu`),
  // because the public site is the one half of the product that has language
  // URLs. The cookie is set as well and agrees with it; the prefix is what the
  // middleware actually reads.
  { name: "site-home", url: "/", anon: true, site: true },
  { name: "site-menu", url: "/menu", anon: true, site: true },
  { name: "site-cart", url: "/cart", anon: true, site: true },
  { name: "site-booking", url: "/bron", anon: true, site: true },
];

/** Steps that have to happen before the shutter.
 *
 *  ⚠️ These reach for `data-help` too, for the same reason the callouts do: an
 *  action keyed to Uzbek button text silently does nothing on the Russian run,
 *  and the frame that comes out is of the wrong screen rather than of no
 *  screen — which is much harder to notice in a directory of 132 files. */
const ACTS = {
  async openFirstEdit(page) {
    await page.locator('[data-help="edit"]').first().click();
    await page.waitForTimeout(1000);
  },
  async dishesTab(page) {
    await page.locator('[data-help="dishes"]').click();
    await page.waitForTimeout(800);
  },
  async openFirstPrep(page) {
    const row = page.locator("table tbody tr button").first();
    if (await row.count()) {
      await row.click();
    } else {
      await page.locator('[data-help="add"]').click();
    }
    await page.waitForTimeout(1000);
  },
};

/** Where each callout points, as a share of the frame.
 *
 *  ⚠️ **Measured from the DOM, not placed by hand on the picture.** A rectangle
 *  typed in as "about 12% from the left" is right until the next release moves
 *  a button by ten pixels, and nothing then fails — the arrow simply points at
 *  the wrong thing, in three languages, until a reader writes in.
 *
 *  ⚠️ **Percentages, not pixels**: the frame is displayed at whatever width the
 *  article column happens to be, and on a phone that is a third of the capture. */
async function measure(page, keys) {
  const out = {};
  const view = page.viewportSize();
  for (const key of keys) {
    try {
      const box = await page
        .locator(`[data-help="${key}"]`)
        .first()
        .boundingBox({ timeout: 2500 });
      if (!box) throw new Error("ko'rinmaydi");
      // ⚠️ **Clipped to the frame.** The shot is the viewport; a table that
      // runs on for another screen and a half reports its full height, and an
      // outline drawn from that spills past the bottom edge of the picture —
      // which reads as a rendering bug, not as a callout.
      const x0 = Math.max(0, box.x / view.width);
      const y0 = Math.max(0, box.y / view.height);
      const x1 = Math.min(1, (box.x + box.width) / view.width);
      const y1 = Math.min(1, (box.y + box.height) / view.height);
      if (x1 <= x0 || y1 <= y0) throw new Error("kadrdan tashqarida");
      out[key] = {
        x: +x0.toFixed(4),
        y: +y0.toFixed(4),
        w: +(x1 - x0).toFixed(4),
        h: +(y1 - y0).toFixed(4),
      };
    } catch {
      // ⚠️ Named in the run's output rather than swallowed. A missing callout
      // is invisible on the page — the legend simply has one entry fewer — so
      // the only place it can be noticed is here.
      console.log(`   ! belgi topilmadi: [data-help="${key}"]`);
    }
  }
  return out;
}

async function shootLang(browser, lang, only) {
  const ctx = await browser.newContext({
    // ⚠️ A desktop frame, at twice the pixels. The article is read on a phone
    // as often as not, and a screenshot that has to be pinched to be legible is
    // one nobody looks at twice.
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    locale: lang === "uz" ? "uz-UZ" : lang === "ru" ? "ru-RU" : "en-US",
    timezoneId: "Asia/Tashkent",
  });
  // The panel and the staff apps read this cookie and have no language URL at
  // all — see frontend/src/lib/i18n/client.tsx.
  await ctx.addCookies(langCookie(lang, BASE));
  await acceptCookies(ctx);

  const page = await ctx.newPage();

  // Sign in once per language; the token lives in the context for every frame.
  //
  // ⚠️ **Wait for hydration before typing.** The form renders on the server and
  // React takes over a moment later, resetting both fields to its own empty
  // state — so a fill that lands first is discarded, the submit posts nothing,
  // and the run produces forty screenshots of the login page. Which is exactly
  // what the first run produced.
  await page.goto(`${BASE}/admin/login`, { waitUntil: "networkidle" });
  await page.waitForTimeout(2500);
  await page.locator("input").first().fill(USER);
  await page.locator('input[type="password"]').fill(PASS);
  await page.locator('button[type="submit"]').click();
  // ⚠️ Not a URL pattern: `/admin/login` matches everything `/admin…` does, so
  // a refused password looked exactly like a successful sign-in and the run
  // went on to photograph the login form forty times, cheerfully, with a tick
  // beside each one.
  await page
    .locator('input[type="password"]')
    .waitFor({ state: "detached", timeout: 20000 });
  await page.waitForTimeout(2500);
  if (page.url().includes("/admin/login")) {
    throw new Error("kirish bo'lmadi — ADMIN_USER / ADMIN_PASS ni tekshiring");
  }

  const figures = {};
  let ok = 0;
  const failed = [];

  for (const shot of SHOTS) {
    if (only.length && !only.includes(shot.name)) continue;
    try {
      // Uzbek is the unprefixed base on the public site, as it is on keel.uz.
      const prefix = shot.site && lang !== "uz" ? `/${lang}` : "";
      await page.goto(BASE + prefix + shot.url, {
        waitUntil: "networkidle",
        timeout: 30000,
      });
      await settle(page, shot.slow ? 3500 : 1800);
      if (shot.act) await ACTS[shot.act](page);

      const file = path.join(OUT, `${shot.name}.${lang}`);
      await page.screenshot({ path: `${file}.png` });
      const size = page.viewportSize();
      figures[shot.name] = {
        w: size.width,
        h: size.height,
        ...(shot.notes ? { notes: await measure(page, shot.notes) } : {}),
      };
      await toWebp(file, 1440);
      ok++;
      console.log(`   ✓ ${shot.name}`);
    } catch (e) {
      failed.push(shot.name);
      console.log(`   ✗ ${shot.name}`, String(e).split("\n")[0]);
    }
  }

  await ctx.close();
  return { figures, ok, failed };
}

async function main() {
  const only = process.argv.slice(2);
  await mkdir(OUT, { recursive: true });

  const browser = await chromium.launch();
  const all = {};
  let ok = 0;
  const failed = [];

  for (const lang of LANGS) {
    console.log(`\n── ${lang} ──`);
    const res = await shootLang(browser, lang, only);
    ok += res.ok;
    failed.push(...res.failed.map((n) => `${n} (${lang})`));
    for (const [name, spec] of Object.entries(res.figures)) {
      (all[name] ??= {})[lang] = spec;
    }
  }
  await browser.close();

  // ⚠️ Merged into what is already there, and merged *per language*, because a
  // partial run (one frame being re-taken in one language) must not delete the
  // measurements of the other hundred and thirty-one.
  let existing = {};
  try {
    existing = JSON.parse(await readFile(FIGURES, "utf8"));
  } catch {}
  for (const [name, byLang] of Object.entries(all)) {
    existing[name] = { ...(existing[name] ?? {}), ...byLang };
  }
  const ordered = Object.fromEntries(
    Object.keys(existing)
      .sort()
      .map((k) => [k, existing[k]]),
  );
  await writeFile(FIGURES, JSON.stringify(ordered, null, 2) + "\n");

  console.log(
    `\n${ok} ta kadr olindi` +
      (failed.length ? `, ${failed.length} ta yiqildi: ${failed.join(", ")}` : ""),
  );
}

main();
