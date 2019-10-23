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
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"
)

func TestSign(t *testing.T) {
	RegisterTestingT(t)

	msg := &testMsg{
		testAud: aud,
	}

	sc, err := newTestSecurityContext(SpiffeID1)
	Expect(err).To(BeNil())

	signature, err := security.GenerateSignature(msg, testClaimsSetter, sc)
	Expect(err).To(BeNil())

	// checking generated signature
	_, _, claims, err := security.ParseJWTWithClaims(signature)
	Expect(err).To(BeNil())
	Expect(claims.Audience).To(Equal(aud))

	Expect(security.VerifySignature(signature, sc.GetCABundle(), SpiffeID1)).To(BeNil())
}

func TestChain(t *testing.T) {
	RegisterTestingT(t)

	msg := &testMsg{
		testAud: aud,
	}

	ca, err := GenerateCA()
	Expect(err).To(BeNil())

	sc1, err := newTestSecurityContextWithCA(SpiffeID1, &ca)
	Expect(err).To(BeNil())

	signature, err := security.GenerateSignature(msg, testClaimsSetter, sc1)
	Expect(err).To(BeNil())

	sc2, err := newTestSecurityContextWithCA(SpiffeID2, &ca)
	Expect(err).To(BeNil())

	signature2, err := security.GenerateSignature(msg, testClaimsSetter, sc2,
		security.WithObo(&security.TokenAndClaims{Token: signature}))
	Expect(err).To(BeNil())

	sc3, err := newTestSecurityContextWithCA(SpiffeID3, &ca)
	Expect(err).To(BeNil())

	signature3, err := security.GenerateSignature(msg, testClaimsSetter, sc3,
		security.WithObo(&security.TokenAndClaims{Token: signature2}))
	msg.token = signature3
	Expect(err).To(BeNil())

	// checking generated signature
	_, _, claims, err := security.ParseJWTWithClaims(signature3)
	Expect(err).To(BeNil())
	Expect(claims.Audience).To(Equal(aud))

	Expect(security.VerifySignature(signature3, sc3.GetCABundle(), SpiffeID3)).To(BeNil())
}
