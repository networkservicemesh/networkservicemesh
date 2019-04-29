package crossconnect_monitor

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type CrossConnectEvent struct {
	monitor.EventImpl
	statistics map[string]*crossconnect.Metrics
}

func CreateCrossConnectEvent(eventType string, entities map[string]monitor.Entity) monitor.Event {
	return CrossConnectEvent{
		EventImpl:  monitor.CrateEventImpl(eventType, entities),
		statistics: make(map[string]*crossconnect.Metrics),
	}
}

func (event CrossConnectEvent) Message() (interface{}, error) {
	eventType, err := convertType(event.EventType())
	if err != nil {
		return nil, err
	}
	xcons, err := convertEntities(event.Entities())
	if err != nil {
		return nil, err
	}
	return &crossconnect.CrossConnectEvent{
		Type:          eventType,
		CrossConnects: xcons,
		Metrics:       event.statistics,
	}, nil
}

func convertType(eventType string) (crossconnect.CrossConnectEventType, error) {
	switch eventType {
	case monitor.UPDATE:
		return crossconnect.CrossConnectEventType_UPDATE, nil
	case monitor.DELETE:
		return crossconnect.CrossConnectEventType_DELETE, nil
	case monitor.INITIAL_STATE_TRANSFER:
		return crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER, nil
	default:
		return 0, fmt.Errorf("unable to cast type %v to CrossConnectEventType", eventType)
	}
}

func convertEntities(entities map[string]monitor.Entity) (map[string]*crossconnect.CrossConnect, error) {
	rv := map[string]*crossconnect.CrossConnect{}

	for k, v := range entities {
		if xcon, ok := v.(*crossconnect.CrossConnect); ok {
			rv[k] = xcon
		} else {
			return nil, fmt.Errorf("unable to cast Entity to CrossConnect")
		}
	}
	return rv, nil
}
