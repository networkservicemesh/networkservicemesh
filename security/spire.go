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
	"errors"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/api/workload"

	proto "github.com/spiffe/spire/proto/spire/api/workload"
)

const (
	agentAddress = "/run/spire/sockets/agent.sock"
)

type spireObtainer struct {
	errCh             chan error
	responseCh        <-chan *Response
	workloadAPIClient workload.X509Client
}

// newSpireCertObtainer creates CertificateObtainer that fetch certificates from spire-agent
func newSpireObtainer() CertificateObtainer {
	workloadAPIClient := workload.NewX509Client(
		&workload.X509ClientConfig{
			Addr: &net.UnixAddr{Net: "unix", Name: agentAddress},
			Log:  logrus.StandardLogger(),
		})

	errCh := make(chan error)

	go func() {
		if err := workloadAPIClient.Start(); err != nil {
			errCh <- err
			return
		}
	}()

	responseCh := certsFromSpireCh(workloadAPIClient.UpdateChan(), errCh)

	return &spireObtainer{
		errCh:             errCh,
		responseCh:        responseCh,
		workloadAPIClient: workloadAPIClient,
	}
}

func (o *spireObtainer) Stop() {
	o.workloadAPIClient.Stop()
}

func (o *spireObtainer) ErrorCh() <-chan error {
	return o.errCh
}

func (o *spireObtainer) CertificateCh() <-chan *Response {
	return o.responseCh
}

func certsFromSpireCh(spireCh <-chan *proto.X509SVIDResponse, errCh chan<- error) <-chan *Response {
	responseCh := make(chan *Response)

	go func() {
		defer close(responseCh)

		for svidResponse := range spireCh {
			logrus.Infof("Received new SVID: %v", svidResponse.Svids[0].SpiffeId)
			response, err := newResponse(svidResponse)
			if err != nil {
				errCh <- err
				return
			}
			responseCh <- response
		}
	}()

	return responseCh
}

func newResponse(svidResponse *proto.X509SVIDResponse) (*Response, error) {
	svid := svidResponse.Svids[0]

	crt, err := certToPemBlocks(svid.GetX509Svid())
	if err != nil {
		return nil, err
	}

	key := keyToPem(svid.GetX509SvidKey())
	keyPair, err := tls.X509KeyPair(crt, key)
	if err != nil {
		return nil, err
	}

	x509crt, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Infof("Length of URIs = %v", len(x509crt.URIs))
	logrus.Infof("URIs[0] = %v", *x509crt.URIs[0])

	caBundle, err := certToPemBlocks(svid.GetBundle())
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caBundle); !ok {
		return nil, errors.New("failed to append ca cert to pool")
	}

	return &Response{
		TLSCert:  &keyPair,
		CABundle: caPool,
	}, nil
}
