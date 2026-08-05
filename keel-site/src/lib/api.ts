// Client for the control plane.
//
// Same-origin (`/api/v1`): in production Caddy puts keel.uz's site and the
// control API behind one host, and in development next.config rewrites do the
// same. Nothing here ever talks to a tenant — a page that could reach one
// customer's database from the browser is a page that could reach all of them.

export const API = "/api/v1";
const TOKEN_KEY = "keel_token";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

export class Unauthorized extends Error {}

async function req<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Content-Type", "application/json");
  const token = getToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(`${API}${path}`, { ...init, headers });
  if (res.status === 401) {
    clearToken();
    throw new Unauthorized("sessiya tugagan");
  }
  const text = await res.text();
  const body = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error(body?.error ?? res.statusText);
  return body as T;
}

// ---- Types ----

export type TenantStatus = "active" | "trial" | "suspended" | "deleted";

export interface Tenant {
  id: string;
  slug: string;
  name: string;
  kind: string;
  domains: string[];
  status: TenantStatus;
  /** A raw timestamp, and it marshals as UTC — never slice a day out of it.
   *  The trial's end as a local day is `period.to`. */
  trialEndsAt?: string;
  /** The day the customer started paying — the anchor every invoice is counted
   *  from. Absent while a tenant is still on trial. */
  subscribedAt?: string;
  /** When the trial sweep switched this tenant off by itself — the answer to
   *  "why did this site go dark", weeks after the server log has rotated. */
  autoSuspendedAt?: string;
  pricePerOrder: number;
  hideWatermark: boolean;
  ownerName: string;
  ownerPhone: string;
  /** The first account of the tenant's own panel. The password is never
   *  returned — only whether one is stored. */
  adminUsername: string;
  hasAdminPassword: boolean;
  /** What happened last time we tried to bring the tenant up. */
  provisionStatus: "" | "ready" | "failed";
  provisionError: string;
  provisionedAt?: string;
  /** Docker's live answer, absent when this deployment cannot start anything. */
  containerStatus?: "running" | "stopped" | "absent" | "unknown";
  note: string;
  createdAt: string;
  updatedAt: string;
}

export interface Totals {
  orders: number;
  revenue: number;
  billable: number;
}

/** One tenant's current window, decided on the server.
 *
 *  `to` is exclusive, which makes it exactly the date of the next invoice —
 *  the one thing an operator on the phone is actually asked. */
export interface Period {
  from: string;
  to: string;
  kind: "subscription" | "trial";
  /** The recorded subscription day, "YYYY-MM-DD", empty when there is none.
   *
   *  Used instead of slicing `tenant.subscribedAt`: that field marshals as UTC,
   *  so its first ten characters are the previous day here. */
  anchor: string;
  /** False when no subscription date was recorded and the cycle is counted
   *  from the day the tenant was opened. */
  anchored: boolean;
}

export type AttentionKind = "" | "trial_ending" | "trial_expired" | "suspended";

/** What this customer needs from a human, computed on every request rather
 *  than stored — a saved flag goes stale the moment the clock passes it. */
export interface Attention {
  kind: AttentionKind;
  /** Whole days left (ending) or days past the deadline (expired), counted on
   *  the server so the browser never does date maths on a UTC timestamp. */
  days: number;
}

export interface TenantRow {
  tenant: Tenant;
  period: Period;
  attention: Attention;
  /** The current billing period — not the calendar month. */
  orders: number;
  revenue: number;
  billable: number;
  lifetime: Totals;
}

export interface TenantDay {
  date: string;
  orders: number;
  revenue: number;
  billable: number;
}

export interface TenantDetail {
  tenant: Tenant;
  days: TenantDay[];
  period: Period;
  attention: Attention;
  totals: Totals;
  lifetime: Totals;
}

export interface Stats {
  tenants: {
    total: number;
    active: number;
    trial: number;
    suspended: number;
    deleted: number;
    watermarkRemoved: number;
  };
  /** How many customers need a phone call, by reason. Counted beside the plain
   *  status tally because they answer different questions: "trial" is how many
   *  are evaluating, "trialExpired" is how many of those ran out and are still
   *  running for free. */
  attention: { trialEnding: number; trialExpired: number; unpaid: number };
  month: { orders: number; revenue: number; billable: number };
  series: TenantDay[];
  top: { id: string; name: string; slug: string; orders: number; billable: number }[];
}

// ---- Calls ----

export async function login(username: string, password: string) {
  const out = await req<{ token: string; user: { username: string } }>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  setToken(out.token);
  return out.user;
}

export const me = () => req<{ username: string }>("/me");
export const stats = () => req<Stats>("/stats");

export function tenants(
  params: { q?: string; status?: string; attention?: string } = {},
) {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.status) qs.set("status", params.status);
  if (params.attention) qs.set("attention", params.attention);
  const s = qs.toString();
  return req<{ items: TenantRow[] }>(`/tenants${s ? `?${s}` : ""}`);
}

export const tenant = (id: string) => req<TenantDetail>(`/tenants/${id}`);

export const createTenant = (body: Record<string, unknown>) =>
  req<Tenant>("/tenants", { method: "POST", body: JSON.stringify(body) });

export const provisionTenant = (id: string) =>
  req<TenantDetail>(`/tenants/${id}/provision`, { method: "POST" });

export const updateTenant = (id: string, body: Record<string, unknown>) =>
  req<TenantDetail>(`/tenants/${id}`, { method: "PUT", body: JSON.stringify(body) });

/** Whole so'm, grouped. Intl is avoided for the same reason as in the tenant
 *  app: it follows the device locale and would print a different separator on
 *  the server than in the browser. */
export function money(n: number): string {
  const s = Math.round(n).toString();
  let out = "";
  for (let i = 0; i < s.length; i++) {
    if (i > 0 && (s.length - i) % 3 === 0) out += " ";
    out += s[i];
  }
  return out;
}

/** "2026-08-17" → "17.08.2026".
 *
 *  Split rather than parsed: `new Date("2026-08-17")` is UTC midnight, which
 *  prints the previous day in any negative offset — the same class of mistake
 *  as reading a day boundary in UTC on the server, and it would show a billing
 *  period starting one day before it does. */
export function dayLabel(d: string): string {
  const [y, m, dd] = (d ?? "").split("-");
  return y && m && dd ? `${dd}.${m}.${y}` : d;
}

export function shortDate(iso: string): string {
  const d = new Date(iso);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()}`;
}
