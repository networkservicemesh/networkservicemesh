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
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func ParseJWTWithClaims(tokenString string) (*jwt.Token, []string, *ChainClaims, error) {
	token, parts, err := new(jwt.Parser).ParseUnverified(tokenString, &ChainClaims{})
	if err != nil {
		return nil, nil, nil, err
	}

	claims, ok := token.Claims.(*ChainClaims)
	if !ok {
		return nil, nil, nil, errors.New("wrong claims format")
	}

	return token, parts, claims, err
}

func ClientInterceptor(securityProvider Provider, cfg TokenConfig) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {

		if !cfg.RequestFilter(req) {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		span := spanhelper.FromContext(ctx, "security.ClientInterceptor")
		defer span.Finish()

		var obo *Signature
		if SecurityContext(ctx) != nil && SecurityContext(ctx).GetRequestOboToken() != nil {
			span.Logger().Info("ClientInterceptor discovered obo-token")
			obo = SecurityContext(ctx).GetRequestOboToken()
		}

		token, err := GenerateSignature(ctx, req, cfg.FillClaims, securityProvider, WithObo(obo))
		if err != nil {
			logrus.Error(err)
			return err
		}
		//logrus.Infof("ClientInterceptor before 'invoke' took %v", time.Since(t))
		p := new(peer.Peer)
		err = invoker(ctx, method, req, reply, cc, append(opts, grpc.PerRPCCredentials(&NSMToken{Token: token}), grpc.Peer(p))...)
		if err != nil {
			return err
		}

		transportSpiffeID, err := spiffeIDFromPeer(p)
		if err != nil {
			return err
		}

		nsReply, ok := reply.(Signed)
		if !ok {
			return errors.New("can't verify response: wrong type")
		}

		s := &Signature{}
		err = s.Parse(nsReply.GetSignature())
		if err != nil {
			return err
		}

		if err := verifySignatureParsed(ctx, s, securityProvider.GetCABundle(), transportSpiffeID); err != nil {
			return status.Errorf(codes.Unauthenticated, "response jwt is not valid: %v", err)
		}

		if SecurityContext(ctx) != nil {
			SecurityContext(ctx).SetResponseOboToken(s)
		}
		return nil
	}
}

func ServerInterceptor(securityProvider Provider, cfg TokenConfig) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		if !cfg.RequestFilter(req) {
			return handler(ctx, req)
		}

		span := spanhelper.FromContext(ctx, "security.ServerInterceptor")
		defer span.Finish()

		spiffeID, err := spiffeIDFromContext(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, err.Error())
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		if len(md["authorization"]) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "no token provided")
		}

		s := &Signature{}
		err = s.Parse(md["authorization"][0])
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "signature parse error")
		}

		if err := verifySignatureParsed(ctx, s, securityProvider.GetCABundle(), spiffeID); err != nil {
			logrus.Error(err)
			return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("token is not valid: %v", err))
		}

		securityContext := NewContext()
		securityContext.SetRequestOboToken(s)
		return handler(WithSecurityContext(ctx, securityContext), req)
	}
}

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
		return "", errors.New("peer has wrong type")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return "", errors.New("peer's certificate list is empty")
	}

	if len(tlsInfo.State.PeerCertificates[0].URIs) == 0 {
		return "", errors.New("certificate doesn't have URIs")
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
