package main

import (
	"context"
	"net"

	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
	proto "github.com/spiffe/go-spiffe/proto/spiffe/workload"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

type spireProxy struct {
	workloadAPIClient proto.SpiffeWorkloadAPIClient
	closeFunc         func() error
}

func newSpireProxy() (*spireProxy, error) {
	cc, err := tools.DialUnixInsecure(security.SpireAgentUnixSocket)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	workloadAPIClient := proto.NewSpiffeWorkloadAPIClient(cc)
	return &spireProxy{
		workloadAPIClient: workloadAPIClient,
		closeFunc:         func() error { return cc.Close() },
	}, nil
}

func (sp *spireProxy) Close() error {
	return sp.closeFunc()
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
	logrus.Info("FetchX509SVID called...")

	header := metadata.Pairs("workload.spiffe.io", "true")
	grpcCtx := metadata.NewOutgoingContext(stream.Context(), header)

	c, err := sp.workloadAPIClient.FetchX509SVID(grpcCtx, request)
	if err != nil {
		logrus.Error(err)
		return err
	}

	for {
		if err := stream.Context().Err(); err != nil {
			logrus.Error(err)
			return err
		}

		msg, err := c.Recv()
		if err != nil {
			logrus.Error(err)
			return err
		}

		logrus.Infof("recv msg: %v", msg)

		if err := stream.Send(msg); err != nil {
			logrus.Error(err)
			return err
		}

		logrus.Infof("sent msg: %v", msg)
	}
}

func main() {
	logrus.Infof("Spire Proxy started...")
	utils.PrintAllEnv(logrus.StandardLogger())
	c := tools.NewOSSignalChannel()
	srv := grpc.NewServer()

	proxy, err := newSpireProxy()
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() { _ = proxy.Close() }()

	proto.RegisterSpiffeWorkloadAPIServer(srv, proxy)

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
