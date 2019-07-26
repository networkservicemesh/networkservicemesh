package kubetest

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	arv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/cert"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	nsmd2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
)

type NodeConf struct {
	Nsmd      *v1.Pod
	Dataplane *v1.Pod
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

// SetupNodes - Setup NSMgr and Dataplane for particular number of nodes in cluster
func SetupNodes(k8s *K8s, nodesCount int, timeout time.Duration) ([]*NodeConf, error) {
	return SetupNodesConfig(k8s, nodesCount, timeout, []*pods.NSMgrPodConfig{}, k8s.GetK8sNamespace())
}

// SetupNodesConfig - Setup NSMgr and Dataplane for particular number of nodes in cluster
func SetupNodesConfig(k8s *K8s, nodesCount int, timeout time.Duration, conf []*pods.NSMgrPodConfig, namespace string) ([]*NodeConf, error) {
	nodes := k8s.GetNodesWait(nodesCount, timeout)
	Expect(len(nodes) >= nodesCount).To(Equal(true),
		"At least one Kubernetes node is required for this test")

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
			dataplaneName := fmt.Sprintf("nsmd-dataplane-%s", node.Name)
			var corePod *v1.Pod
			var dataplanePod *v1.Pod
			debug := false
			if i >= len(conf) {
				corePod = pods.NSMgrPod(nsmdName, node, k8s.GetK8sNamespace())
				dataplanePod = pods.ForwardingPlaneWithConfig(dataplaneName, node, DefaultDataplaneVariables(k8s.GetForwardingPlane()), k8s.GetForwardingPlane())
			} else {
				conf[i].Namespace = namespace
				if conf[i].Nsmd == pods.NSMgrContainerDebug || conf[i].NsmdK8s == pods.NSMgrContainerDebug || conf[i].NsmdP == pods.NSMgrContainerDebug {
					debug = true
				}
				corePod = pods.NSMgrPodWithConfig(nsmdName, node, conf[i])
				dataplanePod = pods.ForwardingPlaneWithConfig(dataplaneName, node, conf[i].DataplaneVariables, k8s.GetForwardingPlane())
			}
			corePods, err := k8s.CreatePodsRaw(PodStartTimeout, true, corePod, dataplanePod)
			if err != nil {
				logrus.Errorf("Failed to Started NSMgr/Dataplane: %v on node %s %v", time.Since(startTime), node.Name, err)
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
			nsmd, dataplane, err := deployNSMgrAndDataplane(k8s, corePods, timeout)
			if err != nil {
				logrus.Errorf("Failed to Started NSMgr/Dataplane: %v on node %s %v", time.Since(startTime), node.Name, err)
				resultError = err
				return
			}
			logrus.Printf("Started NSMgr/Dataplane: %v on node %s", time.Since(startTime), node.Name)
			confs[i] = &NodeConf{
				Nsmd:      nsmd,
				Dataplane: dataplane,
				Node:      &nodes[i],
			}
		}()
	}
	wg.Wait()
	return confs, resultError
}

func deployNSMgrAndDataplane(k8s *K8s, corePods []*v1.Pod, timeout time.Duration) (nsmd, dataplane *v1.Pod, err error) {
	for _, pod := range corePods {
		if !k8s.IsPodReady(pod) {
			return nil, nil, fmt.Errorf("Pod %v is not ready...", pod.Name)
		}
	}
	nsmd = corePods[0]
	dataplane = corePods[1]

	Expect(nsmd.Name).To(Equal(corePods[0].Name))
	Expect(dataplane.Name).To(Equal(corePods[1].Name))

	failures := InterceptGomegaFailures(func() {
		k8s.WaitLogsContains(dataplane, "", "Sending MonitorMechanisms update", timeout)
		_ = k8s.WaitLogsContainsRegex(nsmd, "nsmd", "NSM gRPC API Server: .* is operational", timeout)
		k8s.WaitLogsContains(nsmd, "nsmdp", "nsmdp: successfully started", timeout)
		k8s.WaitLogsContains(nsmd, "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", timeout)
	})
	if len(failures) > 0 {
		showLogs(k8s, nil)
	}
	err = nil
	return
}

// DeployICMP deploys 'icmp-responder-nse' pod with '-routes' flag set
func DeployICMP(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployICMP(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, icmpCommand(false, false, true, false), node, defaultICMPEnv(k8s.UseIPv6())),
	)
}

// DeployICMPWithConfig deploys 'icmp-responder-nse' pod with '-routes' flag set and given grace period
func DeployICMPWithConfig(k8s *K8s, node *v1.Node, name string, timeout time.Duration, gracePeriod int64) *v1.Pod {
	pod := pods.TestCommonPod(name, icmpCommand(false, false, true, false), node, defaultICMPEnv(k8s.UseIPv6()))
	pod.Spec.TerminationGracePeriodSeconds = &gracePeriod
	return deployICMP(k8s, nodeName(node), name, timeout, pod)
}

// DeployDirtyICMP deploys 'icmp-responder-nse' pod with '-dirty' flag set
func DeployDirtyICMP(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployDirtyNSE(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, icmpCommand(true, false, false, false), node, defaultICMPEnv(k8s.UseIPv6())),
	)
}

// DeployNeighborNSE deploys 'icmp-responder-nse' pod with '-neighbors' flag set
func DeployNeighborNSE(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployICMP(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, icmpCommand(false, true, false, false), node, defaultICMPEnv(k8s.UseIPv6())),
	)
}

// DeployUpdatingNSE deploys 'icmp-responder-nse' pod with '-update' flag set
func DeployUpdatingNSE(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployICMP(k8s, nodeName(node), name, timeout,
		pods.TestCommonPod(name, icmpCommand(false, false, false, true), node, defaultICMPEnv(k8s.UseIPv6())),
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

// DeployNSCWebhook - Setup Default Client with webhook
func DeployNSCWebhook(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, nodeName(node), name, "nsm-init-container", timeout,
		pods.NSCPodWebhook(name, node),
	)
}

// DeployMonitoringNSC deploys 'monitoring-nsc' pod
func DeployMonitoringNSC(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, nodeName(node), name, "monitoring-nsc", timeout,
		pods.TestCommonPod(name, []string{"/bin/monitoring-nsc"}, node, defaultNSCEnv()),
	)
}

// NoHealNSMgrPodConfig returns config for NSMgr. The config has properties for disabling healing for nse
func NoHealNSMgrPodConfig(k8s *K8s) []*pods.NSMgrPodConfig {
	return []*pods.NSMgrPodConfig{
		noHealNSMgrPodConfig(k8s),
		noHealNSMgrPodConfig(k8s),
	}
}

func icmpCommand(dirty, neighbors, routes, update bool) []string {
	command := []string{"/bin/icmp-responder-nse"}

	if dirty {
		command = append(command, "-dirty")
	}
	if neighbors {
		command = append(command, "-neighbors")
	}
	if routes {
		command = append(command, "-routes")
	}
	if update {
		command = append(command, "-update")
	}

	return command
}

func defaultICMPEnv(useIPv6 bool) map[string]string {
	if !useIPv6 {
		return map[string]string{
			"ADVERTISE_NSE_NAME":   "icmp-responder",
			"ADVERTISE_NSE_LABELS": "app=icmp",
			"IP_ADDRESS":           "172.16.1.0/24",
		}
	}
	return map[string]string{
		"ADVERTISE_NSE_NAME":   "icmp-responder",
		"ADVERTISE_NSE_LABELS": "app=icmp",
		"IP_ADDRESS":           "100::/64",
	}
}

func defaultNSCEnv() map[string]string {
	return map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   "icmp-responder",
	}
}

func noHealNSMgrPodConfig(k8s *K8s) *pods.NSMgrPodConfig {
	return &pods.NSMgrPodConfig{
		Variables: map[string]string{
			nsmd2.NsmdDeleteLocalRegistry: "true", // Do not use local registry restore for clients/NSEs
			nsm.NsmdHealDSTWaitTimeout:    "1",    // 1 second
			nsm.NsmdHealEnabled:           "true",
		},
		Namespace:          k8s.GetK8sNamespace(),
		DataplaneVariables: DefaultDataplaneVariables(k8s.GetForwardingPlane()),
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
	Expect(icmp.Name).To(Equal(name))

	k8s.WaitLogsContains(icmp, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", timeout)

	logrus.Printf("ICMP Responder %v started done: %v", name, time.Since(startTime))
	return icmp
}

func deployDirtyNSE(k8s *K8s, nodeName, name string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()

	logrus.Infof("Starting dirty NSE on node: %s", nodeName)
	dirty := k8s.CreatePod(template)
	Expect(dirty.Name).To(Equal(name))

	k8s.WaitLogsContains(dirty, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", timeout)

	logrus.Printf("Dirty NSE %v started done: %v", name, time.Since(startTime))
	return dirty
}

func deployNSC(k8s *K8s, nodeName, name, container string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()
	Expect(template).ShouldNot(BeNil())

	logrus.Infof("Starting NSC %s on node: %s", name, nodeName)

	nsc := k8s.CreatePod(template)

	Expect(nsc.Name).To(Equal(name))
	k8s.WaitLogsContains(nsc, container, "nsm client: initialization is completed successfully", timeout)

	logrus.Printf("NSC started done: %v", time.Since(startTime))
	return nsc
}

// DeployAdmissionWebhook - Setup Admission Webhook
func DeployAdmissionWebhook(k8s *K8s, name, image, namespace string, timeout time.Duration) (*arv1beta1.MutatingWebhookConfiguration, *appsv1.Deployment, *v1.Service) {
	_, caCert := CreateAdmissionWebhookSecret(k8s, name, namespace)
	awc := CreateMutatingWebhookConfiguration(k8s, caCert, name, namespace)

	awDeployment := CreateAdmissionWebhookDeployment(k8s, name, image, namespace)
	awService := CreateAdmissionWebhookService(k8s, name, namespace)

	admissionWebhookPod := waitWebhookPod(k8s, awDeployment.Name, timeout)
	Expect(admissionWebhookPod).ShouldNot(BeNil())
	k8s.WaitLogsContains(admissionWebhookPod, admissionWebhookPod.Spec.Containers[0].Name, "Server started", timeout)
	return awc, awDeployment, awService
}

// DeleteAdmissionWebhook - Delete admission webhook
func DeleteAdmissionWebhook(k8s *K8s, secretName string,
	awc *arv1beta1.MutatingWebhookConfiguration, awDeployment *appsv1.Deployment, awService *v1.Service, namespace string) {

	err := k8s.DeleteService(awService, namespace)
	Expect(err).To(BeNil())

	err = k8s.DeleteDeployment(awDeployment, namespace)
	Expect(err).To(BeNil())

	err = k8s.DeleteMutatingWebhookConfiguration(awc)
	Expect(err).To(BeNil())

	err = k8s.DeleteSecret(secretName, namespace)
	Expect(err).To(BeNil())
}

// CreateAdmissionWebhookSecret - Create admission webhook secret
func CreateAdmissionWebhookSecret(k8s *K8s, name, namespace string) (*v1.Secret, []byte) {

	caCertSpec := &cert.Config{
		CommonName: "admission-controller-ca",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	caCert, caKey, err := pkiutil.NewCertificateAuthority(caCertSpec)
	Expect(err).To(BeNil())

	certSpec := &cert.Config{
		CommonName: name + "-svc",
		AltNames: cert.AltNames{
			DNSNames: []string{
				name + "-svc." + namespace,
				name + "-svc." + namespace + ".svc",
			},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	cer, key, err := pkiutil.NewCertAndKey(caCert, caKey, certSpec)
	Expect(err).To(BeNil())

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	keyPem := pem.EncodeToMemory(block)

	block = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cer.Raw,
	}
	certPem := pem.EncodeToMemory(block)

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-certs",
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key.pem":  keyPem,
			"cert.pem": certPem,
		},
	}

	awSecret, err := k8s.CreateSecret(secret, namespace)
	Expect(err).To(BeNil())

	block = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCert.Raw,
	}
	caCertPem := pem.EncodeToMemory(block)

	return awSecret, caCertPem
}

// CreateMutatingWebhookConfiguration - Setup Mutating webhook configuration
func CreateMutatingWebhookConfiguration(k8s *K8s, certPem []byte, name, namespace string) *arv1beta1.MutatingWebhookConfiguration {
	servicePath := "/mutate"

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
		Webhooks: []arv1beta1.Webhook{
			{
				Name: "admission-webhook.networkservicemesh.io",
				ClientConfig: arv1beta1.WebhookClientConfig{
					Service: &arv1beta1.ServiceReference{
						Namespace: namespace,
						Name:      name + "-svc",
						Path:      &servicePath,
					},
					CABundle: certPem,
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
	Expect(err).To(BeNil())

	return awc
}

// CreateAdmissionWebhookDeployment - Setup Admission Webhook deoloyment
func CreateAdmissionWebhookDeployment(k8s *K8s, name, image, namespace string) *appsv1.Deployment {
	deployment := pods.AdmissionWebhookDeployment(name, image)

	awDeployment, err := k8s.CreateDeployment(deployment, namespace)
	Expect(err).To(BeNil())

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
	Expect(err).To(BeNil())

	return awService
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
			list := k8s.ListPods()
			for i := 0; i < len(list); i++ {
				p := &list[i]
				if strings.Contains(p.Name, name) {
					result, err := blockUntilPodReady(k8s.clientset, timeout, p)
					Expect(err).Should(BeNil())
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
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	Expect(strings.Contains(info.ipResponse, checkIP)).To(Equal(true))
	Expect(strings.Contains(info.ipResponse, "nsm")).To(Equal(true))

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
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	Expect(strings.Contains(info.routeResponse, publicDNSAddress)).To(Equal(true))
	Expect(strings.Contains(info.routeResponse, "nsm")).To(Equal(true))

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
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))

	pingNOK := strings.Contains(info.pingResponse, "100% packet loss")
	Expect(pingNOK).To(Equal(false))
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

// HealNscChecker checks that heal worked properly
func HealNscChecker(k8s *K8s, nscPod *v1.Pod) *NSCCheckInfo {
	const attempts = 10
	success := false
	var rv *NSCCheckInfo
	for i := 0; i < attempts; i++ {
		info := &NSCCheckInfo{}
		info.pingResponse = pingNse(k8s, nscPod)

		if !strings.Contains(info.pingResponse, "100% packet loss") {
			success = true
			rv = info
			break
		}
		<-time.After(time.Second)
	}
	Expect(success).To(BeTrue())
	return rv
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
		return nil, fmt.Errorf("exec [%v] returned empty response", showIPCommand)
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
	Expect(err).Should(BeNil())
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

// PrintErrors - Print errors for system NSMgr pods
func PrintErrors(failures []string, k8s *K8s, nodesSetup []*NodeConf, nscInfo *NSCCheckInfo, t *testing.T) {
	if len(failures) > 0 {
		logrus.Errorf("Failures: %v", failures)
		showLogs(k8s, t)
		nscInfo.PrintLogs()

		t.Fail()
	}
}

// ServiceRegistryAt creates new service registry on 5000 port
func ServiceRegistryAt(k8s *K8s, nsmgr *v1.Pod) (serviceregistry.ServiceRegistry, func()) {
	fwd, err := k8s.NewPortForwarder(nsmgr, 5000)
	Expect(err).To(BeNil())

	err = fwd.Start()
	Expect(err).To(BeNil())

	sr := nsmd2.NewServiceRegistryAt(fmt.Sprintf("localhost:%d", fwd.ListenPort))
	return sr, fwd.Stop
}

// PrepareRegistryClients prepare nse and nsm registry clients
func PrepareRegistryClients(k8s *K8s, nsmd *v1.Pod) (registry.NetworkServiceRegistryClient, registry.NsmRegistryClient, func()) {
	serviceRegistry, closeFunc := ServiceRegistryAt(k8s, nsmd)

	nseRegistryClient, err := serviceRegistry.NseRegistryClient()
	Expect(err).To(BeNil())

	nsmRegistryClient, err := serviceRegistry.NsmRegistryClient()
	Expect(err).To(BeNil())

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

	Expect(err).To(BeNil())
	Expect(len(nses)).To(Equal(countExpected), fmt.Sprint(nses))
}
