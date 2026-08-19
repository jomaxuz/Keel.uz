#!/usr/bin/env bash
# Set the version of every part of Keel, in one command.
#
#   scripts/set-version.sh v0.2.0
#
# ⚠️ **Five files, because each binary and bundle carries its own constant.**
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

echo "versiya: $new"
grep -hoE 'v[0-9]+\.[0-9]+\.[0-9]+' VERSION \
  control/internal/handlers/version.go \
  backend/internal/version/version.go \
  keel-site/src/lib/version.ts \
  frontend/src/lib/version.ts | sort -u | sed 's/^/  /'

echo
echo "keyingi qadam: (cd control && go test ./internal/handlers/ -run SameVersion)"
