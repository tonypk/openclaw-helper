//go:build windows

package installer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/tonypk/openclaw-helper/pkg/wsl"
	"golang.org/x/sys/windows"
)

const (
	wslInstallTimeout    = 10 * time.Minute
	ubuntuInstallTimeout = 10 * time.Minute
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

		if err := w.installWSL2(ctx, progress); err != nil {
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
		if err := w.installUbuntu(ctx, progress); err != nil {
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

func (w *WSLInstaller) installWSL2(ctx context.Context, progress func(ProgressEvent)) error {
	if IsAdmin() {
		return w.runDirectWithTimeout(ctx, wslInstallTimeout, "wsl.exe", "--install", "--no-launch")
	}
	// Not admin — use ShellExecute with "runas" to trigger UAC prompt
	progress(ProgressEvent{
		Phase: PhaseWSL, Status: PhaseRunning,
		Message: "Requesting administrator permissions for WSL2 installation...", Progress: 25,
	})
	return w.runElevated("wsl.exe", "--install --no-launch")
}

func (w *WSLInstaller) installUbuntu(ctx context.Context, progress func(ProgressEvent)) error {
	if IsAdmin() {
		return w.runDirectWithTimeout(ctx, ubuntuInstallTimeout, "wsl.exe", "--install", "-d", "Ubuntu", "--no-launch")
	}
	progress(ProgressEvent{
		Phase: PhaseWSL, Status: PhaseRunning,
		Message: "Requesting administrator permissions for Ubuntu installation...", Progress: 65,
	})
	return w.runElevated("wsl.exe", "--install -d Ubuntu --no-launch")
}

// runDirectWithTimeout runs a command directly with a timeout (when already admin).
func (w *WSLInstaller) runDirectWithTimeout(ctx context.Context, timeout time.Duration, name string, args ...string) error {
	tCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(tCtx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if tCtx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("timed out after %v — please check your network and retry", timeout)
	}
	if err != nil {
		output := string(out)
		if strings.Contains(output, "restart") || strings.Contains(output, "reboot") {
			return fmt.Errorf("reboot required: %s", output)
		}
		return fmt.Errorf("%s: %w", output, err)
	}
	return nil
}

// runElevated uses ShellExecuteW with "runas" verb to run a command with admin privileges.
// This triggers the Windows UAC dialog for user approval.
func (w *WSLInstaller) runElevated(exe, args string) error {
	verb, _ := windows.UTF16PtrFromString("runas")
	exePath, _ := windows.UTF16PtrFromString(exe)
	params, _ := windows.UTF16PtrFromString(args)

	// SW_SHOWNORMAL = 1
	ret := shellExecuteW(0, verb, exePath, params, nil, 1)
	if ret <= 32 {
		// Error codes: 2=file not found, 3=path not found, 5=access denied (user cancelled UAC)
		if ret == 5 {
			return fmt.Errorf("administrator permission denied — please allow the UAC prompt to install WSL2")
		}
		return fmt.Errorf("failed to start elevated process (error code: %d)", ret)
	}
	return nil
}

var (
	shell32          = windows.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

func shellExecuteW(hwnd uintptr, verb, file, params, dir *uint16, showCmd int32) uintptr {
	ret, _, _ := procShellExecute.Call(
		hwnd,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		uintptr(unsafe.Pointer(dir)),
		uintptr(showCmd),
	)
	return ret
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
