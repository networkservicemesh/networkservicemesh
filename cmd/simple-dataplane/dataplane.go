package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/vishvananda/netns"

	"github.com/vishvananda/netlink"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	dataplaneSocket = "/var/lib/networkservicemesh/dataplane.sock"
)

var (
	// podName1   = flag.String("pod-1", "", "POD name for process 1")
	// podName2   = flag.String("pod-2", "", "POD name for process 2")
	// namespace  = flag.String("namespace", "default", "PODs namespace")
	dataplane  = flag.String("dataplane-scoket", dataplaneSocket, "Location of the dataplane gRPC socket")
	kubeconfig = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

func buildClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv != "" {
		kubeconfig = &kubeconfigEnv
	}

	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}
	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8s, nil
}

func getContainerID(k8s *kubernetes.Clientset, pn, ns string) (string, error) {
	pl, err := k8s.CoreV1().Pods(ns).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, p := range pl.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, pn) {
			return strings.Split(p.Status.ContainerStatuses[0].ContainerID, "://")[1][:12], nil
		}
	}

	return "", nil
}

func setVethPair(ns1, ns2 netns.NsHandle, p1, p2 string) error {
	ns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current process namespace")
	}
	defer netns.Set(ns)

	linkAttr := netlink.NewLinkAttrs()
	linkAttr.Name = p2
	veth := &netlink.Veth{
		LinkAttrs: linkAttr,
		PeerName:  p1,
	}
	if err := netns.Set(ns1); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns1, err)
	}
	namespaceHandle, err := netlink.NewHandleAt(ns1)
	if err != nil {
		return fmt.Errorf("failure to get pod's handle with error: %+v", err)
	}
	if err := namespaceHandle.LinkAdd(veth); err != nil {
		return fmt.Errorf("failure to add veth to pod with error: %+v", err)
	}

	link, err := netlink.LinkByName(p2)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := link.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	peer, err := netlink.LinkByName(p1)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := peer.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	if err := netlink.LinkSetNsFd(peer, int(ns2)); err != nil {
		return fmt.Errorf("failure to get place veth into peer's pod with error: %+v", err)
	}
	return nil
}

func setVethAddr(ns1, ns2 netns.NsHandle, p1, p2 string) error {

	var addr1 = &net.IPNet{IP: net.IPv4(1, 1, 1, 1), Mask: net.CIDRMask(30, 32)}
	var addr2 = &net.IPNet{IP: net.IPv4(1, 1, 1, 2), Mask: net.CIDRMask(30, 32)}
	var vethAddr1 = &netlink.Addr{IPNet: addr1, Peer: addr2}
	var vethAddr2 = &netlink.Addr{IPNet: addr2, Peer: addr1}

	ns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current process namespace")
	}
	defer netns.Set(ns)

	if err := netns.Set(ns1); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns1, err)
	}
	link, err := netlink.LinkByName(p2)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := link.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	if err := netlink.AddrAdd(link, vethAddr1); err != nil {
		return fmt.Errorf("failure to assign IP to veth interface with error: %+v", err)
	}
	if err := netns.Set(ns2); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", ns1, err)
	}
	peer, err := netlink.LinkByName(p1)
	if err != nil {
		return fmt.Errorf("failure to get pod's interface by name with error: %+v", err)
	}
	if _, ok := peer.(*netlink.Veth); !ok {
		return fmt.Errorf("failure, got unexpected interface type: %+v", reflect.TypeOf(link))
	}
	if err := netlink.AddrAdd(peer, vethAddr2); err != nil {
		return fmt.Errorf("failure to assign IP to veth interface with error: %+v", err)
	}
	return nil
}

func init() {
	runtime.LockOSThread()
}

func main() {
	flag.Parse()

	if *podName1 == "" || *podName2 == "" {
		log.Fatal("Pod names cannot be empty, exitting...")
	}

	k8s, err := buildClient()
	if err != nil {
		log.Fatal("Failed to build kubernetes client, exitting...")
	}

	cid1, err := getContainerID(k8s, *podName1, *namespace)
	if err != nil {
		log.Fatalf("Failed to get container ID for pod %s/%s with error: %+v", *namespace, *podName1, err)
	}
	log.Printf("Discovered Container ID %s for pod %s/%s", cid1, *namespace, *podName1)

	cid2, err := getContainerID(k8s, *podName2, *namespace)
	if err != nil {
		log.Fatalf("Failed to get container ID for pod %s/%s with error: %+v", *namespace, *podName2, err)
	}
	log.Printf("Discovered Container ID %s for pod %s/%s", cid2, *namespace, *podName2)

	ns1, err := netns.GetFromDocker(cid1)
	if err != nil {
		log.Fatalf("Failed to get Linux namespace for pod %s/%s with error: %+v", *namespace, *podName1, err)
	}
	ns2, err := netns.GetFromDocker(cid2)
	if err != nil {
		log.Fatalf("Failed to get Linux namespace for pod %s/%s with error: %+v", *namespace, *podName1, err)
	}
	log.Printf("Discovered namespaces: %s %s", ns1.String(), ns2.String())

	if err := setVethPair(ns1, ns2, *podName1, *podName2); err != nil {
		log.Fatalf("Failed to get add veth pair to pods %s/%s and %s/%s with error: %+v",
			*namespace, *podName1, *namespace, *podName2, err)
	}

	if err := setVethAddr(ns1, ns2, *podName1, *podName2); err != nil {
		log.Fatalf("Failed to assign ip addresses to veth pair for pods %s/%s and %s/%s with error: %+v",
			*namespace, *podName1, *namespace, *podName2, err)
	}

	if err := listInterfaces(ns1); err != nil {
		log.Fatalf("Failed to list interfaces of to %s/%swith error: %+v", *namespace, *podName1, err)
	}
	if err := listInterfaces(ns2); err != nil {
		log.Fatalf("Failed to list interfaces of %s/%swith error: %+v", *namespace, *podName2, err)
	}
}

func listInterfaces(targetNS netns.NsHandle) error {
	ns, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current process namespace")
	}
	defer netns.Set(ns)

	if err := netns.Set(targetNS); err != nil {
		return fmt.Errorf("failed to switch to namespace %s with error: %+v", targetNS, err)
	}
	namespaceHandle, err := netlink.NewHandleAt(targetNS)
	if err != nil {
		return fmt.Errorf("failure to get pod's handle with error: %+v", err)
	}
	interfaces, err := namespaceHandle.LinkList()
	if err != nil {
		return fmt.Errorf("failure to get pod's interfaces with error: %+v", err)
	}
	log.Printf("pod's interfaces:")
	for _, intf := range interfaces {
		addrs, err := namespaceHandle.AddrList(intf, 0)
		if err != nil {
			log.Printf("failed to addresses for interface: %s with error: %v", intf.Attrs().Name, err)
		}
		log.Printf("Name: %s Type: %s Addresses: %+v", intf.Attrs().Name, intf.Type(), addrs)

	}
	return nil
}
