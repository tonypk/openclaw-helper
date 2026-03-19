package installer

import (
	"context"
	"fmt"
	"time"

	"github.com/tonypk/openclaw-helper/internal/checker"
	"github.com/tonypk/openclaw-helper/internal/types"
)

// PrecheckExecutor runs system checks before installation.
type PrecheckExecutor struct {
	checker *checker.SystemChecker
}

// NewPrecheckExecutor creates a precheck executor with the given system checker.
func NewPrecheckExecutor(sc *checker.SystemChecker) *PrecheckExecutor {
	return &PrecheckExecutor{checker: sc}
}

func (p *PrecheckExecutor) Phase() Phase { return PhasePrecheck }

func (p *PrecheckExecutor) Verify() bool { return false }

func (p *PrecheckExecutor) Execute(ctx context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhasePrecheck, Status: PhaseRunning,
		Message:  "Running system checks...", Progress: 10,
		Timestamp: time.Now(),
	})

	// Run checks with overall timeout to prevent indefinite blocking.
	type reportResult struct {
		report types.SystemReport
	}
	ch := make(chan reportResult, 1)
	go func() {
		ch <- reportResult{report: p.checker.RunAll()}
	}()

	var report types.SystemReport
	select {
	case res := <-ch:
		report = res.report
	case <-ctx.Done():
		return false, fmt.Errorf("system checks cancelled")
	case <-time.After(90 * time.Second):
		return false, fmt.Errorf("system checks timed out — your system may be slow to respond. Please retry")
	}

	progress(ProgressEvent{
		Phase: PhasePrecheck, Status: PhaseRunning,
		Message: "System checks complete", Progress: 90,
		Timestamp: time.Now(),
	})

	// On non-Windows, skipped checks are fine
	// On Windows, check critical items: OS, memory, disk
	criticalChecks := []struct {
		name   string
		result types.CheckResult
	}{
		{"OS", report.OS},
		{"Memory", report.Memory},
		{"Disk", report.Disk},
	}

	for _, c := range criticalChecks {
		if c.result.Status == types.StatusFail {
			return false, fmt.Errorf("%s check failed: %s", c.name, c.result.Message)
		}
	}

	// Virtualization warning is non-fatal at precheck (WSL install will catch it)
	if report.Virtualization.Status == types.StatusFail {
		progress(ProgressEvent{
			Phase: PhasePrecheck, Status: PhaseRunning,
			Message:  "Warning: " + report.Virtualization.Message,
			Detail:   report.Virtualization.Detail,
			Timestamp: time.Now(),
		})
	}

	// Warn if WSL needs installing but we don't have admin privileges
	if report.WSL.Status != types.StatusPass && !IsAdmin() {
		progress(ProgressEvent{
			Phase: PhasePrecheck, Status: PhaseRunning,
			Message:   "WSL2 installation requires administrator privileges — a UAC prompt will appear",
			Timestamp: time.Now(),
		})
	}

	progress(ProgressEvent{
		Phase: PhasePrecheck, Status: PhaseCompleted,
		Message: "System checks passed", Progress: 100,
		Timestamp: time.Now(),
	})
	return false, nil
}
