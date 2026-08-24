#!/usr/bin/env bash
# Set the version of every part of Keel, in one command.
#
#   scripts/set-version.sh v0.2.0
#
# ⚠️ **Seven files, because each binary and bundle carries its own constant.**
# That is deliberate (see control/internal/handlers/version.go): a version read
# at runtime can disagree with the code that is running. The cost is that they
# have to be changed together, and "together" done by hand is the thing that
# eventually is not — so it is done here instead, and the test only has to catch
# the case where somebody edited one by hand anyway.
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "ishlatilishi: $0 vX.Y.Z" >&2
  exit 2
fi
new=$1

# ⚠️ Checked, not trusted. A typo here is written to five files and tagged onto
# a release, and "v.0.2" reads as a version right up until something sorts it.
if [[ ! $new =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "versiya vX.Y.Z ko'rinishida bo'lsin (masalan v0.2.0), berilgani: $new" >&2
  exit 2
fi

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

echo "$new" > VERSION
sed -i -E "s/(Version = \")[^\"]+(\")/\1$new\2/" \
  control/internal/handlers/version.go \
  backend/internal/version/version.go
sed -i -E "s/(export const VERSION = \")[^\"]+(\")/\1$new\2/" \
  keel-site/src/lib/version.ts \
  frontend/src/lib/version.ts

# ⚠️ Windows file properties want a bare number, and a four-part one for the
# fixed field. Same version, the shape the platform insists on.
bare=${new#v}
sed -i -E "s/(\"ProductVersion\": \")[^\"]+(\")/\1$bare\2/;s/(\"file_version\": \")[^\"]+(\")/\1$bare.0\2/" \
  backend/desktop/build/windows/info.json

# ⚠️ **The till's own constant, and the only one a missed release fails
# silently.** The others are read by a person asking "which version is this?";
# this one is read by the updater, which compares it against the manifest and
# concludes every till is already current. Bare, like info.json beside it: both
# are Windows-facing, and the manifest is written by hand in the same shape.
sed -i -E "s/(const Version = \")[^\"]+(\")/\1$bare\2/" \
  backend/desktop/version.go

echo "versiya: $new"
grep -hoE 'v[0-9]+\.[0-9]+\.[0-9]+' VERSION \
  control/internal/handlers/version.go \
  backend/internal/version/version.go \
  keel-site/src/lib/version.ts \
  frontend/src/lib/version.ts | sort -u | sed 's/^/  /'

echo
echo "keyingi qadam: (cd control && go test ./internal/handlers/ -run SameVersion)"
