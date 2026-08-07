"use client";

// A `<Link>` that keeps the visitor in the language they are reading.
//
// Every href on the public site is written unprefixed (`/menu`, `/about`) and
// that stays true — this rewrites at render time, so a Russian visitor's links
// point at `/ru/menu` without a single call site knowing about languages.
//
// Two things depend on it, and only one of them is visible. Obviously: clicking
// through the site should not walk back into Uzbek one link at a time. Less
// obviously, **a crawler follows links, and links are how it learns the
// Russian pages exist.** The sitemap lists them too, but a set of pages nothing
// links to is a set of pages a search engine has every reason to treat as an
// afterthought — which is exactly what the Russian menu was.
//
// Usable from server components: this is a client component, and it renders
// inside `LangProvider`.

import Link from "next/link";
import { useI18n } from "@/lib/i18n/client";
import { isLocalizedPath, localePath } from "@/lib/i18n";
import type { ComponentProps } from "react";

type Props = Omit<ComponentProps<typeof Link>, "href"> & { href: string };

export default function LocaleLink({ href, ...rest }: Props) {
  const { lang } = useI18n();
  // Anything not a plain in-site path — an absolute URL, a mailto:, an anchor —
  // is left alone. Prefixing "https://keel.uz" would produce "/ru/https://…".
  const localized =
    href.startsWith("/") && isLocalizedPath(href) ? localePath(lang, href) : href;
  return <Link href={localized} {...rest} />;
}
