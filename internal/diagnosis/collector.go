package diagnosis

import (
	"runtime"
	"strings"

	"github.com/tonypk/openclaw-helper/internal/checker"
	"github.com/tonypk/openclaw-helper/internal/types"
)

// Collect gathers system state into a DiagContext by running checkers.
func Collect(sc *checker.SystemChecker) *DiagContext {
	report := sc.RunAll()
	ctx := &DiagContext{}

	// OS
	if report.OS.Status == types.StatusPass {
		ctx.OSBuild = extractBuild(report.OS.Message)
	}

	// Memory
	ctx.MemoryGB = extractGB(report.Memory.Message)

	// Disk
	ctx.DiskFreeGB = extractGB(report.Disk.Message)

	// Virtualization
	ctx.VirtEnabled = report.Virtualization.Status == types.StatusPass ||
		report.Virtualization.Status == types.StatusSkipped

	// WSL
	ctx.WSLInstalled = report.WSL.Status == types.StatusPass ||
		report.WSL.Status == types.StatusWarn ||
		report.WSL.Status == types.StatusSkipped
	if report.WSL.Status == types.StatusPass || report.WSL.Status == types.StatusSkipped {
		ctx.WSLVersion = 2
	}
	ctx.WSLDistros = extractDistros(report.WSL.Message)

	// Node
	ctx.NodeInstalled = report.Node.Status == types.StatusPass ||
		report.Node.Status == types.StatusSkipped

	// OpenClaw
	ctx.OpenClawInstalled = report.OpenClaw.Status == types.StatusPass ||
		report.OpenClaw.Status == types.StatusSkipped
	ctx.GatewayReachable = report.OpenClaw.Status == types.StatusPass ||
		report.OpenClaw.Status == types.StatusSkipped

	return ctx
}

// CollectWithError adds error log context.
func CollectWithError(sc *checker.SystemChecker, errorLog string) *DiagContext {
	ctx := Collect(sc)
	ctx.ErrorLog = errorLog
	return ctx
}

func extractBuild(msg string) int {
	if runtime.GOOS != "windows" {
		return 99999 // Stub: assume modern
	}
	// Parse "Windows 11 ... (Build 22631)"
	var build int
	for _, word := range strings.Fields(msg) {
		n := 0
		valid := true
		for _, c := range word {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				valid = false
				break
			}
		}
		if valid && n > 10000 {
			build = n
		}
	}
	return build
}

func extractGB(msg string) float64 {
	// Find first number-like string
	var gb float64
	for _, word := range strings.Fields(msg) {
		n := 0.0
		valid := false
		dotSeen := false
		frac := 0.1
		for _, c := range word {
			if c >= '0' && c <= '9' {
				valid = true
				if dotSeen {
					n += float64(c-'0') * frac
					frac *= 0.1
				} else {
					n = n*10 + float64(c-'0')
				}
			} else if c == '.' && !dotSeen {
				dotSeen = true
			} else {
				break
			}
		}
		if valid && n > 0 {
			gb = n
			break
		}
	}
	return gb
}

func extractDistros(msg string) []string {
	// "WSL2 with Ubuntu, Debian" → ["Ubuntu", "Debian"]
	if idx := strings.Index(msg, "with "); idx >= 0 {
		rest := msg[idx+5:]
		parts := strings.Split(rest, ",")
		var distros []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				distros = append(distros, p)
			}
		}
		return distros
	}
	return nil
}
