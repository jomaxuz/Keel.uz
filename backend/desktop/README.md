# Keel Kassa — Windows ilovasi (Wails)

Kassa va zal ekranlari bitta oynada, chek printeriga yeta oladigan yagona
mashinada. Qaror va sabablari: `docs/pos-reja.md` §2.

## Nega `backend/` moduli ichida

Go'ning `internal/` qoidasi: `restaurant-backend/internal/printer` va
`internal/receipt` ni **faqat shu modul ichidagi kod** import qila oladi, va
`replace` direktivasi buni aylanib o'tolmaydi. Alohida modul aynan shu ikki
paketga — butun ishning maqsadiga — yeta olmasdi.

## Nima qayerda quriladi

| | Qayerda | Nega |
|---|---|---|
| `.exe` | **Windows VM** (`wails build`) | Wails cross-compile'ni qo'llab-quvvatlamaydi |
| Go tomonini tekshirish | Linux ham bo'ladi | `GOOS=windows CGO_ENABLED=0 go build ./desktop` |

Ikkinchi qator kutilmagan, lekin haqiqiy: Wails v2 ning Windows tomoni sof Go
(`go-webview2`), ya'ni **cgo yo'q** — `internal/printer` dagi `winspool.drv`
lazy DLL tanlovi bilan bir sabab. Ya'ni Go tomonidagi xato VM'ni ochmasdan
topiladi; VM faqat frontend build'i, ikonka, manifest va haqiqiy ishga tushirish
uchun kerak.

## Build (VM'da)

```
wails doctor          # Go, npm, WebView2 — uchalasi
wails dev             # ishlab chiqish, jonli qayta yuklash bilan
wails build           # build/bin/keel-till.exe
```

## Umumiy kod

Ekranlar **ko'chirilmaydi** — `frontend/src/app/kassa` va
`frontend/src/components/till` shu yerda qayta kompilyatsiya qilinadi
(`vite.config.ts` dagi `@` aliasi). Nusxa bir oyda ikkita kassa bo'lardi, va
xato haqidagi shikoyat doim ikkinchisiga tushardi.

Narxi — ikkita shim (`src/shims/`): umumiy kod `next/navigation` va
`next/headers` ni import qiladi. Butun uchinchi tomon yuzasi — `react`,
`react-dom`, `react-icons`.

## ⚠️ Chek baytlari bu yerda kodlanmaydi

Server `escpos.Options` ni filialning printer yozuvidan yig'adi (charset, cut,
drawer, feed) va tayyor baytlarni `print_job` navbatiga qo'yadi. Ilova ularni
**qayta kodlamaydi** — `cmd/fiscalagent` bosgan yo'ldan boradi: navbatdan ish
oladi, printerga yozadi, natijani qaytaradi. Ikkinchi kodlagich ikki xil chek
degani, va farqni qo'lida qog'oz ushlab turgan mehmon topadi.
