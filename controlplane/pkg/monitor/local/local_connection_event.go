package local

import (
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type event struct {
	monitor.EventImpl
}

func createEvent(eventType string, entities map[string]monitor.Entity) monitor.Event {
	return event{
		EventImpl: monitor.CrateEventImpl(eventType, entities),
	}
}

// Message converts event to local.Event
func (e event) Message() (interface{}, error) {
	eventType, err := convertType(e.EventType())
	if err != nil {
		return nil, err
	}

	connections, err := convertEntities(e.Entities())
	if err != nil {
		return nil, err
	}

	return &connection.ConnectionEvent{
		Type:        eventType,
		Connections: connections,
	}, nil
}

func convertType(eventType string) (connection.ConnectionEventType, error) {
	switch eventType {
	case monitor.UPDATE:
		return connection.ConnectionEventType_UPDATE, nil
	case monitor.DELETE:
		return connection.ConnectionEventType_DELETE, nil
	case monitor.INITIAL_STATE_TRANSFER:
		return connection.ConnectionEventType_INITIAL_STATE_TRANSFER, nil
	default:
		return 0, fmt.Errorf("unable to cast type %v to local.ConnectionEventType", eventType)
	}
}

func convertEntities(entities map[string]monitor.Entity) (map[string]*connection.Connection, error) {
	rv := map[string]*connection.Connection{}

	for k, v := range entities {
		if xcon, ok := v.(*connection.Connection); ok {
			rv[k] = xcon
		} else {
			return nil, fmt.Errorf("unable to cast Entity to local.Connection")
		}
	}
	return rv, nil
}
