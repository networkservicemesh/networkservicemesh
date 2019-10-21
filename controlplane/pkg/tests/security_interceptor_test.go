package tests

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/security"
	testsec "github.com/networkservicemesh/networkservicemesh/pkg/security/test"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

func testNewServerFunc(provider security.Provider) tools.NewServerFunc {
	cfg := &tools.DialConfig{SecurityProvider: provider, OpenTracing: false}
	return new(tools.NewServerBuilder).WithConfig(cfg).NewServerFunc()
}

func testDialFunc(provider security.Provider) tools.DialContextFunc {
	cfg := &tools.DialConfig{SecurityProvider: provider, OpenTracing: false}
	return new(tools.DialBuilder).TCP().WithConfig(cfg).DialContextFunc()
}

type dummyNetworkService struct {
	dialContextFunc tools.DialContextFunc
	newServerFunc   tools.NewServerFunc
	provider        security.Provider
}

func newDummyNetworkService(provider security.Provider) *dummyNetworkService {
	return &dummyNetworkService{
		newServerFunc:   testNewServerFunc(provider),
		dialContextFunc: testDialFunc(provider),
		provider:        provider,
	}
}

func (d *dummyNetworkService) start() (closeFunc func(), err error) {
	ln, err := net.Listen("tcp", "localhost:5252")
	if err != nil {
		return nil, err
	}

	srv := d.newServerFunc(context.Background())
	networkservice.RegisterNetworkServiceServer(srv, d)

	go func() {
		if err := srv.Serve(ln); err != nil {
			logrus.Error(err)
		}
	}()

	return func() { srv.Stop() }, nil
}

func (d *dummyNetworkService) Request(context.Context, *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	rv := &connection.Connection{Id: "1"}
	sign, err := security.GenerateSignature(rv, common.ConnectionFillClaimsFunc, d.provider)
	if err != nil {
		return nil, err
	}
	rv.ResponseJWT = sign
	return rv, nil
}

func (d *dummyNetworkService) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	panic("implement me")
}

func newTestSecurityProvider(ca *tls.Certificate, spiffeID string) security.Provider {
	obt := testsec.NewTestCertificateObtainerWithCA(spiffeID, ca, 1*time.Second)
	return security.NewProviderWithCertObtainer(obt)
}

func TestSecurityInterceptor(t *testing.T) {
	g := NewWithT(t)

	ca, err := testsec.GenerateCA()
	g.Expect(err).To(BeNil())

	srv := newDummyNetworkService(newTestSecurityProvider(&ca, testsec.SpiffeID2))
	closeFunc, err := srv.start()
	g.Expect(err).To(BeNil())
	defer closeFunc()

	clientProvider := newTestSecurityProvider(&ca, testsec.SpiffeID1)
	dialContextFunc := testDialFunc(clientProvider)
	conn, err := dialContextFunc(context.Background(), "localhost:5252")

	g.Expect(err).To(BeNil())
	defer func() { _ = conn.Close() }()

	nsclient := networkservice.NewNetworkServiceClient(conn)
	reply, err := nsclient.Request(context.Background(), &networkservice.NetworkServiceRequest{})
	g.Expect(err).To(BeNil())

	logrus.Info("validating responseJWT")
	err = security.VerifySignature(reply.ResponseJWT, clientProvider.GetCABundle(), testsec.SpiffeID2)
	g.Expect(err).To(BeNil())
}
