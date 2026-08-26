# Restoranga Keel o'rnatish — to'liq qo'llanma

Bitta restoranni noldan ishga tushirish: tarmoq, switch, kabel, IP manzillar,
Wi-Fi, printerlar, monobloklar, va boshqa POS'dan ko'chirish.

Bu fayl `ORNATISH.txt` va `BAR_PRINTERI.txt` ni o'z ichiga oladi va ularning
ustiga tarmoq muhandisligi qismini qo'shadi. Jonli restoranda topilgan
nosozliklar ⚠️ bilan belgilangan — ularning har biri kimningdir kechasiga
tushgan.

---

## 0. Eng muhim gap — buni birinchi o'qing

> **Restoran printerlarida Windows drayveri YO'Q va kerak emas.**

Termal chek printeri tarmoqqa ulanadi va **9100 portida xom ESC/POS baytlarini**
qabul qiladi. Kassa dasturi baytlarni to'g'ridan-to'g'ri yuboradi.

⚠️ Shuning uchun `Устройства и принтеры` da bo'sh ro'yxat ko'rsangiz — **bu
normal**, nosozlik emas. Drayver qidirib vaqt yo'qotmang.

Ikkinchi gap, undan ham muhimroq:

> **Ulanish sxemasini yozib qoldirmasangiz, olti oydan keyin uni qaytadan
> topishga to'g'ri keladi — va o'sha paytda restoran ishlab turgan bo'ladi.**

---

## 1. Borishdan oldin

### Restorandan so'raladigan savollar

| Savol | Nega |
|---|---|
| Nechta kassa, nechta ofitsiant ekrani? | Monoblok soni va tarif |
| Nechta chop etish nuqtasi? (bar, issiq sex, somsa, salat…) | Printer soni va marshrutlash |
| Hozir qaysi dastur ishlaydi? (iiko, Poster, yo'q) | Ko'chirish rejasi — §8 |
| Internet kimniki, routerga kim kiradi? | Parolsiz router = ishning to'xtashi |
| Remont tugaganmi? | Kabel devor ichidan o'tadimi yoki ustidan |
| Fiskal kassa bormi, qaysi model? | §7 |

⚠️ **"Keyin aytamiz" degan javob — "biz bilmaymiz" degani.** Chop etish
nuqtalari sonini joyida sanang: oshxonaga kirib, har bir sexda kim turishini
ko'ring.

### Olib boriladigan narsalar

- Krimper, konnektor (RJ45), **kabel testeri**
- Kabel g'altagi (Cat5e yetadi, Cat6 kerak emas)
- Switch (§2.2 da qaysi biri)
- Uzatgich, ko'p rozetkali filtr
- **Noutbuk** — router sozlash uchun
- Flesh: Keel installeri, `BAR_PRINTERI.txt`
- Marker (kabel uchlarini belgilash), skotch
- **Skrepka** — printerlarda yashirin FEED tugmasi bor

---

## 2. Tarmoq

### 2.1. Markaz qayerda turadi

Bitta joy — odatda ofis yoki kassa orqasidagi shkaf:

- Provayder routeri
- Switch
- UPS

⚠️ **Oshxonaga qo'ymang.** Bug', yog', issiqlik va suv switch'ni bir yilda
o'ldiradi — va u o'lganda **butun restoran to'xtaydi**.

⚠️ **1.5 metrdan baland mahkamlang.** Polda turgan switch — farrosh yuvganda
suv tegadigan switch.

### 2.2. Switch tanlash

| Restoran | Switch |
|---|---|
| 1 kassa, 1–2 printer | 8 portli, boshqarilmaydigan |
| 2 kassa, 4 printer, Wi-Fi | **16 portli**, boshqarilmaydigan |
| Zanjir, VLAN kerak | 24 portli, boshqariladigan |

⚠️ **Portni ikki barobar oling.** Bugun 8 ta qurilma bo'lsa, 16 portli oling.
Restoran bir yildan keyin ikkinchi kassa qo'yadi, va o'shanda **remont tugagan**
bo'ladi.

⚠️ **Boshqariladigan switch kerak emas** — agar VLAN qilmasangiz. Boshqariladigan
switch = sozlanadigan switch = **noto'g'ri sozlanishi mumkin bo'lgan switch**, va
uni sozlagan odam olti oydan keyin yo'q bo'ladi.

⚠️ **PoE kerakmi?** Faqat Wi-Fi access point'ni shiftdan quvvatlantirmoqchi
bo'lsangiz. Printerlar PoE ishlatmaydi.

### 2.3. Kabel — yulduz sxemasi

```
                 ┌── Kassa monobloki 1
                 ├── Kassa monobloki 2
                 ├── Ofitsiant monobloki 1
   Router ── Switch ── Ofitsiant monobloki 2
                 ├── Kassa chek printeri
                 ├── Oshxona printeri
                 ├── Bar printeri
                 ├── Somsaxona printeri
                 ├── Salatxona printeri
                 ├── Fiskal kassa
                 └── Wi-Fi access point
```

⚠️ **Zanjir qilib ulamang.** Bir qurilmadan ikkinchisiga o'tkazilgan kabel —
bitta uzilish undan keyingi **hammasini** o'chiradi, va qaysi biri uzilganini
topish uchun hammasini tekshirish kerak bo'ladi.

⚠️ **Har nuqtaga ikkitadan kabel torting.** Kabel arzon; devorni ikkinchi marta
ochish qimmat.

⚠️ **90 metrdan oshmasin** — Ethernet chegarasi. Undan uzun bo'lsa aloqa
*"ba'zan ishlaydi"* holatiga tushadi, va bu **topish eng qiyin** nosozlik turi.

### 2.4. Kabelni qanday tortish

- Kabel kanalida, elektr kabelidan **kamida 20 sm** narida
- Keskin bukmang (radius diametrdan 4 barobar)
- **Ikkala uchini ham belgilang**: `KASSA-1`, `OSH-PRN`, `BAR-PRN`
- Har uchini **tester** bilan tekshiring

⚠️ Krimper bilan siqilgan konnektor **ko'rinishidan to'g'ri** bo'lib,
ishlamasligi mumkin. Tester 30 soniya oladi; noto'g'ri konnektorni keyin topish
yarim kun oladi.

### 2.5. Wi-Fi

⚠️ **Kassa va printerlar Wi-Fi'da bo'lmasin. Hech qachon.**

Sabab: Wi-Fi uziladi va **hech qanday xato bermaydi**. Chek chiqmaydi, oshxona
buyurtmani ko'rmaydi, va hech kim nega ekanini bilmaydi. Sim uzilsa — chiroq
o'chadi, ko'rinadi.

Wi-Fi faqat:
- Ofitsiant **telefonlari** (Keel Waiter ilovasi)
- Mehmonlar

⚠️ **Mehmon Wi-Fi'sini alohida tarmoqqa chiqaring** (routerdagi *Guest Network*).
Bir tarmoqda bo'lsa, mehmon telefoni printerlarni ko'radi — va kimdir buni bir
kun sinab ko'radi.

⚠️ **Access point'ni zalga qo'ying, shkafga emas.** Metall shkaf ichidagi AP —
signal beruvchi qopqoq.

---

## 3. IP manzillar

### 3.1. Statik, DHCP emas

⚠️ **Printerning IP'si o'zgarsa, kassa unga chop eta olmaydi va ekranda hech
qanday xato chiqmaydi** — chek shunchaki chiqmaydi. DHCP esa svet o'chib
yonganda manzillarni almashtirib yuborishi mumkin.

### 3.2. Taqsimot

Router `192.168.1.1` bo'lsa:

| Diapazon | Nima |
|---|---|
| `.1` | Router |
| `.2–.9` | Zaxira (switch boshqaruvi, AP) |
| `.10–.19` | Monobloklar (kassa `.10`, zal `.11`, `.12`) |
| `.20–.39` | **Printerlar** (kassa `.20`, oshxona `.21`, bar `.22`, somsa `.23`, salat `.24`) |
| `.40–.49` | Fiskal kassa |
| `.100+` | DHCP — telefonlar, mehmon Wi-Fi |

⚠️ **Routerning DHCP diapazonini `.100` dan boshlang.** Aks holda u siz statik
qo'ygan manzilni boshqa qurilmaga beradi, va ikki qurilma bitta IP'da urishadi —
alomati: *"printer ba'zan ishlaydi"*.

⚠️ **Bu jadvalni qog'ozga yozib shkafga yopishtiring.** Bir yildan keyin kelgan
odam (yoki siz) uni o'qiydi.

### 3.3. Nechta tarmoq bo'lishi kerak

Bitta. ⚠️ **VLAN qilmang** — agar restoran buni so'ramagan bo'lsa. Ikki tarmoq
= ikki marta ko'p nosozlik, va ular orasidagi muammoni topish uchun
boshqariladigan switch bilan ishlashni bilish kerak.

Yagona istisno: **mehmon Wi-Fi'si** — u routerning o'z *Guest* rejimida, alohida.

---

## 4. Printerlar

### 4.1. Nechta va qayerda

Har bir **chop etish nuqtasi** = bitta printer:

- **Kassa** — mijoz cheki, kassir cheki, hisob (precheck)
- **Issiq sex (oshxona)** — pishiriladigan taomlar
- **Bar** — ichimliklar
- **Somsaxona / salatxona / grill** — restoranda alohida sex bo'lsa

⚠️ **Ikki sexga bitta printer qo'ymang.** Barmen oshpazning chekini yirtib
oladi, va yo'qolgan chek — pishirilmagan taom.

### 4.2. Printerga IP qo'yish

**Usul A — DHCP yoqib, keyin routerdan band qilish** (osonroq)

1. Printerda DHCP yoqing (modelga qarab: FEED bosib turish yoki utilita)
2. Self-test chiqaring — qaysi IP olganini ko'ring
3. Routerda **DHCP reservation** qiling: MAC → o'sha IP

⚠️ Bu eng ishonchli usul: IP hech qachon o'zgarmaydi, va uni **routerda**
ko'rish mumkin.

**Usul B — printerga qo'lda yozish**

1. Kompyuterni vaqtincha printer tarmog'iga o'tkazing (§4.4)
2. Brauzerda `http://<printer-ip>` → *Network settings*
3. IP, maska `255.255.255.0`, gateway = router
4. Kompyuterni qaytaring

### 4.3. Self-test — har doim ishlaydigan yagona usul

1. Printerni **o'chiring**
2. **FEED** tugmasini **bosib turing** ⚠️ (ba'zi modellarda orqada, teshik
   ichida — skrepka kerak)
3. Bosib turgan holda **yoqing**
4. Qog'oz chiqa boshlaganda qo'yib yuboring

Qog'ozda: **IP Address**, **MAC Address**, **Subnet Mask**, **Gateway**, **Port**,
**DHCP ON/OFF**.

⚠️ **Bu usul tarmoqqa, portga, kassa dasturiga va POS'ga umuman bog'liq emas.**
To'rtta printer, to'rtta chek, to'rtta aniq javob — bir daqiqada. Boshqa hamma
usul taxmin.

### 4.4. Printer boshqa tarmoqda chiqsa

⚠️ **Bu eng ko'p uchraydigan holat va jonli restoranda topilgan.** Ikkita printer
tarmoqqa sozlangan, ikkitasi **zavod holida** qolgan:

```
oshxona   192.168.100.28   ✓ ishlaydi
bar       192.168.123.20   ✗ boshqa olamda yashaydi
```

Bar printeri jismonan simda, chiroq yonyapti, sog'lom — va **hech qachon
ishlamagan**. 65 marta urinilgan, 65 marta yiqilgan.

**Tuzatish**: kompyuterni vaqtincha o'sha tarmoqqa o'tkazib, printer sozlamasini
oching.

```
ncpa.cpl → Ethernet → Свойства → IPv4 → Свойства
  IP:     192.168.123.50      ← printer tarmog'i, oxirgi raqam boshqa
  Maska:  255.255.255.0
  Shlyuz: (bo'sh)
```

Brauzerda `http://192.168.123.20` → tarmoq sozlamalari → to'g'ri IP → saqlash.

⚠️ **Keyin kompyuterni qaytaring** (*Получить IP автоматически*). Buni unutish —
"internet yo'qoldi" degan qo'ng'iroq.

---

## 5. Tarmoqni tekshirish — ishlaydigan buyruqlar

### 5.1. ⚠️ `arp -a` — qurilmalar ro'yxati EMAS

Bu buyruq **shu kompyuter yaqinda gaplashgan** qurilmalarni ko'rsatadi. Hech
qachon murojaat qilinmagan printer **mukammal ishlab turib** ham u yerda
**bo'lmaydi**.

*"`arp -a` da yo'q"* = *"hali gaplashmaganmiz"*, **buzuq degani emas**.

### 5.2. Tarmoqni o'zi aniqlaydigan skaner

PowerShell (**administrator** sifatida), to'liq ko'chiring — ⚠️ **almashtiradigan
joyi yo'q**:

```powershell
$me=(Get-NetIPAddress -AddressFamily IPv4 | ? {$_.IPAddress -notlike '127.*' -and $_.IPAddress -notlike '169.254.*'} | select -First 1).IPAddress
$net=$me -replace '\.\d+$',''
"Bu kompyuter: $me   |   Tarmoq: $net.0/24"
$found=@()
1..254 | % { $ip="$net.$_"
  try { $c=New-Object Net.Sockets.TcpClient
        if($c.ConnectAsync($ip,9100).Wait(150)){ $found+=$ip; "$ip  <-- PRINTER" }
        $c.Dispose() } catch {} }
"--- topildi: $($found.Count) ta ---"
```

⚠️ Namuna IP yozib, uni almashtirishni unutish — jonli restoranda bo'lgan xato.
Skaner **bo'sh** natija berdi va bu *"printer yo'q"* bo'lib ko'rindi, aslida
**noto'g'ri tarmoq** skanerlangan edi.

### 5.3. Har printerga o'z manzilini chiqarish

```powershell
$found | % {
  try { $c=New-Object Net.Sockets.TcpClient($_,9100)
        $s=$c.GetStream()
        $b=[Text.Encoding]::ASCII.GetBytes("`n`n   $_`n`n`n`n")
        $s.Write($b,0,$b.Length); $s.Flush(); $s.Close(); $c.Close()
        "$_  yuborildi" } catch { "$_  XATO" } }
```

Aylanib chiqing — **har bir printer o'z IP'sini tutib turibdi**. Eslab qolish
shart emas, javob qog'ozda.

### 5.4. Port yopiq bo'lsa ham qurilmani topish

Ba'zi printer **bir vaqtda faqat bitta TCP ulanish** qabul qiladi. POS ulanib
turgan bo'lsa, skaner **javob ololmaydi**.

ARP bilan **hamma qurilma** ko'rinadi — port ochiqmi yoki yo'qmi, farqi yo'q:

```powershell
$net=((Get-NetIPAddress -AddressFamily IPv4 | ? {$_.IPAddress -notlike '127.*' -and $_.IPAddress -notlike '169.254.*'} | select -First 1).IPAddress) -replace '\.\d+$',''
1..254 | % { $null=(New-Object Net.NetworkInformation.Ping).SendPingAsync("$net.$_",200) }
Start-Sleep 4
arp -a | Select-String "$net\."
```

Keyin har birini to'rt portda sinang:

```powershell
$ips = (arp -a | Select-String "$net\." | % { ($_ -split '\s+')[1] }) | ? { $_ }
foreach($ip in $ips){ foreach($port in 9100,9101,9102,515){
  try { $c=New-Object Net.Sockets.TcpClient
        if($c.ConnectAsync($ip,$port).Wait(200)){ "$ip : $port  <-- OCHIQ" }
        $c.Dispose() } catch {} } }
```

### 5.5. Bitta manzilni tekshirish

```
ping 192.168.1.21
powershell -Command "Test-NetConnection 192.168.1.21 -Port 9100"
```

⚠️ `Test-NetConnection` — **PowerShell** buyrug'i. `cmd` da yuqoridagi
ko'rinishda yoziladi.

---

## 6. Boshqa POS'dan ko'chirish (iiko, Poster va boshqalar)

### 6.1. ⚠️ Ofitsiant terminali printerlarni KO'RMAYDI

Bu jonli restoranda vaqt yo'qotgan narsa.

iikoFront — **mijoz**. Ofitsiant chek urganda u buyurtmani **serverga** yuboradi,
va chekni **server yoki kassa stansiyasi** chiqaradi.

Ya'ni **ofitsiant monoblokidan skanerlash printerlar haqida hech nima
isbotlamaydi.** Beshta joyga chek chiqishi — POS'ning ishi, o'sha kompyuterning
emas.

Skanerlash **kassa turgan segmentdan** yoki **printerning o'zidan** (§4.3)
qilinadi.

### 6.2. POS'ning o'z sozlamasidan o'qish — eng tez yo'l

**iikoOffice** → *Администрирование* → *Настройки оборудования*

Har bir printerning **nomi, turi, IP:port yoki Windows printer nomi**.

Va *Сервис-печать* — **qaysi bo'lim qaysi printerga** ketishi. Bu Keel'dagi
"qaysi bo'limlarni chiqaradi" sozlamasining aynan o'zi.

⚠️ **Rasmini oling.** Ko'chirish uchun kerak bo'ladigan hamma narsa shu ikki
ekranda.

### 6.3. Ayri yo'l: IP mi, drayver mi

| POS'da yozilgani | Ma'nosi | Keel'da |
|---|---|---|
| `192.168.1.21:9100` | Tarmoq printeri | To'g'ridan-to'g'ri LAN |
| `XP-80C` (nom) | Windows drayveri | Monoblokning o'z printeri |

⚠️ Nom yozilgan bo'lsa, printer **9100 ga javob bermasligi mumkin** — va
skaner uni topmaydi. Bu nosozlik emas, boshqa ulanish usuli.

---

## 7. Monobloklar

### 7.1. Har birida

1. Windows yangilanishlarini **oldindan** tugating ⚠️ (aks holda ish o'rtasida
   qayta yuklanadi)
2. Uyqu rejimini **o'chiring** — `Параметры → Питание → Никогда`
3. Statik IP qo'ying (§3.2)
4. Keel installerini o'rnating
5. Filialni tanlang, qurilmani bog'lang
6. Ekran turini tanlang: **kassa** yoki **zal**

### 7.2. Qaysi mashina nima

| Mashina | Ekran | Kim ishlatadi |
|---|---|---|
| Kassa | `/kassa` | Kassir — pul, chek, smena |
| Zal | `/zal` | Ofitsiant — stollar, buyurtma |
| Oshxona | `/staff/kitchen` | Oshpaz — KDS |
| Kiosk | `/kiosk` | Mehmon — o'zi buyurtma |

### 7.3. Foydali tugmalar

- **`Ctrl+Shift+P`** — printer sozlamalari (kassa ekranida)
- **FEED bosib turib yoqish** — printer self-test

---

## 8. Keel'ga printerlarni kiritish

Kassa ekrani → **Sozlamalar → Printerlar** (yoki `Ctrl+Shift+P`):

1. **Qo'shish** → LAN
2. IP va port: `192.168.1.21`, `9100`
3. Nomi: `Oshxona`
4. ⚠️ **Qaysi chekni chiqaradi** — belgilanmasa printer **hech nima
   chiqarmaydi**. Bu eng ko'p uchraydigan xato: printer ulangan, sinov ishlaydi,
   haqiqiy buyurtmada qog'oz yo'q.
5. **Qaysi bo'limlarni chiqaradi** — bar → ichimliklar, somsaxona → somsalar
   ⚠️ Hech nima belgilanmasa — **hamma bo'lim** chiqadi
6. **Sinov** tugmasi → chek chiqishi kerak

⚠️ **Bo'limi hech bir printerga belgilanmagan taom hamma oshxona printeriga
yuboriladi** — ya'ni ikki joydan chiqadi. Bu ataylab: yo'qolgan chekdan
ortiqchasi yaxshiroq. Lekin bo'limlarni to'g'ri belgilash kerak.

---

## 9. Pul yashigi, skaner, fiskal

### 9.1. Pul yashigi

Kassa chek printeriga **RJ11 kabel** bilan ulanadi (telefon konnektoriga
o'xshaydi, lekin telefon emas).

⚠️ **Kompyuterga emas, printerga.** Printer chekni chiqarganda yashikni ochish
buyrug'ini yuboradi.

Keel'da: printer sozlamasida **"Pul yashigi"** belgilanadi. ⚠️ U faqat **kassir
cheki**da ochiladi — har starterda ochiladigan yashik — bu vilka bilan tirab
qo'yiladigan yashik.

### 9.2. Markirovka skaneri (DataMatrix)

- **2D** skaner kerak (1D shtrix-kod skaneri **DataMatrix o'qimaydi**)
- USB, klaviatura rejimida (HID)
- Ulanadi va darrov ishlaydi — drayver kerak emas

Tekshirish: Bloknot oching, ichimlik qopqog'idagi kodni skanerlang — uzun matn
chiqishi kerak.

### 9.3. Fiskal kassa

Alohida qurilma, o'z tarmog'i yoki USB'si bilan. `docs/` dagi fiskal agent
qo'llanmasiga qarang.

---

## 10. UPS

⚠️ **Kamida switch va router uchun.** Svet bir soniyaga o'chsa:

- Router qayta yuklanadi — 2 daqiqa
- Switch qayta yuklanadi — 1 daqiqa
- Monobloklar qayta yuklanadi — 3 daqiqa
- **Restoran 5 daqiqa buyurtma qabul qilolmaydi**

UPS bu zanjirni uzadi. Kassa monoblokiga ham qo'ying.

---

## 11. Kun bo'yicha tartib

**1-kun — tarmoq**
1. Shkaf joyini tanlash, switch va UPS mahkamlash
2. Kabel tortish, belgilash, tester bilan tekshirish
3. Routerni sozlash: DHCP `.100` dan, statik diapazon bo'sh
4. Wi-Fi: ishchi + mehmon (alohida)

**2-kun — qurilmalar**
5. Har printerga self-test → IP'sini bilish
6. IP'larni to'g'rilash (§4.2)
7. Skanerlash bilan tasdiqlash (§5.2)
8. Monobloklarga Windows sozlamalari + statik IP
9. Keel o'rnatish, filial va qurilma bog'lash

**3-kun — sozlash va o'qitish**
10. Printerlarni Keel'ga kiritish, bo'limlarni taqsimlash
11. Har biriga sinov chek
12. Menyu, narxlar, xodimlar, PIN kodlar
13. **Haqiqiy buyurtma** bilan uchidan uchiga sinash
14. Kassir va ofitsiantlarni o'qitish

---

## 12. Topshirishda qoldiriladigan narsalar

Shkafga yopishtiriladigan varaq:

```
TARMOQ
  Router:        192.168.1.1     (login: ____  parol: ____)
  Switch:        ____ portli, ____ da turadi
  Wi-Fi (ishchi): ____________   parol: ____________
  Wi-Fi (mehmon): ____________   parol: ____________

MONOBLOKLAR
  Kassa 1        192.168.1.10
  Kassa 2        192.168.1.11
  Zal 1          192.168.1.12
  Zal 2          192.168.1.13

PRINTERLAR                          MAC
  Kassa          192.168.1.20      __-__-__-__-__-__
  Oshxona        192.168.1.21      __-__-__-__-__-__
  Bar            192.168.1.22      __-__-__-__-__-__
  Somsaxona      192.168.1.23      __-__-__-__-__-__
  Salatxona      192.168.1.24      __-__-__-__-__-__

Keel panel:  https://______.keel.uz
Qo'llab-quvvatlash: @keeluz
```

⚠️ **MAC manzil printerning o'zgarmas raqami.** IP o'zgarib ketsa, MAC bo'yicha
topasiz: `arp -a`.

---

## 13. Eng ko'p uchraydigan nosozliklar

| Alomat | Sabab | Yechim |
|---|---|---|
| Chek umuman chiqmaydi | Printerda "qaysi chek" belgilanmagan | §8, 4-qadam |
| Sinov ishlaydi, haqiqiy chek yo'q | Bo'lim belgilanmagan yoki kassa dasturi yopiq | §8, 5-qadam |
| Bitta printer ishlaydi, ikkinchisi yo'q | Boshqa tarmoqda | §4.4 |
| Skanerlash hech nima topmadi | Noto'g'ri tarmoq, yoki port band | §5.2, §5.4 |
| "Printer ba'zan ishlaydi" | DHCP IP'ni almashtirgan, yoki kabel 90 m dan uzun | §3.1, §2.3 |
| Chek kechikib, birdan hammasi chiqadi | Kassa dasturi yopiq edi | ⚠️ Keel 30 daqiqadan eski chekni chiqarmaydi |
| Oshxona ekrani bo'sh | Ofitsiant "Oshxonaga yuborish" bosmagan | O'qitish |
| Pul yashigi ochilmaydi | RJ11 kompyuterga ulangan | §9.1 |
| Skaner kod o'qimaydi | 1D skaner | §9.2 — 2D kerak |
| Kassa ekrani sekin | Wi-Fi'da ishlayapti | §2.5 — simga o'tkazing |

⚠️ **Chek chiqmasa, birinchi savol har doim bitta**: *monoblokdagi Keel kassa
dasturi ochiqmi?* Chekni server emas, **kassa dasturi** chiqaradi.

---

## 14. Nimani joyida aniqlash kerak

Bu savollarga faqat restoranda javob bor:

- Switch qayerga sig'adi va u yerda rozetka bormi
- Kabel devor ichidan o'tadimi yoki kanal kerakmi
- Oshxonada printer turadigan quruq joy bormi
- Kassir turgan joyda nechta rozetka bor
- Internet qayerdan kiradi va provayder routeri qayerda
- Har bir sexda kim turadi — chop etish nuqtalari soni shundan

---

## Qo'shimcha fayllar

- `BAR_PRINTERI.txt` — bitta printer ishlamaganda, noldan
- `ANDROID_BUILD.txt` — ofitsiant ilovasini build qilish
- `README.md` — kassa dasturi haqida
