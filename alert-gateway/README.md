# Alert Gateway

A bridge service between Grafana Alerting and messenger channels (Telegram, Bale).
Grafana only knows about a single Webhook Contact Point that POSTs to this service;
based on the rules in `config/config.yaml` the service decides which channel(s)
each alert goes to, then sends it there.

## Running

```bash
cp .env.example .env
# fill in the real token/proxy values in .env
go run .
```

The server comes up on `LISTEN_ADDR` (default `:8080`).

## Connecting to Grafana

1. Alerting → Contact points → New contact point
2. Type: `Webhook`
3. URL: `http://<service-host>:8080/alert`
4. Select this Contact Point in the relevant Notification Policy.

## Adding a new channel (e.g. email)

1. Create a new file in `internal/senders/` that implements the `Sender` interface.
2. Add a new `case` to `New()` in `internal/senders/sender.go`.
3. Define a new channel with `type: email` in `config/config.yaml` and read its
   values from `.env` as `${VAR}`.

No secret or environment-specific value should ever be written directly in `config.yaml`.

## Test

```bash
go test ./...
```

## Project structure

```
main.go                        entry point, reads .env and the config, runs the HTTP server
config/config.yaml              routing rules (secrets as ${VAR})
internal/router/config.go       YAML loading + env var expansion
internal/router/router.go       rule → channel matching logic
internal/senders/sender.go      shared Sender interface
internal/senders/telegram.go    sending to Telegram (with HTTP proxy support)
internal/senders/bale.go        sending to Bale
internal/handler/handler.go     receives the Grafana webhook, formats the message, calls router+sender
internal/handler/grafana_payload.go   Grafana webhook JSON structure
```
