#!/usr/bin/env bash
# ---- One restaurant's app, built from the one codebase ----
#
# Reads what the restaurant looks like from the restaurant's **own** API, writes
# the single file that brands the build, makes the icons, makes sure a signing
# key exists, and produces an APK or an AAB.
#
# ⚠️ **The values are never retyped in the console.** `name`, `logoUrl` and
# `theme.brand` are already the restaurant's answer to "what do we look like",
# and a second copy in the build system is a second thing that can disagree with
# the site the guest saw first.
#
# ⚠️ **The signing key is made once and never again.** An app on the store signed
# by a lost key can never be updated: not by us, not by the restaurant, not by
# Google. So this script creates one if there is none and otherwise leaves the
# existing one entirely alone — there is deliberately no way to ask it for a new
# one.
#
# ⚠️ **One build at a time, and the lock is on an absolute path.** Gradle, the
# Kotlin compiler and the Android tooling together want more memory than this
# machine has spare while it is also running every tenant. `${HOME}` would lock
# the *user* rather than the machine — the mistake the deploy script already
# paid for, where CI and a person held two different locks and neither ever met
# the other.
set -euo pipefail

SLUG="${1:?usage: build.sh <slug> [apk|aab]}"
FORMAT="${2:-apk}"

: "${KEEL_APP_ROOT:=/opt/keel}"
: "${KEEL_APP_KEYS:=/opt/keel/appkeys}"
: "${KEEL_APP_OUT:=/opt/keel/appbuilds}"
: "${KEEL_APP_HOST:=https://${SLUG}.keel.uz}"
# Shared across every restaurant's build: the Firebase project and the maps key
# are Keel's, and only the per-app id differs.
: "${KEEL_MAPS_KEY:=}"
: "${KEEL_FB_PROJECT_ID:=}"
: "${KEEL_FB_API_KEY:=}"
: "${KEEL_FB_SENDER_ID:=}"
: "${KEEL_FB_APP_ID:=}"

case "$FORMAT" in
  apk|aab) ;;
  *) echo "format must be apk or aab" >&2; exit 2 ;;
esac

# ---- One build at a time ----
#
# ⚠️ **The lock is here rather than in whatever calls this.** The console will
# queue builds, but a lock that only exists in the caller is a lock that a person
# with a shell walks straight past — and the collision this prevents is exactly
# "somebody runs it by hand while the console is running it".
#
# ⚠️ **An absolute path, never `${HOME}`.** The deploy script paid for this
# lesson: `${HOME}/...` locks the *user*, so CI (as `deploy-keel`) and a person
# (as `root`) take two different files, each held perfectly, protecting nothing —
# and the one collision it existed to prevent is the one it cannot see. A lock
# nobody contends is indistinguishable from a lock that works.
#
# ⚠️ **Waits rather than refuses.** Gradle and the Kotlin compiler want more
# memory than this machine has spare beside the restaurants, so two at once is a
# machine that swaps; but a build refused outright is a button in the console
# that sometimes does nothing.
: "${KEEL_APP_LOCK:=/opt/keel/.appbuild.lock}"
if [ "${KEEL_APP_LOCKED:-}" != "1" ]; then
  export KEEL_APP_LOCKED=1
  exec flock --timeout 3600 "$KEEL_APP_LOCK" "$0" "$@"
fi

SRC="$KEEL_APP_ROOT/mobile/guest-android"
[ -d "$SRC" ] || { echo "no guest app at $SRC" >&2; exit 1; }

say() { printf '\n== %s\n' "$*"; }

# ---- What this restaurant looks like ----
say "reading $KEEL_APP_HOST"
PROFILE="$(curl -fsS --max-time 30 "$KEEL_APP_HOST/api/v1/restaurant")" || {
  echo "the restaurant did not answer — is $SLUG running?" >&2
  exit 1
}
# ⚠️ **The answer is wrapped, and reading `.name` off the top gives null.** The
# response is `{restaurant, brand, branch, design, ...}`, and `restaurant` is
# the merged view: the company document with the brand laid over it and the
# serving branch over that (`applyBrand` in handlers/public.go). It is the face
# the guest sees, which is exactly what an app is branded from.
#
# Silently null, at that: jq is happy to find nothing, and the build would ship
# an app named after the slug in Keel's own orange.
NAME="$(printf '%s' "$PROFILE" | jq -r '.restaurant.name // ""')"
LOGO="$(printf '%s' "$PROFILE" | jq -r '.restaurant.logoUrl // ""')"
ACCENT="$(printf '%s' "$PROFILE" | jq -r '.restaurant.theme.brand // ""')"
[ -n "$NAME" ] || NAME="$SLUG"
# ⚠️ Empty falls back to Keel's orange rather than failing the build: a
# restaurant that never opened the theme screen still gets an app, and a colour
# is the cheapest thing in here to change later.
[ -n "$ACCENT" ] || ACCENT="#E2590D"

# ⚠️ **The application id is derived from the slug and never from the name.** A
# restaurant renames itself; an application id cannot change without becoming a
# second app installed beside the first, leaving every guest with a dead copy.
# The slug is the one thing about a tenant that is already permanent.
APP_ID="uz.keel.app.$(printf '%s' "$SLUG" | tr -cd '[:alnum:]' | tr '[:upper:]' '[:lower:]')"

say "$NAME → $APP_ID ($ACCENT)"

# ⚠️ **Copied out of the checkout, never built in it.** Two builds in the same
# tree would overwrite each other's `brand.properties` and ship one restaurant's
# app under another's name — and a failed build would leave the checkout carrying
# somebody's colours until the next one happened to fix it.
#
# ⚠️ **The design module comes too, and keeps its position.** It is included by
# relative path (`../android-design`) and shared by all six apps; copying the app
# alone gives "Configuring project ':design' without an existing directory",
# which is a clear enough message but only after the copy has already happened.
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$WORK/guest-android"
cp -a "$SRC/." "$WORK/guest-android/"
cp -a "$KEEL_APP_ROOT/mobile/android-design" "$WORK/android-design"
APP="$WORK/guest-android"
# ⚠️ Never the checkout's own build directory: a stale one from another
# restaurant is how an app ships with somebody else's icon.
rm -rf "$APP/build" "$APP/app/build" "$APP/.gradle" "$WORK/android-design/build"

# ---- The icons ----
if [ -n "$LOGO" ]; then
  LOGO_URL="$LOGO"
  case "$LOGO" in http*) ;; *) LOGO_URL="$KEEL_APP_HOST$LOGO" ;; esac
  say "logo $LOGO_URL"
  if curl -fsS --max-time 30 -o "$APP/logo.src" "$LOGO_URL"; then
    python3 "$KEEL_APP_ROOT/deploy/appbuild/brandimages.py" \
      "$APP/logo.src" "$APP/app/src/main/res"
  else
    # ⚠️ **A missing logo is a warning, not a failure.** The placeholder mark
    # ships, the app works, and the restaurant can be sent one line about it —
    # which is a far better outcome than a build nobody can explain.
    echo "!! the logo could not be fetched — keeping the placeholder" >&2
  fi
else
  echo "!! this restaurant has no logo — keeping the placeholder" >&2
fi

# ---- The signing key ----
KEYDIR="$KEEL_APP_KEYS/$SLUG"
KEYSTORE="$KEYDIR/release.jks"
KEYPROPS="$KEYDIR/keystore.properties"
mkdir -p "$KEYDIR"
chmod 700 "$KEYDIR"
if [ ! -f "$KEYSTORE" ]; then
  say "minting a signing key — this happens once, ever"
  STOREPASS="$(head -c 24 /dev/urandom | base64 | tr -d '/+=' | head -c 28)"
  keytool -genkeypair -noprompt \
    -alias keel -keyalg RSA -keysize 4096 -validity 10000 \
    -keystore "$KEYSTORE" -storepass "$STOREPASS" -keypass "$STOREPASS" \
    -dname "CN=$NAME, OU=Keel, O=Keel, L=Tashkent, C=UZ"
  # ⚠️ **Tightened explicitly.** `keytool` writes the store 0644 whatever the
  # umask, so a keystore left as it lands is world-readable on a machine several
  # people have shells on — and the file is the one thing here that cannot be
  # replaced if it leaks or is lost.
  chmod 600 "$KEYSTORE"
  # ⚠️ **The umask is set in a subshell, not for the rest of the script.** Left
  # to leak it also applied to the APK written six steps later, which arrived
  # 0600 and root-owned — unreadable by the console that has to hand it to
  # somebody, and diagnosed as a permissions bug in the console.
  ( umask 077
    cat > "$KEYPROPS" <<EOF
storeFile=$KEYSTORE
storePassword=$STOREPASS
keyAlias=keel
keyPassword=$STOREPASS
EOF
  )
  # ⚠️ **Say this out loud, every time it happens.** A key nobody knows exists
  # is a key nobody backs up, and the day it is needed is the day it is gone.
  cat >&2 <<EOF
!! A NEW SIGNING KEY was created at $KEYSTORE
!! Back it up off this machine. Without it $APP_ID can never be updated again —
!! not by us, not by the restaurant, not by Google.
EOF
else
  say "using the existing signing key"
fi

# ---- The one file that brands the build ----
cat > "$APP/app/brand.properties" <<EOF
brand.applicationId=$APP_ID
brand.appName=$NAME
brand.serverUrl=$KEEL_APP_HOST
brand.accent=$ACCENT
brand.versionCode=${KEEL_APP_VERSION_CODE:-1}
brand.versionName=${KEEL_APP_VERSION_NAME:-1.0.0}
brand.mapsKey=$KEEL_MAPS_KEY
brand.firebaseAppId=$KEEL_FB_APP_ID
brand.firebaseProjectId=$KEEL_FB_PROJECT_ID
brand.firebaseApiKey=$KEEL_FB_API_KEY
brand.firebaseSenderId=$KEEL_FB_SENDER_ID
EOF

# ---- Build ----
say "building $FORMAT"
TASK="assembleRelease"
ART="app/build/outputs/apk/release/app-release.apk"
if [ "$FORMAT" = "aab" ]; then
  TASK="bundleRelease"
  ART="app/build/outputs/bundle/release/app-release.aab"
fi

cd "$APP"
echo "sdk.dir=${ANDROID_HOME:-/opt/android-sdk}" > local.properties
KEEL_GUEST_KEYSTORE_PROPERTIES="$KEYPROPS" \
  ./gradlew --no-daemon "$TASK"

STAMP="$(date +%Y%m%d-%H%M%S)"
DEST="$KEEL_APP_OUT/$SLUG"
mkdir -p "$DEST"
cp "$ART" "$DEST/$SLUG-$STAMP.$FORMAT"
# ⚠️ Readable on purpose: an APK is the thing being handed out, and the console
# that serves it does not run as root. The keystore beside it is the secret, and
# it lives in a different directory with different permissions.
chmod 644 "$DEST/$SLUG-$STAMP.$FORMAT"
# ⚠️ **One copy, not a stamped one and a `latest` beside it.** The console
# deletes the artifact the moment somebody has downloaded it; a second copy under
# a stable name would survive that deletion and sit on the disk for ever — which
# is the thing the deletion exists to prevent. The console addresses builds by
# id, so nothing needs a stable filename.
[ -f app/src/main/res/play_icon.png ] && cp app/src/main/res/play_icon.png "$DEST/play_icon.png"

say "done: $DEST/$SLUG-$STAMP.$FORMAT"
printf '%s\n' "$DEST/$SLUG-$STAMP.$FORMAT"
