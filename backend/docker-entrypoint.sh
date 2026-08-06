#!/bin/sh
# Take ownership of the uploads directory, then stop being root.
#
# The uploads directory is a bind mount or a volume, and Docker creates it
# **root-owned** before the container starts — the container has no say in it.
# So the sequence has to be: start as root, hand the directory to the
# unprivileged user, and drop.
#
# Why it matters here: this server takes file uploads from the internet and
# writes them to disk under names derived from a request. That is precisely the
# code path where a mistake becomes arbitrary file write, and the difference
# between "the app user overwrote a photo" and "root overwrote something else"
# is the whole point. The server process itself never needs root — it binds
# 8080, not 80.
#
# The recursive chown only runs when the directory is not already ours: it is a
# one-time migration for installs that ran as root, not something to repeat on
# every restart of a customer with ten thousand photographs.
set -e

APP_UID=10001
APP_GID=10001
DIR="${UPLOAD_DIR:-/app/uploads}"

if [ "$(id -u)" = "0" ]; then
	mkdir -p "$DIR"
	if [ "$(stat -c %u "$DIR")" != "$APP_UID" ]; then
		echo "uploads: taking ownership of $DIR (one-time)"
		chown -R "$APP_UID:$APP_GID" "$DIR"
	fi
	exec su-exec "$APP_UID:$APP_GID" "$@"
fi

# Already unprivileged — someone set `user:` in compose. Nothing to hand over,
# and trying to chown would fail and take the server down with it.
exec "$@"
