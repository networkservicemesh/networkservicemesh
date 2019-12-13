package xconnect

import (
	"net/url"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/commit"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/mechanisms/mechanism_kernel"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/mechanisms/mechanism_memif"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/mechanisms/mechanism_vxlan"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/set_ip/set_ip_kernel"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/set_route/set_kernel_route"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/vppagent"
	"github.com/networkservicemesh/networkservicemesh/new/sdk-vppagent/networkservicemesh/xconnect/l2_xconnect"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/chains/client"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/chains/endpoint"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/client_url"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/common/connect"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/copy_client_connection_context"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/networkservicemesh/core/adapters"
	"google.golang.org/grpc"
)

type xconnectServer struct {
	endpoint.Endpoint
}

func NewServer(name string, vppagentCC *grpc.ClientConn, baseDir string, u *url.URL) networkservice.NetworkServiceServer {
	rv := xconnectServer{}
	rv.Endpoint = endpoint.NewServer(
		name,
		vppagent.NewServer(),
		mechanism_memif.NewServer(baseDir),
		mechanism_kernel.NewServer(baseDir),
		mechanism_vxlan.NewServer(),
		client_url.NewServer(u),
		connect.NewServer(client.NewClientFactory(
			name,
			adapters.NewServerToClient(rv),
			mechanism_memif.NewClient(baseDir),
			mechanism_kernel.NewClient(),
			mechanism_vxlan.NewClient(),
			l2_xconnect.NewClient(),
		),
		),
		copy_client_connection_context.NewServer(),
		set_ip_kernel.NewServer(),
		set_kernel_route.NewServer(),
		commit.NewServer(vppagentCC),
	)
	return rv
}
