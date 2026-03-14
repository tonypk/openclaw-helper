//go:build windows

package checker

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tonypk/openclaw-helper/internal/types"
	"github.com/tonypk/openclaw-helper/pkg/wsl"
)

const requiredNodeMajor = 22

// NodeChecker checks Node.js installation inside WSL.
type NodeChecker struct{}

func (c *NodeChecker) Name() string { return "node" }

func (c *NodeChecker) Check() types.CheckResult {
	result := types.CheckResult{Name: "node"}

	stdout, _, err := wsl.Run("", "node -v")
	if err != nil {
		result.Status = types.StatusFail
		result.Message = "Node.js not installed in WSL"
		result.Detail = "Node.js 22+ will be installed during setup"
		return result
	}

	version := strings.TrimSpace(stdout)
	version = strings.TrimPrefix(version, "v")

	parts := strings.SplitN(version, ".", 3)
	if len(parts) == 0 {
		result.Status = types.StatusFail
		result.Message = "Unable to parse Node.js version"
		return result
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		result.Status = types.StatusFail
		result.Message = "Unable to parse Node.js version: " + version
		return result
	}

	if major >= requiredNodeMajor {
		result.Status = types.StatusPass
		result.Message = fmt.Sprintf("Node.js v%s", version)
	} else {
		result.Status = types.StatusFail
		result.Message = fmt.Sprintf("Node.js v%s — version %d+ required", version, requiredNodeMajor)
	}
	return result
}
