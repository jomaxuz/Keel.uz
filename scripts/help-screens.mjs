// Screenshots for the knowledge base.
//
// ⚠️ **From the demo database, never from a customer.** Every frame here shows
// a real screen with B5 Somsa's generated data: an article illustrated with a
// live restaurant's takings, phone numbers and staff names is a help page that
// leaks its own customers.
//
// ⚠️ **The annotations are not drawn into the PNG.** They are coordinates in
// the article data and text in the dictionary, drawn over the image by the
// page — because the knowledge base is trilingual, and a caption baked into a
// screenshot needs three of every image, kept in step by hand, forever.
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
import sharp from "sharp";
import { mkdir, readFile, unlink, writeFile } from "node:fs/promises";
import path from "node:path";

const BASE = process.env.PANEL ?? "http://localhost:3000";
const USER = process.env.ADMIN_USER ?? "admin";
const PASS = process.env.ADMIN_PASS ?? "Demo12345";
// Relative to this file, not to the shell: the script is run from `scripts/`
// (that is where its node_modules live) and writes into the site next door.
const HERE = path.dirname(new URL(import.meta.url).pathname);
const OUT = path.resolve(HERE, "../keel-site/public/help");

/** One frame. `wait` is text that has to be on screen before the shutter — a
 *  screenshot of a spinner is worse than no screenshot, and it is the failure
 *  mode a fixed delay produces on a slow morning. */
const SHOTS = [
  // ---- The panel ----
  {
    name: "dashboard",
    url: "/admin",
    notes: {
      nav: 'aside, nav >> nth=0',
      period: 'button:has-text("7 kun"), button:has-text("Hafta") >> nth=0',
    },
  },
  { name: "orders", url: "/admin/orders" },
  {
    name: "menu",
    url: "/admin/menu",
    notes: {
      add: 'button:has-text("+ Yangi taom")',
      uncosted: 'button:has-text("tannarx yo\'q")',
    },
  },
  { name: "menu-item", url: "/admin/menu", act: "openFirstEdit" },
  { name: "categories", url: "/admin/categories" },
  { name: "stop-list", url: "/admin/stop-list" },
  { name: "reservations", url: "/admin/reservations" },
  { name: "promotions", url: "/admin/promotions" },
  { name: "qr", url: "/admin/qr" },

  // ---- Store ----
  {
    name: "stock",
    url: "/admin/stock",
    notes: { table: "table" },
  },
  { name: "shopping", url: "/admin/shopping" },
  {
    name: "ingredients",
    url: "/admin/ingredients",
    notes: {
      unit: 'select >> nth=0',
      expected: 'table thead th:has-text("Bo\'lishi kerak")',
    },
  },
  {
    name: "tech-cards",
    url: "/admin/tech-cards",
    notes: {
      tabs: 'button:has-text("Zagotovkalar")',
      dishes: 'button:has-text("Taomlar ·")',
      add: 'button:has-text("Yangi zagotovka")',
      batch: 'table thead th:has-text("Bir partiya")',
      rate: 'table thead th:has-text("Narx")',
    },
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
  {
    name: "staff",
    url: "/admin/staff",
    notes: { add: 'button:has-text("Qo\'shish"), a:has-text("Qo\'shish") >> nth=0' },
  },
  {
    name: "roles",
    url: "/admin/roles",
    notes: { list: "main" },
  },
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
  {
    name: "settings",
    url: "/admin/settings",
    notes: { tabs: "main nav, main [role=tablist] >> nth=0" },
  },
  { name: "pos", url: "/admin/pos" },
  { name: "vacancies", url: "/admin/vacancies" },

  // ---- Apps ----
  { name: "kitchen-login", url: "/staff/login", anon: true },
  { name: "courier-login", url: "/kuryer/login", anon: true },
  { name: "kiosk-login", url: "/kiosk/login", anon: true },

  // ---- The guest's side ----
  { name: "site-home", url: "/", anon: true },
  { name: "site-menu", url: "/menu", anon: true },
  { name: "site-cart", url: "/cart", anon: true },
  { name: "site-booking", url: "/bron", anon: true },
];

/** Everything that is true of the running app but not of the product.
 *
 *  ⚠️ **A screenshot documents a screen, not a session.** The dev-server badge,
 *  the "two orders not accepted" bell that fires while the demo data ages, a
 *  half-loaded chart — each of them is a thing the reader will look for on
 *  their own screen and not find, and then wonder what else is different. */
const QUIET = `
  /* Next.js dev overlay and its badge. */
  nextjs-portal, #__next-build-watcher, [data-nextjs-toast] { display: none !important; }
  /* The unaccepted-order bell. It is real, it has its own article, and it
     must not sit on top of forty others. Matched by position because it has
     no hook of its own: the only fixed bottom-right stack in the panel. */
  div.fixed.bottom-4.right-4.z-50 { display: none !important; }
  /* Caret and focus rings from the automation's own clicks. */
  * { caret-color: transparent !important; }
`;

/** Scroll back to the top and let anything the scroll started settle. A shot
 *  taken mid-scroll shows a page nobody can find by opening the same address. */
async function settle(page, ms) {
  await page.addStyleTag({ content: QUIET }).catch(() => {});
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.waitForTimeout(ms);
}

const ACTS = {
  async openFirstEdit(page) {
    await page.getByRole("button", { name: /tahrir|edit/i }).first().click();
    await page.waitForTimeout(900);
  },
  async dishesTab(page) {
    await page.getByRole("button", { name: /Taomlar ·/ }).click();
    await page.waitForTimeout(700);
  },
  async openFirstPrep(page) {
    const btn = page.getByRole("button", { name: /^Tahrirlash$/ }).first();
    if (await btn.count()) {
      await btn.click();
      await page.waitForTimeout(900);
    } else {
      await page.getByRole("button", { name: /Yangi zagotovka/ }).click();
      await page.waitForTimeout(700);
    }
  },
};

/** PNG in, WebP out.
 *
 *  ⚠️ **Forty-four screenshots at twice the pixels is fifteen megabytes**, and
 *  a help page that takes fifteen megabytes to open is one a restaurant on a
 *  phone in a basement never reads — which is the exact person it is for. WebP
 *  at quality 82 holds screenshot text without visible artefacts and costs
 *  roughly a tenth of that. The PNG is deleted: two copies of every frame is a
 *  question about which one the article points at.
 *
 *  ⚠️ Resized to 1440 wide — the shot is taken at 2880 so the text is sharp on
 *  a retina screen, and the frame is *displayed* at most 720 CSS pixels wide.
 *  Above that the extra pixels are weight nobody can see. */
async function toWebp(name) {
  const png = path.join(OUT, `${name}.png`);
  await sharp(png)
    .resize({ width: 1440, withoutEnlargement: true })
    .webp({ quality: 82 })
    .toFile(path.join(OUT, `${name}.webp`));
  await unlink(png);
}

/** Where each callout points, as a share of the frame.
 *
 *  ⚠️ **Measured from the DOM, not placed by hand on the picture.** A rectangle
 *  typed in as "about 12% from the left" is right until the next release moves
 *  a button by ten pixels, and nothing then fails — the arrow simply points at
 *  the wrong thing, in three languages, until a reader writes in. Asking the
 *  page where the element actually is makes the annotation a fact about the
 *  build the screenshot came from.
 *
 *  ⚠️ **Percentages, not pixels**: the frame is displayed at whatever width the
 *  article column happens to be, and on a phone that is a third of the capture.
 */
async function measure(page, notes) {
  const out = {};
  const view = page.viewportSize();
  for (const [key, selector] of Object.entries(notes)) {
    try {
      const box = await page.locator(selector).first().boundingBox({ timeout: 2500 });
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
      console.log("   ! belgi topilmadi:", key, "→", selector);
    }
  }
  return out;
}

async function main() {
  const only = process.argv.slice(2);
  await mkdir(OUT, { recursive: true });

  const browser = await chromium.launch();
  // ⚠️ A desktop frame, at twice the pixels. The article is read on a phone as
  // often as not, and a screenshot that has to be pinched to be legible is one
  // nobody looks at twice.
  const ctx = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    deviceScaleFactor: 2,
    locale: "uz-UZ",
    timezoneId: "Asia/Tashkent",
  });
  const page = await ctx.newPage();

  // Sign in once; the token lives in the context for every panel frame.
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
  await page.locator('input[type="password"]').waitFor({ state: "detached", timeout: 20000 });
  await page.waitForTimeout(2500);
  if (page.url().includes("/admin/login")) {
    throw new Error("kirish bo'lmadi — ADMIN_USER / ADMIN_PASS ni tekshiring");
  }

  let ok = 0;
  const failed = [];
  const figures = {};
  for (const shot of SHOTS) {
    if (only.length && !only.includes(shot.name)) continue;
    try {
      await page.goto(BASE + shot.url, { waitUntil: "networkidle", timeout: 30000 });
      // Give charts, maps and images their first paint.
      await settle(page, shot.slow ? 3500 : 1800);
      if (shot.act) await ACTS[shot.act](page);
      await page.screenshot({ path: path.join(OUT, `${shot.name}.png`) });
      const size = page.viewportSize();
      figures[shot.name] = {
        w: size.width,
        h: size.height,
        ...(shot.notes ? { notes: await measure(page, shot.notes) } : {}),
      };
      await toWebp(shot.name);
      ok++;
      console.log("✓", shot.name);
    } catch (e) {
      failed.push(shot.name);
      console.log("✗", shot.name, String(e).split("\n")[0]);
    }
  }
  await browser.close();

  // ⚠️ Merged into what is already there, because a partial run (one frame
  // being re-taken) must not delete the measurements of the other forty-three.
  const target = path.resolve(HERE, "../keel-site/src/lib/help/figures.json");
  let existing = {};
  try {
    existing = JSON.parse(await readFile(target, "utf8"));
  } catch {}
  const merged = { ...existing, ...figures };
  const ordered = Object.fromEntries(Object.keys(merged).sort().map((k) => [k, merged[k]]));
  await writeFile(target, JSON.stringify(ordered, null, 2) + "\n");

  console.log(`\n${ok} ta olindi` + (failed.length ? `, ${failed.length} ta yiqildi: ${failed.join(", ")}` : ""));
}

main();
