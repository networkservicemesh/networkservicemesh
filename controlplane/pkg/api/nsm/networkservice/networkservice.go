package networkservice

import (
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm/connection"
)

// Request is an unified interface for local/remote network requests
type Request interface {
	IsRemote() bool
	Clone() Request

	GetRequestConnection() connection2.Connection
	SetRequestConnection(connection connection2.Connection)

	GetRequestMechanismPreferences() []connection2.Mechanism
	SetRequestMechanismPreferences(mechanismPreferences []connection2.Mechanism)

	IsValid() error
}
