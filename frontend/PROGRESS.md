
---

## 2026-09-07 (9) — Demo tenantda xarid ro'yxati nega bo'sh edi (uchta xato)

`b5somsa.keel.uz` da xarid ro'yxati bo'sh ekani tekshirildi va sabab
`cmd/demodata` da ekani aniqlandi. Uchala xato ham **jimgina** edi.

1. ⚠️ **Texkartalar 1000 barobar kichik yozilardi.** Karta `menu_item.recipe`
   da **retsept birligida** (gramm/ml/dona) saqlanadi va har bir o'quvchi
   `PerUnit` ga bo'ladi; generator esa **sotib olish birligida** (kilo) yozardi:
   320 g guruch **0.32 g** bo'lib tushardi. Natijada sarf nolga yaxlitlanardi →
   javon hech qachon kamaymasdi → **xarid ro'yxati doimo bo'sh**, tannarx esa
   foizning ulushi bo'lib ko'rinardi. `dona` qatorlari to'g'ri edi (PerUnit=1),
   shuning uchun bitta kartaga qarash xatoni ko'rsatmasdi. Endi
   `recipeUnits()` — bitta joyda, testi bilan.
2. ⚠️ **Sanoq qatorining `Value` i javon qiymatini yozardi**, farqniki emas
   (`models/stocktake.go` ta'rifiga zid) — ya'ni har qator **ortiqcha** bo'lib
   o'qilardi va kamomad ekrani hech nima ko'rsatmasdi. Endi farqning qiymati,
   ishorasi bilan, va sanoq jami — qatorlar yig'indisi. Farq ham realroq: ko'p
   qator to'g'ri, bir nechtasi kam, ba'zan ortiqcha.
3. ⚠️ **Har bir mahsulot to'la sotib olinardi** ("shelf lands at ten days") —
   ya'ni hech qachon hech nima kerak bo'lmaydigan oshxona. Endi oxirgi
   yetkazishda har beshinchi masalliq ataylab kam olinadi: manfiyga tushmaydi,
   lekin ro'yxatda qator paydo bo'ladi.

**Lokal to'liq tekshiruv** (`qa_b5`, keyin o'chirildi): seed menyu → `demodata`
→ `stock_movements_v1` markerini o'chirib qayta ishga tushirish (1174 buyurtma
uchun sarf harakati yozildi) → **xarid ro'yxatida 15 qator, 3 yetkazib beruvchi,
5.5 mln so'm**, har qatorda «nega shuncha» (15 kunlik gorizont, ritm 6.5 kun,
ba'zilarida «tugagan kunlar»); **kamomadda 3 qator**, biri `kartalar sarfi 0`
(ya'ni kamomad emas, kartasi yo'q), qamrov 89%.

⚠️ `yamato` va `krevetkauz` demo tenantlarida ham 1-xato bor — `demodata` qayta
ishga tushirilsa tuzaladi.
