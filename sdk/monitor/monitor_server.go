package monitor

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	defaultSize = 10
	stackName   = "stackName"
)

// Recipient is an unified interface for receiving stream
type Recipient interface {
	SendMsg(msg interface{}) error
}

// Server is an unified interface for GRPC monitoring API server
type Server interface {
	Update(ctx context.Context, entity Entity)
	Delete(ctx context.Context, entity Entity)

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

// WithStack -
//   Wraps 'parent' in a new Context that has the stack trace
//   using Context.Value(...) and returns the result.
//   Note: any previously existing value will be overwritten.
//
func withStack(parent context.Context, stack string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, stackName, stack)
}

// Stack - Return a workspace name
func stack(ctx context.Context) string {
	value := ctx.Value(stackName)
	if value == nil {
		return ""
	}
	return value.(string)
}

// Update sends EventTypeUpdate event for entity to all server recipients
func (s *server) Update(ctx context.Context, entity Entity) {
	ctx = withStack(ctx, string(debug.Stack()))
	s.eventCh <- s.eventFactory.NewEvent(ctx, EventTypeUpdate, map[string]Entity{entity.GetId(): entity})
}

// Delete sends EventTypeDelete event for entity to all server recipients
func (s *server) Delete(ctx context.Context, entity Entity) {
	ctx = withStack(ctx, string(debug.Stack()))
	s.eventCh <- s.eventFactory.NewEvent(ctx, EventTypeDelete, map[string]Entity{entity.GetId(): entity})
}

// AddRecipient adds server recipient
func (s *server) AddRecipient(recipient Recipient) {
	logrus.Infof("MonitorServerImpl.AddRecipient: %v-%v", s.eventFactory.FactoryName(), recipient)
	s.newMonitorRecipientCh <- recipient
}

// DeleteRecipient deletes server recipient
func (s *server) DeleteRecipient(recipient Recipient) {
	logrus.Infof("MonitorServerImpl.DeleteRecipient: %v-%v", s.eventFactory.FactoryName(), recipient)
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
	logrus.Infof("%v - Serve starting...", s.eventFactory.FactoryName())
	for {
		select {
		case newRecipient := <-s.newMonitorRecipientCh:
			initialStateTransferEvent := s.eventFactory.NewEvent(context.Background(), EventTypeInitialStateTransfer, s.entities)
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
			logrus.Infof("%v-New event: %v", s.eventFactory.FactoryName(), event)
			for _, entity := range event.Entities() {
				if event.EventType() == EventTypeUpdate {
					s.sendTrace(event, fmt.Sprintf("%v-send-update", s.eventFactory.FactoryName()))
					s.entities[entity.GetId()] = entity
				}
				if event.EventType() == EventTypeDelete {
					s.sendTrace(event, fmt.Sprintf("%v-send-delete", s.eventFactory.FactoryName()))
					delete(s.entities, entity.GetId())
				}
			}
			s.SendAll(event)
		}
	}
}

func (s *server) sendTrace(event Event, operation string) {
	if tools.IsOpentracingEnabled() && event.Context() != nil {
		span, _ := opentracing.StartSpanFromContext(event.Context(), operation)
		span.LogFields(log.Object("eventType", event.EventType()))
		msg, err := event.Message()
		span.LogFields(log.Object("msg", msg))
		if err != nil {
			span.LogFields(log.Error(err))
		}
		if s := stack(event.Context()); s != "" {
			span.LogFields(log.String("stack", s))
		}
		span.Finish()
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
