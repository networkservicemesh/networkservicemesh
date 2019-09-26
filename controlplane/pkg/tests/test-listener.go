package tests

import (
	"context"
	"sync"
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
	sync.RWMutex
	model.ListenerImpl
	additions int
	updates   int
	deletions int

	endpoints int

	connections   map[string]*model.ClientConnection
	textMarshaler proto.TextMarshaler
	name          string
}

func (impl *testConnectionModelListener) ClientConnectionAdded(_ context.Context, clientConnection *model.ClientConnection) {
	impl.Lock()
	defer impl.Unlock()

	impl.additions++
	impl.connections[clientConnection.GetID()] = clientConnection
	logrus.Infof("(%v, %v)Listener ClientConnectionAdded: %v", impl.name, impl.additions, clientConnection)
}

func (impl *testConnectionModelListener) ClientConnectionDeleted(_ context.Context, clientConnection *model.ClientConnection) {
	impl.Lock()
	defer impl.Unlock()

	impl.deletions++
	logrus.Infof("(%v, %v)Listener ClientConnectionDeleted: %v", impl.name, impl.deletions, clientConnection)
	delete(impl.connections, clientConnection.GetID())
}

func (impl *testConnectionModelListener) ClientConnectionUpdated(_ context.Context, old, new *model.ClientConnection) {
	impl.Lock()
	defer impl.Unlock()

	impl.updates++
	impl.connections[new.GetID()] = new
	logrus.Infof("(%v, %v)Listener ClientConnectionUpdated: %s %v", impl.name, impl.updates, new.GetID(), impl.textMarshaler.Text(new.Xcon))
}

func (impl *testConnectionModelListener) EndpointAdded(_ context.Context, endpoint *model.Endpoint) {
	impl.Lock()
	defer impl.Unlock()

	impl.endpoints++
}
func (impl *testConnectionModelListener) EndpointDeleted(_ context.Context, endpoint *model.Endpoint) {
	impl.Lock()
	defer impl.Unlock()

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
		logrus.Warnf("(%v) Waiting for additions: %d to match %d", impl.name, impl.additions, count)
	}
}
func (impl *testConnectionModelListener) WaitUpdate(count int, duration time.Duration, t *testing.T) {
	st := time.Now()
	for {
		<-time.After(stepTimeout)
		impl.RLock()
		if impl.updates == count {
			impl.RUnlock()
			break
		}
		impl.RUnlock()

		if time.Since(st) > duration {
			t.Fatalf("Failed to wait for add events.. %d timeout happened...", count)
			break
		}
		logrus.Warnf("(%v) Waiting for updates: %d to match %d", impl.name, impl.updates, count)
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
		logrus.Warnf("(%v) Waiting for deletions: %d to match %d", impl.name, impl.deletions, count)
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
func newTestConnectionModelListener(name string) *testConnectionModelListener {
	return &testConnectionModelListener{
		updates:       0,
		additions:     0,
		deletions:     0,
		textMarshaler: proto.TextMarshaler{},
		connections:   map[string]*model.ClientConnection{},
		name:          name,
	}
}
