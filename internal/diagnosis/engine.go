// Package diagnosis provides a rule-based diagnostics engine with repair playbooks.
package diagnosis

import (
	"encoding/json"
	"regexp"
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

// RemoteRule describes a diagnostic rule loaded from remote JSON.
type RemoteRule struct {
	ID            string   `json:"id"`
	Severity      string   `json:"severity"`
	Title         string   `json:"title"`
	TitleZH       string   `json:"title_zh"`
	Description   string   `json:"description"`
	DescZH        string   `json:"desc_zh"`
	ErrorPatterns []string `json:"error_patterns"`
	RepairID      string   `json:"repair_id"`
	AutoRepair    bool     `json:"auto_repair"`
}

// RemoteRulesFile is the top-level JSON structure for diagnostics/rules.json.
type RemoteRulesFile struct {
	Rules []RemoteRule `json:"rules"`
}

// LoadRemoteRules loads diagnostic rules from JSON data and merges with builtins.
// Builtin rules take priority on ID conflict.
func (e *Engine) LoadRemoteRules(data []byte) error {
	var file RemoteRulesFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	// Build set of builtin rule IDs
	builtinIDs := make(map[string]bool)
	for _, r := range e.rules {
		builtinIDs[r.ID] = true
	}

	// Convert remote rules to engine Rules (regex-based matching)
	for _, rr := range file.Rules {
		if builtinIDs[rr.ID] {
			continue // builtin takes priority
		}

		severity := SeverityWarning
		switch rr.Severity {
		case "critical":
			severity = SeverityCritical
		case "info":
			severity = SeverityInfo
		}

		// Compile patterns
		patterns := make([]*regexp.Regexp, 0, len(rr.ErrorPatterns))
		for _, p := range rr.ErrorPatterns {
			re, err := regexp.Compile("(?i)" + p)
			if err != nil {
				continue
			}
			patterns = append(patterns, re)
		}

		capturedRule := rr
		capturedPatterns := patterns
		capturedSeverity := severity

		e.rules = append(e.rules, Rule{
			ID: capturedRule.ID,
			Match: func(ctx *DiagContext) bool {
				if ctx.ErrorLog == "" {
					return false
				}
				for _, re := range capturedPatterns {
					if re.MatchString(ctx.ErrorLog) {
						return true
					}
				}
				return false
			},
			Diagnose: func(ctx *DiagContext) Issue {
				return Issue{
					ID:          capturedRule.ID,
					Severity:    capturedSeverity,
					Title:       capturedRule.Title,
					TitleZH:     capturedRule.TitleZH,
					Description: capturedRule.Description,
					DescZH:      capturedRule.DescZH,
					RepairID:    capturedRule.RepairID,
					AutoRepair:  capturedRule.AutoRepair,
				}
			},
		})
	}

	return nil
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
