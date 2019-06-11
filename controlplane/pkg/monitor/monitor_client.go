package monitor

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// EventStream is an unified interface for blocking receiver
type EventStream interface {
	Recv() (interface{}, error)
}

// EventStreamConstructor is a type for EventStream constructor
type EventStreamConstructor func(ctx context.Context, cc *grpc.ClientConn) (EventStream, error)

// Client is an unified interface for GRPC monitoring API client
type Client interface {
	ErrorChannel() <-chan error
	EventChannel() <-chan Event

	Context() context.Context
	Close()
}

type client struct {
	eventCh <-chan Event
	errorCh <-chan error

	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient creates a new Client on given GRPC connection with given EventFactory and EventStreamConstructor
func NewClient(cc *grpc.ClientConn, eventFactory EventFactory, streamConstructor EventStreamConstructor) (Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := streamConstructor(ctx, cc)
	if err != nil {
		cancel()
		return nil, err
	}

	errorChannel := make(chan error, 1)
	eventChannel := make(chan Event, 1)
	go func() {
		defer close(eventChannel)
		defer close(errorChannel)

		for {
			message, err := stream.Recv()
			if err != nil {
				errorChannel <- err
				return
			}

			if event, err := eventFactory.EventFromMessage(message); err != nil {
				logrus.Errorf("An error during convertion event: %v", err)
			} else {
				eventChannel <- event
			}
		}
	}()

	return &client{
		errorCh: errorChannel,
		eventCh: eventChannel,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// ErrorChannel returns client errorChannel
func (c *client) ErrorChannel() <-chan error {
	return c.errorCh
}

// EventChannel returns client eventChannel
func (c *client) EventChannel() <-chan Event {
	return c.eventCh
}

// Context returns client context
func (c *client) Context() context.Context {
	return c.ctx
}

// Close cancels client context
func (c *client) Close() {
	c.cancel()
}
