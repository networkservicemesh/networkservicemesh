package vppagent

import (
	"sync"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/sirupsen/logrus"
)

type resyncManager struct {
	storedDataChanges sync.Map
}

func newResyncManager(m monitor_crossconnect.MonitorServer) *resyncManager {
	result := &resyncManager{}
	m.AddRecipient(result)
	return result
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

func (m *resyncManager) storeDataChange(id string, dataChange *configurator.Config) {
	m.storedDataChanges.Store(id, dataChange)
}

//TODO: do not use pointer to poinder
func (m *resyncManager) needToResync(id string, dataChange **configurator.Config) bool {
	if value, ok := m.storedDataChanges.Load(id); ok {
		storedDataChange := value.(*configurator.Config)

		if len(storedDataChange.LinuxConfig.Interfaces) != len((*dataChange).LinuxConfig.Interfaces) {
			logrus.Info("RESYNC: Yes")
			storedDataChange.LinuxConfig = (*dataChange).LinuxConfig
			*dataChange = storedDataChange
			return true
		}

		for i, if1 := range storedDataChange.LinuxConfig.Interfaces {
			if if1.Namespace.Reference != (*dataChange).LinuxConfig.Interfaces[i].Namespace.Reference {
				logrus.Info("RESYNC: Yes")
				storedDataChange.LinuxConfig = (*dataChange).LinuxConfig
				*dataChange = storedDataChange
				return true
			}
		}
	}
	logrus.Info("RESYNC: NO")
	m.storeDataChange(id, *dataChange)
	return false
}
