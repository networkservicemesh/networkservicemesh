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
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	podStartTimeout  = 5 * time.Minute
	podDeleteTimeout = 5 * time.Minute
	podExecTimeout	 = 3 * time.Minute
	podGetLogTimeout = 3 * time.Minute
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
			if createdPod != nil {
				pod = createdPod
			}
			if err != nil {
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

	if !waitTimeout(fmt.Sprintf("createAndBlock with pods: %v", pods ), &wg, timeout) {
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

		if time.Since(st) > timeout / 2 {
			logrus.Infof("Pod deploy half time passed: %v", pod)
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
}

func NewK8s() (*K8s, error) {

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

func (o *K8s) deletePods(pods ...*v1.Pod) error {
	for _, pod := range pods {
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
	managers, err := o.versionedClientSet.Networkservicemesh().NetworkServiceManagers("default").List(metaV1.ListOptions{})
	Expect(err).To(BeNil())
	for _, mgr := range managers.Items {
		_ = o.versionedClientSet.Networkservicemesh().NetworkServiceManagers("default").Delete(mgr.Name, &metaV1.DeleteOptions{})
	}
	endpoints, err := o.versionedClientSet.Networkservicemesh().NetworkServiceEndpoints("default").List(metaV1.ListOptions{})
	Expect(err).To(BeNil())
	for _, ep := range endpoints.Items {
		_ = o.versionedClientSet.Networkservicemesh().NetworkServiceEndpoints("default").Delete(ep.Name, &metaV1.DeleteOptions{})
	}
	services, err := o.versionedClientSet.Networkservicemesh().NetworkServices("default").List(metaV1.ListOptions{})
	Expect(err).To(BeNil())
	for _, service := range services.Items {
		_ = o.versionedClientSet.Networkservicemesh().NetworkServices("default").Delete(service.Name, &metaV1.DeleteOptions{})
	}
}

func (l *K8s) Cleanup() {
	for _, result := range l.pods {
		err := l.deletePods(result)
		Expect(err).To(BeNil())
	}
	l.CleanupCRDs()
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
		pods = append(pods, podResult.pod)
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
			<- time.Tick(100 * time.Millisecond)
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
			logrus.Warnf("Waiting for %d nodes to arrive, currenctly have: %d", len(nodes), requiredNumber)
			warnPrinted = true
		}
		time.Sleep(50 * time.Millisecond)
	}

}

func (o *K8s) CreateService(service *v1.Service) {
	_ = o.clientset.CoreV1().Services("default").Delete(service.Name, &metaV1.DeleteOptions{})
	s, err := o.clientset.CoreV1().Services("default").Create(service)
	if err != nil {
		logrus.Errorf("Error creating service: %v %v", s, err)
	}
	logrus.Infof("Service is created: %v", s)
}

func (o *K8s) IsPodReady(pod *v1.Pod) bool {
	return isPodReady(pod)
}
