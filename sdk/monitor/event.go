package monitor

import "context"

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

	// A caller context to use with opentracing.
	Context() context.Context
}

// EventFactory is an interface for Event factory
type EventFactory interface {
	NewEvent(ctx context.Context, eventType EventType, entities map[string]Entity) Event
	EventFromMessage(ctx context.Context, message interface{}) (Event, error)
	FactoryName() string
}

// BaseEvent is a base struct for creating an Event
type BaseEvent struct {
	eventType EventType
	entities  map[string]Entity
	ctx context.Context
}

// NewBaseEvent creates a new BaseEvent
func NewBaseEvent(ctx context.Context,eventType EventType, entities map[string]Entity) BaseEvent {
	return BaseEvent{
		eventType: eventType,
		entities:  entities,
		ctx: ctx,
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

func(e BaseEvent) Context() context.Context {
	return e.ctx
}
