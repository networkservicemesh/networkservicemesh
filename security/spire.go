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
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/api/workload"
	"net"

	proto "github.com/spiffe/spire/proto/spire/api/workload"
)

const (
	agentAddress = "/run/spire/sockets/agent.sock"
)

type spireObtainer struct {
	errCh             chan error
	responseCh        <-chan *response
	workloadAPIClient workload.X509Client
}

// newSpireCertObtainer creates certificateObtainer that fetch certificates from spire-agent
func newSpireObtainer() certificateObtainer {
	workloadAPIClient := workload.NewX509Client(
		&workload.X509ClientConfig{
			Addr: &net.UnixAddr{"unix", agentAddress},
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

func (o *spireObtainer) stop() {
	o.workloadAPIClient.Stop()
}

func (o *spireObtainer) errorCh() <-chan error {
	return o.errCh
}

func (o *spireObtainer) certificateCh() <-chan *response {
	return o.responseCh
}

func certsFromSpireCh(spireCh <-chan *proto.X509SVIDResponse, errCh chan<- error) <-chan *response {
	responseCh := make(chan *response)

	go func() {
		defer close(responseCh)

		for {
			select {
			case svidResponse, ok := <-spireCh:
				if !ok {
					return
				}

				logrus.Infof("Received new SVID: %v", svidResponse.Svids[0].SpiffeId)
				response, err := newResponse(svidResponse)
				if err != nil {
					errCh <- err
					return
				}
				responseCh <- response
			}
		}
	}()

	return responseCh
}

func newResponse(svidResponse *proto.X509SVIDResponse) (*response, error) {
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

	if x509crt, err := x509.ParseCertificate(keyPair.Certificate[0]); err == nil {
		logrus.Infof("Length of URIs = %v", len(x509crt.URIs))
		logrus.Infof("URIs[0] = %v", *x509crt.URIs[0])
	} else {
		logrus.Error(err)
	}

	caBundle, err := certToPemBlocks(svid.GetBundle())
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caBundle); !ok {
		return nil, errors.New("failed to append ca cert to pool")
	}

	return &response{
		TLSCert:  &keyPair,
		CABundle: caPool,
	}, nil
}
