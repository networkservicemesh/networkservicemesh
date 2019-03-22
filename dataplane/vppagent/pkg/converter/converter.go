package converter

import "github.com/ligato/vpp-agent/plugins/vpp/model/rpc"

type Converter interface {
	ToDataRequest(rv *rpc.DataRequest, connect bool) (*rpc.DataRequest, error)
}

type IfaceNameProvider interface {
	GetIfaceName(id string) string
}

type CrossConnectConversionParameters struct {
	BaseDir           string
	IfaceNameProvider IfaceNameProvider
}

type ConnectionContextSide int

const (
	NEITHER ConnectionContextSide = iota + 1
	SOURCE
	DESTINATION
)

type ConnectionConversionParameters struct {
	Terminate         bool
	Side              ConnectionContextSide
	Name              string
	BaseDir           string
	IfaceNameProvider IfaceNameProvider
}
