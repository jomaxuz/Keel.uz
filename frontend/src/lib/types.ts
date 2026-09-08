// Types mirror the Go backend JSON models (backend/internal/models/models.go).

import { hasId } from "@/lib/id";

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
  /** What the portions sold cost the kitchen, and what was left of the
   *  revenue.
   *
   *  ⚠️ **Absent unless somebody typed a cost for this dish.** Zero would read
   *  as "free", which would make the worst margin on the menu look like the
   *  best — and this is a screen people act on. */
  cost?: number;
  margin?: number;
  costed?: boolean;
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
  /** How this branch's scales lay out a printed barcode.
   *
   *  ⚠️ **A branch field, laid over the company document by the settings
   *  screen** — the same shape as `preorder` and the hours above it. It belongs
   *  to the branch because a scale is a physical object in a room: two shops of
   *  one brand can have been set up by two different installers, and a layout
   *  read from the wrong one charges for a quantity nobody weighed. */
  scale?: ScaleLabel;
  /** Whether marked goods are scanned when they arrive, not only when they
   *  sell.
   *
   *  ⚠️ **Off by default, and that is not caution for its own sake.** A check
   *  that started refusing codes nobody had ever received would refuse every
   *  sale of every marked bottle on the day it shipped — at a counter, with a
   *  customer waiting. What it buys is where the refusal lands: switched on, the
   *  same fact turns up in the store room with the box still open. */
  markingInbound?: boolean;
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
  /** Where this restaurant's own notifications go.
   *
   *  ⚠️ A group or a channel, not somebody's chat: an owner's own chat is one
   *  person, one phone and one holiday away from nobody reading any of it. Two
   *  of them, because the suspicious-events one names employees and the
   *  feedback one does not — a floor manager can be given the second without
   *  the first. Group ids are negative, so 0 means "not set". */
  alertChatId?: number;
  feedbackChatId?: number;
  /** Which language both groups are written in.
   *
   *  ⚠️ A third language setting, and not one too many: the panel's reader is
   *  whoever logged in, the receipt's is the guest, and this one's is whoever
   *  the owner added to a group — frequently somebody who will never log in at
   *  all. Empty is Uzbek. */
  notifyLang?: string;
  notifyLangs?: string[];
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

/** Where a banner is shown. "" is the site — see models/banner.go. */
export type BannerPlacement = "site" | "till";

export interface Banner {
  id: string;
  /** Absent on every banner saved before the till lock screen existed, which is
   *  read as "site" everywhere. */
  placement?: BannerPlacement;
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
  /** What this choice alone takes out of the store, per portion.
   *
   *  ⚠️ **A bar sells one bottle in three measures.** A vodka poured at 40, 50
   *  and 100 ml is one dish with a "Hajm" group; until this existed all three
   *  took the dish's own recipe out of the store, so the price varied and the
   *  stock did not. Empty on every choice that is only a price. */
  recipe?: RecipeLine[];
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
  /** What a portion costs the kitchen. ⚠️ Panel only — the public menu API
   *  never carries it, and zero means "not known" rather than "free". */
  cost?: number;
  /** The tech card. ⚠️ When it is not empty the dish is costed from it and the
   *  typed `cost` is ignored: two sources for one number drift, silently. */
  recipe?: RecipeLine[];
  /** What the card works out to at today's ingredient prices. Read-only —
   *  recomputed on every read, never posted back. */
  recipeCost?: number;
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
  /** Whether this product carries a national marking code (Asl Belgisi):
   *  bottled water, soft drinks.
   *
   *  ⚠️ Absent means no, which is every menu written before this existed. The
   *  flag is on the product, which answers the bar's question by itself: a
   *  bottle sold whole is marked, the same drink poured into a glass is a
   *  different menu item and is not. */
  marked?: boolean;
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
  /** The barcode printed on the packet, as the scanner reads it.
   *
   *  ⚠️ **A shop's counter begins here** — it is scanned, never tapped. Empty on
   *  every dish, which is the ordinary state: a portion of osh has no barcode
   *  and never will. Unique within a brand, not globally: two brands under one
   *  owner may genuinely stock the same EAN. */
  barcode?: string;
  /** The axes this model varies along: ["O'lcham", "Rang"].
   *
   *  ⚠️ **Non-empty makes this row the model rather than a thing on the
   *  shelf.** A shirt in five sizes has one row to hang the photograph, the
   *  name and the category on; what is counted, scanned and carried out of the
   *  shop is always a size. The server refuses to sell a row with axes — see
   *  `menuLine` — and the screens must not offer it either. */
  variantAxes?: string[];
  /** The model this is one variant of. Empty on an ordinary product. */
  variantOf?: string;
  /** What distinguishes this one: ["M", "Qora"], in the model's axis order.
   *
   *  ⚠️ **Values, not a name.** "M / Qora" as a string reads the same and
   *  cannot be filtered, grouped or regenerated — and "every black one" is the
   *  first thing a clothes shop asks for. */
  variant?: string[];
  /** Whether this is the object that was purchased, rather than something made
   *  from purchased things.
   *
   *  ⚠️ **The one field the whole shop/restaurant difference reduces to.** A
   *  kitchen turns inputs into outputs, so what is sold and what is stocked are
   *  two documents with a tech card between them; a shop sells the object it
   *  bought, so the server keeps the stock row behind this product in step. The
   *  stock module never learns this field exists — the product still consumes
   *  through a tech card, written by the server, saying "one unit of itself". */
  sellsItself?: boolean;
  /** Parts of one portion this dish may be sold in, as percents (25, 50, 75…).
   *  ⚠️ Empty means whole portions only — every menu written before this
   *  existed, and the safe reading: a missing value taken as "divisible" would
   *  put a half-portion button on two hundred dishes nobody meant to divide. */
  portions?: number[];
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
  /** Which line of a till check this is, when the order is one. */
  lineId?: string;
  /** When the kitchen ticked **this dish**, and when the waiter put it in
   *  front of the guest.
   *
   *  ⚠️ Timestamps rather than flags: every screen prints them as an age
   *  ("5 daq oldin tayyor"), and "ready two minutes ago" and "ready twenty
   *  minutes ago" are the difference between a plate to collect and a plate to
   *  apologise for. */
  readyAt?: string;
  servedAt?: string;
  /** Part of one portion, as a percent (50 = half). Absent is a whole one —
   *  which is what every line was before parts existed. */
  portion?: number;
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
  /** Whether this guest may leave without paying — the till's debt method.
   *
   *  ⚠️ Off by default and **owner only**. Writing a debt is a cashier's act;
   *  deciding who may owe cannot also be, or the evening's shortfall goes onto
   *  a regular's slate and the drawer still counts correct. Enforced by the
   *  server when the check is closed, not by the screen. */
  creditAllowed?: boolean;
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
    /** What this customer still owes, in so'm, and over how many checks.
     *  ⚠️ Money first: "three debts" is not a figure anybody chases. */
    debtTotal: number;
    debtCount: number;
  };
}

/** One unpaid check somebody took away on the slate. */
export interface DebtRow {
  orderId: string;
  number: string;
  at: string;
  total: number;
  /** What was said at the counter ("paying on Friday"). */
  note?: string;
  table?: string;
  /** Who let it go on the slate — the point of the record. */
  by?: string;
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
/** ⚠️ `"monthly"` is a fixed wage, the way a cook is paid — the deliveries then
 *  earn nothing on their own. A mode rather than a field beside the per-order
 *  rate: two figures would leave "which of these do we owe him" to whoever is
 *  reading the screen, and both answers look right. */
export type CourierPayoutMode =
  "deliveryFee" | "perOrder" | "percent" | "monthly";

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
  /** Used only when `payoutMode` is "monthly". */
  monthlyRate?: number;
  /** How often this courier is settled up — the same four windows the kitchen
   *  uses. Empty reads as monthly. */
  payPeriod?: StaffPayPeriod;
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
  /** Wages handed to this courier, newest first.
   *
   *  ⚠️ **The opposite of a settlement**, and the two sit on the same screen:
   *  a settlement is the courier handing *our* cash back, a payment is us
   *  handing *them* their wage. */
  payments?: CourierPayment[];
  /** Everything ever paid to them. Summed on the server from the ledger —
   *  never kept as a total, which drifts the first time a row is corrected. */
  paid?: number;
}

/** One wage handed to a courier. */
export interface CourierPayment {
  id: string;
  courierId: string;
  amount: number;
  /** The window it settles, "YYYY-MM-DD". */
  from: string;
  to: string;
  paidBy?: string;
  note?: string;
  at: string;
}

/** What a panel account may be.
 *
 *  ⚠️ `"operator"` is a call-centre account: orders, reservations and the call
 *  desk, and nothing else. A real `admin_user` — unlike `"stock"`, which is a
 *  **staff** account signed in through the same form (a storekeeper reaching
 *  the counting, recipe and purchase screens that live in the panel).
 *
 *  Both are refused everything else — by the server first (handlers/panelgate.go)
 *  and by the navigation second. The server's list is the one that matters; the
 *  navigation exists so the panel does not draw twenty links that all answer
 *  forbidden, which reads as a broken account rather than as a boundary. */
export type PanelRole = "owner" | "manager" | "operator" | "stock";

export interface AdminUser {
  id: string;
  username: string;
  role: PanelRole;
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
  /** Of `pending`, what guests took away on the slate — owed, not in flight. */
  debt: number;
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
  /** A name from lib/floorColors, never a colour. ⚠️ It reaches an SVG `fill`
   *  on the public booking page, so the server clamps it to the same closed
   *  list on save. Empty is the default, which is what every shape drawn before
   *  this existed has. */
  color?: string;
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
  /** Drawn in this colour on the booking page and the panel's plan.
   *
   *  ⚠️ **The till ignores it.** Over there colour means state — free, sitting,
   *  billed, waited too long — and a decorative colour in that language would
   *  make the one screen scanned across a room during a rush unreadable. */
  color?: string;
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
  /** Receipts the queue gave up on in the last twelve hours.
   *
   *  ⚠️ Silent, like `pos`, and for a sharper reason: the clearing act is at
   *  the printer — paper, a plug — and an alarm nobody in this app can stop is
   *  one people learn to ignore. Bounded to tonight, because a printer that
   *  has been off for a week would show a number that cannot be brought back
   *  to zero, which is read the same way. */
  print?: { failed: number };
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
  /** What this brand sells and how it is rung up.
   *
   *  ⚠️ **Empty is a restaurant**, which is every brand written before the
   *  field existed. The console sets it when the brand is created and nothing
   *  overwrites it afterwards — see `models/businesstype.go`. */
  businessType?: BusinessType;
  sortOrder: number;
  isActive: boolean;
}

/** The kinds of business Keel is set up for. Kept as a string union rather than
 *  an enum so an unrecognised value from a newer console is still assignable —
 *  and reads as a restaurant, which is what `known()` does on the server. */
export type BusinessType =
  | ""
  | "fastfood"
  // Makers: a counter that cooks. They compose what they sell and seat nobody.
  | "coffee"
  | "bakery"
  | "pastry"
  // Shops: what was delivered is what is sold.
  | "grocery"
  | "butcher"
  | "clothing"
  | "cosmetics"
  | "flowers"
  | "pharmacy"
  | "hardware";

/** Everything this build knows how to tailor for. ⚠️ The empty string is a
 *  restaurant, deliberately: it is what every brand written before the field
 *  existed carries. */
const BUSINESS_TYPES: readonly string[] = [
  "",
  "fastfood",
  "coffee",
  "bakery",
  "pastry",
  "grocery",
  "butcher",
  "clothing",
  "cosmetics",
  "flowers",
  "pharmacy",
  "hardware",
];

/** Collapses anything this build does not recognise to a restaurant.
 *
 *  ⚠️ **Every predicate below goes through this, and the bug is why.** They were
 *  written as `!t` and `t === "..."`, each falling back on its own — so a brand
 *  written by a newer console and read by this panel answered *no* to all of
 *  them at once: no tables, no kitchen, no tech cards, and none of the shop
 *  screens either. Not "the tailoring is missing", but the emptiest sidebar the
 *  panel can draw, on the newest customer we have. The comments underneath each
 *  one already promised this behaviour; nothing implemented it.
 *
 *  ⚠️ Mirrors `BusinessType.known()` in Go, which carries the same note for the
 *  same reason — the server was fixed and the panel was not. */
function known(brand?: { businessType?: string } | null): string {
  const t = brand?.businessType ?? "";
  return BUSINESS_TYPES.includes(t) ? t : "";
}

/** Does this brand sell the thing it bought, rather than cook with it?
 *
 *  ⚠️ **The one question the whole shop/restaurant difference reduces to**, and
 *  the mirror of `BusinessType.SellsGoods` in Go. A kitchen turns inputs into
 *  outputs, so what is sold and what is stocked are two documents with a tech
 *  card between them; a shop sells the object it purchased, so they are one.
 *
 *  ⚠️ **Anything unrecognised is a restaurant.** A panel that met a business
 *  type from a newer console must not start offering barcodes to a kitchen. */
export function hasTables(brand?: { businessType?: string } | null): boolean {
  // ⚠️ Only a restaurant seats people. A fast food takes orders at a counter
  // and a shop has no room at all — and both were being offered a floor plan,
  // a booking list and a table QR code they can never use.
  return known(brand) === "";
}

/** Does anything here get cooked to order?
 *
 *  ⚠️ **A tech card is the test.** A kitchen turns inputs into outputs, so a
 *  dish needs a recipe and a prep workshop needs batches. A shop sells the
 *  object it bought: the server keeps its one-line card in step, and a screen
 *  inviting somebody to edit that card by hand is a screen that can only break
 *  it.
 *
 *  ⚠️ **Cooked *to order* is the test, not "has an oven".** A bakery and a
 *  pastry shop bake in batches before the doors open — their document is a
 *  production batch, not a ticket somebody watches — so a kitchen screen there
 *  would sit empty all day beside the screen they actually need. A coffee house
 *  makes every cup after somebody asks for it, which is the shape of a kitchen
 *  ticket even though nothing is cooked. */
export function hasKitchen(brand?: { businessType?: string } | null): boolean {
  const t = known(brand);
  return t === "" || t === "fastfood" || t === "coffee";
}

/** Is what this brand sells assembled from other things it stocks?
 *
 *  ⚠️ **Not the same question as `hasKitchen`, and reading it as the same one
 *  was wrong for exactly one business.** A florist cooks nothing and has no
 *  kitchen, and a bouquet is fifteen stems, a wrap and a ribbon — the most
 *  literal technical card there is. Asking the kitchen question took the cards
 *  away from flower shops, which is the screen that tells them what a bouquet
 *  costs and what a bad week threw out. Mirrors `BusinessType.Composes`. */
export function composes(brand?: { businessType?: string } | null): boolean {
  const t = known(brand);
  // ⚠️ **The florist's lesson, arriving three more times.** Bread is flour,
  // water and salt; a cappuccino is a shot and milk; a cake is six lines and a
  // box. Take the cards away from these and each loses the one screen that says
  // what its own counter costs it.
  return (
    t === "" ||
    t === "fastfood" ||
    t === "flowers" ||
    t === "bakery" ||
    t === "coffee" ||
    t === "pastry"
  );
}

/** Does this brand sell one product in sizes and colours?
 *
 *  ⚠️ **A clothes shop, and only that, by default.** Sizes are the whole shape
 *  of a boutique's catalogue and are meaningless in a pharmacy — a box of
 *  paracetamol has no colour. A grocery does have pack sizes, but they are
 *  separate products with separate barcodes and separate prices, which is what
 *  it already enters them as; offering a matrix generator there is offering a
 *  tool that would double its catalogue by accident.
 *
 *  ⚠️ A default, not a rule — data wins, as everywhere on these screens: a
 *  product that already has variants keeps its editor whatever kind of business
 *  this is. */
export function hasVariants(brand?: { businessType?: string } | null): boolean {
  return known(brand) === "clothing";
}

export function sellsGoods(brand?: { businessType?: string } | null): boolean {
  switch (known(brand)) {
    case "grocery":
    case "butcher":
    case "clothing":
    case "cosmetics":
    case "flowers":
    case "pharmacy":
    case "hardware":
      return true;
    default:
      return false;
  }
}

/** How a branch's scales lay out a printed barcode.
 *
 *  ⚠️ **The one piece of a shop's counter that gets money wrong in silence.** A
 *  scale prints a label whose barcode carries the weight or the price inside it,
 *  and every vendor lays those digits out differently. Read the layout wrong and
 *  the counter still beeps, still shows a product and still prints a receipt —
 *  with the wrong quantity on it. Nobody notices until a stocktake.
 *
 *  ⚠️ **Weight and price are not interchangeable.** A scale set to print price
 *  sends money; read as grams it becomes a quantity in the thousands, which is
 *  the failure that empties a shelf on paper. */
export interface ScaleLabel {
  /** ⚠️ Off by default: a prefix that matched an ordinary EAN would turn a
   *  normal product into a weighed one, in every shop that has no scales. */
  enabled: boolean;
  /** "2" almost everywhere — GS1 reserves the 2x prefixes for in-store and
   *  variable-measure items, which is why a retail product never starts with
   *  one. */
  prefix?: string;
  itemLen?: number;
  valueLen?: number;
  /** "weight" (grams) or "price" (whole som). */
  value?: string;
  /** The serial port a counter scale is wired to, when there is one.
   *
   *  ⚠️ **Separate from the label settings above and not implied by them.** A
   *  shop can have a labelling scale in the back and no scale at the counter, or
   *  the other way round — reading one setting as the other would put a "weigh"
   *  button in front of a cashier with nothing to weigh on. */
  port?: string;
}

export interface Branch {
  id: string;
  brandId: string;
  name: string;
  /** Short code printed in front of this branch's order numbers. */
  code?: string;
  /** Which branch holds this one's stock, when a chain has a central store.
   *
   *  ⚠️ **Empty is "its own store room"**, which is every restaurant that has
   *  ever installed this: a request for something already in the building is
   *  then an errand between two rooms and moves no stock. Pointing at another
   *  branch makes the same request a van — the sending shelf loses what was
   *  loaded and this one gains what was counted off it. Never itself. */
  supplyBranchId?: string;
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
  /** ⚠️ On the branch because a scale is a physical object in a room: two shops
   *  of one brand can have been set up by two different installers. */
  scale?: ScaleLabel;
  /** Whether marked goods are scanned when they arrive, not only when they
   *  sell.
   *
   *  ⚠️ **Off by default, and that is not caution for its own sake.** A check
   *  that started refusing codes nobody had ever received would refuse every
   *  sale of every marked bottle on the day it shipped — at a counter, with a
   *  customer waiting. What it buys is where the refusal lands: switched on, the
   *  same fact turns up in the store room with the box still open. */
  markingInbound?: boolean;
  /** What this room adds to a table's bill. ⚠️ Tables only — the till applies
   *  it, because the setting cannot tell a table from a takeaway coffee. */
  service?: { enabled: boolean; percent: number };
  prepMinutes: number;
  /** How close (metres) staff must be to this address to clock in or out.
   *  0 disables the check. */
  staffRadiusM?: number;
  /** Clocking in also needs a code scanned from the branch screen. */
  requireKioskCode?: boolean;
  /** When on, a PIN does not open the till or the floor screen unless the
   *  person is clocked in. ⚠️ Off by default: a restaurant that has never used
   *  attendance would meet this as every PIN being refused, with a queue at
   *  the counter. */
  requireShift?: boolean;
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
  /** May write a shopping list, whatever their role says.
   *
   *  ⚠️ **It only ever adds.** A role that already grants `buyorder` is not
   *  withdrawn by leaving this unticked — a switch that sometimes grants and
   *  sometimes revokes has to be read against a second document, and that
   *  reading gets done wrong on the day somebody is in a hurry.
   *
   *  ⚠️ The one permission granted per person rather than per job: who notices
   *  the sugar has run out is the barman on Tuesdays and the porter on Fridays,
   *  and inventing "Barmen who may write lists" as a second role ends in a role
   *  per person, which is what roles exist to prevent. */
  canBuyOrder?: boolean;
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
  /** The role's name in the other two languages, so a till shows the word the
   *  person standing at it reads. Empty falls back to `roleName`. */
  roleNameRu?: string;
  roleNameEn?: string;
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
  /** Which language this receipt prints in: "uz" (default), "ru" or "en".
   *
   *  ⚠️ The restaurant's choice, not the screen's: a receipt is read by a guest
   *  at a table and a cook at a pass, neither of whom is signed in to anything.
   *  Per kind, because a kitchen is not a dining room. */
  lang?: string;
  /** Which named lines print bold or at double size.
   *
   *  ⚠️ Not a font: a thermal printer has two built-in faces and a size
   *  multiplier, and nothing that could be called a typeface. Values: "bold",
   *  "big", "boldbig". */
  emphasis?: Record<string, string>;
  /** Blank lines before the header.
   *
   *  ⚠️ For the printers whose cutter eats the top of the next receipt —
   *  `feedLines` already solves the bottom, and the same machine often takes
   *  the first line of the following one. */
  topLines?: number;
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
  /** Print the restaurant's logo above the header.
   *
   *  ⚠️ Never on the kitchen ticket, whatever this says: every dot is time at
   *  the pass and paper off the roll, and a cook does not need telling which
   *  restaurant they work in. */
  logo?: boolean;
  /** Print what each guest owes if the bill is split evenly.
   *
   *  ⚠️ Its own field rather than one of `fields`, whose rule is "missing means
   *  shown" — right for a line every receipt used to print, wrong for one no
   *  receipt has ever printed. Off until somebody asks for it. */
  splitPerGuest?: boolean;
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
  /** The logo the paper will carry, when the template asks for one. Drawn by
   *  the browser at a resolution no thermal head has — the preview answers
   *  "will it be there", the test print answers "how does it come out". */
  logoUrl?: string;
}

/** One television paired to a branch.
 *
 *  ⚠️ **A row per screen**, because the module is priced per screen and because
 *  a set that leaves the building has to be unpairable on its own — blanking
 *  every television in the restaurant to deal with one is a walk round the
 *  dining room with a remote control. */
export interface TVScreen {
  id: string;
  branchId: string;
  name: string;
  /** What it shows: the playlist, the order board, or both. */
  mode: TVScreenMode;
  /** Which room this set hangs in — "zal", "peshtaxta", "terrasa". Empty means
   *  it has not been placed, and an unplaced screen plays only what is meant
   *  for every screen. Always lowercase: it is matched against a slide's
   *  `zones` by string equality, and "Zal" against "zal" is a wall that plays
   *  nothing while every field looks filled in. */
  zone?: string;
  /** When it last spoke to the server — the only honest answer to "is this
   *  screen working?". Missing on one that has never called home. */
  lastSeenAt?: string;
  appVersion?: string;
  pairedBy?: string;
  createdAt: string;
}

export type TVScreenMode = "content" | "board" | "split";

/** One item in a branch's playlist.
 *
 *  ⚠️ **Per branch, not per screen.** Two televisions in one room show the same
 *  restaurant's food; what differs between them is the order board, and that is
 *  already `TVScreen.mode`. A playlist per screen means uploading the same video
 *  four times — and the copy nobody remembers to change is the one still
 *  showing last month's promotion.
 *
 *  Targeting is done with `zones` instead: one library, one upload, and a
 *  slide may name the rooms it belongs in. Naming none means all of them. */
export interface TVSlide {
  id: string;
  branchId: string;
  kind: TVSlideKind;
  url: string;
  name: string;
  /** How long a picture stays up. Ignored for a video. */
  seconds: number;
  order: number;
  /** Off without being deleted — a seasonal offer comes back. */
  active: boolean;
  /** Where this plays. ⚠️ **Empty means every screen, not none.** That default
   *  is what keeps "one loop on all the televisions" the thing you get without
   *  pressing anything, keeps every existing slide working with no migration,
   *  and makes a forgotten zone show a slide in *more* places rather than
   *  fewer — the opposite of the per-screen-playlist design this replaced,
   *  where the list somebody forgot kept playing last month's promotion. */
  zones: string[];
  startsAt?: string;
  endsAt?: string;
  /** ⚠️ **The dates, ready to put in a `<input type="date">`.** A `time.Time`
   *  reaches the browser as UTC, so slicing the day out of it in JavaScript
   *  lands on the previous one for every restaurant east of Greenwich — the
   *  trap CLAUDE.md records against the console's billing period. The server
   *  sends the string it means. */
  startsOn?: string;
  endsOn?: string;
}

export type TVSlideKind = "image" | "video";

/** What a television is told about itself.
 *
 *  ⚠️ Deliberately narrow — a name, a branch and a mode. This is held by a
 *  device we do not control, hanging in a public room; the branch behind it
 *  carries addresses, keys and delivery settings. */
export interface TVScreenSelf {
  id: string;
  name: string;
  mode: TVScreenMode;
  branchId: string;
  branchName: string;
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
  /** Optional translations of the name; empty falls back to the Uzbek one.
   *
   *  ⚠️ The name is text the restaurant typed, not a UI string — so it cannot
   *  come from the dictionary, and a panel running in Russian would otherwise
   *  show an Uzbek word in the role picker. Read it with `contentName()`. */
  nameRu?: string;
  nameEn?: string;
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
/** What a locked till knows about itself.
 *
 *  ⚠️ Answered from the device's own token, because this is the one screen with
 *  nobody signed in to it — see StaffTillSession. */
export interface TillSession {
  /** Whether anybody at this branch has been given a code. */
  pinsUsed: boolean;
  /** The brand this monoblock belongs to, for the lock screen's own label. */
  brandName: string;
  branchName: string;
  /** The pictures the restaurant chose for this screen, in the owner's order. */
  banners: string[];
  /** The subscription countdown, or absent on almost every day. */
  subscription?: SubscriptionNotice | null;
}

/** What a screen draws in its corner when the subscription is running out.
 *
 *  ⚠️ **Absent means silence, and that is the common case.** A component that
 *  renders "everything is fine" here would put a permanent badge on a counter
 *  that has one corner to spend — and a badge that is always there is a badge
 *  nobody reads on the day it changes. */
export interface SubscriptionNotice {
  /** Whole days left. 0 on the last day, negative once it has passed. */
  days: number;
  /** ⚠️ Decided by the server, not re-derived from `days` here. Four screens
   *  draw this notice, and four copies of the same comparison is four chances
   *  for one of them to be a day out of step with the others. */
  level: "warn" | "urgent" | "expired";
  /** "YYYY-MM-DD", already local — built by the server rather than sliced off a
   *  timestamp, which in Tashkent returns the previous day. */
  until: string;
}

export interface TillPerson {
  id: string;
  name: string;
  position?: string;
  canWaiter: boolean;
  canCashier: boolean;
  /** Whether this person may retire the screen — see tillPersonView on the
   *  server. False for cashiers and waiters: taking a bound machine out of
   *  service mid-shift needs somebody who can fetch a fresh link from the
   *  panel, and that is not a thing to leave one mis-tap away. */
  canExit?: boolean;
  /** Whether this person may write the shopping list somebody is sent to the
   *  market with. ⚠️ Its own flag, never inferred from `role`: the screen
   *  decides what to draw from answers, not from the spelling of a job title. */
  canBuyOrder?: boolean;
  /** The role's own name ("Ish boshqaruvchi"), for the corner of the till.
   *
   *  ⚠️ **Never read as a permission.** It is a name a person typed, exactly
   *  like `position`, and the whole point of roles is that the spelling of one
   *  grants nothing — see models/staffrole.go. `canExit` and the two `can*`
   *  flags are the answers; this is only what to print. */
  role?: string;
  /** The same name in RU/EN; empty falls back to `role`. Read them together
   *  with `contentText()` — the word in this corner should be in the language
   *  of the screen, not of whoever typed the role. */
  roleRu?: string;
  roleEn?: string;
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
  /** Set instead of `staffId` on a courier's row.
   *
   *  ⚠️ Two fields rather than one id plus a kind flag: the pay button posts to
   *  a different endpoint for each, and both ids are valid ObjectIDs — a row
   *  with the wrong flag would pay the wrong person silently. Read with
   *  `hasId()`. */
  courierId?: string;
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
  /** The counter rails — a cashier scanning the guest's code.
   *
   *  ⚠️ **The list comes from the server, not from this file.** Which rails
   *  exist, which have adapters and which credential boxes each one issues are
   *  facts about the backend; a panel carrying its own copy goes on showing a
   *  provider after it is dropped, or asks for a credential nobody issues and
   *  has an owner invent one. Same rule as the fiscal providers list. */
  inStore: { providers: InStoreProvider[] };
  /** The marketplaces this restaurant sells through — see PaymentSettingsInput.
   *  Carries no secret: an aggregator's money arrives by bank transfer, not
   *  through an API we call. */
  aggregators?: AggregatorAccount[];
}

/** One counter rail as the settings page draws it. */
export interface InStoreProvider {
  id: string;
  name: string;
  /** "scan" — the cashier reads a code off the guest's phone.
   *  "terminal" — the amount is pushed to a bank terminal. No driver exists
   *  for either terminal yet; the rows are shown so an owner who recognises
   *  the name reads why rather than nothing. */
  kind: "scan" | "terminal";
  /** Whether an adapter exists. ⚠️ Credentials may be saved for a rail that has
   *  none — owners configure before a contract closes — but enabling is
   *  refused, server-side too: a till button that fails for every guest who
   *  presses it is worse than a missing one. */
  ready: boolean;
  note: string;
  /** Which boxes to draw, by name. */
  needs: string[];
  enabled: boolean;
  serviceId: string;
  userId: string;
  baseUrl: string;
  hasSecretKey: boolean;
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
  /** ⚠️ **Only sent by a panel that knows about the counter rails.** The server
   *  reads a missing `inStore` as "this page had no opinion" and keeps what is
   *  stored — because for the minutes after a deploy an open tab still holds
   *  the old page, and its next save would otherwise wipe working credentials
   *  silently, with the page reporting success. */
  inStore?: {
    rails: Record<
      string,
      {
        enabled: boolean;
        serviceId: string;
        userId: string;
        secretKey?: string;
        baseUrl: string;
      }
    >;
  };
  /** The marketplaces the restaurant sells through.
   *
   *  ⚠️ **A payment method, not a delivery service.** `delivery_provider`
   *  answers "who carries the food"; this answers "who holds the money". Yandex
   *  can be both at once, and one record for both would give a restaurant that
   *  merely hires the courier fleet a balance it does not have. */
  aggregators?: AggregatorAccount[];
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
  /** When that manual stop lifts itself, where a deadline was set.
   *
   *  ⚠️ **An absolute moment from the server, not "90 minutes left".** A
   *  remaining time computed there is stale before the screen draws it, and one
   *  computed here is wrong by however far the till's clock has drifted — the
   *  same clock that sends a dead-battery monoblock back to 2010. The countdown
   *  is rendered from this instant, which is the only version both machines
   *  agree on. */
  until?: string;
  /** Stopped in the till. Not liftable from here. */
  pos: boolean;
  /** Stopped because the store it is made from is empty. Not liftable from here
   *  either: the next sync would put it back within minutes, and a button that
   *  springs back with no explanation teaches the room that the panel lies. It
   *  is lifted by recording the delivery, or by counting the shelf. */
  stock: boolean;
  /** What it is linked to over there, when it is linked at all. */
  posProduct: string;
  mapped: boolean;
  /** How many of this dish the branch sells today. ⚠️ 0 means no limit, never
   *  "sell none": every dish that existed before this field has zero. */
  limit: number;
  /** How many have gone today, counted from the orders rather than from a
   *  counter — a stored tally drifts the first time a check is cancelled. */
  sold: number;
  /** Stopped because today's batch is gone. Lifted by raising the limit, not by
   *  the counter's own switch: the next sale would put it straight back. */
  limitOff: boolean;
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
  /** Stopping dishes because the store is empty — see handlers/stockstop.go. */
  stock: {
    /** ⚠️ Off unless the owner switched it on. Refusing a sale is the most
     *  expensive thing this system can do and the balance behind it is an
     *  estimate, so it is done only where somebody has said the numbers are
     *  good enough to do it on. */
    enabled: boolean;
    /** When the shelves were last worked out. ⚠️ The most useful line on the
     *  screen: a stored "on" flag goes stale the moment the clock passes it. */
    syncedAt?: string;
    everyMins: number;
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
  | ""
  | "multikassa"
  | "firstofd"
  | "epos"
  | "regos"
  | "hippo"
  | "simurg"
  /** ⚠️ Rahmat's *cloud* register. The "Multikassa" row is the program Rahmat
   *  resells for the till computer: different transport, different credentials,
   *  and sharing an id would point a cloud customer at their own office LAN. */
  | "rahmat"
  | "qpos"
  | "arca";

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
   *  internet. Changes two visible things: the address box asks for a LAN
   *  address, and the connection check has to be pressed on the till screen —
   *  the owner may well be reading this page from home.
   *
   *  ⚠️ It does **not** decide which credential boxes appear. See `needs`. */
  local: boolean;
  note: string;
  /** Which credential boxes this provider actually needs.
   *
   *  ⚠️ Sent by the server rather than inferred here, because the panel used to
   *  infer it from `local` — "a local register has no account" — which is true
   *  of Multikassa and wrong about REGOS (login and password) and E-POS
   *  (token). Nothing errored: the provider could be chosen, saved and enabled,
   *  and simply never authenticated, because the box holding its credential was
   *  not on screen. */
  needs: CredField[];
}

export type CredField =
  "login" | "password" | "registerId" | "token" | "baseUrl";

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
  /** Debts settled during this shift, and how many. ⚠️ **Part of
   *  `counterCash`, not an addition to it** — the money is already in the
   *  drawer; this only says how much of it is somebody paying off a slate, so
   *  the paper can explain why the box holds more than the shift sold. */
  debtPaid?: number;
  debtPaidCount?: number;
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
  /** One drawer per provider, keyed by id.
   *
   *  ⚠️ **A record rather than a field each, and that is a correction.** Both
   *  sides of this used to name the providers one by one — six fields here, six
   *  in the Go request, six in the `$set`. Adding a seventh meant editing four
   *  lists, and missing one is silent in the worst way: the provider appears in
   *  the panel, the owner fills in their login, it saves, and the filing code
   *  reads an empty drawer. Nobody finds out until an inspector asks for a
   *  receipt that was never sent. */
  creds: Record<string, FiscalCredsInput>;
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
  /** The marking code scanned off this bottle (Asl Belgisi), or absent.
   *
   *  ⚠️ Frozen on the line rather than read from the menu, unlike the ИКПУ: it
   *  describes the object that was handed over, and one code covers exactly one
   *  of them — which is why a marked line never merges into another. */
  markCode?: string;
  /** Which dish this is. ⚠️ Sent by the server for the till's own use — a check
   *  built offline has to be able to say what it sold, and a line that knows
   *  only its printed name cannot be re-priced or matched to the menu. */
  menuItemId?: string;
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
  /** When the kitchen ticked this dish, and when it reached the table. See
   *  `OrderItem` above — the same two facts, on the till's shape. */
  readyAt?: string;
  servedAt?: string;
  /** Part of one portion, as a percent (50 = half). Absent is a whole one. */
  portion?: number;
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
  /** Which sections of the menu this printer takes, by category id.
   *
   *  ⚠️ **Empty means every category — the opposite of `kinds` above.** A
   *  printer with no kinds chosen is half configured and must print nothing; a
   *  printer with no categories chosen is every restaurant that exists today,
   *  one kitchen printer taking all the food. */
  categories?: string[];
  /** Dishes that go here whatever their category says, and dishes that never
   *  do. ⚠️ The handful every menu has: the dessert made at the bar, the soup
   *  the grill section makes. Without them a restaurant would have to
   *  reorganise its menu to match its printers. */
  only?: string[];
  except?: string[];
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
  /** Dishes cooked and not yet carried out, and dishes already in front of the
   *  guest. ⚠️ The first is what the floor screen is read for during service:
   *  "unfired" says what the waiter has not sent, this says what is standing
   *  under the lamp waiting for them. */
  readyWaiting?: number;
  served?: number;
  comment?: string;
  /** What the room adds for service, and the rate that produced it.
   *  ⚠️ Already inside `total` — shown separately because the guest is about
   *  to be told a number out loud. */
  service?: number;
  servicePercent?: number;
  total: number;
  /** When the table was handed its bill. ⚠️ The third state a floor screen
   *  draws: a table that has asked to pay is neither eating nor gone — it is
   *  waiting for a person with a card machine. */
  precheckAt?: string;
  closedAt?: string;
  /** ---- Only on a closed check ----
   *
   *  ⚠️ Left off an open table on purpose: the payment fields there are either
   *  empty or, worse, left over from a provider QR that went up and was never
   *  paid — and a row saying "payme" while the guests are still eating is a row
   *  somebody reads as settled. */
  closedBy?: string;
  paymentMethod?: TillPaymentMethod;
  paymentStatus?: string;
  refund?: {
    at: string;
    by?: string;
    reason: string;
    amount: number;
    method?: string;
  };
  /** The tax filing, once there is one. Carried on the check rather than
   *  fetched separately because the screen that needs it is showing the guest
   *  their QR while they stand there. */
  fiscal?: FiscalReceipt;
  /** The reversal's filing, when the check was refunded. ⚠️ Beside the sale's
   *  rather than replacing it: a refunded check ends its life carrying two tax
   *  documents, and a screen saying "this filing is stuck" has to say which. */
  fiscalRefund?: FiscalReceipt;
  /** Who has this check open on another screen right now, if anybody.
   *
   *  ⚠️ Sent so the room can say so *before* somebody taps. The server refuses
   *  the edit either way, but a table that opens and then refuses every button
   *  reads as a broken till; one that says "Dilnoza is on this" reads as a
   *  colleague. Empty once the hold goes stale. */
  heldBy?: string;
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

/** What a counter can be paid with.
 *
 *  ⚠️ `debt` is not a way of paying — it is a way of not paying yet. A check
 *  closed with it is delivered and unpaid, owed by a named customer, and it
 *  becomes takings on the day the repayment is recorded.
 *
 *  ⚠️ The three provider rails are the guest's own phone: the till shows a QR,
 *  the guest pays, and the provider tells the server. A check may only be
 *  closed with one of them **after** that confirmation — see tillpay.go.
 *
 *  ⚠️ `click_pass` and `uzum_fastpay` are the same banks and the **opposite
 *  direction**: the guest opens a code in their app and the cashier scans it,
 *  and the card is charged inside one request. Separate ids on purpose — the
 *  evidence, the refund route and the answer to "who was standing there" all
 *  differ, and a report that folded them together could answer none of it. */
export type TillPaymentMethod =
  | "cash"
  | "card"
  | "transfer"
  | "debt"
  | "payme"
  | "click"
  | "uzum"
  | "click_pass"
  | "uzum_fastpay";

/** One unpaid check, as the till shows it while a guest settles up. */
export interface TillDebt {
  orderId: string;
  number: string;
  at: string;
  total: number;
  note?: string;
  table?: string;
}

/** The rails that end in a QR code and a wait, rather than in the drawer. */
export const TILL_ONLINE: TillPaymentMethod[] = ["payme", "click", "uzum"];

/** The rails where the cashier scans the guest's code and the card is charged
 *  in the same request.
 *
 *  ⚠️ Kept apart from TILL_ONLINE rather than merged into a single "not cash"
 *  list, because the dialog does something completely different for each: one
 *  draws a QR and polls, the other opens a scanner and waits for a barcode. A
 *  list that meant "some kind of card" would have to be re-split at every use. */
export const TILL_SCAN: TillPaymentMethod[] = ["click_pass", "uzum_fastpay"];

/** What came back from the bank when a code was scanned. */
export interface TillScanResult {
  status: "paid" | "pending" | "failed" | "";
  paid: boolean;
  paymentId?: string;
  cardMask?: string;
  processing?: string;
  provider?: string;
  /** ⚠️ A sentence for the cashier, not the bank's code. An expired QR is
   *  fixed by asking the guest to open the app again — a five-second fix that
   *  reads, untranslated, like the till is broken. */
  error?: string;
  total?: number;
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

/** One dining-room or counter sale as the panel's sales list shows it.
 *
 *  ⚠️ Deliberately not `Check`: that one is the till's working document, with
 *  every line, option and void on it, and this list shows a hundred rows at a
 *  time. Two shapes because they answer two questions. */
export interface CheckRow {
  id: string;
  number: string;
  /** Empty for a counter sale — the only thing that separates the two. */
  table?: string;
  guests?: number;
  server?: string;
  closedBy?: string;
  openedAt: string;
  closedAt?: string;
  /** Portions, not lines: two of one dish is two. Voided food is not in it. */
  items: number;
  subtotal: number;
  discount?: number;
  /** What the room added for service, and the rate. ⚠️ Already inside `total`
   *  — carried so the screen can explain the difference between the two. */
  service?: number;
  servicePercent?: number;
  total: number;
  paymentMethod?: string;
  fiscal?: string;
  open: boolean;
  /** Money handed back. ⚠️ The sale stays — the food was cooked and eaten. */
  refunded?: boolean;
  /** Voided before anybody paid. Shown, never counted. */
  cancelled?: boolean;
  split?: boolean;
}

/** Totals over the whole filtered set, never the page on screen. */
export interface CheckTotals {
  checks: number;
  splits: number;
  refunded: number;
  refundedCount: number;
  cancelled: number;
  open: number;
  guests: number;
  sales: number;
  discount: number;
  /** Service charged in the period. ⚠️ Inside `sales`, not beside it. */
  service: number;
  cash: number;
  card: number;
  other: number;
  avgCheck: number;
  avgGuest: number;
  hall: number;
  counter: number;
}

export interface ChecksPage {
  rows: CheckRow[];
  total: number;
  totals: CheckTotals;
}

/** One line on an opened check.
 *
 *  ⚠️ Voided lines arrive here too, with `sum: 0`. They are the most important
 *  rows on the screen — a void that leaves no trace is the oldest way to take
 *  money out of a restaurant — so they are drawn struck through, with the
 *  reason and who authorised it, rather than filtered out. */
export interface CheckLineView {
  name: string;
  qty: number;
  price: number;
  /** Zero on a voided line: it is on the bill's face and not in its total. */
  sum: number;
  options?: OrderItemOption[];
  comment?: string;
  /** Zero means the table — one bill for the party. */
  guest?: number;
  /** Zero means "with everything else". */
  course?: number;
  firedAt?: string;
  voidedBy?: string;
  voidReason?: string;
  voidedAt?: string;
  wasted?: boolean;
}

export interface CheckRefundInfo {
  at: string;
  by?: string;
  reason: string;
  amount: number;
  method?: string;
}

export interface CheckDetail extends CheckRow {
  refund?: CheckRefundInfo;
  lines: CheckLineView[];
  discounts?: OrderDiscount[];
  openedBy?: string;
  precheckAt?: string;
  fiscalError?: string;
  fiscalSign?: string;
}

/** One receipt the queue was asked to print.
 *
 *  ⚠️ `failed` is the only field worth acting on: tried to the limit and still
 *  owed. `working` means the agent has it right now. */
export interface PrintJobRow {
  id: string;
  kind: string;
  printerName?: string;
  target: string;
  number?: string;
  createdAt: string;
  doneAt?: string;
  tries?: number;
  /** The printer's own words — usually something fixable in seconds. */
  error?: string;
  failed?: boolean;
  working?: boolean;
}

/** One line of the financial report.
 *
 *  ⚠️ `kind` is the whole meaning: "in" and "out" are movements that add up to
 *  the net figure, "info" is a number an owner wants that is **not** a movement
 *  (discounts given, the cost of food sold, the gross margin), and "pending" is
 *  money that is real and has not arrived. */
export interface FinanceLine {
  label: string;
  amount: number;
  count: number;
  kind: "in" | "out" | "info" | "pending";
  /** Indented under the line above — a part of it, not a peer. */
  sub?: boolean;
}

export interface FinanceReportResponse {
  from: string;
  to: string;
  note: string;
  lines: FinanceLine[];
  totals: { in: number; out: number; net: number; pending: number };
}

/** ---- Ingredients and tech cards ---- */

/** One thing the kitchen buys.
 *
 *  ⚠️ Priced by the purchase unit — a kilo, a litre, a piece — because that is
 *  what is written on the invoice. Nobody has a price per gram written
 *  anywhere, and asking for one is asking to be given the wrong number. */
/** One store stock is kept in — the bar, the kitchen, the cellar.
 *
 *  ⚠️ **A restaurant does not have one store**, and until warehouses existed
 *  every count was one list covering all of them: the bar's shortfall and the
 *  kitchen's surplus cancelled out, and the number the owner read was the one
 *  number that could not be acted on. See models/warehouse.go. */
export interface Warehouse {
  id: string;
  name: string;
  note?: string;
  /** "production" marks a central kitchen — the only store a batch may be made
   *  in. Absent is an ordinary store, which is every store that existed before
   *  this field. */
  kind?: string;
  sort: number;
  isActive: boolean;
}

export interface Ingredient {
  id: string;
  name: string;
  /** Which store **this branch** keeps it in.
   *
   *  ⚠️ Empty is the undivided store, which is every ingredient on a restaurant
   *  that has never split one — not "filed nowhere".
   *
   *  ⚠️ A fact about the branch, not the ingredient: the catalogue belongs to
   *  the brand (the tech cards name it by id, so a chain's kitchens share one
   *  row) and the rooms belong to the branch. One field could not be both, and
   *  with two branches the second one could not count anything. */
  warehouseId?: string;
  /** "kg" | "l" | "pcs" */
  unit: string;
  /** What one kilo / litre / piece costs, in whole so'm.
   *  ⚠️ Zero on a prep item: its price is what its batch costs. */
  price: number;
  note?: string;
  updatedAt?: string;
  /** Made in-house: the card for one batch, and what the batch yields.
   *
   *  ⚠️ The yield is where a prep card is honest about evaporation — three
   *  kilos of tomatoes that boil down to two yield 2000, not 3000, and a card
   *  saying otherwise underprices every dish the sauce is in. */
  recipe?: RecipeLine[];
  output?: number;
  /** Cost per gram / millilitre / piece, resolved by the server (prep items
   *  depend on every other rate, so the browser must not recompute it). */
  rate?: number;
  /** Whether this prep item is **made in batches and kept on a shelf** rather
   *  than derived from what the dishes sold.
   *
   *  ⚠️ The central-kitchen switch: with it on the item is counted,
   *  transferred and warned about like anything bought, and a dish consumes it
   *  rather than what it was made of — because it arrived in a tub and its
   *  ingredients were never in this branch. Its inputs are taken by the
   *  production document instead. */
  batched?: boolean;
  made?: boolean;
  batchCost?: number;
  /** A prep item whose own inputs are unpriced. ⚠️ Named rather than shown as
   *  zero: zero would make every dish containing it look cheap. */
  unpriced?: boolean;
  /** Order more when there is less than this. ⚠️ Zero means "do not warn me",
   *  not "warn me at zero": a list where every line eventually turns red is a
   *  list nobody reads, so it is opt-in one ingredient at a time. */
  minQty?: number;
  /** How much is thrown away before it reaches the pot, as a percentage: peel,
   *  bone, trimmings.
   *
   *  ⚠️ **It changes no cost anywhere.** A kilo of potatoes costs a kilo
   *  whether or not a third of it is peel, so the recipe quantity stays brutto
   *  — what leaves the store — and every price on every card and report is
   *  untouched. What it buys is the second number a cook weighs: netto.
   *
   *  ⚠️ Zero means "the same weight goes in as comes out", not "unknown". Most
   *  things are exactly that — flour, salt, oil. */
  waste?: number;
  /** What should be on the shelf now.
   *
   *  ⚠️ An estimate: the last count plus deliveries, less what the cards and
   *  the write-offs account for. It drifts exactly as far as the kitchen
   *  drifts from its cards, and further the older the last count is — every
   *  screen showing it has to say so. */
  expected?: number;
  low?: boolean;
  /** Every price this ingredient has had, oldest first.
   *
   *  ⚠️ What stops a price rise from rewriting last month: a sale is costed at
   *  the price of the day it happened. An edit is recorded from today — the
   *  form cannot tell "we typed it wrong" from "beef went up". */
  history?: { price: number; at: string }[];
  /** Invented at a market by a buyer and not finished by anybody yet.
   *
   *  ⚠️ It has a name and a price and nothing else — no unit anybody chose, no
   *  minimum, no store, no card. Marked rather than hidden, because the only
   *  person who can finish it is the one reading the catalogue. Cleared by
   *  saving the row, never by time. */
  needsCare?: boolean;
  /** How the market sells it — "bog'lam", "qop", "patnis" — and how much of the
   *  stock unit one is.
   *
   *  ⚠️ **The gap between how a thing is bought and how it is kept**, and it is
   *  silent in the worst direction: a buyer with no scales writes "5" for five
   *  bunches into a field measured in kilos, five kilos of mint go on the shelf
   *  instead of a quarter of one, and the difference surfaces a month later at a
   *  count. Both fields or neither — a name with no size converts nothing. */
  packName?: string;
  packQty?: number;
  /** Where a request for this is answered from.
   *
   *  ⚠️ **Empty is the market**, and every catalogue that existed before this
   *  field is empty. A barman writing "cola, sugar, lemons" is thinking about an
   *  empty bar, not about who fetches what — sorting his list is knowledge about
   *  the store, which is the one thing his job does not involve, so the
   *  catalogue answers it and the list splits itself. */
  source?: ShoppingSource;
}

/** One ingredient in a dish, in recipe units (g, ml, pcs).
 *
 *  ⚠️ Brutto — what leaves the store to make the dish. Costing the peeled
 *  weight is how a tech card quietly understates every dish it describes. */
export interface RecipeLine {
  ingredientId: string;
  qty: number;
}

/** One ingredient on one delivery. Quantity in purchase units (kilo, litre,
 *  piece) and the price of one of them — the two figures an invoice carries. */
export interface PurchaseLine {
  ingredientId: string;
  qty: number;
  price: number;
  /** Until when this box is good, as an ISO date.
   *
   *  ⚠️ **On the delivery line, because that is where a date is known.** A
   *  shelf has no expiry date; a box that arrived on Tuesday does, and the same
   *  medicine delivered twice has two. ⚠️ Nothing subtracts against it — see
   *  handlers/expiry.go — so no screen may read it as "what is on the shelf". */
  expiresAt?: string;
  /** The manufacturer's batch number, exactly as printed. ⚠️ Never parsed: it
   *  is a recall's only handle, and every manufacturer writes it differently. */
  series?: string;
}

/** One delivery, as the invoice reads.
 *
 *  ⚠️ Dated by the invoice, not by when it was entered: a delivery is a
 *  measurement carrying its own date, and its prices apply from that day. */
export interface Purchase {
  id: string;
  at: string;
  /** ⚠️ Both: the id groups totals and debts, the text is what the invoice
   *  said the day it was entered — so a renamed supplier does not rewrite last
   *  year's deliveries. */
  supplierId?: string;
  supplier?: string;
  note?: string;
  lines: PurchaseLine[];
  total: number;
  createdBy?: string;
  paid: boolean;
  paidAt?: string;
}

/** One ingredient's flow through a period.
 *
 *  ⚠️ `used` is what the cards of the dishes sold describe, not what the
 *  kitchen consumed — a heavy hand, a dropped tray and a portion given to a
 *  regular are all real and none of them are here. The difference is the
 *  question, not the answer. */
export interface StockRow {
  name: string;
  unit: string;
  in: number;
  used: number;
  /** Thrown away, spilled, eaten by the staff. ⚠️ Its own column: one figure is
   *  what the cards say the dishes took, the other is what somebody wrote
   *  down, and merging them hides which of the two a gap came from. */
  written: number;
  diff: number;
  spent: number;
  /** This ingredient's share of the period's buying, and its Pareto class.
   *
   *  ⚠️ Ranked on what was **bought**, not on what the cards say was used: the
   *  spend is measured, the usage is an estimate only as good as the cards
   *  behind it, and half a menu is usually uncosted. */
  share: number;
  abc?: string;
}

export interface StockReportResponse {
  from: string;
  to: string;
  note: string;
  rows: StockRow[];
  spent: number;
  /** What the write-offs were worth, at the prices of the days they happened. */
  writtenValue: number;
}

/** What one reason cost over a period.
 *
 *  ⚠️ Grouped on the trimmed lower-case reason and shown in the spelling
 *  somebody actually used: the field is free text on purpose, so "buzildi",
 *  "Buzildi" and "buzildi " are three rows that are one thing. */
export interface WriteOffReason {
  reason: string;
  count: number;
  value: number;
}

/** Food that left without being sold. ⚠️ A reason is required — the same rule
 *  as a void, a refund or a cancelled order. */
export interface WriteOff {
  id: string;
  at: string;
  ingredientId: string;
  qty: number;
  reason: string;
  value: number;
  by?: string;
}

/** One thing to buy. ⚠️ `suggested` is the gap to the reorder point, which is
 *  the smallest defensible number — case sizes and next week's bookings are
 *  things only the owner knows. */
/** One product whose shelf label no longer says the truth.
 *
 *  ⚠️ **`reason` is three different jobs, not one.** "No barcode" stops a sale
 *  outright — no code, no scan, no ring-up; "never printed" is a shelf with
 *  nothing on it; "price" is a sticker that contradicts the till, which is the
 *  law's business rather than a matter of tidiness. */
export interface StaleLabel {
  id: string;
  name: string;
  price: number;
  barcode?: string;
  reason: "noBarcode" | "never" | "price";
  /** What the sticker says now, when there is one. */
  wasPrice?: number;
}

/** Which of the six designs a shop's labels come out in.
 *
 *  ⚠️ **A label is not one thing.** The same button prints a price tag clipped
 *  to a shelf edge and read from two metres, a sticker wrapped round a packet
 *  that only a scanner looks at, and a promotion card that has to say what the
 *  price used to be. One layout is wrong for two of those. */
export type LabelStyle =
  "shelf" | "price" | "sticker" | "compact" | "sale" | "full";

export const LABEL_STYLES: LabelStyle[] = [
  "shelf",
  "price",
  "sticker",
  "compact",
  "sale",
  "full",
];

export interface LabelDesign {
  style: LabelStyle;
  /** 58 or 80. ⚠️ 58 by default, unlike the receipts: a label roll is the
   *  narrow one, and a design laid out for 80 mm loses its right-hand end —
   *  which on a price tag is the price. */
  widthMm: number;
  feedLines: number;
  lang?: string;
  /** "shop" · "unit" · "date". ⚠️ **Missing means shown**, so a shop that has
   *  never opened the chooser keeps the lines it was printing. */
  fields?: Record<string, boolean>;
}

/** One design, drawn by the printer's own layout code.
 *
 *  ⚠️ **Never redrawn in the browser.** Two layout engines drift, and the drift
 *  is found by a shop whose stickers do not look like what they picked. */
export interface LabelDesignOption {
  style: LabelStyle;
  lines: string[];
  /** Whether the printer draws bars under it — false for the price tag, which
   *  is the one design chosen for not having them. */
  bars: boolean;
}

export interface LabelDesignView {
  design: LabelDesign;
  options: LabelDesignOption[];
  /** The product the six were drawn against: this shop's own where it has one,
   *  because the length that matters is this catalogue's. */
  sample: { name: string; barcode: string };
}

/** What a delivery just made worth printing.
 *
 *  ⚠️ **Offered, never printed on its own.** Two hundred packets answering
 *  themselves with two hundred stickers is a roll of paper nobody asked for and
 *  a printer switched off within a week. What the delivery removes is the
 *  typing: the products and the counts are already known. */
export interface LabelDue extends StaleLabel {
  /** How many arrived. ⚠️ Whole units, floored: four and a half kilos of loose
   *  rice is one shelf label, not five stickers. */
  copies: number;
}

export interface ShoppingRow {
  ingredientId: string;
  name: string;
  unit: string;
  onHand: number;
  minQty: number;
  suggested: number;
  price: number;
  cost: number;

  /** Which rule decided the quantity: "forecast" — what it will take before
   *  the next delivery arrives; "min" — the gap back up to the reorder point.
   *
   *  ⚠️ **Shown, not hidden.** "Min" is a line somebody drew once and may never
   *  have revisited; "forecast" is this month's cooking. A quantity an owner
   *  cannot take apart is one they either follow blindly or ignore. */
  basis: "forecast" | "min";
  /** Days the branch traded and this line moved nothing, on a line that
   *  otherwise moves nearly every day. ⚠️ An empty shelf is not a quiet one:
   *  these days are left out of the rate the forecast is built from, or an item
   *  that ran out is ordered less and runs out again. */
  stockOuts?: number;
  /** How many days ahead the suggestion is meant to last, and the forecast
   *  daily rate over them. Present only where the forecast had a say. */
  cover?: number;
  daily?: number;
  /** Measured days between deliveries and how many were measured — so the
   *  screen can say where the horizon came from. */
  every?: number;
  deliveries?: number;
  /** Measured shelf life in days, where dates are entered. It caps the
   *  horizon, so a short cover has a reason beside it. */
  shelfLife?: number;
  /** Already asked for on a list somebody is out with, and therefore already
   *  subtracted from the suggestion. */
  requested?: number;
}

/** One call to make. ⚠️ Grouped by supplier because that is how shopping is
 *  actually done: one group per phone number is one call. */
export interface ShoppingGroup {
  supplierId: string;
  name: string;
  phone?: string;
  rows: ShoppingRow[];
  cost: number;
}

/** Who the food comes from.
 *
 *  ⚠️ Optional on a delivery, and the free-text field survives beside it: a
 *  market run has no supplier, and requiring one would stop deliveries being
 *  recorded at all. */
export interface Supplier {
  id: string;
  name: string;
  phone?: string;
  note?: string;
  sort: number;
  isActive: boolean;
}

/** One supplier's line on the report. ⚠️ `owed` ignores the chosen period —
 *  a March invoice is still a debt in May. */
export interface SupplierTotal {
  supplierId: string;
  name: string;
  phone?: string;
  count: number;
  spent: number;
  owed: number;
  owedCount: number;
}

/** Stock moved from one shelf to another.
 *
 *  ⚠️ **Between two ingredients, not two warehouses.** A store belongs to the
 *  ingredient (one ingredient, one warehouse), so a restaurant keeping tonic in
 *  the cellar and behind the bar already has two of them — and a move is the
 *  quantity leaving one and arriving at the other. */
/** One batch made in a central kitchen.
 *
 *  ⚠️ Both halves of one document: `qty` of the item landed on the shelf, and
 *  `lines` came off it. Neither works alone — see the model's note. */
export interface Production {
  id: string;
  at: string;
  warehouseId: string;
  ingredientId: string;
  qty: number;
  lines: { ingredientId: string; name?: string; qty: number }[];
  value: number;
  note?: string;
  by?: string;
}

export interface StockTransfer {
  id: string;
  at: string;
  fromId: string;
  toId: string;
  qty: number;
  note?: string;
  value: number;
  by?: string;
}

/** One line of a count: what was found, what should have been there.
 *
 *  ⚠️ `expected` is the server's figure, frozen when the count was saved — a
 *  count whose baseline came from the screen that recorded it can be made to
 *  agree with anything. */
export interface StocktakeLine {
  ingredientId: string;
  counted: number;
  expected: number;
  diff: number;
  value: number;
}

export interface Stocktake {
  id: string;
  /** Which store was walked into and counted. ⚠️ A count is one room: counting
   *  the bar and the kitchen as one list lets a shortfall behind the bar cancel
   *  against a surplus in the kitchen. */
  warehouseId?: string;
  at: string;
  lines: StocktakeLine[];
  note?: string;
  /** When the explanation was given — a different fact from what it says.
   *
   *  ⚠️ A count with a variance and no `notedAt` is still owed an explanation,
   *  and the gap between `at` and this is worth seeing: a variance explained
   *  three days later was explained by somebody who had time to think. */
  notedAt?: string;
  value: number;
  by?: string;
}

/** One line of the store: what is there, and what it is worth. */
export interface StockBalanceRow {
  ingredientId: string;
  name: string;
  unit: string;
  warehouseId: string;
  /** In purchase units — the way it is counted on a shelf. */
  qty: number;
  /** At today's price: a store is worth what it would cost to replace. */
  value: number;
  minQty?: number;
  low?: boolean;
  /** ⚠️ A prep item is not on a shelf as itself — what it was made from is.
   *  Marked rather than hidden, so an owner looking for "Sous" finds it here
   *  with the reason instead of concluding the list is incomplete. */
  made?: boolean;
  /** Less than nothing on the shelf.
   *
   *  ⚠️ **Not a worse shortage — a different question.** A negative balance is
   *  a statement that cannot be true, so something upstream is: a delivery
   *  nobody entered, or a card that takes more than the kitchen does. It
   *  invalidates every other figure on the screen while it stands, and it used
   *  to sit in the list as an ordinary row with a minus in front of it. */
  negative?: boolean;
}

export interface StockBalances {
  rows: StockBalanceRow[];
  warehouses: Warehouse[];
  /** When each store was last counted, keyed by warehouse id. ⚠️ Null where it
   *  never has been: that is the difference between "measured from 3 March" and
   *  "everything that ever arrived", and the second is a figure nobody should
   *  order against without being told. */
  since: Record<string, string | null>;
  /** What each store holds, in money. */
  value: Record<string, number>;
}

/** One movement of one ingredient. */
export interface StockMovementDoc {
  at: string;
  /** ⚠️ A batch is two kinds, not one. For the sauce it is an arrival, for the
   *  tomatoes a departure — one entry would have to pick a sign, and the sign
   *  depends on which ingredient the reader came here about. */
  kind:
    | "purchase"
    | "sale"
    | "writeoff"
    | "transfer"
    | "production"
    | "production_used";
  /** Negative on the way out. */
  qty: number;
  note?: string;
}

export interface StockMovement {
  ingredient: { id: string; name: string; unit: string; warehouseId: string };
  from: string;
  to: string;
  opening: number;
  in: number;
  /** What the tech cards say the dishes sold used. */
  used: number;
  written: number;
  /** Stock carried between stores. ⚠️ Neither an in nor an out — see
   *  `stock_transfer` in DECISIONS. */
  movedIn: number;
  movedOut: number;
  /** ⚠️ **The columns without which this card did not add up.** The opening and
   *  closing figures always counted batches; the lines between them did not, so
   *  in any restaurant with a central kitchen the arithmetic on the one screen
   *  opened *to explain* a shortfall produced one of its own. */
  produced: number;
  producedUsed: number;
  closing: number;
  docs: StockMovementDoc[];
}

/** One dish whose sales the store cannot account for. */
export interface StockCoverageRow {
  id: string;
  name: string;
  qty: number;
  revenue: number;
  /** "none" — no card was ever written; "partial" — a card that was written
   *  and has since been broken by a deleted ingredient or a prep item that
   *  lost its yield. ⚠️ The second is worse than it looks: the missing line is
   *  skipped rather than flagged, so the dish quietly got *cheaper*, which on
   *  a margin screen reads as good news. */
  issue: "none" | "partial";
  /** For a set: the member that is missing its card. A set has no recipe of
   *  its own, so its own name sends the reader nowhere. */
  via?: string;
}

/** A prep item nothing can be costed through. ⚠️ One of these is usually the
 *  cause of many uncovered dishes — sushi rice with no yield makes every roll
 *  "partial" — which is why they are listed apart rather than mixed in. */
export interface StockCoveragePrep {
  id: string;
  name: string;
  issue: "no_output" | "incomplete";
}

/** How much of what was sold the tech cards account for.
 *
 *  ⚠️ **Not the ABC report's cost coverage.** That one asks whether a margin
 *  can be computed and a hand-typed cost answers it; a typed cost writes
 *  nothing off a shelf. A restaurant can sit at 100% there and 0% here. */
export interface StockCoverage {
  from: string;
  to: string;
  soldTotal: number;
  costedTotal: number;
  /** Per cent of revenue, already rounded by the server. */
  share: number;
  /** Uncovered dishes, **by money** — the ordering is the feature: the first
   *  ten lines are most of the answer, so the work has a visible end. */
  rows: StockCoverageRow[];
  preps: StockCoveragePrep[];
  /** Whether the restaurant asked not to be reminded, and the covered share
   *  below which it wants to be. ⚠️ Sent with the figure so the off-switch can
   *  sit beside the thing it silences — one three screens away in "Settings" is
   *  silenced by people avoiding the screen instead. */
  warnOff: boolean;
  warnFrom: number;
}

// ---- The shopping list somebody is sent to the market with ----

/** One thing to buy, and what came back of it. */
export interface ShoppingLine {
  id: string;
  /** Empty when the person writing the list typed a name the catalogue does
   *  not have. ⚠️ Allowed on purpose: a list somebody cannot finish writing is
   *  a list they write on paper instead, where nothing here can see it. */
  ingredientId?: string;
  name: string;
  /** The catalogue's own unit. ⚠️ **Never chosen by the buyer** — a market
   *  sells mint in bunches and flour in sacks, and "5" typed into a field
   *  measured in kilos is five kilos on the shelf instead of a quarter of one.
   *  The figure is then twenty times too high, the stop list never fires, and
   *  the gap turns up a month later at a count. */
  unit?: string;
  qty: number;
  note?: string;
  /** What came back. Kept beside what was asked rather than replacing it:
   *  "asked for ten, brought six" is the sentence this document exists for. */
  gotQty?: number;
  price?: number;
  gotAt?: string;
  /** The market did not have it. ⚠️ Its own answer, not a quantity of zero: a
   *  line nobody touched and one somebody looked for and could not find are
   *  different facts. */
  missing?: boolean;
  /** What the person who asked for it counted when it turned up.
   *
   *  ⚠️ **Absent is not zero.** A line nobody has checked and a line that
   *  arrived empty are different facts, and a screen that cannot tell them apart
   *  accuses somebody. Use `tookQty ?? gotQty` — never `tookQty || gotQty`. */
  tookQty?: number;
}

/** Where a request is answered from. ⚠️ Two, and there is no third: a supplier
 *  who delivers is still the buyer's line, and a list of places would grow one
 *  entry per restaurant until the routing stopped being a rule. */
export type ShoppingSource = "market" | "store";

export interface ShoppingOrder {
  id: string;
  branchId?: string;
  /** Which day the shopping is for, "YYYY-MM-DD". ⚠️ A string, not a date: the
   *  driver hands every date back in UTC, and "which day" is exactly the
   *  question that would then be off by one, silently. */
  forDate: string;
  /** Which half of the morning this is. ⚠️ One request becomes one document per
   *  place — the storekeeper answers in ten minutes and the buyer at nine, and a
   *  shared document would spend the morning in a state neither of them has a
   *  word for. */
  source?: ShoppingSource;
  /** The request both halves were written in. */
  groupId?: string;
  /** Which branch answers it — its own store room unless a chain points it at a
   *  central one. */
  supplyBranchId?: string;
  /** ⚠️ Three states, and the third is the point: "bought" said the money had
   *  been spent and nothing about whether the goods reached the person who
   *  asked. That gap is where things disappear. */
  status: "sent" | "shipped" | "done";
  lines: ShoppingLine[];
  note?: string;
  createdBy?: string;
  createdAt: string;
  purchaseId?: string;
  /** The van the store half became, when the goods came from another branch. */
  dispatchId?: string;
  shippedAt?: string;
  shippedBy?: string;
  acceptedAt?: string;
  acceptedBy?: string;
  doneAt?: string;
}

/** One request as the panel reads it: what somebody asked for at six, with both
 *  halves of the answer beside it. ⚠️ Grouped because splitting the list is the
 *  server's idea, not the barman's. */
export interface ShoppingRequestGroup {
  groupId: string;
  forDate: string;
  createdAt: string;
  createdBy?: string;
  branch?: string;
  /** The branch that ships the store half, when it is not this one. */
  supply?: string;
  orders: ShoppingOrder[];
}

/** One row of the shortage the store has already worked out, as a starting
 *  point for a list. ⚠️ The form starts from this rather than blank, or the
 *  restaurant would have two competing shopping lists and the buyer no way to
 *  tell which is real. */
/** One ingredient the catalogue already has, for the list writer to pick.
 *
 *  ⚠️ **The whole catalogue, not only what is short.** A list can ask for
 *  something above its minimum, and with only the shortage on offer the writer
 *  had to type the name — which creates a second ingredient no tech card points
 *  at. The screen was quietly manufacturing duplicates.
 *
 *  ⚠️ **No prices here.** This screen is opened on a till the whole room
 *  shares; the buyer's own catalogue carries prices because they need them at a
 *  stall. */
export interface ShoppingCatalogRow {
  ingredientId: string;
  name: string;
  unit: string;
  packName?: string;
  packQty?: number;
  /** Who will answer a line for it. ⚠️ Shown while the list is being written,
   *  not only after it is sent: the split happens on the server either way, but
   *  a writer who cannot see it has no way to notice the one ingredient that is
   *  filed wrongly — and the wrong filing surfaces as a request that sat all
   *  morning on a phone belonging to somebody who was never going to answer
   *  it. */
  source?: ShoppingSource;
}

export interface ShoppingDraftRow {
  ingredientId: string;
  name: string;
  unit: string;
  qty: number;
  onHand: number;
  packName?: string;
  packQty?: number;
  source?: ShoppingSource;
}

// ---- The market run ----

/** One ingredient as the buyer's phone needs it. */
export interface BuyCatalogRow {
  id: string;
  name: string;
  /** The purchase unit — kilos, litres, pieces. Never the recipe's grams. */
  unit: string;
  /** ⚠️ **What it cost last time, and the only guard the price has.** A market
   *  price typed on a phone reprices every dish that uses the ingredient, and
   *  `9 000` entered as `90 000` looks like an ordinary number afterwards.
   *  Nothing downstream can tell them apart — a person at the stall can, if the
   *  last one is in front of them. */
  lastPrice: number;
  /** How the market sells it, when somebody wrote it down. ⚠️ Sent so the phone
   *  can offer the choice and show the conversion, never so it can perform it:
   *  the arithmetic lands on a shelf and belongs to the server. */
  packName?: string;
  packQty?: number;
}

/** One line of a market run on its way to the server. */
export interface BuyLineInput {
  /** Whether `qty` and `price` count packs rather than the unit the store keeps
   *  it in. ⚠️ A flag, not a converted number — see BuyCatalogRow. */
  pack?: boolean;
  /** Empty when the catalogue does not have it yet — `newName` then carries
   *  what was typed. */
  ingredientId?: string;
  newName?: string;
  qty: number;
  price: number;
}

/** What the server says came of a recorded run. */
export interface BuyResult {
  purchase: Purchase;
  /** How many ingredients now cost something different. */
  repriced?: number;
  /** What had to be invented, by name. ⚠️ Shown to the person who pressed the
   *  button rather than left for a manager to discover: an ingredient created
   *  at a market has no unit, no minimum and no card. */
  created?: string[];
  /** The run was already here — a retry that crossed with its own first
   *  attempt. Not an error, and the screen says so rather than showing one. */
  already?: boolean;
}

/** How long a cash shift has been open, and whether that is too long.
 *
 *  ⚠️ **Computed on every read rather than stored.** A saved "overdue" flag goes
 *  stale the moment the clock passes it, and the screens that show this are the
 *  ones somebody opens to decide whether to go and count the drawer.
 *
 *  ⚠️ The shift is never closed automatically: a close writes a *counted*
 *  figure, and a count nobody made would destroy the only measurement the shift
 *  was keeping. */
export interface ShiftAge {
  hours: number;
  overdue: boolean;
  /** The line this branch is measured against, so a screen can name it. */
  maxHours: number;
}

/** One cost the restaurant paid that nothing else records.
 *
 *  ⚠️ **Only what has no document of its own.** A delivery is a purchase and a
 *  wage is a staff payment; both already have their own line in the financial
 *  report, and entering either here as well would count it twice — a
 *  double-counted cost is indistinguishable from a real one. */
export interface Expense {
  id: string;
  at: string;
  category: string;
  amount: number;
  note?: string;
  /** "cash" | "transfer" | "card". ⚠️ Recorded because it decides whether a box
   *  got lighter — the difference between a safe balance that matches the notes
   *  and one that does not. */
  method?: string;
  createdBy?: string;
}

/** What is in the safe and how it got there.
 *
 *  ⚠️ **A place, not a profit and loss.** This answers "where is the money",
 *  the financial report answers "did we make any". Cash moved from the drawer
 *  into the safe is not an expense and money handed to a buyer is not spent
 *  until it buys something — counting a location's movements as outgoings is
 *  how a report subtracts the same money twice. */
export interface SafeBalance {
  in: number;
  out: number;
  /** ⚠️ Can go below zero, and it is shown: a negative safe means the ledger is
   *  missing something that went in, and hiding it would leave the one screen
   *  that could have said so agreeing with a count that cannot be right. */
  balance: number;
  lastAt?: string;
}

/** One movement into or out of it. */
export interface SafeEntry {
  id: string;
  kind: "in" | "out";
  category?: string;
  amount: number;
  at: string;
  by?: string;
  note?: string;
  /** What caused it, where something did. ⚠️ Unique with `refId`, so a retry
   *  cannot put the same hand-over in the ledger twice — a duplicate in a
   *  balance is plausible, wrong, and invisible to everything downstream. */
  refKind?: string;
  refId?: string;
}

/** One person's petty-cash account.
 *
 *  ⚠️ **Three measured figures and one subtraction**, never a stored balance: a
 *  kept total is a second copy of what the ledger already says, and it drifts
 *  the first time a delivery is deleted or an advance corrected — silently, in
 *  a number about money. */
export interface AdvanceBalance {
  staffId: string;
  staffName: string;
  issued: number;
  returned: number;
  /** Deliveries this person recorded and paid for. */
  spent: number;
  /** ⚠️ Can be negative, and that is not an error to hide: a buyer who ran out
   *  and paid for the last crate themselves is owed money, and a balance
   *  clamped at zero would be silent about exactly the debt somebody is waiting
   *  to be paid. */
  balance: number;
  lastAt?: string;
}

/** One movement of it. */
export interface AdvanceEntry {
  id: string;
  staffId: string;
  staffName?: string;
  /** "out" — handed over; "back" — returned unspent. ⚠️ Two kinds rather than a
   *  signed amount: a negative in a money column reads as a correction, and
   *  this ledger has real corrections in it too. */
  kind: "out" | "back";
  amount: number;
  at: string;
  by?: string;
  note?: string;
}

/** What to count, and deliberately not what should be there.
 *
 *  ⚠️ **No expected figure, no running balance, nothing to match.** Both screens
 *  already refused to draw it before a number was typed, and the rule was
 *  right — but it lived here, in the browser, where it bought less than it
 *  looked like: the figure was in the page either way, and typing anything,
 *  reading the number and correcting the entry defeated it in one move.
 *
 *  The variance comes back with the saved count, after it can no longer be
 *  edited, where it is a finding rather than a target. */
export interface StocktakeSheetRow {
  ingredientId: string;
  name: string;
  unit: string;
}

/** One line of the morning briefing.
 *
 *  ⚠️ The words come from a model and `numbers` does not — they are printed
 *  together so an owner can check the sentence against the figure it was
 *  written about, which is the only reason to believe the sentence. */
export interface BriefingResponse {
  cards: BriefingCard[];
  /** Why the platform could not answer, when it said so.
   *
   *  ⚠️ Shown rather than swallowed: "quota exceeded, retry in 18s" is a
   *  completely different morning from "no key configured", and an owner who
   *  enabled this and sees nothing needs the sentence. */
  error?: string;
  /** Absent when the restaurant holds the assistant. */
  entitled?: boolean;
  monthly?: number;
  capped?: boolean;
}

export interface BriefingCard {
  key: string;
  title: string;
  body: string;
  area: string;
  action?: string;
  params?: Record<string, string>;
  numbers?: Record<string, number>;
}

/** One proposed campaign message. `parts` is present only for SMS, and is what
 *  the send will actually be billed as — counted by the server with the same
 *  function the invoice uses. */
export interface CampaignVariant {
  text: string;
  note?: string;
  chars: number;
  parts?: number;
}

/** One person's month at the counter.
 *
 *  ⚠️ Shares are **per thousand**, not per cent: two voids in four hundred
 *  checks is 0% at one decimal and 5‰ here, and the distance between 5‰ and 40‰
 *  is the entire content of the report. */
export interface LossRow {
  id: string;
  name: string;
  checks: number;
  sales: number;
  voids: number;
  voidValue: number;
  voidShare: number;
  /** Voids they did on their own authority — nobody else saw them happen. */
  authedSelf: number;
  discounts: number;
  discountValue: number;
  discountShare: number;
  cashChecks: number;
  cashSales: number;
}

/** What this branch calls unusual.
 *
 *  ⚠️ A zero from the server never means "alert on everything" — it is filled
 *  in with a default before it leaves. The defaults are deliberately high: a
 *  channel that starts quiet can be turned down, one that starts noisy is muted
 *  before anybody finds the setting. */
export interface AlertSettings {
  enabled: boolean;
  voidFrom: number;
  discountFrom: number;
  cashShortFrom: number;
  stockShortFrom: number;
  /** ⚠️ The ceiling on messages per day. A bad night would otherwise send forty,
   *  and forty messages is silence. Past it, events are still recorded and the
   *  panel still shows them. */
  dailyMax: number;
}

/** One thing the owner was told about. */
export interface LossAlert {
  id: string;
  kind: string;
  at: string;
  by?: string;
  authBy?: string;
  amount: number;
  reason?: string;
  subject?: string;
  sentAt?: string;
  sendErr?: string;
}

/** One online order as the counter sees it.
 *
 *  ⚠️ `settle` is the whole point: a cashier does not need another list of
 *  orders, they need what *they* have to do about each one — which depends on
 *  how it was paid and whether it is delivered or collected. */
export interface OnlineOrder {
  id: string;
  number: string;
  type: string;
  status: string;
  total: number;
  paymentMethod: string;
  paymentStatus: string;
  at: string;
  /** "nothing" | "from_courier" | "at_counter" | "unfinished" */
  settle: string;
  who?: string;
  phone?: string;
}

/** What is left of today's assistant allowance.
 *
 *  ⚠️ A **daily** cap sold **monthly**, and the two units are deliberate: the
 *  cap is daily because that is what stops a stuck tab spending a month's
 *  allowance in an afternoon, and it is sold monthly because that is how a
 *  restaurant thinks about a bill. */
export interface AIQuota {
  /** Whether the platform has an assistant configured at all. */
  on: boolean;
  /** Whether this restaurant's plan includes it or bought it. */
  entitled?: boolean;
  used?: number;
  limit?: number;
  /** What the plan gives before anything was bought, so the total can be
   *  accounted for rather than trusted. */
  planLimit?: number;
  extraBlocks?: number;
  blockSize?: number;
  blockPrice?: number;
  /** Who to write to. From the server: it is our handle, and a panel carrying
   *  its own copy would be as many copies as there are tenants. */
  contact?: string;
}

// ---- Support ----

/** Who said a line. ⚠️ The assistant's answers are stored and marked: an owner
 *  scrolling back has to be able to tell what a person told them from what a
 *  model did. */
export type SupportFrom = "owner" | "operator" | "assistant";

export type SupportMessage = {
  id: string;
  threadId: string;
  from: SupportFrom;
  author: string;
  text: string;
  at: string;
};

export type SupportThread = {
  id: string;
  subject: string;
  status: "waiting" | "open" | "closed";
  askedBy: string;
  operatorName?: string;
  unreadForOwner: number;
  lastText: string;
  lastFrom: SupportFrom;
  lastAt: string;
  createdAt: string;
};

/** One dish an import proposes, before anybody has agreed to it.
 *
 * ⚠️ `price: 0` means "the page did not say", which the form shows as an empty
 * box rather than as free — a dish imported at zero that nobody noticed is a
 * dish the restaurant gives away. */
export interface ImportedDish {
  name: string;
  description?: string;
  price: number;
  imageUrl?: string;
  category?: string;
  /** Already on this menu under that name. Decided by the server, not by the
   *  browser: the browser has the menu it loaded, which may be a week old in a
   *  tab somebody left open. */
  exists?: boolean;
}

/** Which install an account may sign in from.
 *
 *  ⚠️ **A password is not an identity in a restaurant**: staff logins are
 *  written on cards, couriers hand theirs to whoever covers a shift, and every
 *  one of those sign-ins is correct. What the server can see is that the phone
 *  changed — so the apps send an install id and the account is bound to it, one
 *  per app. The panel shows the binding and can release it, which is the half
 *  that makes the lock safe to have. See `handlers/logindevice.go`. */
export interface LoginDevice {
  id: string;
  kind: "admin" | "staff" | "courier";
  subjectId: string;
  /** "owner" | "waiter" | "courier" | "team". */
  app: string;
  deviceId: string;
  platform?: string;
  /** Something a person can recognise, when the app sent one. */
  name?: string;
  /** Where it was last seen from. */
  ip?: string;
  createdAt: string;
  lastSeenAt: string;
}

/** One transfer that arrived from a rail or a marketplace.
 *
 *  ⚠️ **What arrives is not revenue** — the sale was counted the day the guest
 *  paid, and this is that same money changing location. Only `commission` is a
 *  cost, and only `commission` reaches the financial report. */
export interface Payout {
  id: string;
  provider: string;
  providerName?: string;
  /** The window the statement covers, "YYYY-MM-DD" — not when it arrived. */
  periodFrom: string;
  periodTo: string;
  /** What the provider says it collected, what it kept, what landed.
   *
   *  ⚠️ All three as the statement reads them. `gross − commission − net` is
   *  shown when it is not zero, because that difference is a real event: a
   *  refund clawed back, a penalty, a correction. */
  gross: number;
  commission: number;
  net: number;
  receivedAt: string;
  account?: string;
  note?: string;
  createdBy?: string;
}

/** One rail: what it collected for us, and what it has sent on. */
export interface PayoutBalance {
  provider: string;
  name: string;
  /** Sales no payout covers yet. */
  sold: number;
  count: number;
  /** Everything up to this date is settled. Empty when nothing has ever been
   *  recorded — and then `sold` is every sale ever made through the rail,
   *  which is a fact and not yet a debt. */
  settledThrough?: string;
  received: number;
  commission: number;
  lastAt?: string;
}

/** A marketplace the restaurant sells through. */
export interface AggregatorAccount {
  id: string;
  name: string;
  enabled: boolean;
  /** Prefills the payout form only — never used to compute money. */
  commissionPercent?: number;
}

export interface PayoutsResponse {
  payouts: Payout[];
  balances: PayoutBalance[];
  rails: AggregatorAccount[];
}

/** One place the restaurant's money physically or legally sits. */
export interface MoneyPlace {
  kind: "safe" | "drawer" | "courier" | "advance" | "bank" | "rail";
  name: string;
  amount: number;
  note?: string;
  /** ⚠️ Whether the figure was **counted** (a drawer at close, a balance read
   *  off a bank app) or **added up from documents**. They fail in opposite
   *  directions: a counted one goes stale, a summed one goes wrong when a
   *  document is missing — and the reader needs to know which to look for. */
  counted?: boolean;
  at?: string;
}

/** ⚠️ Three totals, never one: cash can be spent tonight, the bank this week,
 *  a rail's balance when somebody else decides. */
export interface MoneyPosition {
  cash: MoneyPlace[];
  bank: MoneyPlace[];
  rails: MoneyPlace[];
  cashTotal: number;
  bankTotal: number;
  railsTotal: number;
  /** The ceiling the bank agreed with this branch. Cash above it must be
   *  handed over — see docs/DECISIONS.md → "Inkassatsiya". Zero = not set. */
  cashLimit?: number;
  overLimit?: boolean;
  /** Which branch the ceiling belongs to. ⚠️ A zero id arrives as "000…0" and
   *  is truthy — read it with hasId(). */
  branchId?: string;
}

/** One handover of cash to the bank (or into the office safe on the way).
 *
 *  ⚠️ The reconciliation is frozen at the moment of the handover: recomputed
 *  later it would answer differently every time an old shift is corrected. */
export interface Collection {
  id: string;
  at: string;
  amount: number;
  to: "bank" | "safe";
  takenBy?: string;
  bag?: string;
  note?: string;
  fromAt: string;
  shifts: number;
  counted: number;
  expected: number;
  variance: number;
  safeBefore: number;
  /** amount − counted. Stored even at zero, so "checked and correct" is not
   *  indistinguishable from "nobody checked". */
  diff: number;
  by?: string;
}

export interface CollectionsResponse {
  collections: Collection[];
  /** What a handover made right now would cover — the same figures the
   *  document will freeze, shown before anybody signs. */
  due: Collection;
}

/** Somebody a wage can be handed to at the counter.
 *
 *  ⚠️ Staff and couriers in one list on purpose: a cashier paying somebody out
 *  of the drawer thinks of a person standing in front of them, not of which
 *  file cabinet the record lives in. */
export interface TillPayee {
  id: string;
  kind: "staff" | "courier";
  name: string;
  position?: string;
}

/** One delivery line that runs out.
 *
 *  ⚠️ **`qty` is what was delivered, never what is left.** Nothing subtracts
 *  against a box — consumption keys on the ingredient — so a screen that read
 *  this as a shelf count would be inventing a number, and a pharmacist who
 *  finds one stock figure wrong stops believing every other one. */
/** One line of a van load: what was put on it, and what came off.
 *
 *  ⚠️ **`got` absent is not zero** — it is a crate nobody has counted off yet,
 *  which reads differently on every screen than one that arrived empty. */
export interface DispatchLine {
  ingredientId: string;
  /** Frozen at the moment the slip was printed: three people sign it, and a
   *  renamed ingredient must not rewrite a piece of paper. */
  name: string;
  unit: string;
  qty: number;
  got?: number;
}

/** One van load: one branch's store sending goods to another's.
 *
 *  ⚠️ **Not a transfer** — that one moves between two shelves of the same
 *  branch. This is the central kitchen's van, and it is the movement a chain
 *  has every morning. */
export interface Dispatch {
  id: string;
  brandId?: string;
  fromBranchId: string;
  toBranchId: string;
  /** What the paper calls itself: the day and that day's number. */
  number: string;
  at: string;
  lines: DispatchLine[];
  /** What is on the van in money. ⚠️ Carried, not created — it never reaches
   *  the financial report's expenses. */
  value: number;
  driver?: string;
  note?: string;
  by?: string;
  acceptedAt?: string;
  acceptedBy?: string;
}

/** A dispatch with both branch names resolved, as the list draws it. */
export interface DispatchRow {
  dispatch: Dispatch;
  from: string;
  to: string;
}

/** One line of what this store is holding, for loading a van. */
export interface DispatchStockRow {
  id: string;
  name: string;
  unit: string;
  qty: number;
}

/** What a shortfall turned out to be.
 *
 *  ⚠️ **Six answers and no "other", because the list is the product.** A free
 *  sentence alone makes the queue a pile of prose nobody can count; the kind is
 *  what lets a month of them say "half of ours are deliveries nobody entered",
 *  which is a fixable sentence about a process. The sentence is still required —
 *  the kind says which class, and only the words say which delivery. */
export type ShortageVerdict =
  | "miscount"
  | "waste"
  | "card"
  | "paperwork"
  /** Rung up as something else: this row is short and its twin is over.
   *  ⚠️ Offered everywhere, and a shop cannot do without it — see
   *  lib/shortages.ts. */
  | "swap"
  | "lost";

/** One shortfall: one ingredient, on one count, in one store. */
export interface ShortageRow {
  /** ⚠️ The pair **is** the id — there is no case document until somebody
   *  answers one, so nothing else could be. */
  stocktakeId: string;
  ingredientId: string;
  name: string;
  unit: string;
  branchId?: string;
  branch?: string;
  warehouseId?: string;
  warehouse?: string;
  /** When it was found, and the previous count of that store — the stretch it
   *  accumulated over. ⚠️ Absent where the store had never been counted: that
   *  shortfall is measured from the beginning of time and must not be read as
   *  a month's loss. */
  at: string;
  since?: string;
  expected: number;
  counted: number;
  diff: number;
  /** What is missing, in som, positive. */
  value: number;
  by?: string;
  /** The whole count's note and shortfall — a count's note covers forty lines,
   *  and `share` is how much of it this one is. */
  countNote?: string;
  countLoss: number;
  share: number;

  /** How long the period was, and what the shortfall works out to a day.
   *  ⚠️ A million som over ninety days is shrinkage; the same million over four
   *  days is still happening, and money alone cannot tell them apart. */
  days?: number;
  perDay?: number;
  /** What share of what should have been there is gone. Twelve kilos out of
   *  four hundred is trade; twelve out of fourteen is an event. */
  pct?: number;
  /** What the tech cards accounted for over the same period.
   *  ⚠️ **Zero is the most important thing this screen can say**, and it is not
   *  a shortfall: where nothing consumed the ingredient, the whole counted
   *  difference is a gap in the cards rather than something that left. */
  used: number;
  /** How many of this store's counts in the window found it short. Twice is a
   *  pattern; once is an evening. */
  repeat?: number;
  /** A surplus on the same count worth about the same — the mis-scan, named
   *  before somebody calls it a loss. A question, never a verdict. */
  twin?: string;
  verdict?: ShortageVerdict;
  verdictNote?: string;
  verdictBy?: string;
  verdictAt?: string;
}

/** ---- The accountant's two doors ----
 *
 *  ⚠️ **The operator's own codes, never renamed on the way in.** A status here
 *  is a claim about what Didox holds; a second vocabulary of ours would be a
 *  second thing to keep in step, and the day they diverge the panel describes a
 *  document that does not exist. Mirrors models/accounting.go. */
export interface EdiParty {
  name: string;
  vatRegCode: string;
  vatRegStatus: number;
  account: string;
  bankId: string;
  address: string;
  director: string;
  accountant: string;
}

export interface EdiSettings {
  provider: string;
  enabled: boolean;
  sandbox: boolean;
  tin: string;
  seller: EdiParty;
  hasPartnerKey: boolean;
  hasPassword: boolean;
  lastSyncAt?: string | null;
  lastError?: string;
  /** How much post is waiting to become a delivery. */
  notImported: number;
  /** ⚠️ Always false, and read out on the screen: the one question an
   *  accountant asks about any EDI integration is who signs. */
  signsHere: boolean;
}

export type EdiSettingsInput = {
  enabled: boolean;
  sandbox: boolean;
  tin: string;
  partnerToken?: string;
  password?: string;
  seller: EdiParty;
};

export interface EdiLine {
  no: number;
  name: string;
  catalogCode?: string;
  barcode?: string;
  packageName?: string;
  packageCode?: string;
  qty: number;
  price: number;
  sum: number;
  vatRate?: number;
  vatSum?: number;
  sumWithVat: number;
  ingredientId?: string;
}

export interface EdiDocument {
  id: string;
  docId: string;
  direction: "in" | "out";
  type: string;
  number: string;
  date: string;
  status: number;
  partnerTin?: string;
  partnerName?: string;
  contractNo?: string;
  contractDate?: string;
  total: number;
  vatTotal?: number;
  totalWithVat: number;
  hasVat: boolean;
  hasMarks: boolean;
  /** ⚠️ Empty means "not fetched yet", not "an invoice with nothing on it":
   *  the operator's list endpoint does not carry the lines. */
  lines: EdiLine[];
  purchaseId?: string;
  importedAt?: string;
  importedBy?: string;
  syncedAt: string;
}

export interface EdiDocumentList {
  documents: EdiDocument[];
  lastSyncAt?: string | null;
  enabled: boolean;
}

export interface OneCSettings {
  enabled: boolean;
  login: string;
  hasPassword: boolean;
  branchId?: string;
  days: number;
  /** The address an accountant pastes into 1C. ⚠️ Built by the server rather
   *  than assembled by hand on the page: it is the one field that must be
   *  exact, and a 404 reads as "the integration does not work". */
  url: string;
  lastSeenAt?: string | null;
  lastType?: string;
  lastMode?: string;
  lastError?: string;
  lastImported?: number;
  lastExported?: number;
}

export interface ShortageQueue {
  rows: ShortageRow[];
  /** Rows past the cap. ⚠️ Sent so the screen can say the list is not all of
   *  it, rather than quietly implying it is. */
  more: number;
  from: string;
  to: string;
  days: number;
  open: number;
  openValue: number;
  total: number;
  /** What share of sales the tech cards account for over the same window, and
   *  whether it is low enough that these figures do not mean what they say. */
  coverage: number;
  weak: boolean;
}

export interface ExpiringRow {
  ingredientId: string;
  name: string;
  unit: string;
  series?: string;
  expiresAt: string;
  /** Whole days from today; negative means it has already passed. */
  daysLeft: number;
  qty: number;
  purchaseId: string;
  at: string;
  supplier?: string;
}

/** One card per model: the first variant stands for the rest.
 *
 *  ⚠️ **A guest browsing a clothes shop should see shirts, not sizes.** Twelve
 *  cards of the same photograph is a catalogue nobody scrolls, and the choice
 *  between them cannot be made from a grid anyway — it is made on the page,
 *  where every size is listed with its own price and its own stock.
 *
 *  ⚠️ **The first variant is a complete card on its own.** Variants are copied
 *  from their model when they are generated, so the name, the photograph, the
 *  description and the price are already on it — which is why this needs no
 *  model row and no second request.
 *
 *  ⚠️ **`hasId`, never a bare truthiness check.** A dish that is not a variant
 *  arrives with `variantOf: "000000000000000000000000"` — Go's `omitempty` has
 *  no effect on an ObjectID — and that string is truthy. Read naively, every
 *  ordinary dish on the menu became a variant of the same zero model, so each
 *  category kept exactly **one** card and dropped the rest. That is what this
 *  went live as. */
export function oneCardPerModel(items: MenuItem[]): MenuItem[] {
  const seen = new Set<string>();
  return items.filter((m) => {
    if (!hasId(m.variantOf)) return true;
    if (seen.has(m.variantOf)) return false;
    seen.add(m.variantOf);
    return true;
  });
}
