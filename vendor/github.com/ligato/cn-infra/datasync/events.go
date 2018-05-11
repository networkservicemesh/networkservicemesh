package datasync

import (
	"github.com/golang/protobuf/proto"
)

// ChangeEvent is used to define the data type for the change channel
// (<changeChan> from KeyValProtoWatcher.Watch).
// A data change event contains a key identifying where the change happened
// and two values for data stored under that key: the value *before* the change
// (previous value) and the value *after* the change (current value).
type ChangeEvent interface {
	CallbackResult

	ProtoWatchResp
}

// ResyncEvent is used to define the data type for the resync channel
// (<resyncChan> from KeyValProtoWatcher.Watch).
type ResyncEvent interface {
	CallbackResult

	// GetValues returns key-value pairs sorted by key prefixes
	// (<keyPrefix> variable list from KeyValProtoWatcher.Watch).
	GetValues() map[ /*keyPrefix*/ string]KeyValIterator
}

// CallbackResult can be used by an event receiver to indicate to the event
// producer whether an operation was successful (error is nil) or unsuccessful
// (error is not nil).
type CallbackResult interface {
	// Done allows plugins that are processing data change/resync to send
	// feedback. If there was no error, Done(nil) needs to be called.
	// Use the noError=nil definition for better readability, for example:
	//     Done(noError).
	Done(error)
}

// ProtoWatchResp contains changed value.
type ProtoWatchResp interface {
	ChangeValue
	WithKey
	WithPrevValue
}

// ChangeValue represents a single propagated change.
type ChangeValue interface {
	LazyValueWithRev
	WithChangeType
}

// LazyValueWithRev defines value that is unmarshaled into proto message
// on demand with a revision.
type LazyValueWithRev interface {
	LazyValue
	WithRevision
}

// WithKey is a simple helper interface embedded by all interfaces that require
// access to the key of the key-value pair.
// The intent is to ensure that the same method declaration is used in different
// interfaces (composition of interfaces).
type WithKey interface {
	// GetKey returns the key of the pair
	GetKey() string
}

// WithChangeType is a simple helper interface embedded by all interfaces that
// require access to change type information.
// The intent is to ensure that the same method declaration is used in different
// interfaces (composition of interfaces).
type WithChangeType interface {
	GetChangeType() PutDel
}

// WithRevision is a simple helper interface embedded by all interfaces that
// require access to the value revision.
// The intent is to ensure that the same method declaration is used in different
// interfaces (composition of interfaces).
type WithRevision interface {
	// GetRevision gets revision of current value
	GetRevision() (rev int64)
}

// WithPrevValue is a simple helper interface embedded by all interfaces that
// require access to the previous value.
// The intent is to ensure that the same method declaration is used in different
// interfaces (composition of interfaces).
type WithPrevValue interface {
	// GetPrevValue gets the previous value in the data change event.
	// The caller must provide an address of a proto message buffer
	// as <prevValue>.
	// returns:
	// - <prevValueExist> flag is set to 'true' if previous value does exist
	// - error if <prevValue> can not be properly filled
	GetPrevValue(prevValue proto.Message) (prevValueExist bool, err error)
}

// LazyValue defines value that is unmarshaled into proto message on demand.
// The reason for defining interface with only one method is primarily to unify
// interfaces in this package.
type LazyValue interface {
	// GetValue gets the current value in the data change event.
	// The caller must provide an address of a proto message buffer
	// as <value>.
	// returns:
	// - error if value argument can not be properly filled.
	GetValue(value proto.Message) error
}
