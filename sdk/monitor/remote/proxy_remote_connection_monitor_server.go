package remote

import (
	"context"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/utils/interdomain"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"
)

const (
	proxyLogFormat          = "ProxyNSM-Monitor(%v): %v"
	proxyLogWithParamFormat = "ProxyNSM-Monitor(%v): %v: %v"
)

// ProxyMonitorServer is a monitor.Server for proxy remote/connection GRPC API
type ProxyMonitorServer interface {
	connection.MonitorConnectionServer
}

type proxyMonitorServer struct {
}

type entityHandler func(connectionServer connection.MonitorConnection_MonitorConnectionsServer, entity monitor.Entity, event monitor.Event) error

// NewProxyMonitorServer creates a new ProxyMonitorServer
func NewProxyMonitorServer() ProxyMonitorServer {
	rv := &proxyMonitorServer{}
	return rv
}

// MonitorConnections adds recipient for MonitorServer events
func (s *proxyMonitorServer) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filtered := connectionmonitor.NewMonitorConnectionFilter(selector, recipient)

	logrus.Printf("Monitor Connections request: %s -> %s", selector.GetPathSegments()[0].GetName(), selector.GetPathSegments()[1].GetName())

	remotePeerName, remotePeerURL, err := interdomain.ParseNsmURL(selector.GetPathSegments()[1].GetName())
	if err != nil {
		return errors.Wrap(err, "ProxyNSM-Monitor")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan error)

	go s.monitorConnection(
		ctx,
		selector.GetPathSegments()[0].GetName(), remotePeerName, remotePeerURL,
		s.handleRemoteConnection, filtered, quit)

	select {
	case <-filtered.Context().Done():
		cancel()
		<-quit
	case err := <-quit:
		if err != nil {
			logrus.Errorf(proxyLogWithParamFormat, remotePeerName, "Connection closed", err)
		}
	}

	logrus.Printf("Monitor Connections done: %s -> %s", selector.GetPathSegments()[0].GetName(), selector.GetPathSegments()[1].GetName())

	return nil
}

func (s *proxyMonitorServer) monitorConnection(
	ctx context.Context,
	name, remotePeerName, remotePeerURL string,
	entityHandler entityHandler, connectionServer connection.MonitorConnection_MonitorConnectionsServer,
	quit chan error) {
	logrus.Infof(proxyLogFormat, name, "Added")

	conn, err := tools.DialTCP(remotePeerURL)
	if err != nil {
		logrus.Errorf(proxyLogWithParamFormat, name, "Failed to connect", err)
		quit <- err
		return
	}
	logrus.Infof(proxyLogFormat, name, "Connected")
	defer func() { _ = conn.Close() }()

	monitorClient, err := connectionmonitor.NewMonitorClient(conn, &connection.MonitorScopeSelector{
		PathSegments: []*connection.PathSegment{
			{
				Name: name,
			},
			{
				Name: remotePeerName,
			},
		},
	})
	if err != nil {
		logrus.Errorf(proxyLogWithParamFormat, name, "Failed to start monitor", err)
		quit <- err
		return
	}
	logrus.Infof(proxyLogFormat, name, "Started monitor")
	defer monitorClient.Close()

	for {
		select {
		case <-ctx.Done():
			logrus.Infof(proxyLogFormat, name, "Removed")
			quit <- nil
			return
		case err = <-monitorClient.ErrorChannel():
			quit <- err
			return
		case event := <-monitorClient.EventChannel():
			if event != nil {
				logrus.Infof(proxyLogWithParamFormat, name, "Received event", event)
				for _, entity := range event.Entities() {
					if err = entityHandler(connectionServer, entity, event); err != nil {
						logrus.Errorf(proxyLogWithParamFormat, name, "Error handling entity", err)
					}
				}
			}
		}
	}
}

func (s *proxyMonitorServer) handleRemoteConnection(connectionServer connection.MonitorConnection_MonitorConnectionsServer, entity monitor.Entity, event monitor.Event) error {
	remoteConnection, ok := entity.(*connection.Connection)
	if !ok {
		return errors.Errorf("unable to cast %v to remote.Connection", entity)
	}

	msg, _ := event.Message()
	err := connectionServer.SendMsg(msg)

	logrus.Printf("handleRemoteConnection (%v) %v: %v", remoteConnection.GetId(), event.EventType(), msg)

	return err
}
