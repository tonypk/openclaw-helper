package diagnosis

import (
	"context"
	"fmt"
	"time"
)

// RepairResult is the outcome of executing a repair playbook.
type RepairResult struct {
	RepairID string `json:"repair_id"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	MsgZH    string `json:"msg_zh,omitempty"`
}

// Playbook defines a repair action.
type Playbook struct {
	ID          string
	Description string
	DescZH      string
	// Execute runs the repair. All commands are predefined — no dynamic/LLM-generated commands.
	Execute func(ctx context.Context) RepairResult
}

// PlaybookRegistry holds all available repair playbooks.
type PlaybookRegistry struct {
	playbooks map[string]Playbook
}

// NewPlaybookRegistry creates a registry with all built-in playbooks.
func NewPlaybookRegistry() *PlaybookRegistry {
	r := &PlaybookRegistry{playbooks: make(map[string]Playbook)}
	for _, p := range builtinPlaybooks() {
		r.playbooks[p.ID] = p
	}
	return r
}

// Get returns a playbook by ID.
func (r *PlaybookRegistry) Get(id string) (Playbook, bool) {
	p, ok := r.playbooks[id]
	return p, ok
}

// Run executes a playbook by ID.
func (r *PlaybookRegistry) Run(ctx context.Context, id string) RepairResult {
	p, ok := r.playbooks[id]
	if !ok {
		return RepairResult{
			RepairID: id,
			Success:  false,
			Message:  fmt.Sprintf("unknown playbook: %s", id),
		}
	}
	return p.Execute(ctx)
}

// List returns all available playbook IDs.
func (r *PlaybookRegistry) List() []string {
	ids := make([]string, 0, len(r.playbooks))
	for id := range r.playbooks {
		ids = append(ids, id)
	}
	return ids
}

func builtinPlaybooks() []Playbook {
	return []Playbook{
		{
			ID:          "install_wsl",
			Description: "Install WSL2",
			DescZH:      "安装 WSL2",
			Execute:     playbookInstallWSL,
		},
		{
			ID:          "upgrade_wsl",
			Description: "Upgrade WSL1 to WSL2",
			DescZH:      "升级 WSL1 到 WSL2",
			Execute:     playbookUpgradeWSL,
		},
		{
			ID:          "install_ubuntu",
			Description: "Install Ubuntu in WSL",
			DescZH:      "在 WSL 中安装 Ubuntu",
			Execute:     playbookInstallUbuntu,
		},
		{
			ID:          "install_node",
			Description: "Install Node.js 22 via nvm",
			DescZH:      "通过 nvm 安装 Node.js 22",
			Execute:     playbookInstallNode,
		},
		{
			ID:          "upgrade_node",
			Description: "Upgrade Node.js to v22",
			DescZH:      "升级 Node.js 到 v22",
			Execute:     playbookUpgradeNode,
		},
		{
			ID:          "install_openclaw",
			Description: "Install OpenClaw via npm",
			DescZH:      "通过 npm 安装 OpenClaw",
			Execute:     playbookInstallOpenClaw,
		},
		{
			ID:          "restart_gateway",
			Description: "Restart OpenClaw gateway",
			DescZH:      "重启 OpenClaw 网关",
			Execute:     playbookRestartGateway,
		},
		{
			ID:          "fix_sharp",
			Description: "Fix sharp/native module build issues",
			DescZH:      "修复 sharp 原生模块编译问题",
			Execute:     playbookFixSharp,
		},
		{
			ID:          "fix_path",
			Description: "Fix PATH configuration",
			DescZH:      "修复 PATH 环境变量",
			Execute:     playbookFixPath,
		},
		{
			ID:          "guide_virt",
			Description: "Guide: Enable virtualization in BIOS",
			DescZH:      "教程：在 BIOS 中开启虚拟化",
			Execute:     playbookGuideVirt,
		},
	}
}

// Playbook implementations use wslRun which is defined per-platform.

func playbookInstallWSL(ctx context.Context) RepairResult {
	_, err := wslExec(ctx, "", "wsl --install --no-launch", 10*time.Minute)
	if err != nil {
		return RepairResult{RepairID: "install_wsl", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "install_wsl", Success: true,
		Message: "WSL2 installed. A reboot may be required.",
		MsgZH:   "WSL2 安装完成。可能需要重启电脑。",
	}
}

func playbookUpgradeWSL(ctx context.Context) RepairResult {
	_, err := wslExec(ctx, "", "wsl --set-default-version 2", 2*time.Minute)
	if err != nil {
		return RepairResult{RepairID: "upgrade_wsl", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "upgrade_wsl", Success: true,
		Message: "WSL default version set to 2.",
		MsgZH:   "WSL 默认版本已设为 2。",
	}
}

func playbookInstallUbuntu(ctx context.Context) RepairResult {
	_, err := wslExec(ctx, "", "wsl --install -d Ubuntu --no-launch", 10*time.Minute)
	if err != nil {
		return RepairResult{RepairID: "install_ubuntu", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "install_ubuntu", Success: true,
		Message: "Ubuntu installed in WSL.",
		MsgZH:   "Ubuntu 已安装到 WSL。",
	}
}

func playbookInstallNode(ctx context.Context) RepairResult {
	cmds := []string{
		`curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash`,
		`export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm install 22 && nvm alias default 22`,
	}
	for _, cmd := range cmds {
		if _, err := wslRunInDistro(ctx, "Ubuntu", cmd, 5*time.Minute); err != nil {
			return RepairResult{RepairID: "install_node", Success: false, Message: err.Error()}
		}
	}
	return RepairResult{
		RepairID: "install_node", Success: true,
		Message: "Node.js 22 installed via nvm.",
		MsgZH:   "Node.js 22 已通过 nvm 安装。",
	}
}

func playbookUpgradeNode(ctx context.Context) RepairResult {
	cmd := `export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nvm install 22 && nvm alias default 22`
	if _, err := wslRunInDistro(ctx, "Ubuntu", cmd, 5*time.Minute); err != nil {
		return RepairResult{RepairID: "upgrade_node", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "upgrade_node", Success: true,
		Message: "Node.js upgraded to v22.",
		MsgZH:   "Node.js 已升级到 v22。",
	}
}

func playbookInstallOpenClaw(ctx context.Context) RepairResult {
	cmd := `export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && npm install -g openclaw`
	if _, err := wslRunInDistro(ctx, "Ubuntu", cmd, 10*time.Minute); err != nil {
		return RepairResult{RepairID: "install_openclaw", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "install_openclaw", Success: true,
		Message: "OpenClaw installed.",
		MsgZH:   "OpenClaw 安装完成。",
	}
}

func playbookRestartGateway(ctx context.Context) RepairResult {
	// Kill existing, then restart
	wslRunInDistro(ctx, "Ubuntu", "pkill -f openclaw || true", 10*time.Second)
	cmd := `export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh" && nohup openclaw start > /tmp/openclaw.log 2>&1 &`
	if _, err := wslRunInDistro(ctx, "Ubuntu", cmd, 30*time.Second); err != nil {
		return RepairResult{RepairID: "restart_gateway", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "restart_gateway", Success: true,
		Message: "OpenClaw gateway restarted.",
		MsgZH:   "OpenClaw 网关已重启。",
	}
}

func playbookFixSharp(ctx context.Context) RepairResult {
	cmd := `sudo apt-get update -y && sudo apt-get install -y build-essential python3`
	if _, err := wslRunInDistro(ctx, "Ubuntu", cmd, 5*time.Minute); err != nil {
		return RepairResult{RepairID: "fix_sharp", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "fix_sharp", Success: true,
		Message: "Build dependencies installed. Please retry the installation.",
		MsgZH:   "编译依赖已安装，请重试安装。",
	}
}

func playbookFixPath(ctx context.Context) RepairResult {
	cmd := `export NVM_DIR="$HOME/.nvm" && echo 'export NVM_DIR="$HOME/.nvm"' >> ~/.bashrc && echo '[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"' >> ~/.bashrc`
	if _, err := wslRunInDistro(ctx, "Ubuntu", cmd, 10*time.Second); err != nil {
		return RepairResult{RepairID: "fix_path", Success: false, Message: err.Error()}
	}
	return RepairResult{
		RepairID: "fix_path", Success: true,
		Message: "PATH configuration updated in ~/.bashrc.",
		MsgZH:   "PATH 配置已更新到 ~/.bashrc。",
	}
}

func playbookGuideVirt(_ context.Context) RepairResult {
	return RepairResult{
		RepairID: "guide_virt", Success: true,
		Message: "To enable virtualization:\n1. Restart your computer\n2. Enter BIOS/UEFI (usually F2, F12, or Del key)\n3. Find 'Virtualization Technology' or 'VT-x' or 'AMD-V'\n4. Set it to Enabled\n5. Save and restart",
		MsgZH:   "开启虚拟化步骤：\n1. 重启电脑\n2. 进入 BIOS/UEFI（通常按 F2、F12 或 Del 键）\n3. 找到「虚拟化技术」或「VT-x」或「AMD-V」\n4. 设为「启用」\n5. 保存并重启",
	}
}
