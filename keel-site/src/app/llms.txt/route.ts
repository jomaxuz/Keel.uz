// `/llms.txt` — keel.uz as an index for language models (llmstxt.org).
//
// ⚠️ **Generated per request from the site's own sources**, never a file in
// `public/`: see `lib/llms.ts`. `/ru/llms.txt` and `/en/llms.txt` reach this
// same route through the language middleware, which is why the language is
// read from the header it sets and **never from the cookie** — the unprefixed
// file has to be Uzbek for every crawler, whatever the last visitor chose.

import { headers } from "next/headers";
import { getPosts } from "@/lib/blog";
import { buildLlmsIndex, markdownResponse } from "@/lib/llms";
import { DEFAULT_LANG, isLang, LANG_HEADER } from "@/lib/i18n/url";

export const dynamic = "force-dynamic";

export async function GET() {
  const h = (await headers()).get(LANG_HEADER);
  const lang = isLang(h) ? h : DEFAULT_LANG;
  const posts = await getPosts(lang);
  return markdownResponse(buildLlmsIndex(lang, posts));
}
