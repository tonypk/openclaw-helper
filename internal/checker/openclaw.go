//go:build windows

package checker

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/tonypk/openclaw-helper/internal/types"
	"github.com/tonypk/openclaw-helper/pkg/wsl"
)

const openClawPort = "18789"

// OpenClawChecker checks OpenClaw installation and gateway status.
type OpenClawChecker struct{}

func (c *OpenClawChecker) Name() string { return "openclaw" }

func (c *OpenClawChecker) Check() types.CheckResult {
	result := types.CheckResult{Name: "openclaw"}

	// Check if openclaw CLI is installed
	stdout, _, err := wsl.Run("", "which openclaw 2>/dev/null || command -v openclaw 2>/dev/null")
	if err != nil || strings.TrimSpace(stdout) == "" {
		result.Status = types.StatusFail
		result.Message = "OpenClaw not installed"
		result.Detail = "OpenClaw will be installed during setup"
		return result
	}

	// Check if gateway is reachable
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+openClawPort, 2*time.Second)
	if err != nil {
		result.Status = types.StatusWarn
		result.Message = "OpenClaw installed but gateway not running"
		result.Detail = fmt.Sprintf("Gateway expected on port %s", openClawPort)
		return result
	}
	conn.Close()

	result.Status = types.StatusPass
	result.Message = fmt.Sprintf("OpenClaw running (gateway on port %s)", openClawPort)
	return result
}
