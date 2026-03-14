//go:build windows

package diagnosis

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/tonypk/openclaw-helper/pkg/wsl"
)

func wslExec(ctx context.Context, _ string, command string, timeout time.Duration) (string, error) {
	tCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(tCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", string(out), err)
	}
	return string(out), nil
}

func wslRunInDistro(_ context.Context, distro, command string, timeout time.Duration) (string, error) {
	stdout, _, err := wsl.RunWithTimeout(distro, command, timeout)
	return stdout, err
}
