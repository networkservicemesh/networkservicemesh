// +build webhook

package nsmd_integration_tests

import (
	"strings"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"

	"testing"

	. "github.com/onsi/gomega"
)

func TestDeployWrongNsc(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()

	g.Expect(err).To(BeNil())

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())
	defer kubetest.MakeLogsSnapshot(k8s, t)

	awc, awDeployment, awService := kubetest.DeployAdmissionWebhook(k8s, "nsm-admission-webhook", "networkservicemesh/admission-webhook", k8s.GetK8sNamespace(), defaultTimeout)
	defer kubetest.DeleteAdmissionWebhook(k8s, "nsm-admission-webhook-certs", awc, awDeployment, awService, k8s.GetK8sNamespace())
	_, err = k8s.CreatePodsRaw(defaultTimeout, false, pods.WrongNSCPodWebhook("wrong-nsc-client-pod", nodes[0].Node))
	g.Expect(strings.Contains(err.Error(), "do not use init-container and nsm annotation")).Should(BeTrue())
}
