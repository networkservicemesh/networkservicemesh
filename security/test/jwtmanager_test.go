package test

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/security"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"net"
	"testing"
	"time"
)

func TestJwtManager_Verify(t *testing.T) {
	RegisterTestingT(t)

	p1, p2, err := createParties()
	Expect(err).To(BeNil())

	tokenString, err := p1.GenerateJWT("myNS", "")
	Expect(err).To(BeNil())

	err = p2.VerifyJWT(testSpiffeID, tokenString)
	Expect(err).To(BeNil())
}

func TestClientServerJWTValidation(t *testing.T) {
	RegisterTestingT(t)

	p1, p2, err := createParties()
	Expect(err).To(BeNil())

	srv, err := p2.NewServer()
	Expect(err).To(BeNil())

	helloworld.RegisterGreeterServer(srv, &helloSrv{})

	ln, err := net.Listen("tcp", ":3434")
	Expect(err).To(BeNil())
	defer ln.Close()
	go srv.Serve(ln)

	secureConn, err := p1.DialContext(context.Background(), ":3434")
	Expect(err).To(BeNil())
	defer secureConn.Close()

	token, err := p1.GenerateJWT("myns", "")
	Expect(err).To(BeNil())

	gc := helloworld.NewGreeterClient(secureConn)
	response, err := gc.SayHello(
		context.Background(),
		&helloworld.HelloRequest{Name: "testName"},
		grpc.PerRPCCredentials(&security.NSMToken{Token: token}))

	Expect(err).To(BeNil())
	Expect(response.Message).To(Equal("testName"))

	_, err = grpc.DialContext(context.Background(), ":3434")
	Expect(err).ToNot(BeNil())
}

func TestClientServerExchangeCertificatesJWT(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	obt1, err := newExchangeCertObtainerWithCA(&ca, frequency)
	Expect(err).To(BeNil())

	mgr1 := security.NewManagerWithCertObtainer(obt1, helloworldAud)

	obt2, err := newExchangeCertObtainerWithCA(&ca, frequency)
	Expect(err).To(BeNil())

	mgr2 := security.NewManagerWithCertObtainer(obt2, helloworldAud)

	srv, err := mgr2.NewServer()
	Expect(err).To(BeNil())

	helloworld.RegisterGreeterServer(srv, &helloSrv{})

	ln, err := net.Listen("tcp", ":3434")
	Expect(err).To(BeNil())
	defer ln.Close()
	go srv.Serve(ln)

	secureConn, err := mgr1.DialContext(context.Background(), ":3434")
	Expect(err).To(BeNil())
	defer secureConn.Close()

	gc := helloworld.NewGreeterClient(secureConn)
	for i := 0; i < 3; i++ {
		token, err := mgr1.GenerateJWT("myns", "")
		Expect(err).To(BeNil())

		response, err := gc.SayHello(
			context.Background(),
			&helloworld.HelloRequest{Name: "testName"},
			grpc.PerRPCCredentials(&security.NSMToken{Token: token}))
		if err != nil {
			logrus.Error(err)
			<-time.After(3 * time.Second)
			continue
		}

		logrus.Infof("SayHello successfully finished, response = %v", response)
		Expect(response.Message).To(Equal("testName"))
		<-time.After(3 * time.Second)
	}
}

func TestOboToken(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	var current security.Manager
	var jwt string

	for i := 0; i < 10; i++ {
		current, err = createPartyWithCA(fmt.Sprintf("spiffe://test.com/p%d", i), ca)
		Expect(err).To(BeNil())

		if jwt != "" {
			err = current.VerifyJWT(fmt.Sprintf("spiffe://test.com/p%d", i-1), jwt)
			Expect(err).To(BeNil())
		}

		jwt, err = current.GenerateJWT("networkservice", jwt)
		Expect(err).To(BeNil())
	}
}

func TestClientServerOboToken(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	p1, err := createGreeterParty("spiffe://test.com/p1", 3431, &ca, ":3432", "1")
	Expect(err).To(BeNil())
	defer p1.Close()

	p2, err := createGreeterParty("spiffe://test.com/p2", 3432, &ca, ":3433", "2")
	Expect(err).To(BeNil())
	defer p2.Close()

	p3, err := createGreeterParty("spiffe://test.com/p3", 3433, &ca, "", "3")
	Expect(err).To(BeNil())
	defer p3.Close()

	reply, err := p1.SayHello("", ":3431", "")
	Expect(err).To(BeNil())
	Expect(reply.Message).To(Equal("123"))
}

//func TestServerStreamSecurity(t *testing.T) {
//	RegisterTestingT(t)
//
//	ca, err := generateCA()
//	Expect(err).To(BeNil())
//
//	p, err := createParty("spiffe://test.com/p1", 3431, &ca)
//	Expect(err).To(BeNil())
//
//	connection.RegisterMonitorConnectionServer(p.srv, &testConnectionMonitor{})
//
//	obt, err := newSimpleCertObtainerWithCA("spiffe://test.com/p2", &ca)
//	mgr2 := security.NewManagerWithCertObtainer(obt, )
//}
