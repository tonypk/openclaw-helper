package diagnosis

import "fmt"

// builtinRules returns all diagnostic rules.
func builtinRules() []Rule {
	return []Rule{
		ruleWSLNotInstalled(),
		ruleWSLVersion1(),
		ruleVirtDisabled(),
		ruleNoUbuntu(),
		ruleNodeMissing(),
		ruleNodeTooOld(),
		ruleOpenClawMissing(),
		ruleGatewayDown(),
		ruleDiskLow(),
		ruleMemoryLow(),
		ruleOSTooOld(),
		ruleSharpBuildError(),
		rulePathError(),
		ruleNetworkError(),
		ruleAPIKeyError(),
	}
}

func ruleWSLNotInstalled() Rule {
	return Rule{
		ID:    "wsl_not_installed",
		Match: func(ctx *DiagContext) bool { return !ctx.WSLInstalled },
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "wsl_not_installed", Severity: SeverityCritical,
				Title: "WSL2 not installed", TitleZH: "WSL2 未安装",
				Description: "Windows Subsystem for Linux 2 is required but not installed.",
				DescZH:      "需要安装 Windows Subsystem for Linux 2。",
				RepairID: "install_wsl", AutoRepair: true,
			}
		},
	}
}

func ruleWSLVersion1() Rule {
	return Rule{
		ID: "wsl_version_1",
		Match: func(ctx *DiagContext) bool {
			return ctx.WSLInstalled && ctx.WSLVersion < 2
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "wsl_version_1", Severity: SeverityCritical,
				Title: "WSL1 detected, WSL2 required", TitleZH: "检测到 WSL1，需要 WSL2",
				Description: "Your system has WSL version 1. WSL2 is required for OpenClaw.",
				DescZH:      "系统安装的是 WSL1，OpenClaw 需要 WSL2。",
				RepairID: "upgrade_wsl", AutoRepair: true,
			}
		},
	}
}

func ruleVirtDisabled() Rule {
	return Rule{
		ID:    "virt_disabled",
		Match: func(ctx *DiagContext) bool { return !ctx.VirtEnabled },
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "virt_disabled", Severity: SeverityCritical,
				Title: "Hardware virtualization disabled", TitleZH: "硬件虚拟化未开启",
				Description: "Hardware virtualization (VT-x/AMD-V) must be enabled in BIOS/UEFI settings.",
				DescZH:      "需要在 BIOS/UEFI 设置中开启硬件虚拟化（VT-x/AMD-V）。",
				RepairID: "guide_virt", AutoRepair: false,
			}
		},
	}
}

func ruleNoUbuntu() Rule {
	return Rule{
		ID: "no_ubuntu",
		Match: func(ctx *DiagContext) bool {
			if !ctx.WSLInstalled {
				return false
			}
			for _, d := range ctx.WSLDistros {
				if d == "Ubuntu" {
					return false
				}
			}
			return true
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "no_ubuntu", Severity: SeverityWarning,
				Title: "Ubuntu not found in WSL", TitleZH: "WSL 中未找到 Ubuntu",
				Description: "Ubuntu distribution is recommended for OpenClaw.",
				DescZH:      "推荐安装 Ubuntu 发行版来运行 OpenClaw。",
				RepairID: "install_ubuntu", AutoRepair: true,
			}
		},
	}
}

func ruleNodeMissing() Rule {
	return Rule{
		ID:    "node_missing",
		Match: func(ctx *DiagContext) bool { return !ctx.NodeInstalled },
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "node_missing", Severity: SeverityCritical,
				Title: "Node.js not installed", TitleZH: "Node.js 未安装",
				Description: "Node.js 22+ is required but not found in WSL.",
				DescZH:      "WSL 中未找到 Node.js 22+。",
				RepairID: "install_node", AutoRepair: true,
			}
		},
	}
}

func ruleNodeTooOld() Rule {
	return Rule{
		ID: "node_too_old",
		Match: func(ctx *DiagContext) bool {
			return ctx.NodeInstalled && ctx.NodeMajor > 0 && ctx.NodeMajor < 22
		},
		Diagnose: func(ctx *DiagContext) Issue {
			return Issue{
				ID: "node_too_old", Severity: SeverityCritical,
				Title:   fmt.Sprintf("Node.js %s too old", ctx.NodeVersion),
				TitleZH: fmt.Sprintf("Node.js %s 版本过低", ctx.NodeVersion),
				Description: "Node.js 22 or higher is required.",
				DescZH:      "需要 Node.js 22 或更高版本。",
				RepairID: "upgrade_node", AutoRepair: true,
			}
		},
	}
}

func ruleOpenClawMissing() Rule {
	return Rule{
		ID: "openclaw_missing",
		Match: func(ctx *DiagContext) bool {
			return ctx.NodeInstalled && !ctx.OpenClawInstalled
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "openclaw_missing", Severity: SeverityCritical,
				Title: "OpenClaw not installed", TitleZH: "OpenClaw 未安装",
				Description: "OpenClaw is not installed in WSL.",
				DescZH:      "WSL 中未安装 OpenClaw。",
				RepairID: "install_openclaw", AutoRepair: true,
			}
		},
	}
}

func ruleGatewayDown() Rule {
	return Rule{
		ID: "gateway_down",
		Match: func(ctx *DiagContext) bool {
			return ctx.OpenClawInstalled && !ctx.GatewayReachable
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "gateway_down", Severity: SeverityWarning,
				Title: "Gateway not responding", TitleZH: "网关无响应",
				Description: "OpenClaw gateway on port 18789 is not reachable.",
				DescZH:      "OpenClaw 网关（端口 18789）无法连接。",
				RepairID: "restart_gateway", AutoRepair: true,
			}
		},
	}
}

func ruleDiskLow() Rule {
	return Rule{
		ID:    "disk_low",
		Match: func(ctx *DiagContext) bool { return ctx.DiskFreeGB > 0 && ctx.DiskFreeGB < 10 },
		Diagnose: func(ctx *DiagContext) Issue {
			return Issue{
				ID: "disk_low", Severity: SeverityWarning,
				Title:   fmt.Sprintf("Low disk space (%.1f GB)", ctx.DiskFreeGB),
				TitleZH: fmt.Sprintf("磁盘空间不足（%.1f GB）", ctx.DiskFreeGB),
				Description: "At least 10 GB of free disk space is recommended.",
				DescZH:      "建议至少有 10 GB 可用磁盘空间。",
				AutoRepair: false,
			}
		},
	}
}

func ruleMemoryLow() Rule {
	return Rule{
		ID:    "memory_low",
		Match: func(ctx *DiagContext) bool { return ctx.MemoryGB > 0 && ctx.MemoryGB < 4 },
		Diagnose: func(ctx *DiagContext) Issue {
			return Issue{
				ID: "memory_low", Severity: SeverityWarning,
				Title:   fmt.Sprintf("Low memory (%.0f GB)", ctx.MemoryGB),
				TitleZH: fmt.Sprintf("内存不足（%.0f GB）", ctx.MemoryGB),
				Description: "At least 4 GB of RAM is recommended.",
				DescZH:      "建议至少 4 GB 内存。",
				AutoRepair: false,
			}
		},
	}
}

func ruleOSTooOld() Rule {
	return Rule{
		ID:    "os_too_old",
		Match: func(ctx *DiagContext) bool { return ctx.OSBuild > 0 && ctx.OSBuild < 19041 },
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "os_too_old", Severity: SeverityCritical,
				Title: "Windows version too old for WSL2", TitleZH: "Windows 版本过低",
				Description: "WSL2 requires Windows 10 build 19041 or later.",
				DescZH:      "WSL2 需要 Windows 10 build 19041 或更高版本。",
				AutoRepair: false,
			}
		},
	}
}

func ruleSharpBuildError() Rule {
	return Rule{
		ID: "sharp_build_error",
		Match: func(ctx *DiagContext) bool {
			return containsAny(ctx.ErrorLog, "sharp", "node-gyp", "prebuild", "libvips")
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "sharp_build_error", Severity: SeverityCritical,
				Title: "sharp/native module build failure", TitleZH: "sharp 原生模块编译失败",
				Description: "The 'sharp' image library failed to compile. Usually fixed by installing build-essential.",
				DescZH:      "图片处理库 sharp 编译失败。通常安装 build-essential 可解决。",
				RepairID: "fix_sharp", AutoRepair: true,
			}
		},
	}
}

func rulePathError() Rule {
	return Rule{
		ID: "path_error",
		Match: func(ctx *DiagContext) bool {
			return containsAny(ctx.ErrorLog, "command not found", "not recognized", "no such file")
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "path_error", Severity: SeverityWarning,
				Title: "PATH configuration issue", TitleZH: "PATH 环境变量异常",
				Description: "A command was not found, likely due to incorrect PATH configuration.",
				DescZH:      "命令未找到，可能是 PATH 环境变量配置有误。",
				RepairID: "fix_path", AutoRepair: true,
			}
		},
	}
}

func ruleNetworkError() Rule {
	return Rule{
		ID: "network_error",
		Match: func(ctx *DiagContext) bool {
			return containsAny(ctx.ErrorLog, "ETIMEDOUT", "ECONNREFUSED", "DNS", "network", "getaddrinfo", "ENOTFOUND")
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "network_error", Severity: SeverityWarning,
				Title: "Network connectivity issue", TitleZH: "网络连接问题",
				Description: "A network error was detected. Check your internet connection or try using a mirror/proxy.",
				DescZH:      "检测到网络错误。请检查网络连接，或尝试换源/使用代理。",
				AutoRepair: false,
			}
		},
	}
}

func ruleAPIKeyError() Rule {
	return Rule{
		ID: "api_key_error",
		Match: func(ctx *DiagContext) bool {
			return containsAny(ctx.ErrorLog, "invalid api key", "unauthorized", "401", "api_key")
		},
		Diagnose: func(_ *DiagContext) Issue {
			return Issue{
				ID: "api_key_error", Severity: SeverityWarning,
				Title: "API Key validation failed", TitleZH: "API Key 验证失败",
				Description: "The configured API Key appears to be invalid. Please re-enter it.",
				DescZH:      "配置的 API Key 似乎无效，请重新输入。",
				RepairID: "reconfig_apikey", AutoRepair: false,
			}
		},
	}
}
