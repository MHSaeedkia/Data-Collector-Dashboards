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

## LPA Staleness Monitoring (renamed from KAFKA-Topic-Staleness-Monitor)

- [x] Folder + title renamed; uid kept as `dfsv0halr3im8d` so links survive.
- [x] Added "Staleness timeline" (state-timeline) and "Episode history per topic"
      panels for the new exporter.py episode metrics.
- [ ] Re-import `grafana/LPA-Staleness-Monitoring/dashboard.json` in Grafana. Import
      over the existing dashboard (same uid) rather than creating a second copy.
- [ ] Confirm the episode metrics are actually being scraped:
      `kafka_topic_stale_episodes_total` should appear in Prometheus. The exporter
      must be redeployed with the new `record_staleness()` first.
- [ ] Sanity-check "Stale since" shows a real date for a stale topic and
      "not stale" for a healthy one — the *1000 / 0-sentinel handling is the part
      most likely to need a tweak.
- [ ] Consider shrinking the original "Kafka Topic Data Freshness" panel (h=31);
      it pushes the new history panels far down the page. Left untouched on purpose.

## NiFi logs in Grafana (Loki + Alloy)

Added 2026-08-09 — see `memory/nifi-log-pipeline.md`. Nothing below has been run;
the Docker daemon was unavailable when it was written.

- [x] `loki` + `alloy` services, `loki/config.yaml`, `alloy/config.alloy`,
      `grafana/NiFi-Logs/dashboard.json`.
- [x] External volume name resolves — Alloy started, which it could not do if
      `data-collector_data-collector-nifi-logs` were missing.
- [x] `stage.structured_metadata` is accepted by Alloy v1.5.1 (component healthy).
- [x] Alloy is up at `<host>:12345`, all 6 components healthy. NOTE: healthy only
      means the components loaded — a `loki.source.file` with zero matched targets
      is also green, so this does not prove logs are flowing.
- [ ] Prove data reached Loki:
      `docker compose ... exec loki wget -qO- http://localhost:3100/loki/api/v1/labels`
      should list `job`, `log_type`, `level`.
- [ ] Add the Loki datasource in Grafana (`http://loki:3100`) and import
      `grafana/NiFi-Logs/dashboard.json`.
- [ ] Verify a stack trace arrives as ONE log entry, not 30. If not, the multiline
      firstline regex needs adjusting to the real logback pattern.
- [ ] Verify timestamps are not all clustered at container-start time — that would
      mean the `location = "UTC"` assumption is wrong for this NiFi.
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
