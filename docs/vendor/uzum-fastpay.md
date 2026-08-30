# Uzum FastPay (v2) — do'kon ichida to'lov (kassir mehmonning QR'ini skanerlaydi)

Manba: https://developer.uzumbank.uz/en/fastpay/ (o'qildi 2026-08-30).
⚠️ Docusaurus + Redocly SPA — brauzerda o'qildi, nusxasi shu yerda.

Xizmat: QR orqali bir zumda to'lov va kassa tizimlari (POS / fiskal kassa)
uchun to'lovlar. Imkoniyatlari: to'lov + fiskalizatsiya, bekor qilish,
status tekshirish.

## Kalitlar (Uzum menejeridan)
- `merchant_id` — merchant identifikatori
- `secret_key` — API autentifikatsiyasi uchun kalit
- `service_id` — filial / savdo nuqtasi identifikatori
- `merchant_service_user_id` — **kassa yoki kassir** identifikatori filial ichida

## Autentifikatsiya
    Authorization: merchant_service_user_id:hash:timestamp
    Content-Type: application/json

- `hash` = `sha1(timestamp + secret_key)` (hex, 40 belgi)
- `timestamp` = UNIX timestamp **millisekundlarda**, **UTC+5** mintaqasi bo'yicha

⚠️ Sarlavha `^\d*:(\d{40}):\d*$` regex'iga mos bo'lishi shart (401 aks holda),
va **Authorization bilan so'rov ishlanishi orasidagi farq 50 soniyadan
oshmasligi kerak** (403).

Host: `https://mobile.apelsin.uz`

## Metodlar

### To'lov
    POST /api/apelsin-pay/merchant/v2/payment

```json
{
  "amount": 100000,
  "cashbox_code": "CashCode#44",
  "otp_data": "6385735999467329369938571759997073400776",
  "order_id": "234",
  "transaction_id": "a98dcb7e-5b6e-431d-b780-5fd7d93d7e3d",
  "service_id": 1
}
```
- `amount` — **tiyinda**
- `otp_data` — skanerlangan QR ichidagi ma'lumot, **kamida 40 belgi**
- `order_id` — sotuv raqami, **noyobligi biznikida**; noyob bo'lmasa to'lov o'tmaydi
- `transaction_id` — UUID

Javob:
```json
{
  "payment_id": "a98dcb7e-…", "payment_status": "SUCCESS", "error_code": 0,
  "operation_time": "2023-05-18 18:30:26.318",
  "client_phone_number": "998999999999", "card_type": 2,
  "processing_reference_number": "524818898472"
}
```

### Fiskalizatsiya (fiskal chek havolasini uzatish)
    POST /api/apelsin-pay/merchant/payment/fiscal
    {"payment_id": "…", "service_id": 1, "fiscal_url": "http://…"}

⚠️ Havolada maxsus belgilar bo'lishi mumkin — kerak bo'lsa unescape qilinadi.

### Bekor qilish (qisman bekor qilish YO'Q)
    PUT /api/apelsin-pay/merchant/v2/payment/reversal/{orderId}
    {"service_id": 1, "payment_id": "…"}

### Status
    POST /api/apelsin-pay/merchant/payment/status
    {"payment_id": "…", "service_id": 1}

## Xatolar
⚠️ **Hamma metod HTTP 200 qaytaradi**, muvaffaqiyat `error_code == 0` bilan
belgilanadi. Xato bo'lsa `error_code` (int) va `error_message` (string).

| Kod | Sabab |
|---|---|
| 0 | muvaffaqiyat |
| 400 | `otp_data` 40 belgidan qisqa; `amount` 0/null/manfiy; servis bloklangan; karta faol emas; mablag' yetarli emas; `payment_id` noto'g'ri yoki bo'sh |
| 401 | `Authorization` formati noto'g'ri; `merchant_id`/`service_id`/`merchant_service_user_id` faol emas; hash mos kelmadi; `service_id` bu hamkorga tegishli emas |
| 403 | Authorization bilan ishlanish orasi 50 soniyadan oshdi |
| 404 | `payment_id` bo'yicha operatsiya topilmadi |
| 416 | Mijoz kartasi Safe Mode'da (yangi foydalanuvchi ilovada 3 ta to'lov qilishi kerak) |
| 503 | Servis topilmadi yoki faol emas |

Muhim `error_message` qiymatlari: `apelsin.pay.wrong.prefix.otp.data` (QR 43
belgi bo'lishi kerak), `apelsin.pay.user.otp.data.expired` (QR muddati o'tgan —
mijoz qurilmasidagi vaqt noto'g'ri bo'lishi mumkin), `qr.duplicated`,
`order.id.duplicated`, `transaction.duplicated`, `apelsin.pay.reverse.not.allowed`,
`operation.is.inProcess`, `unsupported.fiscal.url`.
