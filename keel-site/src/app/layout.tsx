import type { Metadata } from "next";
import { Inter, Space_Grotesk } from "next/font/google";
import "./globals.css";
import { I18nProvider } from "@/lib/i18n/client";
import { getLang } from "@/lib/i18n/server";
import { dicts } from "@/lib/i18n/dict";

const sans = Inter({ subsets: ["latin", "cyrillic"], variable: "--font-sans", display: "swap" });
const display = Space_Grotesk({ subsets: ["latin"], variable: "--font-display", display: "swap" });

export async function generateMetadata(): Promise<Metadata> {
  const lang = await getLang();
  const t = dicts[lang];
  return {
    // Without this, Open Graph image paths stay relative and every share
    // preview breaks. One domain here, so it is a constant.
    metadataBase: new URL("https://keel.uz"),
    title: { default: `Keel — ${t.footer.tagline}`, template: "%s | Keel" },
    description: t.hero.lead,
    alternates: { canonical: "https://keel.uz" },
    openGraph: {
      title: "Keel",
      description: t.hero.lead,
      type: "website",
      url: "https://keel.uz",
      siteName: "Keel",
      locale: lang === "ru" ? "ru_RU" : lang === "en" ? "en_US" : "uz_UZ",
    },
    twitter: { card: "summary_large_image", title: "Keel", description: t.hero.lead },
    robots: { index: true, follow: true },
  };
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const lang = await getLang();
  return (
    <html lang={lang} suppressHydrationWarning>
      <head>
        {/* Applied before first paint. Reading the theme in an effect instead
            would show the light page for one frame on every dark-mode load. */}
        <script
          dangerouslySetInnerHTML={{
            __html: `(function(){try{var s=localStorage.getItem('keel-theme');var d=s?s==='dark':matchMedia('(prefers-color-scheme:dark)').matches;if(d)document.documentElement.classList.add('dark')}catch(e){}})()`,
          }}
        />
      </head>
      <body className={`${sans.variable} ${display.variable} font-sans`}>
        <I18nProvider lang={lang}>{children}</I18nProvider>
      </body>
    </html>
  );
}
