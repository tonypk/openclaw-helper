// Package chat provides AI chat assistance with FAQ matching and LLM proxy.
package chat

import (
	"strings"
)

// FAQEntry represents a frequently asked question with answer.
type FAQEntry struct {
	ID       string   `json:"id"`
	Keywords []string `json:"keywords"`
	Question string   `json:"question"`
	QnZH     string   `json:"question_zh"`
	Answer   string   `json:"answer"`
	AnsZH    string   `json:"answer_zh"`
	Category string   `json:"category"`
}

// FAQStore holds FAQ entries and provides keyword-based matching.
type FAQStore struct {
	entries []FAQEntry
}

// NewFAQStore creates a store with built-in FAQ entries.
func NewFAQStore() *FAQStore {
	return &FAQStore{entries: builtinFAQ()}
}

// NewFAQStoreWithEntries creates a store with custom entries (for testing).
func NewFAQStoreWithEntries(entries []FAQEntry) *FAQStore {
	return &FAQStore{entries: entries}
}

// Match finds the best matching FAQ entry for a user query.
// Returns the entry and a confidence score (0.0-1.0).
func (s *FAQStore) Match(query string) (FAQEntry, float64) {
	query = strings.ToLower(query)
	var best FAQEntry
	bestScore := 0.0

	for _, entry := range s.entries {
		score := s.score(query, entry)
		if score > bestScore {
			bestScore = score
			best = entry
		}
	}

	return best, bestScore
}

// SuggestedQuestions returns preset question chips for the current context.
func (s *FAQStore) SuggestedQuestions(phase string) []SuggestedQuestion {
	suggestions := []SuggestedQuestion{
		{Text: "API Key 怎么获取？", TextEN: "How to get an API Key?"},
		{Text: "安装好慢怎么办？", TextEN: "Installation is slow, what to do?"},
		{Text: "出错了怎么办？", TextEN: "What if there's an error?"},
	}

	switch phase {
	case "wsl", "precheck":
		suggestions = append(suggestions,
			SuggestedQuestion{Text: "什么是 WSL？", TextEN: "What is WSL?"},
			SuggestedQuestion{Text: "虚拟化怎么开？", TextEN: "How to enable virtualization?"},
		)
	case "node":
		suggestions = append(suggestions,
			SuggestedQuestion{Text: "Node.js 是什么？", TextEN: "What is Node.js?"},
		)
	case "openclaw", "config":
		suggestions = append(suggestions,
			SuggestedQuestion{Text: "OpenClaw 怎么用？", TextEN: "How to use OpenClaw?"},
			SuggestedQuestion{Text: "推荐哪个 AI 服务？", TextEN: "Which AI service do you recommend?"},
		)
	}

	return suggestions
}

// SuggestedQuestion is a preset question chip.
type SuggestedQuestion struct {
	Text   string `json:"text"`
	TextEN string `json:"text_en"`
}

func (s *FAQStore) score(query string, entry FAQEntry) float64 {
	matches := 0
	total := len(entry.Keywords)
	if total == 0 {
		return 0
	}

	for _, kw := range entry.Keywords {
		if strings.Contains(query, strings.ToLower(kw)) {
			matches++
		}
	}

	// Also check Chinese question
	if entry.QnZH != "" && strings.Contains(query, strings.ToLower(entry.QnZH)) {
		return 1.0
	}

	return float64(matches) / float64(total)
}

func builtinFAQ() []FAQEntry {
	return []FAQEntry{
		{
			ID: "what_is_apikey", Category: "config",
			Keywords: []string{"api", "key", "密钥", "钥匙", "获取", "怎么拿"},
			Question: "How do I get an API Key?",
			QnZH:     "API Key 怎么获取",
			Answer:   "An API Key is like a password that lets OpenClaw use AI services. You can get one by:\n1. Go to the AI provider's website (OpenAI, DeepSeek, or Anthropic)\n2. Create an account\n3. Navigate to API Keys section\n4. Create a new key and copy it\n\nRecommended providers:\n- DeepSeek (deepseek.com) — Cheapest, China-friendly\n- OpenAI (platform.openai.com) — Most popular\n- Anthropic (console.anthropic.com) — Highest quality",
			AnsZH:    "API Key 就像一把「钥匙」，让 OpenClaw 可以使用 AI 服务。获取方式：\n1. 去 AI 服务商网站注册账号\n2. 在 API 设置页面创建一个 Key\n3. 复制粘贴到配置中\n\n推荐服务商：\n- DeepSeek（deepseek.com）— 最便宜，国内友好\n- OpenAI（platform.openai.com）— 最流行\n- Anthropic（console.anthropic.com）— 质量最高",
		},
		{
			ID: "install_slow", Category: "install",
			Keywords: []string{"慢", "slow", "卡住", "长时间", "stuck", "很久", "等待"},
			Question: "Installation is very slow",
			QnZH:     "安装好慢",
			Answer:   "Installation speed depends on your network. Try:\n1. Check your internet connection\n2. If in China, consider using a mirror (npm mirror: npmmirror.com)\n3. WSL download may take 10-20 minutes on slow connections\n4. Node.js npm install can be slow for large packages\n\nIf truly stuck for over 30 minutes, try canceling and retrying.",
			AnsZH:    "安装速度取决于网络。建议：\n1. 检查网络连接\n2. 如果在国内，考虑使用镜像源（npm 镜像：npmmirror.com）\n3. WSL 下载在慢速网络下可能需要 10-20 分钟\n4. npm 安装大包时可能较慢\n\n如果确实卡住超过 30 分钟，可以取消后重试。",
		},
		{
			ID: "what_is_wsl", Category: "system",
			Keywords: []string{"wsl", "什么是", "windows subsystem", "linux"},
			Question: "What is WSL?",
			QnZH:     "什么是 WSL",
			Answer:   "WSL (Windows Subsystem for Linux) lets you run Linux inside Windows. OpenClaw needs it because it's a Node.js application designed to run on Linux. WSL2 provides a full Linux environment without needing a separate computer or virtual machine.",
			AnsZH:    "WSL（Windows Subsystem for Linux）让你在 Windows 里运行 Linux 系统。OpenClaw 需要它，因为 OpenClaw 是一个 Node.js 应用，设计在 Linux 上运行。WSL2 提供了完整的 Linux 环境，不需要额外的电脑或虚拟机。",
		},
		{
			ID: "virt_enable", Category: "system",
			Keywords: []string{"虚拟化", "virtualization", "vt-x", "amd-v", "bios", "开启", "enable"},
			Question: "How to enable virtualization?",
			QnZH:     "虚拟化怎么开启",
			Answer:   "To enable virtualization:\n1. Restart your computer\n2. Enter BIOS/UEFI (press F2, F12, Del, or Esc during startup)\n3. Find 'Virtualization Technology' or 'VT-x' / 'AMD-V'\n4. Set to Enabled\n5. Save (F10) and restart\n\nThe exact key and menu location varies by computer brand.",
			AnsZH:    "开启虚拟化步骤：\n1. 重启电脑\n2. 开机时按 F2/F12/Del/Esc 进入 BIOS\n3. 找到「虚拟化技术」（Virtualization Technology/VT-x/AMD-V）\n4. 设为「启用」（Enabled）\n5. 按 F10 保存并重启\n\n不同品牌电脑的按键和菜单位置不同。",
		},
		{
			ID: "what_is_node", Category: "system",
			Keywords: []string{"node", "nodejs", "是什么", "what is"},
			Question: "What is Node.js?",
			QnZH:     "Node.js 是什么",
			Answer:   "Node.js is the runtime environment that OpenClaw runs on. Think of it as the 'engine' that powers OpenClaw. We install it inside WSL so OpenClaw can run properly.",
			AnsZH:    "Node.js 是 OpenClaw 的运行环境。把它想象成 OpenClaw 的「引擎」。我们在 WSL 里安装它，让 OpenClaw 能正常运行。",
		},
		{
			ID: "error_general", Category: "troubleshoot",
			Keywords: []string{"错误", "error", "失败", "failed", "出错", "报错"},
			Question: "I got an error during installation",
			QnZH:     "安装出错了",
			Answer:   "Don't worry! Here's what to try:\n1. Click 'Retry' to try the failed step again\n2. Check the error message for clues\n3. If it's a network error, check your internet\n4. If it's a permission error, make sure you're running as administrator\n5. You can always ask me for help with the specific error!",
			AnsZH:    "别担心！试试以下方法：\n1. 点击「重试」重新执行失败的步骤\n2. 查看错误信息找线索\n3. 如果是网络错误，检查网络连接\n4. 如果是权限错误，确保以管理员身份运行\n5. 你随时可以把具体错误信息发给我，我来帮你分析！",
		},
		{
			ID: "how_to_use", Category: "usage",
			Keywords: []string{"怎么用", "how to use", "使用", "教程", "tutorial", "openclaw"},
			Question: "How to use OpenClaw?",
			QnZH:     "OpenClaw 怎么用",
			Answer:   "After installation:\n1. Open the OpenClaw console from the dashboard\n2. Connect your messaging apps (Telegram, WhatsApp, Discord)\n3. Configure your AI model and API Key\n4. Start chatting with your AI assistant through your messaging apps!\n\nThe dashboard shows your status and connected channels.",
			AnsZH:    "安装完成后：\n1. 从仪表板打开 OpenClaw 控制台\n2. 连接你的通讯工具（Telegram、WhatsApp、Discord）\n3. 配置 AI 模型和 API Key\n4. 开始通过通讯工具和 AI 助手聊天！\n\n仪表板会显示运行状态和已连接的频道。",
		},
		{
			ID: "recommend_ai", Category: "config",
			Keywords: []string{"推荐", "recommend", "哪个", "which", "ai", "服务", "service", "模型", "model"},
			Question: "Which AI service do you recommend?",
			QnZH:     "推荐哪个 AI 服务",
			Answer:   "Recommendations:\n- DeepSeek — Best value, works great in China, very affordable\n- OpenAI (GPT-4) — Most popular, best ecosystem\n- Anthropic (Claude) — Best reasoning and coding\n\nFor beginners in China, start with DeepSeek. For international users, OpenAI is the easiest.",
			AnsZH:    "推荐：\n- DeepSeek — 性价比最高，国内使用方便，价格实惠\n- OpenAI (GPT-4) — 最流行，生态最好\n- Anthropic (Claude) — 推理和编程能力最强\n\n国内新手建议先用 DeepSeek，海外用户推荐 OpenAI。",
		},
	}
}
