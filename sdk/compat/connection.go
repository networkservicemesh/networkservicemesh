package compat

import (
	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
)

var mapMechanismTypeLocalToUnified = map[local.MechanismType]string{
	local.MechanismType_KERNEL_INTERFACE: kernel.Mechanism,
	local.MechanismType_MEM_INTERFACE:    memif.Mechanism,
}

func MechanismLocalToUnified(mechanism *local.Mechanism) *unified.Mechanism {
	if mechanism == nil {
		return nil
	}
	return &unified.Mechanism{
		Cls:        cls.LOCAL,
		Type:       mapMechanismTypeLocalToUnified[mechanism.GetType()],
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

var mapMechanismTypeUnifiedToLocal = map[string]local.MechanismType{
	kernel.Mechanism: local.MechanismType_KERNEL_INTERFACE,
	memif.Mechanism:  local.MechanismType_MEM_INTERFACE,
}

func MechanismUnifiedToLocal(mechanism *unified.Mechanism) *local.Mechanism {
	if mechanism == nil {
		return nil
	}
	return &local.Mechanism{
		Type:       mapMechanismTypeUnifiedToLocal[mechanism.GetType()],
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

var mapMechanismTypeRemoteToUnified = map[remote.MechanismType]string{
	remote.MechanismType_VXLAN: vxlan.MECHANISM,
}

func MechanismRemoteToUnified(mechanism *remote.Mechanism) *unified.Mechanism {
	if mechanism == nil {
		return nil
	}
	return &unified.Mechanism{
		Cls:        cls.REMOTE,
		Type:       mapMechanismTypeRemoteToUnified[mechanism.GetType()],
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

var mapMechanismTypeUnifiedToRemote = map[string]remote.MechanismType{
	vxlan.MECHANISM: remote.MechanismType_VXLAN,
}

func MechanismUnifiedToRemote(mechanism *unified.Mechanism) *remote.Mechanism {
	if mechanism == nil {
		return nil
	}
	return &remote.Mechanism{
		Type:       mapMechanismTypeUnifiedToRemote[mechanism.GetType()],
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
	if c == nil {
		return nil
	}
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
	if c == nil {
		return nil
	}
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
	if c == nil {
		return nil
	}
	rv := &unified.Connection{
		Id:                         c.GetId(),
		NetworkService:             c.GetNetworkService(),
		Mechanism:                  MechanismRemoteToUnified(c.GetMechanism()),
		Context:                    c.GetContext(),
		Labels:                     c.GetLabels(),
		NetworkServiceManagers:     make([]string, 2),
		NetworkServiceEndpointName: c.GetNetworkServiceEndpointName(),
		State:                      unified.State(c.GetState()),
	}
	if c.GetSourceNetworkServiceManagerName() != "" {
		rv.GetNetworkServiceManagers()[0] = c.GetSourceNetworkServiceManagerName()
	}
	if c.GetDestinationNetworkServiceManagerName() != "" {
		rv.GetNetworkServiceManagers()[1] = c.GetDestinationNetworkServiceManagerName()
	}
	return rv
}

func ConnectionUnifiedToRemote(c *unified.Connection) *remote.Connection {
	if c == nil {
		return nil
	}
	rv := &remote.Connection{
		Id:                         c.GetId(),
		NetworkService:             c.GetNetworkService(),
		Mechanism:                  MechanismUnifiedToRemote(c.GetMechanism()),
		Context:                    c.GetContext(),
		Labels:                     c.GetLabels(),
		NetworkServiceEndpointName: c.GetNetworkServiceEndpointName(),
		State:                      remote.State(c.GetState()),
	}
	if len(c.GetNetworkServiceManagers()) >= 1 {
		rv.SourceNetworkServiceManagerName = c.GetNetworkServiceManagers()[0]
		if len(c.GetNetworkServiceManagers()) >= 2 {
			rv.DestinationNetworkServiceManagerName = c.GetNetworkServiceManagers()[1]
		}
	}
	return rv
}

func MonitorScopeSelectorUnifiedToRemote(c *unified.MonitorScopeSelector) *remote.MonitorScopeSelector {
	if c == nil {
		return nil
	}
	rv := &remote.MonitorScopeSelector{}
	if len(c.GetNetworkServiceManagers()) >= 1 {
		rv.NetworkServiceManagerName = c.GetNetworkServiceManagers()[0]
		if len(c.GetNetworkServiceManagers()) >= 2 {
			rv.DestinationNetworkServiceManagerName = c.GetNetworkServiceManagers()[1]
		}
	}
	return rv
}

func MonitorScopeSelectorRemoteToUnified(c *remote.MonitorScopeSelector) *unified.MonitorScopeSelector {
	if c == nil {
		return nil
	}
	rv := &unified.MonitorScopeSelector{
		NetworkServiceManagers: make([]string, 2),
	}
	if c.GetNetworkServiceManagerName() != "" {
		rv.GetNetworkServiceManagers()[0] = c.GetNetworkServiceManagerName()
	}
	if c.GetDestinationNetworkServiceManagerName() != "" {
		rv.GetNetworkServiceManagers()[1] = c.GetDestinationNetworkServiceManagerName()
	}
	return rv
}

func ConnectionEventLocalToUnified(c *local.ConnectionEvent) *unified.ConnectionEvent {
	if c == nil {
		return nil
	}
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
	if c == nil {
		return nil
	}
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
	if c == nil {
		return nil
	}
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
	if c == nil {
		return nil
	}
	rv := &remote.ConnectionEvent{
		Type:        remote.ConnectionEventType(c.GetType()),
		Connections: make(map[string]*remote.Connection, len(c.GetConnections())),
	}
	for k, v := range c.GetConnections() {
		rv.GetConnections()[k] = ConnectionUnifiedToRemote(v)
	}
	return rv
}
