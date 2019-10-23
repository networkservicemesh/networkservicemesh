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
	"strings"

	"github.com/dgrijalva/jwt-go"
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

		logrus.Infof("ClientInterceptor start working...")

		var obo *TokenAndClaims
		if SecurityContext(ctx) != nil && SecurityContext(ctx).GetRequestOboToken() != nil {
			logrus.Infof("ClientInterceptor discovered obo-token: %v", SecurityContext(ctx).GetRequestOboToken().Token)
			obo = SecurityContext(ctx).GetRequestOboToken()
		}

		token, err := GenerateSignature(req, cfg.FillClaims, securityProvider, WithObo(obo))
		if err != nil {
			logrus.Error(err)
			return err
		}

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

		respToken, parts, claims, err := ParseJWTWithClaims(nsReply.GetSignature())
		if err != nil {
			logrus.Error(err)
			return status.Errorf(codes.Unauthenticated, fmt.Sprintf("response jwt is not valid: %v", err))
		}

		if err := verifySignatureParsed(respToken, parts, claims, securityProvider.GetCABundle(), transportSpiffeID); err != nil {
			return status.Errorf(codes.Unauthenticated, "response jwt is not valid: %v", err)
		}

		if SecurityContext(ctx) != nil {
			logrus.Infof("Setting nsReply.GetSignature() to SecurityContext - %v", nsReply.GetSignature())
			SecurityContext(ctx).SetResponseOboToken(&TokenAndClaims{nsReply.GetSignature(), claims})
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

		logrus.Infof("ServerInterceptor start working...")

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

		jwt := md["authorization"][0]

		token, parts, claims, err := ParseJWTWithClaims(jwt)
		if err != nil {
			logrus.Error(err)
			return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("token is not valid: %v", err))
		}

		if err := verifySignatureParsed(token, parts, claims, securityProvider.GetCABundle(), spiffeID); err != nil {
			logrus.Error(err)
			return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("token is not valid: %v", err))
		}

		securityContext := NewContext()
		securityContext.SetRequestOboToken(&TokenAndClaims{jwt, claims})

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

func verifyJWTChain(token *jwt.Token, parts []string, claims *ChainClaims, ca *x509.CertPool) error {
	currentToken, currentParts, currentClaims := token, parts, claims

	for currentToken != nil {
		err := verifySingleJwt(currentToken, currentParts, currentClaims, ca)
		if err != nil {
			return err
		}

		currentToken, currentParts, currentClaims, err = currentClaims.parseObo()
		if err != nil {
			return err
		}
	}

	return nil
}

func verifySingleJwt(token *jwt.Token, parts []string, claims *ChainClaims, ca *x509.CertPool) error {
	logrus.Infof("Validating JWT: %s", claims.Subject)
	crt, err := claims.verifyAndGetCertificate(ca)
	if err != nil {
		return err
	}

	if len(parts) != 3 {
		return errors.New("length of parts array is incorrect")
	}

	if err := token.Method.Verify(strings.Join(parts[0:2], "."), parts[2], crt.PublicKey); err != nil {
		return errors.Wrap(err, "jwt signature is not valid: %s")
	}

	return nil
}
