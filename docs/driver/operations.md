# Driver Framework Operations Runbook

This runbook covers running the extensible driver framework (RFC 0001) in
production: keeping the `events` log bounded, protecting the server from a
misbehaving driver, and the knobs you tune per deployment.

Everything here applies only when the framework is enabled
(`--enable-driver-framework`, `QPI_ENABLE_DRIVER_FRAMEWORK`, or
`enableDriverFramework: true`). With the flag off, none of the collections,
loops, or limits below exist and the server behaves exactly as before.

## Tuning knobs

Every setting follows the same precedence as the rest of QPI: **CLI flag > env
var > config file > default**.

| Setting | Flag | Env | Config key | Default | Purpose |
| --- | --- | --- | --- | --- | --- |
| Events retention window | `--events-retention` | `QPI_EVENTS_RETENTION` | `eventsRetention` | `720h` (30 days) | How long an entry stays in the `events` log before it is pruned. |
| Prune interval | `--events-prune-interval` | `QPI_EVENTS_PRUNE_INTERVAL` | `eventsPruneInterval` | `1h` | How often the retention loop runs. |
| Per-driver rate limit | `--event-rate-limit` | `QPI_EVENT_RATE_LIMIT` | `eventRateLimit` | `100` | Max inbound events per second accepted from each driver. |

Durations use Go's duration syntax (`720h`, `30m`, `90s`). Set
`eventsRetention` or `eventRateLimit` to `0` to disable pruning or the rate
limit respectively.

Example `qpi.config.yml`:

```yaml
enableDriverFramework: true
eventsRetention: "168h"      # keep one week of events
eventsPruneInterval: "30m"
eventRateLimit: 50           # 50 events/sec per driver
```

## Retention and pruning

The `events` collection is the single trace log of every driver→UI event a
handler chooses to persist (for example a cryostat monitor's readings). Left
unbounded it grows with every reading, so a background loop
(`RunEventsRetentionEngine`) deletes entries older than `eventsRetention` on
each `eventsPruneInterval` tick.

- Pruning compares each row's `ts` (the event's own UTC timestamp) against
  `now - eventsRetention` and deletes in bounded batches, so a large backlog
  never blocks on a single transaction.
- The loop exits immediately if the framework is off or `eventsRetention` is
  `0`, so a legacy deployment starts no extra goroutine.
- Deletions are logged as `[Retention] pruned N expired events`.

To confirm pruning is keeping growth flat, watch the row count over a window
longer than the retention period under steady load; it should plateau rather
than climb.

```sql
SELECT count(*) FROM events;
```

### Index

A composite index `idx_events_type_ts` on `events(type, ts)` backs both the
dashboard's per-type, time-ordered charts and the retention scan. It is created
idempotently by the schema migration; no manual step is required.

## Per-driver rate limiting

Each connected driver's inbound event stream is guarded by its own token-bucket
limiter (`eventRateLimit` events/sec, bursting up to one second's worth). When a
driver exceeds its rate, the surplus events are logged and dropped —
`[DriverListener <id>] rate limit exceeded, dropping event` — and the listener
keeps running. One driver flooding events cannot starve the others: limiters are
per-driver, not global.

Tune `eventRateLimit` to comfortably exceed a healthy driver's emit rate. A
monitor emitting every few seconds needs only a handful per second; the default
of `100` leaves wide headroom. Set it to `0` only if you trust every driver and
want no cap.

## Troubleshooting

**The `events` table keeps growing.** Confirm `eventsRetention > 0` and that the
retention engine logged `[Retention] Engine started`. If retention is long
relative to event volume, growth up to the steady-state size is expected — the
count should plateau once the oldest events start aging out.

**A driver's readings are missing from the dashboard.** Check the server log for
`rate limit exceeded` lines; the driver may be emitting faster than
`eventRateLimit`. Raise the limit or slow the driver's `every()` interval. Also
confirm the event type has a registered handler — unknown types are logged and
dropped by design.

**Pruning deleted too much / too little.** `eventsRetention` is the only lever;
it takes effect on the next `eventsPruneInterval` tick without a restart only if
supplied via config reload — otherwise restart the server after changing it.

## Verify

```
make test-go                               # config, index, prune, rate-limit tests
make test-e2e-driver-framework EXECUTOR=mock
```
