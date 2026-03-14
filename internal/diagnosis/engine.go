// Package diagnosis provides a rule-based diagnostics engine with repair playbooks.
package diagnosis

import (
	"strings"
	"time"
)

// Severity levels for diagnostic issues.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Issue represents a diagnosed problem.
type Issue struct {
	ID          string   `json:"id"`
	Severity    Severity `json:"severity"`
	Title       string   `json:"title"`
	TitleZH     string   `json:"title_zh"`
	Description string   `json:"description"`
	DescZH      string   `json:"desc_zh"`
	RepairID    string   `json:"repair_id,omitempty"`
	AutoRepair  bool     `json:"auto_repair"`
}

// DiagnosticReport is the result of running all diagnostic rules.
type DiagnosticReport struct {
	Issues    []Issue   `json:"issues"`
	Healthy   bool      `json:"healthy"`
	Timestamp time.Time `json:"timestamp"`
}

// Rule matches a symptom and produces an Issue.
type Rule struct {
	ID       string
	Match    func(ctx *DiagContext) bool
	Diagnose func(ctx *DiagContext) Issue
}

// Engine runs diagnostic rules against collected system context.
type Engine struct {
	rules []Rule
}

// NewEngine creates a diagnosis engine with all built-in rules.
func NewEngine() *Engine {
	return &Engine{rules: builtinRules()}
}

// NewEngineWithRules creates an engine with custom rules (for testing).
func NewEngineWithRules(rules []Rule) *Engine {
	return &Engine{rules: rules}
}

// Diagnose runs all rules and returns a report.
func (e *Engine) Diagnose(ctx *DiagContext) DiagnosticReport {
	report := DiagnosticReport{
		Healthy:   true,
		Timestamp: time.Now(),
	}

	for _, rule := range e.rules {
		if rule.Match(ctx) {
			issue := rule.Diagnose(ctx)
			report.Issues = append(report.Issues, issue)
			if issue.Severity == SeverityCritical {
				report.Healthy = false
			}
		}
	}

	return report
}

// DiagContext holds collected system state for diagnosis.
type DiagContext struct {
	WSLInstalled       bool
	WSLVersion         int
	WSLDistros         []string
	NodeInstalled      bool
	NodeVersion        string
	NodeMajor          int
	OpenClawInstalled  bool
	GatewayReachable   bool
	VirtEnabled        bool
	DiskFreeGB         float64
	MemoryGB           float64
	OSBuild            int
	ErrorLog           string
	CurrentPhase       string
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(lower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
