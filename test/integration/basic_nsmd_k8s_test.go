// +build basic

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	nsmd2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
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

	k8s.WaitLogsContains(nsmd, "nsmd-k8s", "nsmd-k8s intialized and waiting for connection", fastTimeout)
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

	Expect(err.Error()).To(Equal("rpc error: code = Unknown desc = no NetworkService with name: my_service"))

	// Lets register few hundred NSEs and check how it works.

	registryClient, err := serviceRegistry.RegistryClient()
	Expect(err).To(BeNil())

	// Cleanup all registered stuff
	k8s.CleanupCRDs()

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

func TestUpdateNsm(t *testing.T) {
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
	k8s.WaitLogsContains(nsmd, "nsmd-k8s", "nsmd-k8s intialized and waiting for connection", fastTimeout)

	e := fwd.Start()
	if e != nil {
		logrus.Printf("Error on forward: %v retrying", e)
	}

	serviceRegistry := nsmd2.NewServiceRegistryAt(fmt.Sprintf("localhost:%d", fwd.ListenPort))

	discovery, err := serviceRegistry.NetworkServiceDiscovery()
	Expect(err).To(BeNil())
	networkService := "icmp-responder"

	//req := &registry.FindNetworkServiceRequest{
	//	NetworkServiceName: networkService,
	//}

	time.Sleep(4 * time.Second)
	//response, err := discovery.FindNetworkService(context.Background(), req)
	//Expect(response).To(BeNil())
	//logrus.Print(err.Error())

	registryClient, err := serviceRegistry.RegistryClient()
	Expect(err).To(BeNil())

	k8s.CleanupCRDs()

	nsmName := "master"
	url1 := "1.1.1.1:1"

	reg := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    networkService,
			Payload: "tcp",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: networkService,
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Url:  url1,
			Name: nsmName,
		},
	}

	failures := InterceptGomegaFailures(func() {
		_, err = registryClient.RegisterNSE(context.Background(), reg)
		Expect(err).To(BeNil())
		//Expect(getNsmUrl(discovery)).To(Equal(url1))

		url2 := "2.2.2.2:2"

		updNsm, err := registryClient.UpdateNSM(context.Background(), &registry.NetworkServiceManager{
			Name: nsmName,
			Url:  url2,
		})
		Expect(err).To(BeNil())
		Expect(updNsm.GetUrl()).To(Equal(url2))
		Expect(getNsmUrl(discovery)).To(Equal(url2))
	})

	if len(failures) > 0 {
		logrus.Errorf("Failues: %v", failures)

		nsmdLogs, _ := k8s.GetLogs(nsmd, "nsmd")
		logrus.Errorf("===================== NSMD output since test is failing %v\n=====================", nsmdLogs)

		nsmdk8sLogs, _ := k8s.GetLogs(nsmd, "nsmd-k8s")
		logrus.Errorf("===================== NSMD K8S output since test is failing %v\n=====================", nsmdk8sLogs)

		nsmdpLogs, _ := k8s.GetLogs(nsmd, "nsmdp")
		logrus.Errorf("===================== NSMD K8S output since test is failing %v\n=====================", nsmdpLogs)

		t.Fail()
	}
}

func getNsmUrl(discovery registry.NetworkServiceDiscoveryClient) string {
	logrus.Infof("Finding NSE...")
	response, err := discovery.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{
		NetworkServiceName: "icmp-responder",
	})
	Expect(err).To(BeNil())
	Expect(len(response.NetworkServiceEndpoints)).To(Equal(1))

	endpoint := response.NetworkServiceEndpoints[0]
	logrus.Infof("Endpoint: %v", endpoint)
	return response.NetworkServiceManagers[endpoint.NetworkServiceManagerName].GetUrl()
}
