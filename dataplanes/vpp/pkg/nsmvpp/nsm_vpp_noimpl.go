package nsmvpp

import (
	"fmt"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

type unimplementedMechanism struct {
	Type common.LocalMechanismType
}

// CreateLocalConnect return error for unimplemented mechanism
func (m unimplementedMechanism) createInterface(apiCh govppapi.Channel, parameters map[string]string) (uint32, error) {
	return 0, fmt.Errorf("%s mechanism not implemented", common.LocalMechanismType_name[int32(m.Type)])
}

func (m unimplementedMechanism) deleteInterface(apiCh govppapi.Channel, intf uint32) error {
	return fmt.Errorf("%s mechanism not implemented", common.LocalMechanismType_name[int32(m.Type)])
}

func (m unimplementedMechanism) validate(parameters map[string]string) error {
	return nil
}
