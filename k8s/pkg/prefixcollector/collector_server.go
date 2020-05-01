package prefixcollector

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/networkservicemesh/networkservicemesh/sdk/prefix_pool"
)

type SubnetWatcher struct {
	subnetCh <-chan *net.IPNet
	stopCh   chan struct{}
}

func (s *SubnetWatcher) Stop() {
	close(s.stopCh)
}

func (s *SubnetWatcher) ResultChan() <-chan *net.IPNet {
	return s.subnetCh
}

type keyFunc func(event watch.Event) (string, error)
type subnetFunc func(event watch.Event) (*net.IPNet, error)

func WatchPodCIDR(clientset *kubernetes.Clientset) (*SubnetWatcher, error) {
	nodeWatcher, err := clientset.CoreV1().Nodes().Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	nodeCast := func(obj runtime.Object) (*v1.Node, error) {
		node, ok := obj.(*v1.Node)
		if !ok {
			return nil, errors.Errorf("Casting object to *v1.Service failed. Object: %v", obj)
		}
		return node, nil
	}

	keyFunc := func(event watch.Event) (string, error) {
		node, err := nodeCast(event.Object)
		if err != nil {
			return "", err
		}
		return node.Name, nil
	}

	subnetFunc := func(event watch.Event) (*net.IPNet, error) {
		node, err := nodeCast(event.Object)
		if err != nil {
			return nil, err
		}
		_, ipNet, err := net.ParseCIDR(node.Spec.PodCIDR)
		if err != nil {
			return nil, err
		}
		return ipNet, nil
	}

	return watchSubnet(nodeWatcher, keyFunc, subnetFunc)
}

func WatchServiceIpAddr(cs *kubernetes.Clientset) (*SubnetWatcher, error) {
	serviceWatcher, err := newServiceWatcher(cs)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	serviceCast := func(obj runtime.Object) (*v1.Service, error) {
		service, ok := obj.(*v1.Service)
		if !ok {
			return nil, errors.Errorf("Casting object to *v1.Service failed. Object: %v", obj)
		}
		return service, nil
	}

	keyFunc := func(event watch.Event) (string, error) {
		service, err := serviceCast(event.Object)
		if err != nil {
			return "", err
		}
		return service.Name, nil
	}

	subnetFunc := func(event watch.Event) (*net.IPNet, error) {
		service, err := serviceCast(event.Object)
		if err != nil {
			return nil, err
		}
		ipAddr := service.Spec.ClusterIP
		return prefix_pool.IpToNet(net.ParseIP(ipAddr)), nil
	}

	return watchSubnet(serviceWatcher, keyFunc, subnetFunc)
}

type serviceWatcher struct {
	resultCh chan watch.Event
	stopCh   chan struct{}
}

func (s *serviceWatcher) Stop() {
	close(s.stopCh)
}

func (s *serviceWatcher) ResultChan() <-chan watch.Event {
	return s.resultCh
}

func newServiceWatcher(cs *kubernetes.Clientset) (watch.Interface, error) {
	ns, err := getNamespaces(cs)
	if err != nil {
		return nil, err
	}
	resultCh := make(chan watch.Event, 10)
	stopCh := make(chan struct{})

	for _, n := range ns {
		w, err := cs.CoreV1().Services(n).Watch(context.TODO(), metav1.ListOptions{})
		if err != nil {
			logrus.Errorf("Unable to watch services in %v namespace: %v", n, err)
			close(stopCh)
			return nil, err
		}

		go func() {
			for {
				select {
				case <-stopCh:
					return
				case e := <-w.ResultChan():
					resultCh <- e
				}
			}
		}()
	}

	return &serviceWatcher{
		resultCh: resultCh,
		stopCh:   stopCh,
	}, nil
}

func getNamespaces(cs *kubernetes.Clientset) ([]string, error) {
	ns, err := cs.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	rv := []string{}
	for _, n := range ns.Items {
		rv = append(rv, n.Name)
	}
	return rv, nil
}

func watchSubnet(resourceWatcher watch.Interface, keyFunc keyFunc, subnetFunc subnetFunc) (*SubnetWatcher, error) {
	subnetCh := make(chan *net.IPNet, 10)
	stopCh := make(chan struct{})

	cache := map[string]string{}
	var lastIpNet *net.IPNet

	go func() {
		for {
			select {
			case <-stopCh:
				resourceWatcher.Stop()
				return
			case event, ok := <-resourceWatcher.ResultChan():
				if !ok {
					close(subnetCh)
					return
				}

				if event.Type == watch.Error || event.Type == watch.Deleted {
					continue
				}

				ipNet, err := subnetFunc(event)
				if err != nil {
					continue
				}

				key, err := keyFunc(event)
				if err != nil {
					continue
				}
				logrus.Infof("Receive resource: name %v, subnet %v", key, ipNet.String())

				if subnet, exist := cache[key]; exist && subnet == ipNet.String() {
					continue
				}
				cache[key] = ipNet.String()

				if lastIpNet == nil {
					lastIpNet = ipNet
					subnetCh <- lastIpNet
					continue
				}

				newIpNet := prefix_pool.MaxCommonPrefixSubnet(lastIpNet, ipNet)
				if newIpNet.String() != lastIpNet.String() {
					logrus.Infof("Subnet extended from %v to %v", lastIpNet, newIpNet)
					lastIpNet = newIpNet
					subnetCh <- lastIpNet
					continue
				}
			}
		}
	}()

	return &SubnetWatcher{
		subnetCh: subnetCh,
		stopCh:   stopCh,
	}, nil
}
