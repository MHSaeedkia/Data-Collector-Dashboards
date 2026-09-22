# How the monitoring stack joins the data-collector project

Date: 2026-08-10

## The arrangement

The monitoring project's **`default` network is declared external** and points at
`data-collector_data-collector-net`:

```yaml
networks:
  default:
    external: true
    name: "data-collector_data-collector-net"
```

So the monitoring stack has no network of its own. Every service in it lands on the
data-collector network automatically and resolves `kafka`, `nifi`, `postgres`,
`lpa-staleness-exporter`, `jobmanager`, ... by plain service name — and those
services can reach `prometheus`, `grafana`, `loki`, `alert-gateway` the same way.

This replaced an earlier arrangement where the shared network was declared as a
second network named `nifi_network` and only `prometheus` +
`nifi-token-refresher` opted into it. Overriding `default` is simpler: no
per-service `networks:` lists to maintain, and new services get it for free.

**Constraint: the user can only change the monitoring project.** Every
cross-project mechanism here therefore has to be pull-only — declaring the other
project's network and volumes as `external` — never asking data-collector to
declare anything.

## Ordering requirement

`data-collector` must be up **first**. Compose never creates an external network,
so starting monitoring alone fails with
`network data-collector_data-collector-net declared as external, but could not be found`.
Same applies to the `nifi-logs` volume (see `nifi-log-pipeline.md`).

## Why this is safe here

Docker DNS on a shared network is flat, so duplicate service names across the two
projects would collide. Checked at the time of the change — no overlap:

- monitoring: prometheus, nifi-token-refresher, grafana, alert-gateway, loki, alloy
- data-collector: jobmanager, taskmanager, kafka, schema-registry, kafka-ui, nifi,
  postgres, redis, redis-exporter, kafka-exporter, lpa-staleness-exporter, web

Published host ports also do not overlap. **Re-check both lists before adding a
service to either project.**

## Related rename

The staleness exporter was renamed `kafka-staleness-exporter` →
`lpa-staleness-exporter` in the data-collector project; `prometheus.yml` was
updated to match (job `lpa-staleness`, target `lpa-staleness-exporter:9309`).
The Grafana dashboard rename is tracked separately in
`staleness-episode-dashboard.md`.
