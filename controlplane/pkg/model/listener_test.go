package model

import (
	. "github.com/onsi/gomega"
	"sync"
	"testing"
	"time"
)

type testListener struct {
	sync.RWMutex
	calls map[string]int
}

func (t *testListener) WaitForValueEqual(key string, value int, timeout time.Duration) bool {
	st := time.Now()

	for ; ; <-time.After(10 * time.Millisecond) {
		if t.calls[key] >= value {
			return true
		}
		if time.Since(st) > timeout {
			return false
		}
	}
}

func (t *testListener) incKey(key string) {
	t.Lock()
	defer t.Unlock()

	t.calls[key]++
}

func (t *testListener) EndpointAdded(endpoint *Endpoint) {
	t.incKey("EndpointAdded")
}

func (t *testListener) EndpointUpdated(endpoint *Endpoint) {
	t.incKey("EndpointUpdated")
}

func (t *testListener) EndpointDeleted(endpoint *Endpoint) {
	t.incKey("EndpointDeleted")
}

func (t *testListener) DataplaneAdded(dataplane *Dataplane) {
	t.incKey("DataplaneAdded")
}

func (t *testListener) DataplaneDeleted(dataplane *Dataplane) {
	t.incKey("DataplaneDeleted")
}

func (t *testListener) ClientConnectionAdded(clientConnection *ClientConnection) {
	t.incKey("ClientConnectionAdded")
}

func (t *testListener) ClientConnectionDeleted(clientConnection *ClientConnection) {
	t.incKey("ClientConnectionDeleted")
}

func (t *testListener) ClientConnectionUpdated(old, new *ClientConnection) {
	t.incKey("ClientConnectionUpdated")
}

func TestModelListener(t *testing.T) {
	RegisterTestingT(t)

	m := NewModel()
	ln := testListener{calls: map[string]int{}}
	m.AddListener(&ln)

	m.AddEndpoint(&Endpoint{})
	m.UpdateEndpoint(&Endpoint{})
	m.DeleteEndpoint("")

	m.AddDataplane(&Dataplane{})
	m.DeleteDataplane("")

	m.AddClientConnection(&ClientConnection{})
	m.UpdateClientConnection(&ClientConnection{})
	m.DeleteClientConnection("")

	timeout := 5 * time.Second
	Expect(ln.WaitForValueEqual("EndpointAdded", 1, timeout)).To(BeTrue())
	Expect(ln.WaitForValueEqual("EndpointUpdated", 1, timeout)).To(BeTrue())
	Expect(ln.WaitForValueEqual("EndpointDeleted", 1, timeout)).To(BeTrue())

	Expect(ln.WaitForValueEqual("DataplaneAdded", 1, timeout)).To(BeTrue())
	Expect(ln.WaitForValueEqual("DataplaneDeleted", 1, timeout)).To(BeTrue())

	Expect(ln.WaitForValueEqual("ClientConnectionAdded", 1, timeout)).To(BeTrue())
	Expect(ln.WaitForValueEqual("ClientConnectionDeleted", 1, timeout)).To(BeTrue())
	Expect(ln.WaitForValueEqual("ClientConnectionUpdated", 1, timeout)).To(BeTrue())
}
