package converter

import (
	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/common"
)

type Converter interface {
	ToDataRequest(rv *configurator.Config, connect bool) (*configurator.Config, error)
}

type CrossConnectConversionParameters struct {
	BaseDir string
	Routes *ExtraRoutesParameters
	EgressInterface common.EgressInterface
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

type ExtraRoutesParameters struct {
	Routes []string // Extra destination routes required
}
