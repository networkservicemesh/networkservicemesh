package registryserver

import (
	"log"
	"time"

	"github.com/go-errors/errors"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	nsmClientset "github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NodeNameLabelKey = "nodeName"
)

type registryService struct {
	clientset *nsmClientset.Clientset
	nsmName   string
}

func (rs registryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE(%v)", request)
	// get network service
	networkService, err := rs.clientset.Networkservicemesh().NetworkServices("default").Create(&v1.NetworkService{
		ObjectMeta: metav1.ObjectMeta{
			Name: request.NetworkService.GetName(),
		},
		Spec: v1.NetworkServiceSpec{
			Payload: request.NetworkService.GetPayload(),
		},
		Status: v1.NetworkServiceStatus{},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	if request.GetNetworkServiceManager().GetUrl() == "" {
		return nil, errors.New("NSERegistration.NetworkServiceManager.Url must be defined")
	}

	nsm := &v1.NetworkServiceManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: rs.nsmName,
		},
		Spec: v1.NetworkServiceManagerSpec{},
		Status: v1.NetworkServiceManagerStatus{
			LastSeen: metav1.Time{Time: time.Now()},
			URL:      request.GetNetworkServiceManager().GetUrl(),
			State:    v1.RUNNING,
		},
	}

	_, err = rs.clientset.Networkservicemesh().NetworkServiceManagers("default").Create(nsm)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	labels := request.GetNetworkserviceEndpoint().GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["networkservicename"] = request.GetNetworkService().GetName()
	if request.GetNetworkserviceEndpoint() != nil {
		nseResponse, err := rs.clientset.Networkservicemesh().NetworkServiceEndpoints("default").Create(&v1.NetworkServiceEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: request.GetNetworkService().GetName(),
				Labels:       labels,
			},
			Spec: v1.NetworkServiceEndpointSpec{
				NetworkServiceName: request.GetNetworkService().GetName(),
				NsmName:            rs.nsmName,
			},
			Status: v1.NetworkServiceEndpointStatus{
				State: v1.RUNNING,
			},
		})
		if err != nil {
			return nil, err
		}

		request.NetworkserviceEndpoint = &registry.NetworkServiceEndpoint{
			NetworkServiceName:        nseResponse.Spec.NetworkServiceName,
			Payload:                   networkService.Spec.Payload,
			NetworkServiceManagerName: nsm.GetObjectMeta().GetName(),
			EndpointName:              nseResponse.GetObjectMeta().GetName(),
			Labels:                    nseResponse.GetObjectMeta().GetLabels(),
			State:                     string(nseResponse.Status.State),
		}
	}
	logrus.Infof("Returned from RegisterNSE: %v", request)
	return request, nil

}

func (rs registryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	if err := rs.clientset.Networkservicemesh().NetworkServiceEndpoints("default").Delete(request.EndpointName, &metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (rs registryService) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	service, e := rs.clientset.Networkservicemesh().NetworkServices("default").Get(request.NetworkServiceName, metav1.GetOptions{})
	if e != nil {
		return nil, e
	}
	payload := service.Spec.Payload

	lo := metav1.ListOptions{}
	lo.LabelSelector = "networkservicename=" + request.NetworkServiceName
	endpointList, e := rs.clientset.Networkservicemesh().NetworkServiceEndpoints("default").List(lo)
	if e != nil {
		return nil, e
	}

	logrus.Println(len(endpointList.Items))
	NSEs := make([]*registry.NetworkServiceEndpoint, len(endpointList.Items))
	NSMs := make(map[string]*registry.NetworkServiceManager)
	for i, endpoint := range endpointList.Items {
		log.Println(endpoint.Name)
		NSEs[i] = &registry.NetworkServiceEndpoint{
			EndpointName:              endpoint.Name,
			NetworkServiceName:        endpoint.Spec.NetworkServiceName,
			NetworkServiceManagerName: endpoint.Spec.NsmName,
			Payload:                   payload,
			Labels:                    endpoint.ObjectMeta.Labels,
		}
		// TODO check in the NSMs map first before potentially looking up again
		manager, e := rs.clientset.Networkservicemesh().NetworkServiceManagers("default").Get(endpoint.Spec.NsmName, metav1.GetOptions{})
		if e != nil {
			return nil, e
		}
		NSMs[endpoint.Spec.NsmName] = &registry.NetworkServiceManager{
			Name: manager.ObjectMeta.Name,
			Url:  manager.Status.URL,
			LastSeen: &timestamp.Timestamp{
				Seconds: manager.Status.LastSeen.ProtoTime().Seconds,
				Nanos:   manager.Status.LastSeen.ProtoTime().Nanos,
			},
		}
	}

	response := &registry.FindNetworkServiceResponse{
		Payload: payload,
		NetworkService: &registry.NetworkService{
			Name:    service.ObjectMeta.Name,
			Payload: service.Spec.Payload,
		},
		NetworkServiceManagers:  NSMs,
		NetworkServiceEndpoints: NSEs,
	}
	return response, nil
}
