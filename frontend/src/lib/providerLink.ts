// Hands an order to an outside delivery service over the web: fills their order
// form's link with everything they ask for, and lays the same details out as
// copy-ready fields for the services whose form cannot be pre-filled.
//
// Restaurants without their own couriers hand orders to Yandex Delivery, a taxi
// company, a door-to-door firm and so on. A real API integration needs a
// business account and a contract with each service — most small restaurants
// have neither. What works for every one of them is the web: open their form
// with the two addresses and the phone already in the URL, paste the rest field
// by field, or read it out to a dispatcher. That is what this builds.

import type { Order, Restaurant } from "./types";

// Every {placeholder} an admin may use in a provider URL. `pickup*` is the
// restaurant (where the courier collects), the rest is the customer's side.
export const PLACEHOLDERS = [
  "{number}",
  "{name}",
  "{phone}",
  "{address}",
  "{comment}",
  "{lat}",
  "{lng}",
  "{total}",
  "{subtotal}",
  "{deliveryFee}",
  "{items}",
  "{itemsCount}",
  "{payment}",
  "{pickupName}",
  "{pickupAddress}",
  "{pickupPhone}",
  "{pickupLat}",
  "{pickupLng}",
] as const;

// A note on Yandex Go app links: they carry the two coordinate pairs and the
// tariff class, and nothing else — the documented parameters are start-lat/lon,
// end-lat/end-lon, tariffClass, ref, lang and the AppMetrica ids. The
// recipient's name, phone, street address and comment cannot travel in the URL,
// which is why the call window lays them out as copyable fields. Filing all of
// that automatically needs the B2B API (provider kind "api").

/** The dishes as one line: "Lavash × 2, Cola × 1". */
export function itemsLine(order: Order): string {
  return order.items.map((it) => `${it.name} × ${it.qty}`).join(", ");
}

function digits(phone: string): string {
  const d = phone.replace(/\D/g, "");
  return d ? `+${d}` : "";
}

function fieldsOf(
  order: Order,
  restaurant?: Restaurant | null,
): Record<string, string> {
  const addr = restaurant?.address;
  const hasPickupGeo = !!addr?.lat && !!addr?.lng;
  return {
    number: order.number,
    name: order.customer.name,
    phone: digits(order.customer.phone),
    address: order.address?.text ?? "",
    comment: order.address?.comment ?? "",
    lat: order.address?.lat ? String(order.address.lat) : "",
    lng: order.address?.lng ? String(order.address.lng) : "",
    total: String(order.total),
    subtotal: String(order.subtotal),
    deliveryFee: String(order.deliveryFee),
    items: itemsLine(order),
    itemsCount: String(order.items.reduce((n, it) => n + it.qty, 0)),
    payment: order.paymentMethod,
    pickupName: restaurant?.name ?? "",
    pickupAddress: addr?.text ?? "",
    pickupPhone: digits(restaurant?.phones?.[0] ?? ""),
    pickupLat: hasPickupGeo ? String(addr!.lat) : "",
    pickupLng: hasPickupGeo ? String(addr!.lng) : "",
  };
}

/** Substitutes {placeholders} in a provider URL, URL-encoding every value. */
export function providerUrl(
  template: string,
  order: Order,
  restaurant?: Restaurant | null,
): string {
  const f = fieldsOf(order, restaurant);
  return template.replace(/\{(\w+)\}/g, (whole, key: string) =>
    key in f ? encodeURIComponent(f[key]) : whole,
  );
}

/**
 * Placeholders the template uses but the order/restaurant cannot fill — the
 * panel warns about these instead of opening a link with empty coordinates.
 */
export function missingPlaceholders(
  template: string,
  order: Order,
  restaurant?: Restaurant | null,
): string[] {
  const f = fieldsOf(order, restaurant);
  const missing = new Set<string>();
  for (const m of template.matchAll(/\{(\w+)\}/g)) {
    const key = m[1];
    if (key in f && !f[key]) missing.add(`{${key}}`);
  }
  return [...missing];
}

export interface CopyRow {
  key: string;
  label: string;
  value: string;
}

export interface CopyLabels {
  pickupAddress: string;
  pickupPhone: string;
  address: string;
  comment: string;
  coords: string;
  customer: string;
  phone: string;
  items: string;
  cashToCollect: string;
}

/**
 * The order broken into the fields a delivery form asks for, each copyable on
 * its own — pasting one field at a time is how an operator fills a form that
 * takes no URL parameters (Yandex Delivery's own site, for instance).
 */
export function copyRows(
  order: Order,
  restaurant: Restaurant | null | undefined,
  labels: CopyLabels,
): CopyRow[] {
  const rows: CopyRow[] = [];
  const addr = restaurant?.address;
  if (addr?.text)
    rows.push({ key: "from", label: labels.pickupAddress, value: addr.text });
  const shopPhone = digits(restaurant?.phones?.[0] ?? "");
  if (shopPhone)
    rows.push({ key: "fromPhone", label: labels.pickupPhone, value: shopPhone });
  if (order.address?.text)
    rows.push({ key: "to", label: labels.address, value: order.address.text });
  if (order.address?.comment)
    rows.push({
      key: "toComment",
      label: labels.comment,
      value: order.address.comment,
    });
  if (order.address?.lat)
    rows.push({
      key: "coords",
      label: labels.coords,
      value: `${order.address.lat.toFixed(6)}, ${order.address.lng.toFixed(6)}`,
    });
  rows.push({ key: "name", label: labels.customer, value: order.customer.name });
  rows.push({
    key: "phone",
    label: labels.phone,
    value: digits(order.customer.phone),
  });
  rows.push({ key: "items", label: labels.items, value: itemsLine(order) });
  // Cash orders are the one detail a courier must not get wrong.
  if (order.paymentMethod === "cash")
    rows.push({
      key: "cash",
      label: labels.cashToCollect,
      value: String(order.total),
    });
  return rows;
}

/**
 * The order details as a plain block the operator can read out on the phone or
 * paste into a provider's own form.
 */
export function orderSummaryText(
  order: Order,
  restaurant: Restaurant | null | undefined,
  labels: {
    order: string;
    from: string;
    customer: string;
    phone: string;
    address: string;
    comment: string;
    items: string;
    total: string;
    payment: string;
    cash: string;
    online: string;
  },
): string {
  const lines = [`${labels.order}: #${order.number}`];
  if (restaurant?.address?.text) {
    const shopPhone = digits(restaurant.phones?.[0] ?? "");
    lines.push(
      `${labels.from}: ${restaurant.name} — ${restaurant.address.text}` +
        (shopPhone ? ` (${shopPhone})` : ""),
    );
  }
  lines.push(`${labels.customer}: ${order.customer.name}`);
  lines.push(`${labels.phone}: ${digits(order.customer.phone)}`);
  if (order.address?.text) lines.push(`${labels.address}: ${order.address.text}`);
  if (order.address?.comment)
    lines.push(`${labels.comment}: ${order.address.comment}`);
  if (order.address?.lat)
    lines.push(`${order.address.lat.toFixed(6)}, ${order.address.lng.toFixed(6)}`);
  lines.push(`${labels.items}: ${itemsLine(order)}`);
  lines.push(
    `${labels.payment}: ${
      order.paymentMethod === "cash"
        ? `${labels.cash} — ${order.total}`
        : labels.online
    }`,
  );
  lines.push(`${labels.total}: ${order.total}`);
  return lines.join("\n");
}
