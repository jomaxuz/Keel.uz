import type { Metadata } from "next";
import { Inter, Space_Grotesk } from "next/font/google";
import "./globals.css";
import { I18nProvider } from "@/lib/i18n/client";
import { getLang } from "@/lib/i18n/server";
import { dicts } from "@/lib/i18n/dict";

const sans = Inter({ subsets: ["latin", "cyrillic"], variable: "--font-sans", display: "swap" });
const display = Space_Grotesk({ subsets: ["latin"], variable: "--font-display", display: "swap" });

export async function generateMetadata(): Promise<Metadata> {
  const t = dicts[await getLang()];
  return {
    title: { default: `Keel — ${t.footer.tagline}`, template: "%s | Keel" },
    description: t.hero.lead,
    openGraph: { title: "Keel", description: t.hero.lead, type: "website" },
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
