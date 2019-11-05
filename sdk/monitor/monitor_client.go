package monitor

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const clientEventCapacity = 10
const connectionStatusReportPeriod = time.Second * 15

// EventStream is an unified interface for blocking receiver
type EventStream interface {
	Recv() (interface{}, error)
	Context() context.Context
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
	eventChannel := make(chan Event, clientEventCapacity)
	go func() {
		type recvResult struct {
			err error
			msg interface{}
		}

		defer close(eventChannel)
		defer close(errorChannel)

		recvCh := make(chan recvResult)
		go func() {
			for {
				message, err := stream.Recv()
				recvCh <- recvResult{msg: message, err: err}
			}
		}()

		for {
			select {
			case r := <-recvCh:
				if r.err != nil {
					logrus.Infof("An error during recv: %v", r.err)
					errorChannel <- r.err
					return
				}
				if event, err := eventFactory.EventFromMessage(context.Background(), r.msg); err != nil {
					logrus.Errorf("An error during conversion event: %v", err)
				} else {
					eventChannel <- event
				}
			case <-stream.Context().Done():
				errorChannel <- stream.Context().Err()
				return
			case <-time.After(connectionStatusReportPeriod):
				logrus.Infof("monitor client(%v) connection status: %v", cc.Target(), cc.GetState().String())
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
