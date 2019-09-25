package livemonitor

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/pkg/livemonitor/api"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Server is an unified interface for GRPC monitoring API server
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
	<- srv.Context().Done()
	return nil
}

func RegisterLivenessMonitorServer(s *grpc.Server, srv api.LivenessMonitorServer){
	api.RegisterLivenessMonitorServer(s, srv)
	logrus.Infof("Liveness Monitor GRPC Server started")
}
