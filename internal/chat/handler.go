package chat

import (
	"context"
	"time"

	"github.com/tonypk/openclaw-helper/internal/diagnosis"
)

const faqConfidenceThreshold = 0.5

// ChatResponse is the response to a chat request.
type ChatResponse struct {
	Message    string               `json:"message"`
	Source     string               `json:"source"` // "faq", "diagnosis", "llm", "offline"
	RepairID   string              `json:"repair_id,omitempty"`
	AutoRepair bool                `json:"auto_repair,omitempty"`
	Suggested  []SuggestedQuestion `json:"suggested,omitempty"`
}

// Handler processes chat messages with a tiered approach:
// 1. Local FAQ matching (instant, offline)
// 2. Diagnosis engine (if error context present)
// 3. LLM API call (requires network + API key)
type Handler struct {
	faq       *FAQStore
	llm       *LLMProxy
	diagEng   *diagnosis.Engine
	ctx       *ChatContext
}

// NewHandler creates a chat handler.
func NewHandler(faq *FAQStore, llm *LLMProxy, diagEng *diagnosis.Engine) *Handler {
	return &Handler{
		faq:     faq,
		llm:     llm,
		diagEng: diagEng,
		ctx:     NewChatContext("zh-CN"),
	}
}

// SetLanguage changes the response language.
func (h *Handler) SetLanguage(lang string) {
	h.ctx.Language = lang
}

// SetPhase updates the installation phase context.
func (h *Handler) SetPhase(phase string) {
	h.ctx.SetPhase(phase)
}

// SetErrorLog updates the error context.
func (h *Handler) SetErrorLog(log string) {
	h.ctx.SetErrorLog(log)
}

// GetContext returns the current chat context.
func (h *Handler) GetContext() *ChatContext {
	return h.ctx
}

// Ask processes a user message and returns a response.
func (h *Handler) Ask(userMessage string) ChatResponse {
	h.ctx.AddMessage("user", userMessage, "")

	// 1. Try FAQ match
	entry, score := h.faq.Match(userMessage)
	if score >= faqConfidenceThreshold {
		answer := entry.Answer
		if h.ctx.IsZH() && entry.AnsZH != "" {
			answer = entry.AnsZH
		}
		h.ctx.AddMessage("assistant", answer, "faq")
		return ChatResponse{
			Message:   answer,
			Source:    "faq",
			Suggested: h.faq.SuggestedQuestions(h.ctx.CurrentPhase),
		}
	}

	// 2. Try diagnosis if error context exists
	if h.ctx.ErrorLog != "" && h.diagEng != nil {
		diagCtx := &diagnosis.DiagContext{
			ErrorLog:     h.ctx.ErrorLog,
			CurrentPhase: h.ctx.CurrentPhase,
		}
		report := h.diagEng.Diagnose(diagCtx)
		if len(report.Issues) > 0 {
			issue := report.Issues[0]
			answer := issue.Description
			if h.ctx.IsZH() && issue.DescZH != "" {
				answer = issue.DescZH
			}
			h.ctx.AddMessage("assistant", answer, "diagnosis")
			return ChatResponse{
				Message:    answer,
				Source:     "diagnosis",
				RepairID:   issue.RepairID,
				AutoRepair: issue.AutoRepair,
				Suggested:  h.faq.SuggestedQuestions(h.ctx.CurrentPhase),
			}
		}
	}

	// 3. Try LLM
	if h.llm != nil && h.llm.HasProviders() {
		reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		systemPrompt := h.ctx.BuildSystemPrompt()
		llmResp, err := h.llm.Chat(reqCtx, systemPrompt, h.ctx.Messages)
		if err == nil {
			h.ctx.AddMessage("assistant", llmResp, "llm")
			return ChatResponse{
				Message:   llmResp,
				Source:    "llm",
				Suggested: h.faq.SuggestedQuestions(h.ctx.CurrentPhase),
			}
		}
		// LLM failed, fall through to offline
	}

	// 4. Offline fallback
	fallback := "I'm sorry, I can't connect to AI services right now. Please check the FAQ or try again later."
	if h.ctx.IsZH() {
		fallback = "抱歉，目前无法连接 AI 服务。请查看常见问题或稍后再试。如果有具体错误信息，请粘贴给我看看。"
	}
	h.ctx.AddMessage("assistant", fallback, "offline")
	return ChatResponse{
		Message:   fallback,
		Source:    "offline",
		Suggested: h.faq.SuggestedQuestions(h.ctx.CurrentPhase),
	}
}

// GetSuggestions returns suggested questions for the current phase.
func (h *Handler) GetSuggestions() []SuggestedQuestion {
	return h.faq.SuggestedQuestions(h.ctx.CurrentPhase)
}
