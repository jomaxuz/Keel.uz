// Thin fetch wrapper around the Go backend REST API.
//
// Public reads can be called from the server (SSR) or the client. Admin calls
// require a JWT stored in localStorage under `TOKEN_KEY`.

import type {
  AdminAlerts,
  AdminCourierDetail,
  AdminStaffDetail,
  AdminLog,
  AbcXyzResponse,
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
  SegmentRow,
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
} from "./types";

// Server-side (SSR) calls the backend directly; client-side calls the same
// origin, which Next.js rewrites proxy to the backend (see next.config.ts). This
// keeps everything single-origin behind one domain — no CORS, no
// mixed-content.
export const API_URL =
  typeof window === "undefined"
    ? process.env.INTERNAL_API_URL ?? "http://localhost:8080/api/v1"
    : process.env.NEXT_PUBLIC_API_URL ?? "/api/v1";

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

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "ApiError";
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

export function setAdminScope(next: { brandId: string; branchId: string }): void {
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

async function request<T>(rawPath: string, opts: RequestOptions = {}): Promise<T> {
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
    try {
      const data = (await res.json()) as { error?: string; message?: string };
      message = data.error ?? data.message ?? message;
    } catch {
      // non-JSON error body — keep statusText
    }
    throw new ApiError(res.status, message);
  }

  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
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
  range?: { from?: string; to?: string },
): Promise<void> {
  const qs = new URLSearchParams({ format: "xlsx" });
  if (range?.from) qs.set("from", range.from);
  if (range?.to) qs.set("to", range.to);

  const token = getToken();
  const res = await fetch(`${await apiBase()}${withScope(`${path}?${qs}`)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new ApiError(res.status, res.statusText);

  const blob = await res.blob();
  const name =
    /filename="([^"]+)"/.exec(res.headers.get("Content-Disposition") ?? "")?.[1] ??
    "hisobot.xlsx";

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
export async function downloadDataArchive(): Promise<void> {
  const token = getToken();
  const res = await fetch(`${await apiBase()}/admin/export/archive`, {
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
    /filename="([^"]+)"/.exec(res.headers.get("Content-Disposition") ?? "")?.[1] ??
    "malumotlar.zip";
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
export function imageUrl(path: string | null | undefined): string | null {
  if (!path) return null;
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  if (path.startsWith("/uploads/")) {
    return `${UPLOADS_URL}${path.slice("/uploads".length)}`;
  }
  return `${UPLOADS_URL}/${path.replace(/^\/+/, "")}`;
}

/** The site's lens: which brand's menu, and which branch serves it. */
export interface SiteScope {
  /** Brand id or slug. */
  brand?: string;
  branchId?: string;
  /** Admin only: the company document as stored, with no branch laid over it. */
  raw?: boolean;
}

function siteQuery(scope?: SiteScope): string {
  const qs = new URLSearchParams();
  if (scope?.brand) qs.set("brand", scope.brand);
  if (scope?.branchId) qs.set("branchId", scope.branchId);
  if (scope?.raw) qs.set("raw", "1");
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
      revalidate: 10,
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

  userMe: () =>
    request<SiteUser>("/users/me", { bearer: getUserToken() }),
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
    request<Reservation[]>("/users/me/reservations", { bearer: getUserToken() }),

  // ---- Brands and branches ----
  getBrands: () => request<BrandsResponse>("/brands", { revalidate: 10 }),

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
  ) => request<SiteUser>(`/admin/users/${id}`, { method: "PUT", body, auth: true }),

  // Tags already in use, so the panel offers them instead of letting a typo
  // create a second "VIP " group.
  adminTags: () => request<string[]>("/admin/tags", { auth: true }),

  // Orders.
  adminOrders: (params?: {
    status?: OrderStatus;
    q?: string;
    userId?: string;
    limit?: number;
  }) => {
    const qs = new URLSearchParams();
    if (params?.status) qs.set("status", params.status);
    if (params?.q) qs.set("q", params.q);
    if (params?.userId) qs.set("userId", params.userId);
    if (params?.limit) qs.set("limit", String(params.limit));
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<Order[]>(`/admin/orders${suffix}`, {
      auth: true,
      cache: "no-store",
      scope: true,
    });
  },
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
  adminCouriers: () => request<Courier[]>("/admin/couriers", { auth: true, scope: true }),
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
    request<{ domain: string; found: string[]; expected: string[]; ok: boolean }>(
      `/admin/domain-check?domain=${encodeURIComponent(domain)}`,
      { auth: true, cache: "no-store" },
    ),

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
    const qs = new URLSearchParams();
    if (range?.from) qs.set("from", range.from);
    if (range?.to) qs.set("to", range.to);
    const suffix = qs.toString() ? `?${qs}` : "";
    return request<AbcXyzResponse>(`/admin/reports/abc-xyz${suffix}`, {
      auth: true,
      scope: true,
    });
  },

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
  testSMS: (phone?: string) =>
    request<{ ok: boolean; message: string; phone: string; provider: string }>(
      "/admin/sms/test",
      { method: "POST", body: { phone: phone ?? "" }, auth: true },
    ),

  // Every attempt against one order, successful or not.
  adminOrderPayments: (id: string) =>
    request<Payment[]>(`/admin/orders/${id}/payments`, {
      auth: true,
      cache: "no-store",
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
    items: { menuItemId: string; posProductId: string; posProductName: string }[],
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
  campaignPreview: (segment: string, text: string) =>
    request<CampaignPreview>("/admin/campaigns/preview", {
      method: "POST",
      body: { segment, text },
      auth: true,
    }),
  sendCampaign: (segment: string, text: string) =>
    request<{ id: string; recipients: number; status: string }>(
      "/admin/campaigns",
      { method: "POST", body: { segment, text }, auth: true },
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
    request<{ url: string; message?: string }>(
      `/admin/calls/${id}/recording`,
      { auth: true, cache: "no-store" },
    ),
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
  updateRestaurant: (body: Restaurant) =>
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
    request<AdminStaffDetail>(
      `/admin/staff/${id}${dateQuery(from, to)}`,
      { auth: true, cache: "no-store" },
    ),
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
    request<{ ok: boolean }>(
      `/admin/staff/${staffId}/payments/${paymentId}`,
      { method: "DELETE", auth: true },
    ),
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
