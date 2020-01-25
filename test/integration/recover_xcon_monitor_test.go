// +build recover

package integration

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

func TestXconMonitorSingleNodeHealFailed(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
				"NSMD_HEAL_RETRY_COUNT":      "2",
				"NSMD_HEAL_DST_TIMEOUTs":     "1",
			},
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	defer kubetest.MakeLogsSnapshot(k8s, t)

	icmpPod := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	g.Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	g.Expect(nscPodNode).ToNot(BeNil())

	eventCh, closeFunc := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc()

	expectedFunc, waitFunc := kubetest.NewEventChecker(t, eventCh)
	k8s.DeletePods(icmpPod)

	expectedFunc(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
	})

	expectedFunc(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		Timeout:   defaultTimeout, // waiting while heal finishes work
	})

	waitFunc()
}

func TestXconMonitorSingleNodeHealSuccess(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
				"NSMD_HEAL_RETRY_COUNT":      "2",
				"NSMD_HEAL_DST_TIMEOUTs":     "1",
			},
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	defer kubetest.MakeLogsSnapshot(k8s, t)

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	g.Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	g.Expect(nsc).ToNot(BeNil())

	eventCh, closeFunc := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc()

	expectedFunc, waitFunc := kubetest.NewEventChecker(t, eventCh)

	icmp1 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-1", defaultTimeout)
	g.Expect(icmp1).ToNot(BeNil())

	k8s.DeletePods(icmp0)

	expectedFunc(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
	})

	expectedFunc(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     true,
		Timeout:   defaultTimeout,
	})

	waitFunc()
}

func TestXconMonitorMultiNodeHealFail(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 2, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
				"NSMD_HEAL_RETRY_COUNT":      "2",
				"NSMD_HEAL_DST_TIMEOUTs":     "1",
			},
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	//defer kubetest.MakeLogsSnapshot(k8s, t)

	icmp := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-0", defaultTimeout)
	g.Expect(icmp).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	g.Expect(nsc).ToNot(BeNil())

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodesConf[1].Nsmd)
	defer closeFunc1()

	// checking goroutine for node0
	expectedFunc0, waitFunc0 := kubetest.NewEventChecker(t, eventCh0)

	// checking goroutine for node1
	expectedFunc1, waitFunc1 := kubetest.NewEventChecker(t, eventCh1)

	k8s.DeletePods(icmp)

	// expected events for node0
	expectedFunc0(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc0(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
	})

	expectedFunc0(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		Timeout:   defaultTimeout, // waiting while heal finishes work
	})

	// expected events for node1
	expectedFunc1(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc1(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
	})

	expectedFunc1(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		Timeout:   defaultTimeout, // waiting while heal finishes work
	})

	waitFunc0()
	waitFunc1()
}

func TestXconMonitorMultiNodeHealSuccess(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 2, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
				"NSMD_HEAL_RETRY_COUNT":      "2",
				"NSMD_HEAL_DST_TIMEOUTs":     "1",
			},
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	defer kubetest.MakeLogsSnapshot(k8s, t)

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-0", defaultTimeout)
	g.Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	g.Expect(nsc).ToNot(BeNil())

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.CrossConnectClientAt(k8s, nodesConf[1].Nsmd)
	defer closeFunc1()

	// checking goroutine for node0
	expectedFunc0, waitFunc0 := kubetest.NewEventChecker(t, eventCh0)

	// checking goroutine for node1
	expectedFunc1, waitFunc1 := kubetest.NewEventChecker(t, eventCh1)

	icmp1 := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-1", defaultTimeout)
	g.Expect(icmp1).ToNot(BeNil())
	k8s.DeletePods(icmp0)

	// expected events for node0
	expectedFunc0(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc0(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
	})

	expectedFunc0(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     true,
		Timeout:   defaultTimeout, // waiting while heal finishes work
	})

	// expected events for node1
	expectedFunc1(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc1(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
	})

	expectedFunc1(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		Timeout:   defaultTimeout, // waiting while heal finishes work
	})

	waitFunc0()
	waitFunc1()
}

func TestXconMonitorNsmgrRestart(t *testing.T) {
	g := NewWithT(t)

	k8s, err := kubetest.NewK8s(g, true)
	defer k8s.Cleanup()
	g.Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodesConfig(k8s, 1, defaultTimeout, []*pods.NSMgrPodConfig{
		{
			Variables: map[string]string{
				nsmd.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
				"NSMD_HEAL_RETRY_COUNT":      "2",
				"NSMD_HEAL_DST_TIMEOUTs":     "1",
			},
		},
	}, k8s.GetK8sNamespace())
	g.Expect(err).To(BeNil())

	defer kubetest.MakeLogsSnapshot(k8s, t)

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	g.Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	g.Expect(nsc).ToNot(BeNil())

	eventCh, closeFunc := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	expectFunc, waitFunc := kubetest.NewEventChecker(t, eventCh)

	expectFunc(&kubetest.SingleEventChecker{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	waitFunc()
	closeFunc()
	k8s.DeletePods(nodesConf[0].Nsmd)

	nodesConf[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig("recovered-nsmgr", nodesConf[0].Node,
		&pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	k8s.WaitLogsContains(nodesConf[0].Nsmd, "nsmd", "All connections are recovered...", defaultTimeout)

	eventChR, closeFuncR := kubetest.CrossConnectClientAt(k8s, nodesConf[0].Nsmd)
	defer closeFuncR()
	expectFuncR, waitFuncR := kubetest.NewEventChecker(t, eventChR)

	checker := &kubetest.OrEventChecker{
		Event1: &kubetest.SingleEventChecker{
			EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
			SrcUp:     true,
			DstUp:     true,
		},
		Event2: &kubetest.MultipleEventChecker{
			Events: []kubetest.EventChecker{
				&kubetest.SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
					Empty:     true,
				},
				&kubetest.SingleEventChecker{
					EventType: crossconnect.CrossConnectEventType_UPDATE,
					SrcUp:     true,
					DstUp:     true,
				},
			},
		},
	}

	expectFuncR(checker)
	waitFuncR()
}
