"use client";

import type { IconType } from "react-icons";
import {
  LuBike,
  LuBan,
  LuClock,
  LuContact,
  LuHandCoins,
  LuSlidersHorizontal,
  LuUtensils,
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
  LuMenu,
  LuWallet,
  LuBanknote,
  LuKeyRound,
  LuX,
} from "react-icons/lu";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { api, clearToken, getToken } from "@/lib/api";
import { useAdminT, type AdminDict } from "@/lib/i18n/admin";
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
  stopList: LuBan,
  pos: LuMonitor,
  categories: LuTags,
  promotions: LuTicketPercent,
  feedback: LuMessageSquare,
  vacancies: LuBriefcase,
  campaigns: LuSend,
  couriers: LuBike,
  staff: LuUsers,
  payroll: LuWallet,
  roles: LuKeyRound,
  cash: LuBanknote,
  users: LuUserRound,
  admins: LuShieldCheck,
  logs: LuScrollText,
  settings: LuSettings,
  account: LuCircleUser,
};

// ⚠️ **Grouped, because twenty-two flat entries is not a list any more.**
//
// A rail that long is scrolled rather than read: the eye gives up somewhere
// around the tenth row and the owner navigates by remembering position, which
// breaks the moment anything is added. Groups turn "where is it" into two
// smaller questions — which part of the business, then which screen — and both
// have short answers.
//
// The grouping is by **who opens it and when**, not by what the data is:
//   • Bugun — during service, repeatedly, often on a phone.
//   • Menyu — set up once, edited weekly.
//   • Mijozlar — reached from a question about a person.
//   • Jamoa — money and people, opened at the end of a period.
//   • Sozlamalar — opened when something has to change.
const NAV_GROUPS = [
  {
    key: "today",
    items: [
      { href: "/admin", key: "dashboard" },
      { href: "/admin/orders", key: "orders" },
      { href: "/admin/reservations", key: "reservations" },
      // The call centre desk. Not a separate role: the person answering the
      // phone during a rush is the same one who confirms the order two
      // minutes later.
      { href: "/admin/calls", key: "calls" },
      // What is off sale right now — opened mid-service, by whoever is at the
      // counter, to answer one question.
      { href: "/admin/stop-list", key: "stopList" },
    ],
  },
  {
    key: "menu",
    items: [
      { href: "/admin/menu", key: "menu" },
      { href: "/admin/categories", key: "categories" },
      { href: "/admin/promotions", key: "promotions" },
      // Mapping our dishes to the till's products. Beside the menu because
      // that is what it is about: a dish added here is a dish to map there.
      { href: "/admin/pos", key: "pos" },
      { href: "/admin/qr", key: "qr" },
    ],
  },
  {
    key: "customers",
    items: [
      { href: "/admin/users", key: "users" },
      { href: "/admin/feedback", key: "feedback" },
      // Owner only: the customer base belongs to the company, and this is the
      // one button that can annoy every guest at once — and spend money.
      { href: "/admin/campaigns", key: "campaigns", ownerOnly: true },
      { href: "/admin/vacancies", key: "vacancies" },
    ],
  },
  {
    key: "money",
    items: [
      { href: "/admin/reports", key: "reports" },
      { href: "/admin/cash", key: "cash" },
      { href: "/admin/couriers", key: "couriers" },
      { href: "/admin/staff", key: "staff" },
      // Beside the staff list because that is where a role is chosen. ⚠️ Owner
      // only: widening a role decides who may take money out of the
      // restaurant, and a manager who could widen one could widen their own.
      { href: "/admin/roles", key: "roles", ownerOnly: true },
      { href: "/admin/payroll", key: "payroll" },
    ],
  },
  {
    key: "system",
    items: [
      { href: "/admin/settings", key: "settings" },
      // Handing out panel accounts and reading the activity log belong to the
      // owner — a manager cannot grant themselves rights or check the trail.
      { href: "/admin/admins", key: "admins", ownerOnly: true },
      { href: "/admin/logs", key: "logs", ownerOnly: true },
      { href: "/admin/account", key: "account" },
    ],
  },
] as const;

/** One icon per group, for the rail.
 *
 *  ⚠️ **Deliberately not reused from ICONS above.** The rail and the panel are
 *  two levels of the same navigation, and giving "Menyu" the group the same
 *  book as "Menyu" the screen makes the two levels look like one duplicated
 *  list — which is exactly the confusion a nested sidebar exists to remove. */
const GROUP_ICONS: Record<string, IconType> = {
  today: LuClock,
  menu: LuUtensils,
  customers: LuContact,
  money: LuHandCoins,
  system: LuSlidersHorizontal,
};

/** Which group a path belongs to.
 *
 *  ⚠️ **Longest match wins.** `/admin` is a prefix of every other route, so a
 *  first-match scan puts every screen in "Bugun" and the rail would highlight
 *  the wrong group everywhere except the dashboard. */
function groupOf(pathname: string): string {
  let best = "";
  let bestLen = -1;
  for (const g of NAV_GROUPS) {
    for (const item of g.items) {
      const hit =
        item.href === "/admin"
          ? pathname === "/admin"
          : pathname === item.href || pathname.startsWith(item.href + "/");
      if (hit && item.href.length > bestLen) {
        best = g.key;
        bestLen = item.href.length;
      }
    }
  }
  return best;
}

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
  // The phone's navigation. Closed on every route change: a panel still open over the screen
  // it just navigated to reads as a link that did not work.
  const [menuOpen, setMenuOpen] = useState(false);
  useEffect(() => setMenuOpen(false), [pathname]);
  // Which group the second column is showing.
  //
  // ⚠️ **The route wins.** A stored selection that outlived a navigation would
  // leave the panel listing "Menyu" while the screen behind it is the cash
  // report — a sidebar that disagrees with the page is worse than one that
  // needs an extra tap. The manual pick only survives until the route moves,
  // which is exactly long enough to look inside a group without leaving.
  const [picked, setPicked] = useState<string | null>(null);
  useEffect(() => setPicked(null), [pathname]);
  const routeGroup = groupOf(pathname);
  // ⚠️ Falls back to the first group rather than to nothing: an unrecognised
  // path (a screen added without a nav entry) must not empty the sidebar.
  const openGroup = picked || routeGroup || NAV_GROUPS[0].key;
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
        {/* ---- The navigation, in two levels ----

            ⚠️ **Twenty-two entries in one column is not a list, it is a
            scroll.** Grouping the flat rail helped — the eye stopped giving up
            around the tenth row — but every entry was still on screen at once,
            so the owner was still reading past four sections to reach the
            fifth. Splitting it moves the question from "find the row" to two
            short questions: which part of the business, then which screen.

            ⚠️ **The rail does not navigate.** Tapping a group swaps the second
            column and nothing else. A rail that jumped to the group's first
            screen would load a report because somebody wanted to look at what
            was in "Jamoa" — a click that fetches data the person did not ask
            for. The panel navigates; the rail only points.

            ⚠️ **The current page always decides which group is open**, so the
            second column can never show a section the screen does not belong
            to. Manual selection is allowed to win only until the route moves. */}
        <aside className="hidden shrink-0 sm:flex">
          {/* Level one: the parts of the business. */}
          <nav className="flex w-[88px] flex-col items-stretch gap-1 border-r border-line bg-ink/[0.03] p-2">
            <Link
              href="/admin"
              className="mb-1 flex h-10 items-center justify-center rounded-lg text-sm font-bold"
              title={t.nav.panel}
            >
              {t.nav.short}
            </Link>
            {NAV_GROUPS.map((group) => {
              const items = group.items.filter(
                (item) => !("ownerOnly" in item) || role === "owner",
              );
              // ⚠️ A group whose every screen is owner-only disappears from the
              // rail as well as the panel: a manager tapping an icon that opens
              // an empty column learns the navigation is unreliable.
              if (items.length === 0) return null;
              const Icon = GROUP_ICONS[group.key];
              const on = group.key === openGroup;
              return (
                <button
                  key={group.key}
                  type="button"
                  onClick={() => setPicked(group.key)}
                  aria-current={on ? "true" : undefined}
                  className={`flex flex-col items-center gap-1 rounded-lg px-1 py-2 text-[10px] font-semibold leading-tight transition ${
                    on
                      ? "bg-brand/10 text-brand"
                      : "text-ink-muted hover:bg-ink/5"
                  }`}
                >
                  {Icon && <Icon className="h-5 w-5 shrink-0" aria-hidden />}
                  {/* ⚠️ The word stays under the icon. Five symbols with no
                      labels is a private alphabet the owner has to learn, and
                      the rail is the one control they cannot navigate without.
                      ⚠️ It wraps rather than truncates: "Sozlamalar" clipped to
                      "Sozlamal…" is the label doing neither job. */}
                  <span className="w-full text-balance text-center">
                    {t.nav.groups[group.key]}
                  </span>
                </button>
              );
            })}
          </nav>

          {/* Level two: the screens inside the chosen part. */}
          <div className="flex w-52 flex-col border-r border-line bg-surface">
            <div className="border-b border-line px-4 py-4">
              <div className="flex items-center gap-2">
                <LangSwitch />
                <ThemeToggle />
              </div>
              {/* Which brand/branch every screen is read through. Renders
                nothing for a company with one of each. */}
              <ScopeSwitcher className="mt-3" />
            </div>
            <nav className="flex-1 space-y-1 overflow-y-auto p-2">
              <GroupLinks
                group={openGroup}
                role={role}
                pathname={pathname}
                t={t}
              />
            </nav>
            <div className="space-y-1 border-t border-line p-2">
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
                className="w-full rounded-lg px-3 py-2 text-left text-sm text-ink-muted hover:bg-ink/5"
              >
                {t.nav.logout}
              </button>
            </div>
          </div>
        </aside>

        <div className="flex-1">
          {/* Mobile top bar */}
          <div className="flex items-center gap-2 border-b border-line bg-surface px-4 py-3 sm:hidden">
            {/* ⚠️ The panel had no navigation at all on a phone: the sidebar is hidden below
              `sm`, so somebody in the kitchen could reach a screen only through a link that
              happened to be on the page they were already looking at. This is the way in.
              Labelled by what it does rather than by its state — a label that flips between
              "open" and "close" has to survive hydration, and both icons are drawn so CSS
              alone decides which is visible (the same rule ThemeToggle follows). */}
            <button
              type="button"
              onClick={() => setMenuOpen((v) => !v)}
              aria-expanded={menuOpen}
              aria-label={t.nav.menuLabel}
              className="flex h-9 w-9 items-center justify-center rounded-lg border border-line text-ink-soft"
            >
              {menuOpen ? (
                <LuX className="h-5 w-5" aria-hidden />
              ) : (
                <LuMenu className="h-5 w-5" aria-hidden />
              )}
            </button>
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
          {/* Rendered only when open: an always-mounted panel with `hidden` keeps twenty-two
            links in the tab order and in the accessibility tree, so a phone reader walks
            through a menu nobody opened. */}
          {menuOpen && (
            <nav className="max-h-[70vh] space-y-1 overflow-auto border-b border-line bg-surface p-3 sm:hidden">
              <NavLinks
                role={role}
                pathname={pathname}
                t={t}
                onNavigate={() => setMenuOpen(false)}
              />
              <Link
                href="/"
                onClick={() => setMenuOpen(false)}
                className="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-ink-muted"
              >
                {t.nav.toSite}
              </Link>
            </nav>
          )}
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

/** One group's screens — the second level of the desktop sidebar.
 *
 *  ⚠️ Reads the same NAV_GROUPS as the phone's flat list. Two hand-kept copies
 *  of a twenty-two entry navigation is two lists that will disagree, and a
 *  section added to one and forgotten in the other is invisible on exactly the
 *  device where it is hardest to notice. */
function GroupLinks({
  group,
  role,
  pathname,
  t,
}: {
  group: string;
  role: string;
  pathname: string;
  t: AdminDict;
}) {
  const found = NAV_GROUPS.find((g) => g.key === group) ?? NAV_GROUPS[0];
  const items = found.items.filter(
    (item) => !("ownerOnly" in item) || role === "owner",
  );
  return (
    <>
      {items.map((item) => {
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
              active ? "bg-brand text-white" : "text-ink-muted hover:bg-ink/5"
            }`}
          >
            {/* A section with no icon still renders its label: a missing entry
                in the map above must not leave a hole. */}
            {Icon && <Icon className="h-[18px] w-[18px] shrink-0" aria-hidden />}
            {t.nav[item.key]}
          </Link>
        );
      })}
    </>
  );
}

/** The whole navigation as one flat, grouped list — the phone's version.
 *
 *  ⚠️ **Deliberately not the desktop's two levels.** A rail-and-panel inside a
 *  dropdown means two taps before anything is even readable, on the device
 *  where the panel is opened one-handed in a kitchen. Here every heading and
 *  every row is visible and the sheet scrolls, which is the right trade at this
 *  width.
 *
 *  Both levels read the same NAV_GROUPS, so the two presentations cannot drift
 *  apart — a section added once appears in both. */
function NavLinks({
  role,
  pathname,
  t,
  onNavigate,
}: {
  role: string;
  pathname: string;
  t: AdminDict;
  /** Called on every tap, so the phone's panel can close itself. */
  onNavigate?: () => void;
}) {
  return (
    <>
      {NAV_GROUPS.map((group) => {
        const items = group.items.filter(
          (item) => !("ownerOnly" in item) || role === "owner",
        );
        // ⚠️ A group whose every entry is owner-only disappears with them —
        // a heading over nothing is a section a manager will keep looking
        // inside.
        if (items.length === 0) return null;
        return (
          <div key={group.key} className="mb-4">
            <div className="px-3 pb-1 text-[11px] font-semibold uppercase tracking-wide text-ink-muted/70">
              {t.nav.groups[group.key]}
            </div>
            {items.map((item) => {
              const active =
                item.href === "/admin"
                  ? pathname === "/admin"
                  : pathname.startsWith(item.href);
              const Icon = ICONS[item.key];
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={onNavigate}
                  className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium ${
                    active
                      ? "bg-brand text-white"
                      : "text-ink-muted hover:bg-ink/5"
                  }`}
                >
                  {/* A section with no icon still renders its label: a missing
                      entry in the map above must not leave a hole. */}
                  {Icon && (
                    <Icon className="h-[18px] w-[18px] shrink-0" aria-hidden />
                  )}
                  {t.nav[item.key]}
                </Link>
              );
            })}
          </div>
        );
      })}
    </>
  );
}
