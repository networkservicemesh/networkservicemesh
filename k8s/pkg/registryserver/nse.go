package registryserver

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"time"

	"github.com/go-errors/errors"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	nsmClientset "github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type registryService struct {
	clientset *nsmClientset.Clientset
}

func (rs registryService) RegisterNSE(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	logrus.Error("Received RegisterNSE request", request)
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
		logrus.Error("k8s error adding netsvc", request, err)
		return nil, err
	}

	nsmurl := ""
	nsmurl, ok := request.Labels["nsmurl"]
	if !ok {
		logrus.Error("nsmurl must be defined", request)
		return nil, errors.New("nsmurl must be defined")
	}

	sum := md5.Sum([]byte(nsmurl))
	sumSlice := sum[:]
	nsmName := hex.EncodeToString(sumSlice)

	_, err = rs.clientset.Networkservicemesh().NetworkServiceManagers("default").Create(&v1.NetworkServiceManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsmName,
		},
		Spec: v1.NetworkServiceManagerSpec{},
		Status: v1.NetworkServiceManagerStatus{
			LastSeen: metav1.Time{time.Now()},
			URL:      nsmurl,
			State:    v1.RUNNING,
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		logrus.Error("k8s error adding nsm", request, err)
		return nil, err
	}

	nseResponse, err := rs.clientset.Networkservicemesh().NetworkServiceEndpoints("default").Create(&v1.NetworkServiceEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: request.GetNetworkServiceName(),
			Labels:       map[string]string{"networkservicename": request.GetNetworkServiceName()},
		},
		Spec: v1.NetworkServiceEndpointSpec{
			NetworkServiceName: request.NetworkServiceName,
			NsmName:            nsmName,
		},
		Status: v1.NetworkServiceEndpointStatus{
			State: v1.RUNNING,
		},
	})
	if err != nil {
		logrus.Error("k8s error adding nse", request, err)
		return nil, err
	}

	request.EndpointName = nseResponse.Name

	return request, nil

}

func (rs registryService) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*common.Empty, error) {
	if err := rs.clientset.Networkservicemesh().NetworkServiceEndpoints("default").Delete(request.EndpointName, &metav1.DeleteOptions{}); err != nil {
		return nil, err
	}
	return &common.Empty{}, nil
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
	for i, endpoint := range endpointList.Items {
		log.Println(endpoint.Name)
		NSEs[i] = &registry.NetworkServiceEndpoint{}
		NSEs[i].EndpointName = endpoint.Name
		manager, e := rs.clientset.Networkservicemesh().NetworkServiceManagers("default").Get(endpoint.Spec.NsmName, metav1.GetOptions{})
		if e != nil {
			return nil, e
		}
		NSEs[i].Labels["nsmurl"] = manager.Status.URL
	}

	response := &registry.FindNetworkServiceResponse{
		Payload:                 payload,
		NetworkServiceEndpoints: NSEs,
	}
	return response, nil
}
