// Types mirror the Go backend JSON models (backend/internal/models/models.go).

export interface GeoPoint {
  text: string;
  lat: number;
  lng: number;
}

export interface WorkingHour {
  day: number; // 0=Sunday .. 6=Saturday
  open: string;
  close: string;
  isClosed: boolean;
}

export interface Socials {
  instagram: string;
  telegram: string;
  facebook: string;
}

// Each zone prices independently: a flat fee, or a distance-based one measured
// from the restaurant. `pricing` is missing on older documents = "fixed".
export interface DeliveryZone {
  name: string;
  polygon: number[][]; // [lat, lng] pairs
  pricing?: "fixed" | "perKm";
  fee: number; // fixed model
  baseFee?: number; // perKm model
  perKm?: number; // perKm model
}

export interface DeliverySettings {
  enabled: boolean;
  minOrder: number;
  freeDeliveryFrom: number | null;
  // How the fee is computed. Missing on older documents = inferred from zones.
  mode?: "radius" | "zones";
  // How close (metres) the courier must be before the app lets them mark the
  // order delivered. 0 disables the check.
  arrivalRadiusM?: number;
  zones: DeliveryZone[];
  baseFee: number;
  perKm: number;
  maxKm: number;
}

// A piece of site copy in the three supported languages; ru/en fall back to uz.
export interface LocalizedText {
  uz: string;
  ru: string;
  en: string;
}

export interface SiteContent {
  aboutTitle: LocalizedText;
  aboutText: LocalizedText;
  footerNote: LocalizedText;
  tagline: LocalizedText;
}

// Restyling knobs exposed to the restaurant in the admin panel.
/** One dish's place in the ABC/XYZ analysis.
 *
 *  ABC is share of takings (Pareto: 80/15/5); XYZ is how steady daily demand
 *  is, as a coefficient of variation. The cross is where the decisions live —
 *  AX is what must never run out, CZ is what could leave the menu. */
export interface AbcXyzRow {
  name: string;
  qty: number;
  revenue: number;
  /** This dish's share of the period's takings, 0–100. */
  share: number;
  /** Running total down the sorted list — what the A/B/C cut is made on. */
  cumulative: number;
  /** Coefficient of variation of daily sales, in percent. */
  variation: number;
  abc: "A" | "B" | "C";
  xyz: "X" | "Y" | "Z";
  /** "AX", "CZ" — the pair, so one field can be filtered on. */
  class: string;
  /** Days of the period the dish sold on at all. The honesty check on XYZ: a
   *  dish sold on two days out of thirty has a variation that means little. */
  days: number;
}

export interface AbcXyzResponse {
  from: string;
  to: string;
  note: string;
  total: number;
  items: AbcXyzRow[];
}

/** What Google Search Console and Yandex Webmaster hand the owner to prove the
 *  site is theirs. Each is the `content` of a meta tag, not the whole tag. */
export interface SeoSettings {
  google?: string;
  yandex?: string;
}

export interface SiteTheme {
  brand: string;
  brandDark: string;
  radius: number | null;
  buttonShape: "pill" | "match" | "";
  font: "classic" | "modern" | "soft" | "";
  background: "warm" | "white" | "cool" | "sand" | "";
  shadow: "soft" | "none" | "strong" | "";
  buttonStyle: "solid" | "outline" | "soft" | "";
  scale: number | null;
}

export interface Restaurant {
  id: string;
  name: string;
  description: string;
  logoUrl: string;
  coverUrl: string;
  phones: string[];
  address: GeoPoint;
  socials: Socials;
  workingHours: WorkingHour[];
  delivery: DeliverySettings;
  currency: string;
  /** The restaurant's own 2GIS MapGL key. Public by nature — MapGL is a
   *  browser library — and protected by the domain restriction set in the
   *  2GIS account, not by secrecy. */
  mapApiKey?: string;
  /** Search-console verification tokens. Public by nature: a token's whole job
   *  is to sit in the page head where a crawler reads it. */
  seo?: SeoSettings;
  content?: SiteContent;
  theme?: SiteTheme;
  // Table booking: the hand-drawn floor plan and the rules around it. Absent on
  // documents written before booking existed.
  booking?: BookingSettings;
  loyalty?: LoyaltySettings;
  updatedAt: string;
}

export interface RestaurantResponse {
  // Address, phones, hours, delivery and floor plan are the serving branch's;
  // currency and socials stay the company's. See the backend GetRestaurant.
  restaurant: Restaurant;
  isOpenNow: boolean;
  /** The branch this answer was given for. */
  branch?: Branch;
  /** The brand whose face the site is wearing. */
  brand?: Brand;
}

export interface Category {
  id: string;
  name: string; // uz (base)
  nameRu?: string;
  nameEn?: string;
  slug: string;
  sortOrder: number;
  isActive: boolean;
  imageUrl: string;
}

// One selectable value inside a MenuOption. `priceDelta` is added to the dish
// price (may be negative). Uzbek is the base name; RU/EN are optional.
export interface OptionChoice {
  name: string;
  nameRu?: string;
  nameEn?: string;
  priceDelta: number;
}

// A group of choices attached to a dish ("Hajm", "Qo'shimcha"). `required`
// means the customer must answer it; `multiple` turns it into checkboxes.
export interface MenuOption {
  name: string;
  nameRu?: string;
  nameEn?: string;
  required: boolean;
  multiple: boolean;
  choices: OptionChoice[];
}

export interface MenuItem {
  id: string;
  categoryId: string;
  name: string; // uz (base)
  description: string;
  nameRu?: string;
  nameEn?: string;
  descriptionRu?: string;
  descriptionEn?: string;
  price: number;
  oldPrice: number | null;
  imageUrl: string;
  images: string[] | null;
  isAvailable: boolean;
  /** Run out at the branch serving this guest today. Not stored on the dish —
   *  the server fills it in from the branch (see Branch.soldOut). A dish can be
   *  available in general and sold out here. */
  soldOut?: boolean;
  /** Non-empty makes this a combo: a fixed set sold for `price`. */
  comboItems?: ComboLine[];
  /** Computed by the server, never stored: what the same dishes cost bought
   *  separately, and what the set contains. A stored copy would go stale the
   *  moment a member dish was repriced. */
  comboBasePrice?: number;
  comboContents?: OrderComboLine[];
  isPopular: boolean;
  sortOrder: number;
  options: MenuOption[] | null;
  tags: string[] | null;
  updatedAt: string;
}

export interface MenuGroup {
  category: Category;
  items: MenuItem[];
}

export type OrderStatus =
  | "pending"
  | "confirmed"
  | "preparing"
  | "on_the_way"
  | "delivered"
  | "cancelled";

export interface OrderCustomer {
  name: string;
  phone: string;
}

export interface OrderAddress {
  text: string;
  lat: number;
  lng: number;
  comment: string;
}

// A selected choice frozen at order time, with the delta that was charged.
export interface OrderItemOption {
  name: string; // option group (uz base)
  choice: string; // selected choice (uz base)
  priceDelta: number;
}

export interface OrderItem {
  menuItemId: string;
  name: string;
  // Unit price actually charged: dish price + selected option deltas.
  price: number;
  qty: number;
  options: OrderItemOption[] | null;
  // The customer's note for this dish alone ("piyozsiz").
  comment?: string;
  /** What a combo contained when it was ordered — the kitchen cooks from this. */
  comboItems?: OrderComboLine[];
}

export interface StatusEvent {
  status: OrderStatus;
  at: string;
}

export interface Order {
  // Dine-in only: the table whose QR code the guest scanned.
  tableId?: string;
  tableNumber?: string;
  id: string;
  // Set when the order was placed by a signed-in customer.
  userId?: string;
  number: string;
  status: OrderStatus;
  customer: OrderCustomer;
  type: "delivery" | "pickup" | "dinein";
  address: OrderAddress;
  items: OrderItem[];
  subtotal: number;
  /** Every discount applied, shown as separate lines on the receipt. */
  discounts?: OrderDiscount[];
  discountTotal?: number;
  /** Points the guest put towards this order, and the cashback it earned. */
  pointsSpent?: number;
  pointsEarned?: number;
  deliveryFee: number;
  total: number;
  paymentMethod: PaymentMethod;
  deliveryZone: string;
  distanceKm: number;
  statusHistory?: StatusEvent[] | null;
  // Why the restaurant cancelled it (only on cancelled orders).
  cancelReason?: string;
  // Assigned courier (delivery orders).
  courierId?: string;
  courierName?: string;
  externalDelivery?: ExternalDelivery | null;
  /** The operator who took this order over the phone. Absent when the guest
   *  placed it themselves — which is what makes it worth showing. */
  takenBy?: string;
  /** Where the money stands. Cash orders stay "unpaid" for their whole life
   *  and that is not a problem; an online order starts "pending" and only the
   *  provider's own callback moves it to "paid". */
  paymentStatus?: PaymentStatus;
  paidAt?: string;
  /** When the order became the kitchen's problem: placed, for cash; paid, for
   *  an online order. */
  queuedAt?: string;
  /** When the kitchen said "done". A timestamp rather than a status on purpose
   *  — see models.Order.ReadyAt: "ready" means different things for delivery,
   *  pickup and dine-in, and a status is read by the courier app, the tracking
   *  page, the statistics and three dictionaries. */
  readyAt?: string;
  /** What happened when this order was pushed to the restaurant's till.
   *  Absent when no POS is connected, which is most installs. */
  pos?: OrderPOS;
  createdAt: string;
  updatedAt: string;
}

export interface OrderTrack {
  paymentMethod?: PaymentMethod;
  paymentStatus?: PaymentStatus;
  /** Set while the order is still owed money and can still be paid. */
  payUrl?: string;
  // Dine-in orders: which table the guest scanned.
  tableNumber?: string;
  number: string;
  status: OrderStatus;
  type: "delivery" | "pickup" | "dinein";
  total: number;
  createdAt: string;
  address?: OrderAddress;
  cancelReason?: string;
  courierName?: string;
  // Only present while the order is "on_the_way".
  courier?: {
    name: string;
    phone: string;
    location: CourierLocation;
  };
}

export interface DeliveryQuote {
  available: boolean;
  deliveryFee: number;
  zone: string;
  distanceKm: number;
  minOrder: number;
  /** Which branch the server picked to cook and carry it (nearest that covers
   * the address). Absent when nobody delivers there. */
  branchId?: string;
  branchName?: string;
  prepMinutes?: number;
}

// ---- Loyalty ----

export type LoyaltyKind = "earn" | "spend" | "revoke" | "adjust";

export interface LoyaltyTxn {
  id: string;
  orderId?: string;
  orderNumber?: string;
  kind: LoyaltyKind;
  /** Signed: positive adds, negative takes away. */
  points: number;
  balanceAfter: number;
  note?: string;
  at: string;
}

export interface LoyaltyInfo {
  enabled: boolean;
  balance: number;
  earnPercent: number;
  maxRedeemPercent: number;
  minOrderToEarn: number;
  transactions: LoyaltyTxn[];
}

/** Cashback settings, edited by the owner. Company-wide: the customer is. */
export interface LoyaltySettings {
  enabled: boolean;
  earnPercent: number;
  minOrderToEarn: number;
  maxRedeemPercent: number;
  welcomePoints: number;
}

// A site customer — created through phone + one-time SMS code.
export interface SiteUser {
  id: string;
  firstName: string;
  lastName: string;
  phone: string;
  /** Cashback balance in points (1 point = 1 so'm). */
  points?: number;
  /** "MM-DD" — the day only. A birth year is data the restaurant has no use
   *  for, and asking for it costs answers. */
  birthday?: string;
  /** The restaurant's own labels and notes about this customer. */
  tags?: string[];
  note?: string;
  /** qr | site | instagram | referral | phone */
  source?: string;
  /** This guest asked not to receive campaign messages. A hard exclusion on
   *  every send; order updates and login codes still go out. */
  noMarketing?: boolean;
  authProvider?: "phone";
  addresses?: UserAddress[];
  createdAt: string;
  updatedAt: string;
}


export interface UserAddress {
  label: string;
  text: string;
  lat: number;
  lng: number;
  comment: string;
}

// Admin dashboard row: customer + order aggregates.
export interface AdminUserRow extends SiteUser {
  ordersCount: number;
  ordersTotal: number;
  avgOrder: number;
  firstOrderAt: string | null;
  lastOrderAt: string | null;
  /** The single number that says whether this customer needs anything doing. */
  daysSinceLast: number | null;
  /** Several at once is normal: a VIP who has gone quiet is both, and that
   *  pair is exactly the one worth acting on. */
  segments: CustomerSegment[];
}

// ---- Feedback ----

export interface Feedback {
  id: string;
  orderId: string;
  orderNumber: string;
  userId?: string;
  customer: { name: string; phone: string };
  /** 1..5. Three or below is a complaint, not a score. */
  rating: number;
  comment: string;
  handled: boolean;
  handledBy?: string;
  handledAt?: string;
  /** What was actually done — the part that makes the record worth keeping. */
  resolution?: string;
  createdAt: string;
}

export interface FeedbackList {
  feedback: Feedback[];
  stats: { count: number; average: number; open: number };
}

export type CustomerSegment =
  | "new"
  | "regular"
  | "vip"
  | "sleeping"
  | "lost"
  | "birthday"
  | "noOrders"
  | "unhappy";

// GET /admin/users/{id}
export interface AdminUserDetail {
  user: SiteUser;
  orders: Order[];
  stats: {
    ordersCount: number;
    ordersTotal: number;
    delivered: number;
    cancelled: number;
    lastOrderAt: string | null;
    firstOrderAt: string | null;
  };
}

export interface PhoneCodeResponse {
  ok: boolean;
  phone: string;
  expiresIn?: number;
  demo: boolean;
  code?: string; // demo provider only
}

/** One dish inside a combo, and how many of it the set contains. */
export interface ComboLine {
  menuItemId: string;
  qty: number;
}

/** What a combo contained, frozen at order time so an old receipt still reads. */
export interface OrderComboLine {
  name: string;
  qty: number;
}


// ---- Discounts: promo codes and campaigns ----
//
// One type, two triggers. A code the guest types and an automatic campaign are
// the same rule to the till; they differ only in what sets them off.

export type PromotionTrigger = "code" | "auto";
export type PromotionKind = "percent" | "fixed" | "freeDelivery";
export type PromotionScope = "order" | "category" | "items";

export interface Promotion {
  id: string;
  brandId?: string;
  branchIds?: string[];
  name: string;
  nameRu?: string;
  nameEn?: string;
  trigger: PromotionTrigger;
  code: string;
  kind: PromotionKind;
  value: number;
  /** Ceiling for a percentage discount; 0 = none. */
  maxDiscount: number;
  minOrder: number;
  scope: PromotionScope;
  categoryIds?: string[];
  menuItemIds?: string[];
  startsAt?: string | null;
  endsAt?: string | null;
  /** Weekdays it runs on (0=Sunday). Empty = every day. */
  days?: number[];
  timeFrom?: string;
  timeTo?: string;
  orderTypes?: string[];
  usageLimit: number;
  perUserLimit: number;
  firstOrderOnly: boolean;
  usedCount: number;
  isActive: boolean;
  sortOrder: number;
  /** Computed by the server: what the rule is actually doing right now.
   *  "active" on a code that expired last month is a lie the list used to tell. */
  status?: PromotionStatus;
}

export type PromotionStatus =
  | "off"
  | "scheduled"
  | "expired"
  | "usedUp"
  | "idle"
  | "running";

/** Who used a code, and how many of them. */
export interface PromotionUsage {
  orders: {
    orderId: string;
    number: string;
    customer: string;
    phone: string;
    amount: number;
    total: number;
    createdAt: string;
  }[];
  stats: {
    redemptions: number;
    /** Distinct customers — one regular using it five times is one person. */
    people: number;
    /** What the campaign has cost so far. */
    discounted: number;
  };
}

/** One discount as it was applied, frozen onto the order. */
export interface OrderDiscount {
  promotionId?: string;
  name: string;
  kind: PromotionKind;
  trigger: PromotionTrigger;
  code?: string;
  amount: number;
}

/** The checkout's preview of the bill — the same pipeline the order runs. */
export interface OrderQuote {
  subtotal: number;
  discounts: OrderDiscount[];
  discountTotal: number;
  deliveryFee: number;
  total: number;
  /** Why a typed code was refused. Never fatal. */
  codeError?: string;
  codeApplied: boolean;
  available: boolean;
  minOrder: number;
  /** What the restaurant receives for the food; the minimum is measured on it. */
  payableSubtotal: number;
  belowMinimum: boolean;
  branchId?: string;
  branchName?: string;
  prepMinutes?: number;
  /** Loyalty: what was applied, what is available, and what this will earn. */
  pointsSpent: number;
  pointsBalance: number;
  pointsMax: number;
  pointsEarn: number;
}

export type PaymentMethod = "cash" | "payme" | "click" | "uzum" | "atmos";

// ---- External delivery services ----

// An outside delivery service (Yandex Delivery, a taxi firm, a door-to-door
// courier company) for restaurants that have no couriers of their own.
export interface DeliveryProvider {
  id: string;
  name: string;
  // "link" opens a URL with the order details substituted, "phone" dials,
  // "api" lets the server file the request with the service itself.
  kind: "link" | "phone" | "api";
  url: string;
  phone: string;
  note: string;
  isActive: boolean;
  sortOrder: number;
  // ---- kind "api" ----
  apiProvider?: string; // "yandex"
  // The token itself never reaches the browser; this only says one is stored.
  hasToken?: boolean;
  apiToken?: string; // write-only: filled in the form, never returned
  apiBaseUrl?: string;
  apiTariff?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ExternalDelivery {
  providerId?: string;
  providerName: string;
  trackingId: string;
  note: string;
  calledAt: string;
  // What the service charged, entered by hand for link/phone providers.
  cost?: number;
  // Present when the service was called through its API.
  claimId?: string;
  status?: string;
  price?: string;
  courierName?: string;
  courierPhone?: string;
  trackUrl?: string;
  syncedAt?: string;
}

// ---- Courier ----

// off = not on shift, free = on shift and available, busy = delivering.
export type CourierStatus = "off" | "free" | "busy";

export interface CourierLocation {
  lat: number;
  lng: number;
  accuracy: number;
  at: string;
}

// What a courier earns per delivered order.
export type CourierPayoutMode = "deliveryFee" | "perOrder" | "percent";

export interface Courier {
  id: string;
  name: string;
  phone: string;
  username: string;
  status: CourierStatus;
  vehicle: string;
  isActive: boolean;
  location?: CourierLocation | null;
  payoutMode?: CourierPayoutMode;
  payoutPerOrder?: number;
  payoutPercent?: number;
  createdAt: string;
  updatedAt: string;
}

// One time window of a courier's work.
export interface CourierPeriod {
  orders: number;
  earnings: number;
  // Cash physically collected — what has to be handed back to the restaurant.
  cash: number;
  total: number;
}

export interface CourierStats {
  today: CourierPeriod;
  week: CourierPeriod;
  month: CourierPeriod;
  all: CourierPeriod;
  active: number;
  /** Cash actually still on the courier: everything collected, less what has
   *  been handed back. `all.cash` alone only ever grows. */
  cashInHand: number;
  cashSettled: number;
}

// A delivered order plus what the courier earned on it.
export interface CourierOrderRow extends Order {
  earned: number;
  deliveredAt: string;
}

/** One handover of collected cash back to the restaurant. */
export interface CourierSettlement {
  id: string;
  courierId: string;
  amount: number;
  takenBy: string;
  note?: string;
  at: string;
}

export interface AdminCourierDetail {
  courier: Courier;
  stats: CourierStats;
  orders: CourierOrderRow[];
  /** Cash handovers, newest first. */
  settlements?: CourierSettlement[];
}

export interface AdminUser {
  id: string;
  username: string;
  role: "owner" | "manager";
  /** A manager pinned to one branch sees only that branch. Empty means the
   *  whole company, which is what an owner gets. */
  branchId?: string;
  mustChangePassword: boolean;
  // The site customer this panel account belongs to (empty on the very first
  // owner, which is seeded before any customer exists).
  userId?: string;
  name?: string;
  phone?: string;
  /** This person's internal extension on the phone system: which handset the
   *  exchange rings first, and how "who answered" is filled in. */
  pbxExtension?: string;
  createdBy?: string;
  lastLoginAt?: string;
  createdAt: string;
}

// One line of the admin activity log. `action` is a stable id like
// "order.cancel" which the panel translates.
export interface AdminLog {
  id: string;
  adminId?: string;
  adminName: string;
  adminRole: string;
  action: string;
  targetType: string;
  targetId: string;
  targetLabel: string;
  details: string;
  at: string;
}

export interface LoginResponse {
  token: string;
  user: AdminUser;
}

// ---- Dashboard statistics (GET /admin/stats?from=&to=) ----

/** Order and money totals for the chosen period. */
export interface StatsPeriod {
  orders: number;
  /** Money actually in hand: cash handed over on delivery, or a card payment
   *  the bank confirmed. Not every order that was placed. */
  revenue: number;
  /** Placed, not cancelled, not yet collected — the food in the kitchen and on
   *  the road. Real work, but not takings. */
  pending: number;
  /** How many orders the revenue came from, so the average can be checked. */
  paid: number;
  avgOrder: number;
  delivery: number;
  pickup: number;
  dineIn: number;
  cancelled: number;
  delivered: number;
  byStatus: Record<string, number>;
  deliveryFee: number;
  cashTotal: number;
}

/** A best-selling dish inside the period. */
export interface StatsDish {
  name: string;
  qty: number;
  total: number;
}

/** One day of the dashboard chart. Every day in the range is present,
 *  including the empty ones — a chart that skips them draws a straight line
 *  across a closed week. */
export interface StatsDay {
  date: string;
  orders: number;
  /** On the same basis as the totals above it: collected, not placed. */
  revenue: number;
}

export interface AdminStats {
  from: string | null;
  to: string | null;
  period: StatsPeriod;
  top: StatsDish[];
  series: StatsDay[];
  // Counts that do not depend on the period, except `new`/`active` which do.
  users: { total: number; new: number; active: number; withAddress: number };
  couriers: { total: number; active: number; online: number };
  admins: { total: number; owners: number; managers: number };
  menu: { dishes: number; available: number; categories: number };
}

// ---- Table booking ----

/** A wall or zone the owner drew so the plan is recognisable. */
export interface FloorShape {
  kind: "wall" | "area";
  label: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

/** One bookable table on the plan. Coordinates are in plan units. */
export interface FloorTable {
  id: string;
  number: string;
  seats: number;
  shape: "rect" | "circle";
  x: number;
  y: number;
  w: number;
  h: number;
  isActive: boolean;
  note: string;
}

export interface BookingSettings {
  enabled: boolean;
  width: number;
  height: number;
  slotMinutes: number;
  maxDaysAhead: number;
  minNoticeMinutes: number;
  maxGuests: number;
  shapes: FloorShape[];
  tables: FloorTable[];
  note: string;
}

export type ReservationStatus =
  | "pending"
  | "confirmed"
  | "seated"
  | "done"
  | "cancelled";

export interface Reservation {
  id: string;
  number: string;
  tableId: string;
  tableNumber: string;
  userId?: string;
  customer: { name: string; phone: string };
  guests: number;
  at: string;
  endsAt: string;
  status: ReservationStatus;
  comment: string;
  cancelReason?: string;
  statusHistory: { status: ReservationStatus; at: string }[];
  createdAt: string;
  updatedAt: string;
}

/** The plan plus which tables are taken for the requested moment. */
export interface BookingPlan {
  booking: BookingSettings;
  at: string;
  endsAt: string;
  busy: { tableId: string; at: string; endsAt: string }[];
  /** Whose room this plan is — bookings are per branch. */
  branchId?: string;
  branchName?: string;
}

/** What the panel polls to know something new arrived (drives the sound). */
export interface AdminAlerts {
  orders: { pending: number; newestAt: string | null };
  reservations: { pending: number; newestAt: string | null };
}

// ---- Brands and branches ----
//
// A company may run several brands (a restaurant and a samsa chain), each with
// one or more branches. A single restaurant has exactly one of each and never
// sees the difference — see the decision record in PROGRESS.md.

export interface BrandFeatures {
  delivery: boolean;
  pickup: boolean;
  dineIn: boolean;
  booking: boolean;
}

export interface Brand {
  id: string;
  name: string;
  slug: string;
  description: string;
  logoUrl: string;
  coverUrl: string;
  content?: SiteContent;
  theme?: SiteTheme;
  features: BrandFeatures;
  sortOrder: number;
  isActive: boolean;
}

export interface Branch {
  id: string;
  brandId: string;
  name: string;
  /** Short code printed in front of this branch's order numbers. */
  code?: string;
  /** Menu item ids that have run out here today. */
  soldOut?: string[];
  phones: string[];
  address: GeoPoint;
  workingHours: WorkingHour[];
  delivery: DeliverySettings;
  booking?: BookingSettings;
  prepMinutes: number;
  /** How close (metres) staff must be to this address to clock in or out.
   *  0 disables the check. */
  staffRadiusM?: number;
  /** Clocking in also needs a code scanned from the branch screen. */
  requireKioskCode?: boolean;
  /** Bumped when the kiosk key is rotated; revokes every screen token. */
  kioskVersion?: number;
  sortOrder: number;
  isActive: boolean;
}

export interface BrandsResponse {
  brands: Brand[];
  branches: Branch[];
}

// ---- Staff attendance (/staff + the admin's staff and payroll screens) ----

/** One weekday of the roster the admin wrote down. Everything the dashboard
 *  says about "too much" or "too little" is measured against this. */
export interface StaffSchedule {
  day: number; // 0=Sunday .. 6=Saturday
  start: string; // "11:00"
  end: string; // "23:00"
  isOff: boolean;
}

export type StaffPayMode = "hourly" | "shift" | "monthly";
export type StaffPayPeriod = "daily" | "10days" | "15days" | "monthly";

export interface Staff {
  id: string;
  branchId?: string;
  name: string;
  phone: string;
  username: string;
  /** Free text: "oshpaz", "ofitsiant", "kassir". */
  position: string;
  schedule: StaffSchedule[];
  payMode: StaffPayMode;
  hourlyRate: number;
  shiftRate: number;
  monthlyRate: number;
  payPeriod: StaffPayPeriod;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

/** Where the employee stood when they pressed the button. */
export interface ShiftPunch {
  lat: number;
  lng: number;
  accuracy: number;
  /** Distance from the branch, in metres, measured at punch time. */
  meters: number;
  at: string;
}

export interface Shift {
  id: string;
  staffId: string;
  branchId?: string;
  /** The working day this shift is filed under, local YYYY-MM-DD. */
  date: string;
  in: string;
  out?: string | null;
  inAt?: ShiftPunch;
  outAt?: ShiftPunch;
  minutes: number;
  /** Set when an admin wrote or corrected this by hand. */
  editedBy?: string;
  note?: string;
}

export interface StaffSession {
  id: string;
  in: string;
  out?: string | null;
  minutes: number;
  open: boolean;
  editedBy?: string;
  note?: string;
}

/** What a day of the calendar is. `status` is derived, never stored. */
export type StaffDayStatus =
  | "off"
  | "absent"
  | "under"
  | "ok"
  | "over"
  | "extra"
  | "open"
  | "upcoming";

export interface StaffDay {
  date: string;
  weekday: number;
  /** Minutes the roster asks for; 0 on a day off. */
  expected: number;
  worked: number;
  /** Signed: positive is overtime. */
  diff: number;
  sessions: StaffSession[];
  /** First clock-in and last clock-out, "HH:MM" or empty. */
  first: string;
  last: string;
  planStart: string;
  planEnd: string;
  open: boolean;
  status: StaffDayStatus;
  pay: number;
}

export interface StaffTotals {
  days: number;
  absent: number;
  expected: number;
  worked: number;
  diff: number;
  overtime: number;
  shortage: number;
  pay: number;
}

/** A window against the one before it. `hasPrev` is false when the previous
 *  window was empty — a percentage against nothing means nothing. */
export interface StaffTrend {
  current: number;
  previous: number;
  percent: number;
  hasPrev: boolean;
}

export interface StaffReport {
  days: StaffDay[];
  totals: StaffTotals;
  from: string;
  to: string;
  today: StaffTrend;
  week: StaffTrend;
  month: StaffTrend;
  periodFrom: string;
  periodTo: string;
  periodPay: number;
  periodPaid: number;
  periodDue: number;
  payMode: StaffPayMode;
  payPeriod: StaffPayPeriod;
  rate: number;
}

/** Where the employee is expected to be, and how close the branch insists on. */
export interface StaffWorkplace {
  branchId: string;
  name: string;
  address: GeoPoint;
  radiusM: number;
}

export interface StaffMe {
  staff: Staff;
  workplace?: StaffWorkplace;
  openShift?: Shift | null;
}

/** One line of the admin staff list: the account plus how the period went. */
export interface StaffRow extends Staff {
  branchName: string;
  totals: StaffTotals;
  onShift: boolean;
  todayIn: string;
  todayOut: string;
  todayWorked: number;
  todayExpected: number;
  todayStatus: StaffDayStatus;
  periodFrom: string;
  periodTo: string;
  periodPay: number;
  periodPaid: number;
  periodDue: number;
}

/** Salary actually handed over — a ledger entry, not a counter reset. */
export interface StaffPayment {
  id: string;
  staffId: string;
  branchId?: string;
  amount: number;
  /** The window this payment settles. */
  from: string;
  to: string;
  paidBy: string;
  note?: string;
  at: string;
}

export interface AdminStaffDetail {
  staff: Staff;
  branchName: string;
  report: StaffReport;
  payments: StaffPayment[];
  paidTotal: number;
}

export interface PayrollRow {
  staffId: string;
  name: string;
  position: string;
  branchName: string;
  payMode: StaffPayMode;
  payPeriod: StaffPayPeriod;
  rate: number;
  isActive: boolean;
  from: string;
  to: string;
  days: number;
  worked: number;
  expected: number;
  earned: number;
  paid: number;
  due: number;
}

export interface PayrollResponse {
  rows: PayrollRow[];
  earned: number;
  paid: number;
  due: number;
}

/** What the branch screen polls: the code to show, and how long it lasts. */
export interface KioskCode {
  code: string;
  /** Seconds until this code expires — the screen refreshes exactly then. */
  expiresIn: number;
  stepSecs: number;
  branchName: string;
  branchId: string;
}

/** Issued once per screen from the admin panel. */
export interface KioskToken {
  token: string;
  branchId: string;
  branchName: string;
  version: number;
}

// ---- Call centre ----

/** What came of a call. Stored as ids and translated in the panel, so renaming
 *  a label never rewrites the log. */
export type CallOutcome =
  | "order"
  | "booking"
  | "info"
  | "complaint"
  | "callback"
  | "refused"
  | "missed"
  | "spam";

export interface Call {
  id: string;
  branchId?: string;
  /** "in" — the guest rang; "out" — the operator rang them. */
  direction: "in" | "out";
  phone: string;
  userId?: string;
  customerName: string;
  outcome: CallOutcome;
  note: string;
  orderId?: string;
  orderNumber?: string;
  reservationId?: string;
  reservationNumber?: string;
  /** The promise: ring this person back at this time. */
  callbackAt?: string;
  callbackDone: boolean;
  operatorId?: string;
  operatorName: string;
  seconds: number;

  // ---- What the phone system said, when there is one ----
  /** "manual" (an operator typed it) or "pbx" (the exchange announced it). */
  source?: string;
  /** The exchange's own call id — what makes five webhooks land on one row. */
  pbxCallId?: string;
  /** Which internal extension took it. */
  extension?: string;
  /** A recording exists. The link is fetched on demand: onlinePBX signs its
   *  download URLs and a stored one quietly stops working. */
  hasRecording?: boolean;
  hangupCause?: string;
  /** Live right now — this is what makes the caller's card open by itself. */
  ringing?: boolean;
  ringingAt?: string;
  answeredAt?: string;
  createdAt: string;
  updatedAt: string;
}

/** One past order, trimmed to what an operator reads out loud. */
export interface CallerOrder {
  id: string;
  number: string;
  status: OrderStatus;
  type: "delivery" | "pickup" | "dinein";
  total: number;
  items: OrderItem[];
  address: OrderAddress;
  courierName?: string;
  cancelReason?: string;
  createdAt: string;
}

/** A dish this customer keeps ordering — what makes "the usual?" possible. */
export interface CallerFavourite {
  menuItemId: string;
  name: string;
  qty: number;
  times: number;
}

/** GET /admin/lookup?phone= — everything about a caller in one answer. */
export interface CallerLookup {
  phone: string;
  /** Null for a first-time caller: a normal case, not an error. */
  user: SiteUser | null;
  ordersCount: number;
  ordersTotal: number;
  avgOrder: number;
  lastOrderAt: string | null;
  segments: CustomerSegment[];
  /** Still being cooked or carried — what most calls are about. */
  activeOrders: CallerOrder[];
  recentOrders: CallerOrder[];
  favourites: CallerFavourite[];
  reservations: Reservation[];
  /** Complaints nobody has answered. Put in front of the operator before
   *  they speak. */
  openComplaints: Feedback[];
  recentCalls: Call[];
}

// ---- Telephony (onlinePBX) ----

/** Admin view of the phone settings. The API key is never returned. */
export interface PBXSettings {
  provider: string;
  enabled: boolean;
  domain: string;
  defaultExtension: string;
  hasApiKey: boolean;
  /** The path to paste into onlinePBX. It carries the secret, so it is shown
   *  rather than hidden — it cannot be configured without being visible. */
  webhookPath: string;
  lastCheckAt?: string;
  lastCheckOk: boolean;
  lastCheck: string;
  /** When an event last arrived. The line that distinguishes "credentials are
   *  right" from "the webhook URL was actually pasted in". */
  lastEventAt?: string;
}

export interface PBXSettingsInput {
  enabled: boolean;
  domain: string;
  apiKey?: string;
  defaultExtension: string;
  rotateToken?: boolean;
}

/** GET /admin/calls/live — is a call ringing for me right now? */
export interface LiveCall {
  ringing: boolean;
  call?: Call;
  extension?: string;
}

export interface CallOperatorStat {
  operatorId: string;
  operatorName: string;
  calls: number;
  orders: number;
}

/** GET /admin/calls/stats — the window defaults to today. */
export interface CallStats {
  total: number;
  incoming: number;
  outgoing: number;
  byOutcome: Partial<Record<CallOutcome, number>>;
  orders: number;
  /** Percent of calls that produced an order. */
  conversion: number;
  /** Counted over the whole log, not the window: a promise made last week is
   *  still owed today. */
  callbacksOpen: number;
  callbacksOverdue: number;
  byOperator: CallOperatorStat[];
}

// ---- Online payment ----

export type PaymentStatus = "unpaid" | "pending" | "paid" | "refunded";

/** What the checkout may offer: cash, plus every provider that is switched on
 *  *and* fully credentialed. A button that leads to a bank error page costs
 *  the order, and the guest blames the restaurant. */
export interface PaymentMethodsResponse {
  methods: PaymentMethod[];
}

/** What POST /orders answers with: the order, plus the bank link when one is
 *  needed. */
export type CreatedOrder = Order & { payUrl?: string };

/** GET /orders/{number}/pay */
export interface PayLink {
  url: string;
  provider?: PaymentMethod;
  status?: PaymentStatus;
  reason?: string;
}

/** One provider transaction against one order — the ledger, not a status
 *  field: "the guest says they paid twice" is only answerable from it. */
export interface Payment {
  id: string;
  orderId: string;
  orderNumber: string;
  provider: PaymentMethod;
  providerTxnId: string;
  amount: number;
  /** 1 created, 2 paid, -1 cancelled before paying, -2 cancelled after. */
  state: number;
  reason?: number;
  createdAt: string;
  updatedAt: string;
}

/** Admin view of the credentials. The secrets themselves are never returned —
 *  only whether each one is stored. */
export interface PaymentSettings {
  returnUrl: string;
  payme: {
    enabled: boolean;
    merchantId: string;
    testMode: boolean;
    accountField: string;
    hasKey: boolean;
    hasTestKey: boolean;
  };
  click: {
    enabled: boolean;
    serviceId: string;
    merchantId: string;
    merchantUserId: string;
    hasSecretKey: boolean;
  };
  uzum: {
    enabled: boolean;
    serviceId: string;
    login: string;
    accountField: string;
    hasPassword: boolean;
  };
  atmos: {
    enabled: boolean;
    storeId: string;
    baseUrl: string;
    /** Three secrets, three flags. They fail apart: a wrong OAuth pair means
     *  no checkout link at all, while a wrong callback key means the link
     *  works, the guest pays, and the confirmation is refused. */
    hasConsumerKey: boolean;
    hasConsumerSecret: boolean;
    hasApiKey: boolean;
  };
}

/** What the settings form sends. Every secret is "empty = keep the stored
 *  one": the form cannot show the key it is editing, so a blank field must
 *  never be read as "erase it". */
export interface PaymentSettingsInput {
  returnUrl: string;
  payme: {
    enabled: boolean;
    merchantId: string;
    key?: string;
    testKey?: string;
    testMode: boolean;
    accountField: string;
  };
  click: {
    enabled: boolean;
    serviceId: string;
    merchantId: string;
    merchantUserId: string;
    secretKey?: string;
  };
  uzum: {
    enabled: boolean;
    serviceId: string;
    login: string;
    password?: string;
    accountField: string;
  };
  atmos: {
    enabled: boolean;
    storeId: string;
    baseUrl: string;
    consumerKey?: string;
    consumerSecret?: string;
    apiKey?: string;
  };
}

// ---- SMS gateway (login codes) ----

/** Which gateway sends the one-time codes. Chosen and paid for by the
 *  restaurant itself, not by the platform. */
export type SMSProvider =
  | "demo"
  | "eskiz"
  | "playmobile"
  | "getsms"
  | "onesignal";

export interface SMSSettings {
  /** Numbers allowed to see a demo login code in the API response, so an owner
   *  can test signing in before a gateway contract exists. Returned in full:
   *  these are the owner's own numbers, and the point is being able to see
   *  which ones currently skip SMS. */
  testPhones?: string[];
  provider: SMSProvider;
  /** Every id the panel may offer, in the order the server wants them shown. */
  providers: SMSProvider[];
  from: string;
  /** What is *actually* sending, which is not always what was chosen: a
   *  half-filled provider silently falls back to demo. */
  active: SMSProvider;
  demo: boolean;
  /** Names the credential still missing, or "" when the gateway is ready. */
  missing: string;
  /** True while the credentials still come from the server's environment
   *  rather than from this page. */
  fromEnv: boolean;
  eskiz: { email: string; baseUrl: string; hasPassword: boolean };
  playmobile: { url: string; login: string; hasPassword: boolean };
  getsms: {
    url: string;
    login: string;
    nickname: string;
    hasPassword: boolean;
  };
  onesignal: {
    appId: string;
    from: string;
    baseUrl: string;
    hasApiKey: boolean;
  };
  lastTestAt: string;
  lastTestOk: boolean;
  lastTest: string;
  lastTestPhone: string;
}

/** Same rule as the payment keys: an empty password means "keep the stored
 *  one", never "erase it". */
export interface SMSSettingsInput {
  /** Up to five numbers that may see a demo code. Anything unparseable is
   *  dropped server-side. */
  testPhones?: string[];
  provider: SMSProvider;
  from: string;
  eskiz: { email: string; password?: string; baseUrl: string };
  playmobile: { url: string; login: string; password?: string };
  getsms: {
    url: string;
    login: string;
    password?: string;
    nickname: string;
  };
  onesignal: {
    appId: string;
    apiKey?: string;
    from: string;
    baseUrl: string;
  };
}

// ---- POS integration (iiko / Clopos / r_keeper) ----

export type POSProvider = "" | "iiko" | "syrve" | "clopos" | "poster" | "rkeeper";

/** One sellable thing in the till, for the mapping screen. */
export interface POSProduct {
  id: string;
  name: string;
  category: string;
  price: number;
  /** Stopped, archived or out of stock over there. Shown but flagged —
   *  mapping a dish to something the kitchen has stopped is the mistake worth
   *  catching at mapping time. */
  unavailable: boolean;
}

/** What each of our dishes points at in one branch's till. */
export interface POSMapping {
  id: string;
  branchId: string;
  menuItemId: string;
  posProductId: string;
  posProductName: string;
}

/** Admin view of a branch's till connection. Secrets are never returned —
 *  only whether each one is stored. */
export interface POSSettings {
  branchId: string;
  provider: POSProvider;
  enabled: boolean;
  /** Send the order by itself the moment it is confirmed. */
  autoSend: boolean;
  iiko: {
    organizationId: string;
    terminalGroup: string;
    orderTypeId: string;
    paymentTypeId: string;
    baseUrl: string;
    hasApiLogin: boolean;
  };
  /** Syrve is iiko's international edition — same fields, stored apart so
   *  switching between them never mixes credentials. */
  syrve: {
    organizationId: string;
    terminalGroup: string;
    orderTypeId: string;
    paymentTypeId: string;
    baseUrl: string;
    hasApiLogin: boolean;
  };
  poster: {
    spotId: number;
    baseUrl: string;
    hasToken: boolean;
  };
  clopos: {
    clientId: string;
    brand: string;
    integratorId: string;
    venueId: number;
    saleTypeId: number;
    baseUrl: string;
    hasSecret: boolean;
  };
  rkeeper: {
    url: string;
    login: string;
    station: string;
    anchor: string;
    hasPassword: boolean;
    hasToken: boolean;
  };
  /** What the last connection check said, so the screen can show it without
   *  dialling the till on every page load. */
  lastCheckAt?: string;
  lastCheckOk: boolean;
  lastCheck: string;
}

/** What the settings form sends. Secrets are "empty = keep the stored one". */
export interface POSSettingsInput {
  provider: POSProvider;
  enabled: boolean;
  autoSend: boolean;
  iiko: {
    apiLogin?: string;
    organizationId: string;
    terminalGroup: string;
    orderTypeId: string;
    paymentTypeId: string;
    baseUrl: string;
  };
  syrve: {
    apiLogin?: string;
    organizationId: string;
    terminalGroup: string;
    orderTypeId: string;
    paymentTypeId: string;
    baseUrl: string;
  };
  poster: {
    token?: string;
    spotId: number;
    baseUrl: string;
  };
  clopos: {
    clientId: string;
    clientSecret?: string;
    brand: string;
    integratorId: string;
    venueId: number;
    saleTypeId: number;
    baseUrl: string;
  };
  rkeeper: {
    url: string;
    login: string;
    password?: string;
    station: string;
    anchor: string;
    token?: string;
  };
}

/** What happened when an order was pushed to the till. */
export interface OrderPOS {
  provider: string;
  /** "" not sent | "sent" | "failed" | "pending" */
  status: string;
  posOrderId?: string;
  note?: string;
  error?: string;
  attempts: number;
  sentAt?: string;
}


/** One ticket on the kitchen screen.
 *
 *  The waiting time arrives from the server rather than being computed here: a
 *  tablet at a pass often has a wrong clock, and the number this screen is
 *  judged by must not depend on it. */
export interface KitchenTicket {
  id: string;
  number: string;
  status: OrderStatus;
  type: string;
  tableNumber?: string;
  items: OrderItem[];
  comment?: string;
  queuedAt: string;
  waitingMin: number;
}


// ---- Campaigns: one message to one segment ----

/** A segment with both numbers: how many are in it, and how many can actually
 *  be messaged. The gap (opted out, no phone) is what an owner needs before
 *  writing anything — a badge saying 24 next to a send of 19 is a screen people
 *  stop believing. */
export interface SegmentRow {
  segment: string;
  total: number;
  reachable: number;
  optedOut: number;
  noPhone: number;
}

export interface CampaignPreview {
  recipients: number;
  optedOut: number;
  noPhone: number;
  /** SMS parts per message. Non-Latin text is 70 characters per part, not 160,
   *  which surprises everybody the first time — and it is billed per part. */
  parts: number;
  /** parts × recipients: what the gateway will charge for. */
  messages: number;
  /** No real gateway configured. The send is refused in this state rather than
   *  reporting success to nobody. */
  demo: boolean;
  provider: string;
}

export interface Campaign {
  id: string;
  segment: string;
  text: string;
  status: "sending" | "done" | "failed";
  total: number;
  optedOut: number;
  noPhone: number;
  sent: number;
  failed: number;
  error?: string;
  parts: number;
  provider: string;
  createdBy: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}
