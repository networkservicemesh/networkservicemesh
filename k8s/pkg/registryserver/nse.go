package registryserver

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model/registry"
	"github.com/ligato/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	nsmClientset "github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"time"

	"golang.org/x/net/context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type registryService struct {
	clientset *nsmClientset.Clientset
}

func (rs registryService) RegisterNSE(ctx context.Context, request *registry.RegisterNSERequest) (*registry.RegisterNSEResponse, error) {
	// get network service
	_, err := rs.clientset.Networkservicemesh().NetworkServices("default").Create(&v1.NetworkService{
		ObjectMeta: metav1.ObjectMeta{
			Name: request.NetworkServiceName,
		},
		Spec: v1.NetworkServiceSpec{
			Payload: request.Payload,
		},
		Status: v1.NetworkServiceStatus{},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	sum := md5.Sum([]byte(request.NsmUrl))
	sumSlice := sum[:]
	nsmName := hex.EncodeToString(sumSlice)

	_, err = rs.clientset.Networkservicemesh().NetworkServiceManagers("default").Create(&v1.NetworkServiceManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsmName,
		},
		Spec: v1.NetworkServiceManagerSpec{},
		Status: v1.NetworkServiceManagerStatus{
			LastSeen: metav1.Time{time.Now()},
			URL:      request.NsmUrl,
			State:    v1.RUNNING,
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	_, err = rs.clientset.Networkservicemesh().NetworkServiceEndpoints("default").Create(&v1.NetworkServiceEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name: request.EndpointName,
		},
		Spec: v1.NetworkServiceEndpointSpec{
			NetworkServiceName: nsmName,
			NsmName:            request.NetworkServiceName,
		},
		Status: v1.NetworkServiceEndpointStatus{
			LastSeen: metav1.Time{time.Now()},
			State:    v1.RUNNING,
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}
	return &registry.RegisterNSEResponse{}, nil

}

func (registryService) RemoveNSE(context.Context, *registry.RemoveNSERequest) (*registry.RemoveNSEResponse, error) {
	panic("implement me")
}

func (registryService) FindNetworkService(context.Context, *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	panic("implement me")
}
