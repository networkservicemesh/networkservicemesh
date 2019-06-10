package model

import (
	. "github.com/onsi/gomega"
	"sync"
	"testing"
	"time"
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

	_ = m.AddEndpoint(&Endpoint{})
	m.AddOrUpdateEndpoint(&Endpoint{})
	_ = m.DeleteEndpoint("")

	_ = m.AddDataplane(&Dataplane{})
	_ = m.DeleteDataplane("")

	editor, _ := m.AddClientConnection("", ClientConnectionRequesting, &ClientConnection{})
	_ = m.CommitClientConnectionChanges(editor)
	_ = m.DeleteClientConnection("")

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
