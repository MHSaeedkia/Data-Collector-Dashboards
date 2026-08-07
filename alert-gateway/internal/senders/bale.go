package senders

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"alert-gateway/internal/router"
)

type baleSender struct {
	botToken string
	chatID   string
	apiBase  string
	client   *http.Client
}

func newBaleSender(cfg router.ChannelConfig) (Sender, error) {
	botToken := cfg.Extra["bot_token"]
	chatID := cfg.Extra["chat_id"]
	apiBase := cfg.Extra["api_base"]
	if botToken == "" || chatID == "" {
		return nil, fmt.Errorf("bale: bot_token or chat_id is empty (check .env)")
	}
	if apiBase == "" {
		apiBase = "https://tapi.bale.ai"
	}

	return &baleSender{
		botToken: botToken,
		chatID:   chatID,
		apiBase:  apiBase,
		client:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (b *baleSender) Send(ctx context.Context, message string) error {
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", b.apiBase, b.botToken)

	form := url.Values{}
	form.Set("chat_id", b.chatID)
	form.Set("text", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("bale: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("bale: sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("bale: unexpected response %d", resp.StatusCode)
	}
	return nil
}
