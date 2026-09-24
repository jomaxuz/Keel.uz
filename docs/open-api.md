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

## Webhooks

The owner registers an `https://` address in the panel and picks events. They
receive a **signing secret** (`whsec_…`, shown once) to give to you.

### Events

| `type` | When |
|---|---|
| `order.created` | an order is placed — site, app, phone operator, till, Uzum Tezkor |
| `order.status_changed` | its status changes |
| `ping` | the owner pressed "Test" in the panel |

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
