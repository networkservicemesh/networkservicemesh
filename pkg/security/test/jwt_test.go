// Copyright (c) 2019 Cisco Systems, Inc.
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
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"
)

const numberOfProviders = 20

func TestSign(t *testing.T) {
	RegisterTestingT(t)

	msg := &testMsg{
		testAud: aud,
	}

	sc, err := newTestSecurityContext(SpiffeID1)
	Expect(err).To(BeNil())

	signature, err := security.GenerateSignature(context.Background(), msg, testClaimsSetter, sc)
	Expect(err).To(BeNil())

	// checking generated signature
	_, _, claims, err := security.ParseJWTWithClaims(signature)
	Expect(err).To(BeNil())
	Expect(claims.Audience).To(Equal(aud))

	Expect(security.VerifySignature(context.Background(), signature, sc.GetCABundle(), SpiffeID1)).To(BeNil())
}

func TestJWTChain_DistinctProviders(t *testing.T) {
	g := NewWithT(t)

	ca, err := GenerateCA()
	g.Expect(err).To(BeNil())

	providers := make([]security.Provider, 0, numberOfProviders)

	for i := 0; i < numberOfProviders; i++ {
		// all providers have different spiffeID
		sc, err := newTestSecurityContextWithCA(fmt.Sprintf("spiffe://test.com/%d", i), &ca)
		g.Expect(err).To(BeNil())

		providers = append(providers, sc)
	}

	chainRequest(g, providers)
}

func TestJWTChain_EqualPairProviders(t *testing.T) {
	g := NewWithT(t)

	ca, err := GenerateCA()
	g.Expect(err).To(BeNil())

	providers := make([]security.Provider, 0, numberOfProviders)

	for i := 0; i < numberOfProviders; i++ {
		// spiffe://test.com/0, spiffe://test.com/0, spiffe://test.com/2, spiffe://test.com/2 ...
		sc, err := newTestSecurityContextWithCA(fmt.Sprintf("spiffe://test.com/%d", i-i%2), &ca)
		g.Expect(err).To(BeNil())

		providers = append(providers, sc)
	}

	chainRequest(g, providers)
}

func TestJWTChain_RepeatedSeq(t *testing.T) {
	g := NewWithT(t)

	ca, err := GenerateCA()
	g.Expect(err).To(BeNil())

	providers := make([]security.Provider, 0, numberOfProviders)

	for i := 0; i < numberOfProviders; i++ {
		// spiffe://test.com/0, ..., spiffe://test.com/4, spiffe://test.com/0, ..., spiffe://test.com/4
		sc, err := newTestSecurityContextWithCA(fmt.Sprintf("spiffe://test.com/%d", i%5), &ca)
		logrus.Info(fmt.Sprintf("spiffe://test.com/%d", i%5))
		g.Expect(err).To(BeNil())

		providers = append(providers, sc)
	}

	chainRequest(g, providers)
}

func TestJWTChain_VPNFirewall(t *testing.T) {
	g := NewWithT(t)

	ca, err := GenerateCA()
	g.Expect(err).To(BeNil())

	ids := []string{
		"spiffe://test.com/nsc",
		"spiffe://test.com/nsmgr",
		"spiffe://test.com/nse",
		"spiffe://test.com/nsmgr",
		"spiffe://test.com/nse",
		"spiffe://test.com/nsmgr",
		"spiffe://test.com/nse",
		"spiffe://test.com/nsmgr",
		"spiffe://test.com/nse",
		"spiffe://test.com/nsmgr",
		"spiffe://test.com/nse",
		"spiffe://test.com/nsmgr",
		"spiffe://test.com/nse",
		"spiffe://test.com/nsmgr",
	}
	providers := make([]security.Provider, 0, len(ids))

	for i := 0; i < len(ids); i++ {
		sc, err := newTestSecurityContextWithCA(ids[i], &ca)
		logrus.Info(fmt.Sprintf("spiffe://test.com/%d", i%5))
		g.Expect(err).To(BeNil())

		providers = append(providers, sc)
	}

	chainRequest(g, providers)
}

// chainRequest accepts list of providers, generates signature for provider N
// using obo-signature from provider N-1, verifies signature on each iteration
func chainRequest(g *WithT, p []security.Provider) {
	msg := &testMsg{
		testAud: aud,
	}

	previousSignatureStr := ""
	previousSignature := &security.Signature{}

	for i := 0; i < len(p); i++ {
		logrus.Infof("Provider %d, spiffeID = %s", i, p[i].GetSpiffeID())

		if previousSignatureStr != "" {
			t := time.Now()
			g.Expect(security.VerifySignature(previousSignatureStr, p[i].GetCABundle(), p[i-1].GetSpiffeID())).To(BeNil())
			logrus.Infof("Perf: Validate on %d iteration: %v", i-1, time.Since(t))
		}

		t := time.Now()
		signature, err := security.GenerateSignature(context.Background(), msg, testClaimsSetter, p[i],
			security.WithObo(previousSignature))
		g.Expect(err).To(BeNil())
		logrus.Infof("Perf: Generate on %d iteration: %v, length = %d", i, time.Since(t), len(signature))

		msg.token = signature
		previousSignatureStr = signature

		err = previousSignature.Parse(signature)
		g.Expect(err).To(BeNil())
	}
}
