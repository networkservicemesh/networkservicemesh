package registryserver

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

const PodNameEnv = utils.EnvVar("POD_NAME")
const PodUIDEnv = utils.EnvVar("POD_UID")

func mapNsmToCustomResource(nsm *registry.NetworkServiceManager) *v1.NetworkServiceManager {
	nsmCr := &v1.NetworkServiceManager{
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

	podUid := PodUIDEnv.StringValue()
	podName := PodNameEnv.StringValue()
	if len(podUid) > 0 && len(podName) > 0 {
		nsmCr.OwnerReferences = append(nsmCr.OwnerReferences, generateOwnerReference(podUid, podName))
	}

	return nsmCr
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

func mapNseToCustomResource(nse *registry.NetworkServiceEndpoint, ns *registry.NetworkService, nsmName string) *v1.NetworkServiceEndpoint {
	labels := nse.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["networkservicename"] = ns.GetName()

	return &v1.NetworkServiceEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name:   nse.GetName(),
		},
		Spec: v1.NetworkServiceEndpointSpec{
			NetworkServiceName: ns.GetName(),
			Payload:            ns.GetPayload(),
			NsmName:            nsmName,
		},
		Status: v1.NetworkServiceEndpointStatus{
			State: v1.RUNNING,
		},
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

func generateOwnerReference(podUID, podName string) metav1.OwnerReference {
	uid := types.UID(podUID)
	return metav1.OwnerReference{
		APIVersion:         "v1",
		Kind:               "Pod",
		Name:               podName,
		UID:                uid,
		Controller:         proto.Bool(true),
		BlockOwnerDeletion: proto.Bool(false),
	}
}
