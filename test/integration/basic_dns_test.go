// +build basic

package nsmd_integration_tests

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	. "github.com/onsi/gomega"
)

func TestDNS(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}
	coreFile := `.:53 {
    log
    hosts {
        172.16.1.1 my.google.com
        fallthrough
    }
}`
	k8s, err := kubetest.NewK8s(true)
	Expect(err).Should(BeNil())
	defer k8s.Cleanup()

	_, err = k8s.CreateConfigMap(&v1.ConfigMap{
		TypeMeta : metav1.TypeMeta{
			Kind:"ConfigMap",
			APIVersion:"v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:"coredns",
			Namespace:k8s.GetK8sNamespace(),
		},

		BinaryData: map[string][]byte{
			"Corefile": []byte(coreFile),
		},
	})

	Expect(err).To(BeNil())

	nodes, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())
	defer kubetest.FailLogger(k8s, nodes, t)

	kubetest.DeployICMP(k8s, nodes[0].Node, "icmp-responder", defaultTimeout)
	nscAndDns := kubetest.DeployNSCDns(k8s, nodes[0].Node, "nsc", "coredns" , defaultTimeout)
	Expect(true, kubetest.IsNsePinged(k8s, nscAndDns))
}
