package compat

import (
	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
)

func MechanismLocalToUnified(mechanism *local.Mechanism) *unified.Mechanism {
	return &unified.Mechanism{
		Cls:        cls.LOCAL,
		Type:       mechanism.GetType().String(),
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListLocalToUnified(mechanism []*local.Mechanism) []*unified.Mechanism {
	rv := make([]*unified.Mechanism, len(mechanism))
	for i, value := range mechanism {
		rv[i] = MechanismLocalToUnified(value)
	}
	return rv
}

func MechanismUnifiedToLocal(mechanism *unified.Mechanism) *local.Mechanism {
	return &local.Mechanism{
		Type:       0,
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListUnifiedToLocal(mechanism []*unified.Mechanism) []*local.Mechanism {
	rv := make([]*local.Mechanism, len(mechanism))
	for i, value := range mechanism {
		rv[i] = MechanismUnifiedToLocal(value)
	}
	return rv
}

func MechanismRemoteToUnified(mechanism *remote.Mechanism) *unified.Mechanism {
	return &unified.Mechanism{
		Cls:        cls.REMOTE,
		Type:       mechanism.GetType().String(),
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListRemoteToUnified(mechanism []*remote.Mechanism) []*unified.Mechanism {
	rv := make([]*unified.Mechanism, len(mechanism))
	for i, value := range mechanism {
		rv[i] = MechanismRemoteToUnified(value)
	}
	return rv
}

func MechanismUnifiedToRemote(mechanism *unified.Mechanism) *remote.Mechanism {
	return &remote.Mechanism{
		Type:       0,
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListUnifiedToRemote(mechanism []*unified.Mechanism) []*remote.Mechanism {
	rv := make([]*remote.Mechanism, len(mechanism))
	for i, value := range mechanism {
		rv[i] = MechanismUnifiedToRemote(value)
	}
	return rv
}

func ConnectionLocalToUnified(c *local.Connection) *unified.Connection {
	return &unified.Connection{
		Id:                         c.GetId(),
		NetworkService:             c.GetNetworkService(),
		Mechanism:                  MechanismLocalToUnified(c.GetMechanism()),
		Context:                    c.GetContext(),
		Labels:                     c.GetLabels(),
		NetworkServiceManagers:     make([]string, 1),
		NetworkServiceEndpointName: c.GetNetworkServiceEndpointName(),
		State:                      unified.State(c.GetState()),
	}
}

func ConnectionUnifiedToLocal(c *unified.Connection) *local.Connection {
	return &local.Connection{
		Id:             c.GetId(),
		NetworkService: c.GetNetworkService(),
		Mechanism:      MechanismUnifiedToLocal(c.GetMechanism()),
		Context:        c.GetContext(),
		Labels:         c.GetLabels(),
		State:          local.State(c.GetState()),
	}
}

func ConnectionRemoteToUnified(c *remote.Connection) *unified.Connection {
	return &unified.Connection{
		Id:                         c.GetId(),
		NetworkService:             c.GetNetworkService(),
		Mechanism:                  MechanismRemoteToUnified(c.GetMechanism()),
		Context:                    c.GetContext(),
		Labels:                     c.GetLabels(),
		NetworkServiceManagers:     []string{c.GetSourceNetworkServiceManagerName(), c.GetDestinationNetworkServiceManagerName()},
		NetworkServiceEndpointName: c.GetNetworkServiceEndpointName(),
		State:                      unified.State(c.GetState()),
	}
}

func ConnectionUnifiedToRemote(c *unified.Connection) *remote.Connection {
	return &remote.Connection{
		Id:                                   c.GetId(),
		NetworkService:                       c.GetNetworkService(),
		Mechanism:                            MechanismUnifiedToRemote(c.GetMechanism()),
		Context:                              c.GetContext(),
		Labels:                               c.GetLabels(),
		SourceNetworkServiceManagerName:      c.GetNetworkServiceManagers()[0],
		DestinationNetworkServiceManagerName: c.GetNetworkServiceManagers()[1],
		NetworkServiceEndpointName:           c.GetNetworkServiceEndpointName(),
		State:                                remote.State(c.GetState()),
	}
}
