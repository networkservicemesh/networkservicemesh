package monitor_crossconnect_server

import (
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
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
	logrus.Info("Checking liveness of crossconnects...")
	liveInodes, err := fs.GetAllNetNs()
	if err != nil {
		return err
	}

	inodesSet := NewInodeSet(liveInodes)
	for _, xcon := range m.crossConnects {
		localInodes, err := getLocalInodes(xcon)
		if err != nil {
			return err
		}
		for _, inode := range localInodes {
			if !inodesSet.Contains(inode) {
				logrus.Infof("Closing crossconnect: %v", *xcon)
				m.crossConnectClose(xcon)
				break
			}
		}
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

func getLocalInodes(xcon *crossconnect.CrossConnect) ([]uint64, error) {
	var localInodes []uint64

	if conn := xcon.GetLocalSource(); conn != nil {
		inodeStr := conn.GetMechanism().GetNetNsInode()
		logrus.Infof("Local source: %s", inodeStr)
		if inode, err := strconv.ParseUint(inodeStr, 10, 64); err == nil {
			localInodes = append(localInodes, inode)
		} else {
			return nil, err
		}
	}

	if conn := xcon.GetLocalDestination(); conn != nil {
		inodeStr := conn.GetMechanism().GetNetNsInode()
		logrus.Infof("Local destination: %s", inodeStr)
		if inode, err := strconv.ParseUint(inodeStr, 10, 64); err == nil {
			localInodes = append(localInodes, inode)
		} else {
			return nil, err
		}
	}

	return localInodes, nil
}
