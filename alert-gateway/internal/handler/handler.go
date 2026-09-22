package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"alert-gateway/internal/router"
	"alert-gateway/internal/senders"
)

// maxBodySize caps how much of the webhook body is read into memory.
const maxBodySize = 4 << 20 // 4 MiB

// sendTimeout bounds a single send to a single channel. It is deliberately
// per-send rather than per-webhook: Grafana batches many alerts into one call,
// and a shared budget lets one slow channel starve every later send.
// It is a backstop above the senders' own 10s HTTP client timeout.
const sendTimeout = 15 * time.Second

type AlertHandler struct {
	rt *router.Router
	// simple sender cache so they are not rebuilt for every alert.
	// net/http serves every request in its own goroutine, so the map must be
	// guarded — an unsynchronized concurrent write is a fatal runtime throw,
	// not a recoverable panic, and would take the whole process down.
	mu          sync.Mutex
	senderCache map[string]senders.Sender
}

func New(rt *router.Router) *AlertHandler {
	return &AlertHandler{
		rt:          rt,
		senderCache: map[string]senders.Sender{},
	}
}

func (h *AlertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log := slog.With("req_id", newRequestID())

	// Logged before anything can reject the request, so that "Grafana called us"
	// is always visible even when the call is later refused.
	log.Info("webhook received",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"content_length", r.ContentLength)

	if r.Method != http.MethodPost {
		log.Warn("webhook rejected: method not allowed", "method", r.Method)
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		log.Error("reading request body failed", "error", err, "read_bytes", len(body))
		http.Error(w, fmt.Sprintf("reading body: %v", err), http.StatusBadRequest)
		return
	}
	log.Debug("raw payload", "bytes", len(body), "body", snippet(body))

	var payload GrafanaWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		// The raw body is included here on purpose: a malformed or unexpected
		// payload shape is otherwise invisible.
		log.Error("invalid JSON payload", "error", err, "bytes", len(body), "body", snippet(body))
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	log.Info("payload decoded",
		"group_status", payload.Status,
		"alert_count", len(payload.Alerts),
		"common_labels", formatMap(payload.CommonLabels))

	if len(payload.Alerts) == 0 {
		log.Warn("payload contained no alerts, nothing to send")
	}

	// r.Context() carries no deadline of its own; it is cancelled if Grafana
	// hangs up. Each individual send gets its own deadline inside route().
	ctx := r.Context()

	var totalSent, totalFailed int
	for i, alert := range payload.Alerts {
		alertLog := log.With(
			"alert_idx", i,
			"alertname", alert.Labels["alertname"])
		sent, failed := h.route(ctx, alertLog, alert)
		totalSent += sent
		totalFailed += failed
	}

	// The single line to grep for when asking "did this webhook actually deliver?".
	log.Info("webhook completed",
		"alerts", len(payload.Alerts),
		"sent", totalSent,
		"failed", totalFailed,
		"duration_ms", elapsedMS(start))

	w.WriteHeader(http.StatusOK)
}

// route resolves the channels for one alert and sends to each, returning how
// many sends succeeded and how many failed.
func (h *AlertHandler) route(ctx context.Context, log *slog.Logger, alert GrafanaAlert) (sent, failed int) {
	channelNames := h.rt.Resolve(alert.Labels)
	message := formatMessage(alert)

	log.Info("alert routed",
		"status", alert.Status,
		"severity", alert.Labels["severity"],
		"channels", strings.Join(channelNames, ","),
		"labels", formatMap(alert.Labels))
	log.Debug("formatted message", "message", message)

	if len(channelNames) == 0 {
		log.Warn("alert resolved to no channels, dropping it")
		return 0, 0
	}

	for _, name := range channelNames {
		sender, err := h.getSender(name)
		if err != nil {
			log.Error("building sender failed", "channel", name, "error", err)
			failed++
			continue
		}

		// Only fires if Grafana hung up mid-batch; remaining sends are pointless.
		if err := ctx.Err(); err != nil {
			log.Error("send skipped, caller cancelled the request", "channel", name, "error", err)
			failed++
			continue
		}

		sendStart := time.Now()
		sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
		err = sender.Send(sendCtx, message)
		cancel()
		if err != nil {
			log.Error("send failed",
				"channel", name,
				"duration_ms", elapsedMS(sendStart),
				"error", err)
			failed++
			continue
		}
		log.Info("send succeeded", "channel", name, "duration_ms", elapsedMS(sendStart))
		sent++
	}
	return sent, failed
}

func (h *AlertHandler) getSender(name string) (senders.Sender, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if s, ok := h.senderCache[name]; ok {
		return s, nil
	}
	cfg, ok := h.rt.ChannelConfig(name)
	if !ok {
		return nil, fmt.Errorf("channel %q is not defined in the config", name)
	}
	s, err := senders.New(cfg)
	if err != nil {
		return nil, err
	}
	slog.Info("sender built", "channel", name, "type", cfg.Type)
	h.senderCache[name] = s
	return s, nil
}

func formatMessage(alert GrafanaAlert) string {
	icon := "🔥"
	if alert.Status == "resolved" {
		icon = "✅"
	}

	// Every interpolated value is escaped: the message is sent with
	// parse_mode=HTML, so a raw "<" or "&" coming from an alert summary
	// (e.g. "value < 5") makes the API reject the whole message with
	// 400 "can't parse entities". Only the <b> tags below are real markup.
	esc := html.EscapeString

	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>%s</b>\n", icon, esc(alert.Labels["alertname"]))
	fmt.Fprintf(&b, "Status: %s\n", esc(alert.Status))
	if sev, ok := alert.Labels["severity"]; ok {
		fmt.Fprintf(&b, "Severity: %s\n", esc(sev))
	}
	if summary, ok := alert.Annotations["summary"]; ok {
		fmt.Fprintf(&b, "Summary: %s\n", esc(summary))
	}
	if desc, ok := alert.Annotations["description"]; ok {
		fmt.Fprintf(&b, "Description: %s\n", esc(desc))
	}
	fmt.Fprintf(&b, "Started: %s", esc(alert.StartsAt))

	return b.String()
}

// newRequestID returns a short random id used to correlate every log line
// belonging to one webhook call.
func newRequestID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}

// formatMap renders a label/annotation map as a stable, sorted k=v list.
func formatMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

// snippet trims a raw body to something safe to put in a log line, cutting on
// rune boundaries so the output stays valid UTF-8.
func snippet(b []byte) string {
	const maxRunes = 1000
	s := strings.TrimSpace(string(b))
	r := []rune(s)
	if len(r) > maxRunes {
		return string(r[:maxRunes]) + "...(truncated)"
	}
	return s
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
