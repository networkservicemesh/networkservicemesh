package monitor

import "errors"

type EventImpl struct {
	eventType string
	entities  map[string]Entity
}

func CrateEventImpl(eventType string, entities map[string]Entity) EventImpl {
	return EventImpl{
		eventType: eventType,
		entities:  entities,
	}
}

func (EventImpl) Message() (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (event EventImpl) EventType() string {
	return event.eventType
}

func (event EventImpl) Entities() map[string]Entity {
	return event.entities
}
