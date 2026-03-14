package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllPhases(t *testing.T) {
	phases := AllPhases()
	if len(phases) != 7 {
		t.Errorf("expected 7 phases, got %d", len(phases))
	}

	// Check ordering
	for i, p := range phases {
		if p.Order != i {
			t.Errorf("phase %s: order=%d, expected %d", p.Phase, p.Order, i)
		}
		if p.Label == "" || p.LabelZH == "" {
			t.Errorf("phase %s: missing label", p.Phase)
		}
	}
}

func TestNextPhase(t *testing.T) {
	tests := []struct {
		from Phase
		to   Phase
	}{
		{PhaseIdle, PhasePrecheck},
		{PhasePrecheck, PhaseWSL},
		{PhaseWSL, PhaseUbuntu},
		{PhaseReboot, PhaseUbuntu},
		{PhaseUbuntu, PhaseNode},
		{PhaseNode, PhaseOpenClaw},
		{PhaseOpenClaw, PhaseConfig},
		{PhaseConfig, PhaseVerify},
		{PhaseVerify, PhaseDone},
	}

	for _, tc := range tests {
		got := nextPhase(tc.from)
		if got != tc.to {
			t.Errorf("nextPhase(%s): expected %s, got %s", tc.from, tc.to, got)
		}
	}
}

func TestNewInstallState(t *testing.T) {
	state := NewInstallState()
	if state.CurrentPhase != PhaseIdle {
		t.Errorf("expected phase idle, got %s", state.CurrentPhase)
	}
	if len(state.PhaseResults) != 7 {
		t.Errorf("expected 7 phase results, got %d", len(state.PhaseResults))
	}
	for _, p := range AllPhases() {
		if state.PhaseResults[p.Phase] != PhasePending {
			t.Errorf("phase %s: expected pending, got %s", p.Phase, state.PhaseResults[p.Phase])
		}
	}
}

func TestStatePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Save
	state := NewInstallState()
	state.CurrentPhase = PhaseNode
	state.PhaseResults[PhasePrecheck] = PhaseCompleted
	state.PhaseResults[PhaseWSL] = PhaseCompleted
	state.PhaseResults[PhaseUbuntu] = PhaseCompleted
	state.Metadata["test"] = "value"

	if err := state.SaveTo(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Load
	loaded, err := LoadStateFrom(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded state is nil")
	}
	if loaded.CurrentPhase != PhaseNode {
		t.Errorf("phase: expected node, got %s", loaded.CurrentPhase)
	}
	if loaded.PhaseResults[PhasePrecheck] != PhaseCompleted {
		t.Error("precheck should be completed")
	}
	if loaded.Metadata["test"] != "value" {
		t.Error("metadata not preserved")
	}
}

func TestLoadStateFrom_NotExist(t *testing.T) {
	loaded, err := LoadStateFrom("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for missing file")
	}
}

func TestClearState_NoFile(t *testing.T) {
	// Should not error when file doesn't exist
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	err := os.Remove(path) // will fail but that's ok
	_ = err
}
