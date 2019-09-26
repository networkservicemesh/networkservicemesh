package livemonitor

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/pkg/livemonitor/api"
)

// Server is an interface for GRPC monitoring API server
type Server interface {
	MonitorLiveness(req *empty.Empty, srv api.LivenessMonitor_MonitorLivenessServer) error
}

type server struct {
	api.LivenessMonitorServer
}

// NewServer creates a new Live Monitor Server
func NewServer() Server {
	return &server{}
}

func (*server) MonitorLiveness(req *empty.Empty, srv api.LivenessMonitor_MonitorLivenessServer) error {
	<-srv.Context().Done()
	return nil
}

// RegisterLivenessMonitorServer register Liveness Monitor Server on grpc server
func RegisterLivenessMonitorServer(s *grpc.Server, srv api.LivenessMonitorServer) {
	api.RegisterLivenessMonitorServer(s, srv)
	logrus.Infof("Liveness Monitor grpc Server started")
}
