package kube_testing

import (
	"strings"

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

func (o *K8s) Exec(pod *v1.Pod, container string, command ...string) (string, string, error) {
	execRequest := o.clientset.CoreV1().RESTClient().Post().
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

	exec, err := remotecommand.NewSPDYExecutor(o.config, "POST", execRequest.URL())
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
	err = exec.Stream(options)

	return stdOut.Str, stdErr.Str, err
}
