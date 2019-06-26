package test

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/security"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
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

func TestClientServerExchangeCertificatesJWT(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	p1, err := createExchangeIntermediary("spiffe://test.com/p1", 3231, &ca, "", "1")
	Expect(err).To(BeNil())
	defer p1.Close()

	p2, err := createExchangeIntermediary("spiffe://test.com/p2", 3232, &ca, "", "2")
	Expect(err).To(BeNil())
	defer p2.Close()

	client, err := p1.NewClient(":3232")
	Expect(err).To(BeNil())
	for i := 0; i < 5; i++ {
		response, err := client.Request(context.Background(), emptyRequest())
		Expect(err).To(BeNil())
		logrus.Info(response)
		<-time.After(1 * time.Second)
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

	p1, err := createSimpleIntermediary("spiffe://test.com/p1", 3431, &ca, ":3432", "1")
	Expect(err).To(BeNil())
	defer p1.Close()

	p2, err := createSimpleIntermediary("spiffe://test.com/p2", 3432, &ca, ":3433", "2")
	Expect(err).To(BeNil())
	defer p2.Close()

	p3, err := createSimpleIntermediary("spiffe://test.com/p3", 3433, &ca, "", "3")
	Expect(err).To(BeNil())
	defer p3.Close()

	client, err := p1.NewClient(":3431")
	Expect(err).To(BeNil())

	response, err := client.Request(context.Background(), emptyRequest())
	Expect(err).To(BeNil())

	logrus.Info(response.GetLabels())
	Expect(response.GetLabels()["2"]).To(Equal("spiffe://test.com/p1"))
	Expect(response.GetLabels()["3"]).To(Equal("spiffe://test.com/p2"))
	Expect(p1.VerifyJWT("spiffe://test.com/p1", response.GetResponseJWT())).To(BeNil())
}
