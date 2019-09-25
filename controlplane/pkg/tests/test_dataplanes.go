package tests

import (
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

var testDataplane1 = &model.Dataplane{
	RegisteredName: "test_data_plane",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []connection.Mechanism{
		&local.Mechanism{
			Type: local.MechanismType_KERNEL_INTERFACE,
		},
	},
	RemoteMechanisms: []connection.Mechanism{
		&remote.Mechanism{
			Type: remote.MechanismType_VXLAN,
			Parameters: map[string]string{
				remote.VXLANSrcIP: "127.0.0.1",
			},
		},
	},
	MechanismsConfigured: true,
}
var testDataplane1_1 = &model.Dataplane{
	RegisteredName: "test_data_plane_11",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []connection.Mechanism{
		&local.Mechanism{
			Type: local.MechanismType_KERNEL_INTERFACE,
		},
	},
	RemoteMechanisms: []connection.Mechanism{
		&remote.Mechanism{
			Type: remote.MechanismType_VXLAN,
			Parameters: map[string]string{
				remote.VXLANSrcIP: "127.0.0.7",
			},
		},
	},
	MechanismsConfigured: true,
}

var testDataplane2 = &model.Dataplane{
	RegisteredName: "test_data_plane2",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []connection.Mechanism{
		&local.Mechanism{
			Type: local.MechanismType_KERNEL_INTERFACE,
		},
	},
	RemoteMechanisms: []connection.Mechanism{
		&remote.Mechanism{
			Type: remote.MechanismType_VXLAN,
			Parameters: map[string]string{
				remote.VXLANSrcIP: "127.0.0.2",
			},
		},
	},
	MechanismsConfigured: true,
}
