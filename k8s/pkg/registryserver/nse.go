package registryserver

import (
	"crypto/md5"
	"encoding/hex"
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

type registryService struct {
	clientset *nsmClientset.Clientset
}

func (rs registryService) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	logrus.Infof("Received RegisterNSE(%v)", request)
	// get network service
	_, err := rs.clientset.Networkservicemesh().NetworkServices("default").Create(&v1.NetworkService{
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

	sum := md5.Sum([]byte(request.GetNetworkServiceManager().GetUrl()))
	sumSlice := sum[:]
	nsmName := hex.EncodeToString(sumSlice)

	_, err = rs.clientset.Networkservicemesh().NetworkServiceManagers("default").Create(&v1.NetworkServiceManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsmName,
		},
		Spec: v1.NetworkServiceManagerSpec{},
		Status: v1.NetworkServiceManagerStatus{
			LastSeen: metav1.Time{Time: time.Now()},
			URL:      request.GetNetworkServiceManager().GetUrl(),
			State:    v1.RUNNING,
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}
	if request.GetNetworkserviceEndpoint() != nil {
		nseResponse, err := rs.clientset.Networkservicemesh().NetworkServiceEndpoints("default").Create(&v1.NetworkServiceEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: request.GetNetworkService().GetName(),
				Labels:       map[string]string{"networkservicename": request.GetNetworkService().GetName()},
			},
			Spec: v1.NetworkServiceEndpointSpec{
				NetworkServiceName: request.GetNetworkService().GetName(),
				NsmName:            nsmName,
			},
			Status: v1.NetworkServiceEndpointStatus{
				State: v1.RUNNING,
			},
		})
		if err != nil {
			return nil, err
		}

		request.GetNetworkserviceEndpoint().EndpointName = nseResponse.Name
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
		NSEs[i] = &registry.NetworkServiceEndpoint{}
		NSEs[i].EndpointName = endpoint.Name
		manager, e := rs.clientset.Networkservicemesh().NetworkServiceManagers("default").Get(endpoint.Spec.NsmName, metav1.GetOptions{})
		if e != nil {
			return nil, e
		}
		// TODO delete line "NSEs[i].Labels["nsmurl"] = manager.Status.URL"
		NSEs[i].Labels["nsmurl"] = manager.Status.URL
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
