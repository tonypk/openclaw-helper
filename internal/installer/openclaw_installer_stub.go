//go:build !windows

package installer

import (
	"context"
	"time"
)

// OpenClawInstaller is a stub on non-Windows platforms.
type OpenClawInstaller struct{}

func (o *OpenClawInstaller) Phase() Phase { return PhaseOpenClaw }

func (o *OpenClawInstaller) Verify() bool { return true }

func (o *OpenClawInstaller) Execute(_ context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseOpenClaw, Status: PhaseCompleted,
		Message: "OpenClaw check skipped (non-Windows)", Progress: 100,
		Timestamp: time.Now(),
	})
	return false, nil
}

// ConfigPhase is a stub on non-Windows platforms.
type ConfigPhase struct{}

func (c *ConfigPhase) Phase() Phase { return PhaseConfig }

func (c *ConfigPhase) Verify() bool { return true }

func (c *ConfigPhase) Execute(_ context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseConfig, Status: PhaseCompleted,
		Message: "Config skipped (non-Windows)", Progress: 100,
		Timestamp: time.Now(),
	})
	return false, nil
}

// VerifyPhase is a stub on non-Windows platforms.
type VerifyPhase struct{}

func (v *VerifyPhase) Phase() Phase { return PhaseVerify }

func (v *VerifyPhase) Verify() bool { return true }

func (v *VerifyPhase) Execute(_ context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseVerify, Status: PhaseCompleted,
		Message: "Verify skipped (non-Windows)", Progress: 100,
		Timestamp: time.Now(),
	})
	return false, nil
}
