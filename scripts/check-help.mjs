// The knowledge base has to stay one manual in three languages.
//
// ⚠️ **The failures this catches are all invisible on the page.** A slug that
// exists in Uzbek and not in Russian renders fine — until `hreflang` has told a
// crawler all three exist and the Russian one 404s. A `see:` pointing at a slug
// that was renamed renders as nothing at all: the "See also" box simply has one
// fewer line, and nobody notices for months. A `fig:` naming a screenshot that
// was never taken renders a broken image in one language only.
//
// Kept here rather than as a unit test in keel-site because that package has no
// test runner, and adding one would put a runner into an image whose Dockerfile
// runs `npm ci`. This is a script somebody runs; the alternative was nothing.
//
// Usage: cd scripts && node check-help.mjs

import { readFile, readdir } from "node:fs/promises";
import path from "node:path";

const HERE = path.dirname(new URL(import.meta.url).pathname);
const HELP = path.resolve(HERE, "../keel-site/src/lib/help");
const SHOTS = path.resolve(HERE, "../keel-site/public/help");

const LANGS = ["uz", "ru", "en"];

/** The articles as data, read out of the source rather than imported.
 *
 *  ⚠️ Deliberately regex over the file: importing TypeScript here would mean a
 *  build step, and this check has to be runnable in one command by somebody who
 *  has just edited a paragraph. The shapes it reads are the ones the files are
 *  written in, and the check fails loudly if a file stops matching. */
async function articlesOf(lang) {
  const src = await readFile(path.join(HELP, `${lang}.articles.ts`), "utf8");
  const out = [];
  for (const chunk of src.split(/\n    slug: "/).slice(1)) {
    const slug = chunk.slice(0, chunk.indexOf('"'));
    out.push({
      slug,
      section: (chunk.match(/section: "([a-z]+)"/) ?? [])[1],
      figs: [...chunk.matchAll(/fig: "([a-z0-9-]+)"/g)].map((m) => m[1]),
      see: [...chunk.matchAll(/see: \[([^\]]*)\]/g)].flatMap((m) =>
        [...m[1].matchAll(/"([a-z0-9-]+)"/g)].map((x) => x[1]),
      ),
      // Every callout key an article names, to be checked against the capture.
      notes: [...chunk.matchAll(/\n          ([a-z]+):\s*"/g)].map((m) => m[1]),
    });
  }
  return out;
}

const problems = [];
const fail = (msg) => problems.push(msg);

const byLang = {};
for (const lang of LANGS) byLang[lang] = await articlesOf(lang);

if (byLang.uz.length === 0) fail("uz.articles.ts dan hech nima o'qilmadi — fayl shakli o'zgargan");

// ---- 1. The same articles, in the same order ----
//
// Order matters as much as membership: the previous/next links walk this list,
// and a reader following them in Russian should land where an Uzbek reader does.
for (const lang of ["ru", "en"]) {
  const base = byLang.uz.map((a) => a.slug);
  const other = byLang[lang].map((a) => a.slug);
  for (const slug of base) {
    if (!other.includes(slug)) fail(`${lang}: "${slug}" maqolasi yo'q`);
  }
  for (const slug of other) {
    if (!base.includes(slug)) fail(`${lang}: "${slug}" ortiqcha (uz da yo'q)`);
  }
  if (base.length === other.length && base.some((s, i) => other[i] !== s)) {
    fail(`${lang}: maqolalar tartibi uz bilan mos emas`);
  }
}

// ---- 2. Same section and same figures in every language ----
//
// A figure present in one language and missing in another is a page that
// explains itself with a picture for some readers and not for others.
for (const lang of ["ru", "en"]) {
  for (const a of byLang[lang]) {
    const uz = byLang.uz.find((x) => x.slug === a.slug);
    if (!uz) continue;
    if (uz.section !== a.section) {
      fail(`${lang}/${a.slug}: bo'lim "${a.section}", uz da "${uz.section}"`);
    }
    if (uz.figs.join(",") !== a.figs.join(",")) {
      fail(`${lang}/${a.slug}: rasmlar mos emas (${a.figs} ≠ ${uz.figs})`);
    }
    if (uz.see.join(",") !== a.see.join(",")) {
      fail(`${lang}/${a.slug}: "shuni ham o'qing" havolalari mos emas`);
    }
  }
}

// ---- 3. Every cross-reference points at a real article ----
const slugs = new Set(byLang.uz.map((a) => a.slug));
for (const a of byLang.uz) {
  for (const ref of a.see) {
    if (!slugs.has(ref)) fail(`${a.slug}: "${ref}" degan maqola yo'q`);
    if (ref === a.slug) fail(`${a.slug}: o'ziga havola qilyapti`);
  }
}

// ---- 4. Every figure has a screenshot in every language ----
//
// ⚠️ **In every language, and that is the check that matters.** A frame missing
// only in Russian falls back to the Uzbek one, which renders perfectly and
// shows a Russian reader a panel they cannot read — the exact failure the
// per-language capture exists to remove, arriving silently.
const files = new Set(
  (await readdir(SHOTS)).filter((f) => f.endsWith(".webp")).map((f) => f.slice(0, -5)),
);
const used = new Set(byLang.uz.flatMap((a) => a.figs));
for (const fig of used) {
  for (const lang of LANGS) {
    if (!files.has(`${fig}.${lang}`)) {
      fail(`"${fig}.${lang}.webp" yo'q — scripts/help-screens.mjs ni ishga tushiring`);
    }
  }
}
// ⚠️ A warning, not a failure: a screenshot taken ahead of the article that
// will use it is a normal state to be in for an afternoon.
const unused = [...new Set([...files].map((f) => f.replace(/\.(uz|ru|en)$/, "")))].filter(
  (f) => !used.has(f),
);

// ---- 5. Callout keys are measured, in every language ----
//
// ⚠️ A key measured in Uzbek and missing in Russian leaves the Russian legend
// one line shorter than the Uzbek one, with no error anywhere.
const figures = JSON.parse(await readFile(path.join(HELP, "figures.json"), "utf8"));
for (const lang of LANGS) {
  const src = await readFile(path.join(HELP, `${lang}.articles.ts`), "utf8");
  for (const m of src.matchAll(/fig: "([a-z0-9-]+)",\n\s*notes: \{([^}]*)\}/g)) {
    const [, fig, block] = m;
    const keys = [...block.matchAll(/\n\s*([a-z]+):/g)].map((k) => k[1]);
    for (const shotLang of LANGS) {
      const have = Object.keys(figures[fig]?.[shotLang]?.notes ?? {});
      for (const k of keys) {
        if (!have.includes(k)) {
          fail(`${fig} (${shotLang}): "${k}" belgisi o'lchanmagan — ${lang} maqolasi uni ishlatadi`);
        }
      }
    }
  }
}

// ---- 6. figures.json holds nothing but the three languages ----
//
// The file changed shape when the capture went trilingual; a leftover `w`/`h`
// at the top level is a frame nothing reads, and it will be copied forward by
// the next merge for ever.
for (const [name, spec] of Object.entries(figures)) {
  for (const key of Object.keys(spec)) {
    if (!LANGS.includes(key)) {
      fail(`figures.json: "${name}" da "${key}" ortiqcha (faqat til kalitlari bo'lishi kerak)`);
    }
  }
}

if (unused.length) console.log("ℹ️  ishlatilmagan rasm:", unused.join(", "));

if (problems.length === 0) {
  console.log(`✓ ${byLang.uz.length} maqola × ${LANGS.length} til — hammasi izchil`);
} else {
  console.log(`✗ ${problems.length} muammo:`);
  for (const p of problems) console.log("  -", p);
  process.exitCode = 1;
}
