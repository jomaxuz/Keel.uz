// `/llms-full.txt` — every page of keel.uz as markdown, in one file.
//
// Same rules as `/llms.txt` beside it: built per request from the sources the
// pages render (`lib/llms.ts`), language from the middleware's header only.
// The console watches this file and pings the search engines for the pages
// whose text changed — see control/internal/handlers/llms.go.

import { headers } from "next/headers";
import { getPostsFull } from "@/lib/blog";
import { buildLlmsFull, markdownResponse } from "@/lib/llms";
import { DEFAULT_LANG, isLang, LANG_HEADER } from "@/lib/i18n/url";

export const dynamic = "force-dynamic";

export async function GET() {
  const h = (await headers()).get(LANG_HEADER);
  const lang = isLang(h) ? h : DEFAULT_LANG;
  const posts = await getPostsFull(lang);
  return markdownResponse(buildLlmsFull(lang, posts));
}
