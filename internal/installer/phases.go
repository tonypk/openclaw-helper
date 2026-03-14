// Package installer provides the installation orchestrator for OpenClaw.
package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Phase represents an installation phase.
type Phase string

const (
	PhaseIdle        Phase = "idle"
	PhasePrecheck    Phase = "precheck"
	PhaseWSL         Phase = "wsl"
	PhaseReboot      Phase = "reboot_pending"
	PhaseUbuntu      Phase = "ubuntu_config"
	PhaseNode        Phase = "node"
	PhaseOpenClaw    Phase = "openclaw"
	PhaseConfig      Phase = "config"
	PhaseVerify      Phase = "verify"
	PhaseDone        Phase = "done"
	PhaseError       Phase = "error"
	PhaseCancelled   Phase = "cancelled"
)

// PhaseStatus represents the status of a single phase.
type PhaseStatus string

const (
	PhasePending    PhaseStatus = "pending"
	PhaseRunning    PhaseStatus = "running"
	PhaseCompleted  PhaseStatus = "completed"
	PhaseFailed     PhaseStatus = "failed"
	PhaseSkipped    PhaseStatus = "skipped"
)

// PhaseInfo describes a phase with display metadata.
type PhaseInfo struct {
	Phase       Phase   `json:"phase"`
	Label       string  `json:"label"`
	LabelZH     string  `json:"label_zh"`
	Order       int     `json:"order"`
}

// AllPhases returns the ordered list of installation phases.
func AllPhases() []PhaseInfo {
	return []PhaseInfo{
		{PhasePrecheck, "System Check", "系统检测", 0},
		{PhaseWSL, "Install WSL2", "安装 WSL2", 1},
		{PhaseUbuntu, "Configure Ubuntu", "配置 Ubuntu", 2},
		{PhaseNode, "Install Node.js", "安装 Node.js", 3},
		{PhaseOpenClaw, "Install OpenClaw", "安装 OpenClaw", 4},
		{PhaseConfig, "Configure", "基础配置", 5},
		{PhaseVerify, "Verify", "验证安装", 6},
	}
}

// phaseOrder maps phase to execution order for comparison.
var phaseOrder = map[Phase]int{
	PhaseIdle:      -1,
	PhasePrecheck:  0,
	PhaseWSL:       1,
	PhaseReboot:    2,
	PhaseUbuntu:    3,
	PhaseNode:      4,
	PhaseOpenClaw:  5,
	PhaseConfig:    6,
	PhaseVerify:    7,
	PhaseDone:      8,
}

// nextPhase returns the phase that follows the given one.
func nextPhase(p Phase) Phase {
	switch p {
	case PhaseIdle:
		return PhasePrecheck
	case PhasePrecheck:
		return PhaseWSL
	case PhaseWSL:
		return PhaseUbuntu
	case PhaseReboot:
		return PhaseUbuntu
	case PhaseUbuntu:
		return PhaseNode
	case PhaseNode:
		return PhaseOpenClaw
	case PhaseOpenClaw:
		return PhaseConfig
	case PhaseConfig:
		return PhaseVerify
	case PhaseVerify:
		return PhaseDone
	default:
		return PhaseDone
	}
}

// ProgressEvent is sent to the frontend during installation.
type ProgressEvent struct {
	Phase      Phase       `json:"phase"`
	Status     PhaseStatus `json:"status"`
	Message    string      `json:"message"`
	Detail     string      `json:"detail,omitempty"`
	Progress   int         `json:"progress"`       // 0-100 for current phase
	Overall    int         `json:"overall"`         // 0-100 for entire install
	Timestamp  time.Time   `json:"timestamp"`
}

// InstallState is persisted to disk for resume capability.
type InstallState struct {
	CurrentPhase  Phase                  `json:"current_phase"`
	PhaseResults  map[Phase]PhaseStatus  `json:"phase_results"`
	StartedAt     time.Time              `json:"started_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	ErrorPhase    Phase                  `json:"error_phase,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}

// NewInstallState creates a fresh installation state.
func NewInstallState() *InstallState {
	now := time.Now()
	results := make(map[Phase]PhaseStatus)
	for _, p := range AllPhases() {
		results[p.Phase] = PhasePending
	}
	return &InstallState{
		CurrentPhase: PhaseIdle,
		PhaseResults: results,
		StartedAt:    now,
		UpdatedAt:    now,
		Metadata:     make(map[string]string),
	}
}

// stateFilePath returns the path to the persisted state file.
func stateFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	stateDir := filepath.Join(dir, "openclaw-helper")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return "", fmt.Errorf("create state dir: %w", err)
	}
	return filepath.Join(stateDir, "install-state.json"), nil
}

// Save persists the install state to disk.
func (s *InstallState) Save() error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadState loads persisted install state, or returns nil if none exists.
func LoadState() (*InstallState, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var state InstallState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}
	return &state, nil
}

// ClearState removes the persisted state file.
func ClearState() error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SaveTo persists state to a specific path (for testing).
func (s *InstallState) SaveTo(path string) error {
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadStateFrom loads state from a specific path (for testing).
func LoadStateFrom(path string) (*InstallState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state InstallState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
