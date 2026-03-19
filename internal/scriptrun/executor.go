package scriptrun

import (
	"context"
	"fmt"
	"log"

	"github.com/tonypk/openclaw-helper/internal/installer"
)

// ScriptPhaseExecutor implements installer.PhaseExecutor by running remote scripts.
type ScriptPhaseExecutor struct {
	phase  installer.Phase
	runner *Runner
	cache  *Cache
}

// NewScriptPhaseExecutor creates a new script-driven phase executor.
func NewScriptPhaseExecutor(phase installer.Phase, runner *Runner, cache *Cache) *ScriptPhaseExecutor {
	return &ScriptPhaseExecutor{
		phase:  phase,
		runner: runner,
		cache:  cache,
	}
}

func (s *ScriptPhaseExecutor) Phase() installer.Phase {
	return s.phase
}

// Verify runs the phase's verify script (if any) and returns true if the phase
// is already completed.
func (s *ScriptPhaseExecutor) Verify() bool {
	manifest := s.cache.GetManifest()
	if manifest == nil {
		return false
	}

	ps, ok := manifest.Phases[string(s.phase)]
	if !ok || ps.Verify == nil {
		return false
	}

	result, err := s.runner.Run(context.Background(), ps.Verify, nil)
	if err != nil {
		return false
	}

	if result.VerifyOK != nil {
		return *result.VerifyOK
	}
	return result.ExitCode == 0
}

// Execute runs the phase's install script with real-time progress reporting.
func (s *ScriptPhaseExecutor) Execute(ctx context.Context, progress func(installer.ProgressEvent)) (bool, error) {
	manifest := s.cache.GetManifest()
	if manifest == nil {
		return false, fmt.Errorf("no manifest available — cannot run %s", s.phase)
	}

	ps, ok := manifest.Phases[string(s.phase)]
	if !ok || ps.Install == nil {
		return false, fmt.Errorf("no install script for phase %s", s.phase)
	}

	progress(installer.ProgressEvent{
		Phase:   s.phase,
		Status:  installer.PhaseRunning,
		Message: fmt.Sprintf("Running %s script...", s.phase),
	})

	result, err := s.runner.Run(ctx, ps.Install, func(msg *ProtocolMessage) {
		switch msg.Type {
		case MsgProgress:
			progress(installer.ProgressEvent{
				Phase:    s.phase,
				Status:   installer.PhaseRunning,
				Message:  msg.Text,
				Progress: msg.Progress,
			})
		case MsgDetail:
			progress(installer.ProgressEvent{
				Phase:  s.phase,
				Status: installer.PhaseRunning,
				Detail: msg.Text,
			})
		case MsgError:
			progress(installer.ProgressEvent{
				Phase:   s.phase,
				Status:  installer.PhaseFailed,
				Message: msg.Text,
			})
		case MsgReboot:
			progress(installer.ProgressEvent{
				Phase:   s.phase,
				Status:  installer.PhaseRunning,
				Message: "Reboot required: " + msg.Text,
			})
		}
	})

	// Store diagnostics in metadata if available
	if result != nil && len(result.Diagnostics) > 0 {
		for k, v := range result.Diagnostics {
			log.Printf("[scriptexec] diag: %s=%s", k, v)
		}
	}

	if err != nil {
		return false, err
	}

	if result.NeedsReboot {
		return true, nil
	}

	progress(installer.ProgressEvent{
		Phase:    s.phase,
		Status:   installer.PhaseCompleted,
		Message:  fmt.Sprintf("%s completed", s.phase),
		Progress: 100,
	})

	return false, nil
}
