package vppagent

import (
	"net/http"
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

func (m *resyncManager) getStoredDataChange(id string) *configurator.Config {
	d, ok := m.storedDataChanges.Load(id)
	if ok {
		return d.(*configurator.Config)
	}
	return nil
}
func (m *resyncManager) getAllDataChanges(exclude string) []*configurator.Config {
	result := []*configurator.Config{}
	m.storedDataChanges.Range(func(k, v interface{}) bool {
		storedDataChange := v.(*configurator.Config)
		if id, ok := k.(string); ok && id != exclude {
			result = append(result, storedDataChange)
		}
		return true
	})
	return result
}

//TODO: do not use pointer to poinder
func (m *resyncManager) needToResync(id string, dataChange **configurator.Config) bool {
	if *dataChange == nil {
		return false
	}
	if value, ok := m.storedDataChanges.Load(id); ok {
		storedDataChange := value.(*configurator.Config)

		if len(storedDataChange.LinuxConfig.Interfaces) != len((*dataChange).LinuxConfig.Interfaces) {
			logrus.Info("RESYNC: Yes")
			storedDataChange.LinuxConfig = (*dataChange).LinuxConfig
			*dataChange = storedDataChange
			return true
		}

		for i, if1 := range storedDataChange.LinuxConfig.Interfaces {
			if2 := (*dataChange).LinuxConfig.Interfaces[i]
			if if1.Namespace != if2.Namespace && (if1.Namespace == nil || if2.Namespace == nil || if1.Namespace.Reference != if2.Namespace.Reference) {
				logrus.Info("RESYNC: Yes")
				storedDataChange.LinuxConfig = (*dataChange).LinuxConfig
				*dataChange = storedDataChange
				return true
			}
		}
	}
	logrus.Info("RESYNC: NO")
	m.storedDataChanges.Store(id, *dataChange)
	return false
}

//TODO: use VPP_AGENT_ENDPOINT if all tests passed
func (m *resyncManager) downstreamResync() {
	client := http.Client{}
	r, _ := http.NewRequest("POST", "http://localhost:9191/scheduler/downstream-resync", nil)
	resp, err := client.Do(r)
	if err != nil {
		logrus.Infof("Cant do downstream. Response %v, err: %v", resp, err)
		return;
	}
	logrus.Info("invoked downstream-resync without errors")
}