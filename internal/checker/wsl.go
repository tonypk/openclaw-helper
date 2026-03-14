//go:build windows

package checker

import (
	"strings"

	"github.com/tonypk/openclaw-helper/internal/types"
	"github.com/tonypk/openclaw-helper/pkg/wsl"
)

// WSLChecker checks WSL2 installation status.
type WSLChecker struct{}

func (c *WSLChecker) Name() string { return "wsl" }

func (c *WSLChecker) Check() types.CheckResult {
	result := types.CheckResult{Name: "wsl"}

	ver, err := wsl.GetWSLVersion()
	if err != nil {
		result.Status = types.StatusFail
		result.Message = "WSL not installed"
		result.Detail = err.Error()
		return result
	}

	if ver < 2 {
		result.Status = types.StatusFail
		result.Message = "WSL1 detected — WSL2 required"
		result.Detail = "Run 'wsl --set-default-version 2' to upgrade"
		return result
	}

	distros, err := wsl.ListDistros()
	if err != nil || len(distros) == 0 {
		result.Status = types.StatusWarn
		result.Message = "WSL2 installed but no Linux distribution found"
		result.Detail = "Ubuntu will be installed during setup"
		return result
	}

	names := make([]string, len(distros))
	for i, d := range distros {
		names[i] = d.Name
	}
	result.Status = types.StatusPass
	result.Message = "WSL2 with " + strings.Join(names, ", ")
	return result
}
