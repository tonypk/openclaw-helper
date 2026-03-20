# Self-Healing Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a self-healing loop to every installation phase — automatic diagnosis, playbook-driven repair, retry, and escalation to AI/crash report.

**Architecture:** New `internal/playbook` and `internal/healing` packages. Playbook store loads repair strategies from remote manifest V2. Healing executor wraps the existing `PhaseExecutor` interface, adding diagnose → repair → retry loop. Frontend gets a new `HealingProgress` component and `useHealingStore`.

**Tech Stack:** Go 1.25.0, Vue3 + TypeScript + Pinia, Tauri v2, JSON-RPC 2.0

**Spec:** `docs/superpowers/specs/2026-03-20-self-healing-engine-design.md`

---

## File Structure

### New Files (Go Backend)

| File | Responsibility |
|------|---------------|
| `internal/playbook/types.go` | Playbook, MatchRule, Strategy, Action type definitions |
| `internal/playbook/store.go` | Load playbooks from cache/remote/embedded, match against errors |
| `internal/playbook/store_test.go` | Unit tests for playbook matching and loading |
| `internal/healing/executor.go` | Self-healing executor: diagnose → repair → retry loop |
| `internal/healing/executor_test.go` | Unit tests for healing loop, retry logic, escalation |
| `internal/healing/events.go` | Healing event types and helpers |
| `internal/scriptrun/fallback/repair/npm-mirror.sh` | Embedded fallback: switch npm registry |
| `internal/scriptrun/fallback/repair/fix-path.sh` | Embedded fallback: rebuild PATH |

### New Files (Frontend)

| File | Responsibility |
|------|---------------|
| `frontend/src/stores/healing.ts` | Pinia store for healing state per phase |
| `frontend/src/components/install/HealingProgress.vue` | Expandable healing progress UI |

### Modified Files

| File | Change |
|------|--------|
| `internal/playbook/` (dir) | Create new package directory |
| `internal/healing/` (dir) | Create new package directory |
| `internal/scriptrun/manifest.go` | Add V2 fields: Playbooks, Diagnostics, FAQ, Config, RepairScripts |
| `internal/scriptrun/cache.go` | Extend Sync() to handle all V2 resource types |
| `internal/scriptrun/parser.go` | Add HEAL message type parsing |
| `internal/installer/phases.go` | Add HealingHistory to InstallState |
| `internal/installer/orchestrator.go` | Inject healing.Executor, delegate phase execution |
| `internal/diagnosis/engine.go` | Add LoadRemoteRules() for JSON-based rules |
| `internal/chat/context.go` | Add HealingHistory field to ChatContext, append to system prompt in BuildSystemPrompt() |
| `internal/report/collector.go` | Include healing_history in CrashReport via new Orchestrator.GetHealingHistory() method |
| `internal/installer/orchestrator.go` | Add GetHealingHistory() method for report collector access |
| `internal/scriptrun/runner.go` | Add RunContent() method to execute raw script bytes |
| `cmd/helper/main.go` | Wire up playbook store + healing executor |
| `frontend/src/api/helper.ts` | Add HealingEvent type |
| `frontend/src/stores/install.ts` | Forward healing events to healing store |
| `frontend/src/views/InstallView.vue` | Integrate HealingProgress component |
| `frontend/src/i18n/en-US.ts` | Add healing.* keys |
| `frontend/src/i18n/zh-CN.ts` | Add healing.* keys |

---

## Task 1: Playbook Types

**Files:**
- Create: `internal/playbook/types.go`

- [ ] **Step 1: Create playbook package directory**

```bash
mkdir -p internal/playbook
```

- [ ] **Step 2: Write playbook type definitions**

Create `internal/playbook/types.go`:

```go
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
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/playbook/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/playbook/types.go
git commit -m "feat: add playbook type definitions"
```

---

## Task 2: Playbook Store

**Files:**
- Create: `internal/playbook/store.go`
- Create: `internal/playbook/store_test.go`

- [ ] **Step 1: Write failing test for playbook matching**

Create `internal/playbook/store_test.go`:

```go
package playbook

import (
	"context"
	"testing"
)

func TestFindMatch_ExactPhase(t *testing.T) {
	store := &Store{}
	store.playbooks = []Playbook{
		{
			ID:       "npm_network_timeout",
			Match:    MatchRule{Phase: "openclaw", ErrorPatterns: []string{"ETIMEDOUT", "ECONNRESET"}},
			Severity: "critical",
			Strategies: []Strategy{
				{Name: "retry_with_mirror", Risk: "low", Actions: []Action{{Type: "run", Script: "repair/npm-mirror.sh"}}},
			},
		},
	}

	pb := store.FindMatch("openclaw", "npm ERR! network ETIMEDOUT 104.16.23.35:443")
	if pb == nil {
		t.Fatal("expected playbook match, got nil")
	}
	if pb.ID != "npm_network_timeout" {
		t.Fatalf("expected npm_network_timeout, got %s", pb.ID)
	}
}

func TestFindMatch_WildcardPhase(t *testing.T) {
	store := &Store{}
	store.playbooks = []Playbook{
		{
			ID:    "dns_resolution_fail",
			Match: MatchRule{Phase: "*", ErrorPatterns: []string{"ENOTFOUND", "DNS"}},
		},
	}

	pb := store.FindMatch("openclaw", "getaddrinfo ENOTFOUND registry.npmjs.org")
	if pb == nil {
		t.Fatal("expected wildcard playbook match, got nil")
	}
}

func TestFindMatch_NoMatch(t *testing.T) {
	store := &Store{}
	store.playbooks = []Playbook{
		{
			ID:    "npm_network_timeout",
			Match: MatchRule{Phase: "openclaw", ErrorPatterns: []string{"ETIMEDOUT"}},
		},
	}

	pb := store.FindMatch("wsl", "some other error")
	if pb != nil {
		t.Fatalf("expected nil, got %s", pb.ID)
	}
}

func TestFindMatch_PhaseMismatch(t *testing.T) {
	store := &Store{}
	store.playbooks = []Playbook{
		{
			ID:    "npm_network_timeout",
			Match: MatchRule{Phase: "openclaw", ErrorPatterns: []string{"ETIMEDOUT"}},
		},
	}

	pb := store.FindMatch("node", "ETIMEDOUT error")
	if pb != nil {
		t.Fatalf("expected nil for wrong phase, got %s", pb.ID)
	}
}

func TestLoadFromJSON(t *testing.T) {
	store := &Store{}
	data := []byte(`{"playbooks":[{"id":"test_pb","match":{"phase":"*","error_patterns":["test"]},"severity":"warning","strategies":[]}]}`)

	err := store.LoadFromJSON(data)
	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}
	if len(store.playbooks) != 1 {
		t.Fatalf("expected 1 playbook, got %d", len(store.playbooks))
	}
	if store.playbooks[0].ID != "test_pb" {
		t.Fatalf("expected test_pb, got %s", store.playbooks[0].ID)
	}
}

func TestReload(t *testing.T) {
	store := &Store{}
	store.playbooks = []Playbook{{ID: "old"}}

	data := []byte(`{"playbooks":[{"id":"new","match":{"phase":"*","error_patterns":["x"]},"severity":"warning","strategies":[]}]}`)
	err := store.LoadFromJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if store.playbooks[0].ID != "new" {
		t.Fatalf("expected new, got %s", store.playbooks[0].ID)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./internal/playbook/ -v`
Expected: FAIL — `Store` struct and methods not defined

- [ ] **Step 3: Implement playbook store**

Create `internal/playbook/store.go`:

```go
package playbook

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
)

// Store manages playbook loading, caching, and matching.
type Store struct {
	playbooks []Playbook
	compiled  []compiledPlaybook
	mu        sync.RWMutex
}

type compiledPlaybook struct {
	playbook Playbook
	patterns []*regexp.Regexp
}

// NewStore creates an empty playbook store.
func NewStore() *Store {
	return &Store{}
}

// LoadFromJSON parses playbooks from JSON data (playbooks/all.json format).
func (s *Store) LoadFromJSON(data []byte) error {
	var file PlaybookFile
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}

	compiled := make([]compiledPlaybook, 0, len(file.Playbooks))
	for _, pb := range file.Playbooks {
		cp := compiledPlaybook{playbook: pb}
		for _, pattern := range pb.Match.ErrorPatterns {
			re, err := regexp.Compile("(?i)" + pattern)
			if err != nil {
				// Skip invalid patterns, use substring fallback
				continue
			}
			cp.patterns = append(cp.patterns, re)
		}
		compiled = append(compiled, cp)
	}

	s.mu.Lock()
	s.playbooks = file.Playbooks
	s.compiled = compiled
	s.mu.Unlock()
	return nil
}

// FindMatch returns the first playbook matching the given phase and error log.
// Returns nil if no match found.
func (s *Store) FindMatch(phase string, errorLog string) *Playbook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logLower := strings.ToLower(errorLog)

	for i, cp := range s.compiled {
		pb := cp.playbook

		// Phase must match exactly or be wildcard "*"
		if pb.Match.Phase != "*" && pb.Match.Phase != phase {
			continue
		}

		// At least one error pattern must match
		matched := false
		if len(cp.patterns) > 0 {
			for _, re := range cp.patterns {
				if re.MatchString(errorLog) {
					matched = true
					break
				}
			}
		} else {
			// Fallback: substring match on raw patterns
			for _, pattern := range pb.Match.ErrorPatterns {
				if strings.Contains(logLower, strings.ToLower(pattern)) {
					matched = true
					break
				}
			}
		}

		if matched {
			result := s.playbooks[i]
			return &result
		}
	}
	return nil
}

// Count returns the number of loaded playbooks.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.playbooks)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./internal/playbook/ -v`
Expected: All 6 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/playbook/store.go internal/playbook/store_test.go
git commit -m "feat: add playbook store with matching logic"
```

---

## Task 3: Manifest V2 + Cache Extension

**Files:**
- Modify: `internal/scriptrun/manifest.go`
- Modify: `internal/scriptrun/cache.go`

- [ ] **Step 1: Extend manifest types for V2**

In `internal/scriptrun/manifest.go`, add these types after the existing `Manifest` struct:

```go
// ResourceEntry describes a hot-updatable resource (playbooks, diagnostics, faq, config).
type ResourceEntry struct {
	Version int    `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// RepairScriptEntry describes a single repair script.
type RepairScriptEntry struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}
```

Add new fields to the existing `Manifest` struct:

```go
type Manifest struct {
	Version          int                          `json:"version"`
	MinHelperVersion string                       `json:"min_helper_version,omitempty"`
	Phases           map[string]*PhaseScripts     `json:"phases"`
	Playbooks        *ResourceEntry               `json:"playbooks,omitempty"`
	Diagnostics      *ResourceEntry               `json:"diagnostics,omitempty"`
	FAQ              *ResourceEntry               `json:"faq,omitempty"`
	Config           *ResourceEntry               `json:"config,omitempty"`
	RepairScripts    map[string]*RepairScriptEntry `json:"repair_scripts,omitempty"`
}
```

- [ ] **Step 2: Add resource fetch methods to Cache**

In `internal/scriptrun/cache.go`, add after the existing `Sync()` method:

```go
// SyncResources downloads playbooks, diagnostics, FAQ, and config if manifest V2.
func (c *Cache) SyncResources() error {
	c.mu.RLock()
	m := c.manifest
	c.mu.RUnlock()

	if m == nil {
		return nil
	}

	resources := map[string]*ResourceEntry{
		"playbooks":   m.Playbooks,
		"diagnostics": m.Diagnostics,
		"faq":         m.FAQ,
		"config":      m.Config,
	}

	for name, res := range resources {
		if res == nil {
			continue
		}
		// Save resource using canonical name (e.g. "playbooks.json") not the URL path,
		// so GetResource(name) can find it at <cacheDir>/playbooks.json
		localPath := filepath.Join(c.dir, name+".json")
		if c.hashMatches(localPath, res.SHA256) {
			continue
		}
		if err := c.downloadResourceTo(res.URL, res.SHA256, localPath); err != nil {
			log.Printf("[cache] failed to download %s: %v", name, err)
		}
	}

	// Download repair scripts
	if m.RepairScripts != nil {
		for _, entry := range m.RepairScripts {
			localPath := filepath.Join(c.dir, entry.URL)
			if c.hashMatches(localPath, entry.SHA256) {
				continue
			}
			if err := c.downloadScript(entry.URL, entry.SHA256); err != nil {
				log.Printf("[cache] failed to download repair script %s: %v", entry.URL, err)
			}
		}
	}

	return nil
}

// downloadResourceTo downloads a resource and saves it to a specific local path (not derived from URL).
func (c *Cache) downloadResourceTo(relURL, expectedHash, localPath string) error {
	// Reuse downloadScript logic but save to localPath instead of c.scriptPath(relURL)
	fullURL := baseRawURL + relURL
	// ... HTTP GET with timeout, hash verification, write to localPath ...
	// Implementation: copy the core of downloadScript() but use localPath for os.WriteFile
	return nil // placeholder — implementer fills in using downloadScript pattern
}

// GetResource returns the content of a cached resource file (playbooks.json, diagnostics.json, etc.).
func (c *Cache) GetResource(name string) ([]byte, error) {
	localPath := filepath.Join(c.dir, name+".json")
	return os.ReadFile(localPath)
}

// GetRepairScript returns the content of a cached repair script.
func (c *Cache) GetRepairScript(relativePath string) ([]byte, error) {
	localPath := filepath.Join(c.dir, relativePath)
	return os.ReadFile(localPath)
}

// ForceSync re-fetches the manifest and all resources, ignoring ETag cache.
func (c *Cache) ForceSync() error {
	c.mu.Lock()
	c.etag = ""
	c.mu.Unlock()
	if err := c.Sync(); err != nil {
		return err
	}
	return c.SyncResources()
}
```

Also update the existing `Sync()` method to call `SyncResources()` at the end:

At the end of the `Sync()` method, before `return nil`, add:
```go
	if err := c.SyncResources(); err != nil {
		log.Printf("[cache] resource sync failed: %v", err)
	}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/scriptrun/`
Expected: No errors

- [ ] **Step 4: Run existing tests**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./internal/scriptrun/ -v`
Expected: All existing tests still pass

- [ ] **Step 5: Commit**

```bash
git add internal/scriptrun/manifest.go internal/scriptrun/cache.go
git commit -m "feat: extend manifest and cache for V2 resources"
```

---

## Task 4: OCH Protocol HEAL Extensions

**Files:**
- Modify: `internal/scriptrun/parser.go`

- [ ] **Step 1: Add HEAL message type to parser**

In `internal/scriptrun/parser.go`, add to the `MessageType` constants:

```go
const (
	// ... existing types ...
	MsgHeal MessageType = "HEAL"
)
```

Add fields to `ProtocolMessage`:

```go
type ProtocolMessage struct {
	// ... existing fields ...
	HealType   string // "START", "STRATEGY", "REPAIR", "RETRY", "RESOLVED", "ESCALATE"
	HealIssue  string
	HealDetail string
}
```

In the `ParseLine()` function, add a case in the switch for `"HEAL"`:

```go
	case "HEAL":
		msg.Type = MsgHeal
		// Format: HEAL:SUBTYPE:payload
		healParts := strings.SplitN(rest, ":", 2)
		msg.HealType = healParts[0]
		if len(healParts) > 1 {
			// For REPAIR, format is REPAIR:script=status
			msg.HealDetail = healParts[1]
		}
		// Extract issue from START/RESOLVED
		if msg.HealType == "START" || msg.HealType == "RESOLVED" {
			msg.HealIssue = msg.HealDetail
		}
```

Where `rest` is the part after `##OCH:HEAL:` — check how the existing parser splits. The existing code does:
```go
parts := strings.SplitN(line[len(protocolPrefix):], ":", 2)
typStr := parts[0]
rest := parts[1]  // if len(parts) > 1
```

So for `##OCH:HEAL:START:npm_network_timeout`, `typStr = "HEAL"`, `rest = "START:npm_network_timeout"`.

- [ ] **Step 2: Run existing parser tests**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./internal/scriptrun/ -v`
Expected: All existing tests still pass (new code only adds a case, doesn't break existing)

- [ ] **Step 3: Commit**

```bash
git add internal/scriptrun/parser.go
git commit -m "feat: add HEAL message type to OCH protocol parser"
```

---

## Task 5: Healing Events

**Files:**
- Create: `internal/healing/events.go`

- [ ] **Step 1: Create healing package and event types**

```bash
mkdir -p internal/healing
```

Create `internal/healing/events.go`:

```go
package healing

// EventType identifies the kind of healing event.
type EventType string

const (
	EventHealStart    EventType = "heal_start"
	EventHealStrategy EventType = "heal_strategy"
	EventHealRepair   EventType = "heal_repair"
	EventHealRetry    EventType = "heal_retry"
	EventHealResolved EventType = "heal_resolved"
	EventHealEscalate EventType = "heal_escalate"
)

// Event represents a healing lifecycle event emitted to the frontend.
type Event struct {
	Type     EventType `json:"type"`
	Issue    string    `json:"issue,omitempty"`
	Strategy string    `json:"strategy,omitempty"`
	Detail   string    `json:"detail,omitempty"`
	Attempt  int       `json:"attempt,omitempty"`
	MaxRetry int       `json:"max_retry,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/healing/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/healing/events.go
git commit -m "feat: add healing event types"
```

---

## Task 6: Healing Executor

**Files:**
- Create: `internal/healing/executor.go`
- Create: `internal/healing/executor_test.go`

- [ ] **Step 1: Write failing tests for healing executor**

Create `internal/healing/executor_test.go`:

```go
package healing

import (
	"context"
	"testing"

	"github.com/tonypk/openclaw-helper/internal/playbook"
)

// mockPhaseRunner simulates script execution for testing.
type mockPhaseRunner struct {
	results []PhaseRunResult // returns results in order
	callIdx int
}

func (m *mockPhaseRunner) RunPhase(ctx context.Context, phase string) PhaseRunResult {
	if m.callIdx >= len(m.results) {
		return PhaseRunResult{Success: true}
	}
	r := m.results[m.callIdx]
	m.callIdx++
	return r
}

// mockRepairRunner simulates repair script execution.
type mockRepairRunner struct {
	successes map[string]bool // script name → success
}

func (m *mockRepairRunner) RunRepairScript(ctx context.Context, script string, timeout int) error {
	if m.successes[script] {
		return nil
	}
	return errRepairFailed
}

func TestExecutor_SuccessOnFirstTry(t *testing.T) {
	runner := &mockPhaseRunner{results: []PhaseRunResult{{Success: true}}}
	store := playbook.NewStore()
	events := make([]Event, 0)

	ex := NewExecutor(&mockRepairRunner{}, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if !result.Success {
		t.Fatal("expected success")
	}
	if len(events) != 0 {
		t.Fatalf("expected no healing events, got %d", len(events))
	}
}

func TestExecutor_FailThenHealThenSucceed(t *testing.T) {
	runner := &mockPhaseRunner{
		results: []PhaseRunResult{
			{Success: false, ErrorLog: "npm ERR! network ETIMEDOUT"},
			{Success: true},
		},
	}
	store := playbook.NewStore()
	store.LoadFromJSON([]byte(`{"playbooks":[{
		"id":"npm_network_timeout",
		"match":{"phase":"openclaw","error_patterns":["ETIMEDOUT"]},
		"severity":"critical",
		"strategies":[{"name":"mirror","risk":"low","actions":[{"type":"run","script":"repair/npm-mirror.sh","timeout":30},{"type":"retry_phase"}]}]
	}]}`))

	repair := &mockRepairRunner{successes: map[string]bool{"repair/npm-mirror.sh": true}}
	events := make([]Event, 0)

	ex := NewExecutor(repair, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if !result.Success {
		t.Fatal("expected success after healing")
	}
	if len(result.HealingHistory) != 1 {
		t.Fatalf("expected 1 healing record, got %d", len(result.HealingHistory))
	}
	if result.HealingHistory[0].ResolvedBy != "mirror" {
		t.Fatalf("expected resolved by mirror, got %s", result.HealingHistory[0].ResolvedBy)
	}
}

func TestExecutor_AllStrategiesFail_Escalate(t *testing.T) {
	runner := &mockPhaseRunner{
		results: []PhaseRunResult{
			{Success: false, ErrorLog: "ETIMEDOUT"},
			{Success: false, ErrorLog: "ETIMEDOUT"},
			{Success: false, ErrorLog: "ETIMEDOUT"},
			{Success: false, ErrorLog: "ETIMEDOUT"},
		},
	}
	store := playbook.NewStore()
	store.LoadFromJSON([]byte(`{"playbooks":[{
		"id":"test",
		"match":{"phase":"*","error_patterns":["ETIMEDOUT"]},
		"severity":"critical",
		"strategies":[{"name":"s1","risk":"low","actions":[{"type":"run","script":"s1.sh","timeout":10},{"type":"retry_phase"}]}]
	}]}`))

	repair := &mockRepairRunner{successes: map[string]bool{}}
	events := make([]Event, 0)

	ex := NewExecutor(repair, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if result.Success {
		t.Fatal("expected failure after exhausting retries")
	}
	// Check escalation event was emitted
	hasEscalate := false
	for _, e := range events {
		if e.Type == EventHealEscalate {
			hasEscalate = true
		}
	}
	if !hasEscalate {
		t.Fatal("expected escalate event")
	}
}

func TestExecutor_NoPlaybook_Escalate(t *testing.T) {
	runner := &mockPhaseRunner{
		results: []PhaseRunResult{{Success: false, ErrorLog: "unknown error xyz"}},
	}
	store := playbook.NewStore()
	events := make([]Event, 0)

	ex := NewExecutor(&mockRepairRunner{}, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if result.Success {
		t.Fatal("expected failure when no playbook matches")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./internal/healing/ -v`
Expected: FAIL — `Executor`, `PhaseRunResult`, etc. not defined

- [ ] **Step 3: Implement healing executor**

Create `internal/healing/executor.go`:

```go
package healing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tonypk/openclaw-helper/internal/playbook"
)

var errRepairFailed = errors.New("repair action failed")

// PhaseRunResult holds the outcome of running a phase script.
type PhaseRunResult struct {
	Success      bool
	ErrorLog     string
	NeedsReboot  bool
	ErrorMessage string
}

// PhaseResult is the final result of ExecutePhase, including healing history.
type PhaseResult struct {
	Success        bool
	NeedsReboot    bool
	ErrorLog       string
	ErrorMessage   string
	HealingHistory []playbook.HealingRecord
	Escalated      bool
}

// PhaseRunner executes a phase's install script. Abstracted for testing.
type PhaseRunner interface {
	RunPhase(ctx context.Context, phase string) PhaseRunResult
}

// RepairRunner executes repair scripts. Abstracted for testing.
type RepairRunner interface {
	RunRepairScript(ctx context.Context, script string, timeout int) error
}

// ForceSync re-fetches the manifest and all resources. Abstracted for testing.
type ManifestSyncer interface {
	ForceSync() error
}

// EventCallback receives healing events for frontend display.
type EventCallback func(Event)

// Executor wraps phase execution with self-healing: diagnose → repair → retry.
// PhaseRunner is passed per-call (since each phase has a different executor).
type Executor struct {
	repairRunner RepairRunner
	store        *playbook.Store
	syncer       ManifestSyncer // for force-sync on playbook miss
	maxRetries   int
	onEvent      EventCallback
}

// NewExecutor creates a healing executor.
func NewExecutor(
	repairRunner RepairRunner,
	store *playbook.Store,
	syncer ManifestSyncer,
	maxRetries int,
	onEvent EventCallback,
) *Executor {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if onEvent == nil {
		onEvent = func(Event) {}
	}
	return &Executor{
		repairRunner: repairRunner,
		store:        store,
		syncer:       syncer,
		maxRetries:   maxRetries,
		onEvent:      onEvent,
	}
}

// ExecutePhase runs a phase with self-healing loop.
// The PhaseRunner is passed per-call since each phase has a different executor adapter.
func (e *Executor) ExecutePhase(ctx context.Context, phase string, runner PhaseRunner) PhaseResult {
	var history []playbook.HealingRecord

	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return PhaseResult{ErrorMessage: "cancelled"}
		default:
		}

		result := runner.RunPhase(ctx, phase)

		if result.Success {
			return PhaseResult{
				Success:        true,
				HealingHistory: history,
			}
		}

		if result.NeedsReboot {
			return PhaseResult{
				NeedsReboot:    true,
				HealingHistory: history,
			}
		}

		// Find matching playbook
		pb := e.store.FindMatch(phase, result.ErrorLog)

		// If no playbook found, try force-syncing manifest (maybe one was just published)
		if pb == nil && e.syncer != nil {
			_ = e.syncer.ForceSync()
			pb = e.store.FindMatch(phase, result.ErrorLog)
		}

		if pb == nil {
			// Still no playbook — escalate immediately
			e.onEvent(Event{Type: EventHealEscalate, Detail: "ai_chat", Issue: "unknown"})
			return PhaseResult{
				ErrorLog:       result.ErrorLog,
				ErrorMessage:   result.ErrorMessage,
				HealingHistory: history,
				Escalated:      true,
			}
		}

		e.onEvent(Event{Type: EventHealStart, Issue: pb.ID, Attempt: attempt + 1, MaxRetry: e.maxRetries})

		record := playbook.HealingRecord{
			Phase:     phase,
			Issue:     pb.ID,
			Attempts:  attempt + 1,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		// Try strategies in order
		repaired := false
		for _, strategy := range pb.Strategies {
			e.onEvent(Event{Type: EventHealStrategy, Issue: pb.ID, Strategy: strategy.Name})

			strategyOK := e.executeStrategy(ctx, strategy)
			record.StrategiesTried = append(record.StrategiesTried, strategy.Name)

			if strategyOK {
				record.ResolvedBy = strategy.Name
				repaired = true
				e.onEvent(Event{Type: EventHealRetry, Issue: pb.ID, Attempt: attempt + 1, MaxRetry: e.maxRetries})
				break
			}
		}

		history = append(history, record)

		if !repaired {
			// All strategies failed for this attempt
			if attempt == e.maxRetries {
				break
			}
			// Continue to next attempt (will re-run phase and re-diagnose)
			continue
		}

		// Strategy succeeded — retry phase (loop continues with next attempt)
	}

	// Exhausted all retries
	e.onEvent(Event{Type: EventHealEscalate, Detail: "ai_chat"})
	return PhaseResult{
		ErrorMessage:   "all repair strategies exhausted",
		HealingHistory: history,
		Escalated:      true,
	}
}

// executeStrategy runs all actions in a strategy. Returns true if all actions succeed.
func (e *Executor) executeStrategy(ctx context.Context, strategy playbook.Strategy) bool {
	for _, action := range strategy.Actions {
		switch action.Type {
		case "run":
			e.onEvent(Event{
				Type:   EventHealRepair,
				Detail: fmt.Sprintf("%s=running", action.Script),
			})
			timeout := action.Timeout
			if timeout == 0 {
				timeout = 60
			}
			err := e.repairRunner.RunRepairScript(ctx, action.Script, timeout)
			if err != nil {
				e.onEvent(Event{
					Type:   EventHealRepair,
					Detail: fmt.Sprintf("%s=failed", action.Script),
				})
				return false
			}
			e.onEvent(Event{
				Type:   EventHealRepair,
				Detail: fmt.Sprintf("%s=success", action.Script),
			})

		case "retry_phase":
			// Signal to caller that phase should be retried
			// This is handled by the loop in ExecutePhase
			return true

		case "reboot":
			// Reboot is handled at a higher level
			return true

		case "set_env", "write_file":
			// These are simple enough to always succeed in this context
			// Actual implementation would shell out
			continue
		}
	}
	return true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./internal/healing/ -v`
Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/healing/executor.go internal/healing/executor_test.go
git commit -m "feat: add self-healing executor with diagnose-repair-retry loop"
```

---

## Task 7: Extend InstallState with HealingHistory

**Files:**
- Modify: `internal/installer/phases.go`

- [ ] **Step 1: Add HealingHistory field to InstallState**

In `internal/installer/phases.go`, add the import for the playbook package and add the field:

```go
import (
	// ... existing imports ...
	"github.com/tonypk/openclaw-helper/internal/playbook"
)
```

Add to `InstallState` struct (after `Metadata`):

```go
type InstallState struct {
	// ... existing fields ...
	HealingHistory []playbook.HealingRecord `json:"healing_history,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/installer/`
Expected: No errors

- [ ] **Step 3: Run existing tests**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./internal/installer/ -v`
Expected: All existing tests pass (new field is optional/omitempty)

- [ ] **Step 4: Commit**

```bash
git add internal/installer/phases.go
git commit -m "feat: add HealingHistory to InstallState"
```

---

## Task 8: Extend Diagnosis Engine for Remote Rules

**Files:**
- Modify: `internal/diagnosis/engine.go`

- [ ] **Step 1: Add LoadRemoteRules to engine**

In `internal/diagnosis/engine.go`, add a method to load rules from JSON:

```go
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
					Severity:    severity,
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
```

Add `"encoding/json"` and `"regexp"` to the imports if not already present.

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/diagnosis/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/diagnosis/engine.go
git commit -m "feat: add remote rule loading to diagnosis engine"
```

---

## Task 9: Extend Chat Handler with Healing Context

**Files:**
- Modify: `internal/chat/context.go` (NOT handler.go — ChatContext is defined in context.go)

- [ ] **Step 1: Add healing context to ChatContext**

In `internal/chat/context.go`, add the `HealingHistory` field to the existing `ChatContext` struct:

```go
type ChatContext struct {
	// ... existing fields (language, currentPhase, errorLog, messages) ...
	HealingHistory string // serialized healing history for LLM context
}
```

Add a setter method on `Handler` in the same file (or handler.go if setters are there):

```go
// SetHealingContext provides healing history for richer AI prompts.
func (h *Handler) SetHealingContext(history string) {
	h.ctx.HealingHistory = history
}
```

In `context.go`, find the `BuildSystemPrompt()` method (where the LLM system message is constructed). Append healing context at the end:

```go
if c.HealingHistory != "" {
	prompt += "\n\nThe user's installation encountered issues. Auto-repair was attempted but failed. Here is the healing history:\n" + c.HealingHistory + "\n\nAnalyze the error and healing attempts to provide specific, actionable advice."
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/chat/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/chat/context.go
git commit -m "feat: add healing context to chat handler for richer AI prompts"
```

---

## Task 10: Extend Crash Report with Healing History

**Files:**
- Modify: `internal/report/collector.go`
- Modify: `internal/installer/orchestrator.go` (add GetHealingHistory method)

- [ ] **Step 1: Add GetHealingHistory to Orchestrator**

In `internal/installer/orchestrator.go`, add a method to expose healing history:

```go
// GetHealingHistory returns the current healing history for crash reports.
func (o *Orchestrator) GetHealingHistory() []playbook.HealingRecord {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.state == nil {
		return nil
	}
	// Return a copy to avoid mutation
	result := make([]playbook.HealingRecord, len(o.state.HealingHistory))
	copy(result, o.state.HealingHistory)
	return result
}
```

- [ ] **Step 2: Add healing history to CrashReport**

In `internal/report/collector.go`, add to the `CrashReport` struct:

```go
type CrashReport struct {
	// ... existing fields ...
	HealingHistory string `json:"healing_history,omitempty"`
}
```

In the `Collect()` function, after existing install status collection, add:

```go
// Collect healing history (uses new GetHealingHistory method, not Status())
if orch != nil {
	history := orch.GetHealingHistory()
	if len(history) > 0 {
		historyJSON, _ := json.MarshalIndent(history, "", "  ")
		report.HealingHistory = string(historyJSON)
	}
}
```

In `FormatGitHubBody()`, add a healing history section:

```go
if report.HealingHistory != "" {
	body += "\n### Healing History\n\n"
	body += "```json\n" + report.HealingHistory + "\n```\n"
}
```

In `FormatTelegramBody()`, add:

```go
if report.HealingHistory != "" {
	body += "\nHealing History\n"
	body += "`" + escapeTelegram(report.HealingHistory) + "`\n"
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/report/ && go build ./internal/installer/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/report/collector.go internal/installer/orchestrator.go
git commit -m "feat: include healing history in crash reports"
```

---

## Task 11: Add RunContent to scriptrun.Runner

**Files:**
- Modify: `internal/scriptrun/runner.go`

- [ ] **Step 1: Add RunContent method**

In `internal/scriptrun/runner.go`, add a method that executes raw script bytes (instead of fetching from cache). This is needed for repair scripts loaded via `Cache.GetRepairScript()`:

```go
// RunContent executes raw script content with the given entry's runtime and timeout settings.
// This is used for repair scripts loaded from cache as raw bytes.
// Mirrors the existing Run() method but skips cache lookup.
func (r *Runner) RunContent(ctx context.Context, entry *ScriptEntry, content []byte, onProgress ProgressFunc) RunResult {
	timeout := time.Duration(entry.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	tCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// executeScript returns (io.Reader, int, error) — 3 values
	reader, exitCode, err := executeScript(tCtx, entry.Runtime, entry.Distro, string(content))
	if err != nil {
		return RunResult{ExitCode: exitCode, ErrorMessage: err.Error()}
	}

	var result RunResult
	result.ExitCode = exitCode
	r.parseOutput(reader, &result, onProgress)
	return result
}
```

Note: This reuses the existing `executeScript()` function (platform-specific in `_windows.go`/`_stub.go`) and `parseOutput()`. The only difference from `Run()` is it skips the cache lookup.

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/scriptrun/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/scriptrun/runner.go
git commit -m "feat: add RunContent method to scriptrun.Runner"
```

---

## Task 12: Wire Up in main.go

**Depends on:** Task 13 (orchestrator methods `SetHealingExecutor` and `EmitHealingEvent` are defined there). Implement Task 12 and 13 together, or implement Task 13 first.

**Files:**
- Modify: `cmd/helper/main.go`

- [ ] **Step 1: Create and wire playbook store + healing executor**

In `cmd/helper/main.go`, in the `runServer()` function, after creating the script cache and before creating the orchestrator:

Add imports:
```go
"github.com/tonypk/openclaw-helper/internal/playbook"
"github.com/tonypk/openclaw-helper/internal/healing"
```

After `cache := scriptrun.NewCache(...)` and before creating executors:

```go
	// Load playbook store
	pbStore := playbook.NewStore()
	if pbData, err := cache.GetResource("playbooks"); err == nil {
		if err := pbStore.LoadFromJSON(pbData); err != nil {
			log.Printf("[main] failed to parse playbooks: %v", err)
		} else {
			log.Printf("[main] loaded %d playbooks", pbStore.Count())
		}
	}

	// Load remote diagnostic rules
	if diagData, err := cache.GetResource("diagnostics"); err == nil {
		if err := diagEngine.LoadRemoteRules(diagData); err != nil {
			log.Printf("[main] failed to load remote diag rules: %v", err)
		}
	}
```

After creating the orchestrator, wire the healing executor:

```go
	// Create repair runner (uses script cache + runner)
	repairRunner := &scriptRepairRunner{runner: runner, cache: cache}

	// Create healing executor (runner is passed per-phase call via adapter)
	healExec := healing.NewExecutor(
		repairRunner,
		pbStore,
		cache, // implements ForceSync() for manifest re-sync on playbook miss
		3,
		func(e healing.Event) {
			// Forward healing events as ProgressEvents for frontend
			log.Printf("[healing] %s: %s %s", e.Type, e.Issue, e.Detail)
			orch.EmitHealingEvent(e)
		},
	)
	orch.SetHealingExecutor(healExec)
```

Add the repair runner adapter in main.go:

```go
// scriptRepairRunner executes repair scripts via the script cache + runner.
type scriptRepairRunner struct {
	runner *scriptrun.Runner
	cache  *scriptrun.Cache
}

func (r *scriptRepairRunner) RunRepairScript(ctx context.Context, script string, timeout int) error {
	content, err := r.cache.GetRepairScript(script)
	if err != nil {
		return fmt.Errorf("repair script not found: %s: %w", script, err)
	}
	runtime := scriptrun.RuntimeWSLBash
	if strings.HasSuffix(script, ".ps1") {
		runtime = scriptrun.RuntimePowerShell
	}
	entry := &scriptrun.ScriptEntry{
		Runtime:        runtime,
		TimeoutSeconds: timeout,
	}
	result := r.runner.RunContent(ctx, entry, content, nil)
	if result.ExitCode != 0 {
		return fmt.Errorf("repair script %s exited with code %d: %s", script, result.ExitCode, result.ErrorMessage)
	}
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./cmd/helper/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add cmd/helper/main.go
git commit -m "feat: wire playbook store and healing executor in main"
```

---

## Task 13: Integrate Healing into Orchestrator

**Files:**
- Modify: `internal/installer/orchestrator.go`

- [ ] **Step 1: Add healing executor to orchestrator**

In `internal/installer/orchestrator.go`, add imports and fields:

```go
import (
	// ... existing ...
	"encoding/json"
	"github.com/tonypk/openclaw-helper/internal/healing"
	"github.com/tonypk/openclaw-helper/internal/playbook"
)

type Orchestrator struct {
	// ... existing fields ...
	healingExec *healing.Executor
}
```

Add setter and event bridge methods:

```go
// SetHealingExecutor configures the self-healing executor for automatic repair.
func (o *Orchestrator) SetHealingExecutor(exec *healing.Executor) {
	o.healingExec = exec
}

// EmitHealingEvent converts a healing.Event into a ProgressEvent for the frontend.
func (o *Orchestrator) EmitHealingEvent(e healing.Event) {
	detail, _ := json.Marshal(e)
	o.emit(ProgressEvent{
		Phase:   o.state.CurrentPhase,
		Status:  PhaseRunning,
		Message: "HEAL:" + string(e.Type),
		Detail:  string(detail),
	})
}
```

- [ ] **Step 2: Replace executor call with healing wrapper**

In the `run()` method, find where `executor.Execute(ctx, progressCb)` is called. Wrap it with healing logic:

```go
		if o.healingExec != nil {
			// Use healing executor (wraps phase execution with repair loop)
			adapter := &phaseRunnerAdapter{executor: executor, progressCb: progressCb}
			healResult := o.healingExec.ExecutePhase(ctx, string(phase), adapter)

			if len(healResult.HealingHistory) > 0 {
				o.mu.Lock()
				o.state.HealingHistory = append(o.state.HealingHistory, healResult.HealingHistory...)
				o.mu.Unlock()
				o.state.Save()
			}

			if healResult.NeedsReboot {
				o.emit(ProgressEvent{Phase: phase, Status: PhaseReboot, Message: "Reboot required"})
				return
			}
			if !healResult.Success {
				o.mu.Lock()
				o.state.ErrorMessage = healResult.ErrorMessage
				o.state.ErrorPhase = string(phase)
				o.mu.Unlock()
				o.emit(ProgressEvent{Phase: phase, Status: PhaseFailed, Message: healResult.ErrorMessage})
				return
			}
		} else {
			// Original path without healing
			needsReboot, err := executor.Execute(ctx, progressCb)
			if err != nil {
				// ... keep existing error handling unchanged ...
			}
			if needsReboot {
				// ... keep existing reboot handling unchanged ...
			}
		}
```

Add the adapter type at file level:

```go
// phaseRunnerAdapter adapts PhaseExecutor to healing.PhaseRunner interface.
// It captures the full script output log (from progress events) so the healing
// executor can match error patterns like "ETIMEDOUT" against the actual output,
// not just the wrapped error message.
type phaseRunnerAdapter struct {
	executor   PhaseExecutor
	progressCb ProgressCallback
}

func (a *phaseRunnerAdapter) RunPhase(ctx context.Context, phase string) healing.PhaseRunResult {
	// Capture all progress event messages to build the full error log
	var logBuilder strings.Builder

	wrappedCb := func(evt ProgressEvent) {
		// Accumulate all messages and details into the log
		if evt.Message != "" {
			logBuilder.WriteString(evt.Message)
			logBuilder.WriteString("\n")
		}
		if evt.Detail != "" {
			logBuilder.WriteString(evt.Detail)
			logBuilder.WriteString("\n")
		}
		// Forward to the original callback for frontend display
		if a.progressCb != nil {
			a.progressCb(evt)
		}
	}

	needsReboot, err := a.executor.Execute(ctx, wrappedCb)
	fullLog := logBuilder.String()

	if err != nil {
		// Combine the error message with the full captured log
		// so pattern matching works against actual script output
		errorLog := fullLog
		if errorLog == "" {
			errorLog = err.Error()
		}
		return healing.PhaseRunResult{
			Success:      false,
			ErrorLog:     errorLog,
			ErrorMessage: err.Error(),
		}
	}
	if needsReboot {
		return healing.PhaseRunResult{NeedsReboot: true}
	}
	return healing.PhaseRunResult{Success: true}
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./cmd/helper/`
Expected: No errors

- [ ] **Step 4: Run all Go tests**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./... -v`
Expected: All tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/installer/orchestrator.go
git commit -m "feat: integrate healing executor into orchestrator run loop"
```

---

## Task 14: Frontend — Healing Store

**Files:**
- Create: `frontend/src/stores/healing.ts`

- [ ] **Step 1: Create healing Pinia store**

Create `frontend/src/stores/healing.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface HealingStrategy {
  name: string
  status: 'pending' | 'running' | 'success' | 'failed'
}

export interface HealingRecord {
  phase: string
  issue: string
  resolvedBy: string
}

export const useHealingStore = defineStore('healing', () => {
  const activePhase = ref<string | null>(null)
  const currentIssue = ref<string | null>(null)
  const strategies = ref<HealingStrategy[]>([])
  const repairLog = ref<string[]>([])
  const history = ref<HealingRecord[]>([])
  const escalated = ref(false)

  const isHealing = computed(() => activePhase.value !== null && !escalated.value)
  const healedCount = computed(() => history.value.length)

  function onHealEvent(event: { type: string; issue?: string; strategy?: string; detail?: string; attempt?: number; max_retry?: number }) {
    switch (event.type) {
      case 'heal_start':
        activePhase.value = event.issue ?? null
        currentIssue.value = event.issue ?? null
        strategies.value = []
        escalated.value = false
        break

      case 'heal_strategy':
        // Mark previous strategy as failed if it was running
        strategies.value = strategies.value.map(s =>
          s.status === 'running' ? { ...s, status: 'failed' as const } : s
        )
        strategies.value = [...strategies.value, { name: event.strategy ?? '', status: 'running' }]
        break

      case 'heal_repair':
        if (event.detail) {
          repairLog.value = [...repairLog.value, event.detail]
        }
        break

      case 'heal_retry':
        // Mark current strategy as success
        strategies.value = strategies.value.map(s =>
          s.status === 'running' ? { ...s, status: 'success' as const } : s
        )
        break

      case 'heal_resolved':
        const resolvedIssue = currentIssue.value ?? ''
        const resolvedBy = strategies.value.find(s => s.status === 'success')?.name ?? ''
        history.value = [...history.value, {
          phase: activePhase.value ?? '',
          issue: resolvedIssue,
          resolvedBy,
        }]
        activePhase.value = null
        currentIssue.value = null
        strategies.value = []
        break

      case 'heal_escalate':
        escalated.value = true
        // Mark all remaining strategies as failed
        strategies.value = strategies.value.map(s =>
          s.status === 'pending' || s.status === 'running'
            ? { ...s, status: 'failed' as const }
            : s
        )
        break
    }
  }

  function reset() {
    activePhase.value = null
    currentIssue.value = null
    strategies.value = []
    repairLog.value = []
    history.value = []
    escalated.value = false
  }

  return {
    activePhase,
    currentIssue,
    strategies,
    repairLog,
    history,
    escalated,
    isHealing,
    healedCount,
    onHealEvent,
    reset,
  }
})
```

- [ ] **Step 2: Verify no TypeScript errors**

Run: `cd /Users/anna/Documents/openclaw-helper/frontend && npx vue-tsc --noEmit`
Expected: No type errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/healing.ts
git commit -m "feat: add healing Pinia store"
```

---

## Task 15: Frontend — HealingProgress Component

**Files:**
- Create: `frontend/src/components/install/HealingProgress.vue`

- [ ] **Step 1: Create HealingProgress component**

Create `frontend/src/components/install/HealingProgress.vue`:

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useHealingStore } from '../../stores/healing'
import { useI18n } from 'vue-i18n'

const healing = useHealingStore()
const { t } = useI18n()

const statusIcon = (status: string) => {
  switch (status) {
    case 'success': return '✓'
    case 'running': return '●'
    case 'failed': return '✗'
    default: return '○'
  }
}

const statusColor = (status: string) => {
  switch (status) {
    case 'success': return '#4ade80'
    case 'running': return '#60a5fa'
    case 'failed': return '#f87171'
    default: return '#475569'
  }
}
</script>

<template>
  <!-- Healing in progress -->
  <div v-if="healing.isHealing" class="healing-panel">
    <div class="healing-header">
      <span class="healing-icon">🔧</span>
      <span class="healing-title">{{ t('healing.autoRepairing') }}</span>
    </div>

    <div class="healing-issue">
      {{ healing.currentIssue }}
    </div>

    <div class="healing-strategies">
      <div
        v-for="s in healing.strategies"
        :key="s.name"
        class="strategy-item"
      >
        <span class="strategy-icon" :style="{ color: statusColor(s.status) }">
          {{ statusIcon(s.status) }}
        </span>
        <span class="strategy-name" :class="{ active: s.status === 'running' }">
          {{ s.name }}
        </span>
      </div>
    </div>

    <div v-if="healing.repairLog.length > 0" class="repair-log">
      <div v-for="(line, i) in healing.repairLog.slice(-5)" :key="i" class="log-line">
        &gt; {{ line }}
      </div>
    </div>
  </div>

  <!-- Healing resolved badge -->
  <div v-if="healing.healedCount > 0 && !healing.isHealing && !healing.escalated" class="healing-badge">
    <span class="badge-text">
      {{ t('healing.resolved', { n: healing.healedCount }) }}
    </span>
    <div class="badge-details">
      <div v-for="h in healing.history" :key="h.issue" class="badge-detail-item">
        {{ h.issue }} → {{ h.resolvedBy }}
      </div>
    </div>
  </div>

  <!-- Healing escalated -->
  <div v-if="healing.escalated" class="healing-escalated">
    {{ t('healing.escalateAI') }}
  </div>
</template>

<style scoped>
.healing-panel {
  background: #1a2332;
  border: 1px solid #2a3a4a;
  border-radius: 8px;
  padding: 12px;
  margin: 8px 0;
  font-size: 13px;
}

.healing-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.healing-icon { font-size: 16px; }
.healing-title { color: #fbbf24; font-weight: 600; }
.healing-issue { color: #64748b; font-size: 12px; margin-bottom: 8px; }

.healing-strategies {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.strategy-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.strategy-icon { font-size: 12px; width: 14px; text-align: center; }
.strategy-name { color: #94a3b8; font-size: 12px; }
.strategy-name.active { color: #60a5fa; }

.repair-log {
  background: #0f172a;
  border-radius: 4px;
  padding: 8px;
  margin-top: 8px;
  font-family: monospace;
  font-size: 10px;
  color: #94a3b8;
  max-height: 80px;
  overflow-y: auto;
}

.log-line { line-height: 1.6; }

.healing-badge {
  background: #1a2332;
  color: #f59e0b;
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 4px;
  display: inline-block;
  margin-left: 8px;
}

.badge-details {
  color: #4ade80;
  font-size: 10px;
  margin-top: 4px;
}

.badge-detail-item { line-height: 1.6; }

.healing-escalated {
  color: #f87171;
  font-size: 12px;
  margin-top: 4px;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/install/HealingProgress.vue
git commit -m "feat: add HealingProgress component"
```

---

## Task 16: Frontend — i18n + InstallView Integration

**Files:**
- Modify: `frontend/src/i18n/en-US.ts`
- Modify: `frontend/src/i18n/zh-CN.ts`
- Modify: `frontend/src/views/InstallView.vue`
- Modify: `frontend/src/stores/install.ts`
- Modify: `frontend/src/api/helper.ts`

- [ ] **Step 1: Add healing i18n keys to en-US**

In `frontend/src/i18n/en-US.ts`, add in the translation object:

```typescript
  healing: {
    autoRepairing: 'Auto-repairing...',
    resolved: 'Auto-repaired {n} issue(s)',
    strategy: 'Strategy: {name}',
    escalateAI: 'Auto-repair failed, AI assistant taking over',
    escalateReport: 'Submit detailed report to developer',
  },
```

- [ ] **Step 2: Add healing i18n keys to zh-CN**

In `frontend/src/i18n/zh-CN.ts`, add:

```typescript
  healing: {
    autoRepairing: '自动修复中...',
    resolved: '自动修复了 {n} 个问题',
    strategy: '策略: {name}',
    escalateAI: '自动修复失败，AI 助手已接管',
    escalateReport: '提交详细报告给开发者',
  },
```

- [ ] **Step 3: Add HealingEvent type to helper.ts**

In `frontend/src/api/helper.ts`, add type:

```typescript
export interface HealingEvent {
  type: 'heal_start' | 'heal_strategy' | 'heal_repair' | 'heal_retry' | 'heal_resolved' | 'heal_escalate'
  issue?: string
  strategy?: string
  detail?: string
  attempt?: number
  max_retry?: number
}
```

- [ ] **Step 4: Forward healing events from install store**

In `frontend/src/stores/install.ts`, import and use healing store:

```typescript
import { useHealingStore } from './healing'
import type { HealingEvent } from '../api/helper'
```

In the `fetchEvents()` method, after processing new progress events into the events array, add healing event forwarding. The backend emits healing events as `ProgressEvent` with `message` starting with `"HEAL:"` and `detail` containing a JSON-serialized `healing.Event` (see `EmitHealingEvent` in `orchestrator.go` which sets `Message: "HEAL:" + string(e.Type)` and `Detail: JSON(e)`):

```typescript
// Inside fetchEvents(), after adding newEvents to the events array:
const healingStore = useHealingStore()
for (const event of newEvents) {
  if (event.message?.startsWith('HEAL:')) {
    try {
      const healData: HealingEvent = JSON.parse(event.detail || '{}')
      healingStore.onHealEvent(healData)
    } catch {
      // Ignore malformed healing events
    }
  }
}
```

This works because Task 13's `EmitHealingEvent()` method on the orchestrator converts each `healing.Event` into a standard `ProgressEvent` with the `"HEAL:"` prefix in the message field and the full event JSON in the detail field. The frontend polls these via the existing `install.events` RPC endpoint.

- [ ] **Step 5: Integrate HealingProgress into InstallView**

In `frontend/src/views/InstallView.vue`, import and add the component:

```typescript
import HealingProgress from '../components/install/HealingProgress.vue'
```

In the template, add after `<StepProgress>` and before `<LogViewer>`:

```vue
<HealingProgress />
```

- [ ] **Step 6: Verify build**

Run: `cd /Users/anna/Documents/openclaw-helper/frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 7: Commit**

```bash
git add frontend/src/i18n/en-US.ts frontend/src/i18n/zh-CN.ts frontend/src/api/helper.ts frontend/src/stores/install.ts frontend/src/stores/healing.ts frontend/src/views/InstallView.vue
git commit -m "feat: integrate healing UI into install view with i18n"
```

---

## Task 17: Embedded Fallback Repair Scripts

**Files:**
- Create: `internal/scriptrun/fallback/repair/npm-mirror.sh`
- Create: `internal/scriptrun/fallback/repair/fix-path.sh`

- [ ] **Step 1: Create repair scripts directory**

```bash
mkdir -p internal/scriptrun/fallback/repair
```

- [ ] **Step 2: Create npm-mirror.sh**

Create `internal/scriptrun/fallback/repair/npm-mirror.sh`:

```bash
#!/bin/bash
set -e

echo "##OCH:HEAL:REPAIR:npm-mirror.sh=running"

# Try multiple registries
REGISTRIES=(
    "https://registry.npmmirror.com"
    "https://registry.npmjs.org"
)

for reg in "${REGISTRIES[@]}"; do
    echo "Trying registry: $reg"
    if curl -s --max-time 5 "$reg" > /dev/null 2>&1; then
        npm config set registry "$reg"
        echo "Set npm registry to $reg"
        echo "##OCH:HEAL:REPAIR:npm-mirror.sh=success"
        exit 0
    fi
done

echo "All registries unreachable"
echo "##OCH:HEAL:REPAIR:npm-mirror.sh=failed"
exit 1
```

- [ ] **Step 3: Create fix-path.sh**

Create `internal/scriptrun/fallback/repair/fix-path.sh`:

```bash
#!/bin/bash
set -e

echo "##OCH:HEAL:REPAIR:fix-path.sh=running"

# Rebuild PATH with known locations
PATHS_TO_ADD=(
    "$HOME/.nvm/versions/node/$(ls -1 $HOME/.nvm/versions/node/ 2>/dev/null | tail -1)/bin"
    "$HOME/.local/bin"
    "/usr/local/bin"
)

BASHRC="$HOME/.bashrc"

for p in "${PATHS_TO_ADD[@]}"; do
    if [ -d "$p" ] && ! echo "$PATH" | grep -q "$p"; then
        echo "export PATH=\"$p:\$PATH\"" >> "$BASHRC"
        export PATH="$p:$PATH"
        echo "Added $p to PATH"
    fi
done

# Source nvm if available
if [ -s "$HOME/.nvm/nvm.sh" ]; then
    source "$HOME/.nvm/nvm.sh"
    echo "Sourced nvm"
fi

echo "##OCH:HEAL:REPAIR:fix-path.sh=success"
exit 0
```

- [ ] **Step 4: Update embed directive**

In `internal/scriptrun/embed_test.go` or wherever the `go:embed` directive exists for fallback scripts, add the repair directory to the embed pattern. Find the existing embed and extend it:

```go
//go:embed fallback/*
var fallbackFS embed.FS
```

This should already cover `fallback/repair/` since it uses a wildcard.

- [ ] **Step 5: Verify compilation**

Run: `cd /Users/anna/Documents/openclaw-helper && go build ./internal/scriptrun/`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add internal/scriptrun/fallback/repair/
git commit -m "feat: add embedded fallback repair scripts"
```

---

## Task 18: openclaw-scripts Repo — Playbooks + Repair Scripts + Manifest V2

**Files (in openclaw-scripts repo):**
- Create: `playbooks/all.json`
- Create: `diagnostics/rules.json`
- Create: `scripts/repair/npm-mirror.sh`
- Create: `scripts/repair/npm-clear-cache.sh`
- Create: `scripts/repair/fix-path.sh`
- Create: `scripts/repair/fix-apt-locks.sh`
- Create: `scripts/repair/set-dns.sh`
- Create: `scripts/repair/clean-disk.sh`
- Modify: `manifest.json` → V2

- [ ] **Step 1: Create playbooks/all.json with 10 initial playbooks**

Create `playbooks/all.json` in the openclaw-scripts repo with all 10 playbooks from the spec (npm_network_timeout, npm_sharp_build, wsl_virt_disabled, wsl_install_fail, node_path_missing, node_version_wrong, ubuntu_apt_fail, dns_resolution_fail, disk_space_low, path_corruption).

- [ ] **Step 2: Create diagnostics/rules.json**

Externalize the 15 diagnostic rules as JSON.

- [ ] **Step 3: Create repair scripts (6 scripts)**

Write the actual repair logic for each script. Each must use `##OCH:HEAL:REPAIR:` protocol for status reporting.

- [ ] **Step 4: Update manifest.json to V2**

Add `playbooks`, `diagnostics`, `faq`, `config`, and `repair_scripts` sections. Compute SHA-256 hashes for all new files.

- [ ] **Step 5: Commit and push openclaw-scripts**

```bash
cd /Users/anna/Documents/openclaw-scripts
git add playbooks/ diagnostics/ scripts/repair/ faq/ config/ manifest.json
git commit -m "feat: add playbooks, repair scripts, and manifest V2"
git push
```

---

## Task 19: Full Integration Test

**Files:**
- Run all tests

- [ ] **Step 1: Run all Go tests**

Run: `cd /Users/anna/Documents/openclaw-helper && go test ./... -v -count=1`
Expected: All tests pass

- [ ] **Step 2: Run frontend build**

Run: `cd /Users/anna/Documents/openclaw-helper/frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 3: Run full Tauri dev build (if on Windows)**

Run: `cd /Users/anna/Documents/openclaw-helper && make dev`
Expected: App launches, installation flow shows healing UI

- [ ] **Step 4: Commit version bump**

```bash
# Update version in tauri.conf.json and Go
git commit -m "chore: bump version to v0.5.0"
```
