package nsm

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	remote_connection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	remote_networkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"strconv"
)

func (srv *networkServiceManager) updateMechanism(nsmConnection nsm.NSMConnection, request nsm.NSMRequest, dataplane *model.Dataplane, extra_parameters map[string]string) error {
	if request.IsRemote() {
		mechanism, err := srv.selectRemoteMechanism(request.(*remote_networkservice.NetworkServiceRequest), dataplane)
		if err != nil {
			return err
		}
		c := nsmConnection.(*remote_connection.Connection)
		c.Mechanism = proto.Clone(mechanism).(*remote_connection.Mechanism)

		if c.Mechanism == nil {
			return fmt.Errorf("Required mechanism are not found... %v ", dataplane.RemoteMechanisms)
		}
		if c.Mechanism.Parameters == nil {
			c.Mechanism.Parameters = map[string]string{}
		}

		for k, v := range extra_parameters {
			c.Mechanism.Parameters[k] = v
		}
	} else {
		c := nsmConnection.(*connection.Connection)
		r := request.(*networkservice.NetworkServiceRequest)

		for _, m := range r.MechanismPreferences {
			dpMechanism := findLocalMechanism(dataplane.LocalMechanisms, m.Type)
			if dpMechanism != nil { // We have matching dataplane mechanism
				c.Mechanism = proto.Clone(m).(*connection.Mechanism)
				break
			}
		}
		if c.Mechanism == nil {
			return fmt.Errorf("Required mechanism are not found... %v ", r.MechanismPreferences)
		}
		if c.Mechanism.Parameters == nil {
			c.Mechanism.Parameters = map[string]string{}
		}

		for k, v := range extra_parameters {
			c.Mechanism.Parameters[k] = v
		}

	}

	return nil
}

func (srv *networkServiceManager) selectRemoteMechanism(request *remote_networkservice.NetworkServiceRequest, dataplane *model.Dataplane) (*remote_connection.Mechanism, error) {
	for _, mechanism := range request.MechanismPreferences {
		dp_mechanism := findRemoteMechanism(dataplane.RemoteMechanisms, remote_connection.MechanismType_VXLAN)
		if dp_mechanism == nil {
			continue
		}
		// TODO: Add other mechanisms support
		if mechanism.Type == remote_connection.MechanismType_VXLAN {
			// Update DST IP to be ours
			remoteSrc := mechanism.Parameters[remote_connection.VXLANSrcIP]
			mechanism.Parameters[remote_connection.VXLANSrcIP] = remoteSrc
			mechanism.Parameters[remote_connection.VXLANDstIP] = dp_mechanism.Parameters[remote_connection.VXLANSrcIP]
			mechanism.Parameters[remote_connection.VXLANVNI] = strconv.FormatUint(srv.serviceRegistry.VniAllocator().Vni(dp_mechanism.Parameters[remote_connection.VXLANSrcIP], remoteSrc), 10)
		}
		return mechanism, nil
	}
	return nil, fmt.Errorf("Failed to select mechanism. No matched mechanisms found...")
}

func findRemoteMechanism(MechanismPreferences []*remote_connection.Mechanism, mechanismType remote_connection.MechanismType) *remote_connection.Mechanism {
	for _, m := range MechanismPreferences {
		if m.Type == mechanismType {
			return m
		}
	}
	return nil
}

func findLocalMechanism(MechanismPreferences []*connection.Mechanism, mechanismType connection.MechanismType) *connection.Mechanism {
	for _, m := range MechanismPreferences {
		if m.Type == mechanismType {
			return m
		}
	}
	return nil
}
