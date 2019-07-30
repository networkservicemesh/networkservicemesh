package main

import (
	"context"
	"errors"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/security"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/api/workload"
	proto "github.com/spiffe/spire/proto/spire/api/workload"
	"google.golang.org/grpc"
	"net"
)

type spireProxy struct {
	workloadAPIClient workload.X509Client
}

func newSpireProxy() *spireProxy {
	workloadAPIClient := workload.NewX509Client(
		&workload.X509ClientConfig{
			Addr: &net.UnixAddr{Net: "unix", Name: security.DefaultAgentAddress},
			Log:  logrus.StandardLogger(),
		})

	errCh := make(chan error)

	go func() {
		if err := workloadAPIClient.Start(); err != nil {
			errCh <- err
			return
		}
	}()

	return &spireProxy{
		workloadAPIClient: workloadAPIClient,
	}
}

func (sp *spireProxy) FetchJWTSVID(context.Context, *proto.JWTSVIDRequest) (*proto.JWTSVIDResponse, error) {
	return nil, errors.New("not supported")
}

func (sp *spireProxy) FetchJWTBundles(*proto.JWTBundlesRequest, proto.SpiffeWorkloadAPI_FetchJWTBundlesServer) error {
	return errors.New("not supported")

}

func (sp *spireProxy) ValidateJWTSVID(context.Context, *proto.ValidateJWTSVIDRequest) (*proto.ValidateJWTSVIDResponse, error) {
	return nil, errors.New("not supported")
}

func (sp *spireProxy) FetchX509SVID(request *proto.X509SVIDRequest, stream proto.SpiffeWorkloadAPI_FetchX509SVIDServer) error {
	for r := range sp.workloadAPIClient.UpdateChan() {
		err := stream.Send(r)
		if err != nil {
			logrus.Error(err)
			return err
		}
	}
	return nil
}

func main() {
	c := tools.NewOSSignalChannel()

	srv := grpc.NewServer()
	proto.RegisterSpiffeWorkloadAPIServer(srv, newSpireProxy())

	ln, err := net.Listen("tcp", "127.0.0.1:7001")
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		if err := srv.Serve(ln); err != nil {
			logrus.Fatal(err)
		}
	}()

	<-c
}
