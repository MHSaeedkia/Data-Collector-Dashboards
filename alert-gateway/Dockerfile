# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/alert-gateway .

FROM alpine:3.21
# ca-certificates is required for the HTTPS calls to Telegram / Bale.
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder /out/alert-gateway /usr/local/bin/alert-gateway
COPY config/ ./config/

USER app
EXPOSE 8080
ENV LISTEN_ADDR=:8080 \
    CONFIG_PATH=/app/config/config.yaml

ENTRYPOINT ["alert-gateway"]
