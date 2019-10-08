package testsec

import (
	"github.com/networkservicemesh/networkservicemesh/pkg/security"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSign(t *testing.T) {
	RegisterTestingT(t)

	msg := &testMsg{
		testAud: aud,
	}

	sc, err := newTestSecurityContext(spiffeID1)
	Expect(err).To(BeNil())

	signature, err := security.GenerateSignature(msg, testClaimsSetter, sc)
	Expect(err).To(BeNil())

	// checking generated signature
	_, _, claims, err := security.ParseJWTWithClaims(signature)
	Expect(err).To(BeNil())
	Expect(claims.Audience).To(Equal(aud))

	Expect(security.VerifySignature(signature, sc.GetCABundle(), spiffeID1)).To(BeNil())
}

func TestChain(t *testing.T) {
	RegisterTestingT(t)

	msg := &testMsg{
		testAud: aud,
	}

	ca, err := generateCA()
	Expect(err).To(BeNil())

	sc1, err := newTestSecurityContextWithCA(spiffeID1, &ca)
	Expect(err).To(BeNil())

	signature, err := security.GenerateSignature(msg, testClaimsSetter, sc1)
	Expect(err).To(BeNil())

	sc2, err := newTestSecurityContextWithCA(spiffeID2, &ca)
	Expect(err).To(BeNil())

	signature2, err := security.GenerateSignature(msg, testClaimsSetter, sc2, security.WithObo(signature))
	Expect(err).To(BeNil())

	sc3, err := newTestSecurityContextWithCA(spiffeID3, &ca)
	Expect(err).To(BeNil())

	signature3, err := security.GenerateSignature(msg, testClaimsSetter, sc3, security.WithObo(signature2))
	msg.token = signature3
	Expect(err).To(BeNil())

	// checking generated signature
	_, _, claims, err := security.ParseJWTWithClaims(signature3)
	Expect(err).To(BeNil())
	Expect(claims.Audience).To(Equal(aud))

	Expect(security.VerifySignature(signature3, sc3.GetCABundle(), spiffeID3)).To(BeNil())
}
