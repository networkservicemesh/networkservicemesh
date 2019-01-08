package nsmd_integration_tests

import (
	"context"
	"fmt"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	nsmd2 "github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/ligato/networkservicemesh/test/kube_testing"
	"github.com/ligato/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestNSMDDRegistryNSE(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	k8s.Prepare("nsmd")

	nsmd := k8s.CreatePod(pods.NSMDPod("nsmd1", nil))

	fwd, err := k8s.NewPortForwarder(nsmd, 5000)
	Expect(err).To(BeNil())
	defer fwd.Stop()

	// We need to wait unti it is started

	k8s.WaitLogsContains(nsmd, "nsmd-k8s", "Registering networkservicemanagers.networkservicemesh.io", 10*time.Second)
	logs, err := k8s.GetLogs(nsmd, "nsmd-k8s")
	logrus.Printf("%s", logs)

	e := fwd.Start()
	if e != nil {
		logrus.Printf("Error on forward: %v retrying", e)
	}

	serviceRegistry := nsmd2.NewServiceRegistryAt(fmt.Sprintf("localhost:%d", fwd.ListenPort))

	discovery, err := serviceRegistry.NetworkServiceDiscovery()
	Expect(err).To(BeNil())
	req := &registry.FindNetworkServiceRequest{
		NetworkServiceName: "my_service",
	}
	response, err := discovery.FindNetworkService(context.Background(), req)
	Expect(response).To(BeNil())
	logrus.Print(err.Error())

	Expect(err.Error()).To(Equal("rpc error: code = Unknown desc = networkservices.networkservicemesh.io \"my_service\" not found"))

	// Lets register few hundred NSEs and check how it works.

	registryClient, err := serviceRegistry.RegistryClient()
	Expect(err).To(BeNil())

	versionedClientSet, err := versioned.NewForConfig(k8s.GetConfig())
	Expect(err).To(BeNil())

	// Cleanup all registered stuff
	cleanupCRDs(versionedClientSet)

	nses := []string{}
	nme := "my-network-service"

	t1 := time.Now()
	//nme := fmt.Sprintf("my-network-service-%d-%d", i, t1.Second())

	reg := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    nme,
			Payload: "tcp",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: nme,
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Url: fmt.Sprintf("%s:%d", nsmd.Status.PodIP, 5001),
		},
	}
	resp, err := registryClient.RegisterNSE(context.Background(), reg)
	Expect(err).To(BeNil())
	logrus.Printf("Register time: time: %v %v", time.Since(t1), resp)

	t2 := time.Now()
	response, err = discovery.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{
		NetworkServiceName: nme,
	})
	Expect(err).To(BeNil())
	logrus.Printf("Find time: time: %v %d", time.Since(t2), len(response.NetworkServiceEndpoints))

	for _, lnse := range response.NetworkServiceEndpoints {
		nses = append(nses, lnse.EndpointName)
	}

	logs, err = k8s.GetLogs(nsmd, "nsmd-k8s")
	logrus.Printf("%s", logs)
	// Remove all added NSEs

	for _, lnse := range nses {
		t2 := time.Now()
		removeNSE := &registry.RemoveNSERequest{
			EndpointName: lnse,
		}
		_, err = registryClient.RemoveNSE(context.Background(), removeNSE)
		response, err2 := discovery.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{
			NetworkServiceName: nme,
		})
		logrus.Printf("Find time: time: %v %d", time.Since(t2), len(response.NetworkServiceEndpoints))
		Expect(err).To(BeNil())
		if err != nil {
			logrus.Printf("Bad")
		}
		Expect(err2).To(BeNil())
	}

	logs, err = k8s.GetLogs(nsmd, "nsmd-k8s")
	logrus.Printf("%s", logs)

}

func cleanupCRDs(versionedClientSet *versioned.Clientset) {
	managers, err := versionedClientSet.Networkservicemesh().NetworkServiceManagers("default").List(v1.ListOptions{})
	Expect(err).To(BeNil())
	for _, mgr := range managers.Items {
		_ = versionedClientSet.Networkservicemesh().NetworkServiceManagers("default").Delete(mgr.Name, &v1.DeleteOptions{})
	}
	endpoints, err := versionedClientSet.Networkservicemesh().NetworkServiceEndpoints("default").List(v1.ListOptions{})
	Expect(err).To(BeNil())
	for _, ep := range endpoints.Items {
		_ = versionedClientSet.Networkservicemesh().NetworkServiceEndpoints("default").Delete(ep.Name, &v1.DeleteOptions{})
	}
	services, err := versionedClientSet.Networkservicemesh().NetworkServices("default").List(v1.ListOptions{})
	Expect(err).To(BeNil())
	for _, service := range services.Items {
		_ = versionedClientSet.Networkservicemesh().NetworkServices("default").Delete(service.Name, &v1.DeleteOptions{})
	}
}
