"use client";

// Taking an order over the phone.
//
// This is the checkout, rebuilt for somebody who is listening rather than
// reading. Two differences drive the whole layout:
//
//   • The operator is typing what they hear, in the order they hear it. So the
//     menu is a search box first and a category list second — a guest says
//     "lagmon", not "second course, item four".
//   • They have to read the total back down the line. So the bill is always on
//     screen and always the server's number (adminOrderQuote), never one this
//     component worked out for itself.
//
// Prices, discounts and the delivery fee are the server's answers throughout.
// An operator who quotes a figure the receipt then disagrees with has created
// an argument at the door.

import { useCallback, useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";
import { formatPrice } from "@/lib/format";
import { useAdminT } from "@/lib/i18n/admin";
import { useI18n } from "@/lib/i18n/client";
import { contentName } from "@/lib/i18n/content";
import AddressPicker from "@/components/map/AddressPicker";
import Modal from "@/components/admin/Modal";
import type {
  Category,
  MenuItem,
  Order,
  OrderItemOption,
  OrderQuote,
  PaymentMethod,
  Restaurant,
  SiteUser,
  UserAddress,
} from "@/lib/types";

/** A basket line. The key is the dish plus the choices, like the site's cart:
 *  the same dish with different options is a separate line. */
interface Line {
  key: string;
  menuItemId: string;
  name: string;
  price: number;
  qty: number;
  options: OrderItemOption[];
  comment: string;
}

function lineKey(id: string, options: OrderItemOption[]): string {
  if (options.length === 0) return id;
  return `${id}|${options.map((o) => `${o.name}:${o.choice}`).join(",")}`;
}

/** Dishes to start the basket with — "repeat the last order" hands these in. */
export interface SeedLine {
  menuItemId: string;
  name: string;
  qty: number;
  options?: OrderItemOption[] | null;
}

export default function OperatorOrderModal({
  phone,
  customer,
  seed,
  callId,
  onClose,
  onCreated,
}: {
  phone: string;
  /** The account, when the caller has one. Null for a first-time caller: the
   *  server creates the account from the number as the order is placed. */
  customer: SiteUser | null;
  seed?: SeedLine[];
  /** The call being written down right now, so the log row can say what it
   *  produced. */
  callId?: string;
  onClose: () => void;
  onCreated: (order: Order) => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();

  const [menu, setMenu] = useState<MenuItem[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);

  const [name, setName] = useState(
    [customer?.firstName, customer?.lastName].filter(Boolean).join(" ").trim(),
  );
  const [type, setType] = useState<"delivery" | "pickup">("delivery");
  const [address, setAddress] = useState({ text: "", lat: 0, lng: 0 });
  const [addressComment, setAddressComment] = useState("");
  const [payment, setPayment] = useState<PaymentMethod>("cash");
  const [promoCode, setPromoCode] = useState("");
  const [usePoints, setUsePoints] = useState(0);

  const [lines, setLines] = useState<Line[]>([]);
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState("");
  // The dish whose options are being answered. A required group cannot be
  // skipped, so picking such a dish opens this rather than adding a line.
  const [choosing, setChoosing] = useState<MenuItem | null>(null);

  const [quote, setQuote] = useState<OrderQuote | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([api.adminMenu(), api.adminCategories(), api.getRestaurant()])
      .then(([m, c, r]) => {
        setMenu(m);
        setCategories(c.filter((x) => x.isActive));
        setRestaurant(r.restaurant);
      })
      .catch(() => setError(t.common.loadFailed));
  }, [t]);

  // "Repeat the last order" — filled once, from what the caller had before.
  useEffect(() => {
    if (!seed || seed.length === 0) return;
    setLines(
      seed.map((s) => {
        const options = s.options ?? [];
        return {
          key: lineKey(s.menuItemId, options),
          menuItemId: s.menuItemId,
          name: s.name,
          // A placeholder: the bill below is the server's, and so is the
          // receipt. Nothing the operator sees comes from this number.
          price: 0,
          qty: s.qty,
          options,
          comment: "",
        };
      }),
    );
  }, [seed]);

  const subtotalItems = useMemo(
    () => lines.map((l) => ({ menuItemId: l.menuItemId, qty: l.qty, options: l.options })),
    [lines],
  );

  // The bill, recomputed by the server whenever the basket or the address
  // moves. Debounced because the operator is typing while the guest talks.
  useEffect(() => {
    if (lines.length === 0) {
      setQuote(null);
      return;
    }
    const timer = setTimeout(() => {
      api
        .adminOrderQuote({
          items: subtotalItems,
          type,
          address:
            type === "delivery" && address.lat
              ? { lat: address.lat, lng: address.lng }
              : undefined,
          userId: customer?.id,
          promoCode: promoCode.trim() || undefined,
          usePoints: usePoints || undefined,
        })
        .then(setQuote)
        .catch(() => setQuote(null));
    }, 350);
    return () => clearTimeout(timer);
  }, [subtotalItems, lines.length, type, address, customer?.id, promoCode, usePoints]);

  const addLine = useCallback((dish: MenuItem, options: OrderItemOption[]) => {
    const key = lineKey(dish.id, options);
    setLines((prev) => {
      const found = prev.find((l) => l.key === key);
      if (found) {
        return prev.map((l) => (l.key === key ? { ...l, qty: l.qty + 1 } : l));
      }
      const unit =
        dish.price + options.reduce((sum, o) => sum + o.priceDelta, 0);
      return [
        ...prev,
        {
          key,
          menuItemId: dish.id,
          name: dish.name,
          price: unit,
          qty: 1,
          options,
          comment: "",
        },
      ];
    });
  }, []);

  function pick(dish: MenuItem) {
    // A dish with any option group has to be answered, not guessed at.
    if (dish.options && dish.options.length > 0) {
      setChoosing(dish);
      return;
    }
    addLine(dish, []);
  }

  function setQty(key: string, qty: number) {
    setLines((prev) =>
      qty <= 0
        ? prev.filter((l) => l.key !== key)
        : prev.map((l) => (l.key === key ? { ...l, qty } : l)),
    );
  }

  const visibleDishes = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return menu
      .filter((d) => d.isAvailable && !d.soldOut)
      .filter((d) => !categoryId || d.categoryId === categoryId)
      .filter(
        (d) =>
          !needle ||
          contentName(d, lang).toLowerCase().includes(needle) ||
          d.name.toLowerCase().includes(needle),
      )
      .slice(0, 60);
  }, [menu, search, categoryId, lang]);

  const currency = restaurant?.currency ?? "UZS";
  const center = restaurant?.address
    ? { lat: restaurant.address.lat, lng: restaurant.address.lng }
    : { lat: 41.311, lng: 69.24 };

  const canSubmit =
    lines.length > 0 &&
    name.trim().length > 0 &&
    (type === "pickup" || (address.lat !== 0 && address.text.trim() !== "")) &&
    !quote?.belowMinimum &&
    (type === "pickup" || quote?.available !== false);

  async function submit() {
    setSaving(true);
    setError("");
    try {
      const order = await api.adminCreateOrder({
        customer: { name: name.trim(), phone },
        userId: customer?.id,
        type,
        address:
          type === "delivery"
            ? { ...address, comment: addressComment.trim() }
            : { text: "", lat: 0, lng: 0, comment: "" },
        items: lines.map((l) => ({
          menuItemId: l.menuItemId,
          name: l.name,
          qty: l.qty,
          options: l.options,
          comment: l.comment.trim(),
        })),
        paymentMethod: payment,
        promoCode: promoCode.trim() || undefined,
        usePoints: usePoints || undefined,
        callId,
      });
      onCreated(order);
    } catch (e) {
      setError(e instanceof Error ? e.message : t.common.saveFailed);
    } finally {
      setSaving(false);
    }
  }

  const savedAddresses: UserAddress[] = customer?.addresses ?? [];

  return (
    <Modal onClose={onClose} wide>
      <div className="max-w-none">
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 className="font-display text-xl font-bold">{t.calls.orderTitle}</h2>
            <p className="text-sm text-ink-muted">{phone}</p>
          </div>
          <button type="button" onClick={onClose} className="btn btn-ghost">
            {t.common.close}
          </button>
        </div>

        <div className="grid gap-5 lg:grid-cols-2">
          {/* ---- Left: the menu ---- */}
          <div>
            <input
              className="input"
              placeholder={t.calls.menuSearch}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              autoFocus
            />
            <div className="mt-2 flex flex-wrap gap-1">
              <button
                type="button"
                onClick={() => setCategoryId("")}
                className={`chip ${categoryId === "" ? "bg-brand text-white" : ""}`}
              >
                {t.common.none}
              </button>
              {categories.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => setCategoryId(c.id)}
                  className={`chip ${categoryId === c.id ? "bg-brand text-white" : ""}`}
                >
                  {contentName(c, lang)}
                </button>
              ))}
            </div>
            <div className="mt-3 max-h-[45vh] space-y-1 overflow-y-auto overscroll-contain pr-1">
              {visibleDishes.map((d) => (
                <button
                  key={d.id}
                  type="button"
                  onClick={() => pick(d)}
                  className="flex w-full items-center justify-between gap-3 rounded-lg border border-line px-3 py-2 text-left text-sm transition-colors hover:border-brand"
                >
                  <span className="truncate">{contentName(d, lang)}</span>
                  <span className="shrink-0 font-semibold">
                    {formatPrice(d.price, currency, lang)}
                    {d.options && d.options.length > 0 && (
                      <span className="ml-2 text-xs font-normal text-ink-muted">
                        {t.calls.choose}
                      </span>
                    )}
                  </span>
                </button>
              ))}
            </div>
          </div>

          {/* ---- Right: the basket and the bill ---- */}
          <div className="space-y-4">
            <div>
              <label className="text-xs font-semibold text-ink-muted">
                {t.calls.customerName}
              </label>
              <input
                className="input mt-1"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t.calls.customerName}
              />
            </div>

            <div className="flex gap-2">
              {(["delivery", "pickup"] as const).map((v) => (
                <button
                  key={v}
                  type="button"
                  onClick={() => setType(v)}
                  className={`chip flex-1 ${type === v ? "bg-brand text-white" : ""}`}
                >
                  {v === "delivery" ? t.calls.delivery : t.calls.pickup}
                </button>
              ))}
            </div>

            {type === "delivery" && (
              <div>
                {savedAddresses.length > 0 && (
                  <div className="mb-2">
                    <span className="text-xs font-semibold text-ink-muted">
                      {t.calls.savedAddresses}
                    </span>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {savedAddresses.map((a, i) => (
                        <button
                          key={i}
                          type="button"
                          onClick={() =>
                            setAddress({ text: a.text, lat: a.lat, lng: a.lng })
                          }
                          className="chip max-w-full truncate"
                          title={a.text}
                        >
                          {a.label || a.text}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
                <AddressPicker
                  value={address}
                  onChange={setAddress}
                  center={center}
                  mapClassName="h-44 w-full"
                  zones={restaurant?.delivery.zones}
                  currency={currency}
                />
                <input
                  className="input mt-2"
                  placeholder={t.calls.addressComment}
                  value={addressComment}
                  onChange={(e) => setAddressComment(e.target.value)}
                />
              </div>
            )}

            <div>
              <span className="text-xs font-semibold text-ink-muted">
                {t.calls.cart}
              </span>
              {lines.length === 0 ? (
                <p className="mt-1 text-sm text-ink-muted">{t.calls.cartEmpty}</p>
              ) : (
                <div className="mt-1 space-y-2">
                  {lines.map((l) => (
                    <div key={l.key} className="rounded-lg border border-line p-2">
                      <div className="flex items-center gap-2">
                        <span className="flex-1 text-sm font-medium">{l.name}</span>
                        <div className="flex items-center gap-1">
                          <button
                            type="button"
                            onClick={() => setQty(l.key, l.qty - 1)}
                            className="h-7 w-7 rounded-full border border-line-strong"
                          >
                            −
                          </button>
                          <span className="w-6 text-center text-sm font-semibold">
                            {l.qty}
                          </span>
                          <button
                            type="button"
                            onClick={() => setQty(l.key, l.qty + 1)}
                            className="h-7 w-7 rounded-full border border-line-strong"
                          >
                            +
                          </button>
                        </div>
                      </div>
                      {l.options.length > 0 && (
                        <p className="text-xs text-ink-muted">
                          {l.options.map((o) => o.choice).join(", ")}
                        </p>
                      )}
                      <input
                        className="input mt-1 text-xs"
                        placeholder={t.calls.itemComment}
                        value={l.comment}
                        onChange={(e) =>
                          setLines((prev) =>
                            prev.map((x) =>
                              x.key === l.key
                                ? { ...x, comment: e.target.value }
                                : x,
                            ),
                          )
                        }
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-xs font-semibold text-ink-muted">
                  {t.calls.payment}
                </label>
                <select
                  className="input mt-1"
                  value={payment}
                  onChange={(e) => setPayment(e.target.value as PaymentMethod)}
                >
                  <option value="cash">{t.calls.cash}</option>
                  <option value="payme">Payme</option>
                  <option value="click">Click</option>
                  <option value="uzum">Uzum</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-semibold text-ink-muted">
                  {t.calls.promoCode}
                </label>
                <input
                  className="input mt-1"
                  value={promoCode}
                  onChange={(e) => setPromoCode(e.target.value)}
                />
              </div>
            </div>

            {/* Points are the caller's own money, so they are offered only when
                there are some — and capped by the server, which answers with
                what it actually allowed. */}
            {(quote?.pointsMax ?? 0) > 0 && (
              <div>
                <label className="text-xs font-semibold text-ink-muted">
                  {t.calls.usePoints}{" "}
                  <span className="font-normal">
                    {t.calls.pointsAvailable(
                      formatPrice(quote?.pointsBalance ?? 0, currency, lang),
                    )}
                  </span>
                </label>
                <input
                  type="number"
                  min={0}
                  max={quote?.pointsMax ?? 0}
                  className="input mt-1"
                  value={usePoints || ""}
                  onChange={(e) => setUsePoints(Number(e.target.value) || 0)}
                />
              </div>
            )}

            {/* The bill, as the server sees it. */}
            <div className="rounded-xl bg-ink/5 p-3 text-sm">
              <Row
                label={t.calls.subtotal}
                value={formatPrice(quote?.subtotal ?? 0, currency, lang)}
              />
              {(quote?.discountTotal ?? 0) > 0 && (
                <Row
                  label={t.calls.discount}
                  value={`− ${formatPrice(quote?.discountTotal ?? 0, currency, lang)}`}
                />
              )}
              {type === "delivery" && (
                <Row
                  label={t.calls.deliveryFee}
                  value={formatPrice(quote?.deliveryFee ?? 0, currency, lang)}
                />
              )}
              <div className="mt-2 flex items-center justify-between border-t border-line pt-2 text-base font-bold">
                <span>{t.calls.total}</span>
                <span>{formatPrice(quote?.total ?? 0, currency, lang)}</span>
              </div>
              {quote?.codeError && (
                <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                  {quote.codeError}
                </p>
              )}
              {quote?.belowMinimum && (
                <p className="mt-1 text-xs text-brand">
                  {t.calls.belowMinimum(
                    formatPrice(quote.minOrder, currency, lang),
                  )}
                </p>
              )}
              {type === "delivery" && quote && !quote.available && (
                <p className="mt-1 text-xs text-brand">{t.calls.notDeliverable}</p>
              )}
            </div>

            {error && <p className="text-sm text-brand">{error}</p>}

            <button
              type="button"
              disabled={!canSubmit || saving}
              onClick={submit}
              className="btn btn-primary w-full disabled:opacity-50"
            >
              {saving ? t.calls.submitting : t.calls.submit}
            </button>
          </div>
        </div>
      </div>

      {choosing && (
        <OptionPicker
          dish={choosing}
          currency={currency}
          onCancel={() => setChoosing(null)}
          onDone={(options) => {
            addLine(choosing, options);
            setChoosing(null);
          }}
        />
      )}
    </Modal>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between text-ink-muted">
      <span>{label}</span>
      <span className="font-medium text-ink">{value}</span>
    </div>
  );
}

/** Answering a dish's option groups. Required groups block the "add" button —
 *  the server refuses the order otherwise, and finding that out at the end of
 *  a phone call is too late. */
function OptionPicker({
  dish,
  currency,
  onDone,
  onCancel,
}: {
  dish: MenuItem;
  currency: string;
  onDone: (options: OrderItemOption[]) => void;
  onCancel: () => void;
}) {
  const t = useAdminT();
  const { lang } = useI18n();
  const groups = dish.options ?? [];
  const [picked, setPicked] = useState<Record<string, string[]>>({});

  function toggle(group: string, choice: string, multiple: boolean) {
    setPicked((prev) => {
      const current = prev[group] ?? [];
      if (!multiple) return { ...prev, [group]: [choice] };
      return {
        ...prev,
        [group]: current.includes(choice)
          ? current.filter((c) => c !== choice)
          : [...current, choice],
      };
    });
  }

  const missing = groups.some(
    (g) => g.required && (picked[g.name]?.length ?? 0) === 0,
  );

  function done() {
    const out: OrderItemOption[] = [];
    for (const g of groups) {
      for (const choiceName of picked[g.name] ?? []) {
        const choice = g.choices.find((c) => c.name === choiceName);
        if (choice) {
          out.push({
            name: g.name,
            choice: choice.name,
            priceDelta: choice.priceDelta,
          });
        }
      }
    }
    onDone(out);
  }

  return (
    <Modal onClose={onCancel}>
      <h3 className="font-display text-lg font-bold">{contentName(dish, lang)}</h3>
      <div className="mt-3 space-y-4">
        {groups.map((g) => (
          <div key={g.name}>
            <div className="text-sm font-semibold">
              {contentName(g, lang)}
              {g.required && <span className="text-brand"> *</span>}
            </div>
            <div className="mt-1 flex flex-wrap gap-1">
              {g.choices.map((c) => {
                const active = (picked[g.name] ?? []).includes(c.name);
                return (
                  <button
                    key={c.name}
                    type="button"
                    onClick={() => toggle(g.name, c.name, g.multiple)}
                    className={`chip ${active ? "bg-brand text-white" : ""}`}
                  >
                    {contentName(c, lang)}
                    {c.priceDelta !== 0 && (
                      <span className="ml-1 text-xs">
                        {c.priceDelta > 0 ? "+" : "−"}
                        {formatPrice(Math.abs(c.priceDelta), currency, lang)}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          </div>
        ))}
      </div>
      {missing && (
        <p className="mt-3 text-xs text-brand">{t.calls.optionsRequired}</p>
      )}
      <div className="mt-4 flex gap-2">
        <button
          type="button"
          disabled={missing}
          onClick={done}
          className="btn btn-primary flex-1 disabled:opacity-50"
        >
          {t.common.add}
        </button>
        <button type="button" onClick={onCancel} className="btn btn-ghost">
          {t.common.cancel}
        </button>
      </div>
    </Modal>
  );
}
