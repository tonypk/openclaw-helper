package healing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tonypk/openclaw-helper/internal/playbook"
)

var errRepairFailed = errors.New("repair action failed")

// PhaseRunResult holds the outcome of running a phase script.
type PhaseRunResult struct {
	Success      bool
	ErrorLog     string
	NeedsReboot  bool
	ErrorMessage string
}

// PhaseResult is the final result of ExecutePhase, including healing history.
type PhaseResult struct {
	Success        bool
	NeedsReboot    bool
	ErrorLog       string
	ErrorMessage   string
	HealingHistory []playbook.HealingRecord
	Escalated      bool
}

// PhaseRunner executes a phase's install script. Abstracted for testing.
type PhaseRunner interface {
	RunPhase(ctx context.Context, phase string) PhaseRunResult
}

// RepairRunner executes repair scripts. Abstracted for testing.
type RepairRunner interface {
	RunRepairScript(ctx context.Context, script string, timeout int) error
}

// ManifestSyncer re-fetches the manifest and all resources. Abstracted for testing.
type ManifestSyncer interface {
	ForceSync() error
}

// EventCallback receives healing events for frontend display.
type EventCallback func(Event)

// Executor wraps phase execution with self-healing: diagnose, repair, retry.
// PhaseRunner is passed per-call (since each phase has a different executor).
type Executor struct {
	repairRunner RepairRunner
	store        *playbook.Store
	syncer       ManifestSyncer // for force-sync on playbook miss
	maxRetries   int
	onEvent      EventCallback
}

// NewExecutor creates a healing executor.
func NewExecutor(
	repairRunner RepairRunner,
	store *playbook.Store,
	syncer ManifestSyncer,
	maxRetries int,
	onEvent EventCallback,
) *Executor {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if onEvent == nil {
		onEvent = func(Event) {}
	}
	return &Executor{
		repairRunner: repairRunner,
		store:        store,
		syncer:       syncer,
		maxRetries:   maxRetries,
		onEvent:      onEvent,
	}
}

// ExecutePhase runs a phase with self-healing loop.
// The PhaseRunner is passed per-call since each phase has a different executor adapter.
func (e *Executor) ExecutePhase(ctx context.Context, phase string, runner PhaseRunner) PhaseResult {
	var history []playbook.HealingRecord

	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return PhaseResult{ErrorMessage: "cancelled"}
		default:
		}

		result := runner.RunPhase(ctx, phase)

		if result.Success {
			if len(history) > 0 {
				e.onEvent(Event{Type: EventHealResolved, Issue: history[len(history)-1].Issue})
			}
			return PhaseResult{
				Success:        true,
				HealingHistory: history,
			}
		}

		if result.NeedsReboot {
			return PhaseResult{
				NeedsReboot:    true,
				HealingHistory: history,
			}
		}

		// Find matching playbook
		pb := e.store.FindMatch(phase, result.ErrorLog)

		// If no playbook found, try force-syncing manifest (maybe one was just published)
		if pb == nil && e.syncer != nil {
			_ = e.syncer.ForceSync()
			pb = e.store.FindMatch(phase, result.ErrorLog)
		}

		if pb == nil {
			// Still no playbook -- escalate immediately
			e.onEvent(Event{Type: EventHealEscalate, Detail: "ai_chat", Issue: "unknown"})
			return PhaseResult{
				ErrorLog:       result.ErrorLog,
				ErrorMessage:   result.ErrorMessage,
				HealingHistory: history,
				Escalated:      true,
			}
		}

		e.onEvent(Event{Type: EventHealStart, Issue: pb.ID, Attempt: attempt + 1, MaxRetry: e.maxRetries})

		record := playbook.HealingRecord{
			Phase:     phase,
			Issue:     pb.ID,
			Attempts:  attempt + 1,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		// Try strategies in order
		repaired := false
		for _, strategy := range pb.Strategies {
			e.onEvent(Event{Type: EventHealStrategy, Issue: pb.ID, Strategy: strategy.Name})

			strategyOK := e.executeStrategy(ctx, strategy)
			record.StrategiesTried = append(record.StrategiesTried, strategy.Name)

			if strategyOK {
				record.ResolvedBy = strategy.Name
				repaired = true
				e.onEvent(Event{Type: EventHealRetry, Issue: pb.ID, Attempt: attempt + 1, MaxRetry: e.maxRetries})
				break
			}
		}

		history = append(history, record)

		if !repaired {
			// All strategies failed for this attempt
			if attempt == e.maxRetries {
				break
			}
			// Force-sync to check for updated playbooks before retrying
			if e.syncer != nil {
				_ = e.syncer.ForceSync()
			}
			continue
		}

		// Strategy succeeded -- retry phase (loop continues with next attempt)
	}

	// Exhausted all retries
	e.onEvent(Event{Type: EventHealEscalate, Detail: "ai_chat"})
	return PhaseResult{
		ErrorMessage:   "all repair strategies exhausted",
		HealingHistory: history,
		Escalated:      true,
	}
}

// executeStrategy runs all actions in a strategy. Returns true if all actions succeed.
func (e *Executor) executeStrategy(ctx context.Context, strategy playbook.Strategy) bool {
	for _, action := range strategy.Actions {
		switch action.Type {
		case "run":
			e.onEvent(Event{
				Type:   EventHealRepair,
				Detail: fmt.Sprintf("%s=running", action.Script),
			})
			timeout := action.Timeout
			if timeout == 0 {
				timeout = 60
			}
			err := e.repairRunner.RunRepairScript(ctx, action.Script, timeout)
			if err != nil {
				e.onEvent(Event{
					Type:   EventHealRepair,
					Detail: fmt.Sprintf("%s=failed", action.Script),
				})
				return false
			}
			e.onEvent(Event{
				Type:   EventHealRepair,
				Detail: fmt.Sprintf("%s=success", action.Script),
			})

		case "retry_phase":
			// Signal to caller that phase should be retried.
			// This is handled by the loop in ExecutePhase.
			return true

		case "reboot":
			// Reboot is handled at a higher level.
			return true

		case "set_env", "write_file":
			// TODO: implement set_env and write_file action types
			continue
		}
	}
	return true
}
