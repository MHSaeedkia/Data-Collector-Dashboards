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

type telegramSender struct {
	botToken string
	chatID   string
	client   *http.Client
}

func newTelegramSender(cfg router.ChannelConfig) (Sender, error) {
	botToken := cfg.Extra["bot_token"]
	chatID := cfg.Extra["chat_id"]
	if botToken == "" || chatID == "" {
		return nil, fmt.Errorf("telegram: bot_token or chat_id is empty (check .env)")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	if proxyURL := cfg.Extra["proxy_url"]; proxyURL != "" {
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("telegram: invalid proxy_url: %w", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(parsed),
		}
	}

	return &telegramSender{botToken: botToken, chatID: chatID, client: client}, nil
}

func (t *telegramSender) Send(ctx context.Context, message string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	form := url.Values{}
	form.Set("chat_id", t.chatID)
	form.Set("text", message)
	form.Set("parse_mode", "HTML")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("telegram: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: sending request (check the proxy): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram: unexpected response %d", resp.StatusCode)
	}
	return nil
}
