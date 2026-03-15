package report

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Telegram bot configuration — injected via ldflags at build time.
var (
	telegramBotToken = ""
	telegramChatID   = ""
)

// sendMessageRequest is the Telegram Bot API sendMessage payload.
type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// TelegramConfigured returns true if bot token and chat ID are set.
func TelegramConfigured() bool {
	return telegramBotToken != "" && telegramChatID != ""
}

// SendToTelegram sends a crash report to the configured Telegram channel.
// Returns nil silently if bot token or chat ID are not configured.
func SendToTelegram(ctx context.Context, r CrashReport) error {
	if telegramBotToken == "" || telegramChatID == "" {
		return nil
	}

	return sendTelegramMessage(ctx, telegramBotToken, telegramChatID, r)
}

// sendTelegramMessage sends the report using the Telegram Bot API.
func sendTelegramMessage(ctx context.Context, botToken, chatID string, r CrashReport) error {
	text := FormatTelegramBody(r)

	// Telegram message limit is 4096 characters
	if len(text) > 4000 {
		text = text[:4000] + "\n\n\\.\\.\\. _truncated_"
	}

	payload := sendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "MarkdownV2",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var result struct {
			OK          bool   `json:"ok"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			return fmt.Errorf("telegram API error: %s", result.Description)
		}
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
