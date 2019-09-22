package crossconnect

import (
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
)

// Event is a monitor.Event for crossconnect GRPC API
type Event struct {
	monitor.BaseEvent

	Statistics map[string]*crossconnect.Metrics
}

// Message converts Event to CrossConnectEvent
func (e *Event) Message() (interface{}, error) {
	eventType, err := eventTypeToXconEventType(e.EventType())
	if err != nil {
		return nil, err
	}

	xcons, err := xconsFromEntities(e.Entities())
	if err != nil {
		return nil, err
	}

	return &crossconnect.CrossConnectEvent{
		Type:          eventType,
		CrossConnects: xcons,
		Metrics:       e.Statistics,
	}, nil
}

type eventFactory struct {
}

func (m *eventFactory) NewEvent(eventType monitor.EventType, entities map[string]monitor.Entity) monitor.Event {
	return &Event{
		BaseEvent:  monitor.NewBaseEvent(eventType, entities),
		Statistics: map[string]*crossconnect.Metrics{},
	}
}

func (m *eventFactory) EventFromMessage(message interface{}) (monitor.Event, error) {
	xconEvent, ok := message.(*crossconnect.CrossConnectEvent)
	if !ok {
		return nil, fmt.Errorf("unable to cast %v to CrossConnectEvent", message)
	}

	eventType, err := xconEventTypeToEventType(xconEvent.GetType())
	if err != nil {
		return nil, err
	}

	entities := entitiesFromXcons(xconEvent.CrossConnects)

	return &Event{
		BaseEvent:  monitor.NewBaseEvent(eventType, entities),
		Statistics: xconEvent.Metrics,
	}, nil
}

func eventTypeToXconEventType(eventType monitor.EventType) (crossconnect.CrossConnectEventType, error) {
	switch eventType {
	case monitor.EventTypeInitialStateTransfer:
		return crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, nil
	case monitor.EventTypeUpdate:
		return crossconnect.CrossConnectEventType_UPDATE, nil
	case monitor.EventTypeDelete:
		return crossconnect.CrossConnectEventType_DELETE, nil
	default:
		return 0, fmt.Errorf("unable to cast %v to CrossConnectEventType", eventType)
	}
}

func xconEventTypeToEventType(connectionEventType crossconnect.CrossConnectEventType) (monitor.EventType, error) {
	switch connectionEventType {
	case crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER:
		return monitor.EventTypeInitialStateTransfer, nil
	case crossconnect.CrossConnectEventType_UPDATE:
		return monitor.EventTypeUpdate, nil
	case crossconnect.CrossConnectEventType_DELETE:
		return monitor.EventTypeDelete, nil
	default:
		return "", fmt.Errorf("unable to cast %v to monitor.EventType", connectionEventType)
	}
}

func xconsFromEntities(entities map[string]monitor.Entity) (map[string]*crossconnect.CrossConnect, error) {
	xcons := map[string]*crossconnect.CrossConnect{}

	for k, v := range entities {
		if conn, ok := v.(*crossconnect.CrossConnect); ok {
			xcons[k] = conn
		} else {
			return nil, fmt.Errorf("unable to cast Entity to CrossConnect")
		}
	}

	return xcons, nil
}

func entitiesFromXcons(xcons map[string]*crossconnect.CrossConnect) map[string]monitor.Entity {
	entities := map[string]monitor.Entity{}

	for k, v := range xcons {
		entities[k] = v
	}

	return entities
}
