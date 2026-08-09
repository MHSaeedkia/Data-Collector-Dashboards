# TODO

## Done

- [x] Dockerfile for `alert-gateway` (multi-stage, static binary, non-root, healthcheck).
- [x] `alert-gateway` service added to `docker-compose.monitoring.yml`.
- [x] Structured logging (`log/slog`) with per-request `req_id`, wired to `LOG_LEVEL`;
      API response bodies now included in send errors.

## Open

- [ ] Run `docker compose -f docker-compose.monitoring.yml build alert-gateway` once the
      Docker daemon is available — the image build has not been executed yet.
- [ ] Set a real `TELEGRAM_PROXY_URL` in `alert-gateway/.env` (or leave it empty);
      it is currently the `<proxy-host>` placeholder and will break Telegram sends
      while Grafana still reports the contact point as healthy.
- [ ] Point the Grafana Webhook contact point at `http://alert-gateway:8080/alert`
      (container port, not the published 8085).

## NiFi logs in Grafana (Loki + Alloy)

Added 2026-08-09 — see `memory/nifi-log-pipeline.md`. Nothing below has been run;
the Docker daemon was unavailable when it was written.

- [x] `loki` + `alloy` services, `loki/config.yaml`, `alloy/config.alloy`,
      `grafana/NiFi-Logs/dashboard.json`.
- [ ] Confirm the external volume name resolves:
      `docker volume ls | grep nifi-logs` should show
      `data-collector_data-collector-nifi-logs`. If the other project was started
      with a different `-p`/directory name, fix the `name:` in the compose volumes block.
- [ ] Bring it up, then check the Alloy UI at `http://localhost:12345` — the
      `loki.source.file` components should show the NiFi files as active targets.
- [ ] Add the Loki datasource in Grafana (`http://loki:3100`) and import
      `grafana/NiFi-Logs/dashboard.json`.
- [ ] Verify a stack trace arrives as ONE log entry, not 30. If not, the multiline
      firstline regex needs adjusting to the real logback pattern.
- [ ] Verify `stage.structured_metadata` is accepted by the pinned Alloy version —
      it is the least certain part of the config. If it errors, drop the stage;
      thread/logger remain searchable in the line text.
- [ ] Bump `grafana/loki:3.3.2` and `grafana/alloy:v1.5.1` to current releases.
- [ ] Watch disk use for the first week. NiFi at DEBUG can produce GBs/day; the
      `ingestion_rate_mb` cap will show up as 429s in the Alloy logs if hit.

## Intermittent-delivery investigation

All three suspected causes were fixed on 2026-08-09 — see
`memory/alert-gateway-observability.md`. Which one actually caused the reported
symptom was never established, because they were fixed before a failing production
log was captured.

- [x] Fixed `senderCache` data race (`sync.Mutex`); verified with `-race` under
      40 concurrent cold-cache webhooks.
- [x] Escape HTML in `formatMessage` via `html.EscapeString`.
- [x] Per-send 15s timeout instead of one shared batch budget.
- [ ] **Confirm the fix actually held** — watch the logs over the next real alert
      storm for `send failed`. If failures persist, the cause is something else.
- [ ] **Verify Bale accepts `parse_mode=HTML`** — it was added so escaping renders
      correctly, but was only tested against a local fake server, never the real
      Bale API. A rejection now shows in the logs with the response body.
- [ ] Consider building senders at startup instead of lazily, so a bad token fails
      loudly at boot rather than on the first real alert.
- [ ] Consider returning non-2xx when every send fails, so Grafana's contact-point
      health stops reporting green on total delivery failure.
