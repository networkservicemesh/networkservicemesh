package monitor

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	defaultSize            = 10
	UPDATE                 = "UPDATE"
	DELETE                 = "DELETE"
	INITIAL_STATE_TRANSFER = "INITIAL_STATE_TRANSFER"
)

type EventSupplier func(eventType string, entities map[string]Entity) Event

type Entity interface {
	GetId() string
}

type Event interface {
	Message() (interface{}, error)
	EventType() string
	Entities() map[string]Entity
}

type Recipient interface {
	SendMsg(msg interface{}) error
}

type MonitorServer interface {
	Update(entity Entity)
	Delete(entity Entity)

	AddRecipient(recipient Recipient)
	DeleteRecipient(recipient Recipient)
	MonitorEntities(stream grpc.ServerStream) error
	SendAll(event Event)
	Serve()
	Entities() map[string]Entity
}

type monitorServerImpl struct {
	eventSupplier            EventSupplier
	eventCh                  chan Event
	newMonitorRecipientCh    chan Recipient
	closedMonitorRecipientCh chan Recipient
	entities                 map[string]Entity
	recipients               []Recipient
}

func NewMonitorServer(eventSupplier EventSupplier) MonitorServer {
	return &monitorServerImpl{
		eventSupplier:            eventSupplier,
		eventCh:                  make(chan Event, defaultSize),
		newMonitorRecipientCh:    make(chan Recipient, defaultSize),
		closedMonitorRecipientCh: make(chan Recipient, defaultSize),
		entities:                 make(map[string]Entity),
		recipients:               make([]Recipient, 0, defaultSize),
	}
}

func (m monitorServerImpl) Entities() map[string]Entity {
	return m.entities
}

func (m *monitorServerImpl) Update(entity Entity) {
	m.eventCh <- m.eventSupplier(UPDATE, map[string]Entity{entity.GetId(): entity})
}

func (m *monitorServerImpl) Delete(entity Entity) {
	m.eventCh <- m.eventSupplier(DELETE, map[string]Entity{entity.GetId(): entity})
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
			initialStateTransferEvent := m.eventSupplier(INITIAL_STATE_TRANSFER, m.entities)
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
			for _, entity := range event.Entities() {
				if event.EventType() == UPDATE {
					m.entities[entity.GetId()] = entity
				}
				if event.EventType() == DELETE {
					delete(m.entities, entity.GetId())
				}
			}
			m.SendAll(event)
		}
	}
}

func (m *monitorServerImpl) SendAll(event Event) {
	m.send(event, m.recipients...)
}

func (m *monitorServerImpl) send(event Event, recipients ...Recipient) {
	for _, recipient := range recipients {
		msg, err := event.Message()
		logrus.Infof("Try to send message %v", msg)
		if err != nil {
			logrus.Errorf("An error during convertion event: %v", err)
			continue
		}
		if err := recipient.SendMsg(msg); err != nil {
			logrus.Errorf("An error during send: %v", err)
		}
	}
}
