package kube_testing

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"strings"
	"sync"
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

func (o *K8s) Exec(pod *v1.Pod, container string, commandParts ...string) (string, string, error) {
	stdOut := new(Writer)
	stdErr := new(Writer)

	command := &Command{
		Stdin:  strings.NewReader(""),
		Stdout: stdOut,
		Stderr: stdErr,
		Parts:  commandParts,
	}

	err := o.ExecCommand(pod, container, command)
	return stdOut.Str, stdErr.Str, err
}

type Command struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Parts  []string
}

func (o *K8s) ExecCommand(pod *v1.Pod, container string, command *Command) error {
	logrus.Infof("Executing: %v in pod %v:%v", command, pod.Name, container)
	execRequest := o.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: container,
			Command:   command.Parts,
			Stdin:     command.Stdin != nil,
			Stdout:    command.Stdout != nil,
			Stderr:    command.Stderr != nil,
			TTY:       false,
		}, scheme.ParameterCodec)

	if len(container) > 0 {
		execRequest = execRequest.Param("container", container)
	}

	exec, err := remotecommand.NewSPDYExecutor(o.config, "POST", execRequest.URL())
	if err != nil {
		return err
	}

	options := remotecommand.StreamOptions{
		Stdin:  command.Stdin,
		Stdout: command.Stdout,
		Stderr: command.Stderr,
		Tty:    false,
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = exec.Stream(options)
	}()
	if !waitTimeout(fmt.Sprintf("Exec %v:%v cmdline: %v", pod.Name, container, command), &wg, podExecTimeout) {
		logrus.Errorf("Failed to do exec. Timeout")
	}

	return err
}
