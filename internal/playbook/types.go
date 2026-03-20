package playbook

// Playbook defines a repair strategy for a specific installation issue.
type Playbook struct {
	ID         string     `json:"id"`
	Match      MatchRule  `json:"match"`
	Severity   string     `json:"severity"` // "critical" or "warning"
	Strategies []Strategy `json:"strategies"`
}

// MatchRule defines how to match a playbook to an error.
type MatchRule struct {
	Phase         string   `json:"phase"`          // phase name or "*" for any
	ErrorPatterns []string `json:"error_patterns"`  // regex patterns matched against logs
	CheckCommands []string `json:"check_commands"`  // optional diagnostic commands
}

// Strategy is an ordered repair attempt with multiple actions.
type Strategy struct {
	Name    string   `json:"name"`
	Risk    string   `json:"risk"` // "low", "medium", "high"
	Actions []Action `json:"actions"`
}

// Action is a single repair step within a strategy.
type Action struct {
	Type    string `json:"type"`              // "run", "retry_phase", "reboot", "set_env", "write_file"
	Script  string `json:"script,omitempty"`  // for "run": relative script path
	Timeout int    `json:"timeout,omitempty"` // for "run": seconds
	Runtime string `json:"runtime,omitempty"` // for "run": "wsl_bash" or "powershell" (default: infer from extension)
	Key     string `json:"key,omitempty"`     // for "set_env"
	Value   string `json:"value,omitempty"`   // for "set_env"
	Path    string `json:"path,omitempty"`    // for "write_file"
	Content string `json:"content,omitempty"` // for "write_file"
	Mode    string `json:"mode,omitempty"`    // for "write_file": e.g. "0644"
	Message string `json:"message,omitempty"` // for "reboot"
}

// PlaybookFile is the top-level JSON structure for playbooks/all.json.
type PlaybookFile struct {
	Playbooks []Playbook `json:"playbooks"`
}

// HealingRecord stores the result of a single healing attempt for state persistence.
type HealingRecord struct {
	Phase           string   `json:"phase"`
	Issue           string   `json:"issue"`
	StrategiesTried []string `json:"strategies_tried"`
	ResolvedBy      string   `json:"resolved_by,omitempty"`
	Attempts        int      `json:"attempts"`
	Timestamp       string   `json:"timestamp"`
}
