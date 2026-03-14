// Package wsl provides a cross-platform wrapper for WSL2 operations.
package wsl

import (
	"context"
	"os/exec"
	"time"
)

// Run executes a command inside WSL. If distro is empty, uses the default distribution.
func Run(distro, command string) (stdout, stderr string, err error) {
	return RunWithTimeout(distro, command, 30*time.Second)
}

// RunWithTimeout executes a command inside WSL with a timeout.
func RunWithTimeout(distro, command string, timeout time.Duration) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := buildArgs(distro, command)
	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	hideWindow(cmd)

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", "", context.DeadlineExceeded
	}
	return string(out), "", err
}

func buildArgs(distro, command string) []string {
	if distro != "" {
		return []string{"-d", distro, "--", "bash", "-lc", command}
	}
	return []string{"--", "bash", "-lc", command}
}
