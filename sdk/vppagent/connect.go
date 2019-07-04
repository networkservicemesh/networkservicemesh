package vppagent

import "github.com/ligato/vpp-agent/api/configurator"

// ConnectionData is a connection data type
type ConnectionData struct {
	InConnName  string
	OutConnName string
	DataChange  *configurator.Config
}
