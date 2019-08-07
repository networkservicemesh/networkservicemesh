package networkservice

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
)

// NewRequest creates a new remote.NetworkServiceRequest
func NewRequest(connection connection.Connection, mechanismPreferences []connection.Mechanism) *NetworkServiceRequest {
	ns := &NetworkServiceRequest{}

	ns.SetRequestConnection(connection)
	ns.SetRequestMechanismPreferences(mechanismPreferences)

	return ns
}

// IsRemote returns if request is remote
func (ns *NetworkServiceRequest) IsRemote() bool {
	return true
}

// Clone clones request
func (ns *NetworkServiceRequest) Clone() networkservice.Request {
	return proto.Clone(ns).(*NetworkServiceRequest)
}

// GetRequestConnection returns request connection
func (ns *NetworkServiceRequest) GetRequestConnection() connection.Connection {
	return ns.GetConnection()
}

// SetRequestConnection sets request connection
func (ns *NetworkServiceRequest) SetRequestConnection(connection connection.Connection) {
	ns.Connection = connection.(*remote.Connection)
}

// GetRequestMechanismPreferences returns request mechanism preferences
func (ns *NetworkServiceRequest) GetRequestMechanismPreferences() []connection.Mechanism {
	preferences := make([]connection.Mechanism, 0, len(ns.MechanismPreferences))
	for _, m := range ns.MechanismPreferences {
		preferences = append(preferences, m)
	}

	return preferences
}

// SetRequestMechanismPreferences sets request mechanism preferences
func (ns *NetworkServiceRequest) SetRequestMechanismPreferences(mechanismPreferences []connection.Mechanism) {
	preferences := make([]*remote.Mechanism, 0, len(mechanismPreferences))
	for _, m := range mechanismPreferences {
		preferences = append(preferences, m.(*remote.Mechanism))
	}

	ns.MechanismPreferences = preferences
}

// IsValid returns if request is valid
func (ns *NetworkServiceRequest) IsValid() error {
	if ns == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if ns.GetConnection() == nil {
		return fmt.Errorf("request.Connection cannot be nil %v", ns)
	}

	if err := ns.GetConnection().IsValid(); err != nil {
		return fmt.Errorf("request.Connection is invalid: %s: %v", err, ns)
	}

	if ns.GetMechanismPreferences() == nil {
		return fmt.Errorf("request.MechanismPreferences cannot be nil: %v", ns)
	}

	if len(ns.GetMechanismPreferences()) < 1 {
		return fmt.Errorf("request.MechanismPreferences must have at least one entry: %v", ns)
	}

	return nil
}

// NewReply creates new instance of Reply
func NewReply(c connection.Connection) networkservice.Reply {
	return &NetworkServiceReply{
		Connection: c.(*remote.Connection),
	}
}

// GetReplyConnection returns Connection
func (r *NetworkServiceReply) GetReplyConnection() connection.Connection {
	return r.GetConnection()
}

// Clone returns deep copy of NetworkServiceReply
func (r *NetworkServiceReply) Clone() networkservice.Reply {
	return proto.Clone(r).(*NetworkServiceReply)
}
