// Package checker provides system environment detection for OpenClaw installation.
package checker

import (
	"github.com/tonypk/openclaw-helper/internal/types"
)

// Checker defines the interface for a single system check.
type Checker interface {
	Name() string
	Check() types.CheckResult
}

// SystemChecker runs all system checks and produces a report.
type SystemChecker struct {
	checkers []Checker
}

// New creates a SystemChecker with all default checkers.
func New() *SystemChecker {
	return &SystemChecker{
		checkers: []Checker{
			&OSChecker{},
			&MemoryChecker{},
			&DiskChecker{},
			&VirtualizationChecker{},
			&WSLChecker{},
			&NodeChecker{},
			&OpenClawChecker{},
		},
	}
}

// RunAll executes all checkers and returns a SystemReport.
func (sc *SystemChecker) RunAll() types.SystemReport {
	report := types.SystemReport{OverallReady: true}

	for _, c := range sc.checkers {
		result := c.Check()
		switch c.Name() {
		case "os":
			report.OS = result
		case "memory":
			report.Memory = result
		case "disk":
			report.Disk = result
		case "virtualization":
			report.Virtualization = result
		case "wsl":
			report.WSL = result
		case "node":
			report.Node = result
		case "openclaw":
			report.OpenClaw = result
		}
		if result.Status == types.StatusFail {
			report.OverallReady = false
		}
	}

	return report
}

// RunSingle runs a single checker by name.
func (sc *SystemChecker) RunSingle(name string) (types.CheckResult, bool) {
	for _, c := range sc.checkers {
		if c.Name() == name {
			return c.Check(), true
		}
	}
	return types.CheckResult{
		Name:    name,
		Status:  types.StatusFail,
		Message: "unknown check: " + name,
	}, false
}
