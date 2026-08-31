// The articles in English.
//
// ⚠️ **A translation, not a rewrite.** Same slugs, same sections, same figures
// and the same warnings in the same places. English is the third audience here
// — a foreign owner, a consultant, a developer reading before an integration —
// so the terms are the ones the panel itself uses in English.

import type { Article } from "./types";

export const articlesEn: Article[] = [
  // ═══════════════════════════════════ Getting started
  {
    slug: "first-day",
    section: "start",
    title: "The first day: where to start",
    lead: "A restaurant goes live in a day. Do it in this order and you take your first order the same evening.",
    keys: ["start", "setup", "launch", "first day"],
    body: [
      {
        p: "Keel is two halves: the *site* your guest sees and the *panel* you see. Whatever you change in the panel appears on the site immediately — there is no separate publish button.",
      },
      { h: "The order" },
      {
        steps: [
          "*Restaurant profile* — name, phone, address and opening hours in `Settings`. With no hours filled in, the site shows «closed».",
          "*Categories* — «Main courses», «Drinks». Categories first: a dish cannot be saved without one.",
          "*Menu* — dishes, prices and photos. The menu can be imported from a link or from your old till.",
          "*Delivery* — price by zone or by distance. Without it a guest cannot enter an address.",
          "*Payments and SMS* — SMS is required for customers to sign in; online payment is optional (cash always works).",
          "*Staff* — accounts for the cashier, the waiter, the cook. Each gets their own login.",
        ],
      },
      {
        warn: "Do not try to fill in the store and the tech cards *on day one*. They only work if deliveries are entered regularly; otherwise the reports show numbers that are wrong. Get selling first, start the store in the second week.",
      },
      {
        fig: "dashboard",
        notes: {
          nav: "The sections, on the left. Groups open and close — every screen is found here.",
          period: "The period picker: every figure on the home screen belongs to it.",
        },
      },
      { see: ["login", "branches", "menu-item", "delivery-zones"] },
    ],
  },
  {
    slug: "login",
    section: "start",
    title: "Signing in and passwords",
    lead: "Where the panel lives, the first account, and what to do when a password is lost.",
    keys: ["login", "password", "sign in", "admin panel", "forgot"],
    body: [
      {
        p: "The panel is at `/admin` on your own site. If the site is `restoran.uz`, the panel is `restoran.uz/admin`.",
      },
      {
        p: "We issue the first account — login and password are sent when you sign up. *Change the password* after your first sign-in: `Settings` → `Account`.",
      },
      { h: "Forgot the password" },
      {
        steps: [
          "Click `Forgot your password?` on the sign-in page.",
          "Enter the phone number attached to the account — a code arrives by SMS.",
          "The code lets you set a new password.",
        ],
      },
      {
        warn: "This path does not work if no SMS provider is configured — the code never leaves. Write to us and we will restore the account by hand.",
      },
      {
        tip: "A panel login belongs to *one person*. Do not give it to a cashier or a waiter: they have their own apps and their own accounts. A panel login left open on a computer in the dining room is the customer database, the payment keys and the reports.",
      },
      { see: ["admins", "staff-add", "security"] },
    ],
  },
  {
    slug: "admins",
    section: "start",
    title: "Panel administrators: owner and manager",
    lead: "Who signs in, who sees what, and where every action is recorded.",
    keys: ["admin", "manager", "owner", "permission", "access"],
    body: [
      { p: "There are two levels in the panel, and the difference matters." },
      {
        table: {
          head: ["Role", "Sees"],
          rows: [
            [
              "*Owner*",
              "Everything: every branch, the customer database, payment keys, reports, campaigns, the list of administrators.",
            ],
            [
              "*Manager*",
              "Only their own branch: orders, menu, store, staff. Cannot send campaigns to customers.",
            ],
          ],
        },
      },
      {
        warn: "A manager cannot reach another branch through the address bar — the server narrows every request to their branch. But sharing the *owner* account cancels that guard entirely: one account per person.",
      },
      { h: "The action log" },
      {
        p: "`Money and team` → `Log`: who did what — changed a price, cancelled an order, cleared a count, *edited a tech card* — which ingredient, and from what to what.",
      },
      {
        tip: "Tech card edits are logged in detail for a reason: raising a norm on a card is the one way to take stock out without leaving a trace in the arithmetic. Read those lines once a month.",
      },
      { fig: "admins" },
      { see: ["staff-roles", "admin-log", "security"] },
    ],
  },
  {
    slug: "branches",
    section: "start",
    title: "Branches and brands",
    lead: "One restaurant, a chain, or two brands in one building — all three work.",
    keys: ["branch", "brand", "chain", "second location"],
    body: [
      {
        p: "A *brand* is a menu and a site. A *branch* is a building with a door: its own address, hours, till, store and staff.",
      },
      {
        list: [
          "One restaurant — one brand, one branch. Nothing ever asks you to choose.",
          "A chain — one brand, several branches. The menu is shared; prices and stock are per branch.",
          "Two concepts in one building — two brands, two sites, one till.",
        ],
      },
      {
        p: "A branch picker appears at the top of the panel when there is more than one. Orders, till and store screens follow that choice.",
      },
      {
        warn: "Store screens *require a branch* and will not open without one. «The company holds 9 kg of meat», spread across three fridges, is a number nobody can count and nobody can order against — so the screen asks which building.",
      },
      {
        warn: "Delivery is switched on *per branch*. Left off in two of them, it is the quietest failure there is: the form looks filled in, nothing errors, and the order from one district drives to the other side of town.",
      },
      { see: ["delivery-setup", "working-hours", "stock-balances"] },
    ],
  },
  {
    slug: "working-hours",
    section: "start",
    title: "Opening hours and holidays",
    lead: "When the site takes orders and when the till reads the stop list.",
    keys: ["hours", "closed", "open", "schedule", "time"],
    body: [
      {
        p: "`Settings` → `Opening hours`: an opening and closing time for each weekday. The site shows «Open now» or «Closed» from them.",
      },
      {
        p: "For a place that works past midnight the closing time is the smaller number: `10:00 – 02:00` means «until the next day».",
      },
      {
        warn: "With the hours *empty* the site treats the restaurant as closed and takes no orders. This is the most common cause of «the site is not taking orders» on day one.",
      },
      {
        tip: "If a till system is connected, the stop list starts being read 15 minutes before opening and stops after closing. Nobody empties a pot at four in the morning — that would be a request for nothing.",
      },
      { see: ["branches", "stop-list"] },
    ],
  },

  // ═══════════════════════════════════ The site
  {
    slug: "site-design",
    section: "site",
    title: "How the site looks: colour, logo, sections",
    lead: "The site's colour and the blocks on the home page are set from the panel — no developer needed.",
    keys: ["design", "colour", "logo", "look", "builder"],
    body: [
      {
        p: "`Settings` → `Site`: the brand colour, the logo and the order of the blocks on the home page.",
      },
      {
        list: [
          "*Accent colour* — buttons and marks take it. You pick one; the system works out the rest so text stays readable.",
          "*Logo and cover* — uploaded in `Settings`. The cover sits at the top of the home page.",
          "*Blocks* — «Popular dishes», «Offers», «About us», «Map», «Reviews». Each can be turned off and reordered.",
          "*Theme* — the guest gets light or dark from their own browser setting. There is no need to force one.",
        ],
      },
      {
        tip: "Choose the colour by looking at your own food photos. A deep blue holds a yellow plov well; red argues with almost any food.",
      },
      { fig: "site-home" },
      { see: ["site-texts", "menu-photos", "site-seo"] },
    ],
  },
  {
    slug: "site-texts",
    section: "site",
    title: "Site texts and «About us»",
    lead: "Every sentence on the home page is written from the panel, in three languages.",
    keys: ["text", "translation", "about us", "language", "russian"],
    body: [
      {
        p: "`Settings` → `Texts`: heading, description, «About us» and the contact block.",
      },
      {
        p: "The site is trilingual: *Uzbek*, *Russian*, *English*. Every field has three boxes. If the Russian or English one is empty the Uzbek text is shown, so the site never goes out with a gap in it.",
      },
      {
        warn: "Translate the dish names too. On a Russian page a name left in Uzbek makes the whole page look like the wrong site — and the guest closes it.",
      },
      {
        tip: "«About us» is the most useful block for search. With the district and city in it, you are far more likely to appear for «somsa in Chilanzar».",
      },
      { see: ["site-seo", "menu-item"] },
    ],
  },
  {
    slug: "site-seo",
    section: "site",
    title: "Domain, SEO and favicon",
    lead: "What it takes for Google and Yandex to find the site.",
    keys: ["domain", "seo", "google", "yandex", "favicon", "https"],
    body: [
      {
        p: "Every restaurant runs on its own domain. You buy it (say `b5somsa.uz`), we connect it and set up HTTPS with a certificate that renews itself.",
      },
      { h: "Connecting a domain" },
      {
        steps: [
          "In your registrar's panel point the `A` record at our IP address (we give you the address).",
          "We add the domain in the console.",
          "Wait from 15 minutes to a few hours — DNS has to propagate.",
        ],
      },
      {
        warn: "«The domain is connected but the site does not open» is almost always DNS that has not propagated yet. Clear the browser cache and try again; if it is still down after an hour, write to us.",
      },
      { h: "Being found" },
      {
        list: [
          "Fill in the title and description — that is exactly what Google shows in results.",
          "Every dish is its own page: with a photo and a description it can be found on its own.",
          "The three languages have three addresses (`/ru/`, `/en/`) and they are linked to each other, so Google does not treat them as duplicates.",
          "*Favicon* — the small icon in the browser tab, made from the logo automatically.",
        ],
      },
      { see: ["site-texts", "site-design"] },
    ],
  },
  {
    slug: "banners",
    section: "site",
    title: "Banners",
    lead: "The carousel at the top of the home page: an offer, a new dish, a holiday.",
    keys: ["banner", "slider", "promo image"],
    body: [
      {
        p: "`Menu` → `Banners`: upload an image, write a heading and point it somewhere — an offer page or a single dish.",
      },
      {
        list: [
          "The order can be changed — the first one is seen most.",
          "A banner can be *hidden* rather than deleted: the holiday ends, you switch it on again next year.",
          "You can upload separate images for phone and desktop: a wide image gets cropped on a phone.",
        ],
      },
      {
        tip: "Do not use more than three. Almost nobody sees the fourth, but it still loads — which means a slower site.",
      },
      { see: ["promotions", "site-design"] },
    ],
  },
  {
    slug: "reviews",
    section: "site",
    title: "Guest reviews on the site",
    lead: "Show ratings on the site, or don't — one switch.",
    keys: ["review", "rating", "stars", "feedback"],
    body: [
      {
        p: "After a delivery the customer is asked for a rating. Reviews collect in `Customers` → `Reviews`.",
      },
      {
        p: "Showing them on the site is up to you: `Settings` → `Site` → «Show reviews». When on, the average rating and how many ratings it comes from are displayed.",
      },
      {
        warn: "There is *no* «only show 4 stars and above» setting and there will not be one. A filtered rating is not a rating, it is advertising; a guest who notices once stops believing every other number on the page.",
      },
      { see: ["feedback"] },
    ],
  },
  {
    slug: "vacancies",
    section: "site",
    title: "The jobs page",
    lead: "Hiring: a posting, an application, and the list of applicants.",
    keys: ["vacancy", "jobs", "hiring", "application"],
    body: [
      {
        p: "`Menu` → `Vacancies`: position, pay and requirements. Postings appear on the site at `/vakansiya`.",
      },
      {
        p: "The applicant's name, phone and note arrive here. A new application raises a notice in the panel.",
      },
      {
        tip: "Rather than deleting a posting, set it inactive — next month you switch it on again without rewriting the text.",
      },
      { fig: "vacancies" },
      { see: ["staff-add"] },
    ],
  },

  // ═══════════════════════════════════ Menu
  {
    slug: "categories",
    section: "menu",
    title: "Categories",
    lead: "The sections of the menu. A dish cannot be saved without one, so this comes first.",
    keys: ["category", "section", "group"],
    body: [
      {
        p: "`Menu` → `Categories`: «Main courses», «Burgers», «Drinks». Drag to reorder — the site and the till show exactly this order.",
      },
      {
        list: [
          "Each category has a name in three languages.",
          "A category can be *hidden*: the dishes stay, the section leaves the site. Useful for a seasonal section.",
          "The image is optional — without one only the name is shown.",
        ],
      },
      {
        warn: "Deleting a category that still holds dishes leaves those dishes without one, and they disappear from the site. Move them first.",
      },
      { fig: "categories" },
      { see: ["menu-item"] },
    ],
  },
  {
    slug: "menu-item",
    section: "menu",
    title: "Adding and editing a dish",
    lead: "Name, price, photo, description — and the dish appears on the site, on the till and in the bot at once.",
    keys: ["dish", "product", "price", "add to menu"],
    body: [
      {
        p: "In `Menu`, the `+ New dish` button. Name, category and price are required; the rest can be filled in later.",
      },
      {
        fig: "menu",
        notes: {
          add: "A new dish is added here.",
          uncosted: "How many dishes have no cost. Click to see only those — useful while costing the menu.",
        },
      },
      { h: "The fields" },
      {
        table: {
          head: ["Field", "What it is for"],
          rows: [
            ["*Name*", "In three languages. If Russian and English are empty, the Uzbek name is used."],
            ["*Price*", "In so'm, whole numbers. Tiyin are not used."],
            ["*Old price*", "The struck-through price, to show a discount. Optional."],
            ["*Cost*", "What one portion costs the kitchen. Never shown on the site. Ignored once a tech card exists."],
            ["*Available*", "Turning it off removes the dish from the site (it is not deleted)."],
            ["*Popular*", "Puts it in the «Popular dishes» block on the home page."],
            ["*Tags*", "«Spicy», «Vegetarian» — they work as filters on the site."],
          ],
        },
      },
      {
        warn: "*Deleting* a dish does not damage order history — the name and price are frozen into past orders. But a dish inside a combo cannot be deleted: the system names the sets still holding it.",
      },
      { fig: "menu-item" },
      { see: ["menu-options", "menu-photos", "tech-cards", "menu-combo"] },
    ],
  },
  {
    slug: "menu-options",
    section: "menu",
    title: "Options: sizes and extras",
    lead: "Small/large, extra cheese, choice of sauce — inside one dish.",
    keys: ["option", "size", "extra", "modifier"],
    body: [
      {
        p: "The dish form has an `Options` section. Each group has a name («Size», «Extras») and a rule for choosing.",
      },
      {
        list: [
          "*Pick one* — size, doneness. One choice is the default.",
          "*Pick several* — extras. The maximum can be capped.",
          "Every choice carries a *price change*: `+5 000` or `0`. It can be negative (a small portion costs less).",
        ],
      },
      {
        tip: "An option can draw from the store: attach an ingredient to «extra cheese» and it is deducted on sale. This works *alongside* the dish's own tech card, not instead of it.",
      },
      {
        warn: "Making «Small» and «Large» two separate dishes looks simpler, but then the reports treat them as two products and you cannot see how the dish itself is selling.",
      },
      { see: ["menu-item", "tech-cards"] },
    ],
  },
  {
    slug: "menu-combo",
    section: "menu",
    title: "Combos (sets)",
    lead: "A «family set»: several dishes at one price.",
    keys: ["combo", "set", "bundle", "family"],
    body: [
      {
        p: "A combo is a menu item too, only it contains other dishes. `Menu` → new dish → the `Combo` section, where you pick the members and quantities.",
      },
      {
        p: "The screen shows the sum of the members' own prices next to the combo price — that is, what the guest saves.",
      },
      {
        warn: "A combo has *no tech card of its own and cannot have one*: its cost comes from its members. A card written here as well would deduct the ingredients twice.",
      },
      {
        tip: "On sale, a combo is expanded into its members. A hundred family sets deduct the ingredients of a hundred sets — not zero.",
      },
      { see: ["menu-item", "tech-cards"] },
    ],
  },
  {
    slug: "menu-photos",
    section: "menu",
    title: "Dish photos",
    lead: "Photos sell — and heavy photos slow the site down. Both are handled.",
    keys: ["photo", "image", "picture", "upload"],
    body: [
      {
        p: "The photo is uploaded on the dish form. The system keeps several sizes and serves the right one: small to a phone, large to a desktop.",
      },
      {
        list: [
          "JPG or PNG. Something close to square looks best.",
          "You can upload a large file — the system resizes it.",
          "A dish without a photo still looks fine: the name and a colour are shown instead.",
        ],
      },
      {
        tip: "Shoot in one style: same background, same light, same angle. Then the menu reads as a finished product rather than an album collected from six phones.",
      },
      { see: ["menu-item", "site-design"] },
    ],
  },
  {
    slug: "menu-import",
    section: "menu",
    title: "Importing a menu",
    lead: "From a link or from your old till — without retyping it.",
    keys: ["import", "migrate", "excel", "iiko", "poster"],
    body: [
      { h: "From a link" },
      {
        steps: [
          "`Menu` → `Import` → paste the address of the page your menu is on.",
          "The system reads the page and proposes a list: name, price, category.",
          "You correct the list on screen.",
          "`Add` writes it into the menu. Until then *nothing is saved*.",
        ],
      },
      {
        warn: "Reading takes up to twenty seconds depending on the page. Closing the tab does not stop the job — come back and the result is there.",
      },
      { h: "From another till system" },
      {
        p: "Export Excel/CSV from iiko, r_keeper, Clopos, Poster or Jowi and upload it here. Three kinds of data come across: *ingredients*, *tech cards* and *stock balances*.",
      },
      {
        tip: "Order matters: ingredients first, then tech cards (they find ingredients by name), balances last.",
      },
      { fig: "site-menu" },
      { see: ["pos", "ingredients", "tech-cards"] },
    ],
  },
  {
    slug: "stop-list",
    section: "menu",
    title: "The stop list: what is off today",
    lead: "Take a dish off sale temporarily without removing it from the menu.",
    keys: ["stop list", "sold out", "off sale", "86"],
    body: [
      {
        p: "`Menu` → `Stop list`. The list belongs to *this branch* and to *today only* — nothing is removed from the menu.",
      },
      { h: "Three sources" },
      {
        table: {
          head: ["Who writes it", "How"],
          rows: [
            ["*A person*", "Somebody at the counter presses «Stop». Cleared by hand."],
            [
              "*The till system*",
              "The stop list in iiko/Poster is read every 3 minutes. It cannot be cleared here — the next read brings it back.",
            ],
            ["*The store*", "A dish short of ingredients stops automatically. *Off by default*."],
          ],
        },
      },
      {
        warn: "Only switch the store-based stop on if deliveries and counts are entered regularly. Otherwise the till refuses to sell food that is sitting on the shelf — the most expensive thing this system can do.",
      },
      {
        tip: "The most useful line on the page is *when it was last read*, not a green «connected» badge: a badge goes stale the moment the clock passes it, a read time does not.",
      },
      { fig: "stop-list" },
      { see: ["pos", "stock-stop"] },
    ],
  },
  {
    slug: "promotions",
    section: "menu",
    title: "Offers and promo codes",
    lead: "Two ways to give a discount, and how they combine.",
    keys: ["offer", "promo code", "discount", "coupon"],
    body: [
      {
        p: "`Menu` → `Offers`. The difference: an *offer* is visible on the site and applies itself; a *promo code* has to be typed in by the customer.",
      },
      {
        list: [
          "*Percentage* — «20% off all pizzas».",
          "*Amount* — «10 000 off orders over 50 000».",
          "*Free delivery* — instead of a discount, the delivery fee is dropped.",
          "Each one carries dates, a minimum order and the categories it applies to.",
        ],
      },
      {
        warn: "Promo codes are *never* listed on the site — only offers are. Otherwise every guest would find the largest code and use it.",
      },
      {
        tip: "A discount can push a price below cost. That is what the margin column in the menu is for: it shows in advance which dish a 20% offer sells at a loss.",
      },
      { fig: "promotions" },
      { see: ["loyalty", "reports-sales"] },
    ],
  },
  {
    slug: "recommendations",
    section: "menu",
    title: "Recommendations and upsell",
    lead: "«Add a drink?» — in the basket and on the dish page.",
    keys: ["recommendation", "upsell", "basket", "cross-sell"],
    body: [
      {
        p: "The dish form has a `Recommended dishes` list. It is shown on the dish page and in the basket.",
      },
      {
        p: "If you set nothing, the system suggests on its own: it finds the dishes usually ordered together.",
      },
      {
        tip: "Do not recommend more than three. The fourth suggestion reads as advertising and, in the basket, distracts from paying.",
      },
      { fig: "site-cart" },
      { see: ["menu-item", "crm"] },
    ],
  },

  // ═══════════════════════════════════ Orders
  {
    slug: "order-flow",
    section: "orders",
    title: "Order statuses",
    lead: "From arrival to handover: what each status means and who changes it.",
    keys: ["order", "status", "on the way", "delivered"],
    body: [
      {
        table: {
          head: ["Status", "Meaning"],
          rows: [
            ["`New`", "The order arrived and nobody has seen it. This is when the sound plays."],
            ["`Confirmed`", "Accepted. If a till system is connected, the order goes to it at exactly this moment."],
            ["`Preparing`", "In the kitchen."],
            ["`On the way`", "Handed to a courier. This is what the customer sees on the tracking page."],
            ["`Delivered`", "Done. From this point the money counts as revenue."],
            ["`Cancelled`", "With a reason. There is no cancelling without one."],
          ],
        },
      },
      {
        p: "Every change is stamped with a time — the card shows when the order was accepted and how long it took.",
      },
      {
        warn: "The order goes to the till on *confirm*, not on arrival. The reason: a till is somebody else's bookkeeping, and a mistake sent there has to be cancelled by hand.",
      },
      { fig: "orders" },
      { see: ["order-accept", "order-cancel", "pos"] },
    ],
  },
  {
    slug: "order-accept",
    section: "orders",
    title: "Accepting an order, and the sound",
    lead: "What happens when an order arrives, and what to do when there is no sound.",
    keys: ["sound", "alert", "new order", "accept", "not audible"],
    body: [
      {
        p: "A new order plays a sound and raises a banner at the bottom right. The sound repeats until `Accept` is pressed — deliberately: a single chime is not heard on a busy evening.",
      },
      { h: "No sound" },
      {
        steps: [
          "Check the `Sound on` button at the bottom left of the panel.",
          "The browser has to allow sound. Click anywhere on the page once — browsers only let audio through after a first click.",
          "Check that the computer's own volume is not muted.",
        ],
      },
      {
        tip: "There is a «Quiet for 5 minutes» button — useful while talking to a guest. The sound comes back on its own.",
      },
      { see: ["order-flow", "kds"] },
    ],
  },
  {
    slug: "order-cancel",
    section: "orders",
    title: "Cancelling an order and refunding",
    lead: "A reason is required — and money paid online does not come back on its own.",
    keys: ["cancel", "refund", "reason"],
    body: [
      {
        p: "The order card has a `Cancel` button. It will not save without a reason: an order cancelled without one tells nobody anything a month later.",
      },
      {
        warn: "Cancelling an order paid online *does not refund it automatically*. The money stays with the payment provider and is refunded from their dashboard. The panel says so plainly — the word «cancelled» does not mean the customer's card got the money back.",
      },
      {
        p: "A till receipt has a different action — a *refund*: the sale stays, the money changes. The reports show this separately from «cancelled».",
      },
      { see: ["payments", "till-shift"] },
    ],
  },
  {
    slug: "preorder",
    section: "orders",
    title: "Pre-orders",
    lead: "The customer orders «for 19:00».",
    keys: ["preorder", "scheduled", "for later", "tomorrow"],
    body: [
      {
        p: "`Settings` → `Orders` turns pre-orders on. Two settings: how far ahead at the earliest (an hour, say) and how many days ahead at the latest.",
      },
      {
        p: "Such an order is marked in the list and rises to the top *when its time comes* — an order placed at 10 for 19:00 does not distract the kitchen all day.",
      },
      {
        tip: "Busy hours can be closed off: if you do not want pre-orders over lunch, that is limited separately from opening hours.",
      },
      { fig: "site-booking" },
      { see: ["order-flow", "reservations"] },
    ],
  },
  {
    slug: "reservations",
    section: "orders",
    title: "Table reservations",
    lead: "The guest books a table on the site, you confirm it.",
    keys: ["booking", "table", "reservation"],
    body: [
      {
        p: "`Orders` → `Reservations`. The guest leaves a date, a time, a number of people and a phone number.",
      },
      {
        table: {
          head: ["Status", "Meaning"],
          rows: [
            ["`New`", "Waiting — you have to confirm it."],
            ["`Confirmed`", "The table is held. The customer gets an SMS."],
            ["`Seated`", "The guest arrived."],
            ["`Done` / `Cancelled`", "Closed. Cancelling needs a reason."],
          ],
        },
      },
      {
        p: "`Settings` → `Reservations` holds the floor plan: tables, where they are and how many they seat.",
      },
      {
        warn: "Confirming sends an SMS. With no SMS provider configured the guest gets no confirmation — and usually phones.",
      },
      { fig: "reservations" },
      { see: ["sms", "till-hall"] },
    ],
  },
  {
    slug: "qr-menu",
    section: "orders",
    title: "QR menu: ordering from the table",
    lead: "A QR code on every table — the guest opens the menu on their phone and orders.",
    keys: ["qr", "table", "qr menu", "code"],
    body: [
      {
        p: "`Menu` → `QR codes` generates a code per table and a print-ready PDF.",
      },
      {
        p: "The guest scans it, the menu opens, they order. The order arrives with the table number, so the waiter knows where to take it.",
      },
      {
        tip: "The code has to be *stuck to the table*, not slipped into a menu folder: the folder moves to another table and the order goes to the wrong one.",
      },
      {
        warn: "A guest ordering by QR also counts as «dining in», but this is not a till receipt. The `Dining room sales` list only shows receipts opened on the till.",
      },
      { fig: "qr" },
      { see: ["till-hall", "order-flow"] },
    ],
  },

  // ═══════════════════════════════════ Delivery
  {
    slug: "delivery-setup",
    section: "delivery",
    title: "Turning delivery on",
    lead: "Minimum order, free-delivery threshold and how the fee is worked out.",
    keys: ["delivery", "minimum order", "free delivery"],
    body: [
      {
        p: "`Settings` → `Delivery`. Switch it on, then choose how the fee is calculated: *by distance* or *by zones on the map*.",
      },
      {
        table: {
          head: ["Setting", "What it means"],
          rows: [
            ["*Minimum order*", "Below this we do not deliver. The basket tells the guest how much is missing."],
            ["*Free from*", "Above this amount delivery is free. Left empty means never."],
            ["*How the fee is calculated*", "Distance or zones. The other one's fields are hidden but the data is kept — change your mind and you do not redraw."],
          ],
        },
      },
      {
        warn: "Delivery is configured *per branch*. With several branches the page warns about coverage, because mistakes here are silent: the form looks filled in, nothing errors, and the result shows up somewhere else entirely — as a branch that simply never gets orders.",
      },
      { h: "The four most common mistakes" },
      {
        list: [
          "*Delivery off* — the branch takes no part in delivery at all.",
          "*No point on the map* — distance is measured from it, so the calculation cannot work.",
          "*Maximum distance = 0* — this means «no limit», not «we do not deliver». Across several branches this is the dangerous one: a single branch takes the whole city.",
          "*Base fee and per-km both 0* — delivery becomes free.",
        ],
      },
      { see: ["delivery-zones", "delivery-radius", "map-provider"] },
    ],
  },
  {
    slug: "delivery-zones",
    section: "delivery",
    title: "Drawing delivery zones",
    lead: "You draw the boundary on the map and price each zone — and the guest sees the same thing.",
    keys: ["zone", "polygon", "boundary", "map", "area", "fee"],
    body: [
      {
        p: "`Settings` → `Delivery` → choose *zones*. A map opens below.",
      },
      { h: "Drawing a zone" },
      {
        steps: [
          "Press `New zone`.",
          "Click along the boundary on the map. At least three points are needed.",
          "Close the shape by joining the last point to the first.",
          "Name the zone («Centre», «Chilanzar») and choose how it is priced.",
          "Save. The zone appears on the site immediately.",
        ],
      },
      { h: "A zone is priced one of two ways" },
      {
        list: [
          "*Fixed fee* — one amount for every address inside it. The simplest, and the clearest to a guest.",
          "*Per kilometre* — a base fee plus each km, measured from the restaurant.",
        ],
      },
      {
        warn: "A zone counts as working with *at least three points*. If any working zone exists, the distance settings (base fee, per-km, maximum) are ignored entirely, and nothing outside the zones is delivered.",
      },
      {
        warn: "Having drawn a zone, *save it*. A drawn but unsaved shape stays on screen and looks entirely real.",
      },
      {
        tip: "At checkout the guest sees the zones on the map with a legend under it: which zone, and what it costs. With no address picked yet the map fits the whole coverage — so a guest can tell at a glance whether you reach them.",
      },
      { see: ["delivery-radius", "map-provider", "delivery-setup"] },
    ],
  },
  {
    slug: "delivery-radius",
    section: "delivery",
    title: "Pricing by distance",
    lead: "When there is no time to draw zones: a base fee plus each kilometre.",
    keys: ["distance", "radius", "km", "fee calculation"],
    body: [
      { p: "Choose *distance* as the method. Three numbers are needed:" },
      {
        table: {
          head: ["Field", "Example"],
          rows: [
            ["*Base fee*", "10 000 — added to every order"],
            ["*Per km*", "3 000 — straight-line distance from the restaurant, rounded up"],
            ["*Maximum distance*", "12 km — beyond this we do not deliver"],
          ],
        },
      },
      {
        p: "The distance is *as the crow flies*, not by road. Set the per-km figure with the real detour in mind.",
      },
      {
        warn: "*A maximum distance of 0* means «no limit». Most people read it as «we do not deliver», and in a chain that setting hands one branch the entire city.",
      },
      { see: ["delivery-zones", "delivery-setup"] },
    ],
  },
  {
    slug: "map-provider",
    section: "delivery",
    title: "Maps: 2GIS, Yandex or Google",
    lead: "Which map to use and where the key comes from.",
    keys: ["map", "2gis", "yandex", "google", "api key"],
    body: [
      {
        p: "`Settings` → `Map` picks the provider. The default is *2GIS* — its addresses are the most complete in Uzbekistan.",
      },
      {
        table: {
          head: ["Provider", "Where the key is", "Cost"],
          rows: [
            ["*2GIS*", "dev.2gis.com — free sign-up", "Effectively free at this volume"],
            ["*Yandex*", "developer.tech.yandex.ru", "Effectively free at this volume"],
            ["*Google*", "console.cloud.google.com", "*Paid* — the bill is yours"],
          ],
        },
      },
      {
        warn: "Each provider has *its own key field*. Try Yandex and go back to 2GIS and the keys will not get crossed — that is exactly how you end up with an empty map and an error in a console nobody in a restaurant reads.",
      },
      {
        warn: "A map key *is not a secret and cannot be one* — it is handed to the browser, as it is in every mapping SDK. The protection is the *domain restriction* in the provider's dashboard: allow the key only on your own domain.",
      },
      {
        tip: "A grey box instead of a map on an old phone means 2GIS needs WebGL. If many of your customers are on old devices, switch to Yandex.",
      },
      { see: ["delivery-zones"] },
    ],
  },
  {
    slug: "couriers",
    section: "delivery",
    title: "Couriers",
    lead: "Creating an account, assigning orders, and the cash a courier is carrying.",
    keys: ["courier", "driver", "account"],
    body: [
      {
        p: "`Money and team` → `Couriers`: name, phone, login and password. A courier *cannot register themselves* — only you issue the account.",
      },
      {
        p: "A courier has three states: `Off duty`, `Free`, `Busy`. You assign an order from the panel.",
      },
      { h: "Cash in hand" },
      {
        p: "On a cash delivery the money stays with the courier and reaches the till when it is *handed in*. That is why the cash report shows it separately — anybody investigating a shortfall asks for that figure first.",
      },
      {
        warn: "A courier handing in cash is *not an expense*: it is cash collected on our behalf arriving in the till. Recording it as an expense means subtracting the restaurant's own revenue from itself.",
      },
      { fig: "couriers" },
      { see: ["courier-app", "finance", "delivery-external"] },
    ],
  },
  {
    slug: "courier-app",
    section: "delivery",
    title: "The courier app",
    lead: "A page on the phone: orders, address, hand-off to maps, the day's total.",
    keys: ["courier app", "phone", "pwa"],
    body: [
      {
        p: "The courier signs in at `yoursite.uz/kuryer` — nothing to install. Add it to the phone's home screen and it opens like an app.",
      },
      {
        list: [
          "The orders assigned to them: address, phone, amount, payment method.",
          "Tapping the address opens the phone's map app and routes there.",
          "`Delivered` closes the order.",
          "At the end of the day: how many were delivered and how much cash is in hand.",
        ],
      },
      {
        warn: "*Live courier tracking on a map does not exist yet.* The customer sees the «on the way» status, but the dot does not move.",
      },
      {
        tip: "Location permission is asked for and used only while an order is assigned. If the phone refuses, the app still works.",
      },
      { fig: "courier-login" },
      { see: ["couriers"] },
    ],
  },
  {
    slug: "delivery-external",
    section: "delivery",
    title: "External delivery services",
    lead: "No couriers of your own — hand the order to a service.",
    keys: ["yandex delivery", "third-party courier", "logistics"],
    body: [
      {
        p: "`Settings` → `Delivery` → `External service` connects a provider. The order is handed over after it is confirmed.",
      },
      {
        p: "With a small number of orders a day this can cost less than employing a courier.",
      },
      {
        tip: "The two can be combined: your own courier takes the near addresses, the service takes the far ones.",
      },
      { see: ["couriers"] },
    ],
  },

  // ═══════════════════════════════════ Till and dining room
  {
    slug: "till-setup",
    section: "till",
    title: "Installing the till",
    lead: "Download the program, register the device, make the first sale.",
    keys: ["till", "pos", "install", "download"],
    body: [
      {
        steps: [
          "On the till computer, download the program from `keel.uz/download`.",
          "Run it — on first launch it asks which restaurant it belongs to.",
          "In the panel take the device code from `Settings` → `Till` and enter it.",
          "The cashier signs in with their PIN.",
        ],
      },
      {
        p: "The same downloaded file works for *every restaurant* — the server address is not compiled in, it is asked for on first launch.",
      },
      {
        warn: "The till works without internet: a sale is kept on the device and sent when the connection returns, so the queue does not stop. Every sale carries its own id, which means a retry cannot charge twice.",
      },
      { see: ["till-shift", "printer-connect", "till-payment"] },
    ],
  },
  {
    slug: "till-shift",
    section: "till",
    title: "Shifts: opening, closing, X and Z reports",
    lead: "So the cash adds up at the end of the day.",
    keys: ["shift", "till", "z report", "x report", "cash up", "shortfall"],
    body: [
      {
        steps: [
          "At the start of the day, `Open shift` — enter the float already in the drawer.",
          "Sales are counted through the day on their own.",
          "At any point an `X report` shows what should be there now and *changes nothing*.",
          "At the end of the day, `Close shift` — count the drawer and enter what you found.",
          "The `Z report` comes back in the closing response — print it.",
        ],
      },
      {
        warn: "The product is the *difference*, not the total. The system works out what is expected and freezes it, you enter what you counted, and if they differ a *reason is required*. A discrepancy without one is not saved.",
      },
      {
        p: "The Z report has a separate «of which debts repaid» line. Money for yesterday's debt paid today lands in today's drawer while the sale was yesterday's — without that line it would be a difference nobody could explain.",
      },
      {
        warn: "A branch has *one open shift*. A second one is refused: with two open, «how much should be in the drawer» has no answer.",
      },
      { see: ["finance", "till-payment"] },
    ],
  },
  {
    slug: "till-hall",
    section: "till",
    title: "The dining room: tables, splitting and merging bills",
    lead: "The waiter's screen: open a table, add dishes, split the bill.",
    keys: ["dining room", "table", "waiter", "split bill"],
    body: [
      {
        p: "The floor plan is shown on the room screen. Tapping a table opens a bill; dishes go on it and the table becomes occupied.",
      },
      {
        list: [
          "*Split* — one bill into several. Splitting off everything is not allowed.",
          "*Merge* — two tables into one bill. The absorbed bill is *voided, not deleted*: its number and lines stay in history.",
          "*Service charge* — set per branch and added *only to a table*. The percentage is copied onto the bill when the table is seated, so changing it in the evening leaves open tables on the old rate.",
        ],
      },
      {
        tip: "Closed bills can be looked up on the till itself (`Sales` → `Closed`) — «print it again» does not need a panel login. Today only, and only this branch.",
      },
      { fig: "checks" },
      { see: ["till-shift", "qr-menu"] },
    ],
  },
  {
    slug: "till-payment",
    section: "till",
    title: "Payment methods on the till",
    lead: "Cash, card terminal, QR payment and debt.",
    keys: ["payment", "cash", "card", "qr", "debt", "payme", "click"],
    body: [
      {
        table: {
          head: ["Method", "How it works"],
          rows: [
            ["*Cash*", "The cashier enters the amount, change is calculated."],
            ["*Card terminal*", "Through the bank's terminal. The system *cannot see it* — the cashier marks it."],
            ["*QR (Payme / Click / Uzum)*", "A QR appears on screen, the guest pays on their own phone, the bill closes itself."],
            ["*Debt*", "Not a payment — a record of not paying. A customer is required."],
          ],
        },
      },
      {
        warn: "With a QR payment the bill *does not close until the payment is confirmed*. This slowness is deliberate: marking it paid on selection works in every test, and in a queue it hands over food for a cancelled, expired, or somebody else's payment.",
      },
      {
        p: "*Debt* is refused without a customer. An anonymous debt is the notebook by the counter: money that appears on nobody's card and that nobody will ask about. At the till the customer is found by phone number.",
      },
      {
        tip: "Debts are also settled at the till — the money arrives in the drawer, so the person taking it is the person standing at the drawer. Sending a cashier to find somebody with a panel login is how a manager's password ends up written down beside the till.",
      },
      { see: ["payments", "till-shift", "crm"] },
    ],
  },
  {
    slug: "kds",
    section: "till",
    title: "The kitchen screen (KDS)",
    lead: "A cook answers one question all evening: what do I cook next.",
    keys: ["kds", "kitchen", "cook", "ready"],
    body: [
      {
        p: "`yoursite.uz/staff/kitchen` — for a tablet or monitor in the kitchen. The employee signs in with their own account.",
      },
      {
        p: "The screen has the ticket, the waiting time and one button. Money, customer, address and filters are *deliberately absent*: a cook who has to read past them to find the dish stops reading the screen.",
      },
      {
        warn: "KDS access is granted *per person* (`Staff` → employee → «Kitchen screen»), and it is off by default. On a shared tablet anybody could press «Ready» — taking the ticket off the kitchen screen and telling the panel the kitchen cooked it.",
      },
      {
        p: "«Ready» is not a new status, it is a timestamp. The order's status does not change; a «Kitchen ready» mark appears in the panel. Send the order back and the mark is cleared.",
      },
      { fig: "kitchen-login" },
      { see: ["staff-roles", "order-flow"] },
    ],
  },
  {
    slug: "kiosk",
    section: "till",
    title: "Kiosk: the self-order screen",
    lead: "A touch screen in the room — the guest orders without queueing.",
    keys: ["kiosk", "self service", "touch screen"],
    body: [
      {
        p: "`yoursite.uz/kiosk` — a screen tied to a branch. The panel issues a kiosk code and the screen registers with it once.",
      },
      {
        list: [
          "The menu is shown as large buttons and the order gets a number.",
          "Payment by QR or through the terminal.",
          "The order goes straight to the kitchen.",
        ],
      },
      {
        tip: "Separate rules apply so the screen does not freeze under rapid taps: zoom is disabled and a double tap does not create a second order.",
      },
      { fig: "kiosk-login" },
      { see: ["kds", "till-setup"] },
    ],
  },

  // ═══════════════════════════════════ Printers
  {
    slug: "printer-connect",
    section: "printers",
    title: "Connecting a printer",
    lead: "Receipt printer at the till, kitchen printer in the kitchen — both connect the same way.",
    keys: ["printer", "connect", "receipt printer", "usb", "network"],
    body: [
      {
        p: "A printer connects *to the till computer* (by USB or over the network) and is then chosen in the till program. The panel does not see printers directly.",
      },
      {
        steps: [
          "Install the printer in Windows as usual and print a test page. If that does not work, nothing after it will.",
          "In the till program open `Settings` → `Printers`.",
          "Pick the printer from the list — the names come from Windows itself.",
          "Give each one a job: `Receipt`, `Kitchen` or `Bar`.",
          "Press `Test print`.",
        ],
      },
      {
        tip: "You do *not* need to share the printer on the network (`net share`). The program asks Windows for the printer name; a shared path is only the fallback.",
      },
      {
        warn: "Do not give one printer two jobs. A restaurant printing the kitchen ticket on the same roll as the guest's receipt ends up reprinting both.",
      },
      { see: ["printer-kitchen", "printer-problems"] },
    ],
  },
  {
    slug: "printer-kitchen",
    section: "printers",
    title: "Kitchen printers: what prints where",
    lead: "Hot section, cold section and bar — each gets its own ticket.",
    keys: ["kitchen printer", "station", "bar", "routing"],
    body: [
      {
        p: "Each category is attached to a printer: «Drinks» → the bar, «Hot dishes» → the kitchen. When an order arrives, each printer gets *only its own* lines.",
      },
      {
        list: [
          "The ticket carries the table or order number, the time and the dish names — no prices.",
          "The customer's note («no onions») prints on this ticket.",
          "If dishes are added to an order, only the *new* lines print.",
        ],
      },
      {
        warn: "A category with no printer attached prints *nowhere*. Check this when you add a category — it is the most common cause of a ticket that never came out.",
      },
      { see: ["printer-connect", "printer-problems", "categories"] },
    ],
  },
  {
    slug: "printer-problems",
    section: "printers",
    title: "The ticket did not print",
    lead: "The quietest failure in the system: the order is on screen, the sale is in the report, and the food was never cooked.",
    keys: ["not printing", "printer broken", "????"],
    body: [
      {
        p: "The panel has a *print queue* at `Settings` → `Printers`: what was sent and what did not come out. When something is stuck, a banner appears in the panel.",
      },
      { h: "Check in this order" },
      {
        steps: [
          "The printer is on and has paper.",
          "The till program is open — printing goes through it, not through the panel.",
          "Windows does not have the printer marked offline.",
          "In the print queue, press `Send again`.",
        ],
      },
      {
        warn: "A job that already printed is not sent again — it answers «not found». This is deliberate: a kitchen ticket printed twice is a dish cooked twice.",
      },
      { h: "Question marks in the ticket (????)" },
      {
        p: "This is an encoding problem: the printer does not know Uzbek or Russian letters. Change the printer's code page in the till program — usually `CP866` or `CP1251` is right. The change is written to the log.",
      },
      { see: ["printer-connect", "printer-kitchen"] },
    ],
  },

  // ═══════════════════════════════════ Store and costing
  {
    slug: "stock-intro",
    section: "stock",
    title: "How the store works",
    lead: "The chain: ingredient → tech card → delivery → sale → write-off → count. Each step gives its own number.",
    keys: ["stock", "store", "inventory", "cost", "start"],
    body: [
      {
        p: "The store section answers two questions: *what a portion costs* and *what is left on the shelf*. Both come out of one chain.",
      },
      {
        steps: [
          "*Ingredients* — what you buy and for how much (kilo, litre, piece).",
          "*Tech cards* — what goes into one portion. The cost comes from here.",
          "*Deliveries* — the invoice: what arrived, from whom, at what price. Prices update from here.",
          "*Sales* — dishes sold are deducted through their cards. This is automatic.",
          "*Write-offs* — spoiled, spilled, staff meals.",
          "*Stocktake* — the count. «What should be there» is measured from it.",
        ],
      },
      {
        warn: "Store screens *require a branch*. With several, pick one at the top; a single-branch restaurant is never asked.",
      },
      {
        tip: "The order to start in: ingredients → tech cards → a first count → deliveries entered regularly. Without a count, «what should be there» is an estimate over every delivery ever made.",
      },
      { fig: "stock" },
      { see: ["ingredients", "tech-cards", "stocktake", "stock-balances"] },
    ],
  },
  {
    slug: "ingredients",
    section: "stock",
    title: "Adding an ingredient",
    lead: "The catalogue: what you buy, in what unit, and at what price.",
    keys: ["ingredient", "unit", "price", "kg"],
    body: [
      {
        p: "`Store` → `Ingredients`. Each one has a name, a unit and a *purchase price*.",
      },
      {
        fig: "ingredients",
        notes: {
          unit: "The unit: kilo, litre or piece. Tech cards then use grams, millilitres and pieces respectively.",
          expected: "«Should be there» — the running figure since the last count.",
        },
      },
      {
        p: "The price is entered *in the unit you buy in*: an invoice says «one kilo, 90 000», and nobody writes down a price per gram. The conversion happens in the tech card.",
      },
      {
        warn: "There are exactly three units: *kilo, litre, piece*. Free text like «bunch» or «crate» cannot be entered — it produces a card that cannot be costed and a cost that looks calculated.",
      },
      {
        list: [
          "*Minimum stock* — below it the ingredient is flagged «running out». *Zero means «do not warn me»*, not «warn me at zero».",
          "*Store* — with several stores, where this branch keeps it. Empty is the undivided store.",
          "*Note* — where it is bought, which grade.",
        ],
      },
      {
        tip: "Editing a price by hand takes effect *from today*, while a delivery carries its own date. A late invoice lands in the history on its own day but does not change today's price.",
      },
      { see: ["tech-cards", "purchases", "shopping-list"] },
    ],
  },
  {
    slug: "tech-cards",
    section: "stock",
    title: "Tech cards: costing a dish",
    lead: "What goes into one portion — and from it, both the cost and what leaves the store.",
    keys: ["tech card", "recipe", "costing", "cost"],
    body: [
      {
        p: "`Store` → `Tech cards` → `Dishes`. For each dish you list ingredients and quantities.",
      },
      {
        fig: "tech-cards",
        notes: {
          tabs: "Two sections: preps (semi-finished items the kitchen makes itself) and dish cards.",
          dishes: "The cards for menu dishes are here.",
          add: "Add a new prep.",
          batch: "What one batch costs — a figure you can check against a pot.",
          rate: "The resulting price of one gram — that is what dishes are charged.",
        },
      },
      { h: "Writing a card" },
      {
        steps: [
          "Open the `Dishes` tab and find the dish. The «no card» filter shows what is left to do.",
          "Press `Add` — choose an ingredient and enter the quantity.",
          "Quantities are *gross*: the weight that leaves the store, not what ends up on the plate. A kilo of potatoes costs a kilo, peel included.",
          "The cost and margin update as you type.",
          "Save.",
        ],
      },
      {
        warn: "Once there is a card, the cost typed on the menu is *not used*. One number cannot have two sources — the card is recalculated on every read at today's ingredient prices.",
      },
      {
        warn: "*An incomplete card does not cost the dish at all*. If one of its ingredients has been deleted, the cost becomes zero — not smaller. Otherwise a deleted ingredient would make a dish look more profitable, and on a margin screen that reads as good news.",
      },
      {
        p: "An ingredient in use cannot be deleted: the system names the dishes still holding it.",
      },
      { fig: "tech-cards-dishes" },
      { see: ["prep-cards", "ingredients", "menu-item", "stock-balances"] },
    ],
  },
  {
    slug: "prep-cards",
    section: "stock",
    title: "Preps: sauces, dough, fillings",
    lead: "What the kitchen makes itself — written once, used by the gram everywhere.",
    keys: ["prep", "semi-finished", "sauce", "dough", "filling"],
    body: [
      {
        p: "A sauce, a stock, a dough, sushi rice — none of them are on the menu, but they all leave the store. They live in `Store` → `Tech cards` → `Preps`.",
      },
      {
        p: "*This is the entire point of the section*: a kitchen with six sauces and forty dishes otherwise writes the same tomatoes into seven places — and the seven copies stop agreeing within a month.",
      },
      { h: "Writing a prep" },
      {
        steps: [
          "`New prep` — give it a name: «Somsa filling», «White sauce».",
          "Choose the unit: kilo (used by the gram), litre (by the millilitre) or piece.",
          "List the ingredients and the quantities *for one batch*.",
          "Enter the *yield*: how much one batch produces.",
          "Save. It now appears in dish cards like any other ingredient.",
        ],
      },
      { fig: "tech-card-prep" },
      {
        warn: "*The yield is where a card is honest.* Three kilos of tomatoes that boil down to two yield 2000 g, not 3000. An overstated yield makes every dish containing that sauce look cheap.",
      },
      {
        warn: "Without a yield a gram of the prep *costs nothing*, which means every dish using it stays uncosted.",
      },
      {
        p: "A prep has *no price to type*: its price is what a batch costs divided by what it yields. That is why the price field is not on screen at all.",
      },
      {
        tip: "A prep can contain another prep. The costing is worked out in passes.",
      },
      { h: "Are preps counted?" },
      {
        p: "Usually *no*: nobody counts «sauce», they count tomatoes, and a dish with sauce reads as a dish with tomatoes. For a chain with a central kitchen there is a «Made in batches» switch — with it on, the prep is counted as a real thing on a shelf and transferred between branches.",
      },
      { see: ["tech-cards", "production", "stocktake"] },
    ],
  },
  {
    slug: "purchases",
    section: "stock",
    title: "Deliveries: entering an invoice",
    lead: "What arrived, from whom and at what price. Prices flow from here into the menu's costing.",
    keys: ["delivery", "invoice", "purchase", "price"],
    body: [
      {
        p: "`Store` → `Deliveries` → `New`. Date, supplier, and lines: ingredient, quantity, unit price.",
      },
      {
        list: [
          "The invoice's *own total* wins: a discount given at the door is on none of the lines.",
          "If a price changed, it is written into the ingredient's price history *dated by the invoice*.",
          "If it did not change, nothing is added to the history — correcting a name should not leave a «price changed» record behind.",
        ],
      },
      { h: "Payment and debt" },
      {
        p: "Every invoice has a «paid» flag. Unpaid ones show as debt to the supplier (`Store` → `Suppliers`).",
      },
      {
        warn: "*Editing* and *deleting* a delivery behave differently. Editing removes the prices that invoice wrote and writes them again. Deleting does not restore prices — it only says «this line should not be here», and the food may already have been costed at that price.",
      },
      {
        tip: "Hand-entered prices are never touched. They cannot say which invoice they came from, and guessing would silently erase a deliberate correction.",
      },
      { fig: "purchases" },
      { see: ["suppliers", "ingredients", "stock-balances"] },
    ],
  },
  {
    slug: "suppliers",
    section: "stock",
    title: "Suppliers and debts",
    lead: "Who you buy from, and who you still owe.",
    keys: ["supplier", "debt", "payment"],
    body: [
      {
        p: "`Store` → `Suppliers`. For each: how much was bought in the period and *how much is still owed*.",
      },
      {
        warn: "Debt is *not tied to the period*: an unpaid March invoice is still a debt in May. The period filter only limits the purchasing figure.",
      },
      {
        p: "Naming a supplier is *optional*: a trip to the bazaar has no supplier, and making the field required means the delivery simply never gets entered.",
      },
      {
        tip: "Renaming a supplier does not change old invoices — they carry a frozen copy of the name.",
      },
      { fig: "suppliers" },
      { see: ["purchases", "shopping-list"] },
    ],
  },
  {
    slug: "writeoffs",
    section: "stock",
    title: "Write-offs: spoiled, spilled, staff meals",
    lead: "Product that left without being sold. A reason is required.",
    keys: ["write-off", "waste", "spoiled", "spilled"],
    body: [
      {
        p: "`Store` → `Write-offs` → ingredient, quantity and a *reason*. A record without one tells nobody anything a month later.",
      },
      {
        p: "A write-off is valued *at that day's price* and frozen — a later price change does not rewrite last month's waste figure.",
      },
      {
        tip: "A prep is written off at what its card says. So that one spoiled pot of sauce is not written off as nothing.",
      },
      { fig: "writeoffs" },
      { see: ["stock-balances", "reports-sales"] },
    ],
  },
  {
    slug: "transfers",
    section: "stock",
    title: "Transfers: branch to branch",
    lead: "Down in one store, up in another — neither a write-off nor a delivery.",
    keys: ["transfer", "move", "branch"],
    body: [
      {
        p: "`Store` → `Transfers`. From which store to which, which ingredient and how much.",
      },
      {
        warn: "A transfer is *neither a write-off nor a delivery*. Recorded as a write-off it adds a reason to the waste report that nobody wasted; recorded as a delivery it puts a purchase price into the history and makes every dish with that ingredient dearer.",
      },
      {
        warn: "*The units have to match*: transferring kilograms into litres leaves both balances consistent and both wrong — and nothing afterwards will notice.",
      },
      {
        p: "The total is called «transferred» and does not enter the financial report's expenses: value is carried, not created.",
      },
      { fig: "transfers" },
      { see: ["production", "stock-balances"] },
    ],
  },
  {
    slug: "production",
    section: "stock",
    title: "Central kitchen: producing a batch",
    lead: "The production kitchen makes 40 kg of sauce on Monday and sends it to three branches.",
    keys: ["production", "batch", "central kitchen", "commissary"],
    body: [
      {
        p: "This section is only for restaurants with a *central kitchen*. A single-kitchen restaurant opens the page once, reads that it is not for them, and never returns.",
      },
      { h: "Two halves, and neither works alone" },
      {
        steps: [
          "On the prep, switch on «Made in batches». It now counts as a real thing on a shelf and can be transferred.",
          "Set the store's type to «production» — batches are only made there.",
          "`Store` → `Production` → record the batch: what, and how much came out.",
        ],
      },
      {
        warn: "Flag on, no document → the shelf never fills and the balance goes negative. Document, no flag → the ingredients are deducted *twice*: here, and again when the dish is sold. The second one surfaces weeks later at a count and looks like the fault of whoever counted.",
      },
      {
        p: "What a batch consumed is *frozen into the document*: the card changes, but March's batch took what it took in March.",
      },
      { fig: "production" },
      { see: ["prep-cards", "transfers"] },
    ],
  },
  {
    slug: "stocktake",
    section: "stock",
    title: "Stocktake: counting",
    lead: "You write down what is on the shelf — and from that moment «should be there» is measured.",
    keys: ["stocktake", "count", "inventory", "shortfall"],
    body: [
      {
        p: "`Store` → `Stocktake`. The ingredient list appears and you enter the counted quantity for each.",
      },
      {
        p: "The expected figure *is not shown until you enter yours*. An empty box next to «should be 9.4» is a sheet with 9.4 written into it.",
      },
      {
        warn: "Where there is a difference, a *note is required*. A discrepancy saved quietly is a shortfall erased by the person who may have caused it.",
      },
      { h: "Counting on a phone" },
      {
        p: "An employee signs in at `yoursite.uz/staff/stock` and counts standing in the store. When counting happens in the store and writing happens in the office, the number written twice is wrong on the second pass — and that error lands in the *difference* column, where it is indistinguishable from a shortfall.",
      },
      {
        warn: "This permission is granted *separately* (`Staff` → employee → «Store»). A count writes the baseline every later shortfall is measured against, which means a saved count silently forgives everything that went missing before it.",
      },
      {
        p: "*Preps are not counted*: a prep is a pot cooked this morning, and what it was made from is already in the count of its ingredients. Counting both would subtract the tomatoes twice.",
      },
      { fig: "stocktake" },
      { see: ["stock-balances", "staff-roles"] },
    ],
  },
  {
    slug: "stock-balances",
    section: "stock",
    title: "Balances and «running out»",
    lead: "Where «should be there» comes from, and what it does not mean.",
    keys: ["balance", "remaining", "running out", "negative"],
    body: [
      {
        p: "`Store` → `Balances`. For each ingredient: the last count + deliveries − what the cards account for − write-offs.",
      },
      {
        warn: "This is an *estimate*, not a measurement. It drifts exactly as far as the kitchen drifts from its cards, and further the older the last count is. That is why the screen states the date of the last count.",
      },
      { h: "A negative balance" },
      {
        p: "Usually one of two things: *a delivery was not entered*, or *the card's quantity is larger than reality*. Count that ingredient first, then check the card.",
      },
      {
        p: "*Running out* is when «should be there» drops below the minimum. Zero means «do not warn me», and there is no alarm here: this is «bear it in mind at the next order».",
      },
      { fig: "stock" },
      { see: ["stocktake", "shopping-list", "tech-cards"] },
    ],
  },
  {
    slug: "shopping-list",
    section: "stock",
    title: "The shopping list",
    lead: "What is running out and who it is bought from — on one page.",
    keys: ["shopping", "buying", "order list"],
    body: [
      {
        p: "`Store` → `Shopping list`. Ingredients below their minimum, *grouped by the last supplier*.",
      },
      {
        p: "The *last* one, not the most frequent: a restaurant that changed butchers needs to be told the new one, while a year's tally would name the old one for exactly the period the answer is most wrong.",
      },
      {
        tip: "A negative balance is treated as an empty shelf. Half of any negative is measurement error, and ordering against it would double the order at the moment the numbers are least trustworthy.",
      },
      { fig: "shopping" },
      { see: ["stock-balances", "purchases"] },
    ],
  },
  {
    slug: "stock-stop",
    section: "stock",
    title: "The store-based stop list",
    lead: "Stopping dishes automatically when ingredients run out — and why it is off by default.",
    keys: ["stop list", "stock", "automatic", "out of stock"],
    body: [
      {
        p: "`Menu` → `Stop list` → «Stop by store». With it on, a dish short of ingredients cannot be sold on the till or on the site.",
      },
      {
        warn: "*Off by default, and deliberately.* Refusing a sale is the most expensive thing this system can do, and the balance behind it is an estimate: a restaurant that has not entered Tuesday's invoice would have its till refusing food that is on the shelf.",
      },
      {
        p: "*An uncounted store stops nothing.* Zero has two meanings — «none» and «nobody said» — and only the first is grounds for refusal. This is the one guard that keeps the menu from emptying on the day the feature is switched on.",
      },
      {
        p: "«Not enough for one portion» is the same statement as «out», only measured against the dish: 200 g of meat against a 500 g card is a dish that cannot be cooked.",
      },
      { see: ["stop-list", "stock-balances"] },
    ],
  },

  // ═══════════════════════════════════ Staff
  {
    slug: "staff-add",
    section: "team",
    title: "Adding an employee",
    lead: "Cashier, waiter, cook — each with their own account and their own app.",
    keys: ["staff", "employee", "cashier", "waiter", "cook", "login"],
    body: [
      { p: "`Money and team` → `Staff` → `Add`." },
      { fig: "staff", notes: { add: "A new employee is added here." } },
      { h: "What you fill in" },
      {
        table: {
          head: ["Field", "Notes"],
          rows: [
            ["*Name*", "Appears in the calendar and the reports."],
            ["*Position*", "Free text («cook», «waiter»). *It is not a permission* — the system never reads it."],
            ["*Branch*", "Which building they work in. Attendance and store work are recorded there."],
            ["*Login and PIN*", "For the till and the staff app."],
            ["*Pay*", "A salary, or a daily/hourly rate. A salary is divided across the month's rostered days."],
            ["*Permissions*", "Kitchen screen, stock counting — each granted separately."],
          ],
        },
      },
      {
        warn: "*Position is not a permission.* Typing «cook» does not open the kitchen screen; that has to be ticked. Treating the position as a permission would hand out keys to whoever's spelling happened to match.",
      },
      {
        p: "The employee signs in at `yoursite.uz/staff`: clock in and out, their own calendar and — where granted — the kitchen screen or stock counting.",
      },
      {
        tip: "Do not *delete* somebody who has left; set them inactive. Their shifts and pay history stay, and access closes immediately.",
      },
      { see: ["staff-roles", "staff-attendance", "payroll"] },
    ],
  },
  {
    slug: "staff-roles",
    section: "team",
    title: "Roles and permissions",
    lead: "Who opens which screen. Every permission is granted on its own and does not follow from a job title.",
    keys: ["role", "permission", "access", "who sees"],
    body: [
      { p: "There are four kinds of account, and they sign into entirely different apps." },
      {
        table: {
          head: ["Account", "Signs into"],
          rows: [
            ["*Owner / manager*", "The panel (`/admin`)"],
            ["*Employee*", "Till, dining room, kitchen screen, stock counting (`/staff`)"],
            ["*Courier*", "The courier app (`/kuryer`)"],
            ["*Customer*", "The site and the Telegram mini app"],
          ],
        },
      },
      { h: "Employee permissions" },
      {
        p: "`Money and team` → `Roles`: a role is created with a set of permissions and then given to an employee. The permissions are independent:",
      },
      {
        list: [
          "*Till* — selling, closing a bill.",
          "*Dining room* — tables, splitting and merging bills.",
          "*Kitchen screen (KDS)* — seeing tickets and pressing «Ready».",
          "*Stock counting* — filling in a stocktake.",
          "*Discounts* — applying a discount at the till.",
          "*Voiding a bill* — voids and refunds.",
        ],
      },
      {
        warn: "The kitchen screen and stock counting are *off by default* — a deliberate departure from this system's usual «empty value means what it did before». On a shared tablet anybody could press «Ready», take the ticket off the kitchen screen, and tell the panel the kitchen cooked it.",
      },
      {
        tip: "Give discounts and voids only to the shift lead. They are the two most expensive buttons on the till, and both are logged.",
      },
      { fig: "roles" },
      { see: ["staff-add", "kds", "admins", "admin-log"] },
    ],
  },
  {
    slug: "staff-attendance",
    section: "team",
    title: "Attendance: clocking in and out",
    lead: "The employee presses «I'm here» on their phone — and location is checked.",
    keys: ["attendance", "shift", "clock in", "timesheet", "geolocation"],
    body: [
      {
        p: "On `/staff` the employee presses `Clock in`. The phone's location is checked: they have to be within 50 metres of the restaurant.",
      },
      {
        p: "The radius is set in `Settings` → `Branch`. *50 metres, deliberately, not 5*: phone GPS is 10–30 metres in the open and worse indoors. A radius smaller than the error catches no cheat and leaves an honest employee standing outside.",
      },
      { h: "The calendar" },
      {
        p: "`Money and team` → `Staff` → the employee. Every day is coloured: on schedule, under, over, or absent. That state is stored nowhere — it is a *comparison* of the roster and the shifts.",
      },
      {
        list: [
          "A shift is recorded on the day it *started*: a cook leaving at 00:40 closes Tuesday.",
          "An empty rostered day reads as a day off and blames nobody.",
          "A second clock-in is refused — two open shifts would double every hour in the calendar.",
        ],
      },
      {
        tip: "Manual correction is *deliberately kept*: phones die, clock-outs get forgotten. An attendance system you cannot correct is abandoned in the second week. Every correction is signed by whoever made it.",
      },
      { see: ["payroll", "staff-qr"] },
    ],
  },
  {
    slug: "staff-qr",
    section: "team",
    title: "Clocking in by QR",
    lead: "A screen at the entrance shows a QR and the employee scans it — without relying on phone GPS.",
    keys: ["qr", "clock in", "kiosk", "attendance"],
    body: [
      {
        p: "A tablet or monitor in the branch runs in `/kiosk` mode showing a QR. The employee scans it and the shift opens.",
      },
      {
        warn: "The code lives *on a screen, not on paper*, and changes every 30 seconds. A printed QR is a password written on a wall: photograph it once and you can clock in from home for a year.",
      },
      { see: ["staff-attendance", "kiosk"] },
    ],
  },
  {
    slug: "payroll",
    section: "team",
    title: "Pay and payments",
    lead: "Who worked how much, what they are owed, what has been handed over.",
    keys: ["payroll", "salary", "wages", "advance"],
    body: [
      {
        p: "`Money and team` → `Payroll`. Per person, per period: days worked, amount earned, amount paid and the balance.",
      },
      {
        p: "A salary is divided across the month's *rostered days* and earned per day attended — so somebody who started mid-month or missed a week still comes out right.",
      },
      {
        p: "A payment is a *record*, not a counter: it carries the period it settles and who handed it over. Once April has begun, «how much did we pay in March» still has an answer.",
      },
      {
        tip: "Record advances here too. Otherwise the month-end figure comes from two sources and they will not agree.",
      },
      { fig: "payroll" },
      { see: ["staff-attendance", "finance"] },
    ],
  },
  {
    slug: "admin-log",
    section: "team",
    title: "The action log",
    lead: "Who changed what — a price, an order, a tech card, a count.",
    keys: ["log", "audit", "history", "who changed"],
    body: [
      {
        p: "`Money and team` → `Log`. Each entry: who, when, what they did and *what changed*.",
      },
      {
        p: "A tech card edit is written out in detail: which ingredient, from what to what. «Recipe changed» is true, is in the log, and answers nothing.",
      },
      {
        warn: "Raising a norm on a card is the one way to take money out *invisibly to the arithmetic*: the card says 200 g of beef, the kitchen puts in 150 g, 50 g is left over on every portion, and a count finds nothing — the card already expects it to be gone. The only visible moment is the moment the card is edited.",
      },
      {
        tip: "A norm rising by more than 20% raises a warning in the panel. The threshold is high on purpose: correcting a card is ordinary work, and an 11% correction is not worth a message.",
      },
      { fig: "logs" },
      { see: ["tech-cards", "admins", "security"] },
    ],
  },

  // ═══════════════════════════════════ Customers
  {
    slug: "crm",
    section: "customers",
    title: "The customer card",
    lead: "One phone number, the whole history: orders, addresses, points, debt, reviews.",
    keys: ["customer", "crm", "database", "history", "phone"],
    body: [
      { p: "`Customers` → `Database`. Search by phone number or name." },
      {
        list: [
          "Order history and average bill.",
          "Saved addresses.",
          "Points balance and movement.",
          "Any debt — the amount and a «settled» button.",
          "Reviews and complaints.",
        ],
      },
      {
        p: "A manager sees *their branch's* order history; the customer profile itself belongs to the company, because the database is shared.",
      },
      { fig: "users" },
      { see: ["loyalty", "segments", "till-payment"] },
    ],
  },
  {
    slug: "loyalty",
    section: "customers",
    title: "Points (cashback)",
    lead: "A percentage comes back on every order and is spent on the next one.",
    keys: ["points", "cashback", "loyalty", "bonus"],
    body: [
      {
        p: "`Settings` → `Loyalty`. Switch it on, set a percentage (5%, say) and cap how much of an order can be paid with points.",
      },
      {
        list: [
          "Points are awarded after *delivery*, not when the order is placed.",
          "A cancelled order awards nothing.",
          "The part paid with points appears in reports «for information», not as an expense.",
        ],
      },
      {
        warn: "Points are *not an expense*: the money did not go out, it never came in. The revenue line is already clear of them, so subtracting them again would count the campaign twice.",
      },
      { see: ["promotions", "crm", "finance"] },
    ],
  },
  {
    slug: "segments",
    section: "customers",
    title: "Segments and campaigns",
    lead: "Writing to the people who have not been in for a month.",
    keys: ["segment", "campaign", "sms blast", "rfm"],
    body: [
      {
        p: "`Customers` → `Campaigns`. Pick a segment, then a channel and a message.",
      },
      {
        table: {
          head: ["Segment", "Who they are"],
          rows: [
            ["*New*", "Ordered once"],
            ["*Regulars*", "Come back consistently"],
            ["*Lapsed*", "Used to come, no longer do"],
            ["*Most valuable*", "Have spent the most"],
          ],
        },
      },
      {
        p: "Segments are computed *relative to your own base*: a «regular» is a regular at your restaurant, not against some general norm.",
      },
      {
        list: [
          "*SMS* — costs money; the number of parts is shown on screen.",
          "*Telegram* — through the bot, free, but only reaches people who have written to it.",
          "*Push* — a browser notification, free.",
        ],
      },
      {
        warn: "Cyrillic in an SMS drops one message from 160 characters to *70*, which doubles or triples the price. Write in Latin script.",
      },
      {
        tip: "Only the *owner* can send a campaign. One button can disturb the entire customer base and spend money.",
      },
      { fig: "campaigns" },
      { see: ["sms", "push", "telegram-bot"] },
    ],
  },
  {
    slug: "push",
    section: "customers",
    title: "Browser notifications (push)",
    lead: "A free channel: the customer gets a message without visiting the site.",
    keys: ["push", "notification", "browser"],
    body: [
      {
        p: "If a customer has allowed notifications on the site, order statuses and campaigns reach them. Unlike SMS, this is *free*.",
      },
      {
        p: "Nothing needs configuring: the keys are generated on first use and *never rotated* — every subscription is bound to the key it was created with.",
      },
      {
        warn: "On iPhone, push only works if the site has been «added to the home screen». That is an Apple restriction, not a setting.",
      },
      { see: ["segments"] },
    ],
  },
  {
    slug: "feedback",
    section: "customers",
    title: "Reviews and complaints",
    lead: "The rating asked for after an order, and what to do with it.",
    keys: ["review", "complaint", "rating"],
    body: [
      {
        p: "`Customers` → `Reviews`. Each one carries the order number — so you can see which evening, which dishes and who delivered.",
      },
      {
        tip: "On a low rating, open the customer card: whether this is a first complaint or a third changes the answer.",
      },
      { fig: "feedback" },
      { see: ["reviews", "crm"] },
    ],
  },
  {
    slug: "calls",
    section: "customers",
    title: "The call centre",
    lead: "Taking orders by phone, and the call log.",
    keys: ["call", "phone", "operator", "call centre"],
    body: [
      {
        p: "`Customers` → `Call centre`. The operator enters a number, the system finds the customer and shows their whole history — and the order is entered right there.",
      },
      {
        p: "With a PBX connected (onlinePBX), an incoming call opens the customer card *by itself*, and the number, duration, recording and who answered land in the log automatically. The operator only writes the outcome and a note.",
      },
      {
        tip: "If the operator is still writing up another call, the screen is not taken over: losing a half-written note is worse than not opening a card.",
      },
      { fig: "calls" },
      { see: ["pbx", "crm"] },
    ],
  },

  // ═══════════════════════════════════ Integrations
  {
    slug: "integrations-overview",
    section: "integrations",
    title: "Integrations: what connects",
    lead: "What is required, what is optional, and what the site cannot run without.",
    keys: ["integration", "connect", "api"],
    body: [
      {
        table: {
          head: ["What", "Required", "Why"],
          rows: [
            ["*SMS*", "*Yes*", "Customers sign in with a code. Without it nobody can register."],
            ["*Map*", "*Yes*", "To choose a delivery address."],
            ["*Online payment*", "No", "Cash always works. A payment provider is an extra channel."],
            ["*Telegram bot*", "No", "A second channel: the mini app and notifications."],
            ["*Till system (POS)*", "No", "If you already run iiko/Poster."],
            ["*PBX*", "No", "If many orders come in by phone."],
            ["*Fiscal receipts*", "If the law requires it", "Filing the receipt with the state system."],
            ["*Marking*", "If you sell drinks", "Adding the DataMatrix code to the receipt."],
          ],
        },
      },
      {
        warn: "Every key is stored in `Settings` and *never returned*: the screen only shows a «key saved» flag. Leaving a field blank and saving keeps the stored key — so that correcting one field does not quietly switch payments off.",
      },
      { fig: "settings" },
      { see: ["sms", "payments", "pos", "telegram-bot"] },
    ],
  },
  {
    slug: "sms",
    section: "integrations",
    title: "Connecting an SMS provider",
    lead: "Customer sign-in, booking confirmations and campaigns all go through it.",
    keys: ["sms", "eskiz", "play mobile", "getsms", "code not arriving"],
    body: [
      {
        p: "`Settings` → `SMS`. Four services: *Eskiz*, *Play Mobile*, *getsms.uz*, *OneSignal*. You sign the contract and you take the sender name through moderation.",
      },
      {
        steps: [
          "Choose the provider and enter its keys.",
          "Enter the sender name (the one that passed moderation).",
          "Check the text of the code message — it *must* contain `{code}`.",
          "Send a `Test SMS` and confirm it arrives on your phone.",
        ],
      },
      {
        warn: "`Test SMS` is the main button on this page. Even when the keys look right, whether the sender name passed moderation and whether there is money on the account are both discovered *when the first customer tries to sign in* — and a restaurant reads that as «the site is broken».",
      },
      {
        warn: "Without `{code}` the text will not save. Otherwise the customer receives *a message with no code*: the gateway accepts it, the SMS arrives, nothing errors anywhere, and the person cannot sign in.",
      },
      {
        p: "Changing the text invalidates the moderation, so the «verified» flag is reset: a green tick sitting over a message the gateway has never seen is lying, and you find out when a customer cannot sign in.",
      },
      {
        tip: "Write in Latin script: 160 characters is one SMS. With Cyrillic the limit drops to *70*, and one message can split into three.",
      },
      { see: ["segments", "login"] },
    ],
  },
  {
    slug: "payments",
    section: "integrations",
    title: "Online payments: Payme, Click, Uzum, ATMOS",
    lead: "The customer pays by card on the site. Keys come from the provider's dashboard.",
    keys: ["payment", "payme", "click", "uzum", "atmos", "card", "online"],
    body: [
      {
        p: "`Settings` → `Payments`. Each provider is enabled separately — several can run at once and the customer chooses at checkout.",
      },
      { h: "How to connect" },
      {
        steps: [
          "Sign a contract with the provider and get a dashboard.",
          "Take the keys (merchant id, secret) from the dashboard into the panel.",
          "The panel gives you a *callback address* — enter it in the provider's dashboard.",
          "Enable it and make a small test payment.",
        ],
      },
      {
        warn: "The callback address is the most-forgotten step. Without it the payment goes through but nobody tells us: the customer's money is gone and the order sits «unpaid».",
      },
      {
        warn: "Cancelling an order *does not refund it*. Refunds are made from the provider's dashboard.",
      },
      {
        p: "A provider that is not configured is *not shown to the customer at all*: a button that leads to a bank error page makes the guest blame the restaurant.",
      },
      { see: ["till-payment", "order-cancel"] },
    ],
  },
  {
    slug: "telegram-bot",
    section: "integrations",
    title: "Telegram bot and mini app",
    lead: "Your own bot: menu, orders and notifications inside Telegram.",
    keys: ["telegram", "bot", "mini app", "token", "botfather"],
    body: [
      {
        steps: [
          "In Telegram message `@BotFather` → `/newbot` → give a name and a username.",
          "Copy the *token* BotFather gives you.",
          "Paste it into `Settings` → `Telegram` in the panel.",
          "Press `Check connection` — the bot's name fills itself in, and the mini app link is built from it.",
        ],
      },
      {
        warn: "`Check connection` is the main button on the page: a dead bot's token differs from a working one only when the first guest tries to open it.",
      },
      {
        warn: "The bot token is a secret. A leaked token is not «money in somebody else's account»: it is the ability to write to *every guest* who ever opened the mini app, as the restaurant.",
      },
      { h: "The mini app" },
      {
        p: "The menu, opened inside Telegram. The guest is signed in automatically from their Telegram account — no SMS code.",
      },
      {
        p: "Telegram does not hand over a phone number, only a name. So the mini app asks for the number *before* checkout: asking at the confirm step loses an order that had already been won.",
      },
      {
        tip: "The bot sends order statuses, and unlike SMS that is free. If half your customers move to the bot, the SMS bill drops noticeably.",
      },
      { see: ["sms", "segments"] },
    ],
  },
  {
    slug: "pos",
    section: "integrations",
    title: "Till systems: iiko, Syrve, Poster, Clopos, r_keeper",
    lead: "If you already run a till, the order goes straight into it.",
    keys: ["iiko", "poster", "clopos", "r_keeper", "syrve", "pos"],
    body: [
      {
        p: "`Settings` → `POS`. *The menu stays ours* — nothing is pulled from the POS: we hold the photos, translations, combos and site copy, and running a menu in two places is confusion.",
      },
      { h: "Connecting" },
      {
        steps: [
          "Choose the provider and enter its keys.",
          "`Check connection` — the answer names the organisation and the terminal, not just «connected».",
          "In `Mapping`, match each dish to a product in the till.",
        ],
      },
      {
        warn: "*An unmapped dish stops the whole order.* Deliberately: a kitchen handed an incomplete ticket cooks exactly what it saw. The error names the dish.",
      },
      {
        p: "The setting belongs *to the branch*: in a chain each kitchen has its own terminal group, and an order printed at the wrong till is worse than one not printed at all. A mapping can be copied from another branch — same brand and same provider.",
      },
      { h: "Provider specifics" },
      {
        table: {
          head: ["System", "What to know"],
          rows: [
            ["*iiko / Syrve*", "The order is created asynchronously: even a 200 does not mean the till has accepted it. The system waits up to 12 seconds and checks."],
            ["*Poster*", "The order arrives as an «incoming order» and somebody at the till has to press `Accept`. There is no cancel API — cancel it on the till."],
            ["*Clopos*", "Auto-accept settings must be on, otherwise the order sits unaccepted."],
            ["*r_keeper*", "The server sits *inside the restaurant*. Our server cannot reach it without a forwarded port or a VPN."],
          ],
        },
      },
      {
        tip: "If the till does not answer, the order is not broken: the reason is written on it and a `Send again` button stays. An order we have and the till does not can be fixed; one lost silently cannot.",
      },
      { fig: "pos" },
      { see: ["stop-list", "menu-import", "order-flow"] },
    ],
  },
  {
    slug: "pbx",
    section: "integrations",
    title: "PBX: onlinePBX",
    lead: "An incoming call opens the customer card by itself.",
    keys: ["pbx", "onlinepbx", "telephony", "call"],
    body: [
      {
        steps: [
          "`Settings` → `Telephony`: enter the onlinePBX keys.",
          "The panel gives you a *webhook address*.",
          "Enter it in the events settings of the onlinePBX dashboard.",
          "Make a test call and check the `Last event` line.",
        ],
      },
      {
        warn: "`Last event` is the most useful line on this page. Even with correct keys the address may never have been entered in the onlinePBX dashboard, and a connection check *cannot show that at all*.",
      },
      {
        p: "The webhook address is itself the key: onlinePBX sends no password. If it leaks, it can be rotated.",
      },
      {
        tip: "Call duration is measured by the *conversation*. Forty seconds of ringing with no answer is a zero-second call.",
      },
      { see: ["calls"] },
    ],
  },
  {
    slug: "fiscal",
    section: "integrations",
    title: "Fiscal receipts",
    lead: "Filing receipts with the state system: which providers exist and what is needed.",
    keys: ["fiscal", "receipt", "tax", "ofd", "ikpu"],
    body: [
      {
        p: "`Settings` → `Fiscal`. Choose a provider and enter its keys. The receipt is filed automatically when the bill closes on the till.",
      },
      {
        warn: "*A provider whose adapter is not written yet cannot be enabled.* Keys can be saved (owners often set this up before the contract is signed), but enabling is refused with a reason: a restaurant that believes it is filing receipts is not a missing feature, it is a *legal problem*.",
      },
      { h: "IKPU and unit codes" },
      {
        p: "Every dish carries an IKPU code and a unit-of-measure code (`Menu` → dish → `Fiscal`). Without them the receipt may be rejected.",
      },
      {
        p: "IKPU is a *fact about the product*: it is read from the menu, so an accountant's correction reaches orders that have not been paid for yet.",
      },
      {
        tip: "The VAT rate is set per branch, and per dish where needed. A dish without one uses the branch's rate.",
      },
      { see: ["marking", "till-shift"] },
    ],
  },
  {
    slug: "marking",
    section: "integrations",
    title: "Marking (Asl Belgisi)",
    lead: "If you sell drinks: the DataMatrix code of each bottle goes onto the receipt.",
    keys: ["marking", "asl belgisi", "datamatrix", "scanner", "drinks"],
    body: [
      {
        p: "There is *no separate «Asl Belgisi» integration and none is needed*: the code travels inside the fiscal receipt and the OFD passes it to the national system.",
      },
      {
        steps: [
          "Flag the products that require marking in the menu (`Menu` → dish → `Marking`).",
          "Connect a DataMatrix scanner to the till — it appears as an ordinary keyboard, no driver needed.",
          "When the line is added to a sale the code is asked for: scan the bottle.",
        ],
      },
      {
        warn: "*Marked lines do not merge*: two bottles are two codes. Tapping the tile twice gives two separate lines, not one line of two.",
      },
      {
        warn: "Duplicate codes are checked across the whole receipt. The scanner beeped, the read was unclear, it was scanned again — and two lines produce one bottle. *The fiscal system accepts such a receipt*, which means the only place to catch it is here.",
      },
      {
        tip: "A whole bottle and a glass are *two different dishes* on the menu. Then the «opened bottle» question answers itself: a glass is never asked for a code.",
      },
      { see: ["fiscal", "menu-item"] },
    ],
  },
  {
    slug: "pos-migration",
    section: "integrations",
    title: "Migrating from another till",
    lead: "Bringing data across from iiko, r_keeper, Clopos, Poster or Jowi.",
    keys: ["migration", "import", "from iiko"],
    body: [
      {
        p: "`Menu` → `Import from POS`. Export Excel/CSV from the old system and upload it here.",
      },
      {
        steps: [
          "*Ingredients* — the catalogue: name, unit, price.",
          "*Tech cards* — ingredients are matched by name, so the first step has to be done before this one.",
          "*Stock balances* — written in as an opening count.",
        ],
      },
      {
        warn: "Names that cannot be matched are listed and *not written*. They have to be matched by hand — guessing would produce a tech card pointing at the wrong ingredient.",
      },
      { see: ["menu-import", "ingredients", "tech-cards"] },
    ],
  },

  // ═══════════════════════════════════ Reports
  {
    slug: "dashboard",
    section: "reports",
    title: "The home screen",
    lead: "How today is going: orders, revenue, average bill, what is waiting.",
    keys: ["dashboard", "home", "statistics", "today"],
    body: [
      {
        p: "The first screen after signing in. The period picker at the top governs every figure on it.",
      },
      {
        list: [
          "Order count and revenue, against the previous period.",
          "Average bill.",
          "Waiting orders — the unaccepted ones shown separately.",
          "*Debt* on its own tile: everything else arrives by itself within the hour, this arrives when somebody phones.",
          "Best-selling dishes.",
        ],
      },
      {
        tip: "The tiles can be tailored: remove the ones you do not need. The number an owner needs is different for every owner.",
      },
      { fig: "dashboard" },
      { see: ["reports-sales", "ai-briefing"] },
    ],
  },
  {
    slug: "reports-sales",
    section: "reports",
    title: "Sales reports",
    lead: "What sold, through which channel, and who sold it.",
    keys: ["report", "sales", "channel", "dishes"],
    body: [
      { p: "`Money and team` → `Reports`. Pick a period and a report." },
      {
        table: {
          head: ["Report", "The question"],
          rows: [
            ["*Dishes*", "What sold, how many, for how much"],
            ["*Channels*", "Site, Telegram, till, QR — which brings more"],
            ["*Team*", "Who sold how much"],
            ["*ABC/XYZ*", "Which dish makes money, which sells steadily"],
            ["*Store*", "What came in and what went out"],
          ],
        },
      },
      {
        warn: "Every report states *its own coverage*: «cost is entered on 1 of 19 dishes — gross margin covers 30% of revenue». Otherwise a restaurant that has costed ten dishes out of two hundred would make decisions from a column describing 6% of the evening.",
      },
      { fig: "reports" },
      { see: ["abc", "finance", "excel"] },
    ],
  },
  {
    slug: "abc",
    section: "reports",
    title: "ABC/XYZ analysis",
    lead: "Which dish makes the money and which one sells predictably.",
    keys: ["abc", "xyz", "analysis", "pareto", "menu engineering"],
    body: [
      {
        p: "*ABC* is by money: group `A` produces most of the revenue, `C` almost none. *XYZ* is by steadiness: `X` sells the same every day, `Z` is unpredictable.",
      },
      {
        list: [
          "`AX` — the load-bearing dishes. They should never reach the stop list.",
          "`CZ` — candidates for removal: they sell little and unpredictably.",
          "`AZ` — bring in a lot, but unevenly. Be careful with stock.",
        ],
      },
      {
        p: "Ingredients have their own ABC, and it answers a *different question*: the dish producing the most revenue is usually not made from the most expensive raw material.",
      },
      {
        warn: "The ingredient ABC is ranked *by money spent*, not by consumption from cards: the first is measured (an invoice with a date and a total), the second is an estimate over a half-uncosted menu.",
      },
      { see: ["reports-sales", "tech-cards"] },
    ],
  },
  {
    slug: "finance",
    section: "reports",
    title: "The financial report and the cash drawer",
    lead: "Where the money came from and where it went — and why this is not a profit report.",
    keys: ["finance", "cash", "money", "profit", "expenses"],
    body: [
      {
        p: "`Money and team` → `Cash`. Shifts, money in and out, and the daily balance.",
      },
      {
        warn: "*The financial report is NOT a profit report*, and it says so itself. With an uncosted menu, «in minus out» is cash movement, not profit. Read as profit, it overstates it by the entire cost of the food.",
      },
      {
        list: [
          "*Discounts and points are not expenses*: the money did not leave, it never arrived.",
          "*A courier handing in cash is not an expense*: it is collected cash arriving in the till.",
          "*A transfer is not an expense*: value is carried, not created.",
        ],
      },
      {
        p: "Cash from deliveries reaches the till after the courier hands it in. Money held by couriers is shown separately — anybody investigating a shortfall asks for that figure first.",
      },
      { fig: "cash" },
      { see: ["till-shift", "couriers", "reports-sales"] },
    ],
  },
  {
    slug: "excel",
    section: "reports",
    title: "Exporting to Excel",
    lead: "Any report becomes an `.xlsx` file in one click.",
    keys: ["excel", "export", "xlsx"],
    body: [
      {
        p: "Every report page has an `Excel` button. The file takes *exactly* the columns and period on screen.",
      },
      {
        tip: "If your accountant needs a report, do not hand over a panel login — send the Excel file. A panel login is the customer database and the payment keys.",
      },
      { see: ["reports-sales", "data-export"] },
    ],
  },
  {
    slug: "ai-briefing",
    section: "reports",
    title: "The morning briefing",
    lead: "A few sentences about yesterday: what changed and what to look at.",
    keys: ["ai", "briefing", "assistant", "summary"],
    body: [
      {
        p: "Every morning the panel shows a short summary of yesterday: revenue against last week, what sold well, what looks odd.",
      },
      {
        p: "It does not replace the numbers — it tells you which number is worth looking at.",
      },
      {
        tip: "There is a daily limit. When it runs out it works again the next day, or more can be added.",
      },
      { see: ["dashboard"] },
    ],
  },

  // ═══════════════════════════════════ Settings
  {
    slug: "theme-lang",
    section: "settings",
    title: "Language and theme",
    lead: "The panel in three languages, light and dark.",
    keys: ["language", "theme", "dark mode"],
    body: [
      {
        p: "The language picker and the theme button are in the top-left of the panel. The choice is *yours alone* — other administrators are unaffected.",
      },
      {
        p: "The site's language comes from the guest's browser, and they can switch it themselves. The site languages have their own addresses (`/ru/`, `/en/`).",
      },
      { see: ["site-texts"] },
    ],
  },
  {
    slug: "notifications",
    section: "settings",
    title: "Sound and notifications",
    lead: "What you are told about, and where.",
    keys: ["sound", "notification", "alert", "telegram"],
    body: [
      {
        list: [
          "*New-order sound* — in the panel, repeating until «Accept».",
          "*Telegram message* — orders, bookings and complaints to a group or a private chat.",
          "*Warnings* — an unprinted ticket, a large discount, a tech card norm going up, a cash shortfall.",
        ],
      },
      {
        p: "To receive Telegram messages you have to write to the bot once — it cannot write first.",
      },
      {
        tip: "Every warning should have exactly one button in the panel that turns it off. If a signal is bothering you, switch it off: a warning that cannot be silenced becomes a warning nobody reads.",
      },
      { see: ["order-accept", "telegram-bot", "admin-log"] },
    ],
  },
  {
    slug: "security",
    section: "settings",
    title: "Security",
    lead: "Who can reach what, and what is never shown.",
    keys: ["security", "password", "key", "protection"],
    body: [
      {
        list: [
          "Passwords are never stored in the clear.",
          "Payment, SMS, PBX and bot keys are *never returned*: the screen only shows a «saved» flag.",
          "A manager cannot leave their branch — including through the address bar.",
          "Sign-in attempts are limited: several wrong passwords in a row block the account temporarily.",
          "Every significant action is written to the log.",
        ],
      },
      {
        warn: "The biggest risk is a *shared account*. If three people work under one «owner» login, the log, the branch boundaries and the permissions all stop meaning anything.",
      },
      {
        tip: "The map key is the exception: it is handed to the browser and cannot be a secret. Its protection is the domain restriction in the provider's dashboard.",
      },
      { see: ["admins", "admin-log", "map-provider"] },
    ],
  },
  {
    slug: "data-export",
    section: "settings",
    title: "Taking your data with you",
    lead: "The menu, the customers and the orders are yours.",
    keys: ["export", "data", "backup"],
    body: [
      {
        p: "A restaurant that cannot take its menu, its orders and its database elsewhere is held by the *cost of leaving*, not by the product. That is why export exists.",
      },
      {
        steps: [
          "`Settings` → `Data`: leave a request.",
          "We grant permission (it is one-time).",
          "You download the files: menu, customers, orders — as `.xlsx` and `.csv`.",
        ],
      },
      {
        tip: "A backup is taken automatically every day and kept by us. Export is a different thing: it is taking the data *with you*.",
      },
      { see: ["excel", "billing"] },
    ],
  },
  {
    slug: "billing",
    section: "settings",
    title: "Billing and subscription",
    lead: "What you pay, when, and what happens if a payment is late.",
    keys: ["billing", "subscription", "price", "invoice"],
    body: [
      {
        p: "The subscription is monthly. `Settings` → `Subscription` shows the current period, the next payment date and the invoice history.",
      },
      {
        p: "The price is made of the till (per month) and orders (per order). The calculator is on the keel.uz home page.",
      },
      {
        tip: "A late payment does not switch the site off straight away — a warning appears in the panel first. But a long delay does stop the site, so keep an eye on the invoice.",
      },
      { see: ["data-export"] },
    ],
  },
  {
    slug: "support",
    section: "settings",
    title: "Asking for help",
    lead: "How to write so the answer comes in the first reply.",
    keys: ["help", "support", "contact"],
    body: [
      {
        p: "There is a chat button at the bottom right of the panel — it reaches us directly. We also answer on Telegram.",
      },
      { h: "What to write" },
      {
        list: [
          "*Which screen* — «Store → Tech cards».",
          "*What you did* — «saved a prep».",
          "*What you expected and what happened* — «the cost should have appeared, it shows zero».",
          "*A screenshot* — one picture replaces ten messages.",
        ],
      },
      {
        tip: "«It does not work» pushes the answer two or three messages away. With those four lines the fix usually comes in the first reply.",
      },
    ],
  },
];
