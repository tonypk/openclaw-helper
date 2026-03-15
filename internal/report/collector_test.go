package report

import (
	"strings"
	"testing"
	"time"

	"github.com/tonypk/openclaw-helper/internal/diagnosis"
	"github.com/tonypk/openclaw-helper/internal/installer"
	"github.com/tonypk/openclaw-helper/internal/types"
)

func sampleReport() CrashReport {
	return CrashReport{
		Title:       "Installation failed at WSL phase",
		Description: "WSL installation timed out after 5 minutes",
		AppVersion:  "0.1.0",
		GoVersion:   "go1.22.0",
		OS:          "windows",
		Arch:        "amd64",
		SystemCheck: types.SystemReport{
			OS:             types.CheckResult{Name: "os", Status: types.StatusPass, Message: "Windows 11 23H2"},
			Memory:         types.CheckResult{Name: "memory", Status: types.StatusPass, Message: "16 GB"},
			Disk:           types.CheckResult{Name: "disk", Status: types.StatusPass, Message: "50 GB available"},
			Virtualization: types.CheckResult{Name: "virtualization", Status: types.StatusPass, Message: "Hyper-V enabled"},
			WSL:            types.CheckResult{Name: "wsl", Status: types.StatusFail, Message: "WSL2 not installed"},
			Node:           types.CheckResult{Name: "node", Status: types.StatusFail, Message: "Not found"},
			OpenClaw:       types.CheckResult{Name: "openclaw", Status: types.StatusFail, Message: "Not installed"},
			OverallReady:   false,
		},
		InstallState: installer.InstallStatus{
			CurrentPhase: "error",
			ErrorPhase:   "wsl",
			ErrorMessage: "wsl --install timed out",
			Phases: []installer.PhaseProgress{
				{Phase: "precheck", Label: "Pre-check", Status: installer.PhaseCompleted},
				{Phase: "wsl", Label: "WSL", Status: installer.PhaseFailed},
			},
			Overall: 14,
		},
		DiagReport: diagnosis.DiagnosticReport{
			Issues: []diagnosis.Issue{
				{
					ID:          "wsl_not_installed",
					Severity:    diagnosis.SeverityCritical,
					Title:       "WSL2 not installed",
					Description: "WSL2 is required for OpenClaw",
				},
			},
			Healthy: false,
		},
		Timestamp:    time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		ErrorPhase:   "wsl",
		ErrorMessage: "wsl --install timed out",
	}
}

func TestFormatGitHubBody(t *testing.T) {
	r := sampleReport()
	body := FormatGitHubBody(r)

	// Check required sections
	requiredParts := []string{
		"## Environment",
		"**App Version**: 0.1.0",
		"windows/amd64",
		"## System Check",
		"| os |",
		"## Installation Error",
		"**Failed Phase**: `wsl`",
		"wsl --install timed out",
		"## Installation Progress",
		"## Diagnostic Issues",
		"WSL2 not installed",
	}

	for _, part := range requiredParts {
		if !strings.Contains(body, part) {
			t.Errorf("GitHub body missing %q", part)
		}
	}

	// Check description is included
	if !strings.Contains(body, "WSL installation timed out after 5 minutes") {
		t.Error("GitHub body missing description")
	}
}

func TestFormatGitHubBody_NoError(t *testing.T) {
	r := sampleReport()
	r.ErrorPhase = ""
	r.ErrorMessage = ""
	body := FormatGitHubBody(r)

	if strings.Contains(body, "## Installation Error") {
		t.Error("should not have Installation Error section when no error")
	}
}

func TestFormatTelegramBody(t *testing.T) {
	r := sampleReport()
	body := FormatTelegramBody(r)

	requiredParts := []string{
		"Bug Report",
		"Version: `0.1.0`",
		"OS: `windows/amd64`",
		"Phase: `wsl`",
		"Failed Checks",
		"Diagnostic Issues",
		"2026-03-15 12:00:00",
	}

	for _, part := range requiredParts {
		if !strings.Contains(body, part) {
			t.Errorf("Telegram body missing %q", part)
		}
	}
}

func TestEscapeTelegram(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"test_value", "test\\_value"},
		{"*bold*", "\\*bold\\*"},
		{"a.b.c", "a\\.b\\.c"},
	}

	for _, tc := range tests {
		got := escapeTelegram(tc.input)
		if got != tc.expected {
			t.Errorf("escapeTelegram(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestBuildSystemSummary(t *testing.T) {
	sr := types.SystemReport{
		OS:     types.CheckResult{Name: "os", Status: types.StatusPass, Message: "Windows 11"},
		Memory: types.CheckResult{Name: "memory", Status: types.StatusPass, Message: "16 GB"},
		Disk:   types.CheckResult{Name: "disk", Status: types.StatusPass, Message: "50 GB"},
	}

	summary := buildSystemSummary(sr)
	if !strings.Contains(summary, "OS: Windows 11 (pass)") {
		t.Errorf("summary missing OS info: %s", summary)
	}
	if !strings.Contains(summary, "Memory: 16 GB") {
		t.Errorf("summary missing memory info: %s", summary)
	}
}
