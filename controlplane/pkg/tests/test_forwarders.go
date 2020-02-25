package tests

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

var testForwarder1 = &model.Forwarder{
	RegisteredName: "test_data_plane",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []*networkservice.Mechanism{
		&networkservice.Mechanism{
			Type: kernel.MECHANISM,
		},
	},
	RemoteMechanisms: []*networkservice.Mechanism{
		&networkservice.Mechanism{
			Type: vxlan.MECHANISM,
			Parameters: map[string]string{
				vxlan.SrcIP: "127.0.0.1",
			},
		},
	},
	MechanismsConfigured: true,
}
var testForwarder1_1 = &model.Forwarder{
	RegisteredName: "test_data_plane_11",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []*networkservice.Mechanism{
		{
			Type: kernel.MECHANISM,
		},
	},
	RemoteMechanisms: []*networkservice.Mechanism{
		&networkservice.Mechanism{
			Type: vxlan.MECHANISM,
			Parameters: map[string]string{
				vxlan.SrcIP: "127.0.0.7",
			},
		},
	},
	MechanismsConfigured: true,
}

var testForwarder2 = &model.Forwarder{
	RegisteredName: "test_data_plane2",
	SocketLocation: "tcp:some_addr",
	LocalMechanisms: []*networkservice.Mechanism{
		&networkservice.Mechanism{
			Type: kernel.MECHANISM,
		},
	},
	RemoteMechanisms: []*networkservice.Mechanism{
		&networkservice.Mechanism{
			Type: vxlan.MECHANISM,
			Parameters: map[string]string{
				vxlan.SrcIP: "127.0.0.2",
			},
		},
	},
	MechanismsConfigured: true,
}
