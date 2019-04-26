package nsm

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/sirupsen/logrus"
)

type networkServiceHealer interface {
	healDstDown(healID string, connection *model.ClientConnection) bool
	healDataplaneDown(healID string, connection *model.ClientConnection) bool
	healDstUpdate(healID string, connection *model.ClientConnection) bool
	healDstNmgrDown(healID string, connection *model.ClientConnection) bool
}

type healer struct {
	nsm *networkServiceManager
}

func (h *healer) healDstDown(healID string, connection *model.ClientConnection) bool {
	logrus.Infof("NSM_Heal(1.1.1-%v) Checking if DST die is NSMD/DST die...", healID)
	// Check if this is a really HealState_DstDown or HealState_DstNmgrDown
	if !h.nsm.isLocalEndpoint(connection.Endpoint) {
		ctx, cancel := context.WithTimeout(context.Background(), h.nsm.properties.HealTimeout*3)
		defer cancel()
		remoteNsmClient, err := h.nsm.createNSEClient(ctx, connection.Endpoint)
		if remoteNsmClient != nil {
			_ = remoteNsmClient.Cleanup()
		}
		if err != nil {
			// This is NSMD die case.
			logrus.Infof("NSM_Heal(1.1.2-%v) Connection healing state is %v...", healID, nsm.HealState_DstNmgrDown)
			return h.healDstNmgrDown(healID, connection)
		}
	}

	logrus.Infof("NSM_Heal(1.1.2-%v) Connection healing state is %v...", healID, nsm.HealState_DstDown)

	// Destination is down, we need to find it again.
	if connection.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		logrus.Infof("NSM_Heal(2.1-%v) Remote NSE heal is done on source side", healID)
		return false
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), h.nsm.properties.HealTimeout*3)
		defer cancel()

		logrus.Infof("NSM_Heal(2.2-%v) Starting DST Heal...", healID)
		// We are client NSMd, we need to try recover our connection srv.

		// Wait for NSE not equal to down one, since we know it will be re-registered with new EndpointName.
		if !h.nsm.waitNSE(ctx, connection, connection.Endpoint.NetworkserviceEndpoint.EndpointName, connection.GetNetworkService()) {
			// Not remote NSE found, we need to update connection
			if dst := connection.Xcon.GetRemoteDestination(); dst != nil {
				dst.SetId("-") // We need to mark this as new connection.
			}
			if dst := connection.Xcon.GetLocalDestination(); dst != nil {
				dst.SetId("-") // We need to mark this as new connection.
			}
			// We need to remove selected endpoint here.
			connection.Endpoint = nil
		}
		// Fallback to heal with choose of new NSE.
		requestCtx, requestCancel := context.WithTimeout(context.Background(), h.nsm.properties.HealRequestTimeout)
		defer requestCancel()
		logrus.Errorf("NSM_Heal(2.3.0-%v) Starting Heal by calling request: %v", healID, connection.Request)
		recoveredConnection, err := h.nsm.request(requestCtx, connection.Request, connection)
		if err != nil {
			logrus.Errorf("NSM_Heal(2.3.1-%v) Failed to heal connection: %v", healID, err)
			// We need to delete connection, since we are not able to Heal it
			h.nsm.model.DeleteClientConnection(connection.ConnectionId)
			if err != nil {
				logrus.Errorf("NSM_Heal(2.3.2-%v) Error in Recovery Close: %v", healID, err)
			}
		} else {
			logrus.Infof("NSM_Heal(2.4-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
			return true
		}
	}
	// Let's Close remote connection and re-create new one.
	return false
}

func (h *healer) healDataplaneDown(healID string, connection *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), h.nsm.properties.HealTimeout)
	defer cancel()

	// Dataplane is down, we only need to re-programm dataplane.
	// 1. Wait for dataplane to appear.
	logrus.Infof("NSM_Heal(3.1-%v) Waiting for Dataplane to recovery...", healID)
	if err := h.nsm.serviceRegistry.WaitForDataplaneAvailable(h.nsm.model, h.nsm.properties.HealDataplaneTimeout); err != nil {
		logrus.Errorf("NSM_Heal(3.1-%v) Dataplane is not available on recovery for timeout %v: %v", h.nsm.properties.HealDataplaneTimeout, healID, err)
		return false
	}
	logrus.Infof("NSM_Heal(3.2-%v) Dataplane is now available...", healID)

	// We could send connection is down now.
	h.nsm.model.UpdateClientConnection(connection)

	if connection.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		// Recovery will be performed by NSM client side.
		logrus.Infof("NSM_Heal(3.3-%v)  Healing will be continued on source side...", healID)
		return true
	}

	// We have Dataplane now, let's try request all again.
	// Update request to contain a proper connection object from previous attempt.
	request := connection.Request.Clone()
	request.SetConnection(connection.GetConnectionSource())
	h.nsm.requestOrClose(fmt.Sprintf("NSM_Heal(3.4-%v) ", healID), ctx, request, connection)
	return true
}

func (h *healer) healDstUpdate(healID string, connection *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), h.nsm.properties.HealTimeout)
	defer cancel()

	// Remote DST is updated.
	// Update request to contain a proper connection object from previous attempt.
	logrus.Infof("NSM_Heal(5.1-%v) Healing Src Update... %v", healID, connection)
	if connection.Request != nil {
		request := connection.Request.Clone()
		request.SetConnection(connection.GetConnectionSource())

		h.nsm.requestOrClose(fmt.Sprintf("NSM_Heal(5.2-%v) ", healID), ctx, request, connection)
		return true
	}
	return false
}

func (h *healer) healDstNmgrDown(healID string, connection *model.ClientConnection) bool {
	logrus.Infof("NSM_Heal(6.1-%v) Starting DST + NSMGR Heal...", healID)

	ctx, cancel := context.WithTimeout(context.Background(), h.nsm.properties.HealTimeout*3)
	defer cancel()

	var initialEndpoint = connection.Endpoint
	networkServiceName := connection.GetNetworkService()
	// Wait for exact same NSE to be available with NSMD connection alive.
	if connection.Endpoint != nil && !h.nsm.waitSpecificNSE(ctx, connection) {
		// Not remote NSE found, we need to update connection
		if dst := connection.Xcon.GetRemoteDestination(); dst != nil {
			dst.SetId("-") // We need to mark this as new connection.
		}
		if dst := connection.Xcon.GetLocalDestination(); dst != nil {
			dst.SetId("-") // We need to mark this as new connection.
		}
		connection.Endpoint = nil
	}
	requestCtx, requestCancel := context.WithTimeout(context.Background(), h.nsm.properties.HealRequestTimeout)
	defer requestCancel()
	recoveredConnection, err := h.nsm.request(requestCtx, connection.Request, connection)
	if err != nil {
		logrus.Errorf("NSM_Heal(6.2.1-%v) Failed to heal connection with same NSE from registry: %v", healID, err)
		if initialEndpoint != nil {
			logrus.Infof("NSM_Heal(6.2.2-%v) Waiting for another NSEs...", healID)
			// In this case, most probable both NSMD and NSE are die, and registry was outdated on moment of waitNSE.
			if h.nsm.waitNSE(ctx, connection, initialEndpoint.NetworkserviceEndpoint.EndpointName, networkServiceName) {
				// Ok we have NSE, lets retry request
				requestCtx, requestCancel := context.WithTimeout(context.Background(), h.nsm.properties.HealRequestTimeout)
				defer requestCancel()
				recoveredConnection, err = h.nsm.request(requestCtx, connection.Request, connection)
				if err != nil {
					if err != nil {
						logrus.Errorf("NSM_Heal(6.2.3-%v) Error in Recovery Close: %v", healID, err)
					}
				} else {
					logrus.Infof("NSM_Heal(6.3-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
					return true
				}
			}
		}

		logrus.Errorf("NSM_Heal(6.4.1-%v) Failed to heal connection: %v", healID, err)
		// We need to delete connection, since we are not able to Heal it
		h.nsm.model.DeleteClientConnection(connection.ConnectionId)
		if err != nil {
			logrus.Errorf("NSM_Heal(6.4.2-%v) Error in Recovery Close: %v", healID, err)
		}
	} else {
		logrus.Infof("NSM_Heal(6.5-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
		return true
	}
	return false
}
