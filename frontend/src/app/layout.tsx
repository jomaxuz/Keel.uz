import type { Metadata } from "next";
import { Inter, Nunito, Playfair_Display, Poppins } from "next/font/google";
import { ThemeProvider } from "@/lib/theme";
import { LangProvider } from "@/lib/i18n/client";
import { getLang } from "@/lib/i18n/server";
import { localeAlternates, siteOrigin, verificationToken } from "@/lib/seo";
import { api, imageUrl } from "@/lib/api";
import { getSiteScope } from "@/lib/siteBrand.server";
import { themeCss } from "@/lib/theme-css";
import type { Restaurant } from "@/lib/types";
import "./globals.css";

// UI text uses Inter; headings/prices use a display serif for a warmer,
// restaurant-menu feel. Both are exposed as CSS vars for Tailwind.
const inter = Inter({
  subsets: ["latin", "cyrillic"],
  variable: "--font-sans",
  display: "swap",
});

// The Keel wordmark on the till's lock screen, and nothing else. Our own name
// is set in our own type wherever it appears — the restaurant's chosen fonts
// dress the restaurant, not us.
//
// ⚠️ `preload: false` for the same reason as the pairings below: this is one
// word on one screen, and preloading it would make every guest on the public
// site fetch a family they never see.
const poppins = Poppins({
  subsets: ["latin"],
  weight: ["600"],
  variable: "--font-poppins",
  display: "swap",
  preload: false,
});

const playfair = Playfair_Display({
  subsets: ["latin", "cyrillic"],
  variable: "--font-display",
  display: "swap",
});

// Extra families for the font pairings the admin panel offers. They are named
// per-family (not per-role) so `themeCss` can point --font-sans/display at
// whichever one the restaurant picked.
//
// ⚠️ **`preload: false` on all three.** The panel offers three pairings, so the
// build carries three families — but any one restaurant uses two, and Next
// preloads every declared font by default. Every guest was therefore pulling
// six woff2 files (~180 KB, latin + cyrillic per family) before the page could
// paint, two of which their restaurant does not use.
//
// Turning preload off does not stop them loading; `display: "swap"` still
// fetches whichever family the theme actually names, as soon as the CSS asks
// for it. What it removes is the *guessing* — the browser no longer races to
// download fonts nobody selected. The two roles above (`--font-sans`,
// `--font-display`) keep their preloads, because those are the defaults every
// site renders with.
const interNamed = Inter({
  subsets: ["latin", "cyrillic"],
  variable: "--font-inter",
  display: "swap",
  preload: false,
});

const playfairNamed = Playfair_Display({
  subsets: ["latin", "cyrillic"],
  variable: "--font-playfair",
  display: "swap",
  preload: false,
});

const nunito = Nunito({
  subsets: ["latin", "cyrillic"],
  variable: "--font-nunito",
  display: "swap",
  preload: false,
});

// Built from the restaurant profile so the browser tab, search results and
// shared links all carry the restaurant's own name, description and logo.
export async function generateMetadata(): Promise<Metadata> {
  let rest: Restaurant | null = null;
  try {
    rest = (await api.getRestaurant(await getSiteScope())).restaurant;
  } catch {
    rest = null;
  }
  const name = rest?.name?.trim() || "Restoran";
  const description =
    rest?.description?.trim() || "Restoran sayti va yetkazib berish";
  const logo = rest?.logoUrl ? imageUrl(rest.logoUrl) : "";
  const cover = rest?.coverUrl ? imageUrl(rest.coverUrl) : "";

  const origin = await siteOrigin();

  // Proof of ownership for Search Console and Webmaster. Each console only
  // accepts its token on the domain it issued it for, so a token copied from
  // another site is inert rather than dangerous — which is why these sit in the
  // public profile next to the map key.
  //
  // An unset token must be `undefined`, not "": Next renders an empty meta tag
  // for the empty string, and an empty tag is exactly what both consoles read
  // as "the tag is there and it is wrong".
  const google = verificationToken(rest?.seo?.google);
  const yandex = verificationToken(rest?.seo?.yandex);

  // This page's own URL and its three language addresses. Inherited by every
  // page, so a new route is canonical and cross-linked without doing anything.
  const alternates = await localeAlternates();

  return {
    // ⚠️ Read from the request, not the build: one build serves every
    // restaurant, so a baked-in base would put somebody else's domain in every
    // canonical tag and every Open Graph image URL — and Google would quietly
    // merge or drop the pages rather than report an error.
    metadataBase: new URL(origin),
    // Sub-pages set only their own part; "%s | Maracanda" is assembled here.
    title: { default: name, template: `%s | ${name}` },
    description,
    // Says which URL is the real one, and which addresses are this same page in
    // the other two languages. Without the canonical, a page reached with a
    // tracking parameter — or on both the free subdomain and the restaurant's
    // own domain — competes with itself; without `hreflang`, the Uzbek and
    // Russian menus compete with each other.
    alternates,
    applicationName: name,
    // The uploaded logo doubles as the favicon; app/icon.svg stays the
    // fallback for a deploy that has not uploaded one yet.
    icons: logo ? { icon: logo, apple: logo, shortcut: logo } : undefined,
    openGraph: {
      type: "website",
      // The canonical, not the site root: a shared link should preview the page
      // that was shared, in the language it was shared in.
      url: alternates.canonical,
      siteName: name,
      title: name,
      description,
      images: cover ? [cover] : undefined,
    },
    twitter: {
      card: cover ? "summary_large_image" : "summary",
      title: name,
      description,
      images: cover ? [cover] : undefined,
    },
    robots: { index: true, follow: true },
    verification:
      google || yandex
        ? {
            ...(google ? { google } : {}),
            // Next has no first-class Yandex field; `other` emits the tag
            // verbatim, which is all Webmaster looks for.
            ...(yandex ? { other: { "yandex-verification": yandex } } : {}),
          }
        : undefined,
  };
}

// Applied before first paint so a dark-mode visitor never sees a light flash.
// Must agree with lib/theme.tsx exactly: this runs before the first paint and
// the provider runs after, so any disagreement is a visible flash of the wrong
// theme. Light is the default and the device preference is not read — see the
// comment in lib/theme.tsx for why a shop window does not follow the phone.
// ⚠️ The till and the waiter screen are always light — see forcedLight() in
// lib/theme.tsx, which must keep saying the same thing. This half runs before
// the first paint; that half runs after, and a disagreement between them is a
// visible flash of the wrong theme on every till boot.
const THEME_SCRIPT = `(function(){try{var p=location.pathname;if(p==="/kassa"||p.indexOf("/kassa/")===0||p==="/zal"||p.indexOf("/zal/")===0)return;if(localStorage.getItem("theme")==="dark"){document.documentElement.classList.add("dark");}}catch(e){}})();`;

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const lang = await getLang();

  // The restaurant's own colours/roundness, inlined below so they apply on the
  // first paint. A backend hiccup simply means the built-in design is used.
  let theme = "";
  try {
    theme = themeCss(
      (await api.getRestaurant(await getSiteScope())).restaurant.theme,
    );
  } catch {
    theme = "";
  }

  return (
    <html
      lang={lang}
      className={`${inter.variable} ${playfair.variable} ${interNamed.variable} ${playfairNamed.variable} ${nunito.variable} ${poppins.variable}`}
      suppressHydrationWarning
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_SCRIPT }} />
        {theme && <style dangerouslySetInnerHTML={{ __html: theme }} />}
      </head>
      <body>
        <ThemeProvider>
          <LangProvider initial={lang}>{children}</LangProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
