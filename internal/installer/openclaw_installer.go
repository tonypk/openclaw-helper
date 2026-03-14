//go:build windows

package installer

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/tonypk/openclaw-helper/pkg/wsl"
)

// OpenClawInstaller installs OpenClaw inside WSL.
type OpenClawInstaller struct{}

func (o *OpenClawInstaller) Phase() Phase { return PhaseOpenClaw }

func (o *OpenClawInstaller) Verify() bool {
	stdout, _, err := wsl.RunWithTimeout("Ubuntu", "which openclaw 2>/dev/null || command -v openclaw 2>/dev/null", 10*time.Second)
	return err == nil && strings.TrimSpace(stdout) != ""
}

func (o *OpenClawInstaller) Execute(ctx context.Context, progress func(ProgressEvent)) (bool, error) {
	if o.Verify() {
		return false, nil
	}

	progress(ProgressEvent{
		Phase: PhaseOpenClaw, Status: PhaseRunning,
		Message: "Installing OpenClaw via npm...", Progress: 10,
	})

	// Load nvm and install OpenClaw globally
	installCmd := `export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && npm install -g openclaw`
	_, _, err := wsl.RunWithTimeout("Ubuntu", installCmd, 10*time.Minute)
	if err != nil {
		return false, fmt.Errorf("OpenClaw npm install failed: %w", err)
	}

	progress(ProgressEvent{
		Phase: PhaseOpenClaw, Status: PhaseRunning,
		Message: "Verifying OpenClaw installation...", Progress: 70,
	})

	if !o.Verify() {
		return false, fmt.Errorf("OpenClaw installation could not be verified")
	}

	progress(ProgressEvent{
		Phase: PhaseOpenClaw, Status: PhaseCompleted,
		Message: "OpenClaw installed successfully", Progress: 100,
	})
	return false, nil
}

// ConfigPhase runs initial configuration (API key, etc.).
// For Phase 2, this is a placeholder — Phase 3 will add full config wizard.
type ConfigPhase struct{}

func (c *ConfigPhase) Phase() Phase { return PhaseConfig }

func (c *ConfigPhase) Verify() bool { return false }

func (c *ConfigPhase) Execute(_ context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseConfig, Status: PhaseCompleted,
		Message: "Configuration will be set up via the UI", Progress: 100,
	})
	return false, nil
}

// VerifyPhase checks that everything is working end-to-end.
type VerifyPhase struct{}

func (v *VerifyPhase) Phase() Phase { return PhaseVerify }

func (v *VerifyPhase) Verify() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:18789", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (v *VerifyPhase) Execute(ctx context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseVerify, Status: PhaseRunning,
		Message: "Starting OpenClaw gateway...", Progress: 20,
	})

	// Start OpenClaw in background
	startCmd := `export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nohup openclaw start > /tmp/openclaw.log 2>&1 &`
	wsl.RunWithTimeout("Ubuntu", startCmd, 30*time.Second)

	progress(ProgressEvent{
		Phase: PhaseVerify, Status: PhaseRunning,
		Message: "Waiting for gateway to start...", Progress: 50,
	})

	// Wait for gateway with retries
	for i := 0; i < 15; i++ {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		conn, err := net.DialTimeout("tcp", "127.0.0.1:18789", 2*time.Second)
		if err == nil {
			conn.Close()
			progress(ProgressEvent{
				Phase: PhaseVerify, Status: PhaseCompleted,
				Message: "OpenClaw gateway is running!", Progress: 100,
			})
			return false, nil
		}
		time.Sleep(2 * time.Second)
	}

	return false, fmt.Errorf("gateway did not start within 30 seconds — check /tmp/openclaw.log in WSL")
}
