package healing

// EventType identifies the kind of healing event.
type EventType string

const (
	EventHealStart    EventType = "heal_start"
	EventHealStrategy EventType = "heal_strategy"
	EventHealRepair   EventType = "heal_repair"
	EventHealRetry    EventType = "heal_retry"
	EventHealResolved EventType = "heal_resolved"
	EventHealEscalate EventType = "heal_escalate"
)

// Event represents a healing lifecycle event emitted to the frontend.
type Event struct {
	Type     EventType `json:"type"`
	Issue    string    `json:"issue,omitempty"`
	Strategy string    `json:"strategy,omitempty"`
	Detail   string    `json:"detail,omitempty"`
	Attempt  int       `json:"attempt,omitempty"`
	MaxRetry int       `json:"max_retry,omitempty"`
}
