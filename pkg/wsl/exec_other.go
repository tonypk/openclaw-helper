//go:build !windows

package wsl

import "os/exec"

func hideWindow(_ *exec.Cmd) {
	// no-op on non-Windows
}
