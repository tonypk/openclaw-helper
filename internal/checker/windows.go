//go:build windows

package checker

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/tonypk/openclaw-helper/internal/types"
)

// OSChecker checks the Windows version.
type OSChecker struct{}

func (c *OSChecker) Name() string { return "os" }

func (c *OSChecker) Check() types.CheckResult {
	result := types.CheckResult{Name: "os"}

	out, err := runPowerShell(`(Get-CimInstance Win32_OperatingSystem).Caption + '|' + (Get-CimInstance Win32_OperatingSystem).BuildNumber`)
	if err != nil {
		result.Status = types.StatusFail
		result.Message = "Failed to detect Windows version"
		result.Detail = err.Error()
		return result
	}

	parts := strings.SplitN(strings.TrimSpace(out), "|", 2)
	caption := parts[0]
	buildStr := ""
	if len(parts) > 1 {
		buildStr = parts[1]
	}

	build, _ := strconv.Atoi(buildStr)
	// WSL2 requires Windows 10 build 19041+
	if build >= 19041 {
		result.Status = types.StatusPass
		result.Message = fmt.Sprintf("%s (Build %d)", caption, build)
	} else {
		result.Status = types.StatusFail
		result.Message = fmt.Sprintf("%s (Build %d) — WSL2 requires build 19041+", caption, build)
	}
	return result
}

// MemoryChecker checks total physical memory.
type MemoryChecker struct{}

func (c *MemoryChecker) Name() string { return "memory" }

func (c *MemoryChecker) Check() types.CheckResult {
	result := types.CheckResult{Name: "memory"}

	var memStatus memoryStatusEx
	memStatus.Length = uint32(unsafe.Sizeof(memStatus))

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

	ret, _, err := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		result.Status = types.StatusFail
		result.Message = "Failed to detect memory"
		result.Detail = err.Error()
		return result
	}

	totalGB := float64(memStatus.TotalPhys) / (1024 * 1024 * 1024)
	if totalGB >= 4.0 {
		result.Status = types.StatusPass
		result.Message = fmt.Sprintf("%.0f GB", totalGB)
	} else {
		result.Status = types.StatusFail
		result.Message = fmt.Sprintf("%.0f GB — minimum 4 GB required", totalGB)
	}
	return result
}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// DiskChecker checks free disk space on the system drive.
type DiskChecker struct{}

func (c *DiskChecker) Name() string { return "disk" }

func (c *DiskChecker) Check() types.CheckResult {
	result := types.CheckResult{Name: "disk"}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytes, totalBytes, totalFreeBytes uint64
	drive, _ := syscall.UTF16PtrFromString(`C:\`)

	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(drive)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if ret == 0 {
		result.Status = types.StatusFail
		result.Message = "Failed to check disk space"
		result.Detail = err.Error()
		return result
	}

	freeGB := float64(freeBytes) / (1024 * 1024 * 1024)
	if freeGB >= 10.0 {
		result.Status = types.StatusPass
		result.Message = fmt.Sprintf("%.1f GB free", freeGB)
	} else {
		result.Status = types.StatusFail
		result.Message = fmt.Sprintf("%.1f GB free — minimum 10 GB required", freeGB)
	}
	return result
}

func runPowerShell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
