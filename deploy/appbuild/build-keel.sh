#!/usr/bin/env bash
# ---- Keel (uz.keel.app): the staff app, built on the server ----
#
# The waiter, the owner, the courier and the team in one app — see
# `mobile/keel-android/README.md`. Unlike `build.sh` there is one of it, not one
# per restaurant: no brand, no slug, one signing key for every phone.
#
# Runs inside the `keel-appbuild` image, beside the guest builds:
#
#   docker run --rm \
#     -v /opt/keel:/opt/keel \
#     -v keel-gradle-cache:/root/.gradle \
#     keel-appbuild:latest \
#     /opt/keel/deploy/appbuild/build-keel.sh apk
#
# ⚠️ **The signing key is made once and never again** — the same rule as the
# guest apps, for the same reason: an app signed by a lost key can never be
# updated, by anybody. It lives in `appkeys/keel-app/` beside theirs, which is
# the one directory a deploy must never touch.
#
# ⚠️ **Built in a copy, never in the checkout.** This runs as root inside the
# container; a `build/` left in `/opt/keel` would be root's, and the next
# deploy's `git reset --hard` (run as `deploy-keel`) stops half-way through it
# with `HEAD` unmoved — the trap CLAUDE.md describes, arriving from a new door.
set -euo pipefail

FORMAT="${1:-apk}"
case "$FORMAT" in
  apk|aab) ;;
  *) echo "format must be apk or aab" >&2; exit 2 ;;
esac

: "${KEEL_APP_ROOT:=/opt/keel}"
: "${KEEL_APP_KEYS:=/opt/keel/appkeys}"
: "${KEEL_APP_OUT:=/opt/keel/appbuilds}"

# ⚠️ **The guest builds' lock, not one of its own.** The point of the lock is the
# machine's memory, and two Gradles at once swap it whichever app they build.
: "${KEEL_APP_LOCK:=/opt/keel/.appbuild.lock}"
if [ "${KEEL_APP_LOCKED:-}" != "1" ]; then
  export KEEL_APP_LOCKED=1
  exec flock --timeout 3600 "$KEEL_APP_LOCK" "$0" "$@"
fi

say() { printf '\n== %s\n' "$*"; }

SRC="$KEEL_APP_ROOT/mobile/keel-android"
[ -d "$SRC" ] || { echo "no Keel app at $SRC" >&2; exit 1; }
# ⚠️ Without it the build still succeeds — and ships with push switched off
# (`firebaseReady` in app/build.gradle.kts). Said here, before ten minutes of
# Gradle, rather than discovered on a phone that never rings.
if ! grep -q '"uz.keel.app"' "$SRC/app/google-services.json" 2>/dev/null; then
  echo "!! app/google-services.json has no uz.keel.app client — this build will have NO push" >&2
fi

# ---- The signing key ----
KEYDIR="$KEEL_APP_KEYS/keel-app"
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
    -dname "CN=Keel, OU=Keel, O=Keel, L=Tashkent, C=UZ"
  chmod 600 "$KEYSTORE"
  ( umask 077
    cat > "$KEYPROPS" <<PROPS
storeFile=$KEYSTORE
storePassword=$STOREPASS
keyAlias=keel
keyPassword=$STOREPASS
PROPS
  )
  cat >&2 <<NOTE
!! A NEW SIGNING KEY was created at $KEYSTORE
!! Back it up off this machine. Without it uz.keel.app can never be updated
!! again — not by us, not by Google.
NOTE
else
  say "using the existing signing key"
fi

# ---- A copy to build in ----
# The design module is included by relative path (`../android-design`) and the
# version is read from the repo root (`../../VERSION`), so both keep their
# places relative to the app.
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$WORK/mobile"
cp -r "$SRC" "$WORK/mobile/keel-android"
cp -r "$KEEL_APP_ROOT/mobile/android-design" "$WORK/mobile/android-design"
cp "$KEEL_APP_ROOT/VERSION" "$WORK/VERSION"
rm -rf "$WORK/mobile/keel-android/app/build" "$WORK/mobile/keel-android/.gradle"
VERSION="$(tr -d '[:space:]' < "$WORK/VERSION")"

# ---- Build ----
say "building Keel $VERSION ($FORMAT)"
TASK="assembleRelease"
ART="app/build/outputs/apk/release/app-release.apk"
if [ "$FORMAT" = "aab" ]; then
  TASK="bundleRelease"
  ART="app/build/outputs/bundle/release/app-release.aab"
fi

cd "$WORK/mobile/keel-android"
echo "sdk.dir=${ANDROID_HOME:-/opt/android-sdk}" > local.properties
# The same two flags as the guest builds, for the same reasons (see build.sh):
# no SDK downloads mid-build, and one JVM inside the container's memory.
KEEL_KEYSTORE_PROPERTIES="$KEYPROPS" \
  ./gradlew --no-daemon \
    -Pandroid.builder.sdkDownload=false \
    -Pkotlin.compiler.execution.strategy=in-process \
    "$TASK"

STAMP="$(date +%Y%m%d-%H%M%S)"
DEST="$KEEL_APP_OUT/keel-app"
mkdir -p "$DEST"
chmod 755 "$DEST"
OUT="$DEST/keel-$VERSION-$STAMP.$FORMAT"
cp "$ART" "$OUT"
chmod 644 "$OUT"

say "done: $OUT"
printf 'KEEL_ARTIFACT=%s\n' "$OUT"
