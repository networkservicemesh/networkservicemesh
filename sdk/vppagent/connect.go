package vppagent

import "github.com/ligato/vpp-agent/api/configurator"

type ConnectionData struct {
	InterfaceName string
	DataChange    *configurator.Config
}
