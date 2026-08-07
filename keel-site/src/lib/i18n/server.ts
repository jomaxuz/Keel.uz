import { cookies, headers } from "next/headers";
import { dicts, type Dict, type Lang } from "./dict";
import { isLang, LANG_HEADER, PATH_HEADER } from "./url";

/** The chosen language. Server-read so the landing page ships already
 *  translated — a page that renders in Uzbek and then flips to Russian is worse
 *  than one that never offered Russian.
 *
 *  ⚠️ **The URL outranks the cookie.** A visitor who has been here has `lang` in
 *  their cookie; when they open a link to `/ru`, the page they get must be the
 *  page the link promised. With the cookie first, every shared link would
 *  silently render in the recipient's own language — and the sender would never
 *  find out, because on their screen it looked right.
 *
 *  The header is set by middleware on every prefixed request and deleted on
 *  every other one, so its absence means "no prefix", never "trust the client". */
export async function getLang(): Promise<Lang> {
  const fromUrl = (await headers()).get(LANG_HEADER);
  if (isLang(fromUrl)) return fromUrl;
  const v = (await cookies()).get("lang")?.value;
  return v === "ru" || v === "en" ? v : "uz";
}

export async function getT(): Promise<Dict> {
  return dicts[await getLang()];
}

/** The path being rendered, with any language prefix already removed. Set by
 *  middleware, which is the only place that sees the URL before routing. */
export async function getPath(): Promise<string> {
  try {
    return (await headers()).get(PATH_HEADER) || "/";
  } catch {
    return "/";
  }
}
