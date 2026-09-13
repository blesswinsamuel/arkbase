package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/blesswinsamuel/arkbase/internal/config"
	"github.com/rs/zerolog/log"
)

type EventPayload struct {
	Event        string        `json:"event"` // "success", "failure"
	DatabaseName string        `json:"database_name"`
	Engine       string        `json:"engine"`
	SizeBytes    int64         `json:"size_bytes"`
	Duration     time.Duration `json:"duration"`
	Destinations []string      `json:"destinations"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

func DispatchNotifications(ctx context.Context, notifications []config.NotificationConfig, payload EventPayload) {
	client := &http.Client{Timeout: 10 * time.Second}

	for _, n := range notifications {
		if len(n.OnEvents) > 0 && !slices.Contains(n.OnEvents, payload.Event) {
			continue
		}

		go func(cfg config.NotificationConfig) {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var err error
			switch cfg.Type {
			case "webhook":
				err = sendWebhook(notifyCtx, client, cfg, payload)
			case "discord":
				err = sendDiscord(notifyCtx, client, cfg, payload)
			case "telegram":
				err = sendTelegram(notifyCtx, client, cfg, payload)
			default:
				log.Warn().Str("type", cfg.Type).Msg("unknown notification type")
			}

			if err != nil {
				log.Error().Err(err).Str("name", cfg.Name).Str("type", cfg.Type).Msg("failed to send notification")
			} else {
				log.Info().Str("name", cfg.Name).Str("type", cfg.Type).Str("event", payload.Event).Msg("notification sent")
			}
		}(n)
	}
}

func sendWebhook(ctx context.Context, client *http.Client, cfg config.NotificationConfig, payload EventPayload) error {
	url := cfg.URL
	if url == "" {
		url = cfg.WebhookURL
	}
	if url == "" {
		return fmt.Errorf("missing webhook URL")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "arkbase/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook responded with status: %d", resp.StatusCode)
	}
	return nil
}

func sendDiscord(ctx context.Context, client *http.Client, cfg config.NotificationConfig, payload EventPayload) error {
	webhookURL := cfg.WebhookURL
	if webhookURL == "" {
		webhookURL = cfg.URL
	}
	if webhookURL == "" {
		return fmt.Errorf("missing discord webhook URL")
	}

	color := 0x22c55e // Green
	title := fmt.Sprintf("✅ Backup Succeeded: %s", payload.DatabaseName)
	if payload.Event == "failure" {
		color = 0xef4444 // Red
		title = fmt.Sprintf("❌ Backup Failed: %s", payload.DatabaseName)
	}

	description := fmt.Sprintf(
		"**Engine**: `%s`\n**Duration**: `%s`\n**Size**: `%.2f MB`\n**Destinations**: `%s`",
		payload.Engine,
		payload.Duration.Round(time.Millisecond),
		float64(payload.SizeBytes)/(1024*1024),
		strings.Join(payload.Destinations, ", "),
	)
	if payload.Error != "" {
		description += fmt.Sprintf("\n**Error**: ```%s```", payload.Error)
	}

	msg := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       title,
				"description": description,
				"color":       color,
				"timestamp":   payload.Timestamp.Format(time.RFC3339),
				"footer": map[string]string{
					"text": "arkbase",
				},
			},
		},
	}

	body, _ := json.Marshal(msg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord responded with status: %d", resp.StatusCode)
	}
	return nil
}

func sendTelegram(ctx context.Context, client *http.Client, cfg config.NotificationConfig, payload EventPayload) error {
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("telegram bot_token and chat_id are required")
	}

	icon := "✅"
	if payload.Event == "failure" {
		icon = "❌"
	}

	text := fmt.Sprintf(
		"%s *arkbase backup: %s*\n*Database*: `%s` (%s)\n*Duration*: `%s`\n*Size*: `%.2f MB`\n*Destinations*: `%s`",
		icon,
		strings.ToUpper(payload.Event),
		payload.DatabaseName,
		payload.Engine,
		payload.Duration.Round(time.Millisecond),
		float64(payload.SizeBytes)/(1024*1024),
		strings.Join(payload.Destinations, ", "),
	)
	if payload.Error != "" {
		text += fmt.Sprintf("\n*Error*: `%s`", payload.Error)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	msg := map[string]any{
		"chat_id":    cfg.ChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, _ := json.Marshal(msg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("telegram responded with status: %d", resp.StatusCode)
	}
	return nil
}
