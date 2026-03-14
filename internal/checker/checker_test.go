package checker

import (
	"testing"

	"github.com/tonypk/openclaw-helper/internal/types"
)

func TestNew(t *testing.T) {
	sc := New()
	if sc == nil {
		t.Fatal("New() returned nil")
	}
	if len(sc.checkers) != 7 {
		t.Errorf("expected 7 checkers, got %d", len(sc.checkers))
	}
}

func TestRunAll(t *testing.T) {
	sc := New()
	report := sc.RunAll()

	// On non-Windows, all checks should be skipped
	checks := []struct {
		name   string
		result types.CheckResult
	}{
		{"os", report.OS},
		{"memory", report.Memory},
		{"disk", report.Disk},
		{"virtualization", report.Virtualization},
		{"wsl", report.WSL},
		{"node", report.Node},
		{"openclaw", report.OpenClaw},
	}

	for _, tc := range checks {
		if tc.result.Name != tc.name {
			t.Errorf("check %q: expected Name=%q, got %q", tc.name, tc.name, tc.result.Name)
		}
		// On macOS/Linux, all should be "skipped"
		if tc.result.Status != types.StatusSkipped {
			t.Logf("check %q: status=%s (expected skipped on non-Windows)", tc.name, tc.result.Status)
		}
	}

	// With all skipped (not failed), OverallReady should be true
	if !report.OverallReady {
		t.Error("expected OverallReady=true when no failures")
	}
}

func TestRunSingle_Known(t *testing.T) {
	sc := New()
	result, found := sc.RunSingle("os")
	if !found {
		t.Error("expected to find 'os' checker")
	}
	if result.Name != "os" {
		t.Errorf("expected Name=os, got %q", result.Name)
	}
}

func TestRunSingle_Unknown(t *testing.T) {
	sc := New()
	result, found := sc.RunSingle("nonexistent")
	if found {
		t.Error("expected not to find 'nonexistent' checker")
	}
	if result.Status != types.StatusFail {
		t.Errorf("expected StatusFail for unknown checker, got %s", result.Status)
	}
}

func TestCheckerNames(t *testing.T) {
	sc := New()
	expectedNames := []string{"os", "memory", "disk", "virtualization", "wsl", "node", "openclaw"}

	for i, c := range sc.checkers {
		if c.Name() != expectedNames[i] {
			t.Errorf("checker[%d]: expected Name=%q, got %q", i, expectedNames[i], c.Name())
		}
	}
}
