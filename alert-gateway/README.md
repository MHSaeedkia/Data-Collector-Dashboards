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

## Logs

Logging is structured (`log/slog`, text format on stdout). `LOG_LEVEL` in `.env`
accepts `debug`, `info` (default), `warn`, `error`.

Every webhook call gets a random `req_id`, and every line produced while handling
that call carries it — so one alert's whole journey can be followed with:

```bash
docker compose -f ../docker-compose.monitoring.yml logs alert-gateway | grep req_id=abc123
```

The line worth watching is `webhook completed`, which reports `alerts`, `sent`,
`failed` and `duration_ms` for the call. `sent=0 failed=N` means Grafana reached
the service but delivery failed; no `webhook received` line at all means Grafana
never called it.

At `debug` level two extra lines are emitted per call: `raw payload` (the exact
JSON Grafana sent) and `formatted message` (the exact text handed to the senders).

Failed sends include the API's own response body in the error, e.g.
`telegram: unexpected response 400: {"ok":false,...,"description":"Bad Request: can't parse entities"}`.

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
