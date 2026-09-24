# Keel Open API — v1

> Tashqi dasturchilar uchun e'lon qilingan shartnoma. Inglizcha, chunki uni
> boshqa kompaniyaning dasturchisi o'qiydi. Qarorlar va "nega shunday" —
> `docs/DECISIONS.md` → «Ochiq API: kalitlar va webhook'lar».
> ⚠️ Bu yerda yozilgan har nom (yo'l, maydon, header, xato kodi, hodisa) —
> **va'da**: o'zgartirish kerak bo'lsa `/v2` yoniga qo'yiladi, `/v1` ostida
> o'zgarmaydi.

Every restaurant on Keel runs its own server on its own domain. Everything
below is relative to that domain:

```
https://<restaurant-domain>/api/open/v1
```

The owner finds the exact address in the panel: **Settings → Integrations →
API and webhooks**.

## Authentication

The restaurant owner creates a key in the panel and gives it to you. It looks
like `keel_` followed by 64 hex characters and is shown to them **once**.

```
Authorization: Bearer keel_…
```

Keys carry scopes:

| Scope | Allows |
|---|---|
| `menu:read` | `GET /branches/{id}/menu` |
| `orders:read` | `GET /orders`, `GET /orders/{ref}` — includes customer name and phone |
| `finance:read` | `GET /money`, `/money/daily`, `/balances`, `/fiscal` — money, **no customer data** |

`/ping` and `/branches` need any valid key. A revoked key stops working on the
next request.

⚠️ **Server to server only.** Never put a key in a browser or a mobile app —
anyone can read it there. Cross-origin browser requests are not allowed.

Rate limit: 120 requests per minute per IP address (`429` beyond that).

## Errors

```json
{ "error": { "code": "insufficient_scope", "message": "this API key does not have the orders:read scope" } }
```

| HTTP | `code` |
|---|---|
| 400 | `invalid_request` |
| 401 | `unauthorized` |
| 403 | `insufficient_scope` |
| 404 | `not_found` |
| 500 | `internal_error` |

Branch on `code`; `message` is for humans and may change.

## Conventions

- Money is an **integer number of so'm** (UZS). No decimals, no tiyin.
- Times are RFC 3339 in UTC (`2026-09-24T12:30:15.69Z`).
- Ids are 24-character hex strings. An absent id is an empty string or a
  missing field, never `"000000000000000000000000"`.
- Lists are always arrays, never `null`.
- New fields may be added to any object at any time. Ignore what you do not know.

## Endpoints

### `GET /ping`

Check your key.

```json
{ "restaurant": "Rayhon", "apiVersion": "v1",
  "key": { "name": "amoCRM", "prefix": "keel_654e48", "scopes": ["orders:read"] } }
```

### `GET /branches`

```json
{ "branches": [
  { "id": "6ab5…", "name": "Chilonzor", "phones": ["+998 90 000 00 00"],
    "address": "Toshkent, Chilonzor 9", "lat": 41.28, "lng": 69.20 } ] }
```

### `GET /branches/{branchId}/menu` — `menu:read`

The menu of a branch, with what can be ordered there **right now**.

```json
{ "branchId": "6ab5…",
  "categories": [ { "id": "…", "name": "Milliy taomlar", "nameRu": "Национальные блюда", "nameEn": "Uzbek classics", "sortOrder": 0 } ],
  "items": [ { "id": "…", "categoryId": "…", "name": "Osh (palov)", "nameRu": "…", "nameEn": "…",
               "description": "…", "price": 45000, "imageUrl": "https://…/uploads/osh.jpg",
               "available": true, "tags": [], "barcode": "" } ] }
```

`available: false` means the restaurant would refuse it on its own site too:
switched off, on a stop list, a combo with a sold-out part, or a daily limit
reached.

### `GET /orders` — `orders:read`

Newest first.

| Query | |
|---|---|
| `branchId` | only this branch |
| `status` | `pending`, `confirmed`, `preparing`, `on_the_way`, `delivered`, `cancelled` |
| `createdFrom`, `createdTo` | RFC 3339; from inclusive, to exclusive |
| `limit` | 1–100, default 50 |
| `cursor` | `nextCursor` from the previous page |

```json
{ "orders": [ Order, … ], "nextCursor": "MTc1…" }
```

`nextCursor` is `""` on the last page. Paging by cursor never skips or repeats
an order while new ones arrive.

### `GET /orders/{ref}` — `orders:read`

`ref` is the order number (`QVUU-R97U`) or its id.

```json
{ "order": Order }
```

### The Order object

```json
{
  "id": "6ab517d7c71997fe78741fed",
  "number": "QVUU-R97U",
  "branchId": "6ab51777ac108b6304170794",
  "type": "pickup",
  "channel": "web",
  "status": "confirmed",
  "cancelReason": "",
  "customer": { "name": "Aziz", "phone": "+998901112233" },
  "address": { "text": "…", "lat": 41.3, "lng": 69.2, "comment": "…" },
  "tableNumber": "",
  "items": [
    { "menuItemId": "…", "name": "Osh (palov)", "price": 45000, "qty": 2,
      "options": [ { "group": "Hajmi", "choice": "Katta", "priceDelta": 10000 } ],
      "comment": "" } ],
  "subtotal": 90000, "discountTotal": 0, "deliveryFee": 0, "serviceCharge": 0, "total": 90000,
  "paymentMethod": "cash", "paymentStatus": "unpaid",
  "scheduledAt": "2026-09-24T14:00:00Z",
  "statusHistory": [ { "status": "pending", "at": "…" }, { "status": "confirmed", "at": "…" } ],
  "createdAt": "2026-09-24T12:30:15.69Z"
}
```

- `type`: `delivery`, `pickup`, `dinein` (a table in the restaurant),
  `uzum_tezkor` (carried by Uzum's courier).
- `address` is present only when the order has one.
- `cancelReason`, `tableNumber`, `scheduledAt`, `comment` are omitted when empty.
- `price` is per unit and already includes the options.

## The money ledger — `GET /money` (`finance:read`)

Every movement of money in the restaurant, one entry each, **already
classified**. Built for accounting services: you get amounts, methods,
suppliers and staff names — never a guest's name or phone.

| Query | |
|---|---|
| `from`, `to` | required. The restaurant's calendar days, both inclusive (`2026-09-01`), or RFC 3339 instants. At most 31 days. |
| `branchId` | only this branch |

```json
{
  "from": "2026-08-31T19:00:00Z", "to": "2026-09-30T19:00:00Z", "currency": "UZS",
  "entries": [ MoneyEntry, … ],
  "totals": {
    "byClass": { "revenue": { "in": 90000, "out": 0, "count": 1 }, "cost": { … } },
    "pnlNet": -4910000
  }
}
```

### Read `class` and `pnl` before anything else

Not every movement of money is income or an expense. Money moves between the
till, the safe and the bank; cash goes to a buyer before it becomes food; an
aggregator passes on money the restaurant already earned. Adding those up as
income or costs counts the same money twice. **Every entry says what it is:**

| `class` | `pnl` | What it is |
|---|---|---|
| `revenue` | ✅ | A paid sale, on the day it was made. `sale` has the breakdown. |
| `refund` | ✅ | A sale's money handed back, on the day it was handed back. |
| `cost` | ✅ | A purchase from a supplier, an expense (rent, utilities, tax, repairs), or an outside delivery service. |
| `payroll` | ✅ | Wages paid — staff and couriers. |
| `commission` | ✅ | What an aggregator or acquirer kept. |
| `transfer` | ❌ | The same money in another place: till → bank (collection), aggregator → bank (payout), courier → till. **Not income.** |
| `advance` | ❌ | Cash handed to an employee to spend (typically the market run). It becomes a cost when it buys something — **that purchase is already a `cost` entry**. |
| `manual` | ❌ | Cash put into or taken out of a drawer or the safe by hand, with the cashier's own `category`. Often the same money a purchase or expense already records — shown, not booked. |
| `variance` | ❌ | A drawer counted over (`in`) or under (`out`) at closing. Missing money, not money spent. |

**The restaurant's own money report** (in the Keel panel) is the sum of the
`pnl: true` entries: `pnlNet`. If your figure for a period differs from
`pnlNet`, something is counted twice.

Discounts and loyalty points are **not** entries: no money moved. They are
inside a sale (`sale.discountTotal`, `sale.pointsSpent`); `amount` is what was
actually charged.

### MoneyEntry

```json
{
  "id": "expense_6ab5235b666e66387034a534",
  "source": "expense",
  "class": "cost", "pnl": true,
  "direction": "out", "amount": 5000000,
  "occurredAt": "2026-09-23T19:00:00Z", "day": "2026-09-24",
  "branchId": "6ab52356666e66387034a523",
  "method": "transfer", "category": "ijara",
  "ref": { "type": "expense", "id": "6ab5235b666e66387034a534" }
}
```

- `amount` is always positive; `direction` (`in` / `out`, from the restaurant's
  side) gives the sign.
- `day` is the restaurant's calendar day (Asia/Tashkent). **Use it for daily
  and monthly grouping** — cutting `occurredAt` (UTC) to a date puts everything
  after 19:00 local on the previous day.
- `source`: `sale`, `refund`, `delivery_service`, `purchase`, `expense`,
  `salary`, `courier_pay`, `payout`, `collection`, `advance`, `cash_entry`,
  `safe_entry`, `courier_settlement`, `shift_variance`. New sources may be
  added; **always decide by `class`**.
- Optional: `method` (cash, card, transfer, payme, click, uzum, …),
  `methodName` (the till button's name), `category` (the restaurant's own
  word, free text), `counterparty` (supplier, employee or aggregator),
  `from` / `to` (for transfers: till, safe, bank, aggregator, courier, staff),
  `note`.
- `sale` (revenue only): `type`, `channel`, `subtotal`, `discountTotal`,
  `pointsSpent`, `deliveryFee`, `serviceCharge`.
- `paid` (purchases only): `{ "paid": false }` — booked when the goods
  arrived; `paidAt` once the supplier was paid.

### A period is never final — re-read and replace

The ledger is computed from the restaurant's documents at the moment you ask.
Documents get corrected: an expense typed twice is deleted, a purchase's price
is fixed, a debt is repaid, a refund is given a week later. So:

- **Fetch a period again and replace what you stored for it** — match by `id`,
  and delete the ids that are gone. Do not append.
- Re-read at least the last 7 days every day, and the whole previous month
  after the month has closed.
- `id` is stable: the same document always gives the same id.

## Daily totals — `GET /money/daily` (`finance:read`)

The ledger summed per day and branch — the same entries as `/money`, added up
here, so the two never disagree. Same query as `/money`, up to **93 days**.

```json
{ "currency": "UZS", "from": "…", "to": "…",
  "days": [
    { "day": "2026-09-24", "branchId": "6ab5…",
      "byClass": { "revenue": { "in": 4200000, "out": 0, "count": 61 },
                   "cost":    { "in": 0, "out": 5000000, "count": 1 } },
      "revenueByMethod": { "cash": 2600000, "payme": 1600000 },
      "pnlNet": -800000 } ] }
```

## Where the money is — `GET /balances` (`finance:read`)

Right now, optionally for one `branchId`. The same figures as the owner's
"Where is the money" screen.

```json
{ "asOf": "…", "currency": "UZS", "branchId": "",
  "cash":      [ { "kind": "safe", "name": "Seyf", "amount": 1200000, "counted": false },
                 { "kind": "drawer", "name": "…", "amount": 350000, "counted": false, "at": "…" },
                 { "kind": "courier", … }, { "kind": "advance", … } ],
  "cashTotal": 1550000,
  "bank":      [ { "kind": "bank", "name": "Kapitalbank", "amount": 48000000, "counted": true, "at": "…" } ],
  "bankTotal": 48000000,
  "inTransit": [ { "kind": "rail", "name": "Uzum", "amount": 3100000 } ],
  "inTransitTotal": 3100000,
  "cashLimit": 5000000, "overCashLimit": false,
  "payables":    { "suppliers": { "amount": 7400000, "count": 5 } },
  "receivables": { "guestDebt": { "amount": 260000, "count": 3 } } }
```

- ⚠️ **There is deliberately no grand total.** Cash can be spent tonight, the
  bank this week, and money an aggregator still holds when they decide.
  Adding them gives a number that is true of nothing.
- `counted: true` — somebody counted it (a bank balance read off the bank's
  app); it goes stale, check `at`. `counted: false` — added up from documents;
  it is wrong when a document is missing.
- The bank figure is the **last counted balance**, never derived from
  movements: money reaches the account from places this system does not see.
- `name` is the restaurant's own wording; decide by `kind`
  (`safe`, `drawer`, `courier`, `advance`, `bank`, `rail`).
- `payables.suppliers` — deliveries not yet paid for; `receivables.guestDebt` —
  meals on the slate not yet paid.

## Fiscal receipts — `GET /fiscal` (`finance:read`)

Receipts filed with the tax committee for sales made in the period (≤ 31
days), **in every status** — a pending or refused receipt is exactly what has
to be chased before the month closes — and the Z-reports of shifts closed in
it.

```json
{ "from": "…", "to": "…",
  "receipts": [ { "orderId": "…", "orderNumber": "QVUU-R97U", "branchId": "…",
                  "kind": "sale", "total": 90000, "method": "cash",
                  "status": "filed", "provider": "multikassa",
                  "fiscalSign": "…", "receiptId": "…", "qrText": "…",
                  "filedAt": "…", "soldAt": "…" } ],
  "zReports": [ { "shiftId": "…", "branchId": "…", "number": "142",
                  "saleCash": 2600000, "saleCard": 1600000, "saleTotal": 4200000,
                  "saleCount": 61, "refundTotal": 0, "closedAt": "…" } ] }
```

`kind` is `sale` or `refund`; `status` is `pending`, `filed` or `failed`, and `error` says why a filing failed.

## Webhooks

The owner registers an `https://` address in the panel and picks events. They
receive a **signing secret** (`whsec_…`, shown once) to give to you.

### Events

| `type` | When |
|---|---|
| `order.created` | an order is placed — site, app, phone operator, till, Uzum Tezkor |
| `order.status_changed` | its status changes |
| `money.day_changed` | a day's money changed — see below |
| `ping` | the owner pressed "Test" in the panel |

### `money.day_changed`

```json
{ "id": "evt_money_2026-09-24_6ab5…_1f3a9c0e2b7d", "type": "money.day_changed",
  "createdAt": "…",
  "data": { "day": "2026-09-24", "branchId": "6ab5…", "pnlNet": -800000, "entries": 62 } }
```

**A hint, not the money: re-read that day** (`GET /money?from=<day>&to=<day>`)
and replace what you stored for it. It is raised for anything that changes the
day's figures — a new sale, an edited purchase, a deleted expense — within the
last 35 days, checked every 10 minutes. `entries: 0` means every entry of the
day was deleted.

Nothing is sent for the days before the address was registered: read those
periods yourself once.

### Request

```
POST <your url>
Content-Type: application/json
User-Agent: Keel-Webhooks/1
Keel-Event: order.status_changed
Keel-Event-Id: evt_6ab517d7c71997fe78741fed_1
Keel-Delivery-Id: 6ab517d8…
Keel-Signature: t=1758717015,v1=5f1c…
```

```json
{
  "id": "evt_6ab517d7c71997fe78741fed_1",
  "type": "order.status_changed",
  "createdAt": "2026-09-24T12:30:15.727Z",
  "data": { "status": "confirmed", "previousStatus": "pending", "order": Order }
}
```

- `data.status` is the status **this event** is about. `data.order` is a
  snapshot taken when the event was queued, so for events that follow each
  other quickly `data.order.status` can already be the later one.
- `createdAt` is when the change happened. A till that was offline sends its
  sales later, dated when they were made.

### Answer quickly

Answer with any `2xx` within **10 seconds**, then do your work. Anything else —
another status, a timeout, a redirect — counts as a failure. Redirects are not
followed.

### Retries

After a failure we try again after 1 min, 5 min, 30 min, 2 h, 6 h, 12 h and
24 h (8 attempts, about two days). The owner sees every attempt in the panel
and can resend a failed one.

### Deliver once, maybe more — deduplicate by `id`

The same event (`id`, also in `Keel-Event-Id`) can arrive more than once — a
retry after your `200` got lost, or the owner resending it. Store the ids you
have processed and ignore repeats.

Events of one restaurant are sent one at a time, oldest first, but a retry can
arrive after a newer event. **Order by `createdAt`**, not by arrival.

### Verify the signature

`Keel-Signature: t=<unix seconds>,v1=<hex>` where

```
v1 = hex( HMAC-SHA256( secret, t + "." + raw_request_body ) )
```

Use the **raw body bytes**, before any JSON parsing. Reject the request if the
signature does not match or `t` is more than 5 minutes away from your clock.

Test vector: secret `secret`, `t = 1`, body `{}` →
`t=1,v1=1122767b193110cfec322b6f199b599edbf608ed087f2d27afb0b97d99523908`.

Node.js:

```js
import crypto from "node:crypto";

function verify(secret, header, rawBody) {
  const parts = Object.fromEntries(header.split(",").map((p) => p.split("=")));
  const t = Number(parts.t);
  if (!t || Math.abs(Date.now() / 1000 - t) > 300) return false;
  const want = crypto.createHmac("sha256", secret).update(`${t}.${rawBody}`).digest("hex");
  return parts.v1?.length === want.length &&
    crypto.timingSafeEqual(Buffer.from(parts.v1), Buffer.from(want));
}
```

Python:

```python
import hmac, hashlib, time

def verify(secret: str, header: str, raw_body: bytes) -> bool:
    parts = dict(p.split("=", 1) for p in header.split(","))
    t = int(parts.get("t", 0))
    if not t or abs(time.time() - t) > 300:
        return False
    want = hmac.new(secret.encode(), f"{t}.".encode() + raw_body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(parts.get("v1", ""), want)
```

When the owner replaces the secret, deliveries still in the queue are signed
with the new one.
