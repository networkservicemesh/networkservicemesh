package converter

import "github.com/ligato/vpp-agent/plugins/vpp/model/rpc"

type Converter interface {
	ToDataRequest(*rpc.DataRequest) (*rpc.DataRequest, error)
}
