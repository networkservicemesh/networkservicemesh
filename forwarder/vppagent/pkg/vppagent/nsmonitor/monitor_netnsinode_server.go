package nsmonitor

import (
	"context"
	"strconv"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/utils/fs"
)

type MonitorNetNsInodeServer struct {
	forwarderHandler    func()
	crossConnectServer  monitor_crossconnect.MonitorServer
	crossConnects       map[string]*crossconnect.CrossConnect
	crossConnectEventCh chan *crossconnect.CrossConnectEvent
}

// NewMonitorNetNsInodeServer creates a new MonitorNetNsInodeServer
func CreateMonitorNetNsInodeServer(crossConnectServer monitor_crossconnect.MonitorServer, handle func()) error {
	rv := &MonitorNetNsInodeServer{
		crossConnectServer:  crossConnectServer,
		crossConnects:       make(map[string]*crossconnect.CrossConnect),
		crossConnectEventCh: make(chan *crossconnect.CrossConnectEvent, 10),
		forwarderHandler:    handle,
	}

	crossConnectServer.AddRecipient(rv)
	go rv.MonitorNetNsInode()
	return nil
}

func (m *MonitorNetNsInodeServer) SendMsg(msg interface{}) error {
	event, ok := msg.(*crossconnect.CrossConnectEvent)
	if !ok {
		return errors.New("wrong type of msg, crossConnectEvent is needed")
	}
	m.crossConnectEventCh <- copyEvent(event)
	return nil
}

func copyEvent(event *crossconnect.CrossConnectEvent) *crossconnect.CrossConnectEvent {
	crossConnectsCopy := map[string]*crossconnect.CrossConnect{}
	for k, v := range event.CrossConnects {
		if v != nil {
			vCopy := *v
			crossConnectsCopy[k] = &vCopy
		}
	}

	return &crossconnect.CrossConnectEvent{
		Type:          event.Type,
		CrossConnects: crossConnectsCopy,
		Metrics:       event.Metrics,
	}
}

func (m *MonitorNetNsInodeServer) MonitorNetNsInode() {
	for {
		select {
		case <-time.After(3 * time.Second):
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

// Accept cross connection and one of connections from it,
func (m *MonitorNetNsInodeServer) checkConnectionLiveness(xcon *crossconnect.CrossConnect, conn *connection.Connection,
	inodeSet *InodeSet) error {
	var inode uint64
	var err error

	switch conn.GetMechanism().GetType() {
	case kernel.MECHANISM:
		inode, err = strconv.ParseUint(kernel.ToMechanism(conn.GetMechanism()).GetNetNsInode(), 10, 64)
		if err != nil {
			return err
		}
	case memif.MECHANISM:
		inode, err = strconv.ParseUint(memif.ToMechanism(conn.GetMechanism()).GetNetNsInode(), 10, 64)
		if err != nil {
			return err
		}
	default:
		return errors.New("Wrong mechanism type passed")
	}

	if !inodeSet.Contains(inode) && conn.State == connection.State_UP {
		logrus.Infof("Connection is down")
		conn.State = connection.State_DOWN
		m.forwarderHandler()
		m.crossConnectServer.Update(context.Background(), xcon)
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
