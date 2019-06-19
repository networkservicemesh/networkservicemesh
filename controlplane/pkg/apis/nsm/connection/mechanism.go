package connection

// MechanismType is an unified interface for local/remote connection mechanism type
type MechanismType interface {
	IsRemote() bool
}

// Mechanism is an unified interface for local/remote connection mechanism
type Mechanism interface {
	IsRemote() bool

	Equals(mechanism Mechanism) bool
	Clone() Mechanism

	GetMechanismType() MechanismType
	SetMechanismType(mechanismType MechanismType)

	GetParameters() map[string]string
	SetParameters(parameters map[string]string)
}
