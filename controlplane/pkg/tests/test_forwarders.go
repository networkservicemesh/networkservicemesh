package tests

import (
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

var testForwarder1 = &model.Forwarder{
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
var testForwarder1_1 = &model.Forwarder{
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

var testForwarder2 = &model.Forwarder{
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
