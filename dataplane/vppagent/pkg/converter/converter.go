package converter

import "github.com/ligato/vpp-agent/plugins/vpp/model/rpc"

type Converter interface {
	ToDataRequest(rv *rpc.DataRequest, connect bool) (*rpc.DataRequest, error)
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
