package testsec

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/networkservicemesh/networkservicemesh/security"
	"google.golang.org/grpc"
)

type stubManager struct {
}

func (*stubManager) DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	return grpc.DialContext(ctx, target, append(opts, grpc.WithInsecure())...)
}

func (*stubManager) GenerateJWT(networkService, obo string) (string, error) {
	return "", nil
}

func (*stubManager) GetCABundle() *x509.CertPool {
	return nil
}

func (*stubManager) GetCertificate() *tls.Certificate {
	return nil
}

func (*stubManager) GetSpiffeID() string {
	return ""
}

func (*stubManager) NewServer(opts ...grpc.ServerOption) *grpc.Server {
	return grpc.NewServer(opts...)
}

func (*stubManager) SignResponse(resp interface{}, obo string) error {
	return nil
}

func (*stubManager) VerifyJWT(spiffeID, tokeString string) error {
	return nil
}

// NewStubSecurityManager create stubbed Manager
func NewStubSecurityManager() security.Manager {
	return &stubManager{}
}
