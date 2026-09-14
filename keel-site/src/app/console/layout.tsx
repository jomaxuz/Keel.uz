"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Logo } from "@/components/Logo";
import LangSwitch from "@/components/LangSwitch";
import ThemeToggle from "@/components/ThemeToggle";
import { useT } from "@/lib/i18n/client";
import { clearToken, getToken, me, type Me } from "@/lib/api";
import { consoleHome } from "@/lib/consoleHome";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { t } = useT();
  const router = useRouter();
  const path = usePathname();
  const isLogin = path === "/console/login";
  // "unknown" until the token has been proven, so a protected page never
  // flashes its contents to somebody whose session has already expired.
  const [state, setState] = useState<"unknown" | "in" | "out">("unknown");
  // Asked once, at the top: every tab below and every page inside reads the same
  // answer, so a role change cannot leave two parts of the console disagreeing.
  const [who, setWho] = useState<Me | null>(null);

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
    me()
      .then((u) => {
        // ⚠️ `/console` is the owner's overview. Everybody else is sent to the
        // screen their work is on *before* anything renders — rendering first
        // would fire the overview's requests and flash "no data" at exactly the
        // people this redirect exists for. The effect runs again on the new path.
        if (path === "/console" && !u.can.overview) {
          router.replace(consoleHome(u));
          return;
        }
        setWho(u);
        setState("in");
      })
      .catch(() => {
        router.replace("/console/login");
        setState("out");
      });
  }, [isLogin, router, path]);

  if (isLogin) return <>{children}</>;
  if (state !== "in") {
    return (
      <p className="container-page py-20 text-sm text-ink-muted">
        {t.dash.loading}
      </p>
    );
  }

  // ⚠️ The layout editor is the one screen that must not be boxed in.
  //
  // Every other console page is a document — a list, a customer card — and a
  // reading width is right for those. The editor is a tool: three panes, and the
  // middle one is a canvas whose whole job is to be as wide as the screen allows.
  // Inside `container-page` it lost a third of the width to margins on exactly the
  // screen somebody sits at for an hour.
  const fullBleed = path.includes("/design");

  // ⚠️ The navigation follows the role, and the role is asked for once.
  //
  // Hiding a tab is a convenience — every one of these endpoints refuses the wrong
  // role on its own. But a console whose every tab answers "no permission" teaches an
  // agent that their tool is broken, and an agent has no use for the platform
  // overview: it is the platform's money and the customers' turnover.
  const can = who?.can;
  const tabs = [
    // Owner only — see consoleHome for why everybody else skips it.
    ...(can?.overview ? [{ href: "/console", label: t.dash.overview }] : []),
    ...(can?.tenants
      ? [
          { href: "/console/tenants", label: t.dash.tenants },
          { href: "/console/visits", label: t.console.nav.visits },
          // Beside the visits, because it is the same job from the other end:
          // what to write to the place you are about to walk into or have just left.
          { href: "/console/outreach", label: t.console.nav.outreach },
        ]
      : []),
    // ⚠️ **Owner and support only.** This used to have no role check ("whoever is
    // at a desk answers"); with a dedicated support role, a sales account
    // answering a technical question is a promise the platform has to keep.
    // Xatoliklar sits beside it: the same queue read from the other end.
    ...(can?.support
      ? [
          { href: "/console/support", label: t.console.nav.support },
          { href: "/console/reports", label: t.console.nav.reports },
        ]
      : []),
    // Money and contracts: owner only.
    ...(can?.partners
      ? [{ href: "/console/referrers", label: t.console.nav.referrers }]
      : []),
    ...(can?.seo ? [{ href: "/console/seo", label: t.console.nav.seo }] : []),
    // ⚠️ Owner and admin: the blog is published under our name, and the hands
    // that run domains and deploys are the hands that run it.
    ...(can?.blog ? [{ href: "/console/blog", label: t.blogAdmin.title }] : []),
    ...(can?.staff
      ? [{ href: "/console/staff", label: t.console.nav.staff }]
      : []),
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
                  tab.href === "/console"
                    ? path === tab.href
                    : path.startsWith(tab.href);
                return (
                  <Link
                    key={tab.href}
                    href={tab.href}
                    className={`rounded-xl px-3 py-2 text-sm font-semibold transition ${
                      active
                        ? "bg-raised text-ink"
                        : "text-ink-muted hover:text-ink"
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
      <main className={fullBleed ? "" : "container-page py-8"}>{children}</main>
    </div>
  );
}
