# TODO

## Done

- [x] Dockerfile for `alert-gateway` (multi-stage, static binary, non-root, healthcheck).
- [x] `alert-gateway` service added to `docker-compose.monitoring.yml`.

## Open

- [ ] Run `docker compose -f docker-compose.monitoring.yml build alert-gateway` once the
      Docker daemon is available — the image build has not been executed yet.
- [ ] Set a real `TELEGRAM_PROXY_URL` in `alert-gateway/.env` (or leave it empty);
      it is currently the `<proxy-host>` placeholder and will break Telegram sends.
- [ ] Point the Grafana Webhook contact point at `http://alert-gateway:8080/alert`.
