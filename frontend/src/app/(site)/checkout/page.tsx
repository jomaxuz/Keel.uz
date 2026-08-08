"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { api, ApiError } from "@/lib/api";
import { useCart } from "@/lib/cart";
import { useTelegram } from "@/lib/telegram";
import { useUser } from "@/lib/user";
import { useTable } from "@/lib/table";
import { readBranchCookie, readBrandCookie } from "@/lib/siteBrand";
import { formatPrice, formatUzPhone } from "@/lib/format";
import { PAYMENT_METHODS } from "@/lib/payment";
import { useI18n } from "@/lib/i18n/client";
import { contentName } from "@/lib/i18n/content";
import type { Dict } from "@/lib/i18n";
import { reverseGeocode } from "@/lib/geocode";
import AddressMap, { type LatLng } from "@/components/map/AddressMap";
import AddressPicker from "@/components/map/AddressPicker";
import RouteButtons from "@/components/map/RouteButtons";
import type {
  Branch,
  OrderQuote,
  PaymentMethod,
  Restaurant,
} from "@/lib/types";

const makeSchema = (t: Dict) =>
  z
    .object({
      name: z.string().min(2, t.checkout.err.name),
      phone: z
        .string()
        .min(7, t.checkout.err.phone)
        .regex(/^[+()\d\s-]+$/, t.checkout.err.phoneFormat),
      type: z.enum(["delivery", "pickup", "dinein"]),
      addressText: z.string().optional(),
      comment: z.string().optional(),
      paymentMethod: z.enum(["cash", "payme", "click", "uzum"]),
    })
    .refine(
      (d) => d.type !== "delivery" || (d.addressText?.trim().length ?? 0) >= 3,
      { message: t.checkout.err.address, path: ["addressText"] },
    );

type FormValues = z.infer<ReturnType<typeof makeSchema>>;

// Default map center (Tashkent) until the restaurant profile loads.
const DEFAULT_CENTER: LatLng = { lat: 41.311081, lng: 69.279737 };

export default function CheckoutPage() {
  const router = useRouter();
  const { lines, subtotal, clear } = useCart();
  const { user, loading: userLoading, update: updateUser } = useUser();
  // Only to label the order (see the payload below); the checkout behaves
  // identically in and out of Telegram.
  const { inTelegram } = useTelegram();
  // Set when the guest reached the site by scanning a table's QR code.
  const { table } = useTable();
  const { lang, t } = useI18n();

  const [restaurant, setRestaurant] = useState<Restaurant | null>(null);
  // Which brand's basket this is, and the branches that can serve it. The brand
  // comes from the cookie the site already runs on (see lib/siteBrand).
  const [brandId, setBrandId] = useState("");
  const [branches, setBranches] = useState<Branch[]>([]);
  // Pickup and dine-in: which branch the guest walks into. Delivery never uses
  // it — there the address decides, and the server picks the branch itself.
  const [pickupBranch, setPickupBranch] = useState("");
  const [point, setPoint] = useState<LatLng | null>(null);
  const [quote, setQuote] = useState<OrderQuote | null>(null);
  const [quoting, setQuoting] = useState(false);
  // What the guest typed, and what has actually been applied. Kept apart so a
  // half-typed code does not blank the discount already showing.
  const [codeInput, setCodeInput] = useState("");
  const [appliedCode, setAppliedCode] = useState("");
  // How many cashback points to put towards this order. The server caps it —
  // this is what the guest asked for, not what they get.
  const [usePoints, setUsePoints] = useState(0);
  // Which payment methods this restaurant can actually take. Asked of the
  // server rather than hardcoded: a provider whose keys are only half typed in
  // must not be offered, because the guest would only find that out at the bank.
  const [methods, setMethods] = useState<PaymentMethod[]>(["cash"]);
  const [submitError, setSubmitError] = useState<string | null>(null);
  // Which address the customer is using: an index into the saved list, or "new".
  const [addressMode, setAddressMode] = useState<number | "new">("new");
  const [saveToProfile, setSaveToProfile] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(makeSchema(t)),
    defaultValues: { type: "delivery", paymentMethod: "cash", addressText: "" },
  });

  const type = watch("type");
  const paymentMethod = watch("paymentMethod");

  useEffect(() => {
    api
      .paymentMethods()
      .then((r) => setMethods(r.methods))
      // Cash always works, so a failed lookup costs the guest nothing.
      .catch(() => setMethods(["cash"]));
  }, []);

  // If the chosen method disappears — the owner switched a provider off while
  // the page was open — fall back to cash rather than submitting a method the
  // server will refuse.
  useEffect(() => {
    if (!methods.includes(paymentMethod)) setValue("paymentMethod", "cash");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [methods, paymentMethod]);

  // A guest sitting at a table almost never wants delivery — start there, but
  // leave every other option one tap away.
  useEffect(() => {
    if (table) setValue("type", "dinein");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [table?.id]);
  const addressText = watch("addressText") ?? "";
  const savedAddresses = user?.addresses ?? [];

  function applySavedAddress(i: number) {
    const a = savedAddresses[i];
    if (!a) return;
    setAddressMode(i);
    setSaveToProfile(false);
    setValue("addressText", a.text, { shouldValidate: true });
    setValue("comment", a.comment ?? "");
    if (a.lat && a.lng) setPoint({ lat: a.lat, lng: a.lng });
  }

  function startNewAddress() {
    setAddressMode("new");
    setValue("addressText", "", { shouldValidate: false });
    setPoint(null);
  }

  // Map click → set point and reverse-geocode to fill the address field.
  async function handleMapPick(p: LatLng) {
    setPoint(p);
    try {
      const text = await reverseGeocode(p.lat, p.lng);
      if (text) setValue("addressText", text, { shouldValidate: true });
    } catch {
      /* reverse geocode is best-effort */
    }
  }

  useEffect(() => {
    const scope = {
      brand: readBrandCookie() || undefined,
      branchId: readBranchCookie() || undefined,
    };
    api
      .getRestaurant(scope)
      .then((r) => {
        setRestaurant(r.restaurant);
        setBrandId(r.brand?.id ?? "");
      })
      .catch(() => setRestaurant(null));
    api
      .getBrands()
      .then((d) => setBranches(d.branches))
      .catch(() => {
        /* one branch is the common case; the picker simply stays hidden */
      });
  }, []);

  // Ordering requires an account (see the cart page).
  useEffect(() => {
    if (!userLoading && !user) {
      router.replace("/login?next=/checkout&reason=order");
    }
  }, [user, userLoading, router]);

  // Default to the customer's first saved address.
  useEffect(() => {
    if ((user?.addresses?.length ?? 0) > 0) {
      const a = user!.addresses![0];
      setAddressMode(0);
      setValue("addressText", a.text, { shouldValidate: true });
      if (a.comment) setValue("comment", a.comment);
      if (a.lat && a.lng) setPoint({ lat: a.lat, lng: a.lng });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id]);

  // Prefill the name/phone from the signed-in customer.
  useEffect(() => {
    if (user) {
      const full = [user.firstName, user.lastName].filter(Boolean).join(" ");
      if (full) setValue("name", full);
      if (user.phone) setValue("phone", formatUzPhone(user.phone));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user]);

  // Debounced delivery quote whenever the picked point or subtotal changes.
  // The bill, priced by the server. Runs for every order type now, not just
  // delivery: campaigns and codes apply to pickup and dine-in too.
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (lines.length === 0) return;
    if (type === "delivery" && !point) {
      setQuote(null);
      return;
    }
    if (debounceRef.current) clearTimeout(debounceRef.current);
    setQuoting(true);
    debounceRef.current = setTimeout(() => {
      api
        .orderQuote({
          items: lines.map((l) => ({ menuItemId: l.menuItemId, qty: l.qty })),
          type,
          address: point ? { lat: point.lat, lng: point.lng } : undefined,
          branchId: type === "delivery" ? undefined : pickupBranch,
          brandId,
          promoCode: appliedCode,
          usePoints,
        })
        .then((q) => setQuote(q))
        .catch(() => setQuote(null))
        .finally(() => setQuoting(false));
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [point, lines, type, brandId, appliedCode, pickupBranch, usePoints]);

  // The branches that can serve this brand, and the one the guest collects from.
  const brandBranches = branches.filter(
    (b) => !brandId || b.brandId === brandId,
  );
  useEffect(() => {
    if (pickupBranch || brandBranches.length === 0) return;
    // A table QR already says which room the guest is in; otherwise start at the
    // first branch and let them change it.
    setPickupBranch(table?.branchId ?? brandBranches[0].id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [brandBranches.length, table?.branchId]);
  // Where "come and collect it" points. Falls back to the profile address so a
  // brand-new install with nothing configured still shows something sensible.
  const pickupAt =
    brandBranches.find((b) => b.id === pickupBranch)?.address ??
    restaurant?.address;

  const currency = restaurant?.currency ?? "UZS";
  const minOrder = quote?.minOrder ?? restaurant?.delivery.minOrder ?? 0;
  // Every figure below is the server's. The cart's own subtotal is only a
  // placeholder until the first quote lands — the bill is not the browser's to
  // compute once discounts are in play.
  const discounts = quote?.discounts ?? [];
  const deliveryFee = quote?.deliveryFee ?? 0;
  const total = quote?.total ?? subtotal;
  const belowMin = quote?.belowMinimum ?? false;

  const center: LatLng = restaurant?.address?.lat
    ? { lat: restaurant.address.lat, lng: restaurant.address.lng }
    : DEFAULT_CENTER;

  if (lines.length === 0) {
    return (
      <main className="container-page py-24 text-center">
        <h1 className="font-display text-2xl font-bold">
          {t.checkout.emptyTitle}
        </h1>
        <p className="mt-3 text-ink-muted">{t.checkout.emptyText}</p>
        <Link href="/menu" className="btn-primary mt-6 px-6 py-3">
          {t.common.goToMenu}
        </Link>
      </main>
    );
  }

  async function onSubmit(values: FormValues) {
    setSubmitError(null);

    if (values.type === "delivery") {
      if (!point) {
        setSubmitError(t.checkout.err.pickPoint);
        return;
      }
      if (belowMin) {
        setSubmitError(
          t.checkout.err.minOrder(formatPrice(minOrder, currency, lang)),
        );
        return;
      }
      if (quote && !quote.available) {
        setSubmitError(t.checkout.err.unavailable);
        return;
      }
    }

    const payload = {
      customer: { name: values.name, phone: values.phone },
      type: values.type,
      // Only meaningful for dine-in; the server checks it against the plan.
      tableId: values.type === "dinein" ? table?.id ?? "" : "",
      // Delivery ignores this: the server resolves the branch from the address.
      branchId:
        values.type === "dinein"
          ? table?.branchId ?? pickupBranch
          : values.type === "pickup"
            ? pickupBranch
            : "",
      address:
        values.type === "delivery"
          ? {
              text: values.addressText ?? "",
              lat: point!.lat,
              lng: point!.lng,
              comment: values.comment ?? "",
            }
          : { text: "", lat: 0, lng: 0, comment: values.comment ?? "" },
      items: lines.map((l) => ({
        menuItemId: l.menuItemId,
        name: l.name,
        price: l.price,
        qty: l.qty,
        comment: l.comment ?? "",
        // Always the base (uz) names — the server re-resolves the price delta.
        options: l.options.map((o) => ({
          name: o.group.name,
          choice: o.choice.name,
          priceDelta: o.priceDelta,
        })),
      })),
      paymentMethod: values.paymentMethod,
      // The server re-checks it; a code that stopped being valid between the
      // preview and the tap simply does not apply, and the order still goes.
      promoCode: appliedCode,
      usePoints,
      // ⚠️ Which door this order came in through, and the server can only learn
      // it from here: the mini app **is** this site in Telegram's WebView, so no
      // header, address or session distinguishes them. Attribution, not
      // authorisation — see Order.Channel.
      channel: inTelegram ? "telegram" : "web",
    };

    try {
      const order = await api.createOrder(payload);
      // Optionally remember the freshly typed address on the profile.
      if (saveToProfile && user && values.type === "delivery" && point) {
        try {
          const updated = await api.updateMe({
            firstName: user.firstName,
            addresses: [
              ...(user.addresses ?? []),
              {
                label: "",
                text: values.addressText ?? "",
                lat: point.lat,
                lng: point.lng,
                comment: values.comment ?? "",
              },
            ],
          });
          updateUser(updated);
        } catch {
          // Saving the address is a nicety — never fail the order for it.
        }
      }
      clear();
      // An online method answers with a bank link. Going straight there is the
      // point: a "pay now" button on the next screen is one more tap between a
      // filled basket and money in the till, and the guest who does not take it
      // becomes an unpaid order somebody has to chase.
      if (order.payUrl) {
        window.location.href = order.payUrl;
        return;
      }
      router.push(`/order/${order.number}`);
    } catch (err) {
      const msg = err instanceof ApiError ? err.message : t.checkout.err.failed;
      setSubmitError(msg);
    }
  }

  const inputCls = "input mt-1";

  return (
    <main className="container-page py-10">
      <h1 className="mb-6 font-display text-3xl font-bold tracking-tight">
        {t.checkout.title}
      </h1>

      <form
        onSubmit={handleSubmit(onSubmit)}
        className="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1fr)_340px]"
      >
        <div className="space-y-6">
          {/* Contact */}
          <section className="rounded-3xl border border-line bg-surface shadow-card p-6">
            <h2 className="font-display text-lg font-bold">
              {t.checkout.contact}
            </h2>
            <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
              <label className="block text-sm">
                <span className="font-medium">{t.checkout.name}</span>
                <input className={inputCls} {...register("name")} />
                {errors.name && (
                  <span className="mt-1 block text-xs text-brand">
                    {errors.name.message}
                  </span>
                )}
              </label>
              <label className="block text-sm">
                <span className="font-medium">{t.checkout.phone}</span>
                <input
                  className={inputCls}
                  placeholder="+998 90 123 45 67"
                  {...register("phone")}
                />
                {errors.phone && (
                  <span className="mt-1 block text-xs text-brand">
                    {errors.phone.message}
                  </span>
                )}
              </label>
            </div>
          </section>

          {/* Delivery type */}
          <section className="rounded-3xl border border-line bg-surface shadow-card p-6">
            <h2 className="font-display text-lg font-bold">
              {t.checkout.typeTitle}
            </h2>
            <div className="mt-4 flex gap-3">
              <label className="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border border-line-strong px-4 py-3 text-sm has-[:checked]:border-brand has-[:checked]:bg-brand/5">
                <input type="radio" value="delivery" {...register("type")} />
                <span className="font-medium">{t.checkout.delivery}</span>
              </label>
              <label className="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border border-line-strong px-4 py-3 text-sm has-[:checked]:border-brand has-[:checked]:bg-brand/5">
                <input type="radio" value="pickup" {...register("type")} />
                <span className="font-medium">{t.checkout.pickup}</span>
              </label>
              {table && (
                <label className="flex flex-1 cursor-pointer items-center gap-2 rounded-xl border border-line-strong px-4 py-3 text-sm has-[:checked]:border-brand has-[:checked]:bg-brand/5">
                  <input type="radio" value="dinein" {...register("type")} />
                  <span className="font-medium">{t.table.orderType}</span>
                </label>
              )}
            </div>

            {type === "dinein" && table && (
              <p className="mt-4 rounded-2xl bg-brand-tint/50 px-4 py-3 text-sm font-medium text-brand-dark">
                {t.table.orderTypeHint(table.number)}
              </p>
            )}

            {type === "delivery" && (
              <div className="mt-5 space-y-4">
                {/* Saved profile addresses first; "new address" opens the map. */}
                {savedAddresses.length > 0 && (
                  <div className="text-sm">
                    <span className="font-medium">
                      {t.checkout.savedAddresses}
                    </span>
                    <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
                      {savedAddresses.map((a, i) => (
                        <button
                          key={`${a.text}-${i}`}
                          type="button"
                          onClick={() => applySavedAddress(i)}
                          className={`rounded-2xl border px-4 py-3 text-left transition-colors ${
                            addressMode === i
                              ? "border-brand bg-brand/5"
                              : "border-line-strong hover:border-brand/60"
                          }`}
                        >
                          <span className="block font-semibold">
                            {a.label || t.checkout.address}
                          </span>
                          <span className="block truncate text-xs text-ink-muted">
                            {a.text}
                          </span>
                        </button>
                      ))}
                      <button
                        type="button"
                        onClick={() => startNewAddress()}
                        className={`rounded-2xl border px-4 py-3 text-left transition-colors ${
                          addressMode === "new"
                            ? "border-brand bg-brand/5"
                            : "border-line-strong hover:border-brand/60"
                        }`}
                      >
                        <span className="block font-semibold text-brand">
                          {t.checkout.newAddress}
                        </span>
                        <span className="block text-xs text-ink-muted">
                          {t.checkout.newAddressHint}
                        </span>
                      </button>
                    </div>
                  </div>
                )}

                {/* Map + text input stay in sync (AddressPicker). Shown for a
                    saved address too, so the customer can see the pin against
                    the delivery zones and nudge it if it is off. Editing here
                    only affects this order — the profile address is untouched
                    unless "save to profile" is ticked. */}
                <>
                  <AddressPicker
                    value={{
                      text: addressText,
                      lat: point?.lat ?? 0,
                      lng: point?.lng ?? 0,
                    }}
                    onChange={(next) => {
                      setValue("addressText", next.text, {
                        shouldValidate: true,
                      });
                      if (next.lat && next.lng)
                        setPoint({ lat: next.lat, lng: next.lng });
                    }}
                    center={center}
                    mapClassName="h-72 w-full"
                    inputClassName={inputCls}
                    zones={restaurant?.delivery.zones}
                    currency={currency}
                  />
                  {errors.addressText && (
                    <span className="block text-xs text-brand">
                      {errors.addressText.message}
                    </span>
                  )}
                  {user && addressMode === "new" && (
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={saveToProfile}
                        onChange={(e) => setSaveToProfile(e.target.checked)}
                      />
                      <span>{t.checkout.saveAddress}</span>
                    </label>
                  )}
                </>

                <label className="block text-sm">
                  <span className="font-medium">{t.checkout.comment}</span>
                  <input
                    className={inputCls}
                    placeholder={t.checkout.commentPh}
                    {...register("comment")}
                  />
                </label>
              </div>
            )}

            {type === "pickup" && (
              <div className="mt-5">
                <p className="text-sm text-ink-muted">
                  {t.checkout.pickupText}
                </p>
                {/* Several places to collect from: the guest picks. With one
                    branch — the usual case — nothing is drawn. */}
                {brandBranches.length > 1 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {brandBranches.map((b) => (
                      <button
                        key={b.id}
                        type="button"
                        onClick={() => setPickupBranch(b.id)}
                        className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
                          pickupBranch === b.id
                            ? "border-brand bg-brand-tint text-brand-dark"
                            : "border-line-strong text-ink-soft hover:border-brand"
                        }`}
                      >
                        {b.name}
                      </button>
                    ))}
                  </div>
                )}
                <p className="mt-1 font-medium">
                  {pickupAt?.text || t.checkout.restaurantAddress}
                </p>
                {pickupAt?.lat ? (
                  <>
                    <div className="mt-3">
                      <AddressMap
                        value={{ lat: pickupAt.lat, lng: pickupAt.lng }}
                        center={{ lat: pickupAt.lat, lng: pickupAt.lng }}
                        className="h-64 w-full"
                        readOnly
                      />
                    </div>
                    {/* Straight into the map app the customer already uses —
                        the route is built for them, no address to copy. */}
                    <RouteButtons
                      className="mt-3"
                      target={{
                        lat: pickupAt.lat,
                        lng: pickupAt.lng,
                        label:
                          brandBranches.find((b) => b.id === pickupBranch)
                            ?.name ??
                          restaurant?.name ??
                          "",
                      }}
                    />
                  </>
                ) : (
                  <p className="mt-3 rounded-xl bg-ink/5 px-4 py-3 text-sm text-ink-muted">
                    {t.checkout.noLocation}
                  </p>
                )}
              </div>
            )}
          </section>

          {/* Payment */}
          <section className="rounded-3xl border border-line bg-surface shadow-card p-6">
            <h2 className="font-display text-lg font-bold">
              {t.checkout.payment}
            </h2>
            <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
              {PAYMENT_METHODS.filter((m) => methods.includes(m)).map((m) => (
                <label
                  key={m}
                  className="flex cursor-pointer items-center justify-center gap-2 rounded-xl border border-line-strong px-4 py-3 text-sm has-[:checked]:border-brand has-[:checked]:bg-brand/5"
                >
                  <input
                    type="radio"
                    value={m}
                    className="sr-only"
                    {...register("paymentMethod")}
                  />
                  <span className="font-medium">{t.payment[m]}</span>
                </label>
              ))}
            </div>
            <p className="mt-3 text-xs text-ink-muted/70">
              {t.checkout.paymentNote}
            </p>
          </section>
        </div>

        {/* Summary */}
        <aside className="h-fit rounded-3xl border border-line bg-surface shadow-card p-6">
          <h2 className="font-display text-lg font-bold">{t.checkout.order}</h2>
          <ul className="mt-4 space-y-2 text-sm">
            {lines.map((l) => (
              <li key={l.lineId} className="flex justify-between gap-2">
                <span className="text-ink-muted">
                  {contentName(l, lang)} × {l.qty}
                  {l.options.length > 0 && (
                    <span className="block text-xs text-ink-muted/70">
                      {l.options
                        .map((o) => contentName(o.choice, lang))
                        .join(" · ")}
                    </span>
                  )}
                  {l.comment && (
                    <span className="block text-xs italic text-ink-muted/70">
                      “{l.comment}”
                    </span>
                  )}
                </span>
                <span className="font-medium">
                  {formatPrice(l.price * l.qty, currency, lang)}
                </span>
              </li>
            ))}
          </ul>

          {/* Promo code. Deliberately below the items and above the total: it
              is the last thing a guest tries before paying. */}
          <div className="mt-4 border-t border-line pt-4">
            <label className="block text-sm">
              <span className="font-medium">{t.checkout.promoCode}</span>
              <div className="mt-1 flex gap-2">
                <input
                  className="input flex-1 uppercase"
                  placeholder={t.checkout.promoCodePh}
                  value={codeInput}
                  onChange={(e) => setCodeInput(e.target.value.toUpperCase())}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      setAppliedCode(codeInput.trim());
                    }
                  }}
                />
                <button
                  type="button"
                  onClick={() =>
                    appliedCode
                      ? (setAppliedCode(""), setCodeInput(""))
                      : setAppliedCode(codeInput.trim())
                  }
                  disabled={!codeInput.trim() && !appliedCode}
                  className="btn-ghost shrink-0 px-4 py-2 text-sm disabled:opacity-50"
                >
                  {appliedCode ? t.checkout.promoRemove : t.checkout.promoApply}
                </button>
              </div>
            </label>
            {quote?.codeError && (
              <p className="mt-2 text-xs text-brand">{quote.codeError}</p>
            )}
            {quote?.codeApplied && (
              <p className="mt-2 text-xs text-emerald-600 dark:text-emerald-400">
                {t.checkout.promoApplied}
              </p>
            )}
          </div>

          {/* Cashback. Only shown when there is a balance to spend — an empty
              "0 points" row is noise on every other order. */}
          {(quote?.pointsBalance ?? 0) > 0 && (
            <div className="mt-4 border-t border-line pt-4">
              <div className="flex items-center justify-between gap-3">
                <span className="text-sm font-medium">{t.checkout.points}</span>
                <span className="text-sm text-ink-muted">
                  {t.checkout.pointsBalance(
                    formatPrice(quote!.pointsBalance, currency, lang),
                  )}
                </span>
              </div>
              {quote!.pointsMax > 0 ? (
                <>
                  <div className="mt-2 flex items-center gap-3">
                    <input
                      type="range"
                      min={0}
                      max={quote!.pointsMax}
                      step={1000}
                      value={Math.min(usePoints, quote!.pointsMax)}
                      onChange={(e) => setUsePoints(Number(e.target.value))}
                      className="flex-1 accent-brand"
                    />
                    <span className="w-28 shrink-0 text-right text-sm font-semibold tabular-nums">
                      {formatPrice(quote!.pointsSpent, currency, lang)}
                    </span>
                  </div>
                  <div className="mt-1 flex justify-between text-xs text-ink-muted">
                    <button
                      type="button"
                      onClick={() => setUsePoints(0)}
                      className="hover:text-brand"
                    >
                      {t.checkout.pointsNone}
                    </button>
                    {/* The ceiling exists so an order cannot become free; say
                        so rather than letting the slider stop for no reason. */}
                    <button
                      type="button"
                      onClick={() => setUsePoints(quote!.pointsMax)}
                      className="hover:text-brand"
                    >
                      {t.checkout.pointsMax(
                        formatPrice(quote!.pointsMax, currency, lang),
                      )}
                    </button>
                  </div>
                </>
              ) : null}
            </div>
          )}

          <div className="mt-4 space-y-1 border-t border-line pt-4 text-sm">
            <div className="flex justify-between">
              <span className="text-ink-muted">{t.checkout.itemsLabel}</span>
              <span>{formatPrice(subtotal, currency, lang)}</span>
            </div>
            {/* Each discount on its own line: one lump sum is unanswerable when
                the guest asks why the price moved. */}
            {discounts.map((d, i) => (
              <div
                key={i}
                className="flex justify-between text-emerald-600 dark:text-emerald-400"
              >
                <span className="min-w-0 truncate pr-2">
                  {d.name}
                  {d.code ? ` · ${d.code}` : ""}
                </span>
                <span className="shrink-0">
                  −{formatPrice(d.amount, currency, lang)}
                </span>
              </div>
            ))}
            {(quote?.pointsSpent ?? 0) > 0 && (
              <div className="flex justify-between text-emerald-600 dark:text-emerald-400">
                <span>{t.checkout.pointsUsed}</span>
                <span>−{formatPrice(quote!.pointsSpent, currency, lang)}</span>
              </div>
            )}
            {type === "delivery" && (
              <div className="flex justify-between">
                <span className="text-ink-muted">{t.checkout.delivery}</span>
                <span>
                  {quoting
                    ? "..."
                    : point
                      ? quote?.available
                        ? deliveryFee === 0
                          ? t.checkout.freeDelivery
                          : formatPrice(deliveryFee, currency, lang)
                        : t.checkout.notAvailable
                      : t.checkout.setAddress}
                </span>
              </div>
            )}
            {/* Which kitchen took it. Only worth saying when the company has
                more than one — otherwise it is noise. */}
            {type === "delivery" &&
              quote?.available &&
              quote.branchName &&
              brandBranches.length > 1 && (
                <div className="flex justify-between">
                  <span className="text-ink-muted">{t.checkout.servedBy}</span>
                  <span>{quote.branchName}</span>
                </div>
              )}
          </div>

          <div className="mt-4 flex justify-between border-t border-line pt-4 text-base font-bold">
            <span>{t.checkout.total}</span>
            <span>{formatPrice(total, currency, lang)}</span>
          </div>

          {/* What this order gives back. A promise, not a movement: the points
              land when the order is delivered. */}
          {(quote?.pointsEarn ?? 0) > 0 && (
            <p className="mt-2 text-xs text-emerald-600 dark:text-emerald-400">
              {t.checkout.pointsEarn(
                formatPrice(quote!.pointsEarn, currency, lang),
              )}
            </p>
          )}

          {belowMin && (
            <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
              {t.checkout.minOrder(formatPrice(minOrder, currency, lang))}
            </p>
          )}
          {submitError && (
            <p className="mt-3 rounded-lg bg-rose-50 px-3 py-2 text-xs text-brand dark:bg-rose-500/10 dark:text-rose-300">
              {submitError}
            </p>
          )}

          <button
            type="submit"
            disabled={isSubmitting}
            className="btn-primary mt-6 w-full px-6 py-3 disabled:opacity-60"
          >
            {isSubmitting ? t.checkout.submitting : t.checkout.submit}
          </button>
        </aside>
      </form>
    </main>
  );
}
