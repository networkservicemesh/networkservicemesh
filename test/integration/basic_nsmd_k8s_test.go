// +build basic

package nsmd_integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	nsmd2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMDDRegistryNSE(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	defer kubetest.MakeLogsSnapshot(k8s, t)
	g.Expect(err).To(BeNil())

	nsmd := k8s.CreatePod(pods.NSMgrPod("nsmgr-1", nil, k8s.GetK8sNamespace()))

	k8s.WaitLogsContains(nsmd, "nsmd", "NSMD: Restore of NSE/Clients Complete...", defaultTimeout)

	fwd, err := k8s.NewPortForwarder(nsmd, 5000)
	g.Expect(err).To(BeNil())
	defer fwd.Stop()

	// We need to wait unti it is started

	k8s.WaitLogsContains(nsmd, "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", fastTimeout)
	logs, err := k8s.GetLogs(nsmd, "nsmd-k8s")
	logrus.Printf("%s", logs)

	e := fwd.Start()
	if e != nil {
		logrus.Printf("Error on forward: %v retrying", e)
	}

	serviceRegistry := nsmd2.NewServiceRegistryAt(fmt.Sprintf("localhost:%d", fwd.ListenPort))

	discovery, err := serviceRegistry.DiscoveryClient()
	g.Expect(err).To(BeNil())
	req := &registry.FindNetworkServiceRequest{
		NetworkServiceName: "my_service",
	}
	response, err := discovery.FindNetworkService(context.Background(), req)
	g.Expect(response).To(BeNil())
	logrus.Print(err.Error())

	g.Expect(err.Error()).To(Equal("rpc error: code = Unknown desc = no NetworkService with name: my_service"))

	// Lets register few hundred NSEs and check how it works.

	registryClient, err := serviceRegistry.NseRegistryClient()
	g.Expect(err).To(BeNil())

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
	}
	resp, err := registryClient.RegisterNSE(context.Background(), reg)
	g.Expect(err).To(BeNil())
	logrus.Printf("Register time: time: %v %v", time.Since(t1), resp)

	t2 := time.Now()
	response, err = discovery.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{
		NetworkServiceName: nme,
	})
	g.Expect(err).To(BeNil())
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
		g.Expect(err).To(BeNil())
		if err != nil {
			logrus.Printf("Bad")
		}
		g.Expect(err2).To(BeNil())
	}

	logs, err = k8s.GetLogs(nsmd, "nsmd-k8s")
	logrus.Printf("%s", logs)

}

func TestUpdateNSM(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nsmd := k8s.CreatePod(pods.NSMgrPod("nsmgr-1", nil, k8s.GetK8sNamespace()))

	fwd, err := k8s.NewPortForwarder(nsmd, 5000)
	g.Expect(err).To(BeNil())
	defer fwd.Stop()

	// We need to wait until it is started
	k8s.WaitLogsContains(nsmd, "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", fastTimeout)
	// To be sure NSMD is already called for register.
	k8s.WaitLogsContains(nsmd, "nsmd", "Waiting for dataplane available...", defaultTimeout)

	e := fwd.Start()
	if e != nil {
		logrus.Printf("Error on forward: %v retrying", e)
	}

	serviceRegistry := nsmd2.NewServiceRegistryAt(fmt.Sprintf("localhost:%d", fwd.ListenPort))

	discovery, err := serviceRegistry.DiscoveryClient()
	g.Expect(err).To(BeNil())

	nseRegistryClient, err := serviceRegistry.NseRegistryClient()
	g.Expect(err).To(BeNil())

	nsmRegistryClient, err := serviceRegistry.NsmRegistryClient()
	g.Expect(err).To(BeNil())

	networkService := "icmp-responder"
	nsmName := "master"
	url1 := "1.1.1.1:1"
	url2 := "2.2.2.2:2"

	response, err := nsmRegistryClient.RegisterNSM(context.Background(), &registry.NetworkServiceManager{
		Name: nsmName,
		Url:  url1,
	})
	logrus.Info(response)
	g.Expect(err).To(BeNil())

	nseResp, err := nseRegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    networkService,
			Payload: "tcp",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: networkService,
		},
	})
	g.Expect(err).To(BeNil())
	logrus.Info(nseResp)
	g.Expect(getNsmUrl(g, discovery)).To(Equal(url1))

	updNsm, err := nsmRegistryClient.RegisterNSM(context.Background(), &registry.NetworkServiceManager{
		Name: nsmName,
		Url:  url2,
	})
	g.Expect(err).To(BeNil())
	g.Expect(updNsm.GetUrl()).To(Equal(url2))
	g.Expect(getNsmUrl(g, discovery)).To(Equal(url2))
}

func TestGetEndpoints(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nsmd, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	k8s.WaitLogsContains(nsmd[0].Nsmd, "nsmd", "NSMD: Restore of NSE/Clients Complete...", defaultTimeout)

	nseRegistryClient, nsmRegistryClient, fwdClose := kubetest.PrepareRegistryClients(k8s, nsmd[0].Nsmd)
	defer fwdClose()

	url := "1.1.1.1:1"

	responseNsm, err := nsmRegistryClient.RegisterNSM(context.Background(), &registry.NetworkServiceManager{
		Url: url,
	})
	g.Expect(err).To(BeNil())
	g.Expect(responseNsm.Url).To(Equal(url))
	letters := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}

	for i := 0; i < len(letters); i++ {
		nsName := "icmp-responder-" + letters[i]
		_, err := nseRegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
			NetworkService: &registry.NetworkService{
				Name:    nsName,
				Payload: "IP",
			},
			NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
				NetworkServiceName: nsName,
			},
		})
		g.Expect(err).To(BeNil())
	}

	nseList, err := nsmRegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	g.Expect(err).To(BeNil())
	g.Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(len(letters)))

	_, err = nseRegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Payload: "IP",
			Name:    "icmp-responder",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			EndpointName: nseList.NetworkServiceEndpoints[0].EndpointName,
		},
	})
	g.Expect(err).NotTo(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("already exists"))
}

func TestDuplicateEndpoint(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nsmd, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	// We need to wait unti it is started
	k8s.WaitLogsContains(nsmd[0].Nsmd, "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", defaultTimeout)

	k8s.WaitLogsContains(nsmd[0].Nsmd, "nsmd", "NSMD: Restore of NSE/Clients Complete...", defaultTimeout)

	nseRegistryClient, nsmRegistryClient, fwdClose := kubetest.PrepareRegistryClients(k8s, nsmd[0].Nsmd)
	defer fwdClose()

	url := "1.1.1.1:1"

	responseNsm, err := nsmRegistryClient.RegisterNSM(context.Background(), &registry.NetworkServiceManager{
		Url: url,
	})
	g.Expect(err).To(BeNil())
	g.Expect(responseNsm.Url).To(Equal(url))

	nsName := "icmp-responder"
	_, err = nseRegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Payload: "IP",
			Name:    "icmp-responder",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			EndpointName: nsName,
		},
	})
	g.Expect(err).To(BeNil())

	nseList, err := nsmRegistryClient.GetEndpoints(context.Background(), &empty.Empty{})
	g.Expect(err).To(BeNil())
	g.Expect(len(nseList.NetworkServiceEndpoints)).To(Equal(1))

	_, err = nseRegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Payload: "IP",
			Name:    "icmp-responder",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			EndpointName: nsName,
		},
	})
	g.Expect(err).NotTo(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("already exists"))
}

func getNsmUrl(g *WithT, discovery registry.NetworkServiceDiscoveryClient) string {
	logrus.Infof("Finding NSE...")
	response, err := discovery.FindNetworkService(context.Background(), &registry.FindNetworkServiceRequest{
		NetworkServiceName: "icmp-responder",
	})
	g.Expect(err).To(BeNil())
	g.Expect(len(response.GetNetworkServiceEndpoints()) >= 1).To(Equal(true))

	endpoint := response.GetNetworkServiceEndpoints()[0]
	logrus.Infof("Endpoint: %v", endpoint)
	return response.NetworkServiceManagers[endpoint.NetworkServiceManagerName].GetUrl()
}

func createSingleNsmgr(k8s *kubetest.K8s, name string) *v1.Pod {
	nsmgr := k8s.CreatePod(pods.NSMgrPod(name, nil, k8s.GetK8sNamespace()))

	// We need to wait until it is started
	k8s.WaitLogsContains(nsmgr, "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", fastTimeout)
	// To be sure NSMD is already called for register.
	k8s.WaitLogsContains(nsmgr, "nsmd", "Waiting for dataplane available...", defaultTimeout)

	return nsmgr
}

func TestLostUpdate(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nsmgr1 := createSingleNsmgr(k8s, "nsmgr-1")

	sr1, closeFunc := kubetest.ServiceRegistryAt(k8s, nsmgr1)

	nsmRegistryClient, err := sr1.NsmRegistryClient()
	g.Expect(err).To(BeNil())

	discovery, err := sr1.DiscoveryClient()
	g.Expect(err).To(BeNil())

	nseRegistryClient, err := sr1.NseRegistryClient()
	g.Expect(err).To(BeNil())

	networkService := "icmp-responder"
	nsmName := "master"
	url1 := "1.1.1.1:1"
	response, err := nsmRegistryClient.RegisterNSM(context.Background(), &registry.NetworkServiceManager{
		Name: nsmName,
		Url:  url1,
	})
	logrus.Info(response)
	g.Expect(err).To(BeNil())

	nseResp, err := nseRegistryClient.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    networkService,
			Payload: "tcp",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: networkService,
		},
	})
	g.Expect(err).To(BeNil())
	logrus.Info(nseResp)
	g.Expect(getNsmUrl(g, discovery)).To(Equal(url1))
	closeFunc()
	k8s.DeletePods(nsmgr1)

	nsmgr2 := createSingleNsmgr(k8s, "nsmgr-2")

	sr2, closeFunc2 := kubetest.ServiceRegistryAt(k8s, nsmgr2)
	defer closeFunc2()

	discovery2, err := sr2.DiscoveryClient()
	g.Expect(err).To(BeNil())

	nseRegistryClient2, err := sr2.NseRegistryClient()
	g.Expect(err).To(BeNil())

	nseResp, err = nseRegistryClient2.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    networkService,
			Payload: "tcp",
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceName: networkService,
		},
	})
	g.Expect(err).To(BeNil())
	logrus.Info(nseResp)
	g.Expect(getNsmUrl(g, discovery2)).ToNot(Equal(url1))
}

func TestRegistryConcurrentModification(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nsmgr1 := createSingleNsmgr(k8s, "nsmgr-1")

	sr1, closeFunc := kubetest.ServiceRegistryAt(k8s, nsmgr1)
	defer closeFunc()

	nsmRegistryClient, err := sr1.NsmRegistryClient()
	g.Expect(err).To(BeNil())

	n := 30
	errorCh := make(chan error)

	// concurrent modification of k8s registry
	go func() {
		for i := 0; i < n; i++ {
			url := fmt.Sprintf("1.1.1.%d:1", i)
			_, err := nsmRegistryClient.RegisterNSM(context.Background(), &registry.NetworkServiceManager{
				Url: url,
			})
			if err != nil {
				errorCh <- err
				break
			}
			logrus.Infof("RegisterNSM(%v)", url)
			<-time.After(100 * time.Millisecond)
		}
		close(errorCh)
	}()

	for {
		select {
		case <-time.Tick(10 * time.Millisecond):
			k8s.CleanupCRDs()
			logrus.Info("CleanupCRDs")
		case err := <-errorCh:
			g.Expect(err).To(BeNil())
			return
		}
	}
}
