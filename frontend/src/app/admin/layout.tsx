"use client";

import type { IconType } from "react-icons";
import {
  LuBike,
  LuBookOpen,
  LuBriefcase,
  LuCalendarCheck,
  LuChartNoAxesColumn,
  LuCircleUser,
  LuLayoutDashboard,
  LuMessageSquare,
  LuMonitor,
  LuPhone,
  LuQrCode,
  LuReceipt,
  LuScrollText,
  LuSend,
  LuSettings,
  LuShieldCheck,
  LuTags,
  LuTicketPercent,
  LuUserRound,
  LuUsers,
  LuWallet,
} from "react-icons/lu";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { api, clearToken, getToken } from "@/lib/api";
import { useAdminT } from "@/lib/i18n/admin";
import AlertBell, { SoundToggle } from "@/components/admin/AlertBell";
import ScopeSwitcher from "@/components/admin/ScopeSwitcher";
import { AdminScopeProvider, useAdminScope } from "@/lib/adminScope";
import LangSwitch from "@/components/site/LangSwitch";
import ThemeToggle from "@/components/site/ThemeToggle";

// Nav labels come from the dictionary, so the whole panel follows the same
// `lang` cookie as the customer site.

// One icon per section, from `react-icons`.
//
// ⚠️ **Imported one icon at a time** (`react-icons/lu`, not `react-icons`): the top-level
// entry point pulls the index of several thousand icons, while a per-set import ships only
// what is named here. That is the difference between twenty shapes and a megabyte on a
// panel somebody opens on a phone in a kitchen.
//
// ⚠️ **Icons beside the words, never instead of them.** This sidebar has twenty-two entries
// and several pairs that no icon distinguishes — staff and couriers, payroll and cash,
// reports and the dashboard. An icon-only rail would make the owner learn a private
// alphabet; a symbol next to a label is what makes a long list scannable.
const ICONS: Record<string, IconType> = {
  dashboard: LuLayoutDashboard,
  orders: LuReceipt,
  reports: LuChartNoAxesColumn,
  reservations: LuCalendarCheck,
  calls: LuPhone,
  qr: LuQrCode,
  menu: LuBookOpen,
  pos: LuMonitor,
  categories: LuTags,
  promotions: LuTicketPercent,
  feedback: LuMessageSquare,
  vacancies: LuBriefcase,
  campaigns: LuSend,
  couriers: LuBike,
  staff: LuUsers,
  payroll: LuWallet,
  users: LuUserRound,
  admins: LuShieldCheck,
  logs: LuScrollText,
  settings: LuSettings,
  account: LuCircleUser,
};

const NAV = [
  { href: "/admin", key: "dashboard" },
  { href: "/admin/orders", key: "orders" },
  // Menu analysis, next to the dashboard rather than buried under settings:
  // it answers the question the front page raises — "these are the takings,
  // which dishes are they?" — and it is read, not configured.
  { href: "/admin/reports", key: "reports" },
  { href: "/admin/reservations", key: "reservations" },
  // The call centre desk. Not a separate role: the person answering the phone
  // during a rush is the same one who confirms the order two minutes later.
  { href: "/admin/calls", key: "calls" },
  { href: "/admin/qr", key: "qr" },
  { href: "/admin/menu", key: "menu" },
  // Mapping our dishes to the till's products. Next to the menu because that
  // is what it is about, and because a dish added here is a dish to map there.
  { href: "/admin/pos", key: "pos" },
  { href: "/admin/categories", key: "categories" },
  { href: "/admin/promotions", key: "promotions" },
  { href: "/admin/feedback", key: "feedback" },
  // Hiring. Beside feedback rather than in settings: both are lists of people waiting for
  // somebody to ring them, and both go stale in exactly the same way.
  { href: "/admin/vacancies", key: "vacancies" },
  // Owner only: the customer base belongs to the company, and this is the one
  // button that can annoy every guest at once — and spend money doing it.
  { href: "/admin/campaigns", key: "campaigns", ownerOnly: true },
  { href: "/admin/couriers", key: "couriers" },
  { href: "/admin/staff", key: "staff" },
  { href: "/admin/payroll", key: "payroll" },
  { href: "/admin/users", key: "users" },
  // Handing out panel accounts and reading the activity log belong to the
  // owner — a manager cannot grant themselves rights or check the trail.
  { href: "/admin/admins", key: "admins", ownerOnly: true },
  { href: "/admin/logs", key: "logs", ownerOnly: true },
  { href: "/admin/settings", key: "settings" },
  { href: "/admin/account", key: "account" },
] as const;

// ScopedMain remounts the current screen when the brand/branch lens moves.
//
// A screen loads its rows once, on mount. Without this, switching to another
// branch would leave the previous branch's orders on the table until the next
// poll — and on screens that never poll, until the operator reloaded the page.
// Remounting is blunt but honest: the lens changed, so the answer must be
// fetched again.
//
// It waits for the first scope load before rendering anything, so no request
// goes out before the API layer knows which branch to ask about.
function ScopedMain({ children }: { children: React.ReactNode }) {
  const { scopeKey, loading } = useAdminScope();
  return (
    <main className="p-6">
      {loading ? null : <div key={scopeKey}>{children}</div>}
    </main>
  );
}

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const t = useAdminT();
  const [ready, setReady] = useState(false);
  const [role, setRole] = useState<string>("");

  const isLogin = pathname === "/admin/login";

  useEffect(() => {
    if (isLogin) {
      setReady(true);
      return;
    }
    if (!getToken()) {
      router.replace("/admin/login");
      return;
    }
    // Validate the token; a 401 means it expired.
    api
      .me()
      .then((user) => {
        setRole(user.role);
        // Force a credential change on first login before anything else.
        if (user.mustChangePassword && pathname !== "/admin/account") {
          router.replace("/admin/account");
          return;
        }
        setReady(true);
      })
      .catch(() => {
        clearToken();
        router.replace("/admin/login");
      });
  }, [isLogin, pathname, router]);

  function logout() {
    clearToken();
    router.replace("/admin/login");
  }

  if (isLogin) return <>{children}</>;

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center text-ink-muted/70">
        {t.common.loading}
      </div>
    );
  }

  return (
    <AdminScopeProvider>
    <div className="flex min-h-screen bg-cream">
      <aside className="hidden w-60 shrink-0 flex-col border-r border-line bg-surface sm:flex">
        <div className="border-b border-line px-6 py-5">
          <Link href="/admin" className="text-lg font-bold">
            {t.nav.panel}
          </Link>
          <div className="mt-3 flex items-center gap-2">
            <LangSwitch />
            <ThemeToggle />
          </div>
          {/* Which brand/branch every screen is read through. Renders nothing
              for a company with one of each. */}
          <ScopeSwitcher className="mt-3" />
        </div>
        <nav className="flex-1 space-y-1 p-3">
          {NAV.filter((item) => !("ownerOnly" in item) || role === "owner").map((item) => {
            const active =
              item.href === "/admin"
                ? pathname === "/admin"
                : pathname.startsWith(item.href);
            const Icon = ICONS[item.key];
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium ${
                  active
                    ? "bg-brand text-white"
                    : "text-ink-muted hover:bg-ink/5"
                }`}
              >
                {/* A section with no icon still renders its label: a missing entry in the map
                    above must not leave a hole in the navigation. */}
                {Icon && <Icon className="h-[18px] w-[18px] shrink-0" aria-hidden />}
                {t.nav[item.key]}
              </Link>
            );
          })}
        </nav>
        <div className="space-y-2 border-t border-line p-3">
          <SoundToggle />
          <Link
            href="/"
            className="block rounded-lg px-3 py-2 text-sm text-ink-muted hover:bg-ink/5"
          >
            {t.nav.toSite}
          </Link>
          <button
            type="button"
            onClick={logout}
            className="mt-1 w-full rounded-lg px-3 py-2 text-left text-sm text-ink-muted hover:bg-ink/5"
          >
            {t.nav.logout}
          </button>
        </div>
      </aside>

      <div className="flex-1">
        {/* Mobile top bar */}
        <div className="flex items-center gap-2 border-b border-line bg-surface px-4 py-3 sm:hidden">
          <span className="font-bold">{t.nav.short}</span>
          <div className="ml-auto flex items-center gap-2">
            {/* The phone bar carries the switch too — the sidebar it normally
                sits in is hidden at this width. */}
            <SoundToggle />
            <LangSwitch />
            <ThemeToggle />
            <button
              type="button"
              onClick={logout}
              className="text-sm text-ink-muted"
            >
              {t.nav.logout}
            </button>
          </div>
        </div>
        <ScopeSwitcher className="border-b border-line bg-surface px-4 py-2 sm:hidden" />
        {/* Keyed on the brand/branch lens: switching branch remounts the screen,
            so every list refetches instead of showing the previous branch's rows
            until something happens to trigger a reload. A single-brand install
            has one constant key and never remounts. */}
        <ScopedMain>{children}</ScopedMain>
        {/* One watcher for the whole panel: new orders and bookings announce
            themselves out loud. */}
        <AlertBell />
      </div>
    </div>
    </AdminScopeProvider>
  );
}
