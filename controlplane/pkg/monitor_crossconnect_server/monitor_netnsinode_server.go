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
}

func NewMonitorNetNsInodeServer(crossConnectServer MonitorCrossConnectServer) *MonitorNetNsInodeServer {
	rv := &MonitorNetNsInodeServer{
		crossConnectServer:  crossConnectServer,
		crossConnects:       make(map[string]*crossconnect.CrossConnect),
		crossConnectEventCh: make(chan *crossconnect.CrossConnectEvent, 10),
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
	for k, v := range event.CrossConnects {
		vCopy := *v
		crossConnectsCopy[k] = &vCopy
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
	liveInodes, err := fs.GetAllNetNs()
	if err != nil {
		return err
	}
	inodesSet := NewInodeSet(liveInodes)

	for _, xcon := range m.crossConnects {
		if conn := xcon.GetLocalSource(); conn != nil {
			if err := m.checkConnectionLiveness(xcon, conn, inodesSet); err != nil {
				return err
			}
		}
		if conn := xcon.GetLocalDestination(); conn != nil {
			if err := m.checkConnectionLiveness(xcon, conn, inodesSet); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *MonitorNetNsInodeServer) checkConnectionLiveness(xcon *crossconnect.CrossConnect, conn *connection.Connection,
	inodeSet *InodeSet) error {
	inode, err := strconv.ParseUint(conn.GetMechanism().GetNetNsInode(), 10, 64)
	if err != nil {
		return err
	}

	if !inodeSet.Contains(inode) && conn.State == connection.State_UP {
		logrus.Infof("Connection is down")
		conn.State = connection.State_DOWN
		m.crossConnectServer.UpdateCrossConnect(xcon)
	}

	return nil
}

func (m *MonitorNetNsInodeServer) handleEvent(event *crossconnect.CrossConnectEvent) {
	switch event.Type {
	case crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER:
		m.crossConnects = map[string]*crossconnect.CrossConnect{}
		fallthrough
	case crossconnect.CrossConnectEventType_UPDATE:
		for _, xcon := range event.GetCrossConnects() {
			m.crossConnects[xcon.GetId()] = xcon
		}
		break
	case crossconnect.CrossConnectEventType_DELETE:
		for _, xcon := range event.GetCrossConnects() {
			delete(m.crossConnects, xcon.GetId())
		}
		break
	}
}
