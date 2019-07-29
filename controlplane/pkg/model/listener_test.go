package model

import (
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

type testListener struct {
	sync.WaitGroup
}

func (t *testListener) EndpointAdded(endpoint *Endpoint) {
	t.Done()
}

func (t *testListener) EndpointUpdated(endpoint *Endpoint) {
	t.Done()
}

func (t *testListener) EndpointDeleted(endpoint *Endpoint) {
	t.Done()
}

func (t *testListener) DataplaneAdded(dataplane *Dataplane) {
	t.Done()
}

func (t *testListener) DataplaneDeleted(dataplane *Dataplane) {
	t.Done()
}

func (t *testListener) ClientConnectionAdded(clientConnection *ClientConnection) {
	t.Done()
}

func (t *testListener) ClientConnectionDeleted(clientConnection *ClientConnection) {
	t.Done()
}

func (t *testListener) ClientConnectionUpdated(old, new *ClientConnection) {
	t.Done()
}

func TestModelListener(t *testing.T) {
	RegisterTestingT(t)

	m := NewModel()
	ln := testListener{}
	ln.Add(8)
	m.AddListener(&ln)

	m.AddEndpoint(&Endpoint{})
	m.UpdateEndpoint(&Endpoint{})
	m.DeleteEndpoint("")

	m.AddDataplane(&Dataplane{})
	m.DeleteDataplane("")

	m.AddClientConnection(&ClientConnection{})
	m.UpdateClientConnection(&ClientConnection{})
	m.DeleteClientConnection("")

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
