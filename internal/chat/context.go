package chat

import (
	"time"
)

// ChatMessage represents a single message in the chat.
type ChatMessage struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source,omitempty"` // "faq", "llm", "diagnosis"
}

// ChatContext tracks the conversation and current system state.
type ChatContext struct {
	Messages     []ChatMessage `json:"messages"`
	CurrentPhase string        `json:"current_phase"`
	ErrorLog     string        `json:"error_log,omitempty"`
	Language     string        `json:"language"` // "zh-CN" or "en-US"
}

// NewChatContext creates a new chat context.
func NewChatContext(lang string) *ChatContext {
	if lang == "" {
		lang = "zh-CN"
	}
	return &ChatContext{
		Language: lang,
	}
}

// AddMessage appends a message to the conversation.
func (c *ChatContext) AddMessage(role, content, source string) {
	c.Messages = append(c.Messages, ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Source:    source,
	})

	// Keep last 20 messages to limit memory
	if len(c.Messages) > 20 {
		c.Messages = c.Messages[len(c.Messages)-20:]
	}
}

// SetPhase updates the current installation phase context.
func (c *ChatContext) SetPhase(phase string) {
	c.CurrentPhase = phase
}

// SetErrorLog updates the error log context.
func (c *ChatContext) SetErrorLog(log string) {
	// Keep last 2000 chars of error log
	if len(log) > 2000 {
		log = log[len(log)-2000:]
	}
	c.ErrorLog = log
}

// IsZH returns true if the language is Chinese.
func (c *ChatContext) IsZH() bool {
	return c.Language == "zh-CN" || c.Language == "zh"
}

// BuildSystemPrompt creates a system prompt for the LLM with current context.
func (c *ChatContext) BuildSystemPrompt() string {
	prompt := `You are the OpenClaw Helper assistant. Help users install and configure OpenClaw, an open-source AI agent platform.

Current context:
- Installation phase: ` + c.CurrentPhase + `
- Language preference: ` + c.Language

	if c.ErrorLog != "" {
		prompt += "\n- Recent error log:\n```\n" + c.ErrorLog + "\n```"
	}

	prompt += `

Guidelines:
- Be friendly and encouraging, many users are complete beginners
- Give step-by-step instructions
- If you can identify the problem, suggest a specific fix
- If the user is in China, recommend DeepSeek and npmmirror
- Always respond in the user's language (` + c.Language + `)
- Keep responses concise but helpful`

	return prompt
}
