# Self-Healing Engine Design Spec

**Date:** 2026-03-20
**Status:** Approved
**Scope:** Enhance openclaw-helper with automatic problem detection, repair, and retry — reducing manual user intervention during installation.

## Overview

Add a self-healing loop to every installation phase. When a phase fails, the system automatically diagnoses the error, finds a matching repair playbook, executes repair strategies in order, and retries. All repair logic (playbooks, diagnostic rules, repair scripts, FAQ, config) is hot-updatable via the remote manifest — only the Go binary and Tauri frontend require a new release.

## Architecture

### Four-Layer Escalation Chain

| Layer | Name | Coverage | Behavior |
|-------|------|----------|----------|
| L1 | Auto-Retry | ~40% | Network timeout → retry with exponential backoff, swap mirror |
| L2 | Playbook Repair | ~50% | Match error → execute repair script chain → retry phase |
| L3 | AI Chat | ~8% | Playbooks exhausted → AI analyzes logs, gives personalized guidance |
| L4 | Crash Report | ~2% | AI can't solve → collect full context → send to Telegram + GitHub |

### Phase Executor Loop

Every installation phase runs through this cycle:

```
Execute Script → Verify Result → Pass? ──YES──→ ✅ Done
                                   │ NO
                                   ▼
                              Diagnose Error
                                   │
                                   ▼
                           Find Playbook (Local/Remote)
                                   │
                                   ▼
                        Execute Repair Actions
                        (Strategy 1 → 2 → 3...)
                                   │
                                   ▼
                           Retry < Max? ──YES──→ Re-execute Phase
                                   │ NO
                                   ▼
                        Re-sync Manifest (force)
                           New Playbook? ──YES──→ Try new playbook
                                   │ NO
                                   ▼
                        AI Chat → Crash Report
```

## Component 1: Remote Playbook System

### Playbook Data Structure

Each playbook is a JSON object stored in `playbooks/all.json` in the openclaw-scripts repository:

```json
{
  "id": "npm_network_timeout",
  "match": {
    "phase": "openclaw",
    "error_patterns": ["ETIMEDOUT", "ECONNRESET", "npm ERR! network"],
    "check_commands": ["curl -s --max-time 5 https://registry.npmjs.org"]
  },
  "severity": "critical",
  "strategies": [
    {
      "name": "retry_with_mirror",
      "risk": "low",
      "actions": [
        {"type": "run", "script": "repair/npm-mirror.sh", "timeout": 30},
        {"type": "retry_phase"}
      ]
    },
    {
      "name": "clear_cache_retry",
      "risk": "low",
      "actions": [
        {"type": "run", "script": "repair/npm-clear-cache.sh", "timeout": 30},
        {"type": "retry_phase"}
      ]
    },
    {
      "name": "offline_install",
      "risk": "medium",
      "actions": [
        {"type": "run", "script": "repair/npm-offline-pack.sh", "timeout": 120}
      ]
    }
  ]
}
```

### Playbook Fields

- **id** — Unique identifier, matches diagnostic rule IDs where applicable.
- **match.phase** — Which installation phase this playbook applies to.
- **match.error_patterns** — Regex patterns matched against script stderr/stdout.
- **match.check_commands** — Optional commands to run for deeper diagnosis (e.g., test connectivity).
- **severity** — `critical` (blocks installation) or `warning` (degraded but can continue).
- **strategies** — Ordered list of repair strategies. Tried sequentially; first success wins.
- **strategy.risk** — `low`, `medium`, `high`. All execute automatically (user chose full-auto). Used for logging and future policy changes.
- **strategy.actions** — Ordered action chain within a strategy.

### Action Types

| Type | Description | Fields |
|------|-------------|--------|
| `run` | Execute a repair script | `script`, `timeout`, `runtime` (optional, default: inferred from extension) |
| `retry_phase` | Re-run the current installation phase | — |
| `reboot` | Prompt user to restart, persist state | `message` |
| `set_env` | Set an environment variable in WSL | `key`, `value` |
| `write_file` | Write content to a file path | `path`, `content`, `mode` |

### Initial Playbooks (ship with v0.5.0)

| ID | Phase | Problem | Strategies |
|----|-------|---------|------------|
| `npm_network_timeout` | openclaw | npm network timeout | Mirror swap → cache clear → offline pack |
| `npm_sharp_build` | openclaw | sharp native module build failure | Install build deps → prebuilt binary → skip optional |
| `wsl_virt_disabled` | wsl | Virtualization not enabled | Enable Hyper-V feature → prompt BIOS instructions + reboot |
| `wsl_install_fail` | wsl | WSL installation error | Reset WSL → manual component install → Windows Update |
| `node_path_missing` | node | Node.js not in PATH after nvm install | Fix .bashrc → source profile → reinstall nvm |
| `node_version_wrong` | node | Node.js < 22 | nvm install 22 → nvm alias default → reinstall |
| `ubuntu_apt_fail` | ubuntu_config | apt-get fails (lock/network) | Kill apt locks → fix sources.list → retry |
| `dns_resolution_fail` | * (any) | DNS resolution failure | Set Google DNS → flush DNS cache → retry |
| `disk_space_low` | * (any) | < 5 GB free | Clean npm cache + apt cache + WSL compact → retry |
| `path_corruption` | * (any) | Command not found (PATH issue) | Rebuild PATH from known locations |

## Component 2: Unified Remote Manifest V2

### Manifest Structure

Extends the existing V1 manifest to cover all hot-updatable resources:

```json
{
  "version": 2,
  "min_helper_version": "0.4.0",
  "phases": {
    "wsl": {
      "install": {"url": "scripts/wsl/install.ps1", "sha256": "...", "runtime": "powershell", "timeout_seconds": 600},
      "verify": {"url": "scripts/wsl/verify.ps1", "sha256": "...", "runtime": "powershell", "timeout_seconds": 60}
    },
    "ubuntu_config": { "...": "existing" },
    "node": { "...": "existing" },
    "openclaw": { "...": "existing" },
    "config": { "...": "existing" },
    "verify": { "...": "existing" }
  },
  "playbooks": {
    "version": 3,
    "url": "playbooks/all.json",
    "sha256": "..."
  },
  "diagnostics": {
    "version": 2,
    "url": "diagnostics/rules.json",
    "sha256": "..."
  },
  "faq": {
    "version": 1,
    "url": "faq/faq.json",
    "sha256": "..."
  },
  "config": {
    "version": 1,
    "url": "config/global.json",
    "sha256": "..."
  },
  "repair_scripts": {
    "npm-mirror.sh": {"url": "scripts/repair/npm-mirror.sh", "sha256": "..."},
    "npm-clear-cache.sh": {"url": "scripts/repair/npm-clear-cache.sh", "sha256": "..."},
    "fix-path.sh": {"url": "scripts/repair/fix-path.sh", "sha256": "..."},
    "enable-virt.ps1": {"url": "scripts/repair/enable-virt.ps1", "sha256": "..."}
  }
}
```

### Cache & Sync Behavior

1. **App startup** — Background goroutine fetches manifest (existing behavior, expanded scope).
2. **During installation** — After a playbook repair fails, force re-sync manifest to check for newly published playbooks.
3. **Three-tier fallback** — Remote fetch → local cache (`%APPDATA%/openclaw-helper/scripts/`) → go:embed embedded files.
4. **Integrity** — All resources verified by SHA-256 before use.
5. **ETag caching** — HTTP 304 reduces bandwidth for unchanged resources.
6. **Per-resource versioning** — Each resource category has an independent version number; only changed resources are downloaded.
7. **min_helper_version** — If remote manifest requires newer Go binary, prompt user to update the app via Tauri updater.

### openclaw-scripts Repository Structure

```
openclaw-scripts/
├── manifest.json              # Unified manifest V2
├── scripts/
│   ├── wsl/                   # Existing: WSL installation
│   ├── ubuntu/                # Existing: Ubuntu config
│   ├── node/                  # Existing: Node.js setup
│   ├── openclaw/              # Existing: OpenClaw install
│   ├── config/                # Existing: Configuration
│   ├── verify/                # Existing: Verification
│   ├── diagnostics/           # Existing: Diagnostic collection
│   └── repair/                # NEW: Repair scripts
│       ├── npm-mirror.sh
│       ├── npm-clear-cache.sh
│       ├── npm-offline-pack.sh
│       ├── fix-path.sh
│       ├── fix-bashrc.sh
│       ├── enable-virt.ps1
│       ├── reset-wsl.ps1
│       ├── fix-apt-locks.sh
│       ├── set-dns.sh
│       └── clean-disk.sh
├── playbooks/                 # NEW: Repair playbooks
│   └── all.json
├── diagnostics/               # NEW: Externalized diagnostic rules
│   └── rules.json
├── faq/                       # NEW: FAQ data
│   └── faq.json
└── config/                    # NEW: Global configuration
    └── global.json
```

## Component 3: Self-Healing Executor

### Go Interface Design

New package `internal/healing` wraps the existing phase execution with self-healing logic:

```go
package healing

type Executor struct {
    scriptRunner  *scriptrun.Runner
    diagEngine    *diagnosis.Engine
    playbookStore *playbook.Store
    cache         *scriptrun.Cache
    maxRetries    int  // default: 3
    eventCh       chan<- Event
}

type Event struct {
    Type     string // "heal_start", "heal_strategy", "heal_repair", "heal_retry", "heal_resolved", "heal_escalate"
    Issue    string
    Strategy string
    Detail   string
    Attempt  int
    MaxRetry int
}

func (h *Executor) ExecutePhase(ctx context.Context, phase Phase) PhaseResult
```

### Execution Flow (pseudocode)

```
func ExecutePhase(ctx, phase):
    for attempt = 0..maxRetries:
        result = scriptRunner.Run(ctx, phase)
        if result.Success:
            return success (with healing_history if any repairs happened)

        issues = diagEngine.Analyze(result.Error, result.Logs)
        playbook = playbookStore.FindMatch(phase, issues)

        if playbook == nil:
            cache.ForceSync()  // maybe new playbook was just published
            playbook = playbookStore.FindMatch(phase, issues)

        if playbook == nil:
            break  // no playbook available, escalate

        emit Event{Type: "heal_start", Issue: playbook.ID}

        for each strategy in playbook.Strategies:
            emit Event{Type: "heal_strategy", Strategy: strategy.Name}

            repairOK = executeStrategy(ctx, strategy)
            if repairOK:
                emit Event{Type: "heal_retry", Attempt: attempt+1}
                break  // retry the phase
            // else try next strategy

    // all retries exhausted
    emit Event{Type: "heal_escalate", Detail: "ai_chat"}
    return failure with full context
```

### State Persistence

Extend `install-state.json` with healing history:

```json
{
  "current_phase": "openclaw",
  "phases": {},
  "healing_history": [
    {
      "phase": "openclaw",
      "issue": "npm_network_timeout",
      "strategies_tried": ["retry_with_mirror", "clear_cache_retry"],
      "resolved_by": "clear_cache_retry",
      "attempts": 2,
      "timestamp": "2026-03-20T10:30:00Z"
    }
  ]
}
```

After reboot or restart, the executor reads healing_history to avoid repeating failed strategies.

### OCH Protocol Extensions

New event types added to the `##OCH:` protocol for real-time frontend updates:

```
##OCH:HEAL:START:<issue_id>
##OCH:HEAL:STRATEGY:<strategy_name>
##OCH:HEAL:REPAIR:<script>:<status>       # status: running|success|failed
##OCH:HEAL:RETRY:<attempt>/<max>
##OCH:HEAL:RESOLVED:<issue_id>
##OCH:HEAL:ESCALATE:<target>              # target: ai_chat|crash_report
```

## Component 4: Frontend Changes

### Installation Progress View

Modify the existing installation progress view to show self-healing state:

**Normal flow** — No change. Phase shows spinner → checkmark.

**Healing in progress** — Phase shows wrench icon (🔧) + amber text "自动修复中". Expandable detail panel shows:
- Detected issue name
- List of strategies with status (✓ done, ● active, ○ pending)
- Real-time repair script output (scrolling log)

**Healing resolved** — Phase shows checkmark + small badge "自动修复了 N 个问题". Expandable shows what was fixed.

**Healing failed → AI** — Phase shows ❌. Automatically navigates to chat view with full error context pre-loaded. AI receives the complete diagnostic report and healing history.

**Healing failed → Crash Report** — After AI chat, if still unresolved, "提交详细报告给开发者" link auto-collects healing history + AI conversation + full logs.

### New Pinia Store

Add `useHealingStore` to track healing state per phase:

```typescript
interface HealingState {
  activePhase: string | null
  currentIssue: string | null
  strategies: Array<{name: string, status: 'pending' | 'running' | 'success' | 'failed'}>
  repairLog: string[]
  history: Array<{phase: string, issue: string, resolvedBy: string}>
}
```

### Localization

Add healing-related i18n keys for both en-US and zh-CN:
- `healing.autoRepairing` — "自动修复中" / "Auto-repairing"
- `healing.resolved` — "自动修复了 {n} 个问题" / "Auto-repaired {n} issues"
- `healing.strategy` — Strategy names (localized)
- `healing.escalateAI` — "自动修复失败，AI 助手已接管" / "Auto-repair failed, AI assistant taking over"
- `healing.escalateReport` — "提交详细报告给开发者" / "Submit detailed report to developer"

## Component 5: Playbook Store (new Go package)

New package `internal/playbook` manages loading, caching, and matching playbooks:

```go
package playbook

type Store struct {
    cache     *scriptrun.Cache
    playbooks []Playbook        // loaded from remote or embedded
    mu        sync.RWMutex
}

type Playbook struct {
    ID         string     `json:"id"`
    Match      MatchRule  `json:"match"`
    Severity   string     `json:"severity"`
    Strategies []Strategy `json:"strategies"`
}

type MatchRule struct {
    Phase         string   `json:"phase"`
    ErrorPatterns []string `json:"error_patterns"`
    CheckCommands []string `json:"check_commands"`
}

type Strategy struct {
    Name    string   `json:"name"`
    Risk    string   `json:"risk"`
    Actions []Action `json:"actions"`
}

type Action struct {
    Type    string `json:"type"`    // run, retry_phase, reboot, set_env, write_file
    Script  string `json:"script"`
    Timeout int    `json:"timeout"`
    Key     string `json:"key"`
    Value   string `json:"value"`
    Path    string `json:"path"`
    Content string `json:"content"`
    Mode    string `json:"mode"`
    Message string `json:"message"`
}

func (s *Store) Load(ctx context.Context) error           // load from cache/remote
func (s *Store) FindMatch(phase string, issues []diagnosis.Issue) *Playbook
func (s *Store) Reload(ctx context.Context) error         // force reload from remote
```

## Changes to Existing Code

### Modified Files

| File | Change |
|------|--------|
| `internal/installer/orchestrator.go` | Inject `healing.Executor`, delegate phase execution to it instead of running scripts directly |
| `internal/installer/phases.go` | Add `HealingHistory` field to `InstallState` |
| `internal/scriptrun/cache.go` | Extend to handle playbooks, diagnostics, FAQ, config (not just phase scripts) |
| `internal/scriptrun/manifest.go` | Parse V2 manifest fields (playbooks, diagnostics, faq, config, repair_scripts) |
| `internal/diagnosis/engine.go` | Support loading rules from remote JSON (not just hardcoded) |
| `internal/diagnosis/rules.go` | Keep hardcoded rules as fallback; merge with remote rules |
| `internal/chat/handler.go` | Accept healing context (history, failed strategies) for richer AI prompts |
| `internal/ipc/protocol.go` | Add healing event types to JSON-RPC notification schema |
| `internal/report/collector.go` | Include healing_history in crash reports |
| `frontend/src/stores/` | Add `useHealingStore` |
| `frontend/src/components/` | Add `HealingProgress.vue` component |
| `frontend/src/views/InstallView.vue` | Integrate healing progress display |
| `frontend/src/i18n/en-US.ts` | Add healing keys |
| `frontend/src/i18n/zh-CN.ts` | Add healing keys |

### New Files

| File | Purpose |
|------|---------|
| `internal/healing/executor.go` | Self-healing executor (core loop) |
| `internal/healing/executor_test.go` | Tests for healing executor |
| `internal/playbook/store.go` | Playbook loading, caching, matching |
| `internal/playbook/store_test.go` | Tests for playbook store |
| `internal/playbook/types.go` | Playbook, Strategy, Action types |
| `internal/scriptrun/fallback/repair/*.sh` | Embedded fallback repair scripts |
| `frontend/src/stores/healing.ts` | Healing state store |
| `frontend/src/components/HealingProgress.vue` | Healing progress UI component |

### openclaw-scripts New Files

| File | Purpose |
|------|---------|
| `playbooks/all.json` | All playbook definitions |
| `diagnostics/rules.json` | Externalized diagnostic rules |
| `faq/faq.json` | FAQ data |
| `config/global.json` | Global configuration |
| `scripts/repair/*.sh` | 10 initial repair scripts |

## Testing Strategy

- **Unit tests** for `healing.Executor` — mock scriptRunner, diagEngine, playbookStore to test the loop logic, retry counts, strategy selection, escalation.
- **Unit tests** for `playbook.Store` — test JSON parsing, pattern matching, fallback behavior.
- **Integration tests** — test manifest V2 parsing and cache sync with all resource types.
- **Manual QA** — trigger known failures (disconnect network, corrupt PATH, fill disk) and verify self-healing resolves them.

## Rollout

- **v0.5.0** — Ship self-healing engine with 10 initial playbooks.
- Post-launch: monitor crash reports for new failure patterns → add playbooks to openclaw-scripts without releasing new app version.
