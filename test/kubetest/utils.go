package kubetest

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/networkservicemesh/networkservicemesh/dataplane/vppagent/pkg/vppagent"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"

	"k8s.io/client-go/util/cert"

	nsmd2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	arv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type NodeConf struct {
	Nsmd      *v1.Pod
	Dataplane *v1.Pod
	Node      *v1.Node
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
				dataplanePod = pods.VPPDataplanePodConfig(dataplaneName, node, DefaultDataplaneVariables())
			} else {
				conf[i].Namespace = namespace
				if conf[i].Nsmd == pods.NSMgrContainerDebug || conf[i].NsmdK8s == pods.NSMgrContainerDebug || conf[i].NsmdP == pods.NSMgrContainerDebug {
					debug = true
				}
				corePod = pods.NSMgrPodWithConfig(nsmdName, node, conf[i])
				dataplanePod = pods.VPPDataplanePodConfig(dataplaneName, node, conf[i].DataplaneVariables)
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
		k8s.WaitLogsContains(nsmd, "nsmd", "NSM gRPC API Server: [::]:5001 is operational", timeout)
		k8s.WaitLogsContains(nsmd, "nsmdp", "nsmdp: successfully started", timeout)
		k8s.WaitLogsContains(nsmd, "nsmd-k8s", "nsmd-k8s initialized and waiting for connection", timeout)
	})
	if len(failures) > 0 {
		printNSMDLogs(k8s, nsmd, 0)
		printDataplaneLogs(k8s, dataplane, 0)
	}
	err = nil
	return
}
// DeployVppAgentICMP - Setup VPP Agent based ICMP responder NSE
func DeployVppAgentICMP(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployICMP(k8s, node, name, timeout, pods.VppagentICMPResponderPod(name, node,
		defaultICMPEnv(),
	))
}

// DeployICMP - etup ICMP responder NSE
func DeployICMP(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployICMP(k8s, node, name, timeout, pods.TestNSEPod(name, node,
		defaultICMPEnv(), defaultICMPCommand(),
	))
}

// DeployICMPWithConfig - Setup ICMP responder NSE with parameters
func DeployICMPWithConfig(k8s *K8s, node *v1.Node, name string, timeout time.Duration, gracePeriod int64) *v1.Pod {
	pod := pods.TestNSEPod(name, node,
		defaultICMPEnv(), defaultICMPCommand(),
	)
	pod.Spec.TerminationGracePeriodSeconds = &gracePeriod
	return deployICMP(k8s, node, name, timeout, pod)
}

// DeployDirtyNSE - Setup ICMP responder NSE with dirty flag set
func DeployDirtyNSE(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployDirtyNSE(k8s, node, name, timeout, pods.TestNSEPod(name, node,
		defaultDirtyNSEEnv(), defaultDirtyNSECommand(),
	))
}

// DeployNeighborNSE deploys icmp with flag -neighbors
func DeployNeighborNSE(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployICMP(k8s, node, name, timeout, pods.TestNSEPod(name, node,
		defaultICMPEnv(), defaultNeighborNSECommand(),
	))
}

// DeployNSC - Setup Default Client
func DeployNSC(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, node, name, "nsc", timeout, pods.NSCPod(name, node,
		defaultNSCEnv()))
}

// DeployNSCWebhook - Setup Default Client with webhook
func DeployNSCWebhook(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, node, name, "nsm-init-container", timeout, pods.NSCPodWebhook(name, node))
}

// DeployVppAgentNSC - Setup Default VPP Based Client
func DeployVppAgentNSC(k8s *K8s, node *v1.Node, name string, timeout time.Duration) *v1.Pod {
	return deployNSC(k8s, node, name, "vppagent-nsc", timeout, pods.VppagentNSC(name, node, defaultNSCEnv()))
}

func defaultICMPEnv() map[string]string {
	return map[string]string{
		"ADVERTISE_NSE_NAME":   "icmp-responder",
		"ADVERTISE_NSE_LABELS": "app=icmp",
		"IP_ADDRESS":           "172.16.1.0/24",
	}
}

func defaultICMPCommand() []string {
	return []string{"/bin/icmp-responder-nse", "-routes"}
}

func defaultDirtyNSEEnv() map[string]string {
	return map[string]string{
		"ADVERTISE_NSE_NAME":   "dirty",
		"ADVERTISE_NSE_LABELS": "app=dirty",
		"IP_ADDRESS":           "10.30.1.0/24",
	}
}

func defaultDirtyNSECommand() []string {
	return []string{"/bin/icmp-responder-nse", "-dirty"}
}

func defaultNeighborNSECommand() []string {
	return []string{"/bin/icmp-responder-nse", "-neighbors"}
}

func defaultNSCEnv() map[string]string {
	return map[string]string{
		"OUTGOING_NSC_LABELS": "app=icmp",
		"OUTGOING_NSC_NAME":   "icmp-responder",
	}
}

func deployICMP(k8s *K8s, node *v1.Node, name string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()

	logrus.Infof("Starting ICMP Responder NSE on node: %s", node.Name)
	icmp := k8s.CreatePod(template)
	Expect(icmp.Name).To(Equal(name))

	k8s.WaitLogsContains(icmp, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", timeout)

	logrus.Printf("ICMP Responder %v started done: %v", name, time.Since(startTime))
	return icmp
}

func deployDirtyNSE(k8s *K8s, node *v1.Node, name string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()

	logrus.Infof("Starting dirty NSE on node: %s", node.Name)
	dirty := k8s.CreatePod(template)
	Expect(dirty.Name).To(Equal(name))

	k8s.WaitLogsContains(dirty, "", "NSE: channel has been successfully advertised, waiting for connection from NSM...", timeout)

	logrus.Printf("Dirty NSE %v started done: %v", name, time.Since(startTime))
	return dirty
}

func deployNSC(k8s *K8s, node *v1.Node, name, container string, timeout time.Duration, template *v1.Pod) *v1.Pod {
	startTime := time.Now()
	Expect(template).ShouldNot(BeNil())

	logrus.Infof("Starting NSC %s on node: %s", name, node.Name)

	nsc := k8s.CreatePod(template)

	Expect(nsc.Name).To(Equal(name))
	k8s.WaitLogsContains(nsc, container, "nsm client: initialization is completed successfully", timeout)

	logrus.Printf("NSC started done: %v", time.Since(startTime))
	return nsc
}

// DeployAdmissionWebhook - Setup Admission Webhook
func DeployAdmissionWebhook(k8s *K8s, name, image, namespace string) (*arv1beta1.MutatingWebhookConfiguration, *appsv1.Deployment, *v1.Service) {
	_, caCert := CreateAdmissionWebhookSecret(k8s, name, namespace)
	awc := CreateMutatingWebhookConfiguration(k8s, caCert, name, namespace)

	awDeployment := CreateAdmissionWebhookDeployment(k8s, name, image, namespace)
	awService := CreateAdmissionWebhookService(k8s, name, namespace)

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
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "nsm-admission-webhook",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nsm-admission-webhook"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nsm-admission-webhook",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            name,
							Image:           image,
							ImagePullPolicy: v1.PullIfNotPresent,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "webhook-certs",
									MountPath: "/etc/webhook/certs",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "webhook-certs",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName: "nsm-admission-webhook-certs",
								},
							},
						},
					},
				},
			},
		},
	}
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

// PrintLogs - Print deployed pod logs
func PrintLogs(k8s *K8s, nodesSetup []*NodeConf) {
	for k := 0; k < len(nodesSetup); k++ {
		nsmdPod := nodesSetup[k].Nsmd
		printNSMDLogs(k8s, nsmdPod, k)

		printDataplaneLogs(k8s, nodesSetup[k].Dataplane, k)
	}
}

func printDataplaneLogs(k8s *K8s, dataplane *v1.Pod, k int) {
	dataplaneLogs, _ := k8s.GetLogs(dataplane, "")
	logrus.Errorf("===================== Dataplane %d output since test is failing %v\n=====================", k, dataplaneLogs)
}

func printNSMDLogs(k8s *K8s, nsmdPod *v1.Pod, k int) {
	nsmdUpdatedPod, err := k8s.GetPod(nsmdPod)
	if err != nil {
		logrus.Errorf("Failed to update POD details %v", err)
		return
	}
	for _, cs := range nsmdUpdatedPod.Status.ContainerStatuses {
		containerLogs, _ := k8s.GetLogs(nsmdPod, cs.Name)
		if cs.RestartCount > 0 {
			prevLogs, _ := k8s.GetLogsWithOptions(nsmdPod, &v1.PodLogOptions{
				Container: cs.Name,
				Previous:  true,
			})
			logrus.Errorf("===================== %s %d previous output since test is failing %v\n=====================", strings.ToUpper(cs.Name), k, prevLogs)
		}
		logrus.Errorf("===================== %s %d output since test is failing %v\n=====================", strings.ToUpper(cs.Name), k, containerLogs)
	}

}

// NSCCheckInfo - Structure to hold client ping information
type NSCCheckInfo struct {
	ipResponse    string
	routeResponse string
	pingResponse  string
	errOut        string
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
	return checkNSCConfig(k8s, nscPodNode, "172.16.1.1", "172.16.1.2")
}
// CheckVppAgentNSC - Perform check of VPP based agent operations.
func CheckVppAgentNSC(k8s *K8s, nscPodNode *v1.Pod) *NSCCheckInfo {
	return checkVppAgentNSCConfig(k8s, nscPodNode, "172.16.1.1")
}
func checkNSCConfig(k8s *K8s, nscPodNode *v1.Pod, checkIP, pingIP string) *NSCCheckInfo {
	var err error
	info := &NSCCheckInfo{}
	info.ipResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "addr")
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	logrus.Printf("NSC IP status Ok")

	Expect(strings.Contains(info.ipResponse, checkIP)).To(Equal(true))
	Expect(strings.Contains(info.ipResponse, "nsm")).To(Equal(true))

	info.routeResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ip", "route")
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	logrus.Printf("NSC Route status, Ok")

	Expect(strings.Contains(info.routeResponse, "8.8.8.8")).To(Equal(true))
	Expect(strings.Contains(info.routeResponse, "nsm")).To(Equal(true))

	info.pingResponse, info.errOut, err = k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "ping", pingIP, "-A", "-c", "5")
	Expect(err).To(BeNil())
	Expect(info.errOut).To(Equal(""))
	Expect(strings.Contains(info.pingResponse, "100% packet loss")).To(Equal(false))

	logrus.Printf("NSC Ping is success:%s", info.pingResponse)
	return info
}

func checkVppAgentNSCConfig(k8s *K8s, nscPodNode *v1.Pod, checkIP string) *NSCCheckInfo {
	info := &NSCCheckInfo{}
	response, errOut, _ := k8s.Exec(nscPodNode, nscPodNode.Spec.Containers[0].Name, "vppctl", "show int addr")
	if strings.Contains(response, checkIP) {
		info.ipResponse = response
		info.errOut = errOut
	}
	Expect(info.ipResponse).ShouldNot(Equal(""))
	Expect(info.errOut).Should(Equal(""))
	logrus.Printf("NSC IP status Ok")
	Expect(true, IsVppAgentNsePinged(k8s, nscPodNode))

	return info
}

// IsBrokeTestsEnabled - Check if broken tests are enabled
func IsBrokeTestsEnabled() bool {
	_, ok := os.LookupEnv("BROKEN_TESTS_ENABLED")
	return ok
}

// DefaultDataplaneVariables - Default variabels for dataplane deployment
func DefaultDataplaneVariables() map[string]string {
	return map[string]string{
		vppagent.DataplaneMetricsCollectorEnabledKey: "false",
	}
}
// GetVppAgentNSEAddr - GetVppAgentNSEAddr - Return vpp agent NSE address
func GetVppAgentNSEAddr(k8s *K8s, nsc *v1.Pod) (net.IP, error) {
	return getNSEAddr(k8s, nsc, parseVppAgentAddr, "vppctl", "show int addr")
}

func parseVppAgentAddr(ipReponse string) (string, error) {
	spitedResponse := strings.Split(ipReponse, "L3 ")
	if len(spitedResponse) < 2 {
		return "", errors.New(fmt.Sprintf("bad ip response %v", ipReponse))
	}
	return spitedResponse[1], nil
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

// IsVppAgentNsePinged - Checvk if vpp agent NSE is pinable
func IsVppAgentNsePinged(k8s *K8s, from *v1.Pod) (result bool) {
	nseIp, err := GetVppAgentNSEAddr(k8s, from)
	Expect(err).Should(BeNil())
	logrus.Infof("%v trying vppctl ping to %v", from.Name, nseIp)
	response, _, _ := k8s.Exec(from, from.Spec.Containers[0].Name, "vppctl", "ping", nseIp.String())
	logrus.Infof("ping result: %s", response)
	if strings.TrimSpace(response) != "" && !strings.Contains(response, "100% packet loss") && !strings.Contains(response, "Fail") {
		result = true
		logrus.Info("Ping successful")
	}

	return result
}

// IsNsePinged - Check if NSE is pinged
func IsNsePinged(k8s *K8s, from *v1.Pod) (result bool) {
	nseIp, err := getNSEAddr(k8s, from, parseAddr, "ip", "addr")
	Expect(err).Should(BeNil())
	logrus.Infof("%v trying ping to %v", from.Name, nseIp)
	response, _, _ := k8s.Exec(from, from.Spec.Containers[0].Name, "ping", nseIp.String(), "-A", "-c", "4")
	logrus.Infof("ping result: %s", response)
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
		PrintLogs(k8s, nodesSetup)
		nscInfo.PrintLogs()

		t.Fail()
	}
}

// FailLogger prints logs from containers in case of fail or panic
func FailLogger(k8s *K8s, nodesSetup []*NodeConf, t *testing.T) {
	if r := recover(); r != nil {
		PrintLogs(k8s, nodesSetup)
		panic(r)
	}

	if t.Failed() {
		PrintLogs(k8s, nodesSetup)
	}

	return
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
