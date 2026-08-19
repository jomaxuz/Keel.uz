// Thin fetch wrapper around the Go backend REST API.
//
// Public reads can be called from the server (SSR) or the client. Admin calls
// require a JWT stored in localStorage under `TOKEN_KEY`.

import type {
  DebtRow,
  TillDebt,
  Banner,
  JobApplication,
  Vacancy,
  AdminAlerts,
  BranchLoad,
  ChecksPage,
  CheckDetail,
  CheckRefundInfo,
  Ingredient,
  Purchase,
  PurchaseLine,
  WriteOff,
  WriteOffReason,
  Stocktake,
  StocktakeSheetRow,
  PrintJobRow,
  AdminCourierDetail,
  AdminStaffDetail,
  AdminLog,
  AbcXyzResponse,
  DashboardPrefs,
  RfmResponse,
  SalesReportResponse,
  ChannelReportResponse,
  CourierReportResponse,
  StaffReportResponse,
  AdminStats,
  AdminUser,
  AdminUserDetail,
  AdminUserRow,
  BookingPlan,
  Branch,
  Call,
  CallOutcome,
  CallStats,
  CallerLookup,
  Brand,
  BrandsResponse,
  Category,
  Courier,
  CourierOrderRow,
  CourierStats,
  CourierStatus,
  DeliveryProvider,
  DeliveryQuote,
  ExternalDelivery,
  Feedback,
  FeedbackList,
  Campaign,
  CampaignPreview,
  KioskCode,
  KioskToken,
  KitchenTicket,
  Check,
  TillPaymentMethod,
  TillFiscalStatus,
  TillPerson,
  FiscalJob,
  FiscalReply,
  FiscalReceipt,
  FiscalDay,
  CashShift,
  CashFigures,
  CashEntry,
  CashReportResponse,
  FinanceReportResponse,
  StockReportResponse,
  StaffRole,
  ReceiptSettings,
  ReceiptTemplate,
  ReceiptPreview,
  PermOption,
  OrderItemOption,
  SegmentRow,
  StopList,
  TelegramSettings,
  LoginResponse,
  LoyaltyInfo,
  MenuGroup,
  MenuItem,
  CreatedOrder,
  Order,
  OrderAddress,
  OrderQuote,
  OrderStatus,
  PayLink,
  Payment,
  PaymentMethod,
  PaymentMethodsResponse,
  PaymentSettings,
  PaymentSettingsInput,
  SMSSettings,
  SMSSettingsInput,
  OrderPOS,
  POSMapping,
  POSProduct,
  POSSettings,
  POSSettingsInput,
  FiscalProviderInfo,
  FiscalSettings,
  FiscalSettingsInput,
  LiveCall,
  PBXSettings,
  PBXSettingsInput,
  OrderTrack,
  PhoneCodeResponse,
  Promotion,
  PromotionTrigger,
  PromotionUsage,
  Reservation,
  ReservationStatus,
  Restaurant,
  RestaurantResponse,
  PayrollResponse,
  Shift,
  SiteUser,
  Staff,
  StaffMe,
  StaffPayment,
  StaffReport,
  StaffRow,
  UserAddress,
  TillReservation,
  BookingSettings,
  Printer,
} from "./types";

// Server-side (SSR) calls the backend directly; client-side calls the same
// origin, which Next.js rewrites proxy to the backend (see next.config.ts). This
// keeps everything single-origin behind one domain — no CORS, no
// mixed-content.
export const API_URL =
  typeof window === "undefined"
    ? (process.env.INTERNAL_API_URL ?? "http://localhost:8080/api/v1")
    : (process.env.NEXT_PUBLIC_API_URL ?? "/api/v1");

// ---- Multi-tenant (Keel SaaS) ----
//
// One Next.js process serves every customer's site; which backend a render
// belongs to is decided by the Host header, nothing else. The browser is
// unaffected — it still talks to its own origin, and the edge routes /api to
// the right container.
//
// TENANT_MODE is absent in a single-restaurant deployment, and then none of
// this runs: that product keeps working exactly as before.
const SAAS = process.env.TENANT_MODE === "saas";
const CONTROL = process.env.CONTROL_ORIGIN ?? "http://keel-control:9000";

// host → slug, cached. Without the cache every server render of every page
// would ask the control plane who it is rendering for.
const slugCache = new Map<string, { tenant: SiteTenant; at: number }>();
const SLUG_TTL = 60_000;

/** What the control plane knows about the site being rendered. */
export interface SiteTenant {
  slug: string;
  status: string;
  /** The paid removal of the "Powered by Keel" footer line. Decided by the
   *  control plane, never by the restaurant's own settings — it is the thing
   *  the customer pays for, and an owner given the switch would simply flip
   *  it. Same reasoning as kioskSecret and soldOut, except this one is the
   *  business model. */
  hideWatermark: boolean;
}

async function resolveHost(host: string): Promise<SiteTenant | null> {
  const clean = host.split(":")[0].toLowerCase();
  const hit = slugCache.get(clean);
  if (hit && Date.now() - hit.at < SLUG_TTL) return hit.tenant;
  try {
    const res = await fetch(
      `${CONTROL}/internal/resolve?host=${encodeURIComponent(clean)}`,
      { cache: "no-store" },
    );
    if (!res.ok) return null;
    const body = (await res.json()) as Partial<SiteTenant>;
    if (!body.slug) return null;
    const tenant: SiteTenant = {
      slug: body.slug,
      status: body.status ?? "",
      hideWatermark: !!body.hideWatermark,
    };
    slugCache.set(clean, { tenant, at: Date.now() });
    return tenant;
  } catch {
    return null;
  }
}

async function slugForHost(host: string): Promise<string | null> {
  return (await resolveHost(host))?.slug ?? null;
}

/** Whether this render should print the "Powered by Keel" line.
 *
 *  Resolved per request rather than read from the environment: the flag is
 *  flipped in the Keel console the moment a customer pays for it, and a value
 *  baked into the container at creation would stay wrong until somebody
 *  re-provisioned the tenant. That is the same trap `rewrites()` sets with
 *  build-time environment variables, one layer along.
 *
 *  A standalone install — one restaurant on its own VPS — is not a Keel tenant
 *  and never shows the line. */
export async function showWatermark(): Promise<boolean> {
  if (!SAAS) return false;
  try {
    const { headers } = await import("next/headers");
    const host = (await headers()).get("host") ?? "";
    const tenant = await resolveHost(host);
    // Unknown host: no line. Printing somebody else's brand on a site we
    // cannot identify is worse than printing nothing.
    return tenant ? !tenant.hideWatermark : false;
  } catch {
    return false; // not in a request scope (a build-time render)
  }
}

/** Where this particular render should send its API calls. */
async function apiBase(): Promise<string> {
  if (typeof window !== "undefined") return API_URL;
  if (!SAAS) return API_URL;
  // `headers()` is only available inside a request; a build-time render has no
  // Host, and falling back keeps `next build` working.
  try {
    const { headers } = await import("next/headers");
    const host = (await headers()).get("host") ?? "";
    const slug = await slugForHost(host);
    if (slug) return `http://keel-${slug}:8080/api/v1`;
  } catch {
    // not in a request scope
  }
  return API_URL;
}
export const UPLOADS_URL = process.env.NEXT_PUBLIC_UPLOADS_URL ?? "/uploads";

const TOKEN_KEY = "admin_token";
const USER_TOKEN_KEY = "user_token";
const COURIER_TOKEN_KEY = "courier_token";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TOKEN_KEY);
}

// Site customer token (separate from admin).
export function getUserToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(USER_TOKEN_KEY);
}

export function setUserToken(token: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(USER_TOKEN_KEY, token);
}

export function clearUserToken(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(USER_TOKEN_KEY);
}

// Courier app token (separate from both admin and customer sessions).
export function getCourierToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(COURIER_TOKEN_KEY);
}

export function setCourierToken(token: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(COURIER_TOKEN_KEY, token);
}

export function clearCourierToken(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(COURIER_TOKEN_KEY);
}

// Staff app token (separate again: an employee is not a courier, and neither
// is an admin — three roles, three sessions, no accidental crossover).
const STAFF_TOKEN_KEY = "staff_token";
// The branch screen's own token. It lives on a tablet on a wall, so it is
// long-lived and revoked by rotating the branch key rather than by expiry.
const KIOSK_TOKEN_KEY = "kiosk_token";

export function getKioskToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(KIOSK_TOKEN_KEY);
}

export function setKioskToken(token: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(KIOSK_TOKEN_KEY, token);
}

export function clearKioskToken(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(KIOSK_TOKEN_KEY);
}

export function getStaffToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(STAFF_TOKEN_KEY);
}

export function setStaffToken(token: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STAFF_TOKEN_KEY, token);
}

export function clearStaffToken(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(STAFF_TOKEN_KEY);
}

// ---- The till's two tokens ----
//
// ⚠️ **The device token and the person token are different things and are kept
// apart deliberately.** The device token says "this monoblock belongs to that
// branch" and is obtained once, with a username and a password, when the till
// is set up. The person token is bought with four digits and says who is
// standing there now; it is thrown away when the screen locks.
//
// Folding them into one would put us back where we started: one identity for
// the whole evening, and every void recorded against whoever unlocked the
// screen at six.
const TILL_TOKEN_KEY = "keel_till_token";
const TILL_DEVICE_KEY = "keel_till_device";

/** The monoblock's own token: "this machine belongs to that branch".
 *
 *  ⚠️ **localStorage, and deliberately so** — unlike the person's token, which
 *  lives in sessionStorage. This one is not a session: it is a setup step done
 *  once with a link from the panel, and a till that had to be re-bound every
 *  time it restarted would be a support call every morning.
 *
 *  Falls back to the staff token so tills installed before device binding
 *  existed keep working rather than dropping to a login form mid-service. */
export function getDeviceToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(TILL_DEVICE_KEY) ?? getStaffToken();
}

export function setTillDeviceToken(token: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TILL_DEVICE_KEY, token);
}

export function hasTillDevice(): boolean {
  if (typeof window === "undefined") return false;
  return !!window.localStorage.getItem(TILL_DEVICE_KEY);
}

export function clearTillDeviceToken(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TILL_DEVICE_KEY);
}

/** The unlocked person's token, if the screen is unlocked. */
export function getTillToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.sessionStorage.getItem(TILL_TOKEN_KEY);
}

/** ⚠️ **sessionStorage, not localStorage.** A till session belongs to this
 *  sitting at this screen: closing the app must lock it, and a token that
 *  survived a restart would hand the next person the last one's name. Same
 *  reasoning as the table context on the public site. */
export function setTillToken(token: string): void {
  if (typeof window === "undefined") return;
  window.sessionStorage.setItem(TILL_TOKEN_KEY, token);
}

/** Which token the till's own calls carry.
 *
 *  ⚠️ The unlocked person when the screen is unlocked, the device otherwise.
 *  This is what makes every void, discount and closed check carry the name of
 *  whoever is actually standing there — the whole point of the PIN. The device
 *  fallback keeps a restaurant that has set no PINs working exactly as before.
 */
export function tillBearer(): string | null {
  return getTillToken() ?? getStaffToken();
}

export function clearTillToken(): void {
  if (typeof window === "undefined") return;
  window.sessionStorage.removeItem(TILL_TOKEN_KEY);
}

export class ApiError extends Error {
  status: number;
  /** The refusal's own fields, when it carried any.
   *
   *  ⚠️ Kept because some refusals are **requests**, not verdicts. A till
   *  action the person may not do answers 409 with `needsOverride` and the
   *  permission's name, so the screen can ask a manager for their PIN instead
   *  of showing a dead end — and a dead end is what teaches a room to share one
   *  code. See handlers/tilloverride.go. */
  data: Record<string, unknown>;
  constructor(
    status: number,
    message: string,
    data: Record<string, unknown> = {},
  ) {
    super(message);
    this.status = status;
    this.data = data;
    this.name = "ApiError";
  }

  /** Which permission this action needed, if it needed one. */
  get needsOverride(): string | null {
    return this.data.needsOverride === true
      ? String(this.data.permission ?? "")
      : null;
  }

  /** That permission in words a person can read. `discount` on a screen in a
   *  restaurant is a word from our database. */
  get permissionName(): string {
    return String(this.data.permissionName ?? "");
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  auth?: boolean;
  // Explicit bearer token (used for customer calls). Overrides `auth`.
  bearer?: string | null;
  // Next.js fetch cache hint for server components.
  cache?: RequestCache;
  revalidate?: number;
  // Append the panel's current brand/branch lens (see setAdminScope). Opt-in,
  // because a few admin endpoints — the brand and branch lists themselves, the
  // customer base, the activity log — are company-wide by design.
  scope?: boolean;
}

// ---- The panel's brand/branch lens ----
//
// Held in a module variable rather than threaded through every call site: it is
// one value for the whole panel, set once by AdminScopeProvider, and read by the
// dozen list endpoints below. Empty ids mean "no filter", which is what a
// single-brand, single-branch install always sends.
let adminScope: { brandId: string; branchId: string } = {
  brandId: "",
  branchId: "",
};

export function setAdminScope(next: {
  brandId: string;
  branchId: string;
}): void {
  adminScope = next;
}

export function getAdminScope(): { brandId: string; branchId: string } {
  return adminScope;
}

function withScope(path: string): string {
  const qs = new URLSearchParams();
  if (adminScope.brandId) qs.set("brandId", adminScope.brandId);
  if (adminScope.branchId) qs.set("branchId", adminScope.branchId);
  const extra = qs.toString();
  if (!extra) return path;
  return path + (path.includes("?") ? "&" : "?") + extra;
}

async function request<T>(
  rawPath: string,
  opts: RequestOptions = {},
): Promise<T> {
  const {
    method = "GET",
    body,
    auth = false,
    bearer,
    cache,
    revalidate,
    scope = false,
  } = opts;
  const path = scope ? withScope(rawPath) : rawPath;

  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (bearer) {
    headers["Authorization"] = `Bearer ${bearer}`;
  } else if (auth) {
    const token = getToken();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  const init: RequestInit & { next?: { revalidate: number } } = {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  };
  if (cache) init.cache = cache;
  if (revalidate !== undefined) init.next = { revalidate };

  const res = await fetch(`${await apiBase()}${path}`, init);

  if (!res.ok) {
    let message = res.statusText;
    let data: Record<string, unknown> = {};
    try {
      data = (await res.json()) as Record<string, unknown>;
      message = String(data.error ?? data.message ?? message);
    } catch {
      // non-JSON error body — keep statusText
    }
    throw new ApiError(res.status, message, data);
  }

  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

/** Builds the query string for a report request.
 *
 *  Shared with `downloadReport` in intent, not in code: the screen and the
 *  spreadsheet must ask for the same period and the same cut, and writing the
 *  parameter list twice is how they drift into answering different questions. */
function reportQuery(params: Record<string, string | undefined>): string {
  const qs = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value) qs.set(key, value);
  }
  const lang = panelLang();
  if (lang) qs.set("lang", lang);
  return qs.toString() ? `?${qs}` : "";
}

/** The language the panel is being read in, for the server to answer in.
 *
 *  Read from the cookie here rather than passed in by each screen, and that is
 *  the point: a report's column titles, its period line and its explanatory
 *  note are written by the server, so every one of these requests has to carry
 *  the language. A call site that forgets produces an Uzbek spreadsheet on a
 *  Russian dashboard — which nothing checks and nobody reports as a bug, they
 *  simply retype the numbers.
 *
 *  Sent explicitly instead of relying on the cookie reaching the API: the API
 *  is usually a different origin, and a cross-origin fetch carries no cookies.
 *  Empty on the server (no `document`), where reports are never requested. */
function panelLang(): string {
  if (typeof document === "undefined") return "";
  return /(?:^|;\s*)lang=([^;]+)/.exec(document.cookie)?.[1] ?? "";
}

/** Downloads a report as a spreadsheet.
 *
 *  ⚠️ Fetched and turned into a blob rather than opened as a plain link.
 *  The panel authenticates with a bearer token, and an `<a href>` carries no
 *  headers — the browser would simply navigate to a 401, which looks to the
 *  operator like a download that silently did nothing.
 *
 *  The file name comes from the server's `Content-Disposition` when it is
 *  there, so the name in the Downloads folder is the one the report chose. */
export async function downloadReport(
  path: string,
  params?: Record<string, string | undefined>,
): Promise<void> {
  const qs = new URLSearchParams({ format: "xlsx" });
  // Any parameter the screen used, not just the period: a report cut by week
  // must download as weeks. Passing only `from`/`to` would hand the accountant
  // a file that disagrees with the screen it was downloaded from — which is the
  // one thing this export exists to prevent.
  for (const [key, value] of Object.entries(params ?? {})) {
    if (value) qs.set(key, value);
  }
  // ⚠️ The panel's language too: the titles, the column headers and the note
  // are written by the server, so without this an owner reading a Russian
  // dashboard downloads a sheet headed in Uzbek — and the column he guesses
  // wrong is a financial mistake with our name on it.
  const lang = panelLang();
  if (lang) qs.set("lang", lang);

  const token = getToken();
  const res = await fetch(`${await apiBase()}${withScope(`${path}?${qs}`)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new ApiError(res.status, res.statusText);

  const blob = await res.blob();
  const name =
    /filename="([^"]+)"/.exec(
      res.headers.get("Content-Disposition") ?? "",
    )?.[1] ?? "hisobot.xlsx";

  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  // Released on the next tick: revoking synchronously races the click in
  // Safari and produces an empty file.
  setTimeout(() => URL.revokeObjectURL(url), 0);
}

/** Downloads the whole-business archive.
 *
 *  A fetch with the bearer header rather than an `<a href>`, for the same
 *  reason the report download is: the panel authenticates with a token and a
 *  link carries no headers, so the browser would navigate to a 403 and the
 *  owner would see "nothing happened".
 *
 *  Unscoped on purpose — this is the whole install, every brand and every
 *  branch. Narrowing it to the lens the panel happens to be on would hand a
 *  departing customer an archive quietly missing half their restaurants. */
export async function downloadDataArchive(lang?: string): Promise<void> {
  const token = getToken();
  // The panel's own language goes with the request: the README inside the archive
  // is written in it, and the panel is the only thing that knows which language
  // the owner is reading. The server falls back to the `lang` cookie, then Uzbek.
  const qs = lang ? `?lang=${encodeURIComponent(lang)}` : "";
  const res = await fetch(`${await apiBase()}/admin/export/archive${qs}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      message = (await res.json())?.error ?? message;
    } catch {
      // A body that is not JSON tells us nothing more than the status did.
    }
    throw new ApiError(res.status, message);
  }
  const blob = await res.blob();
  const name =
    /filename="([^"]+)"/.exec(
      res.headers.get("Content-Disposition") ?? "",
    )?.[1] ?? "malumotlar.zip";
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}

// Resolve a stored image path/URL to an absolute URL the browser can load.
/** The widths the backend will actually generate (handlers/uploads.go).
 *
 *  Typed as a union rather than `number` so a call site cannot ask for 480 and
 *  silently receive the full-size original — the backend falls back rather than
 *  erroring, which is right for a stale URL and wrong for a typo in new code. */
export type ImageWidth = 300 | 600 | 1200;

/** Resolve a stored image path to a URL the browser can load, at the size the
 *  page actually shows it.
 *
 *  ⚠️ **Pass a width for anything in a list.** A restaurant home page was
 *  2.36 MB, and 1.83 MB of that was sixteen menu photographs served full size
 *  into cards about 350 px wide. The backend resizes on request and caches the
 *  result, so the only thing needed here is asking.
 *
 *  Absolute URLs are left alone: a pasted CDN or Instagram link is not ours to
 *  add parameters to. */
export function imageUrl(
  path: string | null | undefined,
  width?: ImageWidth,
): string | null {
  if (!path) return null;
  const q = width ? `?w=${width}` : "";

  // ⚠️ **An absolute URL can still be ours, and usually is.**
  //
  // The upload handler stores `PUBLIC_BASE_URL + /uploads/<name>`, so every
  // photograph an owner uploads is absolute — only the seeded demo images are
  // relative. Returning absolute URLs untouched therefore meant the resize
  // worked on the sample menu and on nothing a real restaurant had ever
  // uploaded, which is the opposite of the point.
  //
  // Ours is decided by the path (`/uploads/…`), not by the host: a tenant reads
  // its own images over `<slug>.keel.uz` *and* over its own domain, and both are
  // served by the same container. A pasted third-party link has no `/uploads/`
  // path, so it keeps being left alone — which also protects signed CDN URLs,
  // where an extra query parameter would break the signature.
  if (path.startsWith("http://") || path.startsWith("https://")) {
    if (!width) return path;
    try {
      const u = new URL(path);
      if (!u.pathname.startsWith("/uploads/")) return path;
      u.searchParams.set("w", String(width));
      return u.toString();
    } catch {
      return path;
    }
  }

  if (path.startsWith("/uploads/")) {
    return `${UPLOADS_URL}${path.slice("/uploads".length)}${q}`;
  }
  return `${UPLOADS_URL}/${path.replace(/^\/+/, "")}${q}`;
}

/** The site's lens: which brand's menu, and which branch serves it. */
export interface SiteScope {
  /** Brand id or slug. */
  brand?: string;
  branchId?: string;
  /** Admin only: the company document as stored, with no branch laid over it. */
  raw?: boolean;
  /** The console's short-lived token that shows the **unpublished** draft. Only
   *  ever set from the page URL — see siteQuery. */
  preview?: string;
}

function siteQuery(scope?: SiteScope): string {
  const qs = new URLSearchParams();
  if (scope?.brand) qs.set("brand", scope.brand);
  if (scope?.branchId) qs.set("branchId", scope.branchId);
  if (scope?.raw) qs.set("raw", "1");
  // ⚠️ **The console's preview token has to travel this far.**
  //
  // The backend serves the unpublished draft when it sees this parameter — but it
  // reads it off the **API** request, and the token arrives on the *page* URL.
  // Nothing forwarded it, so the check I wrote could never fire and the preview
  // always showed the published site: a feature that failed by looking like it
  // worked, which is why the operator reported "the preview does nothing".
  if (scope?.preview) qs.set("preview", scope.preview);
  return qs.toString() ? `?${qs}` : "";
}

// ---- Public API ----

export const api = {
  // Short cache: the profile drives the site theme and copy, so an owner who
  // saves a colour in the admin panel should see it almost immediately.
  // `brand` and `branch` are the site's own lens: which menu the guest is
  // browsing and which kitchen is serving them. Both are optional — a single
  // restaurant never sends either, and the server answers with its only one.
  getRestaurant: (scope?: SiteScope) =>
    request<RestaurantResponse>(`/restaurant${siteQuery(scope)}`, {
      // ⚠️ A preview is never cached. The draft changes every time somebody drags
      // an element, and a ten-second cache would show the previous version — which
      // reads exactly like the save not working.
      ...(scope?.preview ? { cache: "no-store" as const } : { revalidate: 10 }),
    }),

  getCategories: (scope?: SiteScope) =>
    request<Category[]>(`/categories${siteQuery(scope)}`, { revalidate: 60 }),

  getMenu: (scope?: SiteScope) =>
    request<MenuGroup[]>(`/menu${siteQuery(scope)}`, { revalidate: 60 }),

  getMenuItem: (id: string) => request<MenuItem>(`/menu/${id}`),

  createOrder: (body: unknown) =>
    // Answers with the order plus, for an online payment method, the bank link
    // to send the guest to.
    request<CreatedOrder>("/orders", {
      method: "POST",
      body,
      // Link the order to the signed-in customer, if any.
      bearer: getUserToken(),
    }),

  // Which methods the checkout may offer. Cash is always in the list; a
  // provider appears only once it is switched on and fully credentialed.
  paymentMethods: () =>
    request<PaymentMethodsResponse>("/payment-methods", { cache: "no-store" }),

  // The bank link for an order that still owes money. Public and keyed by the
  // receipt number: a guest who closed the tab can pay from another device.
  orderPayLink: (number: string) =>
    request<PayLink>(`/orders/${encodeURIComponent(number)}/pay`, {
      cache: "no-store",
    }),

  // ---- Phone + SMS code login ----
  phoneRequestCode: (phone: string) =>
    request<PhoneCodeResponse>("/auth/phone/request", {
      method: "POST",
      body: { phone },
    }),
  phoneVerify: (phone: string, code: string, name?: string) =>
    request<{ token: string; user: SiteUser }>("/auth/phone/verify", {
      method: "POST",
      body: { phone, code, name },
    }),

  userMe: () => request<SiteUser>("/users/me", { bearer: getUserToken() }),
  updateMe: (body: {
    firstName: string;
    lastName?: string;
    addresses: UserAddress[];
  }) =>
    request<SiteUser>("/users/me", {
      method: "PUT",
      body,
      bearer: getUserToken(),
    }),
  changePhoneRequest: (phone: string) =>
    request<PhoneCodeResponse>("/users/me/phone/request", {
      method: "POST",
      body: { phone },
      bearer: getUserToken(),
    }),
  changePhoneVerify: (phone: string, code: string) =>
    request<SiteUser>("/users/me/phone/verify", {
      method: "POST",
      body: { phone, code },
      bearer: getUserToken(),
    }),
  userOrders: () =>
    request<Order[]>("/users/me/orders", { bearer: getUserToken() }),

  // The customer's cashback balance and the statement that explains it.
  userLoyalty: () =>
    request<LoyaltyInfo>("/users/me/loyalty", { bearer: getUserToken() }),

  // Rating a delivered order. The tracking page asks once, on the page the
  // guest is already looking at.
  orderFeedback: (number: string) =>
    request<{ rated: boolean; rating?: number; comment?: string }>(
      `/orders/${number}/feedback`,
      { cache: "no-store" },
    ),
  submitFeedback: (number: string, rating: number, comment?: string) =>
    request<Feedback>(`/orders/${number}/feedback`, {
      method: "POST",
      body: { rating, comment },
      bearer: getUserToken(),
    }),

  trackOrder: (number: string) =>
    request<OrderTrack>(`/orders/${number}`, { cache: "no-store" }),

  // The whole bill: lines, campaigns, a typed code, delivery. The same
  // pipeline the order will run, so the preview cannot disagree with it.
  orderQuote: (body: {
    items: { menuItemId: string; qty: number; options?: unknown[] }[];
    type: string;
    address?: { lat: number; lng: number };
    branchId?: string;
    brandId?: string;
    promoCode?: string;
    usePoints?: number;
  }) =>
    request<OrderQuote>("/orders/quote", {
      method: "POST",
      body,
      // Per-customer promo limits need to know who is asking.
      bearer: getUserToken(),
    }),

  // What is on offer today. Codes are never listed here.
  getPromotions: (scope?: SiteScope) =>
    request<Promotion[]>(`/promotions${siteQuery(scope)}`, { revalidate: 30 }),

  deliveryQuote: (body: {
    lat: number;
    lng: number;
    subtotal: number;
    brandId?: string;
  }) => request<DeliveryQuote>("/delivery/quote", { method: "POST", body }),

  // ---- Table booking ----
  // The plan plus the tables already taken at `at` (ISO). The site shades them;
  // the server decides again when the booking is actually made.
  bookingPlan: (at?: string, scope?: SiteScope) => {
    const qs = new URLSearchParams();
    if (at) qs.set("at", at);
    if (scope?.brand) qs.set("brand", scope.brand);
    if (scope?.branchId) qs.set("branchId", scope.branchId);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<BookingPlan>(`/booking/plan${suffix}`, {
      cache: "no-store",
    });
  },
  createReservation: (body: {
    tableId: string;
    branchId?: string;
    at: string;
    guests: number;
    customer: { name: string; phone: string };
    comment?: string;
  }) =>
    request<Reservation>("/reservations", {
      method: "POST",
      body,
      // Optional: links the booking to the profile when signed in.
      bearer: getUserToken(),
    }),
  trackReservation: (number: string) =>
    request<Reservation>(`/reservations/${number}`, { cache: "no-store" }),
  userReservations: () =>
    request<Reservation[]>("/users/me/reservations", {
      bearer: getUserToken(),
    }),

  // ---- Brands and branches ----
  getBrands: () => request<BrandsResponse>("/brands", { revalidate: 10 }),

  /** What the restaurant is hiring for. */
  vacancies: () => request<Vacancy[]>("/vacancies", { revalidate: 60 }),
  /** ⚠️ No account needed — see handlers/jobs.go for the three rules that take the
   *  place of a login. */
  applyForVacancy: (
    id: string,
    body: { name: string; phone: string; comment?: string },
  ) =>
    request<{ ok: boolean }>(`/vacancies/${id}/apply`, {
      method: "POST",
      body,
    }),

  adminBanners: () =>
    request<Banner[]>("/admin/banners", { auth: true, cache: "no-store" }),
  createBanner: (body: Partial<Banner>) =>
    request<Banner>("/admin/banners", { method: "POST", body, auth: true }),
  updateBanner: (id: string, body: Partial<Banner>) =>
    request<{ ok: boolean }>(`/admin/banners/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  deleteBanner: (id: string) =>
    request<{ ok: boolean }>(`/admin/banners/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  adminVacancies: () =>
    request<Vacancy[]>("/admin/vacancies", { auth: true, cache: "no-store" }),
  saveVacancy: (
    id: string | null,
    body: Partial<Vacancy> & { branchId?: string },
  ) =>
    request<{ ok: boolean }>(
      id ? `/admin/vacancies/${id}` : "/admin/vacancies",
      {
        method: id ? "PUT" : "POST",
        body,
        auth: true,
      },
    ),
  deleteVacancy: (id: string) =>
    request<{ ok: boolean }>(`/admin/vacancies/${id}`, {
      method: "DELETE",
      auth: true,
    }),
  jobApplications: (params?: { vacancyId?: string; status?: string }) => {
    const qs = new URLSearchParams();
    if (params?.vacancyId) qs.set("vacancyId", params.vacancyId);
    if (params?.status) qs.set("status", params.status);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<JobApplication[]>(`/admin/job-applications${suffix}`, {
      auth: true,
      cache: "no-store",
    });
  },
  updateApplication: (id: string, body: { status: string; note?: string }) =>
    request<{ ok: boolean }>(`/admin/job-applications/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),

  adminBrands: () => request<Brand[]>("/admin/brands", { auth: true }),
  createBrand: (body: Partial<Brand>) =>
    request<Brand>("/admin/brands", { method: "POST", body, auth: true }),
  updateBrand: (id: string, body: Partial<Brand>) =>
    request<Brand>(`/admin/brands/${id}`, { method: "PUT", body, auth: true }),
  deleteBrand: (id: string) =>
    request<{ ok: boolean }>(`/admin/brands/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  adminBranches: (brandId?: string) =>
    request<Branch[]>(
      `/admin/branches${brandId ? `?brandId=${brandId}` : ""}`,
      { auth: true },
    ),
  createBranch: (body: Partial<Branch>) =>
    request<Branch>("/admin/branches", { method: "POST", body, auth: true }),
  updateBranch: (id: string, body: Partial<Branch>) =>
    request<Branch>(`/admin/branches/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  // One tap at the counter: "we're out of samsa (here, today)".
  setSoldOut: (branchId: string, menuItemId: string, soldOut: boolean) =>
    request<{ ok: boolean; soldOut: string[] }>(
      `/admin/branches/${branchId}/sold-out`,
      { method: "PUT", body: { menuItemId, soldOut }, auth: true },
    ),

  deleteBranch: (id: string) =>
    request<{ ok: boolean; deactivated?: boolean }>(`/admin/branches/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // ---- Admin ----

  login: (username: string, password: string) =>
    request<LoginResponse>("/admin/login", {
      method: "POST",
      body: { username, password },
    }),

  me: () => request<AdminUser>("/admin/me", { auth: true }),

  // ---- Forgotten panel password ----
  // Public by necessity: the whole point is that nobody can sign in. The code
  // is texted to the recovery number stored on the account.
  forgotPassword: (username: string) =>
    request<PhoneCodeResponse & { phone: string }>("/admin/password/forgot", {
      method: "POST",
      body: { username },
    }),
  resetPassword: (username: string, code: string, newPassword: string) =>
    request<{ ok: boolean }>("/admin/password/reset", {
      method: "POST",
      body: { username, code, newPassword },
    }),

  // The recovery number itself, set from inside the panel and verified by SMS.
  adminPhoneRequest: (phone: string) =>
    request<PhoneCodeResponse>("/admin/me/phone/request", {
      method: "POST",
      body: { phone },
      auth: true,
    }),
  adminPhoneVerify: (phone: string, code: string) =>
    request<{ ok: boolean; phone: string }>("/admin/me/phone/verify", {
      method: "POST",
      body: { phone, code },
      auth: true,
    }),

  changeCredentials: (body: {
    currentPassword: string;
    newUsername?: string;
    newPassword?: string;
  }) =>
    request<{ ok: boolean }>("/admin/credentials", {
      method: "PUT",
      body,
      auth: true,
    }),

  // Categories (full object required on update — backend ReplaceOne).
  adminCategories: () =>
    request<Category[]>("/admin/categories", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  createCategory: (body: Partial<Category>) =>
    request<Category>("/admin/categories", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),
  updateCategory: (id: string, body: Category) =>
    request<Category>(`/admin/categories/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  deleteCategory: (id: string) =>
    request<{ deleted: boolean }>(`/admin/categories/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // Menu items (full object required on update — backend ReplaceOne).
  adminMenu: () =>
    request<MenuItem[]>("/admin/menu", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  createMenuItem: (body: Partial<MenuItem>) =>
    request<MenuItem>("/admin/menu", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),
  updateMenuItem: (id: string, body: MenuItem) =>
    request<MenuItem>(`/admin/menu/${id}`, { method: "PUT", body, auth: true }),
  deleteMenuItem: (id: string) =>
    request<{ deleted: boolean }>(`/admin/menu/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // Customers.
  adminUser: (id: string) =>
    request<AdminUserDetail>(`/admin/users/${id}`, {
      auth: true,
      cache: "no-store",
    }),
  adminUsers: (params?: { q?: string; segment?: string; tag?: string }) => {
    const qs = new URLSearchParams();
    if (params?.q) qs.set("q", params.q);
    if (params?.segment) qs.set("segment", params.segment);
    if (params?.tag) qs.set("tag", params.tag);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<AdminUserRow[]>(`/admin/users${suffix}`, {
      auth: true,
      cache: "no-store",
    });
  },

  // The restaurant's own notes about a customer. The name and phone are the
  // guest's own and are deliberately not editable here.
  updateAdminUser: (
    id: string,
    body: {
      note?: string;
      tags?: string[];
      source?: string;
      birthday?: string;
      /** The guest asked not to receive campaign messages. */
      noMarketing?: boolean;
    },
  ) =>
    request<SiteUser>(`/admin/users/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),

  // Tags already in use, so the panel offers them instead of letting a typo
  // create a second "VIP " group.
  adminTags: () => request<string[]>("/admin/tags", { auth: true }),

  // Orders.
  adminOrders: (params?: {
    status?: OrderStatus;
    q?: string;
    userId?: string;
    limit?: number;
    /** Only orders placed for a later time, soonest first. A different sort on
     *  purpose: the pre-order tab answers "what is due next", not "what came
     *  in last". */
    scheduled?: boolean;
  }) => {
    const qs = new URLSearchParams();
    if (params?.status) qs.set("status", params.status);
    if (params?.q) qs.set("q", params.q);
    if (params?.userId) qs.set("userId", params.userId);
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.scheduled) qs.set("scheduled", "1");
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<Order[]>(`/admin/orders${suffix}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    });
  },
  // Dining room and counter sales. Deliberately a different endpoint from the
  // orders board rather than a flag on it: the board leaves till checks out on
  // purpose, and these are read by somebody asking about a shift, not about a
  // delivery.
  adminChecks: (params?: {
    from?: string;
    to?: string;
    state?: "all" | "open" | "closed";
    place?: "all" | "hall" | "counter";
    method?: string;
    q?: string;
    limit?: number;
    skip?: number;
  }) => {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(params ?? {})) {
      // "all" is the absence of a filter, not a value the server has to know.
      if (v !== undefined && v !== "" && v !== "all") qs.set(k, String(v));
    }
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<ChecksPage>(`/admin/checks${suffix}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    });
  },

  // One sale, opened. ⚠️ Its own endpoint rather than `adminOrder`: that one
  // returns the order document, where a table's customer, address and courier
  // fields are empty or meaningless, and the till's own facts — the voids, the
  // guest numbers, the course each line was fired with — are not on it.
  adminCheck: (id: string) =>
    request<CheckDetail>(`/admin/checks/${id}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    }),

  // A duplicate of the guest's receipt, laid out by the same renderer the till
  // uses. ⚠️ `toPrinter` is opt-in: whoever opens the panel is usually not in
  // the building, and paper appearing at a counter nobody is standing at is a
  // slip somebody has to work out the meaning of.
  adminPrintCheck: (id: string, toPrinter = false) =>
    request<{
      lines: string[];
      widthMM: number;
      logoUrl: string;
      queued: number;
    }>(`/admin/checks/${id}/print`, {
      method: "POST",
      auth: true,
      body: { toPrinter },
      scope: true,
    }),

  // Money handed back on a closed sale. ⚠️ The reason is required by the
  // server, not only by the form: "refunded 240 000" with no sentence beside
  // it is the line every argument about a shift starts from.
  adminRefundCheck: (id: string, reason: string) =>
    request<{ refund: CheckRefundInfo }>(`/admin/checks/${id}/refund`, {
      method: "POST",
      auth: true,
      body: { reason },
      scope: true,
    }),

  // What the printers were asked to do. ⚠️ Bounded to the last day by default:
  // the queue grows with traffic rather than with the business.
  adminPrintJobs: (params?: { failed?: boolean; hours?: number }) => {
    const qs = new URLSearchParams();
    if (params?.failed) qs.set("failed", "1");
    if (params?.hours) qs.set("hours", String(params.hours));
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<{ jobs: PrintJobRow[]; failed: number }>(
      `/admin/print-jobs${suffix}`,
      { auth: true, cache: "no-store", scope: true },
    );
  },

  // Offer a given-up job to the agent again. ⚠️ The stored bytes, not a
  // rebuilt document: a receipt regenerated after a price changed is not the
  // one the guest was charged for.
  adminRetryPrintJob: (id: string) =>
    request<{ ok: boolean }>(`/admin/print-jobs/${id}/retry`, {
      method: "POST",
      auth: true,
      scope: true,
    }),

  // ---- Ingredients and tech cards ----
  //
  // ⚠️ Admin only, and never part of the public menu: a recipe is a
  // competitor's shopping list with the quantities filled in.
  adminIngredients: () =>
    request<{ ingredients: Ingredient[]; countedAt: string | null }>(
      "/admin/ingredients",
      { auth: true, cache: "no-store", scope: true },
    ),
  adminSaveIngredient: (body: Partial<Ingredient> & { id?: string }) =>
    request<Ingredient>(
      body.id ? `/admin/ingredients/${body.id}` : "/admin/ingredients",
      { method: body.id ? "PUT" : "POST", auth: true, body, scope: true },
    ),
  adminDeleteIngredient: (id: string) =>
    request<{ ok: boolean }>(`/admin/ingredients/${id}`, {
      method: "DELETE",
      auth: true,
      scope: true,
    }),

  // Deliveries. ⚠️ Not stock: this records what came in and what it cost —
  // the prices become the ingredients' prices, dated by the invoice.
  adminPurchases: (params?: { from?: string; to?: string }) =>
    request<{ purchases: Purchase[]; spent: number }>(
      `/admin/purchases${reportQuery(params ?? {})}`,
      { auth: true, cache: "no-store", scope: true },
    ),
  adminCreatePurchase: (body: {
    at: string;
    supplier?: string;
    note?: string;
    lines: PurchaseLine[];
    total?: number;
  }) =>
    request<{ purchase: Purchase; pricesChanged: number }>("/admin/purchases", {
      method: "POST",
      auth: true,
      body,
      scope: true,
    }),
  adminDeletePurchase: (id: string) =>
    request<{ ok: boolean }>(`/admin/purchases/${id}`, {
      method: "DELETE",
      auth: true,
      scope: true,
    }),

  // Food that left without being sold. ⚠️ The reason is required by the
  // server, not only by the form.
  adminWriteOffs: (params?: { from?: string; to?: string }) =>
    request<{
      writeOffs: WriteOff[];
      value: number;
      reasons: WriteOffReason[];
    }>(`/admin/writeoffs${reportQuery(params ?? {})}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  adminCreateWriteOff: (body: {
    at: string;
    ingredientId: string;
    qty: number;
    reason: string;
  }) =>
    request<WriteOff>("/admin/writeoffs", {
      method: "POST",
      auth: true,
      body,
      scope: true,
    }),
  adminDeleteWriteOff: (id: string) =>
    request<{ ok: boolean }>(`/admin/writeoffs/${id}`, {
      method: "DELETE",
      auth: true,
      scope: true,
    }),

  // Counting the store. ⚠️ The sheet says what should be there and since when
  // — the figure is measured from the last count, not kept as a running
  // balance.
  adminStocktakeSheet: () =>
    request<{ rows: StocktakeSheetRow[]; since: string | null }>(
      "/admin/stocktake/sheet",
      { auth: true, cache: "no-store", scope: true },
    ),
  adminStocktakes: () =>
    request<{ stocktakes: Stocktake[] }>("/admin/stocktake", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  adminSaveStocktake: (body: {
    at?: string;
    note?: string;
    lines: { ingredientId: string; counted: number }[];
  }) =>
    request<Stocktake>("/admin/stocktake", {
      method: "POST",
      auth: true,
      body,
      scope: true,
    }),

  // What guests owe, and settling it. ⚠️ A debt is the sale itself — closed,
  // delivered and unpaid — so paying it marks that order paid, dated today.
  adminDebts: (userId?: string) =>
    request<{
      debts: DebtRow[];
      total: number;
      byUser: Record<string, number>;
    }>(`/admin/debts${userId ? `?userId=${userId}` : ""}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  adminPayDebt: (orderId: string, method: string, note?: string) =>
    request<{ ok: boolean }>(`/admin/debts/${orderId}/pay`, {
      method: "POST",
      auth: true,
      body: { method, note },
      scope: true,
    }),

  // How busy each kitchen is right now. Read by the orders board so a dispatcher
  // can see who is behind before deciding to move anything.
  adminBranchLoad: () =>
    request<BranchLoad[]>("/admin/branches/load", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),

  // Hand one order to another kitchen. ⚠️ The money does not change — the fee
  // and the total were agreed with the guest — and the order number keeps its
  // old branch prefix, because it is the guest's tracking link.
  moveOrderBranch: (id: string, branchId: string) =>
    request<Order>(`/admin/orders/${id}/branch`, {
      method: "PUT",
      body: { branchId },
      auth: true,
    }),

  adminOrder: (id: string) =>
    request<Order>(`/admin/orders/${id}`, { auth: true, cache: "no-store" }),

  // ---- Call centre ----

  // Who is ringing: the customer, what of theirs is in the kitchen right now,
  // what they usually order, and what was said last time. One request, because
  // an operator has seconds, not screens.
  adminLookup: (phone: string) =>
    request<CallerLookup>(`/admin/lookup?phone=${encodeURIComponent(phone)}`, {
      auth: true,
      cache: "no-store",
    }),

  // An order taken over the phone. Same pricing pipeline as the site — the
  // operator takes the order, they do not negotiate it.
  adminCreateOrder: (body: {
    customer: { name: string; phone: string };
    userId?: string;
    type: "delivery" | "pickup" | "dinein";
    branchId?: string;
    address?: OrderAddress;
    items: unknown[];
    paymentMethod: PaymentMethod;
    promoCode?: string;
    usePoints?: number;
    /** A pre-order: RFC 3339, the time the caller asked for. An operator is
     *  exempt from the timing rules the site enforces — "in twenty minutes"
     *  and "for the wedding in six weeks" are both normal on the phone. */
    scheduledAt?: string;
    // The call this order came out of, so the log row says what it produced.
    callId?: string;
  }) =>
    request<Order>("/admin/orders", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),

  // The bill preview for a basket an operator is assembling. Same pipeline as
  // the checkout, but the customer is named rather than signed in — their
  // points and "first order only" codes hang off the account, not a token.
  adminOrderQuote: (body: {
    items: { menuItemId: string; qty: number; options?: unknown[] }[];
    type: string;
    address?: { lat: number; lng: number };
    branchId?: string;
    userId?: string;
    promoCode?: string;
    usePoints?: number;
  }) =>
    request<OrderQuote>("/admin/orders/quote", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),

  adminCalls: (params?: {
    q?: string;
    outcome?: string;
    direction?: "in" | "out";
    operatorId?: string;
    from?: string;
    to?: string;
    // "open" is the queue of promises still owed; it sorts soonest-first.
    callback?: "open" | "done" | "any";
    limit?: number;
    before?: string;
  }) => {
    const qs = new URLSearchParams();
    if (params?.q) qs.set("q", params.q);
    if (params?.outcome) qs.set("outcome", params.outcome);
    if (params?.direction) qs.set("direction", params.direction);
    if (params?.operatorId) qs.set("operatorId", params.operatorId);
    if (params?.from) qs.set("from", params.from);
    if (params?.to) qs.set("to", params.to);
    if (params?.callback) qs.set("callback", params.callback);
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.before) qs.set("before", params.before);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<Call[]>(`/admin/calls${suffix}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    });
  },

  adminCallStats: (range?: { from?: string; to?: string }) => {
    const qs = new URLSearchParams();
    if (range?.from) qs.set("from", range.from);
    if (range?.to) qs.set("to", range.to);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<CallStats>(`/admin/calls/stats${suffix}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    });
  },

  createCall: (body: {
    direction: "in" | "out";
    phone: string;
    userId?: string;
    name?: string;
    outcome: CallOutcome;
    note?: string;
    callbackAt?: string;
    seconds?: number;
    orderId?: string;
    reservationId?: string;
  }) =>
    request<Call>("/admin/calls", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),

  // Editable, unlike the audit log: a call is written down while somebody is
  // still talking, and an operator who cannot fix a row stops filling it in.
  updateCall: (
    id: string,
    body: {
      outcome?: CallOutcome;
      note?: string;
      name?: string;
      callbackAt?: string;
      callbackDone?: boolean;
      seconds?: number;
      orderId?: string;
      reservationId?: string;
    },
  ) => request<Call>(`/admin/calls/${id}`, { method: "PUT", body, auth: true }),
  // ---- Couriers (admin) ----
  adminCouriers: () =>
    request<Courier[]>("/admin/couriers", { auth: true, scope: true }),
  adminCourier: (id: string) =>
    request<AdminCourierDetail>(`/admin/couriers/${id}`, { auth: true }),
  createCourier: (body: Partial<Courier> & { password: string }) =>
    request<Courier>("/admin/couriers", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),
  updateCourier: (id: string, body: Partial<Courier> & { password?: string }) =>
    request<Courier>(`/admin/couriers/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  // The courier handed cash back. Recorded as a ledger entry, not a counter
  // reset: "how much did Aziz hand in last Tuesday?" has to stay answerable.
  settleCourierCash: (id: string, amount: number, note?: string) =>
    request<{ ok: boolean }>(`/admin/couriers/${id}/settle`, {
      method: "POST",
      body: { amount, note },
      auth: true,
    }),

  deleteCourier: (id: string) =>
    request<{ ok: boolean }>(`/admin/couriers/${id}`, {
      method: "DELETE",
      auth: true,
    }),
  // Dashboard numbers for a period (both dates optional = all time). Counted
  // in the database, not from a page of orders in the browser.
  // ---- Bookings (admin) ----
  adminReservations: (params?: {
    scope?: "upcoming" | "today" | "past" | "all";
    status?: ReservationStatus;
    q?: string;
  }) => {
    const qs = new URLSearchParams();
    if (params?.scope) qs.set("scope", params.scope);
    if (params?.status) qs.set("status", params.status);
    if (params?.q) qs.set("q", params.q);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<Reservation[]>(`/admin/reservations${suffix}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    });
  },
  adminCreateReservation: (body: {
    tableId: string;
    at: string;
    guests: number;
    customer: { name: string; phone: string };
    comment?: string;
  }) =>
    request<Reservation>("/admin/reservations", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),
  updateReservationStatus: (
    id: string,
    status: ReservationStatus,
    reason?: string,
  ) =>
    request<Reservation>(`/admin/reservations/${id}/status`, {
      method: "PUT",
      body: { status, reason },
      auth: true,
    }),
  deleteReservation: (id: string) =>
    request<{ ok: boolean }>(`/admin/reservations/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // Does the owner's own domain already point at this server? Answered by
  // resolving both it and the address the site is already served on, so there
  // is no configured IP to drift out of date.
  adminDomainCheck: (domain: string) =>
    request<{
      domain: string;
      found: string[];
      expected: string[];
      ok: boolean;
    }>(`/admin/domain-check?domain=${encodeURIComponent(domain)}`, {
      auth: true,
      cache: "no-store",
    }),

  // The last step of the guide: ask the platform to actually serve this
  // hostname. Never throws for a refusal — "your DNS does not point here yet"
  // is an answer the owner acts on, not an exception.
  adminDomainConnect: (domain: string, remove = false) =>
    request<{
      ok: boolean;
      domain: string;
      domains?: string[];
      found?: string[];
      expected?: string[];
      removed?: boolean;
      note?: string;
      /** This install is standalone — there is no platform to ask. */
      unsupported?: boolean;
      error?: string;
    }>("/admin/domain-connect", {
      method: "POST",
      body: { domain, remove },
      auth: true,
    }),

  // Polled by the panel to notice new orders and bookings (plays a sound).
  adminAlerts: () =>
    request<AdminAlerts>("/admin/alerts", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),

  /** Menu analysis for a period: ABC (share of takings) × XYZ (steadiness). */
  abcXyz: (range?: { from?: string; to?: string }) => {
    // Through `reportQuery` like every other report, so the note under the
    // table arrives in the panel's language.
    const suffix = reportQuery({ from: range?.from, to: range?.to });
    return request<AbcXyzResponse>(`/admin/reports/abc-xyz${suffix}`, {
      auth: true,
      scope: true,
    });
  },

  /** Dishes to suggest alongside a dish or a whole basket.
   *
   *  A POST because the basket is the input: a URL carrying eight dish ids is
   *  one that gets truncated, logged and cached by something along the way. */
  recommendations: (body: { itemIds: string[]; branchId?: string }) =>
    request<MenuItem[]>("/recommendations", { method: "POST", body }),

  /** The VAPID public key a browser needs before it can subscribe.
   *
   *  Public by definition — it is handed to every visitor who is offered the
   *  permission, exactly like the map key. */
  pushKey: () => request<{ publicKey: string }>("/push/key"),

  /** Registers this browser against the signed-in customer. */
  pushSubscribe: (sub: {
    endpoint: string;
    keys: { p256dh: string; auth: string };
    device?: string;
  }) =>
    request<{ ok: boolean }>("/users/me/push", {
      method: "POST",
      body: sub,
      auth: true,
    }),

  /** Forgets one browser, or every browser when no endpoint is given —
   *  "stop notifying me" is what a guest means when they switch it off on a
   *  phone they may not even be holding. */
  pushUnsubscribe: (endpoint?: string) =>
    request<{ ok: boolean }>("/users/me/push", {
      method: "DELETE",
      body: { endpoint: endpoint ?? "" },
      auth: true,
    }),

  /** The customer base ranked against itself: recency, frequency, money. */
  adminRfm: () => request<RfmResponse>("/admin/rfm", { auth: true }),

  /** This admin's dashboard layout, with the catalogue of tiles it may name. */
  adminDashboardPrefs: () =>
    request<DashboardPrefs>("/admin/me/dashboard", { auth: true }),

  /** Stores this admin's dashboard layout. */
  adminSaveDashboard: (prefs: { hidden: string[]; order: string[] }) =>
    request<{ hidden: string[]; order: string[] }>("/admin/me/dashboard", {
      method: "PUT",
      body: prefs,
      auth: true,
    }),

  /** Sales over time, plus the comparison with the period before it. */
  salesReport: (params: { from?: string; to?: string; group?: string }) =>
    request<SalesReportResponse>(`/admin/reports/sales${reportQuery(params)}`, {
      auth: true,
      scope: true,
    }),

  /** Which door the orders came in through, and how they were fulfilled. */
  /** The till's history: closed shifts and their differences.
   *
   *  Same shape as the other reports and the same Excel route, because it is
   *  the same report — the screen and the spreadsheet must never be two counts
   *  of the same money. */
  cashReport: (params: { from?: string; to?: string }) =>
    request<CashReportResponse>(`/admin/reports/cash${reportQuery(params)}`, {
      auth: true,
      scope: true,
    }),
  /** Money in, money out, and what is still owed to us.
   *
   *  ⚠️ The note travels with it: this is cash movement, not profit, and the
   *  sentence saying so is the server's — the screen and the spreadsheet must
   *  never carry two different warnings. */
  financeReport: (params: { from?: string; to?: string }) =>
    request<FinanceReportResponse>(
      `/admin/reports/finance${reportQuery(params)}`,
      { auth: true, scope: true },
    ),
  /** What came in against what the dishes sold should have used.
   *
   *  ⚠️ A flow, never a balance: the note travels with it and says so. */
  stockReport: (params: { from?: string; to?: string }) =>
    request<StockReportResponse>(`/admin/reports/stock${reportQuery(params)}`, {
      auth: true,
      scope: true,
    }),
  channelReport: (params: { from?: string; to?: string }) =>
    request<ChannelReportResponse>(
      `/admin/reports/channels${reportQuery(params)}`,
      {
        auth: true,
        scope: true,
      },
    ),

  /** Every courier's period on one page.
   *
   *  Named `admin*` to keep it apart from `staffReport`, which is the employee's
   *  own screen and authenticates with a different token entirely. */
  adminCourierReport: (params: { from?: string; to?: string }) =>
    request<CourierReportResponse>(
      `/admin/reports/couriers${reportQuery(params)}`,
      {
        auth: true,
        scope: true,
      },
    ),

  /** Every employee's attendance and pay for a period. */
  adminStaffReport: (params: { from?: string; to?: string }) =>
    request<StaffReportResponse>(`/admin/reports/staff${reportQuery(params)}`, {
      auth: true,
      scope: true,
    }),

  adminStats: (range?: { from?: string; to?: string }) => {
    const qs = new URLSearchParams();
    if (range?.from) qs.set("from", range.from);
    if (range?.to) qs.set("to", range.to);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<AdminStats>(`/admin/stats${suffix}`, {
      auth: true,
      scope: true,
    });
  },

  // Corrects the delivery point of an existing order (the customer may have
  // dropped their pin on the wrong building). Money is left as agreed; the
  // response reports what the corrected point would cost today.
  updateOrderAddress: (
    orderId: string,
    body: { lat: number; lng: number; text: string; comment: string },
  ) =>
    request<{
      address: { text: string; lat: number; lng: number; comment: string };
      deliveryZone: string;
      distanceKm: number;
      quotedFee: number;
      available: boolean;
      deliveryFee: number;
    }>(`/admin/orders/${orderId}/address`, {
      method: "PUT",
      body,
      auth: true,
    }),

  assignCourier: (orderId: string, courierId: string) =>
    request<{ ok: boolean }>(`/admin/orders/${orderId}/courier`, {
      method: "PUT",
      body: { courierId },
      auth: true,
    }),

  // ---- Feedback and complaints ----
  adminFeedback: (params?: { filter?: string; q?: string }) => {
    const qs = new URLSearchParams();
    if (params?.filter) qs.set("filter", params.filter);
    if (params?.q) qs.set("q", params.q);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<FeedbackList>(`/admin/feedback${suffix}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    });
  },
  handleFeedback: (id: string, resolution: string) =>
    request<Feedback>(`/admin/feedback/${id}/handled`, {
      method: "PUT",
      body: { resolution },
      auth: true,
    }),

  // Put one review on the public site, or take it back off. One at a time and
  // never in bulk: everything in this list was written to the restaurant, not
  // to the internet, so publishing is an act somebody performs while looking
  // at the actual words.
  publishFeedback: (id: string, isPublic: boolean) =>
    request<Feedback>(`/admin/feedback/${id}/public`, {
      method: "PUT",
      body: { public: isPublic },
      auth: true,
    }),

  // ---- Campaigns and promo codes ----
  adminPromotions: (trigger?: PromotionTrigger) =>
    request<Promotion[]>(
      `/admin/promotions${trigger ? `?trigger=${trigger}` : ""}`,
      { auth: true, cache: "no-store", scope: true },
    ),
  createPromotion: (body: Partial<Promotion>) =>
    request<Promotion>("/admin/promotions", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),
  updatePromotion: (id: string, body: Partial<Promotion>) =>
    request<Promotion>(`/admin/promotions/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  // Who used a code and how many of them — counted from the orders.
  promotionUsage: (id: string) =>
    request<PromotionUsage>(`/admin/promotions/${id}/usage`, {
      auth: true,
      cache: "no-store",
    }),
  deletePromotion: (id: string) =>
    request<{ ok: boolean }>(`/admin/promotions/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // ---- Outside delivery services ----
  adminProviders: () =>
    request<DeliveryProvider[]>("/admin/delivery-providers", { auth: true }),
  createProvider: (body: Partial<DeliveryProvider>) =>
    request<DeliveryProvider>("/admin/delivery-providers", {
      method: "POST",
      body,
      auth: true,
    }),
  updateProvider: (id: string, body: Partial<DeliveryProvider>) =>
    request<DeliveryProvider>(`/admin/delivery-providers/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  deleteProvider: (id: string) =>
    request<{ ok: boolean }>(`/admin/delivery-providers/${id}`, {
      method: "DELETE",
      auth: true,
    }),
  callProvider: (
    orderId: string,
    body: {
      providerId: string;
      trackingId?: string;
      note?: string;
      cost?: number;
    },
  ) =>
    request<{ ok: boolean }>(`/admin/orders/${orderId}/external-delivery`, {
      method: "PUT",
      body,
      auth: true,
    }),

  // The server files the request with the service itself (kind "api").
  callProviderApi: (orderId: string, providerId: string) =>
    request<{ ok: boolean; externalDelivery: ExternalDelivery }>(
      `/admin/orders/${orderId}/external-delivery/call`,
      { method: "POST", body: { providerId }, auth: true },
    ),
  syncProviderApi: (orderId: string) =>
    request<{ ok: boolean; externalDelivery: ExternalDelivery }>(
      `/admin/orders/${orderId}/external-delivery/sync`,
      { method: "POST", body: {}, auth: true },
    ),
  cancelProviderApi: (orderId: string, paid = false) =>
    request<{ ok: boolean }>(
      `/admin/orders/${orderId}/external-delivery/cancel`,
      { method: "POST", body: { paid }, auth: true },
    ),

  // ---- Courier app ----
  courierLogin: (username: string, password: string) =>
    request<{ token: string; courier: Courier }>("/courier/login", {
      method: "POST",
      body: { username, password },
    }),
  courierMe: () =>
    request<Courier>("/courier/me", { bearer: getCourierToken() }),
  courierSetStatus: (status: CourierStatus) =>
    request<{ status: CourierStatus }>("/courier/status", {
      method: "PUT",
      body: { status },
      bearer: getCourierToken(),
    }),
  courierSendLocation: (
    points: { lat: number; lng: number; accuracy: number; at: number }[],
  ) =>
    request<{ ok: boolean }>("/courier/location", {
      method: "POST",
      body: { points },
      bearer: getCourierToken(),
    }),
  courierStats: () =>
    request<CourierStats>("/courier/stats", { bearer: getCourierToken() }),
  courierHistory: () =>
    request<CourierOrderRow[]>("/courier/history", {
      bearer: getCourierToken(),
    }),
  courierOrders: (all = false) =>
    request<Order[]>(`/courier/orders${all ? "?all=1" : ""}`, {
      bearer: getCourierToken(),
    }),
  courierAdvanceOrder: (id: string, status: OrderStatus) =>
    request<{ status: OrderStatus }>(`/courier/orders/${id}/status`, {
      method: "PUT",
      body: { status },
      bearer: getCourierToken(),
    }),

  // `reason` is only stored for "cancelled" — the customer reads it on the
  // tracking page.
  updateOrderStatus: (id: string, status: OrderStatus, reason?: string) =>
    request<{ status: OrderStatus }>(`/admin/orders/${id}/status`, {
      method: "PUT",
      body: { status, reason: reason ?? "" },
      auth: true,
    }),

  // ---- Panel accounts + activity log (owner only) ----
  adminAccounts: () => request<AdminUser[]>("/admin/accounts", { auth: true }),
  createAdminAccount: (body: {
    userId: string;
    username: string;
    password: string;
    role: "owner" | "manager";
  }) =>
    request<AdminUser>("/admin/accounts", { method: "POST", body, auth: true }),
  updateAdminAccount: (
    id: string,
    body: { role?: string; password?: string },
  ) =>
    request<AdminUser>(`/admin/accounts/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  deleteAdminAccount: (id: string) =>
    request<{ ok: boolean }>(`/admin/accounts/${id}`, {
      method: "DELETE",
      auth: true,
    }),
  // ---- Online payment (admin) ----

  adminPaymentSettings: () =>
    request<PaymentSettings>("/admin/payments", {
      auth: true,
      cache: "no-store",
    }),
  // Secrets left empty keep whatever is stored — the form never sees them.
  updatePaymentSettings: (body: PaymentSettingsInput) =>
    request<PaymentSettings>("/admin/payments", {
      method: "PUT",
      body,
      auth: true,
    }),
  // ---- SMS gateway (admin) ----
  // Per restaurant, not per platform: each one signs its own contract with
  // Eskiz / Play Mobile / getsms.uz / OneSignal and pays its own bill.

  adminSMSSettings: () =>
    request<SMSSettings>("/admin/sms", {
      auth: true,
      cache: "no-store",
    }),
  // Passwords left empty keep whatever is stored — the form never sees them.
  updateSMSSettings: (body: SMSSettingsInput) =>
    request<SMSSettings>("/admin/sms", {
      method: "PUT",
      body,
      auth: true,
    }),
  // Sends one real message. Never throws for a gateway that refused — that is
  // an answer to show, not an exception.
  //
  //  `probe` asks a different question: it sends the gateway's own fixed test
  //  wording (Eskiz only), which works on an account whose template has not
  //  been moderated. It proves the credentials connect and nothing more, so a
  //  successful probe is never recorded as a passing test.
  testSMS: (phone?: string, probe = false) =>
    request<{
      ok: boolean;
      /** Which question was asked — a delivered probe is information, not
       *  proof that a login code would arrive. */
      probe?: boolean;
      message: string;
      phone: string;
      provider: string;
      /** The exact wording to submit to the gateway for moderation. */
      template?: string;
    }>("/admin/sms/test", {
      method: "POST",
      body: { phone: phone ?? "", probe },
      auth: true,
    }),

  // Every attempt against one order, successful or not.
  adminOrderPayments: (id: string) =>
    request<Payment[]>(`/admin/orders/${id}/payments`, {
      auth: true,
      cache: "no-store",
    }),

  // ---- Fiscalisation: registering the sale with the tax committee ----
  // Per branch (scope: true), like the POS settings below and for the same
  // reason: a cash register is registered to a place.

  // The list is fetched rather than hard-coded here, so that a provider whose
  // adapter is not built yet cannot be offered as if it were.
  fiscalProviders: () =>
    request<FiscalProviderInfo[]>("/admin/fiscal/providers", {
      auth: true,
      cache: "no-store",
    }),
  fiscalSettings: () =>
    request<FiscalSettings>("/admin/fiscal", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  // Secrets left empty keep whatever is stored. Saving an incomplete form is
  // allowed; enabling it is what the server checks.
  updateFiscal: (body: FiscalSettingsInput) =>
    request<FiscalSettings>("/admin/fiscal", {
      method: "PUT",
      body,
      auth: true,
      scope: true,
    }),
  // Says what it connected to, not just "ok". Never throws for a bad
  // connection — that is an answer, not an exception.
  pingFiscal: () =>
    request<{ ok: boolean; message: string }>("/admin/fiscal/ping", {
      method: "POST",
      auth: true,
      scope: true,
    }),
  /** Mint the relay's credential for this branch.
   *
   *  ⚠️ **Returned exactly once.** It is stored only to compare against, which
   *  is what makes rotating it a real revocation instead of a second working
   *  key — and why the panel has to show it until the owner has copied it. */
  /** Bind a monoblock to a branch, or (`rotate`) cut every one of them loose.
   *
   *  ⚠️ Rotating kills **all** of this branch's till screens, not just a lost
   *  one — the tokens carry no device identity, so there is nothing finer to
   *  revoke. The fix is walking to each monoblock with a new link. */
  tillDeviceToken: (branchId: string, rotate = false) =>
    request<{
      token: string;
      branchId: string;
      branchName: string;
      version: number;
    }>(`/admin/branches/${branchId}/till-token${rotate ? "?rotate=1" : ""}`, {
      auth: true,
    }),
  fiscalAgentToken: () =>
    request<{ token: string }>("/admin/fiscal/agent-token", {
      method: "POST",
      auth: true,
      scope: true,
    }),

  // ---- Receipt designs ----
  //
  // ⚠️ The preview is rendered on the server by the printer's own code, so what
  // the owner sees is what the paper says. A preview drawn here would be a
  // second implementation of the same character grid.
  adminReceipts: () =>
    request<ReceiptSettings>("/admin/receipts", {
      auth: true,
      scope: true,
      cache: "no-store",
    }),
  saveReceipts: (body: {
    kitchen: ReceiptTemplate;
    till: ReceiptTemplate;
    customer: ReceiptTemplate;
    printers: Printer[];
  }) =>
    request<ReceiptSettings>("/admin/receipts", {
      method: "PUT",
      body,
      auth: true,
      scope: true,
    }),
  /** Print the sample receipt on one printer.
   *
   *  ⚠️ The only thing that answers "is this address reachable" — a correct
   *  address and a printer that is off, on another subnet or shared under a
   *  different name look identical from a form. */
  testPrint: (printerId: string) =>
    request<{ queued: number }>("/admin/receipts/test-print", {
      method: "POST",
      body: { printerId },
      auth: true,
      scope: true,
    }),
  previewReceipts: (body: {
    kitchen: ReceiptTemplate;
    till: ReceiptTemplate;
    customer: ReceiptTemplate;
  }) =>
    request<ReceiptPreview>("/admin/receipts/preview", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),

  // ---- Roles ----
  //
  // ⚠️ Reading is open to anybody who manages staff — assigning somebody a role
  // needs the list. Writing is owner-only: deciding who may take money out of
  // the restaurant is not a shift-level decision.
  adminRoles: () =>
    request<{ roles: StaffRole[]; perms: PermOption[] }>("/admin/roles", {
      auth: true,
      cache: "no-store",
    }),
  createRole: (body: { name: string; perms: string[] }) =>
    request<StaffRole>("/admin/roles", { method: "POST", body, auth: true }),
  updateRole: (id: string, body: { name: string; perms: string[] }) =>
    request<{ ok: boolean }>(`/admin/roles/${id}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  /** ⚠️ Refused while anybody holds it: a staff record whose role vanished
   *  falls back to the pre-role flags, which for most people means losing the
   *  till mid-shift with nothing on the tablet explaining why. */
  deleteRole: (id: string) =>
    request<{ deleted: boolean }>(`/admin/roles/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // ---- The cash drawer ----
  //
  // Three actions, all of which move physical cash and all of which need a
  // name against them. The screen's whole purpose is the difference between
  // what the drawer should hold and what it does.
  cashShift: () =>
    request<{
      open: CashShift | null;
      last?: CashShift | null;
      figures?: CashFigures;
      entries?: CashEntry[];
    }>("/admin/cash/shift", { auth: true, scope: true, cache: "no-store" }),
  openCashShift: (body: { openingFloat: number; note?: string }) =>
    request<CashShift>("/admin/cash/shift/open", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),
  /** Count the drawer and record the difference.
   *
   *  ⚠️ Answers `fiscalNote` when the register's day could not also be asked to
   *  end — usually because receipts are still unfiled. The count itself always
   *  succeeds: refusing to record it because a PC in the corner is behind would
   *  lose the count and leave the money unexplained. */
  closeCashShift: (body: {
    counted: number;
    varianceNote?: string;
    note?: string;
  }) =>
    request<{ shift: CashShift; figures: CashFigures; fiscalNote?: string }>(
      "/admin/cash/shift/close",
      { method: "POST", body, auth: true, scope: true },
    ),
  addCashEntry: (body: {
    kind: "in" | "out";
    category: string;
    amount: number;
    note?: string;
  }) =>
    request<CashEntry>("/admin/cash/entries", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),

  // ---- POS: the till the restaurant already runs ----
  // Everything here is per branch (scope: true): a chain has one terminal
  // group per kitchen.

  adminPOS: () =>
    request<POSSettings>("/admin/pos", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  // Secrets left empty keep whatever is stored.
  updatePOS: (body: POSSettingsInput) =>
    request<POSSettings>("/admin/pos", {
      method: "PUT",
      body,
      auth: true,
      scope: true,
    }),
  // Proves the credentials and says what it connected to. Never throws for a
  // bad connection — that is an answer, not an exception.
  pingPOS: () =>
    request<{ ok: boolean; message: string }>("/admin/pos/ping", {
      method: "POST",
      auth: true,
      scope: true,
    }),
  posProducts: () =>
    request<POSProduct[]>("/admin/pos/products", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  posMapping: () =>
    request<POSMapping[]>("/admin/pos/mapping", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  savePOSMapping: (
    items: {
      menuItemId: string;
      posProductId: string;
      posProductName: string;
    }[],
  ) =>
    request<{ saved: number; removed: number }>("/admin/pos/mapping", {
      method: "PUT",
      body: { items },
      auth: true,
      scope: true,
    }),
  // Copies another branch's links into the branch the lens is on. A chain on one
  // iiko account has identical product ids, and retyping 200 rows is where the
  // wrong id gets in.
  // ---- Campaigns ----
  //
  // Owner only, and every call here is about spending money on real people's
  // phones: the preview runs the same audience query the send does, so the
  // number in the confirmation is the number that gets billed.
  adminSegments: () =>
    request<SegmentRow[]>("/admin/segments", { auth: true, cache: "no-store" }),
  adminCampaigns: () =>
    request<Campaign[]>("/admin/campaigns", { auth: true, cache: "no-store" }),
  campaignPreview: (body: {
    segment?: string;
    text: string;
    channel?: string;
    image?: string;
    /** One named customer instead of a segment. */
    userId?: string;
  }) =>
    request<CampaignPreview>("/admin/campaigns/preview", {
      method: "POST",
      body,
      auth: true,
    }),
  sendCampaign: (body: {
    segment?: string;
    text: string;
    channel?: string;
    image?: string;
    userId?: string;
  }) =>
    request<{ id: string; recipients: number; status: string }>(
      "/admin/campaigns",
      { method: "POST", body, auth: true },
    ),

  /** Whether the platform has opened the data-export window for this install.
   *  Owner only; the panel shows nothing at all when it is closed. */
  adminExportStatus: () =>
    request<{
      allowed: boolean;
      reason?: string;
      expiresAt?: string;
      downloads?: number;
    }>("/admin/export", { auth: true, cache: "no-store" }),

  copyPOSMapping: (fromBranchId: string, overwrite: boolean) =>
    request<{ copied: number; skipped: number; total: number }>(
      "/admin/pos/mapping/copy",
      {
        method: "POST",
        body: { fromBranchId, overwrite },
        auth: true,
        scope: true,
      },
    ),
  // ---- Stop list ----
  //
  // What is off sale at this branch right now, from both writers at once: the
  // counter's own taps and the till's stop list. One call, because the screen
  // asks one question and three calls give three chances to answer half of it.
  adminStopList: () =>
    request<StopList>("/admin/stop-list", {
      auth: true,
      cache: "no-store",
      scope: true,
    }),
  // Read the till now. The poller already runs; this is for the minute after
  // the dish links were edited, with somebody watching.
  syncPOSStopList: () =>
    request<{ ok: boolean; message?: string; stopped?: number }>(
      "/admin/pos/stop-list/sync",
      { method: "POST", auth: true, scope: true },
    ),

  // The retry button on a receipt.
  sendOrderToPOS: (id: string) =>
    request<{ ok: boolean; message?: string; pos?: OrderPOS }>(
      `/admin/orders/${id}/pos`,
      { method: "POST", auth: true },
    ),

  // ---- Telephony (onlinePBX) ----

  adminPBX: () =>
    request<PBXSettings>("/admin/pbx", { auth: true, cache: "no-store" }),
  updatePBX: (body: PBXSettingsInput) =>
    request<PBXSettings>("/admin/pbx", { method: "PUT", body, auth: true }),
  pingPBX: () =>
    request<{ ok: boolean; message: string }>("/admin/pbx/ping", {
      method: "POST",
      auth: true,
    }),
  // Polled by the call desk. Deliberately tiny: it runs every few seconds on
  // every open desk, the same way the new-order chime does.
  liveCall: () =>
    request<LiveCall>("/admin/calls/live", { auth: true, cache: "no-store" }),
  // Rings the operator's own handset first, then the customer.
  dial: (phone: string, callId?: string) =>
    request<{ ok: boolean; message?: string; uuid?: string }>(
      "/admin/calls/dial",
      { method: "POST", body: { phone, callId }, auth: true },
    ),
  // A fresh signed link to the recording, asked for at the moment somebody
  // presses play.
  callRecording: (id: string) =>
    request<{ url: string; message?: string }>(`/admin/calls/${id}/recording`, {
      auth: true,
      cache: "no-store",
    }),
  // Which handset is mine. Its own call rather than part of the credentials
  // form: needing a password to change a desk number means it never gets set.
  setMyExtension: (extension: string) =>
    request<{ extension: string }>("/admin/me/extension", {
      method: "PUT",
      body: { extension },
      auth: true,
    }),

  adminLogs: (params?: {
    adminId?: string;
    action?: string;
    // Free text: an order number or id pasted from a receipt, a name, a note.
    q?: string;
    limit?: number;
    before?: string;
  }) => {
    const q = new URLSearchParams();
    if (params?.adminId) q.set("adminId", params.adminId);
    if (params?.action) q.set("action", params.action);
    if (params?.q) q.set("q", params.q);
    if (params?.limit) q.set("limit", String(params.limit));
    if (params?.before) q.set("before", params.before);
    const qs = q.toString();
    return request<AdminLog[]>(`/admin/logs${qs ? `?${qs}` : ""}`, {
      auth: true,
    });
  },

  // Restaurant settings (full replace — backend upserts the whole document).
  // Partial on purpose: the server writes only the keys the body carries
  // (UpdateRestaurant), so sending a subset is the supported shape — and the
  // safe one, because a whole document sent from a page loaded ten minutes ago
  // puts ten-minute-old values back over everything.
  updateRestaurant: (body: Partial<Restaurant>) =>
    request<Restaurant>("/admin/restaurant", {
      method: "PUT",
      body,
      auth: true,
    }),

  // ---- Staff app (/staff) ----
  staffLogin: (username: string, password: string) =>
    request<{ token: string; staff: Staff }>("/staff/login", {
      method: "POST",
      body: { username, password },
    }),
  staffMe: () => request<StaffMe>("/staff/me", { bearer: getStaffToken() }),
  /** Hand the till over to whoever tapped their PIN.
   *
   *  ⚠️ Sent with the **device's** token — this names a person on a screen that
   *  is already authenticated, it does not authenticate a stranger. The token
   *  that comes back is weaker on purpose: it runs the floor and the drawer and
   *  reaches nothing else. */
  /** Does this screen have to be unlocked at all?
   *
   *  ⚠️ Answered from the data — "has anybody here been given a PIN" — not from
   *  a setting. A restaurant that has handed out none keeps working exactly as
   *  before, so the upgrade cannot lock a live till out over a checkbox nobody
   *  was told to tick. */
  /** Does this screen lock, and is the device still bound?
   *
   *  ⚠️ **A dead device token is dropped rather than displayed.** It expires,
   *  or the branch's till version is bumped when a monoblock walks out of the
   *  building — and until now the till answered that with the server's own
   *  words, "invalid token", on the lock screen, with no way forward: the pad
   *  refuses, there is no button, and the tablet is useless mid-service. The
   *  token is the one thing that is certainly wrong, so it is cleared and the
   *  call retried the way an unbound till works — with the staff login, which
   *  is exactly the path every till used before device binding existed.
   *
   *  The panel's re-binding link still has to be opened eventually; this is
   *  what keeps the restaurant selling until somebody does. */
  tillSession: async () => {
    const device = hasTillDevice();
    try {
      return await request<{ pinsUsed: boolean }>("/staff/till/session", {
        bearer: getDeviceToken(),
        cache: "no-store",
      });
    } catch (err) {
      if (!device || !(err instanceof ApiError) || err.status !== 401)
        throw err;
      clearTillDeviceToken();
      return request<{ pinsUsed: boolean }>("/staff/till/session", {
        bearer: getStaffToken(),
        cache: "no-store",
      });
    }
  },
  tillUnlock: (pin: string) =>
    request<{ token: string; staff: TillPerson }>("/staff/till/unlock", {
      method: "POST",
      body: { pin },
      bearer: getDeviceToken(),
    }),
  /** Set or clear an employee's till code. Empty clears it.
   *
   *  ⚠️ Its own call rather than a field on the staff form: a form that does not
   *  show the PIN would send it empty on every save and lock that person out of
   *  the till — the same trap as branch.soldOut. */
  setStaffPin: (id: string, pin: string) =>
    request<{ hasPin: boolean }>(`/admin/staff/${id}/pin`, {
      method: "PUT",
      body: { pin },
      auth: true,
    }),
  // The one button the app has. Position is mandatory — the server checks it
  // against the branch and refuses from too far away.
  staffClock: (
    action: "in" | "out",
    at: { lat: number; lng: number; accuracy: number },
    // Scanned from the branch screen; only required when the branch asks.
    code?: string,
  ) =>
    request<Shift>("/staff/clock", {
      method: "POST",
      body: { action, ...at, code: code ?? "" },
      bearer: getStaffToken(),
    }),

  /** Signing in from inside the Telegram mini app.
   *
   *  ⚠️ `initData` is passed through exactly as Telegram gave it: it is a signed
   *  string, and re-encoding it changes the bytes the signature covers. The
   *  server verifies it with the restaurant's own bot token — nothing here is
   *  trusted. */
  telegramLogin: (initData: string) =>
    request<{
      token: string;
      user: SiteUser;
      needsPhone: boolean;
      /** Never chosen a language — the mini app asks before anything else. */
      needsLang: boolean;
    }>("/auth/telegram", { method: "POST", body: { initData } }),
  /** A phone number Telegram vouched for. Signed in only — the number is written
   *  to the account already holding the session. */
  telegramPhone: (contact: string) =>
    request<{ user: SiteUser }>("/users/me/telegram/phone", {
      method: "POST",
      body: { contact },
      bearer: getUserToken(),
    }),
  /** The language the guest chose, kept on their account.
   *
   *  ⚠️ Not part of `updateProfile`: that call is the profile form and writes
   *  name and addresses every time, so sending a language through it from the
   *  mini app's first screen would blank the name of a guest who had one. */
  /** "Write to us" from the contact page. Signed in only — the server refuses
   *  otherwise, and the form says so before anything is typed. */
  submitSiteFeedback: (body: { rating: number; comment: string }) =>
    request<{ ok: boolean }>("/feedback", {
      method: "POST",
      body,
      bearer: getUserToken(),
    }),
  /** Dishes the guest saved. Kept on the account, so a heart survives the next
   *  device — see handlers/favorites.go. */
  favorites: () =>
    request<MenuItem[]>("/users/me/favorites", {
      bearer: getUserToken(),
      cache: "no-store",
    }),
  toggleFavorite: (id: string) =>
    request<{ on: boolean; favorites: string[] }>(`/users/me/favorites/${id}`, {
      method: "POST",
      bearer: getUserToken(),
    }),
  setUserLang: (lang: string) =>
    request<{ ok: boolean; lang: string }>("/users/me/lang", {
      method: "PUT",
      body: { lang },
      bearer: getUserToken(),
    }),

  // ---- The restaurant's own Telegram bot ----
  //
  // Owner only. The token is never returned; the check button is what fills in
  // the bot's username and therefore the mini app link.
  adminTelegram: () =>
    request<TelegramSettings>("/admin/telegram", {
      auth: true,
      cache: "no-store",
    }),
  updateTelegram: (body: { enabled: boolean; botToken?: string }) =>
    request<TelegramSettings>("/admin/telegram", {
      method: "PUT",
      body,
      auth: true,
    }),
  pingTelegram: () =>
    request<{ ok: boolean; message: string }>("/admin/telegram/ping", {
      method: "POST",
      auth: true,
    }),

  // ---- Kitchen screen (KDS) ----
  //
  // Staff token, not an admin one: the tablet at the pass is shared and never
  // logs out, so what it holds must be able to see the tickets and nothing
  // else. The branch is taken from the employee server-side — there is no
  // parameter here to get wrong.
  staffKitchen: () =>
    request<KitchenTicket[]>("/staff/kitchen", {
      bearer: getStaffToken(),
      cache: "no-store",
    }),
  staffKitchenAction: (id: string, action: "start" | "ready") =>
    request<{ ok: boolean }>(`/staff/kitchen/orders/${id}`, {
      method: "PUT",
      body: { action },
      bearer: getStaffToken(),
    }),

  // ---- The till (/kassa and the floor screen) ----
  //
  // One set of endpoints for both screens: they share every rule and differ
  // only in what the person holding the tablet is allowed to do, which the
  // server decides per action. Two API surfaces would have meant two ways to
  // price the same table.
  tillChecks: (mine = false) =>
    request<{ checks: Check[] }>(`/staff/checks${mine ? "?mine=1" : ""}`, {
      bearer: tillBearer(),
      cache: "no-store",
    }),
  tillCheck: (id: string) =>
    request<Check>(`/staff/checks/${id}`, {
      bearer: tillBearer(),
      cache: "no-store",
    }),
  tillOpenCheck: (body: {
    tableId?: string;
    guests?: number;
    serverId?: string;
  }) =>
    request<Check>("/staff/checks", {
      method: "POST",
      body,
      bearer: tillBearer(),
    }),
  tillUpdateCheck: (
    id: string,
    body: {
      guests?: number;
      serverId?: string;
      tableId?: string;
      comment?: string;
    },
  ) =>
    request<Check>(`/staff/checks/${id}`, {
      method: "PUT",
      body,
      bearer: tillBearer(),
    }),
  tillAddLines: (
    id: string,
    items: {
      menuItemId: string;
      qty: number;
      options?: OrderItemOption[];
      comment?: string;
    }[],
  ) =>
    request<Check>(`/staff/checks/${id}/lines`, {
      method: "POST",
      body: { items },
      bearer: tillBearer(),
    }),
  /** Remove a line. On an unfired line this is a typo being corrected and needs
   *  no reason; on a fired one the server demands a reason and a cashier. */
  tillVoidLine: (
    id: string,
    lineId: string,
    body?: {
      qty?: number;
      reason?: string;
      wasted?: boolean;
      /** A code from somebody who may write off cooked food, when the person
       *  at the screen may not. Empty is the ordinary case. */
      pin?: string;
    },
  ) =>
    request<Check>(`/staff/checks/${id}/lines/${lineId}`, {
      method: "DELETE",
      body: body ?? {},
      bearer: tillBearer(),
    }),
  /** Write "no onion" against a dish.
   *
   *  ⚠️ Refused (409) once the line has been fired: the paper at the pass
   *  cannot be edited, and a silent change would leave the screen and the
   *  kitchen disagreeing about the same dish. */
  tillCommentLine: (id: string, lineId: string, comment: string) =>
    request<Check>(`/staff/checks/${id}/lines/${lineId}`, {
      method: "PUT",
      body: { comment },
      bearer: tillBearer(),
    }),
  /** How many of this dish, on a line the kitchen has not seen yet.
   *
   *  ⚠️ **Only the quantity is sent.** The same endpoint writes the guest's
   *  note, and the server tells "no comment in this request" from "clear the
   *  comment" by the field being absent — so pressing "+" must not carry an
   *  empty one, or it wipes "piyozsiz". */
  tillLineQty: (id: string, lineId: string, qty: number) =>
    request<Check>(`/staff/checks/${id}/lines/${lineId}`, {
      method: "PUT",
      body: { qty },
      bearer: tillBearer(),
    }),
  /** Send to the pass what has not been sent: one course, or everything.
   *
   *  Separate from adding a dish on purpose: typing is not ordering. */
  tillFire: (id: string, course?: number) =>
    request<Check>(`/staff/checks/${id}/fire`, {
      method: "POST",
      // ⚠️ No body at all for "everything", which is what every screen sent
      // before courses existed and what a counter sends forever.
      ...(course === undefined ? {} : { body: { course } }),
      bearer: tillBearer(),
    }),
  /** Which guest pays for a line, and which course it goes out with.
   *
   *  ⚠️ Sent one field at a time, like the note and the quantity: the server
   *  reads an absent field as "leave it alone". */
  tillLineGuest: (id: string, lineId: string, guest: number) =>
    request<Check>(`/staff/checks/${id}/lines/${lineId}`, {
      method: "PUT",
      body: { guest },
      bearer: tillBearer(),
    }),
  tillLineCourse: (id: string, lineId: string, course: number) =>
    request<Check>(`/staff/checks/${id}/lines/${lineId}`, {
      method: "PUT",
      body: { course },
      bearer: tillBearer(),
    }),
  /** Hand over sales this till took while it had no network.
   *
   *  ⚠️ Idempotent by the id the till minted: a resend is the same dinner, not
   *  a second one. Each check gets its own answer, so one that cannot be
   *  accepted does not hold up the rest. */
  tillSyncChecks: (checks: unknown[]) =>
    request<{
      results: {
        clientId: string;
        id?: string;
        number?: string;
        error?: string;
        duplicate?: boolean;
      }[];
      serverTime: string;
    }>("/staff/checks/sync", {
      method: "POST",
      body: { checks },
      bearer: tillBearer(),
    }),
  /** The room this screen belongs to: its name, its currency and its plan.
   *
   *  ⚠️ **Not `getRestaurant()`.** The public profile answers "which branch is
   *  this *visitor* served from" — the site's default, or a cookie. A till
   *  belongs to a branch by its token, and on a company with two of them the
   *  counter was drawing the other room's floor plan: the right number of
   *  tables, the right shapes, the wrong building. */
  tillBranch: () =>
    request<{
      id: string;
      name: string;
      currency: string;
      booking: BookingSettings;
      /** The room's service rate. ⚠️ Needed on the device, not only on the
       *  server: a check opened during an outage has to charge what the same
       *  table would have been charged a minute earlier. */
      servicePercent?: number;
    }>("/staff/branch", { bearer: tillBearer(), cache: "no-store" }),
  /** One of a check's receipts, laid out by the server.
   *
   *  ⚠️ `precheck` also **records** that the table has been given its bill —
   *  the floor screen draws that, and "asked twenty minutes ago" is a different
   *  situation from "asked just now". */
  tillPrint: (id: string, kind: "kitchen" | "till" | "customer" | "precheck") =>
    request<{
      lines: string[];
      widthMM: number;
      /** The logo to draw above the text, when the template asks for one and
       *  the browser is the one printing. */
      logoUrl?: string;
      /** How many of the branch's own printers took it. ⚠️ Zero means the
       *  screen should open the browser's print dialog instead — which is how
       *  every restaurant's first evening goes. */
      queued: number;
      check: Check;
    }>(`/staff/checks/${id}/print`, {
      method: "POST",
      body: { kind },
      bearer: tillBearer(),
    }),
  /** Today's bookings still ahead, for the branch this screen belongs to. */
  tillReservations: () =>
    request<{ reservations: TillReservation[] }>("/staff/reservations", {
      bearer: tillBearer(),
      cache: "no-store",
    }),
  /** Move dishes onto another open check — a party that split, or joined. */
  tillMoveLines: (id: string, lineIds: string[], toCheckId: string) =>
    request<Check>(`/staff/checks/${id}/lines/move`, {
      method: "POST",
      body: { lineIds, toCheckId },
      bearer: tillBearer(),
    }),
  // Two bills for one table. ⚠️ Returns both halves: the source keeps the
  // screen (the waiter is still standing at that table), and the new one has to
  // appear in the room immediately or it reads as food that vanished.
  tillSplit: (id: string, lineIds: string[]) =>
    request<{ check: Check; split: Check }>(`/staff/checks/${id}/split`, {
      method: "POST",
      body: { lineIds },
      bearer: tillBearer(),
    }),
  // The X report: what this shift has sold and what should be in the drawer.
  // ⚠️ A GET, and it changes nothing — it can be pressed at four in the
  // afternoon by somebody with a suspicion, as often as they like.
  tillShiftReport: () =>
    request<{ lines: string[]; widthMM: number }>("/staff/cash-shift/report", {
      bearer: tillBearer(),
      cache: "no-store",
    }),
  // Two checks become one. ⚠️ The absorbed check is cancelled server-side, not
  // deleted: it carries voided lines, a number that may be on a printed bill,
  // and who opened it.
  tillMerge: (id: string, intoId: string) =>
    request<Check>(`/staff/checks/${id}/merge`, {
      method: "POST",
      body: { intoId },
      bearer: tillBearer(),
    }),
  // ---- Paying from the guest's own phone ----
  //
  // ⚠️ Asked of the server rather than listed on the screen: a button leading
  // to a bank page that rejects the merchant loses the sale, and the guest
  // blames the restaurant.
  tillPaymentMethods: () =>
    request<{ methods: TillPaymentMethod[] }>(`/staff/payment-methods`, {
      bearer: tillBearer(),
      cache: "no-store",
    }),
  /** Put the check in front of a provider and get the link to show as a QR. */
  tillStartPayment: (id: string, provider: string) =>
    request<{ url: string; provider: string; total: number; number: string }>(
      `/staff/checks/${id}/pay-online`,
      { method: "POST", body: { provider }, bearer: tillBearer() },
    ),
  /** Has the money arrived? ⚠️ The only evidence that closes the check. */
  tillPaymentStatus: (id: string) =>
    request<{ status: string; method: string; paid: boolean }>(
      `/staff/checks/${id}/payment`,
      { bearer: tillBearer(), cache: "no-store" },
    ),
  // ---- A debt settled at the counter ----
  //
  // ⚠️ Searched by phone and never listed: a screen in a dining room showing
  // every debtor is the customer base on display to whoever is standing there.
  tillDebts: (phone: string) =>
    request<{ name?: string; phone?: string; debts: TillDebt[]; total: number }>(
      `/staff/debts?phone=${encodeURIComponent(phone)}`,
      { bearer: tillBearer(), cache: "no-store" },
    ),
  /** Take the money. ⚠️ Dated today — tonight's drawer, not the day of the meal. */
  tillPayDebt: (orderId: string, method: string) =>
    request<{ ok: boolean }>(`/staff/debts/${orderId}/pay`, {
      method: "POST",
      body: { method },
      bearer: tillBearer(),
    }),
  tillClose: (
    id: string,
    body: {
      paymentMethod: TillPaymentMethod;
      discount?: number;
      discountReason?: string;
      /** Who owes it, when the method is `debt`. Required by the server in
       *  that case: "somebody will pay later" is the record the paper book by
       *  the till already keeps badly. */
      userId?: string;
      debtNote?: string;
      /** A code from somebody who may give discounts. */
      pin?: string;
    },
  ) =>
    request<Check>(`/staff/checks/${id}/close`, {
      method: "POST",
      body,
      bearer: tillBearer(),
    }),
  tillCancel: (id: string, reason: string) =>
    request<Check>(`/staff/checks/${id}/cancel`, {
      method: "POST",
      body: { reason },
      bearer: tillBearer(),
    }),

  // ---- Fiscalisation: this screen is the only one that can reach the register
  //
  // The registered cash register runs on a PC inside the restaurant with no
  // route from our server (see internal/fiscal). This tablet is on that same
  // network, so the work splits: the server builds the document, we carry it
  // one hop and bring the answer back. Every call below is one half of that,
  // and none of them composes anything.
  tillFiscalStatus: () =>
    request<TillFiscalStatus>("/staff/fiscal", {
      bearer: tillBearer(),
      cache: "no-store",
    }),
  /** Record the outcome of a connection check we ran against the register. */
  tillFiscalCheck: (body: FiscalReply) =>
    request<{ ok: boolean; message: string }>("/staff/fiscal", {
      method: "PUT",
      body,
      bearer: tillBearer(),
    }),
  /** Ask for the filing to make for a closed check.
   *
   *  Four possible answers, and the screen has to tell them apart: `job` is work
   *  for this browser to do, `queued` means a relay on the register's PC has it
   *  and we wait, `skip` means no register is connected and the sale simply is
   *  not filed, `filed` means it already was — which is what makes retrying
   *  after a lost reply safe rather than a second receipt. */
  tillFileReceipt: (id: string) =>
    request<{
      job?: FiscalJob;
      skip?: boolean;
      filed?: boolean;
      queued?: boolean;
      fiscal?: FiscalReceipt;
    }>(`/staff/checks/${id}/fiscal`, {
      method: "POST",
      bearer: tillBearer(),
    }),
  /** Report what the register said.
   *
   *  ⚠️ May answer with `openShift` instead of a check: the register refused
   *  because its day has not been started, which is what the first sale every
   *  morning runs into. Run that job and file again — the sale is still pending
   *  server-side, so nothing is lost and nothing is filed twice. */
  tillFileReceiptResult: (id: string, body: FiscalReply) =>
    request<Check & { openShift?: FiscalJob }>(`/staff/checks/${id}/fiscal`, {
      method: "PUT",
      body,
      bearer: tillBearer(),
    }),
  /** Sales that took money and have no tax receipt. The screen that makes a
   *  failed filing findable at all — without it the retry button promised on
   *  the payment dialog is reachable from nowhere. */
  tillUnfiledChecks: () =>
    request<{ checks: Check[] }>("/staff/fiscal/unfiled", {
      bearer: tillBearer(),
      cache: "no-store",
    }),
  /** End the register's tax day (the Z-report).
   *
   *  ⚠️ Refused with 409 while any sale is still unfiled: a receipt filed after
   *  the day is totalled belongs to the next day, and that cannot be undone.
   *  Answers `queued` when a relay will do it instead — filing it here as well
   *  would produce two Z-reports for one day. */
  // ---- The cash drawer, from the till ----
  //
  // ⚠️ The same arithmetic the panel uses — one implementation on the server.
  // Two answers about missing money is worse than none.
  tillCashShift: () =>
    request<{
      open: CashShift | null;
      figures?: CashFigures;
      entries?: CashEntry[];
      /** Whether this person may open or close it. Courtesy only: the server
       *  asks again, and asks a manager if the answer is no. */
      canShift: boolean;
    }>("/staff/cash-shift", { bearer: tillBearer(), cache: "no-store" }),
  tillOpenCashShift: (body: {
    openingFloat: number;
    note?: string;
    pin?: string;
  }) =>
    request<CashShift>("/staff/cash-shift/open", {
      method: "POST",
      body,
      bearer: tillBearer(),
    }),
  tillCloseCashShift: (body: {
    counted: number;
    varianceNote?: string;
    note?: string;
    pin?: string;
  }) =>
    request<{
      shift: CashShift;
      figures: CashFigures;
      fiscalNote?: string;
      // ⚠️ The Z report comes back with the close rather than from a second
      // button: a screen that shuts the drawer and then asks somebody to
      // remember to print produces evenings with no Z report at all.
      lines?: string[];
      widthMM?: number;
    }>("/staff/cash-shift/close", {
      method: "POST",
      body,
      bearer: tillBearer(),
    }),
  tillCloseFiscalDay: () =>
    request<{ job?: FiscalJob; queued?: boolean }>("/staff/fiscal/close-day", {
      method: "POST",
      bearer: tillBearer(),
    }),
  tillCloseFiscalDayResult: (body: FiscalReply) =>
    request<FiscalDay>("/staff/fiscal/close-day", {
      method: "PUT",
      body,
      bearer: tillBearer(),
    }),

  // ---- Branch kiosk screen ----
  kioskCode: () =>
    request<KioskCode>("/kiosk/code", {
      bearer: getKioskToken(),
      cache: "no-store",
    }),
  // Issue (or, with rotate, re-issue) the screen token for a branch.
  adminKioskToken: (branchId: string, rotate = false) =>
    request<KioskToken>(
      `/admin/branches/${branchId}/kiosk${rotate ? "?rotate=1" : ""}`,
      { method: "POST", auth: true },
    ),
  staffReport: (from?: string, to?: string) =>
    request<StaffReport>(`/staff/report${dateQuery(from, to)}`, {
      bearer: getStaffToken(),
      cache: "no-store",
    }),

  // ---- Staff (admin) ----
  adminStaff: (params?: { from?: string; to?: string; q?: string }) =>
    request<StaffRow[]>(
      `/admin/staff${dateQuery(params?.from, params?.to, params?.q)}`,
      { auth: true, scope: true, cache: "no-store" },
    ),
  adminStaffMember: (id: string, from?: string, to?: string) =>
    request<AdminStaffDetail>(`/admin/staff/${id}${dateQuery(from, to)}`, {
      auth: true,
      cache: "no-store",
    }),
  createStaff: (body: Partial<Staff> & { password: string }) =>
    request<Staff>("/admin/staff", {
      method: "POST",
      body,
      auth: true,
      scope: true,
    }),
  updateStaff: (id: string, body: Partial<Staff> & { password?: string }) =>
    request<Staff>(`/admin/staff/${id}`, { method: "PUT", body, auth: true }),
  deleteStaff: (id: string) =>
    request<{ ok: boolean }>(`/admin/staff/${id}`, {
      method: "DELETE",
      auth: true,
    }),

  // Attendance written by hand: a dead phone, a forgotten clock-out.
  createShift: (
    staffId: string,
    body: { date: string; in: string; out?: string; note?: string },
  ) =>
    request<Shift>(`/admin/staff/${staffId}/shifts`, {
      method: "POST",
      body,
      auth: true,
    }),
  updateShift: (
    staffId: string,
    shiftId: string,
    body: { date: string; in: string; out?: string; note?: string },
  ) =>
    request<{ ok: boolean }>(`/admin/staff/${staffId}/shifts/${shiftId}`, {
      method: "PUT",
      body,
      auth: true,
    }),
  deleteShift: (staffId: string, shiftId: string) =>
    request<{ ok: boolean }>(`/admin/staff/${staffId}/shifts/${shiftId}`, {
      method: "DELETE",
      auth: true,
    }),

  // ---- Payroll (hisob-kitob) ----
  adminPayroll: () =>
    request<PayrollResponse>("/admin/payroll", {
      auth: true,
      scope: true,
      cache: "no-store",
    }),
  payStaff: (
    staffId: string,
    body: { amount: number; from?: string; to?: string; note?: string },
  ) =>
    request<StaffPayment>(`/admin/staff/${staffId}/payments`, {
      method: "POST",
      body,
      auth: true,
    }),
  deleteStaffPayment: (staffId: string, paymentId: string) =>
    request<{ ok: boolean }>(`/admin/staff/${staffId}/payments/${paymentId}`, {
      method: "DELETE",
      auth: true,
    }),
};

// dateQuery builds the ?from=&to=&q= every attendance endpoint takes. Empty
// dates mean "this calendar month", which the server decides — the client must
// not guess it, or the two disagree at midnight on the first.
function dateQuery(from?: string, to?: string, q?: string): string {
  const qs = new URLSearchParams();
  if (from) qs.set("from", from);
  if (to) qs.set("to", to);
  if (q) qs.set("q", q);
  const s = qs.toString();
  return s ? `?${s}` : "";
}

// Image upload uses multipart/form-data, so it bypasses the JSON `request`.
export async function uploadImage(file: File): Promise<string> {
  const fd = new FormData();
  fd.append("file", file);
  const headers: Record<string, string> = {};
  const token = getToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${await apiBase()}/admin/upload`, {
    method: "POST",
    headers,
    body: fd,
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const data = (await res.json()) as { error?: string };
      message = data.error ?? message;
    } catch {
      /* keep statusText */
    }
    throw new ApiError(res.status, message);
  }
  const data = (await res.json()) as { url: string };
  return data.url;
}

export { request };
