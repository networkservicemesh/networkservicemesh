package nsm

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func (srv *networkServiceManager) updateMechanism(requestId string, connection connection.Connection, request networkservice.Request, dataplane *model.Dataplane) error {
	// 5.x
	if request.IsRemote() {
		if m, err := srv.selectRemoteMechanism(requestId, request.(*remote_networkservice.NetworkServiceRequest), dataplane); err == nil {
			connection.SetConnectionMechanism(m.Clone())
		} else {
			return err
		}
	} else {
		for _, m := range request.GetRequestMechanismPreferences() {
			if dpMechanism := findMechanism(dataplane.LocalMechanisms, m.GetMechanismType()); dpMechanism != nil {
				connection.SetConnectionMechanism(m.Clone())
				break
			}
		}
	}

	if connection.GetConnectionMechanism() == nil {
		return fmt.Errorf("Required mechanism are not found... %v ", request.GetRequestMechanismPreferences())
	}

	if connection.GetConnectionMechanism().GetParameters() == nil {
		connection.GetConnectionMechanism().SetParameters(map[string]string{})
	}

	return nil
}

func (srv *networkServiceManager) selectRemoteMechanism(requestId string, request networkservice.Request, dp *model.Dataplane) (connection.Mechanism, error) {
	for _, mechanism := range request.GetRequestMechanismPreferences() {
		dpMechanism := findMechanism(dp.RemoteMechanisms, remote_connection.MechanismType_VXLAN)
		if dpMechanism == nil {
			continue
		}

		// TODO: Add other mechanisms support

		if mechanism.GetMechanismType() == remote_connection.MechanismType_VXLAN {
			parameters := mechanism.GetParameters()
			dpParameters := dpMechanism.GetParameters()

			parameters[remote_connection.VXLANDstIP] = dpParameters[remote_connection.VXLANSrcIP]

			vni := srv.serviceRegistry.VniAllocator().Vni(dpParameters[remote_connection.VXLANSrcIP], parameters[remote_connection.VXLANSrcIP])
			parameters[remote_connection.VXLANVNI] = strconv.FormatUint(uint64(vni), 10)
		}

		logrus.Infof("NSM:(4.1-%v) Remote mechanism selected %v", requestId, mechanism)
		return mechanism, nil
	}

	return nil, fmt.Errorf("NSM:(5.1-%v) Failed to select mechanism. No matched mechanisms found...", requestId)
}

func findMechanism(mechanismPreferences []connection.Mechanism, mechanismType connection.MechanismType) connection.Mechanism {
	for _, m := range mechanismPreferences {
		if m.GetMechanismType() == mechanismType {
			return m
		}
	}
	return nil
}
