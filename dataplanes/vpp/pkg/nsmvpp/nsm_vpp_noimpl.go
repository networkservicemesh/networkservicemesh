package nsmvpp

import (
	"fmt"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

type UnimplementedMechanism struct {
	Type common.LocalMechanismType
}

// CreateLocalConnect return error for unimplemented mechanism
func (m UnimplementedMechanism) CreateLocalConnect(apiCh govppapi.Channel, srcParameters, dstParameters map[string]string) (string, error) {
	return "", fmt.Errorf("%s mechanism not implemented", common.LocalMechanismType_name[int32(m.Type)])
}

func (m UnimplementedMechanism) DeleteLocalConnect(apiCh govppapi.Channel, connID string) error {
	return fmt.Errorf("%s mechanism not implemented", common.LocalMechanismType_name[int32(m.Type)])
}

func (m UnimplementedMechanism) Validate(parameters map[string]string) error {
	return nil
}
