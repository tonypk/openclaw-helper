package chat

import (
	"testing"

	"github.com/tonypk/openclaw-helper/internal/diagnosis"
)

func TestFAQMatch_APIKey(t *testing.T) {
	store := NewFAQStore()
	entry, score := store.Match("API Key 怎么获取")
	if score < 0.5 {
		t.Errorf("expected high score for API Key query, got %.2f", score)
	}
	if entry.ID != "what_is_apikey" {
		t.Errorf("expected what_is_apikey, got %s", entry.ID)
	}
}

func TestFAQMatch_WSL(t *testing.T) {
	store := NewFAQStore()
	entry, score := store.Match("什么是 WSL")
	if score < 0.5 {
		t.Errorf("expected match for WSL query, got %.2f", score)
	}
	if entry.ID != "what_is_wsl" {
		t.Errorf("expected what_is_wsl, got %s", entry.ID)
	}
}

func TestFAQMatch_NoMatch(t *testing.T) {
	store := NewFAQStore()
	_, score := store.Match("今天天气怎么样")
	if score >= 0.5 {
		t.Errorf("expected low score for unrelated query, got %.2f", score)
	}
}

func TestFAQSuggestions(t *testing.T) {
	store := NewFAQStore()

	suggestions := store.SuggestedQuestions("")
	if len(suggestions) < 3 {
		t.Errorf("expected at least 3 default suggestions, got %d", len(suggestions))
	}

	wslSuggestions := store.SuggestedQuestions("wsl")
	if len(wslSuggestions) <= len(suggestions) {
		t.Error("expected extra suggestions for WSL phase")
	}
}

func TestChatContext(t *testing.T) {
	ctx := NewChatContext("zh-CN")
	if !ctx.IsZH() {
		t.Error("expected Chinese")
	}

	ctx.AddMessage("user", "hello", "")
	ctx.AddMessage("assistant", "hi", "faq")
	if len(ctx.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(ctx.Messages))
	}

	// Test message limit
	for i := 0; i < 25; i++ {
		ctx.AddMessage("user", "msg", "")
	}
	if len(ctx.Messages) > 20 {
		t.Errorf("expected max 20 messages, got %d", len(ctx.Messages))
	}
}

func TestChatContext_ErrorLog(t *testing.T) {
	ctx := NewChatContext("en-US")
	longLog := make([]byte, 3000)
	for i := range longLog {
		longLog[i] = 'x'
	}
	ctx.SetErrorLog(string(longLog))
	if len(ctx.ErrorLog) > 2000 {
		t.Errorf("error log should be truncated to 2000, got %d", len(ctx.ErrorLog))
	}
}

func TestChatContext_SystemPrompt(t *testing.T) {
	ctx := NewChatContext("zh-CN")
	ctx.SetPhase("node")
	ctx.SetErrorLog("npm ERR! ENOTFOUND")

	prompt := ctx.BuildSystemPrompt()
	if prompt == "" {
		t.Error("expected non-empty system prompt")
	}
}

func TestHandler_FAQResponse(t *testing.T) {
	faq := NewFAQStore()
	llm := NewLLMProxy(nil) // no providers
	diag := diagnosis.NewEngine()

	handler := NewHandler(faq, llm, diag)
	resp := handler.Ask("API Key 怎么获取")

	if resp.Source != "faq" {
		t.Errorf("expected faq source, got %s", resp.Source)
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestHandler_OfflineFallback(t *testing.T) {
	faq := NewFAQStore()
	llm := NewLLMProxy(nil) // no providers
	diag := diagnosis.NewEngine()

	handler := NewHandler(faq, llm, diag)
	resp := handler.Ask("请帮我算一道数学题")

	if resp.Source != "offline" {
		t.Errorf("expected offline source for unknown query, got %s", resp.Source)
	}
}

func TestHandler_DiagnosisResponse(t *testing.T) {
	faq := NewFAQStore()
	llm := NewLLMProxy(nil)
	diag := diagnosis.NewEngine()

	handler := NewHandler(faq, llm, diag)
	handler.SetErrorLog("Error: Cannot find module 'sharp'")
	handler.SetPhase("openclaw")

	resp := handler.Ask("安装出了问题")

	if resp.Source != "diagnosis" {
		t.Errorf("expected diagnosis source, got %s", resp.Source)
	}
	if resp.RepairID == "" {
		t.Error("expected repair_id for sharp error")
	}
}

func TestHandler_LanguageSwitch(t *testing.T) {
	faq := NewFAQStore()
	llm := NewLLMProxy(nil)
	diag := diagnosis.NewEngine()

	handler := NewHandler(faq, llm, diag)
	handler.SetLanguage("en-US")

	// Use a query with enough keywords to hit threshold: api + key + 获取 = 3/6 = 0.5
	resp := handler.Ask("api key 获取")
	if resp.Source != "faq" {
		t.Errorf("expected faq source, got %s", resp.Source)
	}
	// In English mode, should return English answer (not Chinese)
	if resp.Message == "" {
		t.Error("expected non-empty answer")
	}
}

func TestLLMProxy_NoProviders(t *testing.T) {
	proxy := NewLLMProxy(nil)
	if proxy.HasProviders() {
		t.Error("expected no providers")
	}
}

func TestLLMProxy_EmptyKey(t *testing.T) {
	proxy := NewLLMProxy([]LLMProvider{
		{Name: "test", BaseURL: "http://localhost", APIKey: "", Model: "test"},
	})
	if proxy.HasProviders() {
		t.Error("expected no providers with empty key")
	}
}
