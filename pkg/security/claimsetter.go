package security

import (
	newapi "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
)

func RequestClaimSetter(claims *ChainClaims, msg interface{}) error {
	request, ok := msg.(networkservice.Request)
	if !ok {
		return errors.New("unable to cast msg to networkserivce.Request")
	}

	claims.Audience = request.GetRequestConnection().GetNetworkService()
	return nil
}

func NewAPIRequestClaimSetter(claims *ChainClaims, msg interface{}) error {
	request, ok := msg.(*newapi.NetworkServiceRequest)
	if !ok {
		return errors.New("unable to cast msg to newapi.NetworkServiceRequest")
	}

	claims.Audience = request.GetRequestConnection().GetNetworkService()
	return nil
}

func ConnectionClaimSetter(claims *ChainClaims, msg interface{}) error {
	conn, ok := msg.(connection.Connection)
	if !ok {
		return errors.New("unable to cast msg to connection.Connection")
	}

	claims.Audience = conn.GetNetworkService()
	return nil
}
