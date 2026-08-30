# CLICK Pass — do'kon ichida to'lov (kassir mehmonning QR'ini skanerlaydi)

Manba: https://docs.click.uz/merchant-api/click-pass (o'qildi 2026-08-30).
⚠️ Sayt Docusaurus SPA — `curl` index sahifani qaytaradi, hujjat brauzerda
ochilgan. Shuning uchun nusxasi shu yerda.

## Kalitlar (Click kabinetidan)
`merchant_id`, `service_id`, `merchant_user_id`, `secret_key`.

## Autentifikatsiya
Har so'rovda HTTP sarlavha:

    Auth: merchant_user_id:digest:timestamp

- `digest` = `sha1(timestamp + secret_key)` (hex)
- `timestamp` = UNIX timestamp, **sekundlarda** (10 raqam)

Majburiy sarlavhalar: `Accept`, `Auth`, `Content-Type`
(`application/json` yoki `application/xml`).

## To'lov statuslari
| Kod | Ma'no |
|---|---|
| `< 0` | xato (tafsiloti `error_note` da) |
| `0` | to'lov yaratildi |
| `1` | ishlanmoqda |
| `2` | muvaffaqiyatli to'landi |

## To'lov
    POST https://api.click.uz/v2/merchant/click_pass/payment

```json
{
  "service_id": 12345,
  "otp_data": "1234567415821",
  "amount": 500,
  "cashbox_code": "KASSA-1",
  "transaction_id": "12345"
}
```

| Parametr | Tip | Izoh |
|---|---|---|
| `service_id` | integer | servis ID |
| `otp_data` | string | QR kodining ichidagi ma'lumot |
| `amount` | float | to'lov summasi |
| `cashbox_code` | string | kassa identifikatori (ixtiyoriy) |
| `transaction_id` | string | provayder tomonidagi tranzaksiya ID (ixtiyoriy) |

Javob:

```json
{
  "error_code": 0, "error_note": "Success",
  "payment_id": 1234567, "payment_status": 1, "confirm_mode": 1,
  "card_type": "private", "processing_type": "UZCARD",
  "card_number": "860002******8331", "phone_number": "998221234567"
}
```

`card_type`: `private` | `corporate`. `processing_type`: `UZCARD` | `HUMO` | `WALLET`.

## Status
    GET https://api.click.uz/v2/merchant/payment/status/:service_id/:payment_id

## Bekor qilish (reversal)
    DELETE https://api.click.uz/v2/merchant/payment/reversal/:service_id/:payment_id

Shartlari:
- to'lov muvaffaqiyatli yakunlangan bo'lishi kerak;
- faqat **joriy oy** to'lovlari; o'tgan oyniki — faqat joriy oyning birinchi kuni;
- onlayn karta bilan to'langan bo'lishi kerak;
- UZCARD tomonidan rad etilishi mumkin.

## Tasdiqlash rejimi (confirm mode)
Servis (`service_id`) darajasida yoqiladi. Yoqilgan bo'lsa **har** Click Pass
to'lovi qo'lda tasdiqlanishi shart, aks holda **30 soniyadan keyin avtomatik
bekor qilinadi**.

    POST   https://api.click.uz/v2/merchant/click_pass/confirm
           {"service_id": 12345, "payment_id": 1234567}
    PUT    https://api.click.uz/v2/merchant/click_pass/confirmation/:service_id   (yoqish)
    DELETE https://api.click.uz/v2/merchant/click_pass/confirmation/:service_id   (o'chirish)

## Xatolar
HTTP kodlari bilan: 200/201 OK, 400, 401, 403, 404, 406, 410, 500, 502.
