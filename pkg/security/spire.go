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
	"net/url"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/spiffe/go-spiffe/spiffe"
)

const (
	// SpireAgentUnixSocket points to unix socket used by default
	SpireAgentUnixSocket = "/run/spire/sockets/agent.sock"

	// SpireAgentUnixAddr is unix socket address with specified scheme
	SpireAgentUnixAddr = "unix://" + SpireAgentUnixSocket
)

type TokenConfig interface {
	FillClaims(claims *ChainClaims, msg interface{}) error
	RequestFilter(req interface{}) bool
}

type spireProvider struct {
	sync.RWMutex
	spiffeID    *url.URL
	trustDomain string
	peer        *spiffe.TLSPeer
}

func NewSpireProvider(addr string) (Provider, error) {
	p, err := spiffe.NewTLSPeer(spiffe.WithWorkloadAPIAddr(addr))
	if err != nil {
		return nil, err
	}

	rv := &spireProvider{
		peer: p,
	}

	go func() {
		spiffeID, err := rv.GetID(context.Background())
		if err != nil {
			logrus.Error(err)
			return
		}
		logrus.Infof("Issued certificate with id: %v", spiffeID)

		tlscrt, err := rv.GetRootCA(context.Background())
		if err != nil {
			logrus.Error(err)
			return
		}
		logrus.Infof("crt %v", tlscrt)
	}()

	return rv, nil
}

func (p *spireProvider) GetTLSConfig(ctx context.Context) (*tls.Config, error) {
	return p.peer.GetConfig(ctx, spiffe.ExpectAnyPeer())
}

func (p *spireProvider) GetCertificate(ctx context.Context) (*tls.Certificate, error) {
	if err := p.peer.WaitUntilReady(ctx); err != nil {
		return nil, err
	}

	return p.peer.GetCertificate()
}

func (p *spireProvider) GetRootCA(ctx context.Context) (*x509.CertPool, error) {
	if err := p.peer.WaitUntilReady(ctx); err != nil {
		return nil, err
	}

	roots, err := p.peer.GetRoots()
	if err != nil {
		return nil, err
	}
	logrus.Infof("roots - %v", roots)

	trustDomain := p.getTrustDomain()
	cp, ok := roots[trustDomain]
	if !ok {
		return nil, errors.Errorf("no root certificates for %v", trustDomain)
	}

	return cp, nil
}

func (p *spireProvider) getTrustDomain() string {
	p.RLock()
	defer p.RUnlock()

	return spiffe.TrustDomainID(p.spiffeID.Host)
}

func (p *spireProvider) GetID(ctx context.Context) (string, error) {
	p.RLock()
	if p.spiffeID != nil {
		rv := p.spiffeID.String()
		p.RUnlock()
		return rv, nil
	}
	p.RUnlock()

	if err := p.peer.WaitUntilReady(ctx); err != nil {
		return "", err
	}

	tlscrt, err := p.peer.GetCertificate()
	if err != nil {
		return "", err
	}

	x509crt, err := x509.ParseCertificate(tlscrt.Certificate[0])
	if err != nil {
		return "", err
	}

	p.Lock()
	p.spiffeID, err = getIDsFromCertificate(x509crt)
	if err != nil {
		p.Unlock()
		return "", err
	}
	rv := p.spiffeID.String()
	p.Unlock()

	return rv, nil
}

func getIDsFromCertificate(peer *x509.Certificate) (*url.URL, error) {
	switch {
	case len(peer.URIs) == 0:
		return nil, errors.New("peer certificate contains no URI SAN")
	case len(peer.URIs) > 1:
		return nil, errors.New("peer certificate contains more than one URI SAN")
	}

	id := peer.URIs[0]

	if err := spiffe.ValidateURI(id, spiffe.AllowAny()); err != nil {
		return nil, err
	}

	return id, nil
}
