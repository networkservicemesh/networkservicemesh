package model

import (
	. "github.com/onsi/gomega"
	"testing"
)

type testListener struct {
	calls map[string]int
}

func (t *testListener) ClientConnectionUpdated(old, new *ClientConnection) {
	t.calls["ClientConnectionUpdated"]++
}

func (t *testListener) EndpointAdded(endpoint *Endpoint) {
	t.calls["EndpointAdded"]++
}

func (t *testListener) EndpointUpdated(endpoint *Endpoint) {
	t.calls["EndpointUpdated"]++
}

func (t *testListener) EndpointDeleted(endpoint *Endpoint) {
	t.calls["EndpointDeleted"]++
}

func (t *testListener) DataplaneAdded(dataplane *Dataplane) {
	t.calls["DataplaneAdded"]++
}

func (t *testListener) DataplaneDeleted(dataplane *Dataplane) {
	t.calls["DataplaneDeleted"]++
}

func (t *testListener) ClientConnectionAdded(clientConnection *ClientConnection) {
	t.calls["ClientConnectionAdded"]++
}

func (t *testListener) ClientConnectionDeleted(clientConnection *ClientConnection) {
	t.calls["ClientConnectionDeleted"]++
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

	Expect(ln.calls["EndpointAdded"]).To(Equal(1))
	Expect(ln.calls["EndpointUpdated"]).To(Equal(1))
	Expect(ln.calls["EndpointDeleted"]).To(Equal(1))

	Expect(ln.calls["DataplaneAdded"]).To(Equal(1))
	Expect(ln.calls["DataplaneDeleted"]).To(Equal(1))

	Expect(ln.calls["ClientConnectionAdded"]).To(Equal(1))
	Expect(ln.calls["ClientConnectionDeleted"]).To(Equal(1))
	Expect(ln.calls["ClientConnectionUpdated"]).To(Equal(1))
}
