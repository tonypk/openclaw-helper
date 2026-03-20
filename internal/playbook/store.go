package playbook

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
)

// Store holds playbooks and provides thread-safe lookup operations.
type Store struct {
	playbooks []Playbook
	compiled  []compiledPlaybook
	mu        sync.RWMutex
}

// compiledPlaybook pairs a Playbook with its compiled regex patterns.
type compiledPlaybook struct {
	pb       Playbook
	patterns []*regexp.Regexp
}

// NewStore creates an empty Store.
func NewStore() *Store {
	return &Store{
		playbooks: []Playbook{},
		compiled:  []compiledPlaybook{},
	}
}

// LoadFromJSON parses a PlaybookFile JSON and compiles regex patterns.
// Replaces all existing playbooks in the store.
func (s *Store) LoadFromJSON(data []byte) error {
	var pf PlaybookFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset store
	s.playbooks = []Playbook{}
	s.compiled = []compiledPlaybook{}

	// Compile patterns for each playbook
	for _, pb := range pf.Playbooks {
		var patterns []*regexp.Regexp

		for _, pattern := range pb.Match.ErrorPatterns {
			// Compile with (?i) prefix for case-insensitive matching
			regexStr := "(?i)" + pattern
			compiled, err := regexp.Compile(regexStr)
			if err != nil {
				// If compilation fails, store nil so fallback to substring match can work
				patterns = append(patterns, nil)
				continue
			}
			patterns = append(patterns, compiled)
		}

		s.playbooks = append(s.playbooks, pb)
		s.compiled = append(s.compiled, compiledPlaybook{
			pb:       pb,
			patterns: patterns,
		})
	}

	return nil
}

// FindMatch finds the first playbook matching the given phase and error log.
// Returns nil if no match is found.
// Thread-safe with RLock.
func (s *Store) FindMatch(phase, errorLog string) *Playbook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, cpb := range s.compiled {
		// Check phase match: exact match or wildcard "*"
		pbPhase := cpb.pb.Match.Phase
		if pbPhase != "*" && pbPhase != phase {
			continue
		}

		// Check error patterns
		if len(cpb.patterns) == 0 {
			// No patterns to match
			continue
		}

		matched := false
		for _, pattern := range cpb.patterns {
			if pattern == nil {
				// Fallback to substring match if regex compilation failed
				// Check if any of the original patterns exist as substring
				for _, origPattern := range cpb.pb.Match.ErrorPatterns {
					if containsIgnoreCase(errorLog, origPattern) {
						matched = true
						break
					}
				}
			} else if pattern.MatchString(errorLog) {
				matched = true
				break
			}

			if matched {
				break
			}
		}

		if matched {
			return &cpb.pb
		}
	}

	return nil
}

// Count returns the number of playbooks in the store.
// Thread-safe with RLock.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.playbooks)
}

// containsIgnoreCase checks if haystack contains needle (case-insensitive).
// Used as fallback when regex compilation fails.
func containsIgnoreCase(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
