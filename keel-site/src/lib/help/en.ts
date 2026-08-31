import type { HelpContent } from "./types";
import { articlesEn } from "./en.articles";

export const helpEn: HelpContent = {
  ui: {
    title: "Knowledge base",
    lead: "The full guide to Keel: setting it up, running it day to day, and what to do when something breaks.",
    searchPlaceholder: "Search: tech card, printer, zone…",
    searchEmpty:
      "Nothing found for «{q}». Try another word, or write to us.",
    searchCount: "{n} articles",
    allArticles: "All articles",
    inSection: "Section",
    back: "Back",
    next: "Next",
    prev: "Previous",
    seeAlso: "See also",
    figureHint: "Click the image to enlarge it.",
    notFound: "Article not found",
    notFoundLead: "There is no such address. Go back to the start of the knowledge base.",
    askUs: "Didn't find the answer?",
    askUsLead:
      "Write on Telegram — a person answers. Tell us which screen you are on and what you expected, and the answer comes in the first reply.",
    updated: "Updated",
  },

  sections: [
    {
      id: "start",
      title: "Getting started",
      lead: "The first day: signing in, branches, opening hours and who sees what.",
    },
    {
      id: "site",
      title: "The site",
      lead: "What the guest sees: look, texts, photos and being found.",
    },
    {
      id: "menu",
      title: "Menu",
      lead: "Categories, dishes, options, combos, offers and the stop list.",
    },
    {
      id: "orders",
      title: "Orders",
      lead: "From arrival to handover: statuses, sound, cancelling, bookings.",
    },
    {
      id: "delivery",
      title: "Delivery",
      lead: "Maps, zones, pricing and couriers.",
    },
    {
      id: "till",
      title: "Till and dining room",
      lead: "The till program, shifts, tables, the kitchen screen and the kiosk.",
    },
    {
      id: "printers",
      title: "Printers",
      lead: "Connecting receipt and kitchen printers, and what to do when a ticket does not print.",
    },
    {
      id: "stock",
      title: "Store and costing",
      lead: "Ingredients, tech cards, deliveries, write-offs, counts and balances.",
    },
    {
      id: "team",
      title: "Staff",
      lead: "Adding people, roles and permissions, attendance and pay.",
    },
    {
      id: "customers",
      title: "Customers",
      lead: "The database, points, segments, campaigns and reviews.",
    },
    {
      id: "integrations",
      title: "Integrations",
      lead: "Payments, SMS, Telegram, till systems, PBX, fiscal receipts and marking.",
    },
    {
      id: "reports",
      title: "Reports",
      lead: "What sold, how much profit, where the shortfall is.",
    },
    {
      id: "settings",
      title: "Settings",
      lead: "Language, theme, security, data export and subscription.",
    },
  ],

  articles: articlesEn,
};
