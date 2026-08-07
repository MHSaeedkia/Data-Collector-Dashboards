# alert-gateway — containerization decisions

Date: 2026-08-07

## Why these choices

- **Multi-stage build, `CGO_ENABLED=0`.** The service is pure Go with no cgo deps,
  so a static binary on a plain `alpine` runtime keeps the image small and avoids
  shipping the Go toolchain.
- **`ca-certificates` is installed in the runtime stage on purpose.** The senders
  talk to `https://api.telegram.org` and `https://tapi.bale.ai`. Without it every
  send fails with an x509 error. Do not drop this line to "slim down" the image.
- **`wget` for the healthcheck** — it comes from busybox in alpine, no extra package.
  Same style as the prometheus healthcheck already in the compose file.
- **`.env` is excluded via `.dockerignore` and injected with `env_file` at runtime.**
  Secrets must never be baked into an image layer. `config.yaml` only contains
  `${VAR}` placeholders (see `alert-gateway/README.md`), so it is safe to `COPY`.
- **`config.yaml` is both COPYed into the image and bind-mounted read-only in compose.**
  The COPY makes the image runnable standalone; the bind mount means editing a
  routing rule only needs a `restart`, not a rebuild.
- **`environment:` in compose sets `LISTEN_ADDR` / `CONFIG_PATH` explicitly.** They
  also exist in `.env` (with `CONFIG_PATH` as a *relative* path). Compose gives
  `environment:` precedence over `env_file:`, so the container always gets the
  absolute `/app/config/config.yaml` regardless of what `.env` says.

## Gotchas

- **Host port is `8085`, container port is `8080`.** 8080 on the host is very likely
  taken by NiFi (this stack monitors a NiFi deployment). Grafana does *not* use the
  published port — it reaches the service over the compose network as
  `http://alert-gateway:8080/alert`. The `ports:` mapping exists only for manual
  `curl` testing and can be removed.
- **`TELEGRAM_PROXY_URL` in `alert-gateway/.env` is still the placeholder**
  `http://<proxy-host>:<proxy-port>`. Any non-empty value makes the Telegram sender
  build a proxying transport, so it must be either a real reachable proxy or empty.
  A `127.0.0.1` proxy will *not* work from inside the container — use
  `host.docker.internal` (prometheus already declares that `extra_hosts` entry).
- The image build was never executed: the Docker daemon was down at the time these
  files were written. `docker compose config` passes and `go build` / `go test ./...`
  pass locally, but the first real `docker compose build` is unverified.

## Not done deliberately

- No `depends_on` was added between `grafana` and `alert-gateway`. Grafana retries
  alert delivery on its own, so ordering buys nothing and only slows startup.
