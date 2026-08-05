"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Logo } from "@/components/Logo";
import LangSwitch from "@/components/LangSwitch";
import ThemeToggle from "@/components/ThemeToggle";
import { useT } from "@/lib/i18n/client";
import { clearToken, getToken, me } from "@/lib/api";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { t } = useT();
  const router = useRouter();
  const path = usePathname();
  const isLogin = path === "/console/login";
  // "unknown" until the token has been proven, so a protected page never
  // flashes its contents to somebody whose session has already expired.
  const [state, setState] = useState<"unknown" | "in" | "out">("unknown");

  useEffect(() => {
    if (isLogin) {
      setState("out");
      return;
    }
    if (!getToken()) {
      router.replace("/console/login");
      setState("out");
      return;
    }
    me().then(() => setState("in")).catch(() => {
      router.replace("/console/login");
      setState("out");
    });
  }, [isLogin, router, path]);

  if (isLogin) return <>{children}</>;
  if (state !== "in") {
    return <p className="container-page py-20 text-sm text-ink-muted">{t.dash.loading}</p>;
  }

  const tabs = [
    { href: "/console", label: t.dash.overview },
    { href: "/console/tenants", label: t.dash.tenants },
  ];

  return (
    <div className="min-h-screen bg-page">
      <header className="border-b border-line bg-surface">
        <div className="container-page flex h-16 items-center justify-between gap-4">
          <div className="flex items-center gap-8">
            <Link href="/" aria-label="Keel">
              <Logo />
            </Link>
            <nav className="flex items-center gap-1">
              {tabs.map((tab) => {
                const active =
                  tab.href === "/console" ? path === tab.href : path.startsWith(tab.href);
                return (
                  <Link
                    key={tab.href}
                    href={tab.href}
                    className={`rounded-xl px-3 py-2 text-sm font-semibold transition ${
                      active ? "bg-raised text-ink" : "text-ink-muted hover:text-ink"
                    }`}
                  >
                    {tab.label}
                  </Link>
                );
              })}
            </nav>
          </div>
          <div className="flex items-center gap-2">
            <div className="hidden sm:block">
              <LangSwitch />
            </div>
            <ThemeToggle />
            <button
              type="button"
              onClick={() => {
                clearToken();
                router.replace("/console/login");
              }}
              className="rounded-xl px-3 py-2 text-sm font-semibold text-ink-muted hover:text-ink"
            >
              {t.dash.signOut}
            </button>
          </div>
        </div>
      </header>
      <main className="container-page py-8">{children}</main>
    </div>
  );
}
