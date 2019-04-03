package tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

var testDataplane1 = &model.Dataplane{
	RegisteredName: "test_data_plane",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []*connection.Mechanism{
		&connection.Mechanism{
			Type: connection.MechanismType_KERNEL_INTERFACE,
		},
	},
	RemoteMechanisms: []*connection2.Mechanism{
		&connection2.Mechanism{
			Type: connection2.MechanismType_VXLAN,
			Parameters: map[string]string{
				connection2.VXLANSrcIP: "127.0.0.1",
			},
		},
	},
	RemoteConfigured: true,
}
var testDataplane1_1 = &model.Dataplane{
	RegisteredName: "test_data_plane_11",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []*connection.Mechanism{
		&connection.Mechanism{
			Type: connection.MechanismType_KERNEL_INTERFACE,
		},
	},
	RemoteMechanisms: []*connection2.Mechanism{
		&connection2.Mechanism{
			Type: connection2.MechanismType_VXLAN,
			Parameters: map[string]string{
				connection2.VXLANSrcIP: "127.0.0.7",
			},
		},
	},
	RemoteConfigured: true,
}

var testDataplane2 = &model.Dataplane{
	RegisteredName: "test_data_plane2",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []*connection.Mechanism{
		&connection.Mechanism{
			Type: connection.MechanismType_KERNEL_INTERFACE,
		},
	},
	RemoteMechanisms: []*connection2.Mechanism{
		&connection2.Mechanism{
			Type: connection2.MechanismType_VXLAN,
			Parameters: map[string]string{
				connection2.VXLANSrcIP: "127.0.0.2",
			},
		},
	},
	RemoteConfigured: true,
}
