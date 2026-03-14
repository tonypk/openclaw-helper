//go:build !windows

package installer

import (
	"context"
	"time"
)

// WSLInstaller is a stub on non-Windows platforms.
type WSLInstaller struct{}

func (w *WSLInstaller) Phase() Phase { return PhaseWSL }

func (w *WSLInstaller) Verify() bool { return true }

func (w *WSLInstaller) Execute(_ context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseWSL, Status: PhaseCompleted,
		Message: "WSL2 check skipped (non-Windows)", Progress: 100,
		Timestamp: time.Now(),
	})
	return false, nil
}

// UbuntuConfigurer is a stub on non-Windows platforms.
type UbuntuConfigurer struct{}

func (u *UbuntuConfigurer) Phase() Phase { return PhaseUbuntu }

func (u *UbuntuConfigurer) Verify() bool { return true }

func (u *UbuntuConfigurer) Execute(_ context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseUbuntu, Status: PhaseCompleted,
		Message: "Ubuntu config skipped (non-Windows)", Progress: 100,
		Timestamp: time.Now(),
	})
	return false, nil
}
