package installer

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PhaseExecutor executes a single installation phase.
type PhaseExecutor interface {
	Phase() Phase
	// Execute runs the phase. It should report progress via the callback.
	// Returns true if a reboot is required before continuing.
	Execute(ctx context.Context, progress func(ProgressEvent)) (needsReboot bool, err error)
	// Verify checks if this phase was already completed (for idempotency).
	Verify() bool
}

// ProgressCallback is called whenever installation progress changes.
type ProgressCallback func(ProgressEvent)

// Orchestrator manages the installation state machine.
type Orchestrator struct {
	mu        sync.RWMutex
	state     *InstallState
	executors map[Phase]PhaseExecutor
	callbacks []ProgressCallback
	cancel    context.CancelFunc
	running   bool
}

// NewOrchestrator creates a new orchestrator, optionally resuming from saved state.
func NewOrchestrator(executors []PhaseExecutor) *Orchestrator {
	execMap := make(map[Phase]PhaseExecutor, len(executors))
	for _, e := range executors {
		execMap[e.Phase()] = e
	}

	o := &Orchestrator{
		executors: execMap,
	}

	// Try to load saved state
	saved, err := LoadState()
	if err != nil {
		log.Printf("[installer] failed to load state: %v", err)
	}
	if saved != nil && saved.CurrentPhase != PhaseDone && saved.CurrentPhase != PhaseIdle {
		o.state = saved
		log.Printf("[installer] resumed from phase: %s", saved.CurrentPhase)
	} else {
		o.state = NewInstallState()
	}

	return o
}

// NewOrchestratorWithState creates an orchestrator with explicit state (for testing).
func NewOrchestratorWithState(executors []PhaseExecutor, state *InstallState) *Orchestrator {
	execMap := make(map[Phase]PhaseExecutor, len(executors))
	for _, e := range executors {
		execMap[e.Phase()] = e
	}
	return &Orchestrator{
		executors: execMap,
		state:     state,
	}
}

// OnProgress registers a callback for progress events.
func (o *Orchestrator) OnProgress(cb ProgressCallback) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.callbacks = append(o.callbacks, cb)
}

// Status returns the current installation state (read-only copy).
func (o *Orchestrator) Status() InstallStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	phases := make([]PhaseProgress, 0, len(AllPhases()))
	for _, p := range AllPhases() {
		status := o.state.PhaseResults[p.Phase]
		phases = append(phases, PhaseProgress{
			Phase:   p.Phase,
			Label:   p.Label,
			LabelZH: p.LabelZH,
			Status:  status,
		})
	}

	return InstallStatus{
		CurrentPhase: o.state.CurrentPhase,
		Phases:       phases,
		Running:      o.running,
		ErrorMessage: o.state.ErrorMessage,
		ErrorPhase:   o.state.ErrorPhase,
		StartedAt:    o.state.StartedAt,
		Overall:      o.calculateOverallLocked(),
	}
}

// InstallStatus is the public view of installation progress.
type InstallStatus struct {
	CurrentPhase Phase           `json:"current_phase"`
	Phases       []PhaseProgress `json:"phases"`
	Running      bool            `json:"running"`
	ErrorMessage string          `json:"error_message,omitempty"`
	ErrorPhase   Phase           `json:"error_phase,omitempty"`
	StartedAt    time.Time       `json:"started_at"`
	Overall      int             `json:"overall"`
}

// PhaseProgress describes one phase's progress for the frontend.
type PhaseProgress struct {
	Phase   Phase       `json:"phase"`
	Label   string      `json:"label"`
	LabelZH string     `json:"label_zh"`
	Status  PhaseStatus `json:"status"`
}

// Start begins the installation from the current phase.
func (o *Orchestrator) Start() error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("installation already running")
	}
	o.running = true

	// Determine starting phase
	startPhase := o.state.CurrentPhase
	if startPhase == PhaseIdle || startPhase == PhaseDone || startPhase == PhaseCancelled {
		o.state = NewInstallState()
		startPhase = PhasePrecheck
	}
	if startPhase == PhaseError {
		// Retry from the failed phase
		startPhase = o.state.ErrorPhase
		o.state.ErrorMessage = ""
		o.state.ErrorPhase = ""
	}
	o.state.CurrentPhase = startPhase
	o.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	o.mu.Lock()
	o.cancel = cancel
	o.mu.Unlock()

	go o.run(ctx, startPhase)
	return nil
}

// Retry retries the failed phase.
func (o *Orchestrator) Retry() error {
	o.mu.RLock()
	if o.running {
		o.mu.RUnlock()
		return fmt.Errorf("installation already running")
	}
	phase := o.state.ErrorPhase
	o.mu.RUnlock()

	if phase == "" {
		return fmt.Errorf("no failed phase to retry")
	}

	o.mu.Lock()
	o.state.CurrentPhase = phase
	o.state.PhaseResults[phase] = PhasePending
	o.state.ErrorMessage = ""
	o.state.ErrorPhase = ""
	o.running = true
	o.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	o.mu.Lock()
	o.cancel = cancel
	o.mu.Unlock()

	go o.run(ctx, phase)
	return nil
}

// Cancel stops the current installation.
func (o *Orchestrator) Cancel() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancel != nil {
		o.cancel()
	}
}

// Reset clears the state and starts fresh.
func (o *Orchestrator) Reset() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return fmt.Errorf("cannot reset while running")
	}
	o.state = NewInstallState()
	return ClearState()
}

func (o *Orchestrator) run(ctx context.Context, startPhase Phase) {
	defer func() {
		o.mu.Lock()
		o.running = false
		o.mu.Unlock()
	}()

	phase := startPhase
	for phase != PhaseDone {
		if ctx.Err() != nil {
			o.mu.Lock()
			o.state.CurrentPhase = PhaseCancelled
			o.state.Save()
			o.mu.Unlock()
			o.emit(ProgressEvent{
				Phase:   phase,
				Status:  PhaseFailed,
				Message: "Installation cancelled",
				Overall: o.calculateOverall(),
			})
			return
		}

		executor, ok := o.executors[phase]
		if !ok {
			// No executor for this phase, skip it
			o.mu.Lock()
			o.state.PhaseResults[phase] = PhaseSkipped
			o.mu.Unlock()
			phase = nextPhase(phase)
			continue
		}

		// Check if already completed (idempotent)
		if executor.Verify() {
			o.mu.Lock()
			o.state.PhaseResults[phase] = PhaseCompleted
			o.state.Save()
			o.mu.Unlock()

			o.emit(ProgressEvent{
				Phase:   phase,
				Status:  PhaseCompleted,
				Message: fmt.Sprintf("%s already completed, skipping", phase),
				Overall: o.calculateOverall(),
			})
			phase = nextPhase(phase)
			continue
		}

		// Execute phase
		o.mu.Lock()
		o.state.CurrentPhase = phase
		o.state.PhaseResults[phase] = PhaseRunning
		o.state.Save()
		o.mu.Unlock()

		o.emit(ProgressEvent{
			Phase:   phase,
			Status:  PhaseRunning,
			Message: fmt.Sprintf("Starting %s...", phase),
			Overall: o.calculateOverall(),
		})

		needsReboot, err := executor.Execute(ctx, func(evt ProgressEvent) {
			evt.Overall = o.calculateOverall()
			o.emit(evt)
		})

		if err != nil {
			o.mu.Lock()
			o.state.CurrentPhase = PhaseError
			o.state.PhaseResults[phase] = PhaseFailed
			o.state.ErrorPhase = phase
			o.state.ErrorMessage = err.Error()
			o.state.Save()
			o.mu.Unlock()

			o.emit(ProgressEvent{
				Phase:   phase,
				Status:  PhaseFailed,
				Message: err.Error(),
				Overall: o.calculateOverall(),
			})
			return
		}

		o.mu.Lock()
		o.state.PhaseResults[phase] = PhaseCompleted
		o.state.Save()
		o.mu.Unlock()

		o.emit(ProgressEvent{
			Phase:    phase,
			Status:   PhaseCompleted,
			Message:  fmt.Sprintf("%s completed", phase),
			Progress: 100,
			Overall:  o.calculateOverall(),
		})

		if needsReboot {
			o.mu.Lock()
			o.state.CurrentPhase = PhaseReboot
			o.state.Save()
			o.mu.Unlock()

			o.emit(ProgressEvent{
				Phase:   PhaseReboot,
				Status:  PhaseRunning,
				Message: "Reboot required — installation will resume after restart",
				Overall: o.calculateOverall(),
			})
			return
		}

		phase = nextPhase(phase)
	}

	// Done
	o.mu.Lock()
	o.state.CurrentPhase = PhaseDone
	o.state.Save()
	o.mu.Unlock()

	o.emit(ProgressEvent{
		Phase:    PhaseDone,
		Status:   PhaseCompleted,
		Message:  "Installation complete!",
		Progress: 100,
		Overall:  100,
	})
}

func (o *Orchestrator) emit(evt ProgressEvent) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}
	o.mu.RLock()
	cbs := make([]ProgressCallback, len(o.callbacks))
	copy(cbs, o.callbacks)
	o.mu.RUnlock()

	for _, cb := range cbs {
		cb(evt)
	}
}

func (o *Orchestrator) calculateOverall() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.calculateOverallLocked()
}

// calculateOverallLocked computes overall progress. Caller must hold at least RLock.
func (o *Orchestrator) calculateOverallLocked() int {
	phases := AllPhases()
	total := len(phases)
	if total == 0 {
		return 0
	}

	completed := 0
	for _, p := range phases {
		status := o.state.PhaseResults[p.Phase]
		if status == PhaseCompleted || status == PhaseSkipped {
			completed++
		}
	}
	return (completed * 100) / total
}
