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

package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/spiffe"
)

const (
	// SpireAgentUnixSocket points to unix socket used by default
	SpireAgentUnixSocket = "/run/spire/sockets/agent.sock"

	// SpireAgentUnixAddr is unix socket address with specified scheme
	SpireAgentUnixAddr = "unix://" + SpireAgentUnixSocket
)

type spireProvider struct {
	peer *spiffe.TLSPeer
}

func NewSpireProvider(addr string) (Provider, error) {
	p, err := spiffe.NewTLSPeer(spiffe.WithWorkloadAPIAddr(addr))
	if err != nil {
		return nil, err
	}

	go func() {
		if err := p.WaitUntilReady(context.Background()); err != nil {
			logrus.Info(err)
			return
		}

		tlscrt, err := p.GetCertificate()
		if err != nil {
			logrus.Info(err)
			return
		}

		x509crt, err := x509.ParseCertificate(tlscrt.Certificate[0])
		if err != nil {
			logrus.Info(err)
			return
		}

		logrus.Infof("Issued certificate with id: %v", x509crt.URIs[0])
	}()

	return &spireProvider{
		peer: p,
	}, nil
}

func (p *spireProvider) GetTLSConfig(ctx context.Context) (*tls.Config, error) {
	return p.peer.GetConfig(ctx, spiffe.ExpectAnyPeer())
}
