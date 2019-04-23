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
	"github.com/sirupsen/logrus"
	"strconv"
)

func (srv *networkServiceManager) updateMechanism(requestId string, nsmConnection nsm.NSMConnection, request nsm.NSMRequest, dataplane *model.Dataplane) error {
	// 5.x
	if request.IsRemote() {
		//5.1 Select appropriate remote mechanism
		mechanism, err := srv.selectRemoteMechanism(requestId, request.(*remote_networkservice.NetworkServiceRequest), dataplane)
		if err != nil {
			return err
		}
		c := nsmConnection.(*remote_connection.Connection)
		newMechanism := proto.Clone(mechanism).(*remote_connection.Mechanism)
		c.Mechanism = newMechanism
		if c.Mechanism == nil {
			return fmt.Errorf("Required mechanism are not found... %v ", dataplane.RemoteMechanisms)
		}
		if c.Mechanism.Parameters == nil {
			c.Mechanism.Parameters = map[string]string{}
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
	}

	return nil
}

func (srv *networkServiceManager) selectRemoteMechanism(requestId string, request *remote_networkservice.NetworkServiceRequest, dp *model.Dataplane) (*remote_connection.Mechanism, error) {
	for _, mechanism := range request.MechanismPreferences {
		dp_mechanism := findRemoteMechanism(dp.RemoteMechanisms, remote_connection.MechanismType_VXLAN)
		if dp_mechanism == nil {
			continue
		}
		// TODO: Add other mechanisms support
		if mechanism.Type == remote_connection.MechanismType_VXLAN {
			// Update DST IP to be ours
			remoteSrc := mechanism.Parameters[remote_connection.VXLANSrcIP]
			mechanism.Parameters[remote_connection.VXLANSrcIP] = remoteSrc
			mechanism.Parameters[remote_connection.VXLANDstIP] = dp_mechanism.Parameters[remote_connection.VXLANSrcIP]
			mechanism.Parameters[remote_connection.VXLANVNI] = strconv.FormatUint(uint64(srv.serviceRegistry.VniAllocator().Vni(dp_mechanism.Parameters[remote_connection.VXLANSrcIP], remoteSrc)), 10)
		}
		logrus.Infof("NSM:(4.1-%v) Remote mechanism selected %v", requestId, mechanism)
		return mechanism, nil
	}
	return nil, fmt.Errorf("NSM:(5.1-%v) Failed to select mechanism. No matched mechanisms found...", requestId)
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
