// +build suite basic

package nsmd_integration_tests

import (
	"testing"

	"github.com/pkg/errors"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestNSMgrRestartRestoreNSE(t *testing.T) {
	g := NewWithT(t)

	if testing.Short() {
		t.Skip("Skip, please run without -short")
		return
	}

	k8s, err := kubetest.NewK8s(g, kubetest.ReuseNSMResouces)
	g.Expect(err).To(BeNil())
	defer k8s.Cleanup()
	defer k8s.ProcessArtifacts(t)

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	g.Expect(err).To(BeNil())

	nsMgrName := nodesConf[0].Nsmd.GetName()

	nse1, name1, err := deployNSEPod(g, k8s, nodesConf[0].Node, "icmp-responder-nse-1")
	g.Expect(err).To(BeNil())
	logrus.Infof("NSE created: %v", name1)

	_, name2, err := deployNSEPod(g, k8s, nodesConf[0].Node, "icmp-responder-nse-2")
	g.Expect(err).To(BeNil())
	logrus.Infof("NSE created: %v", name2)

	_, name3, err := deployNSEPod(g, k8s, nodesConf[0].Node, "icmp-responder-nse-3")
	g.Expect(err).To(BeNil())
	logrus.Infof("NSE created: %v", name3)

	k8s.DeletePods(nodesConf[0].Nsmd)

	err = k8s.DeleteNetworkServices("icmp-responder")
	g.Expect(err).To(BeNil())

	err = k8s.DeleteNSEs(name1, name2)
	g.Expect(err).To(BeNil())
	logrus.Infof("NSEs deleted: %v, %v", name1, name2)

	k8s.DeletePods(nse1)

	nsMgrPod := k8s.CreatePod(pods.NSMgrPodWithConfig(nsMgrName, nodesConf[0].Node, &pods.NSMgrPodConfig{
		Variables: map[string]string{
			nsmd.NsmdDeleteLocalRegistry: "false",
		},
		Namespace: k8s.GetK8sNamespace(),
	}))
	g.Expect(nsMgrPod).NotTo(BeNil())

	// Wait for NSMgr to be deployed, to not get admission error
	kubetest.WaitNSMgrDeployed(k8s, nsMgrPod, defaultTimeout)

	networkServices, err := k8s.GetNetworkServices()
	g.Expect(err).To(BeNil())
	g.Expect(networkServices).To(HaveLen(1))
	g.Expect(networkServices[0].GetName()).To(Equal("icmp-responder"))

	nses, err := k8s.GetNSEs()
	g.Expect(err).To(BeNil())
	g.Expect(nses).To(HaveLen(2))
	g.Expect(nses[0].GetName()).To(Or(Equal(name2), Equal(name3)))
	g.Expect(nses[1].GetName()).To(Or(Equal(name2), Equal(name3)))
}

func deployNSEPod(g *WithT, k8s *kubetest.K8s, node *v1.Node, name string) (*v1.Pod, string, error) {
	nsesBefore, err := k8s.GetNSEs()
	if err != nil {
		return nil, "", err
	}

	namesBefore := map[string]bool{}
	for i := range nsesBefore {
		namesBefore[nsesBefore[i].GetName()] = true
	}

	pod := kubetest.DeployICMP(k8s, node, name, defaultTimeout)
	g.Expect(pod.GetName()).To(Equal(name))

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

	return nil, "", errors.New("no name found")
}
