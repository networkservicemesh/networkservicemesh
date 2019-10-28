package connection

import "github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"

// State is a enum for connection state
type State int8

const (
	// StateUp means that connection is up
	StateUp State = 0
	// StateDown means that connection is down
	StateDown State = 1
)

// Connection is an unified interface for local/remote connections
type Connection interface {
	IsRemote() bool

	Equals(connection Connection) bool
	Clone() Connection

	GetId() string
	GetNetworkService() string
	GetConnectionMechanism() Mechanism
	GetContext() *connectioncontext.ConnectionContext
	GetLabels() map[string]string
	GetConnectionState() State
	GetNetworkServiceEndpointName() string

	IsValid() error
	IsComplete() error
}
