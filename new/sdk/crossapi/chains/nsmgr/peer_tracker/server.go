package peer_tracker

import (
	"context"
	"net/url"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/chains/nsmgr"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/tools/serialize"
	"google.golang.org/grpc/peer"
)

type peerTrackerServer struct {
	nsmgr.Nsmgr
	executor serialize.Executor
	// Outer map is peer url.URL.String(), inner map key is Connection.Id
	connections map[string]map[string]*connection.Connection
	inner       networkservice.NetworkServiceServer
}

func NewServer(inner nsmgr.Nsmgr, closeAll *func(u *url.URL)) nsmgr.Nsmgr {
	rv := &peerTrackerServer{
		executor:    serialize.NewExecutor(),
		connections: make(map[string]map[string]*connection.Connection),
		inner:       inner,
	}
	*closeAll = rv.CloseAllConnectionsForPeer
	return rv
}

func (p *peerTrackerServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conn, err := p.inner.Request(ctx, request)
	if err != nil {
		return nil, err
	}
	mypeer, ok := peer.FromContext(ctx)
	if ok {
		if mypeer.Addr.Network() == "unix" {
			u := &url.URL{
				Scheme: mypeer.Addr.Network(),
				Path:   mypeer.Addr.String(),
			}
			p.executor.Exec(func() {
				_, ok := p.connections[u.String()]
				if !ok {
					p.connections[u.String()] = make(map[string]*connection.Connection)
				}
				p.connections[u.String()][conn.GetId()] = conn
			})
		}
	}
	return conn, nil
}

func (p *peerTrackerServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	_, err := p.inner.Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	mypeer, ok := peer.FromContext(ctx)
	if ok {
		if mypeer.Addr.Network() == "unix" {
			u := &url.URL{
				Scheme: mypeer.Addr.Network(),
				Path:   mypeer.Addr.String(),
			}
			p.executor.Exec(func() {
				delete(p.connections[u.String()], conn.GetId())
			})
		}
	}
	return &empty.Empty{}, nil
}

func (p *peerTrackerServer) CloseAllConnectionsForPeer(u *url.URL) {
	finishedChan := make(chan struct{})
	p.executor.Exec(func() {
		if connMap, ok := p.connections[u.String()]; ok {
			for _, conn := range connMap {
				// TODO - we probably want to do something smarter here with context
				ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
				p.Close(ctx, conn)
			}
		}
		close(finishedChan)
	})
	select {
	case <-finishedChan:
		return
	}
}
