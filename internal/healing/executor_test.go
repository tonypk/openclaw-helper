package healing

import (
	"context"
	"testing"

	"github.com/tonypk/openclaw-helper/internal/playbook"
)

// mockPhaseRunner simulates script execution for testing.
type mockPhaseRunner struct {
	results []PhaseRunResult // returns results in order
	callIdx int
}

func (m *mockPhaseRunner) RunPhase(ctx context.Context, phase string) PhaseRunResult {
	if m.callIdx >= len(m.results) {
		return PhaseRunResult{Success: true}
	}
	r := m.results[m.callIdx]
	m.callIdx++
	return r
}

// mockRepairRunner simulates repair script execution.
type mockRepairRunner struct {
	successes map[string]bool // script name -> success
}

func (m *mockRepairRunner) RunRepairScript(ctx context.Context, script string, timeout int) error {
	if m.successes[script] {
		return nil
	}
	return errRepairFailed
}

func TestExecutor_SuccessOnFirstTry(t *testing.T) {
	runner := &mockPhaseRunner{results: []PhaseRunResult{{Success: true}}}
	store := playbook.NewStore()
	events := make([]Event, 0)

	ex := NewExecutor(&mockRepairRunner{}, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if !result.Success {
		t.Fatal("expected success")
	}
	if len(events) != 0 {
		t.Fatalf("expected no healing events, got %d", len(events))
	}
}

func TestExecutor_FailThenHealThenSucceed(t *testing.T) {
	runner := &mockPhaseRunner{
		results: []PhaseRunResult{
			{Success: false, ErrorLog: "npm ERR! network ETIMEDOUT"},
			{Success: true},
		},
	}
	store := playbook.NewStore()
	store.LoadFromJSON([]byte(`{"playbooks":[{
		"id":"npm_network_timeout",
		"match":{"phase":"openclaw","error_patterns":["ETIMEDOUT"]},
		"severity":"critical",
		"strategies":[{"name":"mirror","risk":"low","actions":[{"type":"run","script":"repair/npm-mirror.sh","timeout":30},{"type":"retry_phase"}]}]
	}]}`))

	repair := &mockRepairRunner{successes: map[string]bool{"repair/npm-mirror.sh": true}}
	events := make([]Event, 0)

	ex := NewExecutor(repair, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if !result.Success {
		t.Fatal("expected success after healing")
	}
	if len(result.HealingHistory) != 1 {
		t.Fatalf("expected 1 healing record, got %d", len(result.HealingHistory))
	}
	if result.HealingHistory[0].ResolvedBy != "mirror" {
		t.Fatalf("expected resolved by mirror, got %s", result.HealingHistory[0].ResolvedBy)
	}
}

func TestExecutor_AllStrategiesFail_Escalate(t *testing.T) {
	runner := &mockPhaseRunner{
		results: []PhaseRunResult{
			{Success: false, ErrorLog: "ETIMEDOUT"},
			{Success: false, ErrorLog: "ETIMEDOUT"},
			{Success: false, ErrorLog: "ETIMEDOUT"},
			{Success: false, ErrorLog: "ETIMEDOUT"},
		},
	}
	store := playbook.NewStore()
	store.LoadFromJSON([]byte(`{"playbooks":[{
		"id":"test",
		"match":{"phase":"*","error_patterns":["ETIMEDOUT"]},
		"severity":"critical",
		"strategies":[{"name":"s1","risk":"low","actions":[{"type":"run","script":"s1.sh","timeout":10},{"type":"retry_phase"}]}]
	}]}`))

	repair := &mockRepairRunner{successes: map[string]bool{}}
	events := make([]Event, 0)

	ex := NewExecutor(repair, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if result.Success {
		t.Fatal("expected failure after exhausting retries")
	}
	// Check escalation event was emitted
	hasEscalate := false
	for _, e := range events {
		if e.Type == EventHealEscalate {
			hasEscalate = true
		}
	}
	if !hasEscalate {
		t.Fatal("expected escalate event")
	}
}

func TestExecutor_NoPlaybook_Escalate(t *testing.T) {
	runner := &mockPhaseRunner{
		results: []PhaseRunResult{{Success: false, ErrorLog: "unknown error xyz"}},
	}
	store := playbook.NewStore()
	events := make([]Event, 0)

	ex := NewExecutor(&mockRepairRunner{}, store, nil, 3, func(e Event) { events = append(events, e) })
	result := ex.ExecutePhase(context.Background(), "openclaw", runner)

	if result.Success {
		t.Fatal("expected failure when no playbook matches")
	}
}
