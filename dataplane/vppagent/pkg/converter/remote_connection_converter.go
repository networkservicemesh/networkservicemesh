package converter

import (
	remote "github.com/ligato/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/ligato/vpp-agent/plugins/vpp/model/rpc"
)

type RemoteConnectionConverter struct {
	*remote.Connection
	name string
}

func NewRemoteConnectionConverter(c *remote.Connection, name string) *RemoteConnectionConverter {
	return &RemoteConnectionConverter{
		Connection: c,
		name:       name,
	}
}

func (c *RemoteConnectionConverter) ToDataRequest(rv *rpc.DataRequest) (*rpc.DataRequest, error) {
	// TODO

	return rv, nil
}
