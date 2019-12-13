package monitor

import (
	"context"
	"runtime"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/next"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/tools/serialize"
)

type monitorServer struct {
	connections map[string]*connection.Connection
	monitors    []connection.MonitorConnection_MonitorConnectionsServer
	executor    serialize.Executor
	finalized   chan struct{}
}

func NewServer(monitorServerPtr *connection.MonitorConnectionServer) networkservice.NetworkServiceServer {
	rv := &monitorServer{
		connections: make(map[string]*connection.Connection),
		monitors:    nil,
		executor:    serialize.NewExecutor(),
		finalized:   make(chan struct{}),
	}
	runtime.SetFinalizer(rv, func(server *monitorServer) {
		close(server.finalized)
	})
	*monitorServerPtr = rv
	return rv
}

func (m *monitorServer) MonitorConnections(selector *connection.MonitorScopeSelector, srv connection.MonitorConnection_MonitorConnectionsServer) error {
	m.executor.Exec(func() {
		monitor := newMonitorFilter(selector, srv)
		m.monitors = append(m.monitors, monitor)
		_ = monitor.Send(&connection.ConnectionEvent{
			Type:        connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
			Connections: m.connections,
		})
	})
	select {
	case <-srv.Context().Done():
	case <-m.finalized:
	}
	return nil
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		m.executor.Exec(func() {
			m.connections[conn.GetId()] = conn
			event := &connection.ConnectionEvent{
				Type:        connection.ConnectionEventType_UPDATE,
				Connections: map[string]*connection.Connection{conn.GetId(): conn},
			}
			m.monitors = send(m.monitors, event)
		})
	}
	return conn, err
}

func (m *monitorServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	m.executor.Exec(func() {
		delete(m.connections, conn.GetId())
		event := &connection.ConnectionEvent{
			Type:        connection.ConnectionEventType_DELETE,
			Connections: map[string]*connection.Connection{conn.GetId(): conn},
		}
		m.monitors = send(m.monitors, event)
	})
	return next.Server(ctx).Close(ctx, conn)
}

func send(monitors []connection.MonitorConnection_MonitorConnectionsServer, event *connection.ConnectionEvent) []connection.MonitorConnection_MonitorConnectionsServer {
	newMonitors := make([]connection.MonitorConnection_MonitorConnectionsServer, len(monitors))
	for _, srv := range monitors {
		select {
		case <-srv.Context().Done():
		default:
			srv.Send(event)
			newMonitors = append(newMonitors, srv)
		}
	}
	return newMonitors
}
