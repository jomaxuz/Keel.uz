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

/** One card of the home page strip: an icon name and two lines of copy. */
export interface PerkCard {
  icon: string;
  title: LocalizedText;
  text: LocalizedText;
}

export interface SiteContent {
  aboutTitle: LocalizedText;
  aboutText: LocalizedText;
  footerNote: LocalizedText;
  tagline: LocalizedText;
  /** Empty = the built-in three cards, which is what every site showed before
   *  this became editable. */
  perks?: PerkCard[];
  /** ⚠️ "hide", not "show": the zero value must be the page as it is today. */
  hidePerks?: boolean;
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

/** ---- RFM ---- */

export interface RfmScale {
  /** Cut points at the 20/40/60/80th percentile of this restaurant's own base.
   *  Relative, never absolute: "spent over a million" means one thing in a
   *  samsa shop and another in a banquet hall, and rots as prices rise. */
  recencyDays: [number, number, number, number];
  frequency: [number, number, number, number];
  monetary: [number, number, number, number];
  /** How many customers the cuts rest on. Shown beside them, like the "sold on
   *  2 days" column in ABC/XYZ: a percentile over eleven people is
   *  arithmetically fine and worth saying out loud. */
  base: number;
}

export interface RfmCellRow {
  /** "champions", "atRisk"… The campaign audience id is this with an `rfm:`
   *  prefix — the rule segments already contain `lost` and mean something
   *  different by it. */
  cell: string;
  count: number;
  revenue: number;
  avgR: number;
  avgF: number;
  avgM: number;
}

export interface RfmResponse {
  /** False when the base is too small to rank. The panel says so and names the
   *  minimum, rather than drawing an empty grid — which would read as a
   *  restaurant with no customers, a much worse piece of news. */
  available: boolean;
  minBase?: number;
  base?: number;
  scale?: RfmScale;
  total?: number;
  revenue?: number;
  cells: RfmCellRow[];
}

/** ---- Dashboard layout (per admin) ---- */

export interface DashboardPrefs {
  /** Every tile the panel can draw, in its default order. Comes from the
   *  server so a panel one version behind still knows what it may send. */
  all: string[];
  /** Tile ids this admin switched off. Empty means today's dashboard: the
   *  list names what is **off**, never what is on, so a tile added in a later
   *  version appears for everybody without anyone opting in. */
  hidden: string[];
  /** Partial order: ids this list names go first, the rest keep their default
   *  positions behind them. */
  order: string[];
  /** What to draw, resolved by the server — order and hiding already applied. */
  visible: string[];
}

/** ---- Sales over time ---- */

export type SalesGroup = "day" | "week" | "month";

export interface SalesBucket {
  /** Sortable: "2026-08-14" for a day or week, "2026-08" for a month. */
  key: string;
  /** Printable: "14.08". Separate from `key` because one string cannot be both
   *  sortable and readable without one of the two jobs being done badly. */
  label: string;
  orders: number;
  cancelled: number;
  /** Orders whose money is in hand, and what it came to. */
  paid: number;
  revenue: number;
  /** Revenue ÷ paid — divided by the orders the money came from, not by every
   *  order placed. The other divisor makes the average fall as the kitchen
   *  gets busier. */
  avgCheck: number;
  /** Placed, not cancelled, not yet collected. Real money, not takings. */
  pending: number;
  discounts: number;
  deliveryFee: number;
  delivery: number;
  pickup: number;
  dineIn: number;
  /** Dishes sold — on the "sold" basis, so only a cancellation un-sells one. */
  items: number;
}

export type SalesTotals = Omit<SalesBucket, "key" | "label">;

export interface SalesCompare {
  from: string;
  to: string;
  previous: SalesTotals;
  /** Percent change per figure. A figure is **absent** when the previous
   *  period was zero: "up 100%" from nothing is not information. */
  percent: Partial<Record<keyof SalesTotals, number>>;
}

export interface SalesHour {
  hour: number;
  orders: number;
  revenue: number;
}

export interface SalesReportResponse {
  from: string;
  to: string;
  group: SalesGroup;
  note: string;
  buckets: SalesBucket[];
  totals: SalesTotals;
  byHour: SalesHour[];
  /** The period's best bucket by takings, or null when nothing was collected. */
  best: SalesBucket | null;
  /** Absent for an open-ended period: "all time" has nothing before it. */
  compare?: SalesCompare;
}

/** ---- Channel analytics ---- */

export interface ChannelRow {
  /** Stable id ("web", "telegram", "operator", "unknown", "delivery",
   *  "pickup", "dinein"); the panel translates it. */
  key: string;
  orders: number;
  revenue: number;
  paid: number;
  avgCheck: number;
  cancelled: number;
  /** Share of the period's orders, 0–100, within this cut. */
  share: number;
  customers: number;
  /** Customers whose **first ever** order falls in this period. */
  newCustomers: number;
}

export interface ChannelReportResponse {
  from: string;
  to: string;
  note: string;
  /** How the order was placed. */
  channels: ChannelRow[];
  /** How it was fulfilled. Overlaps with `channels` on purpose — a guest can
   *  order pickup through the bot — so the two are never summed together. */
  types: ChannelRow[];
}

/** ---- Team reports ---- */

export interface CourierReportRow {
  id: string;
  name: string;
  delivered: number;
  earnings: number;
  carried: number;
  /** Cash collected in the period. */
  cash: number;
  /** Cash still on them **right now**: lifetime collected less lifetime
   *  settled. Deliberately not period-bounded — a debt that clears itself
   *  every month is not a debt. */
  cashInHand: number;
  /** "On the way" → "delivered", averaged. The courier's own leg only. */
  avgMinutes: number;
  /** How many orders that average rests on. Zero means the average is not a
   *  fast delivery, it is no delivery. */
  timedOrders: number;
  active: boolean;
}

export interface CourierReportResponse {
  from: string;
  to: string;
  note: string;
  couriers: CourierReportRow[];
}

export interface StaffReportRow {
  id: string;
  name: string;
  position: string;
  days: number;
  absent: number;
  /** Minutes. */
  expected: number;
  worked: number;
  overtime: number;
  shortage: number;
  /** Earned under the pay rule, and what actually left the till. Not the same
   *  question, so never the same column. */
  pay: number;
  paid: number;
}

export interface StaffReportResponse {
  from: string;
  to: string;
  note: string;
  staff: StaffReportRow[];
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
  /** Which map draws the site. Empty means 2GIS — every install that predates
   *  the setting is on 2GIS. */
  mapProvider?: string;
  /** One key per provider, so switching back and forth never hands one provider
   *  the other's key — which fails as a blank map and nothing else. */
  mapYandexKey?: string;
  mapGoogleKey?: string;
  /** Search-console verification tokens. Public by nature: a token's whole job
   *  is to sit in the page head where a crawler reads it. */
  seo?: SeoSettings;
  content?: SiteContent;
  theme?: SiteTheme;
  // Table booking: the hand-drawn floor plan and the rules around it. Absent on
  // documents written before booking existed.
  booking?: BookingSettings;
  /** Ordering for a later time. Branch-owned like the hours and the zones; the
   *  server lays the serving branch's copy over this document, so the site
   *  reads one picture. Absent on documents written before it existed, which
   *  reads as off — exactly what those restaurants do today. */
  preorder?: PreorderSettings;
  /** Whether guests' ratings and comments appear on the public site. */
  reviews?: ReviewSettings;
  loyalty?: LoyaltySettings;
  updatedAt: string;
}

/** Whether the guests' own words appear on the site.
 *
 *  One switch, deliberately — there is no "only 4 stars and above" knob, and
 *  there should not be: a rule like that turns the section into a wall of
 *  praise the restaurant assembled about itself, which readers discount the
 *  moment they notice. Which reviews appear is chosen one at a time on the
 *  feedback screen, where the owner is looking at the actual words. */
export interface ReviewSettings {
  enabled: boolean;
  /** Show the average and how many ratings it is from. A separate claim from
   *  the comments: a statistic nobody is quoted in. */
  showAverage: boolean;
}

/** "Order now, for later".
 *
 *  `leadMinutes` is the whole feature in one number: how long before the
 *  requested time the kitchen is told. Until then the order is stored and
 *  silent — it is not on the pass, and it has not rung the bell. */
export interface PreorderSettings {
  enabled: boolean;
  /** How long before it is due the panel chimes and the ticket appears. */
  leadMinutes: number;
  /** The earliest a guest may ask for, from now. Separate from the lead: one
   *  is what the kitchen needs, the other what the restaurant will promise. */
  minMinutes: number;
  /** How far ahead a slot may be picked, in days. */
  maxDays: number;
  /** The granularity offered: 30 means half-hour slots. */
  slotMinutes: number;
}

/** The restaurant's own Telegram bot.
 *
 *  The token is never returned — only whether one is stored, the same rule the
 *  payment keys and the SMS password follow. `miniAppUrl` is built server-side
 *  from the username the check button found: a link assembled from a hand-typed
 *  bot name opens somebody else's bot. */
export interface TelegramSettings {
  enabled: boolean;
  botUsername: string;
  hasToken: boolean;
  lastCheckAt: string;
  lastCheckOk: boolean;
  lastCheck: string;
  /** When the bot was last pointed back at this site. */
  webhookAt?: string;
  /** ⚠️ When something last **arrived** from Telegram. The check button proves
   *  we can reach Telegram; only this shows that Telegram can reach us, which is
   *  the difference between a connected bot and a bot that answers. */
  lastUpdateAt?: string;
  /** Empty until a successful check. */
  miniAppUrl?: string;
}

// ---- Page design (the layout, as data) ----
//
// Mirrors backend/internal/models/design.go. Widths are columns of twelve, never
// pixels — see that file for why, and why every style value is an enum.

export type DesignBlock =
  | "hero"
  | "perks"
  | "categories"
  | "menu-grid"
  | "search"
  | "hours-address"
  | "reviews"
  | "about"
  | "gallery"
  | "cta"
  | "banners"
  | "navbar"
  | "footer"
  | "canvas"
  | "popup";

/** Where a freely placed element sits, in percent of its band. Never pixels —
 *  see models/design.go for why that is the decision the whole feature rests on. */
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
  /** The phone layout, drawn separately. Absent means "stack in drawn order". */
  mobile?: DesignBox | null;
  hidden?: boolean;
  hiddenMobile?: boolean;
  text?: { uz: string; ru: string; en: string };
  /** A second line: a stat's label, a quote's attribution. */
  subtext?: { uz: string; ru: string; en: string };
  image?: string;
  /** `carousel`: the photographs, in order. */
  images?: string[];
  link?: string;
  /** `icon`: which one, from a fixed set. */
  icon?: string;
  /** `rating`: how many stars are filled. */
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
    /** "" theme radius · md · lg · full (a circle). */
    radius?: string;
  };
  binding?: { categories?: string[]; popularOnly?: boolean; limit?: number };
}

export interface DesignCanvas {
  height?: number;
  heightMobile?: number;
  background?: string;
  /** A photograph behind the whole band (uploads only). */
  image?: string;
  backgroundOpacity?: number;
  elements?: DesignElement[];
}

export interface DesignSection {
  type: DesignBlock;
  variant?: string;
  /** 1–12. On a phone every band is 12 — that is the whole responsive rule. */
  span: number;
  hidden?: boolean;
  style?: {
    tone?: "" | "surface" | "raised" | "charcoal" | "brand";
    padding?: "" | "sm" | "md" | "lg";
    align?: "" | "left" | "center";
    rounded?: boolean;
  };
  /** Only on `canvas` and `popup`: what was drawn inside. */
  canvas?: DesignCanvas | null;
  /** Typed settings, declared by the schema the console draws its panel from.
   *  A bag rather than named fields: the renderer reads the keys it knows and an
   *  unknown key is inert, so a newer console cannot break an older site. */
  settings?: Record<string, unknown>;
  /** Repeatable items inside the section: slides, photos, links. */
  blocks?: {
    type: string;
    settings?: Record<string, unknown>;
    hidden?: boolean;
  }[];
  binding?: {
    categories?: string[];
    popularOnly?: boolean;
    limit?: number;
  };
}

export interface PageDesign {
  id: string;
  brandId: string;
  status: "draft" | "published";
  sections: DesignSection[];
  /** The designer's own corrections. Refused outright by the backend if it
   *  contains anything that could close a `<style>` element — see sanitizeCSS. */
  customCss?: string;
  /** Which console operator drew it. Also the record that this customer has a
   *  paid design — the fee is settled outside the platform. */
  drawnBy?: string;
  publishedAt?: string;
  updatedAt: string;
}

export interface Banner {
  id: string;
  imageUrl: string;
  /** One of the site's own pages, or a dish. Never an arbitrary URL — see
   *  handlers/banners.go. */
  link?: string;
  title?: { uz: string; ru: string; en: string };
  sortOrder: number;
  isActive: boolean;
  startsAt?: string | null;
  endsAt?: string | null;
}

export interface Vacancy {
  id: string;
  /** How many people applied. ⚠️ Admin only — never sent to the site: "42 already
   *  applied" is either a reason not to bother or a claim about how desperate the
   *  restaurant is, and neither is the applicant's to read. */
  applications?: number;
  title: { uz: string; ru: string; en: string };
  description?: { uz: string; ru: string; en: string };
  salary?: string;
  employment?: string;
  sortOrder: number;
  isActive: boolean;
}

export interface JobApplication {
  id: string;
  vacancyId: string;
  vacancyTitle: string;
  name: string;
  phone: string;
  comment?: string;
  status: "new" | "called" | "hired" | "refused";
  note?: string;
  createdAt: string;
}

export interface RestaurantResponse {
  // Address, phones, hours, delivery and floor plan are the serving branch's;
  // currency and socials stay the company's. See the backend GetRestaurant.
  restaurant: Restaurant;
  isOpenNow: boolean;
  /** The strip the restaurant edits itself. Already filtered to what should show now. */
  banners?: Banner[];
  /** The branch this answer was given for. */
  branch?: Branch;
  /** The brand whose face the site is wearing. */
  brand?: Brand;
  /** The layout drawn in the Keel console, when there is one. Absent means the
   *  site renders the template it always did — see DesignRenderer. */
  design?: PageDesign;
  /** A console-drawn design is live, so the panel's own theme editor is locked.
   *  Answered even for `?raw=1`, because that is what the settings page asks
   *  for and it is the page that has to lock. */
  designLocked?: boolean;
  /** Guests' ratings and words, when the restaurant has switched them on.
   *  Absent — not empty — when the feature is off, so the section draws
   *  nothing rather than a heading over a blank. */
  reviews?: PublicReviews;
}

/** What the site may show of the guests' feedback.
 *
 *  A narrow shape on purpose: the stored record also carries the phone number,
 *  the order number and the complaint handling, and none of that belongs on a
 *  public page. The server builds this by hand so a field added to the model
 *  later cannot arrive here by itself. */
export interface PublicReviews {
  items: { name: string; rating: number; comment: string; at: string }[];
  /** Averaged over **every** rating, not only the published ones — an average
   *  of the reviews somebody chose to show is not an average of anything. */
  average: number;
  count: number;
  showAverage: boolean;
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
  /** ИКПУ — the state product classifier code, for the fiscal receipt.
   *
   *  Optional and often absent: it comes from the restaurant's own accountant,
   *  and an absent code is sent as no field at all rather than as a guess — a
   *  wrong ИКПУ is a receipt filed against the wrong product. */
  ikpu?: string;
  /** The packaging code that goes on the receipt beside the ИКПУ.
   *
   *  Belongs to the classifier code rather than to the dish, so the server
   *  clears it whenever the ИКПУ is cleared — a packaging with nothing to be a
   *  packaging *of* is a number that looks filled in and refers to nothing. */
  packageCode?: string;
  /** VAT rate for this dish, overriding the branch's.
   *
   *  ⚠️ Nullable, and 0 is a real value: zero-rated and "not filled in" are
   *  different answers that produce different receipts. Use `== null` here,
   *  never a truthiness check — `if (item.vatPercent)` reads an explicit 0 as
   *  unset and silently puts the branch's rate on an exempt dish. */
  vatPercent?: number | null;
  /** Measure unit code: 0 = piece, 10 = gram, 11 = kilogram, 22 = metre,
   *  41 = litre. Zero is "piece", which is what a portion is, so almost every
   *  dish leaves this alone. */
  unitCode?: number;
  /** Dishes the owner picked to suggest alongside this one, in their order.
   *  Empty means "work it out from the order history" — which is the normal
   *  state, and why the automatic half exists. */
  recommendedIds?: string[];
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
  /** Which kitchen is cooking it. Set on every order the multi-branch
   *  migration has seen; the panel can hand an order to another one. */
  branchId?: string;
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
  /** Which door the order came in through: "web", "telegram" or "operator".
   *  Absent on orders placed before this existed. */
  channel?: string;
  /** Where the money stands. Cash orders stay "unpaid" for their whole life
   *  and that is not a problem; an online order starts "pending" and only the
   *  provider's own callback moves it to "paid". */
  paymentStatus?: PaymentStatus;
  paidAt?: string;
  /** When the order became the kitchen's problem: placed, for cash; paid, for
   *  an online order — and, on a pre-order, its scheduled time minus the
   *  branch's lead, which is why it can be in the future. */
  queuedAt?: string;
  /** The time the guest asked for. Absent on an ordinary order, which is
   *  "now" and always has been. */
  scheduledAt?: string;
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
  /** Set on a pre-order: the time the guest asked for. Without it the page
   *  says "accepted" for hours and reads as an order nobody looked at. */
  scheduledAt?: string;
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
  /** Set once the account is joined to the bot — by opening the mini app, or by
   *  sharing the number in the chat. Nothing else can reach them through Telegram. */
  telegramId?: number;
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
  /** Whether this one is shown on the public site.
   *
   *  ⚠️ Per review, and off by default: everything in this list was written to
   *  the restaurant, not to the internet — a guest answering "how was your
   *  order?", with their name on it. The settings switch opens the section;
   *  this decides what goes in it. */
  isPublic?: boolean;
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
  "off" | "scheduled" | "expired" | "usedUp" | "idle" | "running";

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
  /** Dishes the serving branch has run out of, by name.
   *
   *  ⚠️ Known only once the branch is: the guest browsed one branch's menu, and
   *  a delivery may be taken by another one entirely. The order would be
   *  refused for these anyway — saying so here turns a wasted checkout into
   *  information the guest can act on. */
  soldOut?: string[];
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
/** A group of tables that behave the same way.
 *
 *  ⚠️ **The reason this exists is that "table 112" is not a table.** A takeaway
 *  counter numbers its orders 100–130, and those numbers are not seats a guest
 *  can reserve — but the till needs them, because a takeaway order has to be
 *  opened against something. */
export interface TableZone {
  id: string;
  name: string;
  /** Whether a guest can reserve here. The hall yes, the takeaway counter no. */
  bookable: boolean;
  /** "map" follows the drawn coordinates, "list" shows numbers in a grid.
   *  ⚠️ Empty means map — every zone that exists today is the drawn plan. */
  layout?: "map" | "list";
  sort: number;
}

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
  /** ⚠️ Empty means the default zone, which is bookable — every table drawn
   *  before zones existed has no id here. */
  zoneId?: string;
}

export interface BookingSettings {
  enabled: boolean;
  /** The parts of the business tables belong to. Empty on a restaurant that
   *  has not split them, which reads as one unnamed bookable zone. */
  zones: TableZone[];
  width: number;
  height: number;
  slotMinutes: number;
  maxDaysAhead: number;
  minNoticeMinutes: number;
  maxGuests: number;
  /** Whether guests choose their own table, or only ask for a time.
   *
   *  ⚠️ Hides the choice, not the bookkeeping: with the plan hidden the server
   *  still assigns a real table (the smallest free one that fits), so
   *  double-booking stays impossible and the panel's map keeps working.
   *
   *  "hide" rather than "show" so the default is what every restaurant already
   *  has — a `showPlan` field would have switched table picking off for all of
   *  them on the day it shipped. */
  hidePlan?: boolean;
  shapes: FloorShape[];
  tables: FloorTable[];
  note: string;
}

export type ReservationStatus =
  "pending" | "confirmed" | "seated" | "done" | "cancelled";

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

/** One kitchen's queue, for the panel's load strip.
 *
 *  ⚠️ Shown, never acted on automatically. Routing new orders away from a busy
 *  branch moves the food further from the guest — ten minutes saved at the
 *  stove come back as twenty on the road — and it punishes the quick branch by
 *  handing it everyone's work. Whether to move an order at seven on a Friday
 *  depends on how many couriers are out, which no ticket count knows. */
export interface BranchLoad {
  branchId: string;
  name: string;
  /** Accepted and being cooked: exactly what the pass is showing. */
  cooking: number;
  /** Placed and not yet accepted by anybody. */
  pending: number;
  /** How long the oldest waiting ticket has waited. The number that actually
   *  says "behind" — five fresh tickets is a normal evening, one that has sat
   *  forty minutes is not. */
  oldestMin: number;
}

/** What the panel polls to know something new arrived (drives the sound). */
export interface AdminAlerts {
  /** Orders wanted **now**. A pre-order is not one of these until its lead
   *  time arrives, at which point it becomes `preorders.dueAt`. */
  orders: { pending: number; newestAt: string | null };
  /** Two separate events, because they ask for different things: a pre-order
   *  arriving is news ("buy the meat"), a pre-order falling due is an
   *  instruction ("start cooking") that arrives hours later with nobody having
   *  touched anything. One timestamp cannot carry both. */
  preorders?: {
    /** Still ahead of the restaurant — for the badge, not the sound. */
    upcoming: number;
    /** A new pre-order was placed. */
    newestAt: string | null;
    /** One has just fallen due. */
    dueAt: string | null;
    /** Due, accepted, and nobody has started cooking it — the panel keeps
     *  ringing while this is above zero. `confirmed` only: a due pre-order
     *  still unaccepted is already in `orders.pending`, and one order must
     *  not raise two alarms with two sounds and one button. */
    dueWaiting?: number;
  };
  reservations: { pending: number; newestAt: string | null };
  /** Orders handed to the till that the till says nobody there has accepted.
   *
   *  ⚠️ No timestamp, because this one never rings: the act that clears it is
   *  on the POS, not in this panel, and an alarm with no button here is one
   *  people learn to ignore. A count and a banner, nothing more. */
  pos?: {
    unaccepted: number;
    /** How long an order must have waited to be counted, for the wording. */
    afterMins: number;
    /** Orders that never reached the till at all — usually one dish with no
     *  mapping. Worse than `unaccepted`: there the ticket is at least on
     *  somebody's screen, here the kitchen has nothing. */
    failed?: number;
    /** Sellable dishes the till has no id for. The cause rather than the
     *  symptom: this is what makes the next order fail, and it is knowable
     *  now. Combos and unavailable dishes are excluded — neither can break an
     *  order, and a warning nobody can clear is one people switch off. */
    unmapped?: number;
  };
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
  /** Menu item ids that have run out here today, marked at the counter. */
  soldOut?: string[];
  /** Menu item ids the branch's till has stopped, mirrored every few minutes.
   *  A separate list because a separate writer owns it: the counter cannot lift
   *  one of these, only the POS can. */
  posSoldOut?: string[];
  /** When the till was last read. A time rather than a flag — "synced" says
   *  nothing about an hour ago. */
  posSoldOutAt?: string;
  posSoldOutError?: string;
  phones: string[];
  address: GeoPoint;
  workingHours: WorkingHour[];
  delivery: DeliverySettings;
  booking?: BookingSettings;
  preorder?: PreorderSettings;
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
  /** May open the kitchen screen (/staff/kitchen).
   *
   *  ⚠️ Granted per person: having a staff login is not the same question as
   *  running the pass. The server enforces it — the hidden button is only the
   *  courtesy half. */
  canKitchen: boolean;
  /** May run the floor screen: open checks, add dishes, send them to the
   *  kitchen. */
  canWaiter: boolean;
  /** May run the till: everything a waiter may do, plus payment, voids and
   *  discounts.
   *
   *  ⚠️ Implies `canWaiter` — the server resolves that, so never test the two
   *  independently. A cashier who could take money but not add a dish would
   *  send every correction across the room. */
  canCashier: boolean;
  /** Whether a till PIN is set. ⚠️ The code itself is never returned — the
   *  panel only ever learns that one exists. */
  hasPin?: boolean;
  /** The role this person holds, and what it resolves to.
   *
   *  ⚠️ `perms` is computed on the way out and never stored: a second copy of
   *  what the role says is a second thing that can disagree with it. */
  roleId?: string;
  roleName?: string;
  perms?: string[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

/** One of the three receipt designs.
 *
 *  ⚠️ Three templates, not one with fields switched off: they are read by three
 *  different people under three different pressures. The kitchen ticket carries
 *  no prices at all, and no setting can add them. */
export interface ReceiptTemplate {
  enabled: boolean;
  /** 58 or 80. ⚠️ A setting, never a guess — 48 characters sent to a 58 mm
   *  printer cuts the end off every line, which on the guest's copy is the
   *  totals column. */
  widthMm: number;
  header: string;
  footer: string;
  /** Blank lines before the cut, so the tear-off does not take the last line
   *  of text with it. Printer-dependent. */
  feedLines: number;
  /** ⚠️ A missing key means **shown**: a template stored before a field existed
   *  has no entry for it, and reading that as "off" would silently drop a line
   *  from receipts that were printing fine. */
  fields: Record<string, boolean>;
}

export interface ReceiptSettings {
  branchId: string;
  kitchen: ReceiptTemplate;
  till: ReceiptTemplate;
  customer: ReceiptTemplate;
  /** The printers this branch has. ⚠️ Nil on a branch that has never had one —
   *  the same JSON trap the plan's slices carry, so read it with `?? []`. */
  printers?: Printer[];
}

/** The preview, as lines of monospace text — rendered by the same code that
 *  drives the printer, so it cannot show something the paper will not. */
export interface ReceiptPreview {
  kitchen: string[];
  till: string[];
  customer: string[];
}

/** A job title and the permissions that come with it.
 *
 *  ⚠️ **A record with an id, not a typed word.** CLAUDE.md's warning that a job
 *  title must never be read as a permission stands — a role is the answer to
 *  it, not an exception: it is picked from a list, and the spelling of its name
 *  changes nothing. */
export interface StaffRole {
  id: string;
  name: string;
  perms: string[];
  /** Shipped with the product. Editable and deletable anyway — a role that
   *  cannot be changed is a role that gets worked around by giving somebody
   *  the wrong one. */
  seeded: boolean;
  sort: number;
  /** How many people hold it. The number an owner needs before widening it. */
  staffCount: number;
}

/** One permission, named by the server so the panel cannot offer a switch the
 *  server does not understand. */
export interface PermOption {
  id: string;
  name: string;
}

/** Who is standing at the till right now.
 *
 *  ⚠️ Deliberately narrow — a name and two permissions. This is drawn on a
 *  screen in a public room, and the staff record behind it carries a salary, a
 *  rota and a phone number. */
export interface TillPerson {
  id: string;
  name: string;
  position?: string;
  canWaiter: boolean;
  canCashier: boolean;
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
  "off" | "absent" | "under" | "ok" | "over" | "extra" | "open" | "upcoming";

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
export interface CallerSuggestion {
  menuItemId: string;
  name: string;
  /** So the operator can quote it without leaving this screen. */
  price: number;
}

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
  /** What to offer them, built from what they usually order. The one channel
   *  where upselling has to be a sentence somebody says rather than a card
   *  somebody sees. */
  suggest: CallerSuggestion[];
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
  "demo" | "eskiz" | "playmobile" | "getsms" | "onesignal";

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
  /** The login-code wording actually in use, with `{code}` for the digits —
   *  the stored one, or the built-in when nothing was set. */
  codeTemplate: string;
  /** The built-in wording, for the "reset" affordance. */
  defaultCodeTemplate: string;
  /** The placeholder the template must contain. */
  codePlaceholder: string;
  /** What one code costs, priced the way the gateway does. ⚠️ The cliff is
   *  invisible: one Cyrillic letter or `oʻ` takes the message out of GSM-7 and
   *  cuts the limit from 160 characters to 70, so a politely-lengthened
   *  template can quietly double the cost of every login. */
  codeParts: number;
  codeGsm7: boolean;
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
  /** The last time a **real** guest could not be sent a code, and why.
   *
   *  ⚠️ A different question from `lastTest`, and the more important one: the
   *  test button says "it worked when I checked", this says "somebody could
   *  not get in". Nobody watches for that on their own — the person it fails
   *  for is a stranger who simply leaves. */
  lastErrorAt?: string;
  lastError?: string;
}

/** Same rule as the payment keys: an empty password means "keep the stored
 *  one", never "erase it". */
export interface SMSSettingsInput {
  /** Up to five numbers that may see a demo code. Anything unparseable is
   *  dropped server-side. */
  testPhones?: string[];
  provider: SMSProvider;
  from: string;
  /** Empty keeps the built-in Uzbek wording. Must contain `{code}` — the
   *  server refuses a template without it, because a message with no code in
   *  it is a failure that produces no error anywhere. */
  codeTemplate?: string;
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

export type POSProvider =
  "" | "iiko" | "syrve" | "clopos" | "poster" | "rkeeper";

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

// ---- Stop list ----

/** One dish on the stop-list screen, with why it is off sale. */
export interface StopListItem {
  menuItemId: string;
  name: string;
  categoryId: string;
  category: string;
  imageUrl: string;
  price: number;
  /** Off the menu entirely, everywhere — not a stop list matter, but the owner
   *  looking for a missing dish should find it here rather than nowhere. */
  hidden: boolean;
  /** Marked by hand at this branch, for today. */
  manual: boolean;
  /** Stopped in the till. Not liftable from here. */
  pos: boolean;
  /** What it is linked to over there, when it is linked at all. */
  posProduct: string;
  mapped: boolean;
}

export interface StopList {
  branchId: string;
  branchName: string;
  items: StopListItem[];
  pos: {
    connected: boolean;
    provider: POSProvider;
    /** When the till was last read successfully or refused. */
    syncedAt?: string;
    /** Why the last read failed, in the till's own words. Empty when fine. */
    syncError: string;
    /** How often the background sync runs, in minutes. */
    everyMins: number;
    /** The branch is closed, so the background sync is paused — which is why
     *  `syncedAt` is old. Without saying so, "last read 9 hours ago" after any
     *  night reads as a broken integration, and the owner's next move is to
     *  re-enter credentials that were never wrong. */
    paused?: boolean;
    /** How many dishes are linked to a product at all — nothing can be stopped
     *  automatically until this is non-zero. */
    mappedItem: number;
  };
}

/** What each of our dishes points at in one branch's till. */
export interface POSMapping {
  id: string;
  branchId: string;
  menuItemId: string;
  posProductId: string;
  posProductName: string;
}

/** A virtual cash register from the tax committee's registry. */
export type FiscalProvider =
  "" | "multikassa" | "firstofd" | "epos" | "regos" | "hippo" | "simurg";

/** One row of the provider list the panel draws.
 *
 *  Fetched rather than hard-coded, because `ready` changes with what we have
 *  built and a stale copy in the frontend goes wrong in the direction that
 *  matters — offering a provider the server cannot dial. */
export interface FiscalProviderInfo {
  id: FiscalProvider;
  name: string;
  /** Whether an adapter exists. A provider that is not ready can still be
   *  chosen and have its credentials saved — owners set this up before the
   *  contract completes — but enabling it is refused, with the reason. */
  ready: boolean;
  /** Whether the register runs inside the restaurant rather than on the
   *  internet. Changes two visible things: the drawer asks for a LAN address
   *  instead of a login, and the connection check has to be pressed on the till
   *  screen — the owner may well be reading this page from home. */
  local: boolean;
  note: string;
}

/** What is stored for one provider. Secrets are never returned, only flagged. */
export interface FiscalCredFlags {
  login: string;
  registerId: string;
  baseUrl: string;
  hasSecret: boolean;
}

/** Admin view of a branch's fiscalisation. */
export interface FiscalSettings {
  provider: FiscalProvider;
  enabled: boolean;
  /** СТИР / ИНН — the taxpayer the receipts are filed under. */
  tin: string;
  /** ⚠️ Null means "not declared", 0 means "not a VAT payer". They are
   *  different answers and the panel has to be able to tell them apart, so
   *  never test this with truthiness. */
  vatPercent: number | null;
  creds: Record<string, FiscalCredFlags>;
  lastCheckAt?: string;
  lastCheckOk: boolean;
  lastCheck: string;
  /** When a receipt was last filed. ⚠️ The most useful line on the page: the
   *  check button proves the credentials worked when it was pressed, this
   *  answers whether sales are being registered now. */
  lastReceiptAt?: string;
  lastErrorAt?: string;
  lastError: string;
  /** When the relay on the register's PC last asked for work.
   *
   *  ⚠️ A timestamp rather than "connected", because the relay's whole failure
   *  mode is going quiet: the PC is rebooted for Windows updates and nothing on
   *  any screen changes. A saved flag would still read "connected" a week
   *  later; an hour-old timestamp says what is actually true. */
  agentSeenAt?: string;
  /** When somebody asked for the register's day to be ended, if it has not
   *  happened yet. The panel cannot do it — see FiscalDay. */
  closeDayRequestedAt?: string;
}

/** The till's history for a period: every shift that was counted, with its
 *  difference and the sentence explaining it.
 *
 *  ⚠️ The rows are the closed shifts themselves rather than a summary, because
 *  the report an owner actually reads is the list of differences — a period
 *  total of zero can hide a shortfall on Tuesday and a surplus on Thursday, and
 *  those are two separate conversations. */
export interface CashReportResponse {
  from: string;
  to: string;
  note: string;
  shifts: CashShift[];
  totals: { expected: number; counted: number; variance: number };
}

/** A till session: the float it started with, and — once counted — what the
 *  drawer held against what it should have.
 *
 *  ⚠️ **The difference is the product.** `expected` is frozen at closing time
 *  and `variance` is stored rather than recomputed, so that changing how
 *  expected cash is worked out can never silently rewrite last month's
 *  shortfalls. */
export interface CashShift {
  id: string;
  branchId?: string;
  openedAt: string;
  openedBy: string;
  openingFloat: number;
  closedAt?: string;
  closedBy?: string;
  expected: number;
  /** Cash taken at the counter during the shift, frozen alongside `expected`.
   *
   *  ⚠️ **The only figure of ours the fiscal register can honestly be compared
   *  against.** `expected` is the whole drawer — float, counter takings,
   *  courier handovers, manual movements — while the register only knows about
   *  cash *sales*. Setting the register beside `expected` looks like a
   *  comparison and is arithmetic nonsense. */
  counterCash: number;
  counted: number;
  /** counted − expected. Negative is a shortfall. */
  variance: number;
  /** Required whenever variance is non-zero — the server refuses it empty. */
  varianceNote?: string;
  note?: string;
  /** What the cash register totalled for the same day. A second, independent
   *  count; never used to correct `expected`. */
  fiscal?: FiscalDay;
}

/** The arithmetic behind "what should be in the drawer". */
export interface CashFigures {
  openingFloat: number;
  /** Cash taken at the counter: dine-in and pickup orders settled in cash. */
  counterCash: number;
  counterOrders: number;
  /** Cash couriers have handed back during this shift. */
  settlements: number;
  settlementCount: number;
  manualIn: number;
  manualOut: number;
  expected: number;
  /** ⚠️ Cash on deliveries that went out and were never settled — real money,
   *  in a courier's pocket, deliberately **not** in `expected`. Counting it
   *  would double every delivery. Shown because an owner looking at a short
   *  till wants this number before asking anybody difficult questions. */
  withCouriers: number;
}

/** Money put in or taken out of the till by hand. Every one carries a name and
 *  a reason: cash that moved with neither becomes an argument three weeks
 *  later, when nobody remembers. */
export interface CashEntry {
  id: string;
  shiftId: string;
  kind: "in" | "out";
  /** Free text, not an enum — every kitchen spends money on something the next
   *  one does not, and a fixed list sends all of it to "other". */
  category: string;
  amount: number;
  note?: string;
  by: string;
  at: string;
}

/** What the cash register totalled for a day, stored on the cash shift it
 *  belongs to.
 *
 *  ⚠️ **A second, independent count of the same takings** — the shift's
 *  `expected` comes from the orders we recorded, this from the machine that
 *  filed them with the state. When a drawer is short, the first useful question
 *  is which of the two the cash agrees with.
 *
 *  ⚠️ It never corrects `expected`. That figure is frozen at closing time on
 *  purpose, and a shortfall that rewrote itself when a second source arrived
 *  would be a shortfall nobody could investigate. */
export interface FiscalDay {
  /** The Z-report's sequence number — what an inspector asks for. */
  number?: string;
  saleCash: number;
  saleCard: number;
  saleTotal: number;
  saleCount: number;
  /** Kept apart from sales: a day of heavy refunds that happens to balance is
   *  a different story from a quiet one, and netting them hides it. */
  refundTotal: number;
  closedAt: string;
  /** Why the day could not be ended, when it could not. */
  error?: string;
}

/** What the panel sends back. Secrets are only present when retyped — an empty
 *  one means "keep the stored one", never "delete it". */
export interface FiscalSettingsInput {
  provider: FiscalProvider;
  enabled: boolean;
  tin: string;
  vatPercent: number | null;
  multikassa: FiscalCredsInput;
  firstofd: FiscalCredsInput;
  epos: FiscalCredsInput;
  regos: FiscalCredsInput;
  hippo: FiscalCredsInput;
  simurg: FiscalCredsInput;
}

export interface FiscalCredsInput {
  login: string;
  password?: string;
  token?: string;
  registerId: string;
  baseUrl: string;
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

/** What the till says became of an order we handed it.
 *
 * Separate from `OrderPOS.status`, which is about our handover. Poster files an
 * order perfectly and leaves it at `status: 0` until a cashier presses accept,
 * so "we sent it" and "the kitchen has it" can be an hour apart. */
export interface OrderTill {
  /** "waiting" (nobody at the till has taken it) | "accepted" | "cancelled" |
   * "unsupported" (this POS cannot be asked). Absent means never asked. */
  state?: string;
  /** The till's own wording — usually the check number once accepted. */
  raw?: string;
  /** When we last asked. A state without a time silently ages into a lie. */
  checkedAt?: string;
  acceptedAt?: string;
  /** Why the last question failed. Not an order failure. */
  error?: string;
}

/** What happened when an order was pushed to the till. */
export interface OrderPOS {
  provider: string;
  /** "" not sent | "sent" | "failed" | "pending" — our handover, not the
   * till's verdict. That is `till` below. */
  status: string;
  posOrderId?: string;
  note?: string;
  error?: string;
  attempts: number;
  sentAt?: string;
  till?: OrderTill;
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
  /** Set only when the order was placed for a particular time. A ticket that
   *  appeared just now but is wanted at 19:00 is not the same job as one
   *  somebody is standing at the counter waiting for. */
  scheduledAt?: string;
}

// ---- The till: open checks on the floor ----

/** Why a cooked dish was taken off a check. Kept on the line rather than
 *  deleting it: the food exists, and a void that leaves no trace is the oldest
 *  way to take money out of a restaurant. */
export interface CheckLineVoid {
  at: string;
  by?: string;
  reason: string;
  /** Whether the food was actually made and thrown away, as opposed to the
   *  kitchen catching it in time. */
  wasted?: boolean;
}

export interface CheckLine {
  lineId: string;
  name: string;
  price: number;
  qty: number;
  sum: number;
  options?: OrderItemOption[];
  comment?: string;
  /** Whether the kitchen has this line. The only colour distinction on the
   *  screen: what is cooking versus what is still a draft on this tablet. */
  fired: boolean;
  /** Which guest pays for it. ⚠️ Zero means the table — one bill for the
   *  party, which is how most meals end and how every check written before
   *  splitting existed reads. */
  guest?: number;
  /** Which course it goes out with. Zero means "with everything else". */
  course?: number;
  /** Present on voided lines, which stay on screen and count for nothing —
   *  hiding them makes the running total unexplainable to the guest. */
  void?: CheckLineVoid;
}

/** One check, with everything both screens need in a single response: the till
 *  is used standing up, and a second round trip to price a table is a second
 *  chance for the network to be why the queue is not moving. */
/** A booking as the till shows it: who is coming, when, and to which table.
 *
 *  ⚠️ Narrower than the panel's reservation on purpose — a waiter does not need
 *  the audit trail, and the phone number belongs to the office. */
/** One receipt printer, as the branch has it set up. */
export interface Printer {
  id: string;
  name: string;
  /** One line: tcp://192.168.1.50:9100 · usb://XP-58 · serial://COM3 ·
   *  \\PC\XP-58 · /dev/usb/lp0 */
  target: string;
  /** kitchen · till · customer · precheck. ⚠️ Empty means nothing prints —
   *  a half-configured printer must not start taking every bill. */
  kinds: string[];
  /** "latin" (Uzbek) or "cyrillic" (Russian). */
  charset?: string;
  cut?: boolean;
  fullCut?: boolean;
  drawer?: boolean;
  copies?: number;
  disabled?: boolean;
}

export interface TillReservation {
  id: string;
  number: string;
  name: string;
  at: string;
  guests: number;
  tableNumber?: string;
  status: string;
  comment?: string;
}

export interface Check {
  id: string;
  number: string;
  status: OrderStatus;
  tableId?: string;
  tableNumber?: string;
  guests?: number;
  serverId?: string;
  serverName?: string;
  openedAt: string;
  /** Computed on the server — the tablet by the till has the wrong clock as
   *  often as the one at the pass does. */
  openMin: number;
  lines: CheckLine[];
  subtotal: number;
  /** Lines typed but not yet sent to the kitchen. The single number the floor
   *  screen is read for. */
  unfired: number;
  comment?: string;
  total: number;
  /** When the table was handed its bill. ⚠️ The third state a floor screen
   *  draws: a table that has asked to pay is neither eating nor gone — it is
   *  waiting for a person with a card machine. */
  precheckAt?: string;
  closedAt?: string;
  /** The tax filing, once there is one. Carried on the check rather than
   *  fetched separately because the screen that needs it is showing the guest
   *  their QR while they stand there. */
  fiscal?: FiscalReceipt;
}

/** What the virtual cash register said about this sale.
 *
 *  ⚠️ `pending` is written before the register is even asked, so a filing whose
 *  answer never came back is a visible unfinished one rather than a sale that
 *  looks as though it was never meant to have a receipt. */
export interface FiscalReceipt {
  status: "pending" | "filed" | "failed";
  provider: string;
  /** The fiscal sign and the QR the guest checks. Both come from the register;
   *  neither is ours to compute. */
  fiscalSign?: string;
  qrText?: string;
  receiptId?: string;
  filedAt?: string;
  /** The register's own words, for the cashier. Never shown to a guest. */
  error?: string;
  attempts?: number;
}

/** One call the till screen has to make on the server's behalf.
 *
 *  ⚠️ The body is **opaque** and must be sent exactly as given. The registered
 *  cash register is a program on a PC inside the restaurant with no route from
 *  our server, so this screen carries the document across the local network —
 *  it does not compose it. Editing anything here would be editing a tax
 *  document from the least trusted machine in the system. */
export interface FiscalJob {
  url: string;
  method: string;
  headers?: Record<string, string>;
  body?: string;
  timeoutMs: number;
}

/** Whether this till files receipts, and how it would. */
export interface TillFiscalStatus {
  enabled: boolean;
  provider?: string;
  name?: string;
  /** Whether the register lives on the restaurant's own network. Changes the
   *  advice when it does not answer: a local one is a cable or an address, and
   *  neither is something the server can check. */
  local: boolean;
  /** Whether a relay on the register's own PC is taking the filings right now.
   *  When it is, this screen waits for a result instead of making the call. */
  relay: boolean;
  hello?: FiscalJob;
  /** ⚠️ A timestamp rather than a flag, for the reason lastEventAt is one: "it
   *  is connected" goes stale the moment the hour moves past it. */
  lastReceiptAt?: string;
  lastErrorAt?: string;
  lastError?: string;
}

/** What the till saw when it made the call for us. */
export interface FiscalReply {
  status: number;
  body: string;
  /** Set when the call never completed — a timeout, a refused connection, a
   *  browser that blocked it. Kept apart from a rejection because only one of
   *  the two is about the receipt. */
  networkError?: string;
}

/** What a counter can be paid with. Deliberately not the website's list: the
 *  bank redirects are a checkout flow the guest drives on their own phone. */
export type TillPaymentMethod = "cash" | "card" | "transfer";

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
  /** Which channel these numbers are about. */
  channel?: string;
  /** Telegram: excluded because they have never opened the bot. A different
   *  problem from "no phone number", with a different fix. */
  noTelegram?: number;
  /** Push: excluded because no browser of theirs is subscribed. A third
   *  problem again, with a third fix — ask them on the site, where they
   *  already are. Kept apart from the other two for exactly that reason. */
  noPush?: number;
  /** Telegram costs nothing per message. Stated so the cost line reads "free"
   *  rather than blank, which looks like "unknown". */
  free?: boolean;
  /** Telegram: whether a bot is actually connected. */
  ready?: boolean;
  /** Why a single named customer cannot be messaged — opted out, no number, no
   *  bot link. The reason is the useful part while somebody is still composing. */
  blocked?: string;
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
  /** "sms" or "telegram". Absent on campaigns sent before the choice existed. */
  channel?: string;
  /** Telegram only: the photograph that went with the message. */
  image?: string;
  noTelegram?: number;
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
