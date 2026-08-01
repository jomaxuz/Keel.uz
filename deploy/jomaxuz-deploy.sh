#!/usr/bin/env bash
# traderbot.uz — avtomatik deploy (GitHub Actions chaqiradi).
#
# Bu skript CI kalitiga "forced command" sifatida bog'langan: o'sha kalit bilan
# ulangan odam faqat shuni ishga tushira oladi — shell ham, port forwarding ham
# yo'q (/root/.ssh/authorized_keys ga qarang).
#
# GitHub tokeni SSH_ORIGINAL_COMMAND orqali keladi. U ish tugashi bilan
# kuchini yo'qotadigan vaqtinchalik token (GITHUB_TOKEN), shuning uchun
# serverda hech qanday doimiy GitHub kaliti saqlanmaydi — tashkilot deploy
# key'larni taqiqlagani bilan muammo yo'q.
set -euo pipefail

REPO_DIR=/opt/jomaxuz
REPO_PATH=jomaxuz/template
BRANCH=main
VHOST=/etc/nginx/sites-available/traderbot.uz
LOCK=/var/lock/jomaxuz-deploy.lock

# Ikkita deploy bir vaqtda ketmasin: ketma-ket push'lar navbatga tursin.
exec 9>"$LOCK"
flock -w 600 9 || { echo "boshqa deploy ketyapti — kutish vaqti tugadi"; exit 1; }

# SSH_ORIGINAL_COMMAND — bu kalit bilan ulangan odam yuborgan matn. Uni
# ko'r-ko'rona token deb qabul qilib bo'lmaydi: forced command tufayli
# ixtiyoriy buyruq shu yerga tushadi, va u git URL'iga qo'shilib ketishi yoki
# quyidagi redaksiya sed'ini buzishi mumkin. Faqat GitHub token shakliga
# mos matn qabul qilinadi, qolgani e'tiborga olinmaydi.
RAW="${SSH_ORIGINAL_COMMAND:-}"
TOKEN=""
if [[ "$RAW" =~ ^gh[a-z]_[A-Za-z0-9]{20,}$ ]]; then
  TOKEN="$RAW"
elif [ -n "$RAW" ]; then
  echo "ogohlantirish: token shakliga mos kelmadi, e'tiborga olinmadi"
fi

cd "$REPO_DIR"

echo "==> git pull ($BRANCH)"
if [ -n "$TOKEN" ]; then
  # Token faqat shu chaqiruvda ishlatiladi va remote'ga yozilmaydi.
  git -c "credential.helper=" pull --ff-only \
      "https://x-access-token:${TOKEN}@github.com/${REPO_PATH}.git" "$BRANCH" \
    2>&1 | sed -E "s/${TOKEN}/***/g"
else
  # Qo'lda ishga tushirish: noutbukdan `ssh -A` bilan.
  GIT_SSH_COMMAND="ssh -o IdentitiesOnly=no -o StrictHostKeyChecking=accept-new" \
    git pull --ff-only "git@github.com:${REPO_PATH}.git" "$BRANCH"
fi
echo "    $(git log --oneline -1)"

echo "==> image'larni yig'ish va servislarni qayta ishga tushirish"
docker compose -f docker-compose.prod.yml up -d --build

echo "==> nginx konfiguratsiyasini sinxronlash"
cp nginx/restaurant.conf "$VHOST"
sed -i 's/restaurant_frontend/traderbot_frontend/g; s/restaurant_backend/traderbot_backend/g' "$VHOST"
# nginx -t muvaffaqiyatsiz bo'lsa reload qilinmaydi — buzuq konfiguratsiya
# bilan qayta yuklash bu serverdagi BOSHQA saytlarni ham yiqitardi.
nginx -t && systemctl reload nginx

echo "==> tekshiruv"

# Bitta manzilni qayta-qayta so'raydi, kutilgan javob kelguncha.
wait_for() {
  local label="$1" want="$2"; shift 2
  local code=""
  for _ in $(seq 1 30); do
    code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$@" || true)
    [ "$code" = "$want" ] && { echo "    $label OK"; return 0; }
    sleep 2
  done
  echo "XATO: $label — kutilgan $want, kelgani $code"
  return 1
}

wait_for "backend " 200 http://127.0.0.1:8090/health
wait_for "frontend" 200 http://127.0.0.1:3100/
# Uchdan-uchgacha: nginx + TLS + upstream. HTTPS orqali, chunki 80-port
# hamma narsani HTTPS ga yo'naltiradi — u yerda 200 hech qachon kelmaydi
# (avvalgi skript aynan shu sababli har deploy da yolg'ondan yiqilardi).
wait_for "nginx   " 200 --resolve traderbot.uz:443:127.0.0.1 https://traderbot.uz/

# Eskirgan image'lar diskni to'ldirmasin.
docker image prune -f >/dev/null 2>&1 || true
echo "==> deploy tugadi: $(git log --oneline -1)"
