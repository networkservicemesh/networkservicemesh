package vppagent

import (
	"sync"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
)

type resyncManager struct {
	storedDataChanges sync.Map
	kvSchedulerClient *kvSchedulerClient
}

func newResyncManager(m monitor_crossconnect.MonitorServer, vppEndpoint string) (*resyncManager, error) {
	kvSchedulerClient, err := newKVSchedulerClient(vppEndpoint)
	if err != nil {
		return nil, err
	}
	result := &resyncManager{}
	m.AddRecipient(result)

	return &resyncManager{
		kvSchedulerClient: kvSchedulerClient,
	}, nil
}

//SendMsg - takes a message from monitor server and check is interface down
func (m *resyncManager) SendMsg(msg interface{}) error {
	event, ok := msg.(*crossconnect.CrossConnectEvent)
	if !ok {
		return nil
	}
	for _, xcon := range event.CrossConnects {
		m.checkConnection(xcon.GetLocalSource())
		m.checkConnection(xcon.GetLocalDestination())
	}
	return nil
}

func (m *resyncManager) checkConnection(conn *connection.Connection) {
	if conn == nil {
		return
	}
	if conn.State != connection.State_DOWN {
		return
	}
	m.storedDataChanges.Delete(conn.Id)
}

func (m *resyncManager) downstreamResync() {
	m.kvSchedulerClient.downstreamResync()
}

func (m *resyncManager) isNeedToResync(id string, dataChange *configurator.Config) bool {
	if value, ok := m.storedDataChanges.Load(id); ok {
		storedDataChange := value.(*configurator.Config)
		if len(storedDataChange.LinuxConfig.Interfaces) != len((*dataChange).LinuxConfig.Interfaces) {
			return true
		}
		for i, if1 := range storedDataChange.LinuxConfig.Interfaces {
			if2 := (*dataChange).LinuxConfig.Interfaces[i]
			if if1.Namespace != if2.Namespace && (if1.Namespace == nil || if2.Namespace == nil || if1.Namespace.Reference != if2.Namespace.Reference) {
				return true
			}
		}
	}
	m.storedDataChanges.Store(id, dataChange)
	return false
}