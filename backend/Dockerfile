# ---- build ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Ship the maintenance commands too (adminreset, seedmenu) so they can be run
# with `docker compose exec backend /app/adminreset ...`.
#
# ⚠️ **`nodynamic` is not optional, and forgetting it fails at runtime rather
# than here.** Every uploaded photograph is converted to WebP; without this tag
# the encoder is reached by dlopen-ing a system libwebp, which this image does
# not have — so the binary builds, starts, serves, and quietly stores JPEGs
# instead. Same family as the alpine timezone trap: nothing errors, the result
# is simply worse for months. The server logs which encoder it got at boot.
RUN CGO_ENABLED=0 go build -tags nodynamic -o /out/ ./cmd/...

# ---- run ----
FROM alpine:3.20
# tzdata so the container clock (and `date`) match TZ; the Go binary also
# embeds the database itself — see cmd/server/main.go.
# su-exec is how the entrypoint drops root after fixing the uploads directory.
RUN apk add --no-cache ca-certificates tzdata su-exec
# The account the server actually runs as. A fixed uid rather than whatever
# adduser picks: it is baked into files on a host volume, and a uid that
# shifted between image builds would leave a customer's photographs unwritable.
RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app
WORKDIR /app
COPY --from=build /out/ /app/
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh && \
	mkdir -p /app/uploads && chown -R app:app /app/uploads
EXPOSE 8080
# Starts as root only long enough to hand /app/uploads to `app`, then drops.
# The server process never runs privileged — see docker-entrypoint.sh.
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/app/server"]
