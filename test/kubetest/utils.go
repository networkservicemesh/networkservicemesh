package kubetest

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/artifacts"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/properties"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/networkservicemesh/networkservicemesh/test/applications/cmd/icmp-responder-nse/flags"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	arv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	nsmd2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/sdk/prefix_pool"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/jaeger"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	nsmrbac "github.com/networkservicemesh/networkservicemesh/test/kubetest/rbac"
)

type NodeConf struct {
	Nsmd      *v1.Pod
	Forwarder *v1.Pod
	Node      *v1.Node
}

// NSCCheckInfo - Structure to hold client ping information
type NSCCheckInfo struct {
	ipResponse    string
	routeResponse string
	pingResponse  string
	errOut        string
}

// PodSupplier - Type to pass supplier of pod
type PodSupplier = func(*K8s, *v1.Node, string, time.Duration) *v1.Pod

// NsePinger - Type to pass pinger for pod
type NsePinger = func(k8s *K8s, from *v1.Pod) bool

// NscChecker - Type to pass checked for pod
type NscChecker = func(*K8s, *v1.Pod) *NSCCheckInfo
type ipParser = func(string) (string, error)

// SetupNodes - Setup NSMgr and Forwarder for particular number of nodes in cluster
func SetupNodes(k8s *K8s, nodesCount int, timeout time.Duration) ([]*NodeConf, error) {
	return SetupNodesConfig(k8s, nodesCount, timeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
}

//FindJaegerPod returns jaeger pod or nil
func FindJaegerPod(k8s *K8s) *v1.Pod {
	pods, err := k8s.ListPods()
	if err != nil {
		logrus.Errorf("Can not find jaeger pod %v", err.Error())
		return nil
	}
	for i := range pods {
		p := &pods[i]
		if strings.Contains(p.Name, "jaeger") {
			return p
		}
	}
	return nil
}

//DeployCorefile - Creates configmap with Corefile content
func DeployCorefile(k8s *K8s, name, content string) error {
	_, err := k8s.CreateConfigMap(&v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: k8s.GetK8sNamespace(),
		},

		BinaryData: map[string][]byte{
			"Corefile": []byte(content),
		},
	})
	return err
}

// SetupNodesConfig - Setup NSMgr and Forwarder for particular number of nodes in cluster
func SetupNodesConfig(k8s *K8s, nodesCount int, timeout time.Duration, conf []*pods.NSMgrPodConfig, namespace string) ([]*NodeConf, error) {
	nodes := k8s.GetNodesWait(nodesCount, timeout)
	k8s.g.Expect(len(nodes) >= nodesCount).To(Equal(true),
		"At least one Kubernetes node is required for this test")
	if artifacts.NeedToSave() {
		if !jaeger.UseService.GetBooleanOrDefault(false) {
			jaegerPod := k8s.CreatePod(pods.Jaeger())
			jaeger.AgentHost.Set(jaegerPod.Status.PodIP)
			k8s.WaitLogsContains(jaegerPod, jaegerPod.Spec.Containers[0].Name, "Starting HTTP server", timeout)
		} else if jaeger.AgentHost.StringValue() == "" {
			template := pods.Jaeger()
			template.Spec.NodeSelector = map[string]string{
				"kubernetes.io/hostname": nodes[0].Labels["kubernetes.io/hostname"],
			}
			jaegerPod := k8s.CreatePod(template)
			_, err := k8s.CreateService(pods.JaegerService(jaegerPod), k8s.namespace)
			k8s.g.Expect(err).Should(BeNil())
			jaeger.AgentHost.Set(getExternalOrInternalAddress(&nodes[0]))
			jaeger.AgentPort.Set(jaeger.GetNodePort())
			k8s.WaitLogsContains(jaegerPod, jaegerPod.Spec.Containers[0].Name, "Starting HTTP server", timeout)
		}
	}

	var wg sync.WaitGroup
	confs := make([]*NodeConf, nodesCount)
	var resultError error
	for ii := 0; ii < nodesCount; ii++ {
		wg.Add(1)
		i := ii
		go func() {
			defer wg.Done()
			startTime := time.Now()
			node := &nodes[i]
			nsmdName := fmt.Sprintf("nsmgr-%s", node.Name)
			forwarderName := fmt.Sprintf("nsmd-forwarder-%s", node.Name)
			var corePod *v1.Pod
			var forwarderPod *v1.Pod
			debug := false
			if i >= len(conf) {
				corePod = pods.NSMgrPod(nsmdName, node, k8s.GetK8sNamespace())
				forwarderPod = pods.ForwardingPlaneWithConfig(forwarderName, node, DefaultForwarderVariables(k8s.GetForwardingPlane()), k8s.GetForwardingPlane())
			} else {
				conf[i].Namespace = namespace
				if conf[i].Nsmd == pods.NSMgrContainerDebug || conf[i].NsmdK8s == pods.NSMgrContainerDebug || conf[i].NsmdP == pods.NSMgrContainerDebug {
					debug = true
				}
				corePod = pods.NSMgrPodWithConfig(nsmdName, node, conf[i])
				forwarderPod = pods.ForwardingPlaneWithConfig(forwarderName, node, conf[i].ForwarderVariables, k8s.GetForwardingPlane())
			}
			corePods, err := k8s.CreatePodsRaw(PodStartTimeout, true, corePod, forwarderPod)

			if err != nil {
				logrus.Errorf("Failed to Started NSMgr/Forwarder: %v on node %s %v", time.Since(startTime), node.Name, err)
				resultError = err
				return
			}
			if debug {
				podContainer := "nsmd"
				if conf[i].Nsmd == pods.NSMgrContainerDebug {
					podContainer = "nsmd"
				} else if conf[i].NsmdP == pods.NSMgrContainerDebug {
					podContainer = "nsmdp"
				}

				k8s.WaitLogsContains(corePod, podContainer, "API server listening at: [::]:40000", timeout)
				logrus.Infof("Debug devenv container is running. Please do\n make k8s-forward pod=%v port1=40000 port2=40000. And attach via debugger...", corePod.Name)
			}
			nsmd, forwarder, err := deployNSMgrAndForwarder(k8s, corePods, timeout)
			if err != nil {
				logrus.Errorf("Failed to Started NSMgr/Forwarder: %v on node %s %v", time.Since(startTime), node.Name, err)
				resultError = err
				return
			}
			logrus.Printf("Started NSMgr/Forwarder: %v on node %s", time.Since(startTime), node.Name)
			confs[i] = &NodeConf{
				Nsmd:      nsmd,
				Forwarder: forwarder,
				Node:      &nodes[i],
			}
		}()
	}
	wg.Wait()
	return confs, resultError
}

func getExternalOrInternalAddress(n *v1.Node) string {
	internalAddr := ""
	for i := range n.Status.Addresses {
		addr := &n.Status.Addresses[i]
		if addr.Type == v1.NodeExternalIP {
			return addr.Address
		} else if addr.Type == v1.NodeInternalIP && internalAddr == "" {
			internalAddr = addr.Address
		}
	}
	return internalAddr
}

func deployNSMgrAndForwarder(k8s *K8s, corePods []*v1.Pod, timeout time.Duration) (nsmd, forwarder *v1.Pod, err error) {
	for _, pod := range corePods {
		if !k8s.IsPodReady(pod) {
			return nil, nil, errors.Errorf("Pod %v is not ready...", pod.Name)
		}
	}
	nsmd = corePods[0]
	forwarder = corePods[1]

	k8s.g.Expect(nsmd.Name).To(Equal(corePods[0].Name))
	k8s.g.Expect(forwarder.Name).To(Equal(corePods[1].Name))

	WaitNSMgrDeployed(k8s, nsmd, timeout)

	err = nil
	return
}

// WaitNSMgrDeployed - wait for NSMgr pod to be fully deployed.
func WaitNSMgrDeployed(k8s *K8s, nsmd *v1.Pod, timeout time.Duration) {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		_ = k8s.WaitLogsContainsRegex(nsmd, "nsmd", "NSM gRPC API Server: .* is operational", timeout)
	}()
	go func() {
		defer wg.Done()
		k8s.WaitLogsContains(nsmd, "nsmdp", "nsmdp: successfully started", timeout)
	}()
	go func() {
		defer wg.Done()
		k8s.WaitLogsContains(nsmd, "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", timeout)
	}()
	wg.Wait()
}

// DeployProxyNSMgr - Setup Proxy NSMgr on Cluster
func DeployProxyNSMgr(k8s *K8s, node *v1.Node, name string, timeout time.Duration) (pnsmd *v1.Pod, err error) {
	template := pods.ProxyNSMgrPod(name, node, k8s.GetK8sNamespace())
	return deployProxyNSMgr(k8s, template, node, timeout)
}

// DeployProxyNSMgrWithConfig - Setup Proxy NSMgr on Cluster with custom config
func DeployProxyNSMgrWithConfig(k8s *K8s, node *v1.Node, name string, timeout time.Duration, config *pods.NSMgrPodConfig) (pnsmd *v1.Pod, err error) {
	template := pods.ProxyNSMgrPodWithConfig(name, node, config)
	return deployProxyNSMgr(k8s, template, node, timeout)
}

func deployProxyNSMgr(k8s *K8s, template *v1.Pod, node *v1.Node, timeout time.Duration) (pnsmd *v1.Pod, err error) {
	startTime := time.Now()

	logrus.Infof("Starting Proxy NSMgr %s on node: %s", template.Name, node.Name)
	tempPods, err := k8s.CreatePodsRaw(PodStartTimeout, true, template)

	if err != nil {
		logrus.Errorf("Failed to Started pNSMgr: %v on node %s %v", time.Since(startTime), node.Name, err)
		return nil, err
	}

	pnsmd = tempPods[0]

	_ = k8s.WaitLogsContainsRegex(pnsmd, "proxy-nsmd", "NSM gRPC API Server: .* is operational", timeout)
	k8s.WaitLogsContains(pnsmd, "proxy-nsmd-k8s", "proxy-nsmd-k8s initialized and waiting for connection", timeout)

	logrus.Printf("Proxy NSMgr started done: %v", time.Since(startTime))
	return
}

// RunProxyNSMgrService deploys proxy NSMgr service to access proxy NSMgr pods with single DNS name
func RunProxyNSMgrService(k8s *K8s) func() {
	svc, err := k8s.CreateService(pods.ProxyNSMgrSvc(), k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())

	return func() {
		_ = k8s.DeleteService(svc, k8s.GetK8sNamespace())
	}
}

// DeployNSMRS - Setup NSMRS on Cluster with default config
func DeployNSMRS(k8s *K8s, node *v1.Node, name string, timeout time.Duration, variables map[string]string) *v1.Pod {
	return deployNSMRS(k8s, nodeName(node), name, timeout,
		pods.NSMRSPod(name, node, variables),
	)
}

// DeployNSMRSWithConfig - Setup NSMRS on Cluster
func DeployNSMRSWithConfig(k8s *K8s, node *v1.Node, name string, timeout time.Duration, config *pods.NSMgrPodConfig) *v1.Pod {
	return deployNSMRS(k8s, nodeName(node), name, timeout,
		pods.NSMRSPodWithConfig(name, node, config),
	)
}

func deployNSMRS(k8s *K8s, nodeName, name string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()

	logrus.Infof("Starting NSM Service Registry Server on node: %s", nodeName)
	nsmrs := k8s.CreatePod(template)
	k8s.g.Expect(nsmrs.Name).To(Equal(name))

	_ = k8s.WaitLogsContainsRegex(nsmrs, "nsmrs", "Service Registry gRPC API Server: .* is operational", timeout)

	logrus.Printf("NSM Service Registry Server %v started done: %v", name, time.Since(startTime))
	return nsmrs
}

// DeployICMP deploys 'icmp-responder-nse' pod with '-routes' flag set
func DeployICMP(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	flags := flags.ICMPResponderFlags{
		Routes: true,
	}
	return deployICMP(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, flags.Commands(), node, defaultICMPEnv(k8s.UseIPv6()), pods.NSEServiceAccount),
	)
}

// DeployICMPAndCoredns deploys 'icmp-responder-nse' pod with '-routes', '-dns' flag set. Also injected coredns server.
func DeployICMPAndCoredns(k8s *K8s, node *v1.Node, name, corednsConfigName string, timeout time.Duration) *v1.Pod {
	flags := flags.ICMPResponderFlags{
		Routes: true,
		DNS:    true,
	}
	return deployICMP(k8s, nodeName(node), name, timeout,
		pods.InjectCoredns(pods.TestCommonPod(name, flags.Commands(), node, defaultICMPEnv(k8s.UseIPv6()), pods.NSEServiceAccount), corednsConfigName),
	)
}

// DeployICMPWithConfig deploys 'icmp-responder-nse' pod with '-routes' flag set and given grace period
func DeployICMPWithConfig(k8s *K8s, node *v1.Node, name string, timeout time.Duration, gracePeriod int64) *v1.Pod {
	flags := flags.ICMPResponderFlags{
		Routes: true,
	}
	pod := pods.TestCommonPod(name, flags.Commands(), node, defaultICMPEnv(k8s.UseIPv6()), pods.NSEServiceAccount)
	pod.Spec.TerminationGracePeriodSeconds = &gracePeriod
	return deployICMP(k8s, nodeName(node), name, timeout, pod)
}

// DeployDirtyICMP deploys 'icmp-responder-nse' pod with '-dirty' flag set
func DeployDirtyICMP(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	flags := flags.ICMPResponderFlags{
		Dirty: true,
	}
	return deployDirtyNSE(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, flags.Commands(), node, defaultICMPEnv(k8s.UseIPv6()), pods.NSEServiceAccount),
	)
}

// DeployNeighborNSE deploys 'icmp-responder-nse' pod with '-neighbors' flag set
func DeployNeighborNSE(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	flags := flags.ICMPResponderFlags{
		Neighbors: true,
	}
	return deployICMP(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, flags.Commands(), node, defaultICMPEnv(k8s.UseIPv6()), pods.NSEServiceAccount),
	)
}

// DeployUpdatingNSE deploys 'icmp-responder-nse' pod with '-update' flag set
func DeployUpdatingNSE(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	flags := flags.ICMPResponderFlags{
		Update: true,
	}
	return deployICMP(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, flags.Commands(), node, defaultICMPEnv(k8s.UseIPv6()), pods.NSEServiceAccount),
	)
}

//DeployMonitoringNSCAndCoredns deploys pod of nsm-dns-monitoring-nsc and coredns
func DeployMonitoringNSCAndCoredns(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	envs := defaultNSCEnv()
	template := pods.TestCommonPod(name, []string{"/bin/monitoring-dns-nsc"}, node, envs, pods.NSCServiceAccount)
	pods.InjectCorednsWithSharedFolder(template)
	result := deployNSC(k8s, nodeName(node), name, "nsc", timeout, template)
	k8s.WaitLogsContains(result, "coredns", "CoreDNS-", timeout)
	return result
}

// DeployNscAndNsmCoredns deploys pod of default client and coredns
func DeployNscAndNsmCoredns(k8s *K8s, node *v1.Node, name, corefileName string, timeout time.Duration) *v1.Pod {
	envs := defaultNSCEnv()
	envs["UPDATE_API_CLIENT_SOCKET"] = "/etc/coredns/client.sock"
	return deployNSC(k8s, nodeName(node), name, "nsm-init", timeout,
		pods.InjectCoredns(pods.NSCPod(name, node, defaultNSCEnv()), corefileName),
	)
}

// DeployNSC - Setup Default Client
func DeployNSC(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, nodeName(node), name, "nsm-init", timeout,
		pods.NSCPod(name, node, defaultNSCEnv()),
	)
}

// DeployNSCMonitor - Setup Default Client
func DeployNSCMonitor(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, nodeName(node), name, "nsm-init", timeout,
		pods.NSCMonitorPod(name, node, defaultNSCEnv()),
	)
}

// DeployNSCWithEnv - Setup Default Client with custom environment
func DeployNSCWithEnv(k8s *K8s, node *v1.Node, name string, timeout time.Duration, env map[string]string) *v1.Pod {
	return deployNSC(k8s, nodeName(node), name, "nsm-init", timeout, pods.NSCPod(name, node,
		env))
}

// DeployNSCWebhook - Setup Default Client with webhook
func DeployNSCWebhook(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, nodeName(node), name, "nsm-init-container", timeout,
		pods.NSCPodWebhook(name, node),
	)
}

// DeployMonitoringNSC deploys 'monitoring-nsc' pod
func DeployMonitoringNSC(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, nodeName(node), name, "monitoring-nsc", timeout,
		pods.TestCommonPod(name, []string{"/bin/monitoring-nsc"}, node, defaultNSCEnv(), pods.NSCServiceAccount),
	)
}

// NoHealNSMgrPodConfig returns config for NSMgr. The config has properties for disabling healing for nse
func NoHealNSMgrPodConfig(k8s *K8s) []*pods.NSMgrPodConfig {
	return []*pods.NSMgrPodConfig{
		noHealNSMgrPodConfig(k8s),
		noHealNSMgrPodConfig(k8s),
	}
}

// InitSpireSecurity deploys pod that proxy Spire certificates to test environment
func InitSpireSecurity(k8s *K8s) func() {
	spireProxy := k8s.CreatePod(pods.SpireProxyPod())

	fwd, err := k8s.NewPortForwarder(spireProxy, 7001)
	k8s.g.Expect(err).To(BeNil())

	err = fwd.Start()
	k8s.g.Expect(err).To(BeNil())

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", fwd.ListenPort))
	k8s.g.Expect(err).To(BeNil())

	provider, err := security.NewSpireProvider("tcp://" + addr.String())
	k8s.g.Expect(err).To(BeNil())

	tools.InitConfig(tools.DialConfig{
		SecurityProvider: provider,
	})

	return func() {
		fwd.Stop()
	}
}

func defaultICMPEnv(useIPv6 bool) map[string]string {
	if !useIPv6 {
		return map[string]string{
			"ENDPOINT_NETWORK_SERVICE": "icmp-responder",
			"ENDPOINT_LABELS":          "app=icmp",
			"IP_ADDRESS":               "172.16.1.0/24",
		}
	}
	return map[string]string{
		"ENDPOINT_NETWORK_SERVICE": "icmp-responder",
		"ENDPOINT_LABELS":          "app=icmp",
		"IP_ADDRESS":               "100::/64",
	}
}

func defaultNSCEnv() map[string]string {
	return map[string]string{
		"CLIENT_LABELS":          "app=icmp",
		"CLIENT_NETWORK_SERVICE": "icmp-responder",
	}
}

func noHealNSMgrPodConfig(k8s *K8s) *pods.NSMgrPodConfig {
	return &pods.NSMgrPodConfig{
		Variables: map[string]string{
			nsmd2.NsmdDeleteLocalRegistry:     "true", // Do not use local registry restore for clients/NSEs
			properties.NsmdHealDSTWaitTimeout: "1",    // 1 second
			properties.NsmdHealEnabled:        "false",
		},
		Namespace:          k8s.GetK8sNamespace(),
		ForwarderVariables: DefaultForwarderVariables(k8s.GetForwardingPlane()),
	}
}

func nodeName(node *v1.Node) string {
	if node == nil {
		return "Random Node"
	}

	return node.Name
}

func deployICMP(k8s *K8s, nodeName, name string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()

	logrus.Infof("Starting ICMP Responder NSE on node: %s", nodeName)
	icmp := k8s.CreatePod(template)
	k8s.g.Expect(icmp.Name).To(Equal(name))

	k8s.WaitLogsContains(icmp, template.Spec.Containers[0].Name, "NSE: channel has been successfully advertised, waiting for connection from NSM...", timeout)

	logrus.Printf("ICMP Responder %v started done: %v", name, time.Since(startTime))
	return icmp
}

func deployDirtyNSE(k8s *K8s, nodeName, name string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()

	logrus.Infof("Starting dirty NSE on node: %s", nodeName)
	dirty := k8s.CreatePod(template)
	k8s.g.Expect(dirty.Name).To(Equal(name))

	k8s.WaitLogsContains(dirty, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", timeout)

	logrus.Printf("Dirty NSE %v started done: %v", name, time.Since(startTime))
	return dirty
}

func deployNSC(k8s *K8s, nodeName, name, container string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()
	k8s.g.Expect(template).ShouldNot(BeNil())

	logrus.Infof("Starting NSC %s on node: %s", name, nodeName)

	nsc := k8s.CreatePod(template)

	k8s.g.Expect(nsc.Name).To(Equal(name))
	k8s.WaitLogsContains(nsc, container, "nsm client: initialization is completed successfully", timeout)

	logrus.Printf("NSC started done: %v", time.Since(startTime))
	return nsc
}

// DeployAdmissionWebhook - Setup Admission Webhook
func DeployAdmissionWebhook(k8s *K8s, name, image, namespace string, timeout time.Duration) (*arv1beta1.MutatingWebhookConfiguration, *appsv1.Deployment, *v1.Service) {
	_, caCert := CreateAdmissionWebhookSecret(k8s, name, namespace)
	awService := CreateAdmissionWebhookService(k8s, name, namespace)
	awDeployment := CreateAdmissionWebhookDeployment(k8s, name, image, namespace)
	admissionWebhookPod := waitWebhookPod(k8s, awDeployment.Name, timeout)
	k8s.g.Expect(admissionWebhookPod).ShouldNot(BeNil())
	awc := CreateMutatingWebhookConfiguration(k8s, caCert, name, namespace)
	return awc, awDeployment, awService
}

// DeleteAdmissionWebhook - Delete admission webhook
func DeleteAdmissionWebhook(k8s *K8s, secretName string,
	awc *arv1beta1.MutatingWebhookConfiguration, awDeployment *appsv1.Deployment, awService *v1.Service, namespace string) {
	err := k8s.DeleteService(awService, namespace)
	k8s.g.Expect(err).To(BeNil())

	err = k8s.DeleteDeployment(awDeployment, namespace)
	k8s.g.Expect(err).To(BeNil())

	err = k8s.DeleteMutatingWebhookConfiguration(awc)
	k8s.g.Expect(err).To(BeNil())

	err = k8s.DeleteSecret(secretName, namespace)
	k8s.g.Expect(err).To(BeNil())
}

// CreateAdmissionWebhookSecret - Create admission webhook secret
func CreateAdmissionWebhookSecret(k8s *K8s, name, namespace string) (*v1.Secret, tls.Certificate) {
	now := time.Now()
	crt := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "admission-controller-ca",
		},
		NotBefore:    now.Add(-time.Hour * 24 * 365),
		NotAfter:     now.Add(time.Hour * 24 * 365),
		SerialNumber: big.NewInt(now.Unix()),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		IsCA:                  true,
		BasicConstraintsValid: true,
		DNSNames: []string{
			name + "-svc." + namespace,
			name + "-svc." + namespace + ".svc",
		},
	}
	// Generate a private key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	k8s.g.Expect(err).Should(BeNil())

	// PEM-encode the private key.
	keyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  keyutil.RSAPrivateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	// Self-sign the certificate using the private key.
	sig, err := x509.CreateCertificate(rand.Reader, &crt, &crt, key.Public(), key)
	k8s.g.Expect(err).Should(BeNil())

	// PEM-encode the signed certificate
	sigBytes := pem.EncodeToMemory(&pem.Block{
		Type:  cert.CertificateBlockType,
		Bytes: sig,
	})

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nsm-admission-webhook-certs",
			Namespace: namespace,
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			v1.TLSCertKey:       sigBytes,
			v1.TLSPrivateKeyKey: keyBytes,
		},
	}
	result, err := tls.X509KeyPair(sigBytes, keyBytes)
	k8s.g.Expect(err).Should(BeNil())
	secret, err = k8s.CreateSecret(secret, namespace)
	k8s.g.Expect(err).Should(BeNil())
	return secret, result
}

// CreateMutatingWebhookConfiguration - Setup Mutating webhook configuration
func CreateMutatingWebhookConfiguration(k8s *K8s, c tls.Certificate, name, namespace string) *arv1beta1.MutatingWebhookConfiguration {
	servicePath := "/mutate"
	caBundle := pem.EncodeToMemory(&pem.Block{
		Type:  cert.CertificateBlockType,
		Bytes: c.Certificate[0],
	})
	mutatingWebhookConf := &arv1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind: "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-cfg",
			Labels: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
		Webhooks: []arv1beta1.MutatingWebhook{
			{
				Name: "admission-webhook.networkservicemesh.io",
				ClientConfig: arv1beta1.WebhookClientConfig{
					Service: &arv1beta1.ServiceReference{
						Namespace: namespace,
						Name:      name + "-svc",
						Path:      &servicePath,
					},
					CABundle: caBundle,
				},
				Rules: []arv1beta1.RuleWithOperations{
					{
						Operations: []arv1beta1.OperationType{
							arv1beta1.Create,
						},
						Rule: arv1beta1.Rule{
							APIGroups:   []string{"apps", "extensions", ""},
							APIVersions: []string{"v1", "v1beta1"},
							Resources:   []string{"deployments", "services", "pods"},
						},
					},
				},
			},
		},
	}
	awc, err := k8s.CreateMutatingWebhookConfiguration(mutatingWebhookConf)
	k8s.g.Expect(err).To(BeNil())
	return awc
}

// CreateAdmissionWebhookDeployment - Setup Admission Webhook deoloyment
func CreateAdmissionWebhookDeployment(k8s *K8s, name, image, namespace string) *appsv1.Deployment {
	deployment := pods.AdmissionWebhookDeployment(name, image)

	awDeployment, err := k8s.CreateDeployment(deployment, namespace)
	k8s.g.Expect(err).To(BeNil())

	return awDeployment
}

// CreateAdmissionWebhookService - Create Admission Webhook Service
func CreateAdmissionWebhookService(k8s *K8s, name, namespace string) *v1.Service {
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-svc",
			Labels: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:       443,
					TargetPort: intstr.FromInt(443),
				},
			},
			Selector: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
	}
	awService, err := k8s.CreateService(service, namespace)
	k8s.g.Expect(err).To(BeNil())

	return awService
}

// GetNodeInternalIP - Pop InternalIP from node addresses
func GetNodeInternalIP(node *v1.Node) (string, error) {
	for i := range node.Status.Addresses {
		if node.Status.Addresses[i].Type == "InternalIP" {
			return node.Status.Addresses[i].Address, nil
		}
	}
	return "", errors.Errorf("node %s does not have Internal IP address", node.ObjectMeta.Name)
}

// GetNodeExternalIP - Pop InternalIP from node addresses
func GetNodeExternalIP(node *v1.Node) (string, error) {
	for i := range node.Status.Addresses {
		if node.Status.Addresses[i].Type == "ExternalIP" {
			return node.Status.Addresses[i].Address, nil
		}
	}
	return "", errors.Errorf("node %s does not have Internal IP address", node.ObjectMeta.Name)
}

// PrintLogs - Print Client print information
func (info *NSCCheckInfo) PrintLogs() {
	if info == nil {
		return
	}
	logrus.Errorf("===================== NSC IP Addr %v\n=====================", info.ipResponse)
	logrus.Errorf("===================== NSC IP Route %v\n=====================", info.routeResponse)
	logrus.Errorf("===================== NSC IP PING %v\n=====================", info.pingResponse)
	logrus.Errorf("===================== NSC errOut %v\n=====================", info.errOut)
}

// CheckNSC - Perform default check for client to NSE operations
func CheckNSC(k8s *K8s, nscPodNode *v1.Pod) *NSCCheckInfo {
	nscLocalRemoteIPs := getNSCLocalRemoteIPs(k8s, nscPodNode)
	return checkNSCConfig(k8s, nscPodNode, nscLocalRemoteIPs[0], nscLocalRemoteIPs[1])
}

func waitWebhookPod(k8s *K8s, name string, timeout time.Duration) *v1.Pod {
	timoutChannel := time.After(timeout)
	for {
		select {
		case <-timoutChannel:
			logrus.Errorf("can find pod %v during %v", name, timeout)
			return nil
		default:
			list, err := k8s.ListPods()
			k8s.g.Expect(err).Should(BeNil())
			for i := 0; i < len(list); i++ {
				p := &list[i]
				if strings.Contains(p.Name, name) {
					result, err := k8s.blockUntilPodReady(k8s.clientset, timeout, p)
					k8s.g.Expect(err).Should(BeNil())
					return result
				}
			}
		}
		<-time.After(time.Millisecond * 100)
	}
}
func checkNSCConfig(k8s *K8s, nscPodNode *v1.Pod, checkIP, pingIP string) *NSCCheckInfo {
	var err error
	info := &NSCCheckInfo{}

	pingCommand := "ping"
	publicDNSAddress := "8.8.8.8"

	if k8s.UseIPv6() {
		pingCommand = "ping6"
		publicDNSAddress = "2001:4860:4860::8888"
	}

	/* Check IP address */
	if !k8s.UseIPv6() {
		info.ipResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
	} else {
		info.ipResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "-6", "addr")
	}
	k8s.g.Expect(err).To(BeNil())
	k8s.g.Expect(info.errOut).To(Equal(""))
	k8s.g.Expect(strings.Contains(info.ipResponse, checkIP)).To(Equal(true))
	k8s.g.Expect(strings.Contains(info.ipResponse, "nsm")).To(Equal(true))

	if err != nil || info.errOut != "" {
		logrus.Println("NSC IP status, NOK")
		logrus.Println("ipResponse:", info.ipResponse)
		logrus.Println("err:", err)
		logrus.Println("info.errOut:", info.errOut)
	} else {
		logrus.Println("NSC IP status, OK")
	}

	/* Check route */
	if !k8s.UseIPv6() {
		info.routeResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
	} else {
		info.routeResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "-6", "route")
	}
	k8s.g.Expect(err).To(BeNil())
	k8s.g.Expect(info.errOut).To(Equal(""))
	k8s.g.Expect(strings.Contains(info.routeResponse, publicDNSAddress)).To(Equal(true))
	k8s.g.Expect(strings.Contains(info.routeResponse, "nsm")).To(Equal(true))

	if err != nil || info.errOut != "" {
		logrus.Println("NSC Route status, NOK")
		logrus.Println("routeResponse:", info.routeResponse)
		logrus.Println("err:", err)
		logrus.Println("info.errOut:", info.errOut)
	} else {
		logrus.Println("NSC Route status, OK")
	}

	/* Check ping */
	info.pingResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, pingCommand, pingIP, "-A", "-c", "5")
	k8s.g.Expect(err).To(BeNil())
	k8s.g.Expect(info.errOut).To(Equal(""))

	pingNOK := strings.Contains(info.pingResponse, "100% packet loss")
	k8s.g.Expect(pingNOK).To(Equal(false))
	if err != nil || info.errOut != "" || pingNOK {
		logrus.Printf("NSC Ping, NOK")
		logrus.Println("pingResponse:", info.pingResponse)
		logrus.Println("err:", err)
		logrus.Println("info.errOut:", info.errOut)
	} else {
		logrus.Printf("NSC Ping, OK")
	}
	return info
}

// IsBrokeTestsEnabled - Check if broken tests are enabled
func IsBrokeTestsEnabled() bool {
	_, ok := os.LookupEnv("BROKEN_TESTS_ENABLED")
	return ok
}

func parseAddr(ipReponse string) (string, error) {
	nsmInterfaceIndex := strings.Index(ipReponse, "nsm")
	if nsmInterfaceIndex == -1 {
		return "", errors.New(fmt.Sprintf("bad ip response %v", ipReponse))
	}
	nsmBlock := ipReponse[nsmInterfaceIndex:]
	inetIndex := strings.Index(nsmBlock, "inet ")
	if inetIndex == -1 {
		return "", errors.New(fmt.Sprintf("bad ip response %v", ipReponse))
	}
	inetBlock := nsmBlock[inetIndex+len("inet "):]
	ip := inetBlock[:strings.Index(inetBlock, " ")]
	return ip, nil
}

func getNSEAddr(k8s *K8s, nsc *v1.Pod, parseIP ipParser, showIPCommand ...string) (net.IP, error) {
	response, _, _ := k8s.Exec(nsc, nsc.Spec.Containers[0].Name, showIPCommand...)
	response = strings.TrimSpace(response)
	if response == "" {
		return nil, errors.Errorf("exec [%v] returned empty response", showIPCommand)
	}
	addr, err := parseIP(response)
	if err != nil {
		return nil, err
	}
	ip, ipNet, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}

	ip, err = prefix_pool.IncrementIP(ip, ipNet)
	if err != nil {
		return nil, err
	}
	return ip, nil
}

func pingNse(k8s *K8s, from *v1.Pod) string {
	nseIp, err := getNSEAddr(k8s, from, parseAddr, "ip", "addr")
	k8s.g.Expect(err).Should(BeNil())
	logrus.Infof("%v trying ping to %v", from.Name, nseIp)
	response, _, _ := k8s.Exec(from, from.Spec.Containers[0].Name, "ping", nseIp.String(), "-A", "-c", "4")
	logrus.Infof("ping result: %s", response)
	return response
}

// IsNsePinged - Checks if the interface to NSE exists and NSE is pinged
func IsNsePinged(k8s *K8s, from *v1.Pod) (result bool) {
	response := pingNse(k8s, from)
	if strings.TrimSpace(response) != "" && !strings.Contains(response, "100% packet loss") && !strings.Contains(response, "Fail") {
		result = true
		logrus.Info("Ping successful")
	}

	return result
}

//NSLookup invokes nslookup on pod with concrete hostname. Tries several times
func NSLookup(k8s *K8s, pod *v1.Pod, hostname string) bool {
	for i := 0; i < 10; i++ {
		logrus.Infof("Trying nslookup from container %v host by name %v", pod.Spec.Containers[0].Name, hostname)
		response, _, _ := k8s.Exec(pod, pod.Spec.Containers[0].Name, "nslookup", hostname)
		logrus.Infof("Response: %v", response)
		if strings.Contains(response, "Address") && strings.Contains(response, "Name:") {
			return true
		}
		<-time.After(time.Second)
	}
	return false
}

//PingByHostName tries ping hostname from the first container of pod
func PingByHostName(k8s *K8s, pod *v1.Pod, hostname string) bool {
	for i := 0; i < 10; i++ {
		logrus.Infof("Trying ping from container %v host by name %v", pod.Spec.Containers[0].Name, hostname)
		response, reason, err := k8s.Exec(pod, pod.Spec.Containers[0].Name, "ping", hostname, "-c", "4")
		if err == nil {
			logrus.Infof("Ping by hostname is success. Response %v", response)
			return true
		}
		logrus.Errorf("Can't ping by hostname. Reason: %v, error: %v", reason, err)
		<-time.After(time.Second)
	}
	return false
}

// ServiceRegistryAt creates new service registry on 5000 port
func ServiceRegistryAt(k8s *K8s, nsmgr *v1.Pod) (serviceregistry.ServiceRegistry, func()) {
	fwd, err := k8s.NewPortForwarder(nsmgr, 5000)
	k8s.g.Expect(err).To(BeNil())

	err = fwd.Start()
	k8s.g.Expect(err).To(BeNil())

	sr := nsmd2.NewServiceRegistryAt(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	return sr, fwd.Stop
}

// PrepareRegistryClients prepare nse and nsm registry clients
func PrepareRegistryClients(k8s *K8s, nsmd *v1.Pod) (registry.NetworkServiceRegistryClient, registry.NsmRegistryClient, func()) {
	serviceRegistry, closeFunc := ServiceRegistryAt(k8s, nsmd)

	nseRegistryClient, err := serviceRegistry.NseRegistryClient(context.Background())
	k8s.g.Expect(err).To(BeNil())

	nsmRegistryClient, err := serviceRegistry.NsmRegistryClient(context.Background())
	k8s.g.Expect(err).To(BeNil())

	return nseRegistryClient, nsmRegistryClient, closeFunc
}

// ExpectNSEsCountToBe checks if nses count becomes 'countExpected' after a time
func ExpectNSEsCountToBe(k8s *K8s, countWas, countExpected int) {
	if countWas == countExpected {
		<-time.After(10 * time.Second)
	} else {
		for i := 0; i < 10; i++ {
			if nses, err := k8s.GetNSEs(); err == nil && len(nses) == countExpected {
				break
			}
			<-time.After(1 * time.Second)
		}
	}

	nses, err := k8s.GetNSEs()

	k8s.g.Expect(err).To(BeNil())
	k8s.g.Expect(len(nses)).To(Equal(countExpected), fmt.Sprint(nses))
}

// ExpectNSMsCountToBe checks if nsms count becomes 'countExpected' after a time
func ExpectNSMsCountToBe(k8s *K8s, countWas, countExpected int) {
	if countWas == countExpected {
		<-time.After(10 * time.Second)
	} else {
		for i := 0; i < 10; i++ {
			if nsmList, err := k8s.GetNSMList(); err == nil && len(nsmList) == countExpected {
				break
			}
			<-time.After(1 * time.Second)
		}
	}

	nsmList, err := k8s.GetNSMList()

	k8s.g.Expect(err).To(BeNil())
	k8s.g.Expect(len(nsmList)).To(Equal(countExpected), fmt.Sprint(nsmList))
}

// DeployPrometheus deploys prometheus roles, configMap, deployment and service
func DeployPrometheus(k8s *K8s) ([]nsmrbac.Role, *appsv1.Deployment, *v1.Service) {
	roles := CreatePrometheusClusterRoles(k8s)
	CreatePrometheusConfigMap(k8s)
	depl := CreatePrometheusDeployment(k8s)
	svc := CreatePrometheusService(k8s)

	return roles, depl, svc
}

// DeletePrometheus deletes prometheus roles, configMap, deployment and service
func DeletePrometheus(k8s *K8s, roles []nsmrbac.Role, depl *appsv1.Deployment, svc *v1.Service) {
	_, err := k8s.DeleteRoles(roles)
	k8s.g.Expect(err).To(BeNil())

	err = k8s.DeleteService(svc, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())

	err = k8s.DeleteDeployment(depl, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())
}

// CreatePrometheusClusterRoles creates prometheus roles
func CreatePrometheusClusterRoles(k8s *K8s) []nsmrbac.Role {
	promRoles, err := k8s.CreateRoles("prometheus", "prometheus_binding")
	k8s.g.Expect(err).To(BeNil())

	return promRoles
}

// CreatePrometheusConfigMap creates prometheus configMap
func CreatePrometheusConfigMap(k8s *K8s) *v1.ConfigMap {
	cfgmap, err := k8s.CreateConfigMap(pods.PrometheusConfigMap(k8s.GetK8sNamespace()))
	k8s.g.Expect(err).To(BeNil())

	return cfgmap
}

// CreatePrometheusDeployment creates prometheus deployment
func CreatePrometheusDeployment(k8s *K8s) *appsv1.Deployment {
	deployment := pods.PrometheusDeployment(k8s.GetK8sNamespace())

	promDepl, err := k8s.CreateDeployment(deployment, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())

	return promDepl
}

// CreatePrometheusService creates prometheus service
func CreatePrometheusService(k8s *K8s) *v1.Service {
	service := pods.PrometheusService(k8s.GetK8sNamespace())

	promSvc, err := k8s.CreateService(service, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())

	return promSvc
}

// DeployCrossConnectMonitor deploys crossconnect-monitor
func DeployCrossConnectMonitor(k8s *K8s, image string) (*appsv1.Deployment, *v1.Service) {
	depl := CreateCrossConnectMonitorDeployment(k8s, image)
	svc := CreateCrossConnectMonitorService(k8s)

	return depl, svc
}

// DeleteCrossConnectMonitor deletes crossconnect-monitor deployment
func DeleteCrossConnectMonitor(k8s *K8s, depl *appsv1.Deployment, svc *v1.Service) {
	err := k8s.DeleteService(svc, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())

	err = k8s.DeleteDeployment(depl, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())
}

// CreateCrossConnectMonitorDeployment creates crossconnect-monitor deployment
func CreateCrossConnectMonitorDeployment(k8s *K8s, image string) *appsv1.Deployment {
	deployment := pods.CrossConnectMonitorDeployment(k8s.GetK8sNamespace(), image)

	ccDepl, err := k8s.CreateDeployment(deployment, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())

	return ccDepl
}

// CreateCrossConnectMonitorService creates crossconnect-monitor service
func CreateCrossConnectMonitorService(k8s *K8s) *v1.Service {
	service := pods.CrossConnectMonitorService(k8s.GetK8sNamespace())

	ccSvc, err := k8s.CreateService(service, k8s.GetK8sNamespace())
	k8s.g.Expect(err).To(BeNil())

	return ccSvc
}
