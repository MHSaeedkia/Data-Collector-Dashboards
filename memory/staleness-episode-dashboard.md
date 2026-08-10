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

- **State timeline, not a graph.** `kafka_topic_stale` in a `state-timeline` with
  `mergeValues: true` collapses consecutive equal samples into one bar, so each bar
  *is* an episode — start, end and width read directly as went-stale, recovered,
  duration. This answers "staleness history" better than any table.
- **`increase(...[$__range])` alongside the raw counter.** The raw
  `stale_episodes_total` is monotonic since exporter start, which is rarely the
  question being asked; the `$__range` version follows the dashboard time picker.
- The pre-existing "Kafka Topic Data Freshness" table was left completely untouched,
  including its h=31 gridPos. That height pushes the new panels well down the page —
  shrinking it would be an improvement but was out of scope.

See also `grafana-alerting-wiring.md` — the alert rule name `KafkaTopicStale` is
unrelated to the dashboard title and is NOT affected by this rename.
