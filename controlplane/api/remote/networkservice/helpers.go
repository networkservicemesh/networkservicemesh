package networkservice

import (
	"github.com/pkg/errors"

	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
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
	ns.Connection = connection.(*connection2.Connection)
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
	preferences := make([]*connection2.Mechanism, 0, len(mechanismPreferences))
	for _, m := range mechanismPreferences {
		preferences = append(preferences, m.(*connection2.Mechanism))
	}

	ns.MechanismPreferences = preferences
}

// IsValid returns if request is valid
func (ns *NetworkServiceRequest) IsValid() error {
	if ns == nil {
		return errors.New("request cannot be nil")
	}

	if ns.GetConnection() == nil {
		return errors.Errorf("request.Connection cannot be nil %v", ns)
	}

	if err := ns.GetConnection().IsValid(); err != nil {
		return errors.Wrapf(err, "request.Connection is invalid: %v", ns)
	}

	if ns.GetMechanismPreferences() == nil {
		return errors.Errorf("request.MechanismPreferences cannot be nil: %v", ns)
	}

	if len(ns.GetMechanismPreferences()) < 1 {
		return errors.Errorf("request.MechanismPreferences must have at least one entry: %v", ns)
	}

	return nil
}
