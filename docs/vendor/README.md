# Provayder hujjatlarining nusxasi

Bu yerdagi fayllar — provayderlarning ochiq hujjatlaridan **o'qib olingan**
nusxalar, manba havolasi va o'qilgan sanasi bilan. Ular saqlanadi, chunki
uchala sayt ham JS bilan chiziladigan SPA: `curl` ularda hujjat matnini
qaytarmaydi, ya'ni "havolaga qara" degan izoh keyingi sessiyada ishlamaydi.

⚠️ Bu yerga faqat **o'qilgan** narsa yoziladi. Loyihaning fiskal
provayderlaridagi bilan bir qoida (`docs/DECISIONS.md` → "Fiskal provayderlar"):
endpointni taxmin qilish — kompilyatsiya bo'ladigan, review'dan o'tadigan va
ishongan restoranda **bironta ham to'lov o'tkazmaydigan** kod.

| Fayl | Nima | Holati |
|---|---|---|
| `click-pass.md` | CLICK Pass — kassir mehmonning QR'ini skanerlaydi | adapter yozildi |
| `uzum-fastpay.md` | Uzum FastPay v2 — xuddi shu, fiskal havola bilan | adapter yozildi |
| `didox.md` | Didox — elektron hujjat aylanishi (ЭСФ), hamkor API | adapter yozildi (imzosiz) |
| `1c-exchange.md` | 1C «Обмен с сайтом» (CommerceML 2) protokoli | almashinuv yozildi |
| `uzum-tezkor-retail.md` | Uzum Tezkor Retail API — ular bizning serverni so'raydi, buyurtmani bizga POST qiladi | token, buyurtma va holat yozildi (`handlers/uzumtezkor.go`); katalog va qoldiq — hali |

**Payme GO** — ochiq API topilmadi (2026-08-30 da tekshirildi:
`developer.help.paycom.uz` faqat Merchant API va Subscribe API'ni, ya'ni
e-commerce tomonini hujjatlaydi; "kassa для приёма оплаты на месте" esa
"ulangandan keyin darhol ishlaydi" deb tasvirlanadi — ya'ni integratsiya
nuqtasi yo'q, Payme Business ilovasining o'z skaneri). Shuning uchun u
ro'yxatda `Ready: false`.

**Bank terminali (summani terminalga yuborish)** — O'zbekistonda ochiq
hujjatlangan ECR protokoli topilmadi (2026-08-30: `humocard.uz` Smart PIN Pad
— "info@nmpc.uz ga yozing"; `rhmt.uz` Rahmat POS — "hamkorlik bo'limiga
murojaat qiling"; `uzkassa.uz` — xuddi shunday). Hammasi shartnoma orqali
beriladi. Shuning uchun terminal drayveri **interfeys sifatida** bor va
ro'yxatdagi provayderlar `Ready: false`.
