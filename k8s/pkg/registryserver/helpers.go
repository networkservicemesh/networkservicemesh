package registryserver

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func mapNsmToCustomResource(nsm *registry.NetworkServiceManager) *v1.NetworkServiceManager {
	return &v1.NetworkServiceManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsm.GetName(),
		},
		Spec: v1.NetworkServiceManagerSpec{},
		Status: v1.NetworkServiceManagerStatus{
			LastSeen: metav1.Time{Time: time.Now()},
			URL:      nsm.GetUrl(),
			State:    v1.RUNNING,
		},
	}
}

func mapNsmFromCustomResource(cdr *v1.NetworkServiceManager) *registry.NetworkServiceManager {
	lastSeen, err := ptypes.TimestampProto(cdr.Status.LastSeen.Time)
	if err != nil {
		logrus.Errorf("Failed time conversion of %v", cdr.Status.LastSeen)
	}

	return &registry.NetworkServiceManager{
		Name:     cdr.GetName(),
		Url:      cdr.Status.URL,
		State:    string(cdr.Status.State),
		LastSeen: lastSeen,
	}
}
