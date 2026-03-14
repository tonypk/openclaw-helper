//go:build !windows

package checker

import (
	"fmt"
	"runtime"

	"github.com/tonypk/openclaw-helper/internal/types"
)

// OSChecker checks the operating system version.
type OSChecker struct{}

func (c *OSChecker) Name() string { return "os" }

func (c *OSChecker) Check() types.CheckResult {
	return types.CheckResult{
		Name:    "os",
		Status:  types.StatusSkipped,
		Message: fmt.Sprintf("Running on %s/%s (Windows check skipped)", runtime.GOOS, runtime.GOARCH),
	}
}

// MemoryChecker checks available system memory.
type MemoryChecker struct{}

func (c *MemoryChecker) Name() string { return "memory" }

func (c *MemoryChecker) Check() types.CheckResult {
	return types.CheckResult{
		Name:    "memory",
		Status:  types.StatusSkipped,
		Message: "Memory check skipped (non-Windows)",
	}
}

// DiskChecker checks available disk space.
type DiskChecker struct{}

func (c *DiskChecker) Name() string { return "disk" }

func (c *DiskChecker) Check() types.CheckResult {
	return types.CheckResult{
		Name:    "disk",
		Status:  types.StatusSkipped,
		Message: "Disk check skipped (non-Windows)",
	}
}

// VirtualizationChecker checks virtualization support.
type VirtualizationChecker struct{}

func (c *VirtualizationChecker) Name() string { return "virtualization" }

func (c *VirtualizationChecker) Check() types.CheckResult {
	return types.CheckResult{
		Name:    "virtualization",
		Status:  types.StatusSkipped,
		Message: "Virtualization check skipped (non-Windows)",
	}
}

// WSLChecker checks WSL2 installation status.
type WSLChecker struct{}

func (c *WSLChecker) Name() string { return "wsl" }

func (c *WSLChecker) Check() types.CheckResult {
	return types.CheckResult{
		Name:    "wsl",
		Status:  types.StatusSkipped,
		Message: "WSL check skipped (non-Windows)",
	}
}

// NodeChecker checks Node.js installation in WSL.
type NodeChecker struct{}

func (c *NodeChecker) Name() string { return "node" }

func (c *NodeChecker) Check() types.CheckResult {
	return types.CheckResult{
		Name:    "node",
		Status:  types.StatusSkipped,
		Message: "Node.js check skipped (non-Windows)",
	}
}

// OpenClawChecker checks OpenClaw installation status.
type OpenClawChecker struct{}

func (c *OpenClawChecker) Name() string { return "openclaw" }

func (c *OpenClawChecker) Check() types.CheckResult {
	return types.CheckResult{
		Name:    "openclaw",
		Status:  types.StatusSkipped,
		Message: "OpenClaw check skipped (non-Windows)",
	}
}
