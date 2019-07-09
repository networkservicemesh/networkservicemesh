package security

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func spiffeIDFromContext(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "missing peer TLSCred")
	}

	return spiffeIDFromPeer(p)
}

func spiffeIDFromPeer(p *peer.Peer) (string, error) {
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", fmt.Errorf("peer has wrong type")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return "", fmt.Errorf("peer's certificate list is empty")
	}

	if len(tlsInfo.State.PeerCertificates[0].URIs) == 0 {
		return "", fmt.Errorf("certificate doesn't have URIs")
	}

	return tlsInfo.State.PeerCertificates[0].URIs[0].String(), nil
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
