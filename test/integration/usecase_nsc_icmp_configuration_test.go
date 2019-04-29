// +build usecase

package nsmd_integration_tests

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kube_testing/pods"

	"github.com/networkservicemesh/networkservicemesh/test/integration/nsmd_test_utils"
	"github.com/networkservicemesh/networkservicemesh/test/kube_testing"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestNSCAndICMPLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, false, false)
}

func TestNSCAndICMPRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, false)
}

func TestNSCAndICMPWebhookLocal(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, true, false)
}

func TestNSCAndICMPWebhookRemote(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, true, false)
}

func TestNSCAndICMPLocalVeth(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 1, false, true)
}

func TestNSCAndICMPRemoteVeth(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	testNSCAndICMP(t, 2, false, true)
}

/**
If passed 1 both will be on same node, if not on different.
*/
func testNSCAndICMP(t *testing.T, nodesCount int, useWebhook bool, disableVHost bool) {
	k8s, err := kube_testing.NewK8s()
	defer k8s.Cleanup()

	Expect(err).To(BeNil())

	s1 := time.Now()
	k8s.PrepareDefault()
	logrus.Printf("Cleanup done: %v", time.Since(s1))

	if useWebhook {
		awc, awDeployment, awService := nsmd_test_utils.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace())
		defer nsmd_test_utils.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
	}

	config := []*pods.NSMgrPodConfig{}
	for i := 0; i < nodesCount; i++ {
		cfg := &pods.NSMgrPodConfig{}
		cfg.Namespace = k8s.GetK8sNamespace()
		if disableVHost {
			cfg.DataplaneVariables = map[string]string{}
			cfg.DataplaneVariables["DATAPLANE_ALLOW_VHOST"] = "false"
		}
		config = append(config, cfg)
	}
	nodes_setup := nsmd_test_utils.SetupNodesConfig(k8s, nodesCount, defaultTimeout, config, k8s.GetK8sNamespace())

	// Run ICMP on latest node
	_ = nsmd_test_utils.DeployICMP(k8s, nodes_setup[nodesCount-1].Node, "icmp-responder-nse-1", defaultTimeout)

	var nscPodNode *v1.Pod
	if useWebhook {
		nscPodNode = nsmd_test_utils.DeployNSCWebhook(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	} else {
		nscPodNode = nsmd_test_utils.DeployNSC(k8s, nodes_setup[0].Node, "nsc-1", defaultTimeout)
	}
	var nscInfo *nsmd_test_utils.NSCCheckInfo

	failures := InterceptGomegaFailures(func() {
		nscInfo = nsmd_test_utils.CheckNSC(k8s, t, nscPodNode)
	})
	// Do dumping of container state to dig into what is happened.
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		nsmd_test_utils.PrintLogs(k8s, nodes_setup)
		nscInfo.PrintLogs()

		t.Fail()
	}
}
