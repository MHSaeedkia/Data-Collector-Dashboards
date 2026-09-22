package senders

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"alert-gateway/internal/router"
)

// Sender is the shared interface of every delivery channel (Telegram, Bale, anything else later).
type Sender interface {
	Send(ctx context.Context, message string) error
}

// New builds the right sender based on the Type in ChannelConfig.
// Adding a new channel just means adding one case here
// and implementing a new file like telegram.go
func New(cfg router.ChannelConfig) (Sender, error) {
	switch cfg.Type {
	case "telegram":
		return newTelegramSender(cfg)
	case "bale":
		return newBaleSender(cfg)
	default:
		return nil, fmt.Errorf("unknown channel type: %q", cfg.Type)
	}
}

// readErrorBody reads a bounded amount of an error response body so the caller
// can put the API's own explanation into the returned error.
func readErrorBody(resp *http.Response) string {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if err != nil {
		return fmt.Sprintf("<body unreadable: %v>", err)
	}
	body := strings.TrimSpace(string(raw))
	if body == "" {
		return "<empty body>"
	}
	return body
}
