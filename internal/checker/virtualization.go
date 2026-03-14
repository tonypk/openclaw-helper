//go:build windows

package checker

import (
	"strings"

	"github.com/tonypk/openclaw-helper/internal/types"
)

// VirtualizationChecker checks if hardware virtualization is enabled.
type VirtualizationChecker struct{}

func (c *VirtualizationChecker) Name() string { return "virtualization" }

func (c *VirtualizationChecker) Check() types.CheckResult {
	result := types.CheckResult{Name: "virtualization"}

	// Try systeminfo first (faster, no elevation needed)
	out, err := runPowerShell(`(systeminfo) -match "Hyper-V"`)
	if err == nil && strings.Contains(out, "Yes") {
		result.Status = types.StatusPass
		result.Message = "Hyper-V supported and enabled"
		return result
	}

	// Fallback: check via WMI
	out, err = runPowerShell(`(Get-CimInstance Win32_Processor).VirtualizationFirmwareEnabled`)
	if err != nil {
		result.Status = types.StatusWarn
		result.Message = "Unable to detect virtualization status"
		result.Detail = err.Error()
		return result
	}

	if strings.TrimSpace(out) == "True" {
		result.Status = types.StatusPass
		result.Message = "Hardware virtualization enabled"
	} else {
		result.Status = types.StatusFail
		result.Message = "Hardware virtualization not enabled — please enable in BIOS/UEFI"
		result.Detail = "Most computers support this feature but it may be disabled by default. You need to restart and enter BIOS settings to enable it."
	}
	return result
}
