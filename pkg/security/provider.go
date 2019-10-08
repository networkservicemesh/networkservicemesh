// Copyright (c) 2019 Cisco and/or its affiliates.
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
	"crypto/tls"
	"crypto/x509"
	"sync"

	"github.com/sirupsen/logrus"
)

// Provider provides methods for secure grpc communication
type Provider interface {
	GetCertificate() *tls.Certificate
	GetCABundle() *x509.CertPool
	GetSpiffeID() string
}

// CertificateObtainer abstracts certificates obtaining
type CertificateObtainer interface {
	Stop()
	ErrorCh() <-chan error
	CertificateCh() <-chan *Response
}

// Response represents pair - TLSCert and CABundle that are returned from CertificateObtainer
type Response struct {
	TLSCert  *tls.Certificate
	CABundle *x509.CertPool
}

type certificateManager struct {
	sync.RWMutex
	caBundle *x509.CertPool
	cert     *tls.Certificate
	readyCh  chan struct{}
}

// NewProvider creates new security.Manager using SpireCertObtainer
func NewProvider() Provider {
	return NewProviderWithCertObtainer(NewSpireObtainer())
}

// NewProviderWithCertObtainer creates new security.Manager with passed CertificateObtainer
func NewProviderWithCertObtainer(obtainer CertificateObtainer) Provider {
	cm := &certificateManager{
		readyCh: make(chan struct{}),
	}
	go cm.exchangeCertificates(obtainer)
	return cm
}

func (m *certificateManager) GetCertificate() *tls.Certificate {
	<-m.readyCh

	m.RLock()
	defer m.RUnlock()
	return m.cert
}

func (m *certificateManager) GetCABundle() *x509.CertPool {
	<-m.readyCh

	m.RLock()
	defer m.RUnlock()
	return m.caBundle
}

func (m *certificateManager) GetSpiffeID() string {
	// todo: don't parse certificate every time, save spiffeID
	crtBytes := m.GetCertificate().Certificate[0]
	x509crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		logrus.Error(err)
		return ""
	}
	return x509crt.URIs[0].String()
}

func (m *certificateManager) exchangeCertificates(obtainer CertificateObtainer) {
	for {
		select {
		case r := <-obtainer.CertificateCh():
			m.setCertificates(r)
		case err := <-obtainer.ErrorCh():
			logrus.Errorf("security.Manager error: %v", err)
		}
	}
}

func (m *certificateManager) setCertificates(r *Response) {
	m.Lock()
	defer m.Unlock()

	if m.cert == nil {
		close(m.readyCh)
	}
	m.cert = r.TLSCert
	m.caBundle = r.CABundle
}
