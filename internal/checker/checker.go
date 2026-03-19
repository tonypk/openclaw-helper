// Package checker provides system environment detection for OpenClaw installation.
package checker

import (
	"log"
	"time"

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

// checkTimeout is the maximum time a single system check may take.
const checkTimeout = 15 * time.Second

// RunAll executes all checkers and returns a SystemReport.
// Each checker is run with a timeout to prevent blocking.
func (sc *SystemChecker) RunAll() types.SystemReport {
	report := types.SystemReport{OverallReady: true}

	for _, c := range sc.checkers {
		result := runWithTimeout(c, checkTimeout)
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

// runWithTimeout runs a single checker with a timeout.
// If the check exceeds the timeout, a warning result is returned.
func runWithTimeout(c Checker, timeout time.Duration) types.CheckResult {
	ch := make(chan types.CheckResult, 1)
	go func() {
		ch <- c.Check()
	}()

	select {
	case result := <-ch:
		return result
	case <-time.After(timeout):
		log.Printf("[checker] %s timed out after %v", c.Name(), timeout)
		return types.CheckResult{
			Name:    c.Name(),
			Status:  types.StatusWarn,
			Message: "Check timed out — skipping",
			Detail:  "This check took too long and was skipped. This is not a critical issue.",
		}
	}
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
