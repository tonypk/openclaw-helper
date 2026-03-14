//go:build windows

package installer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/tonypk/openclaw-helper/pkg/wsl"
)

// WSLInstaller installs WSL2 and Ubuntu on Windows.
type WSLInstaller struct{}

func (w *WSLInstaller) Phase() Phase { return PhaseWSL }

func (w *WSLInstaller) Verify() bool {
	ver, err := wsl.GetWSLVersion()
	if err != nil {
		return false
	}
	return ver >= 2 && wsl.IsInstalled("Ubuntu")
}

func (w *WSLInstaller) Execute(ctx context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseWSL, Status: PhaseRunning,
		Message: "Checking WSL2 status...", Progress: 10,
	})

	// If already have WSL2 + Ubuntu, skip
	if w.Verify() {
		return false, nil
	}

	ver, _ := wsl.GetWSLVersion()
	if ver < 2 {
		// Install WSL2
		progress(ProgressEvent{
			Phase: PhaseWSL, Status: PhaseRunning,
			Message: "Installing WSL2 (this may take a few minutes)...", Progress: 20,
		})

		if err := w.installWSL2(ctx); err != nil {
			// Check if reboot is needed
			if strings.Contains(err.Error(), "restart") || strings.Contains(err.Error(), "reboot") {
				progress(ProgressEvent{
					Phase: PhaseWSL, Status: PhaseRunning,
					Message: "WSL2 installed — reboot required to continue", Progress: 50,
				})
				return true, nil
			}
			return false, fmt.Errorf("WSL2 installation failed: %w", err)
		}
	}

	// Install Ubuntu if not present
	if !wsl.IsInstalled("Ubuntu") {
		progress(ProgressEvent{
			Phase: PhaseWSL, Status: PhaseRunning,
			Message: "Installing Ubuntu...", Progress: 60,
		})
		if err := w.installUbuntu(ctx); err != nil {
			if strings.Contains(err.Error(), "restart") || strings.Contains(err.Error(), "reboot") {
				return true, nil
			}
			return false, fmt.Errorf("Ubuntu installation failed: %w", err)
		}
	}

	progress(ProgressEvent{
		Phase: PhaseWSL, Status: PhaseRunning,
		Message: "WSL2 + Ubuntu ready", Progress: 100,
	})
	return false, nil
}

func (w *WSLInstaller) installWSL2(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "wsl.exe", "--install", "--no-launch")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)
		// wsl --install may exit non-zero but request reboot
		if strings.Contains(output, "restart") || strings.Contains(output, "reboot") {
			return fmt.Errorf("reboot required: %s", output)
		}
		return fmt.Errorf("%s: %w", output, err)
	}
	return nil
}

func (w *WSLInstaller) installUbuntu(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "wsl.exe", "--install", "-d", "Ubuntu", "--no-launch")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)
		if strings.Contains(output, "restart") || strings.Contains(output, "reboot") {
			return fmt.Errorf("reboot required: %s", output)
		}
		return fmt.Errorf("%s: %w", output, err)
	}
	return nil
}

// UbuntuConfigurer configures a fresh Ubuntu WSL installation.
type UbuntuConfigurer struct{}

func (u *UbuntuConfigurer) Phase() Phase { return PhaseUbuntu }

func (u *UbuntuConfigurer) Verify() bool {
	stdout, _, err := wsl.RunWithTimeout("Ubuntu", "cat /etc/os-release | grep -c Ubuntu", 10*time.Second)
	return err == nil && strings.TrimSpace(stdout) != "0"
}

func (u *UbuntuConfigurer) Execute(ctx context.Context, progress func(ProgressEvent)) (bool, error) {
	progress(ProgressEvent{
		Phase: PhaseUbuntu, Status: PhaseRunning,
		Message: "Updating Ubuntu packages...", Progress: 20,
	})

	// Update package lists
	_, _, err := wsl.RunWithTimeout("Ubuntu", "sudo apt-get update -y", 5*time.Minute)
	if err != nil {
		return false, fmt.Errorf("apt-get update failed: %w", err)
	}

	progress(ProgressEvent{
		Phase: PhaseUbuntu, Status: PhaseRunning,
		Message: "Installing build dependencies...", Progress: 50,
	})

	// Install essential build tools
	_, _, err = wsl.RunWithTimeout("Ubuntu",
		"sudo apt-get install -y curl git build-essential",
		5*time.Minute)
	if err != nil {
		return false, fmt.Errorf("dependency install failed: %w", err)
	}

	progress(ProgressEvent{
		Phase: PhaseUbuntu, Status: PhaseRunning,
		Message: "Ubuntu configured", Progress: 100,
	})
	return false, nil
}
