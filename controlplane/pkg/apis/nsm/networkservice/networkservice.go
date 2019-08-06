package networkservice

import "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"

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

// Reply is an unified interface for local/remote network replies
type Reply interface {
	GetReplyConnection() connection.Connection
	GetResponseJWT() string
	Clone() Reply
}
