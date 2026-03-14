package installer

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockExecutor is a test double for PhaseExecutor.
type mockExecutor struct {
	phase       Phase
	verified    bool
	executeFunc func(ctx context.Context, progress func(ProgressEvent)) (bool, error)
}

func (m *mockExecutor) Phase() Phase { return m.phase }
func (m *mockExecutor) Verify() bool { return m.verified }
func (m *mockExecutor) Execute(ctx context.Context, progress func(ProgressEvent)) (bool, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, progress)
	}
	progress(ProgressEvent{
		Phase: m.phase, Status: PhaseCompleted,
		Message: fmt.Sprintf("%s done", m.phase), Progress: 100,
	})
	return false, nil
}

func newSuccessExecutor(phase Phase) *mockExecutor {
	return &mockExecutor{phase: phase}
}

func newVerifiedExecutor(phase Phase) *mockExecutor {
	return &mockExecutor{phase: phase, verified: true}
}

func newFailExecutor(phase Phase, errMsg string) *mockExecutor {
	return &mockExecutor{
		phase: phase,
		executeFunc: func(_ context.Context, _ func(ProgressEvent)) (bool, error) {
			return false, fmt.Errorf("%s", errMsg)
		},
	}
}

func newRebootExecutor(phase Phase) *mockExecutor {
	return &mockExecutor{
		phase: phase,
		executeFunc: func(_ context.Context, progress func(ProgressEvent)) (bool, error) {
			progress(ProgressEvent{
				Phase: phase, Status: PhaseRunning,
				Message: "Reboot needed", Progress: 50,
			})
			return true, nil
		},
	}
}

func allMockExecutors() []PhaseExecutor {
	return []PhaseExecutor{
		newSuccessExecutor(PhasePrecheck),
		newSuccessExecutor(PhaseWSL),
		newSuccessExecutor(PhaseUbuntu),
		newSuccessExecutor(PhaseNode),
		newSuccessExecutor(PhaseOpenClaw),
		newSuccessExecutor(PhaseConfig),
		newSuccessExecutor(PhaseVerify),
	}
}

func TestOrchestrator_FullSuccess(t *testing.T) {
	state := NewInstallState()
	orch := NewOrchestratorWithState(allMockExecutors(), state)

	var mu sync.Mutex
	var events []ProgressEvent
	orch.OnProgress(func(evt ProgressEvent) {
		mu.Lock()
		events = append(events, evt)
		mu.Unlock()
	})

	if err := orch.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Wait for completion
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for installation")
		default:
		}
		status := orch.Status()
		if !status.Running && status.CurrentPhase == PhaseDone {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	status := orch.Status()
	if status.CurrentPhase != PhaseDone {
		t.Errorf("expected done, got %s", status.CurrentPhase)
	}
	if status.Overall != 100 {
		t.Errorf("expected 100%% overall, got %d%%", status.Overall)
	}

	mu.Lock()
	eventCount := len(events)
	mu.Unlock()
	if eventCount == 0 {
		t.Error("expected progress events")
	}
}

func TestOrchestrator_SkipVerified(t *testing.T) {
	executors := []PhaseExecutor{
		newVerifiedExecutor(PhasePrecheck),
		newVerifiedExecutor(PhaseWSL),
		newVerifiedExecutor(PhaseUbuntu),
		newSuccessExecutor(PhaseNode),
		newSuccessExecutor(PhaseOpenClaw),
		newSuccessExecutor(PhaseConfig),
		newSuccessExecutor(PhaseVerify),
	}

	state := NewInstallState()
	orch := NewOrchestratorWithState(executors, state)

	var mu sync.Mutex
	var events []ProgressEvent
	orch.OnProgress(func(evt ProgressEvent) {
		mu.Lock()
		events = append(events, evt)
		mu.Unlock()
	})

	orch.Start()
	waitDone(t, orch, 5*time.Second)

	// Check that verified phases were skipped
	mu.Lock()
	defer mu.Unlock()
	for _, evt := range events {
		if evt.Phase == PhasePrecheck && evt.Status == PhaseCompleted {
			if evt.Message == "" {
				t.Error("skip message empty")
			}
		}
	}
}

func TestOrchestrator_FailAndRetry(t *testing.T) {
	failNode := newFailExecutor(PhaseNode, "npm not found")
	executors := []PhaseExecutor{
		newVerifiedExecutor(PhasePrecheck),
		newVerifiedExecutor(PhaseWSL),
		newVerifiedExecutor(PhaseUbuntu),
		failNode,
		newSuccessExecutor(PhaseOpenClaw),
		newSuccessExecutor(PhaseConfig),
		newSuccessExecutor(PhaseVerify),
	}

	state := NewInstallState()
	orch := NewOrchestratorWithState(executors, state)

	orch.Start()
	waitPhase(t, orch, PhaseError, 5*time.Second)

	status := orch.Status()
	if status.ErrorPhase != PhaseNode {
		t.Errorf("expected error at node, got %s", status.ErrorPhase)
	}
	if status.ErrorMessage != "npm not found" {
		t.Errorf("unexpected error msg: %s", status.ErrorMessage)
	}

	// Now fix it and retry
	failNode.executeFunc = nil // make it succeed
	if err := orch.Retry(); err != nil {
		t.Fatalf("retry: %v", err)
	}

	waitDone(t, orch, 5*time.Second)
	status = orch.Status()
	if status.CurrentPhase != PhaseDone {
		t.Errorf("expected done after retry, got %s", status.CurrentPhase)
	}
}

func TestOrchestrator_RebootPause(t *testing.T) {
	executors := []PhaseExecutor{
		newSuccessExecutor(PhasePrecheck),
		newRebootExecutor(PhaseWSL),
		newSuccessExecutor(PhaseUbuntu),
		newSuccessExecutor(PhaseNode),
		newSuccessExecutor(PhaseOpenClaw),
		newSuccessExecutor(PhaseConfig),
		newSuccessExecutor(PhaseVerify),
	}

	state := NewInstallState()
	orch := NewOrchestratorWithState(executors, state)

	orch.Start()
	waitPhase(t, orch, PhaseReboot, 5*time.Second)

	status := orch.Status()
	if status.Running {
		t.Error("should not be running after reboot pause")
	}
}

func TestOrchestrator_Cancel(t *testing.T) {
	slowExec := &mockExecutor{
		phase: PhaseNode,
		executeFunc: func(ctx context.Context, _ func(ProgressEvent)) (bool, error) {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(30 * time.Second):
				return false, nil
			}
		},
	}

	executors := []PhaseExecutor{
		newVerifiedExecutor(PhasePrecheck),
		newVerifiedExecutor(PhaseWSL),
		newVerifiedExecutor(PhaseUbuntu),
		slowExec,
		newSuccessExecutor(PhaseOpenClaw),
		newSuccessExecutor(PhaseConfig),
		newSuccessExecutor(PhaseVerify),
	}

	state := NewInstallState()
	orch := NewOrchestratorWithState(executors, state)
	orch.Start()

	// Wait for node phase to start
	time.Sleep(100 * time.Millisecond)

	orch.Cancel()

	// Wait for shutdown
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for cancel")
		default:
		}
		if !orch.Status().Running {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestOrchestrator_DoubleStart(t *testing.T) {
	state := NewInstallState()
	slowExec := &mockExecutor{
		phase: PhasePrecheck,
		executeFunc: func(ctx context.Context, _ func(ProgressEvent)) (bool, error) {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(5 * time.Second):
				return false, nil
			}
		},
	}
	orch := NewOrchestratorWithState([]PhaseExecutor{slowExec}, state)
	orch.Start()
	defer func() {
		orch.Cancel()
		// Wait for goroutine to finish
		deadline := time.After(3 * time.Second)
		for {
			select {
			case <-deadline:
				return
			default:
			}
			if !orch.Status().Running {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	err := orch.Start()
	if err == nil {
		t.Error("expected error on double start")
	}
}

func TestOrchestrator_Reset(t *testing.T) {
	state := NewInstallState()
	state.CurrentPhase = PhaseNode
	orch := NewOrchestratorWithState(allMockExecutors(), state)

	if err := orch.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	status := orch.Status()
	if status.CurrentPhase != PhaseIdle {
		t.Errorf("expected idle after reset, got %s", status.CurrentPhase)
	}
}

// waitDone waits for orchestrator to reach PhaseDone.
func waitDone(t *testing.T, orch *Orchestrator, timeout time.Duration) {
	t.Helper()
	waitPhase(t, orch, PhaseDone, timeout)
}

// waitPhase waits for orchestrator to reach a specific phase (when not running).
func waitPhase(t *testing.T, orch *Orchestrator, phase Phase, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			status := orch.Status()
			t.Fatalf("timed out waiting for phase %s (current: %s, running: %v)", phase, status.CurrentPhase, status.Running)
		default:
		}
		status := orch.Status()
		if !status.Running && status.CurrentPhase == phase {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
