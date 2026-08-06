import type { Metadata } from "next";
import { Inter, Nunito, Playfair_Display } from "next/font/google";
import { ThemeProvider } from "@/lib/theme";
import { LangProvider } from "@/lib/i18n/client";
import { getLang } from "@/lib/i18n/server";
import { siteOrigin } from "@/lib/seo";
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

const playfair = Playfair_Display({
  subsets: ["latin", "cyrillic"],
  variable: "--font-display",
  display: "swap",
});

// Extra families for the font pairings the admin panel offers. They are named
// per-family (not per-role) so `themeCss` can point --font-sans/display at
// whichever one the restaurant picked.
const interNamed = Inter({
  subsets: ["latin", "cyrillic"],
  variable: "--font-inter",
  display: "swap",
});

const playfairNamed = Playfair_Display({
  subsets: ["latin", "cyrillic"],
  variable: "--font-playfair",
  display: "swap",
});

const nunito = Nunito({
  subsets: ["latin", "cyrillic"],
  variable: "--font-nunito",
  display: "swap",
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

  return {
    // ⚠️ Read from the request, not the build: one build serves every
    // restaurant, so a baked-in base would put somebody else's domain in every
    // canonical tag and every Open Graph image URL — and Google would quietly
    // merge or drop the pages rather than report an error.
    metadataBase: new URL(origin),
    // Sub-pages set only their own part; "%s | Maracanda" is assembled here.
    title: { default: name, template: `%s | ${name}` },
    description,
    // Says which URL is the real one. Without it the same page reached with a
    // tracking parameter, or on both the free subdomain and the restaurant's
    // own domain, competes with itself.
    alternates: { canonical: origin },
    applicationName: name,
    // The uploaded logo doubles as the favicon; app/icon.svg stays the
    // fallback for a deploy that has not uploaded one yet.
    icons: logo ? { icon: logo, apple: logo, shortcut: logo } : undefined,
    openGraph: {
      type: "website",
      url: origin,
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
  };
}

// Applied before first paint so a dark-mode visitor never sees a light flash.
const THEME_SCRIPT = `(function(){try{var t=localStorage.getItem("theme");if(!t){t=window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light";}if(t==="dark"){document.documentElement.classList.add("dark");}}catch(e){}})();`;

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
    theme = themeCss((await api.getRestaurant(await getSiteScope())).restaurant.theme);
  } catch {
    theme = "";
  }

  return (
    <html
      lang={lang}
      className={`${inter.variable} ${playfair.variable} ${interNamed.variable} ${playfairNamed.variable} ${nunito.variable}`}
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
