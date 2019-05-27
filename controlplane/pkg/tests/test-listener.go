package tests

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

const (
	stepTimeout = time.Millisecond * 50
)

type testConnectionModelListener struct {
	model.ModelListenerImpl
	additions int
	updates   int
	deletions int

	endpoints int

	connections   map[string]*model.ClientConnection
	textMarshaler proto.TextMarshaler
}

func (impl *testConnectionModelListener) ClientConnectionAdded(clientConnection *model.ClientConnection) {
	impl.additions++
	impl.connections[clientConnection.GetId()] = clientConnection
	logrus.Infof("ClientConnectionAdded: %v", clientConnection)
}

func (impl *testConnectionModelListener) ClientConnectionDeleted(clientConnection *model.ClientConnection) {
	impl.deletions++
	logrus.Infof("ClientConnectionDeleted: %v", clientConnection)
	delete(impl.connections, clientConnection.GetId())
}

func (impl *testConnectionModelListener) ClientConnectionUpdated(old, new *model.ClientConnection) {
	impl.updates++
	impl.connections[new.GetId()] = new
	logrus.Infof("ClientConnectionUpdated: %s %v", new.GetId(), impl.textMarshaler.Text(new.Xcon))
}

func (impl *testConnectionModelListener) EndpointAdded(endpoint *model.Endpoint) {
	impl.endpoints++
}
func (impl *testConnectionModelListener) EndpointDeleted(endpoint *model.Endpoint) {
	impl.endpoints--
}

func (impl *testConnectionModelListener) WaitAdd(count int, duration time.Duration, t *testing.T) {
	st := time.Now()
	for {
		<-time.After(stepTimeout)
		if impl.additions == count {
			break
		}
		if time.Since(st) > duration {
			t.Fatalf("Failed to wait for add events.. %d timeout happened...", count)
			break
		}
		logrus.Warnf("Waiting for additions: %d to match %d", impl.additions, count)
	}
}
func (impl *testConnectionModelListener) WaitUpdate(count int, duration time.Duration, t *testing.T) {
	st := time.Now()
	for {
		<-time.After(stepTimeout)
		if impl.updates == count {
			break
		}
		if time.Since(st) > duration {
			t.Fatalf("Failed to wait for add events.. %d timeout happened...", count)
			break
		}
		logrus.Warnf("Waiting for updates: %d to match %d", impl.updates, count)
	}
}

func (impl *testConnectionModelListener) WaitDelete(count int, duration time.Duration, t *testing.T) {
	st := time.Now()
	for {
		<-time.After(stepTimeout)
		if impl.deletions == count {
			break
		}
		if time.Since(st) > duration {
			t.Fatalf("Failed to wait for add events.. %d timeout happened...", count)
			break
		}
		logrus.Warnf("Waiting for deletions: %d to match %d", impl.deletions, count)
	}
}
func (impl *testConnectionModelListener) WaitEndpoints(count int, duration time.Duration, t *testing.T) {
	st := time.Now()
	for {
		<-time.After(stepTimeout)
		if impl.endpoints == count {
			break
		}
		if time.Since(st) > duration {
			t.Fatalf("Failed to wait for add events.. %d timeout happened...", count)
			break
		}
		logrus.Warnf("Waiting for deletions: %d to match %d", impl.deletions, count)
	}
}
func newTestConnectionModelListener() *testConnectionModelListener {
	return &testConnectionModelListener{
		updates:       0,
		additions:     0,
		deletions:     0,
		textMarshaler: proto.TextMarshaler{},
		connections:   map[string]*model.ClientConnection{},
	}
}
