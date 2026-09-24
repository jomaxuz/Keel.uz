// Code samples for keel.uz/developers — one copy, shown in all three languages.
//
// ⚠️ **The contract lives in `docs/open-api.md`, and these follow it.** Field
// names, headers and error codes here are the published API; a rename on the
// server that is not made here too is a page promising something the server no
// longer does. The signature samples are checked against the test vector in
// `backend/internal/webhook/webhook_test.go`.

export const BASE = "https://<restoran-domeni>/api/open/v1";

export const PING = `curl ${BASE}/ping \\
  -H "Authorization: Bearer keel_7fd73b99c1cd8ed9…"`;

export const PING_RESPONSE = `{
  "restaurant": "Rayhon",
  "apiVersion": "v1",
  "key": { "name": "Finze", "prefix": "keel_7fd73b", "scopes": ["finance:read"] }
}`;

export const ERROR = `{
  "error": {
    "code": "insufficient_scope",
    "message": "this API key does not have the orders:read scope"
  }
}`;

export const BRANCHES = `GET /branches

{
  "branches": [
    { "id": "6ab5177…", "name": "Chilonzor", "phones": ["+998 90 000 00 00"],
      "address": "Toshkent, Chilonzor 9", "lat": 41.28, "lng": 69.20 }
  ]
}`;

export const MENU = `GET /branches/{branchId}/menu

{
  "branchId": "6ab5177…",
  "categories": [
    { "id": "…", "name": "Milliy taomlar", "nameRu": "Национальные блюда",
      "nameEn": "Uzbek classics", "sortOrder": 0 }
  ],
  "items": [
    { "id": "…", "categoryId": "…", "name": "Osh (palov)", "nameRu": "…", "nameEn": "…",
      "description": "…", "price": 45000, "imageUrl": "https://…/uploads/osh.jpg",
      "available": true, "tags": [], "barcode": "" }
  ]
}`;

export const ORDERS = `GET /orders?createdFrom=2026-09-01T00:00:00%2B05:00&limit=50

{ "orders": [ Order, … ], "nextCursor": "MTc1ODcw…" }`;

export const ORDER = `{
  "id": "6ab517d7c71997fe78741fed",
  "number": "QVUU-R97U",
  "branchId": "6ab51777ac108b6304170794",
  "type": "pickup",
  "channel": "web",
  "status": "confirmed",
  "customer": { "name": "Aziz", "phone": "+998901112233" },
  "items": [
    { "menuItemId": "…", "name": "Osh (palov)", "price": 45000, "qty": 2,
      "options": [ { "group": "Hajmi", "choice": "Katta", "priceDelta": 10000 } ] }
  ],
  "subtotal": 90000, "discountTotal": 0, "deliveryFee": 0,
  "serviceCharge": 0, "total": 90000,
  "paymentMethod": "cash", "paymentStatus": "unpaid",
  "statusHistory": [
    { "status": "pending", "at": "2026-09-24T12:30:15.69Z" },
    { "status": "confirmed", "at": "2026-09-24T12:30:15.727Z" }
  ],
  "createdAt": "2026-09-24T12:30:15.69Z"
}`;

export const MONEY = `GET /money?from=2026-09-01&to=2026-09-30

{
  "from": "2026-08-31T19:00:00Z", "to": "2026-09-30T19:00:00Z", "currency": "UZS",
  "entries": [ MoneyEntry, … ],
  "totals": {
    "byClass": {
      "revenue": { "in": 90000, "out": 0, "count": 1 },
      "cost":    { "in": 0, "out": 5000000, "count": 1 }
    },
    "pnlNet": -4910000
  }
}`;

export const MONEY_ENTRY = `{
  "id": "expense_6ab5235b666e66387034a534",
  "source": "expense",
  "class": "cost",
  "pnl": true,
  "direction": "out",
  "amount": 5000000,
  "occurredAt": "2026-09-23T19:00:00Z",
  "day": "2026-09-24",
  "branchId": "6ab52356666e66387034a523",
  "method": "transfer",
  "category": "ijara",
  "ref": { "type": "expense", "id": "6ab5235b666e66387034a534" }
}`;

export const DAILY = `GET /money/daily?from=2026-07-01&to=2026-09-30

{
  "currency": "UZS",
  "days": [
    { "day": "2026-09-24", "branchId": "6ab5…",
      "byClass": {
        "revenue": { "in": 4200000, "out": 0, "count": 61 },
        "cost":    { "in": 0, "out": 5000000, "count": 1 }
      },
      "revenueByMethod": { "cash": 2600000, "payme": 1600000 },
      "pnlNet": -800000 }
  ]
}`;

export const BALANCES = `GET /balances

{
  "asOf": "2026-09-24T13:29:00Z", "currency": "UZS", "branchId": "",
  "cash": [
    { "kind": "safe",   "name": "Seyf", "amount": 1200000, "counted": false },
    { "kind": "drawer", "name": "Kassa yashigi", "amount": 350000, "counted": false }
  ],
  "cashTotal": 1550000,
  "bank": [
    { "kind": "bank", "name": "Kapitalbank", "amount": 48000000,
      "counted": true, "at": "2026-09-23T09:00:00Z" }
  ],
  "bankTotal": 48000000,
  "inTransit": [ { "kind": "rail", "name": "Uzum", "amount": 3100000, "counted": false } ],
  "inTransitTotal": 3100000,
  "cashLimit": 5000000, "overCashLimit": false,
  "payables":    { "suppliers": { "amount": 7400000, "count": 5 } },
  "receivables": { "guestDebt": { "amount": 260000, "count": 3 } }
}`;

export const FISCAL = `GET /fiscal?from=2026-09-01&to=2026-09-30

{
  "receipts": [
    { "orderId": "…", "orderNumber": "QVUU-R97U", "branchId": "…",
      "kind": "sale", "total": 90000, "method": "cash",
      "status": "filed", "provider": "multikassa",
      "fiscalSign": "…", "receiptId": "…", "qrText": "…",
      "filedAt": "…", "soldAt": "…" }
  ],
  "zReports": [
    { "shiftId": "…", "branchId": "…", "number": "142",
      "saleCash": 2600000, "saleCard": 1600000, "saleTotal": 4200000,
      "saleCount": 61, "refundTotal": 0, "closedAt": "…" }
  ]
}`;

export const WEBHOOK_REQUEST = `POST https://your-server.example/keel
Content-Type: application/json
User-Agent: Keel-Webhooks/1
Keel-Event: order.status_changed
Keel-Event-Id: evt_6ab517d7c71997fe78741fed_1
Keel-Delivery-Id: 6ab517d8c71997fe78741ff1
Keel-Signature: t=1758717015,v1=5f1c…

{
  "id": "evt_6ab517d7c71997fe78741fed_1",
  "type": "order.status_changed",
  "createdAt": "2026-09-24T12:30:15.727Z",
  "data": { "status": "confirmed", "previousStatus": "pending", "order": Order }
}`;

export const MONEY_EVENT = `{
  "id": "evt_money_2026-09-24_6ab5…_1f3a9c0e2b7d",
  "type": "money.day_changed",
  "createdAt": "2026-09-24T13:40:00Z",
  "data": { "day": "2026-09-24", "branchId": "6ab5…", "pnlNet": -800000, "entries": 62 }
}`;

export const SIGNATURE_FORMULA = `Keel-Signature: t=<unix seconds>,v1=<hex>

v1 = hex( HMAC-SHA256( secret, t + "." + raw_request_body ) )`;

export const TEST_VECTOR = `secret = "secret"
t      = 1
body   = {}
→ t=1,v1=1122767b193110cfec322b6f199b599edbf608ed087f2d27afb0b97d99523908`;

export const VERIFY_NODE = `import crypto from "node:crypto";

function verify(secret, header, rawBody) {
  const parts = Object.fromEntries(header.split(",").map((p) => p.split("=")));
  const t = Number(parts.t);
  if (!t || Math.abs(Date.now() / 1000 - t) > 300) return false;
  const want = crypto.createHmac("sha256", secret).update(\`\${t}.\${rawBody}\`).digest("hex");
  return parts.v1?.length === want.length &&
    crypto.timingSafeEqual(Buffer.from(parts.v1), Buffer.from(want));
}`;

export const VERIFY_PYTHON = `import hmac, hashlib, time

def verify(secret: str, header: str, raw_body: bytes) -> bool:
    parts = dict(p.split("=", 1) for p in header.split(","))
    t = int(parts.get("t", 0))
    if not t or abs(time.time() - t) > 300:
        return False
    want = hmac.new(secret.encode(), f"{t}.".encode() + raw_body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(parts.get("v1", ""), want)`;

export const VERIFY_PHP = `function verify(string $secret, string $header, string $rawBody): bool {
    parse_str(str_replace(',', '&', $header), $p);
    $t = (int)($p['t'] ?? 0);
    if (!$t || abs(time() - $t) > 300) return false;
    $want = hash_hmac('sha256', $t . '.' . $rawBody, $secret);
    return hash_equals($want, $p['v1'] ?? '');
}`;
