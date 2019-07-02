package vppagent

import "github.com/ligato/vpp-agent/api/configurator"

type ConnectionData struct {
	SrcName    string
	DstName    string
	DataChange *configurator.Config
}

type ConnectionSide int

const (
	SOURCE ConnectionSide = iota
	DESTINATION
)
