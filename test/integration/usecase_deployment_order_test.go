// +build usecase

package nsmd_integration_tests

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	nsapiv1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/crds"
)

type Deployment int

const (
	DeployService Deployment = iota
	DeployEndpoint
	DeployClient
	DeployClientWebhook
)

func TestDeploymentOrder2EndpointClient(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployEndpoint, DeployEndpoint,
		DeployClient})
}

func TestDeploymentOrder2EndpointClientWebhook(t *testing.T) {
	if !kubetest.IsBrokeTestsEnabled() {
		t.Skip("Skip, issue - https://github.com/networkservicemesh/networkservicemesh/issues/1372")
		return
	}
	testDeploymentOrder(t, []Deployment{
		DeployEndpoint, DeployEndpoint,
		DeployClientWebhook})
}

func TestDeploymentOrder2EndpointClientAndWebhook(t *testing.T) {
	if !kubetest.IsBrokeTestsEnabled() {
		t.Skip("Skip, issue - https://github.com/networkservicemesh/networkservicemesh/issues/1372")
		return
	}
	testDeploymentOrder(t, []Deployment{
		DeployEndpoint, DeployEndpoint,
		DeployClient,
		DeployClientWebhook})
}

func TestDeploymentOrder4EndpointClientAndWebhook(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployEndpoint, DeployEndpoint, DeployEndpoint, DeployEndpoint,
		DeployClient,
		DeployClientWebhook})
}

func TestDeploymentOrderClientEndpoint(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployClient,
		DeployEndpoint})
}

func TestDeploymentOrder2ClientEndpoint(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployClient, DeployClient,
		DeployEndpoint})
}

func TestDeploymentOrder2Client2Endpoint(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployClient, DeployClient,
		DeployEndpoint, DeployEndpoint})
}

func TestDeploymentOrder4ClientEndpoint(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployClient, DeployClient, DeployClient, DeployClient,
		DeployEndpoint})
}

func TestDeploymentOrder4Client2Endpoint(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployClient, DeployClient, DeployClient, DeployClient,
		DeployEndpoint, DeployEndpoint})
}

func TestDeploymentOrderServiceEndpointClient(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployService,
		DeployEndpoint,
		DeployClient})
}

func TestDeploymentOrderService2Endpoint4Client(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployService,
		DeployEndpoint, DeployEndpoint,
		DeployClient, DeployClient, DeployClient, DeployClient})
}

func TestDeploymentOrderServiceClientEndpoint(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployService,
		DeployClient,
		DeployEndpoint})
}

func TestDeploymentOrderService2ClientEndpoint(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployService,
		DeployClient, DeployClient,
		DeployEndpoint})
}

func TestDeploymentOrderServiceClientEndpointClient(t *testing.T) {
	testDeploymentOrder(t, []Deployment{
		DeployService,
		DeployClient,
		DeployEndpoint,
		DeployClient})
}

func TestDeploymentOrderServiceClientWebhookEndpoint(t *testing.T) {
	if !kubetest.IsBrokeTestsEnabled() {
		t.Skip("Skip, issue - https://github.com/networkservicemesh/networkservicemesh/issues/1372")
		return
	}
	testDeploymentOrder(t, []Deployment{
		DeployService,
		DeployClientWebhook,
		DeployEndpoint})
}

func TestDeploymentOrderService4ClientWebhook2Endpoint(t *testing.T) {
	if !kubetest.IsBrokeTestsEnabled() {
		t.Skip("Skip, issue - https://github.com/networkservicemesh/networkservicemesh/issues/1372")
		return
	}
	testDeploymentOrder(t, []Deployment{
		DeployService,
		DeployClientWebhook, DeployClientWebhook, DeployClientWebhook, DeployClientWebhook,
		DeployEndpoint, DeployEndpoint})
}

func testDeploymentOrder(t *testing.T, order []Deployment) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())

	for _, deploy := range order {
		if deploy == DeployClientWebhook {
			awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace(), defaultTimeout)
			defer kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
			break
		}
	}

	_, err = kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	var nseCount uint64
	var nscPods []*v1.Pod
	var nscCount uint64
	var waitgroup sync.WaitGroup
	var scheduled = make(chan interface{})

	for _, deploy := range order {
		waitgroup.Add(1)

		go func() {
			scheduled <- true
			defer func() { waitgroup.Done() }()
			switch deploy {
			case DeployService:
				nscrd, err := crds.NewNSCRD(k8s.GetK8sNamespace())
				g.Expect(err).To(BeNil())
				nsIcmpResponder := crds.IcmpResponder(map[string]string{}, map[string]string{"app": "icmp"})
				logrus.Printf("About to insert: %v", nsIcmpResponder)
				var result *nsapiv1.NetworkService
				result, err = nscrd.Create(nsIcmpResponder)
				g.Expect(err).To(BeNil())
				logrus.Printf("CRD applied with result: %v", result)
				result, err = nscrd.Get(nsIcmpResponder.ObjectMeta.Name)
				g.Expect(err).To(BeNil())
				logrus.Printf("Registered CRD is: %v", result)
			case DeployEndpoint:
				kubetest.DeployICMP(k8s, nil, "nse-"+strconv.FormatUint(atomic.AddUint64(&nseCount, 1), 10), defaultTimeout)
			case DeployClient:
				nscPods = append(nscPods, kubetest.DeployNSC(k8s, nil, "nsc-"+strconv.FormatUint(atomic.AddUint64(&nscCount, 1), 10), defaultTimeout))
			case DeployClientWebhook:
				nscPods = append(nscPods, kubetest.DeployNSCWebhook(k8s, nil, "nsc-webhook-"+strconv.FormatUint(atomic.AddUint64(&nscCount, 1), 10), defaultTimeout))
			}
		}()
		// ensure the deplyment routine is scheduled
		<-scheduled
	}

	// wait for all deployment routines to end
	waitgroup.Wait()

	for _, p := range nscPods {
		kubetest.CheckNSC(k8s, p)
	}
}
