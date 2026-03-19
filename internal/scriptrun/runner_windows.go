//go:build windows

package scriptrun

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// executeScript runs a script using the appropriate runtime on Windows.
// Returns an io.Reader for stdout, exit code, and error.
func executeScript(ctx context.Context, runtime Runtime, distro string, content string) (io.Reader, int, error) {
	switch runtime {
	case RuntimeWSLBash:
		return executeWSLBash(ctx, distro, content)
	case RuntimePowerShell:
		return executePowerShell(ctx, content)
	default:
		return nil, 1, fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

// executeWSLBash runs a bash script inside WSL.
func executeWSLBash(ctx context.Context, distro string, content string) (io.Reader, int, error) {
	// Write script to temp file, then execute inside WSL
	tmpFile, err := os.CreateTemp("", "och-script-*.sh")
	if err != nil {
		return nil, 1, fmt.Errorf("create temp script: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return nil, 1, fmt.Errorf("write temp script: %w", err)
	}
	tmpFile.Close()

	// Convert Windows path to WSL path
	wslPath := windowsToWSLPath(tmpPath)

	args := []string{}
	if distro != "" {
		args = append(args, "-d", distro)
	}
	args = append(args, "--", "bash", "-l", wslPath)

	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 1, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		return nil, 1, fmt.Errorf("start wsl: %w", err)
	}

	// Read all output before waiting for exit
	pr, pw := io.Pipe()
	go func() {
		io.Copy(pw, stdout)
		pw.Close()
	}()

	exitCode := 0
	waitErr := cmd.Wait()
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return pr, exitCode, waitErr
}

// executePowerShell runs a PowerShell script.
func executePowerShell(ctx context.Context, content string) (io.Reader, int, error) {
	tmpFile, err := os.CreateTemp("", "och-script-*.ps1")
	if err != nil {
		return nil, 1, fmt.Errorf("create temp script: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return nil, 1, fmt.Errorf("write temp script: %w", err)
	}
	tmpFile.Close()

	cmd := exec.CommandContext(ctx, "powershell.exe",
		"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmpPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 1, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, 1, fmt.Errorf("start powershell: %w", err)
	}

	pr, pw := io.Pipe()
	go func() {
		io.Copy(pw, stdout)
		pw.Close()
	}()

	exitCode := 0
	waitErr := cmd.Wait()
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return pr, exitCode, waitErr
}

// windowsToWSLPath converts a Windows path like C:\Users\foo\bar to /mnt/c/Users/foo/bar.
func windowsToWSLPath(winPath string) string {
	abs, err := filepath.Abs(winPath)
	if err != nil {
		return winPath
	}
	// C:\Users\foo -> /mnt/c/Users/foo
	if len(abs) >= 2 && abs[1] == ':' {
		drive := string(abs[0])
		rest := filepath.ToSlash(abs[2:])
		return fmt.Sprintf("/mnt/%s%s", string([]byte{drive[0] | 0x20}), rest)
	}
	return filepath.ToSlash(abs)
}
