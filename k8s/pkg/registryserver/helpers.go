package registryserver

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
)

func mapNsmToCustomResource(nsm *registry.NetworkServiceManager) *v1.NetworkServiceManager {
	return &v1.NetworkServiceManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsm.GetName(),
		},
		Spec: v1.NetworkServiceManagerSpec{
			URL: nsm.GetUrl(),
		},
		Status: v1.NetworkServiceManagerStatus{
			LastSeen: metav1.Time{Time: time.Now()},
			State:    v1.RUNNING,
		},
	}
}

func mapNsmFromCustomResource(cr *v1.NetworkServiceManager) *registry.NetworkServiceManager {
	lastSeen, err := ptypes.TimestampProto(cr.Status.LastSeen.Time)
	if err != nil {
		logrus.Errorf("Failed time conversion of %v", cr.Status.LastSeen)
	}

	return &registry.NetworkServiceManager{
		Name:     cr.GetName(),
		Url:      cr.Spec.URL,
		State:    string(cr.Status.State),
		LastSeen: lastSeen,
	}
}

func mapNseFromCustomResource(cr *v1.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:                      cr.Name,
		NetworkServiceName:        cr.Spec.NetworkServiceName,
		NetworkServiceManagerName: cr.Spec.NsmName,
		Payload:                   cr.Spec.Payload,
		Labels:                    cr.ObjectMeta.Labels,
		State:                     string(cr.Status.State),
	}
}
