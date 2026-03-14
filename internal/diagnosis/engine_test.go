package diagnosis

import (
	"context"
	"testing"
)

func TestDiagnose_Healthy(t *testing.T) {
	engine := NewEngine()
	ctx := &DiagContext{
		WSLInstalled:      true,
		WSLVersion:        2,
		WSLDistros:        []string{"Ubuntu"},
		NodeInstalled:     true,
		NodeVersion:       "22.0.0",
		NodeMajor:         22,
		OpenClawInstalled: true,
		GatewayReachable:  true,
		VirtEnabled:       true,
		DiskFreeGB:        50,
		MemoryGB:          16,
		OSBuild:           22631,
	}
	report := engine.Diagnose(ctx)
	if !report.Healthy {
		t.Errorf("expected healthy, got issues: %+v", report.Issues)
	}
	if len(report.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(report.Issues))
	}
}

func TestDiagnose_WSLNotInstalled(t *testing.T) {
	engine := NewEngine()
	ctx := &DiagContext{
		WSLInstalled: false,
		VirtEnabled:  true,
		DiskFreeGB:   50,
		MemoryGB:     16,
		OSBuild:      22631,
	}
	report := engine.Diagnose(ctx)
	if report.Healthy {
		t.Error("expected unhealthy")
	}
	found := false
	for _, issue := range report.Issues {
		if issue.ID == "wsl_not_installed" {
			found = true
			if issue.Severity != SeverityCritical {
				t.Errorf("expected critical severity")
			}
			if !issue.AutoRepair {
				t.Error("expected auto-repairable")
			}
		}
	}
	if !found {
		t.Error("expected wsl_not_installed issue")
	}
}

func TestDiagnose_NodeTooOld(t *testing.T) {
	engine := NewEngine()
	ctx := &DiagContext{
		WSLInstalled:      true,
		WSLVersion:        2,
		WSLDistros:        []string{"Ubuntu"},
		NodeInstalled:     true,
		NodeVersion:       "18.17.0",
		NodeMajor:         18,
		OpenClawInstalled: false,
		VirtEnabled:       true,
		DiskFreeGB:        50,
		MemoryGB:          16,
		OSBuild:           22631,
	}
	report := engine.Diagnose(ctx)

	var nodeIssue *Issue
	for _, issue := range report.Issues {
		if issue.ID == "node_too_old" {
			nodeIssue = &issue
			break
		}
	}
	if nodeIssue == nil {
		t.Fatal("expected node_too_old issue")
	}
	if nodeIssue.RepairID != "upgrade_node" {
		t.Errorf("expected repair_id=upgrade_node, got %s", nodeIssue.RepairID)
	}
}

func TestDiagnose_ErrorLogMatching(t *testing.T) {
	tests := []struct {
		name     string
		errorLog string
		issueID  string
	}{
		{"sharp", "Error: Cannot find module 'sharp'", "sharp_build_error"},
		{"path", "bash: openclaw: command not found", "path_error"},
		{"network", "Error: getaddrinfo ENOTFOUND registry.npmjs.org", "network_error"},
		{"apikey", "Error: 401 unauthorized invalid api key", "api_key_error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine()
			ctx := &DiagContext{
				WSLInstalled:      true,
				WSLVersion:        2,
				WSLDistros:        []string{"Ubuntu"},
				NodeInstalled:     true,
				NodeMajor:         22,
				OpenClawInstalled: true,
				GatewayReachable:  true,
				VirtEnabled:       true,
				DiskFreeGB:        50,
				MemoryGB:          16,
				OSBuild:           22631,
				ErrorLog:          tc.errorLog,
			}
			report := engine.Diagnose(ctx)

			found := false
			for _, issue := range report.Issues {
				if issue.ID == tc.issueID {
					found = true
				}
			}
			if !found {
				t.Errorf("expected issue %s for error log %q", tc.issueID, tc.errorLog)
			}
		})
	}
}

func TestDiagnose_DiskLow(t *testing.T) {
	engine := NewEngine()
	ctx := &DiagContext{
		WSLInstalled: true, WSLVersion: 2, WSLDistros: []string{"Ubuntu"},
		NodeInstalled: true, NodeMajor: 22,
		OpenClawInstalled: true, GatewayReachable: true,
		VirtEnabled: true, DiskFreeGB: 5, MemoryGB: 16, OSBuild: 22631,
	}
	report := engine.Diagnose(ctx)
	found := false
	for _, issue := range report.Issues {
		if issue.ID == "disk_low" {
			found = true
		}
	}
	if !found {
		t.Error("expected disk_low issue")
	}
}

func TestPlaybookRegistry(t *testing.T) {
	reg := NewPlaybookRegistry()
	ids := reg.List()
	if len(ids) < 5 {
		t.Errorf("expected at least 5 playbooks, got %d", len(ids))
	}

	_, ok := reg.Get("install_wsl")
	if !ok {
		t.Error("expected install_wsl playbook")
	}

	// Test unknown playbook
	result := reg.Run(context.Background(), "nonexistent")
	if result.Success {
		t.Error("expected failure for unknown playbook")
	}

	// Test guide_virt (non-destructive, always succeeds)
	result = reg.Run(context.Background(), "guide_virt")
	if !result.Success {
		t.Error("expected guide_virt to succeed")
	}
	if result.MsgZH == "" {
		t.Error("expected Chinese message")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s      string
		subs   []string
		expect bool
	}{
		{"Error: sharp not found", []string{"sharp"}, true},
		{"Everything is fine", []string{"sharp", "error"}, false},
		{"DNS resolution failed", []string{"DNS", "network"}, true},
		{"", []string{"test"}, false},
	}

	for _, tc := range tests {
		got := containsAny(tc.s, tc.subs...)
		if got != tc.expect {
			t.Errorf("containsAny(%q, %v) = %v, want %v", tc.s, tc.subs, got, tc.expect)
		}
	}
}
