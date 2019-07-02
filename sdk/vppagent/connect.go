package vppagent

import "github.com/ligato/vpp-agent/api/configurator"

// ConnectionData is a connection data type
type ConnectionData struct {
	SrcName    string
	DstName    string
	DataChange *configurator.Config
}

// ConnectionSide is a connection side type
type ConnectionSide int

const (
	// SOURCE is a source connection side type
	SOURCE ConnectionSide = iota
	// DESTINATION is a destination connection side type
	DESTINATION
)
