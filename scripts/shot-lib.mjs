// Shared by the two screenshot scripts.
//
// ⚠️ **Extracted rather than copied**, and that is not tidiness: both scripts
// hide the dev-server badge, and the one time the two copies disagreed the
// badge shipped baked into two published landing screenshots — on a page whose
// whole argument is that these are real screens.

import sharp from "sharp";
import { unlink } from "node:fs/promises";

export const LANGS = ["uz", "ru", "en"];

/** Everything that is true of the running app but not of the product.
 *
 *  ⚠️ **A screenshot documents a screen, not a session.** The dev-server badge,
 *  the "two orders not accepted" bell that fires as the demo data ages, a
 *  half-loaded chart — each of them is a thing the reader will look for on
 *  their own screen and not find, and then wonder what else is different. */
export const QUIET = `
  /* Next.js dev overlay and its badge. */
  nextjs-portal, #__next-build-watcher, [data-nextjs-toast] { display: none !important; }
  /* The unaccepted-order bell. It is real, it has its own article, and it must
     not sit on top of forty others. Matched by position because it has no hook
     of its own: the only fixed bottom-right stack in the panel. */
  div.fixed.bottom-4.right-4.z-50 { display: none !important; }
  /* Caret and focus rings from the automation's own clicks. */
  * { caret-color: transparent !important; }
`;

/** Marks this browser as one that has already been told about cookies.
 *
 *  ⚠️ **Hidden by consenting, not by CSS.** The notice is a real part of the
 *  guest's first visit and it covered the bottom third of the phone frames —
 *  but a rule that merely hides it leaves the page laid out as if it were
 *  there, and the screenshot then shows a gap nobody can explain. Setting the
 *  flag the site itself sets produces the screen a returning guest sees, which
 *  is the screen these pictures are about.
 *
 *  The key is `frontend/src/lib/cookies.ts`. If it is renamed, the notice comes
 *  back in the next capture — visibly, in the frame, which is the right place
 *  for that to be noticed. */
export async function acceptCookies(ctx) {
  await ctx.addInitScript(() => {
    try {
      window.localStorage.setItem("cookie_notice_v1", "seen");
    } catch {}
  });
}

/** Scroll back to the top and let anything the scroll started settle. A shot
 *  taken mid-scroll shows a page nobody can find by opening the same address. */
export async function settle(page, ms = 1800) {
  await page.addStyleTag({ content: QUIET }).catch(() => {});
  await page.evaluate(() => window.scrollTo(0, 0)).catch(() => {});
  await page.waitForTimeout(ms);
}

/** PNG in, WebP out, at a fixed width.
 *
 *  ⚠️ **Captured at twice the pixels and scaled down**, rather than captured at
 *  the final size: the text in a screenshot is the whole content of a
 *  screenshot, and a 1× capture of a 13px label is a grey smudge on a retina
 *  screen. The PNG is deleted — two copies of every frame is a question about
 *  which one the page points at. */
export async function toWebp(file, width, quality = 82) {
  await sharp(`${file}.png`)
    .resize({ width, withoutEnlargement: true })
    .webp({ quality })
    .toFile(`${file}.webp`);
  await unlink(`${file}.png`);
}

/** The language a *public* page is asked for.
 *
 *  Uzbek is the unprefixed base, exactly as it is on keel.uz — those URLs are
 *  the ones already printed, shared and indexed. Everything else (the panel,
 *  the till, the staff and courier apps) has no language URL at all and reads
 *  the `lang` cookie instead, which is why it is set on every context. */
export function prefixFor(lang) {
  return lang === "uz" ? "" : `/${lang}`;
}

export function langCookie(lang, base) {
  return [{ name: "lang", value: lang, url: base, sameSite: "Lax" }];
}
