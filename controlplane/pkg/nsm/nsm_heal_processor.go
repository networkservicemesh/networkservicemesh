package nsm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/properties"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/api/nsm"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
)

type healProcessor struct {
	serviceRegistry serviceregistry.ServiceRegistry
	model           model.Model
	props           *properties.Properties

	healCancellers      map[string]func()
	healCancellersMutex sync.Mutex
	manager             nsm.NetworkServiceRequestManager
	nseManager          nsm.NetworkServiceEndpointManager

	eventCh chan healEvent
}

type healEvent struct {
	healID    string
	cc        *model.ClientConnection
	healState nsm.HealState
	ctx       context.Context
}

func newNetworkServiceHealProcessor(
	serviceRegistry serviceregistry.ServiceRegistry,
	model model.Model,
	properties *properties.Properties,
	manager nsm.NetworkServiceRequestManager,
	nseManager nsm.NetworkServiceEndpointManager) nsm.NetworkServiceHealProcessor {
	p := &healProcessor{
		serviceRegistry: serviceRegistry,
		model:           model,
		props:           properties,
		manager:         manager,
		nseManager:      nseManager,
		eventCh:         make(chan healEvent, 1),
		healCancellers:  make(map[string]func()),
	}
	go p.serve()

	return p
}

func (p *healProcessor) Heal(ctx context.Context, clientConnection nsm.ClientConnection, healState nsm.HealState) {
	cc := clientConnection.(*model.ClientConnection)

	opName := "Heal"
	if clientConnection.GetConnectionSource().IsRemote() {
		opName = "RemoteHeal"
	}
	// We need to create new context, since we need to control a lifetime.
	span := spanhelper.CopySpan(context.Background(), spanhelper.GetSpanHelper(ctx), opName)
	defer span.Finish()
	ctx = span.Context()

	logger := span.Logger()
	ctx = common.WithLog(ctx, logger)

	ctx, cancelFunc := context.WithCancel(ctx)

	p.healCancellersMutex.Lock()
	p.healCancellers[clientConnection.GetID()] = cancelFunc
	p.healCancellersMutex.Unlock()

	healID := create_logid()
	logger.Infof("NSM_Heal(%v) %v", healID, cc)

	if !p.props.HealEnabled {
		logger.Infof("NSM_Heal(%v) Is Disabled/Closing connection %v", healID, cc)
		_ = p.CloseConnection(ctx, cc)
		return
	}

	if modelCC := p.model.GetClientConnection(cc.GetID()); modelCC == nil {
		logger.Errorf("NSM_Heal(%v) Trying to heal not existing connection", healID)
		return
	} else if modelCC.ConnectionState != model.ClientConnectionReady {
		healErr := errors.Errorf("NSM_Heal(%v) Trying to heal connection in bad state", healID)
		span.LogError(healErr)
		return
	}

	cc = p.model.ApplyClientConnectionChanges(ctx, cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.ConnectionState = model.ClientConnectionHealingBegin
	})

	p.eventCh <- healEvent{
		healID:    healID,
		cc:        cc,
		healState: healState,
		ctx:       ctx,
	}
}

func (p *healProcessor) CloseConnection(ctx context.Context, conn nsm.ClientConnection) error {
	var err error

	// Cancel context after closing.
	defer func() {
		p.healCancellersMutex.Lock()
		contextCancelFunc, isHealing := p.healCancellers[conn.GetID()]
		if isHealing {
			logrus.Infof("Closing client connection while healing, cancel client context")
			delete(p.healCancellers, conn.GetID())
			contextCancelFunc()
		}
		p.healCancellersMutex.Unlock()
	}()

	if conn.GetConnectionSource().IsRemote() {
		_, err = p.manager.RemoteManager().Close(ctx, conn.GetConnectionSource())
	} else {
		_, err = p.manager.LocalManager(conn).Close(ctx, conn.GetConnectionSource())
	}
	if err != nil {
		logrus.Errorf("NSM_Heal Error in Close: %v", err)
	}
	return err
}

func (p *healProcessor) serve() {
	for {
		e, ok := <-p.eventCh
		if !ok {
			return
		}

		go func() {
			span := spanhelper.FromContext(e.ctx, "heal")
			defer span.Finish()
			ctx := span.Context()

			logger := span.Logger()
			defer func() {
				logger.Infof("NSM_Heal(%v) Connection %v healing state is finished...", e.healID, e.cc.GetID())
			}()

			healed := false

			ctx = common.WithModelConnection(ctx, e.cc)

			switch e.healState {
			case nsm.HealStateDstDown:
				healed = p.healDstDown(ctx, e.cc)
			case nsm.HealStateForwarderDown:
				healed = p.healForwarderDown(ctx, e.cc)
			case nsm.HealStateDstUpdate:
				healed = p.healDstUpdate(ctx, e.cc)
			case nsm.HealStateDstNmgrDown:
				healed = p.healDstMgrDown(ctx, e.cc)
			}

			if healed {
				span.LogValue("status", "healed")
				logger.Infof("NSM_Heal(%v) Heal: Connection recovered: %v", e.healID, e.cc)
				p.healCancellersMutex.Lock()
				delete(p.healCancellers, e.cc.GetID())
				p.healCancellersMutex.Unlock()
			} else {
				span.LogValue("status", "closing")
				_ = p.CloseConnection(ctx, e.cc)
			}
		}()
	}
}

func (p *healProcessor) healDstDown(ctx context.Context, cc *model.ClientConnection) bool {
	span := spanhelper.FromContext(ctx, "healDstDown")
	defer span.Finish()

	logger := span.Logger()
	// Update context
	ctx = span.Context()

	logger.Infof("NSM_Heal(1.1.1) Checking if DST die is NSMD/DST die...")
	// Check if this is a really HealStateDstDown or HealStateDstNmgrDown
	if !p.nseManager.IsLocalEndpoint(cc.Endpoint) {
		waitCtx, waitCancel := context.WithTimeout(ctx, p.props.HealTimeout*3)
		defer waitCancel()
		remoteNsmClient, err := p.nseManager.CreateNSEClient(waitCtx, cc.Endpoint)
		if remoteNsmClient != nil {
			_ = remoteNsmClient.Cleanup()
		}
		if err != nil {
			// This is NSMD die case.
			logger.Infof("NSM_Heal(1.1.2) Connection healing state is %v...", nsm.HealStateDstNmgrDown)
			span.LogError(err)
			return p.healDstMgrDown(ctx, cc)
		}
	}
	logger.Infof("NSM_Heal(1.1.2) Connection healing state is %v...", nsm.HealStateDstDown)
	// Destination is down, we need to find it again.
	if cc.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		logger.Infof("NSM_Heal(2.1) Remote NSE heal is done on source side")
		return false
	}
	logger.Infof("NSM_Heal(2.2) Starting DST Heal...")
	// We are client NSMd, we need to try recover our connection srv.
	// Wait for NSE not equal to down one, since we know it will be re-registered with new endpoint name.
	ctx = p.waitForNSEUpdateContext(ctx, cc.Endpoint, cc)
	// Fallback to heal with choose of new NSE.
	for attempt := 0; attempt < p.props.HealRetryCount; attempt++ {
		// If client context is cancelled, we need to stop attempts.
		if ctx.Err() != nil {
			logger.Info("Client context is broken, stopping heal attempts")
			break
		}

		attemptSpan := spanhelper.FromContext(ctx, fmt.Sprintf("healing-attempt-%v", attempt))
		requestCtx, requestCancel := context.WithTimeout(attemptSpan.Context(), p.props.HealRequestTimeout)
		defer requestCancel()
		defer attemptSpan.Finish()

		logger.Infof("NSM_Heal(2.3.0) Starting Heal by calling request: %v", cc.Request)
		_, err := p.manager.LocalManager(cc).Request(requestCtx, cc.Request)
		span.LogError(err)
		if err == nil {
			return true
		}
		logger.Errorf("NSM_Heal(2.3.1) Failed to heal connection: %v. Delaying: %v", err, p.props.HealRetryDelay)
		if attempt+1 < p.props.HealRetryCount {
			attemptSpan.Finish()
			<-time.After(p.props.HealRetryDelay)
			continue
		}
	}
	return false
}

func (p *healProcessor) healForwarderDown(ctx context.Context, cc *model.ClientConnection) bool {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, p.props.HealTimeout)
	defer cancel()

	span := spanhelper.FromContext(ctx, "healForwarderDown")
	defer span.Finish()

	logger := span.Logger()
	// Forwarder is down, we only need to re-programm forwarder.
	// 1. Wait for forwarder to appear.
	logger.Infof("NSM_Heal(3.1) Waiting for Forwarder to recovery...")
	if err := p.serviceRegistry.WaitForForwarderAvailable(span.Context(), p.model, p.props.HealForwarderTimeout); err != nil {
		err = errors.Errorf("NSM_Heal(3.1) Forwarder is not available on recovery for timeout %v: %v", p.props.HealForwarderTimeout, err)
		span.LogError(err)
		return false
	}
	logger.Infof("NSM_Heal(3.2) Forwarder is now available...")

	// 3.3. Set source connection down
	p.model.ApplyClientConnectionChanges(span.Context(), cc.GetID(), func(modelCC *model.ClientConnection) {
		modelCC.Xcon.Source.State = connection.State_DOWN
	})

	if cc.Xcon.GetRemoteSource() != nil {
		// NSMd id remote one, we just need to close and return.
		// Recovery will be performed by NSM client side.
		logger.Infof("NSM_Heal(3.4)  Healing will be continued on source side...")
		return true
	}

	request := cc.Request.Clone()
	request.SetRequestConnection(cc.GetConnectionSource())

	if _, err := p.manager.LocalManager(cc).Request(span.Context(), cc.Request); err != nil {
		logger.Errorf("NSM_Heal(3.5) Failed to heal connection: %v", err)
		return false
	}

	return true
}

func (p *healProcessor) healDstUpdate(ctx context.Context, cc *model.ClientConnection) bool {
	span := spanhelper.FromContext(ctx, "healDstUpdate")
	defer span.Finish()
	ctx = span.Context()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, p.props.HealTimeout)
	defer cancel()

	logger := span.Logger()
	// Destination is updated.
	// Update request to contain a proper connection object from previous attempt.
	logger.Infof("NSM_Heal(5.1-%v) Healing Src Update... %v", cc)
	if cc.Request == nil {
		return false
	}

	request := cc.Request.Clone()
	request.SetRequestConnection(cc.GetConnectionSource())

	err := p.performRequest(ctx, request, cc)
	if err != nil {
		span.LogError(err)
		logger.Errorf("NSM_Heal(5.2) Failed to heal connection: %v", err)
		return false
	}

	return true
}

func (p *healProcessor) performRequest(ctx context.Context, request *networkservice.NetworkServiceRequest, cc nsm.ClientConnection) error {
	span := spanhelper.FromContext(ctx, "performRequest")
	defer span.Finish()
	if request.GetConnection().IsRemote() {
		resp, err := p.manager.RemoteManager().Request(span.Context(), request)
		span.LogObject("response", resp)
		return err
	}
	resp, err := p.manager.LocalManager(cc).Request(span.Context(), request)
	span.LogObject("response", resp)
	return err
}

func (p *healProcessor) healDstMgrDown(ctx context.Context, cc *model.ClientConnection) bool {
	span := spanhelper.FromContext(ctx, "healDstNsmgrDown")
	defer span.Finish()
	ctx = span.Context()
	logger := span.Logger()
	logger.Infof("NSM_Heal(6.1) Starting DST + NSMGR Heal...")

	var endpointName string
	// Wait for exact same NSE to be available with NSMD connection alive.
	if cc.Endpoint != nil {
		endpointName = cc.Endpoint.GetNetworkServiceEndpoint().GetName()
		waitCtx, waitCancel := context.WithTimeout(ctx, p.props.HealTimeout*3)
		defer waitCancel()
		if !p.waitNSE(waitCtx, endpointName, cc.GetNetworkService(), p.nseIsSameAndAvailable) {
			span.LogValue("waitNSE", "failed to find endpoint by name with timeout")
			ctx = common.WithIgnoredEndpoints(ctx, map[registry.EndpointNSMName]*registry.NSERegistration{
				cc.Endpoint.GetEndpointNSMName(): cc.Endpoint,
			})
		}
	}
	for attempt := 0; attempt < p.props.HealRetryCount; attempt++ {
		attemptSpan := spanhelper.FromContext(ctx, fmt.Sprintf("healing-attempt-%v", attempt))
		requestCtx, requestCancel := context.WithTimeout(attemptSpan.Context(), p.props.HealRequestTimeout)
		defer requestCancel()
		defer attemptSpan.Finish()
		err := p.performRequest(requestCtx, cc.Request, cc)
		if err == nil {
			attemptSpan.LogObject("state", "healed")
			return true
		}
		err = errors.Errorf("heal(6.2.3) Failed to heal connection: %v. Delaying: %v", err, p.props.HealRetryDelay)
		span.LogError(err)
		attemptSpan.Finish()
		if attempt+1 < p.props.HealRetryCount {
			attemptSpan.Finish()
			<-time.After(p.props.HealRetryDelay)
			continue
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

	// Check remote is accessible.
	if p.nseManager.CheckUpdateNSE(ctx, reg) {
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
	if p.nseManager.CheckUpdateNSE(ctx, reg) {
		logrus.Infof("NSE is available and Remote NSMD is accessible. %s.", reg.NetworkServiceManager.Url)
		// We are able to connect to NSM with required NSE
		return true
	}

	return false
}

func (p *healProcessor) waitNSE(ctx context.Context, endpointName, networkService string, nseValidator nseValidator) bool {
	span := spanhelper.FromContext(ctx, "waitNSE")
	defer span.Finish()
	ctx = span.Context()

	span.LogObject("endpointName", endpointName)
	span.LogObject("networkService", networkService)

	logger := span.Logger()
	discoveryClient, err := p.serviceRegistry.DiscoveryClient(span.Context())
	if err != nil {
		span.LogError(err)
		return false
	}

	st := time.Now()

	nseRequest := &registry.FindNetworkServiceRequest{
		NetworkServiceName: networkService,
	}

	defer func() {
		logger.Infof("Complete Waiting for Remote NSE/NSMD with network service %s. Since elapsed: %v", networkService, time.Since(st))
	}()

	for {
		logger.Infof("NSM: RemoteNSE: Waiting for NSE with network service %s. Since elapsed: %v", networkService, time.Since(st))

		// If client context was cancelled, we need to stop waiting.
		if ctx.Err() != nil {
			logger.Infof("Client context is cancelled, stop waiting for network service %s", networkService)
			return false
		}

		endpointResponse, err := discoveryClient.FindNetworkService(ctx, nseRequest)
		if err == nil {
			for _, ep := range endpointResponse.NetworkServiceEndpoints {
				reg := &registry.NSERegistration{
					NetworkServiceManager:  endpointResponse.GetNetworkServiceManagers()[ep.GetNetworkServiceManagerName()],
					NetworkServiceEndpoint: ep,
					NetworkService:         endpointResponse.GetNetworkService(),
				}

				if nseValidator(ctx, endpointName, reg) {
					return true
				}
			}
		}

		if time.Since(st) > p.props.HealDSTNSEWaitTimeout {
			span.LogError(errors.Errorf("timeout waiting for NetworkService: %v timeout: %v", networkService, time.Since(st)))
			return false
		}
		// Wait a bit
		<-time.After(p.props.HealDSTNSEWaitTick)
	}
}

func (p *healProcessor) waitForNSEUpdateContext(ctx context.Context, endpoint *registry.NSERegistration, cc *model.ClientConnection) context.Context {
	waitCtx, waitCancel := context.WithTimeout(ctx, p.props.HealTimeout*3)
	defer waitCancel()
	if !p.waitNSE(waitCtx, endpoint.NetworkServiceEndpoint.Name, cc.GetNetworkService(), p.nseIsNewAndAvailable) {
		// Mark endpoint as ignored.
		return common.WithIgnoredEndpoints(ctx, map[registry.EndpointNSMName]*registry.NSERegistration{
			endpoint.GetEndpointNSMName(): cc.Endpoint,
		})
	}
	return ctx
}
