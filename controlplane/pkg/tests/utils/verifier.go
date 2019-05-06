package utils

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	. "github.com/onsi/gomega"
	"testing"
)

type Verifier interface {
	Verify(t *testing.T)
}

type ModelVerifier interface {
	EndpointNotExists(name string) ModelVerifier
	EndpointExists(name string, nsm string) ModelVerifier

	ClientConnectionNotExists(connectionID string) ModelVerifier
	ClientConnectionExists(connectionID string, srcID, dstID string, remoteNSM, nse, dataplane string) ModelVerifier

	DataplaneNotExists(name string) ModelVerifier
	DataplaneExists(name string) ModelVerifier

	Verifier
}

type modelVerifier struct {
	model     model.Model
	verifiers []Verifier
}

func NewModelVerifier(model model.Model) ModelVerifier {
	return &modelVerifier{
		model:     model,
		verifiers: []Verifier{},
	}
}

func (v *modelVerifier) EndpointNotExists(name string) ModelVerifier {
	v.verifiers = append(v.verifiers, &endpointVerifier{
		exists: false,
		name:   name,

		model: v.model,
	})

	return v
}

func (v *modelVerifier) EndpointExists(name string, nsm string) ModelVerifier {
	v.verifiers = append(v.verifiers, &endpointVerifier{
		exists: true,
		name:   name,
		nsm:    nsm,

		model: v.model,
	})

	return v
}

func (v *modelVerifier) ClientConnectionNotExists(connectionID string) ModelVerifier {
	v.verifiers = append(v.verifiers, &clientConnectionVerifier{
		exists:       false,
		connectionID: connectionID,

		model: v.model,
	})

	return v
}

func (v *modelVerifier) ClientConnectionExists(connectionID, srcID, dstID string, remoteNSM, nse, dataplane string) ModelVerifier {
	v.verifiers = append(v.verifiers, &clientConnectionVerifier{
		exists:       true,
		connectionID: connectionID,
		srcID:        srcID,
		dstID:        dstID,
		remoteNSM:    remoteNSM,
		nse:          nse,
		dataplane:    dataplane,

		model: v.model,
	})

	return v
}

func (v *modelVerifier) DataplaneNotExists(name string) ModelVerifier {
	v.verifiers = append(v.verifiers, &dataplaneVerifier{
		exists: false,
		name:   name,

		model: v.model,
	})

	return v
}

func (v *modelVerifier) DataplaneExists(name string) ModelVerifier {
	v.verifiers = append(v.verifiers, &dataplaneVerifier{
		exists: true,
		name:   name,

		model: v.model,
	})

	return v
}

func (v *modelVerifier) Verify(t *testing.T) {
	for _, verifier := range v.verifiers {
		verifier.Verify(t)
	}
}

type endpointVerifier struct {
	exists bool
	name   string
	nsm    string

	model model.Model
}

func (v *endpointVerifier) Verify(t *testing.T) {
	nse := v.model.GetEndpoint(v.name)
	if !v.exists {
		Expect(nse).To(BeNil())
		return
	}

	Expect(nse).NotTo(BeNil())

	Expect(nse.Endpoint.GetNetworkServiceManager().GetName()).To(Equal(v.nsm))
}

type clientConnectionVerifier struct {
	exists       bool
	connectionID string
	srcID        string
	dstID        string
	remoteNSM    string
	nse          string
	dataplane    string

	model model.Model
}

func (v *clientConnectionVerifier) Verify(t *testing.T) {
	connection := v.model.GetClientConnection(v.connectionID)
	if !v.exists {
		Expect(connection).To(BeNil())
		return
	}

	Expect(connection).NotTo(BeNil())

	v.verifyXcon(connection.Xcon, t)
	Expect(connection.RemoteNsm.GetName()).To(Equal(v.remoteNSM))
	Expect(connection.Endpoint.GetNetworkserviceEndpoint().GetEndpointName()).To(Equal(v.nse))
	Expect(connection.Dataplane.RegisteredName).To(Equal(v.dataplane))
}

func (v *clientConnectionVerifier) verifyXcon(xcon *crossconnect.CrossConnect, t *testing.T) {
	if source := xcon.GetLocalSource(); source != nil {
		Expect(source.GetId()).To(Equal(v.srcID))
	} else if source := xcon.GetRemoteSource(); source != nil {
		Expect(source.GetId()).To(Equal(v.srcID))
	} else {
		t.Fatalf("Expected xcon.Source not to be nil")
	}

	if destination := xcon.GetLocalDestination(); destination != nil {
		Expect(destination.GetId()).To(Equal(v.dstID))
	} else if destination := xcon.GetRemoteDestination(); destination != nil {
		Expect(destination.GetId()).To(Equal(v.dstID))
	} else {
		t.Fatalf("Expected xcon.Destination not to be nil")
	}
}

type dataplaneVerifier struct {
	exists bool
	name   string

	model model.Model
}

func (v *dataplaneVerifier) Verify(t *testing.T) {
	dataplane := v.model.GetDataplane(v.name)
	if !v.exists {
		Expect(dataplane).To(BeNil())
		return
	}

	Expect(dataplane).NotTo(BeNil())
}
