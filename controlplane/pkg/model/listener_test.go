// +build unit_test

package model

import (
	"context"
	"sync"
	"testing"
	"time"
)

type testListener struct {
	sync.WaitGroup
}

func (t *testListener) EndpointAdded(ctx context.Context, endpoint *Endpoint) {
	t.Done()
}

func (t *testListener) EndpointUpdated(ctx context.Context, endpoint *Endpoint) {
	t.Done()
}

func (t *testListener) EndpointDeleted(ctx context.Context, endpoint *Endpoint) {
	t.Done()
}

func (t *testListener) ForwarderAdded(ctx context.Context, forwarder *Forwarder) {
	t.Done()
}

func (t *testListener) ForwarderDeleted(ctx context.Context, forwarder *Forwarder) {
	t.Done()
}

func (t *testListener) ClientConnectionAdded(ctx context.Context, clientConnection *ClientConnection) {
	t.Done()
}

func (t *testListener) ClientConnectionDeleted(ctx context.Context, clientConnection *ClientConnection) {
	t.Done()
}

func (t *testListener) ClientConnectionUpdated(ctx context.Context, old, new *ClientConnection) {
	t.Done()
}

func TestModelListener(t *testing.T) {
	m := NewModel()
	ln := testListener{}
	ln.Add(8)
	m.AddListener(&ln)

	m.AddEndpoint(context.Background(), &Endpoint{})
	m.UpdateEndpoint(context.Background(), &Endpoint{})
	m.DeleteEndpoint(context.Background(), "")

	m.AddForwarder(context.Background(), &Forwarder{})
	m.DeleteForwarder(context.Background(), "")

	m.AddClientConnection(context.Background(), &ClientConnection{})
	m.UpdateClientConnection(context.Background(), &ClientConnection{})
	m.DeleteClientConnection(context.Background(), "")

	doneCh := make(chan struct{})
	go func() {
		ln.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		return
	case <-time.After(5 * time.Second):
		t.Fatal("not all listeners have been emitted")
	}
}
