// +build basic

package integration

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
)

func Test_createNSMNamespace(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	defer k8s.SaveTestArtifacts(t)

	namespaceName := k8s.GetK8sNamespace()
	namespace, err := k8s.GetNamespace(namespaceName)

	g.Expect(err).To(BeNil())
	g.Expect(namespace.Status.Phase).To(Equal(v1.NamespaceActive))

}
