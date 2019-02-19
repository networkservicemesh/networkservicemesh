package tests

import (
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

type testConnectionModelListener struct {
	model.ModelListenerImpl
	additions int
	updates   int
	deletions int

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

func (impl *testConnectionModelListener) ClientConnectionUpdated(clientConnection *model.ClientConnection) {
	impl.updates++
	impl.connections[clientConnection.GetId()] = clientConnection
	logrus.Infof("ClientConnectionUpdated: %s %v", clientConnection.GetId(), impl.textMarshaler.Text(clientConnection.Xcon))
}

func (impl *testConnectionModelListener) WaitAdd(count int, duration time.Duration, t *testing.T) {
	st := time.Now()
	for {
		<-time.Tick(1 * time.Second)
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
		<-time.Tick(1 * time.Second)
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
		<-time.Tick(1 * time.Second)
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
func newTestConnectionModelListener() *testConnectionModelListener {
	return &testConnectionModelListener{
		updates:       0,
		additions:     0,
		deletions:     0,
		textMarshaler: proto.TextMarshaler{},
		connections:   map[string]*model.ClientConnection{},
	}
}
