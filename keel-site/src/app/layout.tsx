import type { Metadata } from "next";
import { Inter, Space_Grotesk } from "next/font/google";
import "./globals.css";
import CookieNotice from "@/components/CookieNotice";
import { I18nProvider } from "@/lib/i18n/client";
import { getLang, getPath } from "@/lib/i18n/server";
import { dicts } from "@/lib/i18n/dict";
import { alternatesFor, ORIGIN } from "@/lib/i18n/url";
import StructuredData from "@/components/StructuredData";

const sans = Inter({ subsets: ["latin", "cyrillic"], variable: "--font-sans", display: "swap" });
const display = Space_Grotesk({ subsets: ["latin"], variable: "--font-display", display: "swap" });

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getLang();
  const path = await getPath();
  const t = dicts[lang];
  // This page's own URL and its three language addresses. Per-path rather than
  // a constant: `/status` declaring the home page as its canonical would tell a
  // search engine the two are the same page.
  const alternates = alternatesFor(lang, path);
  return {
    // Without this, Open Graph image paths stay relative and every share
    // preview breaks. One domain here, so it is a constant.
    metadataBase: new URL(ORIGIN),
    title: { default: `Keel — ${t.footer.tagline}`, template: "%s | Keel" },
    description: t.hero.lead,
    alternates,
    openGraph: {
      title: "Keel",
      description: t.hero.lead,
      type: "website",
      // The canonical, not the site root: a link shared from the Russian page
      // should preview as the Russian page.
      url: alternates.canonical,
      siteName: "Keel",
      locale: lang === "ru" ? "ru_RU" : lang === "en" ? "en_US" : "uz_UZ",
    },
    // `summary_large_image` needs an image to be worth asking for. It used to
    // be declared with none anywhere on the site, which is not a neutral
    // default: Telegram and WhatsApp — how this product actually gets shared
    // here — render a blank card, and a blank card is worse than a small one.
    // `opengraph-image.tsx` next to this file supplies it.
    twitter: { card: "summary_large_image", title: "Keel", description: t.hero.lead },
    robots: { index: true, follow: true },
    // Proof of ownership for Search Console and Webmaster. From the environment
    // rather than the database — unlike a tenant, this is one site on one
    // domain, and the token is settled once at deploy.
    verification: {
      ...(process.env.GOOGLE_SITE_VERIFICATION
        ? { google: process.env.GOOGLE_SITE_VERIFICATION }
        : {}),
      ...(process.env.YANDEX_VERIFICATION
        ? { other: { "yandex-verification": process.env.YANDEX_VERIFICATION } }
        : {}),
    },
  };
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const lang = await getLang();
  const path = await getPath();
  return (
    <html lang={lang} suppressHydrationWarning>
      <head>
        {/* Applied before first paint. Reading the theme in an effect instead
            would show the light page for one frame on every dark-mode load. */}
        <script
          dangerouslySetInnerHTML={{
            // Light unless this visitor picked dark. The device preference is
            // not read: this page is the product's first impression and it was
            // designed, reviewed and screenshotted in one of the two themes.
            __html: `(function(){try{if(localStorage.getItem('keel-theme')==='dark')document.documentElement.classList.add('dark')}catch(e){}})()`,
          }}
        />
        {/* Emitted in the layout so every page carries it, and read from the
            same dictionary the page is rendered from — a description in Uzbek
            under a Russian page is worse than none. */}
        <StructuredData t={dicts[lang]} path={path} />
      </head>
      <body className={`${sans.variable} ${display.variable} font-sans`}>
        <I18nProvider lang={lang}>
          {children}
          {/* ⚠️ In the root layout so it appears on the landing, the status page and the
              legal pages alike — a notice that only shows on the home page is a notice
              anybody arriving from a search result never sees. */}
          <CookieNotice />
        </I18nProvider>
      </body>
    </html>
  );
}
