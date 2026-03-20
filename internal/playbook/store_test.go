package playbook

import (
	"regexp"
	"testing"
)

func TestFindMatch_ExactPhase(t *testing.T) {
	store := NewStore()

	// Manually set a playbook with exact phase match
	playbook := Playbook{
		ID: "npm_network_timeout",
		Match: MatchRule{
			Phase:         "openclaw",
			ErrorPatterns: []string{"ETIMEDOUT", "ECONNRESET"},
		},
		Severity: "critical",
	}

	pattern1 := regexp.MustCompile("(?i)ETIMEDOUT")
	pattern2 := regexp.MustCompile("(?i)ECONNRESET")

	store.playbooks = []Playbook{playbook}
	store.compiled = []compiledPlaybook{
		{
			pb:       playbook,
			patterns: []*regexp.Regexp{pattern1, pattern2},
		},
	}

	// Should match exact phase and pattern
	result := store.FindMatch("openclaw", "npm ERR! network ETIMEDOUT error occurred")
	if result == nil {
		t.Fatal("expected match, got nil")
	}
	if result.ID != "npm_network_timeout" {
		t.Errorf("expected ID npm_network_timeout, got %s", result.ID)
	}
}

func TestFindMatch_WildcardPhase(t *testing.T) {
	store := NewStore()

	playbook := Playbook{
		ID: "wildcard_playbook",
		Match: MatchRule{
			Phase:         "*",
			ErrorPatterns: []string{"ENOTFOUND"},
		},
		Severity: "warning",
	}

	pattern := regexp.MustCompile("(?i)ENOTFOUND")

	store.playbooks = []Playbook{playbook}
	store.compiled = []compiledPlaybook{
		{
			pb:       playbook,
			patterns: []*regexp.Regexp{pattern},
		},
	}

	// Should match wildcard phase
	result := store.FindMatch("openclaw", "getaddrinfo ENOTFOUND registry.npmjs.org")
	if result == nil {
		t.Fatal("expected match with wildcard phase, got nil")
	}
	if result.ID != "wildcard_playbook" {
		t.Errorf("expected ID wildcard_playbook, got %s", result.ID)
	}
}

func TestFindMatch_NoMatch(t *testing.T) {
	store := NewStore()

	playbook := Playbook{
		ID: "specific_playbook",
		Match: MatchRule{
			Phase:         "openclaw",
			ErrorPatterns: []string{"ETIMEDOUT"},
		},
	}

	pattern := regexp.MustCompile("(?i)ETIMEDOUT")

	store.playbooks = []Playbook{playbook}
	store.compiled = []compiledPlaybook{
		{
			pb:       playbook,
			patterns: []*regexp.Regexp{pattern},
		},
	}

	// Should not match - pattern is not in log
	result := store.FindMatch("openclaw", "npm ERR! some other error")
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestFindMatch_PhaseMismatch(t *testing.T) {
	store := NewStore()

	playbook := Playbook{
		ID: "npm_playbook",
		Match: MatchRule{
			Phase:         "npm_install",
			ErrorPatterns: []string{"ETIMEDOUT"},
		},
	}

	pattern := regexp.MustCompile("(?i)ETIMEDOUT")

	store.playbooks = []Playbook{playbook}
	store.compiled = []compiledPlaybook{
		{
			pb:       playbook,
			patterns: []*regexp.Regexp{pattern},
		},
	}

	// Pattern matches but phase doesn't
	result := store.FindMatch("openclaw", "npm ERR! network ETIMEDOUT error")
	if result != nil {
		t.Errorf("expected nil due to phase mismatch, got %+v", result)
	}
}

func TestLoadFromJSON(t *testing.T) {
	jsonData := `{
		"playbooks": [
			{
				"id": "test_playbook_1",
				"match": {
					"phase": "openclaw",
					"error_patterns": ["ERROR1", "ERROR2"]
				},
				"severity": "critical",
				"strategies": []
			},
			{
				"id": "test_playbook_2",
				"match": {
					"phase": "*",
					"error_patterns": ["ERROR3"]
				},
				"severity": "warning",
				"strategies": []
			}
		]
	}`

	store := NewStore()
	err := store.LoadFromJSON([]byte(jsonData))
	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	count := store.Count()
	if count != 2 {
		t.Errorf("expected 2 playbooks, got %d", count)
	}

	// Verify first playbook
	result := store.FindMatch("openclaw", "error ERROR1")
	if result == nil {
		t.Fatal("expected to find first playbook")
	}
	if result.ID != "test_playbook_1" {
		t.Errorf("expected test_playbook_1, got %s", result.ID)
	}

	// Verify second playbook with wildcard
	result = store.FindMatch("any_phase", "error ERROR3")
	if result == nil {
		t.Fatal("expected to find second playbook with wildcard phase")
	}
	if result.ID != "test_playbook_2" {
		t.Errorf("expected test_playbook_2, got %s", result.ID)
	}
}

func TestReload(t *testing.T) {
	store := NewStore()

	// Load initial playbooks
	jsonData1 := `{
		"playbooks": [
			{
				"id": "playbook_v1",
				"match": {
					"phase": "phase1",
					"error_patterns": ["ERROR_V1"]
				},
				"severity": "critical",
				"strategies": []
			}
		]
	}`

	err := store.LoadFromJSON([]byte(jsonData1))
	if err != nil {
		t.Fatalf("first LoadFromJSON failed: %v", err)
	}

	if store.Count() != 1 {
		t.Errorf("expected 1 playbook after first load, got %d", store.Count())
	}

	result := store.FindMatch("phase1", "ERROR_V1")
	if result == nil {
		t.Fatal("expected to find playbook_v1")
	}

	// Reload with different playbooks (should replace)
	jsonData2 := `{
		"playbooks": [
			{
				"id": "playbook_v2",
				"match": {
					"phase": "phase2",
					"error_patterns": ["ERROR_V2"]
				},
				"severity": "warning",
				"strategies": []
			},
			{
				"id": "playbook_v3",
				"match": {
					"phase": "*",
					"error_patterns": ["ERROR_V3"]
				},
				"severity": "critical",
				"strategies": []
			}
		]
	}`

	err = store.LoadFromJSON([]byte(jsonData2))
	if err != nil {
		t.Fatalf("second LoadFromJSON failed: %v", err)
	}

	if store.Count() != 2 {
		t.Errorf("expected 2 playbooks after reload, got %d", store.Count())
	}

	// Old playbook should not be found
	result = store.FindMatch("phase1", "ERROR_V1")
	if result != nil {
		t.Errorf("old playbook should not be found after reload, got %+v", result)
	}

	// New playbooks should be found
	result = store.FindMatch("phase2", "ERROR_V2")
	if result == nil {
		t.Fatal("expected to find playbook_v2 after reload")
	}

	result = store.FindMatch("any_phase", "ERROR_V3")
	if result == nil {
		t.Fatal("expected to find playbook_v3 after reload")
	}
}
