// +build basic

package nsmd_integration_tests

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMgrRestartRestoreNSE(t *testing.T) {
	RegisterTestingT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(true)
	Expect(err).To(BeNil())
	defer k8s.Cleanup()
	defer kubetest.ShowLogs(k8s, t)

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	defer kubetest.ShowLogs(k8s, t)

	nsMgrName := nodesConf[0].Nsmd.GetName()

	nse1, name1, err := deployNSEPod(k8s, nodesConf[0].Node, "icmp-responder-nse-1")
	Expect(err).To(BeNil())
	logrus.Infof("NSE created: %v", name1)

	_, name2, err := deployNSEPod(k8s, nodesConf[0].Node, "icmp-responder-nse-2")
	Expect(err).To(BeNil())
	logrus.Infof("NSE created: %v", name2)

	_, name3, err := deployNSEPod(k8s, nodesConf[0].Node, "icmp-responder-nse-3")
	Expect(err).To(BeNil())
	logrus.Infof("NSE created: %v", name3)

	k8s.DeletePods(nodesConf[0].Nsmd)

	err = k8s.DeleteNetworkServices("icmp-responder")
	Expect(err).To(BeNil())

	err = k8s.DeleteNSEs(name1, name2)
	Expect(err).To(BeNil())
	logrus.Infof("NSEs deleted: %v, %v", name1, name2)

	k8s.DeletePods(nse1)

	nsMgrPod := k8s.CreatePod(pods.NSMgrPodWithConfig(nsMgrName, nodesConf[0].Node, &pods.NSMgrPodConfig{
		Variables: map[string]string{
			nsmd.NsmdDeleteLocalRegistry: "false",
		},
		Namespace: k8s.GetK8sNamespace(),
	}))
	Expect(nsMgrPod).NotTo(BeNil())

	_ = k8s.WaitLogsContainsRegex(nsMgrPod, "nsmd", "NSM gRPC API Server: .* is operational", defaultTimeout)

	networkServices, err := k8s.GetNetworkServices()
	Expect(err).To(BeNil())
	Expect(networkServices).To(HaveLen(1))
	Expect(networkServices[0].GetName()).To(Equal("icmp-responder"))

	nses, err := k8s.GetNSEs()
	Expect(err).To(BeNil())
	Expect(nses).To(HaveLen(2))
	Expect(nses[0].GetName()).To(Or(Equal(name2), Equal(name3)))
	Expect(nses[1].GetName()).To(Or(Equal(name2), Equal(name3)))
}

func deployNSEPod(k8s *kubetest.K8s, node *v1.Node, name string) (*v1.Pod, string, error) {
	nsesBefore, err := k8s.GetNSEs()
	if err != nil {
		return nil, "", err
	}

	namesBefore := map[string]bool{}
	for i := range nsesBefore {
		namesBefore[nsesBefore[i].GetName()] = true
	}

	pod := kubetest.DeployICMP(k8s, node, name, defaultTimeout)
	Expect(pod.GetName()).To(Equal(name))

	kubetest.ExpectNSEsCountToBe(k8s, len(nsesBefore), len(nsesBefore)+1)

	nsesAfter, err := k8s.GetNSEs()
	if err != nil {
		return nil, "", err
	}

	for i := range nsesAfter {
		if name := nsesAfter[i].GetName(); !namesBefore[name] {
			return pod, name, nil
		}
	}

	return nil, "", fmt.Errorf("no name found")
}
