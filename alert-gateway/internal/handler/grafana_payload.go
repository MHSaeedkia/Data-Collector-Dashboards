package handler

// GrafanaWebhook is the payload structure that Grafana Alerting sends to a
// Webhook Contact Point (recent Unified Alerting versions).
type GrafanaWebhook struct {
	Status       string            `json:"status"` // firing | resolved
	CommonLabels map[string]string `json:"commonLabels"`
	Alerts       []GrafanaAlert    `json:"alerts"`
}

type GrafanaAlert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	EndsAt      string            `json:"endsAt"`
}
