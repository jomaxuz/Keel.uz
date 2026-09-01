import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

/**
 * The staff screens are three-language and the sentences on them keep escaping
 * the dictionary.
 *
 * ⚠️ **The escapes are never the labels — they are the notices.** A button
 * caption is written once, next to twenty other captions, and whoever adds it
 * is already looking at `t.`. The line that says why something did not work
 * gets written later, mid-fix, in the branch of the code nobody had to touch
 * until then: "Restoran ma'lumotini yuklab bo'lmadi." sat inside an `if
 * (!rest)` on the settings page, and a Russian owner met it on exactly the day
 * the panel failed to load. Three of them were found this way, and all three
 * already had a dictionary entry saying the same thing — the sentence had been
 * translated, and then written out again by hand a few files away.
 *
 * So this reads the staff screens and refuses Uzbek text outside the
 * dictionary. It is a word list, not a language detector: it looks for the
 * handful of words that carry these notices, which is enough because the
 * notices are what goes missing.
 */

// Where a person reads Uzbek only if something was hardcoded. The customer
// site is not here: its pages are translated through a different dictionary
// and its Uzbek is the source text.
const ROOTS = [
  "src/app/admin",
  "src/app/kassa",
  "src/app/zal",
  "src/app/kuryer",
  "src/app/staff",
  "src/app/kiosk",
  "src/components/admin",
  "src/components/till",
  // ⚠️ **The shared modules too, and they were the worse half.** `lib/` has no
  // screen of its own, so nothing about editing it says "this text is read by
  // a person" — and both leaks found here were exactly that: a `STATUS_LABEL`
  // map the courier app printed on a Russian phone, and the four sentences
  // `lib/fiscal.ts` produced when the till could not reach the register.
  "src/lib",
];

// Files whose Uzbek is the source text rather than a leak: the dictionaries
// themselves, and the two documents that carry their own uz/ru/en.
const SOURCES = [
  "src/lib/i18n/",
  "src/lib/help/articles.ts",
  "src/lib/privacy.ts",
];

// ⚠️ Verbs and states, not nouns. "menyu", "kassa" and "filial" are the same
// word in the Russian panel, and a list containing them would flag the code
// that names a screen rather than the code that talks to a person.
const UZBEK =
  /(yo'q|bo'l|kerak|qiling|qilish|topilmadi|noto'g'ri|o'chir|saqlan|tanlang|kiriting|allaqachon|avval|ochilmadi|yuborildi|mavjud)/;

function walk(dir: string): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    if (statSync(p).isDirectory()) out.push(...walk(p));
    else if (p.endsWith(".ts") || p.endsWith(".tsx")) out.push(p);
  }
  return out;
}

// Comments are where the Uzbek belongs: half of these files explain a bug in
// the words the bug was reported in.
function withoutComments(src: string): string {
  return src
    .replace(/\/\*[\s\S]*?\*\//g, (m) => "\n".repeat((m.match(/\n/g) ?? []).length))
    .replace(/^\s*\/\/.*$/gm, "");
}

describe("staff screens", () => {
  it("say nothing in Uzbek outside the dictionary", () => {
    const found: string[] = [];
    for (const root of ROOTS) {
      for (const file of walk(root)) {
        // Tests state the fake server's answers in Uzbek on purpose: they are
        // asserting on what the real server sends, not writing a screen.
        if (SOURCES.some((s) => file.startsWith(s)) || /\.test\.tsx?$/.test(file)) continue;
        withoutComments(readFileSync(file, "utf8"))
          .split("\n")
          .forEach((line, i) => {
            const texts = [...line.matchAll(/"([^"]{6,})"|`([^`]{6,})`|>\s*([^<>{}\n]{6,})\s*</g)];
            for (const m of texts) {
              const text = (m[1] ?? m[2] ?? m[3] ?? "").trim();
              if (UZBEK.test(text)) found.push(`${file}:${i + 1}  ${text}`);
            }
          });
      }
    }
    expect(found, "dictionaryga ko'chiring (src/lib/i18n/admin.ts)").toEqual([]);
  });
});
