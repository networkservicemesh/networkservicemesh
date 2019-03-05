package monitor

import (
	"github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"strings"
)

const (
	defaultSize            = 10
	UPDATE                 = "UPDATE"
	DELETE                 = "DELETE"
	INITIAL_STATE_TRANSFER = "INITIAL_STATE_TRANSFER"
)

const (
	src_prefix = "SRC-"
	dst_prefix = "DST-"
)

type Entity interface {
	GetId() string
}

type Event struct {
	EventType string
	Entities  map[string]Entity
}

type Stat struct {
	name       string
	statistics *interfaces.InterfacesState_Interface_Statistics
}

type EventConverter interface {
	Convert(event Event) (interface{}, error)
}

type Recipient interface {
	SendMsg(msg interface{}) error
}

type MonitorServer interface {
	Update(entity Entity)
	Delete(entity Entity)

	UpdateStat(name string, statistics *interfaces.InterfacesState_Interface_Statistics)

	AddRecipient(recipient Recipient)
	DeleteRecipient(recipient Recipient)
	MonitorEntities(stream grpc.ServerStream) error

	Serve()
}

type monitorServerImpl struct {
	eventConverter           EventConverter
	eventCh                  chan Event
	statsCh                  chan Stat
	newMonitorRecipientCh    chan Recipient
	closedMonitorRecipientCh chan Recipient
	entities                 map[string]Entity
	recipients               []Recipient
	srcStats                 map[string]*interfaces.InterfacesState_Interface_Statistics
	dstStats                 map[string]*interfaces.InterfacesState_Interface_Statistics
}

func NewMonitorServer(eventConverter EventConverter) MonitorServer {
	return &monitorServerImpl{
		eventConverter:           eventConverter,
		eventCh:                  make(chan Event, defaultSize),
		statsCh:                  make(chan Stat, defaultSize),
		newMonitorRecipientCh:    make(chan Recipient, defaultSize),
		closedMonitorRecipientCh: make(chan Recipient, defaultSize),
		entities:                 make(map[string]Entity),
		recipients:               make([]Recipient, 0, defaultSize),
		srcStats:                 make(map[string]*interfaces.InterfacesState_Interface_Statistics),
		dstStats:                 make(map[string]*interfaces.InterfacesState_Interface_Statistics),
	}
}

func (m *monitorServerImpl) Update(entity Entity) {
	m.eventCh <- Event{
		EventType: UPDATE,
		Entities:  map[string]Entity{entity.GetId(): entity},
	}
}

func (m *monitorServerImpl) UpdateStat(name string, statistics *interfaces.InterfacesState_Interface_Statistics) {
	m.statsCh <- Stat{
		name:       name,
		statistics: statistics,
	}
}

func (m *monitorServerImpl) Delete(entity Entity) {
	m.eventCh <- Event{
		EventType: DELETE,
		Entities:  map[string]Entity{entity.GetId(): entity},
	}
}

func (m *monitorServerImpl) AddRecipient(recipient Recipient) {
	logrus.Infof("MonitorServerImpl.AddRecipient: %v", recipient)
	m.newMonitorRecipientCh <- recipient
}

func (m *monitorServerImpl) DeleteRecipient(recipient Recipient) {
	logrus.Infof("MonitorServerImpl.DeleteRecipient: %v", recipient)
	m.closedMonitorRecipientCh <- recipient
}

func (m *monitorServerImpl) MonitorEntities(stream grpc.ServerStream) error {
	m.AddRecipient(stream)

	// We need to wait until it will be done and do not exit
	for {
		select {
		case <-stream.Context().Done():
			m.DeleteRecipient(stream)
			return nil
		}
	}
}

func (m *monitorServerImpl) Serve() {
	logrus.Infof("Serve starting...")
	for {
		select {
		case newRecipient := <-m.newMonitorRecipientCh:
			initialStateTransferEvent := Event{
				EventType: INITIAL_STATE_TRANSFER,
				Entities:  m.entities,
			}
			m.send(initialStateTransferEvent, newRecipient)
			m.recipients = append(m.recipients, newRecipient)
		case closedRecipient := <-m.closedMonitorRecipientCh:
			for j, r := range m.recipients {
				if r == closedRecipient {
					m.recipients = append(m.recipients[:j], m.recipients[j+1:]...)
					break
				}
			}
		case event := <-m.eventCh:
			logrus.Infof("New event: %v", event)
			for _, entity := range event.Entities {
				if event.EventType == UPDATE {
					m.entities[entity.GetId()] = entity
				}
				if event.EventType == DELETE {
					delete(m.entities, entity.GetId())
				}
			}
			m.send(event, m.recipients...)
		case stat := <-m.statsCh:
			logrus.Infof("New statistics: %v", stat)
			if strings.HasPrefix(stat.name, src_prefix) {
				id := stat.name[len(src_prefix): len(stat.name)]
				m.srcStats[id] = stat.statistics
			} else if strings.HasPrefix(stat.name, dst_prefix) {
				id := stat.name[len(dst_prefix): len(stat.name)]
				m.dstStats[id] = stat.statistics
			}
		}
	}
}

func (m *monitorServerImpl) send(event Event, recipients ...Recipient) {
	for _, recipient := range recipients {
		msg, err := m.eventConverter.Convert(event)
		if err != nil {
			logrus.Errorf("Error during converting event: %v", err)
		}
		if err := recipient.SendMsg(msg); err != nil {
			logrus.Errorf("Error during send: %+v", err)
		}
	}
}
