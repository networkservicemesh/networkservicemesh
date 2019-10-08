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
	SetID(id string)

	GetNetworkService() string
	SetNetworkService(networkService string)

	GetConnectionMechanism() Mechanism
	SetConnectionMechanism(mechanism Mechanism)

	GetContext() *connectioncontext.ConnectionContext
	SetContext(context *connectioncontext.ConnectionContext)
	UpdateContext(connectionContext *connectioncontext.ConnectionContext) error

	GetLabels() map[string]string

	GetConnectionState() State
	SetConnectionState(state State)

	GetNetworkServiceEndpointName() string

	IsValid() error
	IsComplete() error

	GetSignature() string
	SetSignature(sign string)
}
