package converter

import (
	"github.com/ligato/vpp-agent/api/configurator"
)

type Converter interface {
	ToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error)
}

type CrossConnectConversionParameters struct {
	BaseDir string
}

type ConnectionContextSide int

const (
	NEITHER ConnectionContextSide = iota + 1
	SOURCE
	DESTINATION
)

type ConnectionConversionParameters struct {
	Terminate bool
	Side      ConnectionContextSide
	Name      string
	BaseDir   string
}
