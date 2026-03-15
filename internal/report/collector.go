// Package report provides crash report collection and submission.
package report

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/tonypk/openclaw-helper/internal/checker"
	"github.com/tonypk/openclaw-helper/internal/diagnosis"
	"github.com/tonypk/openclaw-helper/internal/installer"
	"github.com/tonypk/openclaw-helper/internal/types"
)

// CrashReport contains all diagnostic information for a crash report.
type CrashReport struct {
	Title        string                  `json:"title"`
	Description  string                  `json:"description"`
	AppVersion   string                  `json:"app_version"`
	GoVersion    string                  `json:"go_version"`
	OS           string                  `json:"os"`
	Arch         string                  `json:"arch"`
	SystemCheck  types.SystemReport      `json:"system_check"`
	InstallState installer.InstallStatus `json:"install_state"`
	DiagReport   diagnosis.DiagnosticReport `json:"diag_report"`
	Timestamp    time.Time               `json:"timestamp"`

	// Derived summary fields for frontend display
	SystemSummary string `json:"system_summary"`
	ErrorPhase    string `json:"error_phase"`
	ErrorMessage  string `json:"error_message"`
}

// ReportResult is returned after submitting a report.
type ReportResult struct {
	GitHubURL    string `json:"github_url"`
	TelegramSent bool   `json:"telegram_sent"`
}

// Collect gathers all diagnostic information into a CrashReport.
func Collect(appVersion string, sc *checker.SystemChecker, orch *installer.Orchestrator, diagEngine *diagnosis.Engine) CrashReport {
	sysReport := sc.RunAll()
	installStatus := orch.Status()
	diagCtx := diagnosis.CollectWithError(sc, installStatus.ErrorMessage)
	diagReport := diagEngine.Diagnose(diagCtx)

	report := CrashReport{
		AppVersion:   appVersion,
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		SystemCheck:  sysReport,
		InstallState: installStatus,
		DiagReport:   diagReport,
		Timestamp:    time.Now(),
		ErrorPhase:   string(installStatus.ErrorPhase),
		ErrorMessage: installStatus.ErrorMessage,
	}

	report.SystemSummary = buildSystemSummary(sysReport)

	return report
}

func buildSystemSummary(sr types.SystemReport) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("OS: %s (%s)", sr.OS.Message, sr.OS.Status))
	parts = append(parts, fmt.Sprintf("Memory: %s", sr.Memory.Message))
	parts = append(parts, fmt.Sprintf("Disk: %s", sr.Disk.Message))
	parts = append(parts, fmt.Sprintf("Virtualization: %s (%s)", sr.Virtualization.Message, sr.Virtualization.Status))
	parts = append(parts, fmt.Sprintf("WSL: %s (%s)", sr.WSL.Message, sr.WSL.Status))
	parts = append(parts, fmt.Sprintf("Node: %s (%s)", sr.Node.Message, sr.Node.Status))
	parts = append(parts, fmt.Sprintf("OpenClaw: %s (%s)", sr.OpenClaw.Message, sr.OpenClaw.Status))
	return strings.Join(parts, "\n")
}

// FormatGitHubBody formats a CrashReport as a GitHub issue body in Markdown.
func FormatGitHubBody(r CrashReport) string {
	var b strings.Builder

	if r.Description != "" {
		b.WriteString(r.Description)
		b.WriteString("\n\n")
	}

	b.WriteString("## Environment\n\n")
	b.WriteString(fmt.Sprintf("- **App Version**: %s\n", r.AppVersion))
	b.WriteString(fmt.Sprintf("- **Go Version**: %s\n", r.GoVersion))
	b.WriteString(fmt.Sprintf("- **OS/Arch**: %s/%s\n", r.OS, r.Arch))
	b.WriteString(fmt.Sprintf("- **Timestamp**: %s\n", r.Timestamp.Format(time.RFC3339)))
	b.WriteString("\n")

	// System check results
	b.WriteString("## System Check\n\n")
	b.WriteString("| Check | Status | Message |\n")
	b.WriteString("|-------|--------|----------|\n")
	checks := []types.CheckResult{
		r.SystemCheck.OS,
		r.SystemCheck.Memory,
		r.SystemCheck.Disk,
		r.SystemCheck.Virtualization,
		r.SystemCheck.WSL,
		r.SystemCheck.Node,
		r.SystemCheck.OpenClaw,
	}
	for _, c := range checks {
		icon := statusIcon(c.Status)
		b.WriteString(fmt.Sprintf("| %s | %s %s | %s |\n", c.Name, icon, c.Status, c.Message))
	}
	b.WriteString("\n")

	// Install status
	if r.ErrorPhase != "" {
		b.WriteString("## Installation Error\n\n")
		b.WriteString(fmt.Sprintf("- **Failed Phase**: `%s`\n", r.ErrorPhase))
		b.WriteString(fmt.Sprintf("- **Error**: %s\n", r.ErrorMessage))
		b.WriteString("\n")
	}

	// Phase progress
	b.WriteString("## Installation Progress\n\n")
	for _, p := range r.InstallState.Phases {
		icon := phaseIcon(p.Status)
		b.WriteString(fmt.Sprintf("- %s %s: %s\n", icon, p.Label, p.Status))
	}
	b.WriteString("\n")

	// Diagnostic issues
	if len(r.DiagReport.Issues) > 0 {
		b.WriteString("## Diagnostic Issues\n\n")
		for _, issue := range r.DiagReport.Issues {
			sev := severityIcon(issue.Severity)
			b.WriteString(fmt.Sprintf("- %s **%s**: %s\n", sev, issue.Title, issue.Description))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// FormatTelegramBody formats a CrashReport as a concise Telegram message.
func FormatTelegramBody(r CrashReport) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("🐛 *Bug Report: %s*\n\n", escapeTelegram(r.Title)))

	if r.Description != "" {
		b.WriteString(fmt.Sprintf("📝 %s\n\n", escapeTelegram(r.Description)))
	}

	b.WriteString("*Environment:*\n")
	b.WriteString(fmt.Sprintf("• Version: `%s`\n", r.AppVersion))
	b.WriteString(fmt.Sprintf("• OS: `%s/%s`\n", r.OS, r.Arch))
	b.WriteString(fmt.Sprintf("• Go: `%s`\n\n", r.GoVersion))

	if r.ErrorPhase != "" {
		b.WriteString("*Error:*\n")
		b.WriteString(fmt.Sprintf("• Phase: `%s`\n", r.ErrorPhase))
		b.WriteString(fmt.Sprintf("• Message: %s\n\n", escapeTelegram(r.ErrorMessage)))
	}

	// Failed system checks only
	failedChecks := []types.CheckResult{}
	for _, c := range []types.CheckResult{
		r.SystemCheck.OS, r.SystemCheck.Memory, r.SystemCheck.Disk,
		r.SystemCheck.Virtualization, r.SystemCheck.WSL,
		r.SystemCheck.Node, r.SystemCheck.OpenClaw,
	} {
		if c.Status == types.StatusFail || c.Status == types.StatusWarn {
			failedChecks = append(failedChecks, c)
		}
	}
	if len(failedChecks) > 0 {
		b.WriteString("*Failed Checks:*\n")
		for _, c := range failedChecks {
			b.WriteString(fmt.Sprintf("• %s: %s (%s)\n", c.Name, escapeTelegram(c.Message), c.Status))
		}
		b.WriteString("\n")
	}

	if len(r.DiagReport.Issues) > 0 {
		b.WriteString("*Diagnostic Issues:*\n")
		for _, issue := range r.DiagReport.Issues {
			b.WriteString(fmt.Sprintf("• [%s] %s\n", issue.Severity, escapeTelegram(issue.Title)))
		}
	}

	b.WriteString(fmt.Sprintf("\n🕐 %s", r.Timestamp.Format("2006-01-02 15:04:05")))

	return b.String()
}

func statusIcon(s types.CheckStatus) string {
	switch s {
	case types.StatusPass:
		return "✅"
	case types.StatusFail:
		return "❌"
	case types.StatusWarn:
		return "⚠️"
	case types.StatusSkipped:
		return "⏭️"
	default:
		return "⏳"
	}
}

func phaseIcon(s installer.PhaseStatus) string {
	switch s {
	case installer.PhaseCompleted:
		return "✅"
	case installer.PhaseFailed:
		return "❌"
	case installer.PhaseRunning:
		return "🔄"
	case installer.PhaseSkipped:
		return "⏭️"
	default:
		return "⬜"
	}
}

func severityIcon(s diagnosis.Severity) string {
	switch s {
	case diagnosis.SeverityCritical:
		return "🔴"
	case diagnosis.SeverityWarning:
		return "🟡"
	default:
		return "🔵"
	}
}

// escapeTelegram escapes special characters for Telegram MarkdownV2.
func escapeTelegram(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}
