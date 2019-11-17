// Copyright (c) 2019 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testsec

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/big"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"
)

const (
	aud            = "testaud"
	testDomain     = "test"
	spiffeIDFormat = "spiffe://%s/testID_%d"
	nameTypeURI    = 6
)

var (
	SpiffeID1 = fmt.Sprintf(spiffeIDFormat, testDomain, 1)
	SpiffeID2 = fmt.Sprintf(spiffeIDFormat, testDomain, 2)
	SpiffeID3 = fmt.Sprintf(spiffeIDFormat, testDomain, 3)
)

type testMsg struct {
	testAud string
	token   string
}

func (m *testMsg) GetSignature() string {
	return m.token
}

func testFillClaimsFunc(claims *security.ChainClaims, msg interface{}) error {
	claims.Audience = msg.(*testMsg).testAud
	return nil
}

type testSecurityProvider struct {
	ca       *x509.CertPool
	cert     *tls.Certificate
	spiffeID string
}

func newTestSecurityProvider(spiffeID string) (*testSecurityProvider, error) {
	ca, err := GenerateCA()
	if err != nil {
		return nil, err
	}
	return NewTestSecurityProviderWithCA(spiffeID, &ca)
}

func NewTestSecurityProviderWithCA(spiffeID string, ca *tls.Certificate) (*testSecurityProvider, error) {
	caX509, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	cpool := x509.NewCertPool()
	cpool.AddCert(caX509)

	cert, err := generateKeyPair(spiffeID, testDomain, ca)
	if err != nil {
		return nil, err
	}

	return &testSecurityProvider{
		ca:       cpool,
		cert:     &cert,
		spiffeID: spiffeID,
	}, nil
}

func (sc *testSecurityProvider) GetCertificate(ctx context.Context) (*tls.Certificate, error) {
	logrus.Info("GetCertificate")
	return sc.cert, nil
}

func (sc *testSecurityProvider) GetRootCA(ctx context.Context) (*x509.CertPool, error) {
	return sc.ca, nil
}

func (sc *testSecurityProvider) GetID(ctx context.Context) (string, error) {
	return sc.spiffeID, nil
}

func (sc *testSecurityProvider) GetTLSConfig(ctx context.Context) (*tls.Config, error) {
	logrus.Info("GetTLSConfig")
	return &tls.Config{
		ClientAuth:         tls.RequireAnyClientCert,
		InsecureSkipVerify: true,
		GetCertificate: func(*tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
			return sc.GetCertificate(context.Background())
		},
		GetClientCertificate: func(*tls.CertificateRequestInfo) (certificate *tls.Certificate, e error) {
			return sc.GetCertificate(context.Background())
		},
	}, nil
}

func GenerateCA() (tls.Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		BasicConstraintsValid: true,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	pub := &priv.PublicKey

	certBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	return tls.X509KeyPair(certPem, keyPem)
}

func marshalSAN(spiffeID string) ([]byte, error) {
	return asn1.Marshal([]asn1.RawValue{{Tag: nameTypeURI, Class: 2, Bytes: []byte(spiffeID)}})
}

func generateKeyPair(spiffeID, domain string, caTLS *tls.Certificate) (tls.Certificate, error) {
	san, err := marshalSAN(spiffeID)
	if err != nil {
		return tls.Certificate{}, err
	}

	oidSanExtension := []int{2, 5, 29, 17}
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
			CommonName:    domain,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    oidSanExtension,
				Value: san,
			},
		},
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		KeyUsage:           x509.KeyUsageDigitalSignature,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	pub := &priv.PublicKey

	ca, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return tls.Certificate{}, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, pub, caTLS.PrivateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return tls.X509KeyPair(certPem, keyPem)
}
