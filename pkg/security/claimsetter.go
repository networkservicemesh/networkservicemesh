package security

import (
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
)

func RequestClaimSetter(claims *ChainClaims, msg interface{}) error {
	request, ok := msg.(networkservice.Request)
	if !ok {
		return fmt.Errorf("unable to cast msg to networkserivce.Request")
	}

	claims.Audience = request.GetRequestConnection().GetNetworkService()
	return nil
}

func ConnectionClaimSetter(claims *ChainClaims, msg interface{}) error {
	conn, ok := msg.(connection.Connection)
	if !ok {
		return fmt.Errorf("unable to cast msg to connection.Connection")
	}

	claims.Audience = conn.GetNetworkService()
	return nil
}
