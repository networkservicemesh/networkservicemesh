package security

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
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

func (s *spireCertObtainer) ObtainCertificates() <-chan *RetrievedCerts {
	certCh := make(chan *RetrievedCerts)

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
				if c, err := readCertificates(svidResponse); err == nil {
					certCh <- c
				} else {
					logrus.Error(err)
					s.errorCh <- err
					return
				}
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

func oidEqual(o1, o2 []int) bool {
	if len(o1) != len(o2) {
		return false
	}

	for i := 0; i < len(o1); i++ {
		if o1[i] != o2[i] {
			return false
		}
	}

	return true
}

func readCertificates(svidResponse *proto.X509SVIDResponse) (*RetrievedCerts, error) {
	svid := svidResponse.Svids[0]

	crt, err := certToPemBlocks(svid.GetX509Svid())
	if err != nil {
		return nil, err
	} else {
		logrus.Infof("PUBLIC PEM: %v", string(crt))
	}

	key := keyToPem(svid.GetX509SvidKey())
	keyPair, err := tls.X509KeyPair(crt, key)
	if err != nil {
		return nil, err
	}

	logrus.Infof("crt len %v", len(keyPair.Certificate))
	if len(keyPair.Certificate) > 1 {
		if x509crt, err := x509.ParseCertificate(keyPair.Certificate[1]); err == nil {
			//logrus.Infof("Length of DNSNames = %v", len(x509crt.DNSNames))
			//logrus.Infof("DNSNames[0] = %v", x509crt.DNSNames[0])
			logrus.Infof("Length of DNSNames = %v", len(x509crt.URIs))
			logrus.Infof("URI[0] = %v", *x509crt.URIs[0])
			logrus.Info("crt %v", *x509crt)
			//logrus.Infof("Length of x509crt.Extensions = %v", len(x509crt.Extensions))
			//for _, ext := range x509crt.Extensions {
			//	if oidEqual(ext.Id, []int{2, 5, 29, 17}) {
			//
			//	}
			//}
		}
	}
	if x509crt, err := x509.ParseCertificate(keyPair.Certificate[0]); err == nil {
		//logrus.Infof("Length of DNSNames = %v", len(x509crt.DNSNames))
		//logrus.Infof("DNSNames[0] = %v", x509crt.DNSNames[0])
		logrus.Infof("Length of DNSNames = %v", len(x509crt.URIs))
		logrus.Infof("DNSNames[0] = %v", *x509crt.URIs[0])
		//logrus.Infof("Length of x509crt.Extensions = %v", len(x509crt.Extensions))
		//for _, ext := range x509crt.Extensions {
		//	if oidEqual(ext.Id, []int{2, 5, 29, 17}) {
		//
		//	}
		//}
	} else {
		logrus.Error(err)
	}

	caBundle, err := certToPemBlocks(svid.GetBundle())
	// test
	c, _ := x509.ParseCertificates(svid.GetBundle())
	logrus.Info("CA_BUNDLE: %v", *c[0])
	logrus.Info("CA_BUNDLE len: %v", len(c))
	// test end
	if err != nil {
		return nil, err
	} else {
		logrus.Infof("CA_BUNDLE PEM: %v", string(caBundle))
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(caBundle); !ok {
		return nil, errors.New("failed to append ca cert to pool")
	}

	//verify
	leaf, _ := x509.ParseCertificate(keyPair.Certificate[0])
	interm, _ := x509.ParseCertificate(keyPair.Certificate[1])

	intermPool := x509.NewCertPool()
	intermPool.AddCert(interm)

	_, err1 := leaf.Verify(x509.VerifyOptions{
		Roots:         caPool,
		Intermediates: intermPool,
	})
	_, err2 := leaf.Verify(x509.VerifyOptions{
		Roots: caPool,
	})

	caPool.AddCert(interm)
	_, err3 := leaf.Verify(x509.VerifyOptions{
		Roots: caPool,
	})
	logrus.Infof("err1 - %v", err1)
	logrus.Infof("err2 - %v", err2)
	logrus.Infof("err3 - %v", err3)

	return &RetrievedCerts{
		TLSCert:  &keyPair,
		CABundle: caPool,
	}, nil
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
