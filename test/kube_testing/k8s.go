package kube_testing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
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

	nsmrbac "github.com/networkservicemesh/networkservicemesh/test/kube_testing/rbac"
)

const (
	podStartTimeout  = 3 * time.Minute
	podDeleteTimeout = 15 * time.Second
	podExecTimeout   = 1 * time.Minute
	podGetLogTimeout = 1 * time.Minute
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

func (l *K8s) createAndBlock(client kubernetes.Interface, config *rest.Config, namespace string, timeout time.Duration, pods ...*v1.Pod) []*PodDeployResult {

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
				resultChan <- &PodDeployResult{pod, err}
				return
			}
			pod, err = blockUntilPodReady(client, timeout, pod)
			if err != nil {
				resultChan <- &PodDeployResult{pod, err}
				return
			}

			// Let's fetch more information about pod created

			updated_pod, err := client.CoreV1().Pods(namespace).Get(pod.Name, metaV1.GetOptions{})
			if err != nil {
				resultChan <- &PodDeployResult{updated_pod, err}
				return
			}
			resultChan <- &PodDeployResult{updated_pod, nil}

		}(pod)
	}

	if !waitTimeout(fmt.Sprintf("createAndBlock with pods: %v", pods), &wg, timeout) {
		logrus.Errorf("Failed to deploy pod, trying to get any information")
		results := []*PodDeployResult{}
		for _, p := range pods {
			pod, err := client.CoreV1().Pods(namespace).Get(p.Name, metaV1.GetOptions{})
			if err != nil {
				logrus.Errorf("Failed to get pod information: %v", err)
			}
			if pod != nil {
				logrus.Infof("Pod information: %v", pod)
				for _, cs := range pod.Status.ContainerStatuses {
					if !cs.Ready {
						logs, _ := l.GetLogs(pod, cs.Name)
						logrus.Infof("Pod %v container not started: %v Logs: %v", pod.Name, cs.Name, logs)
					}
				}
			}
			results = append(results, &PodDeployResult{
				err: fmt.Errorf("Failed to deploy pod"),
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

func blockUntilPodReady(client kubernetes.Interface, timeout time.Duration, sourcePod *v1.Pod) (*v1.Pod, error) {
	st := time.Now()
	infoPrinted := false
	for {
		pod, err := client.CoreV1().Pods(sourcePod.Namespace).Get(sourcePod.Name, metaV1.GetOptions{})

		// To be sure we not loose pod information.
		if pod == nil {
			pod = sourcePod
		}
		if err != nil {
			return pod, err
		}

		if pod != nil && pod.Status.Phase != v1.PodPending {
			break
		}

		if time.Since(st) > timeout/2 && !infoPrinted {
			logrus.Infof("Pod deploy half time passed: %v", pod)
			infoPrinted = true
		}

		time.Sleep(time.Millisecond * time.Duration(50))

		if time.Since(st) > timeout {
			return pod, podTimeout(pod)
		}
	}

	watcher, err := client.CoreV1().Pods(sourcePod.Namespace).Watch(metaV1.SingleObject(metaV1.ObjectMeta{Name: sourcePod.Name}))

	if err != nil {
		return sourcePod, err
	}

	for {
		select {
		case _, ok := <-watcher.ResultChan():

			if !ok {
				return sourcePod, fmt.Errorf("Some error watching for pod status")
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
	return fmt.Errorf("Timeout during waiting for pod change status for pod %s %v status: ", pod.Name, pod.Status.Conditions)
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

func blockUntilPodWorking(client kubernetes.Interface, context context.Context, pod *v1.Pod) error {

	exists := make(chan error)

	go func() {
		for {
			pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metaV1.GetOptions{})
			if err != nil {
				// Pod not found
				close(exists)
				break
			}

			if pod == nil {
				close(exists)
				break
			}

			time.Sleep(time.Millisecond * time.Duration(50))
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
	clientset          kubernetes.Interface
	versionedClientSet *versioned.Clientset
	pods               []*v1.Pod
	config             *rest.Config
	roles              []nsmrbac.Role
}

func NewK8s() (*K8s, error) {

	client, err := NewK8sWithoutRoles()
	client.roles, _ = client.CreateRoles("admin", "view", "binding")
	return client, err
}

func NewK8sWithoutRoles() (*K8s, error) {

	path := os.Getenv("KUBECONFIG")
	if len(path) == 0 {
		path = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", path)
	Expect(err).To(BeNil())

	client := K8s{
		pods: []*v1.Pod{},
	}
	client.config = config
	client.clientset, err = kubernetes.NewForConfig(config)

	Expect(err).To(BeNil())

	client.versionedClientSet, err = versioned.NewForConfig(config)
	Expect(err).To(BeNil())

	return &client, nil
}

// Immediate deletion does not wait for confirmation that the running resource has been terminated.
// The resource may continue to run on the cluster indefinitely
func (o *K8s) deletePodForce(pod *v1.Pod) error {
	graceTimeout := int64(0)
	delOpt := &metaV1.DeleteOptions{
		GracePeriodSeconds: &graceTimeout,
	}
	err := o.clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, delOpt)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), podDeleteTimeout)
	defer cancel()
	err = blockUntilPodWorking(o.clientset, ctx, pod)
	if err != nil {
		return err
	}
	return nil
}

// Delete POD with completion check
// Make force delete on timeout
func (o *K8s) deletePods(pods ...*v1.Pod) error {
	var ctx []context.Context
	for _, pod := range pods {
		delOpt := &metaV1.DeleteOptions{}
		err := o.clientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, delOpt)
		if err != nil {
			return err
		}

		c, cancel := context.WithTimeout(context.Background(), podDeleteTimeout)
		ctx = append(ctx, c)
		defer cancel()
	}

	for i, pod := range pods {
		err := blockUntilPodWorking(o.clientset, ctx[i], pod)
		if err != nil {
			err = o.deletePodForce(pod)
			logrus.Warnf(`The POD "%s" may continue to run on the cluster`, pod.Name)
			if err != nil {
				logrus.Warn(err)
			}
		}
	}
	return nil
}

func (o *K8s) GetVersion() string {
	version, err := o.clientset.Discovery().ServerVersion()
	Expect(err).To(BeNil())
	return fmt.Sprintf("%s", version)
}

func (o *K8s) GetNodes() []v1.Node {
	nodes, err := o.clientset.CoreV1().Nodes().List(metaV1.ListOptions{})
	Expect(err).To(BeNil())
	return nodes.Items
}

func (o *K8s) ListPods() []v1.Pod {
	podList, err := o.clientset.CoreV1().Pods("default").List(metaV1.ListOptions{})
	Expect(err).To(BeNil())
	return podList.Items
}

func (o *K8s) CleanupCRDs() {

	// Clean up Network Services
	services, _ := o.versionedClientSet.Networkservicemesh().NetworkServices("default").List(metaV1.ListOptions{})
	for _, service := range services.Items {
		_ = o.versionedClientSet.Networkservicemesh().NetworkServices("default").Delete(service.Name, &metaV1.DeleteOptions{})
	}

	// Clean up Network Service Endpoints
	endpoints, _ := o.versionedClientSet.Networkservicemesh().NetworkServiceEndpoints("default").List(metaV1.ListOptions{})
	for _, ep := range endpoints.Items {
		_ = o.versionedClientSet.Networkservicemesh().NetworkServiceEndpoints("default").Delete(ep.Name, &metaV1.DeleteOptions{})
	}

	// Clean up Network Service Managers
	managers, _ := o.versionedClientSet.Networkservicemesh().NetworkServiceManagers("default").List(metaV1.ListOptions{})
	for _, mgr := range managers.Items {
		_ = o.versionedClientSet.Networkservicemesh().NetworkServiceManagers("default").Delete(mgr.Name, &metaV1.DeleteOptions{})
	}
}

func (l *K8s) Cleanup() {
	err := l.deletePods(l.pods...)
	Expect(err).To(BeNil())
	l.pods = nil
	l.CleanupCRDs()
	l.CleanupConfigMaps()
	l.DeleteRoles(l.roles)
}

func (l *K8s) PrepareDefault() {
	l.Prepare("nsmgr", "nsmd", "vppagent", "vpn", "icmp", "nsc", "source", "dest")
}

func (l *K8s) Prepare(noPods ...string) {
	for _, podName := range noPods {
		for _, lpod := range l.ListPods() {
			if strings.Contains(lpod.Name, podName) {
				l.DeletePods(&lpod)
			}
		}
	}
}

/**

 */
func (l *K8s) CreatePods(templates ...*v1.Pod) []*v1.Pod {
	pods, _ := l.CreatePodsRaw(podStartTimeout, true, templates...)
	return pods
}
func (l *K8s) CreatePodsRaw(timeout time.Duration, failTest bool, templates ...*v1.Pod) ([]*v1.Pod, error) {
	results := l.createAndBlock(l.clientset, l.config, "default", timeout, templates...)
	pods := []*v1.Pod{}

	// Add pods into managed list of created pods, do not matter about errors, since we still need to remove them.
	errs := []error{}
	for _, podResult := range results {
		if podResult.pod != nil {
			pods = append(pods, podResult.pod)
		}
		if podResult.err != nil {
			logrus.Errorf("Error Creating Pod: %s %v", podResult.pod.Name, podResult.err)
			errs = append(errs, podResult.err)
		}
	}
	l.pods = append(l.pods, pods...)

	// Make sure unit test is failed
	var err error = nil
	if failTest {
		Expect(len(errs)).To(Equal(0))
	} else {
		// Lets construct error
		err = fmt.Errorf("Errors %v", errs)
	}

	return pods, err
}

func (l *K8s) CreatePod(template *v1.Pod) *v1.Pod {
	results, _ := l.CreatePodsRaw(podStartTimeout, true, template)
	return results[0]
}

func (l *K8s) DeletePods(pods ...*v1.Pod) {
	err := l.deletePods(pods...)
	Expect(err).To(BeNil())

	for _, pod := range pods {
		for idx, pod0 := range l.pods {
			if pod.Name == pod0.Name {
				l.pods = append(l.pods[:idx], l.pods[idx+1:]...)
			}
		}
	}
}

func (k8s *K8s) GetLogs(pod *v1.Pod, container string) (string, error) {
	getLogsOpt := &v1.PodLogOptions{}
	if len(container) > 0 {
		getLogsOpt.Container = container
	}
	var wg sync.WaitGroup
	var result []byte
	var err error
	wg.Add(1)
	go func() {
		defer wg.Done()
		response := k8s.clientset.CoreV1().Pods("default").GetLogs(pod.Name, getLogsOpt)
		result, err = response.DoRaw()
	}()
	if !waitTimeout(fmt.Sprintf("GetLogs %v:%v", pod.Name, container), &wg, podGetLogTimeout) {
		logrus.Errorf("Failed to get logs from: %v.%v", pod.Name, container)
	}

	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", result), nil
}

func (o *K8s) WaitLogsContains(pod *v1.Pod, container string, pattern string, timeout time.Duration) {
	st := time.Now()
	for {
		logs, error := o.GetLogs(pod, container)
		if error != nil {
			logrus.Printf("Error on get logs: %v retrying", error)
		}
		if !strings.Contains(logs, pattern) {
			<-time.Tick(100 * time.Millisecond)
		} else {
			break
		}
		if time.Since(st) > timeout {
			logrus.Errorf("Timeout waiting for logs pattern %s in pod %s. Last logs: %s", pattern, pod.Name, logs)
			Expect(strings.Contains(logs, pattern)).To(Equal(true))
			return
		}
	}
}

func (o *K8s) UpdatePod(pod *v1.Pod) *v1.Pod {
	pod, error := o.clientset.CoreV1().Pods(pod.Namespace).Get(pod.Name, metaV1.GetOptions{})
	Expect(error).To(BeNil())
	return pod
}

func (k8s *K8s) GetClientSet() (kubernetes.Interface, error) {
	return k8s.clientset, nil
}

func (k8s *K8s) GetConfig() *rest.Config {
	return k8s.config
}

func isNodeReady(node v1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == v1.NodeReady {
			resultValue := c.Status == v1.ConditionTrue
			return resultValue
		}
	}
	return false
}

/**
Wait for required number of nodes are up and running fine.
*/
func (k8s *K8s) GetNodesWait(requiredNumber int, timeout time.Duration) []v1.Node {
	st := time.Now()
	warnPrinted := false
	for {
		nodes := k8s.GetNodes()
		ready := 0
		for _, node := range nodes {
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
			Expect(len(nodes)).To(Equal(requiredNumber))
		}
		if since > timeout/10 && !warnPrinted {
			logrus.Warnf("Waiting for %d nodes to arrive, currently have: %d", len(nodes), requiredNumber)
			warnPrinted = true
		}
		time.Sleep(50 * time.Millisecond)
	}

}

func (o *K8s) CreateService(service *v1.Service, namespace string) (*v1.Service, error) {
	_ = o.clientset.CoreV1().Services(namespace).Delete(service.Name, &metaV1.DeleteOptions{})
	s, err := o.clientset.CoreV1().Services(namespace).Create(service)
	if err != nil {
		logrus.Errorf("Error creating service: %v %v", s, err)
	}
	logrus.Infof("Service is created: %v", s)
	return s, err
}

func (o *K8s) DeleteService(service *v1.Service, namespace string) error {
	return o.clientset.CoreV1().Services(namespace).Delete(service.GetName(), &metaV1.DeleteOptions{})
}

func (o *K8s) CreateDeployment(deployment *appsv1.Deployment, namespace string) (*appsv1.Deployment, error) {
	d, err := o.clientset.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil {
		logrus.Errorf("Error creating deployment: %v %v", d, err)
	}
	logrus.Infof("Deployment is created: %v", d)
	return d, err
}

func (o *K8s) DeleteDeployment(deployment *appsv1.Deployment, namespace string) error {
	return o.clientset.AppsV1().Deployments(namespace).Delete(deployment.GetName(), &metaV1.DeleteOptions{})
}

func (o *K8s) CreateMutatingWebhookConfiguration(mutatingWebhookConf *arv1beta1.MutatingWebhookConfiguration) (*arv1beta1.MutatingWebhookConfiguration, error) {
	awc, err := o.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(mutatingWebhookConf)
	if err != nil {
		logrus.Errorf("Error creating MutatingWebhookConfiguration: %v %v", awc, err)
	}
	logrus.Infof("MutatingWebhookConfiguration is created: %v", awc)
	return awc, err
}

func (o *K8s) DeleteMutatingWebhookConfiguration(mutatingWebhookConf *arv1beta1.MutatingWebhookConfiguration) error {
	return o.clientset.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(mutatingWebhookConf.GetName(), &metaV1.DeleteOptions{})
}

func (o *K8s) CreateSecret(secret *v1.Secret, namespace string) (*v1.Secret, error) {
	s, err := o.clientset.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		logrus.Errorf("Error creating secret: %v %v", s, err)
	}
	logrus.Infof("secret is created: %v", s)
	return s, err
}

func (o *K8s) DeleteSecret(name string, namespace string) error {
	return o.clientset.CoreV1().Secrets(namespace).Delete(name, &metaV1.DeleteOptions{})
}

func (o *K8s) IsPodReady(pod *v1.Pod) bool {
	return isPodReady(pod)
}

func (o *K8s) CreateConfigMap(cm *v1.ConfigMap) (*v1.ConfigMap, error) {
	return o.clientset.CoreV1().ConfigMaps(cm.Namespace).Create(cm)
}

func (o *K8s) CleanupConfigMaps() {
	// Clean up Network Service Endpoints
	configMaps, _ := o.clientset.CoreV1().ConfigMaps("default").List(metaV1.ListOptions{})
	for _, cm := range configMaps.Items {
		_ = o.clientset.CoreV1().ConfigMaps("default").Delete(cm.Name, &metaV1.DeleteOptions{})
	}
}

func (o *K8s) CreateRoles(rolesList ...string) ([]nsmrbac.Role, error) {
	createdRoles := []nsmrbac.Role{}
	for _, kind := range rolesList {
		role := nsmrbac.Roles[kind](nsmrbac.RoleNames[kind])
		err := role.Create(o.clientset)
		if err != nil {
			logrus.Errorf("failed creating role: %v %v", role, err)
			return createdRoles, err
		} else {
			logrus.Infof("role is created: %v", role)
			createdRoles = append(createdRoles, role)
		}
	}
	return createdRoles, nil
}

func (o *K8s) DeleteRoles(rolesList []nsmrbac.Role) error {
	for i := range rolesList {
		err := rolesList[i].Delete(o.clientset, rolesList[i].GetName())
		if err != nil {
			logrus.Errorf("failed deleting role: %v %v", rolesList[i], err)
			return err
		}
	}
	return nil
}
