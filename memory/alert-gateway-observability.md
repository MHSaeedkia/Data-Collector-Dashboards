# alert-gateway — logging & the intermittent-delivery investigation

Date: 2026-08-09

## Why the logging looks like this

Added while chasing "sometimes it doesn't send to Bale/Telegram". Design choices:

- **`log/slog`, not a logging library.** The module is vendored (`-mod=vendor` in the
  Dockerfile), so any external dep means re-vendoring. slog is stdlib — zero cost.
- **`LOG_LEVEL` was already in `.env` and `.env.example` but read by nothing.** The
  logging work wired it up rather than inventing a new knob.
- **All logging lives in the handler; senders stay silent and instead return errors
  enriched with the API response body.** One place to look, and the senders keep
  their single responsibility.
- **`req_id` per webhook call** (random 6-byte hex) tags every line for that call.
  Grafana batches multiple alerts into one POST, so without it interleaved sends
  are impossible to attribute.
- **The catch-all `/` route logs unknown paths at WARN.** A wrong webhook URL in
  Grafana otherwise produced a silent 404 with nothing in the logs.

## The three suspects for intermittent failure — all three fixed 2026-08-09

Fixed together, before a failing production log was captured. So it is **not
established which one actually caused the reported symptom** — possibly more than
one, possibly none. If intermittent failures persist, that is real new information:
do not assume these are still the cause.

1. **No HTML escaping** (fixed). `formatMessage` wrapped the alertname in `<b>` and
   sent `parse_mode=HTML` without escaping interpolated label/annotation values. A
   summary containing `<` or `&` ("value < 5") makes the API reject the whole message
   with 400 `can't parse entities`. Best fit for the symptom: failure depends on the
   alert *text*, so it looks random. Now every interpolated value goes through
   `html.EscapeString`; only the `<b>` tags are real markup.
   - **Consequence:** Bale was previously sent *without* `parse_mode`, so escaping
     alone would have made it display `&amp;` literally. Bale now also gets
     `parse_mode=HTML`. This additionally fixes a pre-existing cosmetic bug where
     Bale showed the literal `<b>` tags. **Unverified against the real Bale API** —
     only against a local fake. If Bale rejects `parse_mode`, it will now show up in
     the logs as a non-2xx with the response body.
2. **One 15s budget shared by the entire batch** (fixed). The context was created
   once per webhook and covered every alert × every channel, sent sequentially. Now
   each send gets its own 15s deadline derived from `r.Context()`, and there is no
   batch-level deadline — a large group takes as long as it takes, and Grafana's own
   webhook timeout is the backstop. `r.Context()` still cancels everything if Grafana
   hangs up, which is what the `send skipped, caller cancelled` line reports.
3. **`senderCache` was an unsynchronized map** (fixed). `net/http` serves each request
   in its own goroutine and `getSender` wrote the map with no lock. Concurrent map
   writes are a *fatal* runtime throw, not a recoverable panic — the process dies,
   `restart: on-failure` revives it, and in-flight alerts vanish. Now guarded by a
   `sync.Mutex`. Verified with `go build -race` + 40 concurrent cold-cache webhooks:
   zero race reports, 80/80 sends delivered.

## Reading the logs

`webhook completed ... alerts=N sent=X failed=Y` is the summary line. Absence of any
`webhook received` line means Grafana never reached the service — a Grafana-side or
networking problem, not a gateway problem.

See also `alert-gateway-containerization.md`, `grafana-alerting-wiring.md`.
