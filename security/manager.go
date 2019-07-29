// Copyright (c) 2018 Cisco and/or its affiliates.
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

// Manager provides methods for secure grpc communication
type Manager interface {
	GetCertificate() *tls.Certificate
	GetCABundle() *x509.CertPool
}

// CertificateObtainer abstracts certificates obtaining
type CertificateObtainer interface {
	Stop()
	ErrorCh() <-chan error
	CertificateCh() <-chan *Response
}

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

var once sync.Once
var manager Manager

// GetSecurityManager returns instance of Manager
func GetSecurityManager() Manager {
	logrus.Info("Getting SecurityManager...")
	once.Do(func() {
		logrus.Info("Creating new SecurityManager...")
		manager = NewManager()
	})
	return manager
}

// NewManager creates new security.Manager using SpireCertObtainer
func NewManager() Manager {
	return NewManagerWithCertObtainer(newSpireObtainer())
}

// NewManagerWithCertObtainer creates new security.Manager with passed CertificateObtainer
func NewManagerWithCertObtainer(obtainer CertificateObtainer) Manager {
	cm := &certificateManager{
		readyCh: make(chan struct{}),
	}
	go cm.exchangeCertificates(obtainer)
	return cm
}

func (m *certificateManager) exchangeCertificates(obtainer CertificateObtainer) {
	for {
		select {
		case r := <-obtainer.CertificateCh():
			m.setCertificates(r)
		case err := <-obtainer.ErrorCh():
			logrus.Error(err)
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
