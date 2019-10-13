package kubetest

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	spdy2 "k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForward struct {
	// The initialized Kubernetes client.
	k8s *K8s
	pod *v1.Pod
	// The port on the pod to forward traffic to.
	DestinationPort int
	// The port that the port forward should listen to, random if not set.
	ListenPort int
	stopChan   chan struct{}
	readyChan  chan struct{}
	pf         *portforward.PortForwarder
}

func (k8s *K8s) NewPortForwarder(pod *v1.Pod, port int) (*PortForward, error) {
	pf := &PortForward{
		pod:             pod,
		k8s:             k8s,
		DestinationPort: port,
	}
	return pf, nil
}

// Start a port forward to a pod - blocks until the tunnel is ready for use.
func (p *PortForward) Start() error {
	p.stopChan = make(chan struct{}, 1)
	readyChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)

	listenPort, err := p.getListenPort()
	if err != nil {
		return errors.Wrap(err, "Could not find a port to bind to")
	}

	dialer, err := p.dialer()
	if err != nil {
		return errors.Wrap(err, "Could not create a dialer")
	}

	ports := []string{
		fmt.Sprintf("%d:%d", listenPort, p.DestinationPort),
	}

	discard := ioutil.Discard
	pf, err := portforward.New(dialer, ports, p.stopChan, readyChan, discard, discard)
	if err != nil {
		return errors.Wrap(err, "Could not port forward into pod")
	}
	p.pf = pf

	go func() {
		errChan <- pf.ForwardPorts()
	}()

	select {
	case err = <-errChan:
		return errors.Wrap(err, "Could not create port forward")
	case <-readyChan:
		return nil
	}
}

func (p *PortForward) Stop() {
	p.stopChan <- struct{}{}
	p.pf.Close()
}

func (p *PortForward) getListenPort() (int, error) {
	var err error

	if p.ListenPort == 0 {
		p.ListenPort, err = p.getFreePort()
	}

	return p.ListenPort, err
}

func (p *PortForward) getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	port := listener.Addr().(*net.TCPAddr).Port
	err = listener.Close()
	if err != nil {
		return 0, err
	}

	return port, nil
}

// Create an httpstream.Dialer for use with portforward.New
func (p *PortForward) dialer() (httpstream.Dialer, error) {
	podname := ""
	if p.pod != nil {
		podname = p.pod.Name
	} else {
		return nil, errors.New("Could not do POST request for a non-existing pod")
	}
	url := p.k8s.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(p.k8s.GetK8sNamespace()).
		Name(podname).
		SubResource("portforward").URL()

	transport, upgrader, err := roundTripperFor(p.k8s.config)
	if err != nil {
		return nil, errors.Wrap(err, "Could not create round tripper")
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
	return dialer, nil
}

func roundTripperFor(config *rest.Config) (http.RoundTripper, httpstream.UpgradeRoundTripper, error) {
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, nil, err
	}
	upgradeRoundTripper := spdy2.NewRoundTripper(tlsConfig, true, true)
	wrapper, err := rest.HTTPWrappersForConfig(config, upgradeRoundTripper)
	if err != nil {
		return nil, nil, err
	}
	return wrapper, upgradeRoundTripper, nil
}
