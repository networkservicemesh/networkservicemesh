package test

import (
	"crypto/x509"
	"github.com/networkservicemesh/networkservicemesh/security"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"net"
	"testing"
)

func TestSimpleCertCreation(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	caX509, err := x509.ParseCertificate(ca.Certificate[0])
	Expect(err).To(BeNil())

	roots := x509.NewCertPool()
	roots.AddCert(caX509)

	crt, err := generateKeyPair(testSpiffeID, &ca)
	Expect(err).To(BeNil())

	err = verify(&crt, roots)
	Expect(err).To(BeNil())
}

func TestSimpleCertObtainer(t *testing.T) {
	RegisterTestingT(t)

	obt, err := newSimpleCertObtainer(testSpiffeID)
	Expect(err).To(BeNil())

	mgr := security.NewManagerWithCertObtainer(obt)

	crt := mgr.GetCertificate()
	ca := mgr.GetCABundle()
	verify(crt, ca)
}

func TestClientServer(t *testing.T) {
	RegisterTestingT(t)

	obt, err := newSimpleCertObtainer(testSpiffeID)
	Expect(err).To(BeNil())

	mgr := security.NewManagerWithCertObtainer(obt)

	srv, err := mgr.NewServer()
	Expect(err).To(BeNil())

	helloworld.RegisterGreeterServer(srv, &helloSrv{})

	ln, err := net.Listen("tcp", ":3434")
	Expect(err).To(BeNil())
	defer ln.Close()
	go srv.Serve(ln)

	secureConn, err := mgr.DialContext(context.Background(), ":3434")
	Expect(err).To(BeNil())
	defer secureConn.Close()

	gc := helloworld.NewGreeterClient(secureConn)
	response, err := gc.SayHello(context.Background(), &helloworld.HelloRequest{Name: "testName"})
	Expect(err).To(BeNil())
	Expect(response.Message).To(Equal("testName"))

	_, err = grpc.DialContext(context.Background(), ":3434")
	Expect(err).ToNot(BeNil())
}

//func TestExchangeCertObtainer(t *testing.T) {
//	RegisterTestingT(t)
//
//	ca, err := generateCA()
//	Expect(err).To(BeNil())
//
//	obt1, err := newExchangeCertObtainerWithCA(&ca, frequency)
//	Expect(err).To(BeNil())
//
//	mgr1 := security.NewManagerWithCertObtainer(obt1)
//
//	obt2, err := newExchangeCertObtainerWithCA(&ca, frequency)
//	Expect(err).To(BeNil())
//
//	mgr2 := security.NewManagerWithCertObtainer(obt2)
//
//	srv, err := mgr2.NewServer()
//	Expect(err).To(BeNil())
//
//	helloworld.RegisterGreeterServer(srv, &helloSrv{
//		p: mgr2,
//	})
//
//	ln, err := net.Listen("tcp", ":3434")
//	Expect(err).To(BeNil())
//	defer ln.Close()
//	go srv.Serve(ln)
//
//	secureConn, err := mgr1.DialContext(context.Background(), ":3434")
//	Expect(err).To(BeNil())
//	defer secureConn.Close()
//
//	gc := helloworld.NewGreeterClient(secureConn)
//	for i := 0; i < 3; i++ {
//		response, err := gc.SayHello(context.Background(), &helloworld.HelloRequest{Name: "testName"})
//		Expect(err).To(BeNil())
//		logrus.Infof("SayHello successfully finished, response = %v", response)
//		Expect(response.Message).To(Equal("testName"))
//		<-time.After(4 * time.Second)
//	}
//}
