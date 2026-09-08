# Keel Guest — restoranning o'z Android ilovasi

Mehmon o'rnatadigan ilova. **Bitta kod bazasi, har restoranga alohida build**:
farq faqat `app/brand.properties` va ikkita rasm faylida.

Loyihaning umumiy qoidalari — ildizdagi `CLAUDE.md`. Dizayn `../android-design`
dan **yo'l bo'yicha** ulanadi (`settings.gradle.kts`) va oltita ilova uni
bo'lishadi.

## Nima uchun bosh sahifa yo'q

Restoran ilovasini ochgan odam och. Muqova rasmi, ish vaqti va «Buyurtma berish»
tugmasi — bu odam bilan menyu orasidagi bitta bosish va bitta scroll, va
ularning har biri buyurtma yo'qoladigan joy. Bosh sahifa aytadigan narsa (manzil,
telefon, ish vaqti) odam **izlab boradigan** joyda turadi, menyu oldida emas.

## Brendlash: bitta fayl, ikkita rasm

Quvur (control) faqat shularni yozadi:

```
app/brand.properties                      # id, nom, server, rang, versiya
app/src/main/res/drawable/brand_logo.png  # splash va sarlavha logosi
app/src/main/res/mipmap-*/ic_launcher*    # ikonka (oq fon + logo)
```

Qolgan hamma narsa har restoranda bir xil. ⚠️ **Gradle'ni, manifestni yoki
Kotlin faylini tahrirlaydigan quvur — nosozligi o'qib bo'lmaydigan quvur.**

Qiymatlar tenantning **o'z** `GET /restaurant` javobidan olinadi (`name`,
`logoUrl`, `theme.brand`) — konsolda qayta yozilmaydi: ikkinchi nusxa mehmon
saytda ko'rgan narsa bilan bir kun kelib ziddiyatga tushadi.

### Lokal ishga tushirish

`brand.properties` reponing o'zida Keel demo qiymatlari bilan turadi, ya'ni
klonlagan odam ilovani darhol ishga tushira oladi:

```bash
cp ../team-android/local.properties .   # sdk.dir
./gradlew :app:assembleDebug
```

⚠️ `debug` build `applicationId` ga `.debug` qo'shadi — shuning uchun ishlab
chiquvchining nusxasi haqiqiy restoran ilovasi bilan **yonma-yon** o'rnatiladi
va mehmonning seansini o'chirmaydi.

## Imzo kalitlari

`KEEL_GUEST_KEYSTORE_PROPERTIES` muhit o'zgaruvchisi keystore ma'lumotlari bor
`.properties` faylga ishora qiladi. Fayl bo'lmasa **release imzolanmaydi** va
o'rnatib bo'lmaydi — bu ataylab: debug kalit bilan jimgina imzolangan release
aynan odam do'konga yuklaydigan build bo'ladi.

⚠️ **Har restoranga bitta keystore, va u hech qachon almashtirilmaydi**: yo'qolgan
kalit bilan imzolangan ilovani boshqa hech qachon yangilab bo'lmaydi. Kalitni
quvur birinchi build'da yaratadi va **qayta yaratmaydi**.

## Uchta raqam

- **Narx** — butun so'm, hech qachon tiyin emas. Yuzga bo'lish plovni 450 so'm
  qilib ko'rsatadi.
- **`oldPrice`** — `null` bo'lishi mumkin va nol emas: nol deb o'qilsa menyudagi
  har bir oddiy taom yonida ustidan chizilgan «0 so'm» chiqadi.
- **`isAvailable`** — serverdan **ikkala** sabab bilan javob bo'lib keladi (ega
  o'chirgan yoki stop list). Telefon uni ikkinchi marta hisoblamaydi.
