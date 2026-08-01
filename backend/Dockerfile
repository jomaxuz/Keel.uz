# ---- build ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Ship the maintenance commands too (adminreset, seedmenu) so they can be run
# with `docker compose exec backend /app/adminreset ...`.
RUN CGO_ENABLED=0 go build -o /out/ ./cmd/...

# ---- run ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/ /app/
RUN mkdir -p /app/uploads
EXPOSE 8080
CMD ["/app/server"]
