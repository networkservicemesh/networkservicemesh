package kube_testing

import (
	"context"
	"errors"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
	"sync"
	"time"
)

func createAndBlock(client kubernetes.Interface, config *rest.Config, namespace string, pods ...*v1.Pod) ([]*v1.Pod, error) {

	var wg sync.WaitGroup
	errChan := make(chan error)
	doneChan := make(chan struct{})
	resultChan := make(chan *v1.Pod, len(pods))

	for _, pod := range pods {

		wg.Add(1)
		go func(pod *v1.Pod) {

			var err error
			pod, err = client.CoreV1().Pods(namespace).Create(pod)
			if err != nil {
				errChan <- err
				return
			}

			err = blockUntilPodReady(client, context.Background(), pod)
			if err != nil {
				errChan <- err
				return
			}

			// Let's fetch more information about pod created

			updated_pod, error := client.CoreV1().Pods(namespace).Get(pod.Name, metaV1.GetOptions{})
			if err != nil {
				errChan <- error
				return
			}
			resultChan <- updated_pod
			wg.Done()

		}(pod)
	}

	go func() {
		wg.Wait()
		close(doneChan)
	}()

	select {
	case err := <-errChan:
		return nil, err
	case <-doneChan:
		results := make([]*v1.Pod, len(pods))
		named := map[string]*v1.Pod{}
		for i := 0; i < len(pods); i++ {
			pod := <-resultChan
			named[pod.Name] = pod
		}
		for i := 0; i < len(pods); i++ {
			results[i] = named[pods[i].Name]
		}

		// We need to put pods in right order
		return results, nil
	}
}

func blockUntilPodReady(client kubernetes.Interface, context context.Context, pod *v1.Pod) error {

	err := blockUntilPodExists(client, context, pod)
	if err != nil {
		return err
	}

	watcher, err := client.CoreV1().Pods(pod.Namespace).Watch(metaV1.SingleObject(metaV1.ObjectMeta{Name: pod.Name}))

	if err != nil {
		return err
	}

	for {
		select {
		case <-context.Done():
			return errors.New("context cancelled/timeout")
		case _, ok := <-watcher.ResultChan():

			if !ok {
				return nil
			}

			pod, error := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metaV1.GetOptions{})
			if error == nil {
				if isPodReady(pod) {
					watcher.Stop()
					return nil
				}
			}
		}
	}
}

func isPodReady(pod *v1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false
		}
	}

	return true
}

func blockUntilPodExists(client kubernetes.Interface, context context.Context, pod *v1.Pod) error {

	exists := make(chan error)

	go func() {
		for {
			pod, err := client.CoreV1().Pods(pod.Namespace).Get(pod.Name, metaV1.GetOptions{})
			if err != nil {
				exists <- err
				break
			}

			if pod != nil && pod.Status.Phase != v1.PodPending {
				close(exists)
				break
			}

			time.Sleep(time.Millisecond * time.Duration(50))
		}
	}()

	select {
	case <-context.Done():
		return errors.New("context cancelled/timeout")
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
		return errors.New("context cancelled/timeout")
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
	clientset kubernetes.Interface
	pods      []*v1.Pod
	config    *rest.Config
}

func NewK8s() (*K8s, error) {
	path := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", path)
	Expect(err).To(BeNil())

	client := K8s{
		pods: []*v1.Pod{},
	}
	client.config = config
	client.clientset, err = kubernetes.NewForConfig(config)

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
		err = blockUntilPodWorking(o.clientset, context.Background(), pod)
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

func (l *K8s) Cleanup() {
	for _, result := range l.pods {
		err := l.deletePods(result)
		Expect(err).To(BeNil())
	}
}

func (l *K8s) Prepare(noPods ...string) {
	for _, podName := range noPods {
		for _, lpod := range l.ListPods() {
			if strings.Contains(lpod.Name, podName) {
				l.DeletePods("default", &lpod)
			}
		}
	}
}

func (l *K8s) CreatePods(templates ...*v1.Pod) []*v1.Pod {
	results, err := createAndBlock(l.clientset, l.config, "default", templates...)
	Expect(err).To(BeNil())

	l.pods = append(l.pods, results...)
	return results
}

func (l *K8s) CreatePod(template *v1.Pod) *v1.Pod {
	results, err := createAndBlock(l.clientset, l.config, "default", template)
	Expect(err).To(BeNil())

	l.pods = append(l.pods, results...)
	return results[0]
}

func (l *K8s) DeletePods(namespace string, pods ...*v1.Pod) {
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
	response := k8s.clientset.CoreV1().Pods("default").GetLogs(pod.Name, getLogsOpt)
	result, error := response.DoRaw()
	if error != nil {
		return "", error
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
			time.Sleep(10 * time.Millisecond)
		} else {
			break
		}
		if time.Since(st) > timeout {
			logrus.Errorf("Timeout waiting for logs pattern %s in pod %s. Last logs: %s", pattern, pod.Name, logs)
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
