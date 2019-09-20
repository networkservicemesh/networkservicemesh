package monitor

// EventType is an enum for event types
type EventType string

const (
	// EventTypeInitialStateTransfer is a type of the event sent to recipient when it starts monitoring
	EventTypeInitialStateTransfer EventType = "INITIAL_STATE_TRANSFER"
	// EventTypeUpdate is a type of the event sent on entity update
	EventTypeUpdate EventType = "UPDATE"
	// EventTypeDelete is a type of the event sent on entity delete
	EventTypeDelete EventType = "DELETE"
)

// Entity is an interface for event entities
type Entity interface {
	GetId() string
}

// Event is an unified interface for GRPC monitoring API event
type Event interface {
	EventType() EventType
	Entities() map[string]Entity

	Message() (interface{}, error)
}

// EventFactory is an interface for Event factory
type EventFactory interface {
	NewEvent(eventType EventType, entities map[string]Entity) Event
	EventFromMessage(message interface{}) (Event, error)
}

// BaseEvent is a base struct for creating an Event
type BaseEvent struct {
	eventType EventType
	entities  map[string]Entity
}

// NewBaseEvent creates a new BaseEvent
func NewBaseEvent(eventType EventType, entities map[string]Entity) BaseEvent {
	return BaseEvent{
		eventType: eventType,
		entities:  entities,
	}
}

// EventType returns BaseEvent eventType
func (e BaseEvent) EventType() EventType {
	return e.eventType
}

// Entities returns BaseEvent entities
func (e BaseEvent) Entities() map[string]Entity {
	return e.entities
}
