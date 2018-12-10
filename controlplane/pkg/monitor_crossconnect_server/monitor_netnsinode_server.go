package monitor_crossconnect_server

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/utils/fs"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type MonitorNetNsInodeServer struct {
	MonitorRecipient
	crossConnectServer  MonitorCrossConnectServer
	crossConnects       map[string]*crossconnect.CrossConnect
	crossConnectEventCh chan *crossconnect.CrossConnectEvent
	crossConnectClose   func(crossConnect *crossconnect.CrossConnect) error
}

func NewMonitorNetNsInodeServer(crossConnectServer MonitorCrossConnectServer,
	crossConnectClose func(crossConnect *crossconnect.CrossConnect) error) *MonitorNetNsInodeServer {
	rv := &MonitorNetNsInodeServer{
		crossConnectServer:  crossConnectServer,
		crossConnects:       make(map[string]*crossconnect.CrossConnect),
		crossConnectEventCh: make(chan *crossconnect.CrossConnectEvent, 10),
		crossConnectClose:   crossConnectClose,
	}
	crossConnectServer.AddMonitorRecipient(rv)
	go rv.MonitorNetNsInode()
	return rv
}

func (m *MonitorNetNsInodeServer) Send(event *crossconnect.CrossConnectEvent) error {
	m.crossConnectEventCh <- copyEvent(event)
	return nil
}

func copyEvent(event *crossconnect.CrossConnectEvent) *crossconnect.CrossConnectEvent {
	crossConnectsCopy := map[string]*crossconnect.CrossConnect{}
	logrus.Infof("Coping event type: %v", event.Type)
	if len(event.CrossConnects) != 0 {
		for k, v := range event.CrossConnects {
			logrus.Infof("key: %v, value: %v", k, v)
			vCopy := *v
			crossConnectsCopy[k] = &vCopy
		}
	}

	return &crossconnect.CrossConnectEvent{
		Type:          event.Type,
		CrossConnects: crossConnectsCopy,
	}
}

func (m *MonitorNetNsInodeServer) MonitorNetNsInode() {
	for {
		select {
		case <-time.Tick(3 * time.Second):
			if err := m.checkCrossConnectLiveness(); err != nil {
				logrus.Error(err)
			}
		case event := <-m.crossConnectEventCh:
			m.handleEvent(event)
		}
	}
}

func (m *MonitorNetNsInodeServer) checkCrossConnectLiveness() error {
	logrus.Info("Checking liveness of crossconnects...")
	liveInodes, err := fs.GetAllNetNs()
	if err != nil {
		return err
	}

	inodesSet := NewInodeSet(liveInodes)
	for _, xcon := range m.crossConnects {
		srcInode, dstInode, err := getInodes(xcon)
		if err != nil {
			return err
		}
		if !inodesSet.Contains(srcInode) || !inodesSet.Contains(dstInode) {
			m.crossConnectClose(xcon)
		}
	}

	return nil
}

func (m *MonitorNetNsInodeServer) handleEvent(event *crossconnect.CrossConnectEvent) {
	if event.Type == crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER {
		m.crossConnects = event.GetCrossConnects()
	}

	for _, xcon := range event.GetCrossConnects() {
		if event.GetType() == crossconnect.CrossConnectEventType_UPDATE {
			m.crossConnects[xcon.GetId()] = xcon
		}
		if event.GetType() == crossconnect.CrossConnectEventType_DELETE {
			delete(m.crossConnects, xcon.GetId())
		}
	}
}

func getSourceMechanismParameters(crossConnect *crossconnect.CrossConnect) map[string]string {
	if parameters := crossConnect.GetLocalSource().GetMechanism().GetParameters(); parameters != nil {
		return parameters
	}
	return crossConnect.GetRemoteSource().GetMechanism().GetParameters()
}

func getDestinationMechanismParameters(crossConnect *crossconnect.CrossConnect) map[string]string {
	if parameters := crossConnect.GetLocalDestination().GetMechanism().GetParameters(); parameters != nil {
		return parameters
	}
	return crossConnect.GetRemoteDestination().GetMechanism().GetParameters()
}

func getInode(parameters map[string]string) (uint64, error) {
	return strconv.ParseUint(parameters[connection.NetNsInodeKey], 10, 64)
}

func getInodes(crossConnect *crossconnect.CrossConnect) (uint64, uint64, error) {
	srcParameters := getSourceMechanismParameters(crossConnect)
	srcInode, err := getInode(srcParameters)
	if err != nil {
		return 0, 0, err
	}

	dstParameters := getDestinationMechanismParameters(crossConnect)
	dstInode, err := getInode(dstParameters)
	if err != nil {
		return 0, 0, err
	}

	return srcInode, dstInode, nil
}
