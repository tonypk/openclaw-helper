package report

import (
	"context"
	"testing"
)

func TestSendToTelegram_NoConfig(t *testing.T) {
	// When bot token and chat ID are empty, should return nil silently
	oldToken := telegramBotToken
	oldChat := telegramChatID
	telegramBotToken = ""
	telegramChatID = ""
	defer func() {
		telegramBotToken = oldToken
		telegramChatID = oldChat
	}()

	r := sampleReport()
	err := SendToTelegram(context.Background(), r)
	if err != nil {
		t.Errorf("expected nil error when not configured, got: %v", err)
	}
}
