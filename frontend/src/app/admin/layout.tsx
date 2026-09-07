"use client";

import type { IconType } from "react-icons";
import {
  LuBike,
  LuBan,
  LuCarrot,
  LuNotebookPen,
  LuClipboardCheck,
  LuCookingPot,
  LuHandPlatter,
  LuArrowLeftRight,
  LuShoppingCart,
  LuTrash2,
  LuTruck,
  LuHandshake,
  LuCalendarClock,
  LuTag,
  LuScanBarcode,
  LuWarehouse,
  LuClock,
  LuContact,
  LuHandCoins,
  LuSlidersHorizontal,
  LuUtensils,
  LuBoxes,
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
  LuTv,
  LuTicketPercent,
  LuUserRound,
  LuUsers,
  LuMenu,
  LuWallet,
  LuBanknote,
  LuKeyRound,
  LuCoins,
  LuVault,
  LuFlame,
  LuLandmark,
  LuX,
} from "react-icons/lu";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { api, clearToken, getToken } from "@/lib/api";
import { useAdminT, type AdminDict } from "@/lib/i18n/admin";
import {
  navLabel,
  needsMet,
  orderedFor,
  type BrandLike,
  type Needs,
} from "@/lib/adminNav";
import AlertBell, { SoundToggle } from "@/components/admin/AlertBell";
import SupportWidget from "@/components/admin/SupportWidget";
import AskProvider from "@/components/ui/Ask";
import CrashReporter from "@/components/CrashReporter";
import ScopeSwitcher from "@/components/admin/ScopeSwitcher";
import { AdminScopeProvider, useAdminScope } from "@/lib/adminScope";
import { homeFor } from "@/lib/panelRole";
import { SubscriptionProvider, moduleForPath } from "@/lib/subscription";
import UpgradeGate from "@/components/admin/UpgradeCta";
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
  // ⚠️ The sales the board deliberately does not show: a till check is closed
  // at a table, not delivered, so it has its own screen and its own mark.
  checks: LuHandPlatter,
  menu: LuBookOpen,
  stopList: LuBan,
  // ---- The store ----
  //
  // ⚠️ **Objects, not documents.** Every one of these could have been a sheet
  // of paper with a different corner folded, and five near-identical clipboards
  // in a column is a column nobody reads. A shelf, a carrot, a lorry, a bin and
  // one clipboard for the count that is actually a clipboard.
  stock: LuWarehouse,
  ingredients: LuCarrot,
  // ⚠️ A card, not a clipboard: the count already owns the clipboard, and two
  // clipboards in one column is the column nobody reads.
  techCards: LuNotebookPen,
  purchases: LuTruck,
  writeoffs: LuTrash2,
  transfers: LuArrowLeftRight,
  // ⚠️ **Not the lorry.** Deliveries already have it, and two identical shapes
  // in one column is the failure the note above this map is about — it was the
  // one pair in the store that had it. A delivery is a lorry; a supplier is the
  // person you ring about the lorry.
  suppliers: LuHandshake,
  shopping: LuShoppingCart,
  // ⚠️ A pot, not a factory or a clipboard: what this screen records is a
  // batch **cooked** in the central kitchen. It was the one row in the store
  // with no icon at all, which in a column of nine reads as a row that does not
  // belong to the section.
  production: LuCookingPot,
  stocktake: LuClipboardCheck,
  // ---- The three a shop opens and a kitchen mostly does not ----
  //
  // ⚠️ **They shipped with no icons at all**, which in a column of eleven reads
  // as three rows that do not belong to the section — and they are the section,
  // for a shop. A date, a price tag and a scanner: the object each one is about,
  // by the rule the rest of this map follows.
  expiring: LuCalendarClock,
  labels: LuTag,
  // ⚠️ A scanner, not a barcode: what this screen does is *read* a code the
  // state issued, and it never prints one. The label screen owns the tag.
  marking: LuScanBarcode,
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
  // ---- The money screens ----
  //
  // ⚠️ **Four screens about money in one column, and a banknote on all four
  // would be a column nobody reads** — the same failure the store section had
  // with five clipboards. So each takes the *object* it is about rather than
  // the subject they share: where the money is (coins in a heap), the box it
  // sits in (a vault), what burns it (a flame), and what a bank sends
  // (a bank building).
  money: LuCoins,
  safe: LuVault,
  expenses: LuFlame,
  payouts: LuLandmark,
  users: LuUserRound,
  admins: LuShieldCheck,
  logs: LuScrollText,
  settings: LuSettings,
  tv: LuTv,
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
/** Which sections this role may see.
 *
 *  ⚠️ **One function, because the same filter was written in three places and a
 *  fourth was about to be.** Three copies of a rule about who sees what is
 *  three chances for one of them to fall behind — and the one that falls behind
 *  is a menu somebody can still reach a page through.
 *
 *  ⚠️ **A storekeeper is an allow-list, not a hide-list.** The server refuses
 *  everything off its own list anyway; this exists so the panel does not draw
 *  twenty links that all answer forbidden, which reads as a broken account
 *  rather than as a boundary. */
/** The sections this role can see, narrowed to what this business is.
 *
 *  ⚠️ **`brand` is optional, and omitting it means "do not narrow".** The
 *  redirect that decides whether somebody may stand on a page calls this
 *  without one: hiding a row from a shop's sidebar is presentation, and
 *  bouncing a person off a working page they typed the address of is not. A
 *  shop that opens the booking screen finds a booking screen. */
function navFor(role: string, brand?: BrandLike) {
  const groups =
    role === "stock"
      ? NAV_GROUPS.filter((g) => g.key === "stock")
      : role === "operator"
        ? // ⚠️ **Named pages rather than a group**, because the operator's three
          // sections are three rows of "Bugun" and the rest of that group —
          // the dashboard, the till's checks, the stop list — is not theirs.
          NAV_GROUPS.map((g) => ({
            ...g,
            items: g.items.filter((i) =>
              (OPERATOR_PAGES as readonly string[]).includes(i.href),
            ),
          }))
        : NAV_GROUPS;
  return (
    groups
      .map((g) => ({
        ...g,
        items: orderedFor(g, brand).filter(
          (item) =>
            (!("ownerOnly" in item) || role === "owner") &&
            needsMet((item as { needs?: Needs }).needs, brand),
        ),
      }))
      // ⚠️ A group whose every entry is filtered out disappears with them: a
      // heading over nothing is a section people keep looking inside.
      .filter((g) => g.items.length > 0)
  );
}

/** The whole panel a call-centre operator gets.
 *
 *  ⚠️ **The server's list is the one that decides** (handlers/panelgate.go);
 *  this is what the navigation draws, and the two are deliberately the same
 *  three sections. Their own account is here because a new panel account is
 *  created with a temporary password and is sent to that screen before
 *  anything else — a role that could not reach it would be locked out on the
 *  day it was created. */
const OPERATOR_PAGES = [
  "/admin/orders",
  "/admin/reservations",
  "/admin/calls",
  "/admin/account",
] as const;

/** Whether this role may open this page at all.
 *
 *  ⚠️ **Read from the same navigation it draws**, so a section added to one is
 *  added to the other. A bookmark, a link in a chat or a browser's restored tab
 *  is how somebody arrives at a page no menu offered them.
 *
 *  Unlimited roles are answered `true` without a lookup: `ownerOnly` entries are
 *  hidden from a manager but the *page* refuses them on its own, and turning
 *  this into their gate too would redirect a manager away from screens they are
 *  allowed to read. */
function mayOpen(role: string, pathname: string) {
  if (role !== "operator" && role !== "stock") return true;
  return navFor(role).some((g) =>
    g.items.some(
      (i) => pathname === i.href || pathname.startsWith(i.href + "/"),
    ),
  );
}

const NAV_GROUPS = [
  {
    key: "today",
    items: [
      { href: "/admin", key: "dashboard" },
      { href: "/admin/orders", key: "orders" },
      // Dining room and counter sales. Beside the orders board because it is
      // the other half of the same sentence: that board is online orders only,
      // and until this sat next to it the till's sales were on no panel screen
      // at all.
      { href: "/admin/checks", key: "checks" },
      // ⚠️ Bookings need tables to book. A shop cannot hold one and a fast
      // food does not take them, so the row is not offered — the screen still
      // answers if somebody types the address.
      { href: "/admin/reservations", key: "reservations", needs: "tables" },
      // The call centre desk.
      //
      // ⚠️ **There is now a role for it**, and there deliberately was not: the
      // person answering the phone during a rush is usually the same one who
      // confirms the order two minutes later, so the desk sits beside the
      // orders board rather than behind a login of its own. What changed is who
      // else answers that phone — a restaurant with a hired operator was
      // handing them the panel's every screen to do it, because the only way in
      // was a manager account.
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
      // The QR code taped to a table, which needs a table.
      { href: "/admin/qr", key: "qr", needs: "tables" },
    ],
  },
  {
    // ⚠️ **Its own section, because it grew into one.** Four of these started
    // life under "Menyu" — a dish gets its cost from a card, so it seemed to
    // belong there — and that section reached nine entries, which is the
    // scroll this navigation was reorganised to avoid. They are also opened by
    // a different person at a different time: the menu is set up once and
    // edited weekly, while a delivery is entered the morning it arrives and a
    // count happens at the end of a month.
    key: "stock",
    items: [
      // ⚠️ **First, because it is the one that answers a question rather than
      // recording an answer.** Everything below it writes a movement down; this
      // reads them back as "what is on the shelf now", which is what somebody
      // opens this section to find out. It was the piece missing entirely — the
      // module could record a delivery, a write-off and a count, and had
      // nowhere to say what the store held.
      { href: "/admin/stock", key: "stock" },
      // ⚠️ Directly under the balance, because it is the balance's second
      // half: the amber row said "we are low" and stopped there.
      { href: "/admin/shopping", key: "shopping" },
      // What the kitchen buys, and therefore what a dish costs.
      { href: "/admin/ingredients", key: "ingredients" },
      // ⚠️ **Directly under the ingredients, because it is the next sentence:**
      // the list above says what a kilo costs, this says what goes into a
      // portion. It was not a screen at all — a dish's card was written on the
      // dish, and a prep's card was hidden inside the ingredient form, which is
      // why most restaurants never found the one piece that stops the same
      // tomatoes being listed in seven places.
      // ⚠️ **A card is a kitchen's document.** A shop's product owns a
      // one-line card the server keeps in step with it; a screen inviting
      // somebody to edit that by hand can only break the link between the
      // packet and the shelf it comes off.
      // ⚠️ **"composes", not "kitchen".** A florist has no kitchen and composes
      // everything it sells; asking the wrong question took the cards away from
      // the one shop that needs them most.
      { href: "/admin/tech-cards", key: "techCards", needs: "composes" },
      // Where those prices come from: entering a delivery is how they stop
      // being retyped.
      { href: "/admin/purchases", key: "purchases" },
      // And who they come from. ⚠️ Beside deliveries rather than under
      // settings: "who are we behind with" is asked on a delivery morning.
      { href: "/admin/suppliers", key: "suppliers" },
      // The other direction — food that left without being sold.
      { href: "/admin/writeoffs", key: "writeoffs" },
      // ⚠️ Neither of the two above: stock that only moved. Recording it as
      // either one lies — see models/transfer.go.
      { href: "/admin/transfers", key: "transfers" },
      // ⚠️ Beside the transfer and after it, because that is the order the food
      // travels in a chain: a batch is made in the central kitchen and then
      // moved to the branch that will sell it. A restaurant with one kitchen
      // opens this page once, reads that it is not for them, and never returns.
      // Batches made in a prep workshop — a kitchen turning inputs into
      // outputs, which is the one thing a shop does not do.
      // Batches made in a back room: fifty bouquets for the eighth of March is
      // the same document as a pot of sauce.
      { href: "/admin/production", key: "production", needs: "composes" },
      // And the count that turns the difference between them into an answer.
      { href: "/admin/stocktake", key: "stocktake" },
      // ⚠️ **Directly under the count, because it is the count's second
      // half.** A count freezes what was missing and what it was worth, per
      // line, and then said it to nobody: the finding was reachable only by
      // opening one document and reading down forty rows. This is that finding
      // as work — worst first, in money, answered once.
      //
      // ⚠️ **Not on the storekeeper's list**, and the server agrees
      // (handlers/stocklogin.go): the person who counted the shelf must not be
      // the one who writes the verdict on their own shortfall.
      { href: "/admin/shortages", key: "shortages" },
      // ⚠️ **A pharmacy is inspected on this and a grocery loses money to
      // it.** A restaurant's kitchen cares too, but its dates live on a
      // handful of dairy lines rather than on every box in the room — and two
      // more columns on every delivery is a cost paid by every restaurant for
      // a screen most of them would not open.
      { href: "/admin/expiring", key: "expiring", needs: "goods" },
      // Shelf labels and barcode stickers.
      //
      // ⚠️ **A shop only.** A restaurant's dishes have no shelf and no
      // barcode — a row offering to label a portion of osh is a row that
      // teaches a kitchen to stop reading the sidebar.
      { href: "/admin/labels", key: "labels", needs: "goods" },
      // Marked goods, scanned as they arrive.
      //
      // ⚠️ **Shown to a shop, though a bar receives marked bottles too.** The
      // row is presentation and the page is not gated — a restaurant that
      // stocks marked drinks reaches it by address, exactly as it reaches the
      // booking screen. What a sidebar row costs is a line every kitchen reads
      // past forever.
      { href: "/admin/marking", key: "marking", needs: "goods" },
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
      // ⚠️ **Owner only, and beside the drawer rather than inside it.** The
      // till's cash is counted at the end of a shift by whoever worked it; the
      // safe is the box in the office, and who may see how much is in it is not
      // the same question as who may count a drawer.
      // ⚠️ First of the money screens: "where is our money" is the question
      // the others are evidence for.
      { href: "/admin/money", key: "money", ownerOnly: true },
      { href: "/admin/safe", key: "safe", ownerOnly: true },
      // ⚠️ Beside the safe rather than under reports: this is where a cost is
      // *entered*, and the report is where it is read.
      { href: "/admin/expenses", key: "expenses" },
      // ⚠️ Beside the money screens, not under reports: this is where an owner
      // asks "has Uzum paid us yet", which is a question about today.
      { href: "/admin/payouts", key: "payouts", ownerOnly: true },
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
      // The televisions on the walls. ⚠️ Here rather than under "Bugun": a
      // screen is paired once and then nobody touches it for months — the
      // section people open during service is the board it draws, not this.
      // The screen on the dining-room wall.
      { href: "/admin/tv", key: "tv", needs: "tables" },
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
  stock: LuBoxes,
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
  const pathname = usePathname();
  return (
    <main className="p-6">
      {loading ? null : (
        <div key={scopeKey}>
          {/* ⚠️ Courtesy only — the rule is on the server, in one table matched
              on the request path. This turns a screen of buttons that all
              answer 402 into one sentence and a price. A path this table
              forgets is still refused by the server. */}
          <UpgradeGate module={moduleForPath(pathname)}>{children}</UpgradeGate>
        </div>
      )}
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
  //
  // ⚠️ **Three states, not two.** `null` means "the route decides", and an
  // empty string means "somebody closed it" — which is a different thing and
  // used to be impossible to express: pressing the open section wrote `""`,
  // the falsy check fell straight back to the route's group, and the section
  // sprang open again. A control that visibly refuses to close is read as
  // broken long before anybody works out that it is a fallback.
  const [picked, setPicked] = useState<string | null>(null);
  useEffect(() => setPicked(null), [pathname]);
  const routeGroup = groupOf(pathname);
  // ⚠️ Falls back to the first group rather than to nothing: an unrecognised
  // path (a screen added without a nav entry) must not empty the sidebar.
  const openGroup = picked === null ? routeGroup || NAV_GROUPS[0].key : picked;
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
        // ⚠️ **A page nobody linked to them.** The navigation draws only what a
        // limited role may open, which covers every click — and covers none of
        // the ways somebody actually arrives at a URL: a bookmark, a link in a
        // chat, the tab a browser restored. Landing on one used to render the
        // screen and let its requests answer forbidden one after another, which
        // looks like the panel is broken rather than like a boundary.
        if (!mayOpen(user.role, pathname)) {
          router.replace(homeFor(user.role));
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
      {/* ⚠️ Our own question box instead of the browser's. `window.confirm`
          draws the site's domain over a panel a restaurant is paying for, and
          it blocks the whole tab while it waits. */}
      <AskProvider>
        <SubscriptionProvider>
          <div className="flex min-h-screen bg-cream">
            {/* ---- The navigation: one column, groups that open ----

            ⚠️ **Twenty-two entries in one column is not a list, it is a
            scroll.** The eye gives up around the tenth row, so the entries are
            grouped and only one group is unfolded at a time: the question
            becomes "which part of the business", then "which screen", and both
            have short answers.

            ⚠️ **An earlier attempt made the first level a separate icon rail.**
            That is two columns of chrome for a panel that is often read on a
            1024-wide monoblock, and it spends horizontal space the screens
            themselves need. A heading that opens is the same two questions in
            one column.

            ⚠️ **A heading does not navigate.** Opening "Pul va jamoa" must not
            load a report because somebody wanted to see what is in it — a click
            that fetches data nobody asked for. The rows navigate; the heading
            only unfolds.

            ⚠️ **The current page always decides which group is open**, so the
            sidebar can never be folded shut over the screen it is showing.
            Manual choice wins only until the route moves. */}
            {/* ⚠️ **The sidebar is its own screen height, not the page's.** It
            grew with whatever was beside it, so on a long report the sound
            toggle, the link back to the site and the sign-out sat a thousand
            pixels down — reachable only by scrolling the *report* to its end.
            Sticky and exactly one viewport tall: the sections scroll inside
            it, and the three controls at the bottom stay where they are. */}
            {/* ⚠️ `data-help` marks an element the knowledge base draws a
              callout over (scripts/help-screens.mjs). It is not a test hook and
              not a style hook: the alternative was matching the button by its
              visible text, which stops working the moment the panel is
              screenshotted in Russian — and fails by pointing the arrow at
              nothing rather than by erroring. */}
            <aside
              data-help="nav"
              className="sticky top-0 hidden h-dvh shrink-0 self-start sm:flex"
            >
              <div className="flex h-full w-60 flex-col border-r border-line bg-surface">
                <div className="border-b border-line px-4 py-4">
                  <Link href="/admin" className="text-sm font-bold">
                    {t.nav.panel}
                  </Link>
                  <div className="mt-3 flex items-center gap-2">
                    <LangSwitch />
                    <ThemeToggle />
                  </div>
                  {/* Which brand/branch every screen is read through. Renders
                nothing for a company with one of each. */}
                  <ScopeSwitcher className="mt-3" />
                </div>
                <SidebarGroups
                  role={role}
                  openGroup={openGroup}
                  onPick={setPicked}
                  pathname={pathname}
                  t={t}
                />
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
              {/* ⚠️ In the layout, not on the dashboard. The screen somebody
                needs help with is whichever one is broken, and a help button
                that only exists on the home page is a button people go looking
                for after they have already telephoned. */}
              {/* ⚠️ Beside the support widget, and it is the same problem from
                the other side: this one reports the faults nobody writes in
                about. */}
              <CrashReporter app="panel" />
              {/* ⚠️ **Not for the limited roles.** Support is the restaurant's
                  own line to us, opened by whoever pays the invoice — and the
                  server refuses those endpoints to an operator anyway, so the
                  widget would open onto an error. */}
              {role !== "operator" && role !== "stock" && <SupportWidget />}
            </div>
          </div>
        </SubscriptionProvider>
      </AskProvider>
    </AdminScopeProvider>
  );
}

/** One group's screens — the second level of the desktop sidebar.
 *
 *  ⚠️ Reads the same NAV_GROUPS as the phone's flat list. Two hand-kept copies
 *  of a twenty-two entry navigation is two lists that will disagree, and a
 *  section added to one and forgotten in the other is invisible on exactly the
 *  device where it is hardest to notice. */
/** The sidebar's groups, narrowed to what this business actually is.
 *
 *  ⚠️ **Its own component so it can read the scope.** The layout renders the
 *  provider, so the layout itself is outside it and cannot ask which brand is
 *  selected — the sections belong on this side of that line. */
function SidebarGroups({
  role,
  openGroup,
  onPick,
  pathname,
  t,
}: {
  role: string;
  openGroup: string;
  onPick: (key: string) => void;
  pathname: string;
  t: AdminDict;
}) {
  const { brand } = useAdminScope();
  const setPicked = onPick;
  return (
    <nav className="flex-1 space-y-0.5 overflow-y-auto p-2">
      {navFor(role, brand).map((group) => {
                    const items = group.items;
                    const Icon = GROUP_ICONS[group.key];
                    const on = group.key === openGroup;
                    return (
                      <div key={group.key}>
                        <button
                          type="button"
                          onClick={() => setPicked(on ? "" : group.key)}
                          aria-expanded={on}
                          className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-semibold ${
                            on ? "text-ink" : "text-ink-muted hover:bg-ink/5"
                          }`}
                        >
                          {Icon && (
                            <Icon
                              className="h-[18px] w-[18px] shrink-0"
                              aria-hidden
                            />
                          )}
                          <span className="flex-1 text-left">
                            {t.nav.groups[group.key]}
                          </span>
                          {/* Points down when open. A caret that never moves is
                          decoration; this one is the only thing saying the
                          heading can be closed again. */}
                          <span
                            aria-hidden
                            className={`text-[10px] transition-transform ${on ? "rotate-90" : ""}`}
                          >
                            ▶
                          </span>
                        </button>
                        {on && (
                          <div className="mb-1 ml-3 space-y-0.5 border-l border-line pl-2">
                            <GroupLinks
                              group={group.key}
                              role={role}
                              pathname={pathname}
                              t={t}
                            />
                          </div>
                        )}
                      </div>
                    );
                  })}

    </nav>
  );
}

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
  const { brand } = useAdminScope();
  const groups = navFor(role, brand);
  const found = groups.find((g) => g.key === group) ?? groups[0];
  const items = found?.items ?? [];
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
            {Icon && (
              <Icon className="h-[18px] w-[18px] shrink-0" aria-hidden />
            )}
            {navLabel(item.key, t, brand)}
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
  // ⚠️ At the top, not inside the JSX: a hook read from an expression in the
  // middle of a render is the same call today and a conditional one after the
  // next edit wraps it.
  const { brand } = useAdminScope();
  return (
    <>
      {navFor(role, brand).map((group) => {
        const items = group.items;
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
                  {navLabel(item.key, t, brand)}
                </Link>
              );
            })}
          </div>
        );
      })}
    </>
  );
}
