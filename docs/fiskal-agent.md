# Fiskal ulagichni kassa kompyuteriga o'rnatish

Bu — restoranga beriladigan qo'llanma. Ulagich (`fiscalagent.exe`) kassa
kompyuterida ishlaydi va cheklarni soliq qo'mitasiga yuboradi.

## Kerak bo'ladigan narsalar

1. **Kassa kompyuteri** — fiskal dastur (Multikassa / Rahmat POS) o'rnatilgan
   Windows kompyuteri.
2. **Kalit** — admin panelda: Sozlamalar → Fiskal kassa → "Kalit yaratish".
   ⚠️ Kalit **bir marta** ko'rsatiladi. Nusxa oling.
3. **Internet** — kompyuterdan chiquvchi ulanish. **Hech qanday port ochish
   shart emas.**

## 1. Faylni joylashtirish

`fiscalagent.exe` ni kassa kompyuteriga ko'chiring, masalan:

```
C:\Keel\fiscalagent.exe
```

## 2. Sinab ko'rish

Buyruq satrini (`cmd`) oching va ishga tushiring:

```
C:\Keel\fiscalagent.exe -server https://SIZNING-DOMEN/api/v1 -token KALIT -v
```

To'g'ri ishlasa quyidagicha yozadi:

```
fiskal agent ishga tushdi: https://SIZNING-DOMEN/api/v1
```

Endi kassa ekranida bitta chek yoping — oynada `chek ...: yuborildi` chiqishi
kerak.

### Xatolar nimani anglatadi

| Yozuv | Sabab | Nima qilish |
|---|---|---|
| `agent kaliti qabul qilinmadi` | Kalit noto'g'ri yoki almashtirilgan | Paneldan yangi kalit oling |
| `kassa dasturiga ulanib bo'lmadi` | Fiskal dastur ishlamayapti yoki manzil noto'g'ri | Dasturni ishga tushiring; panelda manzilni tekshiring |
| `kassa dasturi vaqtida javob bermadi` | Dastur osilgan | Dasturni qayta ishga tushiring |
| `smenani ochib bo'lmadi` | Kassa smenasi ochilmadi | Fiskal dasturdan smenani qo'lda oching |
| `serverdan ish so'rab bo'lmadi` | Internet yo'q | Ulanishni tekshiring — ulagich o'zi qayta uriniб turadi |

⚠️ Panelda **"Oxirgi marta ulandi"** qatoriga qarang: u yangilanayotgan bo'lsa
ulagich ishlayapti. Yashil "ulangan" nishoni yo'q, chunki u soat o'tishi bilan
eskiradi va yolg'on gapira boshlaydi.

## 3. Avtomatik ishga tushirish

Kompyuter qayta yuklangandan keyin ulagich **o'zi** ishga tushishi kerak, aks
holda smena o'rtasida jimgina to'xtaydi va buni hech kim sezmaydi.

### Variant A — Vazifalar rejalashtiruvchisi (oddiyroq)

Administrator sifatida `cmd` da:

```
schtasks /create /tn "Keel fiskal agent" /sc onstart /ru SYSTEM /rl HIGHEST ^
  /tr "C:\Keel\fiscalagent.exe -server https://SIZNING-DOMEN/api/v1 -token KALIT"
```

Tekshirish:

```
schtasks /query /tn "Keel fiskal agent"
```

⚠️ **`/sc onstart` — kompyuter yoqilganda**, foydalanuvchi kirganda emas.
Kassa kompyuteri ko'pincha kirilmagan holda turadi.

### Variant B — Windows xizmati (barqarorroq)

[NSSM](https://nssm.cc) yordamida:

```
nssm install KeelFiscal C:\Keel\fiscalagent.exe
nssm set KeelFiscal AppParameters "-server https://SIZNING-DOMEN/api/v1 -token KALIT"
nssm set KeelFiscal Start SERVICE_AUTO_START
nssm set KeelFiscal AppStdout C:\Keel\agent.log
nssm set KeelFiscal AppStderr C:\Keel\agent.log
nssm start KeelFiscal
```

Xizmat o'zi qayta ishga tushadi va jurnal `C:\Keel\agent.log` da qoladi.

⚠️ **Kalitni buyruq satrida ko'rinmasin desangiz** muhit o'zgaruvchilaridan
foydalaning: `KEEL_SERVER` va `KEEL_AGENT_TOKEN`. Ulagich ikkalasini ham
o'qiydi.

## 4. Yangilash

Ulagichni to'xtating, `.exe` ni almashtiring, qaytadan ishga tushiring:

```
nssm stop KeelFiscal
copy /Y yangi\fiscalagent.exe C:\Keel\fiscalagent.exe
nssm start KeelFiscal
```

⚠️ **Yangilash paytida yopilgan cheklar yo'qolmaydi.** Yuborilmagan chek
serverda saqlanadi va ulagich qaytgach o'zi oladi — navbat ulagichning ichida
emas, serverda.

## Ulagichsiz ham ishlaydimi?

Ha. Ulagich bo'lmasa cheklarni **kassa ekranining o'zi** yuboradi, lekin
buning uchun ekran fiskal dastur bilan bir tarmoqda bo'lishi kerak — eng
ishonchlisi kassa ekranini **o'sha kompyuterda** ochish va panelda manzilni
`http://localhost:8080` qilib yozish.

Ulagich uchta narsani yechadi:
- kassa ekranini istalgan qurilmada ochish mumkin bo'ladi;
- brauzer sozlamalariga tegish shart emas;
- **kassa ekrani umuman ochiq bo'lmasa ham** cheklar yuboriladi (masalan
  kassir tabni yopib qo'ygan bo'lsa).

## Xavfsizlik

- Ulagich faqat **chiquvchi** ulanish qiladi. Tarmoqda port ochilmaydi.
- Kalit faqat shu filialning cheklariga tegishli.
- Kalit yo'qolsa paneldan yangisini yarating — eskisi **shu zahoti** ishlamay
  qoladi.
