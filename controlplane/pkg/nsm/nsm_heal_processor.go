package nsm

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type networkServiceHealProcessor interface {
	Heal(clientConnection nsm.ClientConnection, healState nsm.HealState)
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
	request(ctx context.Context, request networkservice.Request, existingCC *model.ClientConnection) (connection.Connection, error)
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

func (p *healProcessor) Heal(clientConnection nsm.ClientConnection, healState nsm.HealState) {
	cc := clientConnection.(*model.ClientConnection)

	healID := create_logid()
	logrus.Infof("NSM_Heal(%v) %v", healID, cc)

	if !p.properties.HealEnabled {
		logrus.Infof("NSM_Heal(%v) Is Disabled/Closing connection %v", healID, cc)

		err := p.conManager.Close(context.Background(), cc)
		if err != nil {
			logrus.Errorf("NSM_Heal(%v) Error in Close: %v", healID, err)
		}
		return
	}

	if modelCC := p.model.GetClientConnection(cc.GetID()); modelCC == nil {
		logrus.Errorf("NSM_Heal(%v) Trying to heal not existing connection", healID)
		return
	} else if modelCC.ConnectionState != model.ClientConnectionReady {
		logrus.Errorf("NSM_Heal(%v) Trying to heal connection in bad state", healID)
		return
	}

	p.model.ApplyClientConnectionChanges(cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionHealing
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
			defer func() {
				logrus.Infof("NSM_Heal(%v) Connection %v healing state is finished...", e.healID, e.cc.GetID())
			}()

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
				logrus.Infof("NSM_Heal(%v) Heal: Connection recovered: %v", e.healID, e.cc)
			} else {
				// Close both connection and dataplane
				err := p.conManager.Close(context.Background(), e.cc)
				if err != nil {
					logrus.Errorf("NSM_Heal(%v) Error in Recovery: %v", e.healID, err)
				}
			}
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout*3)
	defer cancel()

	logrus.Infof("NSM_Heal(2.2-%v) Starting DST Heal...", healID)
	// We are client NSMd, we need to try recover our connection srv.

	endpointName := cc.Endpoint.GetNetworkServiceEndpoint().GetName()
	// Wait for NSE not equal to down one, since we know it will be re-registered with new endpoint name.
	if !p.waitNSE(ctx, cc, endpointName, cc.GetNetworkService(), p.nseIsNewAndAvailable) {
		cc.GetConnectionDestination().SetID("-") // We need to mark this as new connection.
	}

	// Fallback to heal with choose of new NSE.
	requestCtx, requestCancel := context.WithTimeout(context.Background(), p.properties.HealRequestTimeout)
	defer requestCancel()

	logrus.Infof("NSM_Heal(2.3.0-%v) Starting Heal by calling request: %v", healID, cc.Request)
	if _, err := p.conManager.request(requestCtx, cc.Request, cc); err != nil {
		logrus.Errorf("NSM_Heal(2.3.1-%v) Failed to heal connection: %v", healID, err)
		return false
	}

	return true
}

func (p *healProcessor) healDataplaneDown(healID string, cc *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout)
	defer cancel()

	// Dataplane is down, we only need to re-programm dataplane.
	// 1. Wait for dataplane to appear.
	logrus.Infof("NSM_Heal(3.1-%v) Waiting for Dataplane to recovery...", healID)
	if err := p.serviceRegistry.WaitForDataplaneAvailable(ctx, p.model, p.properties.HealDataplaneTimeout); err != nil {
		logrus.Errorf("NSM_Heal(3.1-%v) Dataplane is not available on recovery for timeout %v: %v", p.properties.HealDataplaneTimeout, healID, err)
		return false
	}
	logrus.Infof("NSM_Heal(3.2-%v) Dataplane is now available...", healID)

	// 3.3. Set source connection down
	p.model.ApplyClientConnectionChanges(cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.GetConnectionSource().SetConnectionState(connection.StateDown)
	})

	if cc.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		// Recovery will be performed by NSM client side.
		logrus.Infof("NSM_Heal(3.4-%v)  Healing will be continued on source side...", healID)
		return true
	}

	request := cc.Request.Clone()
	request.SetRequestConnection(cc.GetConnectionSource())

	if _, err := p.conManager.request(ctx, request, cc); err != nil {
		logrus.Errorf("NSM_Heal(3.5-%v) Failed to heal connection: %v", healID, err)
		return false
	}

	return true
}

func (p *healProcessor) healDstUpdate(healID string, cc *model.ClientConnection) bool {
	ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout)
	defer cancel()

	// Destination is updated.
	// Update request to contain a proper connection object from previous attempt.
	logrus.Infof("NSM_Heal(5.1-%v) Healing Src Update... %v", healID, cc)
	if cc.Request == nil {
		return false
	}

	request := cc.Request.Clone()
	request.SetRequestConnection(cc.GetConnectionSource())

	if _, err := p.conManager.request(ctx, request, cc); err != nil {
		logrus.Errorf("NSM_Heal(5.2-%v) Failed to heal connection: %v", healID, err)
		return false
	}

	return true
}

func (p *healProcessor) healDstNmgrDown(healID string, cc *model.ClientConnection) bool {
	logrus.Infof("NSM_Heal(6.1-%v) Starting DST + NSMGR Heal...", healID)

	ctx, cancel := context.WithTimeout(context.Background(), p.properties.HealTimeout*3)
	defer cancel()

	networkService := cc.GetNetworkService()

	var endpointName string
	// Wait for exact same NSE to be available with NSMD connection alive.
	if cc.Endpoint != nil {
		endpointName = cc.Endpoint.GetNetworkServiceEndpoint().GetName()
		if !p.waitNSE(ctx, cc, endpointName, networkService, p.nseIsSameAndAvailable) {
			cc.GetConnectionDestination().SetID("-") // We need to mark this as new connection.
		}
	}

	requestCtx, requestCancel := context.WithTimeout(context.Background(), p.properties.HealRequestTimeout)
	defer requestCancel()

	if _, err := p.conManager.request(requestCtx, cc.Request, cc); err != nil {
		logrus.Warnf("NSM_Heal(6.2.1-%v) Failed to heal connection with same NSE from registry: %v", healID, err)

		// 6.2.2. We are still healing
		p.model.ApplyClientConnectionChanges(cc.GetID(), func(modelCC *model.ClientConnection) {
			modelCC.ConnectionState = model.ClientConnectionHealing
		})

		// In this case, most probable both NSMD and NSE are die, and registry was outdated on moment of waitNSE.
		if endpointName == "" || !p.waitNSE(ctx, cc, endpointName, networkService, p.nseIsNewAndAvailable) {
			return false
		}

		// Ok we have NSE, lets retry request
		requestCtx, requestCancel = context.WithTimeout(context.Background(), p.properties.HealRequestTimeout)
		defer requestCancel()

		if _, err := p.conManager.request(requestCtx, cc.Request, cc); err != nil {
			logrus.Errorf("NSM_Heal(6.2.3-%v) Failed to heal connection: %v", healID, err)
			return false
		}
	}

	return true
}

type nseValidator func(ctx context.Context, endpoint string, reg *registry.NSERegistration) bool

func (p *healProcessor) nseIsNewAndAvailable(ctx context.Context, endpointName string, reg *registry.NSERegistration) bool {
	if endpointName != "" && reg.GetNetworkServiceEndpoint().GetName() == endpointName {
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
	if reg.GetNetworkServiceEndpoint().GetName() != endpointName {
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
					NetworkServiceEndpoint: ep,
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
