package endpoint

import (
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/authorize"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/monitor"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/setid"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/timeout"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/update_path"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/chain"
)

type Endpoint interface {
	networkservice.NetworkServiceServer
	connection.MonitorConnectionServer
	Register(s *grpc.Server)
}

type endpoint struct {
	networkservice.NetworkServiceServer
	connection.MonitorConnectionServer
}

func NewServer(name string, innerFunctionality ...networkservice.NetworkServiceServer) Endpoint {
	rv := &endpoint{}
	rv.NetworkServiceServer = chain.NewNetworkServiceServer(
		append([]networkservice.NetworkServiceServer{
			authorize.NewServer(),
			setid.NewServer(name),
			monitor.NewServer(&rv.MonitorConnectionServer),
			timeout.NewServer(),
			update_path.NewServer(name),
		}, innerFunctionality...)...)
	return rv
}

func (e *endpoint) Register(s *grpc.Server) {
	networkservice.RegisterNetworkServiceServer(s, e)
	connection.RegisterMonitorConnectionServer(s, e)
}
