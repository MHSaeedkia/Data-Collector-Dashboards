# NiFi logs → Loki → Grafana

Date: 2026-08-09

## Why Loki and not Elasticsearch

Prometheus stores samples, not text, so logs needed a second store. Loki won on
fit rather than features: native Grafana datasource (no plugin), LogQL mirrors
PromQL so the queries match the existing dashboards, and a single-binary Loki with
filesystem storage costs ~200-300 MB RAM against 2 GB+ for an Elasticsearch JVM.
Full-text search was not worth that trade here.

Collector is **Grafana Alloy**, not Promtail — Promtail is deprecated in favour of
Alloy.

## The key structural fact

**NiFi's logs are reachable without touching the data-collector project.** That
project already mounts a named volume `data-collector-nifi-logs` at
`/opt/nifi/nifi-current/logs`. Named volumes are shareable across compose projects,
so the monitoring stack declares it `external` under its fully-qualified name
`data-collector_data-collector-nifi-logs` and mounts it read-only into Alloy.

This is the reason full coverage (app + user + bootstrap + request logs) was chosen
over the easier "scrape the container's stdout via the Docker socket" approach —
stdout only carries `nifi-app.log`, because the image's start script ends with a
`tail -F` of that one file.

## Decisions inside the pipeline that are not obvious

- **Log files are listed explicitly, never globbed.** Logback rotates to
  `nifi-app_2026-08-09_10.0.log` *in the same directory*, so a `nifi-*.log` pattern
  re-ingests every rotated file as new. This would silently duplicate history.
- **`stage.multiline` is load-bearing, not a nicety.** A Java stack trace is 30+
  lines; without it each line becomes its own Loki entry and traces are unreadable.
  The firstline regex is the logback timestamp — anything not starting with one is
  a continuation line.
- **Only `level` and `log_type` are labels.** Loki indexes labels, so `thread` and
  `logger` (unbounded cardinality) go to structured metadata instead. Never promote
  processor id or FlowFile UUID to a label.
- **`stage.timestamp` with `location = "UTC"`.** Without it entries carry ingestion
  time, so after any Alloy restart the backfilled lines pile up at "now" and stop
  aligning with the Prometheus panels. The NiFi container sets no TZ, so it is UTC.
- **Loki retention is 360h to match Prometheus' 15d**, so a dashboard never shows a
  metrics window with no logs behind it. Retention only takes effect because
  `compactor.retention_enabled: true` is set — the `limits_config` value alone does
  nothing.
- **`nifi-request.log` is ingested unparsed.** It is Jetty format, not logback, so it
  gets no `level` label and is stamped at ingestion time. Hence the dashboard's
  `level` variable uses `allValue: ".*"` (not `.+`), which in LogQL also matches
  streams that lack the label — `.+` would silently hide the request log.

## Same trick applies to Flink

`data-collector-jobmanager-logs` and `data-collector-taskmanager-logs` are named
volumes on `/opt/flink/log` in the same project. Adding Flink logs is a copy of the
NiFi block with a different path and job label — deliberately not done, since only
NiFi was asked for.

See also `grafana-alerting-wiring.md` (the error-rate LogQL query is a viable
Grafana alert rule, which would route through the alert-gateway).
