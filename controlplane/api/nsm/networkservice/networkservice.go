package networkservice

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
)

// Request is an unified interface for local/remote network requests
type Request interface {
	IsRemote() bool
	Clone() Request

	GetRequestConnection() connection.Connection
	SetRequestConnection(connection connection.Connection)

	GetRequestMechanismPreferences() []connection.Mechanism
	SetRequestMechanismPreferences(mechanismPreferences []connection.Mechanism)

	IsValid() error
}
