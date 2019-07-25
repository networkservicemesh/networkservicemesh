// +build checkfix

package nsmd_integration_tests

import (
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	"github.com/onsi/gomega"
	"strings"
	"testing"
)

//TODO: remove this if test passed
func TestFixTestDeployWrongNsc(t *testing.T) {
	for i := 0; i < 25; i++ {
		TestDeployWrongNsc(t)
	}
}

func TestDeployWrongNsc(t *testing.T) {
	gomega.RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()

	gomega.Expect(err).To(gomega.BeNil())

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	gomega.Expect(err).To(gomega.BeNil())
	defer kubetest.ShowLogs(k8s, t)

	awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace())
	defer kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
	admissionWebhookPod := kubetest.WaitPod(k8s, awDeployment.Name, defaultTimeout)
	gomega.Expect(admissionWebhookPod).ShouldNot(gomega.BeNil())
	k8s.WaitLogsContains(admissionWebhookPod, admissionWebhookPod.Spec.Containers[0].Name, "Server started", defaultTimeout)
	_, err = k8s.CreatePodsRaw(defaultTimeout, false, pods.WrongNSCPodWebhook("wrong-nsc-client-pod", nodes[0].Node))
	gomega.Expect(strings.Contains(err.Error(), "do not use init-container and nsm annotation")).Should(gomega.BeTrue())
}
