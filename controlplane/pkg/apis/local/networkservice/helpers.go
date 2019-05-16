package networkservice

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
)

func (ns *NetworkServiceRequest) IsValid() error {
	if ns == nil {
		return fmt.Errorf("NetorkServiceRequest cannot be nil")
	}

	if ns.GetConnection() == nil {
		return fmt.Errorf("NetworkServiceRequest.Connection cannot be nil %v", ns)
	}

	if err := ns.GetConnection().IsValid(); err != nil {
		return fmt.Errorf("NetworkServiceRequest.Connection is invalid: %s: %v", err, ns)
	}

	if ns.GetMechanismPreferences() == nil {
		return fmt.Errorf("NetworkServiceRequest.MechanismPreferences cannot be nil: %v", ns)
	}
	if len(ns.GetMechanismPreferences()) < 1 {
		return fmt.Errorf("NetworkServiceRequest.MechanismPreferences must have at least one entry: %v", ns)
	}
	return nil
}
func (ns *NetworkServiceRequest) IsRemote() bool {
	return false
}
func (ns *NetworkServiceRequest) GetConnectionId() string {
	return ns.GetConnection().GetId()
}

func (ns *NetworkServiceRequest) Clone() nsm.NSMRequest {
	return proto.Clone(ns).(*NetworkServiceRequest)
}
func (ns *NetworkServiceRequest) SetConnection(conn nsm.NSMConnection) {
	ns.Connection = conn.(*connection.Connection)
}
