# LPA Staleness Monitoring — episode history panels

Date: 2026-08-09

Dashboard renamed from `KAFKA-Topic-Staleness-Monitor` / "Kafka Topic Staleness
Monitor" to `LPA-Staleness-Monitoring` / "LPA Staleness Monitoring".
**The uid `dfsv0halr3im8d` was deliberately left unchanged** so existing links,
bookmarks and any alert rule referencing the dashboard keep working. Renaming a
dashboard is safe; changing its uid is not.

Folder rename was done with `git mv` to keep the file's history.

## The two traps in these episode metrics

1. **`kafka_topic_*_timestamp_seconds` are in SECONDS, Grafana's `dateTimeAsIso`
   unit expects MILLISECONDS.** The queries therefore end in `* 1000`. Without it
   every topic renders as a 1970 date. Any future panel using these metrics needs
   the same multiplication.
2. **`0` is a sentinel, not a time.** `stale_since` is 0 when the topic is healthy
   and `last_recovery` is 0 when it has never recovered — both would render as
   1970-01-01. Explicit value mappings turn them into "not stale" / "never".
   `last_stale_duration` of 0 is mapped to "none yet", which is *technically*
   ambiguous with a genuine sub-second outage — acceptable, but worth knowing.

Note `* 1000` drops `__name__` from the result labels, so those frames have one
fewer column than the raw-metric queries. The organize transform's `excludeByName`
lists `__name__ 1..6` regardless; entries that do not exist are ignored by Grafana.

## Panel design choices

- **A state-timeline was tried and rejected by the user (2026-08-10).** The reasoning
  was that `mergeValues: true` turns each run of stale=1 into one bar, so bars read
  as episodes. In practice the user found it useless and asked for a table instead.
  Do not re-propose it. It was replaced by "Staleness history (per topic)" — a plain
  table of Topic / Stale since / Last recovery / Duration.
- **`increase(...[$__range])` alongside the raw counter.** The raw
  `stale_episodes_total` is monotonic since exporter start, which is rarely the
  question being asked; the `$__range` version follows the dashboard time picker.
- The pre-existing "Kafka Topic Data Freshness" table was left completely untouched,
  including its h=31 gridPos. That height pushes the new panels well down the page —
  shrinking it would be an improvement but was out of scope.

## Datasource references in the JSON (2026-08-10)

The Postgres datasource is an **import-time input**: `__inputs` declares
`DS_POSTGRES` (pluginId `grafana-postgresql-datasource`) and the three places that
need it — panel 6, its target, and the `topic` template variable — reference
`${DS_POSTGRES}`. Grafana only shows the import options form when `__inputs` is
present, so that block is what makes the picker appear; do not drop it.

**The Prometheus uid is still hardcoded** (`dftgy57zq31fkd`, 11 references) because
the file was re-exported from the Grafana UI with plain "Export", which bakes in
real uids. Panels keep working in this Grafana instance but would break in any
other, or if that datasource is ever recreated. Fixing it means adding a
`DS_PROMETHEUS` input and swapping those 11 refs — offered and not yet taken up.

Note: the episode-log panel uses `repeat: "topic"`, so `WHERE topic = '$topic'`
in its SQL is correct — each repeated instance receives a single value. It would
be a quoting bug only if the panel did not repeat.

## Panel history (do not re-add these)

The dashboard converged on ONE history panel: "Episode history per topic".
Two earlier attempts were explicitly rejected by the user:

- a `state-timeline` of `kafka_topic_stale` (2026-08-10) — "useless";
- a "Staleness history (per topic)" table of Stale since / Last recovery / Duration
  (2026-08-10) — deleted as redundant, since those columns already exist in
  "Episode history per topic".

## What these metrics cannot do

Only the **last** completed episode is retained (`last_stale_duration`,
`last_recovery` are gauges, overwritten each time). A true per-episode history —
one row per outage, several rows per topic — is **not derivable** from this
exporter, and no dashboard trick fixes it: Prometheus can count events with
`changes()` but cannot emit one row per event.

Two ways out, both discussed with the user 2026-08-10, neither implemented yet:

- **Loki (recommended).** The exporter *already* logs `topic recovered: %s (stale
  for %.1fs)` — every such line is a completed episode. Make it JSON/logfmt, ship
  the exporter's stdout to Loki (needs `discovery.docker` + `loki.source.docker`,
  not the volume mount used for NiFi, because these logs go to stdout), then a
  table panel over LogQL gives one row per episode. Reuses the pipeline that
  already exists. Retention = Loki's 15d.
- **Postgres.** INSERT `(topic, started_at, ended_at, duration_seconds)` on each
  recovery. Wins for retention beyond 15d, aggregation, and joins against
  `exchange_markets`.

Both only record going forward — episodes already past are lost — and both still
miss episodes shorter than one `poll_interval_seconds`.

Episode state is in-memory in the exporter, so "Episodes (total)" resets to 0 on
every restart and an in-flight episode is forgotten. It reads like an all-time
figure but is really "since this process started".

See also `grafana-alerting-wiring.md` — the alert rule name `KafkaTopicStale` is
unrelated to the dashboard title and is NOT affected by this rename.
