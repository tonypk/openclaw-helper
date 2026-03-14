# OpenClaw Helper / OpenClaw 小助手

[English](#english) | [中文](#中文)

> **[>>> Download for Windows (.exe) <<<](https://github.com/tonypk/openclaw-helper/releases/latest)** | **[>>> Windows 下载 (.exe) <<<](https://github.com/tonypk/openclaw-helper/releases/latest)**

---

## English

**One-click installer for [OpenClaw](https://openclaw.ai/) on Windows.**

OpenClaw Helper is an **installation assistant** that helps non-technical users set up [OpenClaw](https://openclaw.ai/) (an open-source personal AI assistant) on Windows. Once installed, all features are managed through the official [OpenClaw Gateway Dashboard](http://localhost:18789). No command line knowledge required.

### Download

**[Download OpenClaw Helper (Windows .exe)](https://github.com/tonypk/openclaw-helper/releases/latest)** — Double-click to install, no command line needed.

Also available: [macOS (.dmg)](https://github.com/tonypk/openclaw-helper/releases/latest)

Join our [Telegram channel](https://t.me/openclaw_helper) for updates and support.

### Features

- **One-Click Install** - Automatically sets up WSL2, Node.js, and OpenClaw
- **System Detection** - Checks Windows version, RAM, disk space, virtualization, WSL2, Node.js
- **Start/Stop Gateway** - Manage the OpenClaw Gateway service lifecycle
- **Open Console** - Quick access to the official OpenClaw Dashboard (`http://localhost:18789`)
- **Smart Diagnostics** - 15 built-in diagnostic rules with auto-repair playbooks
- **AI Chat Assistant** - Get help during installation with FAQ matching, diagnostics, and LLM-powered answers
- **Auto Updates** - Built-in updater checks for new versions automatically
- **Bilingual** - Full Chinese and English support

### Architecture

```
Tauri v2 (Rust) + Vue3 + Go Helper
         |                    |
    Window/Tray        System Detection
    IPC Bridge         WSL2 Management
    Auto Update        Install Orchestrator
                       AI Chat + Diagnostics
```

- **Frontend**: Vue3 + TypeScript + Pinia + vue-i18n
- **Desktop Shell**: Tauri v2 (Rust) - system tray, auto-update, native window
- **Backend**: Go helper process (sidecar) - system checks, WSL management, installation, diagnostics, AI chat

### Requirements

- Windows 10 (build 19041+) or Windows 11
- 4 GB RAM minimum
- 10 GB free disk space
- Internet connection

### Development

**Prerequisites**: Go 1.22+, Node.js 22+, Rust 1.70+

```bash
# Clone
git clone https://github.com/tonypk/openclaw-helper.git
cd openclaw-helper

# Run Go tests
go test -v ./...

# Build Go helper for current platform
make build-dev

# Frontend development
cd frontend
npm install
npm run dev

# Full Tauri development (requires Go helper binary in src-tauri/binaries/)
# First build Go sidecar:
go build -o frontend/src-tauri/binaries/och-helper-$(rustc -vV | grep host | cut -d' ' -f2) ./cmd/helper
# Then run Tauri dev:
npx @tauri-apps/cli@2 dev
```

### Building for Release

```bash
# Build Go helper for Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o frontend/src-tauri/binaries/och-helper-x86_64-pc-windows-msvc.exe ./cmd/helper

# Build Tauri app (NSIS installer)
cd frontend
npx @tauri-apps/cli@2 build
```

### Project Structure

```
openclaw-helper/
+-- cmd/helper/           # Go helper entry point
+-- internal/
|   +-- checker/          # System detection (OS, RAM, disk, WSL, Node.js)
|   +-- installer/        # Installation state machine (7 phases)
|   +-- diagnosis/        # Diagnostic engine (15 rules, 10 playbooks)
|   +-- chat/             # AI Chat (FAQ, LLM proxy, context)
|   +-- ipc/              # JSON-RPC 2.0 over Unix socket / Named Pipe
|   +-- types/            # Shared types
+-- pkg/wsl/              # WSL2 operation wrappers
+-- frontend/
|   +-- src/              # Vue3 app (6 views, 8 components, 4 stores)
|   +-- src-tauri/        # Tauri v2 Rust backend
+-- .github/workflows/    # CI/CD (test + release)
```

### License

MIT

---

## 中文

**Windows 上 [OpenClaw](https://openclaw.ai/) 的一键安装助手。**

OpenClaw 小助手是一个**安装助手**，帮助不懂技术的用户在 Windows 上轻松安装 [OpenClaw](https://openclaw.ai/)（开源 AI 私人助手）。安装完成后，所有功能通过官方 [OpenClaw 控制台](http://localhost:18789) 使用。无需命令行知识。

### 下载安装

**[下载 OpenClaw 小助手 (Windows .exe)](https://github.com/tonypk/openclaw-helper/releases/latest)** — 双击安装，无需命令行。

也提供：[macOS (.dmg)](https://github.com/tonypk/openclaw-helper/releases/latest)

加入我们的 [Telegram 频道](https://t.me/openclaw_helper) 获取更新和支持。

### 功能特性

- **一键安装** - 自动配置 WSL2、Node.js 和 OpenClaw
- **系统检测** - 检查 Windows 版本、内存、磁盘空间、虚拟化、WSL2、Node.js
- **启动/停止** - 管理 OpenClaw Gateway 服务的启停
- **打开控制台** - 快速跳转到官方 OpenClaw 控制台（`http://localhost:18789`）
- **智能诊断** - 15 条内置诊断规则，配套自动修复方案
- **AI 聊天助手** - 安装过程中随时获取帮助，支持 FAQ 匹配、诊断分析和 LLM 智能回答
- **自动更新** - 内置更新器自动检查新版本
- **中英双语** - 完整的中文和英文界面支持

### 技术架构

```
Tauri v2 (Rust) + Vue3 + Go Helper
         |                    |
    窗口/托盘             系统检测
    IPC 桥接             WSL2 管理
    自动更新             安装编排器
                       AI 聊天 + 诊断
```

- **前端**: Vue3 + TypeScript + Pinia + vue-i18n
- **桌面外壳**: Tauri v2 (Rust) - 系统托盘、自动更新、原生窗口
- **后端**: Go helper 进程 (sidecar) - 系统检测、WSL 管理、安装、诊断、AI 聊天

### 系统要求

- Windows 10 (版本 19041+) 或 Windows 11
- 最低 4 GB 内存
- 10 GB 可用磁盘空间
- 网络连接

### 开发

**前置条件**: Go 1.22+, Node.js 22+, Rust 1.70+

```bash
# 克隆项目
git clone https://github.com/tonypk/openclaw-helper.git
cd openclaw-helper

# 运行 Go 测试
go test -v ./...

# 构建 Go helper（当前平台）
make build-dev

# 前端开发
cd frontend
npm install
npm run dev

# 完整 Tauri 开发（需要先构建 Go helper 到 src-tauri/binaries/）
go build -o frontend/src-tauri/binaries/och-helper-$(rustc -vV | grep host | cut -d' ' -f2) ./cmd/helper
npx @tauri-apps/cli@2 dev
```

### 构建发布版

```bash
# 构建 Windows 版 Go helper
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o frontend/src-tauri/binaries/och-helper-x86_64-pc-windows-msvc.exe ./cmd/helper

# 构建 Tauri 应用（NSIS 安装包）
cd frontend
npx @tauri-apps/cli@2 build
```

### 项目结构

```
openclaw-helper/
+-- cmd/helper/           # Go helper 入口
+-- internal/
|   +-- checker/          # 系统检测（OS、内存、磁盘、WSL、Node.js）
|   +-- installer/        # 安装状态机（7 个阶段）
|   +-- diagnosis/        # 诊断引擎（15 条规则、10 个修复方案）
|   +-- chat/             # AI 聊天（FAQ、LLM 代理、上下文）
|   +-- ipc/              # JSON-RPC 2.0（Unix socket / Named Pipe）
|   +-- types/            # 共享类型
+-- pkg/wsl/              # WSL2 操作封装
+-- frontend/
|   +-- src/              # Vue3 应用（6 个页面、8 个组件、4 个 Store）
|   +-- src-tauri/        # Tauri v2 Rust 后端
+-- .github/workflows/    # CI/CD（测试 + 发布）
```

### 开源协议

MIT
