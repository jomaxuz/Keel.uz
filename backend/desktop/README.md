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

## Sozlash: qo'lda emas, ekrandan

`.exe` birinchi ishga tushganda **ulanish ekrani** chiqadi. Ikki qadam:

1. **Restoran manzili** + ega/menejer logini. Faqat nom yozilsa yetarli —
   `osh` → `osh.keel.uz` (har tenant provisioning'da shu subdomenni oladi,
   `control/handlers/tenants.go`). O'z domeni bo'lsa to'liq yoziladi.
2. **Filial tanlanadi** → ilova qurilma kalitini oladi va sozlamani **o'zi
   yozadi**.

Shundan keyin ekran boshqa chiqmaydi; kassir faqat PIN teradi.

⚠️ **Parol saqlanmaydi.** Login faqat "bu odam shu mashinani bog'lashga
haqli" ekanini isbotlaydi. Saqlanadigan narsa — **filial qurilma kaliti**
(`role: "tilldevice"`, bir yil), va u paneldan bir bosishda bekor qilinadi
(`branch.TillVersion`). Monoblokda panel logini turishi — devorga yozib
qo'yilgan umumiy parolning yo'li.

⚠️ **Filial tanlash alohida qadam**, chunki noto'g'ri filial cheklarni boshqa
oshxonaga, sotuvni boshqa hisobotga yuboradi — va ekranda hech nima xato
ko'rinmaydi.

⚠️ **Nega agent kaliti emas.** `AdminFiscalAgentToken` **har chaqiruvda
almashtiradi**, ya'ni ilova uni so'rasa o'sha filialda ishlab turgan agentni
jimgina o'ldirardi. Bitta mashinaga ikki sir, va ikkinchisini olish birinchisini
buzadi — avtomatik ulanishni imkonsiz qilgan narsa shu edi. Endi agent
endpointlari qurilma kalitini ham qabul qiladi.

### Sozlama qayerda turadi

`%PROGRAMDATA%\Keel\till.json` — ilova yozadi, odam emas.

⚠️ `.exe` yonida **emas**: `Program Files` ichiga yozish uchun admin huquqi
kerak, ya'ni saqlash aynan shu ilova mo'ljallangan mashinalarda yiqilardi.
Eski, qo'lda yozilgan fayl hali ham **o'qiladi** (allaqachon sotayotgan kassa
buzilmasligi uchun), lekin unga qaytib yozilmaydi.

Log: `.exe` yonida `till.log`. ⚠️ GUI dasturda konsol yo'q.

## Oyna

Framesiz va **maksimallashtirilgan** (`options.Maximised`), haqiqiy fullscreen
emas. `pos-reja.md` §2 ikkalasini talab qiladi — chromesiz to'liq ekran **va**
ilova qotganda chiqish yo'li — lekin haqiqiy fullscreen Alt+Tab va Win+D ni ham
olib qo'yadi, ya'ni yagona chiqish yo'li tok tugmasi bo'lib qoladi. Kassirga
ikkalasi bir xil ko'rinadi.

⚠️ **Sarlavha paneli yo'q, va bo'lmasligi kerak.** Kassa ekranining **o'z
paneli bor** (qulf tugmasi bilan), ya'ni ustiga qo'yilgan ikkinchi panel ham
takror, ham zararli: uning 36 px i sahifani oynadan baland qilib o'ng tomonda
scrollbar chiqaradi. Kassa ildizi allaqachon `h-dvh` va o'z ichida suriladi —
uning atrofiga o'ralgan har qanday narsa ortiqcha.

Chiqish yo'llari: **Alt+F4** va **Ctrl+Shift+Q**. Ular bir xil emas —
birinchisini Windows bajaradi va ilova javob bermay qolganda ham ishlaydi,
ikkinchisi esa JavaScript, ya'ni qotgan webview'ni qutqara olmaydi (u
boshqaruvini yo'qotgan ekran uchun). ⚠️ **Ikkalasi ham o'rnatish hujjatida
yozilishi shart**: ko'rinadigan yopish tugmasi yo'q monoblok tok tugmasi bilan
yopiladi.

## Chek chiqarishning ikki yo'li

| Holat | Kim chiqaradi |
|---|---|
| Filialda printer **sozlangan** | Server `escpos.Options` ni yig'adi, baytlarni navbatga qo'yadi, **agent halqasi** chop etadi |
| Sozlanmagan, lekin mashinada printer bor | `PrintLines` binding'i — shu mashinaning printeri, o'z sozlamasi bilan |

Ikkinchisi `lib/print.ts` dagi brauzer dialogini almashtiradi. Ikkalasi ham
**bir xil `escpos.Encode`** ni chaqiradi va **layout'ni qayta hisoblamaydi** —
qatorlar serverdagi `receipt.Render` dan keladi, ya'ni ega tasdiqlagan
ko'rinishdan.
