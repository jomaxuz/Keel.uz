// The screenshots on keel.uz's landing page.
//
// ⚠️ **These are the page's argument, not its decoration.** The whole pitch is
// "this is the real till, the real floor plan, the real kitchen screen" — which
// is why they were taken by hand from a running system in the first place, and
// why a Russian visitor being shown an Uzbek till undercuts exactly the claim
// the picture is there to make. Every frame is now captured in all three.
//
// ⚠️ **The states are real states, held in the demo database.** Table 8 is open
// with four courses on it because somebody opened it; there are fourteen
// tickets on the kitchen screen because there are fourteen orders. Nothing here
// is mocked — a fabricated screen would be the one thing on this page that
// could be caught out.
//
// ⚠️ **Elements this script clicks carry `data-help`** (see CLAUDE.md). Reaching
// for a button by the words on it works perfectly in Uzbek and matches nothing
// in Russian, and that failure does not raise: the run photographs the wrong
// screen and files it under the right name.
//
// Usage:  cd scripts && node landing-shots.mjs [name ...]
//   with the backend on :8080 (MONGO_DB=demo) and the app on :3000. The script
//   sets the demo staff and courier passwords/PINs through the admin API first,
//   so a fresh demo database needs no manual preparation.

import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { acceptCookies, LANGS, langCookie, prefixFor, settle, toWebp } from "./shot-lib.mjs";

const BASE = process.env.APP ?? "http://localhost:3000";
const API = process.env.API ?? "http://localhost:8080/api/v1";
const ADMIN = process.env.ADMIN_USER ?? "admin";
const ADMIN_PASS = process.env.ADMIN_PASS ?? "Demo12345";
/** One password for every demo account. This is the demo database; the real
 *  one has never had a password this script knows. */
const PASS = "Demo12345";

const HERE = path.dirname(new URL(import.meta.url).pathname);
const OUT = path.resolve(HERE, "../keel-site/public/shots");

/** The branch's own coordinates, so the courier app believes it is at work.
 *  ⚠️ Its "on shift" toggle is geofenced on the server, not just in the app —
 *  without a position the screen is a permission prompt, which is a real screen
 *  but not the one the landing page is illustrating. */
const AT_WORK = { latitude: 41.2856, longitude: 69.2034 };

/** Each frame: the size the landing page expects, who is signed in, and what
 *  has to happen before the shutter.
 *
 *  ⚠️ `w`/`h` are the numbers in `src/app/page.tsx`. They set the aspect ratio
 *  the layout reserves before the image loads; a capture with a different
 *  shape does not crop, it stretches. */
const SHOTS = [
  {
    name: "till",
    w: 1600,
    h: 1000,
    as: "cashier",
    path: "/kassa",
    async act(page) {
      await openTable(page, 8);
      // The drinks row — the last category chip. ⚠️ Matched on the chip class
      // and not on "the last button on screen": that selector found the pay
      // button and the first run filed a picture of the payment dialog under
      // the name of the menu grid.
      const chips = page.locator(".till-chip-btn, .till-chip-btn-on");
      const n = await chips.count();
      if (n > 1) await chips.nth(n - 1).click();
      await page.waitForTimeout(1400);
    },
  },
  {
    name: "pay",
    w: 1400,
    h: 1086,
    as: "cashier",
    path: "/kassa",
    // Taller than the frame so the dialog is not squeezed: the shot is a crop
    // around it, which is how the original was framed.
    viewport: { width: 1600, height: 1180 },
// ⚠️ From the very top: the header carries the cashier's name and the
    // open-shift line, which is what makes the frame a till rather than a
    // floating dialog. The left rail is what gets cut instead — it is in the
    // shot above this one.
    clip: { x: 190, y: 0, width: 1400, height: 1086 },
    async act(page) {
      await openTable(page, 8);
      await page.locator('[data-help="pay"]').click();
      await page.waitForTimeout(1200);
    },
  },
  { name: "floor", w: 1400, h: 973, as: "waiter", path: "/zal" },
  { name: "kds", w: 1500, h: 938, as: "kitchen", path: "/staff/kitchen" },
  { name: "stock", w: 1500, h: 938, as: "admin", path: "/admin/stock" },
  { name: "orders", w: 1500, h: 938, as: "admin", path: "/admin/orders" },
  { name: "dashboard", w: 1500, h: 938, as: "admin", path: "/admin" },
  { name: "site", w: 1500, h: 938, as: "guest", path: "/", site: true },
  {
    name: "miniapp",
    w: 560,
    h: 694,
    as: "guest",
    path: "/menu",
    site: true,
    phone: true,
  },
  { name: "courier", w: 560, h: 694, as: "courier", path: "/kuryer", phone: true },
];

/** Open a table from the floor plan.
 *
 *  ⚠️ Matched on the number, not on the state beside it: the button is labelled
 *  "8 · Band" in Uzbek and "8 · Занят" in Russian, and the half that identifies
 *  the table is the half that does not translate. */
async function openTable(page, number) {
  await page.locator(`[aria-label^="${number} ·"]`).first().click();
  await page.waitForTimeout(2500);
}

/** Give the demo accounts passwords and PINs this script knows.
 *
 *  ⚠️ Through the admin API rather than by writing hashes into Mongo: the
 *  password path has its own rules (a minimum length, a bcrypt cost, a journal
 *  entry) and a script that goes around them is a script that keeps working
 *  after that path breaks. */
async function prepareAccounts() {
  const res = await fetch(`${API}/admin/login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ username: ADMIN, password: ADMIN_PASS }),
  });
  if (!res.ok) throw new Error(`admin kirishi bo'lmadi: ${res.status}`);
  const { token } = await res.json();
  const H = { "content-type": "application/json", authorization: `Bearer ${token}` };

  const staff = Object.values(await (await fetch(`${API}/admin/staff`, { headers: H })).json());
  const people = {};
  for (const s of staff) {
    // ⚠️ The whole record goes back, because the update is a `$set` of named
    // fields: sending only the password would blank the schedule and the pay
    // rules of every demo employee.
    await fetch(`${API}/admin/staff/${s.id}`, {
      method: "PUT",
      headers: H,
      body: JSON.stringify({
        name: s.name,
        phone: s.phone ?? "",
        username: s.username,
        position: s.position ?? "",
        branchId: s.branchId,
        schedule: s.schedule ?? [],
        payMode: s.payMode,
        payPeriod: s.payPeriod,
        hourlyRate: s.hourlyRate ?? 0,
        shiftRate: s.shiftRate ?? 0,
        monthlyRate: s.monthlyRate ?? 0,
        isActive: true,
        canKitchen: !!s.canKitchen,
        canWaiter: !!s.canWaiter,
        canCashier: !!s.canCashier,
        password: PASS,
      }),
    });
    const pin = s.username.replace(/\D/g, "").repeat(4).slice(0, 4);
    await fetch(`${API}/admin/staff/${s.id}/pin`, {
      method: "PUT",
      headers: H,
      body: JSON.stringify({ pin }),
    });
    const role = s.canCashier ? "cashier" : s.canWaiter ? "waiter" : s.canKitchen ? "kitchen" : null;
    if (role && !people[role]) people[role] = { user: s.username, pin };
  }

  const couriers = Object.values(await (await fetch(`${API}/admin/couriers`, { headers: H })).json());
  const busy = couriers.find((c) => c.status === "busy") ?? couriers[0];
  await fetch(`${API}/admin/couriers/${busy.id}`, {
    method: "PUT",
    headers: H,
    body: JSON.stringify({
      name: busy.name,
      phone: busy.phone,
      username: busy.username,
      branchId: busy.branchId,
      vehicle: busy.vehicle,
      isActive: true,
      password: PASS,
    }),
  });
  people.courier = { user: busy.username };
  people.admin = { user: ADMIN, pass: ADMIN_PASS };
  return people;
}

/** Fill a login form that has just hydrated.
 *
 *  ⚠️ **Wait before typing.** The form renders on the server and React takes
 *  over a moment later, resetting both fields to its own empty state — a fill
 *  that lands first is discarded, the submit posts nothing, and the run
 *  photographs the login page under every other name. */
async function signIn(page, url, user, pass) {
  await page.goto(BASE + url, { waitUntil: "networkidle" });
  await page.waitForTimeout(2200);
  await page.locator("input").first().fill(user);
  await page.locator('input[type="password"]').fill(pass);
  await page.locator('button[type="submit"]').click();
  await page.waitForTimeout(3500);
}

/** The till and the floor screen ask for a PIN after the account password: the
 *  login says which tablet this is, the PIN says who is standing at it. */
async function enterPin(page, pin) {
  for (const digit of pin.split("")) {
    await page.getByRole("button", { name: digit, exact: true }).first().click();
    await page.waitForTimeout(180);
  }
  await page.waitForTimeout(3500);
}

async function shoot(browser, shot, lang, people) {
  const phone = shot.phone === true;
  const ctx = await browser.newContext({
    viewport: shot.viewport ?? { width: shot.w, height: shot.h },
    // ⚠️ Two device pixels per CSS pixel, then scaled back down in `toWebp`.
    // The text in these frames is the content of these frames.
    deviceScaleFactor: 2,
    isMobile: phone,
    hasTouch: phone,
    locale: lang === "ru" ? "ru-RU" : lang === "en" ? "en-US" : "uz-UZ",
    timezoneId: "Asia/Tashkent",
    ...(shot.as === "courier"
      ? { permissions: ["geolocation"], geolocation: AT_WORK }
      : {}),
  });
  await ctx.addCookies(langCookie(lang, BASE));
  await acceptCookies(ctx);
  // ⚠️ **Dish photos on the till are a per-device setting and they default to
  // off.** A counter with a hundred dishes is faster to read as a list of
  // names, so that default is right — and it makes the marketing screenshot
  // show a grid of grey initials, which is the opposite of the point. Set the
  // way the till itself stores it, so nothing here is a special case.
  await ctx.addInitScript(() => {
    try {
      window.localStorage.setItem("keel_till_images", "1");
    } catch {}
  });
  const page = await ctx.newPage();

  const target = (shot.site ? prefixFor(lang) : "") + shot.path;
  switch (shot.as) {
    case "admin":
      await signIn(page, "/admin/login", people.admin.user, people.admin.pass);
      await page.goto(BASE + target, { waitUntil: "networkidle" });
      break;
    case "cashier":
    case "waiter":
    case "kitchen": {
      const who = people[shot.as];
      await signIn(page, `/staff/login?next=${encodeURIComponent(target)}`, who.user, PASS);
      if (who.pin && (shot.path === "/kassa" || shot.path === "/zal")) {
        await enterPin(page, who.pin);
      }
      break;
    }
    case "courier":
      await signIn(page, "/kuryer/login", people.courier.user, PASS);
      break;
    default:
      await page.goto(BASE + target, { waitUntil: "networkidle" });
  }

  await settle(page, 2200);
  if (shot.act) await shot.act(page);
  // Lazily decoded photographs: the grid asks for them only once the tiles are
  // on screen, and a shutter that beats them produces the same grey initials.
  await settle(page, 1600);

  const file = path.join(OUT, `${shot.name}.${lang}`);
  await page.screenshot({
    path: `${file}.png`,
    ...(shot.clip
      ? {
          clip: {
            x: shot.clip.x,
            y: shot.clip.y,
            width: shot.clip.width,
            height: shot.clip.height,
          },
        }
      : {}),
  });
  await toWebp(file, shot.w);
  await ctx.close();
}

async function main() {
  const only = process.argv.slice(2);
  await mkdir(OUT, { recursive: true });

  const people = await prepareAccounts();
  const browser = await chromium.launch();
  let ok = 0;
  const failed = [];

  for (const lang of LANGS) {
    console.log(`\n── ${lang} ──`);
    for (const shot of SHOTS) {
      if (only.length && !only.includes(shot.name)) continue;
      try {
        await shoot(browser, shot, lang, people);
        ok++;
        console.log(`   ✓ ${shot.name}`);
      } catch (e) {
        failed.push(`${shot.name} (${lang})`);
        console.log(`   ✗ ${shot.name}`, String(e).split("\n")[0]);
      }
    }
  }
  await browser.close();
  console.log(
    `\n${ok} ta kadr olindi` +
      (failed.length ? `, ${failed.length} ta yiqildi: ${failed.join(", ")}` : ""),
  );
}

main();
