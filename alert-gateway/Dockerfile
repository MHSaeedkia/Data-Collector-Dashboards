# syntax=docker/dockerfile:1

FROM golang:1.25.7-alpine AS builder
WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /out/alert-gateway .

FROM alpine:3.21
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
