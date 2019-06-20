package vppagent

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
)

type resyncMonitor struct {
	kvSchedulerClient *kvSchedulerClient
}

func newResyncManager(m monitor_crossconnect.MonitorServer, vppEndpoint string) (*resyncMonitor, error) {
	kvSchedulerClient, err := newKVSchedulerClient(vppEndpoint)

	if err != nil {
		return nil, err
	}

	result := &resyncMonitor{
		kvSchedulerClient: kvSchedulerClient,
	}

	m.AddRecipient(result)

	return result, nil
}

//SendMsg - takes a message from monitor server and check is interface down
func (m *resyncMonitor) SendMsg(msg interface{}) error {
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

func (m *resyncMonitor) checkConnection(conn *connection.Connection) {
	if conn == nil {
		return
	}
	if conn.State != connection.State_DOWN {
		return
	}
	m.kvSchedulerClient.downstreamResync()
}