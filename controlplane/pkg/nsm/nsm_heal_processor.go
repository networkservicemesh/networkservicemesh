package nsm

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type networkServiceHealProcessor interface {
	Heal(connection nsm.ClientConnection, healState nsm.HealState)
}

type healProcessor struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
	properties      *nsm.NsmProperties

	conManager connectionManager
	nseManager networkServiceEndpointManager

	eventCh chan healEvent
}

type connectionManager interface {
	request(ctx context.Context, request networkservice.Request, existingConnection *model.ClientConnection) (connection.Connection, error)
	Close(ctx context.Context, clientConnection nsm.ClientConnection) error
}

type healEvent struct {
	healID    string
	cc        *model.ClientConnection
	healState nsm.HealState
}

func newNetworkServiceHealProcessor(
	serviceRegistry serviceregistry.ServiceRegistry,
	model model.Model,
	properties *nsm.NsmProperties,
	conManager connectionManager,
	nseManager networkServiceEndpointManager) networkServiceHealProcessor {

	p := &healProcessor{
		serviceRegistry: serviceRegistry,
		model:           model,
		properties:      properties,
		conManager:      conManager,
		nseManager:      nseManager,
		eventCh:         make(chan healEvent, 1),
	}
	go p.serve()

	return p
}

func (p *healProcessor) Heal(connection nsm.ClientConnection, healState nsm.HealState) {
	healID := create_logid()
	logrus.Infof("NSM_Heal(%v) %v", healID, connection)

	cc := connection.(*model.ClientConnection)
	if cc.ConnectionState != model.ClientConnectionReady {
		//means that we already closing/healing
		return
	}

	if !p.properties.HealEnabled {
		logrus.Infof("NSM_Heal(%v) Is Disabled/Closing connection %v", healID, connection)

		err := p.conManager.Close(context.Background(), cc)
		if err != nil {
			logrus.Errorf("NSM_Heal(%v) Error in Close: %v", healID, err)
		}
		return
	}

	p.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.ConnectionState = model.ClientConnectionHealing
	})

	p.eventCh <- healEvent{
		healID:    healID,
		cc:        cc,
		healState: healState,
	}
}

func (p *healProcessor) serve() {
	for {
		e, ok := <-p.eventCh
		if !ok {
			return
		}

		go func() {
			healed := false

			switch e.healState {
			case nsm.HealStateDstDown:
				healed = p.healDstDown(e.healID, e.cc)
			case nsm.HealStateDataplaneDown:
				healed = p.healDataplaneDown(e.healID, e.cc)
			case nsm.HealStateDstUpdate:
				healed = p.healDstUpdate(e.healID, e.cc)
			case nsm.HealStateDstNmgrDown:
				healed = p.healDstNmgrDown(e.healID, e.cc)
			}

			if healed {
				e.cc = p.model.ApplyClientConnectionChanges(e.cc.GetID(), func(cc *model.ClientConnection) {
					cc.ConnectionState = model.ClientConnectionReady
				})
			} else {
				// Close both connection and dataplane
				err := p.conManager.Close(context.Background(), e.cc)
				if err != nil {
					logrus.Errorf("NSM_Heal(%v) Error in Recovery: %v", e.healID, err)
				}
			}

			logrus.Infof("NSM_Heal(%v) Connection %v healing state is finished...", e.healID, e.cc.GetID())
		}()
	}
}

func (p *healProcessor) healDstDown(healID string, cc *model.ClientConnection) bool {
	logrus.Infof("NSM_Heal(1.1.1-%v) Checking if DST die is NSMD/DST die...", healID)
	// Check if this is a really HealStateDstDown or HealStateDstNmgrDown
	if !p.nseManager.isLocalEndpoint(cc.Endpoint) {
		ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout*3)
		defer cancel()
		remoteNsmClient, err := p.nseManager.createNSEClient(ctx, cc.Endpoint)
		if remoteNsmClient != nil {
			_ = remoteNsmClient.Cleanup()
		}
		if err != nil {
			// This is NSMD die case.
			logrus.Infof("NSM_Heal(1.1.2-%v) Connection healing state is %v...", healID, nsm.HealStateDstNmgrDown)
			return p.healDstNmgrDown(healID, cc)
		}
	}

	logrus.Infof("NSM_Heal(1.1.2-%v) Connection healing state is %v...", healID, nsm.HealStateDstDown)

	// Destination is down, we need to find it again.
	if cc.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		logrus.Infof("NSM_Heal(2.1-%v) Remote NSE heal is done on source side", healID)
		return false
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout*3)
		defer cancel()

		logrus.Infof("NSM_Heal(2.2-%v) Starting DST Heal...", healID)
		// We are client NSMd, we need to try recover our connection srv.

		endpointName := cc.Endpoint.GetNetworkserviceEndpoint().GetEndpointName()
		// Wait for NSE not equal to down one, since we know it will be re-registered with new EndpointName.
		if !p.waitNSE(ctx, cc, endpointName, cc.GetNetworkService(), p.nseIsNewAndAvailable) {
			cc.GetConnectionDestination().SetID("-")
		}
		// Fallback to heal with choose of new NSE.
		requestCtx, requestCancel := context.WithTimeout(context.Background(), p.properties.HealRequestTimeout)
		defer requestCancel()
		logrus.Errorf("NSM_Heal(2.3.0-%v) Starting Heal by calling request: %v", healID, cc.Request)

		recoveredConnection, err := p.conManager.request(requestCtx, cc.Request, cc)
		if err != nil {
			logrus.Errorf("NSM_Heal(2.3.1-%v) Failed to heal connection: %v", healID, err)
			return false
		} else {
			logrus.Infof("NSM_Heal(2.4-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
			return true
		}
	}
}

func (p *healProcessor) healDataplaneDown(healID string, cc *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout)
	defer cancel()

	// Dataplane is down, we only need to re-programm dataplane.
	// 1. Wait for dataplane to appear.
	logrus.Infof("NSM_Heal(3.1-%v) Waiting for Dataplane to recovery...", healID)
	if err := p.serviceRegistry.WaitForDataplaneAvailable(p.model, p.properties.HealDataplaneTimeout); err != nil {
		logrus.Errorf("NSM_Heal(3.1-%v) Dataplane is not available on recovery for timeout %v: %v", p.properties.HealDataplaneTimeout, healID, err)
		return false
	}
	logrus.Infof("NSM_Heal(3.2-%v) Dataplane is now available...", healID)

	p.model.ApplyClientConnectionChanges(cc.GetID(), func(cc *model.ClientConnection) {
		cc.GetConnectionSource().SetConnectionState(connection.StateDown)
	})

	if cc.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		// Recovery will be performed by NSM client side.
		logrus.Infof("NSM_Heal(3.3-%v)  Healing will be continued on source side...", healID)
		return true
	}

	// We have Dataplane now, let's try request all again.
	// Update request to contain a proper connection object from previous attempt.
	request := cc.Request.Clone()
	request.SetRequestConnection(cc.GetConnectionSource())
	p.requestOrClose(ctx, fmt.Sprintf("NSM_Heal(3.4-%v) ", healID), request, cc)
	return true
}

func (p *healProcessor) healDstUpdate(healID string, cc *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout)
	defer cancel()

	// Destination is updated.
	// Update request to contain a proper connection object from previous attempt.
	logrus.Infof("NSM_Heal(5.1-%v) Healing Src Update... %v", healID, cc)
	if cc.Request != nil {
		request := cc.Request.Clone()
		request.SetRequestConnection(cc.GetConnectionSource())

		p.requestOrClose(ctx, fmt.Sprintf("NSM_Heal(5.2-%v) ", healID), request, cc)
		return true
	}
	return false
}

func (p *healProcessor) healDstNmgrDown(healID string, cc *model.ClientConnection) bool {
	logrus.Infof("NSM_Heal(6.1-%v) Starting DST + NSMGR Heal...", healID)

	ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout*3)
	defer cancel()

	networkService := cc.GetNetworkService()

	var endpointName string
	// Wait for exact same NSE to be available with NSMD connection alive.
	if cc.Endpoint != nil {
		endpointName = cc.Endpoint.GetNetworkserviceEndpoint().GetEndpointName()
		if !p.waitNSE(ctx, cc, endpointName, networkService, p.nseIsSameAndAvailable) {
			cc.GetConnectionDestination().SetID("-")
		}
	}
	requestCtx, requestCancel := context.WithTimeout(context.Background(), p.properties.HealRequestTimeout)
	defer requestCancel()
	recoveredConnection, err := p.conManager.request(requestCtx, cc.Request, cc)
	if err != nil {
		logrus.Errorf("NSM_Heal(6.2.1-%v) Failed to heal connection with same NSE from registry: %v", healID, err)
		if endpointName != "" {
			logrus.Infof("NSM_Heal(6.2.2-%v) Waiting for another NSEs...", healID)
			// In this case, most probable both NSMD and NSE are die, and registry was outdated on moment of waitNSE.
			if p.waitNSE(ctx, cc, endpointName, networkService, p.nseIsNewAndAvailable) {
				// Ok we have NSE, lets retry request
				requestCtx, requestCancel := context.WithTimeout(context.Background(), p.properties.HealRequestTimeout)
				defer requestCancel()
				recoveredConnection, err = p.conManager.request(requestCtx, cc.Request, cc)
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
		p.model.DeleteClientConnection(cc.ConnectionID)
		if err != nil {
			logrus.Errorf("NSM_Heal(6.4.2-%v) Error in Recovery Close: %v", healID, err)
		}
	} else {
		logrus.Infof("NSM_Heal(6.5-%v) Heal: Connection recovered: %v", healID, recoveredConnection)
		return true
	}
	return false
}

func (p *healProcessor) requestOrClose(ctx context.Context, logPrefix string, request networkservice.Request, cc *model.ClientConnection) {
	logrus.Infof("%v delegate to Request %v", logPrefix, request)
	connection, err := p.conManager.request(ctx, request, cc)
	if err != nil {
		logrus.Errorf("%v Failed to heal connection: %v", logPrefix, err)
		// Close in case of any errors in recovery.
		if err = p.conManager.Close(context.Background(), cc); err != nil {
			logrus.Errorf("%v Error in Recovery Close: %v", logPrefix, err)
		}
	} else {
		logrus.Infof("%v Heal: Connection recovered: %v", logPrefix, connection)
	}
}

type nseValidator func(ctx context.Context, endpoint string, reg *registry.NSERegistration) bool

func (p *healProcessor) nseIsNewAndAvailable(ctx context.Context, endpointName string, reg *registry.NSERegistration) bool {
	if endpointName != "" && reg.GetNetworkserviceEndpoint().GetEndpointName() == endpointName {
		// Skip ignored endpoint
		return false
	}

	// Check local only if not waiting for specific NSE.
	if p.model.GetNsm().GetName() == reg.GetNetworkServiceManager().GetName() {
		// Another local endpoint is found, success.
		return true
	}

	// Check remote is accessible.
	if p.nseManager.checkUpdateNSE(ctx, reg) {
		logrus.Infof("NSE is available and Remote NSMD is accessible. %s.", reg.NetworkServiceManager.Url)
		// We are able to connect to NSM with required NSE
		return true
	}

	return false
}

func (p *healProcessor) nseIsSameAndAvailable(ctx context.Context, endpointName string, reg *registry.NSERegistration) bool {
	if reg.GetNetworkserviceEndpoint().GetEndpointName() != endpointName {
		return false
	}

	// Our endpoint, we need to check if it is remote one and NSM is accessible.

	// Check remote is accessible.
	if p.nseManager.checkUpdateNSE(ctx, reg) {
		logrus.Infof("NSE is available and Remote NSMD is accessible. %s.", reg.NetworkServiceManager.Url)
		// We are able to connect to NSM with required NSE
		return true
	}

	return false
}

func (p *healProcessor) waitNSE(ctx context.Context, cc *model.ClientConnection, endpointName, networkService string, nseValidator nseValidator) bool {
	discoveryClient, err := p.serviceRegistry.DiscoveryClient()
	if err != nil {
		logrus.Errorf("Failed to connect to Registry... %v", err)
		// Still try to recovery
		return false
	}

	st := time.Now()

	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: networkService,
	}

	defer func() {
		logrus.Infof("Complete Waiting for Remote NSE/NSMD with network service %s. Since elapsed: %v", networkService, time.Since(st))
	}()

	for {
		logrus.Infof("NSM: RemoteNSE: Waiting for NSE with network service %s. Since elapsed: %v", networkService, time.Since(st))

		endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
		if err == nil {
			for _, ep := range endpointResponse.NetworkServiceEndpoints {
				reg := &registry.NSERegistration{
					NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[ep.GetNetworkServiceManagerName()],
					NetworkserviceEndpoint: ep,
					NetworkService:         endpointResponse.GetNetworkService(),
				}

				if nseValidator(ctx, endpointName, reg) {
					cc.Endpoint = reg
					return true
				}
			}
		}

		if time.Since(st) > p.properties.HealDSTNSEWaitTimeout {
			logrus.Errorf("Timeout waiting for NetworkService: %v", networkService)
			return false
		}
		// Wait a bit
		<-time.After(p.properties.HealDSTNSEWaitTick)
	}
}
