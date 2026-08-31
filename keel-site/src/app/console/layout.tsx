"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Logo } from "@/components/Logo";
import LangSwitch from "@/components/LangSwitch";
import ThemeToggle from "@/components/ThemeToggle";
import { useT } from "@/lib/i18n/client";
import { clearToken, getToken, me, type Me } from "@/lib/api";

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
  const tabs = [
    ...(who?.can.stats === false
      ? []
      : [{ href: "/console", label: t.dash.overview }]),
    { href: "/console/tenants", label: t.dash.tenants },
    { href: "/console/visits", label: t.console.nav.visits },
    // Beside the visits, because it is the same job from the other end: what to
    // write to the place you are about to walk into or have just left.
    { href: "/console/outreach", label: t.console.nav.outreach },
    // ⚠️ **No role check, unlike the tabs around it.** Every one of those hides
    // a screen an agent has no use for; this one is the screen where the person
    // who can help is whoever is at a desk. A support tab only some roles can
    // see is a waiting restaurant held until one particular operator is back
    // from lunch.
    { href: "/console/support", label: t.console.nav.support },
    // ⚠️ Beside Yordam and for the same reason it has no role check: this is
    // the same queue read from the other end — what broke, arriving before
    // somebody writes in to say so.
    { href: "/console/reports", label: t.console.nav.reports },
    // Money, so the same gate the invoices are behind: an agent reading what
    // another channel earns is not part of selling.
    ...(who?.can.billing
      ? [{ href: "/console/referrers", label: t.console.nav.referrers }]
      : []),
    // Beside the platform controls, not the sales ones: this is the site's own
    // presence, and the same hands run it as run domains and deploys.
    ...(who?.can.provision
      ? [{ href: "/console/seo", label: t.console.nav.seo }]
      : []),
    ...(who?.can.staff
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
