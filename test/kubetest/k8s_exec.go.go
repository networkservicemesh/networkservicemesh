package kubetest

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

type Writer struct {
	Str string
}

func (w *Writer) Write(p []byte) (n int, err error) {
	str := string(p)
	if len(str) > 0 {
		w.Str += str
	}
	return len(str), nil
}

//Exec executes command in pod's container
func (k8s *K8s) Exec(pod *v1.Pod, container string, command ...string) (string, string, error) {
	var resp1, resp2 string
	var err error
	for retryCount := 0; retryCount < 10; retryCount++ {
		resp1, resp2, err = k8s.doExec(pod, container, command...)
		if err != nil && strings.Contains(err.Error(), fmt.Sprintf("container not found (\"%v\")", container)) {
			<-time.After(100 * time.Millisecond)
			continue
		}
		break
	}
	return resp1, resp2, err
}
func (k8s *K8s) doExec(pod *v1.Pod, container string, command ...string) (string, string, error) {
	logrus.Infof("Executing: %v in pod %v:%v", command, pod.Name, container)
	execRequest := k8s.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	if len(container) > 0 {
		execRequest = execRequest.Param("container", container)
	}

	exec, err := remotecommand.NewSPDYExecutor(k8s.config, "POST", execRequest.URL())
	if err != nil {
		return "", "", err
	}

	stdIn := strings.NewReader("")
	stdOut := new(Writer)
	stdErr := new(Writer)

	options := remotecommand.StreamOptions{
		Stdin:  stdIn,
		Stdout: stdOut,
		Stderr: stdErr,
		Tty:    false,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = exec.Stream(options)
	}()
	if !waitTimeout(fmt.Sprintf("Exec %v:%v cmdline: %v", pod.Name, container, command), &wg, podExecTimeout) {
		err = errors.Errorf("timed out executing command %v in pod %v", command, pod.Name)
		logrus.Errorf("Failed to do exec. Timeout")
	}

	return stdOut.Str, stdErr.Str, err
}
