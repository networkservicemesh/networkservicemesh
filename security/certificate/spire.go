package security

import (
	"crypto/x509"
	"encoding/pem"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/api/workload"
	proto "github.com/spiffe/spire/proto/spire/api/workload"
	"net"
	"time"
)

type spireCertObtainer struct {
	stopCh            chan struct{}
	errorCh           chan error
	workloadAPIClient workload.X509Client
}

func NewSpireCertObtainer(agentAddress string, timeout time.Duration) CertificateObtainer {
	return &spireCertObtainer{
		stopCh:            make(chan struct{}),
		errorCh:           make(chan error),
		workloadAPIClient: newWorkloadAPIClient(agentAddress, timeout),
	}
}

func newWorkloadAPIClient(agentAddress string, timeout time.Duration) workload.X509Client {
	addr := &net.UnixAddr{
		Net:  "unix",
		Name: agentAddress,
	}
	config := &workload.X509ClientConfig{
		Addr:    addr,
		Timeout: timeout,
	}
	return workload.NewX509Client(config)
}

func (s *spireCertObtainer) ObtainCertificates() <-chan *certs {
	certCh := make(chan *certs)

	go func() {
		if err := s.workloadAPIClient.Start(); err != nil {
			logrus.Error(err.Error())
			s.errorCh <- err
			close(certCh)
			return
		}
	}()
	defer s.workloadAPIClient.Stop()

	go func() {
		defer close(certCh)

		updateCh := s.workloadAPIClient.UpdateChan()
		for {
			select {
			case svidResponse := <-updateCh:
				logrus.Infof("Received new SVID: %v", svidResponse.Svids[0].SpiffeId)
				certCh <- readCertificates(svidResponse)
			case <-s.stopCh:
				return
			}
		}
	}()

	return certCh
}

func (s *spireCertObtainer) Stop() {
	close(s.stopCh)
}

func (s *spireCertObtainer) Error() error {
	return <-s.errorCh
}

func readCertificates(svidResponse *proto.X509SVIDResponse) *certs {
	svid := svidResponse.Svids[0]
	cert, _ := certToPemBlocks(svid.X509Svid)
	ca, _ := certToPemBlocks(svid.Bundle)
	return &certs{
		cert: cert,
		key:  keyToPem(svid.X509SvidKey),
		ca:   ca,
	}
}

func certToPemBlocks(data []byte) ([]byte, error) {
	certs, err := x509.ParseCertificates(data)
	if err != nil {
		return nil, err
	}

	pemData := []byte{}
	for _, cert := range certs {
		b := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}

	return pemData, nil
}

func keyToPem(data []byte) []byte {
	b := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: data,
	}
	return pem.EncodeToMemory(b)
}
