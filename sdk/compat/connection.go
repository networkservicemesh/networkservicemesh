package compat

import (
	"github.com/sirupsen/logrus"

	unified "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	nsm "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
)

var mapMechanismTypeLocalToUnified = map[local.MechanismType]string{
	local.MechanismType_KERNEL_INTERFACE: kernel.MECHANISM,
	local.MechanismType_MEM_INTERFACE:    memif.MECHANISM,
}

func MechanismLocalToUnified(mechanism *local.Mechanism) *unified.Mechanism {
	if mechanism == nil {
		return nil
	}
	typeValue, ok := mapMechanismTypeLocalToUnified[mechanism.GetType()]
	if !ok {
		typeValue = mechanism.GetType().String()
	}
	return &unified.Mechanism{
		Cls:        cls.LOCAL,
		Type:       typeValue,
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListLocalToUnified(mechanism []*local.Mechanism) []*unified.Mechanism {
	if mechanism == nil {
		return nil
	}
	rv := make([]*unified.Mechanism, len(mechanism))
	for i, value := range mechanism {
		rv[i] = MechanismLocalToUnified(value)
	}
	return rv
}

var mapMechanismTypeUnifiedToLocal = map[string]local.MechanismType{
	kernel.MECHANISM: local.MechanismType_KERNEL_INTERFACE,
	memif.MECHANISM:  local.MechanismType_MEM_INTERFACE,
}

func MechanismUnifiedToLocal(mechanism *unified.Mechanism) *local.Mechanism {
	if mechanism == nil {
		return nil
	}
	typeValue, ok := mapMechanismTypeUnifiedToLocal[mechanism.GetType()]
	if !ok {
		mval, ok2 := local.MechanismType_value[mechanism.GetType()]
		if !ok2 {
			logrus.Errorf("Fatal, conversion to local mechanism is not possible %v", mechanism)
			return nil
		}
		typeValue = local.MechanismType(mval)
	}
	return &local.Mechanism{
		Type:       typeValue,
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListUnifiedToLocal(mechanism []*unified.Mechanism) []*local.Mechanism {
	if mechanism == nil {
		return nil
	}
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
	mechanismType, ok := mapMechanismTypeRemoteToUnified[mechanism.GetType()]
	if !ok {
		mechanismType = mechanism.GetType().String()
	}
	return &unified.Mechanism{
		Cls:        cls.REMOTE,
		Type:       mechanismType,
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListRemoteToUnified(mechanism []*remote.Mechanism) []*unified.Mechanism {
	if mechanism == nil {
		return nil
	}
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
	typeValue, ok := mapMechanismTypeUnifiedToRemote[mechanism.GetType()]
	if !ok {
		mval, ok2 := remote.MechanismType_value[mechanism.GetType()]
		if !ok2 {
			logrus.Errorf("Fatal, conversion to remote mechanism is not possible %v", mechanism)
			return nil
		}
		typeValue = remote.MechanismType(mval)
	}
	return &remote.Mechanism{
		Type:       typeValue,
		Parameters: mechanism.GetParameters(),
	}
}

func MechanismListUnifiedToRemote(mechanism []*unified.Mechanism) []*remote.Mechanism {
	if mechanism == nil {
		return nil
	}
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
		ResponseToken:              c.GetResponseJWT(),
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
		ResponseJWT:    c.GetResponseToken(),
	}
}

func ConnectionRemoteToUnified(c *remote.Connection) *unified.Connection {
	if c == nil {
		return nil
	}
	rv := &unified.Connection{
		Id:             c.GetId(),
		NetworkService: c.GetNetworkService(),
		Mechanism:      MechanismRemoteToUnified(c.GetMechanism()),
		Context:        c.GetContext(),
		Labels:         c.GetLabels(),
		NetworkServiceManagers: []string{
			c.GetSourceNetworkServiceManagerName(),
			c.GetDestinationNetworkServiceManagerName(),
		},
		NetworkServiceEndpointName: c.GetNetworkServiceEndpointName(),
		State:                      unified.State(c.GetState()),
		ResponseToken:              c.GetResponseJWT(),
	}
	if c.GetSourceNetworkServiceManagerName() == "" && c.GetDestinationNetworkServiceManagerName() == "" {
		rv.NetworkServiceManagers = nil
	}

	return rv
}

func ConnectionUnifiedToRemote(c *unified.Connection) *remote.Connection {
	if c == nil {
		return nil
	}
	rv := &remote.Connection{
		Id:                                   c.GetId(),
		NetworkService:                       c.GetNetworkService(),
		Mechanism:                            MechanismUnifiedToRemote(c.GetMechanism()),
		Context:                              c.GetContext(),
		Labels:                               c.GetLabels(),
		NetworkServiceEndpointName:           c.GetNetworkServiceEndpointName(),
		SourceNetworkServiceManagerName:      c.GetSourceNetworkServiceManagerName(),
		DestinationNetworkServiceManagerName: c.GetDestinationNetworkServiceManagerName(),
		State:                                remote.State(c.GetState()),
		ResponseJWT:                          c.GetResponseToken(),
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

// ConnectionUnifiedToNSM - convert unified connection to NSM
func ConnectionUnifiedToNSM(c *unified.Connection) nsm.Connection {
	if c == nil {
		return nil
	}
	if c.IsRemote() {
		return ConnectionUnifiedToRemote(c)
	}
	return ConnectionUnifiedToLocal(c)
}

// ConnectionNSMToUnified - convert nsm unified connection to unified.
func ConnectionNSMToUnified(c nsm.Connection) *unified.Connection {
	if c == nil {
		return nil
	}
	if c.IsRemote() {
		return ConnectionRemoteToUnified(c.(*remote.Connection))
	}
	return ConnectionLocalToUnified(c.(*local.Connection))
}

// MechanismNSMToUnified - convert nsm unified connection to unified.
func MechanismNSMToUnified(c nsm.Mechanism) *unified.Mechanism {
	if c == nil {
		return nil
	}
	if c.IsRemote() {
		return MechanismRemoteToUnified(c.(*remote.Mechanism))
	}
	return MechanismLocalToUnified(c.(*local.Mechanism))
}
