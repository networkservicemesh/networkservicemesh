package security

import (
	"context"
	"crypto/tls"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"net"
	"testing"
)

type party struct {
	crtMgr Manager
	//jwtMgr JWTManager
}

func createParty() (*party, error) {
	ca, err := generateCA()
	if err != nil {
		return nil, err
	}
	return createPartyWithCA(ca)
}

func createPartyWithCA(caTLS tls.Certificate) (*party, error) {
	obt, err := newTestCertificateObtainerWithCA(caTLS)
	if err != nil {
		return nil, err
	}
	mgr := NewManagerWithCertObtainer(obt)
	//jwtMgr := NewJWTManager(mgr)

	return &party{
		crtMgr: mgr,
		//jwtMgr: jwtMgr,
	}, nil
}

func createParties() (*party, *party, error) {
	ca, err := generateCA()
	if err != nil {
		return nil, nil, err
	}

	p1, err := createPartyWithCA(ca)
	if err != nil {
		return nil, nil, err
	}

	p2, err := createPartyWithCA(ca)
	if err != nil {
		return nil, nil, err
	}

	return p1, p2, nil
}

func TestJwtManager_Verify(t *testing.T) {
	RegisterTestingT(t)

	p1, p2, err := createParties()
	Expect(err).To(BeNil())

	tokenString, err := p1.crtMgr.GenerateJWT(testSpiffeID, "myNS", nil)
	Expect(err).To(BeNil())

	err = p2.crtMgr.VerifyJWT(tokenString)
	Expect(err).To(BeNil())
}

type jwtHelloSrv struct {
	p *party
}

func (s *jwtHelloSrv) SayHello(ctx context.Context, r *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: r.GetName()}, nil
}

type jwtToken struct {
	tokenString string
}

func (t *jwtToken) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": t.tokenString,
	}, nil
}

func (t *jwtToken) RequireTransportSecurity() bool {
	return true
}

func TestClientServerJWTValidation(t *testing.T) {
	RegisterTestingT(t)

	p1, p2, err := createParties()
	Expect(err).To(BeNil())

	srv, err := p2.crtMgr.NewServer()
	Expect(err).To(BeNil())

	helloworld.RegisterGreeterServer(srv, &jwtHelloSrv{
		p: p2,
	})

	ln, err := net.Listen("tcp", ":3434")
	Expect(err).To(BeNil())
	defer ln.Close()
	go srv.Serve(ln)

	secureConn, err := p1.crtMgr.DialContext(context.Background(), ":3434")
	Expect(err).To(BeNil())
	defer secureConn.Close()

	token, err := p1.crtMgr.GenerateJWT(testSpiffeID, "myns", nil)
	Expect(err).To(BeNil())

	gc := helloworld.NewGreeterClient(secureConn)
	response, err := gc.SayHello(
		context.Background(),
		&helloworld.HelloRequest{Name: "testName"},
		grpc.PerRPCCredentials(&jwtToken{tokenString: token}))

	Expect(err).To(BeNil())
	Expect(response.Message).To(Equal("testName"))

	_, err = grpc.DialContext(context.Background(), ":3434")
	Expect(err).ToNot(BeNil())
}
