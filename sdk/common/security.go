package common

import (
	"github.com/pkg/errors"

	unifiedns "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/security"
)

type NSTokenConfig struct {
}

func (cfg *NSTokenConfig) FillClaims(claims *security.ChainClaims, msg interface{}) error {
	if request, ok := msg.(networkservice.Request); ok {
		claims.Audience = request.GetRequestConnection().GetNetworkService()
		return nil
	}

	if request, ok := msg.(*unifiedns.NetworkServiceRequest); ok {
		claims.Audience = request.GetRequestConnection().GetNetworkService()
		return nil
	}

	return errors.New("unable to cast msg to networkservice's request")
}

func (cfg *NSTokenConfig) RequestFilter(req interface{}) bool {
	if _, ok := req.(networkservice.Request); ok {
		return true
	}

	if _, ok := req.(*unifiedns.NetworkServiceRequest); ok {
		return true
	}

	return false
}

func ConnectionFillClaimsFunc(claims *security.ChainClaims, msg interface{}) error {
	conn, ok := msg.(connection.Connection)
	if !ok {
		return errors.New("unable to cast msg to connection.Connection")
	}

	claims.Audience = conn.GetNetworkService()
	return nil
}
