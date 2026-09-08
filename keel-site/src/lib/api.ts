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
  // ⚠️ **Not for a file.** A multipart body carries a boundary the browser
  // generates, and a `Content-Type` written here names a different one — the
  // server then cannot parse the request, and the error reads as "the file is
  // broken" rather than "we described it wrongly".
  if (!(init.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }
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
  /** Which template this tenant's first brand was created from.
   *
   *  ⚠️ Not the same field as `kind`: that one is free text a person reads,
   *  this one is what the software did. See the note on the server's model. */
  businessType?: string;
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
  /** When this customer's data was erased for good, by whom and why. Present
   *  only on a purged tenant — `status: "deleted"` on its own is still the
   *  reversible kind, which is the distinction the buttons rest on. */
  purgedAt?: string;
  purgedBy?: string;
  purgeReason?: string;
  pricePerOrder: number;
  /** The least this customer is billed for a period they actually used, in
   *  so'm. 0 means "use the platform's default", which is itself 0 unless
   *  configured — a floor never appears on an invoice by accident. */
  minMonthly?: number;
  hideWatermark: boolean;
  /** Show this customer's logo on keel.uz. Opt-in, off by default: their
   *  brand on our marketing page is their decision. */
  showcase: boolean;
  /** This customer pays nothing. Deliberately not the same as
   *  `pricePerOrder: 0`, which is indistinguishable from a cleared field. */
  free: boolean;
  freeReason?: string;
  /** Absent means forever — a real answer for an anchor customer, not an
   *  oversight. */
  freeUntil?: string;
  /** A standing discount, 0–100. The middle ground between paying and free. */
  discountPercent: number;
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
  /** Which console account signed this customer up. Shown to the roles that see
   *  every customer — an agent's own list needs no such column. */
  createdBy?: string;
  createdByRole?: string;
}

export interface Totals {
  orders: number;
  revenue: number;
  /** What the customer is charged. For a period this is the volume ladder over
   *  the period's order count, not the sum of the daily estimates. */
  billable: number;
  /** Our fee as a percentage of what the restaurant took.
   *
   *  The single number that predicts whether a customer starts negotiating:
   *  under ~2% nobody counts, over 3% they do. Neither half means anything
   *  alone — 12 mln so'm is either 1% or 20% of a business. */
  share: number;

  /** The counter, over the same window. ⚠️ Never added into `revenue`: the two
   *  are settled differently (per order vs monthly subscription) and `share` —
   *  the figure that predicts a negotiation — is deliberately computed against
   *  the half we charge for. */
  tillChecks: number;
  tillGuests: number;
  tillRevenue: number;
  tillRefunded: number;
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

export type AttentionKind =
  | ""
  | "trial_ending"
  | "trial_expired"
  | "suspended"
  /** A billing period closed and nothing in the ledger covers it. The one
   *  warning about us rather than about the customer. */
  | "invoice_due"
  /** The customer's site is not running and nobody meant that. Read live from
   *  Docker, not from the stored `provisionStatus` — which is a flag, and a
   *  flag goes stale the moment a container dies. */
  | "down";

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
  /** Our fee as a percentage of the restaurant's takings for the period. */
  share: number;
  lifetime: Totals;
}

export interface TenantDay {
  date: string;
  orders: number;
  /** Site visitors and page views that day. Absent on rows written before
   *  traffic was collected, which reads as 0. */
  visitors?: number;
  views?: number;
  /** Counted but never billed. Absent on rows written before the field
   *  existed, which reads as 0 — nobody can recover what was not counted. */
  cancelled?: number;
  /** ⚠️ Delivered, then marked cancelled — **billed anyway**.
   *
   *  Our fee is per order and cancellations are free, which is right for an
   *  order stopped before cooking and an open invitation if the word can be
   *  applied afterwards. Billing no longer reads the current status alone, so
   *  this costs nothing; it is shown because one is a guest refusing at the
   *  door and twenty a night is a conversation. */
  reversed?: number;
  /** Cancelled after the kitchen had it, never delivered. Not billed — nothing
   *  proves the food left the building, and a rule that guessed would accuse a
   *  restaurant over a guest who was not home. */
  cancelledCooked?: number;
  revenue: number;
  billable: number;

  // ---- The dining room ----
  //
  // ⚠️ **Counted, never billed.** The counter is sold as a monthly
  // subscription, so a per-order fee on top of it would charge the same sale
  // twice — the aggregate excludes till checks from `orders` and `revenue`
  // deliberately. These arrive beside them because "not billed" had silently
  // become "not shown": a restaurant doing its whole trade over the counter
  // appeared here as a customer with no sales at all.
  //
  // ⚠️ Keyed by the day the check **closed**, not the day the table opened. A
  // table seated at 23:40 pays on the next date, and the money has to land on
  // the day the drawer holds it.
  //
  // Absent on rows written before these existed, which reads as 0.
  tillChecks?: number;
  tillGuests?: number;
  /** So'm taken at the counter — paid checks only. A check handed over on
   *  credit closes as delivered and *unpaid*, and counting it would book money
   *  nobody has. */
  tillRevenue?: number;
  /** Refunded back out afterwards. Its own figure rather than netted off:
   *  nine million sold with two refunded is a different day from seven million
   *  sold, and only one of them is worth a call. */
  tillRefunded?: number;
}

// ---- One customer's live numbers ----
//
// Read straight from that customer's own database when their card is opened.
// The overview never asks for this: a list that dialled every tenant gets
// slower with every customer sold, which is what the nightly aggregate exists
// to avoid. One card, one database, opened by a human on purpose.

export interface DayFigures {
  orders: number;
  cancelled: number;
  /** Money actually in hand: cash collected on delivery, or a card payment the
   *  bank confirmed. Not every order placed. */
  revenue: number;
  /** Placed, not cancelled, not yet collected. */
  pending: number;
  /** How many orders the revenue came from. */
  paid: number;
  avgOrder: number;
  delivery: number;
  pickup: number;
  dinein: number;
}

export interface TenantLive {
  today: DayFigures;
  yesterday: DayFigures;
  /** What is in the kitchen or on the road right now. */
  active: {
    pending: number;
    confirmed: number;
    preparing: number;
    onTheWay: number;
    total: number;
  };
  people: { customers: number; new30d: number; active30d: number };
  staff: { total: number; active: number; onShift: number };
  couriers: {
    total: number;
    active: number;
    free: number;
    busy: number;
    off: number;
  };
  menu: {
    items: number;
    available: number;
    categories: number;
    branches: number;
    brands: number;
  };
  reservations: { today: number; upcoming: number };
  /** How many people opened the site, as opposed to how many ordered. */
  traffic: {
    todayVisitors: number;
    todayViews: number;
    yesterdayVisitors: number;
    /** Visitor-days over 30 days, not distinct people: the row key is hashed
     *  with the date so nobody can be followed across days. Slightly high on
     *  purpose. */
    visitors30d: number;
  };
  topItems: { name: string; qty: number }[];
  collectedAt: string;
  /** Set when the tenant database could not be read, or the customer is
   *  stopped. Without it the zeroes read as calm rather than as ignorance. */
  error?: string;
}

export const tenantLive = (id: string) =>
  req<TenantLive>(`/tenants/${id}/live`);

// ---- The restaurant's own Android app ----
//
// ⚠️ **The artifact is deleted the moment it has been downloaded**, and the row
// stays. Two and a half megabytes per build on the machine that serves every
// customer adds up to a directory nobody prunes; but "which version is on the
// store", asked months later, is a question a filesystem cannot answer. So the
// screen shows a history whose files are mostly gone, and says so.

export interface AppBuild {
  id: string;
  slug: string;
  /** "apk" — installs on a phone. "aab" — what Play accepts, and cannot be
   *  installed at all. ⚠️ Asked before the build, because guessing wrong costs
   *  nine minutes and is found out at the end of an upload. */
  format: "apk" | "aab";
  /** queued · building · ready · taken · failed */
  status: string;
  applicationId?: string;
  versionCode?: number;
  versionName?: string;
  size?: number;
  /** ⚠️ Kept after the file is gone: the only way to answer "is the APK on my
   *  laptop the one you built me". */
  sha256?: string;
  error?: string;
  by?: string;
  downloadedBy?: string;
  downloadedAt?: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export const appBuilds = (tenantId: string) =>
  req<{ builds: AppBuild[] }>(`/tenants/${tenantId}/app-builds`);

export const startAppBuild = (tenantId: string, format: "apk" | "aab") =>
  req<AppBuild>(`/tenants/${tenantId}/app-build`, {
    method: "POST",
    body: JSON.stringify({ format }),
  });

/** Fetch the artifact and hand it to the browser.
 *
 *  ⚠️ **Not an `<a href>`.** The endpoint needs the console's bearer token, and
 *  a plain link sends none — which arrives as "sessiya tugagan" on a button that
 *  looks like an ordinary download. So the file comes through `fetch` and is
 *  handed over as a blob.
 *
 *  ⚠️ **The server deletes its copy once the transfer completes**, so a failure
 *  here has to be visible: a silent catch would leave somebody believing they
 *  have a file they do not. */
export async function downloadAppBuild(build: AppBuild): Promise<void> {
  const token = getToken();
  const res = await fetch(`${API}/app-builds/${build.id}/download`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(
      (text ? (JSON.parse(text) as { error?: string }).error : "") ??
        res.statusText,
    );
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${build.slug}-${build.versionName ?? "1.0.0"}.${build.format}`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  // ⚠️ Revoked, or the blob stays in memory for the life of the tab — and these
  // are megabytes, on a page somebody leaves open all day.
  URL.revokeObjectURL(url);
}

// ---- The server everything runs on ----

export interface HostStats {
  cpuPercent: number;
  cores: number;
  load1: number;
  load5: number;
  load15: number;
  memTotal: number;
  memAvailable: number;
  memPercent: number;
  swapTotal: number;
  swapUsed: number;
  diskTotal: number;
  diskFree: number;
  diskPercent: number;
  /** Which filesystem the disk figures describe — a wrong mount should be
   *  visible rather than silently reporting the container's own overlay. */
  diskPath: string;
  uptimeSeconds: number;
  at: string;
  /** What could not be read. A zero that means "unknown" reads as "empty",
   *  and on a disk gauge those are opposite emergencies. */
  errors?: string[];
}

export interface DockerUsage {
  images: number;
  containers: number;
  volumes: number;
  buildCache: number;
  /** What `docker system prune` would free. */
  reclaimable: number;
}

/** Last night's backup, read from the manifest on disk rather than from a
 *  stored flag — see sysstat.Backup for why that distinction is the feature. */
export interface BackupStatus {
  present: boolean;
  date?: string;
  finishedAt?: string;
  ageHours: number;
  /** Whether a night was missed. Decided by the server so the card and the
   *  overview's alarm cannot drift apart — a threshold written twice is a
   *  threshold that disagrees with itself after the first edit. */
  stale: boolean;
  files: number;
  bytes: number;
  /** Dumps the run itself reported as failed: a copy missing three restaurants
   *  is not the same as a copy. */
  failures: number;
}

// ---- Console accounts, the activity log and agent visits ----

export interface StaffRow {
  id: string;
  username: string;
  name: string;
  phone?: string;
  role: string;
  isActive?: boolean;
  createdBy?: string;
  createdAt: string;
  lastLoginAt?: string;
  /** How many customers this account brought in. */
  tenants: number;
}

export interface ConsoleLogRow {
  id: string;
  actor: string;
  actorRole?: string;
  action: string;
  target?: string;
  detail?: string;
  at: string;
}

export interface VisitRow {
  id: string;
  agentId: string;
  agentName: string;
  place: string;
  address?: string;
  phone?: string;
  plannedFor: string;
  status: "planned" | "done";
  outcome?: "positive" | "negative" | "callback";
  comment?: string;
  nextAt?: string;
  visitedAt?: string;
}

export const staffList = () =>
  req<{ items: StaffRow[]; roles: string[] }>("/staff");

export const createStaff = (body: {
  username: string;
  password: string;
  name: string;
  phone?: string;
  role: string;
}) => req<StaffRow>("/staff", { method: "POST", body: JSON.stringify(body) });

export const updateStaff = (
  id: string,
  body: {
    name?: string;
    phone?: string;
    role?: string;
    password?: string;
    isActive?: boolean;
  },
) =>
  req<{ ok: boolean }>(`/staff/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

export const consoleLog = () => req<{ items: ConsoleLogRow[] }>("/console-log");

export const visitList = (params?: {
  status?: string;
  q?: string;
  agentId?: string;
}) => {
  const qs = new URLSearchParams();
  if (params?.status) qs.set("status", params.status);
  if (params?.q) qs.set("q", params.q);
  if (params?.agentId) qs.set("agentId", params.agentId);
  const suffix = qs.toString() ? `?${qs}` : "";
  return req<{
    items: VisitRow[];
    summary: { planned: number; positive: number; negative: number };
    canSeeAll: boolean;
  }>(`/visits${suffix}`);
};

// ---- Support ----

export type SupportFrom = "owner" | "operator" | "assistant";

export type SupportThreadRow = {
  id: string;
  slug: string;
  restaurant: string;
  subject: string;
  status: "waiting" | "open" | "closed";
  askedBy: string;
  askedRole?: string;
  operatorName?: string;
  unreadForUs: number;
  lastText: string;
  lastFrom: SupportFrom;
  lastAt: string;
  createdAt: string;
};

export type SupportMessageRow = {
  id: string;
  threadId: string;
  from: SupportFrom;
  author: string;
  text: string;
  at: string;
};

/** What an operator needs to know about who is asking. ⚠️ A named subset of the
 *  tenant record, not the record: this screen is for answering a question, and
 *  the provisioning detail on that document is not part of one. */
export type SupportTenantCard = {
  slug: string;
  name: string;
  kind?: string;
  owner?: string;
  phone?: string;
  domains?: string[];
  status?: string;
  container?: string;
  free?: boolean;
  till?: string;
  createdAt?: string;
  subscribedAt?: string;
};

export const supportList = (params?: {
  status?: string;
  q?: string;
  slug?: string;
}) => {
  const qs = new URLSearchParams();
  if (params?.status) qs.set("status", params.status);
  if (params?.q) qs.set("q", params.q);
  if (params?.slug) qs.set("slug", params.slug);
  const suffix = qs.toString() ? `?${qs}` : "";
  return req<{ threads: SupportThreadRow[]; waiting: number }>(
    `/support${suffix}`,
  );
};

export const supportThread = (id: string) =>
  req<{
    thread: SupportThreadRow;
    messages: SupportMessageRow[];
    tenant: SupportTenantCard | null;
  }>(`/support/${id}`);

export const supportReply = (
  id: string,
  body: { text?: string; close?: boolean },
) =>
  req<{ ok: boolean }>(`/support/${id}/reply`, {
    method: "POST",
    body: JSON.stringify(body),
  });

export const createVisit = (body: Record<string, unknown>) =>
  req<VisitRow>("/visits", { method: "POST", body: JSON.stringify(body) });

export const updateVisit = (id: string, body: Record<string, unknown>) =>
  req<VisitRow>(`/visits/${id}`, { method: "PUT", body: JSON.stringify(body) });

export const deleteVisit = (id: string) =>
  req<{ ok: boolean }>(`/visits/${id}`, { method: "DELETE" });

export const systemStats = () =>
  req<{ host: HostStats; docker?: DockerUsage; backup?: BackupStatus }>(
    "/system",
  );

/** Frees what the dashboard reports as reclaimable.
 *
 *  ⚠️ Orphaned images, stopped containers and build cache — never volumes, and never
 *  `prune -a`, which would delete the previous tenant image a rollback needs. Returns
 *  what was actually freed, because that number is the reason somebody pressed it. */
export const pruneDocker = () =>
  req<{ freed: number; warning?: string; docker?: DockerUsage }>(
    "/system/prune",
    { method: "POST" },
  );

/** Bytes as a person reads them. */
export function bytes(n: number): string {
  const u = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  let v = n;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v < 10 && i > 0 ? v.toFixed(1) : Math.round(v)}\u00a0${u[i]}`;
}

export interface TenantDetail {
  tenant: Tenant;
  days: TenantDay[];
  period: Period;
  attention: Attention;
  totals: Totals;
  lifetime: Totals;
}

/** What the nightly collector did last time it ran.
 *
 *  Exists because an empty chart had two indistinguishable causes — nobody has
 *  ordered yet, or the collector has never successfully run — and the screen
 *  gave no way to tell which. */
export interface CollectorRun {
  at: string;
  durationMs: number;
  tenants: number;
  ok: number;
  failed: number;
  /** Day-rows written. Zero with tenants > 0 is a platform where nobody
   *  ordered in the window — a fact, not a fault. */
  rows: number;
  errors?: string[];
  trigger: "schedule" | "manual";
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
  attention: {
    trialEnding: number;
    trialExpired: number;
    unpaid: number;
    /** Periods that closed without an invoice. Counted here too because this
     *  is the screen somebody opens when they are *not* already thinking
     *  about invoices. */
    invoiceDue: number;
    /** Sites that are dark without anybody deciding they should be. */
    down: number;
  };
  month: { orders: number; revenue: number; billable: number };
  series: TenantDay[];
  top: {
    id: string;
    name: string;
    slug: string;
    orders: number;
    billable: number;
  }[];
  /** Null when the collector has never completed once — itself the answer. */
  collector: CollectorRun | null;
  /** Last night's backup, so the overview can raise the alarm without a second
   *  request. The same reading the server card shows — one manifest, one
   *  answer; the two must never be able to disagree about whether copies are
   *  still being taken. */
  backup?: BackupStatus;
}

// ---- Invoice ledger ----

/** How the money reached us. Cash until the MChJ is registered — there is no
 *  legal entity yet, so no bank account to receive a transfer into. */
export type PayMethod = "cash" | "transfer";

export interface InvoicePayment {
  amount: number;
  method: PayMethod;
  /** The Keel person who took it. Unsigned cash is how a ledger stops being
   *  evidence. */
  receivedBy: string;
  at: string;
  note?: string;
}

export interface Invoice {
  id: string;
  tenantId: string;
  slug: string;
  name: string;
  /** What somebody reads out on the phone: "KEEL-2026-08-0007". */
  number: string;
  /** Half-open [from, to), like every other period in this system. */
  from: string;
  to: string;
  orders: number;
  revenue: number;
  /** Frozen when the invoice was issued: the daily rows keep accruing, the
   *  number the customer was told does not. */
  amount: number;
  /** The watermark add-on's share of the amount, when the restaurant bought it.
   *  ⚠️ It is already inside `amount`; this is here so a bill three million larger can say
   *  why rather than being queried on the phone. */
  watermarkFee?: number;
  status: "open" | "paid" | "void";
  voidReason?: string;
  paid: InvoicePayment[];
  collected: number;
  outstanding: number;
  issuedBy: string;
  note?: string;
  createdAt: string;
}

export function invoices(params: { tenantId?: string; status?: string } = {}) {
  const qs = new URLSearchParams();
  if (params.tenantId) qs.set("tenantId", params.tenantId);
  if (params.status) qs.set("status", params.status);
  const s = qs.toString();
  return req<{ items: Invoice[]; outstanding: number }>(
    `/invoices${s ? `?${s}` : ""}`,
  );
}

/** Bills the tenant's last **closed** period. Issuing twice for one period
 *  returns the existing invoice rather than creating a second debt. */
export const issueInvoice = (
  tenantId: string,
  body: Record<string, unknown> = {},
) =>
  req<Invoice>(`/tenants/${tenantId}/invoices`, {
    method: "POST",
    body: JSON.stringify(body),
  });

export const payInvoice = (id: string, body: Record<string, unknown>) =>
  req<Invoice>(`/invoices/${id}/pay`, {
    method: "POST",
    body: JSON.stringify(body),
  });

export const voidInvoice = (id: string, reason: string) =>
  req<Invoice>(`/invoices/${id}/void`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });

// ---- Rolling update ----

/** What happened to one tenant during a rollout. */
export interface RolloutItem {
  slug: string;
  name: string;
  status: "updated" | "current" | "skipped" | "failed";
  note?: string;
  at: string;
}

export interface Rollout {
  startedAt: string;
  finishedAt?: string;
  startedBy: string;
  image: string;
  imageId: string;
  status: "running" | "done" | "aborted" | "failed";
  note?: string;
  /** The tenant being worked on right now — a stuck rollout should name the
   *  customer holding it up, not show a percentage. */
  current?: string;
  total: number;
  done: number;
  updated: number;
  failed: number;
  items: RolloutItem[];
}

export interface RolloutState {
  /** False where this deployment cannot start containers at all. */
  enabled: boolean;
  image?: string;
  imageId?: string;
  /** Live counts from container image ids, not from the last rollout's result:
   *  a container recreated by hand, or a tenant created after the rollout
   *  finished, is covered by no record of what we did. */
  current?: number;
  offline?: number;
  stale?: string[];
  staleCount?: number;
  /** A rollout is in flight in the server process right now. Distinct from
   *  `last.status === "running"`, which survives a restart that killed it. */
  running?: boolean;
  error?: string;
  last?: Rollout;
}

export const rollout = () => req<RolloutState>("/rollout");

export const startRollout = () => req<Rollout>("/rollout", { method: "POST" });

// ---- Calls ----

export async function login(username: string, password: string) {
  const out = await req<{ token: string; user: { username: string } }>(
    "/auth/login",
    {
      method: "POST",
      body: JSON.stringify({ username, password }),
    },
  );
  setToken(out.token);
  return out.user;
}

/** Who is signed in, and what they may do.
 *
 *  ⚠️ Permissions arrive already decided rather than as a role name for the browser to
 *  interpret: the rule lives in one place on the server (models.CanSee…) and the two
 *  sides cannot drift. The console hides what a role cannot use — the server refuses it
 *  either way, but a page of buttons that all answer "no permission" teaches somebody
 *  their tool is broken. */
export interface Me {
  username: string;
  name?: string;
  role: string;
  can: {
    allTenants: boolean;
    stats: boolean;
    staff: boolean;
    log: boolean;
    provision: boolean;
    billing: boolean;
  };
}

export const me = () => req<Me>("/me");

/** Runs the aggregate now and answers with the run's own report. The button an
 *  operator wants while looking at an empty chart. */
export const collectNow = () =>
  req<{ collector: CollectorRun | null; error?: string }>("/stats/collect", {
    method: "POST",
  });
export const stats = () => req<Stats>("/stats");

// ---- The platform over a window somebody chooses ----
//
// Separate from `stats` on purpose: that one answers the billing month, which
// has a start nobody picks. This answers "is this growing", which needs a
// yesterday, a week and a year beside each other.

/** One bucket. ⚠️ A day, a week or a month — `bucket` on the response says
 *  which, and the axis must be labelled from that rather than guessed from the
 *  row count. */
export interface OverviewPoint {
  date: string;
  visitors: number;
  views: number;
  orders: number;
  cancelled: number;
  revenue: number;
  billable: number;
  tillChecks: number;
  tillGuests: number;
  tillRevenue: number;
  tillRefunded: number;
}

export interface Overview {
  from: string;
  to: string;
  bucket: "day" | "week" | "month";
  series: OverviewPoint[];
  /** Summed over the window. `date` is empty — it is not a point. */
  total: OverviewPoint;
  /** Customers that traded at all in the window, counted **distinct**: the
   *  denominator without which "8 000 orders" cannot be read. */
  activeTenants: number;
  collector: CollectorRun | null;
}

/** Named windows the console draws as buttons. `custom` is the two date
 *  inputs, and it wins over the shorthand on the server. */
export type OverviewRange = "1d" | "7d" | "30d" | "90d" | "1y" | "custom";

export const overview = (q: {
  range?: OverviewRange;
  from?: string;
  to?: string;
}) => {
  const p = new URLSearchParams();
  // ⚠️ Only ever one of the two shapes on the wire. Sending both would rely on
  // the server's precedence rule staying what it is today, and the failure mode
  // is silent: a screen that answers a different question than its own buttons.
  if (q.from && q.to) {
    p.set("from", q.from);
    p.set("to", q.to);
  } else {
    p.set("range", q.range && q.range !== "custom" ? q.range : "30d");
  }
  return req<Overview>(`/overview?${p.toString()}`);
};

/** One kind of business, totalled. Mirrors handlers/business.go. */
export type BizRow = {
  /** "" is a restaurant, the same convention the business type itself uses. */
  type: string;
  tenants: number;
  online: number;
  /** ⚠️ False when Docker itself could not be reached: nothing is known about
   *  anybody, and drawing "0 online" would be a platform-wide false alarm. */
  onlineKnown: boolean;
  /** Customers of this kind that sold nothing at all in the window. */
  idle: number;
  orders: number;
  tillChecks: number;
  ops: number;
  revenue: number;
  visitors: number;
  /** ⚠️ Two figures, never added: the till is a monthly subscription and the
   *  website is billed per order, so one number would match no invoice. */
  subscription: number;
  perOrder: number;
  lastSale?: string;
  top?: BizTenant;
  bottom?: BizTenant;
};

export type BizTenant = {
  id: string;
  slug: string;
  name: string;
  revenue: number;
  ops: number;
  online: boolean;
  lastSale?: string;
};

/** ⚠️ Thirty days by default so the per-order figure and the monthly
 *  subscription beside it describe the same length of time.
 *
 *  ⚠️ **Two typed dates win over the day count**, the same rule the overview
 *  follows and the same resolver behind it: somebody who filled both boxes
 *  asked a specific question, and quietly answering a different one is worse
 *  than refusing. */
export const business = (q: { days?: number; from?: string; to?: string } = {}) => {
  const p = new URLSearchParams();
  if (q.from && q.to) {
    p.set("from", q.from);
    p.set("to", q.to);
  } else {
    p.set("days", String(q.days ?? 30));
  }
  return req<{
    rows: BizRow[];
    from: string;
    to: string;
    days: number;
    now: string;
  }>(`/business?${p.toString()}`);
};

// ---- The blog ----

export type BlogText = { title: string; excerpt: string; body: string };

export type BlogPost = {
  id?: string;
  slug: string;
  cover?: string;
  uz: BlogText;
  ru: BlogText;
  en: BlogText;
  published: boolean;
  publishedAt?: string;
  /** ⚠️ Read-only here. The counter belongs to the readers, and a save that
   *  carried a stale copy of it would undo every read since the editor opened —
   *  the server ignores whatever is sent. */
  views: number;
  author?: string;
  createdAt?: string;
  updatedAt?: string;
};

export const blogList = () => req<{ posts: BlogPost[] }>("/blog");
export const blogSave = (post: BlogPost) =>
  req<BlogPost>("/blog", { method: "POST", body: JSON.stringify(post) });
export const blogDelete = (id: string) =>
  req<{ deleted: boolean }>(`/blog/${id}`, { method: "DELETE" });

/** Upload one picture and get the address to paste into a post.
 *
 *  ⚠️ **Multipart, so no `Content-Type` is set by hand.** The boundary is
 *  generated by the browser and a header written here would name a different
 *  one — a request the server cannot parse, and an error that reads as "the
 *  file is broken". */
export async function blogUpload(file: File): Promise<{ url: string }> {
  const form = new FormData();
  form.append("file", file);
  return req<{ url: string }>("/blog/upload", { method: "POST", body: form });
}

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
  req<TenantDetail>(`/tenants/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

/** One thing the purge attempted, and whether it worked. */
export interface PurgeStep {
  step: string;
  ok: boolean;
  error?: string;
}

/** Erases a customer's database, photographs and container. Irreversible.
 *
 *  ⚠️ Returns 200 with a per-step report even when a step failed: the work
 *  cannot be retried as a whole, so what the operator needs is which half to
 *  finish by hand — an error status would hide the list that says so. */
export const purgeTenant = (id: string, confirm: string, reason: string) =>
  req<{ purged: boolean; steps: PurgeStep[]; slug: string }>(
    `/tenants/${id}/purge`,
    { method: "POST", body: JSON.stringify({ confirm, reason }) },
  );

/** Whole so'm, grouped. Intl is avoided for the same reason as in the tenant
 *  app: it follows the device locale and would print a different separator on
 *  the server than in the browser. */
/** The digit-group separator: U+00A0, a **non-breaking** space.
 *
 *  ⚠️ An ordinary space here is a line-break opportunity, and a table cell is
 *  exactly where the browser takes it: "12 500 000" wraps into a stack of
 *  three-digit fragments, and a column of money turns into a column of
 *  nonsense. It only shows up once a customer's numbers get long, which is the
 *  moment the column matters most. */
const GROUP = "\u00a0";

export function money(n: number): string {
  const s = Math.round(n).toString();
  let out = "";
  for (let i = 0; i < s.length; i++) {
    if (i > 0 && (s.length - i) % 3 === 0) out += GROUP;
    out += s[i];
  }
  return out;
}

/** Money for a column, not for a receipt: "128,5 mln", "1,28 mlrd".
 *
 *  So'm has no minor unit and large magnitudes, so a real figure runs to ten
 *  or eleven digits — three of those in one table row is more width than any
 *  laptop has, and no amount of non-breaking spaces fixes that. Shortening is
 *  the only thing that does, and it is also how the numbers are said out loud.
 *
 *  **Always paired with the exact value in `title`**, and every screen that
 *  has room — the tenant card, the invoice — keeps showing it in full. A
 *  rounded figure is for scanning a list, never for quoting to a customer. */
export function moneyShort(n: number): string {
  const v = Math.round(n);
  if (v >= 1_000_000_000) return `${trim(v / 1_000_000_000)}\u00a0mlrd`;
  // Below a million the exact number is short enough to be worth keeping:
  // early invoices are a few thousand so'm and rounding them to "0,0 mln"
  // would turn the useful column into a column of zeroes.
  if (v >= 1_000_000) return `${trim(v / 1_000_000)}\u00a0mln`;
  return money(v);
}

/** Two significant-ish digits, comma-separated as Uzbek and Russian write it,
 *  and no trailing ",0" — "7 mln" reads better than "7,0 mln". */
function trim(x: number): string {
  const s = (x < 10 ? x.toFixed(2) : x.toFixed(1)).replace(/\.?0+$/, "");
  return s.replace(".", ",");
}

/** "2026-08-17" → "17.08" — the year dropped.
 *
 *  Used for the billing window in the customer list, where both ends are
 *  almost always the current year and the full form was the widest thing in
 *  the row while being the least important. The exact dates, years and all,
 *  are on the customer's own card. */
export function dayShort(d: string): string {
  const [, m, dd] = (d ?? "").split("-");
  return m && dd ? `${dd}.${m}` : d;
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

// ---- Data export grant ----
//
// The customer's "download everything" button, which does not exist in their
// panel until it is opened here. See control/internal/handlers/export.go: the
// archive is one file holding every one of their customers' names, phones and
// addresses, so taking it has to be a dated, attributable act rather than a
// permanent affordance on a panel whose password has been round three managers.

export interface ExportDownload {
  at: string;
  /** The tenant panel account that pressed it — always their owner. */
  by: string;
  bytes: number;
  files: number;
}

export interface ExportGrant {
  enabled: boolean;
  /** enabled **and** not yet expired, decided by the server's clock. */
  active?: boolean;
  reason?: string;
  grantedBy?: string;
  grantedAt?: string;
  expiresAt?: string;
  downloads: ExportDownload[];
}

export const exportGrant = (tenantId: string) =>
  req<ExportGrant>(`/tenants/${tenantId}/export`);

export const setExportGrant = (
  tenantId: string,
  body: { enabled: boolean; reason?: string; days?: number },
) =>
  req<ExportGrant>(`/tenants/${tenantId}/export`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

// ---- The till subscription ----
//
// What a customer bought for their counter. Read and written only by operators
// who can provision: this is not a sales concession, it is what the customer is
// billed, and the console draws the price before it is saved so nobody agrees
// to a number they have not seen.

export interface TillPlan {
  id: string;
  monthly: number;
  /** Till screens one branch may bind. 0 means no cap. */
  registers: number;
  modules: string[];
  /** Priced per customer rather than from the ladder. */
  individual: boolean;
}

export interface TillSubscription {
  enabled: boolean;
  plan?: string;
  addons: string[];
  /** Blocks of ten daily AI requests bought on top of the plan. */
  aiExtra?: number;
  /** How many televisions this restaurant pays for. ⚠️ The count *is* the
   *  entitlement: it grants the module and prices it, so there is no separate
   *  switch that could disagree with it. */
  tvScreens?: number;
  branches?: number;
  priceOverride?: number;
  since?: string;
  paidUntil?: string;
  note?: string;
  updatedBy?: string;
  updatedAt?: string;
  /** What this configuration costs a month, from the ladder. 0 for an
   *  Enterprise customer with no agreed number — the panel says so in words,
   *  because a blank price and a free customer must not look alike. */
  monthly: number;
  plans: TillPlan[];
}

export const tillSubscription = (tenantId: string) =>
  req<TillSubscription>(`/tenants/${tenantId}/till`);

export const setTillSubscription = (
  tenantId: string,
  body: {
    enabled: boolean;
    plan?: string;
    addons?: string[];
    aiExtra?: number;
    tvScreens?: number;
    branches?: number;
    priceOverride?: number;
    paidUntil?: string | null;
    note?: string;
  },
) =>
  req<TillSubscription>(`/tenants/${tenantId}/till`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

// ---- The page-layout constructor ----
//
// The design is drawn here and written into the tenant's own database (see
// control/internal/handlers/design.go). A draft is invisible to the live site;
// publishing copies it across.

export interface DesignSection {
  type: string;
  variant?: string;
  /** 1–12. On a phone every band is 12 — that is the whole responsive rule. */
  span: number;
  hidden?: boolean;
  canvas?: DesignCanvas | null;
  /** Typed settings, declared by the schema. */
  settings?: Record<string, unknown>;
  /** Repeatable items inside the section. */
  blocks?: {
    type: string;
    settings?: Record<string, unknown>;
    hidden?: boolean;
  }[];
  style?: {
    tone?: string;
    padding?: string;
    align?: string;
    rounded?: boolean;
  };
  binding?: { categories?: string[]; popularOnly?: boolean; limit?: number };
}

/** Where a freely placed element sits, in percent of its band. Never pixels. */
export interface DesignBox {
  x: number;
  y: number;
  w: number;
  h: number;
  z?: number;
}

export interface DesignElement {
  type: string;
  box: DesignBox;
  /** The phone layout, drawn separately — there is no arithmetic that derives
   *  one from the desktop composition. Absent means "stack in drawn order". */
  mobile?: DesignBox | null;
  hidden?: boolean;
  hiddenMobile?: boolean;
  text?: { uz: string; ru: string; en: string };
  subtext?: { uz: string; ru: string; en: string };
  image?: string;
  images?: string[];
  link?: string;
  icon?: string;
  value?: number;
  style?: {
    font?: string;
    weight?: string;
    align?: string;
    color?: string;
    tone?: string;
    size?: number;
    opacity?: number;
    rounded?: boolean;
    shadow?: boolean;
  };
  binding?: { categories?: string[]; popularOnly?: boolean; limit?: number };
}

export interface DesignCanvas {
  height?: number;
  heightMobile?: number;
  background?: string;
  backgroundOpacity?: number;
  elements?: DesignElement[];
}

/** A named style, saved once and applied to other elements.
 *
 *  ⚠️ Applying one **copies** it rather than referencing it — the same rule the
 *  template gallery follows: a preset edited next month must not silently repaint
 *  elements on a page a customer already approved. */
export interface StylePreset {
  name: string;
  style: DesignElement["style"];
}

export interface DesignState {
  blocks: string[];
  /** The restaurant's own menu categories, for the bands that draw from some of
   *  them rather than all. Sent with the design because a picker that arrives
   *  later shows a saved selection as blank chips, which reads as lost work. */
  categories?: { id: string; name: string }[];
  draft: {
    sections: DesignSection[] | null;
    customCss?: string;
    stylePresets?: StylePreset[] | null;
    updatedAt?: string;
    drawnBy?: string;
  };
  live: {
    sections: DesignSection[] | null;
    publishedAt?: string;
    drawnBy?: string;
  };
  published: boolean;
}

export interface DesignTemplate {
  id: string;
  name: string;
  /** One line on what the layout is for. Built-ins carry it; a five-name list is
   *  a list nobody can choose from. */
  note?: string;
  sections: DesignSection[];
  /** ⚠️ Ships in the binary: read-only, and there is nothing to delete. */
  builtin?: boolean;
  createdBy?: string;
  createdAt?: string;
}

export const tenantDesign = (tenantId: string) =>
  req<DesignState>(`/tenants/${tenantId}/design`);

export const saveTenantDesign = (
  tenantId: string,
  sections: DesignSection[],
  customCss = "",
  stylePresets: StylePreset[] = [],
) =>
  req<{ saved: number }>(`/tenants/${tenantId}/design`, {
    method: "PUT",
    body: JSON.stringify({ sections, customCss, stylePresets }),
  });

/** A short-lived link that shows the **unpublished** draft on the real site.
 *
 *  ⚠️ This is what makes the editor an editor: a schematic preview cannot answer
 *  "does this look like the picture the customer sent us", which is the only
 *  question that matters when the brief is a screenshot. Two hours, one brand,
 *  no permissions — see the control plane's PreviewTenantDesign. */
export const previewTenantDesign = (tenantId: string) =>
  req<{ url: string; expiresAt: string }>(
    `/tenants/${tenantId}/design/preview`,
    { method: "POST" },
  );

export const publishTenantDesign = (tenantId: string) =>
  req<{ published: number; publishedAt: string; note: string }>(
    `/tenants/${tenantId}/design/publish`,
    { method: "POST" },
  );

export const revertTenantDesign = (tenantId: string) =>
  req<{ reverted: boolean }>(`/tenants/${tenantId}/design`, {
    method: "DELETE",
  });

/** What each section can be asked. The settings panel is drawn from this rather
 *  than written per type — see SchemaSettings. */
export const designSchema = () =>
  req<{ sections: unknown[] }>("/design-schema");

export const designTemplates = () =>
  req<{ items: DesignTemplate[] }>("/design-templates");

export const saveDesignTemplate = (name: string, sections: DesignSection[]) =>
  req<DesignTemplate>("/design-templates", {
    method: "POST",
    body: JSON.stringify({ name, sections }),
  });

export const deleteDesignTemplate = (id: string) =>
  req<{ deleted: boolean }>(`/design-templates/${id}`, { method: "DELETE" });

// ---- Crash reports ----
//
// ⚠️ **The other end of the support queue.** That one depends on a restaurant
// noticing, deciding it is worth reporting, and describing it; this one has none
// of those steps in it, so most of what lands here is fixed before anybody
// writes in about it.

export type ErrorGroupRow = {
  id: string;
  slug: string;
  restaurant: string;
  app: string;
  key: string;
  message: string;
  where?: string;
  count: number;
  today: number;
  users: number;
  firstAt: string;
  lastAt: string;
  firstVersion?: string;
  latestVersion?: string;
  latestPlatform?: string;
  resolved: boolean;
  resolvedAt?: string;
  resolvedBy?: string;
  resolvedCount?: number;
  note?: string;
};

export type ErrorSample = {
  at: string;
  stack?: string;
  context?: string;
  version?: string;
  platform?: string;
  branch?: string;
  role?: string;
};

export const reportList = (params?: {
  state?: string;
  app?: string;
  slug?: string;
  q?: string;
}) => {
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params ?? {})) if (v) qs.set(k, v);
  const suffix = qs.toString() ? `?${qs}` : "";
  return req<{ groups: ErrorGroupRow[] }>(`/reports${suffix}`);
};

export const reportGroup = (id: string) =>
  req<{ group: ErrorGroupRow & { samples?: ErrorSample[] } }>(`/reports/${id}`);

export const reportResolve = (
  id: string,
  body: { resolved: boolean; note?: string },
) =>
  req<{ ok: boolean }>(`/reports/${id}/resolve`, {
    method: "POST",
    body: JSON.stringify(body),
  });

// ---- Who sends us customers from outside ----
//
// ⚠️ Not the landing page's "partners", which are the logos of the payment and
// till systems we integrate with. These are the firms that walk into twenty
// kitchens a week — the register engineer, the packaging supplier, the
// accountant — and send one our way when there is something in it for them.
// See control/internal/models/referral.go.

export type Referrer = {
  id: string;
  name: string;
  /** The code in the link they hand out: keel.uz/h/<code>. */
  code: string;
  contact?: string;
  note?: string;
  /** Their share of what the customers they sent actually paid us... */
  percent: number;
  /** ...for this many months from the day that customer started paying.
   *  Zero is no limit. */
  months: number;
  isActive: boolean;
  /** Customers attributed to them, and how many of those ever started paying.
   *  ⚠️ Both: the gap between them is the quality of the channel. */
  tenants: number;
  paying: number;
  collected: number;
  commission: number;
};

export type ReferrerTenant = {
  id: string;
  slug: string;
  name: string;
  subscribedAt?: string;
  status: string;
  windowEndsAt?: string;
  collected: number;
  commission: number;
};

export const referrers = () => req<{ referrers: Referrer[] }>("/referrers");

export const referrer = (id: string) =>
  req<{ referrer: Referrer; tenants: ReferrerTenant[] }>(`/referrers/${id}`);

export const createReferrer = (body: Partial<Referrer>) =>
  req<Referrer>("/referrers", { method: "POST", body: JSON.stringify(body) });

export const updateReferrer = (id: string, body: Partial<Referrer>) =>
  req<Referrer>(`/referrers/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });

// ---- Search engines ----
//
// ⚠️ Only IndexNow is a push. Google cannot be pushed to at all — see
// `control/internal/handlers/seo.go` and the console screen, both of which say
// so rather than offering a button that does nothing.

export type SeoStatus = {
  origin: string;
  sitemap: string;
  keyLocation: string;
  hasKey: boolean;
  urls: number;
  error?: string;
  last?: {
    lastPingAt: string;
    lastCount: number;
    lastStatus: number;
    lastMessage?: string;
  };
};

export type SeoPingResult = {
  urls: number;
  /** The engine's own status. 403 is an unreadable key file, 422 is a URL on
   *  another host — different fixes, so it is passed through rather than
   *  flattened into ok/not-ok. */
  status: number;
  message?: string;
  ok: boolean;
};

export const seoStatus = () => req<SeoStatus>("/seo");
export const seoPing = () =>
  req<SeoPingResult>("/seo/indexnow", { method: "POST" });
