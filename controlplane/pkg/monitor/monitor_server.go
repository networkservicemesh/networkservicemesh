package monitor

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	defaultSize = 10
)

// Recipient is an unified interface for receiving stream
type Recipient interface {
	SendMsg(msg interface{}) error
}

// Server is an unified interface for GRPC monitoring API server
type Server interface {
	Update(entity Entity)
	Delete(entity Entity)

	AddRecipient(recipient Recipient)
	DeleteRecipient(recipient Recipient)
	MonitorEntities(stream grpc.ServerStream)
	SendAll(event Event)
	Serve()
	Entities() map[string]Entity
}

type server struct {
	eventFactory             EventFactory
	eventCh                  chan Event
	newMonitorRecipientCh    chan Recipient
	closedMonitorRecipientCh chan Recipient
	entities                 map[string]Entity
	recipients               []Recipient
}

// NewServer creates a new Server with given EventFactory
func NewServer(eventFactory EventFactory) Server {
	return &server{
		eventFactory:             eventFactory,
		eventCh:                  make(chan Event, defaultSize),
		newMonitorRecipientCh:    make(chan Recipient, defaultSize),
		closedMonitorRecipientCh: make(chan Recipient, defaultSize),
		entities:                 make(map[string]Entity),
		recipients:               make([]Recipient, 0, defaultSize),
	}
}

// Update sends EventTypeUpdate event for entity to all server recipients
func (s *server) Update(entity Entity) {
	s.eventCh <- s.eventFactory.NewEvent(EventTypeUpdate, map[string]Entity{entity.GetId(): entity})
}

// Delete sends EventTypeDelete event for entity to all server recipients
func (s *server) Delete(entity Entity) {
	s.eventCh <- s.eventFactory.NewEvent(EventTypeDelete, map[string]Entity{entity.GetId(): entity})
}

// AddRecipient adds server recipient
func (s *server) AddRecipient(recipient Recipient) {
	logrus.Infof("MonitorServerImpl.AddRecipient: %v", recipient)
	s.newMonitorRecipientCh <- recipient
}

// DeleteRecipient deletes server recipient
func (s *server) DeleteRecipient(recipient Recipient) {
	logrus.Infof("MonitorServerImpl.DeleteRecipient: %v", recipient)
	s.closedMonitorRecipientCh <- recipient
}

// MonitorEntities adds stream as server recipient and blocks until it get closed
func (s *server) MonitorEntities(stream grpc.ServerStream) {
	s.AddRecipient(stream)
	defer s.DeleteRecipient(stream)

	// We need to wait until it will be done and do not exit
	<-stream.Context().Done()
}

// SendAll sends event to all server recipients
func (s *server) SendAll(event Event) {
	s.send(event, s.recipients...)
}

// Serve starts a main loop for server
func (s *server) Serve() {
	logrus.Infof("Serve starting...")
	for {
		select {
		case newRecipient := <-s.newMonitorRecipientCh:
			initialStateTransferEvent := s.eventFactory.NewEvent(EventTypeInitialStateTransfer, s.entities)
			s.send(initialStateTransferEvent, newRecipient)
			s.recipients = append(s.recipients, newRecipient)
		case closedRecipient := <-s.closedMonitorRecipientCh:
			for j, r := range s.recipients {
				if r == closedRecipient {
					s.recipients = append(s.recipients[:j], s.recipients[j+1:]...)
					break
				}
			}
		case event := <-s.eventCh:
			logrus.Infof("New event: %v", event)
			for _, entity := range event.Entities() {
				if event.EventType() == EventTypeUpdate {
					s.entities[entity.GetId()] = entity
				}
				if event.EventType() == EventTypeDelete {
					delete(s.entities, entity.GetId())
				}
			}
			s.SendAll(event)
		}
	}
}

// Entities returns server entities
func (s *server) Entities() map[string]Entity {
	return s.entities
}

func (s *server) send(event Event, recipients ...Recipient) {
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
