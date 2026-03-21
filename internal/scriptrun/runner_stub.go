//go:build !windows

package scriptrun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// executeScript runs a script on non-Windows platforms (for development/testing).
// Only supports wsl_bash (runs as local bash) since there's no WSL or PowerShell.
func executeScript(ctx context.Context, runtime Runtime, _ string, content string) (io.Reader, int, error) {
	switch runtime {
	case RuntimeWSLBash:
		return executeLocalBash(ctx, content)
	case RuntimePowerShell:
		// PowerShell not available on non-Windows; return stub output
		return bytes.NewReader([]byte("##OCH:PROGRESS:100:Skipped (non-Windows)\n")), 0, nil
	default:
		return nil, 1, fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

func executeLocalBash(ctx context.Context, content string) (io.Reader, int, error) {
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

	cmd := exec.CommandContext(ctx, "bash", tmpPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, 1, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, 1, fmt.Errorf("start bash: %w", err)
	}

	// IMPORTANT: Read ALL output BEFORE calling cmd.Wait() to avoid deadlock.
	// Previously used io.Pipe() which is unbuffered — if cmd.Wait() is called
	// before the pipe reader is drained, the process blocks on stdout write.
	output, _ := io.ReadAll(stdout)

	exitCode := 0
	waitErr := cmd.Wait()
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return bytes.NewReader(output), exitCode, waitErr
}
