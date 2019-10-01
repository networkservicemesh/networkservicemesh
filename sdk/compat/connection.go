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

func MonitorScopeSelectorUnifiedToRemote(c *unified.MonitorScopeSelector) *remote.MonitorScopeSelector {
	return &remote.MonitorScopeSelector{
		NetworkServiceManagerName:            c.GetNetworkServiceManagers()[0],
		DestinationNetworkServiceManagerName: c.GetNetworkServiceManagers()[1],
	}
}

func MonitorScopeSelectorRemoteToUnified(c *remote.MonitorScopeSelector) *unified.MonitorScopeSelector {
	return &unified.MonitorScopeSelector{
		NetworkServiceManagers: []string{
			c.GetNetworkServiceManagerName(),
			c.GetDestinationNetworkServiceManagerName(),
		},
	}
}

func ConnectionEventLocalToUnified(c *local.ConnectionEvent) *unified.ConnectionEvent {
	rv := &unified.ConnectionEvent{
		Type:        unified.ConnectionEventType(c.GetType()),
		Connections: make(map[string]*unified.Connection, len(c.GetConnections())),
	}
	for k, v := range c.GetConnections() {
		rv.GetConnections()[k] = ConnectionLocalToUnified(v)
	}
	return rv
}

func ConnectionEventUnifiedToLocal(c *unified.ConnectionEvent) *local.ConnectionEvent {
	rv := &local.ConnectionEvent{
		Type:        local.ConnectionEventType(c.GetType()),
		Connections: make(map[string]*local.Connection, len(c.GetConnections())),
	}
	for k, v := range c.GetConnections() {
		rv.GetConnections()[k] = ConnectionUnifiedToLocal(v)
	}
	return rv
}

func ConnectionEventRemoteToUnified(c *remote.ConnectionEvent) *unified.ConnectionEvent {
	rv := &unified.ConnectionEvent{
		Type:        unified.ConnectionEventType(c.GetType()),
		Connections: make(map[string]*unified.Connection, len(c.GetConnections())),
	}
	for k, v := range c.GetConnections() {
		rv.GetConnections()[k] = ConnectionRemoteToUnified(v)
	}
	return rv
}

func ConnectionEventUnifiedToRemote(c *unified.ConnectionEvent) *remote.ConnectionEvent {
	rv := &remote.ConnectionEvent{
		Type:        remote.ConnectionEventType(c.GetType()),
		Connections: make(map[string]*remote.Connection, len(c.GetConnections())),
	}
	for k, v := range c.GetConnections() {
		rv.GetConnections()[k] = ConnectionUnifiedToRemote(v)
	}
	return rv
}
