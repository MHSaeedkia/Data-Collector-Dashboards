package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"alert-gateway/internal/router"
	"alert-gateway/internal/senders"
)

type AlertHandler struct {
	rt *router.Router
	// simple sender cache so they are not rebuilt for every alert
	senderCache map[string]senders.Sender
}

func New(rt *router.Router) *AlertHandler {
	return &AlertHandler{
		rt:          rt,
		senderCache: map[string]senders.Sender{},
	}
}

func (h *AlertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload GrafanaWebhook
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	for _, alert := range payload.Alerts {
		h.route(ctx, alert)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AlertHandler) route(ctx context.Context, alert GrafanaAlert) {
	channelNames := h.rt.Resolve(alert.Labels)
	message := formatMessage(alert)

	for _, name := range channelNames {
		sender, err := h.getSender(name)
		if err != nil {
			log.Printf("error building sender %q: %v", name, err)
			continue
		}
		if err := sender.Send(ctx, message); err != nil {
			log.Printf("error sending to channel %q: %v", name, err)
			continue
		}
		log.Printf("alert %q sent successfully to %q", alert.Labels["alertname"], name)
	}
}

func (h *AlertHandler) getSender(name string) (senders.Sender, error) {
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
	h.senderCache[name] = s
	return s, nil
}

func formatMessage(alert GrafanaAlert) string {
	icon := "🔥"
	if alert.Status == "resolved" {
		icon = "✅"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>%s</b>\n", icon, alert.Labels["alertname"])
	fmt.Fprintf(&b, "Status: %s\n", alert.Status)
	if sev, ok := alert.Labels["severity"]; ok {
		fmt.Fprintf(&b, "Severity: %s\n", sev)
	}
	if summary, ok := alert.Annotations["summary"]; ok {
		fmt.Fprintf(&b, "Summary: %s\n", summary)
	}
	if desc, ok := alert.Annotations["description"]; ok {
		fmt.Fprintf(&b, "Description: %s\n", desc)
	}
	fmt.Fprintf(&b, "Started: %s", alert.StartsAt)

	return b.String()
}
