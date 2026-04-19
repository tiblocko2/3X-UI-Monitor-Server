FROM golang:1.21-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o vpn-monitor .

# ── Final image ──────────────────────────────────────────────────────────────
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/vpn-monitor .

EXPOSE 8080

ENV PORT=8080
ENV DATA_DIR=/var/lib/vpn-monitor

VOLUME ["/var/lib/vpn-monitor"]

CMD ["./vpn-monitor"]
