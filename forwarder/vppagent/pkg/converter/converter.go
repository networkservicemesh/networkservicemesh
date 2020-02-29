package converter

import "go.ligato.io/vpp-agent/v3/proto/ligato/configurator"

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
