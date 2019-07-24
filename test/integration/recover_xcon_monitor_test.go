// +build recover

package nsmd_integration_tests

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	. "github.com/onsi/gomega"
)

func TestXconMonitorSingleNodeHealFailed(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	defer kubetest.ShowLogs(k8s, t)

	icmpPod := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	Expect(icmpPod).ToNot(BeNil())

	nscPodNode := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nscPodNode).ToNot(BeNil())

	eventCh, closeFunc := kubetest.XconProxyMonitor(k8s, nodesConf[0], "0")
	defer closeFunc()

	expectedFunc, waitFunc := kubetest.NewEventChecker(t, eventCh)
	k8s.DeletePods(icmpPod)

	expectedFunc(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	})

	expectedFunc(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	})

	waitFunc()
}

func TestXconMonitorSingleNodeHealSuccess(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	defer kubetest.ShowLogs(k8s, t)

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nsc).ToNot(BeNil())

	eventCh, closeFunc := kubetest.XconProxyMonitor(k8s, nodesConf[0], "0")
	defer closeFunc()

	expectedFunc, waitFunc := kubetest.NewEventChecker(t, eventCh)

	icmp1 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-1", defaultTimeout)
	Expect(icmp1).ToNot(BeNil())

	k8s.DeletePods(icmp0)

	expectedFunc(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	})

	expectedFunc(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	})

	waitFunc()
}

func TestXconMonitorMultiNodeHealFail(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	defer kubetest.ShowLogs(k8s, t)

	icmp := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-0", defaultTimeout)
	Expect(icmp).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nsc).ToNot(BeNil())

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.XconProxyMonitor(k8s, nodesConf[0], "0")
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.XconProxyMonitor(k8s, nodesConf[1], "1")
	defer closeFunc1()

	// checking goroutine for node0
	expectedFunc0, waitFunc0 := kubetest.NewEventChecker(t, eventCh0)

	// checking goroutine for node1
	expectedFunc1, waitFunc1 := kubetest.NewEventChecker(t, eventCh1)

	k8s.DeletePods(icmp)

	// expected events for node0
	expectedFunc0(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc0(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	})

	expectedFunc0(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	})

	// expected events for node1
	expectedFunc1(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc1(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	})

	expectedFunc1(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	})

	waitFunc0()
	waitFunc1()
}

func TestXconMonitorMultiNodeHealSuccess(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 2, defaultTimeout)
	Expect(err).To(BeNil())

	defer kubetest.ShowLogs(k8s, t)

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-0", defaultTimeout)
	Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nsc).ToNot(BeNil())

	// monitor client for node0
	eventCh0, closeFunc0 := kubetest.XconProxyMonitor(k8s, nodesConf[0], "0")
	defer closeFunc0()

	// monitor client for node1
	eventCh1, closeFunc1 := kubetest.XconProxyMonitor(k8s, nodesConf[1], "1")
	defer closeFunc1()

	// checking goroutine for node0
	expectedFunc0, waitFunc0 := kubetest.NewEventChecker(t, eventCh0)

	// checking goroutine for node1
	expectedFunc1, waitFunc1 := kubetest.NewEventChecker(t, eventCh1)

	icmp1 := kubetest.DeployICMP(k8s, nodesConf[1].Node, "icmp-1", defaultTimeout)
	Expect(icmp1).ToNot(BeNil())
	k8s.DeletePods(icmp0)

	// expected events for node0
	expectedFunc0(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc0(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	})

	expectedFunc0(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	})

	// expected events for node1
	expectedFunc1(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
	})

	expectedFunc1(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_UPDATE,
		SrcUp:     true,
		DstUp:     false,
		TillNext:  defaultTimeout, // waiting while heal finishes work
	})

	expectedFunc1(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_DELETE,
		SrcUp:     true,
		DstUp:     false,
		LastEvent: true,
	})

	waitFunc0()
	waitFunc1()
}

func TestXconMonitorNsmgrRestart(t *testing.T) {
	RegisterTestingT(t)

	k8s, err := kubetest.NewK8s(true)
	defer k8s.Cleanup()
	Expect(err).To(BeNil())

	nodesConf, err := kubetest.SetupNodes(k8s, 1, defaultTimeout)
	Expect(err).To(BeNil())

	defer kubetest.ShowLogs(k8s, t)

	icmp0 := kubetest.DeployICMP(k8s, nodesConf[0].Node, "icmp-0", defaultTimeout)
	Expect(icmp0).ToNot(BeNil())

	nsc := kubetest.DeployNSC(k8s, nodesConf[0].Node, "nsc-0", defaultTimeout)
	Expect(nsc).ToNot(BeNil())

	eventCh, closeFunc := kubetest.XconProxyMonitor(k8s, nodesConf[0], "0")
	expectFunc, waitFunc := kubetest.NewEventChecker(t, eventCh)

	expectFunc(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	})

	k8s.DeletePods(nodesConf[0].Nsmd)
	waitFunc()
	closeFunc()

	nodesConf[0].Nsmd = k8s.CreatePod(pods.NSMgrPodWithConfig("recovered-nsmgr", nodesConf[0].Node,
		&pods.NSMgrPodConfig{Namespace: k8s.GetK8sNamespace()})) // Recovery NSEs
	k8s.WaitLogsContains(nodesConf[0].Nsmd, "nsmd", "All connections are recovered...", defaultTimeout)

	eventChR, closeFuncR := kubetest.XconProxyMonitor(k8s, nodesConf[0], "0")
	defer closeFuncR()
	expectFuncR, waitFuncR := kubetest.NewEventChecker(t, eventChR)

	expectFuncR(kubetest.EventDescription{
		EventType: crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER,
		SrcUp:     true,
		DstUp:     true,
		LastEvent: true,
	})

	waitFuncR()
}
