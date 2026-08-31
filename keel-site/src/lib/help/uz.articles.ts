// The articles themselves, in Uzbek.
//
// Split from `uz.ts` because that file is the shape (interface strings and
// section headings) and this one is the manual. They change for different
// reasons and at completely different rates.

import type { Article } from "./types";

export const articlesUz: Article[] = [
  // ═══════════════════════════════════ Boshlash
  {
    slug: "first-day",
    section: "start",
    title: "Birinchi kun: nimadan boshlash",
    lead: "Restoran bir kunda ishga tushadi. Mana shu tartibda qilinsa, kechqurun birinchi buyurtmani qabul qilasiz.",
    keys: ["boshlash", "ishga tushirish", "sozlash", "birinchi"],
    body: [
      {
        p: "Keel ikki qismdan iborat: mehmon ko'radigan *sayt* va siz ko'radigan *panel*. Panelda nima qilsangiz, saytda darhol ko'rinadi — alohida «e'lon qilish» tugmasi yo'q.",
      },
      { h: "Tartib" },
      {
        steps: [
          "*Restoran profili* — `Sozlamalar` da nom, telefon, manzil va ish vaqti. Ish vaqti bo'sh bo'lsa sayt «yopiq» deb ko'rsatadi.",
          "*Kategoriyalar* — «Taomlar», «Ichimliklar» kabi. Avval kategoriya, keyin taom: taom kategoriyasiz saqlanmaydi.",
          "*Menyu* — taomlar, narxlar va rasmlar. Menyuni havoladan yoki eski kassangizdan import qilsa ham bo'ladi.",
          "*Yetkazib berish* — zona yoki masofa bo'yicha narx. Bu sozlanmasa mehmon manzil kirita olmaydi.",
          "*To'lov va SMS* — mijoz kirishi uchun SMS shart; onlayn to'lov ixtiyoriy (naqd har doim ishlaydi).",
          "*Xodimlar* — kassir, ofitsiant, oshpaz uchun hisob. Har biriga alohida login.",
        ],
      },
      {
        warn: "Ombor va texkartani *birinchi kunda* to'ldirishga urinmang. Ular ishlashi uchun kirimlar muntazam kiritilishi kerak, aks holda hisobotlar noto'g'ri raqam ko'rsatadi. Avval sotishni yo'lga qo'ying, omborni ikkinchi haftada boshlang.",
      },
      {
        fig: "dashboard",
        notes: {
          nav: "Chap tomondagi bo'limlar. Guruhlar ochiladi va yopiladi — kerakli ekranni shu yerdan topasiz.",
          period: "Davr tanlagich: bosh sahifadagi barcha raqamlar shu davrga tegishli.",
        },
      },
      { see: ["login", "branches", "menu-item", "delivery-zones"] },
    ],
  },
  {
    slug: "login",
    section: "start",
    title: "Panelga kirish va parol",
    lead: "Panel manzili, birinchi hisob, parolni unutganda nima qilish.",
    keys: ["login", "parol", "kirish", "admin panel", "unutdim"],
    body: [
      {
        p: "Panel saytingizning `/admin` manzilida ochiladi. Masalan sayt `restoran.uz` bo'lsa, panel `restoran.uz/admin`.",
      },
      {
        p: "Birinchi hisobni biz beramiz — login va parol shartnoma paytida yuboriladi. Kirgandan keyin parolni *albatta* almashtiring: `Sozlamalar` → `Hisob`.",
      },
      { h: "Parolni unutdingizmi" },
      {
        steps: [
          "Kirish sahifasidagi `Parolni unutdingizmi?` ni bosing.",
          "Hisobingizga bog'langan telefon raqamini kiriting — SMS kod keladi.",
          "Kod bilan yangi parol qo'yasiz.",
        ],
      },
      {
        warn: "SMS provayderi sozlanmagan bo'lsa bu yo'l ishlamaydi — kod ketmaydi. Bunday holatda bizga yozing, hisobni qo'lda tiklaymiz.",
      },
      {
        tip: "Panel logini *bitta odamniki*. Kassir yoki ofitsiantga panel logini bermang — ularning o'z ilovalari va o'z hisoblari bor. Panel logini zaldagi kompyuterda ochiq qolsa, u mijozlar bazasi, to'lov kalitlari va hisobotlar demakdir.",
      },
      { see: ["admins", "staff-add", "security"] },
    ],
  },
  {
    slug: "admins",
    section: "start",
    title: "Panel adminlari: ega va menejer",
    lead: "Kim panelga kiradi, kim nimani ko'radi va har bir amal qayerda yoziladi.",
    keys: ["admin", "menejer", "ega", "owner", "manager", "ruxsat"],
    body: [
      { p: "Panelda ikki daraja bor va farqi jiddiy." },
      {
        table: {
          head: ["Rol", "Nimani ko'radi"],
          rows: [
            [
              "*Ega* (owner)",
              "Hammasini: barcha filiallar, mijozlar bazasi, to'lov kalitlari, hisobotlar, kampaniyalar, adminlar ro'yxati.",
            ],
            [
              "*Menejer* (manager)",
              "Faqat biriktirilgan filialini: buyurtmalar, menyu, ombor, xodimlar. Mijozlarga ommaviy xabar yubora olmaydi.",
            ],
          ],
        },
      },
      {
        warn: "Menejer URL orqali boshqa filialga o'ta olmaydi — server har so'rovni uning filialiga qisqartiradi. Lekin *ega* hisobini bo'lishib ishlatish bu qo'riqni butunlay bekor qiladi: har kimga o'z hisobi.",
      },
      { h: "Amallar jurnali" },
      {
        p: "`Pul va jamoa` → `Jurnal` da kim nima qilgani yoziladi: narx o'zgargani, buyurtma bekor qilingani, inventarizatsiya tozalangani, *texkarta o'zgargani* — qaysi masalliq va qanchadan qanchaga.",
      },
      {
        tip: "Texkarta o'zgarishi jurnalda alohida yozilishining sababi bor: kartadagi normani oshirib qo'yish — ombordan pul olib ketishning ariqcha izsiz yagona yo'li. Oyiga bir marta shu qatorlarni ko'rib chiqing.",
      },
      { fig: "admins" },
      { see: ["staff-roles", "admin-log", "security"] },
    ],
  },
  {
    slug: "branches",
    section: "start",
    title: "Filial va brend",
    lead: "Bitta restoran, zanjir yoki bir binoda ikki brend — uchalasi ham qo'llab-quvvatlanadi.",
    keys: ["filial", "brend", "zanjir", "ikkinchi filial", "branch"],
    body: [
      {
        p: "*Brend* — menyu va sayt. *Filial* — eshigi bor bino: o'z manzili, ish vaqti, kassasi, ombori va xodimlari.",
      },
      {
        list: [
          "Bitta restoran — bitta brend, bitta filial. Hech qayerda tanlash so'ralmaydi.",
          "Zanjir — bitta brend, bir nechta filial. Menyu umumiy, narxlar va ombor har filialda alohida.",
          "Bir binoda ikki konsepsiya — ikki brend, ikki sayt, bitta kassa.",
        ],
      },
      {
        p: "Panelning yuqorisida filial tanlagich chiqadi (bir nechta filial bo'lsa). Buyurtmalar, kassa va ombor ekranlari shu tanlovga qarab ishlaydi.",
      },
      {
        warn: "Ombor ekranlari *bitta filialni talab qiladi* va tanlanmasa ochilmaydi. Uch muzlatgichga tarqalgan «kompaniyada 9 kg go'sht bor» degan raqamni na sanab, na buyurtma berib bo'ladi — shuning uchun ekran «filialni tanlang» deydi.",
      },
      {
        warn: "Yetkazib berish *har filialda alohida yoqiladi*. Ikki filialda o'chiq qolgani — eng jim xato: forma to'ldirilgandek ko'rinadi, hech nima xato bermaydi, va Yangiyo'lga berilgan buyurtma Chilonzorga ketadi.",
      },
      { see: ["delivery-setup", "working-hours", "stock-balances"] },
    ],
  },
  {
    slug: "working-hours",
    section: "start",
    title: "Ish vaqti va bayram kunlari",
    lead: "Sayt qachon buyurtma qabul qiladi, kassa qachon stop listni o'qiydi.",
    keys: ["ish vaqti", "yopiq", "ochiq", "grafik", "soat"],
    body: [
      {
        p: "`Sozlamalar` → `Ish vaqti` da har hafta kuni uchun ochilish va yopilish soati yoziladi. Sayt shundan «Hozir ochiq» yoki «Yopiq» deb ko'rsatadi.",
      },
      {
        p: "Yarim tundan keyin ishlaydigan restoran uchun yopilish soati kichikroq bo'lishi normal: `10:00 – 02:00` — ertasi kungacha degani.",
      },
      {
        warn: "Ish vaqti *bo'sh* bo'lsa sayt restoranni yopiq deb ko'rsatadi va buyurtma qabul qilmaydi. Birinchi kunda eng ko'p uchraydigan «sayt buyurtma olmayapti» shundan.",
      },
      {
        tip: "Kassa tizimingiz ulangan bo'lsa, stop list ish vaqtidan 15 daqiqa oldin o'qila boshlaydi va yopilgandan keyin to'xtaydi. Soat to'rtda hech kim qozon bo'shatmaydi — bu bekorga so'rov.",
      },
      { see: ["branches", "stop-list"] },
    ],
  },

  // ═══════════════════════════════════ Sayt
  {
    slug: "site-design",
    section: "site",
    title: "Sayt ko'rinishi: rang, logotip, bo'limlar",
    lead: "Saytning rangi va bosh sahifadagi bloklar paneldan o'zgartiriladi — dasturchi kerak emas.",
    keys: ["dizayn", "rang", "logotip", "sayt ko'rinishi", "konstruktor"],
    body: [
      {
        p: "`Sozlamalar` → `Sayt` da restoranning brend rangi, logotipi va bosh sahifadagi bloklar tartibi tanlanadi.",
      },
      {
        list: [
          "*Aksent rang* — tugmalar va belgilar shu rangda bo'ladi. Bitta rang tanlanadi, qolganini tizim o'zi hisoblaydi (matn o'qilishi uchun).",
          "*Logotip va muqova* — `Sozlamalar` dan yuklanadi. Muqova bosh sahifaning yuqorisida.",
          "*Bloklar* — «Mashhur taomlar», «Aksiyalar», «Biz haqimizda», «Xarita», «Fikrlar». Har birini yoqish/o'chirish va tartibini o'zgartirish mumkin.",
          "*Tema* — mehmon o'z brauzeridagi yorug'/qorong'i rejimini oladi. Buni majburlash shart emas.",
        ],
      },
      {
        tip: "Rangni tanlashda taom rasmlaringizga qarang. To'q ko'k fon sariq palovni chiroyli ko'rsatadi; qizil fon esa deyarli har qanday ovqat bilan urishadi.",
      },
      { fig: "site-home" },
      { see: ["site-texts", "menu-photos", "site-seo"] },
    ],
  },
  {
    slug: "site-texts",
    section: "site",
    title: "Sayt matnlari va «Biz haqimizda»",
    lead: "Bosh sahifadagi har bir jumla paneldan yoziladi, uch tilda.",
    keys: ["matn", "tarjima", "biz haqimizda", "til", "rus tili"],
    body: [
      {
        p: "`Sozlamalar` → `Matnlar` da sarlavha, tavsif, «Biz haqimizda» va aloqa bloki yoziladi.",
      },
      {
        p: "Sayt uch tilda: *o'zbek*, *rus*, *ingliz*. Har maydonning uchta katagi bor. Rus yoki ingliz katak bo'sh bo'lsa — o'zbekchasi ko'rsatiladi, ya'ni sayt hech qachon bo'sh joy bilan chiqmaydi.",
      },
      {
        warn: "Taom nomlarini ham tarjima qiling. Ruscha sahifada «Osh (palov)» degan nom mehmonga o'zbekcha sayt ochilgandek tuyuladi — va u kartani yopadi.",
      },
      {
        tip: "«Biz haqimizda» — SEO uchun eng foydali blok. Unda shahar va tuman nomi bo'lsa, «Chilonzorda somsa» degan qidiruvda topilish ehtimoli oshadi.",
      },
      { see: ["site-seo", "menu-item"] },
    ],
  },
  {
    slug: "site-seo",
    section: "site",
    title: "Domen, SEO va favicon",
    lead: "Saytni Google va Yandex topishi uchun nima kerak.",
    keys: ["domen", "seo", "google", "yandex", "favicon", "https"],
    body: [
      {
        p: "Har restoran o'z domenida ishlaydi. Domenni siz olasiz (masalan `b5somsa.uz`), biz uni ulaymiz va HTTPS sertifikatini o'zi yangilanadigan qilib qo'yamiz.",
      },
      { h: "Domenni ulash" },
      {
        steps: [
          "Domen provayderingiz panelida `A` yozuvini bizning IP manzilimizga qarating (IP ni biz beramiz).",
          "Konsolda domenni qo'shamiz.",
          "15 daqiqadan bir necha soatgacha kutiladi — DNS butun dunyo bo'ylab tarqalishi kerak.",
        ],
      },
      {
        warn: "«Domen ulandi, lekin sayt ochilmayapti» — deyarli har doim DNS hali tarqalmagan. Brauzeringizni tozalab qayta urinib ko'ring; bir soatdan keyin ham ochilmasa yozing.",
      },
      { h: "Qidiruvda topilish" },
      {
        list: [
          "Sarlavha va tavsifni to'ldiring — Google natijalarda aynan shularni ko'rsatadi.",
          "Menyuning har bir taomi alohida sahifa: rasm va tavsif bo'lsa, ular ham qidiruvga tushadi.",
          "Uch til uchun uch manzil bor (`/ru/`, `/en/`) va ular bir-biriga bog'langan — Google ularni takroriy sahifa deb hisoblamaydi.",
          "*Favicon* — brauzer tabidagi kichkina belgi. Logotipdan avtomatik yasaladi.",
        ],
      },
      { see: ["site-texts", "site-design"] },
    ],
  },
  {
    slug: "banners",
    section: "site",
    title: "Bannerlar",
    lead: "Bosh sahifaning yuqorisidagi aylanma rasmlar: aksiya, yangi taom, bayram.",
    keys: ["banner", "reklama", "slayder", "aksiya rasmi"],
    body: [
      {
        p: "`Menyu` → `Bannerlar` da rasm yuklanadi, sarlavha yoziladi va havola ko'rsatiladi (masalan aksiya sahifasiga yoki bitta taomga).",
      },
      {
        list: [
          "Tartibni o'zgartirish mumkin — birinchisi eng ko'p ko'riladi.",
          "Bannerni o'chirmasdan *yashirish* mumkin: bayram tugadi, keyingi yili yana yoqasiz.",
          "Telefonda va kompyuterda alohida rasm yuklash mumkin: gorizontal rasm telefonda qirqiladi.",
        ],
      },
      {
        tip: "Uchtadan ko'p banner qo'ymang. To'rtinchisini deyarli hech kim ko'rmaydi, lekin u ham yuklanadi — ya'ni sayt sekinlashadi.",
      },
      { see: ["promotions", "site-design"] },
    ],
  },
  {
    slug: "reviews",
    section: "site",
    title: "Mehmonlar fikri saytda",
    lead: "Baholarni saytda ko'rsatish yoki ko'rsatmaslik — bitta tugma.",
    keys: ["fikr", "baho", "sharh", "yulduz", "reyting"],
    body: [
      {
        p: "Buyurtma yetkazilgandan keyin mijozdan baho so'raladi. Fikrlar `Mijozlar` → `Fikrlar` da to'planadi.",
      },
      {
        p: "Ularni saytda ko'rsatish ixtiyoriy: `Sozlamalar` → `Sayt` → «Fikrlarni ko'rsatish». Yoqilsa, o'rtacha baho va nechta bahodan ekani chiqadi.",
      },
      {
        warn: "«Faqat 4 yulduzdan yuqorisini ko'rsatish» degan sozlama *yo'q va bo'lmaydi*. Tanlab ko'rsatilgan baho — baho emas, reklama; mehmon buni bir marta sezsa, qolgan hamma raqamga ishonmay qo'yadi.",
      },
      { see: ["feedback"] },
    ],
  },
  {
    slug: "vacancies",
    section: "site",
    title: "Vakansiya sahifasi",
    lead: "Ishga odam olish uchun sahifa: e'lon, ariza va arizalar ro'yxati.",
    keys: ["vakansiya", "ish", "xodim izlash", "ariza"],
    body: [
      {
        p: "`Menyu` → `Vakansiyalar` da lavozim, maosh va talablar yoziladi. Sayt `/vakansiya` sahifasida e'lonlar chiqadi.",
      },
      {
        p: "Ariza qoldirgan odamning ismi, telefoni va izohi shu yerda ko'rinadi. Ariza kelganda panelda bildirishnoma chiqadi.",
      },
      {
        tip: "E'lonni yopmasdan «faol emas» qilib qo'ying — kelasi oy yana ochasiz va matnni qayta yozmaysiz.",
      },
      { fig: "vacancies" },
      { see: ["staff-add"] },
    ],
  },

  // ═══════════════════════════════════ Menyu
  {
    slug: "categories",
    section: "menu",
    title: "Kategoriyalar",
    lead: "Menyuning bo'limlari. Taom kategoriyasiz saqlanmaydi, shuning uchun bu birinchi qadam.",
    keys: ["kategoriya", "bo'lim", "guruh"],
    body: [
      {
        p: "`Menyu` → `Kategoriyalar` da bo'lim qo'shiladi: «Milliy taomlar», «Burgerlar», «Ichimliklar». Tartibni sudrab o'zgartirasiz — sayt va kassa aynan shu tartibda ko'rsatadi.",
      },
      {
        list: [
          "Har kategoriyaning uch tilda nomi bor.",
          "Kategoriyani *yashirish* mumkin: taomlar joyida qoladi, saytda ko'rinmaydi. Mavsumiy bo'lim uchun qulay.",
          "Rasm ixtiyoriy — bo'lmasa faqat nom chiqadi.",
        ],
      },
      {
        warn: "Ichida taomi bor kategoriyani o'chirsangiz, taomlar kategoriyasiz qoladi va saytda ko'rinmaydi. Avval taomlarni boshqa kategoriyaga ko'chiring.",
      },
      { fig: "categories" },
      { see: ["menu-item"] },
    ],
  },
  {
    slug: "menu-item",
    section: "menu",
    title: "Taom qo'shish va tahrirlash",
    lead: "Nom, narx, rasm, tavsif — va taom saytda, kassada, botda darhol paydo bo'ladi.",
    keys: ["taom", "mahsulot", "narx", "menyu qo'shish"],
    body: [
      {
        p: "`Menyu` da `+ Yangi taom` tugmasi. Majburiy maydonlar — nom, kategoriya va narx; qolganini keyin to'ldirsa ham bo'ladi.",
      },
      {
        fig: "menu",
        notes: {
          add: "Yangi taom shu yerdan qo'shiladi.",
          uncosted: "Tannarxi kiritilmagan taomlar soni. Bosilsa faqat o'shalar ko'rsatiladi — menyuni tannarxlab chiqish uchun qulay.",
        },
      },
      { h: "Maydonlar" },
      {
        table: {
          head: ["Maydon", "Nima uchun"],
          rows: [
            ["*Nomi*", "Uch tilda. Rus va ingliz bo'sh bo'lsa o'zbekchasi ishlatiladi."],
            ["*Narx*", "So'mda, butun son. Tiyin ishlatilmaydi."],
            ["*Eski narx*", "Chizilgan narx — chegirmani ko'rsatish uchun. Ixtiyoriy."],
            ["*Tannarx*", "Bir porsiya restoranga qancha turadi. Saytda hech qachon ko'rinmaydi. Texkarta tuzilsa bu raqam ishlatilmaydi."],
            ["*Sotuvda*", "O'chirilsa saytdan yo'qoladi (menyudan o'chirilmaydi)."],
            ["*Mashhur*", "Bosh sahifadagi «Mashhur taomlar» blokiga tushadi."],
            ["*Teglar*", "«Achchiq», «Vegetarian» kabi — saytda filtr sifatida ishlaydi."],
          ],
        },
      },
      {
        warn: "Taomni *o'chirish* buyurtmalar tarixini buzmaydi — eski buyurtmalarda nom va narx muzlatilgan holda qoladi. Lekin to'plam (combo) ichidagi taomni o'chirib bo'lmaydi: tizim qaysi to'plamlarda ekanini aytadi.",
      },
      { fig: "menu-item" },
      { see: ["menu-options", "menu-photos", "tech-cards", "menu-combo"] },
    ],
  },
  {
    slug: "menu-options",
    section: "menu",
    title: "Variantlar: o'lcham va qo'shimchalar",
    lead: "Kichik/katta, qo'shimcha pishloq, sousni tanlash — bitta taomda.",
    keys: ["variant", "o'lcham", "qo'shimcha", "modifikator", "opsiya"],
    body: [
      {
        p: "Taom formasida `Variantlar` bo'limi. Har guruhning nomi bor («O'lcham», «Qo'shimcha») va tanlash qoidasi.",
      },
      {
        list: [
          "*Bittasini tanlash* — o'lcham, qo'vurish darajasi. Bittasi standart bo'lib turadi.",
          "*Bir nechtasini tanlash* — qo'shimchalar. Nechtagacha tanlash mumkinligini cheklash mumkin.",
          "Har variantning *narx qo'shimchasi* bor: `+5 000` yoki `0`. Manfiy ham bo'ladi (kichik porsiya arzonroq).",
        ],
      },
      {
        tip: "Variant ombordan ham chiqishi mumkin: qo'shimcha pishloqqa masalliq bog'lasangiz, u sotilganda ombordan yechiladi. Bu taomning o'z texkartasi *bilan birga* ishlaydi, o'rniga emas.",
      },
      {
        warn: "«Kichik» va «Katta» ni ikkita alohida taom qilib qo'yish osonroq ko'rinadi, lekin keyin hisobotda ular ikki mahsulot bo'lib chiqadi va qaysi taom yaxshi sotilayotganini ko'ra olmaysiz.",
      },
      { see: ["menu-item", "tech-cards"] },
    ],
  },
  {
    slug: "menu-combo",
    section: "menu",
    title: "To'plam (combo)",
    lead: "«Oilaviy to'plam»: bir nechta taom bitta narxda.",
    keys: ["combo", "to'plam", "set", "oilaviy"],
    body: [
      {
        p: "To'plam ham menyudagi taom, faqat uning ichida boshqa taomlar bor. `Menyu` → yangi taom → `To'plam` bo'limida a'zolar va miqdorlari tanlanadi.",
      },
      {
        p: "Ekranda a'zolarning alohida narxlari yig'indisi va to'plam narxi ko'rinadi — ya'ni mehmon qancha yutayotgani.",
      },
      {
        warn: "To'plamning *o'z texkartasi yo'q va bo'lmaydi*: tannarxi a'zolaridan chiqadi. Agar to'plamga ham karta yozilsa, ombordan masalliqlar ikki marta yechilardi.",
      },
      {
        tip: "To'plam sotilganda ombor uni a'zolariga yoyib hisoblaydi. Ya'ni yuzta oilaviy to'plam ombordan yuz to'plamning masalliqlarini yechadi — nol emas.",
      },
      { see: ["menu-item", "tech-cards"] },
    ],
  },
  {
    slug: "menu-photos",
    section: "menu",
    title: "Taom rasmlari",
    lead: "Rasm sotadi — lekin og'ir rasm saytni sekinlashtiradi. Ikkalasi ham hal qilingan.",
    keys: ["rasm", "foto", "surat", "yuklash"],
    body: [
      {
        p: "Taom formasida rasm yuklanadi. Tizim uni bir necha o'lchamda saqlaydi va har ekranga mosini beradi — telefonga kichik, kompyuterga katta.",
      },
      {
        list: [
          "Format: JPG yoki PNG. Kvadratga yaqin rasm eng yaxshi ko'rinadi.",
          "Katta fayl yuklashingiz mumkin — tizim o'zi kichraytiradi.",
          "Rasmsiz taom ham normal ko'rinadi: o'rniga nom va rang chiqadi.",
        ],
      },
      {
        tip: "Bitta uslubda suratga oling: bir xil fon, bir xil yorug'lik, bir xil burchak. Menyu shundan «tayyor mahsulot» bo'lib ko'rinadi, oltita turli telefondan yig'ilgan albom bo'lib emas.",
      },
      { see: ["menu-item", "site-design"] },
    ],
  },
  {
    slug: "menu-import",
    section: "menu",
    title: "Menyuni import qilish",
    lead: "Havoladan yoki eski kassangizdan — qo'lda qayta yozmasdan.",
    keys: ["import", "ko'chirish", "excel", "iiko", "poster"],
    body: [
      { h: "Havoladan" },
      {
        steps: [
          "`Menyu` → `Import` → menyuingiz turgan sahifa havolasini qo'ying.",
          "Tizim sahifani o'qib taomlar ro'yxatini taklif qiladi: nom, narx, kategoriya.",
          "Ro'yxatni ekranda tuzatasiz — nomlarni, narxlarni, kategoriyalarni.",
          "`Qo'shish` bosilganda menyuga yoziladi. Shu paytgacha *hech nima saqlanmaydi*.",
        ],
      },
      {
        warn: "O'qish sahifaga qarab 20 soniyagacha davom etadi. Sahifa yopilib qolsa ish to'xtamaydi — qaytib kirsangiz natija joyida turadi.",
      },
      { h: "Boshqa kassa tizimidan" },
      {
        p: "iiko, r_keeper, Clopos, Poster yoki Jowi dan Excel/CSV eksport qilib shu yerga yuklaysiz. Uch xil ma'lumot ko'chiriladi: *masalliqlar*, *texkartalar* va *ombor qoldig'i*.",
      },
      {
        tip: "Tartib muhim: avval masalliqlar, keyin texkartalar (ular masalliqni nom bo'yicha topadi), oxirida qoldiq.",
      },
      { fig: "site-menu" },
      { see: ["pos", "ingredients", "tech-cards"] },
    ],
  },
  {
    slug: "stop-list",
    section: "menu",
    title: "Stop list: bugun sotilmaydigan taomlar",
    lead: "Taomni menyudan o'chirmasdan vaqtincha to'xtatish.",
    keys: ["stop list", "tugadi", "yo'q", "to'xtatish", "sotuvda emas"],
    body: [
      {
        p: "`Menyu` → `Stop list`. Bu ro'yxat *faqat shu filialga* va *faqat bugunga* tegishli — menyudan hech nima o'chirilmaydi.",
      },
      { h: "Uch xil manba" },
      {
        table: {
          head: ["Kim yozadi", "Qanday"],
          rows: [
            ["*Odam*", "Peshtaxtadagi kishi «Stop» bosadi. Qo'lda qaytariladi."],
            [
              "*Kassa tizimi*",
              "iiko/Poster dagi stop list har 3 daqiqada o'qiladi. Bu yerdan qaytarib bo'lmaydi — keyingi o'qishda qaytib keladi.",
            ],
            [
              "*Ombor*",
              "Masalliqlari tugagan taom avtomatik to'xtaydi. Standart holatda *o'chiq*.",
            ],
          ],
        },
      },
      {
        warn: "Ombor bo'yicha to'xtatishni faqat kirimlar va inventarizatsiya muntazam yozilganda yoqing. Aks holda kassa javonda turgan ovqatni sotishdan bosh tortadi — bu tizim qila oladigan eng qimmat xato.",
      },
      {
        tip: "Sahifadagi eng foydali qator — «oxirgi o'qilgan vaqt». «Ulangan» degan yashil belgi soat undan o'tishi bilan eskiradi; oxirgi o'qish vaqti esa eskirmaydi.",
      },
      { fig: "stop-list" },
      { see: ["pos", "stock-stop"] },
    ],
  },
  {
    slug: "promotions",
    section: "menu",
    title: "Aksiya va promokod",
    lead: "Chegirma berishning ikki yo'li va ular qanday qo'shiladi.",
    keys: ["aksiya", "promokod", "chegirma", "skidka", "kupon"],
    body: [
      {
        p: "`Menyu` → `Aksiyalar`. Ikki tur bor va farqi shundaki: *aksiya* saytda ko'rinadi va o'zi ishlaydi, *promokod* esa mijoz kiritishi kerak.",
      },
      {
        list: [
          "*Foizli* — «Barcha pitsalarga 20%».",
          "*Summali* — «50 000 dan yuqori buyurtmaga 10 000 chegirma».",
          "*Bepul yetkazish* — chegirma o'rniga yetkazish narxi olib tashlanadi.",
          "Har biriga muddat, minimal summa va qaysi kategoriyaga tegishli ekani qo'yiladi.",
        ],
      },
      {
        warn: "Promokodlar saytda *hech qachon* ro'yxat bo'lib chiqmaydi — faqat aksiyalar ko'rinadi. Aks holda har mehmon eng katta kodni topib ishlatardi.",
      },
      {
        tip: "Chegirma tannarxdan pastga tushishi mumkin. Menyudagi marja ustuni shuning uchun bor: 20% chegirma qaysi taomni zararga sotayotganingizni oldindan ko'rsatadi.",
      },
      { fig: "promotions" },
      { see: ["loyalty", "reports-sales"] },
    ],
  },
  {
    slug: "recommendations",
    section: "menu",
    title: "Tavsiyalar va upsell",
    lead: "«Bunga ichimlik qo'shasizmi?» — savatda va taom sahifasida.",
    keys: ["tavsiya", "upsell", "qo'shimcha sotuv", "savat"],
    body: [
      {
        p: "Taom formasida `Tavsiya qilinadigan taomlar` ro'yxati bor. U taom sahifasida va savatda ko'rsatiladi.",
      },
      {
        p: "Qo'lda ko'rsatmasangiz ham tizim tavsiya beradi: birga buyurtma qilinadigan taomlarni o'zi topadi.",
      },
      {
        tip: "Uchtadan ko'p tavsiya qilmang. To'rtinchi taklif reklamaga o'xshaydi va savat sahifasida mehmonni to'lovdan chalg'itadi.",
      },
      { fig: "site-cart" },
      { see: ["menu-item", "crm"] },
    ],
  },

  // ═══════════════════════════════════ Buyurtmalar
  {
    slug: "order-flow",
    section: "orders",
    title: "Buyurtma holatlari",
    lead: "Kelganidan berilgunicha: har holat nimani anglatadi va kim o'zgartiradi.",
    keys: ["buyurtma", "holat", "status", "yo'lda", "yetkazildi"],
    body: [
      {
        table: {
          head: ["Holat", "Ma'nosi"],
          rows: [
            ["`Yangi`", "Buyurtma keldi, hali hech kim ko'rmadi. Ovoz shu holatda chalinadi."],
            ["`Tasdiqlangan`", "Qabul qilindi. Kassa tizimi ulangan bo'lsa, buyurtma aynan shu paytda kassaga ketadi."],
            ["`Tayyorlanmoqda`", "Oshxonada."],
            ["`Yo'lda`", "Kuryerga berildi. Mijoz kuzatuv sahifasida shuni ko'radi."],
            ["`Yetkazildi`", "Tugadi. Shu paytdan pul tushumga kiradi."],
            ["`Bekor qilindi`", "Sabab bilan. Sababsiz bekor qilib bo'lmaydi."],
          ],
        },
      },
      {
        p: "Har o'zgarish vaqti bilan yoziladi — buyurtma kartochkasida «qachon qabul qilindi, qancha turdi» ko'rinadi.",
      },
      {
        warn: "Buyurtma kassaga *tasdiqlashda* yuboriladi, kelganda emas. Sababi: kassa — birovning buxgalteriyasi, u yerga ketgan xato chekni qo'lda bekor qilish kerak bo'ladi.",
      },
      { fig: "orders" },
      { see: ["order-accept", "order-cancel", "pos"] },
    ],
  },
  {
    slug: "order-accept",
    section: "orders",
    title: "Buyurtmani qabul qilish va ovoz",
    lead: "Yangi buyurtma kelganda nima bo'ladi va ovoz chalinmasa nima qilish kerak.",
    keys: ["ovoz", "signal", "yangi buyurtma", "qabul qilish", "eshitilmayapti"],
    body: [
      {
        p: "Yangi buyurtma kelganda panelda ovoz chalinadi va o'ng pastda banner chiqadi. Ovoz `Qabul qilish` bosilguncha takrorlanadi — bu ataylab: bir marta chalingan signalni band kechada hech kim eshitmaydi.",
      },
      { h: "Ovoz chalinmayapti" },
      {
        steps: [
          "Panelning chap pastidagi `Ovoz yoqilgan` tugmasini tekshiring.",
          "Brauzer ovozga ruxsat bergan bo'lishi kerak. Sahifada bir marta biror joyga bosing — brauzerlar ovozni birinchi bosishdan keyin qo'yib yuboradi.",
          "Kompyuterning o'z ovozi o'chiq emasligini tekshiring.",
        ],
      },
      {
        tip: "«5 daqiqaga jim» tugmasi bor — mehmon bilan gaplashayotganda foydali. Besh daqiqadan keyin ovoz o'zi qaytadi.",
      },
      { see: ["order-flow", "kds"] },
    ],
  },
  {
    slug: "order-cancel",
    section: "orders",
    title: "Buyurtmani bekor qilish va pulni qaytarish",
    lead: "Sabab majburiy — va onlayn to'langan pul o'zi qaytmaydi.",
    keys: ["bekor", "qaytarish", "refund", "sabab"],
    body: [
      {
        p: "Buyurtma kartochkasida `Bekor qilish`. Sabab yozilmasa saqlanmaydi: sababsiz bekor qilingan buyurtma bir oydan keyin hech kimga hech nima aytmaydi.",
      },
      {
        warn: "Onlayn to'langan buyurtmani bekor qilish *pulni avtomatik qaytarmaydi*. Pul to'lov tizimida qoladi va uni o'sha tizim kabinetidan qaytarasiz. Panel buni ochiq yozadi — «bekor qilindi» degan yozuv mijozning kartasiga pul qaytganini anglatmaydi.",
      },
      {
        p: "Kassadagi chek uchun boshqa amal bor — *qaytarish*: sotuv qoladi, pul o'zgaradi. Bu hisobotlarda «bekor qilindi» dan alohida ko'rinadi.",
      },
      { see: ["payments", "till-shift"] },
    ],
  },
  {
    slug: "preorder",
    section: "orders",
    title: "Oldindan buyurtma",
    lead: "Mijoz «soat 19:00 ga» deb buyurtma beradi.",
    keys: ["oldindan", "predzakaz", "vaqtga", "ertaga"],
    body: [
      {
        p: "`Sozlamalar` → `Buyurtmalar` da oldindan buyurtma yoqiladi. Ikki sozlama bor: eng erta qancha vaqtdan keyin (masalan 1 soat) va eng kech necha kunga (masalan 7 kun).",
      },
      {
        p: "Bunday buyurtma ro'yxatda alohida belgilanadi va *vaqti kelganda* yuqoriga chiqadi — soat 10 da kelgan «19:00 ga» buyurtmasi kun bo'yi oshxonani chalg'itmaydi.",
      },
      {
        tip: "Bandlik soatlarini yoping: tushlik payti oldindan buyurtma qabul qilinmasin desangiz, ish vaqtidan alohida chegaralash mumkin.",
      },
      { fig: "site-booking" },
      { see: ["order-flow", "reservations"] },
    ],
  },
  {
    slug: "reservations",
    section: "orders",
    title: "Stol bron qilish",
    lead: "Mehmon saytdan stol band qiladi, siz tasdiqlaysiz.",
    keys: ["bron", "stol", "band qilish", "rezerv"],
    body: [
      {
        p: "`Buyurtmalar` → `Bronlar`. Mehmon saytda sana, vaqt, kishi soni va telefonini qoldiradi.",
      },
      {
        table: {
          head: ["Holat", "Ma'nosi"],
          rows: [
            ["`Yangi`", "Kutmoqda — siz tasdiqlashingiz kerak."],
            ["`Tasdiqlangan`", "Stol saqlanadi. Mijozga SMS ketadi."],
            ["`O'tirdi`", "Mehmon keldi."],
            ["`Tugadi` / `Bekor`", "Yopildi. Bekor qilish uchun sabab kerak."],
          ],
        },
      },
      {
        p: "`Sozlamalar` → `Bron` da zal chizmasi tuziladi: stollar, ularning joyi va nechta kishilik ekani.",
      },
      {
        warn: "Bron tasdiqlanganda SMS ketadi. SMS provayderi sozlanmagan bo'lsa mehmon hech qanday tasdiq olmaydi va odatda telefon qiladi.",
      },
      { fig: "reservations" },
      { see: ["sms", "till-hall"] },
    ],
  },
  {
    slug: "qr-menu",
    section: "orders",
    title: "QR menyu: stoldan buyurtma",
    lead: "Har stolga QR kod — mehmon telefonidan menyuni ochadi va buyurtma beradi.",
    keys: ["qr", "stol", "qr menyu", "kod"],
    body: [
      {
        p: "`Menyu` → `QR kodlar` da har stol uchun kod yasaladi va PDF bo'lib chop etishga tayyor chiqadi.",
      },
      {
        p: "Mehmon kodni skanerlaydi → menyu ochiladi → buyurtma beradi. Buyurtma stol raqami bilan keladi, ya'ni ofitsiant kimga olib borishni biladi.",
      },
      {
        tip: "Kod stolga *yopishtirilgan* bo'lishi kerak, menyu kitobiga emas: kitob boshqa stolga ketadi va buyurtma noto'g'ri stolga tushadi.",
      },
      {
        warn: "QR bilan buyurtma bergan mehmon ham «zalda» hisoblanadi, lekin bu kassa cheki emas. Zal sotuvlari ro'yxati (`Zal sotuvlari`) faqat kassada ochilgan cheklarni ko'rsatadi.",
      },
      { fig: "qr" },
      { see: ["till-hall", "order-flow"] },
    ],
  },

  // ═══════════════════════════════════ Yetkazib berish
  {
    slug: "delivery-setup",
    section: "delivery",
    title: "Yetkazib berishni yoqish",
    lead: "Minimal summa, bepul yetkazish chegarasi va narx qanday hisoblanishi.",
    keys: ["yetkazish", "dostavka", "minimal", "bepul yetkazish"],
    body: [
      {
        p: "`Sozlamalar` → `Yetkazib berish`. Avval yoqiladi, keyin narx qanday hisoblanishi tanlanadi: *masofa bo'yicha* yoki *xaritadagi zonalar bo'yicha*.",
      },
      {
        table: {
          head: ["Sozlama", "Ma'nosi"],
          rows: [
            ["*Minimal buyurtma*", "Bundan kam summaga yetkazilmaydi. Savatda mehmonga qancha yetishmayotgani yoziladi."],
            ["*Bepul yetkazishdan*", "Shu summadan yuqori buyurtmaga yetkazish tekin. Bo'sh qoldirilsa — hech qachon tekin emas."],
            ["*Hisoblash usuli*", "Masofa yoki zonalar. Ikkinchisining maydonlari yashiriladi, lekin ma'lumot saqlanadi — fikringizni o'zgartirsangiz qaytadan chizish shart emas."],
          ],
        },
      },
      {
        warn: "Yetkazib berish *har filialda alohida*. Bir nechta filial bo'lsa, sozlama sahifasi qamrov haqida ogohlantiradi — bu yerdagi xatolar jimgina bo'ladi: forma to'ldirilgandek ko'rinadi, hech nima xato bermaydi, va natija butunlay boshqa joyda — buyurtma umuman kelmaydigan filial bo'lib bilinadi.",
      },
      { h: "Eng ko'p uchraydigan to'rt xato" },
      {
        list: [
          "*Yetkazish o'chiq* — filial yetkazish buyurtmalarida umuman qatnashmaydi.",
          "*Xaritada nuqta yo'q* — masofa aynan shundan o'lchanadi, ya'ni hisob ishlamaydi.",
          "*Maksimal masofa = 0* — bu «cheklov yo'q», «yetkazmaydi» emas. Bir nechta filialda bu eng xavflisi: bitta filial hamma buyurtmani o'ziga oladi.",
          "*Boshlang'ich narx va km narxi = 0* — yetkazish bepul bo'lib qoladi.",
        ],
      },
      { see: ["delivery-zones", "delivery-radius", "map-provider"] },
    ],
  },
  {
    slug: "delivery-zones",
    section: "delivery",
    title: "Yetkazish zonalarini chizish",
    lead: "Xaritada chegara chizasiz, har zonaga o'z narxini qo'yasiz — mehmon ham shuni ko'radi.",
    keys: ["zona", "polygon", "chegara", "xarita", "hudud", "narx"],
    body: [
      {
        p: "`Sozlamalar` → `Yetkazib berish` → hisoblash usuli sifatida *zonalar* tanlanadi. Pastda xarita ochiladi.",
      },
      { h: "Zona chizish" },
      {
        steps: [
          "`Yangi zona` ni bosing.",
          "Xaritada chegara nuqtalarini ketma-ket bosib chiqing. Kamida uchta nuqta kerak.",
          "Oxirgi nuqtadan birinchisiga tutashtiring — shakl yopiladi.",
          "Zonaga nom bering («Markaz», «Chilonzor») va narxlash usulini tanlang.",
          "Saqlang. Zona darhol saytda ham ko'rinadi.",
        ],
      },
      { h: "Zona narxi ikki xil bo'ladi" },
      {
        list: [
          "*Belgilangan narx* — zona ichidagi har manzilga bir xil summa. Eng sodda va mijozga tushunarli.",
          "*Km bo'yicha* — boshlang'ich narx + har km uchun qo'shimcha, masofa restorandan o'lchanadi.",
        ],
      },
      {
        warn: "Zona *kamida uch nuqtali* bo'lsa ishlaydigan hisoblanadi. Ishlaydigan zona bo'lsa masofa sozlamalari (boshlang'ich narx, km narxi, maksimal masofa) umuman e'tiborga olinmaydi va zonadan tashqariga yetkazilmaydi.",
      },
      {
        warn: "Zonani chizib bo'lib *saqlashni unutmang*. Chizilgan, lekin saqlanmagan shakl ekranda turadi va butunlay haqiqiydek ko'rinadi.",
      },
      {
        tip: "Mehmon checkout sahifasida zonalarni xaritada va tagida ro'yxat bo'lib ko'radi: qaysi zona, qancha. Manzil tanlanmagan bo'lsa xarita butun qamrovga moslanadi — ya'ni mehmon birinchi qarashda «bizga yetkazadimi?» degan savolga javob oladi.",
      },
      { see: ["delivery-radius", "map-provider", "delivery-setup"] },
    ],
  },
  {
    slug: "delivery-radius",
    section: "delivery",
    title: "Masofa bo'yicha narx",
    lead: "Zona chizishga vaqt yo'q bo'lsa: boshlang'ich narx + har kilometr.",
    keys: ["masofa", "radius", "km", "narx hisobi"],
    body: [
      {
        p: "Hisoblash usuli sifatida *masofa* tanlanadi. Uchta raqam kerak:",
      },
      {
        table: {
          head: ["Maydon", "Misol"],
          rows: [
            ["*Boshlang'ich narx*", "10 000 — har buyurtmaga qo'shiladi"],
            ["*Har km uchun*", "3 000 — restorandan mijozgacha bo'lgan to'g'ri masofa, yuqoriga yaxlitlanadi"],
            ["*Maksimal masofa*", "12 km — bundan uzoqqa yetkazilmaydi"],
          ],
        },
      },
      {
        p: "Masofa *to'g'ri chiziq* bo'yicha o'lchanadi, yo'l bo'yicha emas. Shuning uchun km narxini yo'l uzunligini hisobga olib qo'ying.",
      },
      {
        warn: "*Maksimal masofa 0* — «cheklov yo'q» degani. Ko'pchilik buni «yetkazmaydi» deb tushunadi va bir nechta filialli restoranda bu bitta filialga butun shaharni beradi.",
      },
      { see: ["delivery-zones", "delivery-setup"] },
    ],
  },
  {
    slug: "map-provider",
    section: "delivery",
    title: "Xarita: 2GIS, Yandex yoki Google",
    lead: "Qaysi xaritani ishlatish va kalitni qayerdan olish.",
    keys: ["xarita", "2gis", "yandex", "google", "api key", "kalit"],
    body: [
      {
        p: "`Sozlamalar` → `Xarita` da provayder tanlanadi. Standart — *2GIS*, chunki O'zbekistonda uning manzillari eng to'liq.",
      },
      {
        table: {
          head: ["Provayder", "Kalit qayerdan", "Narx"],
          rows: [
            ["*2GIS*", "dev.2gis.com — bepul ro'yxatdan o'tish", "Bu hajmda amalda bepul"],
            ["*Yandex*", "developer.tech.yandex.ru", "Bu hajmda amalda bepul"],
            ["*Google*", "console.cloud.google.com", "*Pullik* — hisobni siz to'laysiz"],
          ],
        },
      },
      {
        warn: "Har provayderning kaliti *alohida maydonda*. Yandex'ni sinab ko'rib 2GIS'ga qaytsangiz, kalitlar aralashib ketmaydi — bo'sh xarita va konsoldagi xato bo'lib chiqadigan xato shu tarzda oldi olingan.",
      },
      {
        warn: "Xarita kaliti *sir emas va sir bo'la olmaydi* — u brauzerga beriladi, bu barcha xarita SDK'larida shunday. Himoyani provayder kabinetidagi *domen cheklovi* beradi: kalitni faqat sizning domeningizdan ishlatishga ruxsat bering.",
      },
      {
        tip: "Eski telefonda xarita o'rniga kulrang quti chiqsa — 2GIS WebGL talab qiladi. Bunday mijozlar ko'p bo'lsa Yandex'ga o'ting: u eskiroq qurilmalarda ham ishlaydi.",
      },
      { see: ["delivery-zones"] },
    ],
  },
  {
    slug: "couriers",
    section: "delivery",
    title: "Kuryerlar",
    lead: "Hisob yaratish, buyurtma biriktirish va kuryer qo'lidagi pulni hisoblash.",
    keys: ["kuryer", "yetkazuvchi", "dostavshik", "hisob"],
    body: [
      {
        p: "`Pul va jamoa` → `Kuryerlar` da hisob yaratiladi: ism, telefon, login va parol. Kuryer *o'zi ro'yxatdan o'ta olmaydi* — hisobni faqat siz berasiz.",
      },
      {
        p: "Kuryerning uch holati bor: `Ishda emas`, `Bo'sh`, `Band`. Buyurtmani unga panel orqali biriktirasiz.",
      },
      { h: "Kuryer qo'lidagi pul" },
      {
        p: "Naqd yetkazishda pul kuryerda qoladi va kassaga *topshirilganda* kiradi. Shuning uchun kassa hisobida u alohida ko'rsatiladi — kamomadni tekshirayotgan odam birinchi navbatda shu raqamni so'raydi.",
      },
      {
        warn: "Kuryer topshirig'i *xarajat emas*: bu bizning nomimizdan yig'ilgan naqdning kassaga kirishi. Uni xarajat deb yozish restoranning o'z tushumini o'zidan ayirish demakdir.",
      },
      { fig: "couriers" },
      { see: ["courier-app", "finance", "delivery-external"] },
    ],
  },
  {
    slug: "courier-app",
    section: "delivery",
    title: "Kuryer ilovasi",
    lead: "Telefondagi sahifa: buyurtmalar, manzil, xaritaga o'tish, kunlik hisob.",
    keys: ["kuryer ilovasi", "telefon", "pwa", "kuryer sahifasi"],
    body: [
      {
        p: "Kuryer `saytingiz.uz/kuryer` manzilidan kiradi — alohida ilova o'rnatish shart emas. Telefonda «Bosh ekranga qo'shish» qilinsa, ilovadek ochiladi.",
      },
      {
        list: [
          "Unga biriktirilgan buyurtmalar ro'yxati: manzil, telefon, summa, to'lov turi.",
          "Manzilni bosgan zahoti telefonning xarita ilovasi ochiladi va yo'l ko'rsatadi.",
          "`Yetkazdim` bosilganda buyurtma yopiladi.",
          "Kun oxirida qancha yetkazgani va qo'lida qancha naqd borligi ko'rinadi.",
        ],
      },
      {
        warn: "Kuryerni xaritada *jonli kuzatish hozircha yo'q*. Mijoz «yo'lda» degan holatni ko'radi, lekin nuqta harakatlanmaydi.",
      },
      {
        tip: "Joylashuvga ruxsat so'raladi — bu faqat buyurtma biriktirilganda ishlatiladi. Telefon ruxsat bermasa ilova baribir ishlaydi.",
      },
      { fig: "courier-login" },
      { see: ["couriers"] },
    ],
  },
  {
    slug: "delivery-external",
    section: "delivery",
    title: "Tashqi yetkazish xizmatlari",
    lead: "O'z kuryeringiz bo'lmasa — buyurtmani tashqi xizmatga topshirish.",
    keys: ["yandex dostavka", "tashqi kuryer", "logistika"],
    body: [
      {
        p: "`Sozlamalar` → `Yetkazib berish` → `Tashqi xizmat` da provayder ulanadi. Buyurtma tasdiqlangandan keyin unga topshiriladi.",
      },
      {
        p: "Bu variant o'z kuryerini yollagandan arzonroq bo'lishi mumkin — ayniqsa kunlik buyurtma soni kam bo'lsa.",
      },
      {
        tip: "Ikkalasini birga ishlatish mumkin: yaqin manzillarni o'z kuryeringiz, uzoqlarini tashqi xizmat olib boradi.",
      },
      { see: ["couriers"] },
    ],
  },

  // ═══════════════════════════════════ Kassa va zal
  {
    slug: "till-setup",
    section: "till",
    title: "Kassani o'rnatish",
    lead: "Dasturni yuklab olish, qurilmani ro'yxatdan o'tkazish va birinchi sotuv.",
    keys: ["kassa", "pos", "o'rnatish", "yuklab olish", "dastur"],
    body: [
      {
        steps: [
          "Kassa kompyuteriga `keel.uz/download` dan dasturni yuklab oling.",
          "Ishga tushiring — birinchi ochilishda qaysi restoranga tegishli ekani so'raladi.",
          "Panelda `Sozlamalar` → `Kassa` dan qurilma kodini oling va kiriting.",
          "Kassir o'z PIN kodi bilan kiradi.",
        ],
      },
      {
        p: "Bitta yuklab olingan fayl *har restoran* uchun ishlaydi — server manzili dasturga oldindan yozilmaydi, u birinchi ochilishda so'raladi.",
      },
      {
        warn: "Kassa internetsiz ham ishlaydi: sotuv qurilmada saqlanadi va aloqa tiklanganda yuboriladi. Navbat to'xtamaydi. Har sotuvning o'z id'si bor, ya'ni qayta yuborish ikki marta pul olmaydi.",
      },
      { see: ["till-shift", "printer-connect", "till-payment"] },
    ],
  },
  {
    slug: "till-shift",
    section: "till",
    title: "Smena: ochish, yopish, X va Z hisobot",
    lead: "Kassadagi pul kun oxirida to'g'ri chiqishi uchun.",
    keys: ["smena", "kassa", "z hisobot", "x hisobot", "yopish", "kamomad"],
    body: [
      {
        steps: [
          "Kun boshida `Smena ochish` — kassadagi boshlang'ich pulni kiritasiz.",
          "Kun davomida sotuv o'zi hisoblanadi.",
          "Istalgan payt `X hisobot` — hozircha qancha bo'lishi kerakligini ko'rsatadi va *hech nimani o'zgartirmaydi*.",
          "Kun oxirida `Smena yopish` — yashiqdagi pulni sanab yozasiz.",
          "`Z hisobot` yopish javobida chiqadi — chop eting.",
        ],
      },
      {
        warn: "Mahsulot — *farq*, jami emas. Tizim qancha bo'lishi kerakligini o'zi hisoblab muzlatadi, siz sanaganingizni yozasiz, va farq bo'lsa *sabab majburiy*. Sababsiz farq saqlanmaydi.",
      },
      {
        p: "Z hisobotda «shundan qarz qaytdi» degan alohida qator bor. Kecha qarzga berilib bugun to'langan pul bugungi yashiqqa tushadi, sotuv esa kechaniki — bu qator busiz tushuntirib bo'lmaydigan farq bo'lib qolardi.",
      },
      {
        warn: "Bitta filialda *bitta ochiq smena* bo'ladi. Ikkinchisini ochmoqchi bo'lsangiz rad etiladi: ikki ochiq smenada «kassada qancha bo'lishi kerak» savoli javobsiz qoladi.",
      },
      { see: ["finance", "till-payment"] },
    ],
  },
  {
    slug: "till-hall",
    section: "till",
    title: "Zal: stollar, chekni bo'lish va birlashtirish",
    lead: "Ofitsiant ekrani: stol ochish, buyurtma qo'shish, hisobni bo'lish.",
    keys: ["zal", "stol", "ofitsiant", "chek bo'lish", "hisob"],
    body: [
      {
        p: "Zal ekranida stollar chizmasi ko'rinadi. Stolga bosilsa chek ochiladi; unga taom qo'shiladi va stol band bo'lib qoladi.",
      },
      {
        list: [
          "*Bo'lish* — bitta chekni bir nechtaga ajratish. Hammasini bo'lib yuborib bo'lmaydi.",
          "*Birlashtirish* — ikki stolni bitta chekka. Yutilgan chek *bekor qilinadi, o'chirilmaydi*: uning raqami va qatorlari tarixda qoladi.",
          "*Xizmat haqi* — filialga qo'yiladi va *faqat stolga* qo'shiladi. Foiz stol o'tirgan paytda chekka ko'chiriladi, ya'ni kechqurun foizni o'zgartirsangiz ochiq stollar eski foizda qoladi.",
        ],
      },
      {
        tip: "Yopilgan cheklarni kassaning o'zida ko'rish mumkin (`Sotuvlar` → `Yopilgan`) — «yana chiqarib bering» degan savol uchun panel logini kerak emas. Faqat bugun va faqat o'z filiali.",
      },
      { fig: "checks" },
      { see: ["till-shift", "qr-menu"] },
    ],
  },
  {
    slug: "till-payment",
    section: "till",
    title: "Kassada to'lov turlari",
    lead: "Naqd, terminal, QR skanerlash va qarz.",
    keys: ["to'lov", "naqd", "karta", "qr", "qarz", "payme", "click"],
    body: [
      {
        table: {
          head: ["Tur", "Qanday ishlaydi"],
          rows: [
            ["*Naqd*", "Kassir summani kiritadi, qaytim hisoblanadi."],
            ["*Terminal*", "Bank terminali orqali. Tizim buni *ko'rmaydi* — kassir o'zi belgilaydi."],
            ["*QR (Payme / Click / Uzum)*", "Ekranda QR chiqadi, mehmon o'z telefonidan to'laydi, chek o'zi yopiladi."],
            ["*Qarz*", "To'lov emas — hali to'lamaslikning yozuvi. Mijoz ko'rsatilishi shart."],
          ],
        },
      },
      {
        warn: "QR to'lovda chek *to'lov tasdiqlanmaguncha yopilmaydi*. Bu ataylab sekin: tanlagan zahoti «to'landi» deb belgilash har sinovda ishlaydi va navbatda bekor qilingan yoki birovning ekranida qilingan to'lov uchun ovqat berib yuboradi.",
      },
      {
        p: "*Qarz* mijozsiz qabul qilinmaydi. Nomsiz qarz — peshtaxtadagi daftarning o'zi: hech kimning kartochkasida ko'rinmaydigan va hech kim so'ramaydigan pul. Kassada mijoz telefon raqami bo'yicha topiladi.",
      },
      {
        tip: "Qarz kassada ham qaytariladi — pul kassaga keladi, ya'ni uni qabul qiladigan odam yashiq oldidagi odam. Kassirni panel logini bor odamni izlashga yuborish menejer parolining kassa yoniga yozib qo'yilishining yo'li.",
      },
      { see: ["payments", "till-shift", "crm"] },
    ],
  },
  {
    slug: "kds",
    section: "till",
    title: "Oshxona ekrani (KDS)",
    lead: "Oshpaz butun kecha bitta savolga javob beradi: keyin nima pishiraman.",
    keys: ["kds", "oshxona", "oshpaz", "ekran", "tayyor"],
    body: [
      {
        p: "`saytingiz.uz/staff/kitchen` — oshxonadagi planshet yoki monitor uchun. Ishchi o'z hisobi bilan kiradi.",
      },
      {
        p: "Ekranda chek, kutish vaqti va bitta tugma bor. Pul, mijoz, manzil va filtrlar *ataylab yo'q*: jamini o'qib o'tib taomni topishi kerak bo'lgan oshpaz ekranni o'qishni to'xtatadi.",
      },
      {
        warn: "KDS'ga ruxsat *har bir odamga alohida* beriladi (`Xodimlar` → xodim → «Oshxona ekrani»). Standart holatda o'chiq. Umumiy planshetda hamma kira olsa, ofitsiant ham «Tayyor» bosa oladi — ya'ni chekni peshtaxtadan yo'qotib, panelga «oshxona pishirdi» deb aytadi.",
      },
      {
        p: "«Tayyor» — yangi holat emas, vaqt belgisi. Buyurtma holati o'zgarmaydi; panelda «Oshxona tayyorladi» nishoni chiqadi. Buyurtma orqaga qaytarilsa bu belgi tozalanadi.",
      },
      { fig: "kitchen-login" },
      { see: ["staff-roles", "order-flow"] },
    ],
  },
  {
    slug: "kiosk",
    section: "till",
    title: "Kiosk: o'zi buyurtma beradigan ekran",
    lead: "Zaldagi sensorli ekran — mehmon navbatsiz buyurtma beradi.",
    keys: ["kiosk", "sensorli", "self service", "terminal"],
    body: [
      {
        p: "`saytingiz.uz/kiosk` — filialga biriktirilgan ekran. Panelda kiosk uchun kod beriladi, ekran bir marta shu kod bilan ro'yxatdan o'tadi.",
      },
      {
        list: [
          "Menyu katta tugmalar bilan chiqadi, buyurtma raqam oladi.",
          "To'lov QR yoki terminal orqali.",
          "Buyurtma to'g'ridan-to'g'ri oshxonaga tushadi.",
        ],
      },
      {
        tip: "Kiosk ekranini tez-tez bosishdan qotib qolmasligi uchun alohida qoidalar bor: zoom o'chirilgan, ikki marta bosish takror buyurtma yaratmaydi.",
      },
      { fig: "kiosk-login" },
      { see: ["kds", "till-setup"] },
    ],
  },

  // ═══════════════════════════════════ Printerlar
  {
    slug: "printer-connect",
    section: "printers",
    title: "Printerni ulash",
    lead: "Chek printeri kassaga, oshxona printeri oshxonaga — ikkalasi ham bir xil ulanadi.",
    keys: ["printer", "ulash", "chek printeri", "usb", "tarmoq"],
    body: [
      {
        p: "Printer *kassa kompyuteriga* ulanadi (USB yoki tarmoq orqali), keyin kassa dasturida tanlanadi. Panel printerni to'g'ridan-to'g'ri ko'rmaydi.",
      },
      {
        steps: [
          "Printerni Windows'ga odatdagidek o'rnating va sinov sahifasini chiqarib ko'ring. Bu ishlamasa qolgani ham ishlamaydi.",
          "Kassa dasturida `Sozlamalar` → `Printerlar` ni oching.",
          "Ro'yxatdan printer nomini tanlang — nomlar Windows'ning o'zidan olinadi.",
          "Har printerga vazifa bering: `Chek`, `Oshxona` yoki `Bar`.",
          "`Sinov chop etish` bosing.",
        ],
      },
      {
        tip: "Printerni tarmoqda «umumiy» (`net share`) qilish *shart emas*. Dastur printer nomini Windows'ning o'zidan so'raydi; umumiy papka yo'li faqat zaxira variant.",
      },
      {
        warn: "Bitta printerga ikki vazifa bermang. Oshxona chekini mijozning cheki bilan bitta qog'ozga chiqargan restoran ikkalasini ham qayta chop etadi.",
      },
      { see: ["printer-kitchen", "printer-problems"] },
    ],
  },
  {
    slug: "printer-kitchen",
    section: "printers",
    title: "Oshxona printeri: qaysi taom qayerga chiqadi",
    lead: "Issiq sex, sovuq sex va bar — har biriga o'z chekini yuborish.",
    keys: ["oshxona printeri", "sex", "bar", "yo'naltirish"],
    body: [
      {
        p: "Har kategoriyaga printer biriktiriladi: «Ichimliklar» → bar printeri, «Issiq taomlar» → oshxona printeri. Buyurtma kelganda har printerga *faqat o'ziga tegishli* qatorlar chiqadi.",
      },
      {
        list: [
          "Chekda stol raqami yoki buyurtma raqami, vaqt va taom nomi bo'ladi — narx bo'lmaydi.",
          "Mijozning izohi («piyozsiz») aynan shu chekda chiqadi.",
          "Buyurtmaga taom qo'shilsa, faqat *yangi* qatorlar chiqadi.",
        ],
      },
      {
        warn: "Kategoriyaga printer biriktirilmagan bo'lsa, uning taomlari *hech qayerga* chiqmaydi. Yangi kategoriya qo'shganda buni tekshiring — bu chiqmagan chekning eng ko'p uchraydigan sababi.",
      },
      { see: ["printer-connect", "printer-problems", "categories"] },
    ],
  },
  {
    slug: "printer-problems",
    section: "printers",
    title: "Chek chiqmayapti",
    lead: "Tizimdagi eng jim nosozlik: buyurtma ekranda, sotuv hisobotda, ovqat esa pishirilmagan.",
    keys: ["chek chiqmadi", "printer ishlamayapti", "chop etilmadi", "????"],
    body: [
      {
        p: "Panelda `Sozlamalar` → `Printerlar` da *chop etish navbati* bor: qaysi topshiriq yuborilgan, qaysi biri chiqmagan. Chiqmagan chek borligida panelda banner ham chiqadi.",
      },
      { h: "Tartib bilan tekshiring" },
      {
        steps: [
          "Printer yoqilganmi va qog'ozi bormi.",
          "Kassa dasturi ochiqmi — chek dastur orqali ketadi, panel orqali emas.",
          "Windows'da printer «oflayn» emasmi.",
          "Chop etish navbatida topshiriqni `Qayta yuborish` bilan qaytadan yuboring.",
        ],
      },
      {
        warn: "Allaqachon chiqqan topshiriq qayta yuborilmaydi — «topilmadi» deydi. Bu ataylab: ikki marta chiqqan oshxona cheki ikki marta pishirilgan taom demakdir.",
      },
      { h: "Chekda savol belgilari (????) chiqyapti" },
      {
        p: "Bu kodlash muammosi: printer o'zbek yoki rus harflarini tanimayapti. Kassa dasturida printer uchun kod sahifasini almashtiring — odatda `CP866` yoki `CP1251` to'g'ri keladi. Almashtirish jurnalga yoziladi.",
      },
      { see: ["printer-connect", "printer-kitchen"] },
    ],
  },

  // ═══════════════════════════════════ Ombor va tannarx
  {
    slug: "stock-intro",
    section: "stock",
    title: "Ombor qanday ishlaydi",
    lead: "Zanjir: masalliq → texkarta → kirim → sotuv → chiqim → sanash. Har bosqich o'z raqamini beradi.",
    keys: ["ombor", "sklad", "qoldiq", "tannarx", "boshlash"],
    body: [
      {
        p: "Ombor bo'limi ikki savolga javob beradi: *bir porsiya qanchaga tushadi* va *javonda nima qoldi*. Ikkalasi ham bitta zanjirdan chiqadi.",
      },
      {
        steps: [
          "*Masalliqlar* — nima sotib olasiz va qanchaga (kilo, litr, dona).",
          "*Texkartalar* — bir porsiyaga nima ketadi. Shundan tannarx chiqadi.",
          "*Kirim* — nakladnoy: nima keldi, kimdan, qanchaga. Narxlar shu yerdan yangilanadi.",
          "*Sotuv* — sotilgan taomlar texkartasi bo'yicha ombordan yechiladi. Bu avtomatik.",
          "*Chiqim* — buzilgani, to'kilgani, xodimlar ovqati.",
          "*Inventarizatsiya* — sanash. «Bo'lishi kerak» aynan shundan boshlab hisoblanadi.",
        ],
      },
      {
        warn: "Ombor ekranlari *bitta filialni talab qiladi*. Bir nechta filial bo'lsa yuqoridan filialni tanlang; bitta filialli restoranda bu so'ralmaydi.",
      },
      {
        tip: "Boshlash tartibi: masalliqlar → texkartalar → birinchi inventarizatsiya → kirimlarni muntazam kiritish. Inventarizatsiyasiz «bo'lishi kerak» butun kirim tarixidan hisoblangan taxmin bo'lib qoladi.",
      },
      { fig: "stock" },
      { see: ["ingredients", "tech-cards", "stocktake", "stock-balances"] },
    ],
  },
  {
    slug: "ingredients",
    section: "stock",
    title: "Masalliq qo'shish",
    lead: "Nomenklatura: nima sotib olasiz, qaysi birlikda va qanchaga.",
    keys: ["masalliq", "ingredient", "narx", "birlik", "kg"],
    body: [
      {
        p: "`Ombor` → `Masalliqlar`. Har masalliqning nomi, o'lchov birligi va *sotib olish narxi* bor.",
      },
      {
        fig: "ingredients",
        notes: {
          unit: "O'lchov birligi: kilo, litr yoki dona. Texkartada mos ravishda gramm, millilitr va dona ishlatiladi.",
          expected: "«Bo'lishi kerak» — oxirgi sanashdan beri hisoblangan taxminiy qoldiq.",
        },
      },
      {
        p: "Narx *sotib olinadigan birlikda* yoziladi: nakladnoyda «bir kilo, 90 000» deb turadi, hech kim grammning narxini yozib yurmaydi. Grammga o'tkazishni texkarta o'zi qiladi.",
      },
      {
        warn: "Uch birlik bor va tamom: *kilo, litr, dona*. «Bog'lam» yoki «quti» kabi erkin matn yozib bo'lmaydi — hisoblab bo'lmaydigan karta va hisoblangandek ko'rinadigan tannarx shundan chiqadi.",
      },
      {
        list: [
          "*Minimal qoldiq* — bundan pastga tushsa «tugayapti» deb belgilanadi. *Nol — «ogohlantirma»*, «nolda ogohlantir» emas.",
          "*Ombor* — bir nechta ombor bo'lsa, shu filial masalliqni qayerda saqlashi. Bo'sh — umumiy ombor.",
          "*Izoh* — qayerdan olinadi, qaysi sort.",
        ],
      },
      {
        tip: "Narxni qo'lda tuzatish *bugundan* kuchga kiradi, kirim esa o'z sanasi bilan. Ya'ni kechikkan nakladnoy tarixga o'z kuni bilan tushadi, lekin bugungi narxni o'zgartirmaydi.",
      },
      { see: ["tech-cards", "purchases", "shopping-list"] },
    ],
  },
  {
    slug: "tech-cards",
    section: "stock",
    title: "Texkarta: taomga karta tuzish",
    lead: "Bir porsiyaga nima ketadi — shundan tannarx va ombordan sarf chiqadi.",
    keys: ["texkarta", "karta", "tannarx", "kalkulyatsiya", "retsept"],
    body: [
      {
        p: "`Ombor` → `Texkartalar` → `Taomlar`. Har taom uchun masalliqlar va miqdorlari yoziladi.",
      },
      {
        fig: "tech-cards",
        notes: {
          tabs: "Ikki bo'lim: zagotovkalar (oshxona o'zi tayyorlaydigan yarim tayyor) va taom kartalari.",
          dishes: "Menyudagi taomlarning kartalari shu yerda.",
          add: "Yangi zagotovka qo'shish.",
          batch: "Bir partiya qanchaga tushadi — qozonga qarab tekshirish mumkin bo'lgan raqam.",
          rate: "Shundan chiqadigan bir gramm narxi — taomlarda aynan shu ishlatiladi.",
        },
      },
      { h: "Karta tuzish" },
      {
        steps: [
          "`Taomlar` bo'limiga o'ting va taomni toping. «Kartasi yo'q» filtri qaysi taomlar qolganini ko'rsatadi.",
          "`Qo'shish` bosing — masalliqni tanlab miqdorini yozing.",
          "Miqdor *brutto* bo'ladi: ombordan chiqadigan og'irlik, tarelkadagi emas. Bir kilo kartoshka po'stlog'i bilan bir kilo turadi.",
          "Yozayotganingizda tannarx va marja darhol ko'rinadi.",
          "Saqlang.",
        ],
      },
      {
        warn: "Karta tuzilgan bo'lsa, menyuda qo'lda yozilgan tannarx *ishlatilmaydi*. Bitta raqamning ikki manbasi bo'lishi mumkin emas — karta har o'qishda bugungi masalliq narxlari bo'yicha qayta hisoblanadi.",
      },
      {
        warn: "*To'liq bo'lmagan karta taomni umuman narxlamaydi*. Kartadagi masalliqlardan biri o'chirilgan bo'lsa, tannarx nol bo'lib qoladi — arzon emas. Aks holda o'chirilgan masalliq taomni «foydaliroq» ko'rsatardi, va marja ekranida bu yaxshi xabar bo'lib ko'rinadi.",
      },
      {
        p: "Ishlatilayotgan masalliqni o'chirib bo'lmaydi: tizim uni qaysi taomlar ushlab turganini aytadi.",
      },
      { fig: "tech-cards-dishes" },
      { see: ["prep-cards", "ingredients", "menu-item", "stock-balances"] },
    ],
  },
  {
    slug: "prep-cards",
    section: "stock",
    title: "Zagotovka: sous, xamir, qiyma",
    lead: "Oshxona o'zi tayyorlaydigan narsa — bir marta yoziladi, hamma taomda grammlab ishlatiladi.",
    keys: ["zagotovka", "yarim tayyor", "sous", "xamir", "qiyma", "polufabrikat"],
    body: [
      {
        p: "Sous, bulon, xamir, sushi guruchi — bular menyuda yo'q, lekin ombordan chiqadi. Ular `Ombor` → `Texkartalar` → `Zagotovkalar` da tuziladi.",
      },
      {
        p: "*Bu bo'limning butun ma'nosi shu*: olti xil sousi va qirq taomi bor oshxona aks holda o'sha pomidorni yetti joyda qayta yozadi — va yetti nusxa bir oy ichida bir-biriga mos kelmay qoladi.",
      },
      { h: "Zagotovka tuzish" },
      {
        steps: [
          "`Yangi zagotovka` — nom bering: «Somsa qiymasi», «Oq sous».",
          "O'lchov birligini tanlang: kilo (grammda ishlatiladi), litr (millilitrda) yoki dona.",
          "Masalliqlarni va *bir partiyaga* ketadigan miqdorlarni yozing.",
          "*Chiqimni* yozing: bir partiyadan qancha chiqadi.",
          "Saqlang. Endi u taom kartalarida oddiy masalliq kabi ko'rinadi.",
        ],
      },
      {
        fig: "tech-card-prep",
        notes: {},
      },
      {
        warn: "*Chiqim — halollik joyi.* 3 kg pomidordan 2 kg sous chiqsa, chiqim 2000 g, 3000 emas. Katta yozilgan chiqim shu sousli har bir taomni arzon ko'rsatadi.",
      },
      {
        warn: "Chiqim yozilmasa zagotovkaning bir grammi *hech qanchaga tushmaydi*, ya'ni uni ishlatgan har bir taom narxlanmagan bo'lib qoladi.",
      },
      {
        p: "Zagotovkaning *narxi yozilmaydi*: uning narxi — bir partiya qanchaga tushsa, chiqimga bo'lingani. Shuning uchun narx maydoni ekranda ham ko'rsatilmaydi.",
      },
      {
        tip: "Zagotovka ichida boshqa zagotovka bo'lishi mumkin (xamir → somsa qiymasi bilan birga «tayyor somsa»). Hisob bosqichma-bosqich yuritiladi.",
      },
      { h: "Zagotovka sanaladimi?" },
      {
        p: "Odatda *yo'q*: hech kim «sous»ni sanamaydi, pomidorni sanaydi, va sous ishlatgan taom pomidor ishlatgan deb o'qiladi. Markaziy oshxonasi bor zanjir uchun «Partiya bilan tayyorlanadi» belgisi bor — u yoqilsa zagotovka javondagi haqiqiy narsa bo'lib sanaladi va filiallarga ko'chiriladi.",
      },
      { see: ["tech-cards", "production", "stocktake"] },
    ],
  },
  {
    slug: "purchases",
    section: "stock",
    title: "Kirim: nakladnoy kiritish",
    lead: "Nima keldi, kimdan va qanchaga. Narxlar shu yerdan menyuning tannarxiga o'tadi.",
    keys: ["kirim", "nakladnoy", "prixod", "yetkazib berish", "narx"],
    body: [
      {
        p: "`Ombor` → `Kirim` → `Yangi`. Sana, yetkazib beruvchi va qatorlar: masalliq, miqdor, birlik narxi.",
      },
      {
        list: [
          "Nakladnoyning *o'z jami* ustun turadi: eshik oldida berilgan chegirma hech bir qatorda yo'q.",
          "Narx o'zgargan bo'lsa, masalliqning narx tarixiga *nakladnoy sanasi bilan* yoziladi.",
          "Narx o'zgarmagan bo'lsa tarixga yozuv qo'shilmaydi — nomni tuzatgani uchun «narx o'zgardi» degan yozuv qolmasligi kerak.",
        ],
      },
      { h: "To'lov va qarz" },
      {
        p: "Har nakladnoyda «to'landi» belgisi bor. To'lanmagani yetkazib beruvchi qarzi bo'lib ko'rinadi (`Ombor` → `Yetkazib beruvchilar`).",
      },
      {
        warn: "Kirimni *tahrirlash* va *o'chirish* boshqacha ishlaydi. Tahrirlaganda o'sha nakladnoy yozgan narxlar olib tashlanib qaytadan yoziladi. O'chirish esa narxlarni qaytarmaydi — u faqat «bu qator bu yerda bo'lmasligi kerak» deydi, va ovqat allaqachon o'sha narx bilan narxlangan bo'lishi mumkin.",
      },
      {
        tip: "Qo'lda kiritilgan narxlarga hech qachon tegilmaydi. Ular qaysi nakladnoydan ekanini ayta olmaydi, taxmin qilish esa ataylab qilingan tuzatishni jimgina o'chirardi.",
      },
      { fig: "purchases" },
      { see: ["suppliers", "ingredients", "stock-balances"] },
    ],
  },
  {
    slug: "suppliers",
    section: "stock",
    title: "Yetkazib beruvchilar va qarz",
    lead: "Kimdan olasiz, kimga qancha qarzdorsiz.",
    keys: ["yetkazib beruvchi", "postavshik", "qarz", "to'lov"],
    body: [
      {
        p: "`Ombor` → `Yetkazib beruvchilar`. Ro'yxatda har biriga: davr bo'yicha qancha olingani va *qancha qarz qolgani*.",
      },
      {
        warn: "Qarz *davrga bog'liq emas*: martdagi to'lanmagan nakladnoy may oyida ham qarz bo'lib turadi. Davr filtri faqat xaridni cheklaydi.",
      },
      {
        p: "Yetkazib beruvchini ko'rsatish *ixtiyoriy*: bozorga chiqishning yetkazib beruvchisi yo'q, majburiy qilish esa kirim umuman yozilmasligiga olib keladi.",
      },
      {
        tip: "Yetkazib beruvchini qayta nomlasangiz eski nakladnoylar o'zgarmaydi — ularda nomning muzlatilgan nusxasi saqlanadi.",
      },
      { fig: "suppliers" },
      { see: ["purchases", "shopping-list"] },
    ],
  },
  {
    slug: "writeoffs",
    section: "stock",
    title: "Chiqim: buzilgan, to'kilgan, xodimlar ovqati",
    lead: "Sotilmasdan ketgan mahsulot. Sabab majburiy.",
    keys: ["chiqim", "spisaniye", "buzildi", "to'kildi", "isrof"],
    body: [
      {
        p: "`Ombor` → `Chiqim` → masalliq, miqdor va *sabab*. Sababsiz yozuv bir oydan keyin hech kimga hech nima aytmaydi.",
      },
      {
        p: "Chiqim *o'sha kunning narxida* baholanadi va muzlatiladi — keyingi narx o'zgarishi o'tgan oyning isrof raqamini qayta yozmaydi.",
      },
      {
        tip: "Zagotovka chiqimi uning kartasi bo'yicha baholanadi. Buzilgan bir qozon sousni nolga yozib qo'ymaslik uchun.",
      },
      { fig: "writeoffs" },
      { see: ["stock-balances", "reports-sales"] },
    ],
  },
  {
    slug: "transfers",
    section: "stock",
    title: "Ko'chirish: filialdan filialga",
    lead: "Mahsulot bir omborda kamayadi, ikkinchisida ko'payadi — na chiqim, na kirim.",
    keys: ["ko'chirish", "peremesheniye", "filial", "transfer"],
    body: [
      {
        p: "`Ombor` → `Ko'chirish`. Qaysi ombordan qaysi omborga, qaysi masalliq va qancha.",
      },
      {
        warn: "Ko'chirish *chiqim ham, kirim ham emas*. Chiqim deb yozilsa isrof hisobotiga hech kim isrof qilmagan sabab qo'shiladi; kirim deb yozilsa xarid narxi tarixga tushib o'sha masalliqli har bir taomni qimmatlashtiradi.",
      },
      {
        warn: "*Birliklar mos kelishi shart*: kilogrammni litrga ko'chirish ikkala balansni ham izchil va ikkalasini ham noto'g'ri qoldiradi — buni keyin hech nima ko'rmaydi.",
      },
      {
        p: "Jami «ko'chirilgan» deb ataladi va moliyaviy hisobotning xarajatlariga kirmaydi: qiymat tashiladi, yaratilmaydi.",
      },
      { fig: "transfers" },
      { see: ["production", "stock-balances"] },
    ],
  },
  {
    slug: "production",
    section: "stock",
    title: "Markaziy oshxona: partiya tayyorlash",
    lead: "Tsex dushanba kuni 40 kg sous qiladi va uchta filialga tarqatadi.",
    keys: ["tsex", "ishlab chiqarish", "partiya", "markaziy oshxona"],
    body: [
      {
        p: "Bu bo'lim faqat *markaziy oshxonasi bor* restoranlar uchun. Bitta oshxonali restoran bu sahifani bir marta ochadi, o'ziga tegishli emasligini o'qiydi va qaytmaydi.",
      },
      { h: "Ikki qadam, va bittasi yolg'iz ishlamaydi" },
      {
        steps: [
          "Zagotovkada «Partiya bilan tayyorlanadi» belgisini yoqing. Endi u javondagi haqiqiy narsa bo'lib sanaladi va ko'chiriladi.",
          "Omborni «ishlab chiqarish» turiga o'tkazing — partiya faqat shu omborda tayyorlanadi.",
          "`Ombor` → `Ishlab chiqarish` → partiyani yozing: nima, qancha chiqdi.",
        ],
      },
      {
        warn: "Belgi bor, hujjat yo'q → javon hech qachon to'lmaydi va qoldiq manfiy chiqadi. Hujjat bor, belgi yo'q → masalliqlar *ikki marta* ayriladi: bir marta bu yerda, bir marta taom sotilganda. Ikkinchisi haftalar keyin sanashda chiqadi va sanagan odamning aybi bo'lib ko'rinadi.",
      },
      {
        p: "Partiya nimani olgani hujjatga *muzlatiladi*: karta o'zgaradi, martdagi partiya esa martda nimani olgan bo'lsa shuni olgan.",
      },
      { fig: "production" },
      { see: ["prep-cards", "transfers"] },
    ],
  },
  {
    slug: "stocktake",
    section: "stock",
    title: "Inventarizatsiya: sanash",
    lead: "Javonda nima borligini yozasiz — va shu paytdan «bo'lishi kerak» hisoblana boshlaydi.",
    keys: ["inventarizatsiya", "sanash", "revizya", "kamomad"],
    body: [
      {
        p: "`Ombor` → `Inventarizatsiya`. Masalliqlar ro'yxati chiqadi, har biriga sanalgan miqdorni yozasiz.",
      },
      {
        p: "«Kutilgan» raqam *siz son yozmaguncha ko'rsatilmaydi*. Yonida «9.4 bo'lishi kerak» yozuvi turgan bo'sh katak — katakka 9.4 yoziladigan varaq.",
      },
      {
        warn: "Farq bo'lsa *izoh majburiy*. Farqni jimgina saqlash — kamomadni uni qilgan bo'lishi mumkin bo'lgan odam tomonidan o'chirilishi.",
      },
      { h: "Telefonda sanash" },
      {
        p: "Ishchi `saytingiz.uz/staff/stock` dan kirib omborda turib sanay oladi. Sanash omborda bo'lib ofisda yozilsa, ikki marta yozilgan raqam ikkinchisida xato bo'ladi — va xato aynan *farq* ustuniga tushib kamomaddan farq qilmay qoladi.",
      },
      {
        warn: "Bu ruxsat *alohida beriladi* (`Xodimlar` → xodim → «Ombor»). Sanash keyingi har bir kamomad o'lchanadigan bazani yozadi, ya'ni saqlangan sanash undan oldin yo'qolgan hamma narsani jimgina kechiradi.",
      },
      {
        p: "*Zagotovkalar sanalmaydi*: ular ertalab pishirilgan qozon, va nimadan qilingani allaqachon masallig'ining sanog'ida. Ikkalasini sanash pomidorni ikki marta ayirardi.",
      },
      { fig: "stocktake" },
      { see: ["stock-balances", "staff-roles"] },
    ],
  },
  {
    slug: "stock-balances",
    section: "stock",
    title: "Qoldiqlar va «tugayapti»",
    lead: "«Bo'lishi kerak» qayerdan chiqadi va nimani anglatmaydi.",
    keys: ["qoldiq", "ostatok", "tugayapti", "balans", "manfiy"],
    body: [
      {
        p: "`Ombor` → `Qoldiqlar`. Har masalliq uchun: oxirgi sanash + kirimlar − texkarta bo'yicha sarf − chiqimlar.",
      },
      {
        warn: "Bu *taxmin*, o'lchov emas. U oshxona kartadan qanchalik chetga chiqsa, shunchalik chetga chiqadi — va oxirgi sanash qanchalik uzoq bo'lsa, shunchalik ko'p. Ekran oxirgi sanash sanasini shuning uchun yozadi.",
      },
      { h: "Manfiy qoldiq" },
      {
        p: "Odatda ikki sababdan: *kirim kiritilmagan* yoki *texkartadagi miqdor haqiqiydan katta*. Avval o'sha masalliqni sanang, keyin kartani tekshiring.",
      },
      {
        p: "*Tugayapti* — minimal qoldiq qo'yilgan masalliqda «bo'lishi kerak» undan pastga tushsa. Nol — «ogohlantirma», va bu yerda qo'ng'iroq yo'q: bu «keyingi buyurtmada yodda tut» degan gap.",
      },
      { fig: "stock" },
      { see: ["stocktake", "shopping-list", "tech-cards"] },
    ],
  },
  {
    slug: "shopping-list",
    section: "stock",
    title: "Xarid ro'yxati",
    lead: "Nima tugayapti va kimdan olinadi — bitta sahifada.",
    keys: ["xarid", "zakupka", "ro'yxat", "sotib olish"],
    body: [
      {
        p: "`Ombor` → `Xarid ro'yxati`. Minimal qoldig'idan pastga tushgan masalliqlar, *oxirgi yetkazib beruvchi bo'yicha guruhlangan*.",
      },
      {
        p: "Eng ko'p olingan emas, aynan *oxirgi*: qassobini almashtirgan restoranga yangisini aytish kerak, yillik sanoq esa eskisini javob eng noto'g'ri bo'lgan davr davomida nomlab turadi.",
      },
      {
        tip: "Manfiy qoldiq bo'sh javon deb hisoblanadi. Manfiy raqamning yarmi o'lchov xatosi, va unga qarab buyurtma berish raqamlar eng ishonchsiz paytda ikki barobar oldirardi.",
      },
      { fig: "shopping" },
      { see: ["stock-balances", "purchases"] },
    ],
  },
  {
    slug: "stock-stop",
    section: "stock",
    title: "Ombor bo'yicha stop list",
    lead: "Masalliqlari tugagan taomni avtomatik to'xtatish — va nega u standart holatda o'chiq.",
    keys: ["stop list", "ombor", "avtomatik", "tugadi"],
    body: [
      {
        p: "`Menyu` → `Stop list` → «Ombor bo'yicha to'xtatish». Yoqilsa, masalliqlari yetmagan taom kassada va saytda sotilmaydi.",
      },
      {
        warn: "*Standart holatda o'chiq va bu ataylab.* Sotuvni rad etish tizim qila oladigan eng qimmat ish, orqasidagi qoldiq esa taxmin: seshanbaning nakladnoyini kiritmagan restoranning kassasi javonda turgan ovqatni rad etardi.",
      },
      {
        p: "*Sanalmagan ombor hech nimani to'xtatmaydi.* Nolning ikki ma'nosi bor — «yo'q» va «hech kim aytmagan» — va faqat birinchisi rad etish uchun sabab. Bu xususiyat yoqilgan kuni menyuni bo'shatmaydigan yagona qo'riq.",
      },
      {
        p: "«Bitta porsiyaga yetmaydi» ham «tugadi» bilan bir gap: 200 g go'sht va 500 g'lik karta — pishirib bo'lmaydigan taom.",
      },
      { see: ["stop-list", "stock-balances"] },
    ],
  },

  // ═══════════════════════════════════ Xodimlar
  {
    slug: "staff-add",
    section: "team",
    title: "Ishchi qo'shish",
    lead: "Kassir, ofitsiant, oshpaz — har biriga o'z hisobi va o'z ilovasi.",
    keys: ["ishchi", "xodim", "qo'shish", "kassir", "ofitsiant", "oshpaz", "login"],
    body: [
      {
        p: "`Pul va jamoa` → `Xodimlar` → `Qo'shish`.",
      },
      {
        fig: "staff",
        notes: {
          add: "Yangi xodim shu yerdan qo'shiladi.",
        },
      },
      { h: "To'ldiriladigan maydonlar" },
      {
        table: {
          head: ["Maydon", "Izoh"],
          rows: [
            ["*Ism*", "Kalendarda va hisobotlarda shu ko'rinadi."],
            ["*Lavozim*", "Erkin matn («oshpaz», «ofitsiant»). *Bu ruxsat emas* — tizim uni hech qayerda o'qimaydi."],
            ["*Filial*", "Qaysi binoda ishlaydi. Davomat va ombor shu filialga yoziladi."],
            ["*Login va PIN*", "Kassaga va ishchi ilovasiga kirish uchun."],
            ["*Ish haqi*", "Oylik yoki kunlik/soatlik stavka. Oylik oyning grafik kunlariga bo'linadi."],
            ["*Ruxsatlar*", "Oshxona ekrani, ombor sanash — har biri alohida yoqiladi."],
          ],
        },
      },
      {
        warn: "*Lavozim ruxsat emas.* «Oshpaz» deb yozilgani oshxona ekranini ochmaydi — uni alohida belgilash kerak. Lavozimni ruxsat deb qabul qilish kimning imlosi mos kelsa o'shanga kalit berish bo'lardi.",
      },
      {
        p: "Ishchi `saytingiz.uz/staff` dan kiradi: smena ochish/yopish, o'z kalendari va (ruxsat berilgan bo'lsa) oshxona ekrani yoki ombor sanash.",
      },
      {
        tip: "Ishdan bo'shatilgan odamni *o'chirmang*, «faol emas» qiling. Smenalari va oylik tarixi joyida qoladi, kirish esa darhol yopiladi.",
      },
      { see: ["staff-roles", "staff-attendance", "payroll"] },
    ],
  },
  {
    slug: "staff-roles",
    section: "team",
    title: "Rollar va ruxsatlar",
    lead: "Kim qaysi ekranni ochadi. Har ruxsat alohida beriladi, lavozimdan kelib chiqmaydi.",
    keys: ["rol", "ruxsat", "dostup", "permission", "kim ko'radi"],
    body: [
      {
        p: "Tizimda to'rt xil hisob bor va ular butunlay boshqa ilovalarga kiradi.",
      },
      {
        table: {
          head: ["Hisob", "Qayerga kiradi"],
          rows: [
            ["*Ega / menejer*", "Panel (`/admin`)"],
            ["*Ishchi*", "Kassa, zal, oshxona ekrani, ombor sanash (`/staff`)"],
            ["*Kuryer*", "Kuryer ilovasi (`/kuryer`)"],
            ["*Mijoz*", "Sayt va Telegram mini app"],
          ],
        },
      },
      { h: "Ishchi ruxsatlari" },
      {
        p: "`Pul va jamoa` → `Rollar` da rol yasalади va unga ruxsatlar beriladi; keyin rol xodimga biriktiriladi. Ruxsatlar alohida-alohida:",
      },
      {
        list: [
          "*Kassa* — sotuv, chek yopish.",
          "*Zal* — stollar, chekni bo'lish va birlashtirish.",
          "*Oshxona ekrani (KDS)* — chekni ko'rish va «Tayyor» bosish.",
          "*Ombor sanash* — inventarizatsiya varag'ini to'ldirish.",
          "*Chegirma berish* — kassada chegirma qo'yish.",
          "*Chekni bekor qilish* — void va qaytarish.",
        ],
      },
      {
        warn: "Oshxona ekrani va ombor sanash *standart holatda o'chiq* — bu tizimning odatdagi «bo'sh qiymat = bugungi xatti-harakat» qoidasidan ataylab chetga chiqish. Umumiy planshetda hamma kira olsa, ofitsiant «Tayyor» bosib chekni oshxona ekranidan yo'qota oladi, panelga esa «oshxona pishirdi» deb yoziladi.",
      },
      {
        tip: "Kassada chegirma va bekor qilishni faqat smena boshlig'iga bering. Bu ikki tugma — kassadagi eng qimmat ikki tugma, va ikkalasi ham jurnalga yoziladi.",
      },
      { fig: "roles" },
      { see: ["staff-add", "kds", "admins", "admin-log"] },
    ],
  },
  {
    slug: "staff-attendance",
    section: "team",
    title: "Davomat: kelish va ketish",
    lead: "Ishchi telefonidan «keldim» bosadi — joylashuv bilan tekshiriladi.",
    keys: ["davomat", "smena", "keldi", "ketdi", "tabel", "geolokatsiya"],
    body: [
      {
        p: "Ishchi `/staff` sahifasidan `Ishga keldim` bosadi. Telefonning joylashuvi tekshiriladi: restorandan 50 metr radiusda bo'lishi kerak.",
      },
      {
        p: "Radius `Sozlamalar` → `Filial` da o'zgartiriladi. *50 metr — ataylab, 5 emas*: telefon GPS'i ochiq havoda 10–30 metr, bino ichida yomonroq. Xato chegarasidan kichik radius firibgarni ushlamaydi, halol ishchini esa eshik oldida qoldiradi.",
      },
      { h: "Kalendar" },
      {
        p: "`Pul va jamoa` → `Xodimlar` → xodim. Har kun rangli: grafik bo'yicha, kam ishlangan, ko'p ishlangan yoki chiqmagan. Bu holat hech qayerda saqlanmaydi — u *grafik* va *smenalar* ni taqqoslash.",
      },
      {
        list: [
          "Smena *boshlangan kunga* yoziladi: 00:40 da chiqqan oshpaz seshanbani yopadi.",
          "To'ldirilmagan grafik kuni dam olish deb o'qiladi, hech kim ayblanmaydi.",
          "Ikki marta «keldim» qabul qilinmaydi — ikkita ochiq smena kalendardagi har soatni ikkilantirardi.",
        ],
      },
      {
        tip: "Qo'lda tuzatish *ataylab qoldirilgan*: telefon o'chadi, chiqish unutiladi. Tuzatib bo'lmaydigan davomat tizimidan ikkinchi haftada voz kechiladi. Har tuzatish kim qilgani bilan imzolanadi.",
      },
      { see: ["payroll", "staff-qr"] },
    ],
  },
  {
    slug: "staff-qr",
    section: "team",
    title: "QR bilan ishga kirish",
    lead: "Kirishdagi ekran QR ko'rsatadi, ishchi skanerlaydi — telefon GPS'iga tayanmasdan.",
    keys: ["qr", "ishga kirish", "kiosk", "davomat"],
    body: [
      {
        p: "Filialda planshet yoki monitor `/kiosk` rejimida QR ko'rsatib turadi. Ishchi telefonidan skanerlaydi va smena ochiladi.",
      },
      {
        warn: "Kod *qog'ozda emas, ekranda* turadi va har 30 soniyada o'zgaradi. Bosma QR — devorga yozilgan parol: uni bir marta rasmga olgan odam uyidan turib bir yil «ishga kirdim» bosib yuraveradi.",
      },
      { see: ["staff-attendance", "kiosk"] },
    ],
  },
  {
    slug: "payroll",
    section: "team",
    title: "Oylik va to'lovlar",
    lead: "Kim qancha ishlagan, qancha olishi kerak, qancha berilgan.",
    keys: ["oylik", "maosh", "payroll", "to'lov", "avans"],
    body: [
      {
        p: "`Pul va jamoa` → `Oylik`. Har xodim uchun davr bo'yicha: ishlangan kunlar, hisoblangan summa, berilgan summa va qoldiq.",
      },
      {
        p: "Oylik maosh *oyning grafik kunlariga* bo'linadi va chiqilgan kun uchun beriladi — o'rtada ishga kirgan yoki bir hafta kelmagan odam ham to'g'ri chiqadi.",
      },
      {
        p: "To'lov — *yozuv*, hisoblagich emas: qaysi davrni yopgani va kim bergani bilan saqlanadi. Aprel boshlangandan keyin ham «martda qancha to'ladik» javobsiz qolmaydi.",
      },
      {
        tip: "Avansni ham shu yerga yozing. Aks holda oy oxirida ikki manbadan hisoblanadi va ikkalasi mos kelmaydi.",
      },
      { fig: "payroll" },
      { see: ["staff-attendance", "finance"] },
    ],
  },
  {
    slug: "admin-log",
    section: "team",
    title: "Amallar jurnali",
    lead: "Kim nima o'zgartirgan — narx, buyurtma, texkarta, inventarizatsiya.",
    keys: ["jurnal", "log", "tarix", "kim o'zgartirdi", "audit"],
    body: [
      {
        p: "`Pul va jamoa` → `Jurnal`. Har yozuvda: kim, qachon, nima qilgan va *nima o'zgargan*.",
      },
      {
        p: "Texkarta o'zgarishi alohida batafsil yoziladi: qaysi masalliq, qanchadan qanchaga. «Retsept o'zgardi» degan yozuv rost, jurnalda turadi va hech nimaga javob bermaydi.",
      },
      {
        warn: "Kartadagi normani oshirib qo'yish — pulni *arifmetik ko'rinmas* qiladigan yagona yo'l: taom 200 g go'sht deydi, oshxona 150 g qo'yadi, har porsiyadan 50 g qoladi, va sanash hech nima topmaydi чунки karta uni allaqachon ketgan deb kutadi. Ko'rinadigan yagona payt — karta tahrirlangan payt.",
      },
      {
        tip: "Norma 20% dan ko'proq oshsa panelda ogohlantirish chiqadi. Bu chegara baland: kartani tuzatish normal ish, 11% tuzatish uchun xabar yubormaslik kerak.",
      },
      { fig: "logs" },
      { see: ["tech-cards", "admins", "security"] },
    ],
  },

  // ═══════════════════════════════════ Mijozlar
  {
    slug: "crm",
    section: "customers",
    title: "Mijoz kartochkasi",
    lead: "Telefon raqami bo'yicha butun tarix: buyurtmalar, manzillar, ballar, qarz, fikrlar.",
    keys: ["mijoz", "crm", "baza", "tarix", "telefon"],
    body: [
      {
        p: "`Mijozlar` → `Baza`. Qidiruv telefon raqami yoki ism bo'yicha.",
      },
      {
        list: [
          "Buyurtmalar tarixi va o'rtacha chek.",
          "Saqlangan manzillar.",
          "Ballar qoldig'i va harakati.",
          "Qarz bo'lsa — summa va «qaytardi» tugmasi.",
          "Fikrlari va shikoyatlari.",
        ],
      },
      {
        p: "Menejer *o'z filialining* buyurtmalar tarixini ko'radi; mijoz profilining o'zi kompaniyaniki, chunki baza umumiy.",
      },
      { fig: "users" },
      { see: ["loyalty", "segments", "till-payment"] },
    ],
  },
  {
    slug: "loyalty",
    section: "customers",
    title: "Ballar (keshbek)",
    lead: "Har buyurtmadan foiz qaytadi, keyingi buyurtmada ishlatiladi.",
    keys: ["ball", "keshbek", "loyalty", "bonus", "chegirma"],
    body: [
      {
        p: "`Sozlamalar` → `Loyalty`. Yoqiladi, foiz qo'yiladi (masalan 5%), va bir buyurtmada ballning necha foizini ishlatish mumkinligi cheklanadi.",
      },
      {
        list: [
          "Ball *yetkazilgan* buyurtmadan keyin beriladi, buyurtma berilganda emas.",
          "Bekor qilingan buyurtma ball bermaydi.",
          "Ball bilan to'langan qism hisobotlarda «ma'lumot» bo'lib turadi, xarajat sifatida emas.",
        ],
      },
      {
        warn: "Ball *xarajat emas*: pul chiqmagan, u umuman kelmagan. Tushum qatori allaqachon undan tozalangan, ya'ni uni yana ayirish kampaniyani ikki marta hisoblash bo'lardi.",
      },
      { see: ["promotions", "crm", "finance"] },
    ],
  },
  {
    slug: "segments",
    section: "customers",
    title: "Segmentlar va kampaniyalar",
    lead: "«Bir oydan beri kelmaganlar» ga xabar yuborish.",
    keys: ["segment", "kampaniya", "rassilka", "sms yuborish", "rfm"],
    body: [
      {
        p: "`Mijozlar` → `Kampaniyalar`. Avval segment tanlanadi, keyin kanal va matn.",
      },
      {
        table: {
          head: ["Segment", "Kim"],
          rows: [
            ["*Yangi*", "Bir marta buyurtma qilganlar"],
            ["*Doimiy*", "Muntazam keladiganlar"],
            ["*Uxlab qolgan*", "Ilgari kelgan, endi kelmayotganlar"],
            ["*Eng qimmatlilar*", "Eng ko'p pul qoldirganlar"],
          ],
        },
      },
      {
        p: "Segmentlar *bazangizga nisbatan* hisoblanadi: «doimiy» degani sizning restoraningizdagi doimiy, umumiy me'yor emas.",
      },
      {
        list: [
          "*SMS* — pul turadi, bo'lak soni ekranda ko'rsatiladi.",
          "*Telegram* — bot orqali, tekin, lekin faqat botga yozganlar oladi.",
          "*Push* — brauzer bildirishnomasi, tekin.",
        ],
      },
      {
        warn: "SMS matnida kirill harfi yoki `oʻ` bo'lsa bir SMS 160 belgidan *70 belgiga* tushadi, ya'ni narx ikki-uch barobar oshadi. Lotin o'zbekcha yozing.",
      },
      {
        tip: "Kampaniyani faqat *ega* yubora oladi. Bu bitta tugma butun mijoz bazasini bezovta qila oladi va pul sarflaydi.",
      },
      { fig: "campaigns" },
      { see: ["sms", "push", "telegram-bot"] },
    ],
  },
  {
    slug: "push",
    section: "customers",
    title: "Brauzer bildirishnomalari (push)",
    lead: "Tekin kanal: mijoz saytga kirmasa ham xabar oladi.",
    keys: ["push", "bildirishnoma", "notification", "brauzer"],
    body: [
      {
        p: "Mijoz saytda ruxsat bergan bo'lsa, unga buyurtma holati va kampaniyalar push bo'lib boradi. SMS'dan farqi — *tekin*.",
      },
      {
        p: "Hech nima sozlash shart emas: kalitlar birinchi ishlatishda o'zi yaratiladi va *hech qachon almashtirilmaydi* — har obuna o'zi yaratilgan kalitga bog'langan.",
      },
      {
        warn: "iPhone'da push faqat sayt «Bosh ekranga qo'shilgan» bo'lsa ishlaydi. Bu Apple cheklovi, sozlama emas.",
      },
      { see: ["segments"] },
    ],
  },
  {
    slug: "feedback",
    section: "customers",
    title: "Fikrlar va shikoyatlar",
    lead: "Buyurtmadan keyin so'raladigan baho va u bilan nima qilish.",
    keys: ["fikr", "shikoyat", "baho", "sharh"],
    body: [
      {
        p: "`Mijozlar` → `Fikrlar`. Har fikrda buyurtma raqami bor — ya'ni qaysi kecha, qaysi taom va kim yetkazgani ko'rinadi.",
      },
      {
        tip: "Past baho kelganda mijoz kartochkasini oching: bu birinchi shikoyatmi yoki uchinchisimi — javob shunga qarab boshqacha bo'ladi.",
      },
      { fig: "feedback" },
      { see: ["reviews", "crm"] },
    ],
  },
  {
    slug: "calls",
    section: "customers",
    title: "Call-markaz",
    lead: "Telefonda buyurtma qabul qilish va qo'ng'iroqlar jurnali.",
    keys: ["qo'ng'iroq", "telefon", "call", "operator"],
    body: [
      {
        p: "`Mijozlar` → `Call-markaz`. Operator raqamni yozadi, tizim mijozni topadi va butun tarixini ko'rsatadi — keyin buyurtma shu yerdan kiritiladi.",
      },
      {
        p: "ATS ulangan bo'lsa (onlinePBX), kiruvchi qo'ng'iroqda mijoz kartochkasi *o'zi ochiladi*, jurnalga raqam, davomiylik, yozuv va kim javob bergani avtomatik tushadi. Operator faqat natija va izoh yozadi.",
      },
      {
        tip: "Operator boshqa qo'ng'iroqni yozayotgan bo'lsa ekran egallab olinmaydi: yarim yozilgan izohni yo'qotish kartochka ochilmaganidan yomonroq.",
      },
      { fig: "calls" },
      { see: ["pbx", "crm"] },
    ],
  },

  // ═══════════════════════════════════ Integratsiyalar
  {
    slug: "integrations-overview",
    section: "integrations",
    title: "Integratsiyalar: nima ulanadi",
    lead: "Nima shart, nima ixtiyoriy va nimasiz sayt ishlamaydi.",
    keys: ["integratsiya", "ulash", "sozlash", "api"],
    body: [
      {
        table: {
          head: ["Nima", "Shartmi", "Nima uchun"],
          rows: [
            ["*SMS*", "*Ha*", "Mijoz kod bilan kiradi. Bo'lmasa hech kim ro'yxatdan o'ta olmaydi."],
            ["*Xarita*", "*Ha*", "Yetkazish manzilini tanlash uchun."],
            ["*Onlayn to'lov*", "Yo'q", "Naqd har doim ishlaydi. To'lov tizimi qo'shimcha kanal."],
            ["*Telegram bot*", "Yo'q", "Ikkinchi kanal: mini app va bildirishnomalar."],
            ["*Kassa tizimi (POS)*", "Yo'q", "Sizda allaqachon iiko/Poster bo'lsa."],
            ["*ATS*", "Yo'q", "Telefonda ko'p buyurtma olsangiz."],
            ["*Fiskal kassa*", "Qonun talab qilsa", "Chekni davlat tizimiga yuborish."],
            ["*Markirovka*", "Ichimlik sotsangiz", "DataMatrix kodini chekka qo'shish."],
          ],
        },
      },
      {
        warn: "Barcha kalitlar `Sozlamalar` da saqlanadi va *hech qachon qaytarilmaydi*: ekranda faqat «kalit saqlangan» degan belgi turadi. Maydonni bo'sh qoldirib saqlasangiz eski kalit joyida qoladi — bitta maydonni tuzatayotganda to'lovni jimgina o'chirib qo'ymaslik uchun.",
      },
      { fig: "settings" },
      { see: ["sms", "payments", "pos", "telegram-bot"] },
    ],
  },
  {
    slug: "sms",
    section: "integrations",
    title: "SMS provayderini ulash",
    lead: "Mijoz kirishi, bron tasdig'i va kampaniyalar shundan ketadi.",
    keys: ["sms", "eskiz", "play mobile", "getsms", "kod kelmayapti"],
    body: [
      {
        p: "`Sozlamalar` → `SMS`. To'rtta xizmat: *Eskiz*, *Play Mobile*, *getsms.uz*, *OneSignal*. Shartnomani siz tuzasiz va jo'natuvchi nomini moderatsiyadan o'tkazasiz.",
      },
      {
        steps: [
          "Provayderni tanlang va kalitlarini kiriting.",
          "Jo'natuvchi nomini yozing (moderatsiyadan o'tgan nom).",
          "Kod xabarining matnini tekshiring — ichida `{code}` bo'lishi *shart*.",
          "`Sinov SMS` yuboring va telefoningizga kelganini ko'ring.",
        ],
      },
      {
        warn: "`Sinov SMS` — bu sahifaning asosiy tugmasi. Kalitlar to'g'ri ko'ringanda ham jo'natuvchi nomi moderatsiyadan o'tganmi va hisobda pul bormi — ikkalasi *birinchi mijoz kirmoqchi bo'lganda* bilinadi, va restoran buni «sayt buzilgan» deb o'qiydi.",
      },
      {
        warn: "Matnda `{code}` bo'lmasa saqlanmaydi. Busiz mijoz *kodsiz xabar* oladi: shlyuz qabul qiladi, SMS keladi, hech qayerda xato chiqmaydi va odam kira olmaydi.",
      },
      {
        p: "Matnni o'zgartirsangiz moderatsiya bekor bo'ladi, shuning uchun «tekshirilgan» belgisi ham nolga tushadi — shlyuz hech qachon ko'rmagan matn ustida turgan yashil belgi yolg'on gapiradi.",
      },
      {
        tip: "Lotin o'zbekchada yozing: 160 belgi bir SMS. Kirill yoki `oʻ` bo'lsa chegara *70 belgiga* tushadi va bitta xabar uch bo'lakka bo'linib ketishi mumkin.",
      },
      { see: ["segments", "login"] },
    ],
  },
  {
    slug: "payments",
    section: "integrations",
    title: "Onlayn to'lov: Payme, Click, Uzum, ATMOS",
    lead: "Mijoz saytda karta bilan to'laydi. Kalitlarni provayder kabinetidan olasiz.",
    keys: ["to'lov", "payme", "click", "uzum", "atmos", "karta", "onlayn"],
    body: [
      {
        p: "`Sozlamalar` → `To'lov`. Har provayder alohida yoqiladi — bir nechtasini birga ishlatish mumkin, mijoz checkout'da tanlaydi.",
      },
      { h: "Ulash tartibi" },
      {
        steps: [
          "Provayder bilan shartnoma tuzasiz va kabinet olasiz.",
          "Kabinetdan kalitlarni (merchant id, kalit) olib panelga kiritasiz.",
          "Panel sizga *callback manzilini* beradi — uni provayder kabinetiga yozasiz.",
          "Yoqib, kichik summaga sinov to'lovi qiling.",
        ],
      },
      {
        warn: "Callback manzili — eng ko'p unutiladigan qadam. Usiz to'lov o'tadi, lekin bizga hech kim aytmaydi: mijozning puli ketadi, buyurtma esa «to'lanmagan» bo'lib turadi.",
      },
      {
        warn: "Buyurtmani bekor qilish *pulni qaytarmaydi*. Qaytarish provayder kabinetidan qilinadi.",
      },
      {
        p: "Sozlanmagan provayder mijozga *umuman ko'rsatilmaydi*: tugma bankning xato sahifasiga olib borsa, mehmon restoranni ayblaydi.",
      },
      { see: ["till-payment", "order-cancel"] },
    ],
  },
  {
    slug: "telegram-bot",
    section: "integrations",
    title: "Telegram bot va mini app",
    lead: "O'z botingiz: menyu, buyurtma va bildirishnomalar Telegram ichida.",
    keys: ["telegram", "bot", "mini app", "token", "botfather"],
    body: [
      {
        steps: [
          "Telegramda `@BotFather` ga yozing → `/newbot` → nom va username bering.",
          "BotFather bergan *tokenni* nusxalang.",
          "Panelda `Sozlamalar` → `Telegram` ga qo'ying.",
          "`Ulanishni tekshirish` bosing — bot nomi o'zi to'ldiriladi va mini app havolasi shundan quriladi.",
        ],
      },
      {
        warn: "`Ulanishni tekshirish` — bu sahifaning asosiy tugmasi: o'lgan botning tokeni ishlaydiganidan faqat birinchi mehmon kirmoqchi bo'lganda farq qiladi.",
      },
      {
        warn: "Bot tokeni — sir. Sizib chiqqan token «birovning hisobidagi pul» emas: bu mini app'ni ochgan *har bir mehmonga* restoran nomidan yozish imkoniyati.",
      },
      { h: "Mini app" },
      {
        p: "Telegram ichida ochiladigan menyu. Mehmon Telegram hisobidan avtomatik kiradi — SMS kod so'ralmaydi.",
      },
      {
        p: "Telegram telefon raqamini bermaydi, faqat ism. Shuning uchun mini app raqamni *checkout'dan oldin* so'raydi — «tasdiqlash» qadamida so'rash allaqachon yutilgan buyurtmani yo'qotardi.",
      },
      {
        tip: "Bot buyurtma holati haqida xabar yuboradi va bu SMS'dan tekin. Mijozning yarmi botga o'tsa SMS xarajati sezilarli tushadi.",
      },
      { see: ["sms", "segments"] },
    ],
  },
  {
    slug: "pos",
    section: "integrations",
    title: "Kassa tizimi: iiko, Syrve, Poster, Clopos, r_keeper",
    lead: "Sizda allaqachon kassa bo'lsa — buyurtma to'g'ridan-to'g'ri unga tushadi.",
    keys: ["iiko", "poster", "clopos", "r_keeper", "syrve", "pos", "kassa integratsiya"],
    body: [
      {
        p: "`Sozlamalar` → `POS`. *Menyu bizniki qoladi* — POS'dan hech nima tortilmaydi: bizda rasm, tarjima, to'plam va sayt matnlari bor, ikki joyda menyu yuritish chalkashlik.",
      },
      { h: "Ulash" },
      {
        steps: [
          "Provayderni tanlang va kalitlarini kiriting.",
          "`Ulanishni tekshirish` — javob qaysi tashkilot va qaysi terminal ekanini yozadi, shunchaki «ulandi» emas.",
          "`Bog'lash` bo'limida har taomni kassadagi mahsulotga bog'laysiz.",
        ],
      },
      {
        warn: "*Bog'lanmagan taom butun buyurtmani to'xtatadi.* Bu ataylab: chala chek olgan oshxona aynan ko'rganini pishiradi. Xato xabarida qaysi taom ekani yoziladi.",
      },
      {
        p: "Sozlama *filialga tegishli*: zanjirda har oshxonaning o'z terminal guruhi bor, va buyurtmaning noto'g'ri kassada chop etilishi umuman chop etilmaganidan yomonroq. Bog'lashni boshqa filialdan ko'chirish mumkin — bir xil brend va bir xil provayder shart.",
      },
      { h: "Provayderlarning o'ziga xosligi" },
      {
        table: {
          head: ["Tizim", "Nimani bilish kerak"],
          rows: [
            ["*iiko / Syrve*", "Buyurtma asinxron yaratiladi: javob 200 bo'lsa ham kassa uni hali qabul qilmagan bo'lishi mumkin. Tizim 12 soniyagacha kutib javobni tekshiradi."],
            ["*Poster*", "Buyurtma «onlayn buyurtma» bo'lib tushadi va kassadagi odam `Qabul qilish` bosishi kerak. Bekor qilish API'si yo'q — kassada bekor qilinadi."],
            ["*Clopos*", "Avtomatik qabul qilish sozlamalari yoqilgan bo'lishi shart, aks holda buyurtma qabul qilinmagan holda turadi."],
            ["*r_keeper*", "Server *restoran ichida* turadi. Bizning serverimiz unga port ochilmasa yoki VPN bo'lmasa yeta olmaydi."],
          ],
        },
      },
      {
        tip: "Kassa javob bermasa buyurtma buzilmaydi: sabab chekda yoziladi va `Qayta yuborish` tugmasi turadi. Bizda bor, kassada yo'q buyurtmani tuzatish mumkin; jimgina yo'qolganini — yo'q.",
      },
      { fig: "pos" },
      { see: ["stop-list", "menu-import", "order-flow"] },
    ],
  },
  {
    slug: "pbx",
    section: "integrations",
    title: "ATS: onlinePBX",
    lead: "Kiruvchi qo'ng'iroqda mijoz kartochkasi o'zi ochiladi.",
    keys: ["ats", "onlinepbx", "telefoniya", "qo'ng'iroq"],
    body: [
      {
        steps: [
          "`Sozlamalar` → `Telefoniya` da onlinePBX kalitlarini kiriting.",
          "Panel sizga *webhook manzilini* beradi.",
          "Uni onlinePBX kabinetidagi hodisalar sozlamasiga yozing.",
          "Bitta sinov qo'ng'irog'i qiling va `Oxirgi hodisa` qatorini tekshiring.",
        ],
      },
      {
        warn: "`Oxirgi hodisa` — bu sahifadagi eng foydali qator. Kalitlar to'g'ri bo'lsa ham manzil onlinePBX paneliga yozilmagan bo'lishi mumkin, va ulanish tekshiruvi buni *umuman ko'rsata olmaydi*.",
      },
      {
        p: "Webhook manzilining o'zi kalit: onlinePBX hech qanday parol yubormaydi. Manzil sizib chiqsa uni almashtirish mumkin.",
      },
      {
        tip: "Qo'ng'iroq davomiyligi *suhbat* bo'yicha o'lchanadi. Qirq soniya jiringlab javob berilmagan qo'ng'iroq — nol soniyalik suhbat.",
      },
      { see: ["calls"] },
    ],
  },
  {
    slug: "fiscal",
    section: "integrations",
    title: "Fiskal kassa",
    lead: "Chekni davlat tizimiga yuborish: qaysi provayderlar bor va nima kerak.",
    keys: ["fiskal", "chek", "soliq", "ofd", "multikassa", "ikpu"],
    body: [
      {
        p: "`Sozlamalar` → `Fiskal`. Provayder tanlanadi va kalitlari kiritiladi. Chek kassada yopilganda avtomatik yuboriladi.",
      },
      {
        warn: "*Adapteri hali yozilmagan provayderni yoqib bo'lmaydi.* Kalitni saqlash mumkin (ega ko'pincha shartnoma tugashidan oldin sozlaydi), lekin yoqish sabab bilan rad etiladi: fayl qilyapman deb o'ylagan restoran — yo'q xususiyat emas, *huquqiy muammo*.",
      },
      { h: "ИКПУ va o'lchov birligi" },
      {
        p: "Har taomga ИКПУ kodi va o'lchov birligi kodi qo'yiladi (`Menyu` → taom → `Fiskal`). Busiz chek rad etilishi mumkin.",
      },
      {
        p: "ИКПУ *mahsulot haqidagi fakt*: menyudan o'qiladi, ya'ni buxgalterning tuzatishi hali to'lanmagan buyurtmalarga ham yetadi.",
      },
      {
        tip: "QQS foizi filialga qo'yiladi, kerak bo'lsa taomga alohida. Taomga qo'yilmagan bo'lsa filialniki ishlatiladi.",
      },
      { see: ["marking", "till-shift"] },
    ],
  },
  {
    slug: "marking",
    section: "integrations",
    title: "Markirovka (Asl Belgisi)",
    lead: "Ichimlik sotsangiz: har shishaning DataMatrix kodi chekka qo'shiladi.",
    keys: ["markirovka", "asl belgisi", "datamatrix", "skaner", "ichimlik"],
    body: [
      {
        p: "Alohida «Asl Belgisi» integratsiyasi *yo'q va kerak emas*: kod fiskal chek ichida ketadi, OFD uni milliy tizimga uzatadi.",
      },
      {
        steps: [
          "Menyuda markirovkalanadigan mahsulotga belgi qo'ying (`Menyu` → taom → `Markirovka`).",
          "Kassaga DataMatrix o'qiydigan skaner ulang — u oddiy klaviatura bo'lib ko'rinadi, drayver kerak emas.",
          "Sotuvda qator qo'shilganda kod so'raladi: shishani skanerlaysiz.",
        ],
      },
      {
        warn: "*Markirovkalangan qator birlashmaydi*: ikki shisha — ikki kod. Bitta tugmani ikki marta bosish bitta ikkilik qator emas, ikkita alohida qator bo'ladi.",
      },
      {
        warn: "Takror kod butun chek bo'yicha tekshiriladi. Skaner chiyilladi, o'qigani noaniq, yana skanerlandi — ikki qator bitta shishani chiqaradi. *Fiskal kassa bunday chekni qabul qiladi*, ya'ni ushlanadigan yagona joy — bizniki.",
      },
      {
        tip: "Butun shisha va stakan — menyuda *ikki xil taom*. Shunda «ochilgan shisha» savoli o'zi hal bo'ladi: stakanda kod so'ralmaydi.",
      },
      { see: ["fiscal", "menu-item"] },
    ],
  },
  {
    slug: "pos-migration",
    section: "integrations",
    title: "Boshqa POS'dan ko'chirish",
    lead: "iiko, r_keeper, Clopos, Poster yoki Jowi'dan ma'lumotni olib kelish.",
    keys: ["ko'chirish", "migratsiya", "import", "iiko dan"],
    body: [
      {
        p: "`Menyu` → `POS'dan import`. Eski tizimingizdan Excel/CSV eksport qilib yuklaysiz.",
      },
      {
        steps: [
          "*Masalliqlar* — nomenklatura: nom, o'lchov birligi, narx.",
          "*Texkartalar* — masalliqlar nom bo'yicha topiladi, shuning uchun avval birinchi qadam bajarilishi kerak.",
          "*Ombor qoldig'i* — boshlang'ich inventarizatsiya sifatida yoziladi.",
        ],
      },
      {
        warn: "Topilmagan nomlar ro'yxat bo'lib ko'rsatiladi va *yozilmaydi*. Ularni qo'lda moslash kerak — taxmin qilib yozish noto'g'ri masalliqli texkartani berardi.",
      },
      { see: ["menu-import", "ingredients", "tech-cards"] },
    ],
  },

  // ═══════════════════════════════════ Hisobotlar
  {
    slug: "dashboard",
    section: "reports",
    title: "Bosh sahifa",
    lead: "Bugun qanday ketyapti: buyurtma, tushum, o'rtacha chek, kutayotganlar.",
    keys: ["dashboard", "bosh sahifa", "statistika", "bugun"],
    body: [
      {
        p: "Panelga kirganda birinchi ko'rinadigan ekran. Yuqorida davr tanlagich — barcha raqamlar shu davrga tegishli.",
      },
      {
        list: [
          "Buyurtmalar soni va tushum, o'tgan davr bilan solishtirilgan holda.",
          "O'rtacha chek.",
          "Kutayotgan buyurtmalar — hali qabul qilinmaganlar alohida.",
          "*Qarz* alohida plitka: qolgani bir soatda o'zi keladi, bu esa kimdir qo'ng'iroq qilganda keladi.",
          "Eng ko'p sotilgan taomlar.",
        ],
      },
      {
        tip: "Plitkalarni o'zingizga moslash mumkin: kerak bo'lmaganini o'chirasiz. Har egaga kerak bo'lgan raqam har xil.",
      },
      { fig: "dashboard" },
      { see: ["reports-sales", "ai-briefing"] },
    ],
  },
  {
    slug: "reports-sales",
    section: "reports",
    title: "Savdo hisobotlari",
    lead: "Nima sotildi, qaysi kanaldan, kim sotdi.",
    keys: ["hisobot", "savdo", "otchet", "kanal", "taom"],
    body: [
      {
        p: "`Pul va jamoa` → `Hisobotlar`. Davr tanlanadi, hisobot turi tanlanadi.",
      },
      {
        table: {
          head: ["Hisobot", "Savol"],
          rows: [
            ["*Taomlar*", "Nima sotildi, nechta, qancha pulga"],
            ["*Kanallar*", "Sayt, Telegram, kassa, QR — qaysi biri ko'proq"],
            ["*Jamoa*", "Kim qancha sotdi"],
            ["*ABC/XYZ*", "Qaysi taom pul keltiradi, qaysi biri barqaror sotiladi"],
            ["*Ombor*", "Nima kirdi, nima chiqdi"],
          ],
        },
      },
      {
        warn: "Har hisobot *o'z qamrovini* aytadi: «tannarx 19 taomdan 1 tasida kiritilgan — yalpi foyda tushumning 30% ini qamraydi». Ikki yuzdan o'ntasini narxlagan restoran aks holda kechaning 6% ini tasvirlaydigan ustunga qarab qaror qabul qilardi.",
      },
      { fig: "reports" },
      { see: ["abc", "finance", "excel"] },
    ],
  },
  {
    slug: "abc",
    section: "reports",
    title: "ABC/XYZ tahlili",
    lead: "Qaysi taom pul keltiradi va qaysi biri ishonchli sotiladi.",
    keys: ["abc", "xyz", "tahlil", "pareto", "menyu tahlili"],
    body: [
      {
        p: "*ABC* — pul bo'yicha: `A` guruhi tushumning katta qismini beradi, `C` deyarli hech nima. *XYZ* — barqarorlik bo'yicha: `X` har kuni bir xil sotiladi, `Z` tasodifiy.",
      },
      {
        list: [
          "`AX` — asosiy taomlar. Ular hech qachon stop listga tushmasligi kerak.",
          "`CZ` — menyudan olib tashlashga nomzod: kam sotiladi va oldindan aytib bo'lmaydi.",
          "`AZ` — ko'p pul keltiradi, lekin notekis. Ombor bilan ehtiyot bo'ling.",
        ],
      },
      {
        p: "Masalliqlar uchun ham alohida ABC bor va u *boshqa savolga* javob beradi: eng ko'p daromad keltiradigan taom ko'pincha eng qimmat masalliqdan qilinmaydi.",
      },
      {
        warn: "Masalliq ABC'si *sotib olingan pul bo'yicha* saflanadi, kartalar bo'yicha sarf emas: birinchisi o'lchangan (sanasi va jami bor nakladnoy), ikkinchisi esa yarim menyusi narxlanmagan taxmin.",
      },
      { see: ["reports-sales", "tech-cards"] },
    ],
  },
  {
    slug: "finance",
    section: "reports",
    title: "Moliyaviy hisobot va kassa",
    lead: "Pul qayerdan keldi, qayerga ketdi — va nima uchun bu foyda hisoboti emas.",
    keys: ["moliya", "kassa", "pul", "foyda", "xarajat"],
    body: [
      {
        p: "`Pul va jamoa` → `Kassa`. Smenalar, kirim-chiqim va kunlik qoldiq.",
      },
      {
        warn: "*Moliyaviy hisobot foyda hisoboti EMAS*, va hisobotning o'zi buni yozadi. Menyusi tannarxlanmagan restoranda «kirim − chiqim» — pul harakati, foyda emas. Uni foyda deb o'qish butun ovqat tannarxi qadar oshirib ko'rsatadi.",
      },
      {
        list: [
          "*Chegirma va ballar xarajat emas*: pul chiqmagan, u umuman kelmagan.",
          "*Kuryer topshirig'i xarajat emas*: bu kuryer yig'gan naqdning kassaga kirishi.",
          "*Ko'chirish xarajat emas*: qiymat tashiladi, yaratilmaydi.",
        ],
      },
      {
        p: "Yetkazishdagi naqd kassaga kuryer topshirgandan keyin kiradi. Kuryer qo'lidagi pul alohida ko'rsatiladi — kamomadni tekshirayotgan odam birinchi navbatda shu raqamni so'raydi.",
      },
      { fig: "cash" },
      { see: ["till-shift", "couriers", "reports-sales"] },
    ],
  },
  {
    slug: "excel",
    section: "reports",
    title: "Excel'ga yuklab olish",
    lead: "Har hisobot bir bosishda `.xlsx` bo'lib chiqadi.",
    keys: ["excel", "eksport", "yuklab olish", "xlsx", "1c"],
    body: [
      {
        p: "Har hisobot sahifasida `Excel` tugmasi bor. Fayl ekrandagi bilan *aynan bir xil* ustunlar va bir xil davrni oladi.",
      },
      {
        tip: "Buxgalteringizga hisobot kerak bo'lsa, panel logini bermang — Excel faylini yuboring. Panel logini mijozlar bazasi va to'lov kalitlari degani.",
      },
      { see: ["reports-sales", "data-export"] },
    ],
  },
  {
    slug: "ai-briefing",
    section: "reports",
    title: "Ertalabki brifing",
    lead: "Kechagi kun haqida bir necha jumla: nima o'zgardi va nimaga qarash kerak.",
    keys: ["ai", "brifing", "yordamchi", "xulosa"],
    body: [
      {
        p: "Har kuni ertalab panelda kechagi kun haqida qisqa xulosa chiqadi: tushum o'tgan haftaga nisbatan, qaysi taom ko'p sotildi, nima g'alati.",
      },
      {
        p: "Bu raqamlarning o'rnini bosmaydi — u qaysi raqamga qarash kerakligini aytadi.",
      },
      {
        tip: "Kunlik limit bor. Limit tugasa ertasi kuni yana ishlaydi, yoki qo'shimcha limit olish mumkin.",
      },
      { see: ["dashboard"] },
    ],
  },

  // ═══════════════════════════════════ Sozlamalar
  {
    slug: "theme-lang",
    section: "settings",
    title: "Til va tema",
    lead: "Panel uch tilda, yorug' va qorong'i rejim.",
    keys: ["til", "tema", "qorong'i", "dark", "rus"],
    body: [
      {
        p: "Panelning yuqori chap burchagida til tanlagich va tema tugmasi bor. Tanlov *sizniki* — boshqa adminlarga ta'sir qilmaydi.",
      },
      {
        p: "Sayt tili esa mehmonning brauzeridan olinadi, va u o'zi almashtira oladi. Sayt tillari uchun alohida manzillar bor (`/ru/`, `/en/`).",
      },
      { see: ["site-texts"] },
    ],
  },
  {
    slug: "notifications",
    section: "settings",
    title: "Ovoz va bildirishnomalar",
    lead: "Nima haqida xabar berilsin va qayerga.",
    keys: ["ovoz", "bildirishnoma", "signal", "telegram xabar"],
    body: [
      {
        list: [
          "*Yangi buyurtma ovozi* — panelda, qabul qilinguncha takrorlanadi.",
          "*Telegram xabar* — buyurtma, bron va shikoyat haqida guruhga yoki shaxsiy chatga.",
          "*Ogohlantirishlar* — chiqmagan chek, katta chegirma, texkarta normasining oshishi, kassa kamomadi.",
        ],
      },
      {
        p: "Telegram xabarini olish uchun botga bir marta yozib qo'yish kerak — bot avval o'zi yoza olmaydi.",
      },
      {
        tip: "Har ogohlantirishning panelda aynan bitta to'xtatuvchi tugmasi bo'lishi kerak. Agar biror signal sizni bezovta qilsa, uni o'chiring — o'chirib bo'lmaydigan ogohlantirish o'qilmaydigan ogohlantirishga aylanadi.",
      },
      { see: ["order-accept", "telegram-bot", "admin-log"] },
    ],
  },
  {
    slug: "security",
    section: "settings",
    title: "Xavfsizlik",
    lead: "Kim nimaga kira oladi va nimalar hech qachon ko'rsatilmaydi.",
    keys: ["xavfsizlik", "parol", "kalit", "himoya"],
    body: [
      {
        list: [
          "Parollar hech qachon ochiq saqlanmaydi.",
          "To'lov, SMS, ATS va bot kalitlari *hech qachon qaytarilmaydi*: ekranda faqat «saqlangan» belgisi.",
          "Menejer o'z filialidan tashqariga chiqa olmaydi — URL orqali ham.",
          "Kirish urinishlari cheklangan: parolni ketma-ket noto'g'ri kiritish vaqtincha bloklaydi.",
          "Har muhim amal jurnalga yoziladi.",
        ],
      },
      {
        warn: "Eng katta xavf — *bo'lishilgan hisob*. Bitta «ega» logini bilan uch kishi ishlasa, jurnal ham, filial qamrovi ham, ruxsatlar ham ma'nosini yo'qotadi.",
      },
      {
        tip: "Xarita kaliti bundan mustasno: u brauzerga beriladi va sir bo'la olmaydi. Himoyasi — provayder kabinetidagi domen cheklovi.",
      },
      { see: ["admins", "admin-log", "map-provider"] },
    ],
  },
  {
    slug: "data-export",
    section: "settings",
    title: "Ma'lumotni olib ketish",
    lead: "Menyu, mijozlar va buyurtmalar — hammasi sizniki.",
    keys: ["eksport", "ma'lumot", "olib ketish", "backup"],
    body: [
      {
        p: "Menyusini, buyurtmalarini va bazasini ko'chira olmaydigan restoran mahsulot bilan emas, *chiqish narxi* bilan ushlab turilgan bo'ladi. Shuning uchun eksport bor.",
      },
      {
        steps: [
          "`Sozlamalar` → `Ma'lumot` da so'rov qoldiring.",
          "Biz ruxsat beramiz (bu bir marta beriladigan ruxsat).",
          "Fayllarni yuklab olasiz: menyu, mijozlar, buyurtmalar — `.xlsx` va `.csv`.",
        ],
      },
      {
        tip: "Zaxira nusxa har kuni avtomatik olinadi va biz saqlaymiz. Eksport — bu boshqa narsa: bu ma'lumotni *o'zingiz bilan* olib ketish.",
      },
      { see: ["excel", "billing"] },
    ],
  },
  {
    slug: "billing",
    section: "settings",
    title: "To'lov va obuna",
    lead: "Qancha to'laysiz, qachon va nima bo'lsa to'xtaydi.",
    keys: ["to'lov", "obuna", "narx", "hisob", "faktura"],
    body: [
      {
        p: "Obuna oylik. Panelda `Sozlamalar` → `Obuna` da joriy davr, keyingi to'lov sanasi va hisoblar tarixi ko'rinadi.",
      },
      {
        p: "Narx kassa (oylik) va buyurtma (har buyurtma uchun) dan iborat. Kalkulyator keel.uz bosh sahifasida.",
      },
      {
        tip: "To'lov kechiksa sayt darhol o'chmaydi — avval panelda ogohlantirish chiqadi. Lekin uzoq kechikish saytni to'xtatadi, shuning uchun hisobni ko'z ostida tuting.",
      },
      { see: ["data-export"] },
    ],
  },
  {
    slug: "support",
    section: "settings",
    title: "Yordam so'rash",
    lead: "Qanday yozsangiz javob birinchi xabardayoq keladi.",
    keys: ["yordam", "support", "qo'llab-quvvatlash", "aloqa"],
    body: [
      {
        p: "Panelning o'ng pastida chat tugmasi bor — u yerdan to'g'ridan-to'g'ri bizga yoziladi. Telegramda ham javob beramiz.",
      },
      { h: "Nima yozish kerak" },
      {
        list: [
          "*Qaysi ekran* — «Ombor → Texkartalar».",
          "*Nima qildingiz* — «zagotovka saqladim».",
          "*Nima kutdingiz va nima bo'ldi* — «tannarx chiqishi kerak edi, nol turibdi».",
          "*Screenshot* — bitta rasm o'nta xabarni almashtiradi.",
        ],
      },
      {
        tip: "«Ishlamayapti» degan xabar javobni ikki-uch xabarga kechiktiradi. Yuqoridagi to'rt qator bo'lsa, odatda birinchi javobda yechim beriladi.",
      },
    ],
  },
];
