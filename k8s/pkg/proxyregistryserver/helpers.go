package proxyregistryserver

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/sirupsen/logrus"
)

func mapNsmFromCustomResource(cr *v1.NetworkServiceManager) *registry.NetworkServiceManager {
	lastSeen, err := ptypes.TimestampProto(cr.Status.LastSeen.Time)
	if err != nil {
		logrus.Errorf("Failed time conversion of %v", cr.Status.LastSeen)
	}

	return &registry.NetworkServiceManager{
		Name:     cr.GetName(),
		Url:      cr.Status.URL,
		State:    string(cr.Status.State),
		LastSeen: lastSeen,
	}
}

func mapNseFromCustomResource(cr *v1.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		EndpointName:              cr.Name,
		NetworkServiceName:        cr.Spec.NetworkServiceName,
		NetworkServiceManagerName: cr.Spec.NsmName,
		Payload:                   cr.Spec.Payload,
		Labels:                    cr.ObjectMeta.Labels,
		State:                     string(cr.Status.State),
	}
}
