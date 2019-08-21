package plugins

import (
	"github.com/gogo/protobuf/proto"

	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
)

// NewRequestWrapper creates a RequestWrapper instance
func NewRequestWrapper(req networkservice.Request) *RequestWrapper {
	w := &RequestWrapper{}
	w.SetRequest(req)
	return w
}

// GetRequest returns request
func (w *RequestWrapper) GetRequest() networkservice.Request {
	if w.GetLocalRequest() != nil {
		return w.GetLocalRequest()
	}
	return w.GetRemoteRequest()
}

// SetRequest sets request
func (w *RequestWrapper) SetRequest(req networkservice.Request) {
	if req.IsRemote() {
		w.Impl = &RequestWrapper_RemoteRequest{
			RemoteRequest: req.(*remote.NetworkServiceRequest),
		}
	} else {
		w.Impl = &RequestWrapper_LocalRequest{
			LocalRequest: req.(*local.NetworkServiceRequest),
		}
	}
}

// Clone clones wrapper
func (w *RequestWrapper) Clone() *RequestWrapper {
	return proto.Clone(w).(*RequestWrapper)
}
