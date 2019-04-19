package local_connection_monitor

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type LocalConnectionEvent struct {
	monitor.EventImpl
}

func CreateLocalConnectionEvent(eventType string, entities map[string]monitor.Entity) monitor.Event {
	return LocalConnectionEvent{
		EventImpl: monitor.CrateEventImpl(eventType, entities),
	}
}
func (c LocalConnectionEvent) Message() (interface{}, error) {
	eventType, err := convertType(c.EventType())
	if err != nil {
		return nil, err
	}

	connections, err := convertEntities(c.Entities())
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
		return 0, fmt.Errorf("unable to cast type %v to ConnectionEventType", eventType)
	}
}

func convertEntities(entities map[string]monitor.Entity) (map[string]*connection.Connection, error) {
	rv := map[string]*connection.Connection{}

	for k, v := range entities {
		if xcon, ok := v.(*connection.Connection); ok {
			rv[k] = xcon
		} else {
			return nil, fmt.Errorf("unable to cast Entity to remote.Connection")
		}
	}
	return rv, nil
}
