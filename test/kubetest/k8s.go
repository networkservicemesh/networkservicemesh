package kubetest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/networkservicemesh/networkservicemesh/test/kubetest/artifact"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"
	jaeger_env "github.com/networkservicemesh/networkservicemesh/test/kubetest/jaeger"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	arv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	pkgerrors "github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"
	"github.com/networkservicemesh/networkservicemesh/sdk/prefix_pool"
	"github.com/networkservicemesh/networkservicemesh/test/kubetest/pods"
	nsmrbac "github.com/networkservicemesh/networkservicemesh/test/kubetest/rbac"
	"github.com/networkservicemesh/networkservicemesh/utils"
)

type ClearOption int32

const (
	NoClear ClearOption = iota
	DefaultClear
	ReuseNSMResources
)

const (
	// PodStartTimeout - Default pod startup time
	PodStartTimeout    = 3 * time.Minute
	podDeleteTimeout   = 15 * time.Second
	podExecTimeout     = 1 * time.Minute
	podGetLogTimeout   = 1 * time.Minute
	accountWaitTimeout = 1 * time.Minute

	//NetworkPluginCNIFailure - pattern to check for CNI issue, pattern required to try redeploy pod
	NetworkPluginCNIFailure = "NetworkPlugin cni failed to set up pod"
)

const (
	envUseIPv6        = "USE_IPV6"
	envUseIPv6Default = false
)

type PodDeployResult struct {
	pod *v1.Pod
	err error
}

func waitTimeout(logPrefix string, wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return true
	case <-time.After(timeout):
		logrus.Errorf("%v Timeout in waitTimeout", logPrefix)
		return false
	}
}

func (k8s *K8s) LookupPodsByName(names ...string) []*v1.Pod {
	var result []*v1.Pod
	pods := k8s.ListPods()

	for i := range pods {
		pod := &pods[i]
		for _, n := range names {
			if strings.Contains(pod.Name, n) {
				result = append(result, pod)
				break
			}
		}
	}
	return result
}

func (k8s *K8s) LookupServiceAccounts(serviceAccounts ...string) []*v1.ServiceAccount {
	exists, err := k8s.clientset.CoreV1().ServiceAccounts(k8s.namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil
	}
	var result []*v1.ServiceAccount
	for i := range exists.Items {
		acc := &exists.Items[i]
		for _, sa := range serviceAccounts {
			if strings.Contains(acc.Name, sa) {
				result = append(result, acc)
			}
		}
	}
	return result
}

func (k8s *K8s) createAndBlock(client kubernetes.Interface, namespace string, timeout time.Duration, pods ...*v1.Pod) []*PodDeployResult {
	var wg sync.WaitGroup

	resultChan := make(chan *PodDeployResult, len(pods))

	for _, pod := range pods {
		wg.Add(1)
		go func(pod *v1.Pod) {
			defer wg.Done()

			var err error
			createdPod, err := client.CoreV1().Pods(namespace).Create(pod)

			// We need to have non nil pod in any case.
			if createdPod != nil && createdPod.Name != "" {
				pod = createdPod
			}
			if err != nil {
				logrus.Errorf("Failed to create pod. Cause: %v pod: %v", err, pod)
				k8s.DescribePod(pod)
				resultChan <- &PodDeployResult{pod, err}
				return
			}
			pod, err = k8s.blockUntilPodReady(client, timeout, pod)
			if err != nil {
				logrus.Errorf("blockUntilPodReady failed. Cause: %v pod: %v", err, pod.Name)
				k8s.DescribePod(pod)
				resultChan <- &PodDeployResult{pod, err}
				return
			}

			// Let's fetch more information about pod created

			updated_pod, err := client.CoreV1().Pods(namespace).Get(pod.Name, metaV1.GetOptions{})
			if err != nil {
				logrus.Errorf("Failed to Get endpoint. Cause: %v pod: %v", err, pod.Name)
				resultChan <- &PodDeployResult{pod, err}
				return
			}
			resultChan <- &PodDeployResult{updated_pod, nil}
		}(pod)
	}

	podNames := ""
	for _, pod := range pods {
		if len(podNames) > 0 {
			podNames += ", "
		}
		podNames += pod.Name
	}
	if !waitTimeout(fmt.Sprintf("createAndBlock with pods: %v", podNames), &wg, timeout) {
		logrus.Errorf("Failed to deploy pod, trying to get any information")
		results := []*PodDeployResult{}
		for _, p := range pods {
			pod, err := client.CoreV1().Pods(namespace).Get(p.Name, metaV1.GetOptions{})
			if err != nil {
				logrus.Errorf("Failed to get pod information: %v", err)
			}
			if pod != nil {
				logrus.Infof("Pod container information: %v", pod.Name)
				for _, cs := range pod.Status.InitContainerStatuses {
					logs, _ := k8s.GetLogs(pod, cs.Name)
					logrus.Infof("Pod %v: container: %v status: %v: Logs: %v", pod.Name, cs.Name, cs.Ready, logs)
				}
				for _, cs := range pod.Status.ContainerStatuses {
					logs, _ := k8s.GetLogs(pod, cs.Name)
					logrus.Infof("Pod %v: container: %v status: %v: Logs: %v", pod.Name, cs.Name, cs.Ready, logs)
				}
				logrus.Infof("pod %s object:\n>>>>>>>>>>>%v\n<<<<<<<<<<", pod.Name, prettyPrint(pod))
			}
			k8s.DescribePod(p)

			results = append(results, &PodDeployResult{
				err: errors.New("Failed to deploy pod"),
				pod: pod,
			})
			return results
		}
		return nil
	}

	results := make([]*PodDeployResult, len(pods))
	named := map[string]*PodDeployResult{}
	for i := 0; i < len(pods); i++ {
		pod := <-resultChan
		named[pod.pod.Name] = pod
	}
	for i := 0; i < len(pods); i++ {
		results[i] = named[pods[i].Name]
	}

	// We need to put pods in right order
	return results
}

func prettyPrint(value interface{}) string {
	res := value
	msg, jerr := json.Marshal(value)
	if jerr == nil {
		res = string(msg)
	}
	return fmt.Sprintf("%v", res)
}

func (k8s *K8s) blockUntilPodReady(client kubernetes.Interface, timeout time.Duration, sourcePod *v1.Pod) (*v1.Pod, error) {
	st := time.Now()
	infoPrinted := false
	lastPodNetworkCheck := time.Now()
	for {
		pod, err := client.CoreV1().Pods(sourcePod.Namespace).Get(sourcePod.Name, metaV1.GetOptions{})

		// To be sure we not loose pod information.
		if pod == nil {
			pod = sourcePod
		}
		if err != nil {
			return pod, err
		}

		// Check every 1 second for pod deploy network problems
		if time.Since(lastPodNetworkCheck) > 1*time.Second {
			if podErr := k8s.IsNetworkProblem(pod); podErr != nil {
				return pod, podErr
			}
			lastPodNetworkCheck = time.Now()
		}

		if pod != nil && pod.Status.Phase != v1.PodPending {
			break
		}

		if time.Since(st) > timeout/2 && !infoPrinted {
			logrus.Infof("Pod deploy half time passed: %v", pod.Name)
			infoPrinted = true
		}

		time.Sleep(time.Millisecond * time.Duration(50))
		if time.Since(st) > timeout {
			return pod, podTimeout(pod)
		}
	}

	// Check if we have event with deploy failure, let's report it.
	if podErr := k8s.IsNetworkProblem(sourcePod); podErr != nil {
		return sourcePod, podErr
	}

	watcher, err := client.CoreV1().Pods(sourcePod.Namespace).Watch(metaV1.SingleObject(metaV1.ObjectMeta{Name: sourcePod.Name}))

	if err != nil {
		return sourcePod, err
	}

	return k8s.waitPodStatus(watcher, sourcePod, client, timeout)
}

func (k8s *K8s) isNetworkProblemEvent(event *v1.Event) bool {
	return strings.Contains(event.Message, NetworkPluginCNIFailure)
}

func (k8s *K8s) waitPodStatus(watcher watch.Interface, sourcePod *v1.Pod, client kubernetes.Interface, timeout time.Duration) (*v1.Pod, error) {
	for {
		select {
		case _, ok := <-watcher.ResultChan():

			if !ok {
				return sourcePod, errors.New("Some error watching for pod status")
			}

			pod, err := client.CoreV1().Pods(sourcePod.Namespace).Get(sourcePod.Name, metaV1.GetOptions{})
			if err == nil {
				if isPodReady(pod) {
					watcher.Stop()
					return pod, nil
				}
			}
		case <-time.After(timeout):
			return sourcePod, podTimeout(sourcePod)
		}
	}
}

func podTimeout(pod *v1.Pod) error {
	return errors.Errorf("Timeout during waiting for pod change status for pod %s %v status: ", pod.Name, pod.Status.Conditions)
}

func isPodReady(pod *v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			// If one of containers is not yet ready, return false
			return false
		}
	}

	return true
}

func (k8s *K8s) waitForServiceAccountCreated(timeout time.Duration, serviceAccountName string) error {
	err := waitFor(accountWaitTimeout, func() bool {
		sa, getErr := k8s.clientset.CoreV1().ServiceAccounts(k8s.namespace).Get(serviceAccountName, metaV1.GetOptions{})
		if getErr != nil {
			logrus.Errorf("An error during get service account: %v, err: %v", sa.Name, getErr.Error())
			return false
		}
		logrus.Info(sa)
		return len(sa.Secrets) != 0
	})
	if err != nil {
		err = errors.Wrapf(err, "account: %v, secrets == 0", serviceAccountName)
	}
	return err
}

func waitFor(timeout time.Duration, condition func() bool) error {
	timeoutCh := time.After(timeout)
	for {
		select {
		case <-timeoutCh:
			return errors.New("time out for wait condition true")
		default:
			if condition() {
				return nil
			}
			<-time.After(100 * time.Millisecond)
		}
	}
}

func blockUntilPodWorking(client kubernetes.Interface, context context.Context, pod *v1.Pod) error {
	exists := make(chan error)
	go func() {
		for {
			_, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metaV1.GetOptions{})
			if err != nil {
				// Pod not found
				close(exists)
				break
			}
			<-time.After(time.Millisecond * time.Duration(50))
		}
	}()

	select {
	case <-context.Done():
		return podTimeout(pod)
	case err, ok := <-exists:
		if err != nil {
			return err
		}

		if ok {
			return errors.New("unintended")
		}

		return nil
	}
}

type K8s struct {
	artifactManager    artifact.Manager
	clientset          kubernetes.Interface
	versionedClientSet *versioned.Clientset
	podLock            sync.Mutex
	resourcesBehaviour ClearOption
	pods               []*v1.Pod
	config             *rest.Config
	roles              []nsmrbac.Role
	namespace          string
	apiServerHost      string
	useIPv6            bool
	forwardingPlane    string
	sa                 []string
	g                  *WithT
	cleanupFunc        func()
	startTime          metaV1.Time
}

type spanRecord struct {
	spanPod map[string]*v1.Pod
}

func (k8s *K8s) reportSpans() {
	if !jaeger.IsOpentracingEnabled() || jaeger_env.ReportSpans.GetBooleanOrDefault(false) {
		return
	}
	logrus.Infof("Finding spans")
	// We need to find all Reporting span and print uniq to console for analysis.
	pods := k8s.ListPods()
	spans := map[string]*spanRecord{}
	for i := 0; i < len(pods); i++ {
		pod := pods[i]
		for ci := 0; ci < len(pod.Spec.Containers); ci++ {
			c := pod.Spec.Containers[ci]
			k8s.findSpans(&pods[i], c, spans)
		}
		for ci := 0; ci < len(pod.Spec.InitContainers); ci++ {
			c := pod.Spec.Containers[ci]
			k8s.findSpans(&pods[i], c, spans)
		}
	}
	for spanID, span := range spans {
		var keys []string
		for k := range span.spanPod {
			keys = append(keys, k)
		}
		logrus.Infof("Span %v pods: %v", spanID, keys)
	}
}

func (k8s *K8s) findSpans(pod *v1.Pod, c v1.Container, spans map[string]*spanRecord) {
	content, err := k8s.GetFullLogs(pod, c.Name, false)
	if err == nil {
		lines := strings.Split(content, "\n")
		for _, l := range lines {
			pos := strings.Index(l, " Reporting span ")
			if pos > 0 {
				value := l[pos:]
				pos = strings.Index(value, ":")
				value = value[0:pos]
				if value != "" {
					podRecordID := fmt.Sprintf("%s:%s", pod.Name, c.Name)
					if span, ok := spans[value]; ok {
						span.spanPod[podRecordID] = pod
					} else {
						spans[value] = &spanRecord{
							spanPod: map[string]*v1.Pod{podRecordID: pod},
						}
					}
				}
			}
		}
	}
}

// ExtK8s - K8s ClientSet with nodes config
type ExtK8s struct {
	K8s        *K8s
	NodesSetup []*NodeConf
}

// NewK8s - Creates a new K8s Clientset with roles for the default config
func NewK8s(g *WithT, prepare ClearOption) (*K8s, error) {
	client, err := NewK8sWithoutRoles(g, prepare)
	if client == nil {
		logrus.Errorf("Error Creating K8s %v", err)
		return client, err
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.roles, _ = client.CreateRoles("admin", "view", "binding")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		client.cleanupFunc = InitSpireSecurity(client)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		client.CreatePod(pods.PrefixServicePod(client.namespace))
	}()
	wg.Wait()

	return client, err
}

// NewK8sForConfig - Creates a new K8s Clientset for the given config with creating roles
func NewK8sForConfig(g *WithT, prepare ClearOption, kubeconfig string) (*K8s, error) {
	client, err := NewK8sWithoutRolesForConfig(g, prepare, kubeconfig)
	client.roles, _ = client.CreateRoles("admin", "view", "binding")
	return client, err
}

// NewK8sWithoutRoles - Creates a new K8s Clientset for the default config
func NewK8sWithoutRoles(g *WithT, prepare ClearOption) (*K8s, error) {
	path := os.Getenv("KUBECONFIG")
	if len(path) == 0 {
		path = os.Getenv("HOME") + "/.kube/config"
	}
	return NewK8sWithoutRolesForConfig(g, prepare, path)
}

// NewK8sWithoutRolesForConfig - Creates a new K8s Clientset for the given config
func NewK8sWithoutRolesForConfig(g *WithT, prepare ClearOption, kubeconfigPath string) (*K8s, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	g.Expect(err).To(BeNil())

	client := K8s{
		pods: []*v1.Pod{},
		sa: []string{
			pods.NSCServiceAccount,
			pods.NSEServiceAccount,
			pods.NSMgrServiceAccount,
			pods.ForwardPlaneServiceAccount,
		},
		g: g,
	}
	client.setForwardingPlane()
	client.config = config
	client.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	client.startTime = metaV1.Now()

	client.apiServerHost = config.Host
	client.initNamespace()
	client.setIPVersion()

	client.versionedClientSet, err = versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	if prepare != NoClear {
		start := time.Now()
		if prepare == ReuseNSMResources {
			client.DeletePodsByName("vpn", "icmp", "nsc", "source", "dest", "xcon", "nse", "spire-proxy")
		} else {
			client.DeletePodsByName("nsmgr", "nsmd", "vppagent", "vpn", "icmp", "nsc", "source", "dest", "xcon", "spire-proxy", "nse")
		}
		client.CleanupConfigMaps()

		client.CleanupServices("nsm-admission-webhook-svc")
		client.CleanupDeployments()
		client.CleanupMutatingWebhookConfigurations()
		client.CleanupSecrets("nsm-admission-webhook-certs")
		logrus.Printf("Cleanup done: %v", time.Since(start))
	}

	if prepare != ReuseNSMResources || len(client.sa) != len(client.LookupServiceAccounts(client.sa...)) {
		_ = nsmrbac.DeleteAllRoles(client.clientset)
		client.CleanupCRDs()
		_ = client.DeleteServiceAccounts()
		client.CreateServiceAccounts()
	}
	client.resourcesBehaviour = prepare
	client.artifactManager = artifact.NewManager(artifact.ConfigFromEnv(), artifact.DefaultPresenterFactory(), []artifact.Finder{
		NewK8sLogFinder(&client),
		//		NewJaegerTracesFinder(client)
	}, nil)

	client.CreateServiceAccounts()

	_, err = client.CreateConfigMap(client.buildNSMConfigMap())
	g.Expect(err).To(BeNil())

	return &client, nil
}

// Immediate deletion does not wait for confirmation that the running resource has been terminated.
// The resource may continue to run on the cluster indefinitely
func (k8s *K8s) deletePodForce(pod *v1.Pod) error {
	graceTimeout := int64(0)
	delOpt := &metaV1.DeleteOptions{
		GracePeriodSeconds: &graceTimeout,
	}
	err := k8s.clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, delOpt)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), podDeleteTimeout)
	defer cancel()
	err = blockUntilPodWorking(k8s.clientset, ctx, pod)
	if err != nil {
		return err
	}
	return nil
}

func (k8s *K8s) checkAPIServerAvailable() {
	u, err := url.Parse(k8s.apiServerHost)
	if err != nil {
		logrus.Error(err)
	}

	logrus.Infof("Checking availability of API server on %v", u.Hostname())
	out, err := exec.Command("ping", u.Hostname(), "-c 5").Output()
	if err != nil {
		logrus.Error(err)
	}

	logrus.Infof(string(out))
}

func (k8s *K8s) initNamespace() {
	var err error
	nsmNamespace := namespace.GetNamespace()
	k8s.namespace, err = k8s.CreateTestNamespace(nsmNamespace)
	if os.Getenv("FIXED_NAMESPACE") == "" {
		if err != nil {
			logrus.Errorf("Error during create of test namespace %v", err)
			k8s.checkAPIServerAvailable()
		}

		k8s.g.Expect(err).To(BeNil())
	}
}

func (k8s *K8s) describePod(pod *v1.Pod) []v1.Event {
	result := []v1.Event{}
	eventsInterface := k8s.clientset.CoreV1().Events(k8s.namespace)
	selector := eventsInterface.GetFieldSelector(&pod.Name, &k8s.namespace, nil, nil)
	options := metaV1.ListOptions{FieldSelector: selector.String()}
	events, err := eventsInterface.List(options)
	if err != nil {
		logrus.Error(err)
	}
	for i := len(events.Items) - 1; i >= 0; i-- {
		if pod.UID == events.Items[i].InvolvedObject.UID {
			result = append(result, events.Items[i])
		}
	}
	return result
}

func (k8s *K8s) buildNSMConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "nsm-config",
			Namespace: k8s.GetK8sNamespace(),
		},
		Data: map[string]string{
			prefix_pool.PrefixesFile: "",
		},
	}
}

// Delete POD with completion check
// Make force delete on timeout
func (k8s *K8s) deletePods(pods ...*v1.Pod) error {
	var result error
	errCh := make(chan error, len(pods))
	for _, my_pod := range pods {
		pod := my_pod
		go func() {
			var deleteErr error
			defer func() {
				errCh <- deleteErr
			}()
			delOpt := &metaV1.DeleteOptions{}
			st := time.Now()
			logrus.Infof("Deleting %v", pod.Name)
			deleteErr = k8s.clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, delOpt)
			if deleteErr != nil {
				if strings.Contains(deleteErr.Error(), "not found") {
					deleteErr = nil
				}
				logrus.Warnf(`The POD "%s" may continue to run on the cluster, %v`, pod.Name, deleteErr)
				return
			}
			c, cancel := context.WithTimeout(context.Background(), podDeleteTimeout)
			defer cancel()
			err := blockUntilPodWorking(k8s.clientset, c, pod)
			if err != nil {
				err = k8s.deletePodForce(pod)
				if err != nil {
					logrus.Warnf(`The POD "%s" may continue to run on the cluster`, pod.Name)
					logrus.Warnf("Force delete error: %v", err)
				} else {
					logrus.Infof("The POD %v force deleted", pod.Name)
				}
			}
			logrus.Warnf(`The POD "%s" Deleted %v`, pod.Name, time.Since(st))
		}()
	}
	for i := 0; i < len(pods); i++ {
		err := <-errCh
		if err != nil {
			if result == nil {
				result = err
			} else {
				result = pkgerrors.Wrap(result, err.Error())
			}
		}
	}
	return result
}
func (k8s *K8s) deletePodsForce(pods ...*v1.Pod) error {
	var err error
	for _, pod := range pods {
		err = k8s.deletePodForce(pod)
		if err != nil {
			logrus.Warnf(`The POD "%s" may continue to run on the cluster %v`, pod.Name, err)
		}
	}
	return err
}

// GetVersion returns the k8s version
func (k8s *K8s) GetVersion() string {
	version, err := k8s.clientset.Discovery().ServerVersion()
	k8s.g.Expect(err).To(BeNil())
	return fmt.Sprintf("%s", version)
}

// GetNodes returns the nodes
func (k8s *K8s) GetNodes() []v1.Node {
	nodes, err := k8s.clientset.CoreV1().Nodes().List(metaV1.ListOptions{})
	if err != nil {
		k8s.checkAPIServerAvailable()
	}
	k8s.g.Expect(err).To(BeNil())
	return nodes.Items
}

// ListPods lists the pods
func (k8s *K8s) ListPods() []v1.Pod {
	return k8s.ListPodsByNs(k8s.namespace)
}

// ListPods lists the pods
func (k8s *K8s) ListPodsNs(ns string) []v1.Pod {
	podList, err := k8s.clientset.CoreV1().Pods(ns).List(metaV1.ListOptions{})
	k8s.g.Expect(err).To(BeNil())
	return podList.Items
}

// ListPods lists the pods
func (k8s *K8s) ListRefPods() []*v1.Pod {
	pods := k8s.ListPods()
	var result []*v1.Pod
	for i := range pods {
		p := &pods[i]
		result = append(result, p)
	}
	return result
}

//ListPodsByNs returns pod list by specific namespace
func (k8s *K8s) ListPodsByNs(ns string) []v1.Pod {
	podList, err := k8s.clientset.CoreV1().Pods(ns).List(metaV1.ListOptions{})
	k8s.g.Expect(err).To(BeNil())
	return podList.Items
}

// CleanupCRDs cleans up CRDs
func (k8s *K8s) CleanupCRDs() {
	k8s.cleanupNetworkServices()
	k8s.DeleteEndpoints()
	// Clean up Network Service Managers
	managers, _ := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServiceManagers(k8s.namespace).List(metaV1.ListOptions{})
	for _, mgr := range managers.Items {
		_ = k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServiceManagers(k8s.namespace).Delete(mgr.Name, &metaV1.DeleteOptions{})
	}
}

func (k8s *K8s) cleanupNetworkServices() {
	services, _ := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServices(k8s.namespace).List(metaV1.ListOptions{})
	for _, service := range services.Items {
		_ = k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServices(k8s.namespace).Delete(service.Name, &metaV1.DeleteOptions{})
	}
}

// DescribePod describes a pod
func (k8s *K8s) DescribePod(pod *v1.Pod) {
	events := k8s.describePod(pod)

	for i := range events {
		event := &events[i]
		logrus.Infof("Pod %s event: %v", pod.Name, prettyPrint(event))
	}
}

//GetPullingImagesDuration returns pod images pulling duration
func (k8s *K8s) GetPullingImagesDuration(pod *v1.Pod) time.Duration {
	events := k8s.describePod(pod)
	var start, end *time.Time
	for i := range events {
		event := &events[i]
		if strings.Contains(event.Reason, "Pulling") {
			if start == nil || start.After(event.FirstTimestamp.Time) {
				start = &event.FirstTimestamp.Time
			}
		}
		if strings.Contains(event.Reason, "Pulled") {
			if end == nil || end.Before(event.LastTimestamp.Time) {
				end = &event.LastTimestamp.Time
			}
		}
	}
	if start == nil || end == nil {
		return 0
	}
	return end.Sub(*start)
}

// PrintImageVersion Prints image version pf pod.
func (k8s *K8s) PrintImageVersion(pod *v1.Pod) {
	logs, err := k8s.GetLogs(pod, pod.Spec.Containers[0].Name)
	k8s.g.Expect(err).Should(BeNil())
	versionSubStr := "Version: "
	index := strings.Index(logs, versionSubStr)
	k8s.g.Expect(index == -1).ShouldNot(BeTrue())
	index += len(versionSubStr)
	builder := strings.Builder{}
	for ; index < len(logs); index++ {
		if logs[index] == '\n' {
			break
		}
		err = builder.WriteByte(logs[index])
		k8s.g.Expect(err).Should(BeNil())
	}
	version := builder.String()
	k8s.g.Expect(strings.TrimSpace(version)).ShouldNot(Equal(""))
	logrus.Infof("Version of %v is %v", pod.Name, version)
}

// DeleteEndpoints clean Network Service Endpoints from registry
func (k8s *K8s) DeleteEndpoints() {
	endpoints, _ := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServiceEndpoints(k8s.namespace).List(metaV1.ListOptions{})
	for i := range endpoints.Items {
		_ = k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServiceEndpoints(k8s.namespace).Delete(endpoints.Items[i].Name, &metaV1.DeleteOptions{})
	}
}

func (k8s *K8s) ProcessArtifacts(t *testing.T) {
	if exception := recover(); exception != nil {
		k8s.artifactManager.ProcessArtifacts()
		panic(exception)
	} else if t.Failed() || k8s.artifactManager.Config().SaveInAnyCase() {
		k8s.artifactManager.ProcessArtifacts()
		k8s.resourcesBehaviour = DefaultClear
	}
}

// Cleanup cleans up
func (k8s *K8s) Cleanup() {
	st := time.Now()
	k8s.reportSpans()
	k8s.cleanups()
	logrus.Infof("Cleanup time: %v", time.Since(st))
}

func filterNsmInfrastructure(pods ...*v1.Pod) []*v1.Pod {
	var result []*v1.Pod
	skipNames := []string{"nsmgr", "forwarder"}
	excludeNames := []string{"pnsmgr"}
	containsFunc := func(where string, what []string) bool {
		for i := range what {
			if strings.Contains(where, what[i]) {
				return true
			}
		}
		return false
	}
	for _, p := range pods {
		if containsFunc(p.Name, skipNames) && !containsFunc(p.Name, excludeNames) {
			continue
		}
		result = append(result, p)
	}
	return result
}

func (k8s *K8s) cleanups() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pods := k8s.ListRefPods()
		if k8s.resourcesBehaviour == ReuseNSMResources {
			_ = k8s.deletePods(filterNsmInfrastructure(pods...)...)
		} else {
			_ = k8s.deletePods(pods...)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if k8s.cleanupFunc != nil {
			k8s.cleanupFunc()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		k8s.CleanupConfigMaps()
	}()

	if k8s.resourcesBehaviour == DefaultClear {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = k8s.DeleteServiceAccounts()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			k8s.roles, _ = k8s.DeleteRoles(k8s.roles)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			k8s.CleanupCRDs()
		}()
	}
	if k8s.resourcesBehaviour == ReuseNSMResources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k8s.DeleteEndpoints()
			k8s.cleanupNetworkServices()
		}()
	}

	wg.Wait()
	k8s.pods = nil
	_ = k8s.DeleteTestNamespace(k8s.namespace)

}

// DeletePodsByName deletes pod if a pod's name contains one of the given strings
func (k8s *K8s) DeletePodsByName(noPods ...string) {
	pods := k8s.ListPods()
	podsList := []*v1.Pod{}
	for _, podName := range noPods {
		for i := range pods {
			lpod := &pods[i]
			if strings.Contains(lpod.Name, podName) {
				podsList = append(podsList, lpod)
			}
		}
	}
	k8s.DeletePods(podsList...)
}

// CreatePods create pods
func (k8s *K8s) CreatePods(templates ...*v1.Pod) []*v1.Pod {
	pods, _ := k8s.CreatePodsRaw(PodStartTimeout, true, templates...)
	return pods
}

// CreatePodsRaw create raw pods
func (k8s *K8s) CreatePodsRaw(timeout time.Duration, failTest bool, templates ...*v1.Pod) ([]*v1.Pod, error) {
	results := k8s.createAndBlock(k8s.clientset, k8s.namespace, timeout, templates...)
	var pods []*v1.Pod

	// Add pods into managed list of created pods, do not matter about errors, since we still need to remove them.
	var errs []error
	for _, podResult := range results {
		if podResult == nil {
			logrus.Errorf("Error - Pod should have been created, but is nil: %v", podResult)
		} else {
			if podResult.pod != nil {
				pods = append(pods, podResult.pod)
			}
			if podResult.err != nil {
				logrus.Errorf("Error Creating Pod: %s %v", podResult.pod.Name, podResult.err)
				errs = append(errs, podResult.err)
			}
		}
	}
	k8s.podLock.Lock()
	defer k8s.podLock.Unlock()
	k8s.pods = append(k8s.pods, pods...)

	// Make sure unit test is failed
	var err error = nil
	if failTest {
		k8s.g.Expect(len(errs)).To(Equal(0), string(debug.Stack()))
	} else {
		// Lets construct error
		err = errors.Errorf("Errors %v", errs)
	}

	return pods, err
}

// GetPod gets a pod
func (k8s *K8s) GetPod(pod *v1.Pod) (*v1.Pod, error) {
	return k8s.clientset.CoreV1().Pods(pod.Namespace).Get(pod.Name, metaV1.GetOptions{})
}

// CreatePod creates a pod
func (k8s *K8s) CreatePod(template *v1.Pod) *v1.Pod {
	results, err := k8s.CreatePodsRaw(PodStartTimeout, true, template)
	if err != nil || len(results) == 0 {
		return nil
	} else {
		return results[0]
	}
}

// DeletePods delete pods
func (k8s *K8s) DeletePods(pods ...*v1.Pod) {
	err := k8s.deletePods(pods...)
	k8s.g.Expect(err).To(BeNil())

	k8s.podLock.Lock()
	defer k8s.podLock.Unlock()
	for _, pod := range pods {
		for idx, pod0 := range k8s.pods {
			if pod.Name == pod0.Name {
				k8s.pods = append(k8s.pods[:idx], k8s.pods[idx+1:]...)
			}
		}
	}
}

// DeletePodsForce delete pods forcefully
func (k8s *K8s) DeletePodsForce(pods ...*v1.Pod) {
	err := k8s.deletePodsForce(pods...)
	k8s.g.Expect(err).To(BeNil())

	k8s.podLock.Lock()
	defer k8s.podLock.Unlock()
	for _, pod := range pods {
		for idx, pod0 := range k8s.pods {
			if pod.Name == pod0.Name {
				k8s.pods = append(k8s.pods[:idx], k8s.pods[idx+1:]...)
			}
		}
	}
}

// GetLogsChannel returns logs channel from pod with the given options
func (k8s *K8s) GetLogsChannel(ctx context.Context, pod *v1.Pod, options *v1.PodLogOptions) (chan string, chan error) {
	linesChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		defer close(linesChan)
		defer close(errChan)

		reader, err := k8s.clientset.CoreV1().Pods(k8s.namespace).GetLogs(pod.Name, options).Stream()
		if err != nil {
			logrus.Errorf("Failed to get logs from %v err: %v", pod.Name, err)
			errChan <- err
			return
		}
		defer func() { _ = reader.Close() }()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case linesChan <- scanner.Text():
			}
		}
		errChan <- scanner.Err()
	}()

	return linesChan, errChan
}

// GetLogsWithOptions returns logs collected from pod with the given options
func (k8s *K8s) GetLogsWithOptions(pod *v1.Pod, options *v1.PodLogOptions) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), podGetLogTimeout)
	defer cancel()

	var builder strings.Builder
	for linesChan, errChan := k8s.GetLogsChannel(ctx, pod, options); ; {
		select {
		case line := <-linesChan:
			_, _ = builder.WriteString(line)
			_, _ = builder.WriteString("\n")
		case err := <-errChan:
			return builder.String(), err
		}
	}
}

// GetLogs returns logs collected from pod::container
func (k8s *K8s) GetLogs(pod *v1.Pod, container string) (string, error) {
	return k8s.GetLogsWithOptions(pod, &v1.PodLogOptions{
		Container: container,
	})
}

// WaitLogsContains waits with timeout for pod::container logs to contain pattern as substring
func (k8s *K8s) WaitLogsContains(pod *v1.Pod, container, pattern string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	matcher := func(s string) bool {
		return strings.Contains(s, pattern)
	}

	description := fmt.Sprintf("Timeout waiting for logs pattern %v in %v::%v.", pattern, pod.Name, container)

	k8s.waitLogsMatch(ctx, pod, container, matcher, description)
}

// WaitLogsContainsRegex waits with timeout for pod::contained logs to contain substring matching regexp pattern
func (k8s *K8s) WaitLogsContainsRegex(pod *v1.Pod, container, pattern string, timeout time.Duration) error {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	matcher := func(s string) bool {
		return r.FindStringSubmatch(s) != nil
	}
	description := fmt.Sprintf("Timeout waiting for logs matching regexp %v in %v::%v.", pattern, pod.Name, container)

	k8s.waitLogsMatch(ctx, pod, container, matcher, description)

	return nil
}

//GetFullLogs - return full logs
func (k8s *K8s) GetFullLogs(pod *v1.Pod, container string, previous bool) (string, error) {
	container = k8s.fixContainer(pod, container)

	getLogsOpt := &v1.PodLogOptions{
		Previous: previous,
	}
	if len(container) > 0 {
		getLogsOpt.Container = container
	}
	response := k8s.clientset.CoreV1().Pods(k8s.namespace).GetLogs(pod.Name, getLogsOpt)
	result, err := response.DoRaw()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", result), nil
}

func (k8s *K8s) fixContainer(pod *v1.Pod, container string) string {
	updatedPod, err := k8s.GetPod(pod)
	if err != nil {
		logrus.Error(errors.WithMessagef(err, "failed to get update pod %v", pod.Name))
	}

	// Check if it is init container name
	for _, c := range updatedPod.Spec.InitContainers {
		if c.Name == container {
			// All ok and it is allowed.
			return container
		}
	}

	if container != "" && len(updatedPod.Spec.Containers) == 1 {
		logrus.Infof("getting logs without container %v=none", container)
		container = ""
	}
	return container
}

func (k8s *K8s) waitLogsMatch(ctx context.Context, pod *v1.Pod, container string, matcher func(string) bool, description string) {
	container = k8s.fixContainer(pod, container)
	var options = &v1.PodLogOptions{
		Follow:    true,
		Container: container,
		SinceTime: &k8s.startTime,
	}
	var builder strings.Builder
	for linesChan, errChan := k8s.GetLogsChannel(ctx, pod, options); ; {
		select {
		case err := <-errChan:
			if err != nil {
				logrus.Warnf("Error on get logs: %v retrying", err)
			} else {
				logrus.Warnf("Stream closed retrying %v::%v", pod.GetName(), container)
				fullLogs, err := k8s.GetFullLogs(pod, container, false)
				if err != nil {
					logrus.Errorf("Failed to retrieve full logs %v %v", pod.GetName(), err)
				} else {
					if matcher(fullLogs) {
						return
					}
					logrus.Errorf("Stack: %v", string(debug.Stack()))
					logrus.Errorf("%v Last logs: %v\n Full Logs: %v", description, builder.String(), fullLogs)
					k8s.g.Expect(false).To(BeTrue())
				}
			}
			<-time.After(1000 * time.Millisecond)

			linesChan, errChan = k8s.GetLogsChannel(ctx, pod, options)
		case line := <-linesChan:
			_, _ = builder.WriteString(line)
			_, _ = builder.WriteString("\n")
			if matcher(line) {
				return
			}
		case <-ctx.Done():
			updPod, err := k8s.GetPod(pod)
			if err != nil {
				logrus.Errorf("error retrieving pod info %v %v", pod.Name, err)
			} else {
				logrus.Infof("pod info: %v %v", pod.Name, prettyPrint(updPod))
			}
			k8s.DescribePod(pod)

			logrus.Errorf("%v Last logs: %v", description, builder.String())
			k8s.g.Expect(false).To(BeTrue(), string(debug.Stack()))
			return
		}
	}
}

// UpdatePod updates a pod
func (k8s *K8s) UpdatePod(pod *v1.Pod) *v1.Pod {
	pod, err := k8s.clientset.CoreV1().Pods(pod.Namespace).Get(pod.Name, metaV1.GetOptions{})
	k8s.g.Expect(err).To(BeNil())
	return pod
}

// GetClientSet returns a clientset
func (k8s *K8s) GetClientSet() (kubernetes.Interface, error) {
	return k8s.clientset, nil
}

// GetConfig returns config
func (k8s *K8s) GetConfig() *rest.Config {
	return k8s.config
}

func isNodeReady(node *v1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == v1.NodeReady {
			resultValue := c.Status == v1.ConditionTrue
			return resultValue
		}
	}
	return false
}

// GetNodesWait wait for required number of nodes are up and running fine
func (k8s *K8s) GetNodesWait(requiredNumber int, timeout time.Duration) []v1.Node {
	st := time.Now()
	warnPrinted := false
	for {
		nodes := k8s.GetNodes()
		ready := 0
		for i := range nodes {
			node := &nodes[i]
			logrus.Infof("Checking node: %s", node.Name)
			if isNodeReady(node) {
				ready++
			}
		}
		if ready >= requiredNumber {
			return nodes
		}
		since := time.Since(st)
		if since > timeout {
			k8s.g.Expect(len(nodes)).To(Equal(requiredNumber))
		}
		if since > timeout/10 && !warnPrinted {
			logrus.Warnf("Waiting for %d nodes to arrive, currently have: %d", requiredNumber, len(nodes))
			warnPrinted = true
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// CreateService creates a service
func (k8s *K8s) CreateService(service *v1.Service, namespace string) (*v1.Service, error) {
	_ = k8s.clientset.CoreV1().Services(namespace).Delete(service.Name, &metaV1.DeleteOptions{})
	s, err := k8s.clientset.CoreV1().Services(namespace).Create(service)
	if err != nil {
		logrus.Errorf("Error creating service: %v %v", s, err)
	}
	logrus.Infof("Service is created: %v", s)
	return s, err
}

// DeleteService deletes a service
func (k8s *K8s) DeleteService(service *v1.Service) error {
	return k8s.clientset.CoreV1().Services(k8s.namespace).Delete(service.GetName(), &metaV1.DeleteOptions{})
}

// CleanupServices cleans up services
func (k8s *K8s) CleanupServices(services ...string) {
	for _, s := range services {
		_ = k8s.clientset.CoreV1().Services(k8s.namespace).Delete(s, &metaV1.DeleteOptions{})
	}
}

// CreateDeployment creates deployment
func (k8s *K8s) CreateDeployment(deployment *appsv1.Deployment, namespace string) (*appsv1.Deployment, error) {
	d, err := k8s.clientset.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil {
		logrus.Errorf("Error creating deployment: %v %v", d, err)
	}
	logrus.Infof("Deployment is created: %v", d)
	return d, err
}

// DeleteDeployment deletes deployment
func (k8s *K8s) DeleteDeployment(deployment *appsv1.Deployment, namespace string) error {
	return k8s.clientset.AppsV1().Deployments(namespace).Delete(deployment.GetName(), &metaV1.DeleteOptions{})
}

// CleanupDeployments cleans up deployment
func (k8s *K8s) CleanupDeployments() {
	deployments, _ := k8s.clientset.AppsV1().Deployments(k8s.namespace).List(metaV1.ListOptions{})
	for i := range deployments.Items {
		d := &deployments.Items[i]
		err := k8s.DeleteDeployment(d, k8s.namespace)
		if err != nil {
			logrus.Errorf("An error during deployment deleting %v", err)
		}
	}
}

// CreateMutatingWebhookConfiguration creates mutating webhook with configuration
func (k8s *K8s) CreateMutatingWebhookConfiguration(mutatingWebhookConf *arv1beta1.MutatingWebhookConfiguration) (*arv1beta1.MutatingWebhookConfiguration, error) {
	awc, err := k8s.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(mutatingWebhookConf)
	if err != nil {
		logrus.Errorf("Error creating MutatingWebhookConfiguration: %v %v", awc, err)
	}
	logrus.Infof("MutatingWebhookConfiguration is created: %v", awc)
	return awc, err
}

// DeleteMutatingWebhookConfiguration deletes mutating webhook with configuration
func (k8s *K8s) DeleteMutatingWebhookConfiguration(mutatingWebhookConf *arv1beta1.MutatingWebhookConfiguration) error {
	return k8s.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(mutatingWebhookConf.GetName(), &metaV1.DeleteOptions{})
}

// CleanupMutatingWebhookConfigurations cleans mutating webhook with configuration
func (k8s *K8s) CleanupMutatingWebhookConfigurations() {
	mwConfigs, _ := k8s.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(metaV1.ListOptions{})
	for _, mwConfig := range mwConfigs.Items {
		mwConfig := mwConfig
		err := k8s.DeleteMutatingWebhookConfiguration(&mwConfig)
		if err != nil {
			logrus.Errorf("Error cleaning up mutating webhook configurations: %v", err)
		}
	}
}

// CreateSecret creates a secret
func (k8s *K8s) CreateSecret(secret *v1.Secret, namespace string) (*v1.Secret, error) {
	s, err := k8s.clientset.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		logrus.Errorf("Error creating secret: %v %v", s, err)
	}
	logrus.Infof("secret is created: %v", s)
	return s, err
}

// DeleteSecret deletes a secret
func (k8s *K8s) DeleteSecret(name, namespace string) error {
	return k8s.clientset.CoreV1().Secrets(namespace).Delete(name, &metaV1.DeleteOptions{})
}

// CleanupSecrets cleans a secret
func (k8s *K8s) CleanupSecrets(secrets ...string) {
	for _, s := range secrets {
		_ = k8s.DeleteSecret(s, k8s.namespace)
	}
}

// IsPodReady returns if a pod is ready
func (k8s *K8s) IsPodReady(pod *v1.Pod) bool {
	return isPodReady(pod)
}

// CreateConfigMap creates a configmap
func (k8s *K8s) CreateConfigMap(cm *v1.ConfigMap) (*v1.ConfigMap, error) {
	logrus.Infof("Creating ConfigMap '%s' in namespace'%s'...", cm.Name, cm.Namespace)
	return k8s.clientset.CoreV1().ConfigMaps(cm.Namespace).Create(cm)
}

// CleanupConfigMaps cleans a configmap
func (k8s *K8s) CleanupConfigMaps() {
	// Clean up Network Service Endpoints
	configMaps, _ := k8s.clientset.CoreV1().ConfigMaps(k8s.namespace).List(metaV1.ListOptions{})
	for _, cm := range configMaps.Items {
		_ = k8s.clientset.CoreV1().ConfigMaps(k8s.namespace).Delete(cm.Name, &metaV1.DeleteOptions{})
	}
}

// CreateTestNamespace creates a test namespace
func (k8s *K8s) CreateTestNamespace(namespace string) (string, error) {
	if len(namespace) == 0 || namespace == "default" {
		return "default", nil
	}
	var nsTemplate *v1.Namespace
	if os.Getenv("FIXED_NAMESPACE") == "true" {
		nsTemplate = &v1.Namespace{
			ObjectMeta: metaV1.ObjectMeta{
				Name: namespace,
			},
		}
	} else {
		nsTemplate = &v1.Namespace{
			ObjectMeta: metaV1.ObjectMeta{
				GenerateName: namespace + "-",
			},
		}
	}

	nsNamespace, err := k8s.clientset.CoreV1().Namespaces().Create(nsTemplate)
	if err != nil {
		nsRes := ""
		if strings.Contains(err.Error(), "already exists") {
			nsRes = namespace
		}
		return nsRes, errors.Wrap(err, "failed to create a namespace")
	}

	logrus.Printf("namespace %v is created", nsNamespace.GetName())

	return nsNamespace.GetName(), nil
}

// CreateServiceAccounts create service accounts with passed names
func (k8s *K8s) CreateServiceAccounts() {
	accountCount := len(k8s.sa) + 1 //1 means default acc
	errs := make(chan error, accountCount)
	for i := range k8s.sa {
		index := i
		go func() {
			n := k8s.sa[index]
			_, err := k8s.clientset.CoreV1().ServiceAccounts(k8s.namespace).Create(&v1.ServiceAccount{
				ObjectMeta: metaV1.ObjectMeta{
					Name: n,
				},
			})
			if err != nil {
				errs <- err
				return
			}
			errs <- k8s.waitForServiceAccountCreated(accountWaitTimeout, n)
		}()
	}

	go func() {
		errs <- k8s.waitForServiceAccountCreated(accountWaitTimeout, pods.DefaultAccount)
	}()
	for i := 0; i < accountCount; i++ {
		err := <-errs
		if err != nil {
			logrus.Error(err)
			//			k8s.g.Expect(true).Should(BeFalse())
		}
	}
}

// DeleteServiceAccounts deletes passed service accounts from cluster
func (k8s *K8s) DeleteServiceAccounts() error {
	var lastErr error
	for _, n := range k8s.sa {
		if err := k8s.clientset.CoreV1().ServiceAccounts(k8s.namespace).Delete(n, &metaV1.DeleteOptions{}); err != nil {
			logrus.Error(err)
			lastErr = err
		}
	}
	return lastErr
}

// DeleteTestNamespace deletes a test namespace
func (k8s *K8s) DeleteTestNamespace(namespace string) error {
	if namespace == "default" {
		return nil
	}
	var immediate int64
	err := k8s.clientset.CoreV1().Namespaces().Delete(namespace, &metaV1.DeleteOptions{GracePeriodSeconds: &immediate})
	if err != nil {
		return errors.Wrapf(err, "failed to delete namespace %q", namespace)
	}

	logrus.Printf("namespace %v is deleted", namespace)

	return nil
}

// GetNamespace returns a namespace
func (k8s *K8s) GetNamespace(namespace string) (*v1.Namespace, error) {
	ns, err := k8s.clientset.CoreV1().Namespaces().Get(namespace, metaV1.GetOptions{})
	if err != nil {
		err = errors.Wrapf(err, "failed to get namespace %q", namespace)
	}
	return ns, err
}

// GetK8sNamespace returns a namespace
func (k8s *K8s) GetK8sNamespace() string {
	return k8s.namespace
}

// CreateRoles create roles
func (k8s *K8s) CreateRoles(rolesList ...string) ([]nsmrbac.Role, error) {
	createdRoles := []nsmrbac.Role{}
	for _, kind := range rolesList {
		role := nsmrbac.Roles[kind](nsmrbac.RoleNames[kind], k8s.GetK8sNamespace())
		createdRoles = append(createdRoles, role)
	}
	var wg sync.WaitGroup
	var roleError error
	for _, r := range createdRoles {
		wg.Add(1)
		role := r
		go func() {
			defer wg.Done()
			if err := role.Create(k8s.clientset); err != nil {
				logrus.Errorf("failed creating role: %v %v", role, err)
				roleError = err
				return
			}
			logrus.Infof("role is created: %v", role)
		}()
	}
	wg.Wait()
	return createdRoles, roleError
}

// DeleteRoles delete roles
func (k8s *K8s) DeleteRoles(rolesList []nsmrbac.Role) ([]nsmrbac.Role, error) {
	for i := range rolesList {
		err := rolesList[i].Delete(k8s.clientset, rolesList[i].GetName())
		if err != nil {
			logrus.Errorf("failed deleting role: %v %v", rolesList[i], err)
			return rolesList[i:], err
		}
	}

	return nil, nil
}

// setIPVersion choose whether or not to use IPv6 in testing
func (k8s *K8s) setIPVersion() {
	useIPv6, ok := os.LookupEnv(envUseIPv6)
	if !ok {
		logrus.Infof("%s not set, using default %t", envUseIPv6, envUseIPv6Default)
		k8s.useIPv6 = envUseIPv6Default
	} else {
		k8s.useIPv6, _ = strconv.ParseBool(useIPv6)
	}
}

// UseIPv6 returns which IP version is going to be used in testing
func UseIPv6() bool {
	return utils.EnvVar(envUseIPv6).GetBooleanOrDefault(envUseIPv6Default)
}

// UseIPv6 returns which IP version is going to be used in testing
func (k8s *K8s) UseIPv6() bool {
	return k8s.useIPv6
}

// setForwardingPlane sets which forwarding plane to be used in testing
func (k8s *K8s) setForwardingPlane() {
	plane, ok := os.LookupEnv(pods.EnvForwardingPlane)
	if !ok {
		logrus.Infof("%s not set, using default forwarder - %s", pods.EnvForwardingPlane, pods.EnvForwardingPlaneDefault)
		k8s.forwardingPlane = pods.EnvForwardingPlaneDefault
	} else {
		logrus.Infof("%s set to: %s", pods.EnvForwardingPlane, plane)
		k8s.forwardingPlane = plane
	}
}

// GetForwardingPlane gets which forwarding plane is going to be used in testing
func (k8s *K8s) GetForwardingPlane() string {
	return k8s.forwardingPlane
}

// GetNSEs returns existing 'nse' resources
func (k8s *K8s) GetNSEs() ([]v1alpha1.NetworkServiceEndpoint, error) {
	nseList, err := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServiceEndpoints(k8s.namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nseList.Items, err
}

// GetNSMs returns existing 'nse' resources
func (k8s *K8s) GetNSMList() ([]v1alpha1.NetworkServiceManager, error) {
	nsmList, err := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServiceManagers(k8s.namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nsmList.Items, err
}

// DeleteNSEs deletes 'nse' resources by names
func (k8s *K8s) DeleteNSEs(names ...string) error {
	nseClient := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServiceEndpoints(k8s.namespace)
	for _, name := range names {
		if err := nseClient.Delete(name, &metaV1.DeleteOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// GetNetworkServices returns existing 'networkservice' resources
func (k8s *K8s) GetNetworkServices() ([]v1alpha1.NetworkService, error) {
	networkServiceList, err := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServices(k8s.namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return networkServiceList.Items, err
}

// DeleteNetworkServices deletes 'networkservice' resources by names
func (k8s *K8s) DeleteNetworkServices(names ...string) error {
	networkServiceClient := k8s.versionedClientSet.NetworkservicemeshV1alpha1().NetworkServices(k8s.namespace)
	for _, name := range names {
		if err := networkServiceClient.Delete(name, &metaV1.DeleteOptions{}); err != nil {
			return err
		}
	}
	return nil
}

//IsNetworkProblem - return error if pod has deploy network problems detected in events.
func (k8s *K8s) IsNetworkProblem(pod *v1.Pod) error {
	// Check if we have CNI issue and try to re-create pod.
	events := k8s.describePod(pod)
	for index := range events {
		msg := &events[index]
		if k8s.isNetworkProblemEvent(msg) {
			return errors.Errorf("pod %s deploy error: %v.", pod.Name, msg.Message)
		}
	}
	return nil
}
