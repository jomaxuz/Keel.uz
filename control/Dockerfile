# ---- build ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ ./cmd/...

# ---- run ----
FROM alpine:3.20
# tzdata so the container clock and `date` agree with TZ. The Go binary embeds
# the database as well (`import _ "time/tzdata"` in cmd/server), which is what
# actually protects the billing boundaries — but a container whose own clock
# disagrees with the process inside it is a debugging trap of its own.
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/ /app/
EXPOSE 9000
CMD ["/app/server"]
